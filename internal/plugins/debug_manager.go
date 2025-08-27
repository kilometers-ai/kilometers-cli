//go:build debug
// +build debug

package plugins

import (
	"context"
	"fmt"
	"log"

	"github.com/kilometers-ai/kilometers-cli/internal/plugins/debug"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// DebugPluginManager manages in-process plugins for debugging
type DebugPluginManager struct {
	plugins     map[string]kmsdk.KilometersPlugin
	debugMode   bool
	apiKey      string
	apiEndpoint string
}

// NewDebugPluginManager creates a new debug plugin manager
func NewDebugPluginManager(debugMode bool) *DebugPluginManager {
	return &DebugPluginManager{
		plugins:   make(map[string]kmsdk.KilometersPlugin),
		debugMode: debugMode,
	}
}

// LoadDebugPlugin loads a plugin in debug mode (in-process)
func (dm *DebugPluginManager) LoadDebugPlugin(pluginName, apiKey, apiEndpoint string) error {
	if !dm.debugMode {
		return fmt.Errorf("debug mode not enabled")
	}

	dm.apiKey = apiKey
	dm.apiEndpoint = apiEndpoint

	var plugin kmsdk.KilometersPlugin

	// Create in-process plugin based on name
	switch pluginName {
	case "api-logger":
		plugin = debug.NewAPILoggerDebug()
		log.Printf("[Debug Plugin Manager] Created in-process api-logger plugin")
	default:
		return fmt.Errorf("debug plugin %s not available", pluginName)
	}

	// Use the PluginJWTAuthenticator for consistent authentication behavior
	authenticator := NewPluginJWTAuthenticator(apiKey, apiEndpoint, dm.debugMode)
	err := authenticator.AuthenticatePlugin(context.Background(), plugin, pluginName)
	if err != nil {
		return fmt.Errorf("debug plugin authentication failed: %w", err)
	}

	if dm.debugMode {
		log.Printf("[Debug Plugin Manager] Plugin %s authenticated successfully", pluginName)
	}

	dm.plugins[pluginName] = plugin
	return nil
}

// ProcessMessage forwards messages to loaded debug plugins
func (dm *DebugPluginManager) ProcessMessage(ctx context.Context, pluginName string, message []byte, direction string) ([]kmsdk.Event, error) {
	plugin, exists := dm.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded in debug mode", pluginName)
	}

	// This runs in the same process - you can set breakpoints!
	return plugin.ProcessMessage(ctx, message, direction)
}

// GetLoadedPlugins returns all loaded debug plugins
func (dm *DebugPluginManager) GetLoadedPlugins() map[string]kmsdk.KilometersPlugin {
	result := make(map[string]kmsdk.KilometersPlugin)
	for name, plugin := range dm.plugins {
		result[name] = plugin
	}
	return result
}

// Shutdown shuts down all debug plugins
func (dm *DebugPluginManager) Shutdown() {
	for name, plugin := range dm.plugins {
		if debugPlugin, ok := plugin.(*debug.APILoggerDebug); ok {
			debugPlugin.Shutdown()
		}
		log.Printf("[Debug Plugin Manager] Shut down debug plugin: %s", name)
	}
	dm.plugins = make(map[string]kmsdk.KilometersPlugin)
}

// IsDebugMode returns whether debug mode is enabled
func (dm *DebugPluginManager) IsDebugMode() bool {
	return dm.debugMode
}
