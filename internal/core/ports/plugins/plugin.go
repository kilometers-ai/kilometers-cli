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

// AuthResponse represents the response from plugin authentication
type AuthResponse struct {
	Authorized bool     `json:"authorized"`
	UserTier   string   `json:"user_tier"`
	Features   []string `json:"features"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
}

// PluginConfig contains configuration for plugin initialization
type PluginConfig struct {
	ApiEndpoint string `json:"api_endpoint"`
	Debug       bool   `json:"debug"`
	ApiKey      string `json:"api_key"`
}

// StreamEvent represents a stream event
type StreamEvent struct {
	Type      StreamEventType   `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

// StreamEventType represents the type of stream event
type StreamEventType string

const (
	StreamEventTypeStart StreamEventType = "start"
	StreamEventTypeEnd   StreamEventType = "end"
	StreamEventTypeError StreamEventType = "error"
)

// PluginInfo contains metadata about a discovered plugin
type PluginInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Path         string `json:"path"`
	RequiredTier string `json:"required_tier"`
	Signature    []byte `json:"signature,omitempty"`
}

// PluginManifest contains plugin metadata from manifest file
type PluginManifest struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	RequiredTier string `json:"required_tier"`
	Author       string `json:"author,omitempty"`
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
	AuthenticatePlugin(ctx context.Context, pluginName string, apiKey string) (*AuthResponse, error)
}

// AuthenticationCache defines the interface for caching plugin authentication
type AuthenticationCache interface {
	// Get retrieves cached authentication for a plugin
	Get(pluginName string, apiKey string) *AuthResponse

	// Set stores authentication result in cache
	Set(pluginName string, apiKey string, auth *AuthResponse)

	// Clear removes cached authentication
	Clear(pluginName string, apiKey string)
}

// Standard Plugin Directories
var StandardPluginDirectories = []string{
	"~/.km/plugins/",
	"/usr/local/share/km/plugins/",
	"./plugins/",
}

// PluginError represents an error from plugin operations
type PluginError struct {
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}

// NewPluginError creates a new plugin error
func NewPluginError(message string) *PluginError {
	return &PluginError{Message: message}
}
