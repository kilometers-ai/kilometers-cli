package ports

import (
	"time"

	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// APIGateway defines the interface for communicating with external APIs
type APIGateway interface {
	// SendEventBatch sends a batch of events to the API
	SendEventBatch(batch *session.EventBatch) error

	// SendEvent sends a single event to the API
	SendEvent(event *event.Event) error

	// CreateSession creates a new session on the server
	CreateSession(session *session.Session) error

	// TestConnection tests the API connection and authentication
	TestConnection() error

	// GetConnectionStatus returns the current connection status
	GetConnectionStatus() ConnectionStatus

	// GetAPIInfo returns information about the API
	GetAPIInfo() (*APIInfo, error)

	// ValidateAPIKey validates the provided API key
	ValidateAPIKey(apiKey string) error

	// GetUsageStats returns API usage statistics
	GetUsageStats() (*APIUsageStats, error)
}

// ConnectionStatus represents the status of the API connection
type ConnectionStatus struct {
	IsConnected   bool          `json:"is_connected"`
	LastConnected time.Time     `json:"last_connected"`
	LastError     string        `json:"last_error,omitempty"`
	Latency       time.Duration `json:"latency"`
	RetryCount    int           `json:"retry_count"`
}

// APIInfo contains information about the API
type APIInfo struct {
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	Region      string    `json:"region"`
	Endpoints   []string  `json:"endpoints"`
	RateLimit   RateLimit `json:"rate_limit"`
}

// RateLimit defines API rate limiting information
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	RequestsPerDay    int `json:"requests_per_day"`
	CurrentUsage      int `json:"current_usage"`
}

// APIUsageStats contains API usage statistics
type APIUsageStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalEvents        int64         `json:"total_events"`
	TotalPayloadSize   int64         `json:"total_payload_size"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastRequestTime    time.Time     `json:"last_request_time"`
}

// ProcessMonitor defines the interface for monitoring external processes
type ProcessMonitor interface {
	// Start starts monitoring a process with the given command and arguments
	Start(command string, args []string) error

	// Stop stops the monitoring process
	Stop() error

	// IsRunning returns true if the process is currently running
	IsRunning() bool

	// GetProcessInfo returns information about the monitored process
	GetProcessInfo() (*ProcessInfo, error)

	// ReadStdin returns a channel for reading stdin data
	ReadStdin() <-chan []byte

	// ReadStdout returns a channel for reading stdout data
	ReadStdout() <-chan []byte

	// ReadStderr returns a channel for reading stderr data
	ReadStderr() <-chan []byte

	// WriteStdin writes data to the process stdin
	WriteStdin(data []byte) error

	// Wait waits for the process to complete
	Wait() error

	// GetExitCode returns the exit code of the process
	GetExitCode() int

	// GetMonitoringStats returns monitoring statistics
	GetMonitoringStats() (*MonitoringStats, error)
}

// ProcessInfo contains information about a monitored process
type ProcessInfo struct {
	PID        int           `json:"pid"`
	Command    string        `json:"command"`
	Args       []string      `json:"args"`
	StartTime  time.Time     `json:"start_time"`
	Status     ProcessStatus `json:"status"`
	ExitCode   int           `json:"exit_code"`
	CPUPercent float64       `json:"cpu_percent"`
	MemoryMB   int64         `json:"memory_mb"`
}

// ProcessStatus represents the status of a monitored process
type ProcessStatus string

const (
	ProcessStatusRunning ProcessStatus = "running"
	ProcessStatusStopped ProcessStatus = "stopped"
	ProcessStatusError   ProcessStatus = "error"
	ProcessStatusUnknown ProcessStatus = "unknown"
)

// MonitoringStats contains statistics about process monitoring
type MonitoringStats struct {
	TotalBytesRead    int64         `json:"total_bytes_read"`
	TotalBytesWritten int64         `json:"total_bytes_written"`
	MessagesProcessed int64         `json:"messages_processed"`
	ErrorCount        int64         `json:"error_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	UptimeSeconds     int64         `json:"uptime_seconds"`
}

// NotificationGateway defines the interface for sending notifications
type NotificationGateway interface {
	// SendNotification sends a notification
	SendNotification(notification *Notification) error

	// SendAlert sends an alert notification
	SendAlert(alert *Alert) error

	// GetNotificationStatus returns the status of notification delivery
	GetNotificationStatus() (*NotificationStatus, error)

	// ConfigureNotifications configures notification settings
	ConfigureNotifications(config *NotificationConfig) error
}

// Notification represents a notification message
type Notification struct {
	ID        string               `json:"id"`
	Type      NotificationType     `json:"type"`
	Title     string               `json:"title"`
	Message   string               `json:"message"`
	Severity  NotificationSeverity `json:"severity"`
	Tags      []string             `json:"tags"`
	Metadata  map[string]string    `json:"metadata"`
	CreatedAt time.Time            `json:"created_at"`
}

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeAlert   NotificationType = "alert"
)

// NotificationSeverity defines the severity of a notification
type NotificationSeverity string

const (
	NotificationSeverityLow      NotificationSeverity = "low"
	NotificationSeverityMedium   NotificationSeverity = "medium"
	NotificationSeverityHigh     NotificationSeverity = "high"
	NotificationSeverityCritical NotificationSeverity = "critical"
)

// Alert represents an alert notification
type Alert struct {
	ID         string               `json:"id"`
	Type       AlertType            `json:"type"`
	Title      string               `json:"title"`
	Message    string               `json:"message"`
	Severity   NotificationSeverity `json:"severity"`
	EventID    *event.EventID       `json:"event_id,omitempty"`
	SessionID  *session.SessionID   `json:"session_id,omitempty"`
	RiskScore  int                  `json:"risk_score"`
	Tags       []string             `json:"tags"`
	Metadata   map[string]string    `json:"metadata"`
	CreatedAt  time.Time            `json:"created_at"`
	ResolvedAt *time.Time           `json:"resolved_at,omitempty"`
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeHighRisk        AlertType = "high_risk"
	AlertTypeConnectionError AlertType = "connection_error"
	AlertTypeProcessError    AlertType = "process_error"
	AlertTypeThresholdBreach AlertType = "threshold_breach"
	AlertTypeSystemHealth    AlertType = "system_health"
)

// NotificationStatus contains the status of notification delivery
type NotificationStatus struct {
	TotalSent         int64     `json:"total_sent"`
	TotalFailed       int64     `json:"total_failed"`
	LastSentAt        time.Time `json:"last_sent_at"`
	LastError         string    `json:"last_error,omitempty"`
	DeliveryRate      float64   `json:"delivery_rate"`
	AvailableChannels []string  `json:"available_channels"`
}

// NotificationConfig defines notification configuration
type NotificationConfig struct {
	EnableNotifications bool                 `json:"enable_notifications"`
	Channels            []string             `json:"channels"`
	MinimumSeverity     NotificationSeverity `json:"minimum_severity"`
	RateLimitPerMinute  int                  `json:"rate_limit_per_minute"`
	QuietHours          []QuietHour          `json:"quiet_hours"`
}

// QuietHour defines a period when notifications should be suppressed
type QuietHour struct {
	StartHour int      `json:"start_hour"`
	EndHour   int      `json:"end_hour"`
	Days      []string `json:"days"` // e.g., ["monday", "tuesday"]
}

// UpdateGateway defines the interface for handling CLI updates
type UpdateGateway interface {
	// CheckForUpdates checks if updates are available
	CheckForUpdates() (*UpdateInfo, error)

	// DownloadUpdate downloads the latest update
	DownloadUpdate(version string) error

	// InstallUpdate installs the downloaded update
	InstallUpdate(version string) error

	// GetCurrentVersion returns the current CLI version
	GetCurrentVersion() string

	// GetUpdateHistory returns update history
	GetUpdateHistory() ([]*UpdateInfo, error)

	// ConfigureAutoUpdate configures automatic updates
	ConfigureAutoUpdate(config *AutoUpdateConfig) error
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Version      string    `json:"version"`
	ReleaseDate  time.Time `json:"release_date"`
	Description  string    `json:"description"`
	DownloadURL  string    `json:"download_url"`
	Size         int64     `json:"size"`
	Checksum     string    `json:"checksum"`
	Critical     bool      `json:"critical"`
	ReleaseNotes string    `json:"release_notes"`
}

// AutoUpdateConfig defines automatic update configuration
type AutoUpdateConfig struct {
	Enabled            bool `json:"enabled"`
	CheckIntervalHours int  `json:"check_interval_hours"`
	AutoInstall        bool `json:"auto_install"`
	IncludeBeta        bool `json:"include_beta"`
	BackupBeforeUpdate bool `json:"backup_before_update"`
}

// LoggingGateway defines the interface for logging operations
type LoggingGateway interface {
	// Log logs a message with the specified level
	Log(level LogLevel, message string, fields map[string]interface{})

	// LogError logs an error
	LogError(err error, message string, fields map[string]interface{})

	// LogEvent logs an event
	LogEvent(event *event.Event, message string)

	// LogSession logs session information
	LogSession(session *session.Session, message string)

	// SetLogLevel sets the logging level
	SetLogLevel(level LogLevel)

	// GetLogLevel returns the current logging level
	GetLogLevel() LogLevel

	// ConfigureLogging configures logging settings
	ConfigureLogging(config *LoggingConfig) error
}

// LogLevel defines the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	Level      LogLevel `json:"level"`
	Format     string   `json:"format"` // "json" or "text"
	Output     string   `json:"output"` // "stdout", "stderr", or file path
	MaxSize    int      `json:"max_size_mb"`
	MaxAge     int      `json:"max_age_days"`
	MaxBackups int      `json:"max_backups"`
	Compress   bool     `json:"compress"`
}
