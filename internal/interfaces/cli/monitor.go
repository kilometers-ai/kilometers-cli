package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"bytes"
	"flag"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/session"
)

// Add global flags for --server and --config
var (
	serverNameFlag string
	configPathFlag string
)

func init() {
	flag.StringVar(&serverNameFlag, "server", "", "Name of the MCP server to proxy (from config)")
	flag.StringVar(&configPathFlag, "config", "", "Path to km monitor config file")
}

// Helper to find config file
func findConfigFile(configPathFlag string) (string, error) {
	if configPathFlag != "" {
		return configPathFlag, nil
	}
	if env := os.Getenv("KM_CONFIG_PATH"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	defaultPath := filepath.Join(home, ".cursor", "km.json")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}
	return "", fmt.Errorf("Could not find km monitor config file. Set KM_CONFIG_PATH or use --config flag.")
}

// Config struct
type MCPServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	URL     string   `json:"url,omitempty"`
}
type KMConfig struct {
	Servers  []MCPServerConfig `json:"servers"`
	HTTPPort int               `json:"httpPort,omitempty"`
}

// NewMonitorCommand creates the monitor command
func NewMonitorCommand(container *CLIContainer) *cobra.Command {
	var serverNameFlag string
	var configPathFlag string

	var monitorCmd = &cobra.Command{
		Use:   "monitor [command] [args...]",
		Short: "Monitor MCP server processes and collect events",
		Long: `Monitor starts an MCP server process and collects events from it.
This is the main functionality for monitoring Model Context Protocol communication.

Examples:
  km monitor npx @modelcontextprotocol/server-github
  km monitor python -m my_mcp_server
  km monitor ./my-mcp-server --port 8080
  km monitor -- npx -y @modelcontextprotocol/server-sequential-thinking`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check for proxy mode
			serverNameFlag, _ = cmd.Flags().GetString("server")
			configPathFlag, _ = cmd.Flags().GetString("config")
			if serverNameFlag != "" {
				// Proxy mode: look up server in config and launch/forward
				configPath, err := findConfigFile(configPathFlag)
				if err != nil {
					return fmt.Errorf("❌ %v", err)
				}
				f, err := os.Open(configPath)
				if err != nil {
					return fmt.Errorf("❌ Failed to open config: %v", err)
				}
				defer f.Close()
				var kmConfig KMConfig
				if err := json.NewDecoder(f).Decode(&kmConfig); err != nil {
					return fmt.Errorf("❌ Failed to parse config: %v", err)
				}
				var server *MCPServerConfig
				for i, s := range kmConfig.Servers {
					if s.Name == serverNameFlag {
						server = &kmConfig.Servers[i]
						break
					}
				}
				if server == nil {
					return fmt.Errorf("❌ Server '%s' not found in config", serverNameFlag)
				}
				if server.Command != "" {
					cmd := exec.Command(server.Command, server.Args...)
					cmd.Stderr = os.Stderr
					stdin, err := cmd.StdinPipe()
					if err != nil {
						return fmt.Errorf("❌ Failed to get stdin pipe: %v", err)
					}
					stdout, err := cmd.StdoutPipe()
					if err != nil {
						return fmt.Errorf("❌ Failed to get stdout pipe: %v", err)
					}
					if err := cmd.Start(); err != nil {
						return fmt.Errorf("❌ Failed to start MCP server: %v", err)
					}
					go func() {
						io.Copy(stdin, os.Stdin)
						stdin.Close()
					}()
					// Replace the brace-matching loop with strict JSON-only forwarding
					reader := io.Reader(stdout)
					var accumulator []byte
					var braceCount int
					var inString, escapeNext bool
					buf := make([]byte, 8192)
					for {
						n, err := reader.Read(buf)
						if n > 0 {
							accumulator = append(accumulator, buf[:n]...)
							start := 0
							for i := 0; i < len(accumulator); i++ {
								b := accumulator[i]
								if inString {
									if escapeNext {
										escapeNext = false
									} else if b == '\\' {
										escapeNext = true
									} else if b == '"' {
										inString = false
									}
								} else {
									if b == '"' {
										inString = true
									} else if b == '{' {
										braceCount++
									} else if b == '}' {
										braceCount--
										if braceCount == 0 {
											obj := accumulator[start : i+1]
											os.Stdout.Write(obj)
											os.Stdout.Write([]byte{'\n'})
											os.Stdout.Sync()
											start = i + 1
										}
									} else if b == '\n' {
										// Check for non-JSON lines (banner or logs)
										line := accumulator[start : i+1]
										if bytes.Contains(line, []byte("MCP Server running on stdio")) {
											os.Stdout.Write(line)
											os.Stdout.Sync()
										} else if len(bytes.TrimSpace(line)) > 0 {
											os.Stderr.Write([]byte("[km-proxy] Non-protocol output dropped: "))
											os.Stderr.Write(line)
										}
										start = i + 1
									}
								}
							}
							if start > 0 {
								accumulator = accumulator[start:]
							}
							if len(accumulator) > 10*1024*1024 {
								accumulator = nil
								braceCount = 0
								inString = false
								escapeNext = false
							}
						}
						if err != nil {
							break
						}
					}
					cmd.Wait()
					return nil
				} else if server.URL != "" {
					return fmt.Errorf("❌ Remote URL proxying not implemented yet")
				} else {
					return fmt.Errorf("❌ Server config must have either 'command' or 'url'")
				}
			}
			// Otherwise, run normal CLI logic
			return runMonitor(container, cmd, args)
		},
	}

	monitorCmd.Flags().SetInterspersed(false)
	monitorCmd.Flags().String("server", "", "Name of the MCP server to proxy (from config)")
	monitorCmd.Flags().String("config", "", "Path to km monitor config file")

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

	// Execute start command
	ctx := context.Background()
	result, err := container.MonitoringService.StartMonitoring(ctx, startCmd)
	if err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to start monitoring: %s", result.Message)
	}

	fmt.Fprintf(os.Stderr, "✅ Started monitoring: %s %v\n", command, commandArgs)
	if result.Metadata != nil {
		if sessionID, ok := result.Metadata["session_id"]; ok {
			fmt.Fprintf(os.Stderr, "Session ID: %s\n", sessionID)
		}
	}
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop monitoring...")

	// Get the process monitor from the DI container for transparent proxy mode
	if container.ProcessMonitor != nil {
		processMonitor := container.ProcessMonitor
		fmt.Fprintf(os.Stderr, "🔧 Starting transparent proxy mode with ProcessMonitor\n")

		// MonitoringService now handles stdout forwarding automatically
		// We only need to handle stdin forwarding here
		go forwardStdinToProcess(processMonitor)

	} else {
		fmt.Fprintf(os.Stderr, "⚠️  ProcessMonitor is nil - transparent proxy mode disabled\n")
	}

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
		fmt.Fprintln(os.Stderr, "⚠️  Warning: Could not retrieve session ID")
		return nil
	}

	sessionID, err := session.NewSessionID(sessionIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Error creating session ID: %v\n", err)
		return nil
	}

	// Stop monitoring
	stopCmd := commands.NewStopMonitoringCommand(sessionID)
	stopResult, err := container.MonitoringService.StopMonitoring(ctx, stopCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Error stopping monitoring: %v\n", err)
	} else if !stopResult.Success {
		fmt.Fprintf(os.Stderr, "⚠️  Error stopping monitoring: %s\n", stopResult.Message)
	} else {
		fmt.Fprintln(os.Stderr, "✅ Monitoring stopped")
	}

	return nil
}

type processStdinWriter struct {
	pm ports.ProcessMonitor
}

func (w *processStdinWriter) Write(p []byte) (int, error) {
	err := w.pm.WriteStdin(p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// forwardStdinToProcess reads from os.Stdin and forwards to the monitored process
func forwardStdinToProcess(processMonitor ports.ProcessMonitor) {
	fmt.Fprintf(os.Stderr, "🔧 Starting stdin forwarding (streaming mode)\n")

	if !processMonitor.IsRunning() {
		fmt.Fprintf(os.Stderr, "⚠️  Process not running, stdin forwarding stopping\n")
		return
	}

	writer := &processStdinWriter{pm: processMonitor}
	_, err := io.Copy(writer, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error streaming stdin: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "🔚 Stdin forwarding: EOF detected, exiting goroutine\n")
}

// NewMonitorStartCommand creates the start subcommand
func NewMonitorStartCommand(container *CLIContainer) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start [command] [args...]",
		Short: "Start monitoring a specific MCP server process",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionConfig := createSessionConfigFromFlags(cmd)
			startCmd := commands.NewStartMonitoringCommand(args[0], args[1:], sessionConfig)
			startCmd.FilteringRules = createFilteringRulesFromFlags(cmd)

			ctx := context.Background()
			result, err := container.MonitoringService.StartMonitoring(ctx, startCmd)
			if err != nil {
				return fmt.Errorf("failed to start monitoring: %w", err)
			}

			if !result.Success {
				return fmt.Errorf("failed to start monitoring: %s", result.Message)
			}

			fmt.Fprintf(os.Stderr, "✅ Started monitoring: %s %v\n", args[0], args[1:])
			if result.Metadata != nil {
				if sessionID, ok := result.Metadata["session_id"]; ok {
					fmt.Fprintf(os.Stderr, "Session ID: %s\n", sessionID)
				}
			}
			return nil
		},
	}

	// Tell Cobra to stop parsing flags after the first positional argument
	startCmd.Flags().SetInterspersed(false)

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

			fmt.Fprintln(os.Stderr, "✅ Monitoring stopped")
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
				fmt.Fprintf(os.Stderr, "Session Status for %s:\n", args[0])
				if result.Metadata != nil {
					if status, ok := result.Metadata["status"]; ok {
						fmt.Fprintf(os.Stderr, "Status: %s\n", status)
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

				fmt.Fprintln(os.Stderr, "Active Monitoring Sessions:")
				// The result would contain session data to display
				if result.Data == nil {
					fmt.Fprintln(os.Stderr, "No active sessions")
				} else {
					fmt.Fprintf(os.Stderr, "Found active sessions: %v\n", result.Data)
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

			fmt.Fprintln(os.Stderr, "✅ Events flushed")
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
