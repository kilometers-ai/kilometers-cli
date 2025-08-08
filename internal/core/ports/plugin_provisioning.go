package ports

import (
	"context"
	"io"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// PluginProvisioningService handles plugin provisioning from the API
type PluginProvisioningService interface {
	// ProvisionPlugins requests customer-specific plugins from the API
	ProvisionPlugins(ctx context.Context, apiKey string) (*domain.PluginProvisionResponse, error)

	// GetSubscriptionStatus checks the current subscription tier
	GetSubscriptionStatus(ctx context.Context, apiKey string) (tier string, err error)
}

// PluginDownloader handles secure plugin downloads
type PluginDownloader interface {
	// DownloadPlugin downloads a plugin from a secure URL
	DownloadPlugin(ctx context.Context, plugin domain.ProvisionedPlugin) (io.ReadCloser, error)

	// VerifySignature verifies the downloaded plugin's signature
	VerifySignature(pluginData []byte, signature string) error
}

// PluginInstaller handles plugin installation and management
type PluginInstaller interface {
	// InstallPlugin installs a downloaded plugin
	InstallPlugin(ctx context.Context, pluginData io.Reader, plugin domain.ProvisionedPlugin) error

	// UninstallPlugin removes an installed plugin
	UninstallPlugin(ctx context.Context, pluginName string) error

	// GetInstalledPlugins returns all installed plugins
	GetInstalledPlugins(ctx context.Context) ([]domain.InstalledPlugin, error)
}

// PluginRegistryStore manages the local plugin registry
type PluginRegistryStore interface {
	// SaveRegistry saves the plugin registry to disk
	SaveRegistry(registry *domain.PluginRegistry) error

	// LoadRegistry loads the plugin registry from disk
	LoadRegistry() (*domain.PluginRegistry, error)

	// UpdatePlugin updates a single plugin in the registry
	UpdatePlugin(plugin domain.InstalledPlugin) error
}
