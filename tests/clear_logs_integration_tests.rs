use km::auth::{JwtClaims, JwtToken};
use km::keyring_token_store::KeyringTokenStore;
use std::sync::Mutex;

// Mutex to serialize keyring tests to prevent interference
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

// Check if running in CI environment
fn should_skip_keyring_test() -> bool {
    std::env::var("CI").is_ok() || std::env::var("GITHUB_ACTIONS").is_ok()
}

#[tokio::test]
async fn test_clear_logs_removes_tokens_from_keyring() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store a token
    let test_token = JwtToken {
        token: "test.token.to.clear".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-clear-test".to_string()),
        },
        refresh_token: Some("refresh-to-clear".to_string()),
    };

    token_store
        .save_tokens(&test_token, test_token.refresh_token.as_deref())
        .expect("Failed to save token");

    // Verify token exists
    assert!(token_store.token_exists());

    // Simulate what handle_clear_logs does
    token_store.clear_tokens().expect("Failed to clear tokens");

    // Verify token no longer exists
    assert!(!token_store.token_exists());

    // Verify cannot load tokens
    assert!(token_store.load_access_token().is_err());
    let refresh_result = token_store
        .load_refresh_token()
        .expect("Should not error when no refresh token");
    assert_eq!(refresh_result, None);
}

#[tokio::test]
async fn test_clear_logs_handles_no_tokens_gracefully() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Verify no tokens exist
    assert!(!token_store.token_exists());

    // Clearing when no tokens exist should not error
    let result = token_store.clear_tokens();
    assert!(result.is_ok());

    // Still should have no tokens
    assert!(!token_store.token_exists());
}

#[tokio::test]
async fn test_clear_logs_removes_both_access_and_refresh_tokens() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store both access and refresh tokens
    let test_token = JwtToken {
        token: "access.token.both".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: Some("refresh.token.both".to_string()),
    };

    token_store
        .save_tokens(&test_token, test_token.refresh_token.as_deref())
        .expect("Failed to save tokens");

    // Verify both exist
    assert!(token_store.token_exists());
    let refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh");
    assert_eq!(refresh, Some("refresh.token.both".to_string()));

    // Clear all tokens
    token_store.clear_tokens().expect("Failed to clear tokens");

    // Verify both are gone
    assert!(!token_store.token_exists());
    let refresh_after = token_store.load_refresh_token().expect("Should not error");
    assert_eq!(refresh_after, None);
}

#[tokio::test]
async fn test_clear_logs_without_include_config_only_clears_tokens() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store a token
    let test_token = JwtToken {
        token: "token.clear.no.config".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    token_store
        .save_tokens(&test_token, None)
        .expect("Failed to save token");

    assert!(token_store.token_exists());

    // Simulate clear-logs without --include-config
    // (Only clears tokens, not the config file)
    token_store.clear_tokens().expect("Failed to clear tokens");

    assert!(!token_store.token_exists());

    // Note: Config file handling is separate from keyring
    // This test verifies that token clearing works independently
}

#[tokio::test]
async fn test_clear_logs_can_be_called_multiple_times() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store a token
    let test_token = JwtToken {
        token: "multi.clear.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    token_store
        .save_tokens(&test_token, None)
        .expect("Failed to save token");

    // Clear once
    token_store.clear_tokens().expect("First clear failed");
    assert!(!token_store.token_exists());

    // Clear again (should not error)
    token_store.clear_tokens().expect("Second clear failed");
    assert!(!token_store.token_exists());

    // Clear a third time (still should not error)
    token_store.clear_tokens().expect("Third clear failed");
    assert!(!token_store.token_exists());
}

#[tokio::test]
async fn test_clear_logs_allows_new_tokens_after_clearing() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store first token
    let first_token = JwtToken {
        token: "first.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims {
            user_id: Some("user-1".to_string()),
            ..Default::default()
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&first_token, None)
        .expect("Failed to save first token");
    assert!(token_store.token_exists());

    // Clear
    token_store.clear_tokens().expect("Failed to clear");
    assert!(!token_store.token_exists());

    // Store new token
    let second_token = JwtToken {
        token: "second.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims {
            user_id: Some("user-2".to_string()),
            ..Default::default()
        },
        refresh_token: Some("second.refresh".to_string()),
    };

    token_store
        .save_tokens(&second_token, second_token.refresh_token.as_deref())
        .expect("Failed to save second token");

    // Verify new token is stored correctly
    assert!(token_store.token_exists());
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load new token");
    assert_eq!(loaded.token, "second.token");
    assert_eq!(loaded.claims.user_id, Some("user-2".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}
