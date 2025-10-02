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
        token: "valid.jwt.token".to_string(),
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
        token: "expired.jwt.token".to_string(),
        expires_at: past_timestamp,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(past_timestamp),
            iat: Some(1234567800),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: Some("refresh-token-456".to_string()),
    }
}

#[tokio::test]
async fn test_monitor_loads_valid_cached_token_from_keyring() {
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

    // Verify token exists and is not expired
    assert!(token_store.token_exists());
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");
    assert!(!AuthClient::is_token_expired(&loaded));
    assert_eq!(loaded.token, valid_token.token);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_monitor_detects_expired_token_needs_refresh() {
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
        .save_tokens(&expired_token, expired_token.refresh_token.as_deref())
        .expect("Failed to save expired token");

    // Verify token exists but is expired
    assert!(token_store.token_exists());
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load expired token");
    assert!(AuthClient::is_token_expired(&loaded));

    // This simulates what get_jwt_token_with_cache does:
    // detects expiration and would fetch a new token

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_monitor_handles_missing_token_in_keyring() {
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

    // Attempting to load should fail gracefully
    let result = token_store.load_access_token();
    assert!(result.is_err());

    // This simulates what get_jwt_token_with_cache does:
    // no token found, so it would fetch a new one from the API
}

#[tokio::test]
async fn test_monitor_caches_new_token_after_refresh() {
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
        .save_tokens(&expired_token, expired_token.refresh_token.as_deref())
        .expect("Failed to save expired token");

    // Verify it's expired
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");
    assert!(AuthClient::is_token_expired(&loaded));

    // Simulate refresh by saving a new valid token
    let new_token = create_valid_token();
    token_store
        .save_tokens(&new_token, new_token.refresh_token.as_deref())
        .expect("Failed to save new token");

    // Verify the new token is cached and valid
    let refreshed = token_store
        .load_access_token()
        .expect("Failed to load refreshed token");
    assert!(!AuthClient::is_token_expired(&refreshed));
    assert_eq!(refreshed.token, new_token.token);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_monitor_handles_token_expiring_soon() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a token that expires in 30 seconds (within the 60s buffer)
    let near_expiry = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 30;

    let soon_to_expire = JwtToken {
        token: "soon.to.expire.token".to_string(),
        expires_at: near_expiry,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(near_expiry),
            iat: Some(1234567890),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&soon_to_expire, None)
        .expect("Failed to save token");

    // Verify token is considered expired due to 60s buffer
    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token");
    assert!(AuthClient::is_token_expired(&loaded));

    // This should trigger a refresh in get_jwt_token_with_cache

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_monitor_preserves_refresh_token_on_cache() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    let token_with_refresh = create_valid_token();
    let refresh_token = token_with_refresh.refresh_token.clone();

    token_store
        .save_tokens(&token_with_refresh, refresh_token.as_deref())
        .expect("Failed to save tokens");

    // Verify both access and refresh tokens are stored
    let loaded_access = token_store
        .load_access_token()
        .expect("Failed to load access token");
    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");

    assert_eq!(loaded_access.token, token_with_refresh.token);
    assert_eq!(loaded_refresh, refresh_token);

    // Clean up
    token_store.clear_tokens().ok();
}
