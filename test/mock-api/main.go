package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Mock API server for testing Kilometers CLI plugin architecture
// This simulates the Kilometers API endpoints for both manual testing and integration tests
// It supports runtime configuration via control API endpoints

type UserFeaturesResponse struct {
	Tier      string   `json:"tier"`
	Features  []string `json:"features"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

// Plugin domain types
type Plugin struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	RequiredTier string `json:"required_tier"`
}

type PluginManifestResponse struct {
	Plugins []PluginManifestEntry `json:"plugins"`
}

type PluginManifestEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Tier    string `json:"tier"`
	URL     string `json:"url"`
}

type PluginAuthResponse struct {
	Authorized bool     `json:"authorized"`
	UserTier   string   `json:"user_tier"`
	Features   []string `json:"features"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
}

// Request logging
type RequestInfo struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"body"`
	Timestamp   time.Time           `json:"timestamp"`
	QueryParams map[string][]string `json:"query_params"`
}

// Server configuration
type MockServerConfig struct {
	// Subscription settings
	SubscriptionTier string          `json:"subscription_tier"`
	CustomerName     string          `json:"customer_name"`
	CustomerID       string          `json:"customer_id"`
	APIKeyValid      bool            `json:"api_key_valid"`
	APIKeys          map[string]bool `json:"api_keys"`

	// Plugin settings
	AvailablePlugins  []Plugin                       `json:"available_plugins"`
	PluginManifest    *PluginManifestResponse        `json:"plugin_manifest"`
	DownloadResponses map[string][]byte              `json:"download_responses"`
	AuthResponses     map[string]*PluginAuthResponse `json:"auth_responses"`

	// Behavior settings
	SimulateErrors bool          `json:"simulate_errors"`
	ResponseDelay  time.Duration `json:"response_delay"`
	ErrorRate      float64       `json:"error_rate"`
}

// Global server state
type MockServer struct {
	config     MockServerConfig
	requestLog []RequestInfo
	mu         sync.RWMutex
}

var server *MockServer

func init() {
	server = &MockServer{
		config: MockServerConfig{
			SubscriptionTier: "pro",
			CustomerName:     "Test User",
			CustomerID:       "test-customer-123",
			APIKeyValid:      true,
			APIKeys: map[string]bool{
				"km_free_123456":    true,
				"km_pro_789012":     true,
				"km_ent_345678":     true,
				"km_downgrade_test": true,
			},
			AvailablePlugins:  []Plugin{},
			DownloadResponses: make(map[string][]byte),
			AuthResponses:     make(map[string]*PluginAuthResponse),
		},
		requestLog: []RequestInfo{},
	}
}

var mockSubscriptions = map[string]UserFeaturesResponse{
	"km_free_123456": {
		Tier:     "free",
		Features: []string{"basic_monitoring", "console_logging"},
	},
	"km_pro_789012": {
		Tier: "pro",
		Features: []string{
			"basic_monitoring",
			"console_logging",
			"api_logging",
			"advanced_filters",
			"ml_analytics",
		},
	},
	"km_ent_345678": {
		Tier: "enterprise",
		Features: []string{
			"basic_monitoring",
			"console_logging",
			"api_logging",
			"advanced_filters",
			"ml_analytics",
			"compliance_reporting",
			"team_collaboration",
		},
	},
	// Special key for testing downgrades
	"km_downgrade_test": {
		Tier:     "pro", // Will change to free after first request
		Features: []string{"basic_monitoring", "console_logging", "api_logging"},
	},
}

var requestCount = make(map[string]int)

func main() {
	// Parse command line flags
	port := flag.String("port", "5194", "Port to run the mock API server on")
	flag.Parse()

	fmt.Println("🚀 Mock Kilometers API Server")
	fmt.Println("=============================")
	fmt.Printf("Listening on http://localhost:%s\n", *port)
	fmt.Println()
	fmt.Println("Test API Keys:")
	fmt.Println("  - km_free_123456     (Free tier)")
	fmt.Println("  - km_pro_789012      (Pro tier)")
	fmt.Println("  - km_ent_345678      (Enterprise tier)")
	fmt.Println("  - km_downgrade_test  (Simulates downgrade)")
	fmt.Println()

	// Original endpoints
	http.HandleFunc("/api/user/features", handleUserFeatures)
	http.HandleFunc("/api/events/batch", handleEventsBatch)
	http.HandleFunc("/health", handleHealth)

	// Plugin management endpoints (from testutil)
	http.HandleFunc("/api/subscription/status", handleSubscriptionStatus)
	http.HandleFunc("/api/plugins/authenticate", handlePluginAuthenticate)
	http.HandleFunc("/api/plugins/available", handlePluginsAvailable)
	http.HandleFunc("/api/plugins/manifest", handlePluginManifest)
	http.HandleFunc("/v1/plugins/manifest", handlePluginManifest)
	http.HandleFunc("/api/plugins/download", handlePluginDownload)

	// Control API endpoints for test configuration
	http.HandleFunc("/_control/config", handleControlConfig)
	http.HandleFunc("/_control/reset", handleControlReset)
	http.HandleFunc("/_control/requests", handleControlRequests)
	http.HandleFunc("/_control/plugins", handleControlPlugins)
	http.HandleFunc("/_control/auth", handleControlAuth)
	http.HandleFunc("/_control/downloads", handleControlDownloads)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

// Request logging middleware
func logRequest(r *http.Request) {
	server.mu.Lock()
	defer server.mu.Unlock()

	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	server.requestLog = append(server.requestLog, RequestInfo{
		Method:      r.Method,
		Path:        r.URL.Path,
		Headers:     headers,
		Body:        body,
		Timestamp:   time.Now(),
		QueryParams: r.URL.Query(),
	})
}

func handleUserFeatures(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	log.Printf("[API] GET /api/user/features - API Key: %s", maskKey(apiKey))

	// Handle downgrade simulation
	if apiKey == "km_downgrade_test" {
		requestCount[apiKey]++
		if requestCount[apiKey] > 1 {
			// After first request, downgrade to free
			response := UserFeaturesResponse{
				Tier:     "free",
				Features: []string{"basic_monitoring", "console_logging"},
			}
			log.Printf("[API] Simulating downgrade for key: %s", maskKey(apiKey))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Look up subscription
	subscription, exists := mockSubscriptions[apiKey]
	if !exists {
		// Unknown key - return free tier
		subscription = UserFeaturesResponse{
			Tier:     "free",
			Features: []string{"basic_monitoring"},
		}
		log.Printf("[API] Unknown key, returning free tier: %s", maskKey(apiKey))
	}

	// Add expiration for non-free tiers
	if subscription.Tier != "free" {
		expires := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
		subscription.ExpiresAt = &expires
	}

	log.Printf("[API] Returning tier: %s with %d features", subscription.Tier, len(subscription.Features))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

func handleEventsBatch(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Decode the batch request
	var batch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	events, ok := batch["events"].([]interface{})
	if !ok {
		events = []interface{}{}
	}

	log.Printf("[API] POST /api/events/batch - API Key: %s, Events: %d", maskKey(apiKey), len(events))

	// Check if the user has API logging feature
	subscription, exists := mockSubscriptions[apiKey]
	if !exists || !contains(subscription.Features, "api_logging") {
		log.Printf("[API] Rejecting batch - no api_logging feature for key: %s", maskKey(apiKey))
		http.Error(w, "API logging not available in your subscription", http.StatusForbidden)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Received %d events", len(events)),
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func maskKey(key string) string {
	if len(key) > 10 {
		return key[:6] + "..." + key[len(key)-4:]
	}
	return key
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Plugin management handlers (from testutil)

func handleSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	server.mu.RLock()
	config := server.config
	server.mu.RUnlock()

	authHeader := r.Header.Get("Authorization")
	apiKeyHeader := r.Header.Get("X-API-Key")
	hasAuth := authHeader != "" || apiKeyHeader != ""

	if !hasAuth || !config.APIKeyValid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid API key",
		})
		return
	}

	features := getFeaturesForTier(config.SubscriptionTier)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"customer_id":   config.CustomerID,
		"customer_name": config.CustomerName,
		"tier":          config.SubscriptionTier,
		"features":      features,
	})
}

func handlePluginAuthenticate(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	server.mu.RLock()
	config := server.config
	server.mu.RUnlock()

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
	if response, exists := config.AuthResponses[pluginName]; exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Default response based on tier
	features := getFeaturesForTier(config.SubscriptionTier)
	response := PluginAuthResponse{
		Authorized: true,
		UserTier:   config.SubscriptionTier,
		Features:   features,
		ExpiresAt:  stringPtr(time.Now().Add(5 * time.Minute).Format(time.RFC3339)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePluginsAvailable(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	authHeader := r.Header.Get("Authorization")
	apiKeyHeader := r.Header.Get("X-API-Key")
	hasAuth := authHeader != "" || apiKeyHeader != ""

	if !hasAuth {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	server.mu.RLock()
	plugins := server.config.AvailablePlugins
	server.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"plugins": plugins,
	})
}

func handlePluginManifest(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	server.mu.RLock()
	config := server.config
	server.mu.RUnlock()

	if config.PluginManifest != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.PluginManifest)
		return
	}

	// Default empty manifest
	manifest := PluginManifestResponse{
		Plugins: []PluginManifestEntry{},
	}

	// Convert available plugins to manifest entries
	for _, plugin := range config.AvailablePlugins {
		// Get the host from the request to generate the correct download URL
		host := r.Host
		if host == "" {
			host = "localhost:5194" // fallback for old behavior
		}

		entry := PluginManifestEntry{
			Name:    plugin.Name,
			Version: plugin.Version,
			Tier:    plugin.RequiredTier,
			URL:     fmt.Sprintf("http://%s/api/plugins/download/%s", host, plugin.Name),
		}
		manifest.Plugins = append(manifest.Plugins, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func handlePluginDownload(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	authHeader := r.Header.Get("Authorization")
	apiKeyHeader := r.Header.Get("X-API-Key")
	hasAuth := authHeader != "" || apiKeyHeader != ""

	if !hasAuth {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Handle both POST with JSON body and GET with path parameter
	var pluginName string

	if r.Method == "POST" {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pluginName, _ = req["plugin_name"].(string)
	} else if r.Method == "GET" {
		// Extract plugin name from path like /api/plugins/download/plugin-name
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 5 {
			pluginName = parts[4]
		}
	}

	server.mu.RLock()
	data, exists := server.config.DownloadResponses[pluginName]
	server.mu.RUnlock()

	if exists {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pluginName))
		w.Write(data)
	} else {
		// Default mock plugin data if not found
		mockData := []byte("#!/bin/bash\necho 'Mock plugin: " + pluginName + "'")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(mockData)
	}
}

// Control API handlers for test configuration

func handleControlConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		server.mu.RLock()
		config := server.config
		server.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	case "POST":
		var newConfig MockServerConfig
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid config", http.StatusBadRequest)
			return
		}

		server.mu.Lock()
		server.config = newConfig
		server.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleControlReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	server.mu.Lock()
	server.requestLog = []RequestInfo{}
	server.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

func handleControlRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	server.mu.RLock()
	requests := server.requestLog
	server.mu.RUnlock()

	// Support filtering by path
	path := r.URL.Query().Get("path")
	if path != "" {
		filtered := []RequestInfo{}
		for _, req := range requests {
			if req.Path == path {
				filtered = append(filtered, req)
			}
		}
		requests = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	})
}

func handleControlPlugins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var plugins []Plugin
		if err := json.NewDecoder(r.Body).Decode(&plugins); err != nil {
			http.Error(w, "Invalid plugins data", http.StatusBadRequest)
			return
		}

		server.mu.Lock()
		server.config.AvailablePlugins = plugins
		server.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleControlAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var authResponses map[string]*PluginAuthResponse
		if err := json.NewDecoder(r.Body).Decode(&authResponses); err != nil {
			http.Error(w, "Invalid auth responses", http.StatusBadRequest)
			return
		}

		server.mu.Lock()
		server.config.AuthResponses = authResponses
		server.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleControlDownloads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var downloads map[string]string // plugin name -> base64 encoded data
		if err := json.NewDecoder(r.Body).Decode(&downloads); err != nil {
			http.Error(w, "Invalid download data", http.StatusBadRequest)
			return
		}

		server.mu.Lock()
		for name, data := range downloads {
			server.config.DownloadResponses[name] = []byte(data)
		}
		server.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Utility functions

func getFeaturesForTier(tier string) []string {
	switch strings.ToLower(tier) {
	case "free":
		return []string{"monitoring", "console_logging"}
	case "pro":
		return []string{"monitoring", "console_logging", "api_logging"}
	case "enterprise":
		return []string{"monitoring", "console_logging", "api_logging", "advanced_analytics", "custom_plugins"}
	default:
		return []string{"monitoring"}
	}
}

func stringPtr(s string) *string {
	return &s
}
