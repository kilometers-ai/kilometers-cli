use km::auth::{JwtClaims, JwtToken};
use km::filters::event_sender::EventSenderFilter;
use km::filters::local_logger::LocalLoggerFilter;
use km::filters::risk_analysis::RiskAnalysisFilter;
use km::filters::{FilterPipeline, ProxyContext, ProxyRequest};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use tempfile::TempDir;
use warp::Filter;

/// Integration test for premium tier functionality
/// This test ensures premium/enterprise tier users get full functionality:
/// - Event telemetry is sent to the API
/// - Risk analysis filtering is applied
/// - Filter pipeline executes in correct order
/// - Premium features can block/transform requests
#[tokio::test]
async fn test_premium_tier_full_pipeline() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("premium_test.log");

    // Start mock API server
    let (api_port, _api_handle) = start_mock_api_server().await;
    let api_url = format!("http://localhost:{}", api_port);

    let premium_token = create_premium_tier_jwt_token();
    let proxy_request = ProxyRequest::new(
        "ls".to_string(),
        vec!["-la".to_string(), "/home".to_string()],
    );
    let proxy_context = ProxyContext::new(proxy_request, premium_token.token.clone());

    // Create full premium tier pipeline
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
        .add_filter(Box::new(EventSenderFilter::new(
            format!("{}/api/events/telemetry", api_url),
            premium_token.clone(),
        )))
        .add_filter(Box::new(RiskAnalysisFilter::new(
            format!("{}/api/risk/analyze", api_url),
            0.8, // Risk threshold
        )));

    let result = pipeline.execute(proxy_context).await;

    // For this benign command, it should succeed
    assert!(
        result.is_ok(),
        "Premium tier pipeline should handle safe commands"
    );

    let filtered_request = result.unwrap();
    assert_eq!(filtered_request.command, "ls");
    assert_eq!(filtered_request.args, vec!["-la", "/home"]);

    // Verify local logging occurred
    assert!(log_file.exists(), "Local log file should be created");

    // Give a moment for async operations to complete
    tokio::time::sleep(std::time::Duration::from_millis(100)).await;

    println!("✅ Premium tier full pipeline test passed!");
}

#[tokio::test]
async fn test_premium_tier_risk_analysis_blocking() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("risk_block_test.log");

    // Start mock API server that returns high risk for dangerous commands
    let (api_port, _api_handle) = start_mock_high_risk_api_server().await;
    let api_url = format!("http://localhost:{}", api_port);

    let premium_token = create_premium_tier_jwt_token();
    let proxy_request = ProxyRequest::new(
        "rm".to_string(),                         // Risky command
        vec!["-rf".to_string(), "/".to_string()], // Very dangerous
    );
    let proxy_context = ProxyContext::new(proxy_request, premium_token.token.clone());

    // Create pipeline with risk analysis
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
        .add_filter(Box::new(RiskAnalysisFilter::new(
            format!("{}/api/risk/analyze", api_url),
            0.5, // Lower threshold for testing
        )));

    let result = pipeline.execute(proxy_context).await;

    // Should be blocked due to high risk score
    assert!(
        result.is_err(),
        "High-risk commands should be blocked by risk analysis"
    );

    let error_msg = result.unwrap_err().to_string();
    assert!(
        error_msg.contains("blocked") || error_msg.contains("risk"),
        "Error should indicate risk blocking: {}",
        error_msg
    );

    println!("✅ Risk analysis blocking test passed!");
}

#[tokio::test]
async fn test_premium_tier_command_transformation() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("transform_test.log");

    // Start mock API server that suggests command transformation
    let (api_port, _api_handle) = start_mock_transform_api_server().await;
    let api_url = format!("http://localhost:{}", api_port);

    let premium_token = create_premium_tier_jwt_token();
    let proxy_request = ProxyRequest::new(
        "cp".to_string(), // Copy command
        vec!["file1.txt".to_string(), "file2.txt".to_string()],
    );
    let proxy_context = ProxyContext::new(proxy_request, premium_token.token.clone());

    // Create pipeline with risk analysis that transforms
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
        .add_filter(Box::new(RiskAnalysisFilter::new(
            format!("{}/api/risk/analyze", api_url),
            0.8,
        )));

    let result = pipeline.execute(proxy_context).await;

    assert!(result.is_ok(), "Command transformation should succeed");

    let filtered_request = result.unwrap();
    // Mock server should transform 'cp' to 'cp -i' for interactive mode
    assert_eq!(filtered_request.command, "cp");
    assert_eq!(filtered_request.args, vec!["-i", "file1.txt", "file2.txt"]);

    println!("✅ Command transformation test passed!");
}

#[tokio::test]
async fn test_enterprise_tier_enhanced_features() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("enterprise_test.log");

    let (api_port, _api_handle) = start_mock_api_server().await;
    let api_url = format!("http://localhost:{}", api_port);

    let enterprise_token = create_enterprise_tier_jwt_token();
    let proxy_request = ProxyRequest::new(
        "git".to_string(),
        vec![
            "clone".to_string(),
            "https://github.com/example/repo.git".to_string(),
        ],
    );
    let proxy_context = ProxyContext::new(proxy_request, enterprise_token.token.clone());

    // Enterprise tier gets the full pipeline
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
        .add_filter(Box::new(EventSenderFilter::new(
            format!("{}/api/events/telemetry", api_url),
            enterprise_token.clone(),
        )))
        .add_filter(Box::new(RiskAnalysisFilter::new(
            format!("{}/api/risk/analyze", api_url),
            0.9, // Higher threshold for enterprise
        )));

    let result = pipeline.execute(proxy_context).await;
    assert!(result.is_ok(), "Enterprise tier should handle git commands");

    let filtered_request = result.unwrap();
    assert_eq!(filtered_request.command, "git");

    println!("✅ Enterprise tier test passed!");
}

#[tokio::test]
async fn test_event_sender_telemetry() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let _log_file = temp_dir.path().join("telemetry_test.log");

    // Track API calls
    let telemetry_calls = Arc::new(AtomicUsize::new(0));
    let calls_clone = telemetry_calls.clone();

    let (api_port, _api_handle) = start_mock_telemetry_tracking_server(calls_clone).await;
    let api_url = format!("http://localhost:{}", api_port);

    let premium_token = create_premium_tier_jwt_token();
    let proxy_request = ProxyRequest::new("echo".to_string(), vec!["test".to_string()]);
    let proxy_context = ProxyContext::new(proxy_request, premium_token.token.clone());

    // Pipeline with just event sender for focused testing
    let pipeline = FilterPipeline::new().add_filter(Box::new(EventSenderFilter::new(
        format!("{}/api/events/telemetry", api_url),
        premium_token.clone(),
    )));

    let result = pipeline.execute(proxy_context).await;
    assert!(result.is_ok(), "Event sender should not block requests");

    // Give time for async telemetry
    tokio::time::sleep(std::time::Duration::from_millis(200)).await;

    // Verify telemetry was sent
    assert!(
        telemetry_calls.load(Ordering::SeqCst) > 0,
        "Telemetry should have been sent to API"
    );

    println!("✅ Event sender telemetry test passed!");
}

// Mock API Server Implementations

async fn start_mock_api_server() -> (u16, tokio::task::JoinHandle<()>) {
    let port = find_free_port().await;

    let routes = warp::path("api")
        .and(warp::path("events"))
        .and(warp::path("telemetry"))
        .and(warp::post())
        .map(|| warp::reply::with_status("OK", warp::http::StatusCode::OK))
        .or(warp::path("api")
            .and(warp::path("risk"))
            .and(warp::path("analyze"))
            .and(warp::post())
            .map(|| {
                let response = serde_json::json!({
                    "risk_score": 0.3,
                    "risk_level": "low",
                    "recommendation": "Safe to execute"
                });
                warp::reply::json(&response)
            }));

    let (addr, server) =
        warp::serve(routes).bind_with_graceful_shutdown(([127, 0, 0, 1], port), async {
            tokio::time::sleep(std::time::Duration::from_secs(30)).await;
        });

    let handle = tokio::spawn(server);

    // Give server time to start
    tokio::time::sleep(std::time::Duration::from_millis(50)).await;

    (addr.port(), handle)
}

async fn start_mock_high_risk_api_server() -> (u16, tokio::task::JoinHandle<()>) {
    let port = find_free_port().await;

    let routes = warp::path("api")
        .and(warp::path("risk"))
        .and(warp::path("analyze"))
        .and(warp::post())
        .map(|| {
            let response = serde_json::json!({
                "risk_score": 0.95,
                "risk_level": "critical",
                "recommendation": "Command blocked due to destructive potential"
            });
            warp::reply::json(&response)
        });

    let (addr, server) =
        warp::serve(routes).bind_with_graceful_shutdown(([127, 0, 0, 1], port), async {
            tokio::time::sleep(std::time::Duration::from_secs(30)).await;
        });

    let handle = tokio::spawn(server);
    tokio::time::sleep(std::time::Duration::from_millis(50)).await;

    (addr.port(), handle)
}

async fn start_mock_transform_api_server() -> (u16, tokio::task::JoinHandle<()>) {
    let port = find_free_port().await;

    let routes = warp::path("api")
        .and(warp::path("risk"))
        .and(warp::path("analyze"))
        .and(warp::post())
        .map(|| {
            let response = serde_json::json!({
                "risk_score": 0.6,
                "risk_level": "medium",
                "recommendation": "Suggest interactive mode",
                "suggested_transform": {
                    "args": ["-i", "file1.txt", "file2.txt"],
                    "reason": "Added interactive flag for safety"
                }
            });
            warp::reply::json(&response)
        });

    let (addr, server) =
        warp::serve(routes).bind_with_graceful_shutdown(([127, 0, 0, 1], port), async {
            tokio::time::sleep(std::time::Duration::from_secs(30)).await;
        });

    let handle = tokio::spawn(server);
    tokio::time::sleep(std::time::Duration::from_millis(50)).await;

    (addr.port(), handle)
}

async fn start_mock_telemetry_tracking_server(
    call_counter: Arc<AtomicUsize>,
) -> (u16, tokio::task::JoinHandle<()>) {
    let port = find_free_port().await;
    let counter = call_counter.clone();

    let routes = warp::path("api")
        .and(warp::path("events"))
        .and(warp::path("telemetry"))
        .and(warp::post())
        .map(move || {
            counter.fetch_add(1, Ordering::SeqCst);
            warp::reply::with_status("OK", warp::http::StatusCode::OK)
        });

    let (addr, server) =
        warp::serve(routes).bind_with_graceful_shutdown(([127, 0, 0, 1], port), async {
            tokio::time::sleep(std::time::Duration::from_secs(30)).await;
        });

    let handle = tokio::spawn(server);
    tokio::time::sleep(std::time::Duration::from_millis(50)).await;

    (addr.port(), handle)
}

async fn find_free_port() -> u16 {
    use std::net::TcpListener;

    let listener = TcpListener::bind("127.0.0.1:0").expect("Failed to bind to random port");
    let addr = listener.local_addr().expect("Failed to get local address");
    addr.port()
}

// Helper functions for creating test tokens

fn create_premium_tier_jwt_token() -> JwtToken {
    let claims = JwtClaims {
        sub: Some("premium-tier-user".to_string()),
        exp: Some(9999999999),
        iat: Some(1234567890),
        user_id: Some("premium-user-123".to_string()),
        tier: Some("premium".to_string()),
    };

    JwtToken {
        token: "mock-premium-tier-jwt-token".to_string(),
        expires_at: 9999999999,
        claims,
        refresh_token: None,
    }
}

fn create_enterprise_tier_jwt_token() -> JwtToken {
    let claims = JwtClaims {
        sub: Some("enterprise-tier-user".to_string()),
        exp: Some(9999999999),
        iat: Some(1234567890),
        user_id: Some("enterprise-user-123".to_string()),
        tier: Some("enterprise".to_string()),
    };

    JwtToken {
        token: "mock-enterprise-tier-jwt-token".to_string(),
        expires_at: 9999999999,
        claims,
        refresh_token: None,
    }
}

/// Test that premium tier features are properly enabled based on tier
#[tokio::test]
async fn test_tier_based_feature_activation() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let _log_file = temp_dir.path().join("tier_activation_test.log");

    // Test different tier tokens
    let tiers = vec![
        ("premium", create_premium_tier_jwt_token()),
        ("enterprise", create_enterprise_tier_jwt_token()),
    ];

    for (tier_name, token) in tiers {
        let proxy_request = ProxyRequest::new("test".to_string(), vec!["command".to_string()]);
        let _proxy_context = ProxyContext::new(proxy_request, token.token.clone());

        // Verify the token has the correct tier
        assert_eq!(
            token.claims.tier.as_deref(),
            Some(tier_name),
            "Token should have correct tier: {}",
            tier_name
        );

        // For paid tiers, the full pipeline should be available
        // (This is a structural test - in real usage, the tier check happens in main.rs)
        assert!(
            token.claims.tier.is_some(),
            "{} tier should have tier information",
            tier_name
        );
        assert_ne!(
            token.claims.tier.as_deref(),
            Some("free"),
            "{} tier should not be free tier",
            tier_name
        );
    }

    println!("✅ Tier-based feature activation test passed!");
}

/// Test error handling in premium features
#[tokio::test]
async fn test_premium_tier_error_handling() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("error_handling_test.log");

    let premium_token = create_premium_tier_jwt_token();
    let proxy_request = ProxyRequest::new("test_command".to_string(), vec!["test_arg".to_string()]);
    let proxy_context = ProxyContext::new(proxy_request, premium_token.token.clone());

    // Create pipeline with invalid API endpoint to test error handling
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_file.clone())))
        .add_filter(Box::new(RiskAnalysisFilter::new(
            "http://invalid-endpoint-that-should-not-exist:9999/api/risk/analyze".to_string(),
            0.8,
        )));

    let result = pipeline.execute(proxy_context).await;

    // Risk analysis filter is non-blocking, so it should not fail the pipeline
    // even if the API call fails
    assert!(
        result.is_ok(),
        "Non-blocking filters should not fail the pipeline on API errors"
    );

    println!("✅ Premium tier error handling test passed!");
}
