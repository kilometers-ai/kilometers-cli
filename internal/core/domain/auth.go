package domain

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SubscriptionTier represents different subscription levels
type SubscriptionTier string

const (
	TierFree       SubscriptionTier = "free"
	TierPro        SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// SubscriptionConfig holds subscription and feature information
type SubscriptionConfig struct {
	Tier        SubscriptionTier `json:"tier"`
	Features    []string         `json:"features"`
	ExpiresAt   time.Time        `json:"expires_at"`
	RefreshURL  string           `json:"refresh_url,omitempty"`
	Signature   string           `json:"signature"`
	LastRefresh time.Time        `json:"last_refresh"`
}

// AuthenticationManager handles subscription validation and feature management
type AuthenticationManager struct {
	config    *SubscriptionConfig
	publicKey *rsa.PublicKey
}

// NewAuthenticationManager creates a new authentication manager
func NewAuthenticationManager() *AuthenticationManager {
	return &AuthenticationManager{
		publicKey: getEmbeddedPublicKey(), // Embedded in binary
	}
}

// IsFeatureEnabled checks if a feature is available for current subscription
func (am *AuthenticationManager) IsFeatureEnabled(feature string) bool {
	if am.config == nil {
		// Free tier by default
		return isFreeFeature(feature)
	}

	// Check if feature is in enabled features list
	for _, enabledFeature := range am.config.Features {
		if enabledFeature == feature {
			return true
		}
	}

	// Check tier-based features
	return am.isTierFeatureEnabled(feature)
}

// GetSubscriptionTier returns the current subscription tier
func (am *AuthenticationManager) GetSubscriptionTier() SubscriptionTier {
	if am.config == nil {
		return TierFree
	}
	return am.config.Tier
}

// ValidateSubscription validates the current subscription config
func (am *AuthenticationManager) ValidateSubscription() error {
	if am.config == nil {
		return nil // Free tier is always valid
	}

	// Check expiration
	if time.Now().After(am.config.ExpiresAt) {
		return fmt.Errorf("subscription expired")
	}

	// Validate signature
	return am.validateSignature()
}

// NeedsRefresh checks if subscription config needs refreshing
func (am *AuthenticationManager) NeedsRefresh() bool {
	if am.config == nil {
		return false
	}

	// Refresh if expires within 24 hours or last refresh was > 6 hours ago
	expiresWithin24h := time.Until(am.config.ExpiresAt) < 24*time.Hour
	lastRefreshOld := time.Since(am.config.LastRefresh) > 6*time.Hour

	return expiresWithin24h || lastRefreshOld
}

// LoadSubscription loads subscription config from storage
func (am *AuthenticationManager) LoadSubscription() error {
	config, err := loadSubscriptionConfig()
	if err != nil {
		// No subscription file = free tier
		return nil
	}

	am.config = config
	return am.ValidateSubscription()
}

// ListFeatures returns available features for current subscription
func (am *AuthenticationManager) ListFeatures(ctx context.Context) []string {
	tier := am.GetSubscriptionTier()
	
	var features []string
	switch tier {
	case TierFree:
		features = []string{
			FeatureBasicMonitoring,
		}
	case TierPro:
		features = []string{
			FeatureBasicMonitoring,
			FeatureAdvancedFilters,
			FeatureCustomRules,
			FeaturePoisonDetection,
			FeatureMLAnalytics,
		}
	case TierEnterprise:
		features = []string{
			FeatureBasicMonitoring,
			FeatureAdvancedFilters,
			FeatureCustomRules,
			FeaturePoisonDetection,
			FeatureMLAnalytics,
			FeatureTeamCollaboration,
			FeatureCustomDashboards,
			FeatureComplianceReporting,
			FeatureAPIIntegrations,
			FeaturePrioritySupport,
		}
	}
	
	return features
}

// SaveSubscription saves subscription config to storage
func (am *AuthenticationManager) SaveSubscription(config *SubscriptionConfig) error {
	// Validate before saving
	tempManager := &AuthenticationManager{
		config:    config,
		publicKey: am.publicKey,
	}
	
	if err := tempManager.validateSignature(); err != nil {
		return fmt.Errorf("invalid subscription signature: %w", err)
	}

	am.config = config
	return saveSubscriptionConfig(config)
}

// Feature definitions
const (
	FeatureBasicMonitoring     = "basic_monitoring"
	FeatureAdvancedFilters     = "advanced_filters"
	FeatureCustomRules         = "custom_rules"
	FeaturePoisonDetection     = "poison_detection"
	FeatureMLAnalytics         = "ml_analytics"
	FeatureTeamCollaboration   = "team_collaboration"
	FeatureCustomDashboards    = "custom_dashboards"
	FeatureComplianceReporting = "compliance_reporting"
	FeatureAPIIntegrations     = "api_integrations"
	FeaturePrioritySupport     = "priority_support"
)

// isFreeFeature returns true if feature is available in free tier
func isFreeFeature(feature string) bool {
	freeFeatures := []string{
		FeatureBasicMonitoring,
	}

	for _, freeFeature := range freeFeatures {
		if freeFeature == feature {
			return true
		}
	}
	return false
}

// isTierFeatureEnabled checks tier-based feature access
func (am *AuthenticationManager) isTierFeatureEnabled(feature string) bool {
	switch am.config.Tier {
	case TierEnterprise:
		return true // Enterprise gets everything
	case TierPro:
		return isProFeature(feature)
	case TierFree:
		return isFreeFeature(feature)
	default:
		return false
	}
}

// isProFeature returns true if feature is available in pro tier
func isProFeature(feature string) bool {
	proFeatures := []string{
		FeatureBasicMonitoring,
		FeatureAdvancedFilters,
		FeatureCustomRules,
		FeaturePoisonDetection,
		FeatureMLAnalytics,
	}

	for _, proFeature := range proFeatures {
		if proFeature == feature {
			return true
		}
	}
	return false
}

// validateSignature validates the subscription signature
func (am *AuthenticationManager) validateSignature() error {
	if am.publicKey == nil {
		return fmt.Errorf("no public key available")
	}

	// Create JWT token from subscription data
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"tier":     am.config.Tier,
		"features": am.config.Features,
		"exp":      am.config.ExpiresAt.Unix(),
	})

	// Parse and validate the signature
	parsedToken, err := jwt.Parse(am.config.Signature, func(token *jwt.Token) (interface{}, error) {
		return am.publicKey, nil
	})

	if err != nil || !parsedToken.Valid {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// getEmbeddedPublicKey returns the embedded public key for signature validation
func getEmbeddedPublicKey() *rsa.PublicKey {
	// This would be embedded in the binary at build time
	publicKeyPEM := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0vx7agoebGcQSuuPiLJX
ZptN9nndrQmbPFRP6gPiw+7VqhJOOZRiGpd34jyQD+q4vPCHBhzJHJw7gGGNIQBr
LIq5e7M7E1dZrD8vFz4k9w6WtZOQ+7zJQZpK8W5ZQq8eFjXjQsGm2Y4W9XpGgAMb
+kE4sW3Q9k5yY+ZbF2Q8W5YoQx8+WqjJcA8fE9rWQ9k5Q+YhF2QpW5jX9rGm2YhJ
cA8fE9rWQ9k5Q+YhF2QpW5jX9rGm2YhJcA8fE9rWQ9k5Q+YhF2QpW5jX9rGm2YhJ
cA8fE9rWQ9k5Q+YhF2QpW5jX9rGm2YhJcA8fE9rWQ9k5Q+YhF2QpW5jX9rGm2YhJ
cQIDAQAB
-----END PUBLIC KEY-----`

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil
	}

	return rsaPubKey
}
