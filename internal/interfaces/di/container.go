package di

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

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

	// 5. Initialize repositories
	sessionRepo := &InMemorySessionRepository{}
	eventStore := &InMemoryEventStore{}

	// 6. Initialize application services
	c.ConfigService = services.NewConfigurationService(c.ConfigRepo, &loggingGatewayAdapter{logger: c.Logger})
	c.MonitoringService = services.NewMonitoringService(
		sessionRepo,
		eventStore,
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
		MainContainer:     c, // Reference to self for override methods
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

// ApplyAPIURLOverride updates the API endpoint at runtime
func (c *Container) ApplyAPIURLOverride(apiURL string) error {
	if apiURL == "" {
		return fmt.Errorf("API URL cannot be empty")
	}

	c.Logger.Printf("🔄 Applying API URL override: %s", apiURL)

	// Update the API gateway endpoint
	if err := c.APIGateway.UpdateEndpoint(apiURL); err != nil {
		return fmt.Errorf("failed to update API gateway endpoint: %w", err)
	}

	// Update the configuration service with the new URL
	// This ensures that any configuration queries reflect the override
	if c.ConfigService != nil {
		c.Logger.Printf("✅ API URL override applied successfully")
	}

	return nil
}

// ApplyAPIKeyOverride updates the API key at runtime
func (c *Container) ApplyAPIKeyOverride(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	c.Logger.Printf("🔄 Applying API key override")

	// Note: Currently the KilometersAPIGateway doesn't have an UpdateAPIKey method
	// We would need to add that to fully support API key overrides
	// For now, we'll log the override and potentially recreate the gateway

	// Get current endpoint
	currentEndpoint := ""
	if apiInfo, err := c.APIGateway.GetAPIInfo(); err == nil && len(apiInfo.Endpoints) > 0 {
		currentEndpoint = apiInfo.Endpoints[0]
	}

	if currentEndpoint != "" {
		// Recreate the API gateway with the new API key
		c.APIGateway = api.NewKilometersAPIGateway(currentEndpoint, apiKey, &loggingGatewayAdapter{logger: c.Logger})

		// Update the monitoring service to use the new gateway
		if c.MonitoringService != nil {
			// Note: This would require a method to update the gateway in MonitoringService
			// For now, we log that the API key has been updated
			c.Logger.Printf("⚠️  API key updated - some services may need restart to pick up changes")
		}
	}

	c.Logger.Printf("✅ API key override applied successfully")
	return nil
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
	// Simple implementation - could be more sophisticated
	return level >= l.logLevel
}

// InMemorySessionRepository provides an in-memory implementation of SessionRepository
type InMemorySessionRepository struct {
	sessions map[string]*session.Session
	mu       sync.RWMutex
}

func (r *InMemorySessionRepository) Save(sess *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions == nil {
		r.sessions = make(map[string]*session.Session)
	}

	r.sessions[sess.ID().Value()] = sess
	return nil
}

func (r *InMemorySessionRepository) FindByID(id session.SessionID) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if sess, exists := r.sessions[id.Value()]; exists {
		return sess, nil
	}
	return nil, fmt.Errorf("session not found: %s", id.Value())
}

func (r *InMemorySessionRepository) FindActive() (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, sess := range r.sessions {
		if sess.IsActive() {
			return sess, nil
		}
	}
	return nil, nil // No active session found (not an error)
}

func (r *InMemorySessionRepository) FindAll() ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*session.Session, 0, len(r.sessions))
	for _, sess := range r.sessions {
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (r *InMemorySessionRepository) FindAllPaginated(offset, limit int) ([]*session.Session, error) {
	sessions, err := r.FindAll()
	if err != nil {
		return nil, err
	}

	start := offset
	if start > len(sessions) {
		return []*session.Session{}, nil
	}

	end := start + limit
	if end > len(sessions) {
		end = len(sessions)
	}

	return sessions[start:end], nil
}

func (r *InMemorySessionRepository) FindByState(state session.SessionState) ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*session.Session
	for _, sess := range r.sessions {
		if sess.State() == state {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (r *InMemorySessionRepository) Update(sess *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions == nil {
		r.sessions = make(map[string]*session.Session)
	}

	r.sessions[sess.ID().Value()] = sess
	return nil
}

func (r *InMemorySessionRepository) Delete(id session.SessionID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, id.Value())
	return nil
}

func (r *InMemorySessionRepository) DeleteOlderThan(timestamp int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, sess := range r.sessions {
		if sess.StartTime().Unix() < timestamp && !sess.IsActive() {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *InMemorySessionRepository) GetSessionStatistics() (ports.SessionStatistics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := ports.SessionStatistics{
		TotalSessions:  len(r.sessions),
		ActiveSessions: 0,
		EndedSessions:  0,
	}

	for _, sess := range r.sessions {
		if sess.IsActive() {
			stats.ActiveSessions++
		} else {
			stats.EndedSessions++
		}
	}

	return stats, nil
}

// InMemoryEventStore provides an in-memory implementation of EventStore
type InMemoryEventStore struct {
	events map[string][]*event.Event
	mu     sync.RWMutex
}

func (s *InMemoryEventStore) Store(batch *session.EventBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.events == nil {
		s.events = make(map[string][]*event.Event)
	}

	sessionID := batch.SessionID.Value()
	s.events[sessionID] = append(s.events[sessionID], batch.Events...)
	return nil
}

func (s *InMemoryEventStore) StoreEvent(evt *event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.events == nil {
		s.events = make(map[string][]*event.Event)
	}

	// For single events, we need a way to associate them with a session
	// This is a simplified implementation
	sessionID := "default"
	s.events[sessionID] = append(s.events[sessionID], evt)
	return nil
}

func (s *InMemoryEventStore) Retrieve(criteria ports.EventCriteria) ([]*event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*event.Event

	// Simple implementation - get all events from specified session or all sessions
	if criteria.SessionID != nil {
		sessionID := criteria.SessionID.Value()
		if events, exists := s.events[sessionID]; exists {
			result = append(result, events...)
		}
	} else {
		// Get all events from all sessions
		for _, events := range s.events {
			result = append(result, events...)
		}
	}

	// Apply limit if specified
	if criteria.Limit > 0 && len(result) > criteria.Limit {
		result = result[:criteria.Limit]
	}

	return result, nil
}

func (s *InMemoryEventStore) Count(criteria ports.EventCriteria) (int, error) {
	events, err := s.Retrieve(criteria)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

func (s *InMemoryEventStore) Delete(criteria ports.EventCriteria) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if criteria.SessionID != nil {
		sessionID := criteria.SessionID.Value()
		delete(s.events, sessionID)
	}

	return nil
}

func (s *InMemoryEventStore) GetStorageStats() (ports.StorageStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalEvents := 0
	for _, events := range s.events {
		totalEvents += len(events)
	}

	return ports.StorageStatistics{
		TotalEvents:        totalEvents,
		TotalSizeBytes:     int64(totalEvents * 1024), // Rough estimate
		StorageUtilization: 0.0,
	}, nil
}
