package cli

import (
	"context"
	"fmt"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins"
	"github.com/spf13/cobra"
)

// newPluginsCommand creates the plugins command group
func newPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage and configure plugins",
		Long: `Plugin management commands for configuring advanced features.

Premium features are available through plugins that can be enabled and configured
based on your subscription tier.`,
	}

	cmd.AddCommand(newPluginsListCommand())
	cmd.AddCommand(newPluginsConfigCommand())
	cmd.AddCommand(newPluginsStatusCommand())

	// Add enhanced plugin management commands
	for _, mgmtCmd := range newPluginManagementCommands() {
		cmd.AddCommand(mgmtCmd)
	}

	// Add private plugin management commands
	for _, privateCmd := range newPrivatePluginCommands() {
		cmd.AddCommand(privateCmd)
	}

	return cmd
}

// newPluginsListCommand creates the plugins list subcommand
func newPluginsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Long:  `Display all plugins available for your current subscription tier.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsList(cmd.Context())
		},
	}

	return cmd
}

// newPluginsConfigCommand creates the plugins config subcommand
func newPluginsConfigCommand() *cobra.Command {
	var pluginName string
	var command string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure a specific plugin",
		Long: `Configure settings for a specific plugin.

Examples:
  km plugins config --plugin advanced-filters --command add-rule --data '{"name":"block-secrets","pattern":"secret.*","action":"redact"}'
  km plugins config --plugin poison-detection --command get-threats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginConfig(cmd.Context(), pluginName, command, args)
		},
	}

	cmd.Flags().StringVarP(&pluginName, "plugin", "p", "", "Plugin name to configure")
	cmd.Flags().StringVarP(&command, "command", "c", "", "Plugin command to execute")
	cmd.MarkFlagRequired("plugin")
	cmd.MarkFlagRequired("command")

	return cmd
}

// newPluginsStatusCommand creates the plugins status subcommand
func newPluginsStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show plugin status",
		Long:  `Display the status of all plugins and their availability.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsStatus(cmd.Context())
		},
	}

	return cmd
}

// Plugin command implementations

func runPluginsList(ctx context.Context) error {
	// Create authentication manager
	authManager := domain.NewAuthenticationManager()
	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	// Create plugin dependencies
	deps := ports.PluginDependencies{
		AuthManager: authManager,
		Config:      domain.LoadConfig(),
	}

	// Create plugin manager
	pluginManager := plugins.NewPluginManager(authManager, deps)
	if err := pluginManager.LoadPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Get available plugins
	availablePlugins := pluginManager.GetAvailablePlugins(ctx)

	fmt.Printf("Available Plugins (%s tier):\n\n", authManager.GetSubscriptionTier())

	if len(availablePlugins) == 0 {
		fmt.Printf("No plugins available for your subscription tier.\n")
		fmt.Printf("Upgrade at: https://kilometers.ai/pricing\n")
		return nil
	}

	for _, plugin := range availablePlugins {
		status := "✅ Available"
		if !plugin.IsAvailable(ctx) {
			status = "❌ Unavailable"
		}

		fmt.Printf("📦 %s\n", plugin.Name())
		fmt.Printf("   Status: %s\n", status)
		fmt.Printf("   Required: %s\n", plugin.RequiredFeature())
		fmt.Printf("   Tier: %s+\n", plugin.RequiredTier())
		fmt.Printf("\n")
	}

	return nil
}

func runPluginConfig(ctx context.Context, pluginName, command string, args []string) error {
	// Create authentication manager
	authManager := domain.NewAuthenticationManager()
	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	// Create plugin dependencies
	deps := ports.PluginDependencies{
		AuthManager: authManager,
		Config:      domain.LoadConfig(),
	}

	// Create plugin manager
	pluginManager := plugins.NewPluginManager(authManager, deps)
	if err := pluginManager.LoadPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Prepare plugin parameters
	params := ports.PluginParams{
		Command: command,
		Args:    args,
		Data:    make(map[string]interface{}),
	}

	// Parse additional data from args if provided
	// In a real implementation, you'd parse JSON or key-value pairs from args
	if command == "add-rule" {
		params.Data["name"] = "example-rule"
		params.Data["pattern"] = "test.*"
		params.Data["action"] = "warn"
	}

	// Execute plugin command
	result, err := pluginManager.ExecutePlugin(ctx, pluginName, params)
	if err != nil {
		return fmt.Errorf("failed to execute plugin command: %w", err)
	}

	// Display result
	if result.Success {
		fmt.Printf("✅ Plugin command executed successfully\n")
		if result.Data != nil {
			fmt.Printf("Result: %+v\n", result.Data)
		}
	} else {
		fmt.Printf("❌ Plugin command failed\n")
		if result.Error != nil {
			fmt.Printf("Error: %s\n", result.Error.Error())
		}
	}

	return nil
}

func runPluginsStatus(ctx context.Context) error {
	// Create authentication manager
	authManager := domain.NewAuthenticationManager()
	if err := authManager.LoadSubscription(); err != nil {
		return fmt.Errorf("failed to load subscription: %w", err)
	}

	tier := authManager.GetSubscriptionTier()
	features := authManager.ListFeatures(ctx)

	fmt.Printf("Plugin System Status\n")
	fmt.Printf("===================\n\n")
	fmt.Printf("Subscription Tier: %s\n", tier)
	fmt.Printf("Active Features: %d\n\n", len(features))

	// Show feature status
	allFeatures := getAllPossibleFeatures()
	fmt.Printf("Feature Availability:\n")
	for _, feature := range allFeatures {
		status := "❌"
		if authManager.IsFeatureEnabled(feature.Name) {
			status = "✅"
		}
		fmt.Printf("  %s %s (%s)\n", status, feature.Name, feature.RequiredTier)
	}

	// Show subscription status
	fmt.Printf("\nSubscription Status:\n")
	if err := authManager.ValidateSubscription(); err != nil {
		fmt.Printf("  ❌ %s\n", err.Error())
	} else {
		fmt.Printf("  ✅ Valid subscription\n")
	}

	if authManager.NeedsRefresh() {
		fmt.Printf("  ⚠️  Needs refresh (run 'km auth refresh')\n")
	}

	return nil
}

// Helper functions

type FeatureDetail struct {
	Name         string
	RequiredTier string
	Description  string
}

func getAllPossibleFeatures() []FeatureDetail {
	return []FeatureDetail{
		{domain.FeatureBasicMonitoring, "Free", "Basic MCP monitoring"},
		{domain.FeatureAdvancedFilters, "Pro", "Advanced filtering rules"},
		{domain.FeatureCustomRules, "Pro", "Custom processing rules"},
		{domain.FeaturePoisonDetection, "Pro", "AI security analysis"},
		{domain.FeatureMLAnalytics, "Pro", "ML-powered analytics"},
		{domain.FeatureTeamCollaboration, "Enterprise", "Team features"},
		{domain.FeatureCustomDashboards, "Enterprise", "Custom dashboards"},
		{domain.FeatureComplianceReporting, "Enterprise", "Compliance reports"},
		{domain.FeatureAPIIntegrations, "Enterprise", "API integrations"},
		{domain.FeaturePrioritySupport, "Enterprise", "Priority support"},
	}
}
