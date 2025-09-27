use crate::auth::JwtToken;
use anyhow::{Context, Result};
use directories::ProjectDirs;
use std::fs;
use std::path::PathBuf;

pub struct TokenCache {
    cache_file_path: PathBuf,
}

impl TokenCache {
    pub fn new() -> Result<Self> {
        let proj_dirs = ProjectDirs::from("ai", "kilometers", "km")
            .context("Failed to determine project directories")?;

        let cache_dir = proj_dirs.cache_dir();
        fs::create_dir_all(cache_dir).context("Failed to create cache directory")?;

        let cache_file_path = cache_dir.join("token_cache.json");

        Ok(Self { cache_file_path })
    }

    pub fn load_token(&self) -> Result<JwtToken> {
        let contents =
            fs::read_to_string(&self.cache_file_path).context("Failed to read token cache file")?;

        let token: JwtToken =
            serde_json::from_str(&contents).context("Failed to parse cached token")?;

        Ok(token)
    }

    pub fn save_token(&self, token: &JwtToken) -> Result<()> {
        let contents = serde_json::to_string_pretty(token).context("Failed to serialize token")?;

        // Ensure parent directory exists
        if let Some(parent) = self.cache_file_path.parent() {
            fs::create_dir_all(parent).context("Failed to create token cache directory")?;
        }

        fs::write(&self.cache_file_path, contents).context("Failed to write token cache file")?;

        // Set restrictive permissions (owner read/write only)
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = fs::metadata(&self.cache_file_path)?.permissions();
            perms.set_mode(0o600);
            fs::set_permissions(&self.cache_file_path, perms)
                .context("Failed to set token cache file permissions")?;
        }

        tracing::debug!("Token cached at {:?}", self.cache_file_path);
        Ok(())
    }

    pub fn clear_cache(&self) -> Result<()> {
        if self.cache_file_path.exists() {
            fs::remove_file(&self.cache_file_path).context("Failed to remove token cache file")?;
            tracing::info!("Token cache cleared");
        }
        Ok(())
    }

    pub fn cache_exists(&self) -> bool {
        self.cache_file_path.exists()
    }

    #[allow(dead_code)]
    pub fn get_cache_path(&self) -> &PathBuf {
        &self.cache_file_path
    }
}

impl Default for TokenCache {
    fn default() -> Self {
        Self::new().expect("Failed to initialize token cache")
    }
}
