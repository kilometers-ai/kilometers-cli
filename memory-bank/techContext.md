# Technical Context: Technology Stack and Implementation Details

## Core Technology Stack

### Language and Runtime
- **Go 1.24.4**: Primary development language
  - **Rationale**: Excellent concurrency primitives for MCP message processing
  - **Benefits**: Single binary deployment, cross-platform support, strong typing
  - **Trade-offs**: Verbose error handling, but acceptable for system tool

### CLI Framework
- **Cobra v1.9.1**: Command-line interface framework
  - **Features**: Command structure, flag parsing, help generation
  - **Integration**: Clean separation between CLI layer and business logic
  - **Command Structure**: `km [command] [flags] --server -- [mcp-server-command]`

### Architecture Patterns
- **Domain-Driven Design**: Clean separation of business logic
- **Hexagonal Architecture**: Ports and adapters for testability
- **CQRS**: Command and query responsibility separation
- **Event-Driven**: Natural fit for MCP message processing

## Protocol and Communication

### MCP Protocol Support
- **JSON-RPC 2.0**: Native Model Context Protocol implementation
  - **Message Types**: Requests, responses, notifications, errors
  - **Transport**: Newline-delimited JSON over stdout/stderr
  - **Compliance**: Full MCP specification adherence

### Platform Integration
- **HTTP/REST**: Kilometers platform API communication
  - **Authentication**: API key-based authentication
  - **Serialization**: JSON message serialization
  - **Error Handling**: Graceful degradation and retry logic

### Process Monitoring
- **os/exec**: MCP server process wrapping and monitoring
  - **Stream Processing**: Real-time stdout/stderr capture
  - **Message Framing**: Newline-delimited JSON parsing
  - **Resource Management**: Bounded memory usage and cleanup

## Core Dependencies

### Primary Dependencies
```go
// CLI and Configuration
github.com/spf13/cobra v1.9.1          // CLI framework
github.com/spf13/viper v1.21.0         // Configuration management

// Testing Framework
github.com/stretchr/testify v1.10.0    // Assertions and test structure
pgregory.net/rapid v1.1.0              // Property-based testing

// JSON Processing
encoding/json                          // Standard library JSON
```

### Standard Library Usage
```go
// Core Go packages
context                                // Context management
sync                                   // Concurrency primitives
time                                   // Time operations
os/exec                               // Process execution
bufio                                 // Buffered I/O
http                                  // HTTP client
```

### Removed Dependencies
The architecture simplification removed several packages:
- ❌ Complex filtering libraries
- ❌ Risk analysis dependencies
- ❌ Pattern matching libraries
- ❌ ML/AI scoring systems

## Project Structure

### Clean Architecture Layers
```
internal/
├── core/                    # Domain Layer (Pure Business Logic)
│   ├── event/              # Event entities and value objects
│   ├── session/            # Session aggregate roots
│   └── testfixtures/       # Test data builders
├── application/            # Application Layer (Use Cases)
│   ├── commands/           # Command DTOs and structures
│   ├── services/           # Application services and orchestration
│   └── ports/              # Interface definitions (ports)
├── infrastructure/        # Infrastructure Layer (Technical Concerns)
│   ├── api/               # API gateway implementations
│   ├── monitoring/        # Process monitor implementations
│   └── config/            # Configuration repository implementations
└── interfaces/            # Interface Layer (External Adapters)
    ├── cli/               # CLI command handlers
    └── di/                # Dependency injection container
```

### Package Dependencies
```
interfaces → application → core
infrastructure → application
```

**Dependency Rules**:
- Core layer has NO external dependencies
- Application layer depends only on core
- Infrastructure implements application ports
- Interface layer orchestrates everything

## Development Environment

### Required Tools
```bash
# Go toolchain
go version go1.24.4 darwin/arm64

# Build automation
make --version

# Development tools
golangci-lint --version    # Code linting
gofmt                      # Code formatting
go test                    # Testing
go mod                     # Dependency management
```

### Development Workflow
```bash
# Clone and setup
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli

# Install dependencies
go mod download

# Build and test
make build
make test

# Run locally
./km monitor --server -- npx -y @modelcontextprotocol/server-github
```

## Testing Strategy

### Testing Framework Stack
- **testify**: Assertions, test suites, and mocking
- **rapid**: Property-based testing for complex behaviors
- **Standard library**: Built-in testing and benchmarking

### Test Categories

#### 1. Unit Tests
```go
// Domain logic testing
func TestSession_AddEvent_BatchesCorrectly(t *testing.T) {
    session := session.NewSession(session.SessionConfig{BatchSize: 3})
    
    for i := 0; i < 5; i++ {
        event := testfixtures.NewEventBuilder().Build()
        batch := session.AddEvent(event)
        
        if i == 2 { // Third event triggers batch
            assert.Len(t, batch, 3)
        }
    }
}
```

#### 2. Integration Tests
```go
// End-to-end testing with mock servers
func TestMonitoring_RealMCPServer(t *testing.T) {
    container := setupTestContainer(t)
    
    result := container.MonitoringService().StartMonitoring(
        commands.StartMonitoringCommand{
            Command:   "npx",
            Arguments: []string{"-y", "@modelcontextprotocol/server-github"},
            BatchSize: 10,
        },
    )
    
    assert.NoError(t, result.Error)
    assert.NotEmpty(t, result.SessionID)
}
```

#### 3. Property-Based Tests
```go
// Invariant testing across random inputs
func TestSession_BatchSizeInvariants(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        batchSize := rapid.IntRange(1, 100).Draw(t, "batchSize")
        eventCount := rapid.IntRange(0, 1000).Draw(t, "eventCount")
        
        session := createSessionWithBatchSize(batchSize)
        totalBatched := addEventsAndCountBatched(session, eventCount)
        
        // Invariant: batched + pending = total
        assert.Equal(t, eventCount, totalBatched + session.PendingCount())
    })
}
```

### Test Infrastructure

#### Mock Servers
```bash
# Mock MCP server for testing
go run test/cmd/run_mock_server.go --port 8080

# Docker-based integration testing
docker-compose -f docker-compose.test.yml up
```

#### Test Data Builders
```go
// Test fixture builders for clean test setup
func NewEventBuilder() *EventBuilder {
    return &EventBuilder{
        method:    "test_method",
        direction: event.Inbound,
        payload:   []byte(`{"test": "data"}`),
    }
}

func NewSessionBuilder() *SessionBuilder {
    return &SessionBuilder{
        batchSize: 10,
    }
}
```

## Performance and Scalability

### Performance Characteristics
- **Latency**: <10ms monitoring overhead per message
- **Memory**: <50MB typical memory footprint
- **Throughput**: 1000+ messages/second processing capability
- **Resource Usage**: Bounded memory growth with batch processing

### Concurrency Model
```go
// Bounded concurrency for message processing
type MessageProcessor struct {
    eventQueue   chan Event         // Buffered event queue
    batchQueue   chan []Event       // Buffered batch queue
    workers      sync.WaitGroup     // Worker pool management
}

// Resource management
func (p *MessageProcessor) Start() {
    p.eventQueue = make(chan Event, 1000)    // Buffer 1000 events
    p.batchQueue = make(chan []Event, 100)   // Buffer 100 batches
    
    // Start worker goroutines
    for i := 0; i < 3; i++ {
        p.workers.Add(1)
        go p.processEvents()
    }
}
```

## Build and Deployment

### Build System
```makefile
# Makefile targets
build:      # Build single binary
test:       # Run all tests
clean:      # Clean build artifacts
install:    # Install to local system
release:    # Build release binaries
```

### Cross-Platform Builds
```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o releases/km-linux-amd64
GOOS=windows GOARCH=amd64 go build -o releases/km-windows-amd64.exe
GOOS=darwin GOARCH=amd64 go build -o releases/km-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o releases/km-darwin-arm64
```

### Release Automation
- **GitHub Actions**: Automated CI/CD pipeline
- **Semantic Versioning**: Automated version management
- **Release Assets**: Cross-platform binary distribution
- **Installation Scripts**: `curl -fsSL https://get.kilometers.ai/install.sh | sh`

## Configuration Management

### Configuration Sources (Precedence Order)
1. **CLI Flags**: Runtime command-line arguments
2. **Environment Variables**: `KM_API_HOST`, `KM_API_KEY`, `KM_BATCH_SIZE`, `KM_DEBUG`
3. **Configuration File**: `~/.kilometers/config.json`
4. **Defaults**: Built-in sensible defaults

### Configuration Structure
```go
type Configuration struct {
    APIHost   string `json:"api_host"`   // Platform API endpoint
    APIKey    string `json:"api_key"`    // Authentication key
    BatchSize int    `json:"batch_size"` // Events per batch (default: 10)
    Debug     bool   `json:"debug"`      // Debug mode flag
}
```

### Configuration Validation
```go
func (c *Configuration) Validate() error {
    if c.APIHost == "" {
        return ErrMissingAPIHost
    }
    if c.APIKey == "" {
        return ErrMissingAPIKey
    }
    if c.BatchSize < 1 || c.BatchSize > 1000 {
        return ErrInvalidBatchSize
    }
    return nil
}
```

## Error Handling and Observability

### Error Patterns
- **Domain Errors**: Business logic error types with clear codes
- **Infrastructure Errors**: Network, file system, process errors
- **Graceful Degradation**: Continue monitoring despite non-critical errors

### Logging Strategy
```go
// Structured logging with levels
log.WithFields(log.Fields{
    "session_id": sessionID,
    "event_count": len(events),
}).Info("Batch sent to platform")

log.WithError(err).WithField("command", cmd).
    Error("Failed to start MCP server process")
```

### Debugging Support
- **Debug Mode**: Verbose logging and detailed error context
- **Event Replay**: Capture and replay MCP message streams
- **Health Checks**: Built-in status reporting and validation

## Security Considerations

### API Security
- **Authentication**: API key-based authentication
- **Transport Security**: HTTPS for all platform communication
- **Key Management**: Environment variable or secure file storage

### Process Security
- **Process Isolation**: MCP servers run in separate processes
- **Resource Limits**: Bounded memory and processing resources
- **Input Validation**: All MCP messages validated before processing

---

This technical context provides the foundation for understanding how the kilometers CLI is built, tested, and deployed while maintaining production-ready quality and architectural excellence. 