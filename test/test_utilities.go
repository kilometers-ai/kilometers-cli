package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
	"kilometers.ai/cli/internal/infrastructure/api"
	"kilometers.ai/cli/internal/interfaces/di"
)

// TestEnvironment encapsulates a complete test environment
type TestEnvironment struct {
	MockMCPServer *MockMCPServer
	MockAPIServer *MockAPIServer
	TempDir       string
	ConfigFile    string
	Container     *di.Container
	ctx           context.Context
	cancel        context.CancelFunc
	mcpPort       int
	apiPort       int
}

// NewTestEnvironment creates a new isolated test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	ctx, cancel := context.WithCancel(context.Background())

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "km-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	env := &TestEnvironment{
		MockMCPServer: NewMockMCPServer(),
		MockAPIServer: NewMockAPIServer(),
		TempDir:       tempDir,
		ctx:           ctx,
		cancel:        cancel,
		mcpPort:       GetFreePort(),
		apiPort:       GetFreePort(),
	}

	// Setup cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

// Start starts all mock servers and initializes the test environment
func (env *TestEnvironment) Start(t *testing.T) error {
	// Start mock servers
	if err := env.MockMCPServer.Start(env.mcpPort); err != nil {
		return fmt.Errorf("failed to start mock MCP server: %w", err)
	}

	if err := env.MockAPIServer.Start(env.apiPort); err != nil {
		return fmt.Errorf("failed to start mock API server: %w", err)
	}

	// Setup authentication tokens for mock API server
	env.MockAPIServer.AddAuthToken("test_token_123") // Config file token
	env.MockAPIServer.AddAuthToken("test_key")       // Environment variable token

	// Wait for servers to be ready
	time.Sleep(100 * time.Millisecond)

	// Create test configuration
	if err := env.createTestConfig(); err != nil {
		return fmt.Errorf("failed to create test config: %w", err)
	}

	// Set environment variables for the test
	env.setTestEnvironmentVariables()

	return nil
}

// StartWithContainer starts the environment and creates a DI container
func (env *TestEnvironment) StartWithContainer(t *testing.T) error {
	if err := env.Start(t); err != nil {
		return err
	}

	// Debug: Print environment variables for troubleshooting
	t.Logf("Test environment variables set:")
	t.Logf("KM_API_URL: %s", os.Getenv("KM_API_URL"))
	t.Logf("KM_CONFIG_FILE: %s", os.Getenv("KM_CONFIG_FILE"))
	t.Logf("KM_API_KEY: %s", os.Getenv("KM_API_KEY"))
	t.Logf("Mock API Server Address: %s", env.GetAPIServerAddress())

	// Create DI container with test configuration
	container, err := di.NewContainer()
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Override API gateway with test-friendly retry policy for faster tests
	if container.APIGateway != nil {
		// Create a simple logging adapter for tests
		testLogger := &testLoggingAdapter{logger: log.New(os.Stderr, "[test] ", log.LstdFlags)}

		// Set shorter timeouts for testing
		container.APIGateway = api.NewTestAPIGateway(
			env.GetAPIServerAddress(),
			"test_key",
			testLogger,
		)
	}

	env.Container = container
	return nil
}

// Cleanup cleans up the test environment
func (env *TestEnvironment) Cleanup() {
	if env.cancel != nil {
		env.cancel()
	}

	if env.MockMCPServer != nil {
		env.MockMCPServer.Stop()
	}

	if env.MockAPIServer != nil {
		env.MockAPIServer.Stop()
	}

	if env.Container != nil {
		env.Container.Shutdown(env.ctx)
	}

	if env.TempDir != "" {
		os.RemoveAll(env.TempDir)
	}

	// Clean up environment variables
	env.cleanupEnvironmentVariables()
}

// createTestConfig creates a test configuration file
func (env *TestEnvironment) createTestConfig() error {
	env.ConfigFile = filepath.Join(env.TempDir, "config.json")

	config := fmt.Sprintf(`{
		"api_endpoint": "%s",
		"api_key": "test_token_123",
		"batch_size": 10,
		"batch_timeout": "5s",
		"high_risk_methods_only": false,
		"payload_size_limit": 1048576,
		"minimum_risk_level": "low",
		"exclude_ping_messages": true,
		"enable_risk_detection": true,
		"method_whitelist": [],
		"method_blacklist": [],
		"log_level": "debug",
		"session_timeout": "1h"
	}`, env.GetAPIServerAddress())

	err := os.WriteFile(env.ConfigFile, []byte(config), 0644)
	if err != nil {
		return err
	}

	// Debug: log what was written to the config file
	fmt.Printf("Test config file created at %s with API endpoint: %s\n", env.ConfigFile, env.GetAPIServerAddress())

	return nil
}

// setTestEnvironmentVariables sets environment variables for testing
func (env *TestEnvironment) setTestEnvironmentVariables() {
	os.Setenv("KM_CONFIG_FILE", env.ConfigFile)
	os.Setenv("KM_API_URL", fmt.Sprintf("http://localhost:%d", env.apiPort))
	os.Setenv("KM_API_KEY", "test_key")
	os.Setenv("KM_DEBUG", "true")
	os.Setenv("KM_TEST_MODE", "true")
}

// cleanupEnvironmentVariables cleans up test environment variables
func (env *TestEnvironment) cleanupEnvironmentVariables() {
	os.Unsetenv("KM_CONFIG_FILE")
	os.Unsetenv("KM_API_URL")
	os.Unsetenv("KM_API_KEY")
	os.Unsetenv("KM_DEBUG")
	os.Unsetenv("KM_TEST_MODE")
}

// GetMCPServerAddress returns the MCP server address
func (env *TestEnvironment) GetMCPServerAddress() string {
	return fmt.Sprintf("localhost:%d", env.mcpPort)
}

// GetAPIServerAddress returns the API server address
func (env *TestEnvironment) GetAPIServerAddress() string {
	return fmt.Sprintf("http://localhost:%d", env.apiPort)
}

// Helper functions for creating test data

// CreateTestEvent creates a test event with the given parameters
func CreateTestEvent(sessionID, method string, direction event.Direction) *event.Event {
	methodObj, _ := event.NewMethod(method)
	payload := fmt.Sprintf(`{"test": "data", "method": "%s", "session_id": "%s"}`, method, sessionID)
	riskScore, _ := event.NewRiskScore(10) // Low risk by default

	evt, err := event.CreateEvent(
		direction,
		methodObj,
		[]byte(payload),
		riskScore,
	)

	if err != nil {
		panic(fmt.Sprintf("Failed to create test event: %v", err))
	}

	return evt
}

// CreateTestSession creates a test session with events
func CreateTestSession(sessionID string, eventCount int) *session.Session {
	sessionIDObj, _ := session.NewSessionID(sessionID)
	config := session.DefaultSessionConfig()
	sess := session.NewSessionWithID(sessionIDObj, config)

	// Start the session
	sess.Start()

	// Add test events
	for i := 0; i < eventCount; i++ {
		event := CreateTestEvent(
			sessionID,
			fmt.Sprintf("test/method_%d", i),
			event.DirectionInbound,
		)
		sess.AddEvent(event)
	}

	return sess
}

// CreateTestEventBatch creates a batch of test events
func CreateTestEventBatch(sessionID string, count int) []map[string]interface{} {
	events := make([]map[string]interface{}, count)

	for i := 0; i < count; i++ {
		events[i] = map[string]interface{}{
			"id":         fmt.Sprintf("event_%d", i),
			"session_id": sessionID,
			"method":     fmt.Sprintf("test/method_%d", i),
			"direction":  "client_to_server",
			"timestamp":  time.Now().Unix(),
			"payload": map[string]interface{}{
				"test_data": fmt.Sprintf("data_%d", i),
			},
		}
	}

	return events
}

// Assertion helpers for integration tests

// AssertEventProcessed verifies that an event was processed correctly
func AssertEventProcessed(t *testing.T, evt *event.Event, expectedRiskLevel string) {
	t.Helper()

	if evt == nil {
		t.Fatal("Event is nil")
	}

	if evt.RiskScore().Level() != expectedRiskLevel {
		t.Errorf("Expected risk level %v, got %v", expectedRiskLevel, evt.RiskScore().Level())
	}

	if evt.Timestamp().IsZero() {
		t.Error("Event timestamp is zero")
	}

	if evt.ID().String() == "" {
		t.Error("Event ID is empty")
	}
}

// AssertSessionState verifies session state
func AssertSessionState(t *testing.T, sess *session.Session, expectedEventCount int, expectedState session.SessionState) {
	t.Helper()

	if sess == nil {
		t.Fatal("Session is nil")
	}

	if sess.TotalEvents() != expectedEventCount {
		t.Errorf("Expected %d events, got %d", expectedEventCount, sess.TotalEvents())
	}

	if sess.State() != expectedState {
		t.Errorf("Expected session state %v, got %v", expectedState, sess.State())
	}
}

// AssertAPIRequestMade verifies that an API request was made
func AssertAPIRequestMade(t *testing.T, env *TestEnvironment, method, path string) {
	t.Helper()

	requests := env.MockAPIServer.GetRequestLog()
	found := false

	for _, req := range requests {
		if req.Method == method && req.Path == path {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected API request %s %s not found in request log", method, path)
	}
}

// AssertMCPMessageSent verifies that an MCP message was sent
func AssertMCPMessageSent(t *testing.T, env *TestEnvironment, method string) {
	t.Helper()

	stats := env.MockMCPServer.GetStats()
	totalMessages, ok := stats["total_messages"].(int64)
	if !ok || totalMessages == 0 {
		t.Error("No MCP messages were sent")
		return
	}

	// Check connection messages
	connections, ok := stats["connections"].(map[string]interface{})
	if !ok {
		t.Error("No MCP connections found")
		return
	}

	found := false
	for connID := range connections {
		messages := env.MockMCPServer.GetConnectionMessages(connID)
		for _, msg := range messages {
			if msg.Method == method {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Errorf("Expected MCP message with method %s not found", method)
	}
}

// Performance testing helpers

// MeasureExecutionTime measures the execution time of a function
func MeasureExecutionTime(t *testing.T, name string, fn func()) time.Duration {
	t.Helper()

	start := time.Now()
	fn()
	duration := time.Since(start)

	t.Logf("Execution time for %s: %v", name, duration)
	return duration
}

// AssertExecutionTime verifies that execution time is within acceptable limits
func AssertExecutionTime(t *testing.T, duration time.Duration, maxDuration time.Duration, operation string) {
	t.Helper()

	if duration > maxDuration {
		t.Errorf("Operation %s took %v, expected <= %v", operation, duration, maxDuration)
	}
}

// Utility functions

// GetFreePort returns an available port for testing
func GetFreePort() int {
	// Actually find a free port by attempting to bind to it
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		// Fallback to time-based allocation if binding fails
		return 9000 + int(time.Now().UnixNano()%1000)
	}
	defer listener.Close()

	// Get the port that was allocated
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(condition func() bool, timeout time.Duration, checkInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}

	return false
}

// CreateTempFile creates a temporary file with content
func CreateTempFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}

	file.Close()
	return file.Name()
}

// MockProcess represents a mock MCP process for testing
type MockProcess struct {
	Command string
	Args    []string
	PID     int
	Running bool
}

// NewMockProcess creates a new mock process
func NewMockProcess(command string, args []string) *MockProcess {
	return &MockProcess{
		Command: command,
		Args:    args,
		PID:     os.Getpid() + int(time.Now().UnixNano()%1000), // Fake PID
		Running: false,
	}
}

// Start starts the mock process
func (p *MockProcess) Start() error {
	if p.Running {
		return fmt.Errorf("process already running")
	}
	p.Running = true
	return nil
}

// Stop stops the mock process
func (p *MockProcess) Stop() error {
	if !p.Running {
		return fmt.Errorf("process not running")
	}
	p.Running = false
	return nil
}

// IsRunning returns whether the process is running
func (p *MockProcess) IsRunning() bool {
	return p.Running
}

// GetPID returns the process ID
func (p *MockProcess) GetPID() int {
	return p.PID
}

// TestBatch represents a test batch for integration testing
type TestBatch struct {
	ID     string
	Size   int
	Events int
}

// testLoggingAdapter adapts the standard logger to the LoggingGateway interface for tests
type testLoggingAdapter struct {
	logger   *log.Logger
	logLevel ports.LogLevel
}

func (l *testLoggingAdapter) LogError(err error, message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("ERROR: %s: %v (fields: %v)", message, err, fields)
	} else {
		l.logger.Printf("ERROR: %s: %v", message, err)
	}
}

func (l *testLoggingAdapter) LogInfo(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("INFO: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("INFO: %s", message)
	}
}

func (l *testLoggingAdapter) LogDebug(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("DEBUG: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("DEBUG: %s", message)
	}
}

func (l *testLoggingAdapter) LogWarning(message string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Printf("WARNING: %s (fields: %v)", message, fields)
	} else {
		l.logger.Printf("WARNING: %s", message)
	}
}

func (l *testLoggingAdapter) LogSession(session *session.Session, message string) {
	l.logger.Printf("SESSION: %s (session: %v)", message, session)
}

func (l *testLoggingAdapter) LogEvent(event *event.Event, message string) {
	l.logger.Printf("EVENT: %s (event: %v)", message, event)
}

func (l *testLoggingAdapter) LogMetric(name string, value float64, labels map[string]string) {
	l.logger.Printf("METRIC: %s = %f (labels: %v)", name, value, labels)
}

func (l *testLoggingAdapter) Log(level ports.LogLevel, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	prefix := ""
	switch level {
	case ports.LogLevelError:
		prefix = "ERROR"
	case ports.LogLevelWarn:
		prefix = "WARNING"
	case ports.LogLevelInfo:
		prefix = "INFO"
	case ports.LogLevelDebug:
		prefix = "DEBUG"
	}

	if fields != nil {
		l.logger.Printf("%s: %s (fields: %v)", prefix, message, fields)
	} else {
		l.logger.Printf("%s: %s", prefix, message)
	}
}

func (l *testLoggingAdapter) SetLogLevel(level ports.LogLevel) {
	l.logLevel = level
}

func (l *testLoggingAdapter) GetLogLevel() ports.LogLevel {
	return l.logLevel
}

func (l *testLoggingAdapter) ConfigureLogging(config *ports.LoggingConfig) error {
	return nil // Simple implementation for tests
}

func (l *testLoggingAdapter) shouldLog(level ports.LogLevel) bool {
	return level <= l.logLevel
}
