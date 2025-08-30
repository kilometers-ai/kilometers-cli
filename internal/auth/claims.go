package auth

import (
	"encoding/json"
	"time"
)

// Claims represents JWT claims for plugin authentication tokens
type Claims struct {
	// Standard JWT claims (RFC 7519)
	Sub string `json:"sub"` // Subject (Customer ID)
	Exp int64  `json:"exp"` // Expiration time
	Iat int64  `json:"iat"` // Issued at
	Jti string `json:"jti"` // JWT ID for revocation tracking
	Iss string `json:"iss"` // Issuer
	Aud string `json:"aud"` // Audience

	// Kilometers-specific plugin claims
	CustomerID    string   `json:"customer_id"`    // Customer identifier
	PluginName    string   `json:"plugin_name"`    // Plugin name
	PluginVersion string   `json:"plugin_version"` // Plugin version
	Tier          string   `json:"plan"`           // Subscription tier/plan
	Features      []string `json:"features"`       // Authorized features
	TokenType     string   `json:"token_type"`     // Token type ("plugin")

	// Policy and security claims
	PolicyVersion string `json:"policy_version,omitempty"` // Policy version (future use)
	PolicyHash    string `json:"policy_hash,omitempty"`    // Policy hash (future use)
}

// IsExpired checks if the token has expired
func (c *Claims) IsExpired() bool {
	return time.Now().Unix() > c.Exp
}

// IsValidFor checks if the token is valid for a specific plugin
func (c *Claims) IsValidFor(pluginName string) bool {
	return c.PluginName == pluginName && c.TokenType == "plugin"
}

// GetExpirationTime returns the expiration time as a Time object
func (c *Claims) GetExpirationTime() time.Time {
	return time.Unix(c.Exp, 0)
}

// GetIssuedTime returns the issued time as a Time object
func (c *Claims) GetIssuedTime() time.Time {
	return time.Unix(c.Iat, 0)
}

// HasFeature checks if a specific feature is authorized
func (c *Claims) HasFeature(feature string) bool {
	for _, f := range c.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetRemainingTime returns the remaining time before expiration
func (c *Claims) GetRemainingTime() time.Duration {
	exp := time.Unix(c.Exp, 0)
	now := time.Now()
	if exp.Before(now) {
		return 0
	}
	return exp.Sub(now)
}

// ToJSON converts claims to JSON for debugging
func (c *Claims) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// Validate performs basic claim validation
func (c *Claims) Validate() error {
	if c.CustomerID == "" {
		return ErrInvalidClaims("missing customer_id")
	}

	if c.PluginName == "" {
		return ErrInvalidClaims("missing plugin_name")
	}

	if c.TokenType != "plugin" {
		return ErrInvalidClaims("invalid token_type")
	}

	if c.IsExpired() {
		return ErrTokenExpired
	}

	if c.Exp <= c.Iat {
		return ErrInvalidClaims("expiration time must be after issued time")
	}

	return nil
}

// GetTierLevel returns a numeric tier level for comparison
func (c *Claims) GetTierLevel() int {
	switch c.Tier {
	case "Free":
		return 1
	case "Pro":
		return 2
	case "Enterprise":
		return 3
	default:
		return 0
	}
}

// IsAtLeastTier checks if the user has at least the specified tier
func (c *Claims) IsAtLeastTier(requiredTier string) bool {
	currentLevel := c.GetTierLevel()

	requiredLevel := 0
	switch requiredTier {
	case "Free":
		requiredLevel = 1
	case "Pro":
		requiredLevel = 2
	case "Enterprise":
		requiredLevel = 3
	}

	return currentLevel >= requiredLevel
}
