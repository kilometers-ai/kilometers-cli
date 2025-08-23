package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestPluginManifestContract(t *testing.T) {
	// Mock server that handles POST requests (backward compatibility test)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GetPluginManifest now calls PostPluginManifest internally, so expect POST
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path != "/api/plugins/manifest" {
			t.Errorf("Expected path /api/plugins/manifest, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Verify POST request body
		var request PluginManifestRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify request structure
		if request.CLIVersion != "1.0.0" {
			t.Errorf("Expected CLI version '1.0.0', got '%s'", request.CLIVersion)
		}

		// Return mock response that matches API contract
		response := PluginManifestResponse{
			Plugins: []PluginManifestEntry{
				{
					Name:      "test-plugin",
					Version:   "1.0.0",
					Tier:      "free",
					URL:       "http://example.com/api/plugins/download/1",
					Hash:      "test-hash",
					Signature: "test-signature",
					Size:      1024,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test config
	client := &PluginApiClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		client:     &http.Client{},
		debug:      true,
		cliVersion: "1.0.0",
	}

	// Test the endpoint (now uses POST internally)
	ctx := context.Background()
	manifest, err := client.GetPluginManifest(ctx)
	if err != nil {
		t.Fatalf("Failed to get plugin manifest: %v", err)
	}

	// Verify response structure
	if len(manifest.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(manifest.Plugins))
	}

	plugin := manifest.Plugins[0]
	if plugin.Name != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got '%s'", plugin.Name)
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", plugin.Version)
	}
	if plugin.Tier != "free" {
		t.Errorf("Expected tier 'free', got '%s'", plugin.Tier)
	}
}

func TestPostPluginManifest(t *testing.T) {
	// Mock server that handles POST requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the endpoint and method
		if r.URL.Path != "/api/plugins/manifest" {
			t.Errorf("Expected path /api/plugins/manifest, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Expected Authorization 'Bearer test-key', got '%s'", auth)
		}

		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		var request PluginManifestRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify request structure
		if request.CLIVersion != "1.0.0" {
			t.Errorf("Expected CLI version '1.0.0', got '%s'", request.CLIVersion)
		}

		if request.Platform.OS != runtime.GOOS {
			t.Errorf("Expected OS '%s', got '%s'", runtime.GOOS, request.Platform.OS)
		}

		if request.Platform.Arch != runtime.GOARCH {
			t.Errorf("Expected Arch '%s', got '%s'", runtime.GOARCH, request.Platform.Arch)
		}

		if len(request.Plugins) != 2 {
			t.Errorf("Expected 2 plugins in request, got %d", len(request.Plugins))
		}

		// Return mock response
		response := PluginManifestResponse{
			Plugins: []PluginManifestEntry{
				{
					Name:      "api-logger",
					Version:   "1.2.4",
					Tier:      "pro",
					URL:       "http://example.com/api/plugins/download/api-logger",
					Hash:      "updated-hash",
					Signature: "updated-signature",
					Size:      2048,
				},
				{
					Name:      "console-logger",
					Version:   "1.0.1",
					Tier:      "free",
					URL:       "http://example.com/api/plugins/download/console-logger",
					Hash:      "console-hash",
					Signature: "console-signature",
					Size:      1024,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test config
	client := &PluginApiClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		client:     &http.Client{},
		debug:      true,
		cliVersion: "1.0.0",
	}

	// Test data - plugins currently installed
	installedPlugins := []InstalledPluginInfo{
		{
			Name:             "api-logger",
			InstalledVersion: stringPtr("1.2.3"),
		},
		{
			Name:             "console-logger",
			InstalledVersion: nil, // Not installed
		},
	}

	// Test the POST endpoint
	ctx := context.Background()
	manifest, err := client.PostPluginManifest(ctx, installedPlugins)
	if err != nil {
		t.Fatalf("Failed to post plugin manifest: %v", err)
	}

	// Verify response structure
	if len(manifest.Plugins) != 2 {
		t.Fatalf("Expected 2 plugins in response, got %d", len(manifest.Plugins))
	}

	// Verify api-logger plugin (should show available update)
	apiLogger := manifest.Plugins[0]
	if apiLogger.Name != "api-logger" {
		t.Errorf("Expected first plugin name 'api-logger', got '%s'", apiLogger.Name)
	}
	if apiLogger.Version != "1.2.4" {
		t.Errorf("Expected api-logger version '1.2.4', got '%s'", apiLogger.Version)
	}

	// Verify console-logger plugin (should show available version)
	consoleLogger := manifest.Plugins[1]
	if consoleLogger.Name != "console-logger" {
		t.Errorf("Expected second plugin name 'console-logger', got '%s'", consoleLogger.Name)
	}
	if consoleLogger.Version != "1.0.1" {
		t.Errorf("Expected console-logger version '1.0.1', got '%s'", consoleLogger.Version)
	}
}

func stringPtr(s string) *string {
	return &s
}