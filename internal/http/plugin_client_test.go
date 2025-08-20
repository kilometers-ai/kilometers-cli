package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPluginManifestContract(t *testing.T) {
	// Mock server that responds with the expected API format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the endpoint is correct
		if r.URL.Path != "/api/plugins/manifest" {
			t.Errorf("Expected path /api/plugins/manifest, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
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
		baseURL: server.URL,
		apiKey:  "test-key",
		client:  &http.Client{},
		debug:   true,
	}

	// Test the endpoint
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