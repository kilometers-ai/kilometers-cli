package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// PluginManagerImpl implements the PluginManager interface
type PluginManagerImpl struct {
	plugins     map[string]ports.Plugin
	authManager *domain.AuthenticationManager
	deps        ports.PluginDependencies
	mutex       sync.RWMutex
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(authManager *domain.AuthenticationManager, deps ports.PluginDependencies) *PluginManagerImpl {
	return &PluginManagerImpl{
		plugins:     make(map[string]ports.Plugin),
		authManager: authManager,
		deps:        deps,
	}
}

// LoadPlugins discovers and loads available plugins
func (pm *PluginManagerImpl) LoadPlugins(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Register built-in plugins based on subscription tier
	if err := pm.registerBuiltinPlugins(ctx); err != nil {
		return fmt.Errorf("failed to register builtin plugins: %w", err)
	}

	return nil
}

// GetPlugin retrieves a plugin by name
func (pm *PluginManagerImpl) GetPlugin(name string) (ports.Plugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' not found", name)
	}

	// Check if user has access to this plugin
	if !pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) {
		return nil, fmt.Errorf("plugin '%s' requires %s feature (subscription: %s)", 
			name, plugin.RequiredFeature(), pm.authManager.GetSubscriptionTier())
	}

	return plugin, nil
}

// GetAvailablePlugins returns plugins available for current subscription
func (pm *PluginManagerImpl) GetAvailablePlugins(ctx context.Context) []ports.Plugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var available []ports.Plugin
	for _, plugin := range pm.plugins {
		if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
			available = append(available, plugin)
		}
	}

	return available
}

// ExecutePlugin runs a plugin with given parameters
func (pm *PluginManagerImpl) ExecutePlugin(ctx context.Context, name string, params ports.PluginParams) (ports.PluginResult, error) {
	plugin, err := pm.GetPlugin(name)
	if err != nil {
		return ports.PluginResult{}, err
	}

	// Validate plugin is available
	if !plugin.IsAvailable(ctx) {
		return ports.PluginResult{}, fmt.Errorf("plugin '%s' is not available", name)
	}

	// Execute the plugin
	result, err := plugin.Execute(ctx, params)
	if err != nil {
		return ports.PluginResult{
			Success: false,
			Error:   err,
		}, err
	}

	return result, nil
}

// ListFeatures returns all available features for current subscription
func (pm *PluginManagerImpl) ListFeatures(ctx context.Context) []string {
	tier := pm.authManager.GetSubscriptionTier()
	
	var features []string
	switch tier {
	case domain.TierFree:
		features = []string{
			domain.FeatureBasicMonitoring,
		}
	case domain.TierPro:
		features = []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeatureCustomRules,
			domain.FeaturePoisonDetection,
			domain.FeatureMLAnalytics,
		}
	case domain.TierEnterprise:
		features = []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeatureCustomRules,
			domain.FeaturePoisonDetection,
			domain.FeatureMLAnalytics,
			domain.FeatureTeamCollaboration,
			domain.FeatureCustomDashboards,
			domain.FeatureComplianceReporting,
			domain.FeatureAPIIntegrations,
			domain.FeaturePrioritySupport,
		}
	}

	return features
}

// registerBuiltinPlugins registers the built-in plugins
func (pm *PluginManagerImpl) registerBuiltinPlugins(ctx context.Context) error {
	plugins := []ports.Plugin{
		NewAdvancedFilterPlugin(),
		NewPoisonDetectionPlugin(),
		NewMLAnalyticsPlugin(),
		NewCompliancePlugin(),
	}

	for _, plugin := range plugins {
		if err := plugin.Initialize(pm.deps); err != nil {
			return fmt.Errorf("failed to initialize plugin '%s': %w", plugin.Name(), err)
		}

		pm.plugins[plugin.Name()] = plugin
	}

	return nil
}

// GetFilterPlugins returns all available filter plugins
func (pm *PluginManagerImpl) GetFilterPlugins(ctx context.Context) []ports.FilterPlugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var filters []ports.FilterPlugin
	for _, plugin := range pm.plugins {
		if filterPlugin, ok := plugin.(ports.FilterPlugin); ok {
			if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
				filters = append(filters, filterPlugin)
			}
		}
	}

	return filters
}

// GetSecurityPlugins returns all available security plugins
func (pm *PluginManagerImpl) GetSecurityPlugins(ctx context.Context) []ports.SecurityPlugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var securityPlugins []ports.SecurityPlugin
	for _, plugin := range pm.plugins {
		if securityPlugin, ok := plugin.(ports.SecurityPlugin); ok {
			if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
				securityPlugins = append(securityPlugins, securityPlugin)
			}
		}
	}

	return securityPlugins
}

// GetAnalyticsPlugins returns all available analytics plugins
func (pm *PluginManagerImpl) GetAnalyticsPlugins(ctx context.Context) []ports.AnalyticsPlugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var analyticsPlugins []ports.AnalyticsPlugin
	for _, plugin := range pm.plugins {
		if analyticsPlugin, ok := plugin.(ports.AnalyticsPlugin); ok {
			if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
				analyticsPlugins = append(analyticsPlugins, analyticsPlugin)
			}
		}
	}

	return analyticsPlugins
}
