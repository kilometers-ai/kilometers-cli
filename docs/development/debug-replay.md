# Debug Replay Feature

The Debug Replay feature allows you to replay MCP JSON-RPC requests from a file when running `km monitor`. This is useful for testing, debugging, and creating reproducible scenarios without needing a real MCP server.

## Usage

```bash
km monitor --debug-replay <file> --server -- echo "dummy command"
```

### Flags

- `--debug-replay <file>`: Path to the replay file containing JSON-RPC messages
- `--server -- <command>`: Required server command (can be a dummy command like `echo`)

**Note**: When using debug replay, the specified server command is not actually executed. The `--server --` syntax is required for command structure consistency, but debug replay takes precedence and plays back the events from the file instead.

## Replay File Format

The replay file should contain one JSON-RPC message per line, with support for:
- JSON-RPC 2.0 messages (requests, responses, notifications, errors)
- `DELAY:` commands to control timing
- Comments (lines starting with `#`)
- Empty lines (ignored)

### Example Replay File

```jsonl
# Initialize MCP connection
{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"0.1.0"},"id":1}

DELAY: 500ms

# Linear issue search request
{"jsonrpc":"2.0","method":"issues/search","params":{"query":"status:open"},"id":2}

# Response with search results
{"jsonrpc":"2.0","result":{"issues":[{"id":"issue-1","title":"Fix login bug"},{"id":"issue-2","title":"Add dark mode"}]},"id":2}

DELAY: 1s

# Create new issue
{"jsonrpc":"2.0","method":"issues/create","params":{"title":"New feature request","description":"Add user preferences"},"id":3}
```

## Common Use Cases

### 1. Testing MCP Server Implementations
```bash
# Test your MCP server with known good/bad inputs
km monitor --debug-replay test_scenarios.jsonl --server -- python -m my_mcp_server
```

### 2. Debugging Integration Issues
```bash
# Replay problematic scenarios for analysis
km monitor --debug-replay problematic_session.jsonl --server -- echo "replay mode"
```

### 3. Performance Testing
```bash
# Test with high-volume message patterns
km monitor --debug-replay load_test.jsonl --server -- echo "load test"
```

### 4. Documentation and Training
```bash
# Show typical MCP interaction patterns
km monitor --debug-replay example_workflow.jsonl --server -- echo "demo"
```

## Delay Commands

Control the timing between messages using `DELAY:` commands:

```jsonl
# Quick succession
{"jsonrpc":"2.0","method":"ping","id":1}
{"jsonrpc":"2.0","result":"pong","id":1}

DELAY: 100ms

# Normal timing
{"jsonrpc":"2.0","method":"status","id":2}

DELAY: 1s

# Slow operation
{"jsonrpc":"2.0","method":"search","params":{"query":"large dataset"},"id":3}

DELAY: 5s

{"jsonrpc":"2.0","result":{"items":[...]},"id":3}
```

## Recording Live Sessions

You can capture real MCP interactions for later replay:

```bash
# Monitor a real session and capture to file
km monitor --server -- npx -y @modelcontextprotocol/server-linear > session_capture.jsonl

# Later, replay the captured session
km monitor --debug-replay session_capture.jsonl --server -- echo "replay"
```

## Integration with Testing

Debug replay integrates well with automated testing:

```go
func TestMCPServerBehavior(t *testing.T) {
    // Run km monitor with debug replay
    cmd := exec.Command("km", "monitor", 
        "--debug-replay", "test/fixtures/linear_search.jsonl",
        "--server", "--", "echo", "test")
    
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err)
    
    // Verify expected monitoring behavior
    assert.Contains(t, string(output), "Session completed")
}
```

## Tips and Best Practices

### Creating Replay Files
1. **Start Simple**: Begin with basic ping/pong patterns
2. **Add Realistic Delays**: Use `DELAY:` commands to simulate real timing
3. **Include Error Cases**: Test both success and failure scenarios
4. **Document Purpose**: Use comments to explain each test scenario

### Debugging
- Use `--debug` flag for verbose output during replay
- Check that JSON-RPC messages are properly formatted
- Verify that IDs match between requests and responses

### Performance
- Large replay files are supported (tested up to thousands of messages)
- Delay commands help prevent overwhelming the monitoring system
- Consider batch sizes when replaying high-volume scenarios

---

The debug replay feature provides a powerful way to test, debug, and demonstrate MCP monitoring capabilities without requiring complex server setups or external dependencies. 