package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// MonitoringService implements the core monitoring logic
type MonitoringService struct {
	processExecutor ports.ProcessExecutor
	messageLogger   ports.MessageHandler
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(
	executor ports.ProcessExecutor,
	logger ports.MessageHandler,
) *MonitoringService {
	return &MonitoringService{
		processExecutor: executor,
		messageLogger:   logger,
	}
}

// StartMonitoring begins monitoring a new MCP server process
func (s *MonitoringService) StartMonitoring(
	ctx context.Context,
	cmd domain.Command,
	correlationID string,
	config domain.MonitorConfig,
) error {
	// Configure API handler with correlation ID directly
	if apiHandler, ok := s.messageLogger.(interface{ SetCorrelationID(string) }); ok {
		apiHandler.SetCorrelationID(correlationID)
	}

	// Execute the server process
	process, err := s.processExecutor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start server process: %w", err)
	}

	// Start monitoring the process
	go s.monitorProcess(ctx, cmd, correlationID, config, process)

	return nil
}

// monitorProcess handles the monitoring of a running process
func (s *MonitoringService) monitorProcess(
	ctx context.Context,
	cmd domain.Command,
	correlationID string,
	config domain.MonitorConfig,
	process ports.Process,
) {
	// Create a proxy to handle stdin/stdout communication
	proxy := NewStreamProxy(process, correlationID, config, s.messageLogger)

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
		process.Signal(ports.SignalTerminate)

		// Give it a moment to terminate gracefully
		terminateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			process.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process terminated gracefully
		case <-terminateCtx.Done():
			// Force kill if it doesn't terminate
			process.Kill()
		}

		fmt.Fprintf(os.Stderr, "[Monitor] Monitoring cancelled\n")

	case <-s.waitForProcess(process):
		// Process completed naturally
		if process.ExitCode() == 0 {
			fmt.Fprintf(os.Stderr, "[Monitor] Process completed successfully\n")
		} else {
			fmt.Fprintf(os.Stderr, "[Monitor] Process exited with code %d\n", process.ExitCode())
		}
	}

	// Stop the proxy
	proxy.Stop()

	// Flush any pending events before shutdown
	if flushable, ok := s.messageLogger.(interface{ Flush(context.Context) error }); ok {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := flushable.Flush(flushCtx); err != nil {
			fmt.Fprintf(os.Stderr, "[Monitor] Failed to flush pending events: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[Monitor] Flushed pending events\n")
		}
	}
}

// waitForProcess returns a channel that closes when the process completes
func (s *MonitoringService) waitForProcess(process ports.Process) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		process.Wait()
	}()
	return done
}
