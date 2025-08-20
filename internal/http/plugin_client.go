package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

// PluginManifestEntry represents a single plugin in the manifest
type PluginManifestEntry struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Tier      string `json:"tier"`
	URL       string `json:"url"`       // Points to API proxy, not GitHub
	Hash      string `json:"hash"`      // SHA256 hash of the binary
	Signature string `json:"signature"` // Ed25519 signature
	Size      int64  `json:"size"`      // Binary size in bytes
}

// PluginManifestResponse represents the API response for plugin manifest
type PluginManifestResponse struct {
	Plugins []PluginManifestEntry `json:"plugins"`
}

// PluginDownloadProgress tracks download progress
type PluginDownloadProgress struct {
	TotalBytes     int64
	DownloadedBytes int64
	ProgressFunc   func(downloaded, total int64)
}

// PluginApiClient handles plugin-related API operations
type PluginApiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	debug   bool
}

// NewPluginApiClient creates a new plugin API client using unified configuration
func NewPluginApiClient(debug bool) (*PluginApiClient, error) {
	loader, storage, err := config.CreateConfigServiceFromDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to create config service: %w", err)
	}
	configService := config.NewConfigService(loader, storage)

	ctx := context.Background()
	unifiedConfig, err := configService.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if !unifiedConfig.HasAPIKey() {
		return nil, fmt.Errorf("no API key configured")
	}

	return &PluginApiClient{
		baseURL: unifiedConfig.APIEndpoint,
		apiKey:  unifiedConfig.APIKey,
		client: &http.Client{
			Timeout: 30 * time.Second, // Longer timeout for downloads
		},
		debug: debug,
	}, nil
}

// GetPluginManifest retrieves the available plugins manifest from the API
func (c *PluginApiClient) GetPluginManifest(ctx context.Context) (*PluginManifestResponse, error) {
	url := fmt.Sprintf("%s/api/plugins/manifest", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("X-API-Key", c.apiKey)

	if c.debug {
		fmt.Printf("[PluginApiClient] Fetching plugin manifest from %s\n", url)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid or expired API key")
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: insufficient subscription tier")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var manifest PluginManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if c.debug {
		fmt.Printf("[PluginApiClient] Retrieved manifest with %d plugins\n", len(manifest.Plugins))
	}

	return &manifest, nil
}

// DownloadPlugin downloads a plugin binary through the API proxy
func (c *PluginApiClient) DownloadPlugin(ctx context.Context, downloadURL string, progress *PluginDownloadProgress) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("X-API-Key", c.apiKey)

	if c.debug {
		fmt.Printf("[PluginApiClient] Downloading plugin from %s\n", downloadURL)
	}

	// Use a client without timeout for downloads
	downloadClient := &http.Client{}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid or expired API key")
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: insufficient subscription tier for this plugin")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Get content length for progress tracking
	contentLength := resp.ContentLength
	if progress != nil && contentLength > 0 {
		progress.TotalBytes = contentLength
	}

	// Read the response body with progress tracking
	var buf bytes.Buffer
	if progress != nil && progress.ProgressFunc != nil {
		// Create a reader that tracks progress
		reader := &progressReader{
			reader:   resp.Body,
			progress: progress,
		}
		_, err = io.Copy(&buf, reader)
	} else {
		_, err = io.Copy(&buf, resp.Body)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}

	data := buf.Bytes()

	if c.debug {
		fmt.Printf("[PluginApiClient] Downloaded %d bytes\n", len(data))
	}

	return data, nil
}

// DownloadPluginStream downloads a plugin and returns a reader for streaming
func (c *PluginApiClient) DownloadPluginStream(ctx context.Context, downloadURL string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("X-API-Key", c.apiKey)

	// Use a client without timeout for downloads
	downloadClient := &http.Client{}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("unauthorized: invalid or expired API key")
	}

	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("forbidden: insufficient subscription tier for this plugin")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, resp.ContentLength, nil
}

// progressReader wraps an io.Reader to track download progress
type progressReader struct {
	reader   io.Reader
	progress *PluginDownloadProgress
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.progress != nil {
		r.progress.DownloadedBytes += int64(n)
		if r.progress.ProgressFunc != nil {
			r.progress.ProgressFunc(r.progress.DownloadedBytes, r.progress.TotalBytes)
		}
	}
	return n, err
}

// GetPluginByName finds a specific plugin in the manifest by name
func (c *PluginApiClient) GetPluginByName(ctx context.Context, name string) (*PluginManifestEntry, error) {
	manifest, err := c.GetPluginManifest(ctx)
	if err != nil {
		return nil, err
	}

	for _, plugin := range manifest.Plugins {
		if plugin.Name == name {
			return &plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin %s not found in manifest", name)
}

// CheckPluginAccess verifies if the user has access to a specific plugin
func (c *PluginApiClient) CheckPluginAccess(ctx context.Context, pluginName string) (bool, error) {
	manifest, err := c.GetPluginManifest(ctx)
	if err != nil {
		// If we can't get the manifest, assume no access
		return false, err
	}

	// Check if the plugin is in the manifest (API filters by tier)
	for _, plugin := range manifest.Plugins {
		if plugin.Name == pluginName {
			return true, nil
		}
	}

	return false, nil
}