package ports

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// ConfigDiscoveryService defines the interface for configuration discovery
type ConfigDiscoveryService interface {
	// DiscoverConfig searches for configuration from all available sources
	DiscoverConfig(ctx context.Context) (*domain.DiscoveredConfig, error)

	// ValidateConfig validates a discovered configuration
	ValidateConfig(config *domain.DiscoveredConfig) error

	// MigrateConfig migrates configuration from old formats
	MigrateConfig(oldConfig map[string]interface{}) (*domain.ConfigMigration, error)
}

// ConfigScanner defines the interface for scanning configuration from a specific source
type ConfigScanner interface {
	// Scan searches for configuration values from this source
	Scan(ctx context.Context) (*domain.DiscoveredConfig, error)

	// Name returns the name of this scanner for logging
	Name() string
}

// EnvironmentScanner scans environment variables for configuration
type EnvironmentScanner interface {
	ConfigScanner

	// SetPrefixes sets the environment variable prefixes to scan
	SetPrefixes(prefixes []string)
}

// FileSystemScanner scans the file system for configuration files
type FileSystemScanner interface {
	ConfigScanner

	// SetSearchPaths sets the paths to search for config files
	SetSearchPaths(paths []string)

	// SetFilePatterns sets the file patterns to match (e.g., "*.yaml", "*.json")
	SetFilePatterns(patterns []string)
}

// APIEndpointDiscoverer discovers API endpoints from various sources
type APIEndpointDiscoverer interface {
	// DiscoverEndpoints searches for API endpoints
	DiscoverEndpoints(ctx context.Context) ([]string, error)

	// ValidateEndpoint checks if an endpoint is valid
	ValidateEndpoint(ctx context.Context, endpoint string) error
}

// CredentialLocator safely locates credentials from various sources
type CredentialLocator interface {
	// LocateAPIKey searches for API key in secure locations
	LocateAPIKey(ctx context.Context) (string, string, error) // key, source, error

	// StoreAPIKey securely stores an API key
	StoreAPIKey(ctx context.Context, key string) error
}

// ConfigValidator validates configuration values
type ConfigValidator interface {
	// ValidateAPIEndpoint validates an API endpoint URL
	ValidateAPIEndpoint(endpoint string) error

	// ValidateAPIKey validates an API key format
	ValidateAPIKey(key string) error

	// ValidateBufferSize validates buffer size value
	ValidateBufferSize(size interface{}) error

	// ValidateLogLevel validates log level value
	ValidateLogLevel(level string) error

	// ValidatePluginsDir validates plugins directory path
	ValidatePluginsDir(path string) error

	// ValidateTimeout validates timeout duration
	ValidateTimeout(timeout interface{}) error
}

// ConfigMigrator handles migration from old configuration formats
type ConfigMigrator interface {
	// DetectLegacyConfig checks if legacy configuration exists
	DetectLegacyConfig() (bool, string, error) // exists, path, error

	// MigrateLegacyConfig migrates from legacy format
	MigrateLegacyConfig(legacyPath string) (*domain.ConfigMigration, map[string]interface{}, error)
}

// ConfigStore handles configuration persistence
type ConfigStore interface {
	// Load loads configuration from storage
	Load() (*domain.Config, error)

	// Save saves configuration to storage
	Save(config *domain.Config) error

	// Exists checks if configuration exists
	Exists() bool

	// Path returns the configuration file path
	Path() string
}

