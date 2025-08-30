package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/application/services"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/logging"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/process"
	"github.com/spf13/cobra"
)

// newMonitorCommand creates the monitor subcommand
func newMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor --server -- [server-command and args]",
		Short: "Monitor MCP server communication",
		Long: `Monitor MCP server communication by acting as a transparent proxy.

The monitor command starts the specified MCP server and captures all JSON-RPC 
messages flowing between the client and server for debugging and analysis.

Examples:
  # Monitor GitHub MCP server
  km monitor --server -- npx -y @modelcontextprotocol/server-github
  
  # Monitor Python MCP server
  km monitor --server -- python -m my_mcp_server
  
  # Monitor Docker-based MCP server
  km monitor --server -- docker run my-mcp-server

The --server flag must come before the -- separator.`,
		DisableFlagParsing: true, // Parse flags manually to handle -- separator correctly
		RunE:               runMonitorCommand,
	}

	return cmd
}

// runMonitorCommand executes the monitor command
func runMonitorCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse flags manually to handle --server -- syntax
	flags, err := ParseMonitorFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Convert to domain config
	config, err := flags.ToMonitorConfig()
	if err != nil {
		return fmt.Errorf("failed to build monitor config: %w", err)
	}

	// Create domain command object
	domainCmd, err := domain.NewCommand(flags.ServerCommand[0], flags.ServerCommand[1:])
	if err != nil {
		return fmt.Errorf("failed to create command: %w", err)
	}

	// Validate command before proceeding
	if err := domainCmd.IsValid(); err != nil {
		return fmt.Errorf("invalid server command: %w", err)
	}

	// Generate correlation ID for this monitoring run
	correlationID := fmt.Sprintf("monitor_%d", time.Now().UnixNano())

	// Start monitoring
	return startMonitoring(ctx, domainCmd, correlationID, config)
}

// Factory functions for creating monitoring infrastructure

// createProcessExecutor creates a new process executor
func createProcessExecutor() ports.ProcessExecutor {
	return process.NewExecutor()
}

// createMessageLogger creates a message logger based on configuration
func createMessageLogger(config domain.MonitorConfig) ports.MessageHandler {
	// Create console logger as the base handler
	consoleLogger := logging.NewConsoleLogger()

	// If API key is configured, wrap with API handler
	appConfig := domain.LoadConfig()
	if appConfig.ApiKey != "" {
		return logging.NewApiHandler(consoleLogger)
	}

	return consoleLogger
}

// createMonitoringService creates the monitoring service with all dependencies
func createMonitoringService(
	executor ports.ProcessExecutor,
	logger ports.MessageHandler,
) *services.MonitoringService {
	return services.NewMonitoringService(executor, logger)
}

// parseBufferSize converts string buffer size to bytes
func parseBufferSize(sizeStr string) (int, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	// Handle different units
	multiplier := 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	}

	// Parse the numeric part
	var size int
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return size * multiplier, nil
}

// startMonitoring begins the monitoring process using the monitoring service
func startMonitoring(ctx context.Context, cmd domain.Command, correlationID string, config domain.MonitorConfig) error {
	// Create the monitoring infrastructure
	executor := createProcessExecutor()
	logger := createMessageLogger(config)

	// Create the monitoring service
	monitoringService := createMonitoringService(executor, logger)

	// Start monitoring
	if err := monitoringService.StartMonitoring(ctx, cmd, correlationID, config); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Wait for context cancellation (Ctrl+C)
	<-ctx.Done()

	return nil
}
