use km::config::Config;
use km::handlers::{handle_clear_logs, handle_logs, handle_show_config};
use std::fs;
use std::path::PathBuf;
use tempfile::TempDir;

#[test]
fn test_handle_show_config_missing_config() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    let result = handle_show_config(&config_path, false);
    assert!(result.is_ok());
}

#[test]
fn test_handle_show_config_with_config_no_secrets() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    // Create a config
    let config = Config::new(
        "test-api-key-12345678".to_string(),
        "https://api.test.com".to_string(),
    );
    config.save(&config_path).unwrap();

    let result = handle_show_config(&config_path, false);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&config_path).ok();
}

#[test]
fn test_handle_show_config_with_secrets() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    // Create a config
    let config = Config::new(
        "test-api-key-12345678".to_string(),
        "https://api.test.com".to_string(),
    );
    config.save(&config_path).unwrap();

    let result = handle_show_config(&config_path, true);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&config_path).ok();
}

#[test]
fn test_handle_show_config_with_short_key() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    // Create a config with a short key (< 8 chars)
    let config = Config::new("short".to_string(), "https://api.test.com".to_string());
    config.save(&config_path).unwrap();

    let result = handle_show_config(&config_path, false);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&config_path).ok();
}

#[test]
fn test_handle_clear_logs_no_files() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    let result = handle_clear_logs(false, &config_path);
    assert!(result.is_ok());
}

#[test]
fn test_handle_clear_logs_with_log_files() {
    let temp_dir = TempDir::new().unwrap();
    std::env::set_current_dir(&temp_dir).unwrap();

    // Create some log files
    fs::write("mcp_traffic.jsonl", "test content").unwrap();
    fs::write("mcp_proxy.log", "test content").unwrap();

    let config_path = temp_dir.path().join("km_config.json");

    let result = handle_clear_logs(false, &config_path);
    assert!(result.is_ok());
    assert!(!PathBuf::from("mcp_traffic.jsonl").exists());
    assert!(!PathBuf::from("mcp_proxy.log").exists());
}

#[test]
fn test_handle_clear_logs_with_config() {
    let temp_dir = TempDir::new().unwrap();
    let config_path = temp_dir.path().join("km_config.json");

    // Create a config
    let config = Config::new("test-key".to_string(), "https://api.test.com".to_string());
    config.save(&config_path).unwrap();
    assert!(config_path.exists());

    let result = handle_clear_logs(true, &config_path);
    assert!(result.is_ok());
    assert!(!config_path.exists());
}

#[test]
fn test_handle_logs_missing_file() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("nonexistent.log");

    let result = handle_logs(log_file, false, false, None, false, None);
    assert!(result.is_err());
    let err_msg = result.unwrap_err().to_string();
    assert!(err_msg.contains("Log file") || err_msg.contains("not found"));
}

#[test]
fn test_handle_logs_with_valid_file() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    // Create a test log file with JSON lines
    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"test\"}"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"response","content":"{\"jsonrpc\":\"2.0\",\"result\":\"ok\"}"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_filter_requests_only() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"test\"}"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"response","content":"{\"jsonrpc\":\"2.0\",\"result\":\"ok\"}"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), true, false, None, false, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_filter_responses_only() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"test\"}"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"response","content":"{\"jsonrpc\":\"2.0\",\"result\":\"ok\"}"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, true, None, false, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_filter_by_method() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"initialize\"}"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"request","content":"{\"jsonrpc\":\"2.0\",\"method\":\"test\"}"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(
        log_file.clone(),
        false,
        false,
        Some("initialize".to_string()),
        false,
        None,
    );
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_with_line_limit() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content = r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"line1"}
{"timestamp":"2024-01-01T00:00:01Z","direction":"request","content":"line2"}
{"timestamp":"2024-01-01T00:00:02Z","direction":"request","content":"line3"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, Some(2));
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_with_tail_mode() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    let log_content =
        r#"{"timestamp":"2024-01-01T00:00:00Z","direction":"request","content":"test"}"#;
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, true, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_with_invalid_json() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("test.log");

    // Create a log file with invalid JSON
    let log_content = "not valid json\ninvalid line";
    fs::write(&log_file, log_content).unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, None);
    // Should succeed but filter out invalid lines
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}

#[test]
fn test_handle_logs_empty_file() {
    let temp_dir = TempDir::new().unwrap();
    let log_file = temp_dir.path().join("empty.log");

    fs::write(&log_file, "").unwrap();

    let result = handle_logs(log_file.clone(), false, false, None, false, None);
    assert!(result.is_ok());

    // Clean up
    fs::remove_file(&log_file).ok();
}
