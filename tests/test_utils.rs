use km::auth::{JwtClaims, JwtToken};
use km::filters::{ProxyContext, ProxyRequest};
use std::collections::HashMap;
use std::path::PathBuf;
use std::process::Command;
use std::fs;
use std::io::Write;
use std::time::{Duration, Instant};
use tempfile::NamedTempFile;
use serde_json::Value;

pub fn create_mock_jwt_token(user_id: Option<String>, tier: Option<String>) -> JwtToken {
    let claims = JwtClaims {
        sub: Some("test-subject".to_string()),
        exp: Some(9999999999), // Far future expiration
        iat: Some(1234567890),
        user_id,
        tier,
    };

    JwtToken {
        token: "mock-jwt-token".to_string(),
        expires_at: 9999999999,
        claims,
    }
}

pub fn create_test_context(command: &str, args: Vec<&str>) -> ProxyContext {
    create_test_context_with_metadata(command, args, HashMap::new())
}

pub fn create_test_context_with_metadata(
    command: &str,
    args: Vec<&str>,
    metadata: HashMap<String, String>,
) -> ProxyContext {
    let request = ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata,
    };

    ProxyContext::new(request, "test-token".to_string())
}

pub fn create_test_request(command: &str, args: Vec<&str>) -> ProxyRequest {
    ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata: HashMap::new(),
    }
}

pub fn create_test_request_with_metadata(
    command: &str,
    args: Vec<&str>,
    metadata: HashMap<String, String>,
) -> ProxyRequest {
    ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata,
    }
}

// JWT token variants for common test scenarios
pub fn create_free_tier_token() -> JwtToken {
    create_mock_jwt_token(Some("free-user".to_string()), Some("free".to_string()))
}

pub fn create_premium_tier_token() -> JwtToken {
    create_mock_jwt_token(
        Some("premium-user".to_string()),
        Some("premium".to_string()),
    )
}

pub fn create_enterprise_tier_token() -> JwtToken {
    create_mock_jwt_token(
        Some("enterprise-user".to_string()),
        Some("enterprise".to_string()),
    )
}

pub fn create_no_tier_token() -> JwtToken {
    create_mock_jwt_token(Some("no-tier-user".to_string()), None)
}

pub fn create_anonymous_token() -> JwtToken {
    create_mock_jwt_token(None, None)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_mock_jwt_token() {
        let token = create_mock_jwt_token(Some("user123".to_string()), Some("premium".to_string()));

        assert_eq!(token.token, "mock-jwt-token");
        assert_eq!(token.claims.sub, Some("test-subject".to_string()));
        assert_eq!(token.claims.user_id, Some("user123".to_string()));
        assert_eq!(token.claims.tier, Some("premium".to_string()));
    }

    #[test]
    fn test_create_test_context() {
        let context = create_test_context("test-cmd", vec!["arg1", "arg2"]);

        assert_eq!(context.request.command, "test-cmd");
        assert_eq!(context.request.args, vec!["arg1", "arg2"]);
        assert_eq!(context.jwt_token, "test-token");
        assert!(context.request.metadata.is_empty());
    }

    #[test]
    fn test_create_test_context_with_metadata() {
        let mut metadata = HashMap::new();
        metadata.insert("key1".to_string(), "value1".to_string());
        metadata.insert("key2".to_string(), "value2".to_string());

        let context = create_test_context_with_metadata("test-cmd", vec!["arg1"], metadata.clone());

        assert_eq!(context.request.command, "test-cmd");
        assert_eq!(context.request.args, vec!["arg1"]);
        assert_eq!(context.request.metadata, metadata);
    }

    #[test]
    fn test_tier_token_variants() {
        let free_token = create_free_tier_token();
        let premium_token = create_premium_tier_token();
        let enterprise_token = create_enterprise_tier_token();
        let no_tier_token = create_no_tier_token();
        let anonymous_token = create_anonymous_token();

        assert_eq!(free_token.claims.tier, Some("free".to_string()));
        assert_eq!(premium_token.claims.tier, Some("premium".to_string()));
        assert_eq!(enterprise_token.claims.tier, Some("enterprise".to_string()));
        assert_eq!(no_tier_token.claims.tier, None);
        assert_eq!(anonymous_token.claims.user_id, None);
        assert_eq!(anonymous_token.claims.tier, None);
    }
}

// Integration test utilities

/// Find the km binary (debug or release) for testing
pub fn find_km_binary() -> PathBuf {
    let debug_path = PathBuf::from("./target/debug/km");
    let release_path = PathBuf::from("./target/release/km");

    if release_path.exists() {
        release_path
    } else if debug_path.exists() {
        debug_path
    } else {
        // Try to build debug version
        let output = Command::new("cargo")
            .args(&["build"])
            .output()
            .expect("Failed to run cargo build");

        if !output.status.success() {
            panic!("Failed to build km binary: {}", String::from_utf8_lossy(&output.stderr));
        }

        if debug_path.exists() {
            debug_path
        } else {
            panic!("km binary not found after build attempt");
        }
    }
}

/// Create sample MCP JSON-RPC requests for testing
pub fn create_sample_mcp_requests() -> Vec<Value> {
    vec![
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "roots": {"listChanged": true},
                    "sampling": {}
                }
            }
        }),
        serde_json::json!({
            "jsonrpc": "2.0",
            "method": "notifications/initialized"
        }),
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list"
        }),
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "echo",
                "arguments": {"text": "Hello from test!"}
            }
        }),
    ]
}

/// Validate that a log file contains valid JSON lines
pub fn validate_log_file_format(log_file_path: &PathBuf) -> Result<Vec<Value>, String> {
    if !log_file_path.exists() {
        return Err("Log file does not exist".to_string());
    }

    let content = fs::read_to_string(log_file_path)
        .map_err(|e| format!("Failed to read log file: {}", e))?;

    if content.trim().is_empty() {
        return Ok(Vec::new());
    }

    let mut entries = Vec::new();
    for (line_num, line) in content.lines().enumerate() {
        if line.trim().is_empty() {
            continue;
        }

        match serde_json::from_str::<Value>(line) {
            Ok(entry) => entries.push(entry),
            Err(e) => return Err(format!("Invalid JSON on line {}: {} - {}", line_num + 1, line, e)),
        }
    }

    Ok(entries)
}

/// Validate that log entries have required fields
pub fn validate_log_entries(entries: &[Value]) -> Result<(), String> {
    for (i, entry) in entries.iter().enumerate() {
        // Check required fields
        if entry.get("timestamp").is_none() {
            return Err(format!("Log entry {} missing timestamp", i));
        }

        // Validate timestamp format if present
        if let Some(timestamp) = entry.get("timestamp").and_then(|t| t.as_str()) {
            if chrono::DateTime::parse_from_rfc3339(timestamp).is_err() {
                return Err(format!("Log entry {} has invalid timestamp format: {}", i, timestamp));
            }
        }

        // Event type validation (if present)
        if let Some(event_type) = entry.get("event_type") {
            if !event_type.is_string() {
                return Err(format!("Log entry {} has non-string event_type", i));
            }
        }
    }

    Ok(())
}

/// Create a temporary test configuration file
pub fn create_test_config(api_key: &str, api_url: &str) -> Result<NamedTempFile, std::io::Error> {
    let mut temp_file = NamedTempFile::new()?;
    let config_content = serde_json::json!({
        "api_key": api_key,
        "api_url": api_url
    });

    writeln!(temp_file, "{}", config_content)?;
    temp_file.flush()?;
    Ok(temp_file)
}

/// Wait for a condition to be true with timeout
pub fn wait_for_condition<F>(condition: F, timeout: Duration, check_interval: Duration) -> bool
where
    F: Fn() -> bool,
{
    let start = Instant::now();
    while start.elapsed() < timeout {
        if condition() {
            return true;
        }
        std::thread::sleep(check_interval);
    }
    false
}

/// Extract JSON-RPC responses from mixed output
pub fn extract_jsonrpc_responses(output: &str) -> Vec<Value> {
    let mut responses = Vec::new();

    for line in output.lines() {
        if let Ok(json) = serde_json::from_str::<Value>(line) {
            // Check if it looks like a JSON-RPC response
            if json.get("jsonrpc").is_some() &&
               (json.get("result").is_some() || json.get("error").is_some() || json.get("method").is_some()) {
                responses.push(json);
            }
        }
    }

    responses
}

/// Common test scenarios for different command types
pub struct TestScenario {
    pub name: String,
    pub command: String,
    pub args: Vec<String>,
    pub expected_risk_level: String,
    pub should_be_blocked: bool,
}

impl TestScenario {
    pub fn safe_commands() -> Vec<TestScenario> {
        vec![
            TestScenario {
                name: "List directory".to_string(),
                command: "ls".to_string(),
                args: vec!["-la".to_string()],
                expected_risk_level: "low".to_string(),
                should_be_blocked: false,
            },
            TestScenario {
                name: "Show current directory".to_string(),
                command: "pwd".to_string(),
                args: vec![],
                expected_risk_level: "low".to_string(),
                should_be_blocked: false,
            },
            TestScenario {
                name: "Echo command".to_string(),
                command: "echo".to_string(),
                args: vec!["hello".to_string(), "world".to_string()],
                expected_risk_level: "low".to_string(),
                should_be_blocked: false,
            },
        ]
    }

    pub fn risky_commands() -> Vec<TestScenario> {
        vec![
            TestScenario {
                name: "Recursive delete".to_string(),
                command: "rm".to_string(),
                args: vec!["-rf".to_string(), "/tmp/test".to_string()],
                expected_risk_level: "high".to_string(),
                should_be_blocked: true,
            },
            TestScenario {
                name: "Format disk".to_string(),
                command: "mkfs".to_string(),
                args: vec!["/dev/sda1".to_string()],
                expected_risk_level: "critical".to_string(),
                should_be_blocked: true,
            },
            TestScenario {
                name: "Change system file permissions".to_string(),
                command: "chmod".to_string(),
                args: vec!["777".to_string(), "/etc/passwd".to_string()],
                expected_risk_level: "high".to_string(),
                should_be_blocked: true,
            },
        ]
    }
}

/// Mock MCP server response generator
pub fn generate_mock_mcp_response(request_id: Option<u64>, method: Option<&str>) -> Value {
    match method {
        Some("initialize") => serde_json::json!({
            "jsonrpc": "2.0",
            "id": request_id,
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {"list": true, "call": true}
                }
            }
        }),
        Some("tools/list") => serde_json::json!({
            "jsonrpc": "2.0",
            "id": request_id,
            "result": {
                "tools": [
                    {
                        "name": "echo",
                        "description": "Echo text back",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "text": {"type": "string"}
                            }
                        }
                    }
                ]
            }
        }),
        Some("tools/call") => serde_json::json!({
            "jsonrpc": "2.0",
            "id": request_id,
            "result": {
                "output": "Mock tool execution result"
            }
        }),
        _ => serde_json::json!({
            "jsonrpc": "2.0",
            "id": request_id,
            "error": {
                "code": -32601,
                "message": "Method not found"
            }
        }),
    }
}
