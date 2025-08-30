package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// JWTVerifier handles verification of EdDSA-signed JWT tokens
type JWTVerifier struct {
	keyRing *KeyRing
}

// NewJWTVerifier creates a new JWT verifier with the provided keyring
func NewJWTVerifier(keyRing *KeyRing) *JWTVerifier {
	return &JWTVerifier{
		keyRing: keyRing,
	}
}

// VerifyToken verifies a JWT token and returns parsed claims
func (v *JWTVerifier) VerifyToken(tokenString string) (*Claims, error) {
	// Parse the token structure first
	claims, keyID, err := v.ParseClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate the signature
	if err := v.verifySignature(tokenString, keyID); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Validate claims
	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return claims, nil
}

// ParseClaims extracts claims from a JWT token without signature verification
func (v *JWTVerifier) ParseClaims(tokenString string) (*Claims, string, error) {
	// Split JWT into parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, "", ErrInvalidTokenFormat
	}

	// Parse header to get key ID
	header, err := v.parseHeader(parts[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse header: %w", err)
	}

	// Parse payload to get claims
	claims, err := v.parsePayload(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse payload: %w", err)
	}

	return claims, header.Kid, nil
}

// verifySignature verifies the JWT signature using Ed25519
func (v *JWTVerifier) verifySignature(tokenString, keyID string) error {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidTokenFormat
	}

	// Create signing input (header.payload)
	signingInput := parts[0] + "." + parts[1]
	signingBytes := []byte(signingInput)

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature using keyring
	if err := v.keyRing.VerifySignature(signingBytes, signature, keyID); err != nil {
		return fmt.Errorf("keyring verification failed: %w", err)
	}

	return nil
}

// parseHeader parses the JWT header
func (v *JWTVerifier) parseHeader(headerB64 string) (*JWTHeader, error) {
	headerBytes, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header JWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Validate header
	if header.Alg != "EdDSA" {
		return nil, ErrUnsupportedAlgorithm(header.Alg)
	}

	if header.Typ != "JWT" {
		return nil, ErrInvalidTokenType(header.Typ)
	}

	if header.Kid == "" {
		return nil, ErrMissingKeyID
	}

	return &header, nil
}

// parsePayload parses the JWT payload into Claims
func (v *JWTVerifier) parsePayload(payloadB64 string) (*Claims, error) {
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return &claims, nil
}

// ValidateTokenForPlugin validates a token specifically for a given plugin
func (v *JWTVerifier) ValidateTokenForPlugin(tokenString, pluginName string) (*Claims, error) {
	claims, err := v.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	if !claims.IsValidFor(pluginName) {
		return nil, ErrTokenNotValidForPlugin(pluginName, claims.PluginName)
	}

	return claims, nil
}

// JWTHeader represents the JWT header structure
type JWTHeader struct {
	Alg string `json:"alg"` // Algorithm (should be "EdDSA")
	Typ string `json:"typ"` // Type (should be "JWT")
	Kid string `json:"kid"` // Key ID
}
