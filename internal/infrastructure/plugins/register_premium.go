//go:build premium
// +build premium

package plugins

import (
	"context"
	premiumPlugins "github.com/kilometers-ai/kilometers-cli-plugins"
)

// registerBuiltinPlugins registers all plugins including premium ones
func (pm *PluginManagerImpl) registerBuiltinPlugins(ctx context.Context) error {
	// Register free plugins
	pm.RegisterPlugin(NewNoOpLoggerPlugin())

	// Register premium plugins from private repo
	premiumPlugins.RegisterAll(pm)

	return nil
}
