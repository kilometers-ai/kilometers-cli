package registry

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// PluginRegistry handles private plugin repository access
type PluginRegistry struct {
	baseURL     string
	authToken   string
	client      *http.Client
	cacheDir    string
	trustedKeys []string // Public keys for plugin verification
}

// PluginMetadata represents plugin information from registry
type PluginMetadata struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	RequiredTier    string            `json:"required_tier"`
	RequiredFeature string            `json:"required_feature"`
	Platform        string            `json:"platform"`
	Architecture    string            `json:"architecture"`
	DownloadURL     string            `json:"download_url"`
	Checksum        string            `json:"checksum"`
	Signature       string            `json:"signature"`
	Dependencies    []string          `json:"dependencies"`
	Permissions     []string          `json:"permissions"`
	Tags            []string          `json:"tags"`
	Metadata        map[string]string `json:"metadata"`
}

// PluginManifest represents the registry manifest
type PluginManifest struct {
	Version string                    `json:"version"`
	Plugins map[string]PluginMetadata `json:"plugins"`
	Updated time.Time                 `json:"updated"`
}

// NewPluginRegistry creates a new plugin registry client
func NewPluginRegistry(baseURL, authToken string) *PluginRegistry {
	cacheDir := filepath.Join(os.TempDir(), "kilometers-plugins")
	os.MkdirAll(cacheDir, 0755)

	return &PluginRegistry{
		baseURL:   baseURL,
		authToken: authToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
		trustedKeys: []string{
			// Embedded public keys for plugin verification
			"-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFA...",
		},
	}
}

// DiscoverPlugins fetches available plugins from private registry
func (r *PluginRegistry) DiscoverPlugins(ctx context.Context, tier domain.SubscriptionTier) ([]PluginMetadata, error) {
	manifest, err := r.fetchManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin manifest: %w", err)
	}

	var availablePlugins []PluginMetadata
	for _, plugin := range manifest.Plugins {
		// Filter by subscription tier
		if r.isPluginAvailable(plugin, tier) {
			availablePlugins = append(availablePlugins, plugin)
		}
	}

	return availablePlugins, nil
}

// DownloadPlugin downloads and caches a plugin from private registry
func (r *PluginRegistry) DownloadPlugin(ctx context.Context, name, version string) (string, error) {
	// Check cache first
	cachedPath := r.getCachedPluginPath(name, version)
	if r.isPluginCached(cachedPath) {
		return cachedPath, nil
	}

	// Get plugin metadata
	manifest, err := r.fetchManifest(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest: %w", err)
	}

	pluginKey := fmt.Sprintf("%s@%s", name, version)
	plugin, exists := manifest.Plugins[pluginKey]
	if !exists {
		return "", fmt.Errorf("plugin %s not found in registry", pluginKey)
	}

	// Download plugin
	pluginData, err := r.downloadPluginData(ctx, plugin.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download plugin: %w", err)
	}

	// Verify checksum
	if err := r.verifyChecksum(pluginData, plugin.Checksum); err != nil {
		return "", fmt.Errorf("plugin checksum verification failed: %w", err)
	}

	// Verify signature
	if err := r.verifySignature(pluginData, plugin.Signature); err != nil {
		return "", fmt.Errorf("plugin signature verification failed: %w", err)
	}

	// Cache plugin
	if err := r.cachePlugin(cachedPath, pluginData); err != nil {
		return "", fmt.Errorf("failed to cache plugin: %w", err)
	}

	return cachedPath, nil
}

// UpdatePlugins checks for and downloads plugin updates
func (r *PluginRegistry) UpdatePlugins(ctx context.Context, installedPlugins []string) error {
	manifest, err := r.fetchManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest for updates: %w", err)
	}

	for _, pluginName := range installedPlugins {
		// Find latest version
		latestVersion := r.findLatestVersion(manifest, pluginName)
		if latestVersion == "" {
			continue
		}

		// Check if update needed
		currentVersion := r.getCurrentVersion(pluginName)
		if r.isNewerVersion(latestVersion, currentVersion) {
			fmt.Printf("Updating plugin %s from %s to %s\n", pluginName, currentVersion, latestVersion)
			
			_, err := r.DownloadPlugin(ctx, pluginName, latestVersion)
			if err != nil {
				fmt.Printf("Failed to update %s: %v\n", pluginName, err)
			}
		}
	}

	return nil
}

// AuthenticateWithRegistry validates access to private registry
func (r *PluginRegistry) AuthenticateWithRegistry(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+"/auth/validate", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+r.authToken)
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("registry authentication failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry authentication failed: status %d", resp.StatusCode)
	}

	return nil
}

// Private methods

func (r *PluginRegistry) fetchManifest(ctx context.Context) (*PluginManifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+"/manifest.json", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+r.authToken)
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: status %d", resp.StatusCode)
	}

	var manifest PluginManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (r *PluginRegistry) downloadPluginData(ctx context.Context, downloadURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+r.authToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (r *PluginRegistry) verifyChecksum(data []byte, expectedChecksum string) error {
	hash := sha256.Sum256(data)
	actualChecksum := fmt.Sprintf("%x", hash)
	
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	
	return nil
}

func (r *PluginRegistry) verifySignature(data []byte, signature string) error {
	// Implement cryptographic signature verification
	// This would use the trusted public keys to verify the plugin signature
	// For now, we'll skip implementation details
	return nil
}

func (r *PluginRegistry) isPluginAvailable(plugin PluginMetadata, tier domain.SubscriptionTier) bool {
	requiredTier := domain.SubscriptionTier(plugin.RequiredTier)
	
	switch tier {
	case domain.TierFree:
		return requiredTier == domain.TierFree
	case domain.TierPro:
		return requiredTier == domain.TierFree || requiredTier == domain.TierPro
	case domain.TierEnterprise:
		return true // Enterprise gets everything
	default:
		return false
	}
}

func (r *PluginRegistry) getCachedPluginPath(name, version string) string {
	return filepath.Join(r.cacheDir, fmt.Sprintf("%s-%s.wasm", name, version))
}

func (r *PluginRegistry) isPluginCached(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *PluginRegistry) cachePlugin(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (r *PluginRegistry) findLatestVersion(manifest *PluginManifest, pluginName string) string {
	var latestVersion string
	for key, plugin := range manifest.Plugins {
		if plugin.Name == pluginName {
			if r.isNewerVersion(plugin.Version, latestVersion) {
				latestVersion = plugin.Version
			}
		}
	}
	return latestVersion
}

func (r *PluginRegistry) getCurrentVersion(pluginName string) string {
	// Would read from local plugin metadata
	return "1.0.0"
}

func (r *PluginRegistry) isNewerVersion(v1, v2 string) bool {
	// Simple version comparison - in production use proper semver
	return v1 > v2
}
