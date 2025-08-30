package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the km CLI
func NewRootCommand(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "km",
		Short: "Kilometers CLI - MCP Server Monitoring Proxy",
		Long: `Kilometers CLI (km) is a transparent proxy for Model Context Protocol (MCP) servers 
that captures and logs JSON-RPC communication for debugging and analysis.

The tool acts as middleware between MCP clients and servers, providing complete 
visibility into message flows without disrupting communication.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newMonitorCommand())
	rootCmd.AddCommand(newVersionCommand(version, commit, date))

	return rootCmd
}

// newVersionCommand creates the version subcommand
func newVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("km version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built: %s\n", date)
			fmt.Printf("go version: %s\n", "go1.21+") // TODO: Get actual Go version
		},
	}
}
