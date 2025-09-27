# Install Script Testing Infrastructure

This directory contains a comprehensive Docker-based testing infrastructure for validating the Kilometers CLI install scripts across multiple platforms and scenarios.

## Overview

The testing infrastructure consists of:

- **Docker containers** for different Linux distributions
- **Mock GitHub API server** for simulating releases
- **Test harnesses** for automated validation
- **CI/CD integration** with GitHub Actions

## Quick Start

### Prerequisites

- Docker and Docker Compose installed
- Make (optional, for convenience commands)

### Run All Tests

```bash
# Using the test harness script
cd scripts/test
./test-install-docker.sh

# Using Make (recommended)
make test

# Using Docker Compose
make test-compose
```

### Run Tests for Specific Platform

```bash
# Test Ubuntu only
./test-install-docker.sh --platform ubuntu-amd64
make test-platform PLATFORM=ubuntu

# Test Alpine Linux
./test-install-docker.sh --platform alpine
make test-platform PLATFORM=alpine
```

### Test Different Scenarios

```bash
# Test timeout scenarios
./test-install-docker.sh --mode timeout
make test MODE=timeout

# Test server errors
./test-install-docker.sh --mode server_error
make test MODE=server_error

# Test with corrupted binaries
./test-install-docker.sh --mode corrupted_binary
make test MODE=corrupted_binary
```

## Directory Structure

```
scripts/test/
├── README.md                    # This file
├── Makefile                     # Convenience commands
├── docker-compose.yml           # Docker Compose configuration
├── test-install-docker.sh       # Main test harness script
├── docker/                      # Docker configurations
│   ├── Dockerfile.ubuntu-amd64  # Ubuntu 22.04 test environment
│   ├── Dockerfile.alpine        # Alpine Linux test environment
│   ├── Dockerfile.debian        # Debian stable test environment
│   ├── Dockerfile.fedora        # Fedora test environment
│   └── test-runner.sh           # Test runner for containers
├── mock-server/                 # Mock GitHub API server
│   ├── Dockerfile               # Mock server container
│   └── server.py                # Python mock server
├── data/                        # Test data and binaries
│   ├── create-test-binaries.sh  # Script to create test binaries
│   └── *.tar.gz                 # Test binary archives
└── results/                     # Test results and logs
```

## Testing Platforms

The infrastructure supports testing on:

- **Ubuntu 22.04** (x86_64) - Most common Linux distribution
- **Alpine Linux** - Minimal distribution, different package manager
- **Debian Stable** - Debian-based systems
- **Fedora Latest** - Red Hat-based systems

Each platform tests:
- Different shells (bash, zsh, fish where available)
- Different user privilege levels
- Different install directories
- PATH configuration

## Test Scenarios

### Normal Operation
- Fresh installation
- Upgrade installation
- Reinstallation over existing binary
- Different user permissions
- Shell compatibility

### Error Conditions
- Missing dependencies (curl/wget)
- Network timeouts
- GitHub API rate limiting
- Server errors (5xx responses)
- Malformed JSON responses
- Corrupted binary downloads
- Missing binary files
- Insufficient disk space
- Permission denied scenarios

### Platform-Specific Tests
- Architecture detection (x86_64, ARM64)
- Operating system detection
- Package manager availability
- Shell environment differences

## Mock Server

The mock GitHub API server simulates various GitHub API responses:

### Endpoints
- `GET /repos/kilometers-ai/kilometers-cli/releases/latest` - Latest release info
- `GET /releases/download/{version}/{filename}` - Binary downloads

### Test Modes
- **normal** - Standard successful responses
- **timeout** - Simulates network timeouts
- **rate_limit** - Returns 403 rate limit errors
- **server_error** - Returns 500 server errors
- **malformed_json** - Returns invalid JSON
- **corrupted_binary** - Returns corrupted binary data
- **missing_binary** - Returns 404 for binary downloads

### Usage
```bash
# Start mock server with specific mode
cd scripts/test
TEST_MODE=timeout docker-compose up -d mock-server

# Or start directly
cd scripts/test/mock-server
python server.py --mode rate_limit --port 8080
```

## Test Data

Test binaries are created automatically and include:
- Minimal functional binaries that respond to `--version` and `--help`
- Different naming conventions for compatibility
- Corrupted archives for error testing
- Platform-specific binaries for different architectures

## GitHub Actions Integration

The testing infrastructure integrates with GitHub Actions:

### Triggers
- Changes to install scripts
- Changes to test infrastructure
- Pull requests affecting install functionality
- Manual workflow dispatch

### Matrix Testing
- Multiple platforms × test modes
- Reduced matrix for PR builds
- Full matrix for main branch builds

### Results
- Test logs uploaded as artifacts
- Summary posted to PR comments
- Step summary with pass/fail status

## Local Development

### Setup Development Environment
```bash
make dev
```
This starts the mock server and prepares the environment for testing.

### Run Individual Tests
```bash
# Test specific platform with verbose output
./test-install-docker.sh --platform ubuntu --verbose

# Test with custom Docker Compose
TEST_MODE=normal docker-compose up test-ubuntu
```

### View Results
```bash
make logs
```

### Clean Up
```bash
make clean      # Clean containers and results
make clean-all  # Clean everything including images
```

## Adding New Platforms

To add a new platform:

1. Create `Dockerfile.{platform}` in `docker/` directory
2. Add platform to `PLATFORMS` array in `test-install-docker.sh`
3. Add service to `docker-compose.yml`
4. Update GitHub Actions matrix in `.github/workflows/test-install-scripts.yml`

### Example Dockerfile Template
```dockerfile
FROM {base-image}

# Install dependencies
RUN {package-manager} install curl wget tar gzip bash

# Create test users
RUN useradd -m -s /bin/bash testuser

# Copy test scripts
COPY ../install.sh /test/install-local.sh
COPY ../../install.sh /test/install-repo.sh
COPY test-runner.sh /test/test-runner.sh

RUN chmod +x /test/*.sh
WORKDIR /test
CMD ["./test-runner.sh"]
```

## Troubleshooting

### Common Issues

**Mock server not responding**
```bash
# Check if container is running
docker ps | grep mock-server

# Check logs
docker logs km-mock-server

# Restart server
make stop-server && make start-server
```

**Test containers failing to build**
```bash
# Clean Docker cache
docker system prune -f

# Rebuild images
make build-images
```

**Permission errors**
```bash
# Ensure scripts are executable
chmod +x scripts/test/*.sh
chmod +x scripts/test/docker/*.sh
chmod +x scripts/test/data/*.sh
```

### Debugging Tests

**Verbose test output**
```bash
./test-install-docker.sh --verbose
```

**Run tests interactively**
```bash
# Start mock server
make start-server

# Run specific test container interactively
docker run -it --rm --network km-test-network \
  -e MOCK_SERVER_HOST=mock-server \
  km-test-ubuntu bash
```

**Check test logs**
```bash
# View all logs
make logs

# View specific log
cat scripts/test/results/test-ubuntu-normal.log
```

## Performance Considerations

- Tests run in parallel where possible
- Docker layer caching reduces build times
- Mock server eliminates external dependencies
- Results are cached between test runs

## Security

- No external network access required for testing
- Mock server only serves test data
- Test binaries are minimal and safe
- No real credentials or API keys used

## Contributing

When modifying the test infrastructure:

1. Test changes locally with `make test`
2. Ensure all platforms pass tests
3. Update documentation if adding new features
4. Verify CI/CD integration works

## Future Enhancements

Potential improvements:
- Windows testing with Wine containers
- Performance benchmarking
- Integration testing with real MCP servers
- Homebrew formula testing
- Package manager integration testing
