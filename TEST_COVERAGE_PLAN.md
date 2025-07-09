# Test Coverage Plan - Kilometers CLI

## Overview
This document outlines the prioritized test coverage plan for the Kilometers CLI project, focusing on critical functionality and regression prevention. Tests are organized by priority and functional area.

## Priority Levels
- **CRITICAL**: Core functionality that would break the CLI entirely
- **HIGH**: Important features that users rely on daily
- **MEDIUM**: Edge cases and error handling that improve reliability
- **LOW**: Nice-to-have coverage for completeness

---

## 1. CLI Command Handling & Argument Parsing

### CRITICAL Priority

- [ ] **Main command handling flow**
  - **Component**: `main.go:handleCommands()`
  - **Description**: Test that CLI correctly routes between wrapping MCP servers vs built-in commands
  - **Priority**: CRITICAL - Core routing logic that determines CLI behavior
  - **Rationale**: Failure here breaks all CLI functionality

- [ ] **Process wrapper creation and execution**
  - **Component**: `main.go:main()` and `NewProcessWrapper()`
  - **Description**: Test successful creation and execution of wrapped MCP server processes
  - **Priority**: CRITICAL - Primary CLI functionality
  - **Rationale**: This is the main use case - wrapping MCP servers

### HIGH Priority

- [ ] **Version command handling**
  - **Component**: `main.go:printVersion()`
  - **Description**: Test version display with build info, Go version, and platform details
  - **Priority**: HIGH - Common user operation
  - **Rationale**: Users frequently check version for troubleshooting

- [ ] **Help command handling**
  - **Component**: `update.go:printHelp()`
  - **Description**: Test help message display with usage examples and environment variables
  - **Priority**: HIGH - User discovery and onboarding
  - **Rationale**: Critical for user adoption and troubleshooting

- [ ] **Update command handling**
  - **Component**: `main.go:handleUpdate()`
  - **Description**: Test update command routing and placeholder functionality
  - **Priority**: HIGH - User maintenance workflow
  - **Rationale**: Users expect update functionality to work correctly

### MEDIUM Priority

- [ ] **Invalid argument handling**
  - **Component**: `main.go:printUsage()`
  - **Description**: Test proper error messages when no arguments provided or invalid usage
  - **Priority**: MEDIUM - User experience improvement
  - **Rationale**: Helps users understand correct usage

---

## 2. MCP Message Parsing & Processing

### CRITICAL Priority

- [ ] **JSON-RPC message parsing**
  - **Component**: `main.go:parseMCPMessage()`
  - **Description**: Test parsing of valid JSON-RPC 2.0 messages (requests, responses, notifications)
  - **Priority**: CRITICAL - Core message processing
  - **Rationale**: Failure breaks all MCP server communication monitoring

- [ ] **Event generation from parsed messages**
  - **Component**: `main.go:monitorStdin()` and `monitorStdout()`
  - **Description**: Test Event struct creation with correct fields (ID, timestamp, direction, method, payload, size)
  - **Priority**: CRITICAL - Data integrity for API submission
  - **Rationale**: Incorrect event data corrupts monitoring data

### HIGH Priority

- [ ] **Message direction detection**
  - **Component**: `main.go:monitorStdin()` and `monitorStdout()`
  - **Description**: Test proper assignment of "request" vs "response" direction
  - **Priority**: HIGH - Data accuracy for analysis
  - **Rationale**: Direction is crucial for understanding communication flow

- [ ] **Invalid JSON handling**
  - **Component**: `main.go:parseMCPMessage()`
  - **Description**: Test graceful handling of malformed JSON and non-JSON data
  - **Priority**: HIGH - Robustness against real-world input
  - **Rationale**: MCP servers may output non-JSON data that shouldn't break monitoring

### MEDIUM Priority

- [ ] **Large payload handling**
  - **Component**: `main.go:parseMCPMessage()`
  - **Description**: Test parsing of messages with large payloads (> 1MB)
  - **Priority**: MEDIUM - Performance edge case
  - **Rationale**: Large payloads could cause memory issues or timeouts

- [ ] **ID type variations**
  - **Component**: `main.go:parseMCPMessage()`
  - **Description**: Test handling of different ID types (string, number, null, missing)
  - **Priority**: MEDIUM - JSON-RPC spec compliance
  - **Rationale**: Different MCP servers may use different ID formats

---

## 3. Process Wrapper & Monitoring

### CRITICAL Priority

- [ ] **Process lifecycle management**
  - **Component**: `main.go:ProcessWrapper.Start()` and `Wait()`
  - **Description**: Test starting, monitoring, and graceful shutdown of wrapped processes
  - **Priority**: CRITICAL - Core wrapper functionality
  - **Rationale**: Process management failure breaks the entire CLI purpose

- [ ] **Stdin/stdout/stderr forwarding**
  - **Component**: `main.go:monitorStdin()`, `monitorStdout()`, `forwardStderr()`
  - **Description**: Test transparent forwarding of all streams while monitoring
  - **Priority**: CRITICAL - Transparency requirement
  - **Rationale**: Users expect wrapped processes to behave identically to unwrapped

### HIGH Priority

- [ ] **Goroutine coordination**
  - **Component**: `main.go:ProcessWrapper.Start()` and `Wait()`
  - **Description**: Test proper synchronization of monitoring goroutines using WaitGroup
  - **Priority**: HIGH - Prevents resource leaks and race conditions
  - **Rationale**: Incorrect coordination can cause deadlocks or resource leaks

- [ ] **Event buffer management**
  - **Component**: `main.go:ProcessWrapper` events channel
  - **Description**: Test handling of event buffer overflow and proper channel closing
  - **Priority**: HIGH - Prevents data loss and deadlocks
  - **Rationale**: High-volume MCP traffic could overflow buffers

### MEDIUM Priority

- [ ] **Process exit code handling**
  - **Component**: `main.go:ProcessWrapper.Wait()`
  - **Description**: Test proper forwarding of wrapped process exit codes
  - **Priority**: MEDIUM - User workflow compatibility
  - **Rationale**: Scripts and CI/CD depend on correct exit codes

- [ ] **Signal handling**
  - **Component**: `main.go:ProcessWrapper`
  - **Description**: Test handling of SIGTERM, SIGINT, and other signals
  - **Priority**: MEDIUM - Graceful shutdown
  - **Rationale**: Users expect Ctrl+C to work correctly

---

## 4. Configuration Loading & Validation

### CRITICAL Priority

- [ ] **Default configuration loading**
  - **Component**: `config.go:LoadConfig()` and `DefaultConfig()`
  - **Description**: Test loading of default configuration when no config file exists
  - **Priority**: CRITICAL - CLI must work out of box
  - **Rationale**: First-time users have no configuration file

- [ ] **Configuration validation**
  - **Component**: `config.go:ValidateConfig()`
  - **Description**: Test validation and correction of invalid configuration values
  - **Priority**: CRITICAL - Prevents runtime errors
  - **Rationale**: Invalid config can cause CLI to crash or behave unexpectedly

### HIGH Priority

- [ ] **Environment variable override**
  - **Component**: `config.go:LoadConfig()`
  - **Description**: Test that environment variables properly override default values
  - **Priority**: HIGH - Common deployment pattern
  - **Rationale**: Environment variables are primary configuration method in CI/CD

- [ ] **Config file loading**
  - **Component**: `config.go:LoadConfig()` and `getConfigPath()`
  - **Description**: Test loading configuration from JSON file in user home directory
  - **Priority**: HIGH - Persistent configuration
  - **Rationale**: Users expect configuration to persist between CLI invocations

### MEDIUM Priority

- [ ] **Config file parsing errors**
  - **Component**: `config.go:LoadConfig()`
  - **Description**: Test handling of malformed JSON config files
  - **Priority**: MEDIUM - Error recovery
  - **Rationale**: Users may manually edit config files and introduce errors

- [ ] **Config file creation**
  - **Component**: `config.go:SaveConfig()`
  - **Description**: Test creation of config directory and file with proper permissions
  - **Priority**: MEDIUM - User convenience
  - **Rationale**: Users may want to persist configuration changes

---

## 5. API Client & Network Communication

### CRITICAL Priority

- [ ] **API connection testing**
  - **Component**: `client.go:TestConnection()`
  - **Description**: Test health check and authentication validation
  - **Priority**: CRITICAL - Determines if API functionality is available
  - **Rationale**: Users need to know if API is reachable before capturing events

- [ ] **Event batch submission**
  - **Component**: `client.go:SendEventBatch()`
  - **Description**: Test successful submission of event batches to API
  - **Priority**: CRITICAL - Core data transmission
  - **Rationale**: Primary purpose is to send monitoring data to API

### HIGH Priority

- [ ] **Authentication handling**
  - **Component**: `client.go:TestConnection()` and `SendEventBatch()`
  - **Description**: Test API key authentication and 401 error handling
  - **Priority**: HIGH - Security requirement
  - **Rationale**: API requires authentication; failure should be clear to user

- [ ] **Network error handling**
  - **Component**: `client.go:SendEventBatch()`
  - **Description**: Test handling of network timeouts, connection failures, and API errors
  - **Priority**: HIGH - Resilience in unreliable environments
  - **Rationale**: Network issues are common; CLI should handle gracefully

### MEDIUM Priority

- [ ] **Event serialization**
  - **Component**: `client.go:eventToDTO()`
  - **Description**: Test conversion of Event struct to EventDto with proper base64 encoding
  - **Priority**: MEDIUM - Data integrity
  - **Rationale**: Incorrect serialization corrupts data but doesn't break CLI

- [ ] **HTTP request formation**
  - **Component**: `client.go:SendEventBatch()`
  - **Description**: Test proper HTTP headers, content-type, and request body formatting
  - **Priority**: MEDIUM - API compatibility
  - **Rationale**: API may be sensitive to header format

---

## 6. Risk Detection & Filtering

### CRITICAL Priority

- [ ] **Risk scoring algorithm**
  - **Component**: `risk.go:AnalyzeEvent()`
  - **Description**: Test risk score calculation combining method, content, and size factors
  - **Priority**: CRITICAL - Core filtering logic
  - **Rationale**: Incorrect risk scoring defeats purpose of intelligent filtering

- [ ] **Event filtering decisions**
  - **Component**: `risk.go:ShouldCaptureEvent()`
  - **Description**: Test filtering logic based on risk levels, method whitelists, and configuration
  - **Priority**: CRITICAL - Determines what data is captured
  - **Rationale**: Wrong filtering captures too much noise or misses important events

### HIGH Priority

- [ ] **High-risk pattern detection**
  - **Component**: `risk.go:NewRiskDetector()` and `analyzeContent()`
  - **Description**: Test detection of sensitive file paths, database queries, and system access
  - **Priority**: HIGH - Security monitoring
  - **Rationale**: Main value proposition is detecting risky operations

- [ ] **Method risk classification**
  - **Component**: `risk.go:getMethodRiskScore()`
  - **Description**: Test risk scoring of different MCP methods (tools/call, resources/read, etc.)
  - **Priority**: HIGH - Accurate risk assessment
  - **Rationale**: Method-based filtering is primary risk signal

### MEDIUM Priority

- [ ] **JSON parameter analysis**
  - **Component**: `risk.go:containsHighRiskJSON()`
  - **Description**: Test detection of dangerous patterns in JSON parameters
  - **Priority**: MEDIUM - Deep content analysis
  - **Rationale**: Provides more sophisticated risk detection

- [ ] **Payload size risk assessment**
  - **Component**: `risk.go:analyzeSizeRisk()`
  - **Description**: Test risk scoring based on message size
  - **Priority**: MEDIUM - Performance protection
  - **Rationale**: Large payloads may indicate data exfiltration

---

## 7. Event Batching & Processing

### CRITICAL Priority

- [ ] **Event batching logic**
  - **Component**: `main.go:addToBatch()` and `sendBatch()`
  - **Description**: Test accumulation of events into batches and automatic sending when full
  - **Priority**: CRITICAL - Efficient API usage
  - **Rationale**: Batching reduces API calls; wrong batching causes data loss

- [ ] **Batch flushing on shutdown**
  - **Component**: `main.go:flushBatch()` and `processEvents()`
  - **Description**: Test that remaining events are sent when CLI exits
  - **Priority**: CRITICAL - Prevents data loss
  - **Rationale**: Events in incomplete batches must not be lost

### HIGH Priority

- [ ] **Periodic batch flushing**
  - **Component**: `main.go:processEvents()` ticker
  - **Description**: Test timer-based flushing of batches every 5 seconds
  - **Priority**: HIGH - Timely data delivery
  - **Rationale**: Long-running sessions need periodic flushing

- [ ] **Concurrent batch handling**
  - **Component**: `main.go:addToBatch()` and `batchMutex`
  - **Description**: Test thread-safe batch operations with proper mutex usage
  - **Priority**: HIGH - Prevents race conditions
  - **Rationale**: Multiple goroutines access batch simultaneously

### MEDIUM Priority

- [ ] **Event ID generation**
  - **Component**: `main.go:generateEventID()`
  - **Description**: Test uniqueness of generated event IDs
  - **Priority**: MEDIUM - Data integrity
  - **Rationale**: Duplicate IDs could cause API issues

- [ ] **Filtering statistics tracking**
  - **Component**: `main.go:incrementTotalEvents()` and `printFilteringStats()`
  - **Description**: Test accurate counting of total and filtered events
  - **Priority**: MEDIUM - User insight
  - **Rationale**: Helps users understand filtering effectiveness

---

## 8. Update System

### HIGH Priority

- [ ] **Version checking**
  - **Component**: `update.go:CheckForUpdate()`
  - **Description**: Test fetching version manifest and comparing with current version
  - **Priority**: HIGH - Maintenance workflow
  - **Rationale**: Users need reliable update notifications

- [ ] **Binary download and replacement**
  - **Component**: `update.go:SelfUpdate()`
  - **Description**: Test downloading new binary and replacing current executable
  - **Priority**: HIGH - Update functionality
  - **Rationale**: Self-update is complex and error-prone

### MEDIUM Priority

- [ ] **Platform detection**
  - **Component**: `update.go:GetPlatformKey()`
  - **Description**: Test correct OS/architecture detection for downloads
  - **Priority**: MEDIUM - Cross-platform support
  - **Rationale**: Wrong platform detection breaks updates

- [ ] **Update error handling**
  - **Component**: `update.go:SelfUpdate()`
  - **Description**: Test handling of download failures, permission errors, and rollback
  - **Priority**: MEDIUM - Update reliability
  - **Rationale**: Failed updates shouldn't break existing installation

### LOW Priority

- [ ] **Version manifest parsing**
  - **Component**: `update.go:VersionInfo`
  - **Description**: Test parsing of version manifest JSON with checksums and download URLs
  - **Priority**: LOW - Data structure validation
  - **Rationale**: Manifest format is controlled by us

---

## 9. Error Handling & Edge Cases

### HIGH Priority

- [ ] **API unavailable handling**
  - **Component**: `main.go:main()` API client nil check
  - **Description**: Test CLI continues to work when API is unreachable
  - **Priority**: HIGH - Offline functionality
  - **Rationale**: Users should be able to wrap MCP servers even without API

- [ ] **Wrapped process failures**
  - **Component**: `main.go:ProcessWrapper.Wait()`
  - **Description**: Test handling when wrapped MCP server crashes or exits with error
  - **Priority**: HIGH - Error propagation
  - **Rationale**: Users expect to see MCP server errors

### MEDIUM Priority

- [ ] **Invalid MCP server commands**
  - **Component**: `main.go:NewProcessWrapper()`
  - **Description**: Test handling of non-existent commands or invalid arguments
  - **Priority**: MEDIUM - User error handling
  - **Rationale**: Users may mistype commands

- [ ] **Resource exhaustion**
  - **Component**: `main.go:ProcessWrapper`
  - **Description**: Test behavior under high memory usage or many concurrent events
  - **Priority**: MEDIUM - Performance edge cases
  - **Rationale**: Resource limits shouldn't cause crashes

### LOW Priority

- [ ] **Logging edge cases**
  - **Component**: `main.go:ProcessWrapper.logger`
  - **Description**: Test log output with special characters, very long messages, etc.
  - **Priority**: LOW - Logging robustness
  - **Rationale**: Logging failures are not critical to core functionality

---

## 10. Integration & End-to-End Tests

### HIGH Priority

- [ ] **Full CLI workflow with real MCP server**
  - **Component**: Integration test
  - **Description**: Test complete flow: CLI starts, wraps real MCP server, captures events, sends to API
  - **Priority**: HIGH - Real-world validation
  - **Rationale**: Unit tests can't catch integration issues

- [ ] **Configuration override integration**
  - **Component**: Integration test
  - **Description**: Test environment variables and config file working together in real scenario
  - **Priority**: HIGH - Deployment validation
  - **Rationale**: Configuration interactions are complex

### MEDIUM Priority

- [ ] **Performance with high-volume MCP traffic**
  - **Component**: Integration test
  - **Description**: Test CLI performance with rapid message flow from MCP server
  - **Priority**: MEDIUM - Scalability validation
  - **Rationale**: Some MCP servers generate high message volumes

- [ ] **Long-running session stability**
  - **Component**: Integration test
  - **Description**: Test CLI stability over extended periods (hours/days)
  - **Priority**: MEDIUM - Reliability validation
  - **Rationale**: CLI may run for long periods in production

### LOW Priority

- [ ] **Cross-platform compatibility**
  - **Component**: Integration test
  - **Description**: Test CLI behavior on different operating systems
  - **Priority**: LOW - Platform validation
  - **Rationale**: Go provides good cross-platform support

---

## Coverage Summary Checklist

### Core Functionality ✓
- [ ] CLI command routing and argument parsing
- [ ] MCP message parsing and event generation
- [ ] Process wrapper and stream monitoring
- [ ] Configuration loading and validation
- [ ] API client communication
- [ ] Risk detection and filtering
- [ ] Event batching and processing

### Secondary Features ✓
- [ ] Update system functionality
- [ ] Error handling and recovery
- [ ] Performance edge cases
- [ ] Integration scenarios

### Test Infrastructure ✓
- [ ] Test utilities and helpers
- [ ] Mock API client for testing
- [ ] Test MCP server for integration tests
- [ ] Performance benchmarks

### Documentation ✓
- [ ] Test case documentation
- [ ] Testing best practices guide
- [ ] CI/CD test integration

---

## Progress Tracking

**Total Tests Planned**: 52  
**Critical Priority**: 8 tests  
**High Priority**: 21 tests  
**Medium Priority**: 17 tests  
**Low Priority**: 6 tests  

**Current Status**: 0% complete  
**Next Phase**: Implement all CRITICAL priority tests first, then HIGH priority tests.

---

## Notes

1. **Test Environment**: Tests should run in isolated environments with mock API servers
2. **Test Data**: Use real MCP message examples from various servers for realistic testing
3. **Performance**: Include benchmarks for message parsing and event processing
4. **CI/CD**: All tests should be automated and run on every commit
5. **Coverage**: Aim for >90% code coverage after implementing all planned tests

This plan focuses on preventing regressions in core CLI functionality while ensuring reliability and user experience remain high. 