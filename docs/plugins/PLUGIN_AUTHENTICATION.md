# 🔐 **Plugin Authentication & Security**

In-depth guide to the Kilometers CLI plugin authentication system, security architecture, and private repository access.

## 📋 **Table of Contents**

- [Overview](#-overview)
- [Security Architecture](#-security-architecture)
- [Authentication Flow](#-authentication-flow)
- [Plugin Binary Security](#-plugin-binary-security)
- [Private Repository Access](#-private-repository-access)
- [Customer Isolation](#-customer-isolation)
- [API Integration](#-api-integration)
- [Security Best Practices](#-security-best-practices)

---

## 🎯 **Overview**

The Kilometers CLI implements a multi-layered security system that ensures:

- **Customer Isolation**: Each customer receives unique, signed binaries
- **Feature Access Control**: Tier-based feature availability
- **Continuous Validation**: Real-time authentication with 5-minute refresh
- **Graceful Degradation**: Premium features fail silently without disrupting basic functionality
- **Zero Trust**: Every plugin operation is authenticated and authorized

### **Key Security Features**

| Feature | Description | Benefit |
|---------|-------------|---------|
| **Customer-Specific Binaries** | Each plugin compiled uniquely per customer | Prevents unauthorized distribution |
| **Digital Signatures** | RSA-signed binaries with SHA256 checksums | Tamper detection |
| **JWT Authentication** | Time-limited tokens with embedded permissions | Granular access control |
| **API Key Validation** | Real-time subscription verification | Instant tier changes |
| **Embedded Secrets** | Customer credentials compiled into binary | Binary-level security |

---

## 🏗️ **Security Architecture**

### **Layered Security Model**

```mermaid
graph TB
    subgraph "Layer 1: Binary Security"
        A[Customer-Specific Build] --> B[Embedded Credentials]
        B --> C[Digital Signature]
    end
    
    subgraph "Layer 2: Authentication"
        D[API Key Validation] --> E[JWT Generation]
        E --> F[Feature Authorization]
    end
    
    subgraph "Layer 3: Runtime Security"
        G[Continuous Validation] --> H[5-min Refresh Cycle]
        H --> I[Graceful Degradation]
    end
    
    subgraph "Layer 4: Isolation"
        J[Process Isolation] --> K[Limited Permissions]
        K --> L[Audit Logging]
    end
    
    C --> D
    F --> G
    I --> J
    
    style A fill:#e3f2fd
    style E fill:#f3e5f5
    style G fill:#e8f5e9
    style J fill:#fff3e0
```

### **Component Interactions**

```mermaid
sequenceDiagram
    participant Binary as Plugin Binary
    participant Manager as Plugin Manager
    participant Auth as Auth Service
    participant API as Kilometers API
    participant Repo as Private Repo
    
    Binary->>Manager: Load Request
    Manager->>Manager: Verify Signature
    Manager->>Auth: Extract Embedded Creds
    Auth->>API: Authenticate(customerID, apiKey)
    
    API->>API: Validate Subscription
    API->>Auth: JWT + Features
    
    alt Premium Plugin
        API->>Repo: Check Access Rights
        Repo->>API: Download Token
        API->>Manager: Private Download URL
    end
    
    Auth->>Manager: Initialize Plugin
    Manager->>Binary: Load with Features
    
    loop Every 5 minutes
        Manager->>API: Refresh Auth
        API->>Manager: Updated JWT
    end
```

---

## 🔄 **Authentication Flow**

### **Initial Authentication**

1. **Plugin Discovery**
   ```go
   // Plugin manager discovers available plugins
   plugins := pm.discovery.DiscoverPlugins(ctx)
   ```

2. **Binary Validation**
   ```go
   // Validate plugin signature before loading
   err := pm.validator.ValidateSignature(ctx, plugin.Path, plugin.Signature)
   ```

3. **Customer Authentication**
   ```go
   // Create authentication request
   authReq := &AuthRequest{
       PluginName:      plugin.Name,
       PluginVersion:   plugin.Version,
       PluginSignature: signature,
       ApiKey:          apiKey,
       Metadata: map[string]string{
           "customer_id":   embeddedCustomerID,
           "required_tier": plugin.RequiredTier,
       },
   }
   ```

4. **JWT Token Generation**
   ```json
   {
     "customer_id": "cust_123",
     "subscription_tier": "Pro",
     "authorized_features": [
       "console_logging",
       "api_logging",
       "advanced_analytics"
     ],
     "plugin_permissions": {
       "api-logger": {
         "allowed": true,
         "rate_limit": 1000,
         "quota": 1000000
       }
     },
     "exp": 1704070800,
     "iat": 1704067200
   }
   ```

### **Continuous Validation**

```go
// Background refresh goroutine
func (pm *PluginManager) backgroundRefresh() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            for name, plugin := range pm.loadedPlugins {
                if err := pm.refreshPluginAuth(plugin); err != nil {
                    // Graceful degradation
                    pm.downgradePlugin(name)
                }
            }
        case <-pm.shutdown:
            return
        }
    }
}
```

---

## 🔒 **Plugin Binary Security**

### **Build-Time Security**

Each plugin binary is built with customer-specific information:

```go
// customer_auth.go (generated at build time)
package main

// Generated for customer: cust_123
// Build time: 2024-01-15T10:00:00Z
// Valid until: 2024-02-15T10:00:00Z

const (
    // Customer identification
    EmbeddedCustomerID    = "cust_123"
    EmbeddedCustomerName  = "Acme Corp"
    
    // Authentication tokens
    EmbeddedAPIKeyHash    = "sha256:a1b2c3d4e5f6..."
    EmbeddedSecretKey     = "encrypted:xyz789..."
    
    // Subscription information
    TargetTier           = "Pro"
    AuthorizedFeatures   = "console_logging,api_logging"
    
    // Build metadata
    BuildTimestamp       = "1704967200"
    BuildHost           = "secure-build-server"
    BuildVersion        = "1.0.0"
)

// Binary self-validation
func init() {
    if !validateBinaryIntegrity() {
        panic("Binary integrity check failed")
    }
}
```

### **Binary Signing Process**

```bash
# 1. Build plugin with customer data
go build -ldflags "-X main.CustomerID=cust_123" -o plugin

# 2. Generate binary hash
sha256sum plugin > plugin.sha256

# 3. Sign with private key
openssl dgst -sha256 -sign private_key.pem plugin > plugin.sig

# 4. Create manifest
cat > plugin.manifest <<EOF
{
  "name": "api-logger",
  "customer_id": "cust_123",
  "build_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "hash": "$(cat plugin.sha256)",
  "signature_algorithm": "SHA256-RSA"
}
EOF

# 5. Package for distribution
tar -czf plugin.kmpkg plugin plugin.sig plugin.manifest
```

### **Runtime Verification**

```go
func (pm *PluginManager) verifyPlugin(path string) error {
    // 1. Check file integrity
    hash, err := calculateFileHash(path)
    if err != nil {
        return err
    }
    
    // 2. Verify signature
    sig, err := readSignature(path + ".sig")
    if err != nil {
        return err
    }
    
    if !verifySignature(hash, sig, publicKey) {
        return errors.New("signature verification failed")
    }
    
    // 3. Check manifest
    manifest, err := readManifest(path + ".manifest")
    if err != nil {
        return err
    }
    
    // 4. Validate customer match
    if manifest.CustomerID != getCurrentCustomerID() {
        return errors.New("customer mismatch")
    }
    
    return nil
}
```

---

## 🏢 **Private Repository Access**

### **Enterprise Plugin Registry**

Enterprise customers have access to private plugin repositories:

```json
{
  "customer_id": "enterprise_456",
  "subscription_tier": "Enterprise",
  "private_registry": {
    "enabled": true,
    "url": "https://plugins.kilometers.ai/private/enterprise_456",
    "access_token": "pat_xxx",
    "allowed_plugins": [
      "custom-compliance-reporter",
      "team-collaboration-hub",
      "advanced-ml-analytics",
      "soc2-audit-logger"
    ]
  }
}
```

### **Private Plugin Discovery**

```go
func (pm *PluginManager) discoverPrivatePlugins(ctx context.Context) ([]PluginInfo, error) {
    // Check if customer has private repo access
    if !pm.auth.HasFeature("private_repository") {
        return nil, nil
    }
    
    // Get private plugin list from API
    resp, err := pm.apiClient.GetPrivatePlugins(ctx, pm.auth.CustomerID)
    if err != nil {
        return nil, err
    }
    
    var plugins []PluginInfo
    for _, p := range resp.Plugins {
        // Generate secure download URL
        downloadURL, err := pm.generateSecureDownloadURL(p)
        if err != nil {
            continue
        }
        
        plugins = append(plugins, PluginInfo{
            Name:         p.Name,
            Version:      p.Version,
            RequiredTier: "Enterprise",
            DownloadURL:  downloadURL,
            Private:      true,
        })
    }
    
    return plugins, nil
}
```

### **Secure Plugin Download**

```go
func (pm *PluginManager) downloadPrivatePlugin(ctx context.Context, plugin PluginInfo) error {
    // Create secure download request
    req, err := http.NewRequestWithContext(ctx, "GET", plugin.DownloadURL, nil)
    if err != nil {
        return err
    }
    
    // Add authentication headers
    req.Header.Set("Authorization", "Bearer " + pm.auth.Token)
    req.Header.Set("X-Customer-ID", pm.auth.CustomerID)
    req.Header.Set("X-Plugin-Name", plugin.Name)
    
    // Download with progress tracking
    resp, err := pm.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Verify response
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed: %s", resp.Status)
    }
    
    // Save to secure location
    pluginPath := filepath.Join(pm.securePluginDir, plugin.Name)
    return pm.saveAndVerifyPlugin(resp.Body, pluginPath, plugin)
}
```

---

## 🔐 **Customer Isolation**

### **Binary-Level Isolation**

Each customer's plugins are isolated at the binary level:

```go
// Plugin binary contains customer-specific code
func validateCustomerAccess() error {
    currentCustomer := getCurrentCustomerFromAPI()
    embeddedCustomer := EmbeddedCustomerID
    
    if currentCustomer != embeddedCustomer {
        return errors.New("customer mismatch - binary not authorized")
    }
    
    return nil
}
```

### **Runtime Process Isolation**

```go
// Plugins run in isolated processes
type PluginProcess struct {
    cmd      *exec.Cmd
    sandbox  *SecuritySandbox
    limits   *ResourceLimits
}

func (pm *PluginManager) startPlugin(plugin PluginInfo) (*PluginProcess, error) {
    // Create security sandbox
    sandbox := &SecuritySandbox{
        NetworkAccess:   plugin.RequiresNetwork,
        FilesystemPaths: plugin.AllowedPaths,
        SystemCalls:     plugin.AllowedSyscalls,
    }
    
    // Set resource limits
    limits := &ResourceLimits{
        MaxMemory:   100 * 1024 * 1024, // 100MB
        MaxCPU:      0.5,                // 50% of one core
        MaxFileSize: 10 * 1024 * 1024,  // 10MB
    }
    
    // Start isolated process
    cmd := exec.Command(plugin.Path)
    cmd.Env = sanitizeEnvironment(os.Environ())
    
    return &PluginProcess{
        cmd:     cmd,
        sandbox: sandbox,
        limits:  limits,
    }, nil
}
```

---

## 🌐 **API Integration**

### **Authentication Endpoint**

```http
POST /api/plugins/authenticate
Authorization: Bearer km_live_customer_api_key
Content-Type: application/json

{
  "plugin_name": "api-logger",
  "plugin_version": "2.1.0",
  "plugin_signature": "sig_abc123...",
  "customer_id": "cust_123"
}
```

**Response:**
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-15T15:00:00Z",
  "authorized_features": [
    "console_logging",
    "api_logging",
    "batch_processing"
  ],
  "rate_limits": {
    "requests_per_minute": 1000,
    "events_per_day": 1000000
  }
}
```

### **Validation Endpoint**

```http
POST /api/plugins/validate
Authorization: Bearer plugin_jwt_token
Content-Type: application/json

{
  "plugin_name": "api-logger",
  "plugin_token": "plugin_jwt_token"
}
```

**Response:**
```json
{
  "valid": true,
  "customer_id": "cust_123",
  "subscription_active": true,
  "features_active": true,
  "next_refresh": "2024-01-15T10:05:00Z"
}
```

---

## 🛡️ **Security Best Practices**

### **For Plugin Developers**

1. **Never Hard-code Secrets**
   ```go
   // ❌ Bad
   const apiKey = "km_live_abc123"
   
   // ✅ Good
   apiKey := os.Getenv("KM_API_KEY")
   ```

2. **Validate All Inputs**
   ```go
   func HandleMessage(data []byte) error {
       // Validate size
       if len(data) > MaxMessageSize {
           return errors.New("message too large")
       }
       
       // Validate JSON
       if !json.Valid(data) {
           return errors.New("invalid JSON")
       }
       
       return processMessage(data)
   }
   ```

3. **Implement Graceful Degradation**
   ```go
   func (p *Plugin) HandleEvent(event Event) error {
       // Try premium feature
       if err := p.sendToAPI(event); err != nil {
           // Fall back to basic logging
           p.logToConsole(event)
       }
       return nil
   }
   ```

### **For System Administrators**

1. **Regular Security Audits**
   ```bash
   # Verify all plugins
   km plugin verify --all
   
   # Check for updates
   km plugin update --check
   
   # Review permissions
   km plugin permissions --list
   ```

2. **Monitor Plugin Activity**
   ```bash
   # View plugin logs
   km plugin logs --tail 100
   
   # Check resource usage
   km plugin stats --realtime
   
   # Audit access attempts
   km audit --filter=plugins --last=24h
   ```

3. **Secure Configuration**
   ```yaml
   # km-config.yaml
   security:
     plugin_verification: strict
     allow_unsigned: false
     sandbox_enabled: true
     audit_logging: true
   
   plugins:
     auto_update: false
     allowed_sources:
       - "https://plugins.kilometers.ai"
     blocked_plugins: []
   ```

### **Security Checklist**

- [ ] API keys stored securely (environment variables or secure vault)
- [ ] Plugin signatures verified before loading
- [ ] Customer ID validation on every plugin operation
- [ ] JWT tokens refreshed before expiration
- [ ] Audit logs enabled for all plugin activities
- [ ] Resource limits configured for all plugins
- [ ] Network access restricted to necessary endpoints
- [ ] File system access limited to specific directories
- [ ] Regular security updates applied
- [ ] Incident response plan in place

---

## 🚨 **Incident Response**

### **Compromised Plugin Detection**

```go
func (pm *PluginManager) detectCompromise(plugin *LoadedPlugin) bool {
    checks := []func(*LoadedPlugin) bool{
        checkBinaryIntegrity,
        checkMemoryFingerprint,
        checkNetworkBehavior,
        checkResourceUsage,
        checkAPIPatterns,
    }
    
    for _, check := range checks {
        if !check(plugin) {
            pm.handleCompromise(plugin)
            return true
        }
    }
    
    return false
}
```

### **Automatic Remediation**

```go
func (pm *PluginManager) handleCompromise(plugin *LoadedPlugin) {
    // 1. Immediately terminate plugin
    plugin.Terminate()
    
    // 2. Revoke authentication
    pm.auth.RevokePluginAccess(plugin.Name)
    
    // 3. Alert security team
    pm.sendSecurityAlert(SecurityAlert{
        Type:      "plugin_compromise",
        Plugin:    plugin.Name,
        Customer:  plugin.CustomerID,
        Timestamp: time.Now(),
        Actions:   []string{"terminated", "access_revoked"},
    })
    
    // 4. Quarantine plugin binary
    pm.quarantinePlugin(plugin)
}
```

---

## 📚 **Further Reading**

- [Plugin Development Guide](DEVELOPMENT.md)
- [Security Architecture](../development/ARCHITECTURE.md)
- [API Documentation](https://api.kilometers.ai/docs)
- [Security Whitepaper](https://kilometers.ai/security)

---

**Questions?** Contact our security team at [security@kilometers.ai](mailto:security@kilometers.ai)
