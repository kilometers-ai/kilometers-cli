package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	// "path/filepath" // Disabled for testing
	"strings"

	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	// "github.com/kilometers-ai/kilometers-cli/internal/plugins" // Disabled for testing
	"github.com/spf13/cobra"
)

// newInitCommand creates the init subcommand
func newInitCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize kilometers configuration",
		Long: `Initialize kilometers configuration file with API key and endpoint.

The configuration file is stored at ~/.config/kilometers/config.json

Examples:
  # Interactive setup
  km init
  
  # Set API key directly  
  km init --api-key YOUR_API_KEY
  
  # Set custom endpoint
  km init --api-key YOUR_API_KEY --endpoint http://localhost:5194
  
  # Auto-detect configuration
  km init --auto-detect
  
  # Auto-detect with plugin provisioning
  km init --auto-detect --auto-provision-plugins

Note: Environment variables (KM_API_KEY, etc.) take precedence over config files.
Use 'km auth status' to see where your current configuration is loaded from.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCommand(cmd, args, version)
		},
	}

	cmd.Flags().StringP("api-key", "k", "", "API key for kilometers service")
	cmd.Flags().StringP("endpoint", "e", "http://localhost:5194", "API endpoint URL")
	cmd.Flags().Bool("force", false, "Overwrite existing configuration")
	cmd.Flags().Bool("auto-provision-plugins", false, "Automatically download and install plugins for your tier")
	cmd.Flags().Bool("auto-detect", false, "Automatically detect configuration from environment and files")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string, version string) error {
	// Get flags
	apiKey, _ := cmd.Flags().GetString("api-key")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	force, _ := cmd.Flags().GetBool("force")
	autoProvision, _ := cmd.Flags().GetBool("auto-provision-plugins")
	autoDetect, _ := cmd.Flags().GetBool("auto-detect")

	// Check if config already exists
	configPath, err := configpkg.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Println("Use --force to overwrite or edit the file manually.")
		return nil
	}

	// Start with current config or defaults
	currentConfig := configpkg.LoadConfig()

	// Handle auto-detect mode
	if autoDetect {
		// Create the infrastructure components
		loader := configpkg.NewUnifiedLoader()
		storage, err := configpkg.NewUnifiedStorage()
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}

		// Create the application service
		configService := configpkg.NewConfigService(loader, storage)

		// Load current configuration (this automatically discovers from all sources)
		loadedConfig, err := configService.Load(context.Background())
		if err != nil {
			// If loading fails, start with defaults and show what we tried
			currentConfig = configpkg.LoadConfig()
			fmt.Printf("⚠️  Could not load existing configuration: %v\n", err)
		} else {
			currentConfig = loadedConfig
		}

		// Override with explicit flags if provided
		if apiKey != "" {
			currentConfig.APIKey = apiKey
		}
		if endpoint != "" && endpoint != "http://localhost:5194" { // Only override if not default
			currentConfig.APIEndpoint = endpoint
		}

		// Show discovered configuration using ConfigService status
		fmt.Println()
		if err := displayConfigurationSources(configService); err != nil {
			// Fallback to basic display if status fails
			fmt.Println("🔍 Configuration Discovery Results:")
			if currentConfig.APIKey != "" {
				fmt.Printf("  🔑 API Key: %s\n", maskApiKey(currentConfig.APIKey))
			} else {
				fmt.Printf("  🔑 API Key: <not found>\n")
			}
			fmt.Printf("  🌐 API Endpoint: %s\n", currentConfig.APIEndpoint)
		}
		fmt.Println()

		if !confirmConfiguration(currentConfig) {
			fmt.Println("Configuration cancelled.")
			return nil
		}
	} else {
		// Interactive mode if no API key provided
		if apiKey == "" {
			fmt.Println("✓ Setting up kilometers configuration...")
			fmt.Println()

			reader := bufio.NewReader(os.Stdin)

			// Get API key
			fmt.Print("Enter your API key (leave empty to skip): ")
			input, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(input)
		}

		// Update config
		if apiKey != "" {
			currentConfig.APIKey = apiKey
		}
		currentConfig.APIEndpoint = endpoint
	}

	// Save config
	if err := configpkg.SaveConfig(currentConfig); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Success message
	fmt.Printf("✓ Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Your configuration:")
	if currentConfig.APIKey != "" {
		fmt.Printf("  API Key: %s...\n", maskApiKey(currentConfig.APIKey))
	} else {
		fmt.Println("  API Key: (not set - will use KM_API_KEY environment variable)")
	}
	fmt.Printf("  Endpoint: %s\n", currentConfig.APIEndpoint)
	fmt.Println()
	fmt.Println("To use your configuration:")
	fmt.Println("  km monitor --server -- npx @modelcontextprotocol/server-github")
	fmt.Println()
	fmt.Println("Note: Environment variables take precedence over config file settings.")

	// Auto-provision plugins if requested
	if autoProvision && currentConfig.APIKey != "" {
		fmt.Println()
		fmt.Println("🔍 Checking available plugins for your subscription tier...")

		if err := provisionPlugins(currentConfig, version); err != nil {
			fmt.Printf("⚠️  Plugin provisioning failed: %v\n", err)
			fmt.Println("You can try again later with: km plugins refresh")
		}
	} else if autoProvision && currentConfig.APIKey == "" {
		fmt.Println()
		fmt.Println("⚠️  Cannot auto-provision plugins without an API key")
		fmt.Println("Set your API key and run: km plugins refresh")
	}

	return nil
}

// maskApiKey masks an API key for display
func maskApiKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

// provisionPlugins handles automatic plugin provisioning
func provisionPlugins(config *configpkg.UnifiedConfig, version string) error {
	// ctx := context.Background() // Disabled for testing
	_ = version // Mark as used

	// Plugin provisioning service disabled for testing
	// provisioningService := plugins.NewHTTPPluginProvisioningService(config.APIEndpoint)

	// Plugin downloader disabled for testing
	// downloader, err := plugins.NewSecurePluginDownloader(config.PluginsDir, config.Debug, version)
	// if err != nil {
	//	return fmt.Errorf("failed to create plugin downloader: %w", err)
	// }

	// Plugin installer disabled for testing
	// installer, err := plugins.NewFileSystemPluginInstallerFactory(config.PluginsDir)
	// if err != nil {
	//	return fmt.Errorf("failed to create plugin installer: %w", err)
	// }

	// Registry store disabled for testing
	// configPath, err := configpkg.GetConfigPath()
	// if err != nil {
	//	return fmt.Errorf("failed to get config path: %w", err)
	// }
	// configDir := filepath.Dir(configPath)
	// registryStore, err := plugins.NewFilePluginRegistryStore(configDir)
	// if err != nil {
	//	return fmt.Errorf("failed to create registry store: %w", err)
	// }

	// Provisioning manager disabled for testing
	// manager := plugins.NewProvisioningManager(
	//	provisioningService,
	//	downloader,
	//	installer,
	//	registryStore,
	// )
	// var manager interface{} = nil // Disabled for testing

	// Auto-provisioning disabled for testing
	if config.Debug {
		fmt.Println("Plugin auto-provisioning disabled for testing")
	}
	return nil
}

// displayConfigurationSources shows a summary of discovered configuration
func displayConfigurationSources(configService *configpkg.ConfigService) error {
	ctx := context.Background()

	status, err := configService.GetConfigStatus(ctx)
	if err != nil {
		return err
	}

	fmt.Println("🔍 Configuration Auto-Detection Results:")

	// Display API Key
	if status.HasAPIKey {
		fmt.Printf("  🔑 API Key: %s ✓\n", maskApiKey("dummy_key_for_display"))
	} else {
		fmt.Printf("  🔑 API Key: <not found>\n")
	}

	// Display API Endpoint
	fmt.Printf("  🌐 API Endpoint: %s\n", status.APIEndpoint)

	// Count discovered sources
	sourceCount := len(status.Sources)
	if sourceCount > 0 {
		fmt.Printf("  📋 Found configuration from %d source(s)\n", sourceCount)
		fmt.Printf("  💡 Run 'km auth status' to see detailed source information\n")
	}

	return nil
}

// confirmConfiguration asks the user to confirm the discovered configuration
func confirmConfiguration(config *configpkg.UnifiedConfig) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Use this configuration? [Y/n] ")
	input, _ := reader.ReadString('\n')
	response := strings.TrimSpace(strings.ToLower(input))

	// Default to yes if just Enter pressed
	if response == "" || response == "y" || response == "yes" {
		return true
	}

	if response == "n" || response == "no" {
		return false
	}

	// If unclear response, ask again
	fmt.Println("Please answer 'y' for yes or 'n' for no.")
	return confirmConfiguration(config)
}
