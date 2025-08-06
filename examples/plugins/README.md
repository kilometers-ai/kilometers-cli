# Kilometers CLI Plugins - Security Model Examples

This directory contains example implementations of secure Kilometers CLI plugins demonstrating the go-plugins security architecture.

## 🔒 Security Model Overview

Kilometers CLI plugins use a multi-layer security approach:

1. **Digital Signatures** - Each plugin binary is cryptographically signed
2. **Embedded Authentication** - Plugin contains customer-specific API credentials  
3. **Runtime Validation** - Plugin authenticates with kilometers-api on startup
4. **Tamper Detection** - Binary integrity verified before execution
5. **Time-Limited Tokens** - 5-minute authentication refresh cycle

## 🔌 Plugin Types

### Free Tier Plugins
- **Console Logger** - Enhanced console output (available to all users)

### Pro Tier Plugins  
- **API Logger** - Send events to Kilometers API
- **Advanced Filters** - ML-powered content filtering
- **ML Analytics** - Advanced analytics and insights

### Enterprise Tier Plugins
- **Compliance Reporting** - SOC2/GDPR compliance reports
- **Team Collaboration** - Multi-user features

## 🛠️ Plugin Development

### 1. Plugin Binary Structure
```
km-plugin-{name}/
├── main.go                 # Plugin main entry point
├── plugin/
│   ├── implementation.go   # Plugin logic
│   └── auth.go            # Embedded authentication
├── manifest.json          # Plugin metadata
└── signature.dat          # Digital signature
```

### 2. Required Interface Implementation
All plugins must implement the `KilometersPlugin` interface:

```go
type KilometersPlugin interface {
    Name() string
    Version() string
    RequiredTier() string
    
    Authenticate(ctx context.Context, apiKey string) (*AuthResponse, error)
    Initialize(ctx context.Context, config PluginConfig) error
    Shutdown(ctx context.Context) error
    
    HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error
    HandleError(ctx context.Context, err error) error
    HandleStreamEvent(ctx context.Context, event StreamEvent) error
}
```

### 3. Security Requirements

#### Embedded Authentication
```go
// Plugins include customer-specific authentication
const EmbeddedCustomerToken = "CUSTOMER_SPECIFIC_TOKEN_HERE"
const PluginSignature = "CRYPTOGRAPHIC_SIGNATURE_HERE" 
```

#### Runtime Validation
```go
func (p *Plugin) Authenticate(ctx context.Context, apiKey string) (*AuthResponse, error) {
    // 1. Validate embedded customer token
    // 2. Authenticate with kilometers-api
    // 3. Return time-limited access token
}
```

#### Tamper Detection
```go
func (p *Plugin) validateBinaryIntegrity() error {
    // 1. Calculate current binary hash
    // 2. Compare with embedded signature
    // 3. Verify with public key
}
```

## 🏭 Build Process

### 1. Customer-Specific Generation
```bash
# Generate plugin for specific customer
./build-plugin.sh --plugin=api-logger --customer=CUSTOMER_ID --api-key=CUSTOMER_API_KEY

# Output: km-plugin-api-logger-{customer-hash}
```

### 2. Digital Signing
```bash
# Sign plugin binary with private key
./sign-plugin.sh km-plugin-api-logger-{customer-hash}

# Verify signature with public key
./verify-plugin.sh km-plugin-api-logger-{customer-hash}
```

### 3. Distribution
```bash
# Upload to secure distribution endpoint
curl -X POST https://api.kilometers.ai/plugins/upload \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -F "plugin=@km-plugin-api-logger-{customer-hash}" \
  -F "customer_id=${CUSTOMER_ID}"
```

## 🔐 Security Features

### Multi-Layer Protection
1. **Build-Time**: Only included in customer-specific builds
2. **Distribution**: Signed binaries prevent tampering
3. **Runtime**: API validation ensures subscription compliance
4. **Periodic**: 5-minute re-authentication cycle
5. **Graceful**: Silent degradation on unauthorized access

### Attack Prevention
- **Source Code Protection**: Plugin source remains private
- **Reverse Engineering Resistance**: Embedded credentials are encrypted
- **API Key Rotation**: Support for credential updates
- **Subscription Enforcement**: Real-time tier validation

## 📋 Usage Examples

### Plugin Installation
```bash
# Download and install customer-specific plugin
km install-plugin api-logger --customer-token=YOUR_TOKEN

# Verify plugin installation
km list-plugins
```

### Plugin Usage
```bash
# Monitor with plugins automatically loaded
km monitor --server -- npx @modelcontextprotocol/server-github

# Expected output:
# [PluginHandler] Loaded 2 plugins:
#   ✓ console-logger v1.0.0 (Free tier)
#   ✓ api-logger v1.0.0 (Pro tier)
```

### Plugin Management
```bash
# Check plugin authentication status
km plugin-status

# Force plugin re-authentication
km plugin-refresh

# Uninstall plugin
km uninstall-plugin api-logger
```

## 🚨 Security Notes

1. **Never commit embedded credentials** to version control
2. **Plugin binaries are customer-specific** and not transferable
3. **Regular security audits** of plugin signing process
4. **Credential rotation** supported without plugin reinstall
5. **Binary verification** before every execution

## 📞 Support

For plugin development questions:
- Technical Documentation: https://docs.kilometers.ai/plugins
- Security Guidelines: https://security.kilometers.ai/plugins
- Developer Support: dev@kilometers.ai