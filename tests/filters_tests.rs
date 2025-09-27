use anyhow::Result;
use async_trait::async_trait;
use km::filters::{FilterDecision, FilterPipeline, ProxyContext, ProxyFilter, ProxyRequest};

// Mock filter for testing
struct MockAllowFilter;

#[async_trait]
impl ProxyFilter for MockAllowFilter {
    async fn check(&self, _ctx: &ProxyContext) -> Result<FilterDecision> {
        Ok(FilterDecision::Allow)
    }

    fn name(&self) -> &str {
        "MockAllow"
    }
}

// Mock blocking filter for testing
struct MockBlockFilter;

#[async_trait]
impl ProxyFilter for MockBlockFilter {
    async fn check(&self, _ctx: &ProxyContext) -> Result<FilterDecision> {
        Ok(FilterDecision::Block {
            reason: "Test block reason".to_string(),
        })
    }

    fn name(&self) -> &str {
        "MockBlock"
    }
}

// Mock transform filter for testing
struct MockTransformFilter;

#[async_trait]
impl ProxyFilter for MockTransformFilter {
    async fn check(&self, _ctx: &ProxyContext) -> Result<FilterDecision> {
        let new_request =
            ProxyRequest::new("transformed-command".to_string(), vec!["arg1".to_string()]);
        Ok(FilterDecision::Transform { new_request })
    }

    fn name(&self) -> &str {
        "MockTransform"
    }
}

#[test]
fn test_proxy_request_creation() {
    let request = ProxyRequest::new(
        "test-command".to_string(),
        vec!["arg1".to_string(), "arg2".to_string()],
    );

    assert_eq!(request.command, "test-command");
    assert_eq!(request.args, vec!["arg1", "arg2"]);
    assert!(request.metadata.is_empty());
}

#[test]
fn test_proxy_context_creation() {
    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request.clone(), "jwt-token".to_string());

    assert_eq!(context.request.command, request.command);
    assert_eq!(context.jwt_token, "jwt-token");
    assert!(context.metadata.is_empty());
}

#[test]
fn test_filter_pipeline_creation() {
    let pipeline = FilterPipeline::new();
    // We can't directly test the internal filters vec, but we can ensure it constructs
    let _ = pipeline;
}

#[tokio::test]
async fn test_filter_pipeline_with_allow_filter() {
    let pipeline = FilterPipeline::new().add_filter(Box::new(MockAllowFilter));

    let request = ProxyRequest::new("test".to_string(), vec!["arg".to_string()]);
    let context = ProxyContext::new(request.clone(), "token".to_string());

    let result = pipeline.execute(context).await.unwrap();
    assert_eq!(result.command, "test");
    assert_eq!(result.args, vec!["arg"]);
}

#[tokio::test]
async fn test_filter_pipeline_with_block_filter() {
    let pipeline = FilterPipeline::new().add_filter(Box::new(MockBlockFilter));

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "token".to_string());

    let result = pipeline.execute(context).await;
    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("Request blocked by MockBlock"));
    assert!(error_msg.contains("Test block reason"));
}

#[tokio::test]
async fn test_filter_pipeline_with_transform_filter() {
    let pipeline = FilterPipeline::new().add_filter(Box::new(MockTransformFilter));

    let request = ProxyRequest::new("original".to_string(), vec!["original-arg".to_string()]);
    let context = ProxyContext::new(request, "token".to_string());

    let result = pipeline.execute(context).await.unwrap();
    assert_eq!(result.command, "transformed-command");
    assert_eq!(result.args, vec!["arg1"]);
}

#[tokio::test]
async fn test_filter_pipeline_multiple_filters() {
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(MockAllowFilter))
        .add_filter(Box::new(MockTransformFilter));

    let request = ProxyRequest::new("original".to_string(), vec!["original-arg".to_string()]);
    let context = ProxyContext::new(request, "token".to_string());

    let result = pipeline.execute(context).await.unwrap();
    // The transform filter should modify the request
    assert_eq!(result.command, "transformed-command");
    assert_eq!(result.args, vec!["arg1"]);
}

#[tokio::test]
async fn test_filter_pipeline_block_stops_execution() {
    let pipeline = FilterPipeline::new()
        .add_filter(Box::new(MockBlockFilter))
        .add_filter(Box::new(MockTransformFilter)); // This should never execute

    let request = ProxyRequest::new("test".to_string(), vec![]);
    let context = ProxyContext::new(request, "token".to_string());

    let result = pipeline.execute(context).await;
    assert!(result.is_err());
    assert!(result.unwrap_err().to_string().contains("MockBlock"));
}
