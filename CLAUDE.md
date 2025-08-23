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
./scripts/tests/run-tests.sh

# Run tests with coverage
./scripts/tests/run-tests.sh --coverage

# Run specific test packages
go test ./internal/auth/...
go test ./internal/plugins/...

# Run integration tests for plugin authentication pipeline
go test ./test/integration/ -v

# Run single integration test
go test ./test/integration/ -run TestPluginAuthenticator_TierValidation -v
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

### Local API Development Environment

#### Option 1: Standalone Environment (Default)
Each service runs independently with its own database:
```bash
# Start the development environment (requires ../kilometers-api)
docker-compose -f docker-compose.dev.yml up --build -d

# View API logs
docker-compose -f docker-compose.dev.yml logs -f api

# View database logs
docker-compose -f docker-compose.dev.yml logs -f postgres

# Stop the environment
docker-compose -f docker-compose.dev.yml down

# Clean up (removes volumes and data)
docker-compose -f docker-compose.dev.yml down -v

# Test API health
curl http://localhost:5194/health
```

#### Option 2: Shared Development Environment (Recommended)
Uses a shared Docker Compose setup in the kilometers-api repo for both database and API:

**Quick Start:**
```bash
# 1. Start the shared environment (from kilometers-api)
cd ../kilometers-api && docker-compose -f docker-compose.shared.yml up -d

# 2. Verify API is healthy
curl http://localhost:5194/health

# 3. Configure CLI to use shared API
export KM_API_ENDPOINT="http://localhost:5194"

# 4. Test CLI integration
./km auth status
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp
```

**Shared Environment Endpoints:**
- **API**: `http://localhost:5194`
- **Swagger UI**: `http://localhost:5194/swagger`
- **Health**: `http://localhost:5194/health`
- **pgAdmin** (with tools profile): `http://localhost:5050`

### Plugin Management
```bash
# List available plugins from API
./km plugins list

# Install plugin (checks local .kmpkg first, then API registry)
./km plugins install console-logger

# Update plugins
./km plugins update

# Remove plugin
./km plugins remove console-logger

# Setup plugins directory
./km plugins install
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

# Plugin configuration
KM_PLUGINS_DIR="/custom/plugins/path"      # Custom plugins directory (default: ~/.km/plugins)

# Advanced configuration  
KM_BUFFER_SIZE="2097152"                   # Buffer size in bytes (default: 1MB)
KM_BATCH_SIZE="20"                         # Batch size for API requests
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

- **`internal/auth/`** - JWT authentication and subscription management
  - `jwt_plugin_authenticator.go` - Plugin-specific JWT authentication
  - `subscription.go` - Tier-based access control
  - `token_cache.go` - 5-minute authentication caching

- **`internal/config/`** - Unified configuration system
  - `unified_loader.go` - Multi-source configuration loading
  - `unified_storage.go` - Configuration persistence
  - `service.go` - Configuration service orchestration

- **`internal/plugins/`** - Plugin system implementation
  - `manager.go` - Plugin lifecycle management
  - `sdk_integration.go` - SDK plugin adapter
  - `provisioning.go` - Plugin auto-provisioning

- **`internal/interfaces/cli/`** - Cobra-based CLI interface
  - `plugins.go` - Plugin management commands
  - `monitor.go` - MCP server monitoring
  - `init.go` - Auto-configuration discovery

- **`internal/streaming/`** - JSON-RPC proxy
  - `proxy.go` - Transparent bidirectional message forwarding

### Key Components

1. **Plugin Manager** (`internal/plugins/manager.go`)
   - Handles local .kmpkg and API registry plugins
   - JWT-based authentication with 5-minute caching
   - Multi-tier access control (Free, Pro, Enterprise)

2. **Configuration Service** (`internal/config/service.go`)
   - Unified configuration loading from multiple sources
   - Transparent source tracking and validation
   - API key management and persistence

3. **Stream Proxy** (`internal/streaming/proxy.go`) 
   - Transparent bidirectional JSON-RPC proxying
   - Sub-millisecond message forwarding with event generation

4. **JWT Authenticator** (`internal/auth/jwt_plugin_authenticator.go`)
   - Plugin-specific token validation
   - Real-time subscription validation with kilometers-api
   - Graceful degradation for unauthorized access

## Plugin Architecture

The CLI implements a sophisticated security model with local and API-based plugin support:

### Plugin Distribution Formats
- **Local .kmpkg Packages**: Self-contained plugin packages with metadata
- **API Registry Plugins**: Customer-specific binaries from kilometers-api
- **Development Plugins**: Local binaries for testing

### Security Model
- **Customer Isolation**: Each plugin built uniquely per customer with embedded secrets
- **Binary Signatures**: Digital signature validation for tamper detection  
- **JWT Authentication**: Time-limited tokens with feature-based authorization
- **Subscription Tiers**: Free (console only), Pro (API logging), Enterprise (custom plugins)

### Plugin Installation Flow
1. **Local Discovery**: Check `~/.km/plugins` for .kmpkg packages first
2. **API Fallback**: Download from kilometers-api registry if not found locally
3. **Authentication**: Validate plugin permissions against user subscription
4. **Loading**: Initialize plugin with JWT token
5. **Message Routing**: Forward MCP messages to authorized plugins

### Plugin SDK Integration
- **Shared Types**: Core types defined in kilometers-plugins-sdk repository
- **Extended Metadata**: .kmpkg packages include platform compatibility and dependencies
- **Backwards Compatibility**: Existing API-based plugins continue to work

### .kmpkg Package Support
```bash
# .kmpkg packages are discovered from configured plugin directory
export KM_PLUGINS_DIR="~/.km/plugins"  # Default location
km plugins install plugin-name         # Checks local .kmpkg files first

# Local packages take priority over API registry
# Fallback to API if plugin not found locally
```

### SDK Type Definitions
The kilometers-plugins-sdk repository defines core types:
- **PluginInfo**: Extended with Author, Platforms, CLI version constraints
- **KmpkgMetadata**: Complete package metadata with dependencies
- **KmpkgPackage**: File system representation with metadata
- **IsCompatible()**: Platform and version validation

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

## Memory Bank Integration

The repository includes a comprehensive Memory Bank system in `memory-bank/` that provides:

### Core Files
- **projectbrief.md** - Foundation document defining requirements and goals
- **productContext.md** - Problems solved and user experience goals  
- **activeContext.md** - Current work focus and recent changes
- **systemPatterns.md** - System architecture and design patterns
- **techContext.md** - Technologies used and development setup
- **progress.md** - Current status and what's left to build

### Cursor Rules Integration
The `.cursor/rules/memory-bank.mdc` file provides guidance for maintaining project memory across sessions and should be consulted for context-aware development.

## Important Notes

- This is a transparent proxy for MCP (Model Context Protocol) servers
- The system is stateless and event-driven for scalability
- Plugin authentication requires active internet connection for tier validation
- All plugin communication uses gRPC with HashiCorp go-plugin framework
- Local .kmpkg packages are discovered before API registry lookup
- The binary uses go-plugin for secure plugin isolation and communication