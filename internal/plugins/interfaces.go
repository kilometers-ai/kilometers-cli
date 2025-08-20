package plugins

import (
	"context"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/streaming"
)

// KilometersPlugin defines the interface for go-plugins compatible plugins
// This interface will be implemented by plugin binaries and communicated via RPC
type KilometersPlugin interface {
	// Metadata
	Name() string
	Version() string
	RequiredTier() string

	// Authentication & Lifecycle
	Authenticate(ctx context.Context, apiKey string) (*auth.PluginAuthResponse, error)
	Initialize(ctx context.Context, config PluginConfig) error
	Shutdown(ctx context.Context) error

	// Message Processing
	HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error
	HandleError(ctx context.Context, err error) error
	HandleStreamEvent(ctx context.Context, event PluginStreamEvent) error
}

// PluginConfig contains configuration for plugin initialization
type PluginConfig struct {
	ApiEndpoint string `json:"api_endpoint"`
	Debug       bool   `json:"debug"`
	ApiKey      string `json:"api_key"`
}

// PluginStreamEvent represents a stream event
type PluginStreamEvent struct {
	Type      PluginStreamEventType `json:"type"`
	Timestamp time.Time             `json:"timestamp"`
	Data      map[string]string     `json:"data"`
}

// PluginStreamEventType represents the type of stream event
type PluginStreamEventType string

const (
	PluginStreamEventTypeStart PluginStreamEventType = "start"
	PluginStreamEventTypeEnd   PluginStreamEventType = "end"
	PluginStreamEventTypeError PluginStreamEventType = "error"
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
	PluginName         string                    `json:"plugin_name"`
	Version            string                    `json:"version"`
	Platform           string                    `json:"platform"`
	BinaryName         string                    `json:"binary_name"`
	BinaryHash         string                    `json:"binary_hash"`
	RequiredTier       string                    `json:"required_tier"`
	BuildTime          time.Time                 `json:"build_time"`
	BuildHost          string                    `json:"build_host"`
	SignatureAlgorithm string                    `json:"signature_algorithm"`
	APIVersion         string                    `json:"api_version"`
	Authentication     PluginAuthentication      `json:"authentication"`
}

// PluginAuthentication represents authentication configuration in manifest
type PluginAuthentication struct {
	Method           string `json:"method"`
	RuntimeValidation bool   `json:"runtime_validation"`
	UniversalBinary  bool   `json:"universal_binary"`
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
	AuthenticatePlugin(ctx context.Context, pluginName string, apiKey string) (*auth.PluginAuthResponse, error)
}

// AuthenticationCache defines the interface for caching plugin authentication
type AuthenticationCache interface {
	// Get retrieves cached authentication for a plugin
	Get(pluginName string, apiKey string) *auth.PluginAuthResponse

	// Set stores authentication result in cache
	Set(pluginName string, apiKey string, auth *auth.PluginAuthResponse)

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

// PluginProvisioningService defines the interface for plugin provisioning
type PluginProvisioningService interface {
	ProvisionPlugins(ctx context.Context, apiKey string) (*PluginProvisionResponse, error)
	GetSubscriptionStatus(ctx context.Context, apiKey string) (string, error)
}

// PluginDownloader defines the interface for downloading plugins
type PluginDownloader interface {
	Download(ctx context.Context, url string) ([]byte, error)
	DownloadPlugin(ctx context.Context, plugin ProvisionedPlugin) (interface{}, error)
}

// PluginInstaller defines the interface for installing plugins
type PluginInstaller interface {
	InstallPlugin(ctx context.Context, plugin ProvisionedPlugin) (*PluginInstallResult, error)
}

// PluginRegistryStore defines the interface for storing plugin registry
type PluginRegistryStore interface {
	Load() (*PluginRegistry, error)
	Save(*PluginRegistry) error
	LoadRegistry() (*PluginRegistry, error)
	SaveRegistry(*PluginRegistry) error
}

// PluginManagerInterface defines the common plugin manager interface
type PluginManagerInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	DiscoverAndLoadPlugins(ctx context.Context, apiKey string) error
	GetLoadedPlugins() interface{}
	HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error
	HandleError(ctx context.Context, err error) error
	HandleStreamEvent(ctx context.Context, event streaming.StreamEvent) error
}
