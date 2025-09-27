use super::{FilterDecision, ProxyContext, ProxyFilter};
use anyhow::{Context, Result};
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct LocalLoggerFilter {
    log_path: PathBuf,
}

#[derive(Debug, Serialize, Deserialize)]
struct LogEntry {
    timestamp: DateTime<Utc>,
    command: String,
    args: Vec<String>,
    user_tier: String,
    metadata: serde_json::Value,
}

impl LocalLoggerFilter {
    pub fn new(log_path: PathBuf) -> Self {
        Self { log_path }
    }

    fn log_request(&self, ctx: &ProxyContext, user_tier: &str) -> Result<()> {
        let entry = LogEntry {
            timestamp: Utc::now(),
            command: ctx.request.command.clone(),
            args: ctx.request.args.clone(),
            user_tier: user_tier.to_string(),
            metadata: serde_json::to_value(&ctx.request.metadata)?,
        };

        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.log_path)
            .context("Failed to open log file")?;

        writeln!(file, "{}", serde_json::to_string(&entry)?)
            .context("Failed to write log entry")?;

        Ok(())
    }
}

#[async_trait]
impl ProxyFilter for LocalLoggerFilter {
    async fn check(&self, ctx: &ProxyContext) -> Result<FilterDecision> {
        match self.log_request(ctx, "free") {
            Ok(_) => {
                tracing::debug!("Logged request locally for free tier user");
            }
            Err(e) => {
                tracing::warn!("Failed to log request locally: {}", e);
            }
        }

        Ok(FilterDecision::Allow)
    }

    fn is_blocking(&self) -> bool {
        false
    }

    fn name(&self) -> &str {
        "LocalLogger"
    }
}
