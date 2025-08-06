package plugins

import (
	"context"
	"time"
)

// KilometersPlugin defines the interface for go-plugins compatible plugins
// This interface will be implemented by plugin binaries and communicated via RPC
type KilometersPlugin interface {
	// Metadata
	Name() string
	Version() string
	RequiredTier() string

	// Authentication & Lifecycle
	Authenticate(ctx context.Context, apiKey string) (*AuthResponse, error)
	Initialize(ctx context.Context, config PluginConfig) error
	Shutdown(ctx context.Context) error

	// Message Processing
	HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error
	HandleError(ctx context.Context, err error) error
	HandleStreamEvent(ctx context.Context, event StreamEvent) error
}

// PluginDiscovery defines the interface for discovering available plugins
type PluginDiscovery interface {
	// DiscoverPlugins searches for plugin binaries in standard locations
	DiscoverPlugins(ctx context.Context) ([]PluginInfo, error)

	// ValidatePlugin checks if a plugin binary is valid and signed
	ValidatePlugin(ctx context.Context, pluginPath string) (*PluginInfo, error)
}

// PluginValidator defines the interface for validating plugin signatures
type PluginValidator interface {
	// ValidateSignature verifies the digital signature of a plugin binary
	ValidateSignature(ctx context.Context, pluginPath string, signature []byte) error

	// GetPluginManifest extracts metadata from a plugin binary
	GetPluginManifest(ctx context.Context, pluginPath string) (*PluginManifest, error)
}

// PluginAuthenticator defines the interface for plugin authentication with the API
type PluginAuthenticator interface {
	// AuthenticatePlugin sends authentication request to kilometers-api
	AuthenticatePlugin(ctx context.Context, request *AuthRequest) (*AuthResponse, error)

	// ValidatePlugin performs periodic validation (5-minute cycle)
	ValidatePlugin(ctx context.Context, pluginName string, token string) (*ValidationResponse, error)

	// RefreshAuthentication refreshes plugin authentication
	RefreshAuthentication(ctx context.Context, pluginName string) (*AuthResponse, error)
}

// PluginLoader defines the interface for loading and managing plugin binaries
type PluginLoader interface {
	// LoadPlugin loads a plugin binary using go-plugins
	LoadPlugin(ctx context.Context, pluginInfo *PluginInfo) (KilometersPlugin, error)

	// UnloadPlugin unloads a plugin and cleans up resources
	UnloadPlugin(ctx context.Context, pluginName string) error

	// ReloadPlugin reloads a plugin (useful for updates)
	ReloadPlugin(ctx context.Context, pluginName string) error
}

// AuthenticationCache defines the interface for caching authentication state
type AuthenticationCache interface {
	// SetAuthentication caches authentication result for a plugin
	SetAuthentication(pluginName string, auth *AuthResponse) error

	// GetAuthentication retrieves cached authentication
	GetAuthentication(pluginName string) (*AuthResponse, error)

	// IsValid checks if cached authentication is still valid
	IsValid(pluginName string) bool

	// ClearAuthentication removes cached authentication
	ClearAuthentication(pluginName string) error

	// RefreshAll refreshes all cached authentications
	RefreshAll(ctx context.Context) error
}

// Data Transfer Objects

// PluginInfo contains basic information about a discovered plugin
type PluginInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	DisplayName  string            `json:"display_name"`
	Description  string            `json:"description"`
	RequiredTier string            `json:"required_tier"`
	Path         string            `json:"path"`
	Signature    []byte            `json:"signature"`
	Metadata     map[string]string `json:"metadata"`
}

// PluginManifest contains detailed plugin metadata extracted from binary
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	RequiredTier string            `json:"required_tier"`
	Features     []string          `json:"features"`
	Capabilities []string          `json:"capabilities"`
	Dependencies map[string]string `json:"dependencies"`
	BuildInfo    BuildInfo         `json:"build_info"`
}

// BuildInfo contains information about how the plugin was built
type BuildInfo struct {
	GoVersion string    `json:"go_version"`
	BuildTime time.Time `json:"build_time"`
	GitCommit string    `json:"git_commit"`
	BuildHost string    `json:"build_host"`
	Signature string    `json:"signature"`
}

// AuthRequest contains plugin authentication request data
type AuthRequest struct {
	PluginName      string            `json:"plugin_name"`
	PluginVersion   string            `json:"plugin_version"`
	PluginSignature string            `json:"plugin_signature"`
	ApiKey          string            `json:"api_key"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// AuthResponse contains plugin authentication response from API
type AuthResponse struct {
	Success            bool              `json:"success"`
	Token              string            `json:"token"`
	ExpiresAt          time.Time         `json:"expires_at"`
	AuthorizedFeatures []string          `json:"authorized_features"`
	SubscriptionTier   string            `json:"subscription_tier"`
	CustomerName       string            `json:"customer_name"`
	PluginVersion      string            `json:"plugin_version"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// ValidationResponse contains plugin validation response from API
type ValidationResponse struct {
	IsValid            bool      `json:"is_valid"`
	AuthorizedFeatures []string  `json:"authorized_features"`
	SubscriptionTier   string    `json:"subscription_tier"`
	ValidUntil         time.Time `json:"valid_until"`
}

// PluginConfig contains configuration passed to plugins during initialization
type PluginConfig struct {
	Debug          bool              `json:"debug"`
	CorrelationID  string            `json:"correlation_id"`
	ApiEndpoint    string            `json:"api_endpoint"`
	BufferSize     int               `json:"buffer_size"`
	BatchSize      int               `json:"batch_size"`
	FlushInterval  time.Duration     `json:"flush_interval"`
	Features       []string          `json:"features"`
	CustomSettings map[string]string `json:"custom_settings"`
}

// StreamEvent represents stream lifecycle events
type StreamEvent struct {
	Type      StreamEventType   `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data"`
	Error     string            `json:"error,omitempty"`
}

// StreamEventType defines the types of stream events
type StreamEventType string

const (
	StreamEventStart        StreamEventType = "connected"
	StreamEventStop         StreamEventType = "disconnected"
	StreamEventError        StreamEventType = "error"
	StreamEventDataSent     StreamEventType = "data_sent"
	StreamEventDataReceived StreamEventType = "data_received"
	StreamEventRestart      StreamEventType = "stream_restart"
)

// Plugin Features Constants
const (
	FeatureConsoleLogging      = "console_logging"
	FeatureAPILogging          = "api_logging"
	FeatureAdvancedAnalytics   = "advanced_analytics"
	FeatureMLFeatures          = "ml_features"
	FeatureTeamCollaboration   = "team_collaboration"
	FeatureComplianceReporting = "compliance_reporting"
	FeatureEnterpriseAnalytics = "enterprise_analytics"
)

// Subscription Tiers
const (
	TierFree       = "Free"
	TierPro        = "Pro"
	TierEnterprise = "Enterprise"
)

// Standard Plugin Directories
var StandardPluginDirectories = []string{
	"~/.km/plugins/",
	"/usr/local/share/km/plugins/",
	"./plugins/",
}

// Plugin Binary Naming Convention: km-plugin-<name>
const PluginBinaryPrefix = "km-plugin-"
