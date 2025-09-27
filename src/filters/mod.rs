use anyhow::Result;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

pub mod event_sender;
pub mod local_logger;
pub mod risk_analysis;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProxyRequest {
    pub command: String,
    pub args: Vec<String>,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone)]
pub struct ProxyContext {
    pub request: ProxyRequest,
    pub jwt_token: String,
    #[allow(dead_code)]
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum FilterDecision {
    Allow,
    Block { reason: String },
    Transform { new_request: ProxyRequest },
}

#[async_trait]
pub trait ProxyFilter: Send + Sync {
    async fn check(&self, ctx: &ProxyContext) -> Result<FilterDecision>;

    fn is_blocking(&self) -> bool {
        true
    }

    fn name(&self) -> &str;
}

pub struct FilterPipeline {
    filters: Vec<Box<dyn ProxyFilter>>,
}

impl Default for FilterPipeline {
    fn default() -> Self {
        Self::new()
    }
}

impl FilterPipeline {
    pub fn new() -> Self {
        Self {
            filters: Vec::new(),
        }
    }

    pub fn add_filter(mut self, filter: Box<dyn ProxyFilter>) -> Self {
        self.filters.push(filter);
        self
    }

    pub async fn execute(&self, mut ctx: ProxyContext) -> Result<ProxyRequest> {
        for filter in &self.filters {
            let decision = match filter.check(&ctx).await {
                Ok(decision) => decision,
                Err(e) => {
                    if filter.is_blocking() {
                        return Err(anyhow::anyhow!("Filter {} failed: {}", filter.name(), e));
                    } else {
                        tracing::warn!("Non-blocking filter {} failed: {}", filter.name(), e);
                        continue;
                    }
                }
            };

            match decision {
                FilterDecision::Allow => {
                    tracing::debug!("Filter {} allowed request", filter.name());
                    continue;
                }
                FilterDecision::Block { reason } => {
                    return Err(anyhow::anyhow!(
                        "Request blocked by {}: {}",
                        filter.name(),
                        reason
                    ));
                }
                FilterDecision::Transform { new_request } => {
                    tracing::info!("Filter {} transformed request", filter.name());
                    ctx.request = new_request;
                }
            }
        }

        Ok(ctx.request)
    }
}

impl ProxyRequest {
    pub fn new(command: String, args: Vec<String>) -> Self {
        Self {
            command,
            args,
            metadata: HashMap::new(),
        }
    }
}

impl ProxyContext {
    pub fn new(request: ProxyRequest, jwt_token: String) -> Self {
        Self {
            request,
            jwt_token,
            metadata: HashMap::new(),
        }
    }
}
