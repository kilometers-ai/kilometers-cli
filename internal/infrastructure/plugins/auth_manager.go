package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// AuthenticationManagerImpl implements the AuthenticationManager interface
type AuthenticationManagerImpl struct {
	config       domain.Config
	subscription domain.SubscriptionInfo
	mutex        sync.RWMutex
	apiClient    ports.APIClient
}

// NewAuthenticationManager creates a new authentication manager
func NewAuthenticationManager(config domain.Config, apiClient ports.APIClient) *AuthenticationManagerImpl {
	am := &AuthenticationManagerImpl{
		config:    config,
		apiClient: apiClient,
		subscription: domain.SubscriptionInfo{
			Tier:     domain.TierFree,
			Features: domain.DefaultFreeFeatures(),
		},
	}

	// Load cached subscription info
	am.loadCachedSubscription()

	// Refresh from API if we have a key
	if config.ApiKey != "" {
		go am.RefreshSubscription(context.Background())
	}

	return am
}

// GetAPIKey returns the configured API key
func (am *AuthenticationManagerImpl) GetAPIKey() string {
	return am.config.ApiKey
}

// GetSubscriptionTier returns the current subscription tier
func (am *AuthenticationManagerImpl) GetSubscriptionTier() domain.SubscriptionTier {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.subscription.Tier
}

// GetEnabledFeatures returns the list of enabled features
func (am *AuthenticationManagerImpl) GetEnabledFeatures() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.subscription.Features
}

// IsFeatureEnabled checks if a specific feature is enabled
func (am *AuthenticationManagerImpl) IsFeatureEnabled(feature string) bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Check if subscription is expired
	if am.subscription.IsExpired() {
		// Only allow free features
		for _, f := range domain.DefaultFreeFeatures() {
			if f == feature {
				return true
			}
		}
		return false
	}

	return am.subscription.HasFeature(feature)
}

// RefreshSubscription refreshes subscription info from the API
func (am *AuthenticationManagerImpl) RefreshSubscription(ctx context.Context) error {
	if am.config.ApiKey == "" {
		// No API key, use free tier
		am.mutex.Lock()
		am.subscription = domain.SubscriptionInfo{
			Tier:          domain.TierFree,
			Features:      domain.DefaultFreeFeatures(),
			LastRefreshed: time.Now(),
		}
		am.mutex.Unlock()
		return nil
	}

	// Call API to get subscription info
	resp, err := am.apiClient.GetUserFeatures(ctx)
	if err != nil {
		// If API fails, continue with cached/free features
		return fmt.Errorf("API call failed: %w", err)
	}

	// Parse expiration time if provided
	var expiresAt *time.Time
	if resp.ExpiresAt != nil && *resp.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *resp.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	// Update subscription info
	am.mutex.Lock()
	oldFeatures := am.subscription.Features
	am.subscription = domain.SubscriptionInfo{
		Tier:          resp.Tier,
		Features:      resp.Features,
		ExpiresAt:     expiresAt,
		LastRefreshed: time.Now(),
	}
	newFeatures := am.subscription.Features
	am.mutex.Unlock()

	// Notify about feature changes
	am.notifyFeatureChanges(oldFeatures, newFeatures)

	// Cache the subscription info
	am.saveSubscriptionCache()

	return nil
}

// notifyFeatureChanges notifies the user about removed features
func (am *AuthenticationManagerImpl) notifyFeatureChanges(oldFeatures, newFeatures []string) {
	// Create maps for easier lookup
	oldMap := make(map[string]bool)
	for _, f := range oldFeatures {
		oldMap[f] = true
	}

	newMap := make(map[string]bool)
	for _, f := range newFeatures {
		newMap[f] = true
	}

	// Find removed features
	var removed []string
	for _, f := range oldFeatures {
		if !newMap[f] {
			removed = append(removed, f)
		}
	}

	// Notify user if features were removed
	if len(removed) > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  Some features are no longer available in your subscription:\n")
		for _, feature := range removed {
			fmt.Fprintf(os.Stderr, "  - %s\n", feature)
		}
		fmt.Fprintf(os.Stderr, "\nUpgrade your subscription to regain access.\n\n")
	}
}

// loadCachedSubscription loads subscription info from cache
func (am *AuthenticationManagerImpl) loadCachedSubscription() {
	cachePath := am.getSubscriptionCachePath()
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return // No cache, use defaults
	}

	var cached domain.SubscriptionInfo
	if err := json.Unmarshal(data, &cached); err != nil {
		return
	}

	// Only use cache if it's less than 24 hours old
	if time.Since(cached.LastRefreshed) < 24*time.Hour {
		am.subscription = cached
	}
}

// saveSubscriptionCache saves subscription info to cache
func (am *AuthenticationManagerImpl) saveSubscriptionCache() {
	cachePath := am.getSubscriptionCachePath()
	os.MkdirAll(filepath.Dir(cachePath), 0755)

	am.mutex.RLock()
	data, _ := json.Marshal(am.subscription)
	am.mutex.RUnlock()

	os.WriteFile(cachePath, data, 0644)
}

// getSubscriptionCachePath returns the path to the subscription cache file
func (am *AuthenticationManagerImpl) getSubscriptionCachePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "kilometers", "subscription.json")
}
