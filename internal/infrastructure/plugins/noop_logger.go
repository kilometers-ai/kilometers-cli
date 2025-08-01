package plugins

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// NoOpLoggerPlugin is a basic logger that does nothing (for free tier)
type NoOpLoggerPlugin struct {
	deps ports.PluginDependencies
}

// NewNoOpLoggerPlugin creates a new no-op logger plugin
func NewNoOpLoggerPlugin() *NoOpLoggerPlugin {
	return &NoOpLoggerPlugin{}
}

// Name returns the plugin name
func (p *NoOpLoggerPlugin) Name() string {
	return "noop-logger"
}

// RequiredFeature returns the required feature flag
func (p *NoOpLoggerPlugin) RequiredFeature() string {
	return domain.FeatureBasicMonitoring
}

// RequiredTier returns the minimum subscription tier
func (p *NoOpLoggerPlugin) RequiredTier() domain.SubscriptionTier {
	return domain.TierFree
}

// Initialize initializes the plugin
func (p *NoOpLoggerPlugin) Initialize(ctx context.Context, deps ports.PluginDependencies) error {
	p.deps = deps
	return nil
}

// Shutdown shuts down the plugin
func (p *NoOpLoggerPlugin) Shutdown(ctx context.Context) error {
	return nil
}

// HandleMessage processes a message (does nothing in free tier)
func (p *NoOpLoggerPlugin) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Silent - no logging to avoid interfering with MCP communication
	return nil
}

// HandleError processes an error
func (p *NoOpLoggerPlugin) HandleError(ctx context.Context, err error) {
	// Silent - no logging to avoid interfering with MCP communication
}

// HandleStreamEvent processes stream events
func (p *NoOpLoggerPlugin) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	// Silent - no logging to avoid interfering with MCP communication
}
