package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/core/session"
)

// NewMonitorCommand creates the monitor command
func NewMonitorCommand(container *CLIContainer) *cobra.Command {
	var monitorCmd = &cobra.Command{
		Use:   "monitor [command] [args...]",
		Short: "Monitor MCP server processes and collect events",
		Long: `Monitor starts an MCP server process and collects events from it.
This is the main functionality for monitoring Model Context Protocol communication.

Examples:
  km monitor npx @modelcontextprotocol/server-github
  km monitor python -m my_mcp_server
  km monitor ./my-mcp-server --port 8080`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(container, cmd, args)
		},
	}

	// Add monitor-specific flags
	monitorCmd.Flags().Int("batch-size", 10, "Number of events to batch before sending")
	monitorCmd.Flags().Int("flush-interval", 30, "Seconds between batch flushes")
	monitorCmd.Flags().Bool("enable-risk-detection", false, "Enable risk detection analysis")
	monitorCmd.Flags().StringSlice("method-whitelist", []string{}, "Only monitor these methods")
	monitorCmd.Flags().StringSlice("method-blacklist", []string{}, "Exclude these methods from monitoring")
	monitorCmd.Flags().Int("payload-size-limit", 0, "Maximum payload size to capture (0 = no limit)")
	monitorCmd.Flags().Bool("high-risk-only", false, "Only capture high-risk events")
	monitorCmd.Flags().Bool("exclude-ping", true, "Exclude ping/pong messages")
	monitorCmd.Flags().String("min-risk-level", "low", "Minimum risk level to capture (low, medium, high)")
	monitorCmd.Flags().String("debug-replay", "", "Path to debug replay file")
	monitorCmd.Flags().Duration("debug-delay", 500*time.Millisecond, "Delay between replay requests")

	// Add subcommands
	monitorCmd.AddCommand(NewMonitorStartCommand(container))
	monitorCmd.AddCommand(NewMonitorStopCommand(container))
	monitorCmd.AddCommand(NewMonitorStatusCommand(container))
	monitorCmd.AddCommand(NewMonitorFlushCommand(container))

	return monitorCmd
}

// runMonitor handles the main monitor command execution
func runMonitor(container *CLIContainer, cmd *cobra.Command, args []string) error {
	// Get command arguments
	command := args[0]
	commandArgs := args[1:]

	// Create session config from flags
	sessionConfig := createSessionConfigFromFlags(cmd)

	// Create start command
	startCmd := commands.NewStartMonitoringCommand(command, commandArgs, sessionConfig)

	// Set filtering rules
	startCmd.FilteringRules = createFilteringRulesFromFlags(cmd)

	// Set debug replay options
	debugReplayFile, _ := cmd.Flags().GetString("debug-replay")
	debugDelay, _ := cmd.Flags().GetDuration("debug-delay")
	if debugReplayFile != "" {
		startCmd.DebugReplayFile = debugReplayFile
		startCmd.DebugDelay = debugDelay
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

// NewMonitorStartCommand creates the start subcommand
func NewMonitorStartCommand(container *CLIContainer) *cobra.Command {
	var startCmd = &cobra.Command{
		Use:   "start [command] [args...]",
		Short: "Start monitoring a specific MCP server process",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionConfig := createSessionConfigFromFlags(cmd)
			startCmd := commands.NewStartMonitoringCommand(args[0], args[1:], sessionConfig)
			startCmd.FilteringRules = createFilteringRulesFromFlags(cmd)

			// Set debug replay options from parent command
			if parentCmd := cmd.Parent(); parentCmd != nil {
				debugReplayFile, _ := parentCmd.Flags().GetString("debug-replay")
				debugDelay, _ := parentCmd.Flags().GetDuration("debug-delay")
				if debugReplayFile != "" {
					startCmd.DebugReplayFile = debugReplayFile
					startCmd.DebugDelay = debugDelay
				}
			}

			ctx := context.Background()
			result, err := container.MonitoringService.StartMonitoring(ctx, startCmd)
			if err != nil {
				return fmt.Errorf("failed to start monitoring: %w", err)
			}

			if !result.Success {
				return fmt.Errorf("failed to start monitoring: %s", result.Message)
			}

			fmt.Printf("✅ Started monitoring: %s %v\n", args[0], args[1:])
			if result.Metadata != nil {
				if sessionID, ok := result.Metadata["session_id"]; ok {
					fmt.Printf("Session ID: %s\n", sessionID)
				}
			}
			return nil
		},
	}

	return startCmd
}

// NewMonitorStopCommand creates the stop subcommand
func NewMonitorStopCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [session-id]",
		Short: "Stop current monitoring session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionID session.SessionID
			var err error

			if len(args) > 0 {
				sessionID, err = session.NewSessionID(args[0])
				if err != nil {
					return fmt.Errorf("invalid session ID: %w", err)
				}
			} else {
				// Find active session
				// For now, we'll use a placeholder - in real implementation,
				// we'd need to query the service for active sessions
				return fmt.Errorf("session ID is required. Use 'km monitor status' to list active sessions")
			}

			stopCmd := commands.NewStopMonitoringCommand(sessionID)

			ctx := context.Background()
			result, err := container.MonitoringService.StopMonitoring(ctx, stopCmd)
			if err != nil {
				return fmt.Errorf("failed to stop monitoring: %w", err)
			}

			if !result.Success {
				return fmt.Errorf("failed to stop monitoring: %s", result.Message)
			}

			fmt.Println("✅ Monitoring stopped")
			return nil
		},
	}
}

// NewMonitorStatusCommand creates the status subcommand
func NewMonitorStatusCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "status [session-id]",
		Short: "Show current monitoring status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) > 0 {
				// Show specific session status
				sessionID, err := session.NewSessionID(args[0])
				if err != nil {
					return fmt.Errorf("invalid session ID: %w", err)
				}

				statusCmd := commands.NewGetSessionStatusCommand(sessionID)
				statusCmd.IncludeStats = true

				result, err := container.MonitoringService.GetSessionStatus(ctx, statusCmd)
				if err != nil {
					return fmt.Errorf("failed to get session status: %w", err)
				}

				if !result.Success {
					return fmt.Errorf("failed to get session status: %s", result.Message)
				}

				// Print session details
				fmt.Printf("Session Status for %s:\n", args[0])
				if result.Metadata != nil {
					if status, ok := result.Metadata["status"]; ok {
						fmt.Printf("Status: %s\n", status)
					}
				}

			} else {
				// List all active sessions
				listCmd := commands.NewListActiveSessionsCommand()
				listCmd.IncludeStats = true

				result, err := container.MonitoringService.ListActiveSessions(ctx, listCmd)
				if err != nil {
					return fmt.Errorf("failed to list active sessions: %w", err)
				}

				if !result.Success {
					return fmt.Errorf("failed to list active sessions: %s", result.Message)
				}

				fmt.Println("Active Monitoring Sessions:")
				// The result would contain session data to display
				if result.Data == nil {
					fmt.Println("No active sessions")
				} else {
					fmt.Printf("Found active sessions: %v\n", result.Data)
				}
			}

			return nil
		},
	}
}

// NewMonitorFlushCommand creates the flush subcommand
func NewMonitorFlushCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "flush [session-id]",
		Short: "Flush current event batch immediately",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionID session.SessionID
			var err error

			if len(args) > 0 {
				sessionID, err = session.NewSessionID(args[0])
				if err != nil {
					return fmt.Errorf("invalid session ID: %w", err)
				}
			} else {
				return fmt.Errorf("session ID is required. Use 'km monitor status' to list active sessions")
			}

			flushCmd := commands.NewFlushEventsCommand(sessionID, false)

			ctx := context.Background()
			result, err := container.MonitoringService.FlushEvents(ctx, flushCmd)
			if err != nil {
				return fmt.Errorf("failed to flush events: %w", err)
			}

			if !result.Success {
				return fmt.Errorf("failed to flush events: %s", result.Message)
			}

			fmt.Println("✅ Events flushed")
			return nil
		},
	}
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
