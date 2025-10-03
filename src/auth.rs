use anyhow::{Context, Result};
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine as _};
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone)]
pub struct AuthClient {
    api_key: String,
    base_url: String,
    client: reqwest::Client,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JwtToken {
    pub token: String,
    pub expires_at: u64,
    pub claims: JwtClaims,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct JwtClaims {
    pub sub: Option<String>,
    #[serde(alias = "plan")]
    pub tier: Option<String>,
    pub exp: Option<u64>,
    pub iat: Option<u64>,
    #[serde(alias = "customer_id")]
    pub user_id: Option<String>,
}

#[derive(Debug, Serialize)]
struct AuthRequest {
    #[serde(rename = "ApiKey")]
    api_key: String,
}

#[derive(Debug, Deserialize)]
struct AuthResponse {
    jwt: String,
    #[serde(rename = "expiresIn")]
    expires_in: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    refresh_token: Option<String>,
}

impl AuthClient {
    pub fn new(api_key: String, base_url: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .unwrap_or_else(|_| reqwest::Client::new());

        Self {
            api_key,
            base_url,
            client,
        }
    }

    pub async fn exchange_for_jwt(&self) -> Result<JwtToken> {
        let auth_request = AuthRequest {
            api_key: self.api_key.clone(),
        };

        let response = self
            .client
            .post(format!("{}/api/auth/exchange", self.base_url))
            .json(&auth_request)
            .send()
            .await
            .context("Failed to send auth request")?;

        if !response.status().is_success() {
            return Err(anyhow::anyhow!(
                "Auth failed with status: {}",
                response.status()
            ));
        }

        let auth_response: AuthResponse = response
            .json()
            .await
            .context("Failed to parse auth response")?;

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let claims = Self::parse_jwt_claims(&auth_response.jwt).unwrap_or_default();

        Ok(JwtToken {
            token: auth_response.jwt,
            expires_at: now + auth_response.expires_in,
            claims,
            refresh_token: auth_response.refresh_token,
        })
    }

    pub fn parse_jwt_claims(token: &str) -> Result<JwtClaims> {
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err(anyhow::anyhow!("Invalid JWT format"));
        }

        let payload = URL_SAFE_NO_PAD
            .decode(parts[1])
            .context("Failed to decode JWT payload")?;

        let claims: JwtClaims =
            serde_json::from_slice(&payload).context("Failed to parse JWT claims")?;

        Ok(claims)
    }

    pub fn is_token_expired(token: &JwtToken) -> bool {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        token.expires_at <= now + 60
    }
}
