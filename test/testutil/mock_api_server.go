package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
	httpinternal "github.com/kilometers-ai/kilometers-cli/internal/http"
)

// MockAPIServer provides a comprehensive mock HTTP server for testing
type MockAPIServer struct {
	*httptest.Server
	Config     MockAPIConfig
	RequestLog []RequestInfo
	mu         sync.Mutex
}

// MockAPIConfig contains all configuration options for the mock server
type MockAPIConfig struct {
	// Subscription settings
	SubscriptionTier string
	CustomerName     string
	CustomerID       string
	APIKeyValid      bool
	APIKeys          map[string]bool // Map of valid API keys

	// Plugin settings
	AvailablePlugins  []plugindomain.Plugin
	PluginManifest    *httpinternal.PluginManifestResponse
	DownloadResponses map[string][]byte // Map of plugin name to binary data

	// Authentication responses per plugin
	AuthResponses map[string]*auth.PluginAuthResponse

	// Behavior settings
	SimulateErrors bool
	ResponseDelay  time.Duration
	ErrorRate      float64 // Percentage of requests to fail (0.0-1.0)

	// Custom handlers for specific endpoints
	CustomHandlers map[string]http.HandlerFunc
}

// RequestInfo captures information about each request for test assertions
type RequestInfo struct {
	Method      string
	Path        string
	Headers     http.Header
	Body        []byte
	Timestamp   time.Time
	QueryParams map[string][]string
}

// MockAPIServerBuilder provides a fluent interface for configuring the mock server
type MockAPIServerBuilder struct {
	t      *testing.T
	config MockAPIConfig
}

// NewMockAPIServer creates a new mock API server builder
func NewMockAPIServer(t *testing.T) *MockAPIServerBuilder {
	return &MockAPIServerBuilder{
		t: t,
		config: MockAPIConfig{
			SubscriptionTier:  "pro",
			CustomerName:      "Test User",
			CustomerID:        "test-customer-123",
			APIKeyValid:       true,
			APIKeys:           make(map[string]bool),
			AvailablePlugins:  []plugindomain.Plugin{},
			DownloadResponses: make(map[string][]byte),
			AuthResponses:     make(map[string]*auth.PluginAuthResponse),
			CustomHandlers:    make(map[string]http.HandlerFunc),
		},
	}
}

// WithTier sets the subscription tier
func (b *MockAPIServerBuilder) WithTier(tier string) *MockAPIServerBuilder {
	b.config.SubscriptionTier = tier
	return b
}

// WithCustomer sets customer information
func (b *MockAPIServerBuilder) WithCustomer(name, id string) *MockAPIServerBuilder {
	b.config.CustomerName = name
	b.config.CustomerID = id
	return b
}

// WithAPIKey configures API key validation
func (b *MockAPIServerBuilder) WithAPIKey(apiKey string, valid bool) *MockAPIServerBuilder {
	b.config.APIKeyValid = valid
	if valid {
		b.config.APIKeys[apiKey] = true
	}
	return b
}

// WithPlugins sets available plugins
func (b *MockAPIServerBuilder) WithPlugins(plugins []plugindomain.Plugin) *MockAPIServerBuilder {
	b.config.AvailablePlugins = plugins
	return b
}

// WithPluginManifest sets the plugin manifest response
func (b *MockAPIServerBuilder) WithPluginManifest(manifest *httpinternal.PluginManifestResponse) *MockAPIServerBuilder {
	b.config.PluginManifest = manifest
	return b
}

// WithPluginDownload configures plugin download responses
func (b *MockAPIServerBuilder) WithPluginDownload(pluginName string, data []byte) *MockAPIServerBuilder {
	b.config.DownloadResponses[pluginName] = data
	return b
}

// WithAuthResponse configures authentication response for a specific plugin
func (b *MockAPIServerBuilder) WithAuthResponse(pluginName string, response *auth.PluginAuthResponse) *MockAPIServerBuilder {
	b.config.AuthResponses[pluginName] = response
	return b
}

// WithResponseDelay adds artificial delay to responses
func (b *MockAPIServerBuilder) WithResponseDelay(delay time.Duration) *MockAPIServerBuilder {
	b.config.ResponseDelay = delay
	return b
}

// WithErrorSimulation enables error simulation
func (b *MockAPIServerBuilder) WithErrorSimulation(errorRate float64) *MockAPIServerBuilder {
	b.config.SimulateErrors = true
	b.config.ErrorRate = errorRate
	return b
}

// WithCustomHandler adds a custom handler for a specific path
func (b *MockAPIServerBuilder) WithCustomHandler(path string, handler http.HandlerFunc) *MockAPIServerBuilder {
	b.config.CustomHandlers[path] = handler
	return b
}

// Build creates the configured mock API server
func (b *MockAPIServerBuilder) Build() *MockAPIServer {
	mock := &MockAPIServer{
		Config:     b.config,
		RequestLog: []RequestInfo{},
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		mock.logRequest(r)

		// Apply response delay if configured
		if b.config.ResponseDelay > 0 {
			time.Sleep(b.config.ResponseDelay)
		}

		// Check for custom handlers first
		if handler, exists := b.config.CustomHandlers[r.URL.Path]; exists {
			handler(w, r)
			return
		}

		// Check authentication
		authHeader := r.Header.Get("Authorization")
		apiKeyHeader := r.Header.Get("X-API-Key")
		hasAuth := authHeader != "" || apiKeyHeader != ""

		// Route to appropriate handler
		switch r.URL.Path {
		case "/api/subscription/status":
			mock.handleSubscriptionStatus(w, r, hasAuth)

		case "/api/plugins/authenticate":
			mock.handlePluginAuthenticate(w, r)

		case "/api/plugins/available":
			mock.handlePluginsAvailable(w, r, hasAuth)

		case "/api/plugins/manifest", "/v1/plugins/manifest":
			mock.handlePluginManifest(w, r, hasAuth)

		case "/api/plugins/download":
			mock.handlePluginDownload(w, r, hasAuth)

		default:
			// Check if it's a download URL pattern
			if matched, pluginID := mock.matchDownloadPath(r.URL.Path); matched {
				mock.handleDirectDownload(w, r, pluginID)
			} else {
				http.NotFound(w, r)
			}
		}
	}))

	return mock
}

// Helper methods for request handling

func (m *MockAPIServer) logRequest(r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	m.RequestLog = append(m.RequestLog, RequestInfo{
		Method:      r.Method,
		Path:        r.URL.Path,
		Headers:     r.Header.Clone(),
		Body:        body,
		Timestamp:   time.Now(),
		QueryParams: r.URL.Query(),
	})
}

func (m *MockAPIServer) handleSubscriptionStatus(w http.ResponseWriter, r *http.Request, hasAuth bool) {
	if !hasAuth || !m.Config.APIKeyValid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid API key",
		})
		return
	}

	features := m.getFeaturesForTier(m.Config.SubscriptionTier)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"customer_id":   m.Config.CustomerID,
		"customer_name": m.Config.CustomerName,
		"tier":          m.Config.SubscriptionTier,
		"features":      features,
	})
}

func (m *MockAPIServer) handlePluginAuthenticate(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var authReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	pluginName, _ := authReq["plugin_name"].(string)

	// Check if we have a specific auth response configured
	if response, exists := m.Config.AuthResponses[pluginName]; exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Default response based on tier
	features := m.getFeaturesForTier(m.Config.SubscriptionTier)
	response := auth.PluginAuthResponse{
		Authorized: true,
		UserTier:   m.Config.SubscriptionTier,
		Features:   features,
		ExpiresAt:  stringPtr(time.Now().Add(5 * time.Minute).Format(time.RFC3339)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockAPIServer) handlePluginsAvailable(w http.ResponseWriter, r *http.Request, hasAuth bool) {
	if !hasAuth {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"plugins": m.Config.AvailablePlugins,
	})
}

func (m *MockAPIServer) handlePluginManifest(w http.ResponseWriter, r *http.Request, hasAuth bool) {
	if m.Config.PluginManifest != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.Config.PluginManifest)
		return
	}

	// Default empty manifest
	manifest := httpinternal.PluginManifestResponse{
		Plugins: []httpinternal.PluginManifestEntry{},
	}

	// Convert available plugins to manifest entries
	for _, plugin := range m.Config.AvailablePlugins {
		entry := httpinternal.PluginManifestEntry{
			Name:    plugin.Name,
			Version: plugin.Version,
			Tier:    string(plugin.RequiredTier),
			URL:     fmt.Sprintf("%s/api/plugins/download/%s", m.URL, plugin.Name),
		}
		manifest.Plugins = append(manifest.Plugins, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func (m *MockAPIServer) handlePluginDownload(w http.ResponseWriter, r *http.Request, hasAuth bool) {
	if !hasAuth {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pluginName, _ := req["plugin_name"].(string)

	if data, exists := m.Config.DownloadResponses[pluginName]; exists {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pluginName))
		w.Write(data)
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Plugin not found",
		})
	}
}

func (m *MockAPIServer) handleDirectDownload(w http.ResponseWriter, r *http.Request, pluginID string) {
	// Look for plugin data by ID or name
	for name, data := range m.Config.DownloadResponses {
		if name == pluginID {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(data)
			return
		}
	}

	// Default mock plugin data if not found
	mockData := []byte("#!/bin/bash\necho 'Mock plugin'")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(mockData)
}

func (m *MockAPIServer) matchDownloadPath(path string) (bool, string) {
	// Match patterns like /api/plugins/download/1 or /api/plugins/download/plugin-name
	var pluginID string
	if _, err := fmt.Sscanf(path, "/api/plugins/download/%s", &pluginID); err == nil {
		return true, pluginID
	}
	return false, ""
}

func (m *MockAPIServer) getFeaturesForTier(tier string) []string {
	switch tier {
	case "free", "Free":
		return []string{"monitoring", "console_logging"}
	case "pro", "Pro":
		return []string{"monitoring", "console_logging", "api_logging"}
	case "enterprise", "Enterprise":
		return []string{"monitoring", "console_logging", "api_logging", "advanced_analytics", "custom_plugins"}
	default:
		return []string{"monitoring"}
	}
}

// Utility methods for test assertions

// GetRequestCount returns the number of requests made to a specific path
func (m *MockAPIServer) GetRequestCount(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, req := range m.RequestLog {
		if req.Path == path {
			count++
		}
	}
	return count
}

// GetLastRequest returns the most recent request to a specific path
func (m *MockAPIServer) GetLastRequest(path string) *RequestInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := len(m.RequestLog) - 1; i >= 0; i-- {
		if m.RequestLog[i].Path == path {
			req := m.RequestLog[i]
			return &req
		}
	}
	return nil
}

// ClearRequestLog clears the request log
func (m *MockAPIServer) ClearRequestLog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RequestLog = []RequestInfo{}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
