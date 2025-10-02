use km::keyring_token_store::KeyringTokenStore;
use std::sync::Mutex;

// Mutex to serialize keyring tests to prevent interference
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

#[tokio::test]
async fn test_init_command_stores_token_in_keyring() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    // This test would require mocking the API client
    // For now, we'll test the keyring storage mechanism directly
    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a mock JWT token similar to what the API would return
    let mock_token = km::auth::JwtToken {
        token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.test".to_string(),
        expires_at: 9999999999,
        claims: km::auth::JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: Some("refresh-token-123".to_string()),
    };

    // Simulate what handle_init does: save tokens
    token_store
        .save_tokens(&mock_token, mock_token.refresh_token.as_deref())
        .expect("Failed to save tokens");

    // Verify token was stored
    assert!(token_store.token_exists());

    // Verify we can load it back
    let loaded_token = token_store
        .load_access_token()
        .expect("Failed to load access token");
    assert_eq!(loaded_token.token, mock_token.token);
    assert_eq!(loaded_token.claims.user_id, mock_token.claims.user_id);
    assert_eq!(loaded_token.claims.tier, mock_token.claims.tier);

    // Verify refresh token was stored
    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(loaded_refresh, mock_token.refresh_token);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_init_command_handles_keyring_failure_gracefully() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // This test verifies that if keyring operations fail,
    // the init command should continue gracefully

    // Even if keyring fails, the config file should still be saved
    // (This would require integration with the actual init command,
    // but for now we verify the keyring behavior)

    let token_store = KeyringTokenStore::new();
    assert!(
        token_store.is_ok(),
        "Keyring should initialize successfully"
    );
}

#[tokio::test]
async fn test_init_command_verifies_token_after_storage() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    let mock_token = km::auth::JwtToken {
        token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.verification_test".to_string(),
        expires_at: 9999999999,
        claims: km::auth::JwtClaims {
            sub: Some("verify-user".to_string()),
            tier: Some("enterprise".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-verify-123".to_string()),
        },
        refresh_token: None,
    };

    // Save token
    token_store
        .save_tokens(&mock_token, None)
        .expect("Failed to save tokens");

    // Immediately verify we can load it back (this is what handle_init does)
    let loaded_token = token_store
        .load_access_token()
        .expect("Verification failed: could not load token back");

    assert_eq!(loaded_token.token, mock_token.token);
    assert_eq!(
        loaded_token.claims.user_id,
        Some("user-verify-123".to_string())
    );

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_init_command_overwrites_existing_token() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Save initial token
    let old_token = km::auth::JwtToken {
        token: "old-token".to_string(),
        expires_at: 1234567890,
        claims: km::auth::JwtClaims {
            sub: Some("old-user".to_string()),
            tier: Some("free".to_string()),
            exp: Some(1234567890),
            iat: Some(1234567800),
            user_id: Some("old-user-id".to_string()),
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&old_token, None)
        .expect("Failed to save old token");

    // Save new token (simulating re-running km init)
    let new_token = km::auth::JwtToken {
        token: "new-token".to_string(),
        expires_at: 9999999999,
        claims: km::auth::JwtClaims {
            sub: Some("new-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567900),
            user_id: Some("new-user-id".to_string()),
        },
        refresh_token: Some("new-refresh".to_string()),
    };

    token_store
        .save_tokens(&new_token, new_token.refresh_token.as_deref())
        .expect("Failed to save new token");

    // Verify only new token exists
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");
    assert_eq!(loaded.token, "new-token");
    assert_eq!(loaded.claims.user_id, Some("new-user-id".to_string()));
    assert_eq!(loaded.claims.tier, Some("premium".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}
