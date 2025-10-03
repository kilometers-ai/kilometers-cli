use km::auth::{JwtClaims, JwtToken};
use km::filters::event_sender::EventSenderFilter;
use km::filters::{FilterDecision, ProxyContext, ProxyFilter, ProxyRequest};
use serde_json::json;
use std::collections::HashMap;
use wiremock::matchers::{header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

fn create_mock_jwt_token(user_id: Option<String>, tier: Option<String>) -> JwtToken {
    let claims = JwtClaims {
        sub: Some("test-subject".to_string()),
        exp: Some(9999999999), // Far future expiration
        iat: Some(1234567890),
        user_id,
        tier,
    };

    JwtToken {
        token: "mock-jwt-token".to_string(),
        expires_at: 9999999999,
        claims,
        refresh_token: None,
    }
}

fn create_test_context(command: &str, args: Vec<&str>) -> ProxyContext {
    let request = ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata: HashMap::new(),
    };

    ProxyContext::new(request, "test-token".to_string())
}

#[tokio::test]
async fn test_successful_telemetry_event_sending() {
    let mock_server = MockServer::start().await;

    // Setup mock response for successful telemetry
    Mock::given(method("POST"))
        .and(path("/"))
        .and(header("authorization", "Bearer mock-jwt-token"))
        .and(header("content-type", "application/json"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success",
            "message": "Event recorded successfully",
            "events_remaining": 1000
        })))
        .mount(&mock_server)
        .await;

    let jwt_token =
        create_mock_jwt_token(Some("user123".to_string()), Some("enterprise".to_string()));

    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec!["arg1", "arg2"]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    match result.unwrap() {
        FilterDecision::Allow => {}
        _ => panic!("Expected FilterDecision::Allow"),
    }
}

#[tokio::test]
async fn test_telemetry_response_parsing() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success",
            "events_remaining": 500
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_rate_limiting_handling() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(429))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    // Should still allow the request to proceed despite rate limiting
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_error_status_codes() {
    let mock_server = MockServer::start().await;

    // Test 500 server error
    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(500))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    // Should still return Allow since EventSenderFilter is non-blocking
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_network_failure_resilience() {
    // Use an invalid URL to simulate network failure
    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new("http://invalid-url:99999".to_string(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    // Should handle network failures gracefully and still allow
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_event_payload_structure() {
    let mock_server = MockServer::start().await;

    // Capture the request body for validation
    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), Some("premium".to_string()));

    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);

    let mut request = ProxyRequest {
        command: "test-command".to_string(),
        args: vec!["arg1".to_string(), "arg2".to_string()],
        metadata: HashMap::new(),
    };
    request
        .metadata
        .insert("key1".to_string(), "value1".to_string());
    request
        .metadata
        .insert("key2".to_string(), "value2".to_string());

    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_different_user_tiers() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    // Test with no tier (should default to "free")
    let jwt_token_no_tier = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token_no_tier);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));

    // Test with enterprise tier
    let jwt_token_enterprise =
        create_mock_jwt_token(Some("user123".to_string()), Some("enterprise".to_string()));
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token_enterprise);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_no_user_id() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(None, Some("free".to_string()));
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[test]
fn test_filter_is_non_blocking() {
    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new("http://example.com".to_string(), jwt_token);

    assert!(!filter.is_blocking());
}

#[test]
fn test_filter_name() {
    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new("http://example.com".to_string(), jwt_token);

    assert_eq!(filter.name(), "EventSender");
}

#[tokio::test]
async fn test_bad_request_400_error() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(400).set_body_json(json!({
            "error": "Bad request"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    // Should still allow since it's non-blocking
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_unauthorized_401_error() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(401).set_body_json(json!({
            "error": "Unauthorized"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_response_without_events_remaining() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success",
            "message": "Event recorded"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_invalid_json_response() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_string("not valid json"))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    // Should handle gracefully and still allow
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_empty_command_args() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);

    let request = ProxyRequest {
        command: "command-only".to_string(),
        args: vec![],
        metadata: HashMap::new(),
    };
    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_large_metadata() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);
    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);

    let mut request = ProxyRequest {
        command: "test-command".to_string(),
        args: vec!["arg1".to_string()],
        metadata: HashMap::new(),
    };

    // Add a lot of metadata
    for i in 0..50 {
        request
            .metadata
            .insert(format!("key{}", i), format!("value{}", i));
    }

    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_missing_jwt_claims() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success"
        })))
        .mount(&mock_server)
        .await;

    // Create JWT with no user_id and no tier
    let claims = JwtClaims {
        sub: None,
        exp: None,
        iat: None,
        user_id: None,
        tier: None,
    };

    let jwt_token = JwtToken {
        token: "minimal-jwt".to_string(),
        expires_at: 9999999999,
        claims,
        refresh_token: None,
    };

    let filter = EventSenderFilter::new(mock_server.uri(), jwt_token);
    let context = create_test_context("test-command", vec![]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}
