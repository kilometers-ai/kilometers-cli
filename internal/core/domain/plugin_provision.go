package domain

import (
	"time"
)

// PluginProvisionRequest represents a request to provision plugins for a customer
type PluginProvisionRequest struct {
	CustomerID string
	Platform   string   // e.g., "darwin-arm64", "linux-amd64"
	Plugins    []string // specific plugins or ["all"]
}

// ProvisionedPlugin represents a plugin ready for download
type ProvisionedPlugin struct {
	Name         string
	Version      string
	DownloadURL  string
	Signature    string
	ExpiresAt    time.Time
	RequiredTier string
}

// PluginProvisionResponse contains the provisioned plugins for a customer
type PluginProvisionResponse struct {
	CustomerID       string
	SubscriptionTier string
	Plugins          []ProvisionedPlugin
}

// PluginRegistry tracks installed plugins and their status
type PluginRegistry struct {
	CustomerID  string
	CurrentTier string
	LastUpdated time.Time
	Plugins     map[string]InstalledPlugin
}

// InstalledPlugin represents a locally installed plugin
type InstalledPlugin struct {
	Name         string
	Version      string
	InstalledAt  time.Time
	Path         string
	Signature    string
	RequiredTier string
	Enabled      bool
}

// ShouldRefresh checks if the registry needs refreshing
func (r *PluginRegistry) ShouldRefresh() bool {
	// Refresh if older than 24 hours
	return time.Since(r.LastUpdated) > 24*time.Hour
}

// IsPluginEnabled checks if a plugin should be active based on current tier
func (r *PluginRegistry) IsPluginEnabled(pluginName string, currentTier string) bool {
	plugin, exists := r.Plugins[pluginName]
	if !exists {
		return false
	}

	// Check tier compatibility
	return IsTierSufficient(currentTier, plugin.RequiredTier)
}

// IsTierSufficient checks if the current tier meets the required tier
func IsTierSufficient(currentTier, requiredTier string) bool {
	tiers := map[string]int{
		"Free":       1,
		"Pro":        2,
		"Enterprise": 3,
	}

	current, currentOk := tiers[currentTier]
	required, requiredOk := tiers[requiredTier]

	if !currentOk || !requiredOk {
		return false
	}

	return current >= required
}
