package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/application"
	apphttp "github.com/kilometers-ai/kilometers-cli/internal/application/http"
	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	plugininfra "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugin"
	"github.com/spf13/cobra"
)

// newInitCommandRefactored creates the refactored init subcommand using DDD architecture
func newInitCommandRefactored(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize kilometers configuration",
		Long: `Initialize kilometers configuration with interactive API key validation and plugin setup.

This command will:
1. Detect or prompt for your API key
2. Validate the key with the Kilometers API
3. Check your subscription tier and entitlements
4. Offer to install/update available plugins
5. Save your configuration

Examples:
  # Interactive setup (recommended)
  km init
  
  # Provide API key directly
  km init --api-key YOUR_API_KEY
  
  # Auto-detect from environment
  km init --auto-detect
  
  # Auto-provision plugins without prompts
  km init --auto-provision-plugins

Note: Environment variables (KM_API_KEY) take precedence over saved configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCommandRefactored(cmd, args, version)
		},
	}

	cmd.Flags().StringP("api-key", "k", "", "API key for kilometers service")
	cmd.Flags().StringP("endpoint", "e", "", "API endpoint URL (defaults to production)")
	cmd.Flags().Bool("force", false, "Overwrite existing configuration")
	cmd.Flags().Bool("auto-provision-plugins", false, "Automatically install plugins without prompts")
	cmd.Flags().Bool("auto-detect", false, "Automatically detect configuration from environment")

	return cmd
}

// runInitCommandRefactored executes the refactored init command with full DDD architecture
func runInitCommandRefactored(cmd *cobra.Command, args []string, version string) error {
	ctx := context.Background()

	// Get flags
	flagAPIKey, _ := cmd.Flags().GetString("api-key")
	flagEndpoint, _ := cmd.Flags().GetString("endpoint")
	force, _ := cmd.Flags().GetBool("force")
	autoProvision, _ := cmd.Flags().GetBool("auto-provision-plugins")
	autoDetect, _ := cmd.Flags().GetBool("auto-detect")

	// Initialize configuration infrastructure using DDD components
	loader := configpkg.NewUnifiedLoader()
	storage, err := configpkg.NewUnifiedStorage()
	if err != nil {
		return fmt.Errorf("failed to create configuration storage: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)

	// Check if config already exists
	configPath, err := configpkg.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	// Check for existing config without --force
	if _, err := os.Stat(configPath); err == nil && !force && !autoDetect {
		status, err := configService.GetConfigStatus(ctx)
		if err == nil && status.HasAPIKey {
			fmt.Printf("✅ Configuration already exists at: %s\n", configPath)
			fmt.Printf("   API Key: %s\n", maskApiKey("configured"))
			fmt.Printf("   Endpoint: %s\n", status.APIEndpoint)
			fmt.Println("\nUse --force to reconfigure or 'km auth status' to see details.")
			return nil
		}
	}

	// Load current configuration (from all sources)
	currentConfig, err := configService.Load(ctx)
	if err != nil {
		// Start with defaults if loading fails
		currentConfig = configpkg.LoadConfig()
	}

	// Initialize with empty API key
	apiKey := ""
	endpoint := currentConfig.APIEndpoint

	// Override with flag values if provided
	if flagAPIKey != "" {
		apiKey = flagAPIKey
	}
	if flagEndpoint != "" {
		endpoint = flagEndpoint
	}

	// Auto-detect mode: discover from environment
	if autoDetect {
		fmt.Println("🔍 Auto-detecting configuration...")

		// The config service already loaded from all sources
		if currentConfig.APIKey != "" && apiKey == "" {
			apiKey = currentConfig.APIKey
			fmt.Printf("   ✓ Found API key from: %s\n", getSourceDisplay(currentConfig, "api_key"))
		}

		if currentConfig.APIEndpoint != "" {
			endpoint = currentConfig.APIEndpoint
			fmt.Printf("   ✓ Found endpoint from: %s\n", getSourceDisplay(currentConfig, "api_endpoint"))
		}

		if apiKey == "" {
			fmt.Println("   ⚠️  No API key found in environment or config files")
		}
	}

	// Interactive setup if no API key provided
	if apiKey == "" && !autoProvision {
		fmt.Println("\n🚀 Kilometers CLI Interactive Setup")
		fmt.Println("====================================")
		fmt.Println("\nThis wizard will help you:")
		fmt.Println("  1. Configure your API key")
		fmt.Println("  2. Validate your subscription")
		fmt.Println("  3. Install available plugins\n")

		reader := bufio.NewReader(os.Stdin)

		// Prompt for API key
		fmt.Print("Enter your Kilometers API key (or press Enter for Free tier): ")
		input, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(input)
	}

	// Set default endpoint if not specified
	if endpoint == "" {
		endpoint = "https://api.kilometers.ai"
	}

	// Handle case where no API key is provided
	if apiKey == "" {
		fmt.Println("\n⚠️  No API key provided")
		fmt.Println("   You'll be limited to Free tier features.")
		fmt.Println("   To unlock Pro features later, run: km auth login --api-key YOUR_KEY")

		// Save basic config without API key
		currentConfig.APIEndpoint = endpoint
		if err := configService.Save(ctx, currentConfig); err != nil {
			fmt.Printf("⚠️  Failed to save configuration: %v\n", err)
		} else {
			fmt.Printf("\n✅ Basic configuration saved to: %s\n", configPath)
		}

		printQuickStart()
		return nil
	}

	// Now we have an API key - validate and provision plugins
	fmt.Println("\n📡 Validating API key...")

	// Create backend client using DDD architecture
	authHeaderService := apphttp.NewAuthHeaderService(apiKey, "kilometers-cli/"+version)
	backendClient := apphttp.NewBackendClient(
		endpoint,
		"kilometers-cli/"+version,
		30*time.Second,
		authHeaderService,
		nil, // No retry policy for init
	)

	// Create infrastructure services
	provisioningService := plugininfra.NewHTTPProvisioningService(backendClient)

	// Create plugin installer
	pluginsDir := currentConfig.PluginsDir
	if pluginsDir == "" {
		homeDir, _ := os.UserHomeDir()
		pluginsDir = filepath.Join(homeDir, ".km", "plugins")
	}

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		fmt.Printf("⚠️  Failed to create plugins directory: %v\n", err)
	}

	installer := plugininfra.NewFileSystemInstaller(pluginsDir, currentConfig.Debug)

	// Create plugin registry
	configDir := filepath.Dir(configPath)
	registry := plugininfra.NewFileSystemRegistry(configDir)

	// Create application service for plugin provisioning
	provisioningApp := application.NewPluginProvisioningService(
		provisioningService,
		installer,
		registry,
	)

	// Run the complete initialization flow with validation and provisioning
	initConfig := application.InitConfig{
		APIKey:        apiKey,
		APIEndpoint:   endpoint,
		AutoProvision: autoProvision,
		Interactive:   !autoProvision, // Interactive unless auto-provision is set
		Force:         force,
	}

	initResult, err := provisioningApp.InitializeWithValidation(ctx, initConfig)
	if err != nil {
		// Check if it's just an API key validation failure
		if strings.Contains(err.Error(), "invalid API key") {
			fmt.Println("\n❌ API key validation failed")
			fmt.Println("   Please check your API key and try again.")
			fmt.Println("   Get your API key from: https://kilometers.ai/dashboard")
			return nil
		}

		fmt.Printf("\n⚠️  Setup encountered issues: %v\n", err)
		fmt.Println("   You can retry with: km init --auto-provision-plugins")
	}

	// Save configuration if validation succeeded
	if initResult != nil && initResult.APIKeyValidated {
		currentConfig.APIKey = apiKey
		currentConfig.APIEndpoint = endpoint

		if err := configService.UpdateAPIKey(ctx, apiKey); err != nil {
			fmt.Printf("⚠️  Failed to save API key: %v\n", err)
		} else {
			fmt.Printf("\n✅ Configuration saved to: %s\n", configPath)
		}

		// Show summary
		if initResult.Subscription != nil {
			fmt.Println("\n📋 Subscription Details:")
			fmt.Printf("   Customer: %s\n", initResult.Subscription.CustomerName)
			fmt.Printf("   Tier: %s\n", string(initResult.Subscription.Tier))
			if len(initResult.Subscription.AvailableFeatures) > 0 {
				fmt.Printf("   Features: %s\n", strings.Join(initResult.Subscription.AvailableFeatures, ", "))
			}
		}

		if initResult.PluginsInstalled > 0 {
			fmt.Printf("\n🎉 Setup complete! Installed %d plugin(s).\n", initResult.PluginsInstalled)
		} else if initResult.PluginsAvailable > 0 {
			fmt.Printf("\n✅ Setup complete! %d plugin(s) available.\n", initResult.PluginsAvailable)
			fmt.Println("   Install them with: km plugins install")
		} else {
			fmt.Println("\n✅ Setup complete!")
		}
	}

	printQuickStart()
	return nil
}

// getSourceDisplay returns a human-readable source description
func getSourceDisplay(config *configpkg.UnifiedConfig, key string) string {
	if source, exists := config.GetSource(key); exists {
		switch source.Source {
		case "env":
			return fmt.Sprintf("environment variable %s", source.SourcePath)
		case "filesystem":
			return fmt.Sprintf("file %s", source.SourcePath)
		case "saved":
			return "saved configuration"
		default:
			return source.Source
		}
	}
	return "unknown"
}

// printQuickStart prints quick start instructions
func printQuickStart() {
	fmt.Println("\n🚀 Quick Start:")
	fmt.Println("   km monitor -- npx @modelcontextprotocol/server-filesystem /tmp")

	fmt.Println("\n💡 Useful Commands:")
	fmt.Println("   km auth status         - Check your configuration and subscription")
	fmt.Println("   km plugins list        - List available plugins")
	fmt.Println("   km plugins install     - Install or update plugins")
	fmt.Println("   km auth login          - Update your API key")
}
