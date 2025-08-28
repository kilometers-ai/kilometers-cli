package plugins

import (
	"context"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/streaming"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// KilometersPlugin is now imported from the SDK
type KilometersPlugin = kmsdk.KilometersPlugin

// PluginConfig is now imported from the SDK
type PluginConfig = kmsdk.PluginConfig

// StreamEvent is now imported from the SDK
type StreamEvent = kmsdk.StreamEvent

// StreamEventType is now imported from the SDK
type StreamEventType = kmsdk.StreamEventType

// Stream event type constants from SDK
const (
	StreamEventTypeStart = kmsdk.StreamEventTypeStart
	StreamEventTypeEnd   = kmsdk.StreamEventTypeEnd
	StreamEventTypeError = kmsdk.StreamEventTypeError
)

// PluginError is now imported from the SDK
type PluginError = kmsdk.PluginError

// NewPluginError creates a new plugin error using SDK
var NewPluginError = kmsdk.NewPluginError

// PluginInfo extends the SDK's PluginInfo with additional host-specific fields
type PluginInfo struct {
	kmsdk.PluginInfo
	Path      string `json:"path"`
	Signature []byte `json:"signature,omitempty"`
}

// PluginManifest contains plugin metadata from manifest file
type PluginManifest struct {
	PluginName         string               `json:"plugin_name"`
	Version            string               `json:"version"`
	Platform           string               `json:"platform"`
	BinaryName         string               `json:"binary_name"`
	BinaryHash         string               `json:"binary_hash"`
	RequiredTier       string               `json:"required_tier"`
	BuildTime          time.Time            `json:"build_time"`
	BuildHost          string               `json:"build_host"`
	SignatureAlgorithm string               `json:"signature_algorithm"`
	APIVersion         string               `json:"api_version"`
	Authentication     PluginAuthentication `json:"authentication"`
}

// PluginAuthentication represents authentication configuration in manifest
type PluginAuthentication struct {
	Method            string `json:"method"`
	RuntimeValidation bool   `json:"runtime_validation"`
	UniversalBinary   bool   `json:"universal_binary"`
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
	HandleError(ctx context.Context, err error)
	HandleStreamEvent(ctx context.Context, event streaming.StreamEvent)
}
