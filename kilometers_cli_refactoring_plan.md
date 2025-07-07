# Kilometers CLI - DDD Refactoring Plan

## Phase 1: Extract Domain Layer (Week 1)
*Goal: Separate business logic from infrastructure without changing external behavior*

### Step 1.1: Create Domain Structure
```
cli/
├── domain/
│   ├── session/           # Session aggregate
│   │   ├── session.go     # MCPSession aggregate root
│   │   ├── event.go       # Event entity
│   │   └── config.go      # SessionConfig value object
│   ├── risk/              # Risk assessment bounded context
│   │   ├── detector.go    # RiskDetector domain service
│   │   ├── patterns.go    # Risk patterns value objects
│   │   └── policy.go      # Risk policies
│   └── filtering/         # Event filtering bounded context
│       ├── filter.go      # EventFilter domain service
│       └── rules.go       # Filtering rules value objects
├── infrastructure/        # Infrastructure implementations
│   ├── proxy/            # MCP proxy (performance critical)
│   ├── api/              # Kilometers API client
│   └── storage/          # Local event storage
├── application/          # Application services
│   └── services/         # Application coordination
└── interfaces/           # CLI interface
    └── cli/
```

### Step 1.2: Extract Event Domain Model
**Create: `domain/session/event.go`**
```go
package session

import (
    "time"
    "encoding/json"
)

// Event - Domain Entity
type Event struct {
    id        EventID
    timestamp time.Time
    direction Direction
    method    string
    payload   []byte
    size      int
    riskScore RiskScore
}

type EventID struct {
    value string
}

type Direction int
const (
    Inbound Direction = iota
    Outbound
)

type RiskScore struct {
    value int
    level RiskLevel
}

type RiskLevel int
const (
    Low RiskLevel = iota
    Medium
    High
)

// Domain methods
func (e Event) IsHighRisk() bool {
    return e.riskScore.level == High
}

func (e Event) ShouldBatch() bool {
    return e.size < 100*1024 // Domain rule: don't batch huge events
}
```

### Step 1.3: Extract Session Aggregate
**Create: `domain/session/session.go`**
```go
package session

import (
    "context"
    "time"
)

// MCPSession - Aggregate Root
type MCPSession struct {
    id           SessionID
    config       SessionConfig
    events       []Event
    startTime    time.Time
    isActive     bool
    eventCounter int
}

func NewMCPSession(config SessionConfig) *MCPSession {
    return &MCPSession{
        id:        NewSessionID(),
        config:    config,
        events:    make([]Event, 0),
        startTime: time.Now(),
        isActive:  true,
    }
}

// Domain behavior - this is the core business logic
func (s *MCPSession) ProcessEvent(rawPayload []byte, direction Direction, method string) (*Event, error) {
    if !s.isActive {
        return nil, ErrSessionInactive
    }

    // Create event (domain entity creation)
    event := Event{
        id:        s.nextEventID(),
        timestamp: time.Now(),
        direction: direction,
        method:    method,
        payload:   rawPayload,
        size:      len(rawPayload),
    }

    // Apply domain rules
    if s.shouldCaptureEvent(event) {
        s.events = append(s.events, event)
        return &event, nil
    }

    return nil, nil // Event filtered out
}

func (s *MCPSession) GetPendingBatch() []Event {
    if len(s.events) >= s.config.BatchSize {
        batch := make([]Event, len(s.events))
        copy(batch, s.events)
        s.events = s.events[:0] // Clear events
        return batch
    }
    return nil
}

// Private domain methods
func (s *MCPSession) shouldCaptureEvent(event Event) bool {
    // Domain logic for event capture decisions
    if s.config.ExcludePingMessages && event.method == "ping" {
        return false
    }
    
    if len(s.config.MethodWhitelist) > 0 {
        return s.isMethodWhitelisted(event.method)
    }
    
    return true
}
```

## Phase 2: Extract Infrastructure Layer (Week 2)
*Goal: Separate I/O concerns from domain logic*

### Step 2.1: Create MCP Proxy Interface
**Create: `infrastructure/proxy/proxy.go`**
```go
package proxy

import "context"

// MCPProxy - Infrastructure interface (for dependency inversion)
type MCPProxy interface {
    Start(ctx context.Context, command string, args []string) error
    SendToServer(data []byte) error
    ReadFromServer() ([]byte, error)
    Close() error
}

// ProcessMCPProxy - Concrete implementation (performance critical)
type ProcessMCPProxy struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    stderr io.ReadCloser
}

func NewProcessMCPProxy() *ProcessMCPProxy {
    return &ProcessMCPProxy{}
}

// Implementation stays exactly the same as current main.go
// This preserves the <5ms performance requirement
```

### Step 2.2: Extract API Client
**Move `client.go` to: `infrastructure/api/client.go`**
```go
package api

// KilometersAPIClient - Infrastructure interface
type KilometersAPIClient interface {
    SendEventBatch(events []Event) error
    TestConnection() error
}

// HTTPKilometersClient - Concrete implementation
// (Keep existing client.go implementation)
```

## Phase 3: Create Application Layer (Week 3)
*Goal: Orchestrate domain and infrastructure without business logic*

### Step 3.1: Create Session Service
**Create: `application/services/session_service.go`**
```go
package services

// SessionService - Application Service (orchestration only)
type SessionService struct {
    sessions   map[SessionID]*session.MCPSession
    proxy      proxy.MCPProxy
    apiClient  api.KilometersAPIClient
    riskDetector risk.Detector
    eventFilter  filtering.Filter
}

func NewSessionService(
    proxy proxy.MCPProxy,
    apiClient api.KilometersAPIClient,
    riskDetector risk.Detector,
    eventFilter filtering.Filter,
) *SessionService {
    return &SessionService{
        sessions:    make(map[SessionID]*session.MCPSession),
        proxy:      proxy,
        apiClient:  apiClient,
        riskDetector: riskDetector,
        eventFilter: eventFilter,
    }
}

// StartSession - Application workflow
func (s *SessionService) StartSession(config session.SessionConfig, command string, args []string) error {
    // Create domain aggregate
    mcpSession := session.NewMCPSession(config)
    s.sessions[mcpSession.ID()] = mcpSession
    
    // Start infrastructure (proxy)
    ctx := context.Background()
    return s.proxy.Start(ctx, command, args)
}

// ProcessInboundData - Application workflow  
func (s *SessionService) ProcessInboundData(sessionID SessionID, data []byte) error {
    mcpSession := s.sessions[sessionID]
    if mcpSession == nil {
        return ErrSessionNotFound
    }
    
    // Parse MCP message (infrastructure concern)
    msg := s.parseMCPMessage(data)
    if msg == nil {
        return nil // Not an MCP message
    }
    
    // Risk assessment (domain service)
    riskScore := s.riskDetector.AnalyzeMessage(msg, data)
    
    // Process through domain aggregate
    event, err := mcpSession.ProcessEvent(data, session.Inbound, msg.Method)
    if err != nil {
        return err
    }
    
    if event != nil {
        // Set risk score (domain rule)
        event.SetRiskScore(riskScore)
        
        // Check for batching (domain rule)
        if batch := mcpSession.GetPendingBatch(); batch != nil {
            // Send to infrastructure
            return s.apiClient.SendEventBatch(batch)
        }
    }
    
    return nil
}
```

## Phase 4: Refactor Main (Week 4)
*Goal: Slim down main.go to just dependency injection and CLI interface*

### Step 4.1: Create Dependency Container
**Create: `interfaces/cli/container.go`**
```go
package cli

// Container - Dependency injection for CLI
type Container struct {
    SessionService *services.SessionService
    Config        *session.SessionConfig
}

func NewContainer(config *session.SessionConfig) (*Container, error) {
    // Infrastructure dependencies
    proxy := proxy.NewProcessMCPProxy()
    apiClient := api.NewHTTPKilometersClient(config.APIEndpoint, config.APIKey)
    
    // Domain services
    riskDetector := risk.NewDetector()
    eventFilter := filtering.NewFilter(config)
    
    // Application service
    sessionService := services.NewSessionService(proxy, apiClient, riskDetector, eventFilter)
    
    return &Container{
        SessionService: sessionService,
        Config:        config,
    }, nil
}
```

### Step 4.2: Simplified Main
**Refactor: `main.go`**
```go
package main

import (
    "os"
    "log"
    "github.com/kilometers-ai/kilometers/cli/interfaces/cli"
)

func main() {
    // Handle built-in commands (unchanged)
    if handleCommands() {
        return
    }

    // Load configuration (unchanged)
    config, err := LoadConfig()
    if err != nil {
        log.Fatalf("Configuration error: %v", err)
    }

    // Create dependencies
    container, err := cli.NewContainer(config)
    if err != nil {
        log.Fatalf("Initialization error: %v", err)
    }

    // Create and run CLI app
    app := cli.NewApp(container)
    if err := app.Run(os.Args); err != nil {
        log.Fatalf("Runtime error: %v", err)
    }
}
```

## Phase 5: Add Testing (Week 5)
*Goal: Comprehensive testing now that concerns are separated*

### Domain Tests
```go
// domain/session/session_test.go
func TestSessionProcessEvent(t *testing.T) {
    config := session.DefaultConfig()
    session := session.NewMCPSession(config)
    
    event, err := session.ProcessEvent(
        []byte(`{"method":"ping"}`),
        session.Inbound,
        "ping",
    )
    
    assert.NoError(t, err)
    assert.Nil(t, event) // Should be filtered
}

func TestSessionBatching(t *testing.T) {
    config := session.Config{BatchSize: 2}
    session := session.NewMCPSession(config)
    
    // Add events
    session.ProcessEvent(mockEventData(), session.Inbound, "test")
    session.ProcessEvent(mockEventData(), session.Inbound, "test")
    
    batch := session.GetPendingBatch()
    assert.Len(t, batch, 2)
}
```

### Application Tests
```go
// application/services/session_service_test.go
func TestSessionServiceIntegration(t *testing.T) {
    mockProxy := &proxy.MockMCPProxy{}
    mockAPI := &api.MockKilometersClient{}
    
    service := services.NewSessionService(mockProxy, mockAPI, ...)
    
    err := service.StartSession(config, "test-command", []string{})
    assert.NoError(t, err)
    
    err = service.ProcessInboundData(sessionID, mockMCPData)
    assert.NoError(t, err)
    
    // Verify interactions
    assert.True(t, mockAPI.BatchSent)
}
```

## Benefits of This Refactoring

### 1. Performance Preserved
- Proxy code stays in infrastructure layer with minimal overhead
- Event processing moved to domain layer (business logic)
- Batching logic clearly separated

### 2. Testability Improved
- Domain logic can be unit tested without I/O
- Infrastructure can be mocked for integration tests
- Application services can be tested with real domain objects

### 3. Maintainability Enhanced
- Clear separation of concerns
- Domain rules are explicit and centralized
- Infrastructure changes don't affect business logic

### 4. Future-Proofing
- Easy to add new MCP protocols (implement MCPProxy interface)
- Risk detection can evolve independently
- API client can be swapped out

## Implementation Strategy

### Week 1: Domain Extraction
- Extract Event, Session, Config value objects
- Move risk detection logic to domain service
- Create domain interfaces

### Week 2: Infrastructure Isolation
- Extract MCP proxy to infrastructure
- Move API client to infrastructure layer
- Create infrastructure interfaces

### Week 3: Application Orchestration
- Create session service for coordination
- Move goroutine management to application layer
- Implement dependency injection

### Week 4: Interface Simplification
- Slim down main.go to dependency setup
- Create CLI application wrapper
- Clean up command handling

### Week 5: Testing & Validation
- Add comprehensive domain tests
- Add application service tests
- Validate performance hasn't regressed

This plan maintains the <5ms overhead requirement while introducing proper DDD structure for maintainability and testability.