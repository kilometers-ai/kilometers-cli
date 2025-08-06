package domain

import (
	"time"
)

// AuthToken represents an authentication token with expiration
type AuthToken struct {
	Token        string
	Type         string // e.g., "Bearer"
	ExpiresAt    time.Time
	IssuedAt     time.Time
	Scope        []string
	RefreshToken string // Optional refresh token
}

// IsExpired checks if the token has expired
func (t *AuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// ShouldRefresh checks if the token should be refreshed
// Returns true if the token will expire within the given duration
func (t *AuthToken) ShouldRefresh(refreshAhead time.Duration) bool {
	if t.IsExpired() {
		return true
	}

	refreshTime := t.ExpiresAt.Add(-refreshAhead)
	return time.Now().After(refreshTime)
}

// TimeUntilExpiry returns the duration until the token expires
func (t *AuthToken) TimeUntilExpiry() time.Duration {
	return time.Until(t.ExpiresAt)
}

// TokenRequest represents a request for a new token
type TokenRequest struct {
	APIKey       string
	Scope        []string
	GrantType    string // e.g., "api_key", "refresh_token"
	RefreshToken string // Used for refresh_token grant type
}

// TokenResponse represents the response from a token request
type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"` // Seconds until expiration
	RefreshToken string   `json:"refresh_token,omitempty"`
	Scope        []string `json:"scope,omitempty"`
}

// ToAuthToken converts the response to an AuthToken
func (r *TokenResponse) ToAuthToken() *AuthToken {
	now := time.Now()
	return &AuthToken{
		Token:        r.AccessToken,
		Type:         r.TokenType,
		ExpiresAt:    now.Add(time.Duration(r.ExpiresIn) * time.Second),
		IssuedAt:     now,
		Scope:        r.Scope,
		RefreshToken: r.RefreshToken,
	}
}

// AuthCache represents cached authentication state
type AuthCache struct {
	Tokens      map[string]*AuthToken // Key is scope or purpose
	APIKey      string
	LastRefresh time.Time
}

// GetToken retrieves a token for the given scope
func (c *AuthCache) GetToken(scope string) *AuthToken {
	if c.Tokens == nil {
		return nil
	}
	return c.Tokens[scope]
}

// SetToken stores a token for the given scope
func (c *AuthCache) SetToken(scope string, token *AuthToken) {
	if c.Tokens == nil {
		c.Tokens = make(map[string]*AuthToken)
	}
	c.Tokens[scope] = token
	c.LastRefresh = time.Now()
}

// ClearExpiredTokens removes all expired tokens from the cache
func (c *AuthCache) ClearExpiredTokens() {
	for scope, token := range c.Tokens {
		if token.IsExpired() {
			delete(c.Tokens, scope)
		}
	}
}
