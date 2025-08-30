package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
)

// MockAPIClient wraps HTTP calls to the mock-api server
type MockAPIClient struct {
	t      *testing.T
	client *http.Client
	url    string
	// Server management
	serverCmd    *exec.Cmd
	serverCancel context.CancelFunc
	port         string
	// Public fields for backward compatibility
	ApiKeyValid       bool
	SubscriptionTier  string
	CustomerName      string
	CustomerID        string
	AvailablePlugins  []plugindomain.Plugin
	DownloadResponses map[string][]byte
	AuthResponses     map[string]*auth.PluginAuthResponse
}

// MockServerConfig represents the server configuration
type MockServerConfig struct {
	SubscriptionTier  string                              `json:"subscription_tier"`
	CustomerName      string                              `json:"customer_name"`
	CustomerID        string                              `json:"customer_id"`
	APIKeyValid       bool                                `json:"api_key_valid"`
	APIKeys           map[string]bool                     `json:"api_keys"`
	AvailablePlugins  []MockPlugin                        `json:"available_plugins"`
	DownloadResponses map[string][]byte                   `json:"download_responses"`
	AuthResponses     map[string]*auth.PluginAuthResponse `json:"auth_responses"`
}

// MockPlugin represents a plugin in the mock API
type MockPlugin struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description,omitempty"`
	RequiredTier string `json:"required_tier"`
}

// NewMockAPIClient creates a new mock API client with auto-managed server
func NewMockAPIClient(t *testing.T) *MockAPIClient {
	// Find an available port
	port := findAvailablePort(t)
	url := fmt.Sprintf("http://localhost:%s", port)

	// Create the client
	client := &MockAPIClient{
		t:                 t,
		client:            &http.Client{Timeout: 10 * time.Second},
		url:               url,
		port:              port,
		ApiKeyValid:       true,
		SubscriptionTier:  "pro",
		CustomerName:      "Test User",
		CustomerID:        "test-customer-123",
		DownloadResponses: make(map[string][]byte),
		AuthResponses:     make(map[string]*auth.PluginAuthResponse),
	}

	// Start the server
	client.startServer(t)

	// Wait for server to be ready
	client.waitForServer(t)

	return client
}

// findAvailablePort finds an available port for the mock server
func findAvailablePort(t *testing.T) string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}

// startServer starts the mock-api server
func (m *MockAPIClient) startServer(t *testing.T) {
	// Create context for server management
	ctx, cancel := context.WithCancel(context.Background())
	m.serverCancel = cancel

	// Get the project root directory (go up from test/integration)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Start the mock-api server
	mockAPIPath := filepath.Join(projectRoot, "test", "mock-api", "main.go")
	cmd := exec.CommandContext(ctx, "go", "run", mockAPIPath, "-port", m.port)
	cmd.Dir = projectRoot

	// Set up server output for debugging (optional - can be commented out)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	m.serverCmd = cmd

	err = cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start mock-api server: %v", err)
	}

	// Register cleanup to stop server when test finishes
	t.Cleanup(func() {
		m.stopServer()
	})
}

// waitForServer waits for the server to be ready to accept connections
func (m *MockAPIClient) waitForServer(t *testing.T) {
	// Try to connect to the server with retries
	maxRetries := 50
	retryDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		resp, err := m.client.Get(fmt.Sprintf("%s/health", m.url))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return // Server is ready
		}
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(retryDelay)
	}

	t.Fatalf("Mock-api server failed to start within timeout on port %s", m.port)
}

// stopServer stops the mock-api server
func (m *MockAPIClient) stopServer() {
	if m.serverCancel != nil {
		m.serverCancel()
	}
	if m.serverCmd != nil && m.serverCmd.Process != nil {
		m.serverCmd.Process.Kill()
		m.serverCmd.Wait()
	}
}

// URL returns the server URL
func (m *MockAPIClient) URL() string {
	return m.url
}

// Close resets and stops the mock server
func (m *MockAPIClient) Close() {
	// First try to reset the server state
	resp, err := m.client.Post(fmt.Sprintf("%s/_control/reset", m.url), "application/json", nil)
	if err == nil && resp != nil {
		resp.Body.Close()
	}

	// Stop the server
	m.stopServer()
}

// Build configures the server and returns the client
func (m *MockAPIClient) Build() *MockAPIClient {
	// Convert domain plugins to mock plugins
	var mockPlugins []MockPlugin
	for _, p := range m.AvailablePlugins {
		mockPlugins = append(mockPlugins, MockPlugin{
			Name:         p.Name,
			Version:      p.Version,
			Description:  p.Description,
			RequiredTier: string(p.RequiredTier),
		})
	}

	// Configure server with initial values
	config := MockServerConfig{
		SubscriptionTier:  m.SubscriptionTier,
		CustomerName:      m.CustomerName,
		CustomerID:        m.CustomerID,
		APIKeyValid:       m.ApiKeyValid,
		APIKeys:           map[string]bool{"test-key": m.ApiKeyValid, "test-key-123": m.ApiKeyValid, "env-key-123": m.ApiKeyValid, "interactive-key-123": m.ApiKeyValid, "new-key": m.ApiKeyValid, "test-api-key": m.ApiKeyValid, "test-api-key-123": m.ApiKeyValid},
		AvailablePlugins:  mockPlugins,
		DownloadResponses: m.DownloadResponses,
		AuthResponses:     m.AuthResponses,
	}

	m.ConfigureServer(config)

	if len(mockPlugins) > 0 {
		m.SetPlugins(mockPlugins)
	}

	return m
}

// ConfigureServer updates the mock server configuration
func (m *MockAPIClient) ConfigureServer(config MockServerConfig) {
	data, err := json.Marshal(config)
	if err != nil {
		m.t.Fatalf("Failed to marshal config: %v", err)
	}

	resp, err := m.client.Post(fmt.Sprintf("%s/_control/config", m.url), "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.t.Logf("Warning: Could not configure server: %v", err)
		return
	}
	defer resp.Body.Close()
}

// SetPlugins sets available plugins on the mock server
func (m *MockAPIClient) SetPlugins(plugins []MockPlugin) {
	data, err := json.Marshal(plugins)
	if err != nil {
		m.t.Fatalf("Failed to marshal plugins: %v", err)
	}

	resp, err := m.client.Post(fmt.Sprintf("%s/_control/plugins", m.url), "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.t.Logf("Warning: Could not set plugins: %v", err)
		return
	}
	defer resp.Body.Close()
}

// SetDownloadResponses sets download responses on the mock server
func (m *MockAPIClient) SetDownloadResponses(downloads map[string][]byte) {
	// Convert bytes to string for JSON transmission
	stringDownloads := make(map[string]string)
	for name, data := range downloads {
		stringDownloads[name] = string(data)
	}

	data, err := json.Marshal(stringDownloads)
	if err != nil {
		m.t.Fatalf("Failed to marshal downloads: %v", err)
	}

	resp, err := m.client.Post(fmt.Sprintf("%s/_control/downloads", m.url), "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.t.Logf("Warning: Could not set downloads: %v", err)
		return
	}
	defer resp.Body.Close()
}

// GetRequestCount returns the number of requests made to a specific path
func (m *MockAPIClient) GetRequestCount(path string) int {
	resp, err := m.client.Get(fmt.Sprintf("%s/_control/requests?path=%s", m.url, path))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}

	return result.Count
}

// WithAPIKey sets API key validity (builder pattern for compatibility)
func (m *MockAPIClient) WithAPIKey(key string, valid bool) *MockAPIClient {
	m.ApiKeyValid = valid
	return m
}

// WithPlugins sets available plugins (builder pattern for compatibility)
func (m *MockAPIClient) WithPlugins(plugins []plugindomain.Plugin) *MockAPIClient {
	m.AvailablePlugins = plugins
	return m
}

// WithPluginDownload adds a download response (builder pattern for compatibility)
func (m *MockAPIClient) WithPluginDownload(pluginName string, data []byte) *MockAPIClient {
	if m.DownloadResponses == nil {
		m.DownloadResponses = make(map[string][]byte)
	}
	m.DownloadResponses[pluginName] = data
	return m
}

// WithAuthResponse sets auth response for a plugin (builder pattern for compatibility)
func (m *MockAPIClient) WithAuthResponse(pluginName string, response *auth.PluginAuthResponse) *MockAPIClient {
	authResponses := map[string]*auth.PluginAuthResponse{pluginName: response}
	data, err := json.Marshal(authResponses)
	if err != nil {
		m.t.Fatalf("Failed to marshal auth responses: %v", err)
	}

	resp, err := m.client.Post(fmt.Sprintf("%s/_control/auth", m.url), "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.t.Logf("Warning: Could not set auth responses: %v", err)
		return m
	}
	defer resp.Body.Close()

	return m
}

// WithTier sets subscription tier (builder pattern for compatibility)
func (m *MockAPIClient) WithTier(tier string) *MockAPIClient {
	m.SubscriptionTier = tier
	return m
}

// WithCustomer sets customer info (builder pattern for compatibility)
func (m *MockAPIClient) WithCustomer(name, id string) *MockAPIClient {
	m.CustomerName = name
	// CustomerID is set in Build() method
	return m
}
