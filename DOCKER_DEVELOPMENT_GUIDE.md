# Docker Development Guide

Quick reference for Docker-based development workflows with kilometers-cli and kilometers-api.

## Repository Setup

Ensure both repositories are cloned as siblings:
```
your-workspace/
├── kilometers-api/     # Contains docker-compose.shared.yml
└── kilometers-cli/     # This repository
```

## Development Options

### Option 1: Standalone CLI Development
For CLI-only development without API integration:
```bash
# Build and test CLI locally
go build -o km ./cmd/main.go
./scripts/test/run-tests.sh

# Use with external API
export KM_API_ENDPOINT="https://api.kilometers.ai"
./km auth login --api-key "your-production-key"
```

### Option 2: Shared Environment (Recommended)
For full-stack development with local API:

#### Quick Start
```bash
# 1. Start shared environment
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml up -d

# 2. Configure CLI
cd ../kilometers-cli
export KM_API_ENDPOINT="http://localhost:5194"

# 3. Test integration
./km auth status
curl http://localhost:5194/health
```

#### Development Workflow
```bash
# Start development session
cd ../kilometers-api && docker-compose -f docker-compose.shared.yml up -d
cd ../kilometers-cli

# Develop CLI features
go build -o km ./cmd/main.go
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp

# Test integration
./scripts/test/run-tests.sh
go test ./test/integration/ -v

# Stop when done
cd ../kilometers-api && docker-compose -f docker-compose.shared.yml down
```

## Shared Environment Services

| Service | Port | URL | Purpose |
|---------|------|-----|---------|
| kilometers-api | 5194 | http://localhost:5194 | API endpoints |
| postgres | 5432 | localhost:5432 | Database |
| pgadmin | 5050 | http://localhost:5050 | DB management |
| swagger | 5194 | http://localhost:5194/swagger | API docs |

## Common Commands

### Environment Management
```bash
# Start all services
docker-compose -f docker-compose.shared.yml up -d

# Start with database admin
docker-compose -f docker-compose.shared.yml --profile tools up -d

# View logs
docker-compose -f docker-compose.shared.yml logs -f api

# Stop services
docker-compose -f docker-compose.shared.yml down

# Clean reset
docker-compose -f docker-compose.shared.yml down -v
```

### CLI Development
```bash
# Build CLI
go build -o km ./cmd/main.go

# Test configuration
./km auth status

# Test monitoring
./km monitor -- npx -y @modelcontextprotocol/server-filesystem /tmp

# Run tests
./scripts/test/run-tests.sh
go test ./test/integration/ -v
```

### API Development (when needed)
```bash
# Rebuild API after changes
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml up --build api -d

# View API logs
docker-compose -f docker-compose.shared.yml logs -f api

# Test API directly
curl http://localhost:5194/health
curl http://localhost:5194/swagger
```

## Configuration

### Environment Variables
```bash
# API endpoint
export KM_API_ENDPOINT="http://localhost:5194"

# Plugin configuration
export KM_PLUGINS_DIR="$HOME/.km/plugins"
export KM_DEBUG=true
export KM_LOG_LEVEL=debug

# Authentication
export KM_API_KEY="km_test_your_key"
```

### Config Files
Create `.env` in CLI directory:
```env
KM_API_ENDPOINT=http://localhost:5194
KM_DEBUG=true
KM_LOG_LEVEL=debug
KM_PLUGINS_DIR=/Users/yourname/.km/plugins
```

## Troubleshooting

### Common Issues

**API not responding:**
```bash
# Check if running
docker-compose -f docker-compose.shared.yml ps

# Check logs
docker-compose -f docker-compose.shared.yml logs api

# Test health
curl http://localhost:5194/health
```

**Database issues:**
```bash
# Check database
docker-compose -f docker-compose.shared.yml logs postgres

# Connect to database
docker exec -it kilometers-shared-postgres psql -U postgres -d kilometers_dev
```

**CLI connection issues:**
```bash
# Verify configuration
./km auth status

# Check API endpoint
curl $KM_API_ENDPOINT/health

# Enable debug logging
export KM_DEBUG=true && ./km auth status
```

### Clean Reset
```bash
# Complete environment reset
cd ../kilometers-api
docker-compose -f docker-compose.shared.yml down -v
docker system prune -f
docker-compose -f docker-compose.shared.yml up -d

# CLI reset
cd ../kilometers-cli
go clean && rm -f km
rm -rf ~/.config/kilometers/ ~/.km/
```

## When to Use Each Option

### Use Standalone Development When:
- ✅ Developing CLI-only features
- ✅ Working with production API
- ✅ Quick CLI testing/debugging
- ✅ No database changes needed

### Use Shared Environment When:
- ✅ Testing CLI ↔ API integration
- ✅ Plugin authentication development
- ✅ Database schema changes
- ✅ Full-stack feature development
- ✅ Reproducing production issues locally
- ✅ Integration testing

## Further Reading

- [Shared API Development Guide](docs/development/SHARED_API_DEVELOPMENT.md) - Detailed CLI development workflows
- [kilometers-api Docker Guide](../kilometers-api/docs/DOCKER_SHARED_DEVELOPMENT.md) - API-side documentation
- [Plugin Development](docs/plugins/DEVELOPMENT.md) - Plugin system development
- [Testing Guide](docs/development/BUILD_RUN_TEST.md) - Testing strategies and commands