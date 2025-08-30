# Kilometers CLI Test Infrastructure

This directory contains comprehensive testing scripts for the Kilometers CLI project, including Docker environment management, end-to-end testing, and integration test suites.

## 📁 Directory Structure

```
scripts/tests/
├── README.md                     # This file
├── docker-compose-up.sh          # Start Docker environments
├── docker-compose-down.sh        # Stop Docker environments
├── test-plugin-e2e.sh           # End-to-end plugin testing
├── run-tests.sh                  # Core test runner
├── test-and-check.sh            # Build and test verification
├── test-mcp-monitoring.sh       # MCP monitoring tests
├── test-plugin-integration.sh   # Plugin integration tests
└── test-plugin-provisioning.sh  # Plugin provisioning tests
```

## 🚀 Quick Start

### 1. Start Docker Environment

Choose your environment and start it:

```bash
# Interactive selection
./scripts/tests/docker-compose-up.sh

# Direct environment selection
./scripts/tests/docker-compose-up.sh shared    # Recommended for E2E testing
./scripts/tests/docker-compose-up.sh dev       # Local development
./scripts/tests/docker-compose-up.sh test      # Isolated testing
```

### 2. Run End-to-End Tests

```bash
# Full E2E test suite (builds CLI and tests everything)
./scripts/tests/test-plugin-e2e.sh

# With custom API endpoint
./scripts/tests/test-plugin-e2e.sh --endpoint http://localhost:5194

# Skip CLI build (if already built)
./scripts/tests/test-plugin-e2e.sh --skip-build
```

### 3. Clean Up

```bash
# Stop specific environment
./scripts/tests/docker-compose-down.sh shared

# Stop all environments
./scripts/tests/docker-compose-down.sh --all

# Stop and remove volumes (destructive)
./scripts/tests/docker-compose-down.sh shared --volumes
```

## 🛠 Script Details

### Docker Environment Management

#### `docker-compose-up.sh`
Starts Docker Compose environments with intelligent environment detection and health checks.

**Features:**
- Interactive environment selection
- Health checks for API services
- Automatic image pulling
- Container conflict resolution
- Comprehensive logging

**Environments:**
- **`shared`** - Uses shared API from kilometers-api repo (recommended for E2E testing)
- **`dev`** - Standalone development environment with local database
- **`test`** - Isolated testing environment with mock services

**Usage:**
```bash
./docker-compose-up.sh [environment]
./docker-compose-up.sh shared              # Start shared environment
./docker-compose-up.sh                     # Interactive selection
```

#### `docker-compose-down.sh`
Stops Docker Compose environments with cleanup options.

**Features:**
- Environment-specific container management
- Volume removal options (destructive)
- Orphaned container cleanup
- Force shutdown options

**Usage:**
```bash
./docker-compose-down.sh [environment] [options]
./docker-compose-down.sh shared            # Stop shared environment
./docker-compose-down.sh shared --volumes  # Stop and remove volumes
./docker-compose-down.sh --all             # Stop all environments
```

### End-to-End Testing

#### `test-plugin-e2e.sh`
Comprehensive end-to-end testing of the plugin manifest and download system.

**Test Coverage:**
1. **API Health Checks** - Verify API endpoint connectivity
2. **Customer Registration** - Register test customer and obtain API key
3. **JWT Token Exchange** - Test API key to JWT conversion
4. **CLI Authentication** - Verify CLI can authenticate with API
5. **Plugin Manifest** - Test manifest retrieval and parsing
6. **Plugin Installation** - Test plugin download and installation
7. **Plugin Removal** - Test plugin uninstallation
8. **Error Handling** - Test invalid credentials and non-existent plugins

**Prerequisites:**
- Docker environment must be running
- `curl` and `jq` must be installed
- Go compiler for building CLI binary

**Usage:**
```bash
./test-plugin-e2e.sh [options]

Options:
  -e, --endpoint URL    API endpoint (default: http://localhost:5194)
  -v, --verbose         Enable verbose output
  --skip-build         Skip building CLI binary
  --cleanup           Clean up test data on exit
  -h, --help           Show help
```

**Example Workflow:**
```bash
# 1. Start environment
./docker-compose-up.sh shared

# 2. Run full E2E test suite
./test-plugin-e2e.sh

# 3. Clean up
./docker-compose-down.sh shared
```

### Core Testing Scripts

#### `run-tests.sh`
Primary test runner for Go unit and integration tests.

```bash
./run-tests.sh                    # Run all tests
./run-tests.sh --coverage         # Run with coverage report
```

#### `test-and-check.sh`
Build verification and quick sanity checks.

```bash
./test-and-check.sh               # Build and run basic checks
```

#### `test-mcp-monitoring.sh`
Tests MCP server monitoring functionality.

```bash
./test-mcp-monitoring.sh          # Test MCP monitoring features
```

#### `test-plugin-integration.sh`
Integration tests for plugin system components.

```bash
./test-plugin-integration.sh     # Test plugin authentication pipeline
```

#### `test-plugin-provisioning.sh`
Tests plugin provisioning and installation workflows.

```bash
./test-plugin-provisioning.sh    # Test plugin provisioning system
```

## 🔧 Environment Configuration

### Docker Compose Files

The scripts automatically detect and use the appropriate Docker Compose files:

- **`docker-compose.shared.yml`** - Shared environment configuration
- **`docker-compose.dev.yml`** - Development environment configuration  
- **`docker-compose.test.yml`** - Test environment configuration

### Environment Variables

Key environment variables used by the test scripts:

```bash
# API Configuration
KM_API_ENDPOINT="http://localhost:5194"    # API endpoint URL
KM_API_KEY="your-api-key-here"             # API authentication key
KM_DEBUG="true"                            # Enable debug logging

# Plugin Configuration
KM_PLUGINS_DIR="~/.km/plugins"             # Plugin installation directory
KM_PLUGIN_PUBLIC_KEY="base64-key"          # Plugin verification public key
```

## 📊 Test Reporting

### Log Files

All test scripts generate detailed log files in the `logs/` directory:

```
logs/
├── docker-compose-up-YYYYMMDD-HHMMSS.log
├── docker-compose-down-YYYYMMDD-HHMMSS.log
├── plugin-e2e-test-YYYYMMDD-HHMMSS.log
└── test-report-YYYYMMDD-HHMMSS.txt
```

### Test Reports

The E2E test script generates comprehensive test reports:

```
================================================================
              E2E PLUGIN TEST REPORT
================================================================
Timestamp: 2024-08-18 10:30:45
API Endpoint: http://localhost:5194
Test Email: test-1692349845@example.com
Log File: /path/to/logs/plugin-e2e-test-20240818-103045.log

RESULTS:
  Total Tests: 15
  Passed: 15
  Failed: 0
  Success Rate: 100%

🎉 ALL TESTS PASSED!
================================================================
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Docker Environment Not Running
```bash
Error: Cannot connect to API endpoint: http://localhost:5194
```
**Solution:** Start the Docker environment first:
```bash
./docker-compose-up.sh shared
```

#### 2. Missing Dependencies
```bash
Error: jq is required but not installed
```
**Solution:** Install required tools:
```bash
# macOS
brew install jq curl

# Ubuntu/Debian
sudo apt-get install jq curl

# Alpine Linux
apk add jq curl
```

#### 3. Port Conflicts
```bash
Error: Port 5194 is already in use
```
**Solution:** Stop conflicting services or use different ports:
```bash
./docker-compose-down.sh --all
```

#### 4. Permission Denied
```bash
Permission denied: ./scripts/tests/test-plugin-e2e.sh
```
**Solution:** Make scripts executable:
```bash
chmod +x scripts/tests/*.sh
```

### Debugging Tips

1. **Enable Verbose Logging:**
   ```bash
   ./test-plugin-e2e.sh --verbose
   ```

2. **Check Docker Logs:**
   ```bash
   docker compose -f docker-compose.shared.yml logs -f api
   ```

3. **Verify API Health:**
   ```bash
   curl http://localhost:5194/health
   ```

4. **Check Environment Variables:**
   ```bash
   ./km auth status
   ```

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Plugin Tests
on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Install Dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq curl
      
      - name: Start Test Environment
        run: ./scripts/tests/docker-compose-up.sh test
      
      - name: Run E2E Tests
        run: ./scripts/tests/test-plugin-e2e.sh --verbose
      
      - name: Cleanup
        if: always()
        run: ./scripts/tests/docker-compose-down.sh --all --force
```

### Local Development Workflow

```bash
# Daily development workflow
./scripts/tests/docker-compose-up.sh shared
./scripts/tests/test-plugin-e2e.sh --skip-build
./scripts/tests/docker-compose-down.sh shared

# Full test suite before PR
./scripts/tests/docker-compose-up.sh test
./scripts/tests/run-tests.sh --coverage
./scripts/tests/test-plugin-e2e.sh
./scripts/tests/docker-compose-down.sh test --volumes
```

## 📚 Additional Resources

- [Docker Development Guide](../../docs/development/DOCKER_DEVELOPMENT.md)
- [Plugin System Documentation](../../docs/plugins/DEVELOPMENT.md)
- [Authentication Guide](../../docs/development/AUTH_REFRESH.md)
- [Build and Test Guide](../../docs/development/BUILD_RUN_TEST.md)

---

For questions or issues with the test infrastructure, please check the logs and refer to the troubleshooting section above.