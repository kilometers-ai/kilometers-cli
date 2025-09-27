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

fn log_mcp_traffic(
    direction: &str,
    content: &str,
    log_file_path: &Path,
    duration_ms: Option<f64>,
) {
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

pub fn run_proxy(
    program: &str,
    args: &[String],
    log_file_path: &Path,
) -> io::Result<()> {
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
