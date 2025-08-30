package plugins

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
)

// HTTPPluginAuthenticator is a wrapper around the unified authenticator for backward compatibility
type HTTPPluginAuthenticator struct {
	authenticator *auth.PluginAuthenticator
}

// NewHTTPPluginAuthenticator creates a new HTTP-based plugin authenticator
func NewHTTPPluginAuthenticator(apiEndpoint string, debug bool) *HTTPPluginAuthenticator {
	return &HTTPPluginAuthenticator{
		authenticator: auth.NewPluginAuthenticator(apiEndpoint, debug),
	}
}

// AuthenticatePlugin sends authentication request to kilometers-api
func (a *HTTPPluginAuthenticator) AuthenticatePlugin(ctx context.Context, pluginName string, apiKey string) (*auth.PluginAuthResponse, error) {
	response, err := a.authenticator.AuthenticatePlugin(ctx, pluginName, apiKey)
	if err != nil {
		return nil, err
	}

	// Convert to the expected AuthResponse type used by plugins
	return &auth.PluginAuthResponse{
		Authorized: response.Authorized,
		UserTier:   response.UserTier,
		Features:   response.Features,
		ExpiresAt:  response.ExpiresAt,
	}, nil
}

// InMemoryAuthCache is a wrapper around the unified auth cache for backward compatibility
type InMemoryAuthCache struct {
	cache *auth.PluginAuthCache
}

// NewInMemoryAuthCache creates a new in-memory authentication cache
func NewInMemoryAuthCache(ttl interface{}) *InMemoryAuthCache {
	// Use default 5-minute TTL from the unified auth
	return &InMemoryAuthCache{
		cache: auth.NewPluginAuthCache(5 * 60 * 1000000000), // 5 minutes in nanoseconds
	}
}

// Get retrieves cached authentication for a plugin
func (c *InMemoryAuthCache) Get(pluginName string, apiKey string) *auth.PluginAuthResponse {
	response := c.cache.Get(pluginName, apiKey)
	if response == nil {
		return nil
	}

	// Convert to the expected AuthResponse type used by plugins
	return &auth.PluginAuthResponse{
		Authorized: response.Authorized,
		UserTier:   response.UserTier,
		Features:   response.Features,
		ExpiresAt:  response.ExpiresAt,
	}
}

// Set stores authentication result in cache
func (c *InMemoryAuthCache) Set(pluginName string, apiKey string, authResponse *auth.PluginAuthResponse) {
	// Convert to the unified auth response type
	response := &auth.PluginAuthResponse{
		Authorized: authResponse.Authorized,
		UserTier:   authResponse.UserTier,
		Features:   authResponse.Features,
		ExpiresAt:  authResponse.ExpiresAt,
	}
	c.cache.Set(pluginName, apiKey, response)
}

// Clear removes cached authentication
func (c *InMemoryAuthCache) Clear(pluginName string, apiKey string) {
	c.cache.Clear(pluginName, apiKey)
}
