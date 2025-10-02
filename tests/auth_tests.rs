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
