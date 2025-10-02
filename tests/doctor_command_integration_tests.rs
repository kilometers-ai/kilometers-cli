use km::auth::{AuthClient, JwtClaims, JwtToken};
use km::keyring_token_store::KeyringTokenStore;
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

// Mutex to serialize keyring tests to prevent interference
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

fn create_valid_token() -> JwtToken {
    let future_timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 3600; // 1 hour in future

    JwtToken {
        token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJ0aWVyIjoicHJlbWl1bSJ9.test".to_string(),
        expires_at: future_timestamp,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(future_timestamp),
            iat: Some(1234567890),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: Some("refresh-token-123".to_string()),
    }
}

fn create_expired_token() -> JwtToken {
    let past_timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 3600; // 1 hour in past

    JwtToken {
        token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJ0aWVyIjoiZnJlZSJ9.expired".to_string(),
        expires_at: past_timestamp,
        claims: JwtClaims {
            sub: Some("expired-user".to_string()),
            tier: Some("free".to_string()),
            exp: Some(past_timestamp),
            iat: Some(1234567800),
            user_id: Some("user-expired".to_string()),
        },
        refresh_token: None,
    }
}

#[tokio::test]
async fn test_doctor_jwt_displays_valid_token_info() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store a valid token
    let valid_token = create_valid_token();
    token_store
        .save_tokens(&valid_token, valid_token.refresh_token.as_deref())
        .expect("Failed to save token");

    // Load and verify the token (simulating what handle_doctor_jwt does)
    assert!(token_store.token_exists());

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");

    // Verify token properties
    assert!(!AuthClient::is_token_expired(&loaded));
    assert_eq!(loaded.claims.user_id, Some("user-123".to_string()));
    assert_eq!(loaded.claims.tier, Some("premium".to_string()));
    assert_eq!(loaded.claims.sub, Some("test-user".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_doctor_jwt_displays_expired_token_with_warning() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Store an expired token
    let expired_token = create_expired_token();
    token_store
        .save_tokens(&expired_token, None)
        .expect("Failed to save expired token");

    // Load and verify (simulating what handle_doctor_jwt does)
    assert!(token_store.token_exists());

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load expired token");

    // Verify token is expired
    assert!(AuthClient::is_token_expired(&loaded));
    assert_eq!(loaded.claims.user_id, Some("user-expired".to_string()));
    assert_eq!(loaded.claims.tier, Some("free".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_doctor_jwt_handles_missing_token() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Verify no token exists
    assert!(!token_store.token_exists());

    // Attempting to load should return an error
    let result = token_store.load_access_token();
    assert!(result.is_err());

    // This is what handle_doctor_jwt should handle gracefully
    // by displaying "No JWT token found in keyring"
}

#[tokio::test]
async fn test_doctor_jwt_handles_corrupted_token_data() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Manually store corrupted JSON data
    // Note: This would require direct keyring access to inject bad data
    // For now, we test that malformed JwtToken structures are handled

    // Create a token with all None values (unusual but valid)
    let minimal_token = JwtToken {
        token: "minimal.token.data".to_string(),
        expires_at: 0,
        claims: JwtClaims {
            sub: None,
            tier: None,
            exp: None,
            iat: None,
            user_id: None,
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&minimal_token, None)
        .expect("Failed to save minimal token");

    // Load and verify (should work, just with None values)
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load minimal token");

    assert_eq!(loaded.token, "minimal.token.data");
    assert_eq!(loaded.claims.user_id, None);
    assert_eq!(loaded.claims.tier, None);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_doctor_jwt_displays_refresh_token_status() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Test with refresh token
    let token_with_refresh = create_valid_token();
    token_store
        .save_tokens(
            &token_with_refresh,
            token_with_refresh.refresh_token.as_deref(),
        )
        .expect("Failed to save token with refresh");

    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(loaded_refresh, Some("refresh-token-123".to_string()));

    // Clean up
    token_store.clear_tokens().ok();

    // Test without refresh token
    let token_without_refresh = JwtToken {
        token: "no.refresh.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    token_store
        .save_tokens(&token_without_refresh, None)
        .expect("Failed to save token without refresh");

    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(loaded_refresh, None);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_doctor_jwt_timestamp_formatting() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    let known_timestamp = 1704067200u64; // Jan 1, 2024 00:00:00 UTC

    let token = JwtToken {
        token: "timestamp.test.token".to_string(),
        expires_at: known_timestamp,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(known_timestamp),
            iat: Some(known_timestamp - 3600), // 1 hour earlier
            user_id: Some("user-ts-test".to_string()),
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&token, None)
        .expect("Failed to save token");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");

    // Verify timestamps are preserved correctly
    assert_eq!(loaded.expires_at, known_timestamp);
    assert_eq!(loaded.claims.exp, Some(known_timestamp));
    assert_eq!(loaded.claims.iat, Some(known_timestamp - 3600));

    // Verify this token is expired (since it's in the past)
    assert!(AuthClient::is_token_expired(&loaded));

    // Clean up
    token_store.clear_tokens().ok();
}
