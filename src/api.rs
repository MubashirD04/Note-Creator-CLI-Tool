use serde::{Deserialize, Serialize};
use colored::Colorize;
use std::time::Duration;
use crate::config::{get_or_prompt_groq_key, persist_verified_key, clear_groq_key, KeySource};

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

const MAX_KEY_RETRIES: u8 = 2;

pub async fn summarize_text(body: &str) -> Result<String, Box<dyn std::error::Error>> {
    if body.trim().is_empty() {
        return Err("Cannot summarize empty text.".into());
    }

    for attempt in 1..=MAX_KEY_RETRIES {
        let (api_key, source) = get_or_prompt_groq_key()?;

        match try_summarize(body, &api_key).await {
            Ok(summary) => {
                if source == KeySource::Prompt {
                    if let Err(e) = persist_verified_key(&api_key) {
                        eprintln!(
                            "{}: {}","Warning: could not save key to OS keychain, you'll be asked again next run".red(),e
                        );
                    }
                }
                return Ok(summary);
            }
            Err(AttemptError::InvalidKey(msg)) => {
                if source == KeySource::Keychain {
                    let _ = clear_groq_key();
                }

                if attempt < MAX_KEY_RETRIES {
                    eprintln!(
                        "{}",
                        format!("{} Please re-enter a valid key.", msg).yellow()
                    );
                    continue;
                } else {
                    return Err(msg.into());
                }
            }
            Err(AttemptError::Other(e)) => return Err(e),
        }
    }

    unreachable!("loop always returns on its final iteration")
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
        model: "llama-3.1-8b-instant".to_string(),
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
        .post("https://api.groq.com/openai/v1/chat/completions")
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