use colored::Colorize;
use dialoguer::Password;
use keyring::Entry;
use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::io;
use std::path::PathBuf;

const KEYRING_SERVICE: &str = "notes-cli";
const KEYRING_USER: &str = "groq_api_key";

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub destination_path: String,
}

fn get_config_dir() -> io::Result<PathBuf> {
    let mut path = dirs::config_dir().ok_or_else(|| {
        io::Error::new(
            io::ErrorKind::NotFound,
            "Could not find system config directory",
        )
    })?;
    path.push("notes-cli");
    fs::create_dir_all(&path)?;
    Ok(path)
}

pub fn get_config_path() -> io::Result<PathBuf> {
    let mut path = get_config_dir()?;
    path.push("config.json");
    Ok(path)
}

pub fn save_config(config: &Config) -> io::Result<()> {
    let config_path = get_config_path()?;
    let json = serde_json::to_string_pretty(&config)?;
    fs::write(config_path, json)?;
    Ok(())
}

pub fn load_config() -> Config {
    let default_config = Config {
        destination_path: "D:/Notes".to_string(),
    };

    let Ok(config_path) = get_config_path() else {
        return default_config;
    };
    let Ok(content) = fs::read_to_string(config_path) else {
        return default_config;
    };
    serde_json::from_str(&content).unwrap_or(default_config)
}

const MAX_KEY_ATTEMPTS: u8 = 3;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum KeySource {
    Env,
    Keychain,
    Prompt,
}

pub fn get_or_prompt_groq_key() -> Result<(String, KeySource), Box<dyn std::error::Error>> {
    if let Ok(env_key) = env::var("GROQ_API_KEY") {
        if !env_key.trim().is_empty() {
            return Ok((env_key.trim().to_string(), KeySource::Env));
        }
    }

    let keyring_entry = Entry::new(KEYRING_SERVICE, KEYRING_USER).ok();

    if let Some(entry) = &keyring_entry {
        if let Ok(stored_key) = entry.get_password() {
            let trimmed_stored = stored_key.trim().to_string();
            if !trimmed_stored.is_empty() {
                return Ok((trimmed_stored, KeySource::Keychain));
            }
        }
    }

    for attempt in 1..=MAX_KEY_ATTEMPTS {
        let prompt_result = Password::new()
            .with_prompt("Enter your Groq API key")
            .interact();

        let raw_key = match prompt_result {
            Ok(key) => key,

            Err(e) => {
                return Err(format!("Could not read API key from input (is this running in a non-interactive shell?): {}",e).into());
            }
        };

        let trimmed_key = raw_key.trim().to_string();

        if trimmed_key.is_empty() {
            eprintln!(
                "{}",
                format!("API key cannot be empty. ({}/{} attempts)", attempt, MAX_KEY_ATTEMPTS).red()
            );

            continue;
        }

        if !trimmed_key.starts_with("gsk_") {
            eprintln!(
                "{}","Warning: Groq API keys usually start with 'gsk_'. Continuing anyway.".yellow()
            );
        }

        return Ok((trimmed_key, KeySource::Prompt));
    }

    Err(format!(
        "No valid API key provided after {} attempts.",
        MAX_KEY_ATTEMPTS
    ).into())
}

pub fn persist_verified_key(key: &str) -> Result<(), Box<dyn std::error::Error>> {
    let keyring_entry = Entry::new(KEYRING_SERVICE, KEYRING_USER)?;
    keyring_entry.set_password(key)?;
    println!(
        "{}",
        "Groq API key verified and securely saved to OS Keychain/Credential Manager!".green()
    );
    Ok(())
}

pub fn set_groq_key(key: &str) -> Result<(), Box<dyn std::error::Error>> {
    let trimmed_key = key.trim();

    if trimmed_key.is_empty() {
        return Err("API key cannot be empty.".into());
    }

    if !trimmed_key.starts_with("gsk_") {
        eprintln!(
            "{}",
            "Warning: Groq API keys usually start with 'gsk_'. Saving anyway.".yellow()
        );
    }

    let keyring_entry = Entry::new(KEYRING_SERVICE, KEYRING_USER)?;
    keyring_entry.set_password(trimmed_key)?;
    println!(
        "{}",
        "Groq API key securely saved to OS Keychain/Credential Manager!".green()
    );

    Ok(())
}

pub fn clear_groq_key() -> Result<(), Box<dyn std::error::Error>> {
    let keyring_entry = Entry::new(KEYRING_SERVICE, KEYRING_USER)?;
    let _ = keyring_entry.delete_password();
    Ok(())
}