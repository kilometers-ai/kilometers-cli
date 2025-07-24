package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/core/session"
)

// NewMonitorCommand creates the monitor command
func NewMonitorCommand(container *CLIContainer) *cobra.Command {
	var monitorCmd = &cobra.Command{
		Use:   "monitor --server \"command args\"",
		Short: "Monitor MCP server processes and collect events",
		Long: `Monitor starts an MCP server process and collects events from it.
This is the main functionality for monitoring Model Context Protocol communication.

The --server flag is required and should contain the complete command to run the MCP server.
Supports both quoted and unquoted usage.

Supported Usage:
  km monitor --server "npx -y @modelcontextprotocol/server-github"
  km monitor --server npx -y @modelcontextprotocol/server-github
  km monitor --server "python -m my_mcp_server --port 8080"
  km monitor --server python -m my_mcp_server --port 8080
  km monitor --batch-size 20 --server "npx -y @modelcontextprotocol/server-linear"
  km monitor --server npx -y @modelcontextprotocol/server-github --debug-replay file.jsonl

JSON Configuration (for AI agents):
  {
    "mcpServers": {
      "github": {
        "command": "km",
        "args": ["monitor", "--server", "npx -y @modelcontextprotocol/server-github"]
      }
    }
  }

Press Ctrl+C to stop monitoring.`,
		Args: cobra.ArbitraryArgs, // Allow arguments for server command parsing
		// Disable all flag parsing for arguments - we'll handle manually
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(container, cmd, args)
		},
	}

	// Note: Flag parsing is disabled - all arguments are parsed manually

	return monitorCmd
}

// runMonitor handles the main monitor command execution
func runMonitor(container *CLIContainer, cmd *cobra.Command, args []string) error {
	// Process command arguments using manual parsing
	command, commandArgs, err := processMonitorArguments(cmd, args)
	if err != nil {
		return err
	}

	// Parse monitor flags manually and create session config
	sessionConfig, filteringRules, err := parseMonitorFlags(args)
	if err != nil {
		return err
	}

	// Create start command
	startCmd := commands.NewStartMonitoringCommand(command, commandArgs, sessionConfig)

	// Set filtering rules from parsed flags
	startCmd.FilteringRules = filteringRules

	// Debug replay options can be added to parseMonitorFlags if needed

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

// showMonitorHelp displays comprehensive help for the monitor command
func showMonitorHelp() {
	fmt.Print(`Monitor starts an MCP server process and collects events from it.
This is the main functionality for monitoring Model Context Protocol communication.

The --server flag is required and should contain the complete command to run the MCP server.
Supports both quoted and unquoted usage.

USAGE:
  km monitor --server "command args"
  km monitor --server command args
  km monitor --server "command args" [monitor-flags]

EXAMPLES:
  # Quoted server command (recommended for complex commands):
  km monitor --server "npx -y @modelcontextprotocol/server-github"
  km monitor --server "python -m my_mcp_server --port 8080"
  
  # Unquoted server command (works when no conflicting flags):
  km monitor --server npx -y @modelcontextprotocol/server-github
  km monitor --server python -m my_mcp_server --port 8080
  
  # With additional monitor flags:
  km monitor --server "npx -y @modelcontextprotocol/server-github" --batch-size 20
  km monitor --server npx -y @modelcontextprotocol/server-linear --batch-size 5

JSON CONFIGURATION (for AI agents):
  {
    "mcpServers": {
      "github": {
        "command": "km",
        "args": ["monitor", "--server", "npx -y @modelcontextprotocol/server-github"]
      }
    }
  }

MONITOR FLAGS:
  --batch-size int          Number of events to batch before sending (default: 10)
  --debug-replay string     Path to debug replay file

GLOBAL FLAGS:
  --api-key string     API key for Kilometers platform
  --api-url string     API endpoint URL (default "https://api.dev.kilometers.ai")
  --config string      Config file path (default is $HOME/.km/config.json)
  --debug              Enable debug mode
  -h, --help           Show this help message

NOTES:
  - Press Ctrl+C to stop monitoring
  - The --server flag separates km monitor flags from MCP server flags
  - Use quotes around server commands with flags to avoid conflicts

`)
}

// processMonitorArguments manually parses arguments since we disabled Cobra flag parsing
func processMonitorArguments(cmd *cobra.Command, args []string) (string, []string, error) {

	// Handle help flag manually
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showMonitorHelp()
			os.Exit(0) // Exit gracefully after showing help
		}
	}

	// Find --server flag in arguments
	serverIndex := -1
	for i, arg := range args {
		if arg == "--server" {
			serverIndex = i
			break
		}
	}

	if serverIndex == -1 {
		return "", nil, fmt.Errorf("--server flag is required")
	}

	if serverIndex+1 >= len(args) {
		return "", nil, fmt.Errorf("--server flag requires a value")
	}

	// Known monitor flags that should NOT be part of server command
	monitorFlags := map[string]bool{
		"--batch-size": true, "--flush-interval": true, "--enable-risk-detection": true,
		"--method-whitelist": true, "--method-blacklist": true, "--payload-size-limit": true,
		"--high-risk-only": true, "--exclude-ping": true, "--min-risk-level": true,
		"--debug-replay": true, "--debug-delay": true,
	}

	// Find where server command ends (either at next monitor flag or end of args)
	serverEndIndex := len(args)
	for i := serverIndex + 1; i < len(args); i++ {
		if monitorFlags[args[i]] {
			serverEndIndex = i
			break
		}
	}

	// Extract server command parts
	serverCommandParts := args[serverIndex+1 : serverEndIndex]

	if len(serverCommandParts) == 0 {
		return "", nil, fmt.Errorf("--server flag requires a command")
	}

	// Check if first part contains spaces (quoted command with args)
	if strings.Contains(serverCommandParts[0], " ") {
		// Parse the quoted string into command and args
		return parseServerCommand(serverCommandParts[0])
	}

	// First part is command, rest are arguments
	return serverCommandParts[0], serverCommandParts[1:], nil
}

// parseServerCommand parses a server command string into command and arguments
func parseServerCommand(serverCmd string) (string, []string, error) {
	// Simple space splitting since this comes from JSON/CLI, not shell
	parts := strings.Fields(strings.TrimSpace(serverCmd))
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("server command cannot be empty")
	}

	return parts[0], parts[1:], nil
}

// parseMonitorFlags manually parses monitor flags from arguments
func parseMonitorFlags(args []string) (session.SessionConfig, commands.FilteringRulesConfig, error) {
	// Defaults
	sessionConfig := session.SessionConfig{
		BatchSize:           10,
		FlushInterval:       30 * time.Second,
		MaxSessionSize:      0, // No limit
		EnableRiskFiltering: false,
	}

	filteringRules := commands.FilteringRulesConfig{
		MethodWhitelist:        []string{},
		MethodBlacklist:        []string{},
		PayloadSizeLimit:       0,
		MinimumRiskLevel:       "low",
		ExcludePingMessages:    true,
		OnlyHighRiskMethods:    false,
		EnableContentFiltering: false,
		ContentBlacklist:       []string{},
	}

	// Simple parsing for --batch-size flag as example
	for i := 0; i < len(args); i++ {
		if args[i] == "--batch-size" && i+1 < len(args) {
			if batchSize := parseSimpleInt(args[i+1]); batchSize > 0 {
				sessionConfig.BatchSize = batchSize
			}
			i++ // Skip the value
		}
		// Add more flag parsing as needed
	}

	return sessionConfig, filteringRules, nil
}

// parseSimpleInt safely converts string to int for flag values
func parseSimpleInt(s string) int {
	val := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			val = val*10 + int(r-'0')
		} else {
			return 0 // Invalid
		}
	}
	return val
}

// Helper functions to create configurations from flags
func createSessionConfigFromFlags(cmd *cobra.Command) session.SessionConfig {
	batchSize := getBatchSizeFromFlags(cmd)
	flushInterval := getFlushIntervalFromFlags(cmd)

	return session.SessionConfig{
		BatchSize:           batchSize,
		FlushInterval:       time.Duration(flushInterval) * time.Second,
		MaxSessionSize:      0, // No limit
		EnableRiskFiltering: getEnableRiskDetectionFromFlags(cmd),
	}
}

func createFilteringRulesFromFlags(cmd *cobra.Command) commands.FilteringRulesConfig {
	return commands.FilteringRulesConfig{
		MethodWhitelist:        getMethodWhitelistFromFlags(cmd),
		MethodBlacklist:        getMethodBlacklistFromFlags(cmd),
		PayloadSizeLimit:       getPayloadSizeLimitFromFlags(cmd),
		MinimumRiskLevel:       getMinRiskLevelFromFlags(cmd),
		ExcludePingMessages:    getExcludePingFromFlags(cmd),
		OnlyHighRiskMethods:    getHighRiskOnlyFromFlags(cmd),
		EnableContentFiltering: getEnableRiskDetectionFromFlags(cmd),
		ContentBlacklist:       []string{},
	}
}

// Helper functions to extract flags
func getBatchSizeFromFlags(cmd *cobra.Command) int {
	if val, _ := cmd.Flags().GetInt("batch-size"); val > 0 {
		return val
	}
	return 10
}

func getFlushIntervalFromFlags(cmd *cobra.Command) int {
	if val, _ := cmd.Flags().GetInt("flush-interval"); val > 0 {
		return val
	}
	return 30
}

func getEnableRiskDetectionFromFlags(cmd *cobra.Command) bool {
	val, _ := cmd.Flags().GetBool("enable-risk-detection")
	return val
}

func getMethodWhitelistFromFlags(cmd *cobra.Command) []string {
	val, _ := cmd.Flags().GetStringSlice("method-whitelist")
	return val
}

func getMethodBlacklistFromFlags(cmd *cobra.Command) []string {
	val, _ := cmd.Flags().GetStringSlice("method-blacklist")
	return val
}

func getPayloadSizeLimitFromFlags(cmd *cobra.Command) int {
	val, _ := cmd.Flags().GetInt("payload-size-limit")
	return val
}

func getHighRiskOnlyFromFlags(cmd *cobra.Command) bool {
	val, _ := cmd.Flags().GetBool("high-risk-only")
	return val
}

func getExcludePingFromFlags(cmd *cobra.Command) bool {
	val, _ := cmd.Flags().GetBool("exclude-ping")
	return val
}

func getMinRiskLevelFromFlags(cmd *cobra.Command) string {
	if val, _ := cmd.Flags().GetString("min-risk-level"); val != "" {
		return val
	}
	return "low"
}
