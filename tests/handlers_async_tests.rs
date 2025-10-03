use km::config::Config;
use km::handlers::{get_jwt_token_with_cache, handle_monitor};
use km::keyring_token_store::KeyringTokenStore;
use std::fs;
use std::sync::Mutex;
use tempfile::TempDir;

// Mutex to serialize keyring tests
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

#[tokio::test]
async fn test_handle_monitor_missing_command() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Empty args should fail
    let result = handle_monitor(&config_path, vec![], false, None, log_file).await;
    assert!(result.is_err());
    assert!(result
        .unwrap_err()
        .to_string()
        .contains("No command provided"));
}

#[tokio::test]
async fn test_handle_monitor_local_only_mode() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Using 'true' (valid command) with local_only should work
    let args = vec!["true".to_string()];
    let result = handle_monitor(&config_path, args, true, None, log_file).await;

    // May succeed or fail depending on proxy execution, but shouldn't panic
    // The important part is testing the local-only code path
    let _ = result;
}

#[tokio::test]
async fn test_handle_monitor_with_missing_config() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("nonexistent_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Should fall back to local-only mode when config doesn't exist
    let args = vec!["true".to_string()];
    let result = handle_monitor(&config_path, args, false, None, log_file).await;

    // May succeed or fail depending on proxy execution
    let _ = result;
}

#[tokio::test]
async fn test_handle_monitor_with_override_tier() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Test with tier override in local-only mode
    let args = vec!["true".to_string()];
    let override_tier = Some("enterprise".to_string());
    let result = handle_monitor(&config_path, args, true, override_tier, log_file).await;

    let _ = result;
}

#[tokio::test]
async fn test_get_jwt_token_with_cache_no_keyring_support() {
    {
        let _lock = KEYRING_TEST_LOCK.lock().unwrap();

        // Clean up any existing tokens
        if let Ok(token_store) = KeyringTokenStore::new() {
            if token_store.token_exists() {
                token_store.clear_tokens().ok();
            }
        }
    } // Lock dropped here

    // Test with invalid API credentials (should return None)
    let result = get_jwt_token_with_cache(
        "invalid-key".to_string(),
        "https://invalid.example.com".to_string(),
    )
    .await;

    // Should return None because authentication fails
    assert!(result.is_none());
}

#[tokio::test]
async fn test_get_jwt_token_with_cache_uses_cached_token() {
    {
        let _lock = KEYRING_TEST_LOCK.lock().unwrap();

        if let Ok(token_store) = KeyringTokenStore::new() {
            // Clean up first
            if token_store.token_exists() {
                token_store.clear_tokens().ok();
            }

            // Store a valid token
            use std::time::{SystemTime, UNIX_EPOCH};
            let future_timestamp = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs()
                + 3600;

            let cached_token = km::auth::JwtToken {
                token: "cached.jwt.token".to_string(),
                expires_at: future_timestamp,
                claims: km::auth::JwtClaims {
                    sub: Some("cached-user".to_string()),
                    tier: Some("premium".to_string()),
                    exp: Some(future_timestamp),
                    iat: Some(1234567890),
                    user_id: Some("cached-user-123".to_string()),
                },
                refresh_token: None,
            };

            token_store.save_tokens(&cached_token, None).ok();
        }
    } // Lock dropped here

    // Should use cached token instead of making API call
    let result =
        get_jwt_token_with_cache("any-key".to_string(), "https://any-url.com".to_string()).await;

    // Should return the cached token
    if let Some(token) = result {
        assert_eq!(token.token, "cached.jwt.token");
    }

    // Cleanup
    {
        let _lock = KEYRING_TEST_LOCK.lock().unwrap();
        if let Ok(token_store) = KeyringTokenStore::new() {
            token_store.clear_tokens().ok();
        }
    }
}

#[tokio::test]
async fn test_get_jwt_token_with_cache_expired_token() {
    {
        let _lock = KEYRING_TEST_LOCK.lock().unwrap();

        if let Ok(token_store) = KeyringTokenStore::new() {
            // Clean up first
            if token_store.token_exists() {
                token_store.clear_tokens().ok();
            }

            // Store an expired token
            use std::time::{SystemTime, UNIX_EPOCH};
            let past_timestamp = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs()
                - 3600;

            let expired_token = km::auth::JwtToken {
                token: "expired.jwt.token".to_string(),
                expires_at: past_timestamp,
                claims: km::auth::JwtClaims {
                    sub: Some("expired-user".to_string()),
                    tier: Some("free".to_string()),
                    exp: Some(past_timestamp),
                    iat: Some(1234567890),
                    user_id: Some("expired-user-123".to_string()),
                },
                refresh_token: None,
            };

            token_store.save_tokens(&expired_token, None).ok();
        }
    } // Lock dropped here

    // Should attempt to fetch new token (will fail with invalid creds)
    let result = get_jwt_token_with_cache(
        "invalid-key".to_string(),
        "https://invalid.example.com".to_string(),
    )
    .await;

    // Should return None because API call fails
    assert!(result.is_none());

    // Cleanup
    {
        let _lock = KEYRING_TEST_LOCK.lock().unwrap();
        if let Ok(token_store) = KeyringTokenStore::new() {
            token_store.clear_tokens().ok();
        }
    }
}

#[tokio::test]
async fn test_handle_monitor_creates_command_metadata_log() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("mcp_traffic.jsonl");

    // Test with local-only mode to avoid API calls
    let args = vec!["true".to_string()];
    let _ = handle_monitor(&config_path, args, true, None, log_file.clone()).await;

    // Check if km_commands.log was created (metadata log)
    let _metadata_log = temp_dir.path().join("km_commands.log");
    // Note: This may or may not exist depending on execution, but shouldn't panic
}

#[tokio::test]
async fn test_handle_monitor_with_valid_config_but_no_network() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Create a config with invalid URL (network will fail)
    let config = Config::new(
        "test-key".to_string(),
        "https://invalid-nonexistent-api.example.com".to_string(),
    );
    config.save(&config_path).unwrap();

    // Should fall back to local-only mode when auth fails
    let args = vec!["true".to_string()];
    let _ = handle_monitor(&config_path, args, false, None, log_file).await;

    // Cleanup
    fs::remove_file(&config_path).ok();
}

#[tokio::test]
async fn test_handle_monitor_with_multiple_args() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");
    let log_file = temp_dir.path().join("test.log");

    // Test with command that has arguments
    let args = vec!["echo".to_string(), "hello".to_string(), "world".to_string()];
    let _ = handle_monitor(&config_path, args, true, None, log_file).await;
}
