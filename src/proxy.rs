use chrono::Utc;
use serde_json::Value;
use std::collections::HashMap;
use std::fs::OpenOptions;
use std::io::{self, BufRead, BufReader, Write};
use std::path::Path;
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::Instant;

pub fn spawn_proxy_process(program: &str, args: &[String]) -> io::Result<Child> {
    tracing::info!("Spawning proxy process: {:?}", program);
    tracing::info!("With args: {:?}", args);

    let child = Command::new(program)
        .args(args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()?;

    tracing::info!("Proxy process spawned: {:?}", child.id());
    Ok(child)
}

#[allow(dead_code)]
pub struct ProxyTelemetry {
    pub request_count: u64,
    pub response_count: u64,
    pub error_count: u64,
}

impl Default for ProxyTelemetry {
    fn default() -> Self {
        Self::new()
    }
}

impl ProxyTelemetry {
    #[allow(dead_code)]
    pub fn new() -> Self {
        Self {
            request_count: 0,
            response_count: 0,
            error_count: 0,
        }
    }
}

fn log_mcp_traffic(direction: &str, content: &str, log_file_path: &Path, duration_ms: Option<f64>) {
    if let Ok(mut file) = OpenOptions::new()
        .create(true)
        .append(true)
        .open(log_file_path)
    {
        let mut log_entry = serde_json::json!({
            "timestamp": Utc::now().to_rfc3339(),
            "direction": direction,
            "content": content,
        });

        // Add duration for response entries
        if let Some(duration) = duration_ms {
            log_entry["duration_ms"] = serde_json::json!(duration);
        }

        let _ = writeln!(file, "{}", log_entry);
    }
}

pub fn run_proxy(program: &str, args: &[String], log_file_path: &Path) -> io::Result<()> {
    let mut child = spawn_proxy_process(program, args)?;

    // Clone log file path for threads
    let log_file_path_stdin = log_file_path.to_path_buf();
    let log_file_path_stdout = log_file_path.to_path_buf();

    // Shared map to track request timestamps by request ID
    let request_timings: Arc<Mutex<HashMap<Value, Instant>>> = Arc::new(Mutex::new(HashMap::new()));
    let request_timings_stdin = request_timings.clone();
    let request_timings_stdout = request_timings;

    // we want to take ownership of the pipes
    let mut child_stdin = child
        .stdin
        .take()
        .ok_or_else(|| io::Error::other("Failed to read stdin"))?;
    let child_stdout = child
        .stdout
        .take()
        .ok_or_else(|| io::Error::other("Failed to read stdout"))?;

    let stdin_thread = thread::spawn(move || {
        let stdin = io::stdin();
        let reader = stdin.lock();

        for line in reader.lines() {
            match line {
                Ok(content) => {
                    // Log what we're forwarding (to stderr so it doesn't mix)
                    tracing::debug!("[PROXY → Child] {}", content);

                    // Log MCP traffic to file (no duration for requests)
                    log_mcp_traffic("request", &content, &log_file_path_stdin, None);

                    // Try to parse as JSON for telemetry and timing
                    if let Ok(json) = serde_json::from_str::<Value>(&content) {
                        if json.get("jsonrpc").is_some() {
                            tracing::debug!(
                                "[TELEMETRY] MCP Request detected: method={:?}",
                                json.get("method")
                            );

                            // Track request timing if it has an ID
                            if let Some(id) = json.get("id") {
                                if let Ok(mut timings) = request_timings_stdin.lock() {
                                    timings.insert(id.clone(), Instant::now());
                                }
                            }
                        }
                    }

                    // Write to child and add newline
                    if let Err(e) = writeln!(child_stdin, "{}", content) {
                        tracing::error!("Error writing to child: {}", e);
                        break;
                    }
                    // Flush to ensure it's sent immediately
                    if let Err(e) = child_stdin.flush() {
                        tracing::error!("Error flushing: {}", e);
                        break;
                    }
                }
                Err(e) => {
                    tracing::error!("Error reading stdin: {}", e);
                    break;
                }
            }
        }
        tracing::debug!("[PROXY] Input stream ended");
    });

    // Thread 2: Child stdout → Our stdout
    let stdout_thread = thread::spawn(move || {
        let reader = BufReader::new(child_stdout);

        for line in reader.lines() {
            match line {
                Ok(content) => {
                    // Log what we're receiving
                    tracing::debug!("[Child → PROXY] {}", content);

                    // Try to parse as JSON for telemetry and timing
                    let mut duration_ms: Option<f64> = None;
                    if let Ok(json) = serde_json::from_str::<Value>(&content) {
                        if json.get("jsonrpc").is_some() {
                            tracing::debug!(
                                "[TELEMETRY] MCP Response detected: id={:?}",
                                json.get("id")
                            );

                            // Calculate duration if we have a matching request
                            if let Some(id) = json.get("id") {
                                if let Ok(mut timings) = request_timings_stdout.lock() {
                                    if let Some(start_time) = timings.remove(id) {
                                        duration_ms =
                                            Some(start_time.elapsed().as_secs_f64() * 1000.0);
                                        tracing::debug!(
                                            "Request {} took {:.2}ms",
                                            id,
                                            duration_ms.unwrap()
                                        );
                                    }
                                }
                            }
                        }
                    }

                    // Log MCP traffic to file with duration if available
                    log_mcp_traffic("response", &content, &log_file_path_stdout, duration_ms);

                    // Forward to our stdout
                    println!("{}", content);
                    // Flush stdout too
                    if let Err(e) = io::stdout().flush() {
                        tracing::error!("Error flushing stdout: {}", e);
                        break;
                    }
                }
                Err(e) => {
                    tracing::error!("Error reading child stdout: {}", e);
                    break;
                }
            }
        }
        tracing::debug!("[PROXY] Output stream ended");
    });

    // Wait for both threads to finish
    let _ = stdin_thread.join();
    let _ = stdout_thread.join();

    // Then wait for child process and propagate exit status
    match child.wait() {
        Ok(status) => {
            if status.success() {
                tracing::info!("Child process exited successfully");
                Ok(())
            } else {
                tracing::error!("Child process exited with error: {:?}", status);
                Err(io::Error::other(format!(
                    "Child process failed with status: {:?}",
                    status
                )))
            }
        }
        Err(e) => {
            tracing::error!("Error waiting for child: {}", e);
            Err(e)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    #[test]
    fn test_proxy_telemetry_new() {
        let telemetry = ProxyTelemetry::new();
        assert_eq!(telemetry.request_count, 0);
        assert_eq!(telemetry.response_count, 0);
        assert_eq!(telemetry.error_count, 0);
    }

    #[test]
    fn test_proxy_telemetry_default() {
        let telemetry = ProxyTelemetry::default();
        assert_eq!(telemetry.request_count, 0);
        assert_eq!(telemetry.response_count, 0);
        assert_eq!(telemetry.error_count, 0);
    }

    #[test]
    fn test_log_mcp_traffic_request() {
        let temp_dir = TempDir::new().unwrap();
        let log_path = temp_dir.path().join("test_mcp.log");

        log_mcp_traffic(
            "request",
            r#"{"jsonrpc":"2.0","method":"test"}"#,
            &log_path,
            None,
        );

        let contents = fs::read_to_string(&log_path).unwrap();
        assert!(contents.contains("\"direction\":\"request\""));
        // Content is JSON-escaped, so check for the method
        assert!(contents.contains("jsonrpc"));
        assert!(contents.contains("method"));
        assert!(!contents.contains("duration_ms"));
    }

    #[test]
    fn test_log_mcp_traffic_response_with_duration() {
        let temp_dir = TempDir::new().unwrap();
        let log_path = temp_dir.path().join("test_mcp.log");

        log_mcp_traffic(
            "response",
            r#"{"jsonrpc":"2.0","result":"ok"}"#,
            &log_path,
            Some(123.45),
        );

        let contents = fs::read_to_string(&log_path).unwrap();
        assert!(contents.contains("\"direction\":\"response\""));
        assert!(contents.contains("\"duration_ms\":123.45"));
    }

    #[test]
    fn test_log_mcp_traffic_multiple_entries() {
        let temp_dir = TempDir::new().unwrap();
        let log_path = temp_dir.path().join("test_mcp.log");

        log_mcp_traffic("request", "request1", &log_path, None);
        log_mcp_traffic("response", "response1", &log_path, Some(100.0));
        log_mcp_traffic("request", "request2", &log_path, None);

        let contents = fs::read_to_string(&log_path).unwrap();
        let lines: Vec<&str> = contents.lines().collect();
        assert_eq!(lines.len(), 3);
    }

    #[test]
    fn test_log_mcp_traffic_creates_file_if_not_exists() {
        let temp_dir = TempDir::new().unwrap();
        let log_path = temp_dir.path().join("new_log.log");

        assert!(!log_path.exists());

        log_mcp_traffic("request", "test", &log_path, None);

        assert!(log_path.exists());
    }

    #[test]
    fn test_spawn_proxy_process_invalid_command() {
        let result = spawn_proxy_process("this-command-does-not-exist-xyz123", &[]);
        assert!(result.is_err());
    }

    #[test]
    fn test_spawn_proxy_process_with_valid_command() {
        // Use 'echo' which is available on all platforms
        let result = spawn_proxy_process("echo", &["test".to_string()]);
        assert!(result.is_ok());

        if let Ok(mut child) = result {
            // Clean up the process
            let _ = child.kill();
            let _ = child.wait();
        }
    }

    #[test]
    fn test_spawn_proxy_process_with_args() {
        let args = vec!["Hello".to_string(), "World".to_string()];
        let result = spawn_proxy_process("echo", &args);
        assert!(result.is_ok());

        if let Ok(mut child) = result {
            let _ = child.kill();
            let _ = child.wait();
        }
    }
}
