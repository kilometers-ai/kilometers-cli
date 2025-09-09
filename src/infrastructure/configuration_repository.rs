use crate::domain::auth::Configuration;
use anyhow::Result; // Flexible error handling (like exceptions but explicit)
use std::fs; // File system operations
#[cfg(test)]
use std::path::PathBuf;

// Repository pattern - encapsulates data access logic
pub struct ConfigurationRepository {
    config_dir: String, // Directory path stored as owned String
}

impl ConfigurationRepository {
    // Default constructor with hardcoded path (in real app, would be configurable)
    pub fn new() -> Self {
        Self {
            // .to_string() converts string literal (&str) to owned String
            // Like strdup in C or new string(literal) in C#
            config_dir: "/Users/milesangelo/Source/active/kilometers.ai/kilometers-cli-proxy"
                .to_string(),
        }
    }

    #[cfg(test)]
    pub fn with_path(path: PathBuf) -> Self {
        Self {
            config_dir: path
                .parent()
                .map(|p| p.to_string_lossy().to_string())
                .unwrap_or_else(|| ".".to_string()),
        }
    }

    pub fn save_configuration(&self, config: Configuration) -> Result<()> {
        let config_file = self.get_config_path();
        let config_content = format!(r#"{{"api_key": "{}"}}"#, config.api_key);
        fs::write(&config_file, config_content)?;
        Ok(())
    }

    pub fn load_configuration(&self) -> Result<Configuration> {
        let config_file = self.get_config_path();
        let content = fs::read_to_string(&config_file)?;
        let json: serde_json::Value = serde_json::from_str(&content)?;

        let api_key = json["api_key"]
            .as_str()
            .ok_or_else(|| anyhow::anyhow!("API key not found in configuration"))?
            .to_string();

        Ok(Configuration::new(api_key))
    }

    pub fn get_config_path(&self) -> String {
        format!("{}/km_config.json", self.config_dir)
    }
}
