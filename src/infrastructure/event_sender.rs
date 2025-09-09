use anyhow::Result;
use chrono::Utc;
use std::collections::VecDeque;
use std::sync::Arc;
use tokio::sync::Mutex;
use uuid::Uuid;

// Configuration import removed as it's not used directly
use crate::domain::proxy::{McpEvent, McpEventBatch};
use crate::infrastructure::api_client::{ApiClient, EventSendError};
use crate::infrastructure::configuration_repository::ConfigurationRepository;

// Service for batching and sending MCP events to the API
pub struct EventSender {
    api_client: Arc<Mutex<ApiClient>>,
    event_buffer: Arc<Mutex<VecDeque<McpEvent>>>,
    batch_size: usize,
    config_repo: ConfigurationRepository,
}

impl Default for EventSender {
    fn default() -> Self {
        Self::new()
    }
}

impl EventSender {
    pub fn new() -> Self {
        Self {
            api_client: Arc::new(Mutex::new(ApiClient::new(
                "http://localhost:5195".to_string(),
            ))),
            event_buffer: Arc::new(Mutex::new(VecDeque::new())),
            batch_size: 1, // Send in batches of 1 event
            config_repo: ConfigurationRepository::new(),
        }
    }

    // Initialize with authentication
    pub async fn initialize(&self) -> Result<()> {
        if let Ok(config) = self.config_repo.load_configuration() {
            let mut client = self.api_client.lock().await;
            client.set_auth_token(config.api_key);
        }
        Ok(())
    }

    // Add an event to the buffer
    pub async fn buffer_event(&self, event: McpEvent) {
        let mut buffer = self.event_buffer.lock().await;
        buffer.push_back(event);

        // Check if we should flush the buffer
        if buffer.len() >= self.batch_size {
            self.flush_buffer_internal(&mut buffer).await;
        }
    }

    // Flush all buffered events
    pub async fn flush_buffer(&self) {
        let mut buffer = self.event_buffer.lock().await;
        self.flush_buffer_internal(&mut buffer).await;
    }

    // Internal method to flush buffer (assumes buffer is already locked)
    async fn flush_buffer_internal(&self, buffer: &mut VecDeque<McpEvent>) {
        if buffer.is_empty() {
            return;
        }

        let events: Vec<McpEvent> = buffer.drain(..).collect();
        let batch = McpEventBatch {
            events,
            cli_version: "0.1.0".to_string(),
            batch_timestamp: Utc::now(),
            correlation_id: Some(Uuid::new_v4().to_string()),
        };

        let client = self.api_client.lock().await;
        match client.send_event_batch(batch).await {
            Ok(()) => {
                // Events sent successfully
            }
            Err(e) => {
                eprintln!("Failed to send event batch to API: {}", e);
                // In a production system, you might want to retry or log to disk
            }
        }
    }

    // Send a single event immediately (bypass buffering)
    #[allow(dead_code)]
    pub async fn send_event_immediate(&self, event: McpEvent) -> Result<(), EventSendError> {
        let client = self.api_client.lock().await;
        client.send_event(event).await
    }
}

// Helper functions to create events from log content
pub fn create_request_event(content: String, method: Option<String>) -> McpEvent {
    McpEvent::new_request(content, method, None)
}

pub fn create_response_event(content: String, method: Option<String>) -> McpEvent {
    McpEvent::new_response(content, method, None)
}

// Helper to extract method from MCP JSON content
pub fn extract_method_from_content(content: &str) -> Option<String> {
    // Try to parse as JSON and extract the "method" field
    if let Ok(json) = serde_json::from_str::<serde_json::Value>(content) {
        if let Some(method) = json.get("method") {
            if let Some(method_str) = method.as_str() {
                return Some(method_str.to_string());
            }
        }
    }
    None
}
