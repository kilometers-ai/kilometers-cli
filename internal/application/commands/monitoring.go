package commands

import (
	"fmt"
	"time"

	"kilometers.ai/cli/internal/core/session"
)

// StartMonitoringCommand represents a command to start monitoring
type StartMonitoringCommand struct {
	Command         string                `json:"command"`
	Arguments       []string              `json:"arguments"`
	SessionConfig   session.SessionConfig `json:"session_config"`
	DebugReplayFile string                `json:"debug_replay_file,omitempty"`
}

// NewStartMonitoringCommand creates a new start monitoring command
func NewStartMonitoringCommand(command string, arguments []string, sessionConfig session.SessionConfig) *StartMonitoringCommand {
	return &StartMonitoringCommand{
		Command:       command,
		Arguments:     arguments,
		SessionConfig: sessionConfig,
	}
}

// Validate validates the start monitoring command
func (c *StartMonitoringCommand) Validate() error {
	if c.Command == "" {
		return fmt.Errorf("command is required")
	}

	if c.SessionConfig.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	if c.SessionConfig.MaxSessionSize < 0 {
		return fmt.Errorf("max session size must be 0 or greater")
	}

	return nil
}

// StopMonitoringCommand represents a command to stop monitoring
type StopMonitoringCommand struct {
	SessionID session.SessionID `json:"session_id"`
}

// NewStopMonitoringCommand creates a new stop monitoring command
func NewStopMonitoringCommand(sessionID session.SessionID) *StopMonitoringCommand {
	return &StopMonitoringCommand{
		SessionID: sessionID,
	}
}

// Validate validates the stop monitoring command
func (c *StopMonitoringCommand) Validate() error {
	if c.SessionID.String() == "" {
		return fmt.Errorf("session ID is required")
	}

	return nil
}

// PauseMonitoringCommand pauses an active monitoring session
type PauseMonitoringCommand struct {
	BaseCommand
	SessionID session.SessionID `json:"session_id"`
	Reason    string            `json:"reason,omitempty"`
}

// NewPauseMonitoringCommand creates a new pause monitoring command
func NewPauseMonitoringCommand(sessionID session.SessionID, reason string) *PauseMonitoringCommand {
	return &PauseMonitoringCommand{
		BaseCommand: NewBaseCommand("pause_monitoring"),
		SessionID:   sessionID,
		Reason:      reason,
	}
}

// Validate validates the pause monitoring command
func (c *PauseMonitoringCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	return nil
}

// ResumeMonitoringCommand resumes a paused monitoring session
type ResumeMonitoringCommand struct {
	BaseCommand
	SessionID session.SessionID `json:"session_id"`
}

// NewResumeMonitoringCommand creates a new resume monitoring command
func NewResumeMonitoringCommand(sessionID session.SessionID) *ResumeMonitoringCommand {
	return &ResumeMonitoringCommand{
		BaseCommand: NewBaseCommand("resume_monitoring"),
		SessionID:   sessionID,
	}
}

// Validate validates the resume monitoring command
func (c *ResumeMonitoringCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	return nil
}

// UpdateFilteringRulesCommand removed - filtering functionality simplified

// FlushEventsCommand forces flushing of events for a session
type FlushEventsCommand struct {
	BaseCommand
	SessionID session.SessionID `json:"session_id"`
	Force     bool              `json:"force"`
}

// NewFlushEventsCommand creates a new flush events command
func NewFlushEventsCommand(sessionID session.SessionID, force bool) *FlushEventsCommand {
	return &FlushEventsCommand{
		BaseCommand: NewBaseCommand("flush_events"),
		SessionID:   sessionID,
		Force:       force,
	}
}

// Validate validates the flush events command
func (c *FlushEventsCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	return nil
}

// GetMonitoringStatusCommand represents a command to get monitoring status
type GetMonitoringStatusCommand struct {
	SessionID *session.SessionID `json:"session_id,omitempty"`
}

// NewGetMonitoringStatusCommand creates a new get monitoring status command
func NewGetMonitoringStatusCommand(sessionID *session.SessionID) *GetMonitoringStatusCommand {
	return &GetMonitoringStatusCommand{
		SessionID: sessionID,
	}
}

// Validate validates the get monitoring status command
func (c *GetMonitoringStatusCommand) Validate() error {
	// SessionID is optional for this command
	return nil
}

// ListActiveSessionsCommand lists all active monitoring sessions
type ListActiveSessionsCommand struct {
	BaseCommand
	IncludeStats bool `json:"include_stats"`
	Limit        int  `json:"limit"`
	Offset       int  `json:"offset"`
}

// NewListActiveSessionsCommand creates a new list active sessions command
func NewListActiveSessionsCommand() *ListActiveSessionsCommand {
	return &ListActiveSessionsCommand{
		BaseCommand:  NewBaseCommand("list_active_sessions"),
		IncludeStats: true,
		Limit:        50,
		Offset:       0,
	}
}

// Validate validates the list active sessions command
func (c *ListActiveSessionsCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.Limit < 0 {
		return NewValidationError("limit cannot be negative")
	}

	if c.Offset < 0 {
		return NewValidationError("offset cannot be negative")
	}

	return nil
}

// MonitoringResult represents the result of monitoring operations
type MonitoringResult struct {
	Success       bool                   `json:"success"`
	SessionID     session.SessionID      `json:"session_id"`
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	EventsCount   int                    `json:"events_count"`
	BatchesCount  int                    `json:"batches_count"`
	FilteredCount int                    `json:"filtered_count"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration"`
	ProcessInfo   *ProcessInfo           `json:"process_info,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessInfo contains information about the monitored process
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Status    string    `json:"status"`
	ExitCode  int       `json:"exit_code"`
	StartTime time.Time `json:"start_time"`
}

// SessionStatus represents the status of a monitoring session
type SessionStatus struct {
	SessionID   session.SessionID `json:"session_id"`
	State       string            `json:"state"`
	ProcessInfo *ProcessInfo      `json:"process_info,omitempty"`
	Statistics  *SessionStats     `json:"statistics,omitempty"`
	// FilteringRules removed - filtering functionality simplified
	RecentEvents []EventSummary `json:"recent_events,omitempty"`
	LastActivity time.Time      `json:"last_activity"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// SessionStats contains statistics for a monitoring session
type SessionStats struct {
	TotalEvents      int           `json:"total_events"`
	FilteredEvents   int           `json:"filtered_events"`
	CapturedEvents   int           `json:"captured_events"`
	TotalBatches     int           `json:"total_batches"`
	HighRiskEvents   int           `json:"high_risk_events"`
	MediumRiskEvents int           `json:"medium_risk_events"`
	LowRiskEvents    int           `json:"low_risk_events"`
	AverageEventSize float64       `json:"average_event_size"`
	TotalPayloadSize int64         `json:"total_payload_size"`
	SessionDuration  time.Duration `json:"session_duration"`
	EventsPerSecond  float64       `json:"events_per_second"`
	LastEventTime    time.Time     `json:"last_event_time"`
}

// EventSummary provides a summary of an event
type EventSummary struct {
	ID        string    `json:"id"`
	Method    string    `json:"method"`
	Direction string    `json:"direction"`
	Size      int       `json:"size"`
	RiskScore int       `json:"risk_score"`
	Timestamp time.Time `json:"timestamp"`
}
