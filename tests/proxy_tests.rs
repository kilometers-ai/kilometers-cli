use km::proxy::ProxyTelemetry;

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

// Note: spawn_proxy_process and run_proxy are difficult to test in unit tests
// as they involve actual process spawning and I/O operations. These would be
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
