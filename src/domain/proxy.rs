// Chrono is a date/time library for Rust
use chrono::{DateTime, Utc};
use serde::Serialize;
use uuid::Uuid;

// `#[derive(Debug, Clone)]` auto-generates Debug and Clone traits
// Debug allows printing with {:?}, Clone allows .clone()
#[derive(Debug, Clone)]
pub struct ProxyCommand {
    pub command: String,
    // `Vec<T>` is a growable array (like ArrayList/List in other languages)
    pub args: Vec<String>,
}

impl ProxyCommand {
    // Constructor taking ownership of both parameters
    pub fn new(command: String, args: Vec<String>) -> Self {
        // Field init shorthand: `command: command` -> `command`
        Self { command, args }
    }
}

// Struct to represent a log entry
#[derive(Debug, Clone)]
pub struct LogEntry {
    // Generic type: DateTime parameterized with Utc timezone
    pub timestamp: DateTime<Utc>,
    pub entry_type: LogEntryType,
    pub content: String,
}

// Simple enum with two variants (no associated data)
#[derive(Debug, Clone)]
pub enum LogEntryType {
    Request,
    Response,
}

impl LogEntry {
    // Factory method for creating a Request log entry
    // No `self` parameter = associated function (like static method)
    pub fn request(content: String) -> Self {
        Self {
            timestamp: Utc::now(),             // Get current UTC time
            entry_type: LogEntryType::Request, // Enum variant
            content,                           // Field init shorthand
        }
    }

    // Factory method for creating a Response log entry
    pub fn response(content: String) -> Self {
        Self {
            timestamp: Utc::now(),
            entry_type: LogEntryType::Response,
            content,
        }
    }

    // Instance method (takes &self)
    pub fn format_for_log(&self) -> String {
        // Pattern match to convert enum to string
        let prefix = match self.entry_type {
            LogEntryType::Request => "REQUEST",
            LogEntryType::Response => "RESPONSE",
        };
        // `format!` macro creates a String with interpolation
        // Similar to sprintf/string formatting in other languages
        format!(
            "[{}] {}: {}",
            self.timestamp.to_rfc3339(),
            prefix,
            self.content
        )
    }
}

// MCP Event structure that matches the API's McpEventDto
#[derive(Debug, Clone, Serialize)]
pub struct McpEvent {
    pub id: String,
    pub timestamp: DateTime<Utc>,
    #[serde(rename = "correlationId")]
    pub correlation_id: Option<String>,
    pub direction: String,
    pub method: Option<String>,
    #[serde(with = "base64_serde")]
    pub payload: Vec<u8>,
    pub size: i32,
}

// Custom base64 serialization module
mod base64_serde {
    use base64::{engine::general_purpose, Engine as _};
    use serde::{Serialize, Serializer};

    pub fn serialize<S>(bytes: &Vec<u8>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        general_purpose::STANDARD
            .encode(bytes)
            .serialize(serializer)
    }
}

impl McpEvent {
    pub fn new_request(
        content: String,
        method: Option<String>,
        correlation_id: Option<Uuid>,
    ) -> Self {
        let payload = content.as_bytes().to_vec();
        Self {
            id: Uuid::new_v4().to_string(),
            timestamp: Utc::now(),
            correlation_id: correlation_id.map(|id| id.to_string()),
            direction: "request".to_string(),
            method,
            size: payload.len() as i32,
            payload,
        }
    }

    pub fn new_response(
        content: String,
        method: Option<String>,
        correlation_id: Option<Uuid>,
    ) -> Self {
        let payload = content.as_bytes().to_vec();
        Self {
            id: Uuid::new_v4().to_string(),
            timestamp: Utc::now(),
            correlation_id: correlation_id.map(|id| id.to_string()),
            direction: "response".to_string(),
            method,
            size: payload.len() as i32,
            payload,
        }
    }
}

// Batch structure for sending multiple events
#[derive(Debug, Serialize)]
pub struct McpEventBatch {
    pub events: Vec<McpEvent>,
    #[serde(rename = "cliVersion")]
    pub cli_version: String,
    #[serde(rename = "batchTimestamp")]
    pub batch_timestamp: DateTime<Utc>,
    #[serde(rename = "correlationId")]
    pub correlation_id: Option<String>,
}
