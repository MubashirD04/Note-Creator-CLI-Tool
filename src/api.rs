use crate::config::{clear_groq_key, get_or_prompt_groq_key};
use serde::{Deserialize, Serialize};
use std::time::Duration;

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

pub async fn summarize_text(body: &str) -> Result<String, Box<dyn std::error::Error>> {
    if body.trim().is_empty() {
        return Err("Cannot summarize empty text.".into());
    }

    let api_key = get_or_prompt_groq_key()?;

    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(30))
        .build()?;

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
        .header("Authorization", format!("Bearer {}", api_key))
        .header("Content-Type", "application/json")
        .json(&payload)
        .send()
        .await
        .map_err(|e| -> Box<dyn std::error::Error> {
            if e.is_timeout() {
                "Request to Groq API timed out after 30s. Check your connection and try again."
                    .into()
            } else if e.is_connect() {
                "Could not connect to Groq API. Check your internet connection.".into()
            } else {
                format!("Network error while contacting Groq API: {}", e).into()
            }
        })?;

    let status = response.status();

    if !status.is_success() {
        let err_text = response.text().await.unwrap_or_default();
        let parsed_message = serde_json::from_str::<GroqErrorBody>(&err_text)
            .map(|b| b.error.message)
            .unwrap_or_else(|_| err_text.clone());

        if status.as_u16() == 401 {
            // The stored key is invalid/expired. Clear it so the *next* run
            // re-prompts instead of silently retrying the same bad key forever.
            let _ = clear_groq_key();
            return Err(format!(
                "Groq API rejected the API key (401 Unauthorized): {}. The saved key has been cleared - run the command again to enter a new one.",
                parsed_message
            )
            .into());
        }

        if status.as_u16() == 429 {
            return Err(format!(
                "Groq API rate limit exceeded: {}. Wait a bit and try again.",
                parsed_message
            )
            .into());
        }

        return Err(format!("Groq API error ({}): {}", status, parsed_message).into());
    }

    let raw_body = response.text().await?;
    let groq_res: GroqResponse = serde_json::from_str(&raw_body).map_err(|e| {
        format!(
            "Failed to parse Groq API response ({}). Raw response: {}",
            e, raw_body
        )
    })?;

    let summary = groq_res
        .choices
        .first()
        .map(|c| c.message.content.clone())
        .ok_or("Groq API returned no choices in its response.")?;

    if summary.trim().is_empty() {
        return Err("Groq API returned an empty summary.".into());
    }

    Ok(summary)
}
