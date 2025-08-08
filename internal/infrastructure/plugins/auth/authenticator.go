package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// HTTPPluginAuthenticator implements plugin authentication via HTTP API
type HTTPPluginAuthenticator struct {
	apiEndpoint string
	httpClient  *http.Client
	debug       bool
}

// NewHTTPPluginAuthenticator creates a new HTTP-based plugin authenticator
func NewHTTPPluginAuthenticator(apiEndpoint string, debug bool) *HTTPPluginAuthenticator {
	return &HTTPPluginAuthenticator{
		apiEndpoint: apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: debug,
	}
}

// AuthenticatePlugin sends authentication request to kilometers-api
func (a *HTTPPluginAuthenticator) AuthenticatePlugin(ctx context.Context, pluginName string, apiKey string) (*ports.AuthResponse, error) {
	if a.debug {
		fmt.Printf("[PluginAuthenticator] Authenticating plugin %s with API\n", pluginName)
	}

	// Create authentication request
	authReq := PluginAuthRequest{
		PluginName: pluginName,
		APIKey:     apiKey,
		Timestamp:  time.Now().Unix(),
	}

	// Send request to API
	response, err := a.sendAuthRequest(ctx, authReq)
	if err != nil {
		if a.debug {
			fmt.Printf("[PluginAuthenticator] Authentication failed for %s: %v\n", pluginName, err)
		}
		return nil, err
	}

	if a.debug {
		fmt.Printf("[PluginAuthenticator] Authentication successful for %s (tier: %s)\n",
			pluginName, response.UserTier)
	}

	return response, nil
}

// sendAuthRequest sends the authentication request to the API
func (a *HTTPPluginAuthenticator) sendAuthRequest(ctx context.Context, authReq PluginAuthRequest) (*ports.AuthResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/authenticate", a.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authReq.APIKey))

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		// For POC, fall back to mock response if API is unavailable
		if a.debug {
			fmt.Printf("[PluginAuthenticator] API unavailable, using mock response: %v\n", err)
		}
		return a.createMockResponse(authReq.PluginName), nil
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var authResp PluginAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %w", err)
	}

	if !authResp.Success {
		return nil, fmt.Errorf("authentication rejected: %s", authResp.Error)
	}

	return &ports.AuthResponse{
		Authorized: authResp.Authorized,
		UserTier:   authResp.UserTier,
		Features:   authResp.Features,
		ExpiresAt:  authResp.ExpiresAt,
	}, nil
}

// createMockResponse creates a mock authentication response for testing
func (a *HTTPPluginAuthenticator) createMockResponse(pluginName string) *ports.AuthResponse {
	// For POC, provide mock responses based on plugin name
	tier := "Free"
	features := []string{"console_logging"}

	switch pluginName {
	case "api-logger":
		tier = "Pro"
		features = []string{"console_logging", "api_logging", "basic_analytics"}
	case "advanced-analytics":
		tier = "Pro"
		features = []string{"console_logging", "api_logging", "basic_analytics", "advanced_analytics"}
	case "enterprise-security":
		tier = "Enterprise"
		features = []string{"console_logging", "api_logging", "basic_analytics", "advanced_analytics", "security_monitoring", "compliance_reporting"}
	}

	expiry := time.Now().Add(5 * time.Minute).Format(time.RFC3339)

	return &ports.AuthResponse{
		Authorized: true,
		UserTier:   tier,
		Features:   features,
		ExpiresAt:  &expiry,
	}
}

// Request/Response types for plugin authentication

// PluginAuthRequest represents the authentication request sent to the API
type PluginAuthRequest struct {
	PluginName string `json:"plugin_name"`
	APIKey     string `json:"api_key"`
	Timestamp  int64  `json:"timestamp"`
}

// PluginAuthResponse represents the authentication response from the API
type PluginAuthResponse struct {
	Success    bool     `json:"success"`
	Error      string   `json:"error,omitempty"`
	Authorized bool     `json:"authorized"`
	UserTier   string   `json:"user_tier"`
	Features   []string `json:"features"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	CustomerID string   `json:"customer_id,omitempty"`
	PluginID   string   `json:"plugin_id,omitempty"`
}

// Production authentication methods (placeholders for future implementation)

// authenticateWithCustomerCredentials would authenticate using customer-specific credentials
func (a *HTTPPluginAuthenticator) authenticateWithCustomerCredentials(ctx context.Context, pluginName string, customerID string, apiKey string) (*ports.AuthResponse, error) {
	// In production, this would:
	// 1. Extract embedded customer credentials from plugin
	// 2. Send customer ID + plugin ID + API key to authentication endpoint
	// 3. Receive back customer-specific authorization with features
	// 4. Cache the result with appropriate expiration

	return nil, fmt.Errorf("customer credential authentication not implemented in POC")
}

// validatePluginJWT would validate a JWT token embedded in the plugin
func (a *HTTPPluginAuthenticator) validatePluginJWT(jwtToken string) (*ports.AuthResponse, error) {
	// In production, this would:
	// 1. Parse and validate the JWT signature
	// 2. Extract customer ID, plugin ID, and permissions from claims
	// 3. Verify token has not expired
	// 4. Return parsed authorization information

	return nil, fmt.Errorf("JWT validation not implemented in POC")
}

// refreshAuthToken would refresh an expired authentication token
func (a *HTTPPluginAuthenticator) refreshAuthToken(ctx context.Context, refreshToken string) (*ports.AuthResponse, error) {
	// In production, this would:
	// 1. Send refresh token to API
	// 2. Receive new access token and expiration
	// 3. Update cached authentication
	// 4. Return refreshed authorization

	return nil, fmt.Errorf("token refresh not implemented in POC")
}
