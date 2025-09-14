use km::domain::proxy::{LogEntry, LogEntryType};
use km::infrastructure::log_repository::LogRepository;
use std::path::PathBuf;
use tempfile::TempDir;

struct TestLogRepository {
    _temp_dir: TempDir,
    log_path: PathBuf,
}

impl TestLogRepository {
    fn new() -> Self {
        let temp_dir = TempDir::new().unwrap();
        let log_path = temp_dir.path().join("test_mcp_proxy.log");

        Self {
            _temp_dir: temp_dir,
            log_path,
        }
    }

    fn create_repository(&self) -> LogRepository {
        // Use the mock log path for testing
        struct MockLogRepository {
            log_path: String,
        }

        impl MockLogRepository {
            fn new(log_path: String) -> Self {
                Self { log_path }
            }

            fn write_entry(&self, entry: &LogEntry) -> anyhow::Result<()> {
                use std::fs::create_dir_all;
                use std::fs::OpenOptions;
                use std::io::Write;

                if let Some(parent) = PathBuf::from(&self.log_path).parent() {
                    create_dir_all(parent)?;
                }

                let mut log_file = OpenOptions::new()
                    .create(true)
                    .append(true)
                    .open(&self.log_path)?;

                writeln!(log_file, "{}", entry.format_for_log())?;
                Ok(())
            }

            async fn clear_logs(&self) -> anyhow::Result<()> {
                match std::fs::remove_file(&self.log_path) {
                    Ok(()) => {
                        println!("Successfully cleared logs from: {}", self.log_path);
                        Ok(())
                    }
                    Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                        println!("No log file found at: {} (already clear)", self.log_path);
                        Ok(())
                    }
                    Err(e) => Err(anyhow::anyhow!("Failed to clear logs: {}", e)),
                }
            }
        }

        // Create a real LogRepository but we'll manually override the path for testing
        let repo = LogRepository::new();

        // For testing, we need to work with the actual LogRepository struct
        // Since we can't modify the internal path, we'll test the behavior indirectly
        repo
    }
}

#[cfg(test)]
mod log_repository_tests {
    use super::*;
    use chrono::Utc;

    #[tokio::test]
    async fn test_log_entry_creation_request() {
        let entry = LogEntry::request("test request content".to_string());

        assert!(matches!(entry.entry_type, LogEntryType::Request));
        assert_eq!(entry.content, "test request content");
        assert!((Utc::now() - entry.timestamp).num_seconds() < 2);
    }

    #[tokio::test]
    async fn test_log_entry_creation_response() {
        let entry = LogEntry::response("test response content".to_string());

        assert!(matches!(entry.entry_type, LogEntryType::Response));
        assert_eq!(entry.content, "test response content");
        assert!((Utc::now() - entry.timestamp).num_seconds() < 2);
    }

    #[tokio::test]
    async fn test_log_entry_format_for_log() {
        let entry = LogEntry::request("test content".to_string());
        let formatted = entry.format_for_log();

        assert!(formatted.contains("REQUEST"));
        assert!(formatted.contains("test content"));
        assert!(formatted.contains(&entry.timestamp.to_rfc3339()));
    }

    #[tokio::test]
    async fn test_log_entry_format_for_response() {
        let entry = LogEntry::response("response content".to_string());
        let formatted = entry.format_for_log();

        assert!(formatted.contains("RESPONSE"));
        assert!(formatted.contains("response content"));
    }

    #[tokio::test]
    async fn test_repository_creation() {
        let repo = LogRepository::new();
        // Test that repository can be created without panicking
        // The actual path testing is difficult due to platform differences
        assert_eq!(format!("{:?}", repo).contains("LogRepository"), true);
    }

    #[test]
    fn test_log_entry_clone() {
        let entry1 = LogEntry::request("original content".to_string());
        let entry2 = entry1.clone();

        assert_eq!(entry1.content, entry2.content);
        assert_eq!(entry1.timestamp, entry2.timestamp);
        assert!(matches!(entry2.entry_type, LogEntryType::Request));
    }

    #[test]
    fn test_log_entry_debug_format() {
        let entry = LogEntry::request("debug test".to_string());
        let debug_str = format!("{:?}", entry);

        assert!(debug_str.contains("LogEntry"));
        assert!(debug_str.contains("debug test"));
        assert!(debug_str.contains("Request"));
    }

    #[test]
    fn test_log_entry_type_debug() {
        let request_type = LogEntryType::Request;
        let response_type = LogEntryType::Response;

        assert_eq!(format!("{:?}", request_type), "Request");
        assert_eq!(format!("{:?}", response_type), "Response");
    }

    #[test]
    fn test_log_entry_type_clone() {
        let request_type = LogEntryType::Request;
        let cloned_type = request_type.clone();

        assert!(matches!(cloned_type, LogEntryType::Request));
    }

    #[tokio::test]
    async fn test_log_entry_with_special_characters() {
        let special_content = "content with\nnewlines\tand\ttabs and \"quotes\"";
        let entry = LogEntry::request(special_content.to_string());
        let formatted = entry.format_for_log();

        assert!(formatted.contains(special_content));
        assert!(formatted.contains("REQUEST"));
    }

    #[tokio::test]
    async fn test_log_entry_with_empty_content() {
        let entry = LogEntry::request("".to_string());
        let formatted = entry.format_for_log();

        assert!(formatted.contains("REQUEST"));
        // Should still have timestamp and format structure
        assert!(formatted.len() > 10);
    }

    #[tokio::test]
    async fn test_log_entry_with_json_content() {
        let json_content = r#"{"method": "test", "params": {"key": "value"}}"#;
        let entry = LogEntry::response(json_content.to_string());
        let formatted = entry.format_for_log();

        assert!(formatted.contains("RESPONSE"));
        assert!(formatted.contains(json_content));
    }
}
