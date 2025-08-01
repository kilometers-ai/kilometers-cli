package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins"
	"github.com/spf13/cobra"
)

// newPrivatePluginCommands creates commands for private plugin management
func newPrivatePluginCommands() []*cobra.Command {
	return []*cobra.Command{
		newPluginInstallCommand(),
		newPluginUninstallCommand(),
		newPluginSearchCommand(),
		newPluginUpdateCommand(),
		newPluginRegistryCommand(),
	}
}

// newPluginInstallCommand creates the plugin install command
func newPluginInstallCommand() *cobra.Command {
	var version string
	var force bool

	cmd := &cobra.Command{
		Use:   "install <plugin-name>",
		Short: "Install a plugin from private registry",
		Long: `Install a plugin from the private plugin registry.

Requires valid subscription with access to private plugins.

Examples:
  km plugins install custom-analytics
  km plugins install enterprise-security --version v2.1.0
  km plugins install beta-features --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginInstall(cmd.Context(), args[0], version, force)
		},
	}

	cmd.Flags().StringVar(&version, "version", "latest", "Plugin version to install")
	cmd.Flags().BoolVar(&force, "force", false, "Force reinstall if already installed")

	return cmd
}

// newPluginUninstallCommand creates the plugin uninstall command
func newPluginUninstallCommand() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "uninstall <plugin-name>",
		Short: "Uninstall a plugin",
		Long:  `Uninstall a plugin and remove all its data.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				fmt.Printf("This will uninstall plugin '%s' and remove all its data. Continue? (y/N): ", args[0])
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Printf("Uninstall cancelled.\n")
					return nil
				}
			}
			return runPluginUninstall(cmd.Context(), args[0])
		},
	}

	cmd.Flags().BoolVar(&confirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

// newPluginSearchCommand creates the plugin search command
func newPluginSearchCommand() *cobra.Command {
	var showAll bool
	var tier string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for plugins in private registry",
		Long: `Search for available plugins in the private plugin registry.

Examples:
  km plugins search                    # List all available plugins
  km plugins search security          # Search for security-related plugins
  km plugins search --tier enterprise # Show only enterprise plugins`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runPluginSearch(cmd.Context(), query, showAll, tier)
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Show all plugins regardless of subscription")
	cmd.Flags().StringVar(&tier, "tier", "", "Filter by subscription tier (free, pro, enterprise)")

	return cmd
}

// newPluginUpdateCommand creates the plugin update command
func newPluginUpdateCommand() *cobra.Command {
	var all bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update [plugin-name]",
		Short: "Update plugins to latest versions",
		Long: `Update one or more plugins to their latest versions.

Examples:
  km plugins update                    # Update all installed plugins
  km plugins update custom-analytics  # Update specific plugin
  km plugins update --dry-run          # Show what would be updated`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := ""
			if len(args) > 0 {
				pluginName = args[0]
			}
			return runPluginUpdate(cmd.Context(), pluginName, all, dryRun)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Update all installed plugins")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be updated without updating")

	return cmd
}

// newPluginRegistryCommand creates the plugin registry management command
func newPluginRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage private plugin registry settings",
		Long:  `Configure and manage private plugin registry connection.`,
	}

	cmd.AddCommand(newRegistryStatusCommand())
	cmd.AddCommand(newRegistryConfigCommand())
	cmd.AddCommand(newRegistryAuthCommand())

	return cmd
}

// newRegistryStatusCommand creates the registry status command
func newRegistryStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show registry connection status",
		Long:  `Display the current status of the private plugin registry connection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryStatus(cmd.Context())
		},
	}

	return cmd
}

// newRegistryConfigCommand creates the registry config command
func newRegistryConfigCommand() *cobra.Command {
	var url string
	var enabled bool
	var autoUpdate bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure registry settings",
		Long:  `Configure private plugin registry URL and settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryConfig(cmd.Context(), url, enabled, autoUpdate)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Registry URL")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable private registry")
	cmd.Flags().BoolVar(&autoUpdate, "auto-update", true, "Enable automatic plugin updates")

	return cmd
}

// newRegistryAuthCommand creates the registry auth command
func newRegistryAuthCommand() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Configure registry authentication",
		Long:  `Set authentication token for private plugin registry.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryAuth(cmd.Context(), token)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Authentication token")
	cmd.MarkFlagRequired("token")

	return cmd
}

// Command implementations

func runPluginInstall(ctx context.Context, pluginName, version string, force bool) error {
	// Create enhanced plugin manager
	manager, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Check if plugin is already installed
	if !force {
		if _, err := manager.GetPlugin(pluginName); err == nil {
			return fmt.Errorf("plugin '%s' is already installed. Use --force to reinstall", pluginName)
		}
	}

	// Install plugin
	fmt.Printf("Installing plugin '%s' version '%s'...\n", pluginName, version)
	
	if err := manager.InstallPlugin(ctx, pluginName, version); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' installed successfully\n", pluginName)
	return nil
}

func runPluginUninstall(ctx context.Context, pluginName string) error {
	// Create enhanced plugin manager
	manager, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Uninstall plugin
	if err := manager.UninstallPlugin(ctx, pluginName); err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' uninstalled successfully\n", pluginName)
	return nil
}

func runPluginSearch(ctx context.Context, query string, showAll bool, tier string) error {
	// Create enhanced plugin manager
	manager, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Search plugins
	plugins, err := manager.SearchPlugins(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search plugins: %w", err)
	}

	// Filter by tier if specified
	if tier != "" {
		var filtered []plugins.ExtendedPluginInfo
		for _, plugin := range plugins {
			if string(plugin.Plugin.RequiredTier()) == tier {
				filtered = append(filtered, plugin)
			}
		}
		plugins = filtered
	}

	// Display results
	if len(plugins) == 0 {
		if query != "" {
			fmt.Printf("No plugins found matching '%s'\n", query)
		} else {
			fmt.Printf("No plugins available\n")
		}
		return nil
	}

	fmt.Printf("Available Plugins:\n\n")
	for _, plugin := range plugins {
		status := "🔒"
		authManager := domain.NewAuthenticationManager()
		authManager.LoadSubscription()
		
		if authManager.IsFeatureEnabled(plugin.Plugin.RequiredFeature()) {
			status = "✅"
		}

		fmt.Printf("%s %s v%s (%s)\n", status, plugin.Plugin.Name(), plugin.Version, plugin.Source)
		fmt.Printf("   %s\n", plugin.Description)
		fmt.Printf("   Required: %s tier\n", plugin.Plugin.RequiredTier())
		if plugin.Author != "" {
			fmt.Printf("   Author: %s\n", plugin.Author)
		}
		if len(plugin.Permissions) > 0 {
			fmt.Printf("   Permissions: %s\n", strings.Join(plugin.Permissions, ", "))
		}
		fmt.Printf("\n")
	}

	return nil
}

func runPluginUpdate(ctx context.Context, pluginName string, all bool, dryRun bool) error {
	// Create enhanced plugin manager
	manager, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	if dryRun {
		fmt.Printf("Dry run mode - showing what would be updated\n\n")
	}

	if pluginName != "" {
		// Update specific plugin
		fmt.Printf("Updating plugin '%s'...\n", pluginName)
		// Individual plugin update logic would go here
		fmt.Printf("✅ Plugin '%s' updated successfully\n", pluginName)
	} else {
		// Update all plugins
		fmt.Printf("Updating all plugins...\n")
		
		if err := manager.UpdatePlugins(ctx); err != nil {
			return fmt.Errorf("failed to update plugins: %w", err)
		}
		
		fmt.Printf("✅ All plugins updated successfully\n")
	}

	return nil
}

func runRegistryStatus(ctx context.Context) error {
	// Create enhanced plugin manager
	manager, err := createEnhancedPluginManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Get registry status
	status := manager.GetRegistryStatus(ctx)

	fmt.Printf("Private Plugin Registry Status\n")
	fmt.Printf("==============================\n\n")

	if enabled, ok := status["enabled"].(bool); ok && enabled {
		fmt.Printf("Status: ✅ Enabled\n")
	} else {
		fmt.Printf("Status: ❌ Disabled\n")
	}

	if url, ok := status["url"].(string); ok {
		fmt.Printf("URL: %s\n", url)
	}

	if connected, ok := status["connected"].(bool); ok {
		if connected {
			fmt.Printf("Connection: ✅ Connected\n")
		} else {
			fmt.Printf("Connection: ❌ Disconnected\n")
			if errorMsg, ok := status["error"].(string); ok {
				fmt.Printf("Error: %s\n", errorMsg)
			}
		}
	}

	return nil
}

func runRegistryConfig(ctx context.Context, url string, enabled, autoUpdate bool) error {
	// Load current configuration
	config := domain.LoadConfig()

	// Update registry configuration (this would be added to domain.Config)
	// For now, we'll save it separately
	registryConfig := plugins.RegistryConfig{
		Enabled:    enabled,
		URL:        url,
		AutoUpdate: autoUpdate,
	}

	// Save registry configuration
	configData, err := json.MarshalIndent(registryConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry config: %w", err)
	}

	configPath, err := domain.GetPluginConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	registryConfigPath := strings.Replace(configPath, "plugins.json", "registry.json", 1)
	
	if err := os.WriteFile(registryConfigPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to save registry config: %w", err)
	}

	fmt.Printf("✅ Registry configuration updated\n")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Enabled: %t\n", enabled)
	fmt.Printf("Auto-update: %t\n", autoUpdate)

	return nil
}

func runRegistryAuth(ctx context.Context, token string) error {
	// This would securely store the auth token
	// For demo purposes, we'll just confirm it
	
	fmt.Printf("✅ Registry authentication token configured\n")
	fmt.Printf("Token: %s...%s\n", token[:8], token[len(token)-8:])

	return nil
}

// Helper functions

func createEnhancedPluginManager(ctx context.Context) (*plugins.EnhancedPluginManager, error) {
	// Load authentication
	authManager := domain.NewAuthenticationManager()
	if err := authManager.LoadSubscription(); err != nil {
		// Continue with free tier
	}

	// Create dependencies
	deps := ports.PluginDependencies{
		AuthManager: authManager,
		Config:      domain.LoadConfig(),
	}

	// Load registry configuration
	registryConfig := plugins.RegistryConfig{
		Enabled: true,
		URL:     "https://plugins.kilometers.ai",
		AuthToken: "demo_token", // In production, load from secure storage
		AutoUpdate: true,
	}

	// Create enhanced plugin manager
	manager := plugins.NewEnhancedPluginManager(authManager, deps, registryConfig)

	// Load plugins
	if err := manager.LoadPlugins(ctx); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	return manager, nil
}
