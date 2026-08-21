use serde::{Deserialize, Serialize};
use colored::Colorize;
use std::time::Duration;
use crate::config::{find_cached_key, prompt_once, persist_verified_key, clear_groq_key, KeySource};

#[derive(Serialize)]
struct ChatMessage {
    role: String,
    content: String,
}

#[derive(Serialize)]
struct GroqRequest {
    model: String,
    messages: Vec<ChatMessage>,
    temperature: f32,
}


#[derive(Deserialize)]
struct ChatContent {
    content: String,
}

#[derive(Deserialize)]
struct GroqChoice {
    message: ChatContent,
}

#[derive(Deserialize)]
struct GroqResponse {
    choices: Vec<GroqChoice>,
}

#[derive(Deserialize)]
struct GroqErrorBody {
    error: GroqErrorDetail,
}

#[derive(Deserialize)]
struct GroqErrorDetail {
    message: String,
}

enum AttemptError {
    InvalidKey(String),
    Other(Box<dyn std::error::Error>),
}

const MAX_PROMPT_ATTEMPTS: u32 = 4;
const DEFAULT_GROQ_API_BASE: &str = "https://api.groq.com/openai/v1";

/// Base URL for the Groq chat completions endpoint. Overridable via
/// GROQ_API_BASE_URL so tests can point it at a local mock server.
fn groq_api_base() -> String {
    std::env::var("GROQ_API_BASE_URL").unwrap_or_else(|_| DEFAULT_GROQ_API_BASE.to_string())
}

pub async fn summarize_text(body: &str) -> Result<String, Box<dyn std::error::Error>> {
    if body.trim().is_empty() {
        return Err("Cannot summarize empty text.".into());
    }

    if let Some((cached_key, source)) = find_cached_key() {
        match try_summarize(body, &cached_key).await {
            Ok(summary) => return Ok(summary),
            Err(AttemptError::InvalidKey(msg)) => {
                if source == KeySource::Keychain {
                    let _ = clear_groq_key();
                }
                eprintln!(
                    "{}",
                    format!("{} You'll be asked to enter one now.", msg).yellow()
                );
            }
            Err(AttemptError::Other(e)) => return Err(e),
        }
    }

    for attempt in 1..=MAX_PROMPT_ATTEMPTS {
        let typed_key = prompt_once(attempt, MAX_PROMPT_ATTEMPTS)?;

        let Some(api_key) = typed_key else {
            continue;
        };

        match try_summarize(body, &api_key).await {
            Ok(summary) => {
                if let Err(e) = persist_verified_key(&api_key) {
                    eprintln!(
                        "{}: {}",
                        "Warning: could not save key to OS keychain, you'll be asked again next run".yellow(),
                        e
                    );
                }
                return Ok(summary);
            }
            Err(AttemptError::InvalidKey(msg)) => {
                if attempt < MAX_PROMPT_ATTEMPTS {
                    eprintln!(
                        "{}",
                        format!("{} ({} attempt(s) left.)", msg, MAX_PROMPT_ATTEMPTS - attempt).yellow()
                    );
                } else {
                    return Err(format!("{} No attempts remaining.", msg).into());
                }
            }
            Err(AttemptError::Other(e)) => return Err(e),
        }
    }

    Err(format!(
        "No valid Groq API key provided after {} attempts.",
        MAX_PROMPT_ATTEMPTS
    )
    .into())
}

async fn try_summarize(body: &str, api_key: &str) -> Result<String, AttemptError> {
    let sanitized_key = api_key.trim();
    if sanitized_key.is_empty() {
        return Err(AttemptError::InvalidKey(
            "Stored API key is empty.".to_string(),
        ));
    }
    if sanitized_key.chars().any(|c| c.is_control()) {
        return Err(AttemptError::InvalidKey(
            "Stored API key contains invalid characters (likely a corrupted keychain entry)."
                .to_string(),
        ));
    }

    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(30))
        .build()
        .map_err(|e| AttemptError::Other(format!("Failed to build HTTP client: {}", e).into()))?;

    let system_prompt = "You are a concise AI assistant. Summarize the provided text into a clean, well-formatted Markdown summary with key takeaways.";

    let payload = GroqRequest {
        model: "openai/gpt-oss-20b".to_string(),
        messages: vec![
            ChatMessage {
                role: "system".to_string(),
                content: system_prompt.to_string(),
            },
            ChatMessage {
                role: "user".to_string(),
                content: body.to_string(),
            },
        ],
        temperature: 0.5,
    };

    let response = client
        .post(format!("{}/chat/completions", groq_api_base()))
        .header("Authorization", format!("Bearer {}", sanitized_key))
        .header("Content-Type", "application/json")
        .json(&payload)
        .send()
        .await
        .map_err(|e| -> AttemptError {
            let msg = if e.is_timeout() {
                "Request to Groq API timed out after 30s. Check your connection and try again.".to_string()
            } else if e.is_connect() {
                "Could not connect to Groq API. Check your internet connection.".to_string()
            } else {
                format!("Network error while contacting Groq API: {}", e)
            };
            AttemptError::Other(msg.into())
        })?;

    let status = response.status();

    if !status.is_success() {
        let err_text = response.text().await.unwrap_or_default();
        let parsed_message = serde_json::from_str::<GroqErrorBody>(&err_text)
            .map(|b| b.error.message)
            .unwrap_or_else(|_| err_text.clone());

        if status.as_u16() == 401 {
            return Err(AttemptError::InvalidKey(format!(
                "Groq API rejected the API key (401 Unauthorized): {}.",parsed_message
            )));
        }

        if status.as_u16() == 429 {
            return Err(AttemptError::Other(
                format!(
                    "Groq API rate limit exceeded: {}. Wait a bit and try again.",parsed_message).into(),
            ));
        }

        return Err(AttemptError::Other(
            format!("Groq API error ({}): {}", status, parsed_message).into(),
        ));
    }

    let raw_body = response
        .text()
        .await
        .map_err(|e| AttemptError::Other(format!("Failed to read Groq API response body: {}", e).into()))?;
    let groq_res: GroqResponse = serde_json::from_str(&raw_body).map_err(|e| {
        AttemptError::Other(
            format!(
                "Failed to parse Groq API response ({}). Raw response: {}",e, raw_body).into(),
        )
    })?;

    let summary = groq_res
        .choices
        .first()
        .map(|c| c.message.content.clone())
        .ok_or_else(|| AttemptError::Other("Groq API returned no choices in its response.".into()))?;

    if summary.trim().is_empty() {
        return Err(AttemptError::Other("Groq API returned an empty summary.".into()));
    }

    Ok(summary)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;

    fn err_string(e: AttemptError) -> String {
        match e {
            AttemptError::InvalidKey(msg) => msg,
            AttemptError::Other(err) => err.to_string(),
        }
    }

    #[tokio::test]
    async fn empty_key_is_rejected_without_network_call() {
        let result = try_summarize("hello world", "").await;
        assert!(matches!(result, Err(AttemptError::InvalidKey(_))));
        assert!(err_string(result.unwrap_err()).contains("empty"));
    }

    #[tokio::test]
    async fn whitespace_only_key_is_rejected() {
        let result = try_summarize("hello world", "   ").await;
        assert!(matches!(result, Err(AttemptError::InvalidKey(_))));
    }

    #[tokio::test]
    async fn key_with_control_characters_is_rejected() {
        let bad_key = "gsk_abc\u{0007}def";
        let result = try_summarize("hello world", bad_key).await;
        assert!(matches!(result, Err(AttemptError::InvalidKey(_))));
        assert!(err_string(result.unwrap_err()).contains("invalid characters"));
    }

    #[tokio::test]
    async fn summarize_text_rejects_empty_body() {
        let result = summarize_text("   \n\t  ").await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Cannot summarize empty text"));
    }

    #[tokio::test]
    #[serial]
    async fn successful_response_returns_summary() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"choices":[{"message":{"content":"A concise summary."}}]}"#)
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Ok(summary) => assert_eq!(summary, "A concise summary."),
            Err(e) => panic!("expected Ok, got error: {}", err_string(e)),
        }
    }

    #[tokio::test]
    #[serial]
    async fn unauthorized_response_maps_to_invalid_key() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(401)
            .with_header("content-type", "application/json")
            .with_body(r#"{"error":{"message":"Invalid API Key"}}"#)
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_badkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::InvalidKey(msg)) => {
                assert!(msg.contains("401"));
                assert!(msg.contains("Invalid API Key"));
            }
            other => panic!("expected InvalidKey error, got: {}", other.is_ok()),
        }
    }

    #[tokio::test]
    #[serial]
    async fn rate_limit_response_maps_to_other_error() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(429)
            .with_header("content-type", "application/json")
            .with_body(r#"{"error":{"message":"Rate limit reached"}}"#)
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => {
                let msg = e.to_string();
                assert!(msg.contains("rate limit"));
                assert!(msg.contains("Rate limit reached"));
            }
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }

    #[tokio::test]
    #[serial]
    async fn server_error_maps_to_other_error() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(500)
            .with_header("content-type", "text/plain")
            .with_body("internal server error")
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => {
                let msg = e.to_string();
                assert!(msg.contains("500"));
                assert!(msg.contains("internal server error"));
            }
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }

    #[tokio::test]
    #[serial]
    async fn malformed_json_response_is_reported() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body("not valid json")
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => assert!(e.to_string().contains("Failed to parse")),
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }

    #[tokio::test]
    #[serial]
    async fn empty_choices_array_is_reported() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"choices":[]}"#)
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => assert!(e.to_string().contains("no choices")),
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }

    #[tokio::test]
    #[serial]
    async fn empty_summary_content_is_reported() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"choices":[{"message":{"content":"   "}}]}"#)
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => assert!(e.to_string().contains("empty summary")),
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }

    #[tokio::test]
    #[serial]
    async fn error_body_that_is_not_json_falls_back_to_raw_text() {
        let mut server = mockito::Server::new_async().await;
        unsafe { std::env::set_var("GROQ_API_BASE_URL", server.url()); }

        let _mock = server
            .mock("POST", "/chat/completions")
            .with_status(403)
            .with_header("content-type", "text/plain")
            .with_body("forbidden: no soup for you")
            .create_async()
            .await;

        let result = try_summarize("hello world", "gsk_validkey").await;
        unsafe { std::env::remove_var("GROQ_API_BASE_URL"); }

        match result {
            Err(AttemptError::Other(e)) => assert!(e.to_string().contains("forbidden: no soup for you")),
            other => panic!("expected Other error, got Ok: {}", matches!(other, Ok(_))),
        }
    }
}