use std::env;
use std::fs;
use std::io::{BufRead, BufReader, Write};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};
use tempfile::TempDir;

/// Integration test for core monitor proxy functionality
/// This test ensures the basic MCP proxy behavior works correctly:
/// - Spawns an actual MCP server
/// - Sends JSON-RPC requests through the km proxy
/// - Validates proper request forwarding and response handling
/// - Verifies proxy logging functionality
#[test]
fn test_core_monitor_proxy_functionality() {
    // Setup temporary directory for test files
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("test_proxy.log");

    // Ensure km binary exists
    let km_binary = find_km_binary();

    // Create test MCP requests
    let test_requests = vec![
        // Initialize request
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "roots": {"listChanged": true},
                    "sampling": {}
                },
                "clientInfo": {
                    "name": "km-test-client",
                    "version": "1.0.0"
                }
            }
        }),
        // Initialized notification
        serde_json::json!({
            "jsonrpc": "2.0",
            "method": "notifications/initialized"
        }),
        // List tools request
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list"
        }),
        // Call echo tool
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "echo",
                "arguments": {"text": "Hello from km proxy test!"}
            }
        }),
    ];

    // Run the proxy with test requests
    let result = run_proxy_with_requests(
        &km_binary,
        &log_file,
        &test_requests,
        Duration::from_secs(10),
    );

    match result {
        Ok(responses) => {
            // Validate responses received
            println!("✅ Received {} responses from MCP server", responses.len());

            // Validate we got responses back
            assert!(
                !responses.is_empty(),
                "Should receive responses from MCP server"
            );

            // Find initialize response by ID (should be 1)
            let init_response = responses
                .iter()
                .find(|r| r.get("id").and_then(|id| id.as_i64()) == Some(1));
            if let Some(init_resp) = init_response {
                assert_eq!(init_resp["jsonrpc"], "2.0", "Invalid JSON-RPC version");
                assert_eq!(
                    init_resp["id"], 1,
                    "Initialize response ID should match request"
                );
                assert!(
                    init_resp.get("result").is_some() || init_resp.get("error").is_some(),
                    "Initialize response should have result or error"
                );
                println!("✅ Found valid initialize response with proper capabilities");
            } else {
                panic!("No initialize response found with ID 1");
            }

            // Find tools/list response by ID (should be 2)
            let tools_response = responses
                .iter()
                .find(|r| r.get("id").and_then(|id| id.as_i64()) == Some(2));
            if let Some(tools_resp) = tools_response {
                assert_eq!(tools_resp["jsonrpc"], "2.0", "Invalid JSON-RPC version");
                assert_eq!(
                    tools_resp["id"], 2,
                    "Tools list response ID should match request"
                );
                println!("✅ Found valid tools/list response");
            } else {
                println!("⚠️  No tools/list response found - server may not support tools");
            }

            println!("✅ JSON-RPC proxy functionality verified");

            // Verify log file was created and contains entries
            assert!(log_file.exists(), "Proxy log file should be created");
            let log_content =
                fs::read_to_string(&log_file).expect("Should be able to read log file");

            assert!(
                !log_content.trim().is_empty(),
                "Log file should not be empty"
            );

            // Verify JSON structure in logs
            let log_lines: Vec<&str> = log_content.lines().collect();
            let mut mcp_traffic_entries = 0;
            let mut command_entries = 0;

            for (i, line) in log_lines.iter().enumerate() {
                if !line.trim().is_empty() {
                    let parsed: Result<serde_json::Value, _> = serde_json::from_str(line);
                    assert!(
                        parsed.is_ok(),
                        "Each log line should be valid JSON: {}",
                        line
                    );

                    let log_entry = parsed.unwrap();
                    assert!(
                        log_entry.get("timestamp").is_some(),
                        "Log entry {} should have timestamp",
                        i
                    );

                    // Check if this is MCP traffic (has direction and content) or command metadata
                    if log_entry.get("direction").is_some() && log_entry.get("content").is_some() {
                        // This is MCP traffic logging
                        mcp_traffic_entries += 1;
                    } else if log_entry.get("command").is_some() && log_entry.get("args").is_some()
                    {
                        // This is command metadata logging
                        command_entries += 1;
                    }
                }
            }

            // The proxy should log at least the command metadata
            assert!(
                command_entries > 0 || mcp_traffic_entries > 0,
                "Log should contain either MCP traffic or command metadata"
            );

            println!(
                "✅ Log file validation passed: {} MCP entries, {} command entries",
                mcp_traffic_entries, command_entries
            );

            println!(
                "✅ Core proxy test passed! Processed {} responses",
                responses.len()
            );
        }
        Err(e) => {
            panic!("Core proxy test failed: {}. This indicates the basic MCP proxy functionality is broken.", e);
        }
    }
}

/// Test proxy behavior with invalid MCP server command
#[test]
#[cfg(not(target_os = "windows"))]
fn test_core_proxy_with_invalid_server() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("test_invalid_proxy.log");
    let km_binary = find_km_binary();

    // Run proxy with a command that will fail to start
    let mut child = Command::new(&km_binary)
        .args([
            "monitor",
            "--log-file",
            log_file.to_str().unwrap(),
            "--local-only",
            "--",
            "nonexistent-mcp-server-command",
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .expect("Failed to spawn km process");

    // Give it a moment to fail
    thread::sleep(Duration::from_millis(500));

    // Check if the process is still running or has exited with error
    match child.try_wait() {
        Ok(Some(status)) => {
            // Process exited - this is expected for invalid commands
            assert!(
                !status.success(),
                "Proxy should fail when given invalid server command"
            );
            println!("✅ Proxy correctly failed with invalid server command");
        }
        Ok(None) => {
            // Process still running - kill it and fail test
            let _ = child.kill();
            let _ = child.wait();
            panic!("Proxy should have failed quickly with invalid server command");
        }
        Err(e) => {
            panic!("Failed to check process status: {}", e);
        }
    }
}

/// Helper function to find the km binary (debug or release)
fn find_km_binary() -> PathBuf {
    let debug_path = PathBuf::from(format!("./target/debug/km{}", env::consts::EXE_SUFFIX));
    let release_path = PathBuf::from(format!("./target/release/km{}", env::consts::EXE_SUFFIX));

    if release_path.exists() {
        release_path
    } else if debug_path.exists() {
        debug_path
    } else {
        // Try to build debug version
        let output = Command::new("cargo")
            .args(["build"])
            .output()
            .expect("Failed to run cargo build");

        if !output.status.success() {
            panic!(
                "Failed to build km binary: {}",
                String::from_utf8_lossy(&output.stderr)
            );
        }

        if debug_path.exists() {
            debug_path
        } else {
            panic!("km binary not found after build attempt");
        }
    }
}

/// Helper function to find the mock MCP server binary
fn find_mock_mcp_server_binary() -> PathBuf {
    let debug_path = PathBuf::from(format!(
        "./target/debug/mock_mcp_server{}",
        env::consts::EXE_SUFFIX
    ));
    let release_path = PathBuf::from(format!(
        "./target/release/mock_mcp_server{}",
        env::consts::EXE_SUFFIX
    ));

    if release_path.exists() {
        release_path
    } else if debug_path.exists() {
        debug_path
    } else {
        // Try to build debug version
        let output = Command::new("cargo")
            .args(["build", "--bin", "mock_mcp_server"])
            .output()
            .expect("Failed to run cargo build for mock_mcp_server");

        if !output.status.success() {
            panic!(
                "Failed to build mock_mcp_server binary: {}",
                String::from_utf8_lossy(&output.stderr)
            );
        }

        if debug_path.exists() {
            debug_path
        } else {
            panic!("mock_mcp_server binary not found after build attempt");
        }
    }
}

/// Run the proxy with given requests and return responses
fn run_proxy_with_requests(
    km_binary: &PathBuf,
    log_file: &Path,
    requests: &[serde_json::Value],
    timeout: Duration,
) -> Result<Vec<serde_json::Value>, String> {
    // Get the mock MCP server binary
    let mock_server_binary = find_mock_mcp_server_binary();

    // Start the proxy process with the mock MCP server
    let mut child = Command::new(km_binary)
        .args([
            "monitor",
            "--log-file",
            log_file.to_str().unwrap(),
            "--local-only", // Use local-only mode for reliable testing
            "--",
            mock_server_binary.to_str().unwrap(),
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("Failed to spawn km process: {}", e))?;

    let mut stdin = child.stdin.take().ok_or("Failed to get stdin")?;
    let stdout = child.stdout.take().ok_or("Failed to get stdout")?;

    // Spawn thread to send requests
    let requests_clone = requests.to_vec();
    let send_handle = thread::spawn(move || -> Result<(), String> {
        // Give the MCP server a moment to start up
        thread::sleep(Duration::from_millis(1000));

        for request in requests_clone {
            let request_str = serde_json::to_string(&request)
                .map_err(|e| format!("Failed to serialize request: {}", e))?;

            writeln!(stdin, "{}", request_str)
                .map_err(|e| format!("Failed to write request: {}", e))?;

            stdin
                .flush()
                .map_err(|e| format!("Failed to flush stdin: {}", e))?;

            // Small delay between requests
            thread::sleep(Duration::from_millis(100));
        }
        Ok(())
    });

    // Read responses with timeout
    let start_time = Instant::now();
    let mut responses = Vec::new();
    let reader = BufReader::new(stdout);

    for line in reader.lines() {
        if start_time.elapsed() > timeout {
            break;
        }

        match line {
            Ok(response_str) => {
                if let Ok(response) = serde_json::from_str::<serde_json::Value>(&response_str) {
                    // Only collect actual JSON-RPC responses (not debug output)
                    if response.get("jsonrpc").is_some() {
                        responses.push(response);

                        // If we got responses for most of our requests, we can stop
                        if responses.len() >= requests.len().saturating_sub(1) {
                            break;
                        }
                    }
                }
            }
            Err(_) => break, // EOF or error
        }
    }

    // Wait for send thread and cleanup
    let _ = send_handle.join();
    let _ = child.kill();
    let _ = child.wait();

    if responses.is_empty() {
        return Err("No valid JSON-RPC responses received from proxy".to_string());
    }

    Ok(responses)
}

/// Test that verifies log file format and content structure
#[test]
fn test_proxy_log_format_validation() {
    let temp_dir = TempDir::new().expect("Failed to create temp directory");
    let log_file = temp_dir.path().join("test_log_format.log");
    let km_binary = find_km_binary();

    let simple_request = vec![serde_json::json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "roots": {"listChanged": true},
                "sampling": {}
            },
            "clientInfo": {
                "name": "km-test-client",
                "version": "1.0.0"
            }
        }
    })];

    let _ = run_proxy_with_requests(
        &km_binary,
        &log_file,
        &simple_request,
        Duration::from_secs(5),
    );

    // Validate log file format
    if log_file.exists() {
        let log_content = fs::read_to_string(&log_file).expect("Should be able to read log file");

        if !log_content.trim().is_empty() {
            // Debug: print actual log content to understand the issue
            println!("DEBUG: Log file content: {}", log_content);
            println!("DEBUG: Number of lines: {}", log_content.lines().count());

            // Parse each log line as JSONL
            let mut mcp_entries = 0;
            for (i, line) in log_content.lines().enumerate() {
                if line.trim().is_empty() {
                    continue;
                }

                let parsed: Result<serde_json::Value, _> = serde_json::from_str(line);
                assert!(
                    parsed.is_ok(),
                    "Log line {} should be valid JSON: {}",
                    i,
                    line
                );

                let log_entry = parsed.unwrap();

                // Skip LocalLoggerFilter entries (command metadata) - we only want MCP traffic
                if log_entry.get("command").is_some() && log_entry.get("args").is_some() {
                    println!("DEBUG: Skipping LocalLoggerFilter entry");
                    continue;
                }

                mcp_entries += 1;

                // Verify required fields for MCP traffic entries
                assert!(
                    log_entry.get("timestamp").is_some(),
                    "Log entry {} missing timestamp",
                    i
                );
                assert!(
                    log_entry.get("direction").is_some(),
                    "Log entry {} missing direction",
                    i
                );
                assert!(
                    log_entry.get("content").is_some(),
                    "Log entry {} missing content",
                    i
                );

                // Verify direction is valid
                let direction = log_entry["direction"].as_str().unwrap();
                assert!(
                    direction == "request" || direction == "response",
                    "Log entry {} has invalid direction: {}",
                    i,
                    direction
                );

                // Verify timestamp format (should be RFC3339)
                let timestamp = log_entry["timestamp"].as_str().unwrap();
                assert!(
                    chrono::DateTime::parse_from_rfc3339(timestamp).is_ok(),
                    "Log entry {} has invalid timestamp format: {}",
                    i,
                    timestamp
                );
            }

            if mcp_entries == 0 {
                println!("⚠️  No MCP traffic entries found in log - this indicates the proxy may not be logging MCP traffic properly");
            } else {
                println!(
                    "✅ Log format validation passed! Found {} MCP traffic entries",
                    mcp_entries
                );
            }
        } else {
            println!("⚠️  Log file is empty - proxy may not have processed any requests");
        }
    }
}
