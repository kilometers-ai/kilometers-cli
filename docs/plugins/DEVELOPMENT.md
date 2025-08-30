# 🔌 **Plugin Development Guide**

Complete guide for developing secure, enterprise-grade plugins for the Kilometers CLI monitoring platform.

## 🎯 **Overview**

Kilometers CLI uses a secure plugin architecture based on HashiCorp's go-plugin framework with enterprise-grade security enhancements:

- **🔒 Customer-Specific Binaries**: Each plugin is built uniquely per customer
- **📝 Digital Signatures**: Binary integrity validation with tamper detection
- **🎫 JWT Authentication**: Plugin-specific tokens with embedded feature access
- **⏰ Real-Time Validation**: Live subscription status checking with caching
- **🚫 Graceful Degradation**: Silent failures for unauthorized access

## 🏗️ **Plugin Architecture**

### **Plugin Interface**

All plugins must implement the `KilometersPlugin` interface:

```go
type KilometersPlugin interface {
    // Initialize the plugin with configuration
    Initialize(ctx context.Context, config PluginConfig) error
    
    // Handle streaming MCP events
    HandleStreamEvent(ctx context.Context, event StreamEvent) error
    
    // Gracefully shutdown the plugin
    Shutdown(ctx context.Context) error
}
```

### **Plugin Configuration**

```go
type PluginConfig struct {
    // Customer identification
    CustomerID   string `json:"customer_id"`
    
    // Authentication
    APIKey      string `json:"api_key"`
    APIEndpoint string `json:"api_endpoint"`
    JWT         string `json:"jwt"`
    
    // Feature access
    Features    []string `json:"features"`
    Tier        string   `json:"tier"`
    
    // Runtime settings
    Debug       bool     `json:"debug"`
    LogLevel    string   `json:"log_level"`
}
```

### **Stream Events**

```go
type StreamEvent struct {
    // Event identification
    ID           string    `json:"id"`
    CorrelationID string   `json:"correlation_id"`
    Timestamp    time.Time `json:"timestamp"`
    
    // Event type and data
    Type         StreamEventType `json:"type"`
    Direction    string          `json:"direction"` // "request" or "response"
    
    // JSON-RPC message content
    Message      json.RawMessage `json:"message"`
    
    // Context information
    ServerInfo   ServerInfo      `json:"server_info"`
    ClientInfo   ClientInfo      `json:"client_info"`
}
```

## 🛠️ **Creating a New Plugin**

### **1. Project Structure**

```
my-plugin/
├── main.go                 # Plugin entry point
├── plugin/
│   ├── my_plugin.go       # Plugin implementation
│   └── grpc.go            # gRPC communication
├── manifest.json          # Plugin metadata
├── go.mod
└── go.sum
```

### **2. Plugin Entry Point** (`main.go`)

```go
package main

import (
    "log"
    "github.com/hashicorp/go-plugin"
    "your-org/my-plugin/plugin"
)

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: plugin.HandshakeConfig{
            ProtocolVersion:  1,
            MagicCookieKey:   "KILOMETERS_PLUGIN",
            MagicCookieValue: "kilometers-cli-plugin",
        },
        Plugins: map[string]plugin.Plugin{
            "kilometers": &plugin.KilometersGRPCPlugin{},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

### **3. Plugin Implementation** (`plugin/my_plugin.go`)

```go
package plugin

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
)

type MyPlugin struct {
    config PluginConfig
    client *APIClient
}

func (p *MyPlugin) Initialize(ctx context.Context, config PluginConfig) error {
    p.config = config
    
    // Validate JWT and features
    if err := p.validateAuthentication(); err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    
    // Initialize API client for Pro+ features
    if p.hasFeature("api_logging") {
        var err error
        p.client, err = NewAPIClient(config.APIEndpoint, config.JWT)
        if err != nil {
            return fmt.Errorf("API client initialization failed: %w", err)
        }
    }
    
    log.Printf("Plugin initialized for customer: %s", config.CustomerID)
    return nil
}

func (p *MyPlugin) HandleStreamEvent(ctx context.Context, event StreamEvent) error {
    // Console logging (Free tier)
    if p.hasFeature("console_logging") {
        p.logToConsole(event)
    }
    
    // API logging (Pro+ tier)
    if p.hasFeature("api_logging") && p.client != nil {
        if err := p.logToAPI(ctx, event); err != nil {
            // Silent failure for premium features
            if p.config.Debug {
                log.Printf("API logging failed: %v", err)
            }
        }
    }
    
    return nil
}

func (p *MyPlugin) Shutdown(ctx context.Context) error {
    if p.client != nil {
        return p.client.Close()
    }
    return nil
}

func (p *MyPlugin) validateAuthentication() error {
    // Validate embedded JWT token
    claims, err := ValidateJWT(p.config.JWT)
    if err != nil {
        return err
    }
    
    // Check customer match
    if claims.CustomerID != p.config.CustomerID {
        return fmt.Errorf("customer mismatch")
    }
    
    // Check expiration
    if time.Now().After(claims.ExpiresAt) {
        return fmt.Errorf("token expired")
    }
    
    return nil
}

func (p *MyPlugin) hasFeature(feature string) bool {
    for _, f := range p.config.Features {
        if f == feature {
            return true
        }
    }
    return false
}

func (p *MyPlugin) logToConsole(event StreamEvent) {
    // Silent console logging
    if p.config.Debug {
        log.Printf("[%s] %s %s", event.Type, event.Direction, event.ID)
    }
}

func (p *MyPlugin) logToAPI(ctx context.Context, event StreamEvent) error {
    return p.client.LogEvent(ctx, event)
}
```

### **4. gRPC Communication** (`plugin/grpc.go`)

```go
package plugin

import (
    "context"
    "github.com/hashicorp/go-plugin"
    "google.golang.org/grpc"
)

// KilometersGRPCPlugin implements the plugin.GRPCPlugin interface
type KilometersGRPCPlugin struct {
    plugin.Plugin
    Impl KilometersPlugin
}

func (p *KilometersGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
    RegisterKilometersPluginServer(s, &GRPCServer{Impl: p.Impl})
    return nil
}

func (p *KilometersGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
    return &GRPCClient{client: NewKilometersPluginClient(c)}, nil
}

// GRPCServer implements the gRPC server for the plugin
type GRPCServer struct {
    Impl KilometersPlugin
}

func (s *GRPCServer) Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
    config := PluginConfig{
        CustomerID:  req.CustomerID,
        APIKey:      req.APIKey,
        APIEndpoint: req.APIEndpoint,
        JWT:         req.JWT,
        Features:    req.Features,
        Tier:        req.Tier,
        Debug:       req.Debug,
        LogLevel:    req.LogLevel,
    }
    
    err := s.Impl.Initialize(ctx, config)
    return &InitializeResponse{Success: err == nil}, err
}

func (s *GRPCServer) HandleStreamEvent(ctx context.Context, req *HandleStreamEventRequest) (*HandleStreamEventResponse, error) {
    event := StreamEvent{
        ID:            req.Event.Id,
        CorrelationID: req.Event.CorrelationId,
        Timestamp:     req.Event.Timestamp.AsTime(),
        Type:          StreamEventType(req.Event.Type),
        Direction:     req.Event.Direction,
        Message:       req.Event.Message,
        // ... other fields
    }
    
    err := s.Impl.HandleStreamEvent(ctx, event)
    return &HandleStreamEventResponse{Success: err == nil}, err
}

func (s *GRPCServer) Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownResponse, error) {
    err := s.Impl.Shutdown(ctx)
    return &ShutdownResponse{Success: err == nil}, err
}

// GRPCClient implements the gRPC client for the plugin
type GRPCClient struct {
    client KilometersPluginClient
}

func (c *GRPCClient) Initialize(ctx context.Context, config PluginConfig) error {
    req := &InitializeRequest{
        CustomerID:  config.CustomerID,
        APIKey:      config.APIKey,
        APIEndpoint: config.APIEndpoint,
        JWT:         config.JWT,
        Features:    config.Features,
        Tier:        config.Tier,
        Debug:       config.Debug,
        LogLevel:    config.LogLevel,
    }
    
    _, err := c.client.Initialize(ctx, req)
    return err
}

func (c *GRPCClient) HandleStreamEvent(ctx context.Context, event StreamEvent) error {
    req := &HandleStreamEventRequest{
        Event: &StreamEventProto{
            Id:            event.ID,
            CorrelationId: event.CorrelationID,
            Timestamp:     timestamppb.New(event.Timestamp),
            Type:          string(event.Type),
            Direction:     event.Direction,
            Message:       event.Message,
            // ... other fields
        },
    }
    
    _, err := c.client.HandleStreamEvent(ctx, req)
    return err
}

func (c *GRPCClient) Shutdown(ctx context.Context) error {
    _, err := c.client.Shutdown(ctx, &ShutdownRequest{})
    return err
}
```

### **5. Plugin Manifest** (`manifest.json`)

```json
{
    "name": "my-plugin",
    "version": "1.0.0",
    "description": "Custom monitoring plugin for specific use case",
    "author": "Your Organization",
    "license": "MIT",
    "compatibility": {
        "kilometers_cli": ">=1.0.0",
        "go_version": ">=1.21"
    },
    "features": {
        "required": ["console_logging"],
        "optional": ["api_logging", "advanced_analytics"]
    },
    "permissions": {
        "network": true,
        "filesystem": false,
        "system": false
    },
    "configuration": {
        "schema": {
            "type": "object",
            "properties": {
                "log_level": {
                    "type": "string",
                    "enum": ["debug", "info", "warn", "error"],
                    "default": "info"
                },
                "custom_setting": {
                    "type": "string",
                    "description": "Custom plugin setting"
                }
            }
        }
    },
    "build": {
        "go_build_flags": ["-ldflags", "-s -w"],
        "minimum_size": true,
        "debug_symbols": false
    },
    "security": {
        "customer_specific": true,
        "digital_signature": true,
        "jwt_validation": true,
        "api_authentication": true
    }
}
```

## 🔒 **Security Implementation**

### **JWT Validation**

```go
func ValidateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Verify signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        
        // Return embedded secret (customer-specific)
        return []byte(getEmbeddedSecret()), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, fmt.Errorf("invalid token")
}

type Claims struct {
    CustomerID string   `json:"customer_id"`
    Features   []string `json:"features"`
    Tier       string   `json:"tier"`
    ExpiresAt  time.Time `json:"exp"`
    jwt.StandardClaims
}
```

### **Feature Access Control**

```go
func (p *MyPlugin) requireFeature(feature string) error {
    if !p.hasFeature(feature) {
        return fmt.Errorf("feature not available: %s", feature)
    }
    return nil
}

func (p *MyPlugin) HandleStreamEvent(ctx context.Context, event StreamEvent) error {
    // Always allow console logging
    p.logToConsole(event)
    
    // Premium features require validation
    if p.hasFeature("api_logging") {
        if err := p.logToAPI(ctx, event); err != nil {
            // Silent failure - don't expose errors to unauthorized users
            return nil
        }
    }
    
    return nil
}
```

### **Customer Isolation**

```go
func getEmbeddedSecret() string {
    // This secret is embedded at build time per customer
    // It's unique for each customer binary
    return "{{CUSTOMER_SPECIFIC_SECRET}}"
}

func getCustomerID() string {
    // Customer ID is embedded at build time
    return "{{CUSTOMER_ID}}"
}

func validateCustomerBinary() error {
    // Verify this binary was built for the correct customer
    if getCurrentCustomerID() != getCustomerID() {
        return fmt.Errorf("invalid customer binary")
    }
    return nil
}
```

## 🏗️ **Building Plugins**

### **Development Build**

```bash
# Standard development build
go build -o my-plugin main.go

# With debug symbols
go build -gcflags="all=-N -l" -o my-plugin main.go

# Test the plugin
km monitor --plugin ./my-plugin -- test-server
```

### **Production Build** (Customer-Specific)

```bash
# Build secure customer-specific plugin
./scripts/plugin/build-plugin.sh \
  --plugin my-plugin \
  --customer customer_123 \
  --api-key km_live_customer_key \
  --tier Pro \
  --sign

# Output: dist/km-plugin-my-plugin-{hash}.kmpkg
```

### **Build Process**

1. **🔍 Source Validation**: Verify plugin source integrity
2. **🔑 Credential Embedding**: Inject customer-specific secrets
3. **🏗️ Compilation**: Build optimized binary
4. **📝 Signing**: Generate digital signature
5. **📦 Packaging**: Create signed package
6. **✅ Verification**: Validate package integrity

## 🧪 **Testing Plugins**

### **Unit Testing**

```go
func TestPluginInitialization(t *testing.T) {
    plugin := &MyPlugin{}
    
    config := PluginConfig{
        CustomerID:  "test_customer",
        APIKey:      "test_key",
        Features:    []string{"console_logging"},
        Tier:        "Free",
        Debug:       true,
    }
    
    err := plugin.Initialize(context.Background(), config)
    assert.NoError(t, err)
}

func TestStreamEventHandling(t *testing.T) {
    plugin := &MyPlugin{}
    // ... initialization
    
    event := StreamEvent{
        ID:        "test_event",
        Type:      StreamEventRequest,
        Direction: "request",
        Message:   json.RawMessage(`{"method": "test"}`),
        Timestamp: time.Now(),
    }
    
    err := plugin.HandleStreamEvent(context.Background(), event)
    assert.NoError(t, err)
}
```

### **Integration Testing**

```bash
# Test plugin with kilometers-cli
./scripts/test/test-plugin-integration.sh --plugin my-plugin --debug

# Test with real MCP server
km monitor --plugin ./my-plugin --debug -- npx -y @modelcontextprotocol/server-filesystem /tmp
```

### **Security Testing**

```bash
# Test tamper detection
./scripts/plugin/demo-security-model.sh --plugin my-plugin

# Test customer isolation
./scripts/plugin/verify-plugin.sh my-plugin.kmpkg --verbose

# Test feature access control
km monitor --plugin ./my-plugin --api-key invalid_key -- test-server
```

## 📋 **Plugin Checklist**

### **✅ Development Checklist**

- [ ] Implement `KilometersPlugin` interface
- [ ] Add gRPC communication layer
- [ ] Create plugin manifest
- [ ] Implement JWT validation
- [ ] Add feature access control
- [ ] Handle graceful degradation
- [ ] Add comprehensive error handling
- [ ] Write unit and integration tests
- [ ] Add debug logging support

### **✅ Security Checklist**

- [ ] Validate JWT tokens
- [ ] Check customer ID match
- [ ] Verify token expiration
- [ ] Implement feature-based access control
- [ ] Silent failure for unauthorized features
- [ ] Secure credential embedding
- [ ] Digital signature validation
- [ ] Customer binary isolation

### **✅ Production Checklist**

- [ ] Performance optimization
- [ ] Memory leak prevention
- [ ] Resource cleanup
- [ ] Error logging
- [ ] Monitoring integration
- [ ] Documentation completion
- [ ] Security review
- [ ] Customer testing

## 📚 **Advanced Topics**

### **Custom API Clients**

```go
type APIClient struct {
    baseURL    string
    jwt        string
    httpClient *http.Client
}

func NewAPIClient(baseURL, jwt string) (*APIClient, error) {
    return &APIClient{
        baseURL: baseURL,
        jwt:     jwt,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }, nil
}

func (c *APIClient) LogEvent(ctx context.Context, event StreamEvent) error {
    req, err := c.createRequest(ctx, "POST", "/api/events", event)
    if err != nil {
        return err
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("API request failed: %d", resp.StatusCode)
    }
    
    return nil
}
```

### **Plugin Configuration**

```go
type PluginConfig struct {
    // Standard configuration
    CustomerID  string `json:"customer_id"`
    APIKey      string `json:"api_key"`
    
    // Custom plugin configuration
    CustomSettings map[string]interface{} `json:"custom_settings"`
}

func (p *MyPlugin) loadConfiguration(config PluginConfig) error {
    // Load custom settings
    if settings, ok := config.CustomSettings["my_plugin"]; ok {
        return p.parseCustomSettings(settings)
    }
    return nil
}
```

### **Performance Optimization**

```go
type MyPlugin struct {
    // Batch processing
    eventBatch   []StreamEvent
    batchSize    int
    batchTimeout time.Duration
    
    // Connection pooling
    apiClient    *APIClient
    
    // Async processing
    eventChan    chan StreamEvent
    workerCount  int
}

func (p *MyPlugin) HandleStreamEvent(ctx context.Context, event StreamEvent) error {
    // Async processing for performance
    select {
    case p.eventChan <- event:
        return nil
    default:
        // Channel full, process synchronously
        return p.processEvent(ctx, event)
    }
}
```

## 🤝 **Contributing**

### **Plugin Submission Process**

1. **Development**: Create and test your plugin
2. **Documentation**: Add comprehensive documentation
3. **Review**: Submit for security and code review
4. **Certification**: Pass security certification process
5. **Distribution**: Plugin added to official registry

### **Community Guidelines**

- Follow security best practices
- Maintain backward compatibility
- Add comprehensive tests
- Document all features
- Respect customer privacy
- Handle errors gracefully

---

## 📞 **Support**

- **Plugin Development**: [docs.kilometers.ai/plugins](https://docs.kilometers.ai/plugins)
- **Security Questions**: [security@kilometers.ai](mailto:security@kilometers.ai)
- **Technical Support**: [GitHub Discussions](https://github.com/kilometers-ai/kilometers-cli/discussions)
- **Enterprise**: [enterprise@kilometers.ai](mailto:enterprise@kilometers.ai)

---

**Happy plugin development! 🚀**