//go:build !debug
// +build !debug

package cli

import (
	"context"
	"fmt"

	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
)

// runInProcessPluginTest - non-debug version that returns an error
func runInProcessPluginTest(ctx context.Context, pluginName, version string, verbose, noAuth, pluginDebug bool, config *configpkg.UnifiedConfig) error {
	fmt.Printf("🔬 Starting in-process plugin test for %s\n", pluginName)
	fmt.Printf("⚠️  Note: This requires a debug build (go build -tags debug)\n\n")

	// In-process debugging is not available without debug build tag
	return fmt.Errorf("in-process debugging requires a debug build. Build with: go build -tags debug")
}