package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/application/services"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins"
	"github.com/spf13/cobra"
)

// newInitCommand creates the init subcommand
func newInitCommand() *cobra.Command {
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

Note: Environment variables KILOMETERS_API_KEY and KILOMETERS_API_ENDPOINT 
will take precedence over configuration file settings.`,
		RunE: runInitCommand,
	}

	cmd.Flags().StringP("api-key", "k", "", "API key for kilometers service")
	cmd.Flags().StringP("endpoint", "e", "http://localhost:5194", "API endpoint URL")
	cmd.Flags().Bool("force", false, "Overwrite existing configuration")
	cmd.Flags().Bool("auto-provision-plugins", false, "Automatically download and install plugins for your tier")
	cmd.Flags().Bool("auto-detect", false, "Automatically detect configuration from environment and files")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	// Get flags
	apiKey, _ := cmd.Flags().GetString("api-key")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	force, _ := cmd.Flags().GetBool("force")
	autoProvision, _ := cmd.Flags().GetBool("auto-provision-plugins")
	autoDetect, _ := cmd.Flags().GetBool("auto-detect")

	// Check if config already exists
	configPath, err := domain.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Println("Use --force to overwrite or edit the file manually.")
		return nil
	}

	// Start with current config or defaults
	config := domain.LoadConfig()

	// Handle auto-detect mode
	if autoDetect {
		discoveredConfig, err := runAutoDetect()
		if err != nil {
			return fmt.Errorf("auto-detection failed: %w", err)
		}

		// Convert discovered config to standard config
		config = *discoveredConfig.ToConfig()

		// Override with explicit flags if provided
		if apiKey != "" {
			config.ApiKey = apiKey
		}
		if endpoint != "" && endpoint != "http://localhost:5194" { // Only override if not default
			config.ApiEndpoint = endpoint
		}

		// Show discovered configuration and ask for confirmation
		fmt.Println()
		services.PrintDiscoveredConfig(discoveredConfig)
		fmt.Println()

		if !confirmConfiguration(&config) {
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
			config.ApiKey = apiKey
		}
		config.ApiEndpoint = endpoint
	}

	// Save config
	if err := domain.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Success message
	fmt.Printf("✓ Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Your configuration:")
	if config.ApiKey != "" {
		fmt.Printf("  API Key: %s...\n", maskApiKey(config.ApiKey))
	} else {
		fmt.Println("  API Key: (not set - will use KILOMETERS_API_KEY environment variable)")
	}
	fmt.Printf("  Endpoint: %s\n", config.ApiEndpoint)
	fmt.Println()
	fmt.Println("To use your configuration:")
	fmt.Println("  km monitor --server -- npx @modelcontextprotocol/server-github")
	fmt.Println()
	fmt.Println("Note: Environment variables take precedence over config file settings.")

	// Auto-provision plugins if requested
	if autoProvision && config.ApiKey != "" {
		fmt.Println()
		fmt.Println("🔍 Checking available plugins for your subscription tier...")

		if err := provisionPlugins(&config); err != nil {
			fmt.Printf("⚠️  Plugin provisioning failed: %v\n", err)
			fmt.Println("You can try again later with: km plugins refresh")
		}
	} else if autoProvision && config.ApiKey == "" {
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
func provisionPlugins(config *domain.Config) error {
	ctx := context.Background()

	// Create plugin provisioning service
	provisioningService := plugins.NewHTTPPluginProvisioningService(config.ApiEndpoint)

	// Create plugin downloader
	downloader, err := plugins.NewSecurePluginDownloader(plugins.DefaultPublicKey)
	if err != nil {
		return fmt.Errorf("failed to create plugin downloader: %w", err)
	}

	// Create plugin installer
	pluginDir := "~/.km/plugins"
	installer, err := plugins.NewFileSystemPluginInstaller(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to create plugin installer: %w", err)
	}

	// Create registry store
	configDir := "~/.config/kilometers"
	registryStore, err := plugins.NewFilePluginRegistryStore(configDir)
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
	return manager.AutoProvisionPlugins(ctx, config)
}

// runAutoDetect runs the configuration discovery process
func runAutoDetect() (*domain.DiscoveredConfig, error) {
	ctx := context.Background()

	// Create discovery service
	discoveryService, err := services.NewConfigDiscoveryService()
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	// Run discovery
	discoveredConfig, err := discoveryService.DiscoverConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("configuration discovery failed: %w", err)
	}

	// Validate discovered configuration
	if err := discoveryService.ValidateConfig(discoveredConfig); err != nil {
		fmt.Printf("⚠️  Warning: Some discovered values failed validation:\n%v\n", err)
		fmt.Println()
		// Continue anyway - user can fix values interactively
	}

	return discoveredConfig, nil
}

// confirmConfiguration asks the user to confirm the discovered configuration
func confirmConfiguration(config *domain.Config) bool {
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
