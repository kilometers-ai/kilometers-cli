use crate::auth::JwtToken;
use anyhow::{Context, Result};
use keyring::Entry;

const SERVICE_NAME: &str = "ai.kilometers.km";
const ACCESS_TOKEN_KEY: &str = "km-access-token";
const REFRESH_TOKEN_KEY: &str = "km-refresh-token";

/// Check if running in a CI environment where keyring access may not be available
fn is_ci_environment() -> bool {
    std::env::var("CI").is_ok() || std::env::var("GITHUB_ACTIONS").is_ok()
}

pub struct KeyringTokenStore {
    access_token_entry: Entry,
    refresh_token_entry: Entry,
}

impl KeyringTokenStore {
    pub fn new() -> Result<Self> {
        // In CI environments, keyring access may hang or fail
        // Return an error that can be handled gracefully
        if is_ci_environment() {
            tracing::debug!("Running in CI environment, keyring operations will be skipped");
            return Err(anyhow::anyhow!("Keyring not available in CI environment"));
        }

        let access_token_entry = Entry::new(SERVICE_NAME, ACCESS_TOKEN_KEY)
            .context("Failed to create keyring entry for access token")?;

        let refresh_token_entry = Entry::new(SERVICE_NAME, REFRESH_TOKEN_KEY)
            .context("Failed to create keyring entry for refresh token")?;

        Ok(Self {
            access_token_entry,
            refresh_token_entry,
        })
    }

    pub fn save_tokens(&self, access_token: &JwtToken, refresh_token: Option<&str>) -> Result<()> {
        // Serialize access token to JSON
        let access_token_json =
            serde_json::to_string(access_token).context("Failed to serialize access token")?;

        // Save access token to keyring
        self.access_token_entry
            .set_password(&access_token_json)
            .context("Failed to save access token to keyring")?;

        tracing::debug!("Access token saved to keyring");

        // Save refresh token if provided and non-empty
        // Note: Linux and Windows keyring implementations reject empty strings
        if let Some(refresh) = refresh_token {
            if !refresh.is_empty() {
                self.refresh_token_entry
                    .set_password(refresh)
                    .context("Failed to save refresh token to keyring")?;
                tracing::debug!("Refresh token saved to keyring");
            } else {
                tracing::debug!("Skipping empty refresh token (not supported by keyring)");
            }
        }

        Ok(())
    }

    pub fn load_access_token(&self) -> Result<JwtToken> {
        let access_token_json = self
            .access_token_entry
            .get_password()
            .context("Failed to retrieve access token from keyring")?;

        let token: JwtToken = serde_json::from_str(&access_token_json)
            .context("Failed to deserialize access token")?;

        Ok(token)
    }

    #[allow(dead_code)]
    pub fn load_refresh_token(&self) -> Result<Option<String>> {
        match self.refresh_token_entry.get_password() {
            Ok(refresh_token) => Ok(Some(refresh_token)),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(e) => Err(anyhow::anyhow!("Failed to retrieve refresh token: {}", e)),
        }
    }

    pub fn clear_tokens(&self) -> Result<()> {
        // Try to delete access token
        match self.access_token_entry.delete_credential() {
            Ok(_) => tracing::info!("Access token cleared from keyring"),
            Err(keyring::Error::NoEntry) => {
                tracing::debug!("No access token found in keyring to clear")
            }
            Err(e) => return Err(anyhow::anyhow!("Failed to clear access token: {}", e)),
        }

        // Try to delete refresh token
        match self.refresh_token_entry.delete_credential() {
            Ok(_) => tracing::info!("Refresh token cleared from keyring"),
            Err(keyring::Error::NoEntry) => {
                tracing::debug!("No refresh token found in keyring to clear")
            }
            Err(e) => return Err(anyhow::anyhow!("Failed to clear refresh token: {}", e)),
        }

        Ok(())
    }

    pub fn token_exists(&self) -> bool {
        self.access_token_entry.get_password().is_ok()
    }
}

// Note: Default implementation removed as it would panic in CI environments
// Use KeyringTokenStore::new() directly and handle the Result
