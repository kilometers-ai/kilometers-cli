package application

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
	pluginports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugin"
)

// PluginProvisioningService orchestrates plugin provisioning and installation
type PluginProvisioningService struct {
	provisioning pluginports.ProvisioningService
	installer    pluginports.PluginInstaller
	registry     pluginports.PluginRegistry
}

// NewPluginProvisioningService creates a new plugin provisioning service
func NewPluginProvisioningService(
	provisioning pluginports.ProvisioningService,
	installer pluginports.PluginInstaller,
	registry pluginports.PluginRegistry,
) *PluginProvisioningService {
	return &PluginProvisioningService{
		provisioning: provisioning,
		installer:    installer,
		registry:     registry,
	}
}

// InitConfig represents initialization configuration options
type InitConfig struct {
	APIKey        string
	APIEndpoint   string
	AutoProvision bool
	Interactive   bool
	Force         bool
}

// InitResult represents the result of initialization
type InitResult struct {
	APIKeyValidated  bool
	Subscription     *plugindomain.Subscription
	PluginsInstalled int
	PluginsAvailable int
	Errors           []error
}

// InitializeWithValidation performs complete initialization with API key validation and plugin provisioning
func (s *PluginProvisioningService) InitializeWithValidation(ctx context.Context, config InitConfig) (*InitResult, error) {
	result := &InitResult{}
	reader := bufio.NewReader(os.Stdin)

	// Step 1: API Key validation
	apiKey := config.APIKey
	if apiKey == "" && config.Interactive {
		fmt.Println("🔑 API Key Setup")
		fmt.Println("================")
		fmt.Print("Enter your Kilometers API key (or press Enter to skip): ")
		input, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(input)
	}

	if apiKey == "" {
		fmt.Println("\n⚠️  No API key provided. You'll be using the Free tier.")
		fmt.Println("   To unlock Pro features, run: km auth login --api-key YOUR_KEY")
		return result, nil
	}

	// Step 2: Validate API key and get subscription
	fmt.Printf("\n🔍 Validating API key...\n")
	validation, err := s.provisioning.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}

	if !validation.IsValid {
		fmt.Printf("❌ Invalid API key: %s\n", validation.Message)
		return result, fmt.Errorf("invalid API key")
	}

	result.APIKeyValidated = true
	result.Subscription = validation.Subscription

	fmt.Printf("✅ API key validated!\n")
	fmt.Printf("   Customer: %s\n", validation.Subscription.CustomerName)
	fmt.Printf("   Tier: %s\n", string(validation.Subscription.Tier))
	fmt.Printf("   Features: %s\n", strings.Join(validation.Subscription.AvailableFeatures, ", "))

	// Step 3: Check available plugins
	fmt.Printf("\n🔌 Checking available plugins...\n")
	availablePlugins, err := s.provisioning.GetAvailablePlugins(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get available plugins: %w", err)
	}

	result.PluginsAvailable = len(availablePlugins)

	// Step 4: Check installed plugins
	installed, err := s.installer.GetInstalled(ctx)
	if err != nil {
		fmt.Printf("⚠️  Could not check installed plugins: %v\n", err)
	}

	// Step 5: Determine what needs installation/update
	var toInstall []plugindomain.Plugin
	var toUpdate []plugindomain.PluginInstallStatus

	installedMap := make(map[string]plugindomain.PluginInstallStatus)
	for _, inst := range installed {
		installedMap[inst.Plugin.Name] = inst
	}

	for _, plugin := range availablePlugins {
		if !validation.Subscription.CanAccessPlugin(plugin) {
			continue // Skip plugins not available for this tier
		}

		if inst, exists := installedMap[plugin.Name]; exists {
			if inst.NeedsUpdate {
				toUpdate = append(toUpdate, inst)
			}
		} else {
			toInstall = append(toInstall, plugin)
		}
	}

	// Step 6: Display status and prompt for action
	fmt.Printf("\n📊 Plugin Status:\n")
	fmt.Printf("   Available: %d plugins\n", len(availablePlugins))
	fmt.Printf("   Installed: %d plugins\n", len(installed))
	fmt.Printf("   To Install: %d plugins\n", len(toInstall))
	fmt.Printf("   To Update: %d plugins\n", len(toUpdate))

	if len(toInstall) == 0 && len(toUpdate) == 0 {
		fmt.Println("\n✅ All plugins are up to date!")
		return result, nil
	}

	// Step 7: Interactive prompt for installation
	shouldInstall := config.AutoProvision
	if !shouldInstall && config.Interactive && (len(toInstall) > 0 || len(toUpdate) > 0) {
		fmt.Println("\n📦 Plugin Installation")

		if len(toInstall) > 0 {
			fmt.Println("\nPlugins to install:")
			for _, p := range toInstall {
				fmt.Printf("  • %s (v%s) - %s\n", p.Name, p.Version, p.Description)
			}
		}

		if len(toUpdate) > 0 {
			fmt.Println("\nPlugins to update:")
			for _, p := range toUpdate {
				fmt.Printf("  • %s: %s → %s\n", p.Plugin.Name, p.CurrentVersion, p.Plugin.Version)
			}
		}

		fmt.Print("\nWould you like to install/update these plugins? [Y/n]: ")
		input, _ := reader.ReadString('\n')
		response := strings.TrimSpace(strings.ToLower(input))
		shouldInstall = response == "" || response == "y" || response == "yes"
	}

	// Step 8: Install/update plugins if confirmed
	if shouldInstall {
		fmt.Println("\n🚀 Installing plugins...")

		// Install new plugins
		for _, plugin := range toInstall {
			fmt.Printf("  📥 Installing %s...", plugin.Name)

			data, err := s.provisioning.DownloadPlugin(ctx, apiKey, plugin.Name)
			if err != nil {
				fmt.Printf(" ❌ Failed: %v\n", err)
				result.Errors = append(result.Errors, err)
				continue
			}

			if err := s.installer.Install(ctx, plugin, data); err != nil {
				fmt.Printf(" ❌ Failed: %v\n", err)
				result.Errors = append(result.Errors, err)
				continue
			}

			if err := s.registry.AddPlugin(ctx, plugin, ""); err != nil {
				fmt.Printf(" ⚠️  Registry update failed: %v\n", err)
			}

			fmt.Printf(" ✅ Done!\n")
			result.PluginsInstalled++
		}

		// Update existing plugins
		for _, status := range toUpdate {
			fmt.Printf("  🔄 Updating %s...", status.Plugin.Name)

			data, err := s.provisioning.DownloadPlugin(ctx, apiKey, status.Plugin.Name)
			if err != nil {
				fmt.Printf(" ❌ Failed: %v\n", err)
				result.Errors = append(result.Errors, err)
				continue
			}

			// Find the updated plugin info
			var updatedPlugin *plugindomain.Plugin
			for _, p := range availablePlugins {
				if p.Name == status.Plugin.Name {
					updatedPlugin = &p
					break
				}
			}

			if updatedPlugin == nil {
				fmt.Printf(" ❌ Failed: plugin not found in available list\n")
				continue
			}

			if err := s.installer.Install(ctx, *updatedPlugin, data); err != nil {
				fmt.Printf(" ❌ Failed: %v\n", err)
				result.Errors = append(result.Errors, err)
				continue
			}

			fmt.Printf(" ✅ Done!\n")
			result.PluginsInstalled++
		}

		if result.PluginsInstalled > 0 {
			fmt.Printf("\n✅ Successfully installed/updated %d plugin(s)!\n", result.PluginsInstalled)
		}

		if len(result.Errors) > 0 {
			fmt.Printf("⚠️  %d plugin(s) failed to install\n", len(result.Errors))
		}
	} else {
		fmt.Println("\n⏭️  Skipping plugin installation.")
		fmt.Println("   You can install plugins later with: km plugins install")
	}

	return result, nil
}

// CheckEntitlements checks what plugins a user is entitled to
func (s *PluginProvisioningService) CheckEntitlements(ctx context.Context, apiKey string) (*plugindomain.ProvisioningResult, error) {
	// Validate API key first
	validation, err := s.provisioning.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("invalid API key: %s", validation.Message)
	}

	// Get available plugins
	availablePlugins, err := s.provisioning.GetAvailablePlugins(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	// Filter plugins by entitlement
	var entitledPlugins []plugindomain.Plugin
	for _, plugin := range availablePlugins {
		if validation.Subscription.CanAccessPlugin(plugin) {
			entitledPlugins = append(entitledPlugins, plugin)
		}
	}

	return &plugindomain.ProvisioningResult{
		Subscription:     *validation.Subscription,
		AvailablePlugins: entitledPlugins,
	}, nil
}
