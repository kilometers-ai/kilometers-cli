package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"kilometers.ai/cli/internal/core/event"
)

// SessionID is a value object representing a unique session identifier
type SessionID struct {
	value string
}

// NewSessionID creates a new SessionID with validation
func NewSessionID(value string) (SessionID, error) {
	if value == "" {
		return SessionID{}, fmt.Errorf("session ID cannot be empty")
	}
	return SessionID{value: value}, nil
}

// GenerateSessionID creates a new unique SessionID
func GenerateSessionID() SessionID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return SessionID{value: hex.EncodeToString(bytes)}
}

// Value returns the string value of the SessionID
func (s SessionID) Value() string {
	return s.value
}

// String implements the Stringer interface
func (s SessionID) String() string {
	return s.value
}

// SessionConfig represents configuration for a monitoring session
type SessionConfig struct {
	BatchSize           int           `json:"batch_size"`
	FlushInterval       time.Duration `json:"flush_interval"`
	MaxSessionSize      int           `json:"max_session_size"`
	EnableRiskFiltering bool          `json:"enable_risk_filtering"`
}

// DefaultSessionConfig returns the default session configuration
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		BatchSize:           10,
		FlushInterval:       30 * time.Second,
		MaxSessionSize:      1000,
		EnableRiskFiltering: false,
	}
}

// SessionState represents the state of a monitoring session
type SessionState string

const (
	SessionStateCreated SessionState = "created"
	SessionStateActive  SessionState = "active"
	SessionStateEnded   SessionState = "ended"
)

// EventBatch represents a batch of events ready for transmission
type EventBatch struct {
	ID        string         `json:"id"`
	SessionID SessionID      `json:"session_id"`
	Events    []*event.Event `json:"events"`
	CreatedAt time.Time      `json:"created_at"`
	Size      int            `json:"size"`
}

// NewEventBatch creates a new event batch
func NewEventBatch(sessionID SessionID, events []*event.Event) *EventBatch {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	batchID := hex.EncodeToString(bytes)

	return &EventBatch{
		ID:        batchID,
		SessionID: sessionID,
		Events:    events,
		CreatedAt: time.Now(),
		Size:      len(events),
	}
}

// Session represents a monitoring session aggregate root
type Session struct {
	mu            sync.RWMutex
	id            SessionID
	config        SessionConfig
	state         SessionState
	events        []*event.Event
	startTime     time.Time
	endTime       *time.Time
	currentBatch  []*event.Event
	totalEvents   int
	batchedEvents int
	lastFlushTime time.Time
	domainEvents  []DomainEvent
}

// DomainEvent represents a domain event that occurred in the session
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	SessionID() SessionID
}

// BatchReadyEvent is emitted when a batch is ready for transmission
type BatchReadyEvent struct {
	sessionID  SessionID
	batch      *EventBatch
	occurredAt time.Time
}

func (e BatchReadyEvent) EventName() string {
	return "batch_ready"
}

func (e BatchReadyEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e BatchReadyEvent) SessionID() SessionID {
	return e.sessionID
}

func (e BatchReadyEvent) Batch() *EventBatch {
	return e.batch
}

// SessionEndedEvent is emitted when a session ends
type SessionEndedEvent struct {
	sessionID       SessionID
	totalEvents     int
	batchedEvents   int
	sessionDuration time.Duration
	occurredAt      time.Time
}

func (e SessionEndedEvent) EventName() string {
	return "session_ended"
}

func (e SessionEndedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e SessionEndedEvent) SessionID() SessionID {
	return e.sessionID
}

func (e SessionEndedEvent) TotalEvents() int {
	return e.totalEvents
}

func (e SessionEndedEvent) BatchedEvents() int {
	return e.batchedEvents
}

func (e SessionEndedEvent) SessionDuration() time.Duration {
	return e.sessionDuration
}

// NewSession creates a new monitoring session
func NewSession(config SessionConfig) *Session {
	return &Session{
		id:            GenerateSessionID(),
		config:        config,
		state:         SessionStateCreated,
		events:        make([]*event.Event, 0),
		currentBatch:  make([]*event.Event, 0),
		startTime:     time.Now(),
		lastFlushTime: time.Now(),
		domainEvents:  make([]DomainEvent, 0),
	}
}

// NewSessionWithID creates a new session with a specific ID
func NewSessionWithID(id SessionID, config SessionConfig) *Session {
	return &Session{
		id:            id,
		config:        config,
		state:         SessionStateCreated,
		events:        make([]*event.Event, 0),
		currentBatch:  make([]*event.Event, 0),
		startTime:     time.Now(),
		lastFlushTime: time.Now(),
		domainEvents:  make([]DomainEvent, 0),
	}
}

// ID returns the session ID
func (s *Session) ID() SessionID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Config returns the session configuration
func (s *Session) Config() SessionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// State returns the current session state
func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// StartTime returns the session start time
func (s *Session) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

// EndTime returns the session end time if ended
func (s *Session) EndTime() *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endTime
}

// Duration returns the session duration
func (s *Session) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.endTime != nil {
		return s.endTime.Sub(s.startTime)
	}
	return time.Since(s.startTime)
}

// TotalEvents returns the total number of events in the session
func (s *Session) TotalEvents() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalEvents
}

// BatchedEvents returns the number of events that have been batched
func (s *Session) BatchedEvents() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.batchedEvents
}

// IsActive returns true if the session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SessionStateActive
}

// IsEnded returns true if the session has ended
func (s *Session) IsEnded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SessionStateEnded
}

// Start activates the session for monitoring
func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateCreated {
		if s.state == SessionStateActive {
			return fmt.Errorf("session is already active")
		}
		return fmt.Errorf("session can only be started from created state, current state: %s", s.state)
	}

	s.state = SessionStateActive
	s.startTime = time.Now()
	s.lastFlushTime = time.Now()

	return nil
}

// AddEvent adds an event to the session and returns a batch if ready
func (s *Session) AddEvent(evt *event.Event) (*EventBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateActive {
		return nil, fmt.Errorf("cannot add events to session in state: %s", s.state)
	}

	// Check session size limit
	if s.config.MaxSessionSize > 0 && s.totalEvents >= s.config.MaxSessionSize {
		return nil, fmt.Errorf("session has reached maximum size limit: %d", s.config.MaxSessionSize)
	}

	// Add event to session
	s.events = append(s.events, evt)
	s.currentBatch = append(s.currentBatch, evt)
	s.totalEvents++

	// Check if batch is ready
	if len(s.currentBatch) >= s.config.BatchSize {
		return s.createBatch()
	}

	// Check if flush interval has passed
	if time.Since(s.lastFlushTime) >= s.config.FlushInterval && len(s.currentBatch) > 0 {
		return s.createBatch()
	}

	return nil, nil
}

// ForceFlush creates a batch from current events regardless of batch size
func (s *Session) ForceFlush() (*EventBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateActive {
		return nil, fmt.Errorf("cannot flush events from session in state: %s", s.state)
	}

	if len(s.currentBatch) == 0 {
		return nil, nil
	}

	return s.createBatch()
}

// createBatch creates a batch from current events (must be called with lock held)
func (s *Session) createBatch() (*EventBatch, error) {
	if len(s.currentBatch) == 0 {
		return nil, nil
	}

	// Create a copy of events for the batch
	batchEvents := make([]*event.Event, len(s.currentBatch))
	copy(batchEvents, s.currentBatch)

	batch := NewEventBatch(s.id, batchEvents)

	// Update counters
	s.batchedEvents += len(batchEvents)
	s.lastFlushTime = time.Now()

	// Clear current batch
	s.currentBatch = make([]*event.Event, 0)

	// Emit domain event
	domainEvent := BatchReadyEvent{
		sessionID:  s.id,
		batch:      batch,
		occurredAt: time.Now(),
	}
	s.domainEvents = append(s.domainEvents, domainEvent)

	return batch, nil
}

// End terminates the session and returns any remaining events as a batch
func (s *Session) End() (*EventBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateActive {
		return nil, fmt.Errorf("session can only be ended from active state, current state: %s", s.state)
	}

	// Create final batch if there are remaining events
	var finalBatch *EventBatch
	if len(s.currentBatch) > 0 {
		var err error
		finalBatch, err = s.createBatch()
		if err != nil {
			return nil, err
		}
	}

	// Update session state
	endTime := time.Now()
	s.endTime = &endTime
	s.state = SessionStateEnded

	// Calculate duration inline to avoid deadlock
	sessionDuration := endTime.Sub(s.startTime)

	// Emit domain event
	domainEvent := SessionEndedEvent{
		sessionID:       s.id,
		totalEvents:     s.totalEvents,
		batchedEvents:   s.batchedEvents,
		sessionDuration: sessionDuration,
		occurredAt:      time.Now(),
	}
	s.domainEvents = append(s.domainEvents, domainEvent)

	return finalBatch, nil
}

// GetDomainEvents returns all domain events that have occurred in the session
func (s *Session) GetDomainEvents() []DomainEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	events := make([]DomainEvent, len(s.domainEvents))
	copy(events, s.domainEvents)
	return events
}

// ClearDomainEvents clears all domain events (typically called after processing)
func (s *Session) ClearDomainEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domainEvents = make([]DomainEvent, 0)
}

// GetEventHistory returns all events in the session (for debugging/analysis)
func (s *Session) GetEventHistory() []*event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]*event.Event, len(s.events))
	copy(history, s.events)
	return history
}

// String returns a string representation of the session
func (s *Session) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("Session{ID: %s, State: %s, TotalEvents: %d, BatchedEvents: %d, Duration: %v}",
		s.id.Value(),
		s.state,
		s.totalEvents,
		s.batchedEvents,
		s.Duration(),
	)
}
