package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// HTTPPluginAuthenticator implements plugin authentication using HTTP API
type HTTPPluginAuthenticator struct {
	apiEndpoint string
	httpClient  *http.Client
	userAgent   string
}

// NewHTTPPluginAuthenticator creates a new HTTP-based plugin authenticator
func NewHTTPPluginAuthenticator(apiEndpoint string) *HTTPPluginAuthenticator {
	return &HTTPPluginAuthenticator{
		apiEndpoint: apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "kilometers-cli-plugin-loader/1.0",
	}
}

// AuthenticatePlugin sends authentication request to kilometers-api
func (a *HTTPPluginAuthenticator) AuthenticatePlugin(ctx context.Context, request *plugins.AuthRequest) (*plugins.AuthResponse, error) {
	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/authenticate", a.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.userAgent)

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send authentication request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if jsonErr := json.Unmarshal(responseBody, &errorResp); jsonErr == nil {
			return nil, fmt.Errorf("authentication failed: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	var authResp plugins.AuthResponse
	if err := json.Unmarshal(responseBody, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse authentication response: %w", err)
	}

	return &authResp, nil
}

// ValidatePlugin performs periodic validation (5-minute cycle)
func (a *HTTPPluginAuthenticator) ValidatePlugin(ctx context.Context, pluginName string, token string) (*plugins.ValidationResponse, error) {
	// Prepare request body
	request := map[string]string{
		"plugin_name":  pluginName,
		"plugin_token": token,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/validate", a.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.userAgent)

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send validation request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if jsonErr := json.Unmarshal(responseBody, &errorResp); jsonErr == nil {
			return nil, fmt.Errorf("validation failed: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	var validationResp plugins.ValidationResponse
	if err := json.Unmarshal(responseBody, &validationResp); err != nil {
		return nil, fmt.Errorf("failed to parse validation response: %w", err)
	}

	return &validationResp, nil
}

// RefreshAuthentication refreshes plugin authentication
func (a *HTTPPluginAuthenticator) RefreshAuthentication(ctx context.Context, pluginName string) (*plugins.AuthResponse, error) {
	// For now, we don't have a specific refresh endpoint
	// In production, this would call a refresh endpoint or re-authenticate
	return nil, fmt.Errorf("refresh authentication not implemented")
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error"`
}

// MemoryAuthenticationCache implements in-memory authentication caching
type MemoryAuthenticationCache struct {
	cache   map[string]*CachedAuth
	timeout time.Duration
}

// CachedAuth represents cached authentication information
type CachedAuth struct {
	Auth       *plugins.AuthResponse
	CachedAt   time.Time
	ValidUntil time.Time
}

// NewMemoryAuthenticationCache creates a new in-memory authentication cache
func NewMemoryAuthenticationCache(timeout time.Duration) *MemoryAuthenticationCache {
	if timeout == 0 {
		timeout = 5 * time.Minute // Default 5-minute cache
	}

	return &MemoryAuthenticationCache{
		cache:   make(map[string]*CachedAuth),
		timeout: timeout,
	}
}

// SetAuthentication caches authentication result for a plugin
func (c *MemoryAuthenticationCache) SetAuthentication(pluginName string, auth *plugins.AuthResponse) error {
	cachedAuth := &CachedAuth{
		Auth:       auth,
		CachedAt:   time.Now(),
		ValidUntil: auth.ExpiresAt,
	}

	c.cache[pluginName] = cachedAuth
	return nil
}

// GetAuthentication retrieves cached authentication
func (c *MemoryAuthenticationCache) GetAuthentication(pluginName string) (*plugins.AuthResponse, error) {
	cachedAuth, exists := c.cache[pluginName]
	if !exists {
		return nil, fmt.Errorf("no cached authentication for plugin %s", pluginName)
	}

	// Check if cached auth is still valid
	if !c.isCachedAuthValid(cachedAuth) {
		delete(c.cache, pluginName)
		return nil, fmt.Errorf("cached authentication for plugin %s has expired", pluginName)
	}

	return cachedAuth.Auth, nil
}

// IsValid checks if cached authentication is still valid
func (c *MemoryAuthenticationCache) IsValid(pluginName string) bool {
	cachedAuth, exists := c.cache[pluginName]
	if !exists {
		return false
	}

	return c.isCachedAuthValid(cachedAuth)
}

// ClearAuthentication removes cached authentication
func (c *MemoryAuthenticationCache) ClearAuthentication(pluginName string) error {
	delete(c.cache, pluginName)
	return nil
}

// RefreshAll refreshes all cached authentications
func (c *MemoryAuthenticationCache) RefreshAll(ctx context.Context) error {
	// For now, we'll just remove expired entries
	// In production, this would re-authenticate with the API

	for pluginName, cachedAuth := range c.cache {
		if !c.isCachedAuthValid(cachedAuth) {
			delete(c.cache, pluginName)
		}
	}

	return nil
}

// Private methods

func (c *MemoryAuthenticationCache) isCachedAuthValid(cachedAuth *CachedAuth) bool {
	now := time.Now()

	// Check if cache entry itself has expired (based on cache timeout)
	if now.Sub(cachedAuth.CachedAt) > c.timeout {
		return false
	}

	// Check if the authentication token has expired
	if now.After(cachedAuth.ValidUntil) {
		return false
	}

	return true
}
