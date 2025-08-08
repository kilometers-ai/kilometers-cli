# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Building
```bash
# Basic build
go build -o km ./cmd/main.go

# Production build with version info
go build -ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%d)" -o km ./cmd/main.go
```

### Testing
```bash
# Run all tests using the project script (preferred)
./scripts/test/run-tests.sh

# Run tests with coverage
./scripts/test/run-tests.sh --coverage

# Run specific test packages
go test ./internal/core/domain/...
go test ./internal/application/services/...

# Run integration tests for plugin authentication pipeline
go test ./test/integration/ -v
```

### Development Commands
```bash
# Monitor an MCP server (basic usage)
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /path/to/directory

# With API key for premium features
export KM_API_KEY="km_live_your_api_key"
./km monitor -- your-mcp-server --args

# Initialize with automatic configuration discovery
./km init --auto-detect
```

## Architecture Overview

### Clean Architecture Structure
The project follows Clean Architecture with Hexagonal Architecture patterns:

- **`internal/core/`** - Core domain layer (entities, business rules)
  - `domain/` - Domain models: Config, Command, JsonRPC message structures
  - `ports/` - Interface definitions (hexagonal architecture ports)

- **`internal/application/`** - Application layer (use cases, services)
  - `services/monitor_service.go` - Core monitoring orchestration
  - `services/stream_proxy.go` - Transparent JSON-RPC message proxying

- **`internal/infrastructure/`** - Infrastructure layer (adapters)
  - `plugins/` - Plugin system implementation (manager, discovery, auth)
  - `http/` - HTTP clients for kilometers-api integration
  - `process/` - MCP server process execution
  - `logging/` - Logging implementations

- **`internal/interfaces/`** - Interface layer (CLI, API handlers)
  - `cli/` - Cobra-based CLI interface

### Key Components

1. **Monitor Service** (`application/services/monitor_service.go`)
   - Orchestrates MCP server monitoring workflow
   - Manages plugin lifecycle and message routing

2. **Stream Proxy** (`application/services/stream_proxy.go`) 
   - Transparent bidirectional JSON-RPC proxying
   - Sub-millisecond message forwarding with event generation

3. **Plugin Manager** (`infrastructure/plugins/manager.go`)
   - Handles plugin discovery, authentication, and lifecycle
   - JWT-based authentication with 5-minute caching

4. **Plugin System**
   - Customer-specific binaries with embedded authentication
   - Multi-tier access control (Free, Pro, Enterprise)
   - Real-time subscription validation with kilometers-api

## Plugin Architecture

The CLI implements a sophisticated security model:

- **Customer Isolation**: Each plugin built uniquely per customer with embedded secrets
- **Binary Signatures**: Digital signature validation for tamper detection  
- **JWT Authentication**: Time-limited tokens with feature-based authorization
- **Subscription Tiers**: Free (console only), Pro (API logging), Enterprise (custom plugins)

Plugin flow: Discovery → Authentication → Loading → Message Routing → Cleanup

## Development Patterns

### Error Handling
- Use structured errors with context
- Implement graceful degradation for plugin failures
- Log errors appropriately without exposing sensitive data

### Testing Strategy
- Unit tests for core domain logic
- Integration tests for plugin system
- Use the project test script for comprehensive validation
- Mock external dependencies (kilometers-api, plugin binaries)

### Security Considerations
- Never log sensitive data (API keys, customer secrets)
- Validate all plugin inputs and authentication tokens
- Use time-limited caches for authentication (5-minute TTL)
- Implement proper process isolation for plugins

## Integration Testing

### Plugin Authentication Pipeline Tests
- **Location**: `test/integration/plugin_auth_pipeline_test.go`
- **Coverage**: Core plugin authentication → subscription validation → message routing pipeline
- **Mock Components**: HTTP API server, plugin discovery, authentication cache
- **Key Test Scenarios**:
  - Basic plugin authentication flow
  - Subscription tier validation (Free/Pro/Enterprise)  
  - Plugin discovery and loading attempts
  - Message routing to authorized plugins
  - Authentication caching behavior

### Running Integration Tests
```bash
# Run all integration tests
go test ./test/integration/ -v

# Run specific test
go test ./test/integration/ -run TestPluginAuthenticator_TierValidation -v
```

## Important Notes

- This is a transparent proxy for MCP (Model Context Protocol) servers
- The system is stateless and event-driven for scalability
- Plugin authentication requires active internet connection for tier validation
- All plugin communication uses gRPC with HashiCorp go-plugin framework
- The binary uses go-plugin for secure plugin isolation and communication