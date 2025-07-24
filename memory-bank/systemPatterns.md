# System Patterns: Architecture Implementation Guide

## Domain-Driven Design Implementation

### Bounded Context: MCP Monitoring

The kilometers CLI operates within a single, focused bounded context: **MCP (Model Context Protocol) Monitoring**. This bounded context encapsulates all the domain knowledge around monitoring, capturing, and processing MCP communications between AI assistants and their supporting infrastructure.

#### Ubiquitous Language
- **MCP Server**: External process that implements Model Context Protocol
- **Event**: Individual MCP JSON-RPC 2.0 message (request, response, notification)
- **Session**: Logical grouping of events during a monitoring period
- **Batching**: Collecting events before transmission to optimize platform communication
- **Process Monitor**: Infrastructure component that wraps and monitors MCP server processes
- **Debug Replay**: Capability to replay captured events for testing and troubleshooting

### Core Domain Layer

#### 1. Event Domain Model
```go
// Event represents a single MCP communication message
type Event struct {
    id           EventID
    method       string
    direction    Direction  // Inbound or Outbound
    payload      []byte     // Raw JSON-RPC message
    timestamp    time.Time
    metadata     map[string]interface{}
}

// Value Objects
type EventID struct {
    value string // UUID
}

type Direction int
const (
    Inbound Direction = iota  // From AI assistant to MCP server
    Outbound                  // From MCP server to AI assistant
)
```

**Design Patterns**:
- **Entity Pattern**: Event has identity and lifecycle
- **Value Object Pattern**: EventID, Direction are immutable values
- **Factory Pattern**: Event creation with validation

#### 2. Session Aggregate Root
```go
// Session manages the lifecycle of a monitoring session
type Session struct {
    id          SessionID
    events      []Event
    config      SessionConfig
    state       SessionState
    createdAt   time.Time
    metadata    map[string]interface{}
}

type SessionConfig struct {
    BatchSize int
}

type SessionState int
const (
    Created SessionState = iota
    Active
    Ended
)
```

**Design Patterns**:
- **Aggregate Root Pattern**: Session controls access to Events
- **State Machine Pattern**: Clear session lifecycle management
- **Encapsulation**: Internal event collection with controlled access

#### 3. Core Business Rules
```go
// Domain Services for business logic
type SessionManager interface {
    CreateSession(config SessionConfig) (*Session, error)
    StartSession(sessionID SessionID) error
    AddEvent(sessionID SessionID, event Event) error
    EndSession(sessionID SessionID) error
}
```

**Design Patterns**:
- **Domain Service Pattern**: Business logic that doesn't belong to entities
- **Interface Segregation**: Small, focused interfaces

## Hexagonal Architecture Implementation

### Core Architecture Layers

```
┌─ Interface Layer (Adapters) ──────────────────┐
│  • CLI Commands (Cobra framework)             │
│  • Configuration adapters                     │
│  • Dependency injection container             │
├─ Application Layer (Use Cases) ───────────────┤
│  • Command handlers (CQRS pattern)           │
│  • Application services (orchestration)       │
│  • Port interfaces (dependency abstractions)  │
├─ Domain Layer (Core Business Logic) ──────────┤
│  • Event entities and value objects           │
│  • Session aggregate roots                    │
│  • Core business rules and domain services    │
├─ Infrastructure Layer (Technical Adapters) ───┤
│  • Process monitor (MCP server wrapping)      │
│  • API gateway (platform communication)      │
│  • Configuration repository                   │
└────────────────────────────────────────────────┘
```

### Port and Adapter Patterns

#### Ports (Interfaces in Application Layer)
```go
// Primary Ports (driven by external actors)
type MonitoringService interface {
    StartMonitoring(cmd StartMonitoringCommand) error
    StopMonitoring(cmd StopMonitoringCommand) error
    GetStatus(cmd GetStatusCommand) (*MonitoringStatus, error)
}

type ConfigurationService interface {
    LoadConfiguration() (*Configuration, error)
    UpdateConfiguration(cmd UpdateConfigurationCommand) error
    ValidateConfiguration(config Configuration) error
}

// Secondary Ports (driving external dependencies)
type APIGateway interface {
    SendEvents(events []Event) error
    ReportSessionStatus(session Session) error
}

type ProcessMonitor interface {
    StartProcess(command string, args []string) error
    StopProcess() error
    IsRunning() bool
    GetEventStream() <-chan Event
}

type SessionRepository interface {
    Save(session Session) error
    FindByID(id SessionID) (*Session, error)
    GetActive() (*Session, error)
}
```

#### Adapters (Implementations in Infrastructure Layer)
```go
// Infrastructure implementations
type KilometersAPIGateway struct {
    httpClient *http.Client
    config     *Configuration
}

type MCPProcessMonitor struct {
    cmd     *exec.Cmd
    stdout  io.ReadCloser
    stderr  io.ReadCloser
    events  chan Event
}

type InMemorySessionRepository struct {
    sessions map[SessionID]*Session
    active   *Session
    mutex    sync.RWMutex
}
```

### CQRS Pattern Implementation

#### Command Side (Write Operations)
```go
// Commands represent intent to change state
type StartMonitoringCommand struct {
    Command    string
    Arguments  []string
    BatchSize  int
    DebugReplay string
}

type StopMonitoringCommand struct {
    SessionID SessionID
}

// Command Handlers
type MonitoringCommandHandler struct {
    sessionRepo   SessionRepository
    processMonitor ProcessMonitor
    apiGateway    APIGateway
}

func (h *MonitoringCommandHandler) Handle(cmd StartMonitoringCommand) error {
    // Create new session
    session, err := h.createSession(cmd)
    if err != nil {
        return err
    }
    
    // Start monitoring process
    if cmd.DebugReplay != "" {
        return h.handleDebugReplay(session, cmd.DebugReplay)
    }
    
    return h.startProcessMonitoring(session, cmd)
}
```

#### Query Side (Read Operations)
```go
// Queries represent requests for information
type GetSessionStatusQuery struct {
    SessionID SessionID
}

type SessionStatusProjection struct {
    SessionID    SessionID
    State        SessionState
    EventCount   int
    CreatedAt    time.Time
    LastEventAt  time.Time
}

// Query Handlers
type MonitoringQueryHandler struct {
    sessionRepo SessionRepository
}

func (h *MonitoringQueryHandler) Handle(query GetSessionStatusQuery) (*SessionStatusProjection, error) {
    session, err := h.sessionRepo.FindByID(query.SessionID)
    if err != nil {
        return nil, err
    }
    
    return &SessionStatusProjection{
        SessionID:   session.ID(),
        State:       session.State(),
        EventCount:  len(session.Events()),
        CreatedAt:   session.CreatedAt(),
        LastEventAt: session.LastEventAt(),
    }, nil
}
```

## Event-Driven Architecture

### Event Flow Pattern
```go
// Domain Events for cross-aggregate communication
type SessionStarted struct {
    SessionID SessionID
    Config    SessionConfig
    Timestamp time.Time
}

type EventCaptured struct {
    SessionID SessionID
    Event     Event
    Timestamp time.Time
}

type BatchReady struct {
    SessionID SessionID
    Events    []Event
    Timestamp time.Time
}

// Event handling
type EventDispatcher interface {
    Dispatch(event DomainEvent) error
    Subscribe(eventType reflect.Type, handler EventHandler) error
}

type EventHandler interface {
    Handle(event DomainEvent) error
}
```

### Message Processing Pipeline
```go
// Stream processing for MCP messages
type MessageProcessor struct {
    eventStream chan Event
    batcher     *EventBatcher
    dispatcher  EventDispatcher
}

func (p *MessageProcessor) ProcessStream() {
    for event := range p.eventStream {
        // Validate event
        if err := p.validateEvent(event); err != nil {
            log.WithError(err).Warn("Invalid event received")
            continue
        }
        
        // Add to batch
        batch := p.batcher.Add(event)
        if batch != nil {
            // Batch is ready, dispatch
            p.dispatcher.Dispatch(BatchReady{
                Events: batch,
                Timestamp: time.Now(),
            })
        }
    }
}
```

## Dependency Injection Pattern

### Container Implementation
```go
// DI Container for clean dependency management
type Container struct {
    // Core components
    sessionRepo     SessionRepository
    apiGateway      APIGateway
    processMonitor  ProcessMonitor
    
    // Services
    monitoringService     MonitoringService
    configurationService  ConfigurationService
    
    // CLI
    cliContainer *CLIContainer
}

func NewContainer(config *Configuration) (*Container, error) {
    container := &Container{}
    
    // Initialize infrastructure layer
    container.sessionRepo = NewInMemorySessionRepository()
    container.apiGateway = api.NewKilometersAPIGateway(config)
    container.processMonitor = monitoring.NewMCPProcessMonitor()
    
    // Initialize application layer
    container.monitoringService = services.NewMonitoringService(
        container.sessionRepo,
        container.processMonitor,
        container.apiGateway,
    )
    
    container.configurationService = services.NewConfigurationService(
        config.NewCompositeConfigRepository(),
    )
    
    // Initialize interface layer
    container.cliContainer = cli.NewCLIContainer(
        container.monitoringService,
        container.configurationService,
    )
    
    return container, nil
}
```

## Configuration Management Pattern

### Multi-Source Configuration
```go
// Configuration sources with precedence
type ConfigurationSource interface {
    Load() (*Configuration, error)
    Validate() error
}

// Composite pattern for multiple sources
type CompositeConfigRepository struct {
    sources []ConfigurationSource
}

func (r *CompositeConfigRepository) Load() (*Configuration, error) {
    config := &Configuration{}
    
    // Load in precedence order: Defaults < File < Environment < CLI
    for _, source := range r.sources {
        sourceConfig, err := source.Load()
        if err != nil {
            continue // Skip failed sources
        }
        
        config = config.Merge(sourceConfig)
    }
    
    return config, nil
}

// Configuration structure (simplified)
type Configuration struct {
    APIHost   string `json:"api_host"`
    APIKey    string `json:"api_key"`
    BatchSize int    `json:"batch_size"`
    Debug     bool   `json:"debug"`
}
```

## Testing Patterns

### Ports and Adapters Testing
```go
// Test doubles for external dependencies
type MockAPIGateway struct {
    sentEvents []Event
    shouldFail bool
}

func (m *MockAPIGateway) SendEvents(events []Event) error {
    if m.shouldFail {
        return errors.New("mock error")
    }
    m.sentEvents = append(m.sentEvents, events...)
    return nil
}

// Unit test focusing on domain logic
func TestSessionBatching(t *testing.T) {
    session := session.NewSession(session.SessionConfig{BatchSize: 3})
    
    // Add events one by one
    for i := 0; i < 5; i++ {
        event := testfixtures.NewEventBuilder().
            WithMethod(fmt.Sprintf("method_%d", i)).
            Build()
        
        batch := session.AddEvent(event)
        
        if i == 2 { // Third event should trigger batch
            assert.Len(t, batch, 3)
        } else if i == 4 { // Fifth event shouldn't trigger yet
            assert.Nil(t, batch)
        }
    }
}
```

### Property-Based Testing
```go
// Test invariants across random inputs
func TestSessionInvariants(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        batchSize := rapid.IntRange(1, 100).Draw(t, "batchSize")
        eventCount := rapid.IntRange(0, 1000).Draw(t, "eventCount")
        
        session := session.NewSession(session.SessionConfig{
            BatchSize: batchSize,
        })
        
        var totalBatched int
        for i := 0; i < eventCount; i++ {
            event := generateRandomEvent(t)
            batch := session.AddEvent(event)
            if batch != nil {
                totalBatched += len(batch)
            }
        }
        
        // Invariant: total batched + remaining should equal event count
        remaining := session.PendingEventCount()
        assert.Equal(t, eventCount, totalBatched+remaining)
    })
}
```

## Performance Patterns

### Resource Management
```go
// Bounded channels for backpressure
type EventProcessor struct {
    eventQueue  chan Event         // Bounded queue
    batchQueue  chan []Event       // Bounded batch queue
    workers     sync.WaitGroup
}

func (p *EventProcessor) Start() {
    p.eventQueue = make(chan Event, 1000)      // Buffer events
    p.batchQueue = make(chan []Event, 100)     // Buffer batches
    
    // Start worker goroutines
    for i := 0; i < 3; i++ {
        p.workers.Add(1)
        go p.processEvents()
    }
}

// Memory pooling for high-frequency allocations
type EventPool struct {
    pool sync.Pool
}

func (p *EventPool) Get() *Event {
    if event := p.pool.Get(); event != nil {
        return event.(*Event)
    }
    return &Event{}
}

func (p *EventPool) Put(event *Event) {
    event.Reset() // Clear for reuse
    p.pool.Put(event)
}
```

## Error Handling Patterns

### Domain Error Types
```go
// Domain-specific errors
type DomainError struct {
    Code    string
    Message string
    Cause   error
}

func (e DomainError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Error categories
var (
    ErrSessionNotFound    = DomainError{Code: "SESSION_NOT_FOUND", Message: "Session not found"}
    ErrInvalidEventState  = DomainError{Code: "INVALID_EVENT_STATE", Message: "Event in invalid state"}
    ErrBatchSizeExceeded  = DomainError{Code: "BATCH_SIZE_EXCEEDED", Message: "Batch size exceeded"}
)
```

### Graceful Degradation
```go
// Circuit breaker for external dependencies
type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    state           CircuitState
    failures        int
    lastFailure     time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == Open {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = HalfOpen
        } else {
            return ErrCircuitOpen
        }
    }
    
    err := fn()
    if err != nil {
        cb.recordFailure()
        return err
    }
    
    cb.recordSuccess()
    return nil
}
```

---

This architecture provides **clean separation of concerns**, **testability**, and **maintainability** while focusing on the core value proposition of MCP monitoring. The simplified design removes complexity while preserving architectural excellence. 