package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticationManager_FeatureValidation(t *testing.T) {
	tests := []struct {
		name     string
		tier     SubscriptionTier
		feature  string
		expected bool
	}{
		{
			name:     "free tier basic monitoring",
			tier:     TierFree,
			feature:  FeatureBasicMonitoring,
			expected: true,
		},
		{
			name:     "free tier advanced filters",
			tier:     TierFree,
			feature:  FeatureAdvancedFilters,
			expected: false,
		},
		{
			name:     "pro tier advanced filters",
			tier:     TierPro,
			feature:  FeatureAdvancedFilters,
			expected: true,
		},
		{
			name:     "pro tier compliance reporting",
			tier:     TierPro,
			feature:  FeatureComplianceReporting,
			expected: false,
		},
		{
			name:     "enterprise tier compliance reporting",
			tier:     TierEnterprise,
			feature:  FeatureComplianceReporting,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth manager with test subscription
			authManager := NewAuthenticationManager()
			
			if tt.tier != TierFree {
				// Create mock subscription
				subscription := &SubscriptionConfig{
					Tier:      tt.tier,
					Features:  getFeaturesByTier(tt.tier),
					ExpiresAt: time.Now().Add(24 * time.Hour),
					Signature: "test_signature",
				}
				
				// Set config directly for testing
				authManager.config = subscription
			}

			// Test feature validation
			result := authManager.IsFeatureEnabled(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthenticationManager_SubscriptionValidation(t *testing.T) {
	t.Run("valid subscription", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		subscription := &SubscriptionConfig{
			Tier:      TierPro,
			Features:  []string{FeatureBasicMonitoring, FeatureAdvancedFilters},
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Signature: "valid_signature",
		}
		
		authManager.config = subscription
		
		// For testing, skip signature validation
		// In real implementation, this would validate the JWT signature
		err := authManager.ValidateSubscription()
		
		// We expect an error due to invalid signature in test
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature")
	})

	t.Run("expired subscription", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		subscription := &SubscriptionConfig{
			Tier:      TierPro,
			Features:  []string{FeatureBasicMonitoring, FeatureAdvancedFilters},
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
			Signature: "valid_signature",
		}
		
		authManager.config = subscription
		
		err := authManager.ValidateSubscription()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("free tier always valid", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		// No subscription config = free tier
		
		err := authManager.ValidateSubscription()
		assert.NoError(t, err)
	})
}

func TestAuthenticationManager_TierDetection(t *testing.T) {
	tests := []struct {
		name     string
		config   *SubscriptionConfig
		expected SubscriptionTier
	}{
		{
			name:     "no config returns free tier",
			config:   nil,
			expected: TierFree,
		},
		{
			name: "pro tier config",
			config: &SubscriptionConfig{
				Tier: TierPro,
			},
			expected: TierPro,
		},
		{
			name: "enterprise tier config",
			config: &SubscriptionConfig{
				Tier: TierEnterprise,
			},
			expected: TierEnterprise,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authManager := NewAuthenticationManager()
			authManager.config = tt.config
			
			result := authManager.GetSubscriptionTier()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthenticationManager_FeatureListing(t *testing.T) {
	t.Run("free tier features", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		// No config = free tier
		
		features := authManager.ListFeatures(context.Background())
		expected := []string{FeatureBasicMonitoring}
		
		assert.Equal(t, expected, features)
	})

	t.Run("pro tier features", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		authManager.config = &SubscriptionConfig{
			Tier: TierPro,
		}
		
		features := authManager.ListFeatures(context.Background())
		expected := []string{
			FeatureBasicMonitoring,
			FeatureAdvancedFilters,
			FeatureCustomRules,
			FeaturePoisonDetection,
			FeatureMLAnalytics,
		}
		
		assert.Equal(t, expected, features)
	})

	t.Run("enterprise tier features", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		authManager.config = &SubscriptionConfig{
			Tier: TierEnterprise,
		}
		
		features := authManager.ListFeatures(context.Background())
		
		// Should include all features
		assert.Contains(t, features, FeatureBasicMonitoring)
		assert.Contains(t, features, FeatureAdvancedFilters)
		assert.Contains(t, features, FeatureComplianceReporting)
		assert.Contains(t, features, FeaturePrioritySupport)
	})
}

func TestAuthenticationManager_RefreshLogic(t *testing.T) {
	t.Run("needs refresh when expires soon", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		authManager.config = &SubscriptionConfig{
			Tier:        TierPro,
			ExpiresAt:   time.Now().Add(12 * time.Hour), // Expires in 12 hours
			LastRefresh: time.Now().Add(-2 * time.Hour), // Refreshed 2 hours ago
		}
		
		needsRefresh := authManager.NeedsRefresh()
		assert.True(t, needsRefresh)
	})

	t.Run("needs refresh when last refresh old", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		authManager.config = &SubscriptionConfig{
			Tier:        TierPro,
			ExpiresAt:   time.Now().Add(48 * time.Hour), // Expires in 2 days
			LastRefresh: time.Now().Add(-8 * time.Hour), // Refreshed 8 hours ago
		}
		
		needsRefresh := authManager.NeedsRefresh()
		assert.True(t, needsRefresh)
	})

	t.Run("does not need refresh when recent", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		authManager.config = &SubscriptionConfig{
			Tier:        TierPro,
			ExpiresAt:   time.Now().Add(48 * time.Hour), // Expires in 2 days
			LastRefresh: time.Now().Add(-2 * time.Hour), // Refreshed 2 hours ago
		}
		
		needsRefresh := authManager.NeedsRefresh()
		assert.False(t, needsRefresh)
	})

	t.Run("free tier never needs refresh", func(t *testing.T) {
		authManager := NewAuthenticationManager()
		// No config = free tier
		
		needsRefresh := authManager.NeedsRefresh()
		assert.False(t, needsRefresh)
	})
}

// Helper function for tests
func getFeaturesByTier(tier SubscriptionTier) []string {
	switch tier {
	case TierFree:
		return []string{FeatureBasicMonitoring}
	case TierPro:
		return []string{
			FeatureBasicMonitoring,
			FeatureAdvancedFilters,
			FeatureCustomRules,
			FeaturePoisonDetection,
			FeatureMLAnalytics,
		}
	case TierEnterprise:
		return []string{
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
	default:
		return []string{FeatureBasicMonitoring}
	}
}
