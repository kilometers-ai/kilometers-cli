package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/ports"
)

// InitFlags holds the command-line flags for the init command
type InitFlags struct {
	ConfigDir      string
	APIKey         string
	APIURL         string
	BatchSize      int
	Debug          bool
	NonInteractive bool
}

// NewInitCommand creates the init command
func NewInitCommand(container *CLIContainer) *cobra.Command {
	flags := &InitFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Kilometers CLI configuration",
		Long: `Initialize the Kilometers CLI configuration by setting up
required configuration values interactively or via command-line flags.

This command will guide you through setting up:
- API Key
- API Endpoint
- Batch size
- Debug mode
- Other monitoring options

You can use flags for non-interactive setup or run without flags for interactive mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(container, flags)
		},
	}

	// Add command-line flags
	cmd.Flags().StringVar(&flags.ConfigDir, "config-dir", "", "Directory to store configuration file")
	cmd.Flags().StringVar(&flags.APIKey, "api-key", "", "API key for Kilometers platform")
	cmd.Flags().StringVar(&flags.APIURL, "api-url", "", "API endpoint URL")
	cmd.Flags().IntVar(&flags.BatchSize, "batch-size", 0, "Batch size for event processing")
	cmd.Flags().BoolVar(&flags.Debug, "debug", false, "Enable debug mode")
	cmd.Flags().BoolVar(&flags.NonInteractive, "non-interactive", false, "Run in non-interactive mode")

	return cmd
}

// runInit handles the interactive configuration setup
func runInit(container *CLIContainer, flags *InitFlags) error {
	// Check if we should run in non-interactive mode
	nonInteractive := flags.NonInteractive ||
		flags.APIKey != "" ||
		flags.APIURL != "" ||
		flags.BatchSize > 0 ||
		flags.ConfigDir != ""

	if nonInteractive {
		return runNonInteractiveInit(container, flags)
	}

	return runInteractiveInit(container)
}

// runNonInteractiveInit handles non-interactive configuration setup
func runNonInteractiveInit(container *CLIContainer, flags *InitFlags) error {
	// Load existing configuration or defaults
	config, err := container.ConfigRepo.Load()
	if err != nil {
		// Use default configuration if loading fails
		config = getDefaultConfig()
	}

	// Apply command-line flags
	if flags.APIKey != "" {
		config.APIKey = flags.APIKey
	}
	if flags.APIURL != "" {
		config.APIEndpoint = flags.APIURL
	}
	if flags.BatchSize > 0 {
		config.BatchSize = flags.BatchSize
	}
	if flags.Debug {
		config.Debug = flags.Debug
	}

	// Validate required fields
	if config.APIKey == "" {
		return fmt.Errorf("API key is required. Provide it with --api-key flag or run without flags for interactive mode")
	}

	// Handle custom config directory
	if flags.ConfigDir != "" {
		// Create a new repository with custom config path
		customConfigPath := filepath.Join(flags.ConfigDir, "config.json")
		// Ensure directory exists
		if err := os.MkdirAll(flags.ConfigDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Save to custom location - we'll need to manually write the file
		if err := saveConfigToPath(config, customConfigPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✅ Configuration saved to: %s\n", customConfigPath)
	} else {
		// Save using the container's config repository
		if err := container.ConfigRepo.Save(config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✅ Configuration saved to: %s\n", container.ConfigRepo.GetConfigPath())
	}

	return nil
}

// runInteractiveInit handles the interactive configuration setup
func runInteractiveInit(container *CLIContainer) error {
	fmt.Println("🚀 Kilometers CLI Configuration Setup")
	fmt.Println("")
	fmt.Println("This will guide you through setting up your Kilometers CLI configuration.")
	fmt.Println("You can press Enter to accept default values shown in brackets.")
	fmt.Println("")

	scanner := bufio.NewScanner(os.Stdin)

	// Load existing configuration or defaults
	config, err := container.ConfigRepo.Load()
	if err != nil {
		// Use default configuration if loading fails
		config = getDefaultConfig()
	}

	// API Key (required)
	fmt.Print("API Key (required): ")
	scanner.Scan()
	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		return fmt.Errorf("API Key is required. Get yours from https://app.dev.kilometers.ai")
	}
	config.APIKey = apiKey

	// API URL (optional, default to production)
	fmt.Printf("API URL [%s]: ", config.APIEndpoint)
	scanner.Scan()
	apiURL := strings.TrimSpace(scanner.Text())
	if apiURL != "" {
		config.APIEndpoint = apiURL
	}

	// Debug mode (optional)
	fmt.Printf("Enable debug mode? [%t] (y/N): ", config.Debug)
	scanner.Scan()
	debugResponse := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if debugResponse == "y" || debugResponse == "yes" {
		config.Debug = true
	} else if debugResponse == "n" || debugResponse == "no" {
		config.Debug = false
	}

	// Batch size (optional)
	fmt.Printf("Batch size [%d]: ", config.BatchSize)
	scanner.Scan()
	batchSizeStr := strings.TrimSpace(scanner.Text())
	if batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil && batchSize > 0 {
			config.BatchSize = batchSize
		}
	}

	// Risk detection (optional)
	fmt.Printf("Enable risk detection? [%t] (y/N): ", config.EnableRiskDetection)
	scanner.Scan()
	riskResponse := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if riskResponse == "y" || riskResponse == "yes" {
		config.EnableRiskDetection = true
	} else if riskResponse == "n" || riskResponse == "no" {
		config.EnableRiskDetection = false
	}

	// Save configuration
	if err := container.ConfigRepo.Save(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("")
	fmt.Println("✅ Configuration saved successfully!")
	fmt.Printf("   Config file: %s\n", container.ConfigRepo.GetConfigPath())
	fmt.Println("")

	// Show environment variables for session
	fmt.Println("📝 For your current session, you can also set these environment variables:")
	fmt.Printf("   export KILOMETERS_API_KEY=\"%s\"\n", apiKey)
	fmt.Printf("   export KILOMETERS_API_URL=\"%s\"\n", config.APIEndpoint)
	if config.Debug {
		fmt.Println("   export KM_DEBUG=true")
	}
	fmt.Println("")

	fmt.Println("🎉 Ready to use Kilometers CLI!")
	fmt.Println("")
	fmt.Println("Try it out:")
	fmt.Println("   km monitor npx @modelcontextprotocol/server-github")
	fmt.Println("")
	fmt.Println("Dashboard: https://app.dev.kilometers.ai")

	return nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *ports.Configuration {
	return &ports.Configuration{
		APIEndpoint:           "https://api.dev.kilometers.ai",
		BatchSize:             10,
		FlushInterval:         30,
		Debug:                 false,
		EnableRiskDetection:   false,
		MethodWhitelist:       []string{},
		MethodBlacklist:       []string{},
		PayloadSizeLimit:      0,
		HighRiskMethodsOnly:   false,
		ExcludePingMessages:   true,
		MinimumRiskLevel:      "low",
		EnableLocalStorage:    false,
		StoragePath:           "",
		MaxStorageSize:        0,
		RetentionDays:         30,
		MaxConcurrentRequests: 10,
		RequestTimeout:        30,
		RetryAttempts:         3,
		RetryDelay:            1000,
	}
}

// saveConfigToPath saves configuration to a specific file path
func saveConfigToPath(config *ports.Configuration, configPath string) error {
	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}
