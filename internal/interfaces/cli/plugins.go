package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
	"github.com/spf13/cobra"
)

// newPluginsCommand creates plugins command with API support
func newPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage Kilometers CLI plugins",
		Long: `Manage Kilometers CLI plugins with API integration.

Download, install, update, and remove plugins from the Kilometers plugin repository.
Requires a valid API key for accessing the plugin manifest and downloads.`,
		Example: `  # List available plugins
  km plugins list

  # Install a plugin
  km plugins install console-logger

  # Update all plugins
  km plugins update

  # Update specific plugin
  km plugins update console-logger

  # Remove a plugin
  km plugins remove console-logger`,
	}

	// Add enhanced subcommands
	cmd.AddCommand(newPluginsListCommand())
	cmd.AddCommand(newPluginsInstallCommand())
	cmd.AddCommand(newPluginsUpdateCommand())
	cmd.AddCommand(newPluginsRemoveCommand())

	return cmd
}

// newPluginsListCommand creates the enhanced list command
func newPluginsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Long:  `List all plugins available for your subscription tier from the API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsList(cmd, args)
		},
	}
}

// newPluginsInstallCommand creates the enhanced install command
func newPluginsInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install [plugin-name]",
		Short: "Install a plugin",
		Long:  `Download and install a plugin from the kilometers plugin repository.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsInstall(cmd, args)
		},
	}
}

// newPluginsUpdateCommand creates the enhanced update command
func newPluginsUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update [plugin-name]",
		Short: "Update plugins",
		Long:  `Update one or all installed plugins to their latest versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsUpdate(cmd, args)
		},
	}
}

// newPluginsRemoveCommand creates the enhanced remove command
func newPluginsRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [plugin-name]",
		Short: "Remove a plugin",
		Long:  `Remove an installed plugin from the system.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsRemove(cmd, args)
		},
	}
}

// runPluginsList handles the enhanced plugins list command
func runPluginsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	loader, storage, err := configpkg.CreateConfigServiceFromDefaults()
	if err != nil {
		return fmt.Errorf("failed to create config service: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)

	config, err := configService.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if !config.HasAPIKey() {
		fmt.Println("⚠️  API key required to list available plugins.")
		fmt.Println("Run 'km auth login' to configure your API key.")
		return nil
	}

	// Create plugin manager
	pluginsDir := config.PluginsDir
	if pluginsDir == "" {
		pluginsDir = "~/.km/plugins"
	}

	manager, err := plugins.NewPluginManager(
		&plugins.PluginManagerConfig{
			PluginDirectories: []string{pluginsDir},
			ApiEndpoint:       config.APIEndpoint,
			Debug:             config.Debug,
		},
		nil, nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// List available plugins from API
	fmt.Println("🔍 Fetching available plugins...")
	availablePlugins, err := manager.ListAvailablePlugins(ctx)
	if err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	if len(availablePlugins) == 0 {
		fmt.Println("No plugins are available for your subscription tier.")
		return nil
	}

	// Display plugins in table format
	fmt.Printf("\nAvailable Plugins (%d):\n\n", len(availablePlugins))

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tTIER\tSTATUS\tSIZE")
	fmt.Fprintln(w, "----\t-------\t----\t------\t----")

	for _, plugin := range availablePlugins {
		status := "Available"
		// Check if installed
		if installed, _ := manager.IsPluginInstalled(plugin.Name); installed {
			status = "Installed"
		}

		size := formatSize(plugin.Size)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			plugin.Name,
			plugin.Version,
			plugin.Tier,
			status,
			size,
		)
	}

	w.Flush()
	fmt.Println("\nTo install a plugin, run: km plugins install <plugin-name>")

	return nil
}

// runPluginsInstall handles the enhanced plugins install command
func runPluginsInstall(cmd *cobra.Command, args []string) error {
	// Setup plugins directory if no arguments
	if len(args) == 0 {
		return setupPluginsDirectory()
	}

	// Install specific plugin
	pluginName := args[0]
	ctx := context.Background()

	// Load configuration
	loader, storage, err := configpkg.CreateConfigServiceFromDefaults()
	if err != nil {
		return fmt.Errorf("failed to create config service: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)

	config, err := configService.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if !config.HasAPIKey() {
		return fmt.Errorf("API key required. Run 'km auth login' first")
	}

	// Create enhanced plugin manager
	pluginsDir := config.PluginsDir
	if pluginsDir == "" {
		pluginsDir = "~/.km/plugins"
	}

	manager, err := plugins.NewPluginManager(
		&plugins.PluginManagerConfig{
			PluginDirectories: []string{pluginsDir},
			ApiEndpoint:       config.APIEndpoint,
			Debug:             config.Debug,
		},
		nil, nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	fmt.Printf("📦 Installing plugin: %s\n", pluginName)

	// Install the plugin
	if err := manager.InstallPlugin(ctx, pluginName, config.APIKey); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Printf("✅ Successfully installed plugin: %s\n", pluginName)
	return nil
}

// runPluginsUpdate handles the enhanced plugins update command
func runPluginsUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	loader, storage, err := configpkg.CreateConfigServiceFromDefaults()
	if err != nil {
		return fmt.Errorf("failed to create config service: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)

	config, err := configService.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if !config.HasAPIKey() {
		return fmt.Errorf("API key required. Run 'km auth login' first")
	}

	pluginsDir := config.PluginsDir
	if pluginsDir == "" {
		pluginsDir = "~/.km/plugins"
	}

	manager, err := plugins.NewPluginManager(
		&plugins.PluginManagerConfig{
			PluginDirectories: []string{pluginsDir},
			ApiEndpoint:       config.APIEndpoint,
			Debug:             config.Debug,
		},
		nil, nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	if len(args) > 0 {
		// Update specific plugin
		pluginName := args[0]
		fmt.Printf("🔄 Updating plugin: %s\n", pluginName)

		if err := manager.UpdatePlugin(ctx, pluginName, config.APIKey); err != nil {
			return fmt.Errorf("failed to update plugin: %w", err)
		}

		fmt.Printf("✅ Successfully updated plugin: %s\n", pluginName)
	} else {
		// Check for updates for all plugins
		fmt.Println("🔍 Checking for plugin updates...")
		updates, err := manager.CheckForUpdates(ctx)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		if len(updates) == 0 {
			fmt.Println("✅ All plugins are up to date")
			return nil
		}

		fmt.Printf("📦 Found %d plugin update(s)\n", len(updates))
		for _, update := range updates {
			fmt.Printf("Updating %s from %s to %s\n",
				update.Name, update.CurrentVersion, update.NewVersion)

			if err := manager.UpdatePlugin(ctx, update.Name, config.APIKey); err != nil {
				fmt.Printf("⚠️  Failed to update %s: %v\n", update.Name, err)
				continue
			}

			fmt.Printf("✅ Updated %s\n", update.Name)
		}
	}

	return nil
}

// runPluginsRemove handles the enhanced plugins remove command
func runPluginsRemove(cmd *cobra.Command, args []string) error {
	pluginName := args[0]
	ctx := context.Background()

	// Load configuration
	loader, storage, err := configpkg.CreateConfigServiceFromDefaults()
	if err != nil {
		return fmt.Errorf("failed to create config service: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)

	config, err := configService.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create enhanced plugin manager
	pluginsDir := config.PluginsDir
	if pluginsDir == "" {
		pluginsDir = "~/.km/plugins"
	}

	manager, err := plugins.NewPluginManager(
		&plugins.PluginManagerConfig{
			PluginDirectories: []string{pluginsDir},
			ApiEndpoint:       config.APIEndpoint,
			Debug:             config.Debug,
		},
		nil, nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	fmt.Printf("🗑️  Removing plugin: %s\n", pluginName)

	// Remove the plugin
	if err := manager.UninstallPlugin(ctx, pluginName); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	fmt.Printf("✅ Successfully removed plugin: %s\n", pluginName)
	return nil
}

// setupPluginsDirectory creates the plugins directory
func setupPluginsDirectory() error {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Define plugins directory path
	pluginsDir := filepath.Join(homeDir, ".km", "plugins")

	// Check if directory exists
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		fmt.Printf("🔍 Plugins directory not found at: %s\n", pluginsDir)
		fmt.Println("📁 Creating plugins directory...")

		// Create directory with appropriate permissions
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			return fmt.Errorf("failed to create plugins directory: %w", err)
		}

		fmt.Printf("✅ Successfully created plugins directory at: %s\n", pluginsDir)
	} else if err != nil {
		return fmt.Errorf("failed to check plugins directory: %w", err)
	} else {
		fmt.Printf("✅ Plugins directory already exists at: %s\n", pluginsDir)
	}

	fmt.Println("🔌 Plugins directory is ready for use")
	return nil
}

// formatSize formats file size for display
func formatSize(size int64) string {
	if size == 0 {
		return "Unknown"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}
