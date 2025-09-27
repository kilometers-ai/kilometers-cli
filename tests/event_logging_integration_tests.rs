use anyhow::Result;
use km::auth::{JwtClaims, JwtToken};
use km::filters::event_sender::EventSenderFilter;
use km::filters::local_logger::LocalLoggerFilter;
use km::filters::{FilterDecision, FilterPipeline, ProxyContext, ProxyFilter, ProxyRequest};
use serde_json::json;
use serde_json::Value;
use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::sync::Arc;
use tempfile::TempDir;
use tokio::task;
use wiremock::matchers::{header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

fn create_mock_jwt_token(user_id: Option<String>, tier: Option<String>) -> JwtToken {
    let claims = JwtClaims {
        sub: Some("test-subject".to_string()),
        exp: Some(9999999999),
        iat: Some(1234567890),
        user_id,
        tier,
    };

    JwtToken {
        token: "mock-jwt-token".to_string(),
        expires_at: 9999999999,
        claims,
    }
}

fn create_test_context(command: &str, args: Vec<&str>) -> ProxyContext {
    let mut request = ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata: HashMap::new(),
    };

    request
        .metadata
        .insert("test_session".to_string(), "session123".to_string());

    ProxyContext::new(request, "test-token".to_string())
}

// Mock filter that transforms requests for testing
struct MockTransformFilter;

#[async_trait::async_trait]
impl ProxyFilter for MockTransformFilter {
    async fn check(&self, ctx: &ProxyContext) -> Result<FilterDecision> {
        let mut new_request = ctx.request.clone();
        new_request.command = format!("transformed-{}", ctx.request.command);
        new_request.args.push("--transformed".to_string());

        Ok(FilterDecision::Transform { new_request })
    }

    fn name(&self) -> &str {
        "MockTransform"
    }
}

#[tokio::test]
async fn test_filter_pipeline_with_event_logging() {
    let mock_server = MockServer::start().await;
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("integration_test.log");

    // Setup mock server
    Mock::given(method("POST"))
        .and(path("/"))
        .and(header("authorization", "Bearer mock-jwt-token"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "status": "success",
            "events_remaining": 1000
        })))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(
        Some("integration-user".to_string()),
        Some("enterprise".to_string()),
    );

    // Create pipeline with both logging filters
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_path.clone())))
        .add_filter(Box::new(EventSenderFilter::new(
            mock_server.uri(),
            jwt_token,
        )));

    let context = create_test_context("integration-test", vec!["arg1", "arg2"]);

    let result = pipeline.execute(context).await;

    assert!(result.is_ok());
    let final_request = result.unwrap();
    assert_eq!(final_request.command, "integration-test");
    assert_eq!(final_request.args, vec!["arg1", "arg2"]);

    // Verify local log was written
    assert!(log_path.exists());
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 1);
    let log_entry: Value = serde_json::from_str(&lines[0]).unwrap();
    assert_eq!(
        log_entry.get("command").unwrap().as_str().unwrap(),
        "integration-test"
    );
}

#[tokio::test]
async fn test_pipeline_with_transformation() {
    let mock_server = MockServer::start().await;
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("transform_test.log");

    Mock::given(method("POST"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({"status": "success"})))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);

    // Pipeline: LocalLogger -> Transform -> EventSender
    // LocalLogger should log the original request, EventSender should send the transformed one
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_path.clone())))
        .add_filter(Box::new(MockTransformFilter))
        .add_filter(Box::new(EventSenderFilter::new(
            mock_server.uri(),
            jwt_token,
        )));

    let context = create_test_context("original-command", vec!["original-arg"]);

    let result = pipeline.execute(context).await;

    assert!(result.is_ok());
    let final_request = result.unwrap();

    // Final request should be transformed
    assert_eq!(final_request.command, "transformed-original-command");
    assert!(final_request.args.contains(&"--transformed".to_string()));

    // Local log should have the original command (since LocalLogger runs before transform)
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 1);
    let log_entry: Value = serde_json::from_str(&lines[0]).unwrap();
    assert_eq!(
        log_entry.get("command").unwrap().as_str().unwrap(),
        "original-command"
    );
}

#[tokio::test]
async fn test_non_blocking_behavior_with_failures() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("failure_test.log");

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);

    // Use invalid URL for EventSender to simulate network failure
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(log_path.clone())))
        .add_filter(Box::new(EventSenderFilter::new(
            "http://invalid-url:99999".to_string(),
            jwt_token,
        )));

    let context = create_test_context("test-command", vec!["arg1"]);

    let result = pipeline.execute(context).await;

    // Should still succeed despite EventSender failure (both filters are non-blocking)
    assert!(result.is_ok());
    let final_request = result.unwrap();
    assert_eq!(final_request.command, "test-command");

    // Local log should still be written
    assert!(log_path.exists());
}

#[tokio::test]
async fn test_concurrent_pipeline_execution() {
    let mock_server = MockServer::start().await;
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("concurrent_test.log");

    Mock::given(method("POST"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({"status": "success"})))
        .mount(&mock_server)
        .await;

    let jwt_token = create_mock_jwt_token(Some("concurrent-user".to_string()), None);

    let pipeline = Arc::new(
        FilterPipeline::new()
            .add_filter(Box::new(LocalLoggerFilter::new(log_path.clone())))
            .add_filter(Box::new(EventSenderFilter::new(
                mock_server.uri(),
                jwt_token,
            ))),
    );

    let mut tasks = Vec::new();

    // Execute pipeline concurrently
    for i in 0..5 {
        let pipeline_clone = Arc::clone(&pipeline);
        let task = task::spawn(async move {
            let context =
                create_test_context(&format!("concurrent-cmd-{}", i), vec![&format!("arg{}", i)]);
            pipeline_clone.execute(context).await
        });
        tasks.push(task);
    }

    let results = futures::future::join_all(tasks).await;

    // All executions should succeed
    for (i, result) in results.into_iter().enumerate() {
        let pipeline_result = result.unwrap();
        assert!(pipeline_result.is_ok());
        let final_request = pipeline_result.unwrap();
        assert_eq!(final_request.command, format!("concurrent-cmd-{}", i));
    }

    // Verify all entries were logged
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 5);

    // Verify each entry is valid JSON and has correct structure
    for line in lines {
        let entry: Value = serde_json::from_str(&line).unwrap();
        assert!(entry
            .get("command")
            .unwrap()
            .as_str()
            .unwrap()
            .starts_with("concurrent-cmd-"));
        assert!(entry.get("timestamp").is_some());
    }
}

#[tokio::test]
async fn test_different_jwt_token_scenarios() {
    let mock_server = MockServer::start().await;
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("jwt_test.log");

    Mock::given(method("POST"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({"status": "success"})))
        .mount(&mock_server)
        .await;

    // Test scenarios with different JWT configurations
    let test_cases = vec![
        ("No user ID, no tier", create_mock_jwt_token(None, None)),
        (
            "With user ID, no tier",
            create_mock_jwt_token(Some("user123".to_string()), None),
        ),
        (
            "No user ID, with tier",
            create_mock_jwt_token(None, Some("premium".to_string())),
        ),
        (
            "Full JWT",
            create_mock_jwt_token(Some("user456".to_string()), Some("enterprise".to_string())),
        ),
    ];

    for (test_name, jwt_token) in test_cases {
        let pipeline = FilterPipeline::new()
            .add_filter(Box::new(LocalLoggerFilter::new(log_path.clone())))
            .add_filter(Box::new(EventSenderFilter::new(
                mock_server.uri(),
                jwt_token,
            )));

        let context = create_test_context("jwt-test", vec!["test"]);
        let result = pipeline.execute(context).await;

        assert!(result.is_ok(), "Failed for case: {}", test_name);
    }

    // Verify all requests were logged
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 4);
}

#[tokio::test]
async fn test_pipeline_error_resilience() {
    let temp_dir = TempDir::new().unwrap();
    let invalid_log_path = temp_dir.path().join("nonexistent_dir").join("test.log");

    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);

    // Pipeline with potentially failing components
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(LocalLoggerFilter::new(invalid_log_path))) // May fail to write
        .add_filter(Box::new(EventSenderFilter::new(
            "http://invalid:99999".to_string(),
            jwt_token,
        ))); // Will fail network

    let context = create_test_context("resilience-test", vec!["arg1"]);

    let result = pipeline.execute(context).await;

    // Should still succeed despite both filters potentially failing (non-blocking)
    assert!(result.is_ok());
    let final_request = result.unwrap();
    assert_eq!(final_request.command, "resilience-test");
}

#[test]
fn test_filter_properties() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test.log");
    let jwt_token = create_mock_jwt_token(Some("user123".to_string()), None);

    let local_logger = LocalLoggerFilter::new(log_path);
    let event_sender = EventSenderFilter::new("http://example.com".to_string(), jwt_token);

    // Both filters should be non-blocking
    assert!(!local_logger.is_blocking());
    assert!(!event_sender.is_blocking());

    // Verify filter names
    assert_eq!(local_logger.name(), "LocalLogger");
    assert_eq!(event_sender.name(), "EventSender");
}
