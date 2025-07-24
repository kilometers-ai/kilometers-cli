# Kilometers CLI Refactoring Plan

## Executive Summary

This document outlines a comprehensive refactoring plan to transform the Kilometers CLI from its current monolithic structure into a well-architected application following Domain-Driven Design (DDD), Hexagonal Architecture, SOLID principles, and DRY principles.

## Current State Analysis

### Issues Identified

1. **Monolithic Structure**
   - All code in `main` package (1000+ lines in main.go)
   - No clear separation of concerns
   - Business logic mixed with infrastructure

2. **Poor Testability**
   - Tightly coupled components
   - Direct instantiation of dependencies
   - Difficult to mock external dependencies

3. **Incomplete Domain Modeling**
   - Started domain model in `core/session` but not integrated
   - Business rules scattered throughout codebase
   - No clear bounded contexts

4. **Violation of SOLID Principles**
   - Single Responsibility: Functions/structs doing too many things
   - Open/Closed: Hard to extend without modifying existing code
   - Dependency Inversion: High-level modules depend on low-level details

5. **Code Duplication**
   - Risk detection logic repeated
   - Configuration handling duplicated
   - Event processing patterns repeated

## Target Architecture

### Domain-Driven Design Structure

```
kilometers-cli/
├── cmd/
│   └── km/
│       └── main.go              # Minimal entry point
├── internal/
│   ├── core/                    # Domain Layer
│   │   ├── session/             # Session Aggregate
│   │   ├── event/               # Event Value Objects
│   │   ├── risk/                # Risk Analysis Domain Service
│   │   └── filtering/           # Filtering Domain Service
│   ├── application/             # Application Layer
│   │   ├── commands/            # Command Handlers
│   │   ├── queries/             # Query Handlers
│   │   └── services/            # Application Services
│   ├── infrastructure/          # Infrastructure Layer
│   │   ├── api/                 # API Client Adapter
│   │   ├── config/              # Configuration Adapter
│   │   ├── monitoring/          # Process Monitoring Adapter
│   │   └── persistence/         # Local Storage Adapter
│   └── interfaces/              # Interface Layer
│       ├── cli/                 # CLI Command Interface
│       └── dto/                 # Data Transfer Objects
├── pkg/                         # Public packages
│   └── mcp/                     # MCP protocol handling
└── test/                        # Test utilities and fixtures
```

### Hexagonal Architecture (Ports & Adapters)

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Interface                         │
│                    (Primary Adapter)                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                    Application Layer                         │
│              (Commands, Queries, Services)                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                      Domain Core                             │
│        (Entities, Value Objects, Domain Services)            │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Session    │  │    Event     │  │  Risk Analysis  │   │
│  │  Aggregate   │  │ Value Object │  │ Domain Service  │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                 Infrastructure Adapters                      │
│                  (Secondary Adapters)                        │
│                                                              │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ API Client │  │Process Monitor│  │ Config Provider  │   │
│  └────────────┘  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Refactoring Steps

### Phase 1: Domain Layer Foundation (Week 1)

1. **Event Domain Model**
   ```go
   // internal/core/event/event.go
   - Create Event entity with proper encapsulation
   - Implement EventID, Direction, Method value objects
   - Add RiskScore value object with business logic
   ```

2. **Session Aggregate**
   ```go
   // internal/core/session/session.go
   - Define Session aggregate root
   - Implement session lifecycle (Start, AddEvent, End)
   - Add batch management within session
   ```

3. **Risk Analysis Domain Service**
   ```go
   // internal/core/risk/analyzer.go
   - Extract risk analysis logic from risk.go
   - Create RiskAnalyzer interface
   - Implement pattern-based risk detection
   ```

4. **Filtering Domain Service**
   ```go
   // internal/core/filtering/filter.go
   - Define filtering rules as value objects
   - Create EventFilter interface
   - Implement method-based, size-based, risk-based filters
   ```

### Phase 2: Application Layer (Week 2)

1. **Command Handlers**
   ```go
   // internal/application/commands/
   - StartMonitoringCommand
   - ConfigureFilteringCommand
   - InitializeConfigCommand
   - UpdateCommand
   ```

2. **Application Services**
   ```go
   // internal/application/services/
   - MonitoringService (orchestrates session lifecycle)
   - ConfigurationService
   - UpdateService
   ```

3. **Ports Definition**
   ```go
   // internal/application/ports/
   - EventRepository interface
   - ConfigurationRepository interface
   - APIGateway interface
   - ProcessMonitor interface
   ```

### Phase 3: Infrastructure Adapters (Week 3)

1. **API Client Adapter**
   ```go
   // internal/infrastructure/api/
   - Implement APIGateway interface
   - Extract current client.go logic
   - Add retry and circuit breaker patterns
   ```

2. **Process Monitoring Adapter**
   ```go
   // internal/infrastructure/monitoring/
   - Implement ProcessMonitor interface
   - Extract process wrapper logic
   - Add proper stream handling
   ```

3. **Configuration Adapter**
   ```go
   // internal/infrastructure/config/
   - Implement ConfigurationRepository
   - Support file and environment sources
   - Add validation layer
   ```

### Phase 4: Interface Layer (Week 4)

1. **CLI Command Structure**
   ```go
   // internal/interfaces/cli/
   - Implement command pattern
   - Use cobra or similar for better CLI
   - Add proper error handling and help
   ```

2. **Dependency Injection**
   ```go
   // internal/interfaces/cli/wire.go
   - Set up dependency injection container
   - Wire all components together
   - Support different configurations (dev/prod)
   ```

## SOLID Principles Implementation

### Single Responsibility Principle

Each class/module has one reason to change:
- `Session`: Manages event lifecycle
- `RiskAnalyzer`: Analyzes event risk
- `EventFilter`: Filters events
- `APIClient`: Handles API communication

### Open/Closed Principle

Use interfaces and composition:
```go
type EventProcessor interface {
    Process(event Event) error
}

type CompositeProcessor struct {
    processors []EventProcessor
}
```

### Liskov Substitution Principle

Ensure subtypes are substitutable:
```go
type EventStore interface {
    Store(event Event) error
}

// Both implementations fulfill the contract
type InMemoryStore struct{}
type PersistentStore struct{}
```

### Interface Segregation Principle

Small, focused interfaces:
```go
type EventReader interface {
    Read(id EventID) (*Event, error)
}

type EventWriter interface {
    Write(event Event) error
}

type EventStore interface {
    EventReader
    EventWriter
}
```

### Dependency Inversion Principle

Depend on abstractions:
```go
type MonitoringService struct {
    eventStore   EventStore      // interface
    apiGateway   APIGateway      // interface
    riskAnalyzer RiskAnalyzer    // interface
}
```

## DRY Implementation

1. **Extract Common Patterns**
   - Create generic batch processor
   - Implement reusable filtering framework
   - Build common error handling utilities

2. **Template Method Pattern**
   ```go
   type BaseCommand struct {
       validate() error
       execute() error
       handleError(err error)
   }
   ```

3. **Configuration DSL**
   - Create builder pattern for complex configurations
   - Reuse validation logic

## Testing Strategy

### Unit Tests (80% coverage target)

1. **Domain Layer Tests**
   - Test each value object
   - Test aggregate invariants
   - Test domain services

2. **Application Layer Tests**
   - Mock all ports
   - Test command handlers
   - Test service orchestration

3. **Infrastructure Tests**
   - Test adapters with real dependencies
   - Use testcontainers for integration tests

### Test Structure
```go
// internal/core/session/session_test.go
func TestSession_AddEvent_ShouldEnforceBatchSize(t *testing.T) {
    // Given
    session := NewSession(WithBatchSize(2))
    
    // When
    session.AddEvent(event1)
    session.AddEvent(event2)
    batch := session.AddEvent(event3)
    
    // Then
    assert.NotNil(t, batch)
    assert.Len(t, batch.Events(), 2)
}
```

## Migration Strategy

### Phase 1: Parallel Development
- Keep existing code running
- Build new structure alongside
- Migrate piece by piece

### Phase 2: Gradual Migration
1. Extract domain models first
2. Build application services
3. Replace infrastructure components
4. Update CLI interface last

### Phase 3: Cleanup
- Remove old code
- Update documentation
- Train team on new architecture

## Success Metrics

1. **Code Quality**
   - 80%+ test coverage
   - All SOLID principles followed
   - Clear domain boundaries

2. **Performance**
   - No performance regression
   - Better memory usage through proper design

3. **Maintainability**
   - New features added without modifying core
   - Easy to understand and onboard new developers
   - Clear documentation

## Risk Mitigation

1. **Backwards Compatibility**
   - Keep CLI interface unchanged initially
   - Support old config format
   - Provide migration tools

2. **Testing**
   - Comprehensive test suite before refactoring
   - Integration tests for critical paths
   - Performance benchmarks

3. **Incremental Delivery**
   - Small, reviewable PRs
   - Feature flags for new code
   - Rollback strategy

## Timeline

- **Week 1**: Domain Layer Foundation
- **Week 2**: Application Layer
- **Week 3**: Infrastructure Adapters
- **Week 4**: Interface Layer & Integration
- **Week 5**: Testing & Documentation
- **Week 6**: Migration & Cleanup

## Next Steps

1. Review and approve this plan
2. Set up new project structure
3. Create Linear issues for each component
4. Begin Phase 1 implementation
