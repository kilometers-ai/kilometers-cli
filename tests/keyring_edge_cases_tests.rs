use km::auth::{JwtClaims, JwtToken};
use km::keyring_token_store::KeyringTokenStore;
use std::sync::Mutex;

// Mutex to serialize keyring tests to prevent interference
static KEYRING_TEST_LOCK: Mutex<()> = Mutex::new(());

#[tokio::test]
async fn test_keyring_handles_very_long_tokens() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a very long token (simulating a JWT with lots of claims)
    let long_token_string = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.".to_string()
        + &"a".repeat(5000) // Very long payload
        + ".signature";

    let long_token = JwtToken {
        token: long_token_string.clone(),
        expires_at: 9999999999,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-long-token".to_string()),
        },
        refresh_token: None,
    };

    // Should be able to save and load long tokens
    token_store
        .save_tokens(&long_token, None)
        .expect("Failed to save long token");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load long token");
    assert_eq!(loaded.token, long_token_string);

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_handles_special_characters_in_claims() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a token with special characters in claims
    let special_token = JwtToken {
        token: "special.chars.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims {
            sub: Some("user@example.com".to_string()),
            tier: Some("premium+enterprise".to_string()),
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("user-123-abc_XYZ".to_string()),
        },
        refresh_token: Some("refresh/token+with=special&chars".to_string()),
    };

    token_store
        .save_tokens(&special_token, special_token.refresh_token.as_deref())
        .expect("Failed to save token with special characters");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token with special characters");

    assert_eq!(loaded.claims.sub, Some("user@example.com".to_string()));
    assert_eq!(loaded.claims.tier, Some("premium+enterprise".to_string()));
    assert_eq!(loaded.claims.user_id, Some("user-123-abc_XYZ".to_string()));

    let loaded_refresh = token_store
        .load_refresh_token()
        .expect("Failed to load refresh token");
    assert_eq!(
        loaded_refresh,
        Some("refresh/token+with=special&chars".to_string())
    );

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_handles_unicode_in_claims() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a token with unicode characters
    let unicode_token = JwtToken {
        token: "unicode.token.test".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims {
            sub: Some("用户".to_string()),     // Chinese characters
            tier: Some("премиум".to_string()), // Cyrillic characters
            exp: Some(9999999999),
            iat: Some(1234567890),
            user_id: Some("🚀-user-123".to_string()), // Emoji
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&unicode_token, None)
        .expect("Failed to save unicode token");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load unicode token");

    assert_eq!(loaded.claims.sub, Some("用户".to_string()));
    assert_eq!(loaded.claims.tier, Some("премиум".to_string()));
    assert_eq!(loaded.claims.user_id, Some("🚀-user-123".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_handles_empty_string_values() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a token with empty string values (unusual but valid)
    let empty_token = JwtToken {
        token: "".to_string(), // Empty token string
        expires_at: 0,
        claims: JwtClaims {
            sub: Some("".to_string()),
            tier: Some("".to_string()),
            exp: Some(0),
            iat: Some(0),
            user_id: Some("".to_string()),
        },
        refresh_token: Some("".to_string()),
    };

    token_store
        .save_tokens(&empty_token, empty_token.refresh_token.as_deref())
        .expect("Failed to save token with empty strings");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token with empty strings");

    assert_eq!(loaded.token, "");
    assert_eq!(loaded.claims.sub, Some("".to_string()));
    assert_eq!(loaded.claims.tier, Some("".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_handles_max_timestamp_values() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Create a token with maximum timestamp values
    let max_timestamp_token = JwtToken {
        token: "max.timestamp.token".to_string(),
        expires_at: u64::MAX,
        claims: JwtClaims {
            sub: Some("test-user".to_string()),
            tier: Some("premium".to_string()),
            exp: Some(u64::MAX),
            iat: Some(u64::MAX),
            user_id: Some("user-max-ts".to_string()),
        },
        refresh_token: None,
    };

    token_store
        .save_tokens(&max_timestamp_token, None)
        .expect("Failed to save token with max timestamps");

    let loaded = token_store
        .load_access_token()
        .expect("Failed to load token with max timestamps");

    assert_eq!(loaded.expires_at, u64::MAX);
    assert_eq!(loaded.claims.exp, Some(u64::MAX));
    assert_eq!(loaded.claims.iat, Some(u64::MAX));

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_concurrent_access_safety() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    // Note: This test is serialized by the mutex, so we can't truly test
    // concurrent access. However, we can verify that multiple sequential
    // operations work correctly.

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // Perform multiple rapid save/load operations
    for i in 0..10 {
        let token = JwtToken {
            token: format!("concurrent.token.{}", i),
            expires_at: 9999999999,
            claims: JwtClaims {
                user_id: Some(format!("user-{}", i)),
                ..Default::default()
            },
            refresh_token: None,
        };

        token_store
            .save_tokens(&token, None)
            .unwrap_or_else(|_| panic!("Failed to save token {}", i));

        let loaded = token_store
            .load_access_token()
            .unwrap_or_else(|_| panic!("Failed to load token {}", i));

        assert_eq!(loaded.token, format!("concurrent.token.{}", i));
    }

    // Clean up
    token_store.clear_tokens().ok();
}

#[tokio::test]
async fn test_keyring_initialization_is_idempotent() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Creating multiple instances of KeyringTokenStore should work fine
    let store1 = KeyringTokenStore::new().expect("Failed to create first store");
    let store2 = KeyringTokenStore::new().expect("Failed to create second store");
    let store3 = KeyringTokenStore::new().expect("Failed to create third store");

    // Clean up any existing tokens
    if store1.token_exists() {
        store1.clear_tokens().ok();
    }

    // All instances should see the same underlying keyring data
    let token = JwtToken {
        token: "shared.token".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    store1
        .save_tokens(&token, None)
        .expect("Failed to save with store1");

    // All stores should see the token
    assert!(store1.token_exists());
    assert!(store2.token_exists());
    assert!(store3.token_exists());

    // All stores should be able to load it
    let loaded1 = store1
        .load_access_token()
        .expect("Failed to load with store1");
    let loaded2 = store2
        .load_access_token()
        .expect("Failed to load with store2");
    let loaded3 = store3
        .load_access_token()
        .expect("Failed to load with store3");

    assert_eq!(loaded1.token, "shared.token");
    assert_eq!(loaded2.token, "shared.token");
    assert_eq!(loaded3.token, "shared.token");

    // Clean up with any store
    store1.clear_tokens().ok();

    // All stores should see it's gone
    assert!(!store1.token_exists());
    assert!(!store2.token_exists());
    assert!(!store3.token_exists());
}

#[tokio::test]
async fn test_keyring_handles_only_refresh_token_update() {
    let _lock = KEYRING_TEST_LOCK.lock().unwrap();

    // Clean up any existing tokens first
    if let Ok(token_store) = KeyringTokenStore::new() {
        if token_store.token_exists() {
            token_store.clear_tokens().ok();
        }
    }

    let token_store = KeyringTokenStore::new().expect("Failed to create keyring token store");

    // First, save a token without refresh token
    let token_without_refresh = JwtToken {
        token: "access.token.only".to_string(),
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    token_store
        .save_tokens(&token_without_refresh, None)
        .expect("Failed to save token without refresh");

    // Verify no refresh token
    let refresh1 = token_store.load_refresh_token().expect("Should not error");
    assert_eq!(refresh1, None);

    // Now save with a refresh token
    let token_with_refresh = JwtToken {
        token: "access.token.only".to_string(), // Same access token
        expires_at: 9999999999,
        claims: JwtClaims::default(),
        refresh_token: Some("new.refresh.token".to_string()),
    };

    token_store
        .save_tokens(
            &token_with_refresh,
            token_with_refresh.refresh_token.as_deref(),
        )
        .expect("Failed to save token with refresh");

    // Verify refresh token is now present
    let refresh2 = token_store.load_refresh_token().expect("Should not error");
    assert_eq!(refresh2, Some("new.refresh.token".to_string()));

    // Clean up
    token_store.clear_tokens().ok();
}
