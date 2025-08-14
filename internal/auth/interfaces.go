package auth

import (
	"context"
	"time"
)

// TokenProvider handles token acquisition and refresh
type TokenProvider interface {
	// GetToken obtains a new token using the provided request
	GetToken(ctx context.Context, request *TokenRequest) (*AuthToken, error)

	// RefreshToken refreshes an existing token
	RefreshToken(ctx context.Context, refreshToken string) (*AuthToken, error)

	// ValidateToken checks if a token is still valid
	ValidateToken(ctx context.Context, token string) (bool, error)
}

// TokenCache handles token storage and retrieval
type TokenCache interface {
	// GetToken retrieves a cached token for the given scope
	GetToken(scope string) (*AuthToken, error)

	// SetToken stores a token for the given scope
	SetToken(scope string, token *AuthToken) error

	// RemoveToken removes a token for the given scope
	RemoveToken(scope string) error

	// Clear removes all cached tokens
	Clear() error
}

// AuthManager orchestrates authentication operations
type AuthManager interface {
	// GetValidToken returns a valid token, refreshing if necessary
	GetValidToken(ctx context.Context, scope string) (*AuthToken, error)

	// ForceRefresh forces a token refresh regardless of expiration
	ForceRefresh(ctx context.Context, scope string) (*AuthToken, error)

	// ClearCache clears all cached authentication data
	ClearCache() error
}

// TokenRefreshStrategy defines when tokens should be refreshed
type TokenRefreshStrategy interface {
	// ShouldRefresh determines if a token should be refreshed
	ShouldRefresh(token *AuthToken) bool

	// GetRefreshInterval returns how often to check for refresh
	GetRefreshInterval() time.Duration
}
