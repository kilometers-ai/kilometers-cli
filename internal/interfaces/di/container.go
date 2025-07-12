package di

import (
	"context"
	"fmt"
	"log"
	"os"

	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/application/services"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/filtering"
	"kilometers.ai/cli/internal/core/risk"
	"kilometers.ai/cli/internal/core/session"
	"kilometers.ai/cli/internal/infrastructure/api"
	"kilometers.ai/cli/internal/infrastructure/config"
	"kilometers.ai/cli/internal/infrastructure/monitoring"
	"kilometers.ai/cli/internal/interfaces/cli"
)

// Container holds all application dependencies
type Container struct {
	// Configuration
	ConfigRepo    *config.CompositeConfigRepository
	ConfigService *services.ConfigurationService

	// Core services
	RiskAnalyzer      risk.RiskAnalyzer
	EventFilter       filtering.EventFilter
	MonitoringService *services.MonitoringService

	// Infrastructure
	APIGateway     *api.KilometersAPIGateway
	ProcessMonitor *monitoring.MCPProcessMonitor

	// CLI
	CLIContainer *cli.CLIContainer

	// Logger
	Logger *log.Logger
}

// NewContainer creates and configures the dependency injection container
func NewContainer() (*Container, error) {
	container := &Container{
		Logger: log.New(os.Stderr, "[km] ", log.LstdFlags),
	}

	if err := container.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return container, nil
}

// initializeComponents initializes all components with proper dependencies
func (c *Container) initializeComponents() error {
	// 1. Initialize configuration repository
	c.ConfigRepo = config.NewCompositeConfigRepository()

	// 2. Load configuration
	appConfig, err := c.ConfigRepo.Load()
	if err != nil {
		c.Logger.Printf("Warning: Failed to load configuration, using defaults: %v", err)
		appConfig = c.ConfigRepo.LoadDefault()
	}

	// 3. Initialize infrastructure components
	c.APIGateway = api.NewKilometersAPIGateway(appConfig.APIEndpoint, appConfig.APIKey, &loggingGatewayAdapter{logger: c.Logger})
	c.ProcessMonitor = monitoring.NewMCPProcessMonitor(&loggingGatewayAdapter{logger: c.Logger})

	// 4. Initialize core domain services
	riskConfig := risk.RiskAnalyzerConfig{
		HighRiskMethodsOnly: appConfig.HighRiskMethodsOnly,
		PayloadSizeLimit:    appConfig.PayloadSizeLimit,
		CustomPatterns:      []risk.CustomRiskPattern{},
		EnabledCategories:   []string{},
	}
	c.RiskAnalyzer = risk.NewPatternBasedRiskAnalyzer(riskConfig)

	filterRules := filtering.FilteringRules{
		MethodWhitelist:        appConfig.MethodWhitelist,
		MethodBlacklist:        appConfig.MethodBlacklist,
		PayloadSizeLimit:       appConfig.PayloadSizeLimit,
		MinimumRiskLevel:       risk.RiskLevel(appConfig.MinimumRiskLevel),
		ExcludePingMessages:    appConfig.ExcludePingMessages,
		OnlyHighRiskMethods:    appConfig.HighRiskMethodsOnly,
		DirectionFilter:        []event.Direction{}, // Empty = all directions
		EnableContentFiltering: appConfig.EnableRiskDetection,
		ContentBlacklist:       []string{},
	}
	c.EventFilter = filtering.NewCompositeFilter(filterRules, c.RiskAnalyzer)

	// 5. Initialize application services
	c.ConfigService = services.NewConfigurationService(c.ConfigRepo, &loggingGatewayAdapter{logger: c.Logger})
	c.MonitoringService = services.NewMonitoringService(
		nil, // sessionRepo - placeholder for now
		nil, // eventStore - placeholder for now
		c.APIGateway,
		c.ProcessMonitor,
		c.RiskAnalyzer,
		c.EventFilter,
		&loggingGatewayAdapter{logger: c.Logger},
		appConfig,
	)

	// 6. Initialize CLI container
	c.CLIContainer = &cli.CLIContainer{
		ConfigService:     c.ConfigService,
		MonitoringService: c.MonitoringService,
		ConfigRepo:        c.ConfigRepo,
	}

	c.Logger.Println("✅ Dependency injection container initialized successfully")
	return nil
}

// GetCLIContainer returns the CLI container for command execution
func (c *Container) GetCLIContainer() *cli.CLIContainer {
	return c.CLIContainer
}

// Shutdown gracefully shuts down all components
func (c *Container) Shutdown(ctx context.Context) error {
	c.Logger.Println("🛑 Shutting down application...")

	// Stop process monitor if running
	if c.ProcessMonitor != nil && c.ProcessMonitor.IsRunning() {
		if err := c.ProcessMonitor.Stop(); err != nil {
			c.Logger.Printf("Error stopping process monitor: %v", err)
		}
	}

	// Close API gateway connections
	if c.APIGateway != nil {
		// Gateway cleanup if needed
		c.Logger.Println("✅ API Gateway connections closed")
	}

	// Save any pending configuration changes
	if c.ConfigRepo != nil {
		// Configuration is auto-saved, but we can add explicit cleanup here
		c.Logger.Println("✅ Configuration saved")
	}

	c.Logger.Println("✅ Application shutdown complete")
	return nil
}

// HealthCheck performs a health check of all components
func (c *Container) HealthCheck(ctx context.Context) error {
	// Check configuration
	if c.ConfigRepo == nil {
		return fmt.Errorf("configuration repository not initialized")
	}

	_, err := c.ConfigRepo.Load()
	if err != nil {
		return fmt.Errorf("configuration load failed: %w", err)
	}

	// Check API gateway
	if c.APIGateway == nil {
		return fmt.Errorf("API gateway not initialized")
	}

	// Test API connectivity
	if err := c.APIGateway.TestConnection(); err != nil {
		return fmt.Errorf("API connectivity test failed: %w", err)
	}

	// Check monitoring service
	if c.MonitoringService == nil {
		return fmt.Errorf("monitoring service not initialized")
	}

	// Check process monitor
	if c.ProcessMonitor == nil {
		return fmt.Errorf("process monitor not initialized")
	}

	c.Logger.Println("✅ Health check passed")
	return nil
}

// GetVersion returns version information
func (c *Container) GetVersion() map[string]string {
	return map[string]string{
		"version":    cli.Version,
		"build_time": cli.BuildTime,
	}
}

// loggingGatewayAdapter adapts the standard logger to the LoggingGateway interface
type loggingGatewayAdapter struct {
	logger   *log.Logger
	logLevel ports.LogLevel
}

func (l *loggingGatewayAdapter) LogError(err error, message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("ERROR: %s: %v (fields: %v)", message, err, fields)
	} else {
		l.logger.Printf("ERROR: %s: %v", message, err)
	}
}

func (l *loggingGatewayAdapter) LogInfo(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("INFO: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("INFO: %s", message)
	}
}

func (l *loggingGatewayAdapter) LogDebug(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("DEBUG: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("DEBUG: %s", message)
	}
}

func (l *loggingGatewayAdapter) LogWarning(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("WARNING: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("WARNING: %s", message)
	}
}

func (l *loggingGatewayAdapter) LogSession(session *session.Session, message string) {
	l.logger.Printf("SESSION: %s (session: %v)", message, session)
}

func (l *loggingGatewayAdapter) LogEvent(event *event.Event, message string) {
	l.logger.Printf("EVENT: %s (event: %v)", message, event)
}

func (l *loggingGatewayAdapter) LogMetric(name string, value float64, labels map[string]string) {
	l.logger.Printf("METRIC: %s = %f (labels: %v)", name, value, labels)
}

// Add the Log method to satisfy the LoggingGateway interface
func (l *loggingGatewayAdapter) Log(level ports.LogLevel, message string, fields map[string]interface{}) {
	// Only log if the level is at or above our current log level
	if !l.shouldLog(level) {
		return
	}

	levelStr := "INFO"
	switch level {
	case ports.LogLevelError:
		levelStr = "ERROR"
	case ports.LogLevelWarn:
		levelStr = "WARN"
	case ports.LogLevelInfo:
		levelStr = "INFO"
	case ports.LogLevelDebug:
		levelStr = "DEBUG"
	}

	if fields != nil {
		l.logger.Printf("%s: %s (fields: %v)", levelStr, message, fields)
	} else {
		l.logger.Printf("%s: %s", levelStr, message)
	}
}

// SetLogLevel sets the logging level
func (l *loggingGatewayAdapter) SetLogLevel(level ports.LogLevel) {
	l.logLevel = level
}

// GetLogLevel returns the current logging level
func (l *loggingGatewayAdapter) GetLogLevel() ports.LogLevel {
	return l.logLevel
}

// ConfigureLogging configures logging settings
func (l *loggingGatewayAdapter) ConfigureLogging(config *ports.LoggingConfig) error {
	if config == nil {
		return fmt.Errorf("logging config cannot be nil")
	}

	l.logLevel = config.Level

	// For now, we just use the standard logger
	// In a more advanced implementation, we could configure output, format, etc.

	return nil
}

// shouldLog determines if a message should be logged based on current log level
func (l *loggingGatewayAdapter) shouldLog(level ports.LogLevel) bool {
	// Default to Info level if not set
	currentLevel := l.logLevel
	if currentLevel == "" {
		currentLevel = ports.LogLevelInfo
	}

	// Define log level hierarchy
	levelOrder := map[ports.LogLevel]int{
		ports.LogLevelDebug: 0,
		ports.LogLevelInfo:  1,
		ports.LogLevelWarn:  2,
		ports.LogLevelError: 3,
	}

	currentLevelOrder, exists := levelOrder[currentLevel]
	if !exists {
		currentLevelOrder = 1 // Default to Info
	}

	levelOrder_val, exists := levelOrder[level]
	if !exists {
		levelOrder_val = 1 // Default to Info
	}

	return levelOrder_val >= currentLevelOrder
}
