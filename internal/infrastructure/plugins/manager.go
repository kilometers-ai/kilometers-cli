package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// PluginManagerImpl implements the PluginManager interface
type PluginManagerImpl struct {
	plugins      map[string]ports.Plugin
	authManager  ports.AuthenticationManager
	mutex        sync.RWMutex
	lastRefresh  time.Time
	dependencies ports.PluginDependencies
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(authManager ports.AuthenticationManager, apiClient ports.APIClient, config domain.Config) *PluginManagerImpl {
	pm := &PluginManagerImpl{
		plugins:     make(map[string]ports.Plugin),
		authManager: authManager,
		dependencies: ports.PluginDependencies{
			Config:      config,
			AuthManager: authManager,
			APIClient:   apiClient,
		},
	}

	// Register built-in plugins
	pm.registerBuiltinPlugins(context.Background())

	return pm
}

// RegisterPlugin registers a plugin with the manager
func (pm *PluginManagerImpl) RegisterPlugin(plugin ports.Plugin) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.plugins[plugin.Name()] = plugin
	return nil
}

// RefreshFeatures refreshes subscription info from the API
func (pm *PluginManagerImpl) RefreshFeatures(ctx context.Context) error {
	// Refresh subscription info from API
	if err := pm.authManager.RefreshSubscription(ctx); err != nil {
		return fmt.Errorf("failed to refresh subscription: %w", err)
	}

	pm.lastRefresh = time.Now()
	return nil
}

// GetEnabledPlugins returns only the plugins that are enabled for the current subscription
func (pm *PluginManagerImpl) GetEnabledPlugins() []ports.Plugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Check if we need to refresh (every 5 minutes)
	if time.Since(pm.lastRefresh) > 5*time.Minute {
		go pm.RefreshFeatures(context.Background())
	}

	var enabled []ports.Plugin
	tier := pm.authManager.GetSubscriptionTier()

	for _, plugin := range pm.plugins {
		// Check if plugin's required feature is enabled
		requiredFeature := plugin.RequiredFeature()
		if pm.authManager.IsFeatureEnabled(requiredFeature) {
			// Also verify tier requirement
			if pm.isTierSufficient(tier, plugin.RequiredTier()) {
				enabled = append(enabled, plugin)
			}
		}
	}

	return enabled
}

// IsPluginEnabled checks if a specific plugin is enabled
func (pm *PluginManagerImpl) IsPluginEnabled(name string) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return false
	}

	tier := pm.authManager.GetSubscriptionTier()
	return pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) &&
		pm.isTierSufficient(tier, plugin.RequiredTier())
}

// InitializePlugins initializes all enabled plugins
func (pm *PluginManagerImpl) InitializePlugins(ctx context.Context) error {
	for _, plugin := range pm.GetEnabledPlugins() {
		if err := plugin.Initialize(ctx, pm.dependencies); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
		}
	}
	return nil
}

// ShutdownPlugins shuts down all enabled plugins
func (pm *PluginManagerImpl) ShutdownPlugins(ctx context.Context) error {
	for _, plugin := range pm.GetEnabledPlugins() {
		if err := plugin.Shutdown(ctx); err != nil {
			// Log error but continue shutting down other plugins
			fmt.Printf("Error shutting down plugin %s: %v\n", plugin.Name(), err)
		}
	}
	return nil
}

// GetMessageHandler returns a composite message handler that delegates to all enabled plugins
func (pm *PluginManagerImpl) GetMessageHandler() ports.MessageHandler {
	return &compositeMessageHandler{
		pluginManager: pm,
	}
}

// isTierSufficient checks if the user's tier meets the plugin's requirements
func (pm *PluginManagerImpl) isTierSufficient(userTier, requiredTier domain.SubscriptionTier) bool {
	tierLevels := map[domain.SubscriptionTier]int{
		domain.TierFree:       0,
		domain.TierPro:        1,
		domain.TierEnterprise: 2,
	}

	userLevel, ok1 := tierLevels[userTier]
	requiredLevel, ok2 := tierLevels[requiredTier]

	if !ok1 || !ok2 {
		return false
	}

	return userLevel >= requiredLevel
}

// compositeMessageHandler implements MessageHandler by delegating to all enabled plugins
type compositeMessageHandler struct {
	pluginManager *PluginManagerImpl
}

// HandleMessage delegates to all enabled plugins
func (h *compositeMessageHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	for _, plugin := range h.pluginManager.GetEnabledPlugins() {
		if err := plugin.HandleMessage(ctx, data, direction); err != nil {
			// Log error but continue with other plugins
			fmt.Printf("Plugin %s error handling message: %v\n", plugin.Name(), err)
		}
	}
	return nil
}

// HandleError delegates to all enabled plugins
func (h *compositeMessageHandler) HandleError(ctx context.Context, err error) {
	for _, plugin := range h.pluginManager.GetEnabledPlugins() {
		plugin.HandleError(ctx, err)
	}
}

// HandleStreamEvent delegates to all enabled plugins
func (h *compositeMessageHandler) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	for _, plugin := range h.pluginManager.GetEnabledPlugins() {
		plugin.HandleStreamEvent(ctx, event)
	}
}

// SetCorrelationID sets the correlation ID for plugins that support it
func (h *compositeMessageHandler) SetCorrelationID(correlationID string) {
	for _, plugin := range h.pluginManager.GetEnabledPlugins() {
		if setter, ok := plugin.(interface{ SetCorrelationID(string) }); ok {
			setter.SetCorrelationID(correlationID)
		}
	}
}
