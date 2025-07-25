package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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
		quiet       bool
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
  km monitor --quiet --server -- npx -y @modelcontextprotocol/server-github

JSON Configuration (for AI agents):
  {
    "mcpServers": {
      "github": {
        "command": "km",
        "args": ["monitor", "--quiet", "--server", "--", "npx", "-y", "@modelcontextprotocol/server-github"]
      }
    }
  }

Use --quiet when wrapping MCP servers to avoid interfering with JSON-RPC communication.
Press Ctrl+C to stop monitoring.`,
		Args: cobra.ArbitraryArgs, // Allow arguments after --
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(container, cmd, args, server, batchSize, debugReplay, quiet)
		},
	}

	// Define monitor flags (simplified)
	monitorCmd.Flags().BoolVar(&server, "server", false, "Required: indicates that everything after -- is the MCP server command")
	monitorCmd.Flags().IntVar(&batchSize, "batch-size", 10, "Number of events to batch before sending")
	monitorCmd.Flags().StringVar(&debugReplay, "debug-replay", "", "Path to debug replay file")
	monitorCmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress status messages (for MCP wrapper mode)")

	// Mark --server as required
	monitorCmd.MarkFlagRequired("server")

	return monitorCmd
}

// runMonitor handles the main monitor command execution
func runMonitor(container *CLIContainer, cmd *cobra.Command, args []string, server bool,
	batchSize int, debugReplay string, quiet bool) error {

	// Parse server command from args (everything after -- should be the server command)
	command, commandArgs, err := parseServerCommand(args)
	if err != nil {
		return err
	}

	// Set up environment variable inheritance FIRST
	// Pass all parent process environment variables to the child MCP server
	processMonitor := container.MonitoringService.GetProcessMonitor()

	// Convert os.Environ() to map for passing to child process
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	processMonitor.SetEnvironment(envMap)

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

	// Output startup messages to stderr in quiet mode, stdout otherwise
	output := os.Stdout
	if quiet {
		output = os.Stderr
	}

	fmt.Fprintf(output, "✅ Started monitoring: %s %v\n", command, commandArgs)
	if result.Metadata != nil {
		if sessionID, ok := result.Metadata["session_id"]; ok {
			fmt.Fprintf(output, "Session ID: %s\n", sessionID)
		}
	}
	fmt.Fprintf(output, "Press Ctrl+C to stop monitoring...\n")

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
		if !quiet {
			fmt.Println("⚠️  Warning: Could not retrieve session ID")
		}
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
	// With cobra.ArbitraryArgs, cobra does not provide a clean way
	// to get only the arguments after --. We need to find the separator.
	// Note: cmd.Flags().ArgsLenAtDash() is not reliable with ArbitraryArgs
	// so we manually parse os.Args, which is simpler and more robust.

	argsAfterDash := []string{}
	foundDash := false
	for _, arg := range os.Args {
		if foundDash {
			argsAfterDash = append(argsAfterDash, arg)
			continue
		}
		if arg == "--" {
			foundDash = true
		}
	}

	if !foundDash {
		// This should ideally be caught by cobra's MarkFlagRequired("server")
		// and the logic that -- is required. But as a safeguard:
		return "", nil, fmt.Errorf("the --server flag requires a server command, separated by --. Usage: km monitor --server -- <command>")
	}

	if len(argsAfterDash) == 0 {
		return "", nil, fmt.Errorf("server command is required after --")
	}

	return argsAfterDash[0], argsAfterDash[1:], nil
}
