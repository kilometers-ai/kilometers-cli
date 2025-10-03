use km::proxy::{spawn_proxy_process, ProxyTelemetry};

#[test]
fn test_proxy_telemetry_creation() {
    let telemetry = ProxyTelemetry::new();

    assert_eq!(telemetry.request_count, 0);
    assert_eq!(telemetry.response_count, 0);
    assert_eq!(telemetry.error_count, 0);
}

#[test]
fn test_proxy_telemetry_default_values() {
    let telemetry = ProxyTelemetry::new();

    // All counters should start at zero
    assert_eq!(telemetry.request_count, 0);
    assert_eq!(telemetry.response_count, 0);
    assert_eq!(telemetry.error_count, 0);
}

#[test]
fn test_proxy_telemetry_default_trait() {
    let telemetry = ProxyTelemetry::default();

    assert_eq!(telemetry.request_count, 0);
    assert_eq!(telemetry.response_count, 0);
    assert_eq!(telemetry.error_count, 0);
}

#[test]
fn test_spawn_proxy_process_with_invalid_command() {
    let result = spawn_proxy_process("this-command-definitely-does-not-exist-xyz123", &[]);
    assert!(result.is_err());
}

#[test]
fn test_spawn_proxy_process_with_echo() {
    let result = spawn_proxy_process("echo", &["test".to_string()]);
    assert!(result.is_ok());

    if let Ok(mut child) = result {
        // Ensure we clean up the process
        let _ = child.kill();
        let _ = child.wait();
    }
}

#[test]
fn test_spawn_proxy_process_with_multiple_args() {
    let args = vec!["hello".to_string(), "world".to_string()];
    let result = spawn_proxy_process("echo", &args);
    assert!(result.is_ok());

    if let Ok(mut child) = result {
        let _ = child.kill();
        let _ = child.wait();
    }
}

// Note: run_proxy is difficult to test in unit tests
// as it involves actual process spawning and I/O operations. These would be
// better tested in integration tests with mock processes.

#[cfg(test)]
mod json_detection_tests {
    use serde_json::Value;

    #[test]
    fn test_jsonrpc_request_detection() {
        let json_str = r#"{"jsonrpc": "2.0", "method": "test", "id": 1}"#;
        let json: Value = serde_json::from_str(json_str).unwrap();

        assert!(json.get("jsonrpc").is_some());
        assert_eq!(json.get("method").unwrap(), "test");
    }

    #[test]
    fn test_jsonrpc_response_detection() {
        let json_str = r#"{"jsonrpc": "2.0", "result": "success", "id": 1}"#;
        let json: Value = serde_json::from_str(json_str).unwrap();

        assert!(json.get("jsonrpc").is_some());
        assert_eq!(json.get("id").unwrap(), 1);
    }

    #[test]
    fn test_non_jsonrpc_content() {
        let json_str = r#"{"data": "test", "status": "ok"}"#;
        let json: Value = serde_json::from_str(json_str).unwrap();

        assert!(json.get("jsonrpc").is_none());
    }

    #[test]
    fn test_invalid_json() {
        let invalid_json = "{ invalid json content }";
        let result = serde_json::from_str::<Value>(invalid_json);

        assert!(result.is_err());
    }
}
