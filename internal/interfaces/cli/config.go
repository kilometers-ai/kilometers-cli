package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/ports"
)

// NewConfigCommand creates the config command
func NewConfigCommand(container *CLIContainer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage configuration settings",
		Long: `Manage configuration settings for the Kilometers CLI.

This command allows you to view and manage configuration values
for the Kilometers CLI application.`,
	}

	// Add subcommands
	configCmd.AddCommand(NewConfigShowCommand(container))
	configCmd.AddCommand(NewConfigPathCommand(container))

	return configCmd
}

// NewConfigShowCommand creates the show subcommand
func NewConfigShowCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := container.ConfigRepo.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			printConfig(config)

			return nil
		},
	}
}

func printConfig(config *ports.Configuration) {
	fmt.Println("Current Configuration:")
	fmt.Printf("API Host: %s\n", config.APIHost)
	fmt.Printf("API Key: %s\n", maskAPIKey(config.APIKey))
	fmt.Printf("Batch Size: %d\n", config.BatchSize)
	fmt.Printf("Debug: %t\n", config.Debug)
}

// maskAPIKey masks the API key for display
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// NewConfigPathCommand creates the path subcommand
func NewConfigPathCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := container.ConfigRepo.GetConfigPath()
			fmt.Printf("Configuration file path: %s\n", path)
			return nil
		},
	}
}
