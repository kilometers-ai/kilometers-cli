package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	config2 "github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/process"
	procp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/process"
	streamp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/streaming"
	infraproc "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/process"
	"github.com/kilometers-ai/kilometers-cli/internal/logging"
	"github.com/kilometers-ai/kilometers-cli/internal/monitoring"
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

	// Create process command object
	processCmd, err := process.NewCommand(flags.ServerCommand[0], flags.ServerCommand[1:])
	if err != nil {
		return fmt.Errorf("failed to create command: %w", err)
	}

	// Validate command before proceeding
	if err := processCmd.IsValid(); err != nil {
		return fmt.Errorf("invalid server command: %w", err)
	}

	// Generate correlation ID for this monitoring run
	correlationID := fmt.Sprintf("monitor_%d", time.Now().UnixNano())

	// Start monitoring
	return startMonitoring(ctx, processCmd, correlationID, config)
}

// Factory functions for creating monitoring infrastructure

func createProcessExecutor() procp.Executor {
	return infraproc.NewExecutor()
}

func createMessageLogger(config config2.MonitorConfig) (streamp.MessageHandler, error) {
	loader, storage, err := config2.CreateConfigServiceFromDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to create config service: %w", err)
	}
	configService := config2.NewConfigService(loader, storage)

	ctx := context.Background()
	appConfig, err := configService.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if appConfig.HasAPIKey() {
		if appConfig.Debug {
			fmt.Println("Plugin system disabled for testing, using console logger.")
		}
		return logging.NewConsoleLogger(), nil
	}

	return logging.NewConsoleLogger(), nil
}

func createMonitoringService(
	executor procp.Executor,
	messageHandler streamp.MessageHandler,
) *monitoring.Service {
	// The existing monitoring service might need to be adapted to the new interfaces.
	// For now, this demonstrates the wiring.
	return monitoring.NewService(executor, messageHandler)
}

func initializePlugins(ctx context.Context, logger streamp.MessageHandler) error {
	return nil // Disabled for testing
}

func shutdownPlugins(ctx context.Context, logger streamp.MessageHandler) error {
	return nil // Disabled for testing
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
func startMonitoring(ctx context.Context, cmd process.Command, correlationID string, config config2.MonitorConfig) error {
	executor := createProcessExecutor()
	logger, err := createMessageLogger(config)
	if err != nil {
		return fmt.Errorf("failed to create message logger: %w", err)
	}

	if err := initializePlugins(ctx, logger); err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := shutdownPlugins(shutdownCtx, logger); err != nil {
			fmt.Printf("[Warning] Failed to shutdown plugins: %v\n", err)
		}
	}()

	monitoringService := createMonitoringService(executor, logger)

	// The monitoringService.StartMonitoring will need to be updated to accept the process.Command
	if err := monitoringService.StartMonitoring(ctx, cmd, correlationID, config); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	<-ctx.Done()

	return nil
}
