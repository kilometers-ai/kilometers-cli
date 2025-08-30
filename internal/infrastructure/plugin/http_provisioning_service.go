package plugininfra

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	apphttp "github.com/kilometers-ai/kilometers-cli/internal/application/http"
	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
)

// HTTPProvisioningService implements plugin provisioning via HTTP API
type HTTPProvisioningService struct {
	client *apphttp.BackendClient
}

// NewHTTPProvisioningService creates a new HTTP-based provisioning service
func NewHTTPProvisioningService(client *apphttp.BackendClient) *HTTPProvisioningService {
	return &HTTPProvisioningService{
		client: client,
	}
}

// ValidateAPIKey validates an API key and returns subscription info
func (s *HTTPProvisioningService) ValidateAPIKey(ctx context.Context, apiKey string) (*plugindomain.ValidationResult, error) {
	// Call the API to validate the key and get subscription info
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	status, _, body, err := s.client.GetJSON(ctx, "/api/subscription/status", headers, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	if status == 401 || status == 403 {
		return &plugindomain.ValidationResult{
			IsValid: false,
			Message: "Invalid or expired API key",
		}, nil
	}

	if status != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", status)
	}

	var response struct {
		Success      bool     `json:"success"`
		CustomerID   string   `json:"customer_id"`
		CustomerName string   `json:"customer_name"`
		Tier         string   `json:"tier"`
		Features     []string `json:"features"`
		Message      string   `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return &plugindomain.ValidationResult{
			IsValid: false,
			Message: response.Message,
		}, nil
	}

	return &plugindomain.ValidationResult{
		IsValid: true,
		Subscription: &plugindomain.Subscription{
			Tier:              plugindomain.SubscriptionTier(response.Tier),
			CustomerID:        response.CustomerID,
			CustomerName:      response.CustomerName,
			AvailableFeatures: response.Features,
		},
	}, nil
}

// GetAvailablePlugins returns plugins available for the subscription
func (s *HTTPProvisioningService) GetAvailablePlugins(ctx context.Context, apiKey string) ([]plugindomain.Plugin, error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	// Include platform information
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	query := map[string]string{
		"platform": platform,
	}

	status, _, body, err := s.client.GetJSON(ctx, "/api/plugins/available", headers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get available plugins: %w", err)
	}

	if status != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", status)
	}

	var response struct {
		Success bool                  `json:"success"`
		Plugins []plugindomain.Plugin `json:"plugins"`
		Message string                `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("failed to get plugins: %s", response.Message)
	}

	return response.Plugins, nil
}

// DownloadPlugin downloads a specific plugin
func (s *HTTPProvisioningService) DownloadPlugin(ctx context.Context, apiKey string, pluginName string) ([]byte, error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	// Include platform information
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	payload := map[string]interface{}{
		"plugin_name": pluginName,
		"platform":    platform,
	}

	status, _, body, err := s.client.PostJSON(ctx, "/api/plugins/download", payload, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}

	if status != 200 {
		// Try to parse error response
		var errorResp struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Message != "" {
			return nil, fmt.Errorf("download failed: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", status)
	}

	// The response should contain a download URL
	var response struct {
		Success     bool   `json:"success"`
		DownloadURL string `json:"download_url"`
		Checksum    string `json:"checksum"`
		Message     string `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("download failed: %s", response.Message)
	}

	// Download the actual plugin binary
	return s.downloadFromURL(ctx, response.DownloadURL)
}

// downloadFromURL downloads data from a URL
func (s *HTTPProvisioningService) downloadFromURL(ctx context.Context, downloadURL string) ([]byte, error) {
	// For URLs that are relative to our API server, extract just the path
	// Otherwise, use the full URL
	path := downloadURL
	if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
		// Parse the URL to get just the path
		if idx := strings.Index(downloadURL, "/download/"); idx >= 0 {
			path = downloadURL[idx:]
		}
	}

	status, _, body, err := s.client.GetStream(ctx, path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download from URL: %w", err)
	}

	if status != 200 {
		return nil, fmt.Errorf("download failed with status %d", status)
	}

	// Body already contains the data from the response
	return body, nil
}
