# Unified Configuration System

The Kilometers CLI uses a unified configuration system that loads settings from multiple sources with clear precedence, providing transparency and consistency across all commands.

## Overview

The unified configuration system provides:

1. **Clear Precedence** - Well-defined loading order from multiple sources
2. **Source Transparency** - See exactly where each config value came from
3. **Consistent API** - All commands use the same configuration loading
4. **Environment Standardization** - Standardized `KM_*` environment variables
5. **Validation** - Built-in validation for all configuration values

## Configuration Sources

Configuration is loaded from multiple sources in this order (highest to lowest priority):

1. **CLI Flags** - `--api-key`, `--debug`, etc. (Priority 1)
2. **Environment Variables** - `KM_*` prefixed variables (Priority 2)  
3. **Saved Configuration** - `~/.config/kilometers/config.json` (Priority 3)
4. **Project .env Files** - `.env` in current directory (Priority 4)
5. **Defaults** - Built-in application defaults (Priority 99)

## Environment Variables

### Standard Variables
All environment variables use the `KM_*` prefix:

```bash
# Core configuration
export KM_API_KEY="km_pro_your_api_key"           # API key for premium features
export KM_API_ENDPOINT="https://api.kilometers.ai" # API endpoint URL
export KM_DEBUG="true"                            # Enable debug logging
export KM_LOG_LEVEL="debug"                       # Set log level

# Advanced configuration  
export KM_BUFFER_SIZE="2097152"                   # Buffer size in bytes
export KM_BATCH_SIZE="20"                         # Batch size for API requests
export KM_PLUGINS_DIR="/path/to/plugins"          # Custom plugins directory
export KM_AUTO_PROVISION="true"                   # Auto-provision plugins
export KM_TIMEOUT="60s"                           # Default timeout duration
```

### Legacy Support Removed
Previous `KILOMETERS_*` environment variables are no longer supported. Use `KM_*` instead.

## Configuration Files

### Saved Configuration
- **Location**: `~/.config/kilometers/config.json`
- **Managed by**: `km auth` commands (`login`, `logout`, `status`)
- **Format**: JSON with version 2.0 schema
- **Priority**: 3

Example:
```json
{
  "api_key": "km_pro_your_api_key",
  "api_endpoint": "https://api.kilometers.ai",
  "buffer_size": 1048576,
  "batch_size": 10,
  "log_level": "info",
  "debug": false,
  "default_timeout": 30000000000,
  "saved_at": "2025-01-15T10:30:00Z",
  "version": "2.0"
}
```

### Project .env Files
- **Location**: `.env` in current working directory
- **Format**: Key-value pairs with `KM_*` variables
- **Priority**: 4

Example:
```bash
# Project-specific configuration
KM_API_KEY=km_dev_project_key
KM_LOG_LEVEL=debug
KM_BUFFER_SIZE=2097152
```

## Usage

### Managing Configuration

```bash
# Set API key (saves to ~/.config/kilometers/config.json)
km auth login --api-key "km_pro_your_api_key"

# Check current configuration and sources
km auth status

# Clear API key
km auth logout
```

### Environment Variable Override

```bash
# Override saved config with environment variable
export KM_API_KEY="km_dev_override_key"
km auth status

# Use different API endpoint
export KM_API_ENDPOINT="http://localhost:5194"
km monitor -- your-mcp-server
```

### Project-specific Configuration

```bash
# Create project .env file
echo "KM_API_KEY=km_dev_project_key" > .env
echo "KM_LOG_LEVEL=debug" >> .env

# Commands will use project configuration
km auth status
```

## Configuration Transparency

Use `km auth status` to see exactly where each configuration value is loaded from:

```bash
$ km auth status
🔑 API Key: km_pro...key
🌐 API Endpoint: https://api.kilometres.ai
📍 API Key Source: env (KM_API_KEY)
📍 API Endpoint Source: filesystem (/path/to/.env:KM_API_ENDPOINT)
```

Source types:
- `cli` - Command line flag
- `env` - Environment variable  
- `file` - Configuration file (saved config or .env)
- `default` - Built-in default value

## Implementation Architecture

The unified configuration system follows clean architecture principles:

### Core Domain
- **`UnifiedConfig`** - Main configuration model with source tracking
- **`ConfigSource`** - Metadata about where each value came from

### Application Layer  
- **`ConfigService`** - Main configuration orchestration service
- **`LoadConfig()`** - Loads configuration from all sources
- **`UpdateAPIKey()`** - Updates and persists API key
- **`GetConfigStatus()`** - Returns configuration with source information

### Infrastructure Layer
- **`UnifiedLoader`** - Implements configuration loading from multiple sources
- **`UnifiedStorage`** - Handles configuration persistence
- **`SimpleFileSystemScanner`** - Scans .env files and saved configuration

### Usage in Components
All CLI commands use the same configuration loading pattern:

```go
// Load configuration
configService, err := config.CreateConfigServiceFromDefaults()
if err != nil {
    return fmt.Errorf("failed to create config service: %w", err)
}

unifiedConfig, err := configService.Load(ctx)
if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
}

// Use configuration
if unifiedConfig.HasAPIKey() {
    // Use API key
    apiKey := unifiedConfig.APIKey
}
```

## Validation

All configuration values are validated:

- **API Keys** - Must not be empty when required
- **API Endpoints** - Must be valid URLs if specified
- **Buffer Size** - Must be greater than 0
- **Batch Size** - Must be greater than 0
- **Log Level** - Must be valid level (debug, info, warn, error, fatal)
- **Timeout** - Must be non-negative duration

## Migration from Legacy Systems

The unified system automatically handles:

1. **Old config file format** - Legacy JSON configs are ignored (use `km auth login` to set new config)
2. **Environment variables** - Only `KM_*` variables are supported
3. **Configuration discovery** - Removed complex discovery system in favor of simple, predictable loading

## Security Considerations

1. **API Key Protection**
   - API keys are masked in output (first 6 + last 4 characters)
   - Configuration files have restricted permissions (0600)
   - Source information helps identify potential leaks

2. **File Permissions**
   - Saved configuration files are created with user-only access
   - Environment variables are only read from current process

3. **Source Transparency**
   - Always shows where configuration values came from
   - Helps identify configuration sources in debugging

## Troubleshooting

### Configuration Not Found
```bash
$ km auth status
❌ No API key configured
   Run 'km auth login --api-key YOUR_KEY' to configure
```

**Solution**: Set API key using `km auth login --api-key "your_key"`

### Wrong Configuration Used
Use `km auth status` to see configuration sources:
```bash
$ km auth status
📍 API Key Source: filesystem (/project/.env:KM_API_KEY)
```

**Solution**: Check the source priority and adjust as needed (remove .env file, unset environment variable, etc.)

### Environment Variable Not Loaded
Ensure variable is exported:
```bash
export KM_API_KEY="your_key"  # Not just KM_API_KEY="your_key"
```

### .env File Not Found
Check file location and variable names:
```bash
ls -la .env                    # File exists?
grep KM_ .env                  # Using KM_ prefix?
```

## Example Workflows

### Development Setup
```bash
# Set development API key
echo "KM_API_KEY=km_dev_local_key" > .env
echo "KM_LOG_LEVEL=debug" >> .env

# Verify configuration
km auth status
```

### Production Deployment
```bash
# Set production API key via environment
export KM_API_KEY="km_prod_secure_key"
export KM_API_ENDPOINT="https://api.kilometers.ai"

# Deploy application
./km monitor -- production-mcp-server
```

### CI/CD Pipeline
```bash
# Set API key in CI environment
export KM_API_KEY="$CI_KILOMETERS_API_KEY"

# Run tests with configuration
./scripts/test/run-tests.sh
```