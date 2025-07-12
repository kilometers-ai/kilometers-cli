package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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

			fmt.Println("Current Configuration:")
			fmt.Println(strings.Repeat("-", 40))

			fmt.Printf("API Endpoint: %s\n", config.APIEndpoint)
			fmt.Printf("Batch Size: %d\n", config.BatchSize)
			fmt.Printf("Flush Interval: %d seconds\n", config.FlushInterval)
			fmt.Printf("Debug Mode: %t\n", config.Debug)
			fmt.Printf("Risk Detection: %t\n", config.EnableRiskDetection)
			fmt.Printf("Method Whitelist: %v\n", config.MethodWhitelist)
			fmt.Printf("Method Blacklist: %v\n", config.MethodBlacklist)
			fmt.Printf("Payload Size Limit: %d\n", config.PayloadSizeLimit)
			fmt.Printf("High Risk Methods Only: %t\n", config.HighRiskMethodsOnly)
			fmt.Printf("Exclude Ping Messages: %t\n", config.ExcludePingMessages)
			fmt.Printf("Minimum Risk Level: %s\n", config.MinimumRiskLevel)
			fmt.Printf("Max Concurrent Requests: %d\n", config.MaxConcurrentRequests)
			fmt.Printf("Request Timeout: %d seconds\n", config.RequestTimeout)
			fmt.Printf("Retry Attempts: %d\n", config.RetryAttempts)
			fmt.Printf("Retry Delay: %d ms\n", config.RetryDelay)

			return nil
		},
	}
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
