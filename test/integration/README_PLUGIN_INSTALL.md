# Plugin Install Integration Test

This directory contains comprehensive integration tests for the `km plugins install` command.

## Overview

The `plugin_install_test.go` file tests the complete plugin installation workflow, including:

- **Plugin Discovery**: Finding plugins via API registry
- **Authentication**: Validating API keys and subscription tiers
- **Local vs Remote Installation**: Testing both .kmpkg local packages and API downloads
- **Error Handling**: Invalid plugins, insufficient permissions, network failures
- **Plugin Management**: List, install, update workflows

## Architecture

The integration test uses the **unified mock API server** approach:

```
┌─────────────────┐    HTTP     ┌─────────────────┐    Control API    ┌─────────────────┐
│                 │  requests   │                 │     endpoints     │                 │
│  Integration    ├─────────────►   Mock API      ├───────────────────►   Test Runner   │
│     Test        │             │    Server       │                   │                 │
│                 │◄─────────────┤  (localhost:    │◄───────────────────┤   (Go Test)     │
└─────────────────┘   responses  │     5194)       │   configuration   └─────────────────┘
                                 └─────────────────┘
```

### Key Components

1. **Mock API Server** (`test/mock-api/main.go`)
   - Standalone HTTP server with plugin endpoints
   - Control API for runtime configuration (`/_control/*`)
   - Supports all plugin management operations

2. **Test Utilities** (`test/testutil/mock_api_server.go`)
   - Backward-compatible wrapper for integration tests
   - Field-based configuration that syncs with HTTP server
   - Request logging and assertion helpers

3. **Integration Tests** (`plugin_install_test.go`)
   - Comprehensive test scenarios
   - Isolated temporary environments
   - Real CLI command execution

## Running the Tests

### Option 1: Manual Server + Test
```bash
# Terminal 1: Start mock server
cd test/mock-api
go run main.go

# Terminal 2: Run tests (in project root)
export TEST_WITH_MOCK_SERVER=true
go test -v ./test/integration/plugin_install_test.go
```

### Option 2: Automated Script
```bash
# Run the complete test suite with automatic server management
./scripts/test-plugin-install.sh
```

### Option 3: Individual Test Scenarios
```bash
# Run specific test cases
go test -v ./test/integration/plugin_install_test.go -run=TestPluginInstallCommand/InstallConsoleLogger
go test -v ./test/integration/plugin_install_test.go -run=TestPluginListCommand
```

## Test Scenarios

### Successful Installations

1. **InstallConsoleLogger**: Basic plugin installation from API registry
2. **InstallAPILogger**: Pro-tier plugin requiring elevated permissions
3. **InstallFromLocalKmpkg**: Local .kmpkg package installation (priority over API)
4. **SetupPluginsDirectory**: Directory initialization when no plugin specified

### Error Scenarios

1. **InstallWithoutAPIKey**: Missing authentication
2. **InstallPluginNotAvailableForTier**: Insufficient subscription level
3. **InstallNonExistentPlugin**: Plugin not found in registry

### Discovery and Listing

1. **TestPluginListCommand**: Plugin discovery and display
2. **TestPluginInstallWithRealServer**: End-to-end with running server

## Mock Server Configuration

The test dynamically configures the mock server via the testutil wrapper:

```go
mockAPI := testutil.NewMockAPIServer(t).Build()
mockAPI.ApiKeyValid = true
mockAPI.SubscriptionTier = "pro"
mockAPI.AvailablePlugins = []plugindomain.Plugin{
    {
        Name: "console-logger",
        Version: "1.2.3",
        RequiredTier: plugindomain.TierFree,
        Size: 1024 * 10,
    },
}
mockAPI.DownloadResponses = map[string][]byte{
    "console-logger": []byte("#!/bin/bash\\necho 'Mock plugin'\\n"),
}
```

### Control Endpoints

The mock server exposes control endpoints for test configuration:

- `POST /_control/config` - Update server configuration
- `POST /_control/plugins` - Set available plugins
- `POST /_control/downloads` - Configure download responses
- `POST /_control/auth` - Set authentication responses
- `GET /_control/requests` - View request log
- `POST /_control/reset` - Clear request log

## Plugin Installation Flow

The integration test verifies this complete workflow:

1. **Configuration Loading**: API key and endpoint resolution
2. **Plugin Manager Creation**: Directory setup and API client initialization
3. **Local Package Check**: Search for .kmpkg files in plugins directory
4. **API Registry Fallback**: Query remote plugin manifest if not found locally
5. **Authentication**: Validate API key and subscription permissions
6. **Download**: Retrieve plugin binary from API or extract from .kmpkg
7. **Installation**: Place plugin in correct directory with proper permissions
8. **Verification**: Confirm plugin is available for execution

## Environment Variables

- `TEST_WITH_MOCK_SERVER=true` - Enable tests that require a running server
- `KM_API_ENDPOINT` - Override API endpoint (set by tests)
- `KM_API_KEY` - Set API key for authentication (set by tests)
- `HOME` - Isolated to temporary directory per test

## Integration with Main Test Suite

These tests are designed to be run as part of the broader test suite:

```bash
# Run all integration tests
go test -v ./test/integration/...

# Run with coverage
go test -coverprofile=coverage.out ./test/integration/...
go tool cover -html=coverage.out
```

## Debugging

### Request Logging

The mock server logs all requests. To view them:

```bash
curl http://localhost:5194/_control/requests | jq '.'
```

### Test Output

Run tests with verbose output to see detailed plugin installation steps:

```bash
go test -v ./test/integration/plugin_install_test.go -run=TestPluginInstallCommand
```

### Server Health

Check if the mock server is responding:

```bash
curl http://localhost:5194/health
```

## Future Enhancements

- **Plugin Signature Verification**: Test cryptographic validation
- **Update Scenarios**: Plugin versioning and update workflows  
- **Concurrent Installs**: Multi-plugin installation testing
- **Network Failures**: Retry logic and error recovery
- **Plugin Dependencies**: Complex installation chains

## Related Files

- `internal/interfaces/cli/plugins.go` - CLI command implementation
- `internal/plugins/manager.go` - Plugin management core logic
- `test/mock-api/main.go` - Standalone mock API server
- `scripts/test-plugin-install.sh` - Automated test runner