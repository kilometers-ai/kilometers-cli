# CLI Development with Shared API Environment

This guide covers developing the kilometers-cli against a shared kilometers-api environment using Docker Compose.

## Overview

The shared development environment provides:
- **Real API Integration** - Test against actual API endpoints (not mocks)
- **Database Persistence** - Shared PostgreSQL with consistent state
- **Authentication Testing** - Full plugin authentication workflow
- **Production-like Setup** - Mirror production architecture locally

## Prerequisites

### Repository Structure
Ensure both repositories are cloned as siblings:
```
parent-directory/
├── kilometers-api/          # Contains docker-compose.shared.yml
└── kilometers-cli/          # This repository
```

### Required Tools
- Docker and Docker Compose
- Go 1.21+ (for CLI development)
- curl (for testing)

## Quick Start Guide

### 1. Start Shared Environment
```bash
# Navigate to kilometers-api and start shared services
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml up -d

# Verify services are healthy
curl http://localhost:5194/health
```

Expected health response:
```json
{
  "status": "Healthy",
  "checks": {
    "api": {"status": "Healthy"},
    "database": {"status": "Healthy"}
  }
}
```

### 2. Configure CLI
```bash
# Return to CLI directory
cd ../kilometers-cli

# Configure CLI to use shared API
export KM_API_ENDPOINT="http://localhost:5194"

# Verify configuration
./km auth status
```

### 3. Test Integration
```bash
# Test basic connectivity
./km auth status

# Test with API key (if you have one)
./km auth login --api-key "km_test_your_key_here"

# Test monitoring functionality
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp
```

## Development Workflows

### Feature Development Workflow

1. **Start Environment**
   ```bash
   cd ../kilometers-api
   docker-compose -f docker-compose.shared.yml up -d
   ```

2. **Develop CLI Changes**
   ```bash
   cd ../kilometers-cli
   
   # Make your changes to CLI code
   vim internal/config/service.go
   
   # Build and test
   go build -o km ./cmd/main.go
   ./km auth status
   ```

3. **Test Integration**
   ```bash
   # Test specific CLI features
   ./km monitor -- your-test-server --args
   
   # Run integration tests
   ./scripts/test/run-tests.sh
   
   # Test plugin authentication
   go test ./test/integration/ -v
   ```

4. **Debug Issues**
   ```bash
   # Check API logs
   cd ../kilometers-api
   docker-compose -f docker-compose.shared.yml logs -f api
   
   # Check database state
   docker-compose -f docker-compose.shared.yml --profile tools up -d
   # Visit http://localhost:5050 for pgAdmin
   ```

### Plugin Development Workflow

For developing CLI plugin features:

1. **Setup Plugin Environment**
   ```bash
   # Start shared environment
   cd ../kilometers-api
   docker-compose -f docker-compose.shared.yml up -d
   
   # Configure plugin directory
   export KM_PLUGINS_DIR="$HOME/.km/plugins"
   mkdir -p "$KM_PLUGINS_DIR"
   ```

2. **Test Plugin Authentication**
   ```bash
   cd ../kilometers-cli
   
   # Test plugin discovery
   ./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp
   
   # Check plugin authentication logs
   cd ../kilometers-api
   docker-compose -f docker-compose.shared.yml logs -f api | grep -i plugin
   ```

3. **Debug Plugin Issues**
   ```bash
   # Enable debug logging
   export KM_DEBUG=true
   export KM_LOG_LEVEL=debug
   
   # Test with detailed logging
   ./km monitor -- your-mcp-server
   ```

### API-CLI Integration Testing

When developing features that require API changes:

1. **Modify API** (if needed)
   ```bash
   cd ../kilometers-api
   # Make API changes
   # Rebuild API container
   docker-compose -f docker-compose.shared.yml up --build api -d
   ```

2. **Test CLI Against Updated API**
   ```bash
   cd ../kilometers-cli
   go build -o km ./cmd/main.go
   ./km auth status  # Should work with updated API
   ```

3. **Run Full Integration Suite**
   ```bash
   # Run CLI integration tests
   go test ./test/integration/ -v
   
   # Run API integration tests
   cd ../kilometers-api
   dotnet test --filter "Category=Integration"
   ```

## Configuration Options

### Environment Variables

**API Endpoint Configuration:**
```bash
export KM_API_ENDPOINT="http://localhost:5194"    # Shared API
export KM_API_ENDPOINT="https://api.kilometers.ai" # Production API
```

**Plugin Configuration:**
```bash
export KM_PLUGINS_DIR="/custom/plugins/path"
export KM_DEBUG=true
export KM_LOG_LEVEL=debug
```

**Authentication:**
```bash
export KM_API_KEY="km_test_your_api_key"
```

### Configuration Files

Create `.env` file in CLI directory:
```bash
# .env file for shared development
KM_API_ENDPOINT=http://localhost:5194
KM_DEBUG=true
KM_LOG_LEVEL=debug
KM_PLUGINS_DIR=/Users/yourname/.km/plugins
```

## Testing Scenarios

### Basic Connectivity Tests
```bash
# Test API connectivity
curl http://localhost:5194/health

# Test CLI configuration
./km auth status

# Test CLI-API communication
./km monitor --help
```

### Authentication Tests
```bash
# Test without API key
./km auth status

# Test with API key
./km auth login --api-key "km_test_key"
./km auth status

# Test token refresh
# (Make API calls and verify tokens are refreshed)
```

### Plugin Integration Tests
```bash
# Test plugin discovery
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp

# Test plugin authentication
# (Check API logs for plugin authentication requests)

# Test different plugin tiers
# (Test with Free, Pro, Enterprise API keys)
```

### Error Handling Tests
```bash
# Test with API down
cd ../kilometers-api && docker-compose -f docker-compose.shared.yml stop api
cd ../kilometers-cli && ./km auth status

# Test with database down
cd ../kilometers-api && docker-compose -f docker-compose.shared.yml stop postgres
# API should gracefully handle database issues

# Test with invalid API keys
./km auth login --api-key "invalid_key"
```

## Debugging Guide

### API Issues
```bash
# Check API logs
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml logs -f api

# Check API health
curl http://localhost:5194/health

# Test API endpoints directly
curl http://localhost:5194/api/health
curl -H "Content-Type: application/json" \
     -X POST http://localhost:5194/api/auth/login \
     -d '{"apiKey":"your-key"}'
```

### Database Issues
```bash
# Check database logs
docker-compose -f docker-compose.shared.yml logs postgres

# Connect to database
docker exec -it kilometers-shared-postgres psql -U postgres -d kilometers_dev

# Check database health
docker-compose -f docker-compose.shared.yml ps postgres
```

### CLI Issues
```bash
# Enable debug logging
export KM_DEBUG=true
export KM_LOG_LEVEL=debug

# Check configuration loading
./km auth status

# Test specific commands
./km monitor --dry-run -- echo "test"
```

### Network Issues
```bash
# Check port availability
lsof -i :5194  # API port
lsof -i :5432  # Database port

# Test container networking
docker exec kilometers-shared-api ping kilometers-shared-postgres
```

## Cleanup and Reset

### Soft Reset (Keep Data)
```bash
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml restart
```

### Hard Reset (Clean State)
```bash
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml down -v
docker-compose -f docker-compose.shared.yml up -d
```

### Clean Development Environment
```bash
# Stop all containers
docker-compose -f docker-compose.shared.yml down -v

# Remove CLI build artifacts
cd ../kilometers-cli
go clean
rm -f km

# Clear CLI configuration
rm -rf ~/.config/kilometers/
rm -rf ~/.km/
```

## Performance Considerations

### API Response Times
The shared API runs in development mode with:
- Database connection pooling disabled for simplicity
- Verbose logging enabled
- No caching layers

Expect slower response times compared to production.

### Database Performance
The shared PostgreSQL runs with:
- Default PostgreSQL settings
- No performance tuning
- Full logging enabled

For performance testing, consider optimizing the database configuration.

### Container Resource Usage
Monitor container resource usage:
```bash
# Check container resource usage
docker stats

# View container details
docker-compose -f docker-compose.shared.yml ps
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: CLI Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        path: kilometers-cli
        
    - uses: actions/checkout@v4
      with:
        repository: kilometers-ai/kilometers-api
        path: kilometers-api
        
    - name: Start shared environment
      run: |
        cd kilometers-api
        docker-compose -f docker-compose.shared.yml up -d
        
    - name: Wait for API
      run: |
        timeout 60 bash -c 'until curl -f http://localhost:5194/health; do sleep 2; done'
        
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        
    - name: Run CLI integration tests
      run: |
        cd kilometers-cli
        export KM_API_ENDPOINT="http://localhost:5194"
        go test ./test/integration/... -v
```

## Troubleshooting Common Issues

### "Connection Refused" Errors
1. Verify API is running: `docker-compose -f docker-compose.shared.yml ps`
2. Check API logs: `docker-compose -f docker-compose.shared.yml logs api`
3. Test connectivity: `curl http://localhost:5194/health`

### "API Key Invalid" Errors
1. Verify API key format: `km_[env]_[key]`
2. Check API logs for authentication errors
3. Test with a known good API key

### Plugin Authentication Failures
1. Check plugin directory exists and is writable
2. Verify plugin binary permissions
3. Check API logs for plugin authentication requests
4. Test with a simpler MCP server first

### Database Migration Issues
1. Check database logs: `docker-compose -f docker-compose.shared.yml logs postgres`
2. Verify database health in API health endpoint
3. Connect to database manually to check schema
4. Restart API to retry migrations: `docker-compose -f docker-compose.shared.yml restart api`