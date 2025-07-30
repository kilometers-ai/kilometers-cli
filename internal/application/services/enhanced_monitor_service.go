package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins"
)

// EnhancedMonitoringService implements monitoring with plugin support
type EnhancedMonitoringService struct {
	processExecutor ports.ProcessExecutor
	messageLogger   ports.MessageHandler
	pluginManager   ports.PluginManager
	authManager     *domain.AuthenticationManager
}

// NewEnhancedMonitoringService creates a new enhanced monitoring service
func NewEnhancedMonitoringService(
	executor ports.ProcessExecutor,
	logger ports.MessageHandler,
	authManager *domain.AuthenticationManager,
) *EnhancedMonitoringService {
	// Create plugin dependencies
	deps := ports.PluginDependencies{
		AuthManager:   authManager,
		MessageLogger: logger,
		Config:        domain.LoadConfig(),
	}

	// Create plugin manager
	pluginManager := plugins.NewPluginManager(authManager, deps)

	return &EnhancedMonitoringService{
		processExecutor: executor,
		messageLogger:   logger,
		pluginManager:   pluginManager,
		authManager:     authManager,
	}
}

// StartMonitoring begins monitoring with plugin support
func (s *EnhancedMonitoringService) StartMonitoring(
	ctx context.Context,
	cmd domain.Command,
	correlationID string,
	config domain.MonitorConfig,
) error {
	// Load plugins based on subscription
	if err := s.pluginManager.LoadPlugins(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[Monitor] Warning: Failed to load plugins: %v\n", err)
		// Continue without plugins
	}

	// Display subscription status and available features
	s.displaySubscriptionInfo(ctx)

	// Configure API handler with correlation ID
	if apiHandler, ok := s.messageLogger.(interface{ SetCorrelationID(string) }); ok {
		apiHandler.SetCorrelationID(correlationID)
	}

	// Execute the server process
	process, err := s.processExecutor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to start server process: %w", err)
	}

	// Start monitoring with plugin integration
	go s.monitorProcessWithPlugins(ctx, cmd, correlationID, config, process)

	return nil
}

// monitorProcessWithPlugins handles monitoring with plugin integration
func (s *EnhancedMonitoringService) monitorProcessWithPlugins(
	ctx context.Context,
	cmd domain.Command,
	correlationID string,
	config domain.MonitorConfig,
	process ports.Process,
) {
	// Create enhanced proxy with plugin support
	proxy := NewEnhancedStreamProxy(
		process,
		correlationID,
		config,
		s.messageLogger,
		s.pluginManager,
	)

	// Start the proxy
	proxyCtx, proxyCancel := context.WithCancel(ctx)
	defer proxyCancel()

	go func() {
		if err := proxy.Start(proxyCtx); err != nil {
			fmt.Fprintf(os.Stderr, "[Monitor] Proxy error: %v\n", err)
		}
	}()

	// Wait for completion or cancellation
	select {
	case <-ctx.Done():
		// Graceful shutdown
		s.handleGracefulShutdown(process)
		fmt.Fprintf(os.Stderr, "[Monitor] Monitoring cancelled\n")

	case <-s.waitForProcess(process):
		// Process completed
		if process.ExitCode() == 0 {
			fmt.Fprintf(os.Stderr, "[Monitor] Process completed successfully\n")
		} else {
			fmt.Fprintf(os.Stderr, "[Monitor] Process exited with code %d\n", process.ExitCode())
		}
	}

	// Cleanup
	proxy.Stop()
	s.flushAndCleanup(ctx)

	// Generate plugin reports if available
	s.generatePluginReports(ctx)
}

// displaySubscriptionInfo shows current subscription and available features
func (s *EnhancedMonitoringService) displaySubscriptionInfo(ctx context.Context) {
	tier := s.authManager.GetSubscriptionTier()
	availablePlugins := s.pluginManager.GetAvailablePlugins(ctx)

	fmt.Fprintf(os.Stderr, "[Monitor] Subscription: %s\n", tier)
	
	if len(availablePlugins) > 0 {
		fmt.Fprintf(os.Stderr, "[Monitor] Active plugins: ")
		for i, plugin := range availablePlugins {
			if i > 0 {
				fmt.Fprintf(os.Stderr, ", ")
			}
			fmt.Fprintf(os.Stderr, "%s", plugin.Name())
		}
		fmt.Fprintf(os.Stderr, "\n")
	} else {
		fmt.Fprintf(os.Stderr, "[Monitor] No premium plugins available (Free tier)\n")
	}
	
	fmt.Fprintf(os.Stderr, "[Monitor] Starting monitoring...\n")
}

// handleGracefulShutdown performs graceful process termination
func (s *EnhancedMonitoringService) handleGracefulShutdown(process ports.Process) {
	process.Signal(ports.SignalTerminate)

	terminateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		process.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Graceful termination
	case <-terminateCtx.Done():
		// Force kill
		process.Kill()
	}
}

// flushAndCleanup performs final cleanup operations
func (s *EnhancedMonitoringService) flushAndCleanup(ctx context.Context) {
	// Flush message logger
	if flushable, ok := s.messageLogger.(interface{ Flush(context.Context) error }); ok {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := flushable.Flush(flushCtx); err != nil {
			fmt.Fprintf(os.Stderr, "[Monitor] Failed to flush pending events: %v\n", err)
		}
	}

	// Cleanup plugins
	for _, plugin := range s.pluginManager.GetAvailablePlugins(ctx) {
		if err := plugin.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "[Monitor] Plugin cleanup error (%s): %v\n", plugin.Name(), err)
		}
	}
}

// generatePluginReports creates summary reports from active plugins
func (s *EnhancedMonitoringService) generatePluginReports(ctx context.Context) {
	// Security report
	securityPlugins := s.pluginManager.(*plugins.PluginManagerImpl).GetSecurityPlugins(ctx)
	for _, securityPlugin := range securityPlugins {
		report, err := securityPlugin.GetSecurityReport(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[Security] Failed to generate report: %v\n", err)
			continue
		}

		fmt.Fprintf(os.Stderr, "\n[Security Report]\n")
		fmt.Fprintf(os.Stderr, "Total Messages: %d\n", report.TotalMessages)
		fmt.Fprintf(os.Stderr, "Security Issues: %d\n", len(report.SecurityIssues))
		
		if len(report.SecurityIssues) > 0 {
			fmt.Fprintf(os.Stderr, "Risk Distribution: %+v\n", report.RiskDistribution)
		}
		
		if len(report.Recommendations) > 0 {
			fmt.Fprintf(os.Stderr, "Recommendations:\n")
			for _, rec := range report.Recommendations {
				fmt.Fprintf(os.Stderr, "  - %s\n", rec)
			}
		}
	}

	// Analytics report
	analyticsPlugins := s.pluginManager.(*plugins.PluginManagerImpl).GetAnalyticsPlugins(ctx)
	for _, analyticsPlugin := range analyticsPlugins {
		analytics, err := analyticsPlugin.GetAnalytics(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[Analytics] Failed to get analytics: %v\n", err)
			continue
		}

		fmt.Fprintf(os.Stderr, "\n[Analytics Report]\n")
		for key, value := range analytics {
			fmt.Fprintf(os.Stderr, "%s: %v\n", key, value)
		}
	}
}

// waitForProcess returns a channel that closes when the process completes
func (s *EnhancedMonitoringService) waitForProcess(process ports.Process) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		process.Wait()
	}()
	return done
}
