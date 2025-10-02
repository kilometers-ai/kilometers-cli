use km::filters::risk_analysis::RiskAnalysisFilter;
use km::filters::{FilterDecision, ProxyContext, ProxyFilter, ProxyRequest};
use wiremock::matchers::{header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

#[tokio::test]
async fn test_risk_analysis_filter_creation() {
    let filter = RiskAnalysisFilter::new("https://api.test.com/analyze".to_string(), 0.8);

    assert_eq!(filter.name(), "RiskAnalysis");
    assert!(!filter.is_blocking()); // Non-blocking filter
}

#[tokio::test]
async fn test_risk_analysis_low_risk_allows_request() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .and(header("authorization", "Bearer test-token"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.3,
            "risk_level": "low",
            "recommendation": "Safe to proceed",
            "details": {}
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("test-command".to_string(), vec!["arg1".to_string()]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    match decision {
        FilterDecision::Allow => {
            // Expected behavior
        }
        _ => panic!("Expected Allow decision for low risk"),
    }
}

#[tokio::test]
async fn test_risk_analysis_high_risk_blocks_request() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.95,
            "risk_level": "critical",
            "recommendation": "Block this request",
            "details": {}
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("dangerous-command".to_string(), vec![]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    match decision {
        FilterDecision::Block { reason } => {
            assert!(reason.contains("0.95"));
            assert!(reason.contains("0.8"));
            assert!(reason.contains("Block this request"));
        }
        _ => panic!("Expected Block decision for high risk"),
    }
}

#[tokio::test]
async fn test_risk_analysis_with_transform_suggestion() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.5,
            "risk_level": "medium",
            "recommendation": "Transform command for safety",
            "suggested_transform": {
                "command": "safer-command",
                "args": ["safe-arg"],
                "reason": "Using safer alternative"
            }
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("risky-command".to_string(), vec!["risky-arg".to_string()]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    match decision {
        FilterDecision::Transform { new_request } => {
            assert_eq!(new_request.command, "safer-command");
            assert_eq!(new_request.args, vec!["safe-arg"]);
        }
        _ => panic!("Expected Transform decision"),
    }
}

#[tokio::test]
async fn test_risk_analysis_partial_transform_command_only() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.4,
            "risk_level": "low",
            "recommendation": "Use safer command",
            "suggested_transform": {
                "command": "new-command",
                "reason": "Command substitution"
            }
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("old-command".to_string(), vec!["keep-arg".to_string()]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    match decision {
        FilterDecision::Transform { new_request } => {
            assert_eq!(new_request.command, "new-command");
            assert_eq!(new_request.args, vec!["keep-arg"]); // Args unchanged
        }
        _ => panic!("Expected Transform decision"),
    }
}

#[tokio::test]
async fn test_risk_analysis_partial_transform_args_only() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.4,
            "risk_level": "low",
            "recommendation": "Modify arguments",
            "suggested_transform": {
                "args": ["new-arg1", "new-arg2"],
                "reason": "Safer arguments"
            }
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("command".to_string(), vec!["old-arg".to_string()]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    match decision {
        FilterDecision::Transform { new_request } => {
            assert_eq!(new_request.command, "command"); // Command unchanged
            assert_eq!(new_request.args, vec!["new-arg1", "new-arg2"]);
        }
        _ => panic!("Expected Transform decision"),
    }
}

#[tokio::test]
async fn test_risk_analysis_api_failure() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(500))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;
    assert!(result.is_err());
    assert!(result.unwrap_err().to_string().contains("500"));
}

#[tokio::test]
async fn test_risk_analysis_api_timeout() {
    // Test with unreachable endpoint to simulate timeout
    let filter = RiskAnalysisFilter::new("http://localhost:1".to_string(), 0.8);

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;
    assert!(result.is_err());
}

#[tokio::test]
async fn test_risk_analysis_invalid_json_response() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_string("not json"))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;
    assert!(result.is_err());
}

#[tokio::test]
async fn test_risk_analysis_threshold_boundary() {
    let mock_server = MockServer::start().await;

    // Test exactly at threshold
    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.8,
            "risk_level": "medium",
            "recommendation": "At threshold",
            "details": {}
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "test-token".to_string());

    let decision = filter.check(&context).await.unwrap();

    // Should allow at threshold (not strictly greater)
    match decision {
        FilterDecision::Allow => {}
        _ => panic!("Expected Allow at threshold"),
    }
}

#[tokio::test]
async fn test_risk_analysis_sends_correct_payload() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .and(wiremock::matchers::body_json(serde_json::json!({
            "command": "my-command",
            "args": ["arg1", "arg2"],
            "metadata": {}
        })))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.1,
            "risk_level": "low",
            "recommendation": "OK"
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new(
        "my-command".to_string(),
        vec!["arg1".to_string(), "arg2".to_string()],
    );
    let context = ProxyContext::new(request, "test-token".to_string());

    let _ = filter.check(&context).await.unwrap();
    // If we get here, the mock matched, meaning payload was correct
}

#[tokio::test]
async fn test_risk_analysis_includes_bearer_token() {
    let mock_server = MockServer::start().await;

    Mock::given(method("POST"))
        .and(path("/api/risk/analyze"))
        .and(header("authorization", "Bearer my-jwt-token"))
        .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
            "risk_score": 0.1,
            "risk_level": "low",
            "recommendation": "OK"
        })))
        .mount(&mock_server)
        .await;

    let filter = RiskAnalysisFilter::new(format!("{}/api/risk/analyze", mock_server.uri()), 0.8);

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "my-jwt-token".to_string());

    let _ = filter.check(&context).await.unwrap();
    // Mock matched, meaning bearer token was sent correctly
}
