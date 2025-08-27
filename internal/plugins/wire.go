//go:build wireinject
// +build wireinject

package plugins

import (
	"time"

	"github.com/google/wire"
)

// PluginManagerOptions contains all the options for creating a PluginManager
type PluginManagerOptions struct {
	Debug       bool
	ApiEndpoint string
	PluginDirs  []string
	CLIVersion  string
}

// InitializePluginManager creates a new PluginManager with all dependencies injected
func InitializePluginManager(opts PluginManagerOptions) (*PluginManager, error) {
	wire.Build(
		// Provide the PluginManagerConfig from options
		providePluginManagerConfigFromOptions,
		
		// Validator
		provideWirePluginValidatorFromOptions,
		wire.Bind(new(PluginValidator), new(*WirePluginValidator)),
		
		// Discovery
		provideFileSystemDiscoveryFromOptions,
		wire.Bind(new(PluginDiscovery), new(*FileSystemPluginDiscovery)),
		
		// Authenticator
		provideHTTPAuthenticatorFromOptions,
		wire.Bind(new(PluginAuthenticator), new(*HTTPPluginAuthenticator)),
		
		// Auth Cache
		provideAuthCache,
		wire.Bind(new(AuthenticationCache), new(*InMemoryAuthCache)),
		
		// Plugin Manager
		NewPluginManager,
	)
	
	return nil, nil
}

// providePluginManagerConfigFromOptions creates a PluginManagerConfig from options
func providePluginManagerConfigFromOptions(opts PluginManagerOptions) *PluginManagerConfig {
	return &PluginManagerConfig{
		PluginDirectories:   opts.PluginDirs,
		AuthRefreshInterval: 5 * time.Minute,
		ApiEndpoint:         opts.ApiEndpoint,
		Debug:               opts.Debug,
		MaxPlugins:          10,
		LoadTimeout:         30 * time.Second,
		CLIVersion:          opts.CLIVersion,
	}
}

// provideWirePluginValidatorFromOptions creates a WirePluginValidator from options
func provideWirePluginValidatorFromOptions(opts PluginManagerOptions) (*WirePluginValidator, error) {
	return NewWirePluginValidator(opts.Debug)
}

// provideFileSystemDiscoveryFromOptions creates a FileSystemPluginDiscovery from options
func provideFileSystemDiscoveryFromOptions(opts PluginManagerOptions) *FileSystemPluginDiscovery {
	return NewFileSystemPluginDiscovery(opts.PluginDirs, opts.Debug)
}

// provideHTTPAuthenticatorFromOptions creates an HTTPPluginAuthenticator from options
func provideHTTPAuthenticatorFromOptions(opts PluginManagerOptions) *HTTPPluginAuthenticator {
	return NewHTTPPluginAuthenticator(opts.ApiEndpoint, opts.Debug)
}

// provideAuthCache creates an auth cache
func provideAuthCache() *InMemoryAuthCache {
	return NewInMemoryAuthCache(5 * time.Minute)
}

// InitializeSimplePluginManager creates a minimal PluginManager for testing
func InitializeSimplePluginManager() (*PluginManager, error) {
	opts := PluginManagerOptions{
		Debug:       true,
		ApiEndpoint: "http://localhost:5194",
		PluginDirs:  StandardPluginDirectories,
		CLIVersion:  "test",
	}
	return InitializePluginManager(opts)
}