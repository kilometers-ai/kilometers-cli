# Technical Context: Technologies and Development

## Technology Stack

### Core Language: Go 1.24.4+
**Why Go**: 
- Excellent concurrency support for real-time stream processing
- Single binary deployment ideal for CLI tools
- Strong standard library for network and process management
- Cross-platform compilation for multi-OS support
- Memory efficiency for long-running monitoring processes

### Primary Dependencies

#### CLI Framework: Cobra v1.9.1
```go
github.com/spf13/cobra v1.9.1
```
- **Purpose**: Command-line interface structure and flag parsing
- **Benefits**: Industry standard, rich feature set, excellent documentation
- **Usage**: All CLI commands, subcommands, and argument handling

#### Testing Framework: Testify v1.10.0  
```go
github.com/stretchr/testify v1.10.0
```
- **Purpose**: Assertions, mocking, and test suites
- **Benefits**: Comprehensive testing utilities, familiar API
- **Usage**: Unit tests, integration tests, mock implementations

#### Property-Based Testing: Rapid v1.2.0
```go
pgregory.net/rapid v1.2.0
```
- **Purpose**: Property-based testing for complex domain logic
- **Benefits**: Finds edge cases in session and event handling
- **Usage**: Testing session batching, event ordering, filter logic

## Development Environment

### Prerequisites
```bash
# Required
go version      # 1.24.4+
git --version   # Any recent version

# Optional but recommended  
docker --version          # For integration testing
docker-compose --version  # For test environment
az --version             # For Azure deployments (future)
```

### Project Setup
```bash
# Clone and setup
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli

# Install dependencies
go mod download
go mod tidy

# Verify setup
go build ./...
go test ./...
```

### Build System

#### Makefile Targets
```makefile
make help           # Show available commands
make test           # Run all tests
make test-coverage  # Run tests with coverage
make build          # Build binary
make install        # Install to GOPATH
make clean          # Clean artifacts
```

#### Build Scripts
- `build-releases.sh`: Cross-platform release builds
- `local-dev-setup.sh`: Local development environment setup
- `scripts/run-tests.sh`: Comprehensive testing with multiple configurations

### Development Workflow

#### Local Development
```bash
# Quick iteration cycle
go run cmd/main.go --help                    # Test CLI
go run cmd/main.go monitor echo "test"       # Test monitoring
go test -short ./...                         # Fast unit tests
go test -v ./integration_test/...           # Integration tests
```

#### Testing Strategy

##### Unit Tests
```bash
# Run specific package tests
go test ./internal/core/event/...
go test ./internal/core/session/...
go test ./internal/application/services/...

# With coverage
go test -cover ./internal/...
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

##### Integration Tests  
```bash
# Full integration test suite
go test -v -timeout 180s ./integration_test/...

# Docker-based testing
docker-compose -f docker-compose.test.yml up --build
```

##### Property-Based Tests
```bash
# Session behavior testing
go test -v ./internal/core/session/ -run TestSession_PropertyBased
```

## Architecture Constraints

### Performance Requirements
- **Message Throughput**: Handle 1000+ MCP messages/second
- **Memory Usage**: Bounded memory for long-running sessions
- **Startup Time**: Sub-second CLI command execution
- **Binary Size**: Single binary under 50MB

### Platform Support
- **Primary**: Linux x64, macOS ARM64/x64
- **Secondary**: Windows x64
- **Future**: ARM64 Linux for container deployments

### Protocol Constraints
- **MCP Version**: Compatible with MCP 2024-11-05 specification
- **JSON-RPC**: Strict JSON-RPC 2.0 compliance
- **Message Framing**: Newline-delimited JSON for stdio transport
- **Buffer Limits**: Handle messages up to 10MB (configurable)

## Infrastructure Integration

### CI/CD Pipeline (GitHub Actions)
```yaml
# Key workflows
.github/workflows/test.yml      # Test on push/PR
.github/workflows/release.yml   # Build and release
.github/workflows/security.yml  # Security scanning
```

### Release Process
1. **Automated Testing**: All tests must pass
2. **Cross-Platform Builds**: Linux, macOS, Windows binaries
3. **Version Tagging**: Semantic versioning
4. **Release Assets**: Binaries and checksums
5. **Installation Scripts**: Updated download URLs

### Deployment Targets
- **GitHub Releases**: Primary distribution method
- **Direct Downloads**: Via install.sh/install.ps1 scripts
- **Future**: Package managers (Homebrew, apt, etc.)

## Development Tools

### Code Quality
```bash
# Static analysis
go vet ./...
go fmt ./...

# Optional: golangci-lint
golangci-lint run

# Security scanning
gosec ./...
```

### IDE Configuration
**VS Code Settings** (`.vscode/settings.json`):
```json
{
    "go.lintTool": "golangci-lint",
    "go.testTimeout": "300s",
    "go.coverOnSave": true,
    "go.coverOnSingleTest": true
}
```

### Debugging
```bash
# Enable debug logging
export KM_DEBUG=true
go run cmd/main.go monitor --debug python server.py

# Verbose test output
go test -v -count=1 ./internal/...

# Race condition detection
go test -race ./internal/...
```

## Configuration Management

### Configuration Sources (Priority Order)
1. **Command Line Flags**: `--api-key`, `--api-url`, etc.
2. **Environment Variables**: `KM_API_KEY`, `KM_DEBUG`, etc.
3. **Configuration File**: `~/.km/config.json`
4. **Built-in Defaults**: Sensible defaults for all settings

### Configuration Schema
```json
{
  "api_endpoint": "https://api.dev.kilometers.ai",
  "api_key": "",
  "batch_size": 10,
  "flush_interval": "30s",
  "debug": false,
  "enable_risk_detection": true,
  "method_whitelist": [],
  "method_blacklist": ["ping", "pong"],
  "payload_size_limit": 1048576,
  "high_risk_methods_only": false,
  "exclude_ping_messages": true,
  "minimum_risk_level": "low"
}
```

## Security Considerations

### API Key Management
- **Storage**: Local configuration file with appropriate permissions
- **Transmission**: HTTPS only for API communication
- **Rotation**: Support for API key updates without restart

### Process Security
- **Isolation**: Monitored processes run with original permissions
- **Input Validation**: All MCP messages validated before processing
- **Resource Limits**: Bounded memory and CPU usage

### Data Handling
- **Local Storage**: Minimal local data retention
- **Transmission**: Encrypted communication with platform
- **Privacy**: No sensitive data logged or stored locally

## Known Technical Debt

### Current Issues
1. **Buffer Size Limitations**: Fixed 4KB buffers for large MCP messages
2. **Message Parsing**: Incomplete JSON-RPC 2.0 parsing implementation
3. **Error Handling**: Inconsistent error context and recovery
4. **Resource Management**: Potential memory leaks in long sessions

### Planned Improvements
1. **Streaming JSON Parser**: Handle arbitrarily large messages
2. **Connection Pooling**: Efficient API communication
3. **Metrics Collection**: Detailed performance and usage metrics
4. **Plugin Architecture**: Extensible risk detection and filtering

This technical foundation provides a solid base for reliable MCP monitoring while maintaining development velocity and code quality. 