// Serde is a framework for serializing/deserializing Rust data structures
use serde::{Deserialize, Serialize};

// `#[derive()]` is a procedural macro that auto-generates code
// `Serialize` trait allows converting this struct to JSON/other formats
#[derive(Serialize)]
pub struct ApiKeyCredentials {
    // Field-level attribute to rename during serialization
    // Rust uses snake_case, but API expects camelCase
    #[serde(rename = "apiKey")]
    pub api_key: String, // `String` is an owned, growable UTF-8 string
}

// Implementation block for methods on ApiKeyCredentials
impl ApiKeyCredentials {
    // Constructor function (convention: named `new`)
    // Takes ownership of the String parameter
    pub fn new(api_key: String) -> Self {
        // Struct literal syntax with field init shorthand
        // When field name = variable name, can omit the repetition
        Self { api_key }
    }
}

// Multiple derives: each adds different capabilities
// `Deserialize` - can be created from JSON/other formats
// `Debug` - can be printed with {:?} formatter
// `Clone` - can be duplicated with .clone() method
#[derive(Deserialize, Debug, Clone)]
pub struct Customer {
    pub id: String,
    pub email: String,
    // `Option<T>` is Rust's way of handling nullable values
    // Either Some(value) or None
    pub organization: Option<String>,
    #[serde(rename = "subscriptionPlan")]
    pub subscription_plan: String,
    #[serde(rename = "subscriptionStatus")]
    pub subscription_status: String,
    #[serde(rename = "hasPassword")]
    pub has_password: bool, // Boolean type
    #[serde(rename = "lastLoginAt")]
    pub last_login_at: Option<String>, // Optional field - might be null
    #[serde(rename = "createdAt")]
    pub created_at: String,
}

#[derive(Deserialize, Debug)]
#[allow(dead_code)]
pub struct AuthToken {
    #[serde(rename = "accessToken")]
    pub access_token: String,
    #[serde(rename = "refreshToken")]
    pub refresh_token: String,
    #[serde(rename = "accessTokenExpiresAt")]
    pub access_token_expires_at: String,
    #[serde(rename = "refreshTokenExpiresAt")]
    pub refresh_token_expires_at: String,
}

#[derive(Deserialize, Debug)]
#[allow(dead_code)]
pub struct AuthenticationResult {
    pub success: bool,
    pub customer: Customer,
    pub token: AuthToken,
}

// `enum` in Rust can hold different variants with associated data
// More powerful than enums in most languages - like tagged unions
#[derive(Debug)]
pub enum AuthenticationError {
    InvalidApiKey,            // Unit variant (no data)
    NetworkError(String),     // Tuple variant with one String
    ServerError(u16, String), // Tuple variant with u16 (unsigned 16-bit int) and String
    ParseError(String),       // Tuple variant with error message
}

// Implement the Display trait to make our error printable
// Traits are like interfaces in other languages
impl std::fmt::Display for AuthenticationError {
    // Required method for Display trait
    // `&self` - immutable reference to self
    // `&mut` - mutable reference (can modify)
    // `'_` - anonymous lifetime (compiler infers it)
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        // `match` is pattern matching - like switch but more powerful
        // Must handle all possible variants
        match self {
            AuthenticationError::InvalidApiKey => write!(f, "Invalid API key"),
            // Destructure to get the inner value
            AuthenticationError::NetworkError(msg) => write!(f, "Network error: {}", msg),
            // Can destructure multiple values
            AuthenticationError::ServerError(code, msg) => {
                write!(f, "Server error {}: {}", code, msg)
            }
            AuthenticationError::ParseError(msg) => write!(f, "Parse error: {}", msg),
        }
    }
}

// Implement Error trait (marker trait with default implementations)
// Empty impl block means we use all default methods
// This allows our type to be used with ? operator and Result
impl std::error::Error for AuthenticationError {}

#[derive(Clone)]
pub struct Configuration {
    pub api_key: String,
}

impl Configuration {
    pub fn new(api_key: String) -> Self {
        Self { api_key }
    }
}
