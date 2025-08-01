//go:build !premium
// +build !premium

package plugins

import (
	"context"
)

// registerBuiltinPlugins registers only free plugins in the free build
func (pm *PluginManagerImpl) registerBuiltinPlugins(ctx context.Context) error {
	// Register only the basic no-op logger for free builds
	pm.RegisterPlugin(NewNoOpLoggerPlugin())
	return nil
}
