# Cursor Prompt: Fix km Tool MCP Message Processing Issues

## Context
I need to fix critical MCP (Model Context Protocol) message processing issues in the kilometers-cli project. The km tool is failing to parse MCP JSON-RPC messages correctly when monitoring Linear MCP server and other MCP servers that return large JSON payloads.

## Project Structure
- Base directory: `/projects/kilometers.ai/kilometers-cli`
- Architecture: Domain-Driven Design with Hexagonal Architecture
- Language: Go
- Testing: Comprehensive unit tests required (target 80%+ coverage)

## Linear Issues to Implement (by Priority)

### Priority 1 - URGENT (Implement in this order):

1. **KIL-64: Implement Proper MCP Message Framing and Stream Handling**
   - URL: https://linear.app/kilometers-ai/issue/KIL-64
   - This blocks KIL-61, so must be done first
   - MCP messages are newline-delimited JSON-RPC 2.0
   - Need to handle line-based reading for both stdout/stderr

2. **KIL-62: Fix Buffer Size Limitation for Large MCP Messages**
   - URL: https://linear.app/kilometers-ai/issue/KIL-62
   - Current buffer (4096 bytes) is too small
   - Getting "bufio.Scanner: token too long" errors
   - Need 1MB+ buffer for Linear search results

3. **KIL-61: Fix MCP JSON-RPC Message Parsing**
   - URL: https://linear.app/kilometers-ai/issue/KIL-61
   - `parseEventFromData` in `monitoring_service.go` is just a stub
   - Need to parse actual JSON-RPC structure

### Priority 2 - HIGH (After urgent fixes):

4. **KIL-63: Improve Error Handling and Debugging**
   - URL: https://linear.app/kilometers-ai/issue/KIL-63
   - Add debug mode with --debug flag
   - Better error context and logging

5. **KIL-65: Create Test Harness for MCP Message Processing**
   - URL: https://linear.app/kilometers-ai/issue/KIL-65
   - Comprehensive tests for all edge cases
   - Mock server enhancements

## Implementation Instructions

### For Each Issue:
1. Read the full Linear issue description for acceptance criteria
2. Implement the fix following the existing DDD/Hexagonal architecture patterns
3. Add comprehensive unit tests (minimum 80% coverage)
4. Test with the mock MCP server in `test/mock_mcp_server.go`
5. Verify the fix works with real MCP servers

### Quick Fix Code (from KIL-61 comment):

```go
// Update parseEventFromData in monitoring_service.go:
func (s *MonitoringService) parseEventFromData(data []byte, direction event.Direction) (*event.Event, error) {
    trimmedData := bytes.TrimSpace(data)
    if len(trimmedData) == 0 {
        return nil, fmt.Errorf("empty data")
    }

    var msg struct {
        JSONRPC string          `json:"jsonrpc"`
        Method  string          `json:"method,omitempty"`
        ID      json.RawMessage `json:"id,omitempty"`
        Params  json.RawMessage `json:"params,omitempty"`
        Result  json.RawMessage `json:"result,omitempty"`
        Error   json.RawMessage `json:"error,omitempty"`
    }

    if err := json.Unmarshal(trimmedData, &msg); err != nil {
        s.logger.LogError(err, "Failed to parse JSON-RPC message", map[string]interface{}{
            "data_preview": string(trimmedData[:min(len(trimmedData), 200)]),
            "data_size":    len(trimmedData),
        })
        return nil, fmt.Errorf("invalid JSON: %w", err)
    }

    methodName := msg.Method
    if methodName == "" {
        if msg.Error != nil {
            methodName = "error_response"
        } else {
            methodName = "response"
        }
    }

    method, err := event.NewMethod(methodName)
    if err != nil {
        return nil, err
    }

    riskScore := 10
    if strings.Contains(methodName, "write") || 
       strings.Contains(methodName, "delete") || 
       strings.Contains(methodName, "create") {
        riskScore = 50
    }

    score, err := event.NewRiskScore(riskScore)
    if err != nil {
        return nil, err
    }

    return event.CreateEvent(direction, method, trimmedData, score)
}

// Update buffer reading in process_monitor.go:
func (m *MCPProcessMonitor) monitorStdout(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            m.logger.Log(ports.LogLevelError, "Stdout monitor panicked", map[string]interface{}{
                "error": r,
            })
        }
    }()

    reader := bufio.NewReaderSize(m.stdout, 1024*1024) // 1MB buffer

    for {
        select {
        case <-ctx.Done():
            return
        default:
            line, err := reader.ReadString('\n')
            if err != nil {
                if err != io.EOF {
                    m.logger.LogError(err, "Error reading stdout", nil)
                    m.updateStats(0, 0, 0, 1, true)
                }
                return
            }

            if len(line) > 0 {
                data := []byte(line)
                select {
                case m.stdoutChan <- data:
                    m.updateStats(int64(len(data)), 0, 1, 0, false)
                    m.logger.Log(ports.LogLevelDebug, "Stdout line received", map[string]interface{}{
                        "bytes": len(data),
                    })
                case <-ctx.Done():
                    return
                default:
                    m.logger.Log(ports.LogLevelWarn, "Stdout channel full, dropping data", map[string]interface{}{
                        "bytes": len(data),
                    })
                }
            }
        }
    }
}
```

## Testing Requirements

### For KIL-64 (MCP Framing):
- Test newline-delimited JSON parsing
- Test handling of empty lines
- Test partial message handling
- Test simultaneous stdout/stderr

### For KIL-62 (Buffer Size):
- Test with messages > 64KB
- Test with messages up to 10MB
- Verify no "token too long" errors
- Test memory usage remains reasonable

### For KIL-61 (JSON Parsing):
- Test valid JSON-RPC requests
- Test JSON-RPC responses
- Test malformed JSON recovery
- Test method extraction

### For KIL-63 (Error Handling):
- Test debug mode flag
- Test error context in logs
- Test error aggregation

### For KIL-65 (Test Harness):
- Enhance `test/mock_mcp_server.go`
- Add methods: SendLargeMessage, SendMalformedMessage
- Create integration tests
- Add benchmarks

## Success Verification

After implementing each fix:
1. Run all unit tests: `go test ./internal/...`
2. Run integration tests: `go test ./integration_test/...`
3. Test with mock MCP server: `go run test/mock_mcp_server.go`
4. Test with real Linear MCP server using the km tool

## Code Quality Requirements
- Follow existing DDD patterns in the codebase
- Maintain separation between domain, application, and infrastructure layers
- Use dependency injection patterns already established
- Add proper error handling with context
- Include comprehensive logging
- Write clean, idiomatic Go code
- Add inline documentation for complex logic

## Additional Context
- The project recently completed a major refactoring (KIL-36 through KIL-52)
- Follow the patterns established in that refactoring
- The codebase uses ports & adapters pattern
- All infrastructure code should implement port interfaces
- Domain logic should remain pure and testable

Please implement these fixes one at a time in priority order, with full test coverage for each fix.