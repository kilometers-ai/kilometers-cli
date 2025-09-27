use anyhow::{Context, Result};
use clap::Parser;

mod auth;
mod cli;
mod config;
mod filters;
mod proxy;
mod token_cache;

use cli::{Cli, Commands};
use config::Config;
use filters::event_sender::EventSenderFilter;
use filters::local_logger::LocalLoggerFilter;
use filters::risk_analysis::RiskAnalysisFilter;
use filters::{FilterPipeline, ProxyContext, ProxyRequest};
use std::fs;
use std::path::{Path, PathBuf};
use token_cache::TokenCache;

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    // Initialize logging with verbosity level
    tracing_subscriber::fmt()
        .with_max_level(cli.get_log_level())
        .init();

    tracing::debug!("Starting km cli with command: {:?}", cli.command);

    match cli.command {
        Commands::Init { api_key, api_url } => handle_init(&cli.config, api_key, api_url).await?,
        Commands::Monitor {
            args,
            local_only,
            override_tier,
            log_file,
        } => handle_monitor(&cli.config, args, local_only, override_tier, log_file).await?,
        Commands::ClearLogs { include_config } => handle_clear_logs(include_config, &cli.config)?,
        Commands::Config { show_secrets } => handle_show_config(&cli.config, show_secrets)?,
        Commands::Logs {
            file,
            requests,
            responses,
            method,
            tail,
            lines,
        } => handle_logs(file, requests, responses, method, tail, lines)?,
    }

    Ok(())
}

async fn handle_init(
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

    let api_key = api_key
        .or_else(|| {
            print!("Enter your API key: ");
            std::io::Write::flush(&mut std::io::stdout()).ok();
            let mut input = String::new();
            std::io::stdin().read_line(&mut input).ok();
            Some(input.trim().to_string())
        })
        .context("API key is required")?;

    let config = Config::new(api_key, api_url);
    config.save(config_path)?;

    println!("✓ Configuration saved to {:?}", config_path);
    Ok(())
}

async fn get_jwt_token_with_cache(api_key: String, api_url: String) -> Option<auth::JwtToken> {
    let token_cache = match TokenCache::new() {
        Ok(cache) => cache,
        Err(e) => {
            tracing::warn!("Failed to initialize token cache: {}", e);
            return None;
        }
    };

    // Try to load from cache first
    if token_cache.cache_exists() {
        if let Ok(cached_token) = token_cache.load_token() {
            if !auth::AuthClient::is_token_expired(&cached_token) {
                tracing::debug!("Using cached JWT token");
                return Some(cached_token);
            } else {
                tracing::debug!("Cached token expired, fetching new token");
            }
        } else {
            tracing::debug!("Failed to load cached token, fetching new token");
        }
    } else {
        tracing::debug!("No token cache found, fetching new token");
    }

    // Exchange for new token
    let auth_client = auth::AuthClient::new(api_key, api_url);
    match auth_client.exchange_for_jwt().await {
        Ok(new_token) => {
            // Save to cache
            if let Err(e) = token_cache.save_token(&new_token) {
                tracing::warn!("Failed to cache token: {} - continuing without cache", e);
            } else {
                tracing::debug!("Token cached successfully");
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

async fn handle_monitor(
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

fn handle_clear_logs(include_config: bool, config_path: &Path) -> Result<()> {
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

    // Clear token cache
    if let Ok(token_cache) = TokenCache::new() {
        if token_cache.cache_exists() {
            if let Err(e) = token_cache.clear_cache() {
                tracing::warn!("Failed to clear token cache: {}", e);
            } else {
                println!("✓ Cleared token cache");
            }
        }
    }

    println!("All logs cleared.");
    Ok(())
}

fn handle_show_config(config_path: &PathBuf, show_secrets: bool) -> Result<()> {
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

fn handle_logs(
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
