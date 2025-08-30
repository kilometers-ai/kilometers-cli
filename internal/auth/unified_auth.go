package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// UnifiedAuthenticator provides both token-based and JWT-based authentication
type UnifiedAuthenticator struct {
	tokenProvider TokenProvider
	tokenCache    TokenCache
	jwtVerifier   *JWTVerifier
	keyRing       *KeyRing
	debug         bool
}

// NewUnifiedAuthenticator creates a comprehensive authentication system
func NewUnifiedAuthenticator(tokenProvider TokenProvider, tokenCache TokenCache, debug bool) *UnifiedAuthenticator {
	keyRing := NewKeyRing()
	return &UnifiedAuthenticator{
		tokenProvider: tokenProvider,
		tokenCache:    tokenCache,
		jwtVerifier:   NewJWTVerifier(keyRing),
		keyRing:       keyRing,
		debug:         debug,
	}
}

// AuthenticateWithToken performs token-based authentication
func (ua *UnifiedAuthenticator) AuthenticateWithToken(ctx context.Context, scope string) (*AuthToken, error) {
	// Check cache first
	if cached, err := ua.tokenCache.GetToken(scope); err == nil && cached != nil {
		if !ua.isTokenExpired(cached) {
			return cached, nil
		}
	}

	// Get new token
	request := &TokenRequest{
		Scope:     []string{scope},
		APIKey:    "", // This would be filled by the implementation
		GrantType: "api_key",
	}

	token, err := ua.tokenProvider.GetToken(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("token authentication failed: %w", err)
	}

	// Cache the token
	if err := ua.tokenCache.SetToken(scope, token); err != nil && ua.debug {
		fmt.Printf("[UnifiedAuth] Warning: failed to cache token: %v\n", err)
	}

	return token, nil
}

// AuthenticateWithJWT performs JWT-based authentication for plugins
func (ua *UnifiedAuthenticator) AuthenticateWithJWT(ctx context.Context, pluginName string, jwtToken string) (*PluginAuthResponse, error) {
	if ua.debug {
		fmt.Printf("[UnifiedAuth] Authenticating plugin %s with JWT\n", pluginName)
	}

	// Verify and parse the JWT token
	claims, err := ua.jwtVerifier.ValidateTokenForPlugin(jwtToken, pluginName)
	if err != nil {
		if ua.debug {
			fmt.Printf("[UnifiedAuth] JWT validation failed for %s: %v\n", pluginName, err)
		}
		return nil, fmt.Errorf("JWT authentication failed: %w", err)
	}

	if ua.debug {
		fmt.Printf("[UnifiedAuth] JWT authentication successful for %s (tier: %s)\n", pluginName, claims.Tier)
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

// ValidateSignature verifies a digital signature using the keyring
func (ua *UnifiedAuthenticator) ValidateSignature(data []byte, signature []byte, keyID string) error {
	return ua.keyRing.VerifySignature(data, signature, keyID)
}

// ClearCache clears all cached authentication data
func (ua *UnifiedAuthenticator) ClearCache() error {
	return ua.tokenCache.Clear()
}

// isTokenExpired checks if a token is expired
func (ua *UnifiedAuthenticator) isTokenExpired(token *AuthToken) bool {
	if token.ExpiresAt.IsZero() {
		return false // No expiration time
	}
	return time.Now().After(token.ExpiresAt.Add(-5 * time.Minute)) // Refresh 5 minutes before expiry
}

// PluginAuthResponse represents the unified plugin authentication response
type PluginAuthResponse struct {
	Authorized bool     `json:"authorized"`
	UserTier   string   `json:"user_tier"`
	Features   []string `json:"features"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// KeyRing manages Ed25519 public keys for signature verification
type KeyRing struct {
	mu      sync.RWMutex
	keys    map[string]ed25519.PublicKey
	primary string // Current primary key ID
}

// NewKeyRing creates a new KeyRing with embedded public keys
func NewKeyRing() *KeyRing {
	kr := &KeyRing{
		keys:    make(map[string]ed25519.PublicKey),
		primary: PrimaryKeyID,
	}

	// Load embedded keys
	if err := kr.loadEmbeddedKeys(); err != nil {
		// In production, this would be a fatal error
		// For now, create empty keyring
		return kr
	}

	return kr
}

// VerifySignature verifies an Ed25519 signature using the specified key ID
func (kr *KeyRing) VerifySignature(data []byte, signature []byte, keyID string) error {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	publicKey, exists := kr.keys[keyID]
	if !exists {
		return fmt.Errorf("unknown key ID: %s", keyID)
	}

	// Verify Ed25519 signature
	if !ed25519.Verify(publicKey, data, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// AddKey adds a new public key to the keyring
func (kr *KeyRing) AddKey(keyID string, publicKeyData []byte) error {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	// Decode base64 if needed
	var publicKey ed25519.PublicKey
	if len(publicKeyData) == ed25519.PublicKeySize {
		publicKey = publicKeyData
	} else {
		decoded, err := base64.StdEncoding.DecodeString(string(publicKeyData))
		if err != nil {
			return fmt.Errorf("failed to decode public key: %w", err)
		}
		if len(decoded) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(decoded))
		}
		publicKey = decoded
	}

	kr.keys[keyID] = publicKey
	return nil
}

// GetPrimaryKeyID returns the current primary key ID
func (kr *KeyRing) GetPrimaryKeyID() string {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.primary
}

// loadEmbeddedKeys loads the embedded public keys
func (kr *KeyRing) loadEmbeddedKeys() error {
	// Load from embedded keys (this would import from the existing keys_embedded.go)
	for keyID, keyData := range EmbeddedPublicKeys {
		if err := kr.AddKey(keyID, []byte(keyData)); err != nil {
			return fmt.Errorf("failed to load embedded key %s: %w", keyID, err)
		}
	}
	return nil
}
