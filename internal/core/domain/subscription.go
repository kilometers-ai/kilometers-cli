package domain

import "time"

// SubscriptionTier represents user subscription levels
type SubscriptionTier string

const (
	TierFree       SubscriptionTier = "free"
	TierPro        SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// Feature flags
const (
	FeatureBasicMonitoring     = "basic_monitoring"
	FeatureConsoleLogging      = "console_logging"
	FeatureAPILogging          = "api_logging"
	FeatureAdvancedFilters     = "advanced_filters"
	FeaturePoisonDetection     = "poison_detection"
	FeatureMLAnalytics         = "ml_analytics"
	FeatureComplianceReporting = "compliance_reporting"
	FeatureTeamCollaboration   = "team_collaboration"
)

// SubscriptionInfo contains user subscription details
type SubscriptionInfo struct {
	Tier          SubscriptionTier `json:"tier"`
	Features      []string         `json:"features"`
	ExpiresAt     *time.Time       `json:"expires_at,omitempty"`
	LastRefreshed time.Time        `json:"-"`
}

// IsExpired checks if the subscription has expired
func (s *SubscriptionInfo) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// HasFeature checks if a feature is enabled
func (s *SubscriptionInfo) HasFeature(feature string) bool {
	for _, f := range s.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// DefaultFreeFeatures returns the features available in free tier
func DefaultFreeFeatures() []string {
	return []string{
		FeatureBasicMonitoring,
		FeatureConsoleLogging,
	}
}
