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

/// Where a returned API key came from, so callers know whether it's already
/// been persisted (and whether it's safe to clear from the keychain if it
/// turns out to be bad).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum KeySource {
    Env,
    Keychain,
}

/// Look for an already-available key (env var or keychain) without ever
/// prompting. Returns None if neither has one, so the caller knows it needs
/// to fall back to interactive prompting.
pub fn find_cached_key() -> Option<(String, KeySource)> {
    if let Ok(env_key) = env::var("GROQ_API_KEY") {
        let trimmed = env_key.trim().to_string();
        if !trimmed.is_empty() {
            return Some((trimmed, KeySource::Env));
        }
    }

    let keyring_entry = Entry::new(KEYRING_SERVICE, KEYRING_USER).ok()?;
    let stored_key = keyring_entry.get_password().ok()?;
    let trimmed_stored = stored_key.trim().to_string();

    if trimmed_stored.is_empty() {
        None
    } else {
        Some((trimmed_stored, KeySource::Keychain))
    }
}

pub fn prompt_once(attempt: u32, max_attempts: u32) -> Result<Option<String>, Box<dyn std::error::Error>> {
    let prompt_result = Password::new()
        .with_prompt(format!("Enter your Groq API key (attempt {}/{})", attempt, max_attempts))
        .interact();

    let raw_key = match prompt_result {
        Ok(key) => key,
        Err(e) => {
            return Err(format!(
                "Could not read API key from input (is this running in a non-interactive shell?): {}",
                e
            ).into());
        }
    };

    let trimmed_key = raw_key.trim().to_string();

    if trimmed_key.is_empty() {
        eprintln!(
            "{}",
            format!("API key cannot be empty. ({}/{} attempts used)", attempt, max_attempts).red()
        );
        return Ok(None);
    }

    if !trimmed_key.starts_with("gsk_") {
        eprintln!(
            "{}",
            "Warning: Groq API keys usually start with 'gsk_'. Continuing anyway.".yellow()
        );
    }

    Ok(Some(trimmed_key))
}

/// Save a key to the OS keychain that has just been confirmed to work
/// against the Groq API. Only call this after a successful API response.
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

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;
    use tempfile::tempdir;

    #[test]
    fn load_config_falls_back_to_default_when_file_missing() {
        let dir = tempdir().unwrap();
        let missing_path = dir.path().join("does-not-exist").join("config.json");
        assert!(fs::read_to_string(&missing_path).is_err());

        let config = load_config();
        assert!(!config.destination_path.is_empty());
    }

    #[test]
    fn save_and_load_config_roundtrip_via_serde() {
        let config = Config {
            destination_path: "Z:/CustomNotes".to_string(),
        };
        let json = serde_json::to_string_pretty(&config).unwrap();
        let restored: Config = serde_json::from_str(&json).unwrap();
        assert_eq!(restored.destination_path, "Z:/CustomNotes");
    }

    #[test]
    fn load_config_falls_back_on_corrupted_json() {
        let corrupted = "{ this is not valid json";
        let default_config = Config {
            destination_path: "D:/Notes".to_string(),
        };
        let result: Config = serde_json::from_str(corrupted).unwrap_or(default_config);
        assert_eq!(result.destination_path, "D:/Notes");
    }

    #[test]
    #[serial]
    fn find_cached_key_prefers_env_var_when_set() {
        unsafe { env::set_var("GROQ_API_KEY", "  gsk_from_env  "); }
        let found = find_cached_key();
        unsafe { env::remove_var("GROQ_API_KEY"); }

        assert_eq!(found, Some(("gsk_from_env".to_string(), KeySource::Env)));
    }

    #[test]
    #[serial]
    fn find_cached_key_ignores_empty_env_var_and_falls_through() {
        unsafe { env::set_var("GROQ_API_KEY", "   "); }
        let found = find_cached_key();
        unsafe { env::remove_var("GROQ_API_KEY"); }

        // With an empty env var, lookup must not report the empty value as a
        // usable env-sourced key.
        if let Some((_, source)) = found {
            assert_ne!(source, KeySource::Env);
        }
    }

    #[test]
    #[serial]
    fn find_cached_key_trims_whitespace_from_env_var() {
        unsafe { env::set_var("GROQ_API_KEY", "\tgsk_padded\n"); }
        let found = find_cached_key();
        unsafe { env::remove_var("GROQ_API_KEY"); }

        assert_eq!(found, Some(("gsk_padded".to_string(), KeySource::Env)));
    }

    #[test]
    fn get_config_path_ends_with_expected_filename() {
        if let Ok(path) = get_config_path() {
            assert_eq!(path.file_name().unwrap(), "config.json");
            assert!(path.to_string_lossy().contains("notes-cli"));
        }
    }
}