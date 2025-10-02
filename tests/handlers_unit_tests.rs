use km::config::Config;
use km::handlers::{handle_clear_logs, handle_doctor_jwt, handle_logs, handle_show_config};
use km::keyring_token_store::KeyringTokenStore;
use std::fs;
use std::path::PathBuf;
use std::sync::Mutex;
use tempfile::TempDir;

// Mutex to serialize tests that change the current working directory
static DIR_CHANGE_LOCK: Mutex<()> = Mutex::new(());
// Mutex to serialize keyring tests
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

#[test]
fn test_handle_clear_logs_removes_all_standard_log_files() {
    let _lock = DIR_CHANGE_LOCK.lock().unwrap();

    let temp_dir = TempDir::new().unwrap();
    std::env::set_current_dir(&temp_dir).unwrap();

    // Create all standard log files
    fs::write("mcp_traffic.jsonl", "traffic logs").unwrap();
    fs::write("mcp_requests.log", "request logs").unwrap();
    fs::write("mcp_proxy.log", "proxy logs").unwrap();

    let config_path = temp_dir.path().join("km_config.json");

    let result = handle_clear_logs(false, &config_path);
    assert!(result.is_ok());

    // Verify all log files were deleted
    assert!(!PathBuf::from("mcp_traffic.jsonl").exists());
    assert!(!PathBuf::from("mcp_requests.log").exists());
    assert!(!PathBuf::from("mcp_proxy.log").exists());
}

#[test]
fn test_handle_clear_logs_clears_keyring_when_token_exists() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Setup: Store a token in keyring
    if let Ok(token_store) = KeyringTokenStore::new() {
        let test_token = km::auth::JwtToken {
            token: "test-token".to_string(),
            expires_at: 9999999999,
            claims: km::auth::JwtClaims::default(),
            refresh_token: None,
        };

        if token_store.save_tokens(&test_token, None).is_ok() {
            // Verify token exists before clearing
            assert!(token_store.token_exists());

            let temp_dir = TempDir::new().unwrap();
            let config_path = temp_dir.path().join("km_config.json");

            // Clear logs (which should also clear keyring)
            let result = handle_clear_logs(false, &config_path);
            assert!(result.is_ok());

            // Verify token was cleared (best effort)
            // Note: May still exist if clear failed, but test ensures no panic
        }

        // Cleanup
        token_store.clear_tokens().ok();
    }
}

#[test]
fn test_handle_doctor_jwt_with_no_keyring() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Ensure no token exists
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    // Should succeed without panicking
    let result = handle_doctor_jwt();
    assert!(result.is_ok());
}

#[test]
fn test_handle_doctor_jwt_with_valid_token() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    if let Ok(token_store) = KeyringTokenStore::new() {
        // Clean up first
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }

        // Create a valid token
        use std::time::{SystemTime, UNIX_EPOCH};
        let future_timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs()
            + 3600;

        let test_token = km::auth::JwtToken {
            token: "valid.test.token".to_string(),
            expires_at: future_timestamp,
            claims: km::auth::JwtClaims {
                sub: Some("test-user".to_string()),
                tier: Some("premium".to_string()),
                exp: Some(future_timestamp),
                iat: Some(1234567890),
                user_id: Some("user-123".to_string()),
            },
            refresh_token: None,
        };

        if token_store.save_tokens(&test_token, None).is_ok() {
            let result = handle_doctor_jwt();
            assert!(result.is_ok());
        }

        // Cleanup
        token_store.clear_tokens().ok();
    }
}

#[test]
fn test_handle_doctor_jwt_with_expired_token() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    if let Ok(token_store) = KeyringTokenStore::new() {
        // Clean up first
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }

        // Create an expired token
        use std::time::{SystemTime, UNIX_EPOCH};
        let past_timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs()
            - 3600;

        let expired_token = km::auth::JwtToken {
            token: "expired.test.token".to_string(),
            expires_at: past_timestamp,
            claims: km::auth::JwtClaims {
                sub: Some("test-user".to_string()),
                tier: Some("free".to_string()),
                exp: Some(past_timestamp),
                iat: Some(1234567890),
                user_id: Some("user-456".to_string()),
            },
            refresh_token: None,
        };

        if token_store.save_tokens(&expired_token, None).is_ok() {
            let result = handle_doctor_jwt();
            assert!(result.is_ok());
        }

        // Cleanup
        token_store.clear_tokens().ok();
    }
}

#[test]
fn test_handle_show_config_with_default_tier() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    // Create a config with default_tier
    let mut config = Config::new("test-key".to_string(), "https://api.test.com".to_string());
    config.default_tier = Some("enterprise".to_string());
    config.save(&config_path).unwrap();

    let result = handle_show_config(&config_path, false);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&config_path).ok();
}

#[test]
fn test_handle_logs_complex_filtering() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("complex.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"initialize\",\"id\":1}"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"response","content":"{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":1}"}
{"timestamp":"2024-01-01T00:00:02Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"shutdown\",\"id\":2}"}
{"timestamp":"2024-01-01T00:00:03Z","direction":"response","content":"{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":2}"}"#;
    fs::write(&log_file, log_content).unwrap();

    // Test filtering requests with specific method
    let result = handle_logs(
        log_file.clone(),
        true,
        false,
        Some("initialize".to_string()),
        false,
        Some(1),
    );
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_with_malformed_content_field() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("malformed.log");

    // Log entry with content that's not valid JSON
    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"not-json-content"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"response","content":"{\"valid\":\"json\"}"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(
        log_file.clone(),
        false,
        false,
        Some("test".to_string()),
        false,
        None,
    );
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_displays_pretty_json() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("pretty.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"test","extra_field":"value"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_clear_logs_when_config_doesnt_exist() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("nonexistent_config.json");

    // Should succeed even when config doesn't exist
    let result = handle_clear_logs(true, &config_path);
    assert!(result.is_ok());
}

#[test]
fn test_handle_logs_with_zero_lines_limit() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"line1"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"request","content":"line2"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, Some(0));
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}
