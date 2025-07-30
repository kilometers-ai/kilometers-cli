package plugins

import (
	"context"
	"testing"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPlugin for testing
type MockPlugin struct {
	name            string
	requiredFeature string
	requiredTier    domain.SubscriptionTier
	available       bool
}

func (m *MockPlugin) Name() string                                 { return m.name }
func (m *MockPlugin) RequiredFeature() string                     { return m.requiredFeature }
func (m *MockPlugin) RequiredTier() domain.SubscriptionTier       { return m.requiredTier }
func (m *MockPlugin) Initialize(deps ports.PluginDependencies) error { return nil }
func (m *MockPlugin) IsAvailable(ctx context.Context) bool        { return m.available }
func (m *MockPlugin) Execute(ctx context.Context, params ports.PluginParams) (ports.PluginResult, error) {
	return ports.PluginResult{Success: true}, nil
}
func (m *MockPlugin) Cleanup() error { return nil }

// MockMessageHandler for testing
type MockMessageHandler struct{}

func (m *MockMessageHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	return nil
}
func (m *MockMessageHandler) HandleError(ctx context.Context, err error)           {}
func (m *MockMessageHandler) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {}
func (m *MockMessageHandler) LogMessage(message *domain.JSONRPCMessage)           {}
func (m *MockMessageHandler) LogWarning(message string)                           {}

func TestPluginManager_LoadPlugins(t *testing.T) {
	// Create mock auth manager
	authManager := domain.NewAuthenticationManager()
	
	// Create mock dependencies
	deps := ports.PluginDependencies{
		AuthManager:   authManager,
		MessageLogger: &MockMessageHandler{},
		Config:        domain.DefaultConfig(),
	}

	// Create plugin manager
	pluginManager := NewPluginManager(authManager, deps)

	// Test plugin loading
	ctx := context.Background()
	err := pluginManager.LoadPlugins(ctx)
	require.NoError(t, err)

	// Verify plugins are loaded (free tier should have no plugins)
	availablePlugins := pluginManager.GetAvailablePlugins(ctx)
	assert.Empty(t, availablePlugins, "Free tier should have no plugins available")
}

func TestPluginManager_GetPlugin(t *testing.T) {
	// Create auth manager with Pro subscription
	authManager := domain.NewAuthenticationManager()
	authManager.SaveSubscription(&domain.SubscriptionConfig{
		Tier: domain.TierPro,
		Features: []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeaturePoisonDetection,
		},
	})

	// Create dependencies
	deps := ports.PluginDependencies{
		AuthManager:   authManager,
		MessageLogger: &MockMessageHandler{},
		Config:        domain.DefaultConfig(),
	}

	// Create plugin manager
	pluginManager := NewPluginManager(authManager, deps)
	
	// Load plugins
	ctx := context.Background()
	err := pluginManager.LoadPlugins(ctx)
	require.NoError(t, err)

	// Test getting existing plugin
	plugin, err := pluginManager.GetPlugin("advanced-filters")
	assert.NoError(t, err)
	assert.NotNil(t, plugin)
	assert.Equal(t, "advanced-filters", plugin.Name())

	// Test getting non-existent plugin
	_, err = pluginManager.GetPlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManager_FeatureAccess(t *testing.T) {
	tests := []struct {
		name         string
		tier         domain.SubscriptionTier
		features     []string
		pluginName   string
		shouldAccess bool
	}{
		{
			name:         "free tier cannot access pro plugin",
			tier:         domain.TierFree,
			features:     []string{domain.FeatureBasicMonitoring},
			pluginName:   "advanced-filters",
			shouldAccess: false,
		},
		{
			name:         "pro tier can access pro plugin",
			tier:         domain.TierPro,
			features:     []string{domain.FeatureBasicMonitoring, domain.FeatureAdvancedFilters},
			pluginName:   "advanced-filters",
			shouldAccess: true,
		},
		{
			name:         "pro tier cannot access enterprise plugin",
			tier:         domain.TierPro,
			features:     []string{domain.FeatureBasicMonitoring, domain.FeatureAdvancedFilters},
			pluginName:   "compliance-reporting",
			shouldAccess: false,
		},
		{
			name: "enterprise tier can access all plugins",
			tier: domain.TierEnterprise,
			features: []string{
				domain.FeatureBasicMonitoring,
				domain.FeatureAdvancedFilters,
				domain.FeatureComplianceReporting,
			},
			pluginName:   "compliance-reporting",
			shouldAccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth manager with test subscription
			authManager := domain.NewAuthenticationManager()
			
			if tt.tier != domain.TierFree {
				authManager.SaveSubscription(&domain.SubscriptionConfig{
					Tier:     tt.tier,
					Features: tt.features,
				})
			}

			// Create dependencies
			deps := ports.PluginDependencies{
				AuthManager:   authManager,
				MessageLogger: &MockMessageHandler{},
				Config:        domain.DefaultConfig(),
			}

			// Create plugin manager
			pluginManager := NewPluginManager(authManager, deps)
			
			// Load plugins
			ctx := context.Background()
			err := pluginManager.LoadPlugins(ctx)
			require.NoError(t, err)

			// Test plugin access
			_, err = pluginManager.GetPlugin(tt.pluginName)
			
			if tt.shouldAccess {
				assert.NoError(t, err, "Should have access to plugin")
			} else {
				assert.Error(t, err, "Should not have access to plugin")
			}
		})
	}
}

func TestPluginManager_ExecutePlugin(t *testing.T) {
	// Create auth manager with Pro subscription
	authManager := domain.NewAuthenticationManager()
	authManager.SaveSubscription(&domain.SubscriptionConfig{
		Tier: domain.TierPro,
		Features: []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
		},
	})

	// Create dependencies
	deps := ports.PluginDependencies{
		AuthManager:   authManager,
		MessageLogger: &MockMessageHandler{},
		Config:        domain.DefaultConfig(),
	}

	// Create plugin manager
	pluginManager := NewPluginManager(authManager, deps)
	
	// Load plugins
	ctx := context.Background()
	err := pluginManager.LoadPlugins(ctx)
	require.NoError(t, err)

	// Test plugin execution
	params := ports.PluginParams{
		Command: "test",
		Args:    []string{},
		Data:    make(map[string]interface{}),
	}

	result, err := pluginManager.ExecutePlugin(ctx, "advanced-filters", params)
	assert.NoError(t, err)
	assert.True(t, result.Success)

	// Test execution of non-accessible plugin
	_, err = pluginManager.ExecutePlugin(ctx, "compliance-reporting", params)
	assert.Error(t, err)
}

func TestPluginManager_ListFeatures(t *testing.T) {
	tests := []struct {
		name           string
		tier           domain.SubscriptionTier
		expectedCount  int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:          "free tier features",
			tier:          domain.TierFree,
			expectedCount: 1,
			shouldContain: []string{domain.FeatureBasicMonitoring},
			shouldNotContain: []string{domain.FeatureAdvancedFilters, domain.FeatureComplianceReporting},
		},
		{
			name:          "pro tier features",
			tier:          domain.TierPro,
			expectedCount: 5,
			shouldContain: []string{
				domain.FeatureBasicMonitoring,
				domain.FeatureAdvancedFilters,
				domain.FeaturePoisonDetection,
			},
			shouldNotContain: []string{domain.FeatureComplianceReporting},
		},
		{
			name:          "enterprise tier features",
			tier:          domain.TierEnterprise,
			expectedCount: 10,
			shouldContain: []string{
				domain.FeatureBasicMonitoring,
				domain.FeatureAdvancedFilters,
				domain.FeatureComplianceReporting,
				domain.FeaturePrioritySupport,
			},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create auth manager with test subscription
			authManager := domain.NewAuthenticationManager()
			
			if tt.tier != domain.TierFree {
				authManager.SaveSubscription(&domain.SubscriptionConfig{
					Tier: tt.tier,
				})
			}

			// Create dependencies
			deps := ports.PluginDependencies{
				AuthManager:   authManager,
				MessageLogger: &MockMessageHandler{},
				Config:        domain.DefaultConfig(),
			}

			// Create plugin manager
			pluginManager := NewPluginManager(authManager, deps)

			// Test feature listing
			ctx := context.Background()
			features := pluginManager.ListFeatures(ctx)

			assert.Len(t, features, tt.expectedCount)

			for _, feature := range tt.shouldContain {
				assert.Contains(t, features, feature)
			}

			for _, feature := range tt.shouldNotContain {
				assert.NotContains(t, features, feature)
			}
		})
	}
}

func TestPluginManager_GetPluginsByType(t *testing.T) {
	// Create auth manager with Pro subscription
	authManager := domain.NewAuthenticationManager()
	authManager.SaveSubscription(&domain.SubscriptionConfig{
		Tier: domain.TierPro,
		Features: []string{
			domain.FeatureBasicMonitoring,
			domain.FeatureAdvancedFilters,
			domain.FeaturePoisonDetection,
			domain.FeatureMLAnalytics,
		},
	})

	// Create dependencies
	deps := ports.PluginDependencies{
		AuthManager:   authManager,
		MessageLogger: &MockMessageHandler{},
		Config:        domain.DefaultConfig(),
	}

	// Create plugin manager
	pluginManager := NewPluginManager(authManager, deps)
	pluginManagerImpl := pluginManager.(*PluginManagerImpl)
	
	// Load plugins
	ctx := context.Background()
	err := pluginManager.LoadPlugins(ctx)
	require.NoError(t, err)

	// Test getting filter plugins
	filterPlugins := pluginManagerImpl.GetFilterPlugins(ctx)
	assert.NotEmpty(t, filterPlugins, "Should have filter plugins")

	// Test getting security plugins
	securityPlugins := pluginManagerImpl.GetSecurityPlugins(ctx)
	assert.NotEmpty(t, securityPlugins, "Should have security plugins")

	// Test getting analytics plugins
	analyticsPlugins := pluginManagerImpl.GetAnalyticsPlugins(ctx)
	assert.NotEmpty(t, analyticsPlugins, "Should have analytics plugins")
}
