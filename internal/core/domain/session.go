package domain

import (
	"fmt"
	"time"
)

// SessionID represents a unique monitoring session identifier
type SessionID string

// SessionStatus represents the current state of a monitoring session
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// MonitorConfig contains configuration for monitoring behavior
type MonitorConfig struct {
	BufferSize int
}

// DefaultMonitorConfig returns sensible defaults for monitoring
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		BufferSize: 1024 * 1024, // 1MB buffer
	}
}

// MonitoringSession represents an aggregate root for a monitoring session
type MonitoringSession struct {
	id            SessionID
	serverCommand Command
	startTime     time.Time
	endTime       *time.Time
	status        SessionStatus
	messages      []JSONRPCMessage
	config        MonitorConfig
	errorMessage  string
}

// NewMonitoringSession creates a new monitoring session
func NewMonitoringSession(cmd Command, config MonitorConfig) *MonitoringSession {
	sessionID := SessionID(fmt.Sprintf("session_%d", time.Now().Unix()))

	return &MonitoringSession{
		id:            sessionID,
		serverCommand: cmd,
		startTime:     time.Now(),
		status:        SessionStatusPending,
		messages:      make([]JSONRPCMessage, 0),
		config:        config,
	}
}

// ID returns the session identifier
func (s *MonitoringSession) ID() SessionID {
	return s.id
}

// ServerCommand returns the server command for this session
func (s *MonitoringSession) ServerCommand() Command {
	return s.serverCommand
}

// StartTime returns when the session was created
func (s *MonitoringSession) StartTime() time.Time {
	return s.startTime
}

// EndTime returns when the session ended (nil if still running)
func (s *MonitoringSession) EndTime() *time.Time {
	return s.endTime
}

// Status returns the current session status
func (s *MonitoringSession) Status() SessionStatus {
	return s.status
}

// Config returns the monitoring configuration
func (s *MonitoringSession) Config() MonitorConfig {
	return s.config
}

// Messages returns all captured messages
func (s *MonitoringSession) Messages() []JSONRPCMessage {
	return append([]JSONRPCMessage(nil), s.messages...) // Return copy
}

// MessageCount returns the number of captured messages
func (s *MonitoringSession) MessageCount() int {
	return len(s.messages)
}

// ErrorMessage returns the error message if session failed
func (s *MonitoringSession) ErrorMessage() string {
	return s.errorMessage
}

// Start transitions the session to running state
func (s *MonitoringSession) Start() error {
	if s.status != SessionStatusPending {
		return fmt.Errorf("cannot start session in status %s", s.status)
	}

	s.status = SessionStatusRunning
	return nil
}

// AddMessage adds a JSON-RPC message to the session
func (s *MonitoringSession) AddMessage(msg JSONRPCMessage) error {
	if s.status != SessionStatusRunning {
		return fmt.Errorf("cannot add message to session in status %s", s.status)
	}

	s.messages = append(s.messages, msg)
	return nil
}

// Complete marks the session as completed successfully
func (s *MonitoringSession) Complete() error {
	if s.status != SessionStatusRunning {
		return fmt.Errorf("cannot complete session in status %s", s.status)
	}

	endTime := time.Now()
	s.endTime = &endTime
	s.status = SessionStatusCompleted
	return nil
}

// Fail marks the session as failed with an error message
func (s *MonitoringSession) Fail(errorMsg string) error {
	if s.status == SessionStatusCompleted || s.status == SessionStatusCancelled {
		return fmt.Errorf("cannot fail session in status %s", s.status)
	}

	endTime := time.Now()
	s.endTime = &endTime
	s.status = SessionStatusFailed
	s.errorMessage = errorMsg
	return nil
}

// Cancel marks the session as cancelled
func (s *MonitoringSession) Cancel() error {
	if s.status == SessionStatusCompleted || s.status == SessionStatusFailed {
		return fmt.Errorf("cannot cancel session in status %s", s.status)
	}

	endTime := time.Now()
	s.endTime = &endTime
	s.status = SessionStatusCancelled
	return nil
}

// Duration returns how long the session has been running (or ran)
func (s *MonitoringSession) Duration() time.Duration {
	if s.endTime != nil {
		return s.endTime.Sub(s.startTime)
	}
	return time.Since(s.startTime)
}

// IsActive returns true if the session is currently running
func (s *MonitoringSession) IsActive() bool {
	return s.status == SessionStatusRunning
}

// IsCompleted returns true if the session finished successfully
func (s *MonitoringSession) IsCompleted() bool {
	return s.status == SessionStatusCompleted
}

// HasFailed returns true if the session failed
func (s *MonitoringSession) HasFailed() bool {
	return s.status == SessionStatusFailed
}
