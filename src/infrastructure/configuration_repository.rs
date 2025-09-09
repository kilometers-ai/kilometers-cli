use crate::domain::auth::Configuration;
use anyhow::Result; // Flexible error handling (like exceptions but explicit)
use std::fs; // File system operations
use std::path::PathBuf;

// Repository pattern - encapsulates data access logic
pub struct ConfigurationRepository {
    config_dir: String, // Directory path stored as owned String
}

impl Default for ConfigurationRepository {
    fn default() -> Self {
        Self::new()
    }
}

impl ConfigurationRepository {
    // Default constructor using standard config directory
    pub fn new() -> Self {
        let config_dir = dirs::config_dir()
            .map(|dir| dir.join("km"))
            .unwrap_or_else(|| PathBuf::from(".").join(".config").join("km"));

        Self {
            config_dir: config_dir.to_string_lossy().to_string(),
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
        // Create config directory if it doesn't exist
        fs::create_dir_all(&self.config_dir)?;

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
        format!("{}/config.json", self.config_dir)
    }
}
