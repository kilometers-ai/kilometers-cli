package plugins

import (
	"context"
	"fmt"
	"os"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// PluginJWTAuthenticator handles JWT authentication for plugins
// This provides a single responsibility for authenticating plugins with the Kilometers API
type PluginJWTAuthenticator struct {
	apiEndpoint string
	apiKey      string
	tokenCache  auth.TokenCache
	debug       bool
}

// NewPluginJWTAuthenticator creates a new authenticator with the given configuration
func NewPluginJWTAuthenticator(apiKey, apiEndpoint string, debug bool) *PluginJWTAuthenticator {
	// Determine API endpoint - use provided or default
	if apiEndpoint == "" {
		apiEndpoint = "https://api.kilometers.ai"
		if envEndpoint := os.Getenv("KM_API_ENDPOINT"); envEndpoint != "" {
			apiEndpoint = envEndpoint
		}
	}

	return &PluginJWTAuthenticator{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
		tokenCache:  auth.NewMemoryTokenCache(),
		debug:       debug,
	}
}

// NewPluginJWTAuthenticatorWithCache creates a new authenticator with a custom token cache
func NewPluginJWTAuthenticatorWithCache(apiKey, apiEndpoint string, tokenCache auth.TokenCache, debug bool) *PluginJWTAuthenticator {
	// Determine API endpoint - use provided or default
	if apiEndpoint == "" {
		apiEndpoint = "https://api.kilometers.ai"
		if envEndpoint := os.Getenv("KM_API_ENDPOINT"); envEndpoint != "" {
			apiEndpoint = envEndpoint
		}
	}

	return &PluginJWTAuthenticator{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
		tokenCache:  tokenCache,
		debug:       debug,
	}
}

// AuthenticatePlugin authenticates a plugin using JWT flow
// This is the core authentication logic that can be reused across different plugin loading scenarios
func (pa *PluginJWTAuthenticator) AuthenticatePlugin(ctx context.Context, plugin kmsdk.KilometersPlugin, pluginName string) error {
	if pa.apiKey == "" {
		if pa.debug {
			fmt.Printf("🔑 No API key provided - plugin should handle Free tier authentication\n")
		}
		// Call plugin's Authenticate method with empty token for Free tier
		err := plugin.Authenticate(ctx, "")
		if err != nil {
			return fmt.Errorf("plugin authentication failed: %w", err)
		}

		if pa.debug {
			fmt.Printf("✅ Plugin authenticated (Free tier)\n")
		}
		return nil
	}

	// Use JWT authentication flow when API key is provided
	if pa.debug {
		fmt.Printf("🔑 Authenticating plugin via JWT flow: %s...\n", pa.apiKey[:minInt(len(pa.apiKey), 10)])
	}

	// Create JWT authenticator
	jwtAuth := auth.NewJWTPluginAuthenticator(pa.apiEndpoint, pa.apiKey, pa.tokenCache)

	// Get plugin-specific JWT token
	authResp, err := jwtAuth.AuthenticatePlugin(ctx, pluginName)
	if err != nil {
		return fmt.Errorf("failed to get JWT token for plugin: %w", err)
	}

	if !authResp.Authorized {
		return fmt.Errorf("plugin not authorized")
	}

	if pa.debug {
		fmt.Printf("🔑 Received plugin JWT token for tier: %s\n", authResp.UserTier)
	}

	// Call plugin's Authenticate method with JWT token
	err = plugin.Authenticate(ctx, authResp.Token)
	if err != nil {
		return fmt.Errorf("plugin JWT authentication failed: %w", err)
	}

	if pa.debug {
		fmt.Printf("✅ Plugin authenticated successfully via JWT\n")
	}

	return nil
}

// GetAPIEndpoint returns the configured API endpoint
func (pa *PluginJWTAuthenticator) GetAPIEndpoint() string {
	return pa.apiEndpoint
}

// GetAPIKey returns the configured API key
func (pa *PluginJWTAuthenticator) GetAPIKey() string {
	return pa.apiKey
}

// SetDebug updates the debug mode
func (pa *PluginJWTAuthenticator) SetDebug(debug bool) {
	pa.debug = debug
}

// Helper function for min (Go < 1.21 compatibility)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
