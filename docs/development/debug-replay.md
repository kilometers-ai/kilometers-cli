# Debug Replay Feature

The Debug Replay feature allows you to replay MCP JSON-RPC requests from a file when running `km monitor`. This is useful for testing, debugging, and creating reproducible scenarios without needing a real MCP server.

## Usage

```bash
km monitor --debug-replay <file> [--debug-delay <duration>] <dummy-command>
```

### Flags

- `--debug-replay <file>`: Path to the replay file containing JSON-RPC messages
- `--debug-delay <duration>`: Default delay between messages (default: 500ms)

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

# List available tools
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}

# Tool list response
{"jsonrpc":"2.0","result":{"tools":[{"name":"read_file","description":"Read a file"}]},"id":2}

DELAY: 1s

# Call a tool
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read_file","arguments":{"path":"test.txt"}},"id":3}

# Error response
{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"},"id":4}

# Notification (no id)
{"jsonrpc":"2.0","method":"log","params":{"level":"info","message":"Test completed"}}
```

## Examples

### Basic Usage

```bash
# Replay messages with default 500ms delay between each
km monitor --debug-replay test/messages.jsonl dummy-process

# Replay with custom delay
km monitor --debug-replay test/messages.jsonl --debug-delay 1s dummy-process
```

### Testing Large Payloads

Create a replay file with large JSON payloads to test buffer handling:

```jsonl
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search","arguments":{"query":"test","data":"<very large string>"}},"id":1}
```

### Simulating Real MCP Server Behavior

```jsonl
# Client initializes
{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{}},"id":1}

DELAY: 100ms

# Server responds
{"jsonrpc":"2.0","result":{"protocolVersion":"0.1.0","capabilities":{"tools":{}}},"id":1}

DELAY: 200ms

# Client lists tools
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}

# Continue with more interactions...
```

## Use Cases

1. **Testing**: Create specific test scenarios without needing real MCP servers
2. **Debugging**: Reproduce issues with exact message sequences
3. **Development**: Test km monitor behavior with various message patterns
4. **Performance Testing**: Replay high-volume message sequences
5. **CI/CD**: Automated testing without external dependencies

## Tips

- Use `DELAY:` commands to simulate realistic timing between messages
- Include both requests and responses to simulate full conversations
- Test edge cases like malformed JSON, large payloads, and error responses
- Comments help document what each message represents

## Limitations

- The replay file must contain valid JSON-RPC 2.0 messages (except for DELAY commands and comments)
- Messages are replayed sequentially
- No support for conditional logic or branching
- The dummy command argument is required but not actually executed 