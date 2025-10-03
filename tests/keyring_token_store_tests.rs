use km::auth::JwtToken;
use km::keyring_token_store::KeyringTokenStore;
use std::sync::Mutex;

// Mutex to serialize keyring tests to prevent interference
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

// Check if running in CI environment
fn should_skip_keyring_test() -> bool {
    std::env::var("CI").is_ok() || std::env::var("GITHUB_ACTIONS").is_ok()
}

fn create_test_jwt_token() -> JwtToken {
    JwtToken {
        token: "test.jwt.token".to_string(),
        expires_at: 9999999999, // Far future
        claims: km::auth::JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: None,
    }
}

fn create_test_jwt_token_with_refresh() -> JwtToken {
    JwtToken {
        token: "test.jwt.token".to_string(),
        expires_at: 9999999999, // Far future
        claims: km::auth::JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-123".to_string()),
        },
        refresh_token: Some("test.refresh.token".to_string()),
    }
}

#[test]
fn test_keyring_token_store_save_and_load_access_token() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Clear any existing tokens first
    if token_store.token_exists() {
        token_store
            .clear_tokens()
            .expect("Failed to clear existing tokens");
    }

    let test_token = create_test_jwt_token();

    // Save token (no refresh token)
    token_store
        .save_tokens(&test_token, None)
        .expect("Failed to save token");
    assert!(token_store.token_exists());

    // Load access token
    let loaded_token = token_store
        .load_access_token()
        .expect("Failed to load access token");
    assert_eq!(loaded_token.token, test_token.token);
    assert_eq!(loaded_token.expires_at, test_token.expires_at);
    assert_eq!(loaded_token.claims.tier, test_token.claims.tier);

    // Clean up
    token_store.clear_tokens().ok();
}

#[test]
fn test_keyring_token_store_save_and_load_with_refresh_token() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Clear any existing tokens first
    if token_store.token_exists() {
        token_store
            .clear_tokens()
            .expect("Failed to clear existing tokens");
    }

    let test_token = create_test_jwt_token_with_refresh();
    let refresh_token = test_token.refresh_token.as_deref();

    // Save both access and refresh tokens
    token_store
        .save_tokens(&test_token, refresh_token)
        .expect("Failed to save tokens");
    assert!(token_store.token_exists());

    // Load access token
    let loaded_token = token_store
        .load_access_token()
        .expect("Failed to load access token");
    assert_eq!(loaded_token.token, test_token.token);

    // Load refresh token
    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(loaded_refresh, Some("test.refresh.token".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}

#[test]
fn test_keyring_token_store_clear() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");
    let test_token = create_test_jwt_token();

    // Save token
    token_store
        .save_tokens(&test_token, None)
        .expect("Failed to save token");
    assert!(token_store.token_exists());

    // Clear tokens
    token_store.clear_tokens().expect("Failed to clear tokens");
    assert!(!token_store.token_exists());
}

#[test]
fn test_keyring_token_store_load_nonexistent_access_token() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Clear any existing tokens first
    if token_store.token_exists() {
        token_store.clear_tokens().expect("Failed to clear tokens");
    }

    // Try to load non-existent token
    assert!(!token_store.token_exists());
    let result = token_store.load_access_token();
    assert!(result.is_err());
}

#[test]
fn test_keyring_token_store_load_nonexistent_refresh_token() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Clear any existing tokens first
    if token_store.token_exists() {
        token_store.clear_tokens().expect("Failed to clear tokens");
    }

    let test_token = create_test_jwt_token();

    // Save only access token (no refresh token)
    token_store
        .save_tokens(&test_token, None)
        .expect("Failed to save token");

    // Try to load non-existent refresh token
    let result = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(result, None);

    // Clean up
    token_store.clear_tokens().ok();
}

#[test]
fn test_keyring_token_store_new() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let token_store = KeyringTokenStore::new();
    assert!(token_store.is_ok());
}

#[test]
fn test_keyring_token_store_clear_when_empty() {
    if should_skip_keyring_test() {
        println!("Skipping keyring test in CI environment");
        return;
    }

    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Ensure tokens are cleared
    if token_store.token_exists() {
        token_store.clear_tokens().expect("Failed to clear tokens");
    }

    // Clearing when already empty should not error
    let result = token_store.clear_tokens();
    assert!(result.is_ok());
}
