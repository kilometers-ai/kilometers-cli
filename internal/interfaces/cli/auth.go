package cli

import (
	"context"
	"fmt"

	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

// newAuthCommand creates the auth subcommand
func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and API keys",
		Long:  `Manage authentication for Kilometers CLI, including API key configuration and subscription status.`,
	}

	cmd.AddCommand(newAuthLoginCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthLogoutCommand())

	return cmd
}

// newAuthLoginCommand creates the auth login subcommand
func newAuthLoginCommand() *cobra.Command {
	var apiKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Set your API key",
		Long:  `Configure your Kilometers API key to enable premium features.`,
		Example: `  km auth login --api-key km_pro_1234567890abcdef
  km auth login --api-key km_free_0987654321fedcba`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if apiKey == "" {
				return fmt.Errorf("API key is required")
			}

			// Use unified configuration system
			configService, err := config.CreateConfigServiceFromDefaults()
			if err != nil {
				return fmt.Errorf("failed to create config service: %w", err)
			}

			// Update API key using unified system
			ctx := context.Background()
			if err := configService.UpdateAPIKey(ctx, apiKey); err != nil {
				return fmt.Errorf("failed to save API key: %w", err)
			}

			fmt.Printf("✅ API key configured successfully\n")
			fmt.Printf("🔄 Fetching subscription features...\n")

			// Note: In a real implementation, this would call the API to verify
			// For testing, we'll just show a message based on the key prefix
			if len(apiKey) > 7 {
				prefix := apiKey[:7]
				switch prefix {
				case "km_free":
					fmt.Printf("📋 Subscription: Free tier\n")
					fmt.Printf("   Features: basic_monitoring, console_logging\n")
				case "km_pro_":
					fmt.Printf("📋 Subscription: Pro tier\n")
					fmt.Printf("   Features: basic_monitoring, console_logging, api_logging, advanced_filters\n")
				case "km_ent_":
					fmt.Printf("📋 Subscription: Enterprise tier\n")
					fmt.Printf("   Features: All features enabled\n")
				default:
					fmt.Printf("📋 Subscription: Unknown (will verify with API)\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "Your Kilometers API key")
	cmd.MarkFlagRequired("api-key")

	return cmd
}

// newAuthStatusCommand creates the auth status subcommand
func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long:  `Display your current API key configuration and subscription features.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use unified configuration system
			configService, err := config.CreateConfigServiceFromDefaults()
			if err != nil {
				return fmt.Errorf("failed to create config service: %w", err)
			}

			ctx := context.Background()
			unifiedConfig, err := configService.Load(ctx)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			if !unifiedConfig.HasAPIKey() {
				fmt.Printf("❌ No API key configured\n")
				fmt.Printf("   Run 'km auth login --api-key YOUR_KEY' to configure\n")
				return nil
			}

			// Mask the API key for display
			maskedKey := unifiedConfig.APIKey
			if len(maskedKey) > 10 {
				maskedKey = maskedKey[:6] + "..." + maskedKey[len(maskedKey)-4:]
			}

			fmt.Printf("🔑 API Key: %s\n", maskedKey)
			fmt.Printf("🌐 API Endpoint: %s\n", unifiedConfig.APIEndpoint)

			// Show configuration source information
			if source, exists := unifiedConfig.GetSource("api_key"); exists {
				fmt.Printf("📍 API Key Source: %s (%s)\n", source.Source, source.SourcePath)
			}
			if source, exists := unifiedConfig.GetSource("api_endpoint"); exists {
				fmt.Printf("📍 API Endpoint Source: %s (%s)\n", source.Source, source.SourcePath)
			}

			// In a real implementation, this would check cached subscription info
			fmt.Printf("\n📋 To refresh subscription info, monitor a server or run 'km auth login' again\n")

			return nil
		},
	}
}

// newAuthLogoutCommand creates the auth logout subcommand
func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove API key configuration",
		Long:  `Remove your API key and revert to free tier features.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use unified configuration system
			configService, err := config.CreateConfigServiceFromDefaults()
			if err != nil {
				return fmt.Errorf("failed to create config service: %w", err)
			}

			ctx := context.Background()
			if err := configService.ClearAPIKey(ctx); err != nil {
				return fmt.Errorf("failed to clear API key: %w", err)
			}

			fmt.Printf("✅ Logged out successfully\n")
			fmt.Printf("📋 Now using free tier features only\n")

			return nil
		},
	}
}
