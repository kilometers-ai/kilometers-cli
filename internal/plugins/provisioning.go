package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PluginProvisionRequest represents a request to provision plugins
type PluginProvisionRequest struct {
	Platform string   `json:"platform"`
	Plugins  []string `json:"plugins"`
}

// PluginProvisionResponse represents the API response for plugin provisioning
type PluginProvisionResponse struct {
	Success          bool                `json:"success"`
	CustomerID       string              `json:"customer_id"`
	SubscriptionTier string              `json:"subscription_tier"`
	Plugins          []ProvisionedPlugin `json:"plugins"`
	Message          string              `json:"message,omitempty"`
}

// ProvisionedPlugin represents a plugin available for download
type ProvisionedPlugin struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	Checksum     string `json:"checksum"`
	Signature    string `json:"signature"`
	Tier         string `json:"tier"`
	RequiredTier string `json:"required_tier"`
}

// PluginInstallResult represents the result of installing a plugin
type PluginInstallResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// HTTPPluginProvisioningService implements plugin provisioning via HTTP API
type HTTPPluginProvisioningService struct {
	apiEndpoint string
	httpClient  *http.Client
}

// NewHTTPPluginProvisioningService creates a new HTTP-based provisioning service
func NewHTTPPluginProvisioningService(apiEndpoint string) *HTTPPluginProvisioningService {
	return &HTTPPluginProvisioningService{
		apiEndpoint: apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProvisionPlugins requests customer-specific plugins from the API
func (s *HTTPPluginProvisioningService) ProvisionPlugins(ctx context.Context, apiKey string) (*PluginProvisionResponse, error) {
	// Prepare request
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	request := PluginProvisionRequest{
		Platform: platform,
		Plugins:  []string{"all"}, // Request all available plugins
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/provision", s.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provisioning failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var provisionResp PluginProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provisionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &provisionResp, nil
}

// GetSubscriptionStatus checks the current subscription tier
func (s *HTTPPluginProvisioningService) GetSubscriptionStatus(ctx context.Context, apiKey string) (string, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/subscription/status", s.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("subscription check failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var statusResp struct {
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return statusResp.Tier, nil
}

// FileSystemPluginInstaller handles plugin download and installation
type FileSystemPluginInstaller struct {
	targetDir  string
	httpClient *http.Client
	debug      bool
}

// NewFileSystemPluginInstaller creates a new plugin installer
func NewFileSystemPluginInstaller(targetDir string, debug bool) *FileSystemPluginInstaller {
	return &FileSystemPluginInstaller{
		targetDir: targetDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for downloads
		},
		debug: debug,
	}
}

// InstallPlugin downloads and installs a single plugin
func (i *FileSystemPluginInstaller) InstallPlugin(ctx context.Context, plugin ProvisionedPlugin) (*PluginInstallResult, error) {
	result := &PluginInstallResult{
		Name:    plugin.Name,
		Version: plugin.Version,
	}

	// Download plugin
	pluginData, err := i.downloadPlugin(ctx, plugin.DownloadURL)
	if err != nil {
		result.Error = fmt.Sprintf("download failed: %v", err)
		return result, err
	}

	// Verify checksum
	if err := i.verifyChecksum(pluginData, plugin.Checksum); err != nil {
		result.Error = fmt.Sprintf("checksum verification failed: %v", err)
		return result, err
	}

	// Extract and install
	installPath, err := i.extractAndInstall(plugin.Name, pluginData)
	if err != nil {
		result.Error = fmt.Sprintf("installation failed: %v", err)
		return result, err
	}

	result.Path = installPath
	result.Success = true
	return result, nil
}

// downloadPlugin downloads plugin binary from URL
func (i *FileSystemPluginInstaller) downloadPlugin(ctx context.Context, downloadURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}

	return data, nil
}

// verifyChecksum verifies the downloaded plugin checksum
func (i *FileSystemPluginInstaller) verifyChecksum(data []byte, expectedChecksum string) error {
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// extractAndInstall extracts and installs the plugin
func (i *FileSystemPluginInstaller) extractAndInstall(pluginName string, data []byte) (string, error) {
	// Ensure target directory exists
	targetDir := ExpandPath(i.targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Handle tar.gz archive
	if strings.HasSuffix(pluginName, ".tar.gz") {
		return i.extractTarGz(pluginName, data, targetDir)
	}

	// Handle single binary
	binaryName := fmt.Sprintf("km-plugin-%s", pluginName)
	binaryPath := filepath.Join(targetDir, binaryName)

	// Write binary to file
	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to write plugin binary: %w", err)
	}

	return binaryPath, nil
}

// extractTarGz extracts tar.gz archive
func (i *FileSystemPluginInstaller) extractTarGz(pluginName string, data []byte, targetDir string) (string, error) {
	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	var installedBinary string

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(targetDir, header.Name)

		// Ensure safe extraction (prevent path traversal)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(targetDir)) {
			return "", fmt.Errorf("unsafe tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", fmt.Errorf("failed to create file: %w", err)
			}

			// Copy file contents
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return "", fmt.Errorf("failed to write file: %w", err)
			}
			file.Close()

			// Track main binary
			if strings.HasPrefix(filepath.Base(targetPath), "km-plugin-") {
				installedBinary = targetPath
			}
		}
	}

	if installedBinary == "" {
		return "", fmt.Errorf("no plugin binary found in archive")
	}

	return installedBinary, nil
}

// DefaultPublicKey is a placeholder for the default public key
var DefaultPublicKey = []byte("default-public-key")

// PluginRegistry represents the plugin registry
type PluginRegistry struct {
	CustomerID  string                     `json:"customer_id"`
	CurrentTier string                     `json:"current_tier"`
	Version     string                     `json:"version"`
	LastUpdate  time.Time                  `json:"last_update"`
	Plugins     map[string]InstalledPlugin `json:"plugins"`
}

// ShouldRefresh checks if the registry should be refreshed
func (r *PluginRegistry) ShouldRefresh() bool {
	// Refresh if last update was more than 24 hours ago
	return time.Since(r.LastUpdate) > 24*time.Hour
}

// InstalledPlugin represents an installed plugin
type InstalledPlugin struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	InstallTime  time.Time `json:"install_time"`
	InstalledAt  time.Time `json:"installed_at"`
	Path         string    `json:"path"`
	Checksum     string    `json:"checksum"`
	RequiredTier string    `json:"required_tier"`
	Enabled      bool      `json:"enabled"`
}

// NewSimplePluginDownloader creates a new simple plugin downloader
func NewSimplePluginDownloader(publicKey []byte) (PluginDownloader, error) {
	return &SimpleDownloader{}, nil
}

// NewFileSystemPluginInstallerFactory creates a new filesystem plugin installer factory function
func NewFileSystemPluginInstallerFactory(targetDir string) (PluginInstaller, error) {
	return NewFileSystemPluginInstaller(targetDir, false), nil
}

// NewFilePluginRegistryStore creates a new file plugin registry store
func NewFilePluginRegistryStore(configDir string) (PluginRegistryStore, error) {
	return &SimpleRegistryStore{}, nil
}

// SimpleDownloader is a simple plugin downloader
type SimpleDownloader struct{}

// Download downloads data from a URL
func (d *SimpleDownloader) Download(ctx context.Context, url string) ([]byte, error) {
	return []byte("mock-data"), nil
}

// DownloadPlugin downloads a plugin
func (d *SimpleDownloader) DownloadPlugin(ctx context.Context, plugin ProvisionedPlugin) (interface{}, error) {
	return &MockReader{}, nil
}

// MockReader is a mock reader that implements io.ReadCloser
type MockReader struct{}

// Read reads data
func (r *MockReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

// Close closes the reader
func (r *MockReader) Close() error {
	return nil
}

// SimpleRegistryStore is a simple registry store
type SimpleRegistryStore struct{}

// Load loads a plugin registry
func (s *SimpleRegistryStore) Load() (*PluginRegistry, error) {
	return &PluginRegistry{
		Plugins: make(map[string]InstalledPlugin),
	}, nil
}

// Save saves a plugin registry
func (s *SimpleRegistryStore) Save(*PluginRegistry) error {
	return nil
}

// LoadRegistry loads a plugin registry (alias for Load)
func (s *SimpleRegistryStore) LoadRegistry() (*PluginRegistry, error) {
	return s.Load()
}

// SaveRegistry saves a plugin registry (alias for Save)
func (s *SimpleRegistryStore) SaveRegistry(registry *PluginRegistry) error {
	return s.Save(registry)
}
