package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/ports"
)

// NewInitCommand creates the init command
func NewInitCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize Kilometers CLI configuration",
		Long: `Initialize the Kilometers CLI configuration by setting up
required configuration values interactively.

This command will guide you through setting up:
- API Key
- API Endpoint
- Batch size
- Debug mode
- Other monitoring options`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(container)
		},
	}
}

// runInit handles the interactive configuration setup
func runInit(container *CLIContainer) error {
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
		config = &ports.Configuration{
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
