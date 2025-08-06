package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// HTTPTokenProvider implements token provider using HTTP API
type HTTPTokenProvider struct {
	apiEndpoint string
	httpClient  *http.Client
}

// NewHTTPTokenProvider creates a new HTTP-based token provider
func NewHTTPTokenProvider(apiEndpoint string) *HTTPTokenProvider {
	return &HTTPTokenProvider{
		apiEndpoint: apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetToken obtains a new token using the provided request
func (p *HTTPTokenProvider) GetToken(ctx context.Context, request *domain.TokenRequest) (*domain.AuthToken, error) {
	// Prepare request body
	body := map[string]interface{}{
		"grant_type": request.GrantType,
		"api_key":    request.APIKey,
	}

	if len(request.Scope) > 0 {
		body["scope"] = request.Scope
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/auth/token", p.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp domain.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResp.ToAuthToken(), nil
}

// RefreshToken refreshes an existing token
func (p *HTTPTokenProvider) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthToken, error) {
	// Prepare request body
	body := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/auth/token", p.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp domain.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResp.ToAuthToken(), nil
}

// ValidateToken checks if a token is still valid
func (p *HTTPTokenProvider) ValidateToken(ctx context.Context, token string) (bool, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/auth/validate", p.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("validation request failed with status %d: %s", resp.StatusCode, string(body))
}
