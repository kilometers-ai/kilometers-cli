package ports

import (
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// EventRepository defines the interface for event persistence
type EventRepository interface {
	// Save persists an event to the repository
	Save(event *event.Event) error

	// SaveBatch persists multiple events in a single transaction
	SaveBatch(events []*event.Event) error

	// FindByID retrieves an event by its ID
	FindByID(id event.EventID) (*event.Event, error)

	// FindBySessionID retrieves all events for a specific session
	FindBySessionID(sessionID session.SessionID) ([]*event.Event, error)

	// FindBySessionIDPaginated retrieves events for a session with pagination
	FindBySessionIDPaginated(sessionID session.SessionID, offset, limit int) ([]*event.Event, error)

	// CountBySessionID returns the total number of events for a session
	CountBySessionID(sessionID session.SessionID) (int, error)

	// FindByRiskLevel retrieves events by risk level
	FindByRiskLevel(level string) ([]*event.Event, error)

	// FindByMethod retrieves events by method name
	FindByMethod(method string) ([]*event.Event, error)

	// DeleteBySessionID removes all events for a specific session
	DeleteBySessionID(sessionID session.SessionID) error

	// DeleteOlderThan removes events older than a specified timestamp
	DeleteOlderThan(timestamp int64) error

	// GetEventStatistics returns statistics about stored events
	GetEventStatistics() (EventStatistics, error)
}

// EventStatistics provides aggregate information about stored events
type EventStatistics struct {
	TotalEvents             int     `json:"total_events"`
	HighRiskEvents          int     `json:"high_risk_events"`
	MediumRiskEvents        int     `json:"medium_risk_events"`
	LowRiskEvents           int     `json:"low_risk_events"`
	TotalSessions           int     `json:"total_sessions"`
	AverageEventsPerSession float64 `json:"average_events_per_session"`
	TotalPayloadSize        int64   `json:"total_payload_size"`
	AveragePayloadSize      float64 `json:"average_payload_size"`
}

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	// Save persists a session to the repository
	Save(session *session.Session) error

	// FindByID retrieves a session by its ID
	FindByID(id session.SessionID) (*session.Session, error)

	// FindActive retrieves the currently active session if any
	FindActive() (*session.Session, error)

	// FindAll retrieves all sessions
	FindAll() ([]*session.Session, error)

	// FindAllPaginated retrieves sessions with pagination
	FindAllPaginated(offset, limit int) ([]*session.Session, error)

	// FindByState retrieves sessions by their state
	FindByState(state session.SessionState) ([]*session.Session, error)

	// Update updates an existing session
	Update(session *session.Session) error

	// Delete removes a session from the repository
	Delete(id session.SessionID) error

	// DeleteOlderThan removes sessions older than a specified timestamp
	DeleteOlderThan(timestamp int64) error

	// GetSessionStatistics returns statistics about stored sessions
	GetSessionStatistics() (SessionStatistics, error)
}

// SessionStatistics provides aggregate information about stored sessions
type SessionStatistics struct {
	TotalSessions           int     `json:"total_sessions"`
	ActiveSessions          int     `json:"active_sessions"`
	EndedSessions           int     `json:"ended_sessions"`
	AverageSessionDuration  float64 `json:"average_session_duration_seconds"`
	TotalEvents             int     `json:"total_events"`
	AverageEventsPerSession float64 `json:"average_events_per_session"`
}

// ConfigurationRepository defines the interface for configuration persistence
type ConfigurationRepository interface {
	// Load retrieves the current configuration
	Load() (*Configuration, error)

	// Save persists the configuration
	Save(config *Configuration) error

	// LoadDefault returns the default configuration
	LoadDefault() *Configuration

	// Validate validates the configuration
	Validate(config *Configuration) error

	// GetConfigPath returns the path to the configuration file
	GetConfigPath() string

	// BackupConfig creates a backup of the current configuration
	BackupConfig() error

	// RestoreConfig restores configuration from backup
	RestoreConfig() error
}

// Configuration represents the application configuration
type Configuration struct {
	APIHost   string `json:"api_host"`
	APIKey    string `json:"api_key,omitempty"`
	BatchSize int    `json:"batch_size"`
	Debug     bool   `json:"debug"`
}

// EventStore defines the interface for event storage operations
type EventStore interface {
	// Store stores an event batch
	Store(batch *session.EventBatch) error

	// StoreEvent stores a single event
	StoreEvent(event *event.Event) error

	// Retrieve retrieves events by criteria
	Retrieve(criteria EventCriteria) ([]*event.Event, error)

	// Count returns the number of events matching criteria
	Count(criteria EventCriteria) (int, error)

	// Delete removes events by criteria
	Delete(criteria EventCriteria) error

	// GetStorageStats returns storage statistics
	GetStorageStats() (StorageStatistics, error)
}

// EventCriteria defines criteria for event retrieval
type EventCriteria struct {
	SessionID    *session.SessionID `json:"session_id,omitempty"`
	EventID      *event.EventID     `json:"event_id,omitempty"`
	Method       *string            `json:"method,omitempty"`
	Direction    *event.Direction   `json:"direction,omitempty"`
	RiskLevel    *string            `json:"risk_level,omitempty"`
	MinRiskScore *int               `json:"min_risk_score,omitempty"`
	MaxRiskScore *int               `json:"max_risk_score,omitempty"`
	FromTime     *int64             `json:"from_time,omitempty"`
	ToTime       *int64             `json:"to_time,omitempty"`
	MinSize      *int               `json:"min_size,omitempty"`
	MaxSize      *int               `json:"max_size,omitempty"`
	Limit        int                `json:"limit,omitempty"`
	Offset       int                `json:"offset,omitempty"`
}

// StorageStatistics provides information about storage usage
type StorageStatistics struct {
	TotalEvents        int     `json:"total_events"`
	TotalSizeBytes     int64   `json:"total_size_bytes"`
	OldestEventTime    int64   `json:"oldest_event_time"`
	NewestEventTime    int64   `json:"newest_event_time"`
	StorageUtilization float64 `json:"storage_utilization_percent"`
}

// BackupRepository defines the interface for backup operations
type BackupRepository interface {
	// CreateBackup creates a backup of all data
	CreateBackup() (*BackupInfo, error)

	// RestoreBackup restores data from a backup
	RestoreBackup(backupID string) error

	// ListBackups returns all available backups
	ListBackups() ([]*BackupInfo, error)

	// DeleteBackup removes a backup
	DeleteBackup(backupID string) error

	// GetBackupInfo returns information about a specific backup
	GetBackupInfo(backupID string) (*BackupInfo, error)
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	ID           string `json:"id"`
	CreatedAt    int64  `json:"created_at"`
	SizeBytes    int64  `json:"size_bytes"`
	EventCount   int    `json:"event_count"`
	SessionCount int    `json:"session_count"`
	Description  string `json:"description"`
	Version      string `json:"version"`
}
