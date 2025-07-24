package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/core/session"
)

// NewMonitorCommand creates the monitor command
func NewMonitorCommand(container *CLIContainer) *cobra.Command {
	var (
		server      bool
		batchSize   int
		debugReplay string
	)

	var monitorCmd = &cobra.Command{
		Use:   "monitor [flags] --server -- <command> [args...]",
		Short: "Monitor MCP server processes and collect events",
		Long: `Monitor starts an MCP server process and collects events from it.
This is the main functionality for monitoring Model Context Protocol communication.

The --server flag is required, followed by -- and then the complete command to run the MCP server.
This follows standard Unix conventions for separating tool flags from command arguments.

Examples:
  km monitor --server -- npx -y @modelcontextprotocol/server-github
  km monitor --server -- python -m my_mcp_server --port 8080
  km monitor --batch-size 20 --server -- npx -y @modelcontextprotocol/server-linear
  km monitor --server -- npx -y @modelcontextprotocol/server-github --debug-replay file.jsonl

JSON Configuration (for AI agents):
  {
    "mcpServers": {
      "github": {
        "command": "km",
        "args": ["monitor", "--server", "--", "npx", "-y", "@modelcontextprotocol/server-github"]
      }
    }
  }

Press Ctrl+C to stop monitoring.`,
		Args: cobra.ArbitraryArgs, // Allow arguments after --
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(container, cmd, args, server, batchSize, debugReplay)
		},
	}

	// Define monitor flags (simplified)
	monitorCmd.Flags().BoolVar(&server, "server", false, "Required: indicates that everything after -- is the MCP server command")
	monitorCmd.Flags().IntVar(&batchSize, "batch-size", 10, "Number of events to batch before sending")
	monitorCmd.Flags().StringVar(&debugReplay, "debug-replay", "", "Path to debug replay file")

	// Mark --server as required
	monitorCmd.MarkFlagRequired("server")

	return monitorCmd
}

// runMonitor handles the main monitor command execution
func runMonitor(container *CLIContainer, cmd *cobra.Command, args []string, server bool,
	batchSize int, debugReplay string) error {

	// Parse server command from args (everything after -- should be the server command)
	command, commandArgs, err := parseServerCommand(args)
	if err != nil {
		return err
	}

	// Create session config with simplified settings
	sessionConfig := session.SessionConfig{
		BatchSize:      batchSize,
		MaxSessionSize: 0, // No limit
	}

	// Create start command (simplified without filtering)
	startCmd := commands.NewStartMonitoringCommand(command, commandArgs, sessionConfig)
	if debugReplay != "" {
		startCmd.DebugReplayFile = debugReplay
	}

	// Execute start command
	ctx := context.Background()
	result, err := container.MonitoringService.StartMonitoring(ctx, startCmd)
	if err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to start monitoring: %s", result.Message)
	}

	fmt.Printf("✅ Started monitoring: %s %v\n", command, commandArgs)
	if result.Metadata != nil {
		if sessionID, ok := result.Metadata["session_id"]; ok {
			fmt.Printf("Session ID: %s\n", sessionID)
		}
	}
	fmt.Println("Press Ctrl+C to stop monitoring...")

	// Wait for signal to stop
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Get session ID for stop command
	var sessionIDStr string
	if result.Metadata != nil {
		if sessionID, ok := result.Metadata["session_id"]; ok {
			sessionIDStr = fmt.Sprintf("%v", sessionID)
		}
	}

	if sessionIDStr == "" {
		fmt.Println("⚠️  Warning: Could not retrieve session ID")
		return nil
	}

	sessionID, err := session.NewSessionID(sessionIDStr)
	if err != nil {
		fmt.Printf("⚠️  Error creating session ID: %v\n", err)
		return nil
	}

	// Stop monitoring
	stopCmd := commands.NewStopMonitoringCommand(sessionID)
	stopResult, err := container.MonitoringService.StopMonitoring(ctx, stopCmd)
	if err != nil {
		fmt.Printf("⚠️  Error stopping monitoring: %v\n", err)
	} else if !stopResult.Success {
		fmt.Printf("⚠️  Error stopping monitoring: %s\n", stopResult.Message)
	} else {
		fmt.Println("✅ Monitoring stopped")
	}

	return nil
}

// parseServerCommand extracts the server command from the arguments
// Expects: command arg1 arg2 ... (everything from args slice)
func parseServerCommand(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("server command is required after --server --")
	}

	// First argument is the command, rest are arguments
	return args[0], args[1:], nil
}
