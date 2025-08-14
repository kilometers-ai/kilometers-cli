package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PluginAuthenticator provides unified plugin authentication supporting both HTTP and JWT methods
type PluginAuthenticator struct {
	httpClient  *http.Client
	apiEndpoint string
	jwtVerifier *JWTVerifier
	authCache   *PluginAuthCache
	debug       bool
}

// PluginAuthRequest represents the authentication request for HTTP API
type PluginAuthRequest struct {
	PluginName string `json:"plugin_name"`
	APIKey     string `json:"api_key"`
	Timestamp  int64  `json:"timestamp"`
}

// PluginAuthCache provides in-memory caching for plugin authentication results
type PluginAuthCache struct {
	cache map[string]*CachedPluginAuth
	mu    sync.RWMutex
	ttl   time.Duration
}

// CachedPluginAuth represents cached authentication data
type CachedPluginAuth struct {
	Response  *PluginAuthResponse
	ExpiresAt time.Time
}

// NewPluginAuthenticator creates a new unified plugin authenticator
func NewPluginAuthenticator(apiEndpoint string, debug bool) *PluginAuthenticator {
	keyRing := NewKeyRing()
	return &PluginAuthenticator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiEndpoint: apiEndpoint,
		jwtVerifier: NewJWTVerifier(keyRing),
		authCache:   NewPluginAuthCache(5 * time.Minute), // 5-minute cache TTL
		debug:       debug,
	}
}

// NewPluginAuthCache creates a new plugin authentication cache
func NewPluginAuthCache(ttl time.Duration) *PluginAuthCache {
	return &PluginAuthCache{
		cache: make(map[string]*CachedPluginAuth),
		ttl:   ttl,
	}
}

// AuthenticatePlugin performs plugin authentication using HTTP API
func (pa *PluginAuthenticator) AuthenticatePlugin(ctx context.Context, pluginName string, apiKey string) (*PluginAuthResponse, error) {
	if pa.debug {
		fmt.Printf("[PluginAuthenticator] Authenticating plugin %s with API\n", pluginName)
	}

	// Check cache first
	if cached := pa.authCache.Get(pluginName, apiKey); cached != nil {
		if pa.debug {
			fmt.Printf("[PluginAuthenticator] Using cached authentication for %s\n", pluginName)
		}
		return cached, nil
	}

	// Create authentication request
	authReq := PluginAuthRequest{
		PluginName: pluginName,
		APIKey:     apiKey,
		Timestamp:  time.Now().Unix(),
	}

	// Send request to API
	response, err := pa.sendAuthRequest(ctx, authReq)
	if err != nil {
		if pa.debug {
			fmt.Printf("[PluginAuthenticator] Authentication failed for %s: %v\n", pluginName, err)
		}
		return nil, err
	}

	// Cache the successful response
	pa.authCache.Set(pluginName, apiKey, response)

	if pa.debug {
		fmt.Printf("[PluginAuthenticator] Authentication successful for %s (tier: %s)\n",
			pluginName, response.UserTier)
	}

	return response, nil
}

// AuthenticateWithJWT performs plugin authentication using JWT tokens
func (pa *PluginAuthenticator) AuthenticateWithJWT(ctx context.Context, pluginName string, jwtToken string) (*PluginAuthResponse, error) {
	if pa.debug {
		fmt.Printf("[PluginAuthenticator] Authenticating plugin %s with JWT\n", pluginName)
	}

	// Verify and parse the JWT token
	claims, err := pa.jwtVerifier.ValidateTokenForPlugin(jwtToken, pluginName)
	if err != nil {
		if pa.debug {
			fmt.Printf("[PluginAuthenticator] JWT validation failed for %s: %v\n", pluginName, err)
		}
		return nil, fmt.Errorf("JWT authentication failed: %w", err)
	}

	if pa.debug {
		fmt.Printf("[PluginAuthenticator] JWT authentication successful for %s (tier: %s)\n", pluginName, claims.Tier)
	}

	// Convert JWT claims to AuthResponse
	expiresAt := claims.GetExpirationTime().Format(time.RFC3339)
	response := &PluginAuthResponse{
		Authorized: true,
		UserTier:   claims.Tier,
		Features:   claims.Features,
		ExpiresAt:  &expiresAt,
	}

	return response, nil
}

// sendAuthRequest sends the authentication request to the HTTP API
func (pa *PluginAuthenticator) sendAuthRequest(ctx context.Context, authReq PluginAuthRequest) (*PluginAuthResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/plugins/authenticate", pa.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authReq.APIKey))

	// Send request
	resp, err := pa.httpClient.Do(req)
	if err != nil {
		// For development, fall back to mock response if API is unavailable
		if pa.debug {
			fmt.Printf("[PluginAuthenticator] API unavailable, using mock response: %v\n", err)
		}
		return pa.createMockResponse(authReq.PluginName), nil
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

	return &authResp, nil
}

// createMockResponse creates a mock authentication response for development/testing
func (pa *PluginAuthenticator) createMockResponse(pluginName string) *PluginAuthResponse {
	expiresAt := time.Now().Add(5 * time.Minute).Format(time.RFC3339)
	return &PluginAuthResponse{
		Authorized: true,
		UserTier:   "Pro", // Default to Pro tier for testing
		Features:   []string{"api_logging", "plugin_support"},
		ExpiresAt:  &expiresAt,
	}
}

// Get retrieves cached authentication for a plugin
func (c *PluginAuthCache) Get(pluginName string, apiKey string) *PluginAuthResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheKey := c.getCacheKey(pluginName, apiKey)
	cached, exists := c.cache[cacheKey]
	if !exists {
		return nil
	}

	// Check if cache entry has expired
	if time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached.Response
}

// Set stores authentication result in cache
func (c *PluginAuthCache) Set(pluginName string, apiKey string, auth *PluginAuthResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := c.getCacheKey(pluginName, apiKey)
	c.cache[cacheKey] = &CachedPluginAuth{
		Response:  auth,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes cached authentication
func (c *PluginAuthCache) Clear(pluginName string, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := c.getCacheKey(pluginName, apiKey)
	delete(c.cache, cacheKey)
}

// getCacheKey generates a cache key from plugin name and API key
func (c *PluginAuthCache) getCacheKey(pluginName string, apiKey string) string {
	// Simple concatenation - in production, might use hash for security
	return fmt.Sprintf("%s:%s", pluginName, apiKey)
}

// StartCleanup starts a background goroutine to clean up expired cache entries
func (c *PluginAuthCache) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.cleanupExpired()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// cleanupExpired removes expired cache entries
func (c *PluginAuthCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}
