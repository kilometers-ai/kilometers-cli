package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

// ProvisioningManager orchestrates plugin provisioning
type ProvisioningManager struct {
	provisioningService PluginProvisioningService
	downloader          PluginDownloader
	installer           PluginInstaller
	registryStore       PluginRegistryStore
}

// NewProvisioningManager creates a new plugin provisioning manager
func NewProvisioningManager(
	provisioningService PluginProvisioningService,
	downloader PluginDownloader,
	installer PluginInstaller,
	registryStore PluginRegistryStore,
) *ProvisioningManager {
	return &ProvisioningManager{
		provisioningService: provisioningService,
		downloader:          downloader,
		installer:           installer,
		registryStore:       registryStore,
	}
}

// AutoProvisionPlugins automatically provisions and installs plugins for the customer
func (m *ProvisioningManager) AutoProvisionPlugins(ctx context.Context, config *config.UnifiedConfig) error {
	if !config.HasAPIKey() {
		return fmt.Errorf("API key is required for plugin provisioning")
	}

	// Request plugin provisioning from API
	provisionResp, err := m.provisioningService.ProvisionPlugins(ctx, config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to provision plugins: %w", err)
	}

	// Load or create plugin registry
	registry, err := m.loadOrCreateRegistry(provisionResp.CustomerID, provisionResp.SubscriptionTier)
	if err != nil {
		return fmt.Errorf("failed to load plugin registry: %w", err)
	}

	// Download and install each plugin
	successCount := 0
	for _, plugin := range provisionResp.Plugins {
		if err := m.downloadAndInstallPlugin(ctx, plugin, registry); err != nil {
			fmt.Printf("⚠️  Failed to install %s: %v\n", plugin.Name, err)
			continue
		}
		successCount++
		fmt.Printf("✅ Installed %s plugin (v%s)\n", plugin.Name, plugin.Version)
	}

	// Save updated registry
	if err := m.registryStore.SaveRegistry(registry); err != nil {
		return fmt.Errorf("failed to save plugin registry: %w", err)
	}

	if successCount == 0 {
		return fmt.Errorf("no plugins were successfully installed")
	}

	fmt.Printf("\n🎉 Successfully installed %d/%d plugins\n", successCount, len(provisionResp.Plugins))
	return nil
}

// RefreshPlugins checks for plugin updates and tier changes
func (m *ProvisioningManager) RefreshPlugins(ctx context.Context, config *config.UnifiedConfig) error {
	// Get current subscription status
	currentTier, err := m.provisioningService.GetSubscriptionStatus(ctx, config.APIKey)
	if err != nil {
		return fmt.Errorf("failed to check subscription status: %w", err)
	}

	// Load current registry
	registry, err := m.registryStore.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load plugin registry: %w", err)
	}

	// Check for tier change
	if registry.CurrentTier != currentTier {
		fmt.Printf("🔄 Subscription tier changed from %s to %s\n", registry.CurrentTier, currentTier)
		return m.handleTierChange(ctx, config, registry, currentTier)
	}

	// Check if refresh is needed
	if !registry.ShouldRefresh() {
		return nil
	}

	// Re-provision plugins to check for updates
	return m.AutoProvisionPlugins(ctx, config)
}

// Private methods

func (m *ProvisioningManager) loadOrCreateRegistry(customerID, tier string) (*PluginRegistry, error) {
	registry, err := m.registryStore.LoadRegistry()
	if err != nil {
		// Create new registry if none exists
		registry = &PluginRegistry{
			CustomerID:  customerID,
			CurrentTier: tier,
			Plugins:     make(map[string]InstalledPlugin),
		}
	}

	registry.CustomerID = customerID
	registry.CurrentTier = tier

	return registry, nil
}

func (m *ProvisioningManager) downloadAndInstallPlugin(
	ctx context.Context,
	plugin ProvisionedPlugin,
	registry *PluginRegistry,
) error {
	// Check if already installed with same version
	if installed, exists := registry.Plugins[plugin.Name]; exists && installed.Version == plugin.Version {
		fmt.Printf("⏭️  %s already up to date (v%s)\n", plugin.Name, plugin.Version)
		return nil
	}

	fmt.Printf("📦 Downloading %s plugin...\n", plugin.Name)

	// Download plugin
	readerInterface, err := m.downloader.DownloadPlugin(ctx, plugin)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if closer, ok := readerInterface.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	// Install plugin
	result, err := m.installer.InstallPlugin(ctx, plugin)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("installation failed: %s", result.Error)
	}

	// Update registry
	registry.Plugins[plugin.Name] = InstalledPlugin{
		Name:         plugin.Name,
		Version:      plugin.Version,
		InstalledAt:  time.Now(),
		RequiredTier: plugin.RequiredTier,
		Enabled:      auth.IsTierSufficient(registry.CurrentTier, plugin.RequiredTier),
	}

	return nil
}

func (m *ProvisioningManager) handleTierChange(
	ctx context.Context,
	config *config.UnifiedConfig,
	registry *PluginRegistry,
	newTier string,
) error {
	oldTier := registry.CurrentTier

	// Update registry tier
	registry.CurrentTier = newTier

	// Update plugin enablement based on new tier
	for name, plugin := range registry.Plugins {
		wasEnabled := plugin.Enabled
		plugin.Enabled = auth.IsTierSufficient(newTier, plugin.RequiredTier)

		if wasEnabled && !plugin.Enabled {
			fmt.Printf("🔒 Disabling %s plugin (requires %s tier)\n", name, plugin.RequiredTier)
		} else if !wasEnabled && plugin.Enabled {
			fmt.Printf("🔓 Enabling %s plugin\n", name)
		}

		registry.Plugins[name] = plugin
	}

	// Save updated registry
	if err := m.registryStore.SaveRegistry(registry); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// If upgraded, check for new plugins
	if isTierUpgrade(oldTier, newTier) {
		fmt.Println("\n🎊 Checking for new plugins available in your tier...")
		return m.AutoProvisionPlugins(ctx, config)
	}

	return nil
}

func isTierUpgrade(oldTier, newTier string) bool {
	tiers := map[string]int{
		"Free":       1,
		"Pro":        2,
		"Enterprise": 3,
	}

	oldLevel := tiers[oldTier]
	newLevel := tiers[newTier]

	return newLevel > oldLevel
}
