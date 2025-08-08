package ports

// Re-export plugin interfaces that external plugins need
// This allows external modules to import plugin interfaces without accessing internal packages

import (
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// Re-export plugin interfaces
type Plugin = ports.Plugin
type PluginDependencies = ports.PluginDependencies
type PluginManager = ports.PluginManager
type AuthenticationManager = ports.AuthenticationManager
type APIClient = ports.APIClient
type UserFeaturesResponse = ports.UserFeaturesResponse
type MessageHandler = ports.MessageHandler
type StreamEvent = ports.StreamEvent

// RegisterPlugin is a helper function to register plugins with a PluginManager
// This provides a type-safe way for external plugins to register themselves
func RegisterPlugin(pm interface{ RegisterPlugin(plugin Plugin) error }, plugin Plugin) error {
	return pm.RegisterPlugin(plugin)
}
