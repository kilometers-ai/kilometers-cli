package application

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
)

// AuthService wraps the JWT plugin authentication flow and exposes a minimal API
// tailored for runtime plugin authentication in dev/test flows.
type AuthService struct {
	apiEndpoint string
	apiKey      string
	cache       auth.TokenCache
}

func NewAuthService(apiEndpoint, apiKey string, cache auth.TokenCache) *AuthService {
	return &AuthService{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
		cache:       cache,
	}
}

// GetPluginToken obtains a token for the given plugin name using the JWT flow.
// When apiKey is empty, the caller should treat it as Free tier and pass empty token.
func (s *AuthService) GetPluginToken(ctx context.Context, pluginName string) (string, error) {
	if s.apiKey == "" {
		return "", nil
	}

	jwtAuth := auth.NewJWTPluginAuthenticator(s.apiEndpoint, s.apiKey, s.cache)
	resp, err := jwtAuth.AuthenticatePlugin(ctx, pluginName)
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}
