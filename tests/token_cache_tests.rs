use km::auth::JwtToken;
use km::token_cache::TokenCache;
use std::sync::Mutex;
use tempfile::TempDir;

// Mutex to serialize token cache tests to prevent interference
static TOKEN_CACHE_TEST_LOCK: Mutex<()> = Mutex::new(());

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
    }
}

#[test]
fn test_token_cache_save_and_load() {
    let _lock = TOKEN_CACHE_TEST_LOCK.lock().unwrap();
    let _temp_dir = TempDir::new().unwrap();
    // This ensures TokenCache creates its own directory structure

    let token_cache = TokenCache::new().expect("Failed to create token cache");

    // Clear any existing cache first
    if token_cache.cache_exists() {
        token_cache.clear_cache().expect("Failed to clear existing cache");
    }

    let test_token = create_test_jwt_token();

    // Save token
    token_cache.save_token(&test_token).expect("Failed to save token");
    assert!(token_cache.cache_exists());

    // Load token
    let loaded_token = token_cache.load_token().expect("Failed to load token");
    assert_eq!(loaded_token.token, test_token.token);
    assert_eq!(loaded_token.expires_at, test_token.expires_at);
    assert_eq!(loaded_token.claims.tier, test_token.claims.tier);
}

#[test]
fn test_token_cache_clear() {
    let _lock = TOKEN_CACHE_TEST_LOCK.lock().unwrap();
    let token_cache = TokenCache::new().expect("Failed to create token cache");
    let test_token = create_test_jwt_token();

    // Save token
    token_cache.save_token(&test_token).expect("Failed to save token");
    assert!(token_cache.cache_exists());

    // Clear cache
    token_cache.clear_cache().expect("Failed to clear cache");
    assert!(!token_cache.cache_exists());
}

#[test]
fn test_token_cache_load_nonexistent() {
    let _lock = TOKEN_CACHE_TEST_LOCK.lock().unwrap();
    let token_cache = TokenCache::new().expect("Failed to create token cache");

    // Clear any existing cache first
    if token_cache.cache_exists() {
        token_cache.clear_cache().expect("Failed to clear cache");
    }

    // Try to load non-existent cache
    assert!(!token_cache.cache_exists());
    let result = token_cache.load_token();
    assert!(result.is_err());
}

#[test]
fn test_token_cache_new() {
    let token_cache = TokenCache::new();
    assert!(token_cache.is_ok());

    let cache = token_cache.unwrap();
    assert!(cache.get_cache_path().to_string_lossy().contains("kilometers"));
    assert!(cache.get_cache_path().to_string_lossy().contains("km"));
}