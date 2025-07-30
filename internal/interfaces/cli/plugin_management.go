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

// newPluginManagementCommands creates enhanced plugin management commands
func newPluginManagementCommands() []*cobra.Command {
	return []*cobra.Command{
		newPluginEnableCommand(),
		newPluginDisableCommand(),
		newPluginConfigureCommand(),
		newPluginExportCommand(),
		newPluginImportCommand(),
		newPluginResetCommand(),
	}
}

// newPluginEnableCommand creates the plugin enable command
func newPluginEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <plugin-name>",
		Short: "Enable a specific plugin",
		Long:  `Enable a plugin to include it in monitoring operations.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginEnable(cmd.Context(), args[0])
		},
	}

	return cmd
}

// newPluginDisableCommand creates the plugin disable command
func newPluginDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <plugin-name>",
		Short: "Disable a specific plugin",
		Long:  `Disable a plugin to exclude it from monitoring operations.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginDisable(cmd.Context(), args[0])
		},
	}

	return cmd
}

// newPluginConfigureCommand creates the plugin configure command
func newPluginConfigureCommand() *cobra.Command {
	var configData string
	var configFile string

	cmd := &cobra.Command{
		Use:   "configure <plugin-name>",
		Short: "Configure plugin settings",
		Long: `Configure specific settings for a plugin.

You can provide configuration either inline with --data or from a file with --file.

Examples:
  km plugins configure advanced-filters --data '{"threshold":0.8,"patterns":[".*secret.*"]}'
  km plugins configure poison-detection --file ./security-config.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginConfigure(cmd.Context(), args[0], configData, configFile)
		},
	}

	cmd.Flags().StringVar(&configData, "data", "", "Configuration data as JSON string")
	cmd.Flags().StringVar(&configFile, "file", "", "Configuration file path")

	return cmd
}

// newPluginExportCommand creates the plugin export command
func newPluginExportCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export <plugin-name>",
		Short: "Export plugin configuration",
		Long:  `Export plugin configuration to a file for sharing or backup.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginExport(cmd.Context(), args[0], outputFile)
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path (default: stdout)")

	return cmd
}

// newPluginImportCommand creates the plugin import command
func newPluginImportCommand() *cobra.Command {
	var inputFile string

	cmd := &cobra.Command{
		Use:   "import <plugin-name>",
		Short: "Import plugin configuration",
		Long:  `Import plugin configuration from a file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginImport(cmd.Context(), args[0], inputFile)
		},
	}

	cmd.Flags().StringVar(&inputFile, "file", "", "Input file path")
	cmd.MarkFlagRequired("file")

	return cmd
}

// newPluginResetCommand creates the plugin reset command
func newPluginResetCommand() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "reset <plugin-name>",
		Short: "Reset plugin to default configuration",
		Long:  `Reset a plugin to its default configuration, removing all custom settings.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				fmt.Printf("This will reset all configuration for plugin '%s'. Continue? (y/N): ", args[0])
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Printf("Reset cancelled.\n")
					return nil
				}
			}
			return runPluginReset(cmd.Context(), args[0])
		},
	}

	cmd.Flags().BoolVar(&confirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

// Command implementations

func runPluginEnable(ctx context.Context, pluginName string) error {
	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Check if plugin exists and user has access
	if err := validatePluginAccess(ctx, pluginName); err != nil {
		return err
	}

	// Enable the plugin
	configs.EnablePlugin(pluginName)

	// Save configurations
	if err := domain.SavePluginConfigs(configs); err != nil {
		return fmt.Errorf("failed to save plugin configs: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' enabled successfully\n", pluginName)
	return nil
}

func runPluginDisable(ctx context.Context, pluginName string) error {
	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Disable the plugin
	configs.DisablePlugin(pluginName)

	// Save configurations
	if err := domain.SavePluginConfigs(configs); err != nil {
		return fmt.Errorf("failed to save plugin configs: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' disabled successfully\n", pluginName)
	return nil
}

func runPluginConfigure(ctx context.Context, pluginName, configData, configFile string) error {
	// Check if plugin exists and user has access
	if err := validatePluginAccess(ctx, pluginName); err != nil {
		return err
	}

	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Parse configuration data
	var settings map[string]interface{}
	
	if configFile != "" {
		// Load from file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if configData != "" {
		// Parse from command line
		if err := json.Unmarshal([]byte(configData), &settings); err != nil {
			return fmt.Errorf("failed to parse config data: %w", err)
		}
	} else {
		return fmt.Errorf("either --data or --file must be provided")
	}

	// Update plugin configuration
	currentConfig := configs.GetPluginConfig(pluginName)
	for key, value := range settings {
		currentConfig.Settings[key] = value
	}
	configs.SetPluginConfig(pluginName, currentConfig)

	// Save configurations
	if err := domain.SavePluginConfigs(configs); err != nil {
		return fmt.Errorf("failed to save plugin configs: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' configured successfully\n", pluginName)
	fmt.Printf("Updated settings: %v\n", settings)
	return nil
}

func runPluginExport(ctx context.Context, pluginName, outputFile string) error {
	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Export plugin configuration
	exportData, err := configs.ExportConfig(pluginName)
	if err != nil {
		return fmt.Errorf("failed to export plugin config: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Output to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write export file: %w", err)
		}
		fmt.Printf("✅ Plugin configuration exported to %s\n", outputFile)
	} else {
		fmt.Printf("%s\n", string(jsonData))
	}

	return nil
}

func runPluginImport(ctx context.Context, pluginName, inputFile string) error {
	// Check if plugin exists and user has access
	if err := validatePluginAccess(ctx, pluginName); err != nil {
		return err
	}

	// Read import file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse import data
	var importData map[string]interface{}
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Import configuration
	if err := configs.ImportConfig(pluginName, importData); err != nil {
		return fmt.Errorf("failed to import plugin config: %w", err)
	}

	// Save configurations
	if err := domain.SavePluginConfigs(configs); err != nil {
		return fmt.Errorf("failed to save plugin configs: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' configuration imported successfully\n", pluginName)
	return nil
}

func runPluginReset(ctx context.Context, pluginName string) error {
	// Load current configurations
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Reset plugin to default configuration
	defaultConfig := domain.PluginConfig{
		Name:     pluginName,
		Enabled:  true,
		Settings: make(map[string]interface{}),
		Version:  "1.0",
	}
	configs.SetPluginConfig(pluginName, defaultConfig)

	// Save configurations
	if err := domain.SavePluginConfigs(configs); err != nil {
		return fmt.Errorf("failed to save plugin configs: %w", err)
	}

	fmt.Printf("✅ Plugin '%s' reset to default configuration\n", pluginName)
	return nil
}

// Helper functions

func validatePluginAccess(ctx context.Context, pluginName string) error {
	// Create authentication manager
	authManager := domain.NewAuthenticationManager()
	if err := authManager.LoadSubscription(); err != nil {
		// Continue with free tier validation
	}

	// Create plugin dependencies
	deps := ports.PluginDependencies{
		AuthManager: authManager,
		Config:      domain.LoadConfig(),
	}

	// Create plugin manager to validate access
	pluginManager := plugins.NewPluginManager(authManager, deps)
	if err := pluginManager.LoadPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Check if plugin exists and is accessible
	_, err := pluginManager.GetPlugin(pluginName)
	if err != nil {
		return err
	}

	return nil
}
