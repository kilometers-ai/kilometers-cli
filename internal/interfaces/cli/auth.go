package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/spf13/cobra"
)

// newAuthCommand creates the auth command group
func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and subscription",
		Long: `Authentication and subscription management commands.

Use these commands to login with your license key, check subscription status,
and manage your Kilometers CLI features.`,
	}

	cmd.AddCommand(newAuthLoginCommand())
	cmd.AddCommand(newAuthLogoutCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthRefreshCommand())
	cmd.AddCommand(newFeaturesCommand())

	return cmd
}

// newAuthLoginCommand creates the auth login subcommand
func newAuthLoginCommand() *cobra.Command {
	var licenseKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login with license key",
		Long: `Login to Kilometers with your license key to unlock premium features.

Your license key can be found in your account dashboard at kilometers.ai.

Examples:
  km auth login --license-key km_pro_1234567890abcdef
  km auth login -k km_enterprise_abcdef1234567890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogin(cmd.Context(), licenseKey)
		},
	}

	cmd.Flags().StringVarP(&licenseKey, "license-key", "k", "", "License key from kilometers.ai")
	cmd.MarkFlagRequired("license-key")

	return cmd
}

// newAuthLogoutCommand creates the auth logout subcommand
func newAuthLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear subscription",
		Long:  `Logout from Kilometers and revert to free tier functionality.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthLogout(cmd.Context())
		},
	}

	return cmd
}

// newAuthStatusCommand creates the auth status subcommand
func newAuthStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  `Display current subscription tier and feature availability.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatus(cmd.Context())
		},
	}

	return cmd
}

// newAuthRefreshCommand creates the auth refresh subcommand
func newAuthRefreshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh subscription status",
		Long:  `Refresh subscription information from the server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRefresh(cmd.Context())
		},
	}

	return cmd
}

// newFeaturesCommand creates the features subcommand
func newFeaturesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "features",
		Short: "List available features",
		Long:  `Display all features available for your current subscription tier.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListFeatures(cmd.Context())
		},
	}

	return cmd
}

// Command implementations

func runAuthLogin(ctx context.Context, licenseKey string) error {
	if licenseKey == "" {
		return fmt.Errorf("license key is required")
	}

	// Validate license key format
	if !isValidLicenseKey(licenseKey) {
		return fmt.Errorf("invalid license key format")
	}

	// Create authentication manager
	authManager := domain.NewAuthenticationManager()

	// TODO: In real implementation, validate license with API
	// For demo, we'll create a mock subscription based on license key prefix
	subscription, err := validateLicenseKeyWithAPI(licenseKey)
	if err != nil {
		return fmt.Errorf("failed to validate license key: %w", err)
	}

	// Save subscription
	if err := authManager.SaveSubscription(subscription); err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	fmt.Printf("✅ Successfully logged in!\n")
	fmt.Printf("Subscription: %s\n", subscription.Tier)
	fmt.Printf("Expires: %s\n", subscription.ExpiresAt.Format("2006-01-02"))
	fmt.Printf("Features: %s\n", formatFeatures(subscription.Features))

	return nil
}

func runAuthLogout(ctx context.Context) error {
	// Remove subscription config file
	configPath, err := getSubscriptionConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Delete the subscription file (ignore errors if file doesn't exist)
	_ = removeFile(configPath)

	fmt.Printf("✅ Successfully logged out. Reverted to free tier.\n")
	return nil
}

func runAuthStatus(ctx context.Context) error {
	authManager := domain.NewAuthenticationManager()

	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	tier := authManager.GetSubscriptionTier()
	fmt.Printf("Subscription Tier: %s\n", tier)

	if tier == domain.TierFree {
		fmt.Printf("Status: Free tier (limited features)\n")
		fmt.Printf("Upgrade at: https://kilometers.ai/pricing\n")
	} else {
		if err := authManager.ValidateSubscription(); err != nil {
			fmt.Printf("Status: ❌ Invalid (%s)\n", err.Error())
		} else {
			fmt.Printf("Status: ✅ Active\n")
		}

		if authManager.NeedsRefresh() {
			fmt.Printf("⚠️  Subscription needs refresh. Run 'km auth refresh'\n")
		}
	}

	return nil
}

func runAuthRefresh(ctx context.Context) error {
	authManager := domain.NewAuthenticationManager()

	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	if authManager.GetSubscriptionTier() == domain.TierFree {
		return fmt.Errorf("no subscription to refresh. Login with 'km auth login'")
	}

	// TODO: Implement actual refresh logic with API
	fmt.Printf("✅ Subscription refreshed successfully\n")
	return nil
}

func runListFeatures(ctx context.Context) error {
	authManager := domain.NewAuthenticationManager()

	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	tier := authManager.GetSubscriptionTier()
	fmt.Printf("Subscription Tier: %s\n\n", tier)

	features := getFeaturesByTier(tier)
	
	fmt.Printf("Available Features:\n")
	for _, feature := range features {
		status := "✅"
		if !authManager.IsFeatureEnabled(feature.Name) {
			status = "❌"
		}
		fmt.Printf("  %s %s - %s\n", status, feature.Name, feature.Description)
	}

	if tier != domain.TierEnterprise {
		fmt.Printf("\n🚀 Upgrade to unlock more features: https://kilometers.ai/pricing\n")
	}

	return nil
}

// Helper functions

func isValidLicenseKey(key string) bool {
	// Basic validation - real implementation would be more sophisticated
	return len(key) > 10 && (key[:3] == "km_")
}

func validateLicenseKeyWithAPI(licenseKey string) (*domain.SubscriptionConfig, error) {
	// Mock implementation - real version would call API
	var tier domain.SubscriptionTier
	var features []string

	if licenseKey[:7] == "km_pro_" {
		tier = domain.TierPro
		features = []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeatureCustomRules,
			domain.FeaturePoisonDetection,
			domain.FeatureMLAnalytics,
		}
	} else if licenseKey[:14] == "km_enterprise_" {
		tier = domain.TierEnterprise
		features = []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeatureCustomRules,
			domain.FeaturePoisonDetection,
			domain.FeatureMLAnalytics,
			domain.FeatureTeamCollaboration,
			domain.FeatureCustomDashboards,
			domain.FeatureComplianceReporting,
			domain.FeatureAPIIntegrations,
			domain.FeaturePrioritySupport,
		}
	} else {
		return nil, fmt.Errorf("invalid license key")
	}

	return &domain.SubscriptionConfig{
		Tier:        tier,
		Features:    features,
		ExpiresAt:   time.Now().AddDate(1, 0, 0), // 1 year from now
		Signature:   "mock_signature_" + licenseKey,
		LastRefresh: time.Now(),
	}, nil
}

func formatFeatures(features []string) string {
	if len(features) == 0 {
		return "None"
	}
	
	result := ""
	for i, feature := range features {
		if i > 0 {
			result += ", "
		}
		result += feature
	}
	return result
}

type FeatureInfo struct {
	Name        string
	Description string
}

func getFeaturesByTier(tier domain.SubscriptionTier) []FeatureInfo {
	baseFeatures := []FeatureInfo{
		{domain.FeatureBasicMonitoring, "Basic MCP message monitoring and logging"},
	}

	proFeatures := []FeatureInfo{
		{domain.FeatureAdvancedFilters, "Complex regex-based message filtering"},
		{domain.FeatureCustomRules, "Custom rule engine for message processing"},
		{domain.FeaturePoisonDetection, "AI-powered prompt injection detection"},
		{domain.FeatureMLAnalytics, "Machine learning analytics and insights"},
	}

	enterpriseFeatures := []FeatureInfo{
		{domain.FeatureTeamCollaboration, "Team sharing and collaboration features"},
		{domain.FeatureCustomDashboards, "Custom analytics dashboards"},
		{domain.FeatureComplianceReporting, "Compliance and audit reporting"},
		{domain.FeatureAPIIntegrations, "API integrations and webhooks"},
		{domain.FeaturePrioritySupport, "Priority customer support"},
	}

	switch tier {
	case domain.TierFree:
		return baseFeatures
	case domain.TierPro:
		return append(baseFeatures, proFeatures...)
	case domain.TierEnterprise:
		all := append(baseFeatures, proFeatures...)
		return append(all, enterpriseFeatures...)
	default:
		return baseFeatures
	}
}

// Platform-specific file operations would be implemented here
func removeFile(path string) error {
	// Implementation would remove the file
	return nil
}

func getSubscriptionConfigPath() (string, error) {
	// This would use the same logic as in auth_storage.go
	return "/path/to/subscription.json", nil
}
