use anyhow::{Context, Result};
use std::fs;
use std::path::{Path, PathBuf};

use crate::auth::{self, AuthClient, JwtToken};
use crate::config::Config;
use crate::filters::event_sender::EventSenderFilter;
use crate::filters::local_logger::LocalLoggerFilter;
use crate::filters::risk_analysis::RiskAnalysisFilter;
use crate::filters::{FilterPipeline, ProxyContext, ProxyRequest};
use crate::keyring_token_store::KeyringTokenStore;
use crate::proxy;

pub async fn handle_init(
    config_path: &PathBuf,
    api_key: Option<String>,
    api_url: String,
) -> Result<()> {
    if Config::exists(config_path) {
        println!("Configuration already exists at {:?}", config_path);
        print!("Overwrite? (y/N): ");
        std::io::Write::flush(&mut std::io::stdout())?;

        let mut input = String::new();
        std::io::stdin().read_line(&mut input)?;
        if !input.trim().eq_ignore_ascii_case("y") {
            println!("Cancelled.");
            return Ok(());
        }
    }

    // Load .env file if it exists
    if let Ok(dotenv_path) = std::env::current_dir().map(|mut p| {
        p.push(".env");
        p
    }) {
        if dotenv_path.exists() {
            dotenvy::from_path(&dotenv_path).ok();
        }
    }

    // Check for KM_API_URL environment variable (highest priority)
    let api_url = std::env::var("KM_API_URL").unwrap_or(api_url);

    let api_key = api_key
        .or_else(|| {
            print!("Enter your API key: ");
            std::io::Write::flush(&mut std::io::stdout()).ok();
            let mut input = String::new();
            std::io::stdin().read_line(&mut input).ok();
            Some(input.trim().to_string())
        })
        .context("API key is required")?;

    // Validate API key by exchanging for JWT
    println!("Validating API key...");
    let auth_client = auth::AuthClient::new(api_key.clone(), api_url.clone());

    match auth_client.exchange_for_jwt().await {
        Ok(jwt_token) => {
            println!("✓ Authentication successful");

            // Store tokens in keyring
            tracing::debug!(
                "JWT token to save: token_len={}, expires_at={}, has_refresh={}",
                jwt_token.token.len(),
                jwt_token.expires_at,
                jwt_token.refresh_token.is_some()
            );
            tracing::debug!(
                "JWT claims: user_id={:?}, tier={:?}",
                jwt_token.claims.user_id,
                jwt_token.claims.tier
            );

            match KeyringTokenStore::new() {
                Ok(token_store) => {
                    let refresh_token = jwt_token.refresh_token.as_deref();

                    // Try to serialize to see what will be saved
                    match serde_json::to_string(&jwt_token) {
                        Ok(serialized) => {
                            tracing::debug!("Serialized JWT token: {}", serialized);
                        }
                        Err(e) => {
                            tracing::error!("Failed to serialize JWT token: {}", e);
                        }
                    }

                    if let Err(e) = token_store.save_tokens(&jwt_token, refresh_token) {
                        tracing::error!("Failed to save tokens to keyring: {}", e);
                        println!("⚠ Warning: Could not save tokens to keyring: {}", e);
                        println!("  Tokens will be refreshed on first use");
                    } else {
                        tracing::info!("Successfully saved tokens to keyring");
                        println!("✓ Tokens stored securely in keyring");

                        // Immediately verify we can read it back
                        match token_store.load_access_token() {
                            Ok(loaded_token) => {
                                tracing::debug!("Verified token can be loaded back from keyring");
                                tracing::debug!(
                                    "Loaded token: user_id={:?}, tier={:?}",
                                    loaded_token.claims.user_id,
                                    loaded_token.claims.tier
                                );
                            }
                            Err(e) => {
                                tracing::error!("Failed to load token back from keyring: {}", e);
                                println!("⚠ Warning: Token was saved but cannot be loaded: {}", e);
                            }
                        }
                    }
                }
                Err(e) => {
                    tracing::error!("Failed to initialize keyring: {}", e);
                    println!("⚠ Warning: Could not initialize keyring: {}", e);
                    println!("  Tokens will be refreshed on first use");
                }
            }

            // Save config only after successful authentication
            let config = Config::new(api_key, api_url);
            config.save(config_path)?;
            println!("✓ Configuration saved to {:?}", config_path);

            // Display user info if available
            if let Some(user_id) = &jwt_token.claims.user_id {
                println!("✓ Authenticated as user: {}", user_id);
            }
            if let Some(tier) = &jwt_token.claims.tier {
                println!("✓ User tier: {}", tier);
            }

            Ok(())
        }
        Err(e) => {
            println!("✗ Authentication failed: {}", e);
            println!("\nPlease check:");
            println!("  • Your API key is correct");
            println!("  • You have network connectivity");
            println!("  • The API URL is correct: {}", api_url);
            Err(anyhow::anyhow!(
                "Failed to authenticate with provided API key"
            ))
        }
    }
}

pub async fn get_jwt_token_with_cache(api_key: String, api_url: String) -> Option<JwtToken> {
    let token_store = match KeyringTokenStore::new() {
        Ok(store) => store,
        Err(e) => {
            tracing::warn!("Failed to initialize keyring token store: {}", e);
            return None;
        }
    };

    // Try to load from keyring first
    if token_store.token_exists() {
        if let Ok(cached_token) = token_store.load_access_token() {
            if !auth::AuthClient::is_token_expired(&cached_token) {
                tracing::debug!("Using cached JWT token from keyring");
                return Some(cached_token);
            } else {
                tracing::debug!("Cached token expired, fetching new token");
            }
        } else {
            tracing::debug!("Failed to load cached token from keyring, fetching new token");
        }
    } else {
        tracing::debug!("No token found in keyring, fetching new token");
    }

    // Exchange for new token
    let auth_client = auth::AuthClient::new(api_key, api_url);
    match auth_client.exchange_for_jwt().await {
        Ok(new_token) => {
            // Save to keyring
            let refresh_token = new_token.refresh_token.as_deref();
            if let Err(e) = token_store.save_tokens(&new_token, refresh_token) {
                tracing::warn!(
                    "Failed to save token to keyring: {} - continuing without keyring",
                    e
                );
            } else {
                tracing::debug!("Token saved to keyring successfully");
            }
            Some(new_token)
        }
        Err(e) => {
            tracing::warn!(
                "Authentication failed: {} - continuing in local-only mode",
                e
            );
            None
        }
    }
}

pub async fn handle_monitor(
    config_path: &Path,
    args: Vec<String>,
    local_only: bool,
    override_tier: Option<String>,
    log_file: PathBuf,
) -> Result<()> {
    if args.is_empty() {
        return Err(anyhow::anyhow!("No command provided to proxy"));
    }

    let program = args[0].clone();
    let program_args = args[1..].to_vec();

    tracing::info!("Proxying command: {} {:?}", program, program_args);

    // Load config with environment variable support, but gracefully handle missing config
    let default_api_url = "https://api.kilometers.ai".to_string();
    let (jwt_token_option, api_url) = if local_only {
        tracing::info!("Running in local-only mode - skipping authentication");
        (None, default_api_url)
    } else {
        match Config::load_with_env(config_path) {
            Ok(config) => {
                let (api_key, api_url) = (config.api_key, config.api_url.clone());
                let token = get_jwt_token_with_cache(api_key, api_url.clone()).await;
                (token, api_url)
            }
            Err(e) => {
                tracing::info!("No configuration found - running in local-only mode. Use 'km init' to set up cloud features.");
                tracing::debug!("Config load error: {}", e);
                (None, default_api_url)
            }
        }
    };

    let (user_tier, jwt_token) = if let Some(token) = jwt_token_option {
        let tier = override_tier
            .as_deref()
            .or(token.claims.tier.as_deref())
            .unwrap_or("free");
        tracing::info!("User tier: {}", tier);
        (tier.to_string(), Some(token))
    } else {
        tracing::info!("Authentication failed - running in local-only mode");
        ("free".to_string(), None)
    };

    let proxy_request = ProxyRequest::new(program.clone(), program_args.clone());
    let proxy_context = ProxyContext::new(
        proxy_request,
        jwt_token
            .as_ref()
            .map(|t| t.token.clone())
            .unwrap_or_default(),
    );

    let pipeline = if local_only || jwt_token.is_none() {
        if local_only {
            tracing::info!("Using local logging only (--local-only specified)");
        } else {
            tracing::info!("Using local logging only (authentication failed)");
        }
        // Use separate log file for command metadata vs MCP traffic
        let metadata_log = log_file
            .parent()
            .unwrap_or_else(|| std::path::Path::new("."))
            .join("km_commands.log");
        FilterPipeline::new().add_filter(Box::new(LocalLoggerFilter::new(metadata_log)))
    } else if let Some(token) = jwt_token.clone() {
        tracing::info!(
            "Using filter pipeline with telemetry for {} tier",
            user_tier
        );
        let mut pipeline = FilterPipeline::new()
            .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
            .add_filter(Box::new(EventSenderFilter::new(
                format!("{}/api/events/telemetry", api_url),
                token.clone(),
            )));

        if user_tier != "free" {
            tracing::info!("Adding risk analysis for paid tier user");
            pipeline = pipeline.add_filter(Box::new(RiskAnalysisFilter::new(
                format!("{}/api/risk/analyze", api_url),
                0.8,
            )));
        }

        pipeline
    } else {
        // Fallback case (should not happen but be safe)
        tracing::info!("Fallback to local logging only");
        // Use separate log file for command metadata vs MCP traffic
        let metadata_log = log_file
            .parent()
            .unwrap_or_else(|| std::path::Path::new("."))
            .join("km_commands.log");
        FilterPipeline::new().add_filter(Box::new(LocalLoggerFilter::new(metadata_log)))
    };

    match pipeline.execute(proxy_context).await {
        Ok(filtered_request) => {
            tracing::info!("Request approved, executing proxy");
            proxy::run_proxy(&filtered_request.command, &filtered_request.args, &log_file)?;
        }
        Err(e) => {
            return Err(anyhow::anyhow!("Request blocked: {}", e));
        }
    }

    Ok(())
}

pub fn handle_clear_logs(include_config: bool, config_path: &Path) -> Result<()> {
    let log_files = vec!["mcp_traffic.jsonl", "mcp_requests.log", "mcp_proxy.log"];

    for file in log_files {
        if PathBuf::from(file).exists() {
            fs::remove_file(file)?;
            println!("✓ Deleted {}", file);
        }
    }

    if include_config && config_path.exists() {
        fs::remove_file(config_path)?;
        println!("✓ Deleted config at {:?}", config_path);
    }

    // Clear tokens from keyring
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            if let Err(e) = token_store.clear_tokens() {
                tracing::warn!("Failed to clear tokens from keyring: {}", e);
            } else {
                println!("✓ Cleared tokens from keyring");
            }
        }
    }

    println!("All logs cleared.");
    Ok(())
}

pub fn handle_show_config(config_path: &PathBuf, show_secrets: bool) -> Result<()> {
    if !Config::exists(config_path) {
        println!("No configuration found. Run 'km init' to create one.");
        return Ok(());
    }

    let config = Config::load(config_path)?;

    println!("Configuration at {:?}:", config_path);
    println!("  API URL: {}", config.api_url);

    if show_secrets {
        println!("  API Key: {}", config.api_key);
    } else {
        let masked = if config.api_key.len() > 8 {
            format!(
                "{}...{}",
                &config.api_key[..4],
                &config.api_key[config.api_key.len() - 4..]
            )
        } else {
            "****".to_string()
        };
        println!("  API Key: {} (use --show-secrets to reveal)", masked);
    }

    if let Some(tier) = &config.default_tier {
        println!("  Default Tier: {}", tier);
    }

    Ok(())
}

pub fn handle_logs(
    file: PathBuf,
    requests: bool,
    responses: bool,
    method: Option<String>,
    tail: bool,
    lines: Option<usize>,
) -> Result<()> {
    if !file.exists() {
        return Err(anyhow::anyhow!("Log file {:?} not found", file));
    }

    let contents = fs::read_to_string(&file)?;
    let all_lines: Vec<&str> = contents.lines().collect();

    let filtered_lines: Vec<&str> = all_lines
        .iter()
        .filter(|line| {
            if let Ok(json) = serde_json::from_str::<serde_json::Value>(line) {
                // Filter by direction
                if requests && json.get("direction") == Some(&serde_json::json!("response")) {
                    return false;
                }
                if responses && json.get("direction") == Some(&serde_json::json!("request")) {
                    return false;
                }

                // Filter by method
                if let Some(ref m) = method {
                    if let Some(content) = json.get("content").and_then(|c| c.as_str()) {
                        if let Ok(rpc) = serde_json::from_str::<serde_json::Value>(content) {
                            if rpc.get("method").and_then(|v| v.as_str()) != Some(m) {
                                return false;
                            }
                        }
                    }
                }

                true
            } else {
                false
            }
        })
        .copied()
        .collect();

    let display_lines = if let Some(n) = lines {
        filtered_lines.iter().rev().take(n).rev().copied().collect()
    } else {
        filtered_lines
    };

    for line in display_lines {
        if let Ok(json) = serde_json::from_str::<serde_json::Value>(line) {
            println!("{}", serde_json::to_string_pretty(&json)?);
        }
    }

    if tail {
        println!("\nTail mode not yet implemented.");
    }

    Ok(())
}

pub fn handle_doctor_jwt() -> Result<()> {
    println!("JWT Token Information:");
    println!();

    // Try to initialize keyring token store
    let token_store = match KeyringTokenStore::new() {
        Ok(store) => {
            tracing::debug!("Keyring token store initialized successfully");
            store
        }
        Err(e) => {
            tracing::error!("Failed to initialize keyring: {}", e);
            println!("✗ Failed to initialize keyring: {}", e);
            println!("\nThe keyring may not be available on this system.");
            println!("Run 'km init' to configure authentication.");
            return Ok(());
        }
    };

    // Check if token exists
    let exists = token_store.token_exists();
    tracing::debug!("Token exists in keyring: {}", exists);

    if !exists {
        println!("✗ No JWT token found in keyring");
        println!("\nRun 'km init' to authenticate and store a token.");

        // Try to get raw password to debug
        tracing::debug!("Attempting to read raw keyring entry for debugging...");
        return Ok(());
    }

    // Load the token
    tracing::debug!("Attempting to load access token from keyring...");
    match token_store.load_access_token() {
        Ok(jwt_token) => {
            // Display token (truncated for security)
            let token_display = if jwt_token.token.len() > 40 {
                format!(
                    "{}...{}",
                    &jwt_token.token[..20],
                    &jwt_token.token[jwt_token.token.len() - 20..]
                )
            } else {
                jwt_token.token.clone()
            };
            println!("  Token: {}", token_display);

            // Display expiration
            use std::time::{SystemTime, UNIX_EPOCH};
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();

            let is_expired = AuthClient::is_token_expired(&jwt_token);
            let expires_in = jwt_token.expires_at.saturating_sub(now);

            let expires_at_str = chrono::DateTime::from_timestamp(jwt_token.expires_at as i64, 0)
                .map(|dt| dt.format("%Y-%m-%d %H:%M:%S UTC").to_string())
                .unwrap_or_else(|| "Invalid timestamp".to_string());

            println!(
                "  Expires At: {} (in {} seconds)",
                expires_at_str, expires_in
            );

            if is_expired {
                println!("  Status: ✗ EXPIRED");
            } else {
                println!("  Status: ✓ Valid");
            }

            println!();
            println!("  Claims:");

            if let Some(user_id) = &jwt_token.claims.user_id {
                println!("    User ID: {}", user_id);
            }

            if let Some(tier) = &jwt_token.claims.tier {
                println!("    Tier: {}", tier);
            }

            if let Some(sub) = &jwt_token.claims.sub {
                println!("    Subject: {}", sub);
            }

            if let Some(iat) = jwt_token.claims.iat {
                let iat_str = chrono::DateTime::from_timestamp(iat as i64, 0)
                    .map(|dt| dt.format("%Y-%m-%d %H:%M:%S UTC").to_string())
                    .unwrap_or_else(|| "Invalid timestamp".to_string());
                println!("    Issued At: {}", iat_str);
            }

            println!();

            if is_expired {
                println!("⚠ Token is expired. Run 'km monitor' to automatically refresh it,");
                println!("  or run 'km init' to re-authenticate.");
            }

            Ok(())
        }
        Err(e) => {
            tracing::error!("Failed to load JWT token from keyring: {}", e);
            println!("✗ Failed to load JWT token: {}", e);
            println!("\nThe token may be corrupted. Run 'km init' to re-authenticate.");
            Ok(())
        }
    }
}
