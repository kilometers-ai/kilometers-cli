package plugindomain

import "time"

// SubscriptionTier represents user subscription levels
type SubscriptionTier string

const (
	TierFree       SubscriptionTier = "free"
	TierPro        SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// Plugin represents a plugin with its metadata
type Plugin struct {
	Name         string           `json:"name"`
	Version      string           `json:"version"`
	Description  string           `json:"description"`
	RequiredTier SubscriptionTier `json:"required_tier"`
	Size         int64            `json:"size"`
	Checksum     string           `json:"checksum"`
	DownloadURL  string           `json:"download_url"`
	Signature    string           `json:"signature"`
}

// Subscription represents a user's subscription status
type Subscription struct {
	Tier              SubscriptionTier `json:"tier"`
	CustomerID        string           `json:"customer_id"`
	CustomerName      string           `json:"customer_name"`
	ExpiresAt         *time.Time       `json:"expires_at,omitempty"`
	AvailableFeatures []string         `json:"available_features"`
}

// PluginInstallStatus represents installation state
type PluginInstallStatus struct {
	Plugin         Plugin
	IsInstalled    bool
	LocalPath      string
	NeedsUpdate    bool
	CurrentVersion string
}

// ProvisioningResult represents the result of plugin provisioning
type ProvisioningResult struct {
	Subscription     Subscription
	AvailablePlugins []Plugin
	InstalledCount   int
	FailedCount      int
	Errors           []error
}

// CanAccessPlugin checks if a subscription tier can access a plugin
func (s Subscription) CanAccessPlugin(plugin Plugin) bool {
	switch plugin.RequiredTier {
	case TierFree:
		return true
	case TierPro:
		return s.Tier == TierPro || s.Tier == TierEnterprise
	case TierEnterprise:
		return s.Tier == TierEnterprise
	default:
		return false
	}
}

// ValidationResult represents API key validation outcome
type ValidationResult struct {
	IsValid      bool
	Subscription *Subscription
	Message      string
}
