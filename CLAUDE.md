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
# Configure API key (required for premium features)
./km auth login --api-key "km_pro_your_api_key"

# Check configuration status and sources
./km auth status

# Monitor an MCP server (basic usage)
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /path/to/directory

# With API key via environment variable
export KM_API_KEY="km_live_your_api_key"
./km monitor -- your-mcp-server --args

# Initialize with automatic configuration discovery
./km init --auto-detect
```

## Configuration System

### Unified Configuration Loading
The CLI uses a unified configuration system that loads settings from multiple sources with clear precedence:

1. **CLI Flags** (highest priority) - `--api-key`, `--debug`, etc.
2. **Environment Variables** - `KM_*` prefixed variables
3. **Configuration Files** - `.env` files and saved config
4. **Defaults** (lowest priority) - Built-in application defaults

### Environment Variables
```bash
# Core configuration
KM_API_KEY="km_pro_your_api_key"           # API key for premium features
KM_API_ENDPOINT="https://api.kilometers.ai" # API endpoint URL
KM_DEBUG="true"                            # Enable debug logging
KM_LOG_LEVEL="debug"                       # Set log level (debug, info, warn, error)

# Advanced configuration  
KM_BUFFER_SIZE="2097152"                   # Buffer size in bytes (default: 1MB)
KM_BATCH_SIZE="20"                         # Batch size for API requests
KM_PLUGINS_DIR="/path/to/plugins"          # Custom plugins directory
KM_AUTO_PROVISION="true"                   # Auto-provision plugins
KM_TIMEOUT="60s"                           # Default timeout duration
```

### Configuration Files
- **Saved Config**: `~/.config/kilometers/config.json` (managed by `km auth` commands)
- **Project Config**: `.env` file in current directory (supports `KM_*` variables)
- **User Config**: `.env` file in `~/.config/kilometers/`

### Configuration Transparency
Use `km auth status` to see exactly where each configuration value is loaded from:
```bash
$ km auth status
🔑 API Key: km_pro...key
🌐 API Endpoint: http://localhost:5194
📍 API Key Source: filesystem (/path/to/.env:KM_API_KEY)
📍 API Endpoint Source: env (KM_API_ENDPOINT)
```

## Architecture Overview

### Clean Architecture Structure
The project follows Clean Architecture with Hexagonal Architecture patterns:

- **`internal/core/`** - Core domain layer (entities, business rules)
  - `domain/` - Domain models: UnifiedConfig, Command, JsonRPC message structures
  - `ports/` - Interface definitions (hexagonal architecture ports)

- **`internal/application/`** - Application layer (use cases, services)
  - `services/config_service.go` - Unified configuration management
  - `services/monitor_service.go` - Core monitoring orchestration
  - `services/stream_proxy.go` - Transparent JSON-RPC message proxying

- **`internal/infrastructure/`** - Infrastructure layer (adapters)
  - `config/` - Unified configuration loading and storage
  - `plugins/` - Plugin system implementation (manager, discovery, auth)
  - `http/` - HTTP clients for kilometers-api integration
  - `process/` - MCP server process execution
  - `logging/` - Logging implementations

- **`internal/interfaces/`** - Interface layer (CLI, API handlers)
  - `cli/` - Cobra-based CLI interface

### Key Components

1. **Configuration Service** (`application/services/config_service.go`)
   - Unified configuration loading from multiple sources
   - Transparent source tracking and validation
   - API key management and persistence

2. **Monitor Service** (`application/services/monitor_service.go`)
   - Orchestrates MCP server monitoring workflow
   - Manages plugin lifecycle and message routing

3. **Stream Proxy** (`application/services/stream_proxy.go`) 
   - Transparent bidirectional JSON-RPC proxying
   - Sub-millisecond message forwarding with event generation

4. **Plugin Manager** (`infrastructure/plugins/manager.go`)
   - Handles plugin discovery, authentication, and lifecycle
   - JWT-based authentication with 5-minute caching

5. **Plugin System**
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