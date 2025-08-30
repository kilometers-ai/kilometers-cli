package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
	}

	// Add subcommands
	rootCmd.AddCommand(newMonitorCommand())

	return rootCmd
}
