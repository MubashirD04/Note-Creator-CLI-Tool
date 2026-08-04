use serde::{Deserialize, Serialize};
use std::fs;
use std::io;
use std::path::PathBuf;

#[derive(Debug, Serialize, Deserialize)]
struct Config {
    pub destination_path: String,
    pub groq_api_key: Option<String>,
}

pub fn get_config_dir() -> io::Result<PathBuf> {
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
    let config_dir = get_config_dir()?;
    let config_path = config_dir.join("config.json");
    let env_path = config_dir.join(".env");

    let json = serde_json::to_string_pretty(&config)?;
    fs::write(config_path, json)?;

    if let Some(key) = &config.groq_api_key {
        let env_content = format!("GROQ_API_KEY={}\n", key);
        fs::write(env_path, env_content)?;
    }
    Ok(())
}

pub fn load_destination_path() -> Config {
    let default_config = Config {
        destination_path: "D:/Notes".to_string(),
        groq_api_key: None,
    };

    let Ok(config_path) = get_config_path() else {
        return default_config;
    };
    let Ok(content) = fs::read_to_string(config_path) else {
        return default_config;
    };
    serde_json::from_str(&content).unwrap_or(default_config)
}
