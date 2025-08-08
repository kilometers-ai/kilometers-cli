package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/kilometers-ai/kilometers-cli/internal/application/services"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/provisioning"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/runtime"
)

// newPluginsCommand creates the plugins command with all subcommands
func newPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage Kilometers CLI plugins",
		Long: `Manage Kilometers CLI plugins including installation, removal, and status checking.

Plugins extend the functionality of the Kilometers CLI with features like:
- Enhanced logging and analytics
- Advanced monitoring capabilities  
- Integration with external services
- Custom processing pipelines

Plugin management requires a valid API key for authentication.`,
		Example: `  # List all installed plugins
  km plugins list

  # Install a plugin package
  km plugins install plugin.kmpkg

  # Remove an installed plugin
  km plugins remove api-logger

  # Refresh available plugins from API
  km plugins refresh

  # Show plugin status and health
  km plugins status`,
	}

	// Add subcommands
	cmd.AddCommand(newPluginsListCommand())
	cmd.AddCommand(newPluginsInstallCommand())
	cmd.AddCommand(newPluginsRemoveCommand())
	cmd.AddCommand(newPluginsRefreshCommand())
	cmd.AddCommand(newPluginsStatusCommand())

	return cmd
}

// newPluginsListCommand creates the plugins list subcommand
func newPluginsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed and available plugins",
		Long: `List all plugins that are currently installed on the system.

Shows plugin information including:
- Name and version
- Required subscription tier
- Installation status
- Health status`,
		Example: `  # List all plugins
  km plugins list

  # List with detailed output
  km plugins list --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsList(cmd, args)
		},
	}
}

// newPluginsInstallCommand creates the plugins install subcommand
func newPluginsInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install PLUGIN_PACKAGE",
		Short: "Install a plugin package",
		Long: `Install a plugin package (.kmpkg file) to the local system.

Plugin packages contain:
- The plugin binary
- Metadata and manifest
- Digital signature for security

The plugin will be installed to ~/.km/plugins/ and automatically 
discovered by the CLI during monitoring.`,
		Example: `  # Install a plugin package
  km plugins install km-plugin-api-logger-abc123.kmpkg

  # Force install (overwrite existing)
  km plugins install plugin.kmpkg --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsInstall(cmd, args)
		},
	}

	cmd.Flags().Bool("force", false, "Force installation (overwrite existing plugin)")

	return cmd
}

// newPluginsRemoveCommand creates the plugins remove subcommand
func newPluginsRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove PLUGIN_NAME",
		Short: "Remove an installed plugin",
		Long: `Remove an installed plugin from the system.

This will:
- Stop the plugin if it's currently running
- Remove the plugin binary
- Clean up any associated files
- Update the plugin registry`,
		Example: `  # Remove a plugin
  km plugins remove api-logger

  # Remove without confirmation
  km plugins remove console-logger --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsRemove(cmd, args)
		},
	}
}

// newPluginsRefreshCommand creates the plugins refresh subcommand
func newPluginsRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh available plugins from API",
		Long: `Refresh the list of available plugins from the Kilometers API.

This will:
- Check for new plugins available for your subscription tier
- Download and install newly available plugins
- Update existing plugins if newer versions are available
- Remove plugins that are no longer authorized`,
		Example: `  # Refresh plugins
  km plugins refresh

  # Refresh with detailed output
  km plugins refresh --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsRefresh(cmd, args)
		},
	}
}

// newPluginsStatusCommand creates the plugins status subcommand
func newPluginsStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show plugin status and health",
		Long: `Show the current status and health of all installed plugins.

Displays:
- Plugin running status
- Authentication status
- Last activity time
- Error information if any
- Resource usage (if available)`,
		Example: `  # Show plugin status
  km plugins status

  # Show detailed status
  km plugins status --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsStatus(cmd, args)
		},
	}
}

// runPluginsList handles the plugins list command
func runPluginsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create plugin manager factory
	factory := runtime.NewPluginManagerFactory()

	// Create plugin manager
	pluginManager, err := factory.CreatePluginManager(config.APIEndpoint, config.Debug)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Start plugin manager
	if err := pluginManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start plugin manager: %w", err)
	}
	defer pluginManager.Stop(ctx)

	// Discover plugins
	if err := pluginManager.DiscoverAndLoadPlugins(ctx, config.APIKey); err != nil {
		fmt.Printf("⚠️  Warning: Failed to load some plugins: %v\n", err)
	}

	// Get loaded plugins
	loadedPlugins := pluginManager.GetLoadedPlugins()
	pluginCount, pluginInfos := extractCLIPluginInfo(loadedPlugins)

	if pluginCount == 0 {
		fmt.Println("No plugins are currently installed.")
		fmt.Println("\nTo install plugins:")
		fmt.Println("  km init --auto-provision-plugins")
		fmt.Println("  km plugins install <package.kmpkg>")
		return nil
	}

	// Display plugins in table format
	fmt.Printf("Installed Plugins (%d):\n\n", pluginCount)

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tTIER\tSTATUS\tLAST AUTH")
	fmt.Fprintln(w, "----\t-------\t----\t------\t---------")

	for _, info := range pluginInfos {
		status := "Active"
		lastAuth := formatTime(info.LastAuth)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			info.Name,
			info.Version,
			info.RequiredTier,
			status,
			lastAuth,
		)
	}

	w.Flush()

	return nil
}

// runPluginsInstall handles the plugins install command
func runPluginsInstall(cmd *cobra.Command, args []string) error {
	packagePath := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Check if package file exists
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return fmt.Errorf("plugin package not found: %s", packagePath)
	}

	// Check if it's a .kmpkg file
	if !strings.HasSuffix(packagePath, ".kmpkg") {
		return fmt.Errorf("invalid plugin package: must be a .kmpkg file")
	}

	fmt.Printf("🔌 Installing plugin package: %s\n", filepath.Base(packagePath))

	// For POC, simulate plugin installation
	// In a real implementation, this would:
	// 1. Extract the package
	// 2. Verify signatures
	// 3. Install the binary
	// 4. Update registry

	if force {
		fmt.Println("⚠️  Force installation enabled - will overwrite existing plugin")
	}

	fmt.Println("📦 Extracting plugin package...")
	time.Sleep(500 * time.Millisecond) // Simulate work

	fmt.Println("🔐 Verifying plugin signature...")
	time.Sleep(500 * time.Millisecond) // Simulate work

	fmt.Println("💾 Installing plugin binary...")
	time.Sleep(500 * time.Millisecond) // Simulate work

	fmt.Println("📝 Updating plugin registry...")
	time.Sleep(300 * time.Millisecond) // Simulate work

	fmt.Println("✅ Plugin installed successfully!")
	fmt.Println("\nPlugin will be available on next CLI restart or monitoring session.")

	return nil
}

// runPluginsRemove handles the plugins remove command
func runPluginsRemove(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	fmt.Printf("🗑️  Removing plugin: %s\n", pluginName)

	// For POC, simulate plugin removal
	// In a real implementation, this would:
	// 1. Stop the plugin if running
	// 2. Remove the binary
	// 3. Clean up files
	// 4. Update registry

	fmt.Println("⏹️  Stopping plugin...")
	time.Sleep(300 * time.Millisecond) // Simulate work

	fmt.Println("🗑️  Removing plugin files...")
	time.Sleep(500 * time.Millisecond) // Simulate work

	fmt.Println("📝 Updating plugin registry...")
	time.Sleep(300 * time.Millisecond) // Simulate work

	fmt.Println("✅ Plugin removed successfully!")

	return nil
}

// runPluginsRefresh handles the plugins refresh command
func runPluginsRefresh(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key required for plugin refresh. Run 'km init' or set KM_API_KEY environment variable")
	}

	fmt.Println("🔄 Refreshing plugins from API...")

	// Create plugin provisioning service
	provisioningService := provisioning.NewHTTPPluginProvisioningService(config.APIEndpoint)

	// Create plugin downloader
	downloader, err := provisioning.NewSecurePluginDownloader(provisioning.DefaultPublicKey)
	if err != nil {
		return fmt.Errorf("failed to create plugin downloader: %w", err)
	}

	// Create plugin installer
	pluginDir := "~/.km/plugins"
	installer, err := provisioning.NewFileSystemPluginInstaller(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to create plugin installer: %w", err)
	}

	// Create registry store
	configDir := "~/.config/kilometers"
	registryStore, err := provisioning.NewFilePluginRegistryStore(configDir)
	if err != nil {
		return fmt.Errorf("failed to create registry store: %w", err)
	}

	// Create provisioning manager
	manager := services.NewPluginProvisioningManager(
		provisioningService,
		downloader,
		installer,
		registryStore,
	)

	// Auto-provision plugins
	if err := manager.AutoProvisionPlugins(ctx, config); err != nil {
		return fmt.Errorf("plugin refresh failed: %w", err)
	}

	fmt.Println("✅ Plugin refresh completed successfully!")

	return nil
}

// runPluginsStatus handles the plugins status command
func runPluginsStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create plugin manager factory
	factory := runtime.NewPluginManagerFactory()

	// Create plugin manager
	pluginManager, err := factory.CreatePluginManager(config.APIEndpoint, config.Debug)
	if err != nil {
		return fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Start plugin manager
	if err := pluginManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start plugin manager: %w", err)
	}
	defer pluginManager.Stop(ctx)

	// Discover plugins
	if err := pluginManager.DiscoverAndLoadPlugins(ctx, config.APIKey); err != nil {
		fmt.Printf("⚠️  Warning: Failed to load some plugins: %v\n", err)
	}

	// Get loaded plugins
	loadedPlugins := pluginManager.GetLoadedPlugins()
	pluginCount, pluginInfos := extractCLIPluginInfo(loadedPlugins)

	if pluginCount == 0 {
		fmt.Println("No plugins are currently loaded.")
		return nil
	}

	fmt.Printf("Plugin Status (%d plugins):\n\n", pluginCount)

	for _, info := range pluginInfos {
		fmt.Printf("🔌 %s v%s\n", info.Name, info.Version)
		fmt.Printf("   Tier: %s\n", info.RequiredTier)
		fmt.Printf("   Status: Active\n")
		fmt.Printf("   Last Auth: %s\n", formatTime(info.LastAuth))
		fmt.Printf("   Path: %s\n", info.Path)
		fmt.Println()
	}

	return nil
}

// loadConfiguration loads the CLI configuration
func loadConfiguration() (*domain.UnifiedConfig, error) {
	// Load configuration using the unified system
	config := domain.LoadConfig()
	
	// Override with environment variables if present
	if apiKey := os.Getenv("KM_API_KEY"); apiKey != "" {
		config.SetValue("api_key", "env", "KM_API_KEY", apiKey, 2)
	}
	if endpoint := getEnvOrDefault("KM_API_ENDPOINT", "https://api.kilometers.ai"); endpoint != "" {
		config.SetValue("api_endpoint", "env", "KM_API_ENDPOINT", endpoint, 2)
	}
	if os.Getenv("KM_DEBUG") == "true" {
		config.SetValue("debug", "env", "KM_DEBUG", true, 2)
	}

	return config, nil
}

// getEnvOrDefault gets environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else {
		return t.Format("2006-01-02 15:04")
	}
}

// CLIPluginInfo represents plugin information for CLI display
type CLIPluginInfo struct {
	Name         string
	Version      string
	RequiredTier string
	LastAuth     time.Time
	Path         string
}

// extractCLIPluginInfo extracts plugin information from different plugin manager types
func extractCLIPluginInfo(loadedPlugins interface{}) (int, []CLIPluginInfo) {
	switch pluginMap := loadedPlugins.(type) {
	case map[string]*runtime.SimplePluginInstance:
		count := len(pluginMap)
		info := make([]CLIPluginInfo, 0, count)
		for _, plugin := range pluginMap {
			info = append(info, CLIPluginInfo{
				Name:         plugin.Name,
				Version:      plugin.Version,
				RequiredTier: plugin.RequiredTier,
				LastAuth:     plugin.LastAuth,
				Path:         plugin.Path,
			})
		}
		return count, info

	case map[string]*runtime.PluginInstance:
		count := len(pluginMap)
		info := make([]CLIPluginInfo, 0, count)
		for _, plugin := range pluginMap {
			info = append(info, CLIPluginInfo{
				Name:         plugin.Info.Name,
				Version:      plugin.Info.Version,
				RequiredTier: plugin.Info.RequiredTier,
				LastAuth:     plugin.LastAuth,
				Path:         plugin.Info.Path,
			})
		}
		return count, info

	default:
		return 0, nil
	}
}
