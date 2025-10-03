use km::auth::{AuthClient, JwtClaims, JwtToken};
use std::time::{SystemTime, UNIX_EPOCH};

#[test]
fn test_auth_client_creation() {
    let client = AuthClient::new(
        "test-api-key".to_string(),
        "https://test.api.com".to_string(),
    );

    // We can't access private fields, so we just ensure it constructs without panic
    assert!(format!("{:?}", client).contains("AuthClient"));
}

#[test]
fn test_jwt_claims_default() {
    let claims = JwtClaims::default();

    assert_eq!(claims.sub, None);
    assert_eq!(claims.tier, None);
    assert_eq!(claims.exp, None);
    assert_eq!(claims.iat, None);
    assert_eq!(claims.user_id, None);
}

#[test]
fn test_jwt_claims_with_values() {
    let claims = JwtClaims {
        sub: Some("user123".to_string()),
        tier: Some("pro".to_string()),
        exp: Some(1234567890),
        iat: Some(1234567800),
        user_id: Some("uid456".to_string()),
    };

    assert_eq!(claims.sub, Some("user123".to_string()));
    assert_eq!(claims.tier, Some("pro".to_string()));
    assert_eq!(claims.exp, Some(1234567890));
    assert_eq!(claims.iat, Some(1234567800));
    assert_eq!(claims.user_id, Some("uid456".to_string()));
}

#[test]
fn test_jwt_token_creation() {
    let claims = JwtClaims {
        sub: Some("test-user".to_string()),
        tier: Some("free".to_string()),
        exp: Some(1700000000),
        iat: Some(1600000000),
        user_id: Some("user-id-123".to_string()),
    };

    let token = JwtToken {
        token: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9".to_string(),
        expires_at: 1700000000,
        claims,
        refresh_token: None,
    };

    assert_eq!(token.token, "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9");
    assert_eq!(token.expires_at, 1700000000);
    assert_eq!(token.claims.sub, Some("test-user".to_string()));
}

#[test]
fn test_parse_jwt_claims_invalid_format() {
    let result = AuthClient::parse_jwt_claims("invalid.jwt");
    assert!(result.is_err());
}

#[test]
fn test_parse_jwt_claims_wrong_parts_count() {
    let result = AuthClient::parse_jwt_claims("header.payload");
    assert!(result.is_err());
}

#[test]
fn test_is_token_expired_not_expired() {
    let future_timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 3600; // 1 hour in future

    let token = JwtToken {
        token: "test-token".to_string(),
        expires_at: future_timestamp,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    assert!(!AuthClient::is_token_expired(&token));
}

#[test]
fn test_is_token_expired_is_expired() {
    let past_timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 3600; // 1 hour in past

    let token = JwtToken {
        token: "test-token".to_string(),
        expires_at: past_timestamp,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    assert!(AuthClient::is_token_expired(&token));
}

#[test]
fn test_is_token_expired_within_buffer() {
    let near_expiry = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 30; // 30 seconds in future (within 60s buffer)

    let token = JwtToken {
        token: "test-token".to_string(),
        expires_at: near_expiry,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    assert!(AuthClient::is_token_expired(&token));
}

#[test]
fn test_parse_jwt_claims_empty_payload() {
    // JWT with empty payload: header.{}.signature
    let result = AuthClient::parse_jwt_claims("eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.e30.signature");

    // Empty JSON object should parse successfully with default values
    assert!(result.is_ok());
    let claims = result.unwrap();
    assert_eq!(claims.sub, None);
    assert_eq!(claims.tier, None);
}

#[test]
fn test_parse_jwt_claims_invalid_base64() {
    let result = AuthClient::parse_jwt_claims("header.invalid!!!base64.signature");
    assert!(result.is_err());
}

#[test]
fn test_parse_jwt_claims_not_json() {
    // Valid base64 but not JSON
    let result = AuthClient::parse_jwt_claims("header.bm90IGpzb24.signature");
    assert!(result.is_err());
}

#[test]
fn test_parse_jwt_claims_with_plan_alias() {
    // Test that "plan" field maps to "tier"
    // Base64 encoded: {"plan": "enterprise"}
    let jwt = "header.eyJwbGFuIjogImVudGVycHJpc2UifQ.signature";
    let result = AuthClient::parse_jwt_claims(jwt);

    assert!(result.is_ok());
    let claims = result.unwrap();
    assert_eq!(claims.tier, Some("enterprise".to_string()));
}

#[test]
fn test_parse_jwt_claims_with_customer_id_alias() {
    // Test that "customer_id" field maps to "user_id"
    // Base64 encoded: {"customer_id": "cust123"}
    let jwt = "header.eyJjdXN0b21lcl9pZCI6ICJjdXN0MTIzIn0.signature";
    let result = AuthClient::parse_jwt_claims(jwt);

    assert!(result.is_ok());
    let claims = result.unwrap();
    assert_eq!(claims.user_id, Some("cust123".to_string()));
}

#[test]
fn test_parse_jwt_claims_full_token() {
    // Test with all fields populated
    // {"sub": "user", "tier": "pro", "exp": 1234567890, "iat": 1234567800, "user_id": "uid"}
    let jwt = "header.eyJzdWIiOiAidXNlciIsICJ0aWVyIjogInBybyIsICJleHAiOiAxMjM0NTY3ODkwLCAiaWF0IjogMTIzNDU2NzgwMCwgInVzZXJfaWQiOiAidWlkIn0.signature";
    let result = AuthClient::parse_jwt_claims(jwt);

    assert!(result.is_ok());
    let claims = result.unwrap();
    assert_eq!(claims.sub, Some("user".to_string()));
    assert_eq!(claims.tier, Some("pro".to_string()));
    assert_eq!(claims.exp, Some(1234567890));
    assert_eq!(claims.iat, Some(1234567800));
    assert_eq!(claims.user_id, Some("uid".to_string()));
}

#[test]
fn test_parse_jwt_claims_extra_fields_ignored() {
    // Test that extra unknown fields are ignored
    // {"sub": "user", "unknown_field": "ignored"}
    let jwt = "header.eyJzdWIiOiAidXNlciIsICJ1bmtub3duX2ZpZWxkIjogImlnbm9yZWQifQ.signature";
    let result = AuthClient::parse_jwt_claims(jwt);

    assert!(result.is_ok());
    let claims = result.unwrap();
    assert_eq!(claims.sub, Some("user".to_string()));
}

#[test]
fn test_jwt_token_with_refresh_token() {
    let claims = JwtClaims::default();

    let token = JwtToken {
        token: "access-token".to_string(),
        expires_at: 1700000000,
        claims,
        refresh_token: Some("refresh-token-value".to_string()),
    };

    assert_eq!(token.refresh_token, Some("refresh-token-value".to_string()));
}

#[test]
fn test_jwt_token_serialization() {
    let claims = JwtClaims {
        sub: Some("test-user".to_string()),
        tier: Some("pro".to_string()),
        exp: Some(1700000000),
        iat: Some(1600000000),
        user_id: Some("user-123".to_string()),
    };

    let token = JwtToken {
        token: "jwt-value".to_string(),
        expires_at: 1700000000,
        claims,
        refresh_token: Some("refresh".to_string()),
    };

    let json = serde_json::to_string(&token).unwrap();
    assert!(json.contains("jwt-value"));
    assert!(json.contains("refresh"));
    assert!(json.contains("test-user"));
}

#[test]
fn test_jwt_token_deserialization() {
    let json = r#"{
        "token": "jwt-value",
        "expires_at": 1700000000,
        "claims": {
            "sub": "test-user",
            "tier": "pro",
            "exp": 1700000000,
            "iat": 1600000000,
            "user_id": "user-123"
        },
        "refresh_token": "refresh-value"
    }"#;

    let token: JwtToken = serde_json::from_str(json).unwrap();

    assert_eq!(token.token, "jwt-value");
    assert_eq!(token.expires_at, 1700000000);
    assert_eq!(token.claims.sub, Some("test-user".to_string()));
    assert_eq!(token.claims.tier, Some("pro".to_string()));
    assert_eq!(token.refresh_token, Some("refresh-value".to_string()));
}

#[test]
fn test_jwt_token_deserialization_without_refresh() {
    let json = r#"{
        "token": "jwt-value",
        "expires_at": 1700000000,
        "claims": {
            "sub": "test-user"
        }
    }"#;

    let token: JwtToken = serde_json::from_str(json).unwrap();

    assert_eq!(token.token, "jwt-value");
    assert_eq!(token.refresh_token, None);
}

#[test]
fn test_is_token_expired_edge_case_59_seconds() {
    let now_plus_59 = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 59; // 59 seconds in future (within 60s buffer)

    let token = JwtToken {
        token: "test-token".to_string(),
        expires_at: now_plus_59,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    // Should be considered expired due to 60s buffer
    assert!(AuthClient::is_token_expired(&token));
}

#[test]
fn test_is_token_expired_edge_case_61_seconds() {
    let now_plus_61 = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 61; // 61 seconds in future (outside 60s buffer)

    let token = JwtToken {
        token: "test-token".to_string(),
        expires_at: now_plus_61,
        claims: JwtClaims::default(),
        refresh_token: None,
    };

    // Should NOT be considered expired
    assert!(!AuthClient::is_token_expired(&token));
}
