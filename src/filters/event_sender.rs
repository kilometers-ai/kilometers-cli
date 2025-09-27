use super::{FilterDecision, ProxyContext, ProxyFilter};
use crate::auth::JwtToken;
use anyhow::{Context, Result};
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct EventSenderFilter {
    api_endpoint: String,
    client: reqwest::Client,
    jwt_token: JwtToken,
}

#[derive(Debug, Serialize)]
struct TelemetryEvent {
    event_type: String,
    timestamp: DateTime<Utc>,
    user_id: Option<String>,
    user_tier: String,
    command: String,
    args: Vec<String>,
    session_id: String,
    metadata: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct TelemetryResponse {
    status: String,
    #[allow(dead_code)]
    message: Option<String>,
    events_remaining: Option<u64>,
}

impl EventSenderFilter {
    pub fn new(api_endpoint: String, jwt_token: JwtToken) -> Self {
        Self {
            api_endpoint,
            client: reqwest::Client::new(),
            jwt_token,
        }
    }

    async fn send_telemetry_event(&self, ctx: &ProxyContext) -> Result<()> {
        let session_id = Uuid::new_v4().to_string();

        let event = TelemetryEvent {
            event_type: "command_execution".to_string(),
            timestamp: Utc::now(),
            user_id: self.jwt_token.claims.user_id.clone(),
            user_tier: self
                .jwt_token
                .claims
                .tier
                .as_deref()
                .unwrap_or("free")
                .to_string(),
            command: ctx.request.command.clone(),
            args: ctx.request.args.clone(),
            session_id,
            metadata: ctx
                .request
                .metadata
                .iter()
                .map(|(k, v)| (k.clone(), serde_json::Value::String(v.clone())))
                .collect(),
        };

        let response = self
            .client
            .post(&self.api_endpoint)
            .bearer_auth(&self.jwt_token.token)
            .json(&event)
            .send()
            .await
            .context("Failed to send telemetry event")?;

        match response.status().as_u16() {
            200..=299 => {
                if let Ok(telemetry_response) = response.json::<TelemetryResponse>().await {
                    tracing::info!(
                        "Telemetry event sent successfully: {}",
                        telemetry_response.status
                    );
                    if let Some(remaining) = telemetry_response.events_remaining {
                        tracing::debug!("Events remaining this month: {}", remaining);
                    }
                } else {
                    tracing::info!("Telemetry event sent successfully");
                }
                Ok(())
            }
            429 => {
                tracing::warn!("Rate limit reached for telemetry events - continuing execution");
                Ok(())
            }
            status => Err(anyhow::anyhow!("Telemetry failed with status {}", status)),
        }
    }
}

#[async_trait]
impl ProxyFilter for EventSenderFilter {
    async fn check(&self, ctx: &ProxyContext) -> Result<FilterDecision> {
        match self.send_telemetry_event(ctx).await {
            Ok(_) => {
                tracing::debug!("Telemetry event sent for command: {}", ctx.request.command);
            }
            Err(e) => {
                tracing::warn!(
                    "Failed to send telemetry event: {} - continuing execution",
                    e
                );
            }
        }

        Ok(FilterDecision::Allow)
    }

    fn is_blocking(&self) -> bool {
        false
    }

    fn name(&self) -> &str {
        "EventSender"
    }
}
