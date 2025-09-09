use crate::domain::auth::{ApiKeyCredentials, AuthenticationError, AuthenticationResult};
use crate::domain::proxy::{McpEvent, McpEventBatch};
use reqwest; // HTTP client library (like HttpClient in C#)
             // serde imports removed as they're not used in this file

// Composition pattern - contains owned instances (not pointers)
pub struct ApiClient {
    base_url: String,           // Owned String (like std::string in C++)
    client: reqwest::Client,    // HTTP client instance
    auth_token: Option<String>, // Store auth token for authenticated requests
}

impl ApiClient {
    // Constructor takes ownership of base_url (move semantics, like std::move in C++)
    pub fn new(base_url: String) -> Self {
        Self {
            base_url,                       // Move the String into the struct
            client: reqwest::Client::new(), // Create new HTTP client
            auth_token: None,               // No auth token initially
        }
    }

    // Constructor with auth token
    #[allow(dead_code)]
    pub fn new_with_token(base_url: String, auth_token: String) -> Self {
        Self {
            base_url,
            client: reqwest::Client::new(),
            auth_token: Some(auth_token),
        }
    }

    // Set auth token after construction
    pub fn set_auth_token(&mut self, token: String) {
        self.auth_token = Some(token);
    }

    // Method takes credentials by value (move semantics - consumes the struct)
    // Return type is Result<Success, Error> - like C#'s Result<T> pattern or exceptions
    pub async fn authenticate(
        &self,
        credentials: ApiKeyCredentials,
    ) -> Result<AuthenticationResult, AuthenticationError> {
        let url = format!("{}/api/auth/token", self.base_url);

        // Method chaining (fluent interface pattern)
        // .await is like await in C# - suspends until async operation completes
        // .map_err transforms one error type to another (functional programming)
        let response = self
            .client
            .post(&url) // &url borrows the string (like const reference)
            .json(&credentials) // Serialize to JSON
            .send() // Send HTTP request
            .await // Wait for response
            .map_err(|e| AuthenticationError::NetworkError(e.to_string()))?; // Convert error type

        // Pattern matching on status code (like switch but more powerful)
        if response.status().is_success() {
            // Deserialize JSON response to our struct
            // <AuthenticationResult> is explicit type parameter (like templates in C++)
            response
                .json::<AuthenticationResult>()
                .await
                .map_err(|e| AuthenticationError::ParseError(e.to_string()))
        } else if response.status().as_u16() == 401 {
            Err(AuthenticationError::InvalidApiKey) // Return error enum variant
        } else {
            let status = response.status().as_u16();
            // .unwrap_or_else is like null coalescing operator ?? in C#
            let error_text = response
                .text()
                .await
                .unwrap_or_else(|_| "Unknown error".to_string());
            Err(AuthenticationError::ServerError(status, error_text))
        }
    }

    // Send a single MCP event to the API
    #[allow(dead_code)]
    pub async fn send_event(&self, event: McpEvent) -> Result<(), EventSendError> {
        let Some(api_key) = &self.auth_token else {
            return Err(EventSendError::NotAuthenticated);
        };

        let url = format!("{}/api/events", self.base_url);

        let response = self
            .client
            .post(&url)
            .header("X-API-Key", api_key)
            .header("Content-Type", "application/json")
            .json(&event)
            .send()
            .await
            .map_err(|e| EventSendError::NetworkError(e.to_string()))?;

        if response.status().is_success() {
            Ok(())
        } else if response.status().as_u16() == 401 {
            Err(EventSendError::NotAuthenticated)
        } else {
            let status = response.status().as_u16();
            let error_text = response
                .text()
                .await
                .unwrap_or_else(|_| "Unknown error".to_string());
            Err(EventSendError::ServerError(status, error_text))
        }
    }

    // Send a batch of MCP events to the API
    pub async fn send_event_batch(&self, batch: McpEventBatch) -> Result<(), EventSendError> {
        let Some(api_key) = &self.auth_token else {
            return Err(EventSendError::NotAuthenticated);
        };

        let url = format!("{}/api/events/batch", self.base_url);

        let response = self
            .client
            .post(&url)
            .header("X-API-Key", api_key)
            .header("Content-Type", "application/json")
            .json(&batch)
            .send()
            .await
            .map_err(|e| EventSendError::NetworkError(e.to_string()))?;

        if response.status().is_success() {
            Ok(())
        } else if response.status().as_u16() == 401 {
            Err(EventSendError::NotAuthenticated)
        } else {
            let status = response.status().as_u16();
            let error_text = response
                .text()
                .await
                .unwrap_or_else(|_| "Unknown error".to_string());
            Err(EventSendError::ServerError(status, error_text))
        }
    }
}

// Error type for event sending operations
#[derive(Debug)]
pub enum EventSendError {
    NotAuthenticated,
    NetworkError(String),
    ServerError(u16, String),
}

impl std::fmt::Display for EventSendError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EventSendError::NotAuthenticated => write!(f, "Not authenticated"),
            EventSendError::NetworkError(msg) => write!(f, "Network error: {}", msg),
            EventSendError::ServerError(code, msg) => write!(f, "Server error {}: {}", code, msg),
        }
    }
}

impl std::error::Error for EventSendError {}
