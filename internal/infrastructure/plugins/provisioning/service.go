package provisioning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

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
func (s *HTTPPluginProvisioningService) ProvisionPlugins(ctx context.Context, apiKey string) (*domain.PluginProvisionResponse, error) {
	// Prepare request
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	request := map[string]interface{}{
		"platform": platform,
		"plugins":  []string{"all"}, // Request all available plugins
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
	var provisionResp domain.PluginProvisionResponse
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

	// Set headers
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
		return "", fmt.Errorf("status check failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var statusResp struct {
		SubscriptionTier string `json:"subscription_tier"`
		Active           bool   `json:"active"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !statusResp.Active {
		return "", fmt.Errorf("subscription is not active")
	}

	return statusResp.SubscriptionTier, nil
}
