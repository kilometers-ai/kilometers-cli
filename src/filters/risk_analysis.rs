use super::{FilterDecision, ProxyContext, ProxyFilter};
use anyhow::{Context, Result};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone)]
pub struct RiskAnalysisFilter {
    api_endpoint: String,
    client: reqwest::Client,
    threshold: f32,
}

#[derive(Debug, Serialize)]
struct RiskAnalysisRequest {
    command: String,
    args: Vec<String>,
    metadata: serde_json::Value,
}

#[derive(Debug, Deserialize)]
struct RiskAnalysisResponse {
    risk_score: f32,
    risk_level: String,
    recommendation: String,
    #[allow(dead_code)]
    details: Option<serde_json::Value>,
    suggested_transform: Option<TransformSuggestion>,
}

#[derive(Debug, Deserialize)]
struct TransformSuggestion {
    command: Option<String>,
    args: Option<Vec<String>>,
    reason: String,
}

impl RiskAnalysisFilter {
    pub fn new(api_endpoint: String, threshold: f32) -> Self {
        Self {
            api_endpoint,
            client: reqwest::Client::new(),
            threshold,
        }
    }

    async fn analyze_risk(&self, ctx: &ProxyContext) -> Result<RiskAnalysisResponse> {
        let request = RiskAnalysisRequest {
            command: ctx.request.command.clone(),
            args: ctx.request.args.clone(),
            metadata: serde_json::to_value(&ctx.request.metadata)?,
        };

        let response = self
            .client
            .post(&self.api_endpoint)
            .bearer_auth(&ctx.jwt_token)
            .json(&request)
            .send()
            .await
            .context("Failed to send risk analysis request")?;

        if !response.status().is_success() {
            return Err(anyhow::anyhow!(
                "Risk analysis failed with status: {}",
                response.status()
            ));
        }

        response
            .json::<RiskAnalysisResponse>()
            .await
            .context("Failed to parse risk analysis response")
    }
}

#[async_trait]
impl ProxyFilter for RiskAnalysisFilter {
    async fn check(&self, ctx: &ProxyContext) -> Result<FilterDecision> {
        let analysis = self.analyze_risk(ctx).await?;

        tracing::info!(
            "Risk analysis: score={}, level={}, recommendation={}",
            analysis.risk_score,
            analysis.risk_level,
            analysis.recommendation
        );

        if analysis.risk_score > self.threshold {
            return Ok(FilterDecision::Block {
                reason: format!(
                    "Risk score {} exceeds threshold {}. {}",
                    analysis.risk_score, self.threshold, analysis.recommendation
                ),
            });
        }

        if let Some(transform) = analysis.suggested_transform {
            if transform.command.is_some() || transform.args.is_some() {
                let mut new_request = ctx.request.clone();

                if let Some(new_command) = transform.command {
                    new_request.command = new_command;
                }

                if let Some(new_args) = transform.args {
                    new_request.args = new_args;
                }

                tracing::info!("Applying transform: {}", transform.reason);

                return Ok(FilterDecision::Transform { new_request });
            }
        }

        Ok(FilterDecision::Allow)
    }

    fn is_blocking(&self) -> bool {
        false
    }

    fn name(&self) -> &str {
        "RiskAnalysis"
    }
}
