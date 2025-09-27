use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub api_key: String,
    pub api_url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_tier: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ConfigEnv {
    pub km_api_key: Option<String>,
    pub km_api_url: Option<String>,
    pub km_default_tier: Option<String>,
}

impl Config {
    pub fn load(path: &Path) -> Result<Self> {
        let contents = fs::read_to_string(path).context("Failed to read config file")?;

        serde_json::from_str(&contents).context("Failed to parse config file")
    }

    pub fn load_with_env(path: &Path) -> Result<Self> {
        // Load .env file if it exists (optional)
        if let Ok(dotenv_path) = std::env::current_dir().map(|mut p| {
            p.push(".env");
            p
        }) {
            if dotenv_path.exists() {
                dotenv::from_path(&dotenv_path).ok();
            }
        }

        // Try to load environment variables first
        let env_config = envy::from_env::<ConfigEnv>().ok();

        // Try to load base config from JSON file, or create from env vars
        let mut config = if path.exists() {
            Self::load(path)?
        } else if let Some(ref env) = env_config {
            // Create config from environment variables if file doesn't exist
            let api_key = env
                .km_api_key
                .as_ref()
                .context("No config file found and KM_API_KEY not set")?
                .clone();
            let api_url = env
                .km_api_url.clone()
                .unwrap_or_else(|| "https://api.kilometers.ai".to_string());

            Self {
                api_key,
                api_url,
                default_tier: env.km_default_tier.clone(),
            }
        } else {
            return Err(anyhow::anyhow!(
                "No config file found and no environment variables set"
            ));
        };

        // Override config file values with environment variables if present
        if let Some(env_config) = env_config {
            if let Some(api_key) = env_config.km_api_key {
                config.api_key = api_key;
            }
            if let Some(api_url) = env_config.km_api_url {
                config.api_url = api_url;
            }
            if env_config.km_default_tier.is_some() {
                config.default_tier = env_config.km_default_tier;
            }
        }

        Ok(config)
    }

    pub fn save(&self, path: &Path) -> Result<()> {
        let contents = serde_json::to_string_pretty(self).context("Failed to serialize config")?;

        fs::write(path, contents).context("Failed to write config file")?;

        Ok(())
    }

    pub fn new(api_key: String, api_url: String) -> Self {
        Self {
            api_key,
            api_url,
            default_tier: None,
        }
    }

    pub fn exists(path: &Path) -> bool {
        path.exists()
    }
}
