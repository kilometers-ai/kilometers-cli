use anyhow::Result;
use km::filters::local_logger::LocalLoggerFilter;
use km::filters::{FilterDecision, ProxyContext, ProxyFilter, ProxyRequest};
use serde_json::Value;
use std::collections::HashMap;
use std::fs::{self, File};
use std::io::{BufRead, BufReader};
use std::sync::Arc;
use tempfile::TempDir;
use tokio::task;

fn create_test_context(command: &str, args: Vec<&str>) -> ProxyContext {
    let mut request = ProxyRequest {
        command: command.to_string(),
        args: args.into_iter().map(|s| s.to_string()).collect(),
        metadata: HashMap::new(),
    };

    // Add some test metadata
    request
        .metadata
        .insert("test_key".to_string(), "test_value".to_string());
    request
        .metadata
        .insert("session_id".to_string(), "session123".to_string());

    ProxyContext::new(request, "test-token".to_string())
}

#[tokio::test]
async fn test_log_file_creation() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    // Ensure file doesn't exist initially
    assert!(!log_path.exists());

    let filter = LocalLoggerFilter::new(log_path.clone());
    let context = create_test_context("test-command", vec!["arg1", "arg2"]);

    let result = filter.check(&context).await;

    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));

    // Verify file was created
    assert!(log_path.exists());

    // Verify file has content
    let metadata = fs::metadata(&log_path).unwrap();
    assert!(metadata.len() > 0);
}

#[tokio::test]
async fn test_log_entry_format() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    let filter = LocalLoggerFilter::new(log_path.clone());
    let context = create_test_context("test-command", vec!["arg1", "arg2"]);

    let result = filter.check(&context).await;
    assert!(result.is_ok());

    // Read the log file and parse the JSON
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 1);

    let log_entry: Value = serde_json::from_str(&lines[0]).unwrap();

    // Verify all required fields are present
    assert!(log_entry.get("timestamp").is_some());
    assert_eq!(
        log_entry.get("command").unwrap().as_str().unwrap(),
        "test-command"
    );
    assert_eq!(log_entry.get("args").unwrap().as_array().unwrap().len(), 2);
    assert_eq!(
        log_entry.get("user_tier").unwrap().as_str().unwrap(),
        "free"
    );
    assert!(log_entry.get("metadata").is_some());

    // Verify args content
    let args = log_entry.get("args").unwrap().as_array().unwrap();
    assert_eq!(args[0].as_str().unwrap(), "arg1");
    assert_eq!(args[1].as_str().unwrap(), "arg2");

    // Verify metadata
    let metadata = log_entry.get("metadata").unwrap();
    assert!(metadata.get("test_key").is_some());
    assert!(metadata.get("session_id").is_some());

    // Verify timestamp format (should be valid ISO-8601)
    let timestamp_str = log_entry.get("timestamp").unwrap().as_str().unwrap();
    assert!(chrono::DateTime::parse_from_rfc3339(timestamp_str).is_ok());
}

#[tokio::test]
async fn test_append_mode() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    let filter = LocalLoggerFilter::new(log_path.clone());

    // Write first entry
    let context1 = create_test_context("command1", vec!["arg1"]);
    let result1 = filter.check(&context1).await;
    assert!(result1.is_ok());

    // Write second entry
    let context2 = create_test_context("command2", vec!["arg2"]);
    let result2 = filter.check(&context2).await;
    assert!(result2.is_ok());

    // Read the log file
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 2);

    // Verify both entries
    let entry1: Value = serde_json::from_str(&lines[0]).unwrap();
    let entry2: Value = serde_json::from_str(&lines[1]).unwrap();

    assert_eq!(entry1.get("command").unwrap().as_str().unwrap(), "command1");
    assert_eq!(entry2.get("command").unwrap().as_str().unwrap(), "command2");
}

#[tokio::test]
async fn test_file_write_failures() {
    // Try to write to a read-only directory
    let temp_dir = TempDir::new().unwrap();
    let readonly_dir = temp_dir.path().join("readonly");
    fs::create_dir(&readonly_dir).unwrap();

    // Make directory read-only
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let mut perms = fs::metadata(&readonly_dir).unwrap().permissions();
        perms.set_mode(0o444); // read-only
        fs::set_permissions(&readonly_dir, perms).unwrap();
    }

    let log_path = readonly_dir.join("test_proxy.log");
    let filter = LocalLoggerFilter::new(log_path);
    let context = create_test_context("test-command", vec!["arg1"]);

    let result = filter.check(&context).await;

    // Should still return Allow despite write failure (non-blocking filter)
    assert!(result.is_ok());
    assert!(matches!(result.unwrap(), FilterDecision::Allow));
}

#[tokio::test]
async fn test_concurrent_writes() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    let filter = Arc::new(LocalLoggerFilter::new(log_path.clone()));
    let mut tasks = Vec::new();

    // Spawn multiple concurrent write tasks
    for i in 0..10 {
        let filter_clone = Arc::clone(&filter);
        let task = task::spawn(async move {
            let context = create_test_context(&format!("command{}", i), vec![&format!("arg{}", i)]);
            filter_clone.check(&context).await
        });
        tasks.push(task);
    }

    // Wait for all tasks to complete
    let results = futures::future::join_all(tasks).await;

    // All tasks should succeed
    for result in results {
        let filter_result = result.unwrap();
        assert!(filter_result.is_ok());
        assert!(matches!(filter_result.unwrap(), FilterDecision::Allow));
    }

    // Verify all entries were written
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 10);

    // Verify each line is valid JSON
    for line in lines {
        let entry: Value = serde_json::from_str(&line).unwrap();
        assert!(entry.get("command").is_some());
        assert!(entry.get("timestamp").is_some());
    }
}

#[tokio::test]
async fn test_empty_metadata() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    let filter = LocalLoggerFilter::new(log_path.clone());

    let request = ProxyRequest {
        command: "test-command".to_string(),
        args: vec!["arg1".to_string()],
        metadata: HashMap::new(), // Empty metadata
    };
    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;
    assert!(result.is_ok());

    // Read and verify the log entry
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 1);

    let log_entry: Value = serde_json::from_str(&lines[0]).unwrap();
    let metadata = log_entry.get("metadata").unwrap();

    // Should be an empty object
    assert!(metadata.is_object());
    assert_eq!(metadata.as_object().unwrap().len(), 0);
}

#[tokio::test]
async fn test_large_metadata() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");

    let filter = LocalLoggerFilter::new(log_path.clone());

    let mut request = ProxyRequest {
        command: "test-command".to_string(),
        args: vec!["arg1".to_string()],
        metadata: HashMap::new(),
    };

    // Add lots of metadata
    for i in 0..100 {
        request.metadata.insert(
            format!("key_{}", i),
            format!("value_{}_with_some_longer_content_to_test_serialization", i),
        );
    }

    let context = ProxyContext::new(request, "test-token".to_string());

    let result = filter.check(&context).await;
    assert!(result.is_ok());

    // Verify the entry was written successfully
    let file = File::open(&log_path).unwrap();
    let reader = BufReader::new(file);
    let lines: Vec<String> = reader.lines().collect::<Result<_, _>>().unwrap();

    assert_eq!(lines.len(), 1);

    let log_entry: Value = serde_json::from_str(&lines[0]).unwrap();
    let metadata = log_entry.get("metadata").unwrap().as_object().unwrap();

    assert_eq!(metadata.len(), 100);
}

#[test]
fn test_filter_is_non_blocking() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");
    let filter = LocalLoggerFilter::new(log_path);

    assert!(!filter.is_blocking());
}

#[test]
fn test_filter_name() {
    let temp_dir = TempDir::new().unwrap();
    let log_path = temp_dir.path().join("test_proxy.log");
    let filter = LocalLoggerFilter::new(log_path);

    assert_eq!(filter.name(), "LocalLogger");
}
