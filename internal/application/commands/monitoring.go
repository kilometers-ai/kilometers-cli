package commands

import (
	"fmt"
	"strings"
	"time"

	"kilometers.ai/cli/internal/core/session"
)

// StartMonitoringCommand initiates a new monitoring session
type StartMonitoringCommand struct {
	BaseCommand
	ProcessCommand   string                `json:"process_command"`
	ProcessArgs      []string              `json:"process_args"`
	SessionConfig    session.SessionConfig `json:"session_config"`
	FilteringRules   FilteringRulesConfig  `json:"filtering_rules"`
	WorkingDirectory string                `json:"working_directory,omitempty"`
	Environment      map[string]string     `json:"environment,omitempty"`
}

// FilteringRulesConfig represents filtering configuration in commands
type FilteringRulesConfig struct {
	MethodWhitelist        []string `json:"method_whitelist"`
	MethodBlacklist        []string `json:"method_blacklist"`
	PayloadSizeLimit       int      `json:"payload_size_limit"`
	MinimumRiskLevel       string   `json:"minimum_risk_level"`
	ExcludePingMessages    bool     `json:"exclude_ping_messages"`
	OnlyHighRiskMethods    bool     `json:"only_high_risk_methods"`
	EnableContentFiltering bool     `json:"enable_content_filtering"`
	ContentBlacklist       []string `json:"content_blacklist"`
}

// NewStartMonitoringCommand creates a new start monitoring command
func NewStartMonitoringCommand(processCommand string, processArgs []string, sessionConfig session.SessionConfig) *StartMonitoringCommand {
	return &StartMonitoringCommand{
		BaseCommand:    NewBaseCommand("start_monitoring"),
		ProcessCommand: processCommand,
		ProcessArgs:    processArgs,
		SessionConfig:  sessionConfig,
		FilteringRules: FilteringRulesConfig{
			ExcludePingMessages: true,
			MinimumRiskLevel:    "low",
		},
	}
}

// Validate validates the start monitoring command
func (c *StartMonitoringCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.ProcessCommand == "" {
		return NewValidationError("process command is required")
	}

	if c.SessionConfig.BatchSize <= 0 {
		return NewValidationError("batch size must be greater than 0")
	}

	if c.SessionConfig.FlushInterval < 0 {
		return NewValidationError("flush interval cannot be negative")
	}

	if c.SessionConfig.MaxSessionSize < 0 {
		return NewValidationError("max session size cannot be negative")
	}

	// Validate filtering rules
	if c.FilteringRules.PayloadSizeLimit < 0 {
		return NewValidationError("payload size limit cannot be negative")
	}

	validRiskLevels := []string{"low", "medium", "high"}
	isValidRiskLevel := false
	for _, level := range validRiskLevels {
		if c.FilteringRules.MinimumRiskLevel == level {
			isValidRiskLevel = true
			break
		}
	}
	if !isValidRiskLevel {
		return NewValidationError(fmt.Sprintf("minimum risk level must be one of: %s", strings.Join(validRiskLevels, ", ")))
	}

	return nil
}

// StopMonitoringCommand terminates a monitoring session
type StopMonitoringCommand struct {
	BaseCommand
	SessionID   session.SessionID `json:"session_id"`
	ForceStop   bool              `json:"force_stop"`
	GracePeriod time.Duration     `json:"grace_period"`
}

// NewStopMonitoringCommand creates a new stop monitoring command
func NewStopMonitoringCommand(sessionID session.SessionID) *StopMonitoringCommand {
	return &StopMonitoringCommand{
		BaseCommand: NewBaseCommand("stop_monitoring"),
		SessionID:   sessionID,
		ForceStop:   false,
		GracePeriod: 30 * time.Second,
	}
}

// Validate validates the stop monitoring command
func (c *StopMonitoringCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	if c.GracePeriod < 0 {
		return NewValidationError("grace period cannot be negative")
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

// UpdateFilteringRulesCommand updates filtering rules for an active session
type UpdateFilteringRulesCommand struct {
	BaseCommand
	SessionID      session.SessionID    `json:"session_id"`
	FilteringRules FilteringRulesConfig `json:"filtering_rules"`
	ApplyToFuture  bool                 `json:"apply_to_future"`
}

// NewUpdateFilteringRulesCommand creates a new update filtering rules command
func NewUpdateFilteringRulesCommand(sessionID session.SessionID, rules FilteringRulesConfig) *UpdateFilteringRulesCommand {
	return &UpdateFilteringRulesCommand{
		BaseCommand:    NewBaseCommand("update_filtering_rules"),
		SessionID:      sessionID,
		FilteringRules: rules,
		ApplyToFuture:  true,
	}
}

// Validate validates the update filtering rules command
func (c *UpdateFilteringRulesCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	if c.FilteringRules.PayloadSizeLimit < 0 {
		return NewValidationError("payload size limit cannot be negative")
	}

	validRiskLevels := []string{"low", "medium", "high"}
	isValidRiskLevel := false
	for _, level := range validRiskLevels {
		if c.FilteringRules.MinimumRiskLevel == level {
			isValidRiskLevel = true
			break
		}
	}
	if !isValidRiskLevel {
		return NewValidationError(fmt.Sprintf("minimum risk level must be one of: %s", strings.Join(validRiskLevels, ", ")))
	}

	return nil
}

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

// GetSessionStatusCommand retrieves the status of a monitoring session
type GetSessionStatusCommand struct {
	BaseCommand
	SessionID     session.SessionID `json:"session_id"`
	IncludeStats  bool              `json:"include_stats"`
	IncludeEvents bool              `json:"include_events"`
	EventLimit    int               `json:"event_limit"`
}

// NewGetSessionStatusCommand creates a new get session status command
func NewGetSessionStatusCommand(sessionID session.SessionID) *GetSessionStatusCommand {
	return &GetSessionStatusCommand{
		BaseCommand:   NewBaseCommand("get_session_status"),
		SessionID:     sessionID,
		IncludeStats:  true,
		IncludeEvents: false,
		EventLimit:    10,
	}
}

// Validate validates the get session status command
func (c *GetSessionStatusCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.SessionID.Value() == "" {
		return NewValidationError("session ID is required")
	}

	if c.EventLimit < 0 {
		return NewValidationError("event limit cannot be negative")
	}

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
	SessionID     session.SessionID `json:"session_id"`
	Status        string            `json:"status"`
	Message       string            `json:"message"`
	EventsCount   int               `json:"events_count"`
	BatchesCount  int               `json:"batches_count"`
	FilteredCount int               `json:"filtered_count"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       *time.Time        `json:"end_time,omitempty"`
	Duration      time.Duration     `json:"duration"`
	ProcessInfo   *ProcessInfo      `json:"process_info,omitempty"`
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
	SessionID      session.SessionID     `json:"session_id"`
	State          string                `json:"state"`
	ProcessInfo    *ProcessInfo          `json:"process_info,omitempty"`
	Statistics     *SessionStats         `json:"statistics,omitempty"`
	FilteringRules *FilteringRulesConfig `json:"filtering_rules,omitempty"`
	RecentEvents   []EventSummary        `json:"recent_events,omitempty"`
	LastActivity   time.Time             `json:"last_activity"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
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
