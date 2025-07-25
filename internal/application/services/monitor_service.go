package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	httpClient "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
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
	// Create API session and configure handler (non-blocking)
	go s.createApiSession(ctx, correlationID)

	// Execute the server process
	process, err := s.processExecutor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start server process: %w", err)
	}

	// Start monitoring the process
	go s.monitorProcess(ctx, cmd, correlationID, config, process)

	return nil
}

// createApiSession creates a session in the kilometers-api and configures the handler
func (s *MonitoringService) createApiSession(ctx context.Context, correlationID string) {
	// Only create API session if API key is configured
	config := domain.LoadConfig()
	if config.ApiKey == "" {
		return
	}

	apiClient := httpClient.NewApiClient()
	if apiClient == nil {
		return
	}

	sessionResp, err := apiClient.CreateSession(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[API] Failed to create session: %v\n", err)
		return
	}

	// Configure the API handler with the correlation ID
	if apiHandler, ok := s.messageLogger.(interface{ SetCorrelationID(string) }); ok {
		apiHandler.SetCorrelationID(sessionResp.SessionId)
		fmt.Fprintf(os.Stderr, "[API] Created session: %s\n", sessionResp.SessionId)
	}
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
