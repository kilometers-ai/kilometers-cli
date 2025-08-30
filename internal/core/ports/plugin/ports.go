package pluginports

import (
	"context"

	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
)

// ProvisioningService handles plugin provisioning from the API
type ProvisioningService interface {
	// ValidateAPIKey validates an API key and returns subscription info
	ValidateAPIKey(ctx context.Context, apiKey string) (*plugindomain.ValidationResult, error)

	// GetAvailablePlugins returns plugins available for the subscription
	GetAvailablePlugins(ctx context.Context, apiKey string) ([]plugindomain.Plugin, error)

	// DownloadPlugin downloads a specific plugin
	DownloadPlugin(ctx context.Context, apiKey string, pluginName string) ([]byte, error)
}

// PluginInstaller handles plugin installation to the filesystem
type PluginInstaller interface {
	// Install installs a plugin binary to the plugins directory
	Install(ctx context.Context, plugin plugindomain.Plugin, data []byte) error

	// Uninstall removes a plugin from the system
	Uninstall(ctx context.Context, pluginName string) error

	// GetInstalled returns list of installed plugins
	GetInstalled(ctx context.Context) ([]plugindomain.PluginInstallStatus, error)

	// CheckForUpdates checks if any installed plugins have updates
	CheckForUpdates(ctx context.Context, availablePlugins []plugindomain.Plugin) ([]plugindomain.PluginInstallStatus, error)
}

// PluginRegistry manages plugin registration state
type PluginRegistry interface {
	// Load loads the plugin registry
	Load(ctx context.Context) (map[string]plugindomain.PluginInstallStatus, error)

	// Save saves the plugin registry
	Save(ctx context.Context, registry map[string]plugindomain.PluginInstallStatus) error

	// AddPlugin adds a plugin to the registry
	AddPlugin(ctx context.Context, plugin plugindomain.Plugin, path string) error

	// RemovePlugin removes a plugin from the registry
	RemovePlugin(ctx context.Context, pluginName string) error
}
