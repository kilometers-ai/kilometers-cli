package auth

import "fmt"

// JWT verification errors
var (
	ErrInvalidTokenFormat = fmt.Errorf("invalid JWT token format")
	ErrTokenExpired       = fmt.Errorf("token has expired")
	ErrMissingKeyID       = fmt.Errorf("missing key ID in token header")
)

// ErrInvalidClaims creates an error for invalid claims
func ErrInvalidClaims(reason string) error {
	return fmt.Errorf("invalid claims: %s", reason)
}

// ErrUnsupportedAlgorithm creates an error for unsupported algorithms
func ErrUnsupportedAlgorithm(alg string) error {
	return fmt.Errorf("unsupported algorithm: %s", alg)
}

// ErrInvalidTokenType creates an error for invalid token types
func ErrInvalidTokenType(typ string) error {
	return fmt.Errorf("invalid token type: %s", typ)
}

// ErrTokenNotValidForPlugin creates an error when token is not valid for a plugin
func ErrTokenNotValidForPlugin(expected, actual string) error {
	return fmt.Errorf("token not valid for plugin %s (token is for %s)", expected, actual)
}
