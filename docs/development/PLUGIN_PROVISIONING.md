# Plugin Provisioning Implementation

## Overview

This document describes the automatic plugin provisioning feature added to the `km init` command, enabling seamless plugin installation based on customer subscription tiers.

## Architecture

### Domain Layer
- **`plugin_provision.go`**: Core domain models for plugin provisioning
  - `PluginProvisionRequest`: Request for customer-specific plugins
  - `ProvisionedPlugin`: Plugin ready for download
  - `PluginRegistry`: Tracks installed plugins and their status
  - `InstalledPlugin`: Locally installed plugin information

### Application Layer
- **`plugin_provisioning_service.go`**: Orchestrates plugin provisioning
  - `PluginProvisioningManager`: Main service for plugin lifecycle
  - `AutoProvisionPlugins()`: Downloads and installs plugins
  - `RefreshPlugins()`: Handles tier changes and updates

### Infrastructure Layer
- **`provisioning.go`**: HTTP client for API communication
- **`downloader.go`**: Secure plugin download with signature verification
- **`installer.go`**: File system plugin installation
- **`registry.go`**: Plugin registry persistence

### Interface Layer
- **`init.go`**: Enhanced with `--auto-provision-plugins` flag

## Usage

### Basic Init with Auto-Provisioning
```bash
km init --api-key YOUR_API_KEY --auto-provision-plugins
```

### Manual Plugin Refresh
```bash
km plugins refresh  # Future command
```

## Features

### 1. Automatic Plugin Discovery
- Queries API for available plugins based on subscription tier
- Downloads customer-specific binaries
- Verifies digital signatures
- Installs to local plugin directory

### 2. Tier Management
- **Upgrade Handling**: Automatically downloads new plugins when tier increases
- **Downgrade Handling**: Disables premium plugins when tier decreases
- **Graceful Degradation**: Plugins fail silently if tier insufficient

### 3. Security
- Customer-specific plugin binaries
- RSA signature verification
- Secure download URLs with expiration
- Binary integrity validation

## Implementation Details

### Plugin Package Structure
```
km-plugin-NAME-HASH.kmpkg/
├── km-plugin-NAME-HASH      # Executable binary
├── km-plugin-NAME-HASH.sig  # Digital signature
└── km-plugin-NAME-HASH.manifest  # Metadata
```

### Registry Format
```json
{
  "customer_id": "cust_123",
  "current_tier": "Pro",
  "last_updated": "2024-01-15T10:00:00Z",
  "plugins": {
    "console-logger": {
      "name": "console-logger",
      "version": "1.0.0",
      "installed_at": "2024-01-15T10:00:00Z",
      "path": "/home/user/.km/plugins/km-plugin-console-logger-abc123",
      "required_tier": "Free",
      "enabled": true
    }
  }
}
```

### API Endpoints Required

1. **POST /api/plugins/provision**
   - Request customer-specific plugins
   - Returns download URLs and metadata

2. **GET /api/subscription/status**
   - Check current subscription tier
   - Used for refresh operations

## Testing

### Unit Tests
- `plugin_provisioning_service_test.go`: Comprehensive service tests
  - Successful provisioning
  - Error handling
  - Tier changes

### Integration Tests
- `init_test.go`: CLI command integration tests
- `test-plugin-provisioning.sh`: End-to-end test script

## Future Enhancements

1. **km plugins refresh**: Dedicated command for manual updates
2. **km plugins list**: Show installed plugins and status
3. **km plugins remove**: Uninstall specific plugins
4. **Background Updates**: Automatic plugin updates
5. **Plugin Verification**: Continuous integrity checking
