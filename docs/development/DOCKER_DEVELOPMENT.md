# Docker Development Environment

This guide explains how to use the Docker Compose development environment to run a complete integration environment with the Kilometers API and PostgreSQL database.

## Prerequisites

- Docker and Docker Compose installed
- Both `kilometers-cli` and `kilometers-api` repositories cloned in the same parent directory:
  ```
  your-workspace/
  ├── kilometers-cli/     (this repository)
  └── kilometers-api/     (API repository)
  ```

## Quick Start

### Option 1: Using the Helper Script (Recommended)

```bash
# Start development environment with automated setup
./scripts/dev-env.sh start

# Start with pgAdmin included
./scripts/dev-env.sh start --with-tools

# Check status
./scripts/dev-env.sh status

# Test CLI integration
export KM_API_KEY=your-api-key-here
./scripts/dev-env.sh test-cli

# Stop environment
./scripts/dev-env.sh stop
```

### Option 2: Manual Docker Compose

1. **Start the development environment:**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

2. **Wait for services to be ready** (about 30-60 seconds for API to fully start)

3. **Test the API is running:**
   ```bash
   curl http://localhost:5000/health
   ```

4. **Test CLI integration:**
   ```bash
   # Set environment variables for CLI
   export KM_API_URL=http://localhost:5000
   export KM_API_KEY=your-api-key-here
   
   # Test monitoring
   ./km monitor --server -- echo '{"jsonrpc":"2.0","method":"test"}'
   ```

## Services

### PostgreSQL Database
- **Host:** `localhost:5432`
- **Database:** `kilometers_dev`
- **Username:** `postgres`
- **Password:** `postgres`
- **Extensions:** uuid-ossp, pgcrypto (auto-installed)

### Kilometers API
- **URL:** `http://localhost:5000`
- **Environment:** Development
- **Health Check:** `http://localhost:5000/health`
- **Built from:** `../kilometers-api` (relative path)

### pgAdmin (Optional)
- **URL:** `http://localhost:5050`
- **Email:** `dev@kilometers.ai`
- **Password:** `devpassword`
- **Enable with:** `docker-compose -f docker-compose.dev.yml --profile tools up -d`

## Common Commands

### Start Services
```bash
# Start all services in background
docker-compose -f docker-compose.dev.yml up -d

# Start with pgAdmin for database management
docker-compose -f docker-compose.dev.yml --profile tools up -d

# Start with logs visible
docker-compose -f docker-compose.dev.yml up
```

### Monitor Services
```bash
# View all logs
docker-compose -f docker-compose.dev.yml logs -f

# View API logs only
docker-compose -f docker-compose.dev.yml logs -f api

# View database logs only
docker-compose -f docker-compose.dev.yml logs -f postgres
```

### Service Management
```bash
# Check service status
docker-compose -f docker-compose.dev.yml ps

# Restart API (e.g., after code changes)
docker-compose -f docker-compose.dev.yml restart api

# Rebuild API (after significant changes)
docker-compose -f docker-compose.dev.yml up -d --build api

# Stop all services
docker-compose -f docker-compose.dev.yml down

# Stop and remove volumes (clean slate)
docker-compose -f docker-compose.dev.yml down -v
```

## Development Workflow

### 1. API Development
When making changes to the API:

```bash
# Rebuild and restart API
docker-compose -f docker-compose.dev.yml up -d --build api

# Or restart without rebuild (for config changes)
docker-compose -f docker-compose.dev.yml restart api
```

### 2. CLI Testing
```bash
# Set environment for local testing
export KM_API_URL=http://localhost:5000
export KM_API_KEY=$(curl -X POST http://localhost:5000/auth/generate-key)

# Test various CLI commands
./km init
./km plugins list
./km monitor --server -- your-command-here
```

### 3. Database Access

**Via psql:**
```bash
# Connect directly to database
docker exec -it kilometers-dev-postgres psql -U postgres -d kilometers_dev

# Run SQL commands
psql -h localhost -U postgres -d kilometers_dev -c "SELECT * FROM your_table;"
```

**Via pgAdmin:**
1. Start with tools profile: `docker-compose -f docker-compose.dev.yml --profile tools up -d`
2. Open http://localhost:5050
3. Login with `dev@kilometers.ai` / `devpassword`
4. Add server: Host `postgres`, Port `5432`, User `postgres`, Password `postgres`

## Troubleshooting

### Services Won't Start
```bash
# Check if ports are in use
lsof -i :5000  # API port
lsof -i :5432  # PostgreSQL port
lsof -i :5050  # pgAdmin port

# Check service health
docker-compose -f docker-compose.dev.yml ps
docker-compose -f docker-compose.dev.yml logs api
```

### API Connection Issues
```bash
# Verify API is responding
curl -v http://localhost:5000/health

# Check API logs
docker-compose -f docker-compose.dev.yml logs -f api

# Restart API service
docker-compose -f docker-compose.dev.yml restart api
```

### Database Connection Issues
```bash
# Test database connectivity
docker exec kilometers-dev-postgres pg_isready -U postgres

# Check database logs
docker-compose -f docker-compose.dev.yml logs -f postgres

# Connect to database directly
docker exec -it kilometers-dev-postgres psql -U postgres -d kilometers_dev
```

### Clean Reset
```bash
# Stop everything and clean up
docker-compose -f docker-compose.dev.yml down -v
docker system prune -f

# Rebuild from scratch
docker-compose -f docker-compose.dev.yml up -d --build
```

## Environment Variables

You can customize the environment by creating a `.env` file in the CLI repository:

```bash
# .env file example
KM_API_URL=http://localhost:5000
KM_API_KEY=your-development-api-key
KM_DEBUG=true
KM_LOG_LEVEL=debug

# Database connection (for direct access)
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/kilometers_dev
```

## Integration with CLI Development

### Testing Plugins
```bash
# Start dev environment
docker-compose -f docker-compose.dev.yml up -d

# Test plugin provisioning
KM_API_KEY=your-key ./km plugins list

# Test plugin execution with monitoring
KM_API_KEY=your-key ./km monitor --server -- echo 'test'
```

### Authentication Testing
```bash
# Test authentication flow
curl -X POST http://localhost:5000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@kilometers.ai","password":"test"}'

# Use generated token with CLI
export KM_API_KEY=generated-token-here
./km monitor --server -- your-command
```

## Performance Considerations

- **Startup Time:** API takes 30-60 seconds to fully initialize
- **Resource Usage:** ~500MB RAM for full stack
- **Storage:** PostgreSQL data persists in Docker volume
- **Network:** Services communicate via Docker network (172.21.0.0/16)

## Security Notes

⚠️ **Development Only**: This configuration is for development purposes only and includes:
- Hardcoded passwords
- Permissive CORS settings
- Development JWT keys
- Debug logging enabled

Do not use this configuration in production environments.
