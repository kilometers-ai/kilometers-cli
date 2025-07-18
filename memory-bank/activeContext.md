# Active Context: Current Focus and Priorities

## Current Work Focus

### Primary Objective: Fix Critical MCP Message Processing Issues
The kilometers CLI is experiencing critical issues when monitoring real MCP servers, particularly with large JSON payloads from Linear MCP server. These issues are blocking real-world usage and need immediate resolution.

### Active Linear Issues (Priority Order)

#### 🚨 URGENT - Must Fix First

##### KIL-64: Implement Proper MCP Message Framing and Stream Handling
- **Status**: Not started
- **Priority**: Highest - blocks KIL-61
- **Issue**: MCP messages are newline-delimited JSON-RPC 2.0, but current implementation doesn't handle proper line-based reading
- **Impact**: Messages are being truncated or corrupted during parsing
- **Location**: `internal/infrastructure/monitoring/process_monitor.go`
- **Required**: Implement line-based buffering for stdout/stderr streams

##### KIL-62: Fix Buffer Size Limitation for Large MCP Messages  
- **Status**: Not started
- **Priority**: Critical
- **Issue**: Current 4KB buffer causes "bufio.Scanner: token too long" errors
- **Impact**: Cannot monitor Linear search results or other large payloads
- **Location**: `internal/infrastructure/monitoring/process_monitor.go` lines 335-417
- **Required**: Increase buffer to 1MB+ and implement proper size handling

##### KIL-61: Fix MCP JSON-RPC Message Parsing
- **Status**: Partially implemented
- **Priority**: Critical 
- **Issue**: `parseEventFromData` in `monitoring_service.go` is mostly a stub
- **Impact**: Events are not being properly parsed from real MCP messages
- **Location**: `internal/application/services/monitoring_service.go` lines 485-552
- **Required**: Complete JSON-RPC 2.0 parsing with proper error handling

#### 🔥 HIGH PRIORITY - After Urgent Fixes

##### KIL-63: Improve Error Handling and Debugging
- **Status**: Planned
- **Issue**: Poor error context and debugging capabilities
- **Required**: Enhanced logging, debug mode, better error messages

##### KIL-65: Create Test Harness for MCP Message Processing
- **Status**: Partially implemented
- **Issue**: Mock server needs enhancement for testing edge cases
- **Required**: Comprehensive test coverage for message processing

## Recent Changes and Context

### Architecture Refactoring (Completed)
The project was successfully refactored from monolithic structure to DDD/Hexagonal Architecture:
- ✅ Domain layer with proper entities and value objects
- ✅ Application services with command pattern
- ✅ Infrastructure adapters for external dependencies
- ✅ Dependency injection container setup
- ✅ Comprehensive test structure

### Current Implementation Status

#### Working Components
- **CLI Interface**: Cobra-based CLI with all commands implemented
- **Configuration System**: Multi-source config with validation
- **Domain Models**: Event, Session, Risk, Filtering models complete
- **Dependency Injection**: Full DI container with proper wiring
- **Test Infrastructure**: Mock servers and integration tests setup

#### Broken/Incomplete Components
- **MCP Message Processing**: Core functionality broken for real servers
- **Large Payload Handling**: Buffer limitations prevent real-world usage
- **Error Recovery**: Poor error handling in stream processing
- **Debug Capabilities**: Limited debugging tools for troubleshooting

## Immediate Next Steps

### Phase 1: Message Processing Fixes (Current Sprint)
1. **Fix Message Framing** (KIL-64)
   - Implement newline-delimited JSON reading
   - Handle partial messages and buffering
   - Test with real MCP servers

2. **Increase Buffer Sizes** (KIL-62)  
   - Replace fixed buffers with dynamic sizing
   - Implement 1MB+ message support
   - Add configurable size limits

3. **Complete JSON-RPC Parsing** (KIL-61)
   - Implement full JSON-RPC 2.0 message structure parsing
   - Handle requests, responses, notifications, and errors
   - Add comprehensive validation

### Phase 2: Stability and Testing (Next Sprint)
1. **Enhanced Error Handling** (KIL-63)
   - Add debug mode with detailed logging
   - Implement graceful error recovery
   - Improve error context and user messaging

2. **Test Coverage** (KIL-65)
   - Create comprehensive test scenarios
   - Test with various MCP server implementations
   - Add property-based tests for edge cases

## Technical Focus Areas

### Core Issues to Address

#### 1. Stream Processing Architecture
**Current Problem**: Fixed-size buffers with line-by-line reading
```go
// Current broken approach
reader := bufio.NewReaderSize(stdout, 4096) // Too small!
```

**Required Solution**: Dynamic buffering with proper message framing
```go
// Need to implement proper newline-delimited JSON streaming
reader := bufio.NewReaderSize(stdout, 1024*1024) // 1MB buffer
// + accumulator pattern for partial messages
```

#### 2. JSON-RPC Protocol Handling
**Current Problem**: Incomplete message parsing
```go
// Current stub implementation
func parseEventFromData(data []byte, direction event.Direction) (*event.Event, error) {
    // Very basic implementation, needs complete rewrite
}
```

**Required Solution**: Full JSON-RPC 2.0 compliance
- Handle all message types (request, response, notification, error)
- Proper validation and error handling
- Support for batch messages

#### 3. Error Propagation and Recovery
**Current Problem**: Errors are logged but not properly handled
**Required Solution**: Graceful degradation and recovery mechanisms

## Development Environment Setup

### For Working on Current Issues
```bash
# Setup for message processing work
cd kilometers-cli

# Test with real Linear MCP server
npx @modelcontextprotocol/server-linear

# Monitor with current broken implementation
go run cmd/main.go monitor npx @modelcontextprotocol/server-linear

# Run specific tests
go test -v ./internal/infrastructure/monitoring/...
go test -v ./integration_test/process_monitoring_test.go
```

### Debug Mode Configuration
```bash
# Enable maximum debugging
export KM_DEBUG=true
export KM_LOG_LEVEL=debug

# Test with verbose output
go run cmd/main.go --debug monitor --verbose npx @modelcontextprotocol/server-linear
```

## Success Criteria for Current Phase

### Definition of Done for Message Processing Fixes
1. **Linear MCP Server Compatibility**: Successfully monitor Linear MCP server with large search results
2. **Buffer Handling**: Handle messages up to 10MB without errors
3. **Parse Accuracy**: Correctly parse 100% of valid JSON-RPC 2.0 messages
4. **Error Recovery**: Gracefully handle malformed messages without crashing
5. **Test Coverage**: 90%+ test coverage for message processing components

### Validation Approach
1. **Unit Tests**: All message processing functions have comprehensive tests
2. **Integration Tests**: End-to-end tests with mock MCP servers
3. **Real-World Testing**: Successful monitoring of Linear, GitHub MCP servers
4. **Performance Testing**: Handle 1000+ messages/second without memory leaks
5. **Error Testing**: Graceful handling of malformed, oversized, and truncated messages

This active context represents the critical path for making kilometers CLI production-ready for real-world MCP monitoring scenarios. 