package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// JWTPluginAuthenticator authenticates plugins using JWT tokens instead of API keys
// This implements the new JWT-only plugin authentication flow:
// 1. Exchange API key → JWT token via /api/auth/token
// 2. Use JWT token for plugin authentication via /api/plugins/authenticate
type JWTPluginAuthenticator struct {
	apiEndpoint   string
	tokenProvider *HTTPTokenProvider
	tokenCache    TokenCache
	httpClient    *http.Client
	apiKey        string // Stored for JWT token exchange
}

// NewJWTPluginAuthenticator creates a new JWT-based plugin authenticator
func NewJWTPluginAuthenticator(apiEndpoint string, apiKey string, tokenCache TokenCache) *JWTPluginAuthenticator {
	return &JWTPluginAuthenticator{
		apiEndpoint:   apiEndpoint,
		tokenProvider: NewHTTPTokenProvider(apiEndpoint),
		tokenCache:    tokenCache,
		apiKey:        apiKey,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 30 * time.Second,
		},
	}
}

// AuthenticatePlugin authenticates a plugin using JWT tokens
func (a *JWTPluginAuthenticator) AuthenticatePlugin(ctx context.Context, pluginName string) (*AuthResponse, error) {
	// 1. Get or refresh JWT token
	jwtToken, err := a.getOrRefreshJWT(ctx, pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT token: %w", err)
	}

	// 2. Authenticate plugin using JWT token
	return a.callPluginAuth(ctx, pluginName, jwtToken)
}

// getOrRefreshJWT gets a valid JWT token from cache or exchanges API key
func (a *JWTPluginAuthenticator) getOrRefreshJWT(ctx context.Context, pluginName string) (string, error) {
	cacheKey := fmt.Sprintf("plugin:%s", pluginName)

	// Try to get cached token first
	if cachedToken, err := a.tokenCache.GetToken(cacheKey); err == nil && cachedToken != nil {
		// Check if token is still valid (with 2-minute buffer for refresh)
		if !cachedToken.ShouldRefresh(2 * time.Minute) {
			return cachedToken.Token, nil
		}

		// Try to refresh if refresh token available
		if len(cachedToken.RefreshToken) > 0 {
			refreshedToken, err := a.tokenProvider.RefreshToken(ctx, cachedToken.RefreshToken)
			if err == nil {
				// Cache the refreshed token
				if err := a.tokenCache.SetToken(cacheKey, refreshedToken); err != nil {
					// Log warning but continue - we have a valid token
					fmt.Printf("Warning: failed to cache refreshed token: %v\n", err)
				}
				return refreshedToken.Token, nil
			}
		}
	}

	// No valid cached token, exchange API key for new JWT
	tokenRequest := &TokenRequest{
		GrantType: "api_key",
		APIKey:    a.apiKey,
		Scope:     []string{"plugins"},
	}

	authToken, err := a.tokenProvider.GetToken(ctx, tokenRequest)
	if err != nil {
		return "", fmt.Errorf("failed to exchange API key for JWT: %w", err)
	}

	// Cache the new token
	if err := a.tokenCache.SetToken(cacheKey, authToken); err != nil {
		// Log warning but continue - we have a valid token
		fmt.Printf("Warning: failed to cache JWT token: %v\n", err)
	}

	return authToken.Token, nil
}

// callPluginAuth calls the /api/plugins/authenticate endpoint with JWT token
func (a *JWTPluginAuthenticator) callPluginAuth(ctx context.Context, pluginName string, jwtToken string) (*AuthResponse, error) {
	// Prepare plugin authentication request
	pluginReq := map[string]interface{}{
		"pluginName":      pluginName,
		"pluginVersion":   "1.0.0", // TODO: Make this configurable
		"pluginSignature": a.generatePluginSignature(pluginName, "1.0.0"),
		"jwtToken":        jwtToken,
	}

	requestBody, err := json.Marshal(pluginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin auth request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/authenticate", a.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "kilometers-cli/1.0")

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send plugin auth request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return nil, fmt.Errorf("plugin authentication failed with status %d: %s", resp.StatusCode, string(body[:n]))
	}

	// Parse response
	var pluginResp struct {
		Success            bool     `json:"success"`
		Token              string   `json:"token"`
		ExpiresAt          string   `json:"expiresAt"`
		AuthorizedFeatures []string `json:"authorizedFeatures"`
		SubscriptionTier   string   `json:"subscriptionTier"`
		CustomerName       string   `json:"customerName"`
		PluginVersion      string   `json:"pluginVersion"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pluginResp); err != nil {
		return nil, fmt.Errorf("failed to decode plugin auth response: %w", err)
	}

	if !pluginResp.Success {
		return nil, fmt.Errorf("plugin authentication failed: success=false")
	}

	// Handle expiration time (AuthResponse expects *string)
	var expiresAt *string
	if pluginResp.ExpiresAt != "" {
		expiresAt = &pluginResp.ExpiresAt
	}

	return &AuthResponse{
		Authorized: true,
		UserTier:   pluginResp.SubscriptionTier,
		Features:   pluginResp.AuthorizedFeatures,
		ExpiresAt:  expiresAt,
		Token:      pluginResp.Token, // Plugin-specific JWT token
	}, nil
}

// generatePluginSignature generates a plugin signature (matches API logic)
func (a *JWTPluginAuthenticator) generatePluginSignature(pluginName, pluginVersion string) string {
	content := fmt.Sprintf("%s:%s:kilometers-plugins", pluginName, pluginVersion)
	hash := sha256.Sum256([]byte(content))
	return base64.StdEncoding.EncodeToString(hash[:])
}
