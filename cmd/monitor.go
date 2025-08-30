package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
	"github.com/kilometers-ai/kilometers-cli/internal/proxy"
	"github.com/spf13/cobra"
)

// newMonitorCommand creates the monitor subcommand
func newMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor [flags] -- <mcp-server-command> [mcp-server-args...]",
		Short: "Monitor and proxy MCP server communications",
		Long: `Monitor intercepts and proxies MCP server communications, applying
security policies and logging through a plugin system.

Example:
  km monitor -- npx @modelcontextprotocol/server-postgresql
  km monitor --plugins-dir ./plugins -- python -m mcp_server_filesystem`,
		Args: cobra.MinimumNArgs(1),
		RunE: runMonitor,
	}

	return cmd
}

func runMonitor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize plugin manager
	pluginManager, err := plugins.NewPluginManager(cfg.PluginsDir, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize plugin manager: %w", err)
	}
	defer pluginManager.Shutdown()

	//Load plugins
	if err := pluginManager.LoadPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Parse MCP server command
	mcpCmd := strings.Join(args, " ")
	fmt.Printf("Starting MCP proxy for: %s\n", mcpCmd)

	// Start MCP server process
	mcpProcess := exec.CommandContext(ctx, "/bin/sh", "-c", mcpCmd)
	mcpProcess.Stderr = os.Stderr // Pass through stderr for debugging

	// Create MCP proxy
	mcpProxy, err := proxy.NewMCPProxy(pluginManager)
	if err != nil {
		return fmt.Errorf("failed to create MCP proxy: %w", err)
	}

	// Start proxying
	if err := mcpProxy.Start(ctx, mcpProcess); err != nil {
		return fmt.Errorf("proxy failed: %w", err)
	}

	return nil
}

// // runMonitorCommand executes the monitor command
// func runMonitorCommand(cmd *cobra.Command, args []string) error {
// 	// ctx := cmd.Context()

// 	// // Parse flags manually to handle --server -- syntax
// 	// // flags, err := ParseMonitorFlags(args)
// 	// // if err != nil {
// 	// // 	return fmt.Errorf("failed to parse flags: %w", err)
// 	// // }

// 	// // Convert to domain config
// 	// config, err := flags.ToMonitorConfig()
// 	// if err != nil {
// 	// 	return fmt.Errorf("failed to build monitor config: %w", err)
// 	// }

// 	// // Create domain command object
// 	// domainCmd, err := domain.NewCommand(flags.ServerCommand[0], flags.ServerCommand[1:])
// 	// if err != nil {
// 	// 	return fmt.Errorf("failed to create command: %w", err)
// 	// }

// 	// // Validate command before proceeding
// 	// if err := domainCmd.IsValid(); err != nil {
// 	// 	return fmt.Errorf("invalid server command: %w", err)
// 	// }

// 	// // Generate correlation ID for this monitoring run
// 	// correlationID := fmt.Sprintf("monitor_%d", time.Now().UnixNano())

// 	// // Start monitoring
// 	// return startMonitoring(ctx, domainCmd, correlationID, config)
// 	return nil
// }

// // Factory functions for creating monitoring infrastructure

// // createProcessExecutor creates a new process executor
// func createProcessExecutor() ports.ProcessExecutor {
// 	return process.NewExecutor()
// }

// // createMessageLogger creates a message logger based on configuration
// func createMessageLogger(config domain.MonitorConfig) ports.MessageHandler {
// 	// Create console logger as the base handler
// 	consoleLogger := logging.NewConsoleLogger()

// 	// If API key is configured, wrap with API handler
// 	appConfig := domain.LoadConfig()
// 	if appConfig.ApiKey != "" {
// 		return logging.NewApiHandler(consoleLogger)
// 	}

// 	return consoleLogger
// }

// // createMonitoringService creates the monitoring service with all dependencies
// func createMonitoringService(
// 	executor ports.ProcessExecutor,
// 	logger ports.MessageHandler,
// ) *services.MonitoringService {
// 	return services.NewMonitoringService(executor, logger)
// }

// // parseBufferSize converts string buffer size to bytes
// func parseBufferSize(sizeStr string) (int, error) {
// 	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

// 	// Handle different units
// 	multiplier := 1
// 	if strings.HasSuffix(sizeStr, "KB") {
// 		multiplier = 1024
// 		sizeStr = strings.TrimSuffix(sizeStr, "KB")
// 	} else if strings.HasSuffix(sizeStr, "MB") {
// 		multiplier = 1024 * 1024
// 		sizeStr = strings.TrimSuffix(sizeStr, "MB")
// 	} else if strings.HasSuffix(sizeStr, "GB") {
// 		multiplier = 1024 * 1024 * 1024
// 		sizeStr = strings.TrimSuffix(sizeStr, "GB")
// 	}

// 	// Parse the numeric part
// 	var size int
// 	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
// 		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
// 	}

// 	return size * multiplier, nil
// }

// // startMonitoring begins the monitoring process using the monitoring service
// func startMonitoring(ctx context.Context, cmd domain.Command, correlationID string, config domain.MonitorConfig) error {
// 	// Create the monitoring infrastructure
// 	executor := createProcessExecutor()
// 	logger := createMessageLogger(config)

// 	// Create the monitoring service
// 	monitoringService := createMonitoringService(executor, logger)

// 	// Start monitoring
// 	if err := monitoringService.StartMonitoring(ctx, cmd, correlationID, config); err != nil {
// 		return fmt.Errorf("failed to start monitoring: %w", err)
// 	}

// 	// Wait for context cancellation (Ctrl+C)
// 	<-ctx.Done()

// 	return nil
// }
