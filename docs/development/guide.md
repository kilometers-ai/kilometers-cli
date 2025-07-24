# Kilometers CLI Developer Guide

## Table of Contents
1. [Project Overview](#project-overview)
2. [Prerequisites & Setup](#prerequisites--setup)
3. [Building the Project](#building-the-project)
4. [Running the CLI](#running-the-cli)
5. [Testing](#testing)
6. [Docker Development Environment](#docker-development-environment)
7. [Development Workflow](#development-workflow)
8. [Architecture Guide](#architecture-guide)
9. [Troubleshooting](#troubleshooting)
10. [CI/CD & Releases](#cicd--releases)

## Project Overview

Kilometers CLI is a monitoring tool for Model Context Protocol (MCP) events, built using Domain-Driven Design (DDD) and Hexagonal Architecture principles. It monitors AI assistant interactions, analyzes risks, and provides insights through the Kilometers platform.

### Key Features
- **MCP Event Monitoring**: Monitors MCP server processes and collects events
- **Risk Analysis**: Analyzes events for potential security risks
- **Event Filtering**: Configurable filtering to reduce noise
- **Session Management**: Groups events into logical sessions
- **Platform Integration**: Sends data to Kilometers platform for visualization

### Technology Stack
- **Go 1.24.4+**: Primary language
- **Cobra**: CLI framework
- **WebSocket**: Real-time MCP communication
- **JSON-RPC 2.0**: MCP protocol implementation
- **Docker**: Containerized testing
- **GitHub Actions**: CI/CD

## Prerequisites & Setup

### Required Software
```bash
# Go 1.24.4 or later
go version

# Git
git --version

# Docker & Docker Compose (for testing)
docker --version
docker-compose --version

# Optional: Azure CLI (for deployments)
az --version
```

### Project Setup
```bash
# Clone the repository
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli

# Install dependencies
go mod download

# Verify setup
go mod tidy
go build ./...
```

### Development Dependencies
```bash
# Install testing dependencies (already in go.mod)
# - github.com/stretchr/testify: Testing framework
# - pgregory.net/rapid: Property-based testing
# - github.com/spf13/cobra: CLI framework
```

## Building the Project

### Local Development Build
```bash
# Build for current platform
go build -o km cmd/main.go

# Run the binary
./km --help
```

### Cross-Platform Builds
```bash
# Build for all supported platforms
./build-releases.sh

# Manual cross-compilation examples
GOOS=linux GOARCH=amd64 go build -o km-linux-amd64 cmd/main.go
GOOS=darwin GOARCH=arm64 go build -o km-darwin-arm64 cmd/main.go
GOOS=windows GOARCH=amd64 go build -o km-windows-amd64.exe cmd/main.go
```

### Build with Version Information
```bash
# Build with version and build info
VERSION="0.1.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(git rev-parse --short HEAD)

go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.Commit=$COMMIT" -o km cmd/main.go
```

## Running the CLI

### Basic Usage
```bash
# Build and run
go run cmd/main.go --help

# Or use built binary
./km --help
```

### Available Commands
```bash
# Initialize configuration
./km init

# Configure settings
./km config set api-key "your-api-key"
./km config get api-key

# Monitor MCP processes
./km monitor --config ~/.km/config.json

# Setup integrations
./km setup

# Validate configuration
./km validate

# Update CLI
./km update

# Generate shell completions
./km completion bash > km_completion.sh
```

### Configuration
```bash
# Default config location: ~/.km/config.json
# Override with --config flag
./km --config ./custom-config.json monitor

# Environment variables (override config file)
export KM_API_KEY="your-api-key"
export KM_API_URL="https://api.kilometers.ai"
export KM_DEBUG="true"
./km monitor
```

## Testing

### Unit Tests

#### Run All Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Domain Layer Tests (100% Coverage)
```bash
# Test individual domain components
go test -v ./internal/core/event/...
go test -v ./internal/core/session/...
go test -v ./internal/core/risk/...
go test -v ./internal/core/filtering/...

# Property-based testing with rapid
go test -v ./internal/core/event -test.timeout=30s
```

#### Application Layer Tests
```bash
# Test application services and commands
go test -v ./internal/application/...
```

#### Infrastructure Layer Tests
```bash
# Test infrastructure components
go test -v ./internal/infrastructure/...
```

### Integration Tests

#### Prerequisites for Integration Tests
```bash
# Ensure Docker is running
docker ps

# Install Node.js (for mock MCP server scripts)
node --version  # 16+ recommended
```

#### Run Integration Tests
```bash
# Run all integration tests
go test -v ./integration_test/...

# Run specific integration test suites
go test -v ./integration_test/ -run TestCLI
go test -v ./integration_test/ -run TestProcessMonitoring
go test -v ./integration_test/ -run TestAPIIntegration
go test -v ./integration_test/ -run TestConfiguration

# Run with timeout for longer tests
go test -v ./integration_test/... -timeout=5m

# Run with parallel execution
go test -v ./integration_test/... -parallel=4
```

#### Performance Tests
```bash
# Run performance-focused integration tests
go test -v ./integration_test/ -run "Performance|Stress|Load"

# Run with memory profiling
go test -v ./integration_test/ -memprofile=mem.prof -cpuprofile=cpu.prof

# View profiles
go tool pprof mem.prof
go tool pprof cpu.prof
```

### Test Coverage Analysis
```bash
# Generate comprehensive coverage report
go test -coverprofile=coverage.out ./...

# View coverage by package
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
# or browse to file on other platforms

# Check specific coverage thresholds
go test -cover ./... | grep -E "(coverage|%)"
```

## Docker Development Environment

### Using Docker for Testing

#### Start Test Environment
```bash
# Start all test services
docker-compose -f docker-compose.test.yml up -d

# Check service health
docker-compose -f docker-compose.test.yml ps

# View logs
docker-compose -f docker-compose.test.yml logs -f mock-mcp
docker-compose -f docker-compose.test.yml logs -f mock-api
```

#### Run Tests in Docker
```bash
# Run integration tests in Docker
docker-compose -f docker-compose.test.yml run --rm integration-test

# Run stress tests
docker-compose -f docker-compose.test.yml run --rm stress-test

# Run with custom environment
docker-compose -f docker-compose.test.yml run --rm \
  -e TEST_TIMEOUT=300s \
  -e TEST_PARALLEL=8 \
  integration-test
```

#### Individual Service Testing
```bash
# Test against mock MCP server only
docker-compose -f docker-compose.test.yml up -d mock-mcp
go test -v ./integration_test/ -run TestProcessMonitoring

# Test against mock API server only
docker-compose -f docker-compose.test.yml up -d mock-api
go test -v ./integration_test/ -run TestAPIIntegration
```

#### Cleanup
```bash
# Stop and remove all containers
docker-compose -f docker-compose.test.yml down

# Remove volumes and networks
docker-compose -f docker-compose.test.yml down -v

# Clean up test artifacts
docker system prune -f
```

### Custom Docker Testing

#### Build Custom Test Image
```bash
# Build integration test image
docker build -f test/docker/Dockerfile.integration -t km-integration-test .

# Run custom tests
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  km-integration-test \
  go test -v ./integration_test/...
```

## Development Workflow

### 1. Development Setup
```bash
# Set up pre-commit hooks (optional)
echo "#!/bin/bash
go fmt ./...
go vet ./...
go test ./...
" > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Set up VS Code settings (optional)
mkdir -p .vscode
cat > .vscode/settings.json << 'EOF'
{
    "go.lintTool": "golangci-lint",
    "go.testTimeout": "300s",
    "go.coverOnSave": true,
    "go.coverOnSingleTest": true
}
EOF
```

### 2. Feature Development
```bash
# Create feature branch
git checkout -b feature/new-feature

# Develop with TDD approach
# 1. Write failing test
go test -v ./internal/core/... -run TestNewFeature

# 2. Implement minimum code to pass
# 3. Refactor and improve

# 4. Run all tests
go test ./...

# 5. Run integration tests
go test -v ./integration_test/...
```

### 3. Code Quality Checks
```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Install and run golangci-lint (optional)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run

# Check for security issues (optional)
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
gosec ./...
```

### 4. Testing Workflow
```bash
# Quick tests during development
go test -short ./...

# Full test suite before commit
go test ./...
go test -v ./integration_test/...

# Performance baseline
go test -bench=. ./...
```

### 5. Documentation Updates
```bash
# Update CLI help documentation
go run cmd/main.go --help > docs/cli-help.txt

# Generate API documentation
go doc -all ./internal/core/... > docs/api.md

# Update README if needed
# Update this guide.md if workflow changes
```

## Architecture Guide

### Project Structure
```
kilometers-cli/
├── cmd/                          # Application entry point
│   └── main.go                   # Main executable
├── internal/                     # Private application code
│   ├── application/              # Application layer (DDD)
│   │   ├── commands/             # Command handlers
│   │   ├── services/             # Application services
│   │   └── ports/                # Interface definitions
│   ├── core/                     # Domain layer (DDD)
│   │   ├── event/                # Event domain model
│   │   ├── session/              # Session aggregate
│   │   ├── risk/                 # Risk analysis domain
│   │   └── filtering/            # Event filtering domain
│   ├── infrastructure/           # Infrastructure layer
│   │   ├── api/                  # External API client
│   │   ├── config/               # Configuration repository
│   │   └── monitoring/           # Process monitoring
│   └── interfaces/               # Interface adapters
│       ├── cli/                  # CLI interface
│       └── di/                   # Dependency injection
├── integration_test/             # Integration tests
├── test/                         # Test utilities and mocks
└── docker-compose.test.yml       # Test environment
```

### Domain-Driven Design Layers

#### 1. Domain Layer (`internal/core/`)
Pure business logic, no external dependencies:
```bash
# Test domain layer in isolation
go test -v ./internal/core/...
```

#### 2. Application Layer (`internal/application/`)
Use cases and application services:
```bash
# Test application logic
go test -v ./internal/application/...
```

#### 3. Infrastructure Layer (`internal/infrastructure/`)
External integrations and technical concerns:
```bash
# Test infrastructure components
go test -v ./internal/infrastructure/...
```

#### 4. Interface Layer (`internal/interfaces/`)
User interfaces and external adapters:
```bash
# Test CLI interfaces
go test -v ./internal/interfaces/...
```

### Key Architectural Patterns

1. **Hexagonal Architecture**: Clean separation of concerns
2. **Dependency Injection**: Loose coupling between layers
3. **Repository Pattern**: Data access abstraction
4. **Command Pattern**: Encapsulated operations
5. **Observer Pattern**: Event handling and monitoring

## Troubleshooting

### Common Build Issues

#### Go Version Mismatch
```bash
# Check Go version
go version

# Update Go modules
go mod tidy

# Clear module cache
go clean -modcache
```

#### Dependency Issues
```bash
# Reset dependencies
rm go.sum
go mod download
go mod tidy

# Verify dependencies
go mod verify
```

### Common Test Issues

#### Integration Test Failures
```bash
# Check Docker status
docker ps
docker-compose -f docker-compose.test.yml ps

# Reset test environment
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.test.yml up -d

# Check test logs
docker-compose -f docker-compose.test.yml logs
```

#### Port Conflicts
```bash
# Check for port usage
lsof -i :8080  # Mock MCP server
lsof -i :8081  # Mock API server

# Kill conflicting processes
kill -9 <PID>
```

#### Permission Issues
```bash
# Fix binary permissions
chmod +x km
chmod +x build-releases.sh

# Fix test file permissions
chmod +x test/docker/*.sh
```

### Performance Issues

#### Memory Usage
```bash
# Profile memory usage
go test -memprofile=mem.prof ./integration_test/...
go tool pprof mem.prof

# Check for memory leaks
go test -race ./...
```

#### CPU Usage
```bash
# Profile CPU usage
go test -cpuprofile=cpu.prof ./integration_test/...
go tool pprof cpu.prof
```

### Debugging

#### Enable Debug Mode
```bash
# Debug CLI execution
./km --debug monitor

# Debug with environment variable
export KM_DEBUG=true
./km monitor
```

#### Verbose Testing
```bash
# Verbose test output
go test -v ./...

# Test with race detection
go test -race ./...

# Test with detailed output
go test -v -x ./...
```

## CI/CD & Releases

### GitHub Actions

The project uses GitHub Actions for CI/CD:
- **Unit Tests**: Run on every push and PR
- **Integration Tests**: Run with Docker environment
- **Cross-Platform Builds**: Generate binaries for all platforms
- **Release Automation**: Tag-based releases

### Manual Release Process

```bash
# 1. Update version
VERSION="0.2.0"
git tag -a "v$VERSION" -m "Release v$VERSION"

# 2. Build releases
./build-releases.sh

# 3. Test releases
./releases/km-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) --version

# 4. Push tag
git push origin "v$VERSION"

# 5. Upload to release (automated via GitHub Actions)
```

### Azure Storage Upload (Optional)
```bash
# Upload to Azure Storage
az storage blob upload-batch \
  --source releases \
  --destination releases/latest \
  --account-name STORAGE_ACCOUNT
```

---

## Quick Reference

### Essential Commands
```bash
# Setup
go mod download && go build -o km cmd/main.go

# Test everything
go test ./... && go test -v ./integration_test/...

# Build releases
./build-releases.sh

# Docker testing
docker-compose -f docker-compose.test.yml up --build
```

### Performance Benchmarks
- **Unit Tests**: < 5 seconds
- **Integration Tests**: < 2 minutes
- **Docker Tests**: < 3 minutes
- **Cross-Platform Builds**: < 30 seconds

### Coverage Goals
- **Domain Layer**: 100% (achieved)
- **Application Layer**: > 95%
- **Infrastructure Layer**: > 80%
- **Overall Project**: > 85%

---

*This guide is maintained by the Kilometers CLI development team. For updates or questions, please refer to the project repository.* 