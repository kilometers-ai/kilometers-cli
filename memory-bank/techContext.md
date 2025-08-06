# Technical Context - Kilometers CLI

## Technology Stack

### Core Language
**Go 1.21+**
- **Rationale**: Cross-platform compilation, excellent concurrency, minimal runtime dependencies
- **Benefits**: Single binary distribution, strong standard library, mature ecosystem
- **Trade-offs**: Less familiar to some developers, but excellent for CLI tools

### Key Dependencies

#### CLI Framework
**Cobra CLI** (`github.com/spf13/cobra`)
- Industry standard for Go CLI applications
- Excellent flag parsing and command organization
- Built-in help generation and completion
- Used by kubectl, docker, and other major tools

#### JSON Processing
**Standard Library** (`encoding/json`)
- No external dependencies for JSON-RPC parsing
- High performance with streaming support
- Well-tested and reliable

#### Process Management
**Standard Library** (`os/exec`, `context`)
- Native process execution and management
- Cross-platform process control
- Context-based cancellation and timeouts

#### Testing Framework
**Standard Library + Testify** (`github.com/stretchr/testify`)
- Standard Go testing with enhanced assertions
- Mock generation capabilities
- Property-based testing with `github.com/leanovate/gopter`

#### Plugin Framework
**Go-Plugin** (`github.com/hashicorp/go-plugin`)
- HashiCorp's plugin system for out-of-process plugins
- GRPC-based communication for plugin isolation
- Cross-platform process management and lifecycle

#### Protocol Buffers
**Protocol Buffers + GRPC** (`google.golang.org/protobuf`, `google.golang.org/grpc`)
- Type-safe plugin communication protocol
- High-performance serialization and RPC
- Generated Go stubs from `.proto` definitions

## Development Environment

### Prerequisites
```bash
# Go installation
go version # >= 1.21

# Protocol Buffer Compiler
brew install protobuf  # macOS
# OR apt-get install protobuf-compiler  # Ubuntu

# Development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/vektra/mockery/v2@latest
go install github.com/goreleaser/goreleaser@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Project Setup
```bash
# Initialize Go module
go mod init github.com/kilometers-ai/kilometers-cli

# Install dependencies
go get github.com/spf13/cobra@latest
go get github.com/stretchr/testify@latest
go get github.com/hashicorp/go-plugin@latest
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest

# Generate protocol buffer code
./scripts/generate-proto.sh
```

### Build Configuration
```yaml
# .goreleaser.yml
project_name: km
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
```

## Technical Constraints

### Performance Requirements
- **Latency**: <10ms overhead per JSON-RPC message
- **Memory**: <50MB resident memory during monitoring
- **Throughput**: Handle 1000+ messages/second
- **Payload Size**: Support 1MB+ individual messages

### Platform Support
- **Linux**: amd64, arm64 (primary deployment target)
- **macOS**: amd64, arm64 (development environment)
- **Windows**: amd64 (compatibility target)

### Security Considerations
- **Process Isolation**: Server commands run as child processes
- **Stream Security**: No modification of JSON-RPC content
- **File Permissions**: Configuration files written with restricted permissions
- **Environment Variables**: Sensitive data handling for API keys

## JSON-RPC 2.0 Specification

### Message Format
```json
// Request
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {...},
  "id": 1
}

// Response
{
  "jsonrpc": "2.0",
  "result": {...},
  "id": 1
}

// Notification
{
  "jsonrpc": "2.0",
  "method": "notification",
  "params": {...}
}
```

### MCP-Specific Extensions
- **Initialize**: Server capability negotiation
- **Tools**: Tool discovery and execution
- **Resources**: Resource access and management
- **Sampling**: Content sampling for AI context

## Stream Processing Architecture

### Message Framing
```go
// Line-based framing (most common)
scanner := bufio.NewScanner(stream)
scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max token size

// Length-prefixed framing (alternative)
header := make([]byte, 4)
io.ReadFull(stream, header)
length := binary.BigEndian.Uint32(header)
```

### Concurrent Stream Handling
```go
type StreamManager struct {
    stdin  chan []byte
    stdout chan []byte
    stderr chan []byte
    errors chan error
}
```

## Build and Release Pipeline

### Local Development
```bash
# Generate protobuf code (when .proto files change)
./scripts/generate-proto.sh

# Development build
go build -o km ./cmd

# Build plugin examples
cd examples/plugins/console-logger && go build -o ~/.km/plugins/km-plugin-console-logger ./main.go

# Run tests
go test ./...

# Lint code
golangci-lint run

# Integration test
./test-mcp-monitoring.sh
```

### Cross-Platform Builds
```bash
# Build for all platforms
goreleaser build --snapshot --rm-dist

# Build specific platform
GOOS=linux GOARCH=amd64 go build -o km-linux ./cmd
```

### Release Process
1. **Version Tagging**: Semantic versioning (v1.0.0)
2. **Automated Building**: GoReleaser with GitHub Actions
3. **Asset Distribution**: Binary releases + install scripts
4. **Documentation**: Auto-generated CLI docs

## Configuration Management

### Environment Variables
```bash
export KM_DEBUG=true              # Enable debug logging
export KM_LOG_FORMAT=json         # Output format (json|text)
export KILOMETERS_API_KEY=xxx     # API integration key
```

### Configuration File
```yaml
# ~/.km/config.yaml
debug: false
log_format: "text"
monitor:
  buffer_size: "1MB"
  timeout: "30s"
```

## Dependencies and Constraints

### Zero External Runtime Dependencies
- Single binary deployment
- No Python, Node.js, or other runtime requirements
- Minimal system library dependencies

### Backward Compatibility
- Maintain CLI interface stability
- Graceful handling of unknown message formats

### Monitoring Overhead
- Non-intrusive monitoring (never block MCP communication)
- Configurable buffer sizes and timeouts
- Graceful degradation on resource constraints

## Error Handling Strategy

### Categories
1. **Fatal Errors**: Prevent monitoring startup
2. **Recoverable Errors**: Log but continue monitoring
3. **Warning Conditions**: Report but don't impact functionality

### Logging Levels
```go
type LogLevel int

const (
    LogError LogLevel = iota
    LogWarn
    LogInfo
    LogDebug
)
```

### Error Context
- Rich error context with operation details
- Stack traces in debug mode
- User-friendly error messages
- Actionable error resolution guidance 