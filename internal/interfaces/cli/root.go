package cli

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/services"
	"kilometers.ai/cli/internal/infrastructure/config"
)

var (
	Version   = "dev"     // Overridden by ldflags
	BuildTime = "unknown" // Overridden by ldflags
)

// CLIContainer holds all the dependencies for CLI commands
type CLIContainer struct {
	ConfigService     *services.ConfigurationService
	MonitoringService *services.MonitoringService
	ConfigRepo        *config.CompositeConfigRepository
}

// RootCommand represents the base command when called without any subcommands
func NewRootCommand(container *CLIContainer) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "km",
		Short: "Kilometers CLI - MCP Event Monitoring and Analysis",
		Long: `Kilometers CLI is a tool for monitoring Model Context Protocol (MCP) events,
analyzing risks, and providing insights into AI assistant interactions.

It supports monitoring MCP server processes, collecting events, and sending
them to the Kilometers platform for analysis and visualization.`,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Global setup that runs before any command
			return nil
		},
	}

	// Set custom version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("{{.Name}} version {{.Version}}\nBuild time: %s\nGo version: %s\nPlatform: %s/%s\n",
		BuildTime, goVersion(), runtime.GOOS, runtime.GOARCH))

	// Add persistent flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().String("config", "", "Config file path (default is $HOME/.km/config.json)")
	rootCmd.PersistentFlags().String("api-key", "", "API key for Kilometers platform")
	rootCmd.PersistentFlags().String("api-url", "https://api.dev.kilometers.ai", "API endpoint URL")

	// Add subcommands
	rootCmd.AddCommand(NewMonitorCommand(container))
	rootCmd.AddCommand(NewConfigCommand(container))
	rootCmd.AddCommand(NewInitCommand(container))
	rootCmd.AddCommand(NewSetupCommand(container))
	rootCmd.AddCommand(NewValidateCommand(container))
	rootCmd.AddCommand(NewUpdateCommand(container))

	return rootCmd
}

// goVersion returns the Go version used to build the binary
func goVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.GoVersion
	}
	return "unknown"
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(container *CLIContainer) {
	rootCmd := NewRootCommand(container)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
