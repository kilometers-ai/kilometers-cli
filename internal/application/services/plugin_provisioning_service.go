package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// PluginProvisioningManager orchestrates plugin provisioning
type PluginProvisioningManager struct {
	provisioningService plugins.PluginProvisioningService
	downloader          plugins.PluginDownloader
	installer           plugins.PluginInstaller
	registryStore       plugins.PluginRegistryStore
}

// NewPluginProvisioningManager creates a new plugin provisioning manager
func NewPluginProvisioningManager(
	provisioningService plugins.PluginProvisioningService,
	downloader plugins.PluginDownloader,
	installer plugins.PluginInstaller,
	registryStore plugins.PluginRegistryStore,
) *PluginProvisioningManager {
	return &PluginProvisioningManager{
		provisioningService: provisioningService,
		downloader:          downloader,
		installer:           installer,
		registryStore:       registryStore,
	}
}

// AutoProvisionPlugins automatically provisions and installs plugins for the customer
func (m *PluginProvisioningManager) AutoProvisionPlugins(ctx context.Context, config *domain.Config) error {
	if config.ApiKey == "" {
		return fmt.Errorf("API key is required for plugin provisioning")
	}

	// Request plugin provisioning from API
	provisionResp, err := m.provisioningService.ProvisionPlugins(ctx, config.ApiKey)
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
func (m *PluginProvisioningManager) RefreshPlugins(ctx context.Context, config *domain.Config) error {
	// Get current subscription status
	currentTier, err := m.provisioningService.GetSubscriptionStatus(ctx, config.ApiKey)
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

func (m *PluginProvisioningManager) loadOrCreateRegistry(customerID, tier string) (*domain.PluginRegistry, error) {
	registry, err := m.registryStore.LoadRegistry()
	if err != nil {
		// Create new registry if none exists
		registry = &domain.PluginRegistry{
			CustomerID:  customerID,
			CurrentTier: tier,
			Plugins:     make(map[string]domain.InstalledPlugin),
		}
	}

	registry.CustomerID = customerID
	registry.CurrentTier = tier

	return registry, nil
}

func (m *PluginProvisioningManager) downloadAndInstallPlugin(
	ctx context.Context,
	plugin domain.ProvisionedPlugin,
	registry *domain.PluginRegistry,
) error {
	// Check if already installed with same version
	if installed, exists := registry.Plugins[plugin.Name]; exists && installed.Version == plugin.Version {
		fmt.Printf("⏭️  %s already up to date (v%s)\n", plugin.Name, plugin.Version)
		return nil
	}

	fmt.Printf("📦 Downloading %s plugin...\n", plugin.Name)

	// Download plugin
	reader, err := m.downloader.DownloadPlugin(ctx, plugin)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer reader.Close()

	// Install plugin
	if err := m.installer.InstallPlugin(ctx, reader, plugin); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	// Update registry
	registry.Plugins[plugin.Name] = domain.InstalledPlugin{
		Name:         plugin.Name,
		Version:      plugin.Version,
		InstalledAt:  time.Now(),
		RequiredTier: plugin.RequiredTier,
		Enabled:      domain.IsTierSufficient(registry.CurrentTier, plugin.RequiredTier),
	}

	return nil
}

func (m *PluginProvisioningManager) handleTierChange(
	ctx context.Context,
	config *domain.Config,
	registry *domain.PluginRegistry,
	newTier string,
) error {
	oldTier := registry.CurrentTier

	// Update registry tier
	registry.CurrentTier = newTier

	// Update plugin enablement based on new tier
	for name, plugin := range registry.Plugins {
		wasEnabled := plugin.Enabled
		plugin.Enabled = domain.IsTierSufficient(newTier, plugin.RequiredTier)

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
