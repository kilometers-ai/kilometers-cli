use serde_json::{json, Value};
use std::io::{self, BufRead, BufReader, Write};

/// A simple mock MCP server that responds to basic JSON-RPC requests
/// This is used for integration testing without external dependencies
fn main() -> io::Result<()> {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut stdout_lock = stdout.lock();

    // Read from stdin and respond to JSON-RPC requests
    let reader = BufReader::new(stdin.lock());

    for line in reader.lines() {
        let line = line?;
        if line.trim().is_empty() {
            continue;
        }

        // Parse the JSON-RPC request
        match serde_json::from_str::<Value>(&line) {
            Ok(request) => {
                let response = handle_request(&request);
                if let Some(resp) = response {
                    let response_str = serde_json::to_string(&resp).unwrap();
                    writeln!(stdout_lock, "{}", response_str)?;
                    stdout_lock.flush()?;
                }
            }
            Err(_) => {
                // Invalid JSON, ignore
                continue;
            }
        }
    }

    Ok(())
}

fn handle_request(request: &Value) -> Option<Value> {
    let method = request.get("method")?.as_str()?;
    let id = request.get("id");

    match method {
        "initialize" => Some(json!({
            "jsonrpc": "2.0",
            "id": id,
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {
                        "listChanged": true
                    },
                    "resources": {
                        "subscribe": true,
                        "listChanged": true
                    },
                    "prompts": {
                        "listChanged": true
                    },
                    "logging": {}
                },
                "serverInfo": {
                    "name": "mock-mcp-server",
                    "version": "1.0.0"
                }
            }
        })),
        "notifications/initialized" => {
            // Notifications don't get responses
            None
        }
        "tools/list" => Some(json!({
            "jsonrpc": "2.0",
            "id": id,
            "result": {
                "tools": [
                    {
                        "name": "echo",
                        "description": "Echo back the provided text",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "text": {
                                    "type": "string",
                                    "description": "Text to echo back"
                                }
                            },
                            "required": ["text"]
                        }
                    }
                ]
            }
        })),
        "tools/call" => {
            let params = request.get("params")?;
            let tool_name = params.get("name")?.as_str()?;
            let arguments = params.get("arguments")?;

            match tool_name {
                "echo" => {
                    let text = arguments
                        .get("text")?
                        .as_str()
                        .unwrap_or("No text provided");
                    Some(json!({
                        "jsonrpc": "2.0",
                        "id": id,
                        "result": {
                            "content": [
                                {
                                    "type": "text",
                                    "text": format!("Echo: {}", text)
                                }
                            ],
                            "isError": false
                        }
                    }))
                }
                _ => {
                    // Unknown tool
                    Some(json!({
                        "jsonrpc": "2.0",
                        "id": id,
                        "error": {
                            "code": -32601,
                            "message": "Unknown tool",
                            "data": {
                                "toolName": tool_name
                            }
                        }
                    }))
                }
            }
        }
        _ => {
            // Unknown method
            Some(json!({
                "jsonrpc": "2.0",
                "id": id,
                "error": {
                    "code": -32601,
                    "message": "Method not found",
                    "data": {
                        "method": method
                    }
                }
            }))
        }
    }
}
