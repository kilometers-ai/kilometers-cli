# Automatic Configuration Discovery

The Kilometers CLI includes powerful automatic configuration discovery that can detect settings from multiple sources, eliminating manual configuration steps and providing a zero-config experience for most users.

## Overview

The `km init --auto-detect` command automatically discovers configuration from:

1. **Environment Variables** - Both current (`KILOMETERS_*`) and legacy (`KM_*`) prefixes
2. **Configuration Files** - YAML, JSON, and .env files in standard locations
3. **API Endpoints** - Docker Compose files, running containers, and well-known URLs
4. **Credentials** - Secure credential stores and encrypted caches
5. **Legacy Configs** - Automatic migration from old configuration formats

## Usage

### Basic Auto-Detection

```bash
km init --auto-detect
```

This command will:
1. Scan all available sources for configuration
2. Display discovered values with their sources
3. Ask for confirmation before saving
4. Validate all configuration values

### Auto-Detection with Plugin Provisioning

```bash
km init --auto-detect --auto-provision-plugins
```

Combines configuration discovery with automatic plugin installation based on your subscription tier.

### Override Specific Values

```bash
km init --auto-detect --api-key YOUR_KEY --endpoint https://custom.api.com
```

Auto-detected values can be overridden with command-line flags.

## Discovery Sources

### 1. Environment Variables

The discovery service checks for configuration in environment variables with the following precedence:

#### Standard Prefixes
- `KILOMETERS_*` - Current standard prefix (highest priority)
- `KM_*` - Legacy prefix for backward compatibility

#### Supported Variables
```bash
KILOMETERS_API_KEY       # API authentication key
KILOMETERS_API_ENDPOINT  # API server URL
KILOMETERS_BUFFER_SIZE   # Buffer size for monitoring (e.g., "2MB")
KILOMETERS_LOG_LEVEL     # Logging level (debug, info, warn, error)
KILOMETERS_PLUGINS_DIR   # Custom plugins directory
KILOMETERS_AUTO_PROVISION # Auto-provision plugins (true/false)
KILOMETERS_TIMEOUT       # Default timeout (e.g., "30s")
```

#### Special Cases
In CI/CD environments (detected by `CI`, `GITHUB_ACTIONS`, or `JENKINS_HOME` variables), the service also checks for non-prefixed variables like `API_KEY` and `API_ENDPOINT`.

### 2. Configuration Files

The service searches for configuration files in the following locations (in order):

1. **Project Directory**
   - `./config.yaml`, `./config.yml`, `./config.json`
   - `./km.config.yaml`, `./km.config.yml`, `./km.config.json`
   - `./.kmrc`, `./.kilometersrc`

2. **User Home Directory**
   - `~/.km/config.*`
   - `~/.kilometers/config.*`
   - `~/.config/km/config.*`
   - `~/.config/kilometers/config.*`

3. **System Directory**
   - `/etc/kilometers/config.*`

#### Supported Formats

**YAML Example:**
```yaml
api_key: your-api-key-here
api_endpoint: https://api.kilometers.ai
buffer_size: 4MB
log_level: debug
plugins_dir: ~/.km/plugins
auto_provision: true
default_timeout: 45s
```

**JSON Example:**
```json
{
  "api_key": "your-api-key-here",
  "api_endpoint": "https://api.kilometers.ai",
  "buffer_size": 4194304,
  "log_level": "debug",
  "plugins_dir": "~/.km/plugins",
  "auto_provision": true,
  "default_timeout": "45s"
}
```

### 3. Docker Compose Files

The service scans for API endpoints in Docker Compose files:

```yaml
services:
  kilometers-api:
    image: kilometers/api:latest
    ports:
      - "5194:5194"
    environment:
      - API_ENDPOINT=http://localhost:5194
```

Checked files:
- `docker-compose.yml`, `docker-compose.yaml`
- `compose.yml`, `compose.yaml`
- `docker-compose.dev.yml`, `docker-compose.local.yml`

### 4. Environment Files

The service checks `.env` files for configuration:

```bash
# .env file
KILOMETERS_API_KEY=your-api-key-here
KILOMETERS_API_ENDPOINT=https://api.kilometers.ai
```

Locations checked:
- `./.env`, `./.env.local`, `./.env.development`
- `../.env` (parent directory)
- `../../.env` (grandparent directory)

### 5. Credential Stores

The service securely locates API keys from:

1. **Encrypted Cache** - `~/.km/.credentials.enc`
2. **Credential Files** - `~/.km/credentials`, `~/.kilometers/credentials`
3. **OS Keychains** - macOS Keychain, Windows Credential Manager, Linux Secret Service (planned)

## Priority and Precedence

Configuration values are discovered with the following priority (highest to lowest):

1. **Command Line Flags** - Explicitly provided values
2. **Environment Variables** - Currently set in shell
3. **Project Config** - Files in current directory
4. **User Config** - Files in home directory
5. **System Config** - Files in /etc
6. **Auto-Discovered** - API endpoints, credentials from special locations

## Validation

All discovered values are validated before use:

- **API Endpoints** - Must be valid HTTP/HTTPS URLs
- **API Keys** - Minimum 20 characters, no whitespace or placeholders
- **Buffer Sizes** - Between 1KB and 100MB
- **Log Levels** - Must be valid (debug, info, warn, error, etc.)
- **Timeouts** - Between 1 second and 5 minutes

## Migration from Legacy Formats

The service automatically migrates old configuration formats:

| Old Field | New Field | Notes |
|-----------|-----------|-------|
| `apiKey` | `api_key` | camelCase to snake_case |
| `apiEndpoint` | `api_endpoint` | |
| `api_url` | `api_endpoint` | Alternative naming |
| `debug: true` | `log_level: debug` | Boolean to string |

## Security Considerations

1. **Credential Protection**
   - API keys are masked in output (shows only first/last 4 characters)
   - Credentials stored with AES-256-GCM encryption
   - Machine-specific encryption keys
   - File permissions restricted to user only (0600)

2. **Validation**
   - Placeholder values are detected and rejected
   - API endpoints are validated before use
   - Configuration sources are shown for transparency

## Troubleshooting

### No Configuration Found

If auto-detection finds no configuration:
1. Check environment variables are exported: `export KILOMETERS_API_KEY=...`
2. Verify file permissions on config files
3. Ensure config files use correct field names (snake_case)

### Validation Warnings

If you see validation warnings:
1. Check API key length (minimum 20 characters)
2. Verify API endpoint includes protocol (http:// or https://)
3. Ensure buffer sizes use valid units (B, KB, MB, GB)

### Multiple Configurations

If multiple configurations are found:
1. The tool shows all discovered values with their sources
2. Higher priority sources override lower priority ones
3. Use `--force` to overwrite existing configurations

## Example Output

```bash
$ km init --auto-detect
🔍 Scanning for configuration...
  • Scanning environment...
  • Scanning filesystem...
  • Discovering API endpoints...
  • Looking for API credentials...

✅ Configuration discovery complete! Found 4 sources.

Discovered configuration:
------------------------
• API Endpoint     : https://api.kilometers.ai
                     Source: environment (KILOMETERS_API_ENDPOINT)
• API Key          : km_1****5678
                     Source: environment (KILOMETERS_API_KEY)
• Buffer Size      : 2097152
                     Source: filesystem (~/.km/config.yaml)
• Log Level        : debug
                     Source: environment (KILOMETERS_LOG_LEVEL)
• Plugins Dir      : ~/.km/plugins
                     Source: filesystem (~/.km/config.yaml)

Use this configuration? [Y/n]
```

## Implementation Details

The configuration discovery system uses a modular architecture:

- **ConfigDiscoveryService** - Orchestrates the discovery process
- **EnvironmentScanner** - Scans environment variables
- **FileSystemScanner** - Searches for configuration files
- **APIEndpointDiscoverer** - Finds API endpoints from various sources
- **CredentialLocator** - Securely locates credentials
- **ConfigValidator** - Validates all configuration values

Each component can be used independently or as part of the full discovery process.

