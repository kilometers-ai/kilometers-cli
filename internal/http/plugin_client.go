package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

// PluginManifestEntry represents a single plugin in the manifest
type PluginManifestEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Tier    string `json:"tier"`
	URL     string `json:"url"`  // Points to API proxy, not GitHub
	Hash    string `json:"hash"` // SHA256 hash of the binary
	//Signature string `json:"signature"` // Ed25519 signature
	Size int64 `json:"size"` // Binary size in bytes
}

// PluginManifestResponse represents the API response for plugin manifest
type PluginManifestResponse struct {
	Plugins []PluginManifestEntry `json:"plugins"`
}

// PluginManifestRequest represents the request payload for POST /plugins/manifest
type PluginManifestRequest struct {
	Plugins    []InstalledPluginInfo `json:"plugins"`
	Platform   PlatformInfo          `json:"platform"`
	CLIVersion string                `json:"cliVersion"`
}

// InstalledPluginInfo represents information about an installed plugin
type InstalledPluginInfo struct {
	Name             string  `json:"name"`
	InstalledVersion *string `json:"installedVersion"` // nullable
}

// PlatformInfo represents platform information
type PlatformInfo struct {
	OS   string `json:"os"`   // linux, darwin, windows
	Arch string `json:"arch"` // amd64, arm64, 386, etc
}

// PluginDownloadProgress tracks download progress
type PluginDownloadProgress struct {
	TotalBytes      int64
	DownloadedBytes int64
	ProgressFunc    func(downloaded, total int64)
}

// PluginApiClient handles plugin-related API operations
type PluginApiClient struct {
	baseURL    string
	apiKey     string
	client     *http.Client
	debug      bool
	cliVersion string
}

// NewPluginApiClient creates a new plugin API client using unified configuration
func NewPluginApiClient(debug bool, cliVersion string) (*PluginApiClient, error) {
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
			Timeout: 240 * time.Second, // Longer timeout for downloads
		},
		debug:      debug,
		cliVersion: cliVersion,
	}, nil
}

// GetPluginManifest retrieves the available plugins manifest using the POST endpoint
// This method gets installed plugins from the system and sends them to the API
func (c *PluginApiClient) GetPluginManifest(ctx context.Context) (*PluginManifestResponse, error) {
	// For backwards compatibility in the interface, we'll call PostPluginManifest with empty installed plugins
	// In practice, callers should use PostPluginManifest directly with actual installed plugin info
	return c.PostPluginManifest(ctx, []InstalledPluginInfo{})
}

// PostPluginManifest sends installed plugin information to the API and retrieves available plugins
func (c *PluginApiClient) PostPluginManifest(ctx context.Context, installedPlugins []InstalledPluginInfo) (*PluginManifestResponse, error) {
	// Prepare request payload
	request := PluginManifestRequest{
		Plugins: installedPlugins,
		Platform: PlatformInfo{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		CLIVersion: "0.1.0", //TODO, Replace with actual CLI version
	}

	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/plugins/manifest", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", fmt.Sprintf("%s", c.apiKey))

	if c.debug {
		fmt.Printf("[PluginApiClient] Posting plugin manifest to %s with %d installed plugins\n", url, len(installedPlugins))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad request: %s", string(body))
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid or expired API key")
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
		fmt.Printf("[PluginApiClient] Retrieved manifest with %d plugins from POST request\n", len(manifest.Plugins))
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
	//req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("X-API-Key", c.apiKey)

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
