package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// EnhancedPluginManager supports both built-in and private repository plugins
type EnhancedPluginManager struct {
	builtinPlugins  map[string]ports.Plugin
	privatePlugins  map[string]ports.Plugin
	wasmLoader      *WASMPluginLoader
	authManager     *domain.AuthenticationManager
	deps            ports.PluginDependencies
	mutex           sync.RWMutex
	registryConfig  RegistryConfig
}

// RegistryConfig contains private registry configuration
type RegistryConfig struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	AuthToken   string `json:"auth_token"`
	AutoUpdate  bool   `json:"auto_update"`
	CacheSize   int    `json:"cache_size"`
	TrustedKeys []string `json:"trusted_keys"`
}

// PluginSource indicates where a plugin comes from
type PluginSource string

const (
	SourceBuiltin PluginSource = "builtin"
	SourcePrivate PluginSource = "private"
	SourceExternal PluginSource = "external"
)

// ExtendedPluginInfo provides additional plugin information
type ExtendedPluginInfo struct {
	Plugin      ports.Plugin
	Source      PluginSource
	Version     string
	Description string
	Author      string
	Homepage    string
	Permissions []string
}

// NewEnhancedPluginManager creates a plugin manager with private repository support
func NewEnhancedPluginManager(
	authManager *domain.AuthenticationManager,
	deps ports.PluginDependencies,
	registryConfig RegistryConfig,
) *EnhancedPluginManager {
	
	var wasmLoader *WASMPluginLoader
	if registryConfig.Enabled {
		wasmLoader = NewWASMPluginLoader(registryConfig.URL, registryConfig.AuthToken)
	}

	return &EnhancedPluginManager{
		builtinPlugins: make(map[string]ports.Plugin),
		privatePlugins: make(map[string]ports.Plugin),
		wasmLoader:     wasmLoader,
		authManager:    authManager,
		deps:           deps,
		registryConfig: registryConfig,
	}
}

// LoadPlugins discovers and loads all available plugins
func (pm *EnhancedPluginManager) LoadPlugins(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Load built-in plugins
	if err := pm.loadBuiltinPlugins(ctx); err != nil {
		return fmt.Errorf("failed to load builtin plugins: %w", err)
	}

	// Load private repository plugins if enabled
	if pm.registryConfig.Enabled && pm.wasmLoader != nil {
		if err := pm.loadPrivatePlugins(ctx); err != nil {
			// Don't fail if private plugins can't be loaded
			fmt.Printf("Warning: Failed to load private plugins: %v\n", err)
		}
	}

	return nil
}

// GetPlugin retrieves a plugin by name (searches built-in first, then private)
func (pm *EnhancedPluginManager) GetPlugin(name string) (ports.Plugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Check built-in plugins first
	if plugin, exists := pm.builtinPlugins[name]; exists {
		if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) {
			return plugin, nil
		}
		return nil, fmt.Errorf("plugin '%s' requires %s feature (subscription: %s)", 
			name, plugin.RequiredFeature(), pm.authManager.GetSubscriptionTier())
	}

	// Check private plugins
	if plugin, exists := pm.privatePlugins[name]; exists {
		if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) {
			return plugin, nil
		}
		return nil, fmt.Errorf("plugin '%s' requires %s feature (subscription: %s)", 
			name, plugin.RequiredFeature(), pm.authManager.GetSubscriptionTier())
	}

	return nil, fmt.Errorf("plugin '%s' not found", name)
}

// GetAvailablePlugins returns all plugins available for current subscription
func (pm *EnhancedPluginManager) GetAvailablePlugins(ctx context.Context) []ports.Plugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var available []ports.Plugin

	// Check built-in plugins
	for _, plugin := range pm.builtinPlugins {
		if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
			available = append(available, plugin)
		}
	}

	// Check private plugins
	for _, plugin := range pm.privatePlugins {
		if pm.authManager.IsFeatureEnabled(plugin.RequiredFeature()) && plugin.IsAvailable(ctx) {
			available = append(available, plugin)
		}
	}

	return available
}

// GetExtendedPluginInfo returns detailed plugin information
func (pm *EnhancedPluginManager) GetExtendedPluginInfo(ctx context.Context) []ExtendedPluginInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var plugins []ExtendedPluginInfo

	// Built-in plugins
	for name, plugin := range pm.builtinPlugins {
		info := ExtendedPluginInfo{
			Plugin:      plugin,
			Source:      SourceBuiltin,
			Version:     "builtin",
			Description: fmt.Sprintf("Built-in %s plugin", name),
			Author:      "Kilometers.ai",
			Homepage:    "https://kilometers.ai",
			Permissions: []string{"basic"},
		}
		plugins = append(plugins, info)
	}

	// Private plugins  
	for name, plugin := range pm.privatePlugins {
		// Get additional metadata from WASM plugin
		wasmPlugin, ok := plugin.(*WASMPlugin)
		if ok {
			info := ExtendedPluginInfo{
				Plugin:      plugin,
				Source:      SourcePrivate,
				Version:     wasmPlugin.version,
				Description: wasmPlugin.metadata.Description,
				Author:      wasmPlugin.metadata.Metadata["author"],
				Homepage:    wasmPlugin.metadata.Metadata["homepage"],
				Permissions: wasmPlugin.metadata.Permissions,
			}
			plugins = append(plugins, info)
		} else {
			info := ExtendedPluginInfo{
				Plugin:      plugin,
				Source:      SourcePrivate,
				Version:     "unknown",
				Description: fmt.Sprintf("Private %s plugin", name),
				Permissions: []string{"extended"},
			}
			plugins = append(plugins, info)
		}
	}

	return plugins
}

// InstallPlugin downloads and installs a plugin from private registry
func (pm *EnhancedPluginManager) InstallPlugin(ctx context.Context, name, version string) error {
	if pm.wasmLoader == nil {
		return fmt.Errorf("private registry not configured")
	}

	// Download and load plugin
	plugin, err := pm.wasmLoader.LoadPlugin(ctx, name, version)
	if err != nil {
		return fmt.Errorf("failed to install plugin %s@%s: %w", name, version, err)
	}

	// Initialize plugin
	if err := plugin.Initialize(pm.deps); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Add to private plugins
	pm.mutex.Lock()
	pm.privatePlugins[name] = plugin
	pm.mutex.Unlock()

	// Save to plugin configuration
	configs, _ := domain.LoadPluginConfigs()
	configs.EnablePlugin(name)
	configs.UpdatePluginSetting(name, "source", "private")
	configs.UpdatePluginSetting(name, "version", version)
	domain.SavePluginConfigs(configs)

	return nil
}

// UninstallPlugin removes a plugin
func (pm *EnhancedPluginManager) UninstallPlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if it's a private plugin
	if plugin, exists := pm.privatePlugins[name]; exists {
		// Cleanup plugin
		if err := plugin.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup plugin %s: %w", name, err)
		}

		// Remove from private plugins
		delete(pm.privatePlugins, name)

		// Remove from configuration
		configs, _ := domain.LoadPluginConfigs()
		configs.DisablePlugin(name)
		domain.SavePluginConfigs(configs)

		return nil
	}

	// Check if it's a built-in plugin (can only disable, not uninstall)
	if _, exists := pm.builtinPlugins[name]; exists {
		// Just disable it
		configs, _ := domain.LoadPluginConfigs()
		configs.DisablePlugin(name)
		domain.SavePluginConfigs(configs)
		return nil
	}

	return fmt.Errorf("plugin '%s' not found", name)
}

// UpdatePlugins checks for and installs plugin updates
func (pm *EnhancedPluginManager) UpdatePlugins(ctx context.Context) error {
	if pm.wasmLoader == nil {
		return fmt.Errorf("private registry not configured")
	}

	// Get list of installed private plugins
	var installedPlugins []string
	pm.mutex.RLock()
	for name := range pm.privatePlugins {
		installedPlugins = append(installedPlugins, name)
	}
	pm.mutex.RUnlock()

	// Update plugins
	return pm.wasmLoader.UpdatePlugins(ctx, installedPlugins)
}

// SearchPlugins searches for plugins in private registry
func (pm *EnhancedPluginManager) SearchPlugins(ctx context.Context, query string) ([]ExtendedPluginInfo, error) {
	if pm.wasmLoader == nil {
		return nil, fmt.Errorf("private registry not configured")
	}

	// Get all available plugins from registry
	tier := pm.authManager.GetSubscriptionTier()
	availablePlugins, err := pm.wasmLoader.DiscoverPlugins(ctx, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to search plugins: %w", err)
	}

	// Filter by query (simple substring match)
	var results []ExtendedPluginInfo
	for _, plugin := range availablePlugins {
		wasmPlugin, ok := plugin.(*WASMPlugin)
		if ok {
			// Simple search in name and description
			if pm.matchesQuery(wasmPlugin.name, query) || 
			   pm.matchesQuery(wasmPlugin.metadata.Description, query) {
				
				info := ExtendedPluginInfo{
					Plugin:      plugin,
					Source:      SourcePrivate,
					Version:     wasmPlugin.version,
					Description: wasmPlugin.metadata.Description,
					Author:      wasmPlugin.metadata.Metadata["author"],
					Homepage:    wasmPlugin.metadata.Metadata["homepage"],
					Permissions: wasmPlugin.metadata.Permissions,
				}
				results = append(results, info)
			}
		}
	}

	return results, nil
}

// ExecutePlugin runs a plugin with given parameters
func (pm *EnhancedPluginManager) ExecutePlugin(ctx context.Context, name string, params ports.PluginParams) (ports.PluginResult, error) {
	plugin, err := pm.GetPlugin(name)
	if err != nil {
		return ports.PluginResult{}, err
	}

	if !plugin.IsAvailable(ctx) {
		return ports.PluginResult{}, fmt.Errorf("plugin '%s' is not available", name)
	}

	return plugin.Execute(ctx, params)
}

// ListFeatures returns all available features for current subscription
func (pm *EnhancedPluginManager) ListFeatures(ctx context.Context) []string {
	return pm.authManager.ListFeatures(ctx)
}

// GetRegistryStatus returns information about private registry connection
func (pm *EnhancedPluginManager) GetRegistryStatus(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"enabled": pm.registryConfig.Enabled,
		"url":     pm.registryConfig.URL,
	}

	if pm.wasmLoader != nil {
		// Test registry connection
		if err := pm.wasmLoader.registry.AuthenticateWithRegistry(ctx); err != nil {
			status["connected"] = false
			status["error"] = err.Error()
		} else {
			status["connected"] = true
		}
	} else {
		status["connected"] = false
		status["error"] = "WASM loader not initialized"
	}

	return status
}

// Private methods

func (pm *EnhancedPluginManager) loadBuiltinPlugins(ctx context.Context) error {
	// Load the same built-in plugins as before
	plugins := []ports.Plugin{
		NewAdvancedFilterPlugin(),
		NewPoisonDetectionPlugin(),
		NewMLAnalyticsPlugin(),
		NewCompliancePlugin(),
	}

	for _, plugin := range plugins {
		if err := plugin.Initialize(pm.deps); err != nil {
			return fmt.Errorf("failed to initialize builtin plugin '%s': %w", plugin.Name(), err)
		}

		pm.builtinPlugins[plugin.Name()] = plugin
	}

	return nil
}

func (pm *EnhancedPluginManager) loadPrivatePlugins(ctx context.Context) error {
	// Load plugin configuration to see which private plugins should be loaded
	configs, err := domain.LoadPluginConfigs()
	if err != nil {
		return fmt.Errorf("failed to load plugin configs: %w", err)
	}

	// Load enabled private plugins
	for name, config := range configs.Plugins {
		if config.Enabled && config.Settings["source"] == "private" {
			version, ok := config.Settings["version"].(string)
			if !ok {
				version = "latest"
			}

			plugin, err := pm.wasmLoader.LoadPlugin(ctx, name, version)
			if err != nil {
				fmt.Printf("Warning: Failed to load private plugin %s: %v\n", name, err)
				continue
			}

			if err := plugin.Initialize(pm.deps); err != nil {
				fmt.Printf("Warning: Failed to initialize private plugin %s: %v\n", name, err)
				continue
			}

			pm.privatePlugins[name] = plugin
		}
	}

	return nil
}

func (pm *EnhancedPluginManager) matchesQuery(text, query string) bool {
	// Simple case-insensitive substring match
	// In production, you might want more sophisticated search
	return len(query) == 0 || 
		   (len(text) > 0 && strings.Contains(strings.ToLower(text), strings.ToLower(query)))
}
