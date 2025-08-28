package monitoring

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/process"
	procp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/process"
	streamp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/streaming"
	infrasm "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/streaming"
)

// Service implements the core monitoring logic
type Service struct {
	processExecutor procp.Executor
	messageLogger   streamp.MessageHandler
}

// NewService creates a new monitoring service
func NewService(
	executor procp.Executor,
	logger streamp.MessageHandler,
) *Service {
	return &Service{
		processExecutor: executor,
		messageLogger:   logger,
	}
}

// StartMonitoring begins monitoring a new MCP server process
func (s *Service) StartMonitoring(
	ctx context.Context,
	cmd process.Command,
	correlationID string,
	config config.MonitorConfig,
) error {
	// Set correlation ID for plugins that support it
	if setter, ok := s.messageLogger.(interface{ SetCorrelationID(string) }); ok {
		setter.SetCorrelationID(correlationID)
	}

	// Execute the server process
	proc, err := s.processExecutor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start server process: %w", err)
	}

	// Start monitoring the process
	go s.monitorProcess(ctx, cmd, correlationID, config, proc)

	return nil
}

// monitorProcess handles the monitoring of a running process
func (s *Service) monitorProcess(
	ctx context.Context,
	cmd process.Command,
	correlationID string,
	config config.MonitorConfig,
	proc procp.Process,
) {
	// Create a proxy to handle stdin/stdout communication
	proxy := infrasm.NewStdioProxy(proc, correlationID, config, s.messageLogger)

	// Start the proxy in a separate goroutine
	proxyCtx, proxyCancel := context.WithCancel(ctx)
	defer proxyCancel()

	go func() {
		if err := proxy.Start(proxyCtx); err != nil {
			fmt.Fprintf(os.Stderr, "[Monitor] Proxy error: %v\n", err)
		}
	}()

	// Wait for the process to complete or context cancellation
	select {
	case <-ctx.Done():
		// Context cancelled, signal the process to terminate
		proc.Signal(process.SignalTerminate)

		// Give it a moment to terminate gracefully
		terminateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			proc.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process terminated gracefully
		case <-terminateCtx.Done():
			// Force kill if it doesn't terminate
			proc.Kill()
		}

		fmt.Fprintf(os.Stderr, "[Monitor] Monitoring cancelled\n")

	case <-s.waitForProcess(proc):
		// Process completed naturally
		if proc.ExitCode() == 0 {
			fmt.Fprintf(os.Stderr, "[Monitor] Process completed successfully\n")
		} else {
			fmt.Fprintf(os.Stderr, "[Monitor] Process exited with code %d\n", proc.ExitCode())
		}
	}

	// Stop the proxy
	proxy.Stop()

	// Note: Plugin shutdown/flush is now handled by the plugin manager
}

// waitForProcess returns a channel that closes when the process completes
func (s *Service) waitForProcess(process procp.Process) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		process.Wait()
	}()
	return done
}
