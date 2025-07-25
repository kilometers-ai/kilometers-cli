package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
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

Note: Environment variables KILOMETERS_API_KEY and KILOMETERS_API_ENDPOINT 
will take precedence over configuration file settings.`,
		RunE: runInitCommand,
	}

	cmd.Flags().StringP("api-key", "k", "", "API key for kilometers service")
	cmd.Flags().StringP("endpoint", "e", "http://localhost:5194", "API endpoint URL")
	cmd.Flags().Bool("force", false, "Overwrite existing configuration")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	// Get flags
	apiKey, _ := cmd.Flags().GetString("api-key")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	force, _ := cmd.Flags().GetBool("force")

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

	return nil
}

// maskApiKey masks an API key for display
func maskApiKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
