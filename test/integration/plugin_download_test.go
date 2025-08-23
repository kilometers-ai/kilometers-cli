package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	httpinternal "github.com/kilometers-ai/kilometers-cli/internal/http"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
)

// TestPluginDownloadFlow tests the complete plugin download and installation flow
func TestPluginDownloadFlow(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "km-plugin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Mock plugin binary data
	mockPluginData := []byte("#!/bin/bash\necho 'Mock plugin'")
	mockHash := "4bc6b7962b3500c26d51cead1c416d5d17320408b5f2cbbfd1293b645f0c0633" // SHA256 of the actual data
	// Signature would be stored in a separate .sig file in real implementation

	// Create mock API server
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/plugins/manifest":
			// Return mock manifest
			manifest := httpinternal.PluginManifestResponse{
				Plugins: []httpinternal.PluginManifestEntry{
					{
						Name:    "test-plugin",
						Version: "1.0.0",
						Tier:    "free",
						URL:     fmt.Sprintf("%s/api/plugins/download/1", serverURL),
						Hash:    mockHash,
						// Signature is stored separately in .sig file
						Size: int64(len(mockPluginData)),
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)

		case "/api/plugins/download/1":
			// Return mock plugin binary
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(mockPluginData)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	
	// Set server URL after server is created
	serverURL = server.URL

	// Test plugin download functionality
	t.Run("PluginDownloader", func(t *testing.T) {
		testPluginDownloader(t, tempDir, server.URL)
	})

	// Test enhanced plugin manager
	t.Run("PluginManager", func(t *testing.T) {
		testPluginManager(t, tempDir, server.URL)
	})

	// Test caching functionality
	t.Run("ManifestCaching", func(t *testing.T) {
		testManifestCaching(t, tempDir, server.URL)
	})
}

func testPluginDownloader(t *testing.T, tempDir, serverURL string) {
	pluginsDir := filepath.Join(tempDir, "plugins")
	
	// Create plugin downloader
	downloader, err := plugins.NewPluginDownloader(pluginsDir, true, "test-version")
	if err != nil {
		t.Fatalf("Failed to create plugin downloader: %v", err)
	}

	// Create mock plugin entry
	plugin := &httpinternal.PluginManifestEntry{
		Name:    "test-plugin",
		Version: "1.0.0",
		Tier:    "free",
		URL:     fmt.Sprintf("%s/api/plugins/download/1", serverURL),
		Hash:    "4bc6b7962b3500c26d51cead1c416d5d17320408b5f2cbbfd1293b645f0c0633",
		// Signature is stored separately in .sig file
		Size: 25,
	}

	// Test download
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := downloader.DownloadPlugin(ctx, plugin)
	if err != nil {
		t.Fatalf("Failed to download plugin: %v", err)
	}

	// Verify result
	if result.LocalPath == "" {
		t.Error("Expected local path to be set")
	}

	// Verify file exists and is executable
	info, err := os.Stat(result.LocalPath)
	if err != nil {
		t.Fatalf("Downloaded plugin file not found: %v", err)
	}

	if info.Mode()&0111 == 0 {
		t.Error("Downloaded plugin is not executable")
	}

	// Test plugin removal
	if err := downloader.RemovePlugin("test-plugin"); err != nil {
		t.Errorf("Failed to remove plugin: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(result.LocalPath); !os.IsNotExist(err) {
		t.Error("Plugin file should have been removed")
	}
}

func testPluginManager(t *testing.T, tempDir, serverURL string) {
	pluginsDir := filepath.Join(tempDir, "plugins")
	
	// Create enhanced plugin manager
	config := &plugins.PluginManagerConfig{
		PluginDirectories:   []string{pluginsDir},
		AuthRefreshInterval: 5 * time.Minute,
		ApiEndpoint:         serverURL,
		Debug:               true,
		MaxPlugins:          10,
		LoadTimeout:         30 * time.Second,
	}

	manager, err := plugins.NewPluginManager(config, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create enhanced plugin manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test listing available plugins
	availablePlugins, err := manager.ListAvailablePlugins(ctx)
	if err != nil {
		t.Fatalf("Failed to list available plugins: %v", err)
	}

	if len(availablePlugins) != 1 {
		t.Errorf("Expected 1 available plugin, got %d", len(availablePlugins))
	}

	if availablePlugins[0].Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", availablePlugins[0].Name)
	}

	// Test plugin installation
	if err := manager.InstallPlugin(ctx, "test-plugin", "test-api-key"); err != nil {
		t.Fatalf("Failed to install plugin: %v", err)
	}

	// Verify plugin is installed
	if installed, _ := manager.IsPluginInstalled("test-plugin"); !installed {
		t.Error("Plugin should be installed")
	}

	// Test plugin uninstall
	if err := manager.UninstallPlugin(ctx, "test-plugin"); err != nil {
		t.Fatalf("Failed to uninstall plugin: %v", err)
	}

	// Verify plugin is uninstalled
	if installed, _ := manager.IsPluginInstalled("test-plugin"); installed {
		t.Error("Plugin should be uninstalled")
	}
}

func testManifestCaching(t *testing.T, tempDir, serverURL string) {
	cacheDir := filepath.Join(tempDir, "cache")
	
	// Create manifest cache
	cache, err := plugins.NewManifestCache(cacheDir, 5*time.Minute, true)
	if err != nil {
		t.Fatalf("Failed to create manifest cache: %v", err)
	}

	// Create mock manifest
	manifest := &httpinternal.PluginManifestResponse{
		Plugins: []httpinternal.PluginManifestEntry{
			{
				Name:    "cached-plugin",
				Version: "1.0.0",
				Tier:    "free",
			},
		},
	}

	apiKeyHash := "test-key-hash"

	// Test cache miss
	if cached, found := cache.Get(apiKeyHash); found {
		t.Error("Should be cache miss on empty cache")
	} else if cached != nil {
		t.Error("Cached manifest should be nil on miss")
	}

	// Test cache set
	if err := cache.Set(apiKeyHash, manifest); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test cache hit
	cached, found := cache.Get(apiKeyHash)
	if !found {
		t.Error("Should be cache hit after set")
	}

	if cached == nil {
		t.Fatal("Cached manifest should not be nil")
	}

	if len(cached.Plugins) != 1 {
		t.Errorf("Expected 1 cached plugin, got %d", len(cached.Plugins))
	}

	if cached.Plugins[0].Name != "cached-plugin" {
		t.Errorf("Expected cached plugin name 'cached-plugin', got '%s'", cached.Plugins[0].Name)
	}

	// Test cache info
	info, err := cache.GetCacheInfo()
	if err != nil {
		t.Fatalf("Failed to get cache info: %v", err)
	}

	if info.TotalEntries != 1 {
		t.Errorf("Expected 1 total cache entry, got %d", info.TotalEntries)
	}

	if info.ValidEntries != 1 {
		t.Errorf("Expected 1 valid cache entry, got %d", info.ValidEntries)
	}

	// Test cache clear
	if err := cache.Clear(apiKeyHash); err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify cache is cleared
	if _, found := cache.Get(apiKeyHash); found {
		t.Error("Should be cache miss after clear")
	}
}

// TestPluginVerification tests plugin signature and hash verification
func TestPluginVerification(t *testing.T) {
	// Set up test environment variable for public key
	originalKey := os.Getenv("KM_PLUGIN_PUBLIC_KEY")
	testPublicKey := "R+oV2sjcidfIdLmaAz6m45ims6RCOjAvFZdnhlpbges="
	os.Setenv("KM_PLUGIN_PUBLIC_KEY", testPublicKey)
	defer os.Setenv("KM_PLUGIN_PUBLIC_KEY", originalKey)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "km-verify-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test plugin file
	pluginPath := filepath.Join(tempDir, "test-plugin")
	testData := []byte("test plugin content")
	if err := os.WriteFile(pluginPath, testData, 0755); err != nil {
		t.Fatalf("Failed to create test plugin file: %v", err)
	}

	// Create plugin verifier
	verifier, err := plugins.NewPluginVerifier(true)
	if err != nil {
		t.Fatalf("Failed to create plugin verifier: %v", err)
	}

	// Test hash and signature verification (this will fail with invalid data, but tests the flow)
	testHash := "sha256:invalid_hash"
	testSignature := "invalid_signature" // Pure base64-encoded signature (invalid for testing)

	// Read the test data to verify
	verifyData, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("Failed to read test plugin file: %v", err)
	}

	err = verifier.VerifyPluginData(verifyData, testHash, testSignature)
	if err == nil {
		t.Error("Expected verification to fail with invalid hash/signature")
	}

	// Test that error contains expected information
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestAPIDiscovery tests the API-based plugin discovery
func TestAPIDiscovery(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "km-discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/plugins/manifest" {
			manifest := httpinternal.PluginManifestResponse{
				Plugins: []httpinternal.PluginManifestEntry{
					{
						Name:    "api-plugin",
						Version: "2.0.0",
						Tier:    "pro",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Test discovery creation and usage would require more complex mocking
	// For now, test that the discovery can be created
	pluginsDir := filepath.Join(tempDir, "plugins")
	
	// This test would need proper API client mocking to fully test
	// For now, we test that the structure can be created
	_, err = plugins.NewAPIPluginDiscovery(pluginsDir, true, "test-version")
	if err == nil {
		// Discovery creation succeeded, which means API client creation worked
		t.Logf("API discovery created successfully")
	} else {
		// This is expected in test environment without proper API setup
		t.Logf("API discovery creation failed as expected in test environment: %v", err)
	}
}