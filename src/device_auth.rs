use anyhow::{Context, Result};
use serde::Deserialize;
use std::time::Duration;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StartResponse {
    pub device_code: String,
    pub user_code: String,
    pub verification_uri: String,
    pub verification_uri_complete: String,
    pub expires_in: u64,
    pub interval: u64,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PollSuccessResponse {
    pub token: TokenInfo,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TokenInfo {
    pub access_token: String,
    #[allow(dead_code)]
    pub access_token_expires_at: String,
    #[allow(dead_code)]
    pub token_type: String,
}

pub struct DeviceAuthClient {
    base_url: String,
    client: reqwest::Client,
}

impl DeviceAuthClient {
    pub fn new(base_url: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());
        Self { base_url, client }
    }

    pub async fn start(&self) -> Result<StartResponse> {
        let res = self
            .client
            .post(format!("{}/api/auth/device-code/start", self.base_url))
            .json(&serde_json::json!({}))
            .send()
            .await
            .context("Failed to start device code flow")?;
        if !res.status().is_success() {
            return Err(anyhow::anyhow!("Start failed: {}", res.status()));
        }
        res.json().await.context("Failed to parse start response")
    }

    pub async fn poll(&self, device_code: &str) -> Result<Result<PollSuccessResponse, String>> {
        let res = self
            .client
            .post(format!("{}/api/auth/device-code/poll", self.base_url))
            .json(&serde_json::json!({"deviceCode": device_code}))
            .send()
            .await
            .context("Failed to poll device code")?;
        if res.status().is_success() {
            let ok: PollSuccessResponse =
                res.json().await.context("Failed to parse poll success")?;
            return Ok(Ok(ok));
        }
        let status = res.status();
        let err = res
            .json::<serde_json::Value>()
            .await
            .ok()
            .and_then(|v| {
                v.get("error")
                    .and_then(|e| e.as_str())
                    .map(|s| s.to_string())
            })
            .unwrap_or_else(|| format!("http_{}", status));
        Ok(Err(err))
    }
}
