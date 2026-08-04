use serde::{Deserialize, Serialize};
use std::env;

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

pub async fn summarize_text(body: &str) -> Result<String, Box<dyn std::error::Error>> {
    let api_key = env::var("GROQ_API_KEY").map_err(|_| {
        "GROQ_API_KEY environment variable not set. Please set it before running summarize."
    })?;

    let client = reqwest::Client::new();
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
        .await?;

    if !response.status().is_success() {
        let err_text = response.text().await?;
        return Err(format!("Groq API error: {}", err_text).into());
    }

    let groq_res: GroqResponse = response.json().await?;
    
    let summary = groq_res
        .choices
        .first()
        .map(|c| c.message.content.clone())
        .ok_or("Empty response choices from Groq API")?;

    Ok(summary)
}