package application

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// PluginRuntimeService starts/stops a plugin process and performs runtime authentication.
type PluginRuntimeService struct{}

func NewPluginRuntimeService() *PluginRuntimeService {
	return &PluginRuntimeService{}
}

type StartResult struct {
	Plugin kmsdk.KilometersPlugin
	Client *plugin.Client
}

// Start launches the plugin and returns a handle; when debug options are enabled,
// the underlying loader already wires a plugin logger via CreatePluginLogger.
func (s *PluginRuntimeService) Start(ctx context.Context, desc domain.PluginDescriptor, opts domain.DebugOptions) (*StartResult, error) {
	result, err := plugins.LoadSinglePlugin(ctx, desc.Name, desc.Path, opts.Enabled, false, "")
	if err != nil {
		return nil, err
	}
	return &StartResult{Plugin: result.Plugin, Client: result.Client}, nil
}

// Authenticate calls the plugin's Authenticate method with the provided token.
func (s *PluginRuntimeService) Authenticate(ctx context.Context, p kmsdk.KilometersPlugin, token string) error {
	if err := p.Authenticate(ctx, token); err != nil {
		return fmt.Errorf("plugin authentication failed: %w", err)
	}
	return nil
}

// Stop terminates the plugin process.
func (s *PluginRuntimeService) Stop(client *plugin.Client) {
	if client != nil {
		client.Kill()
	}
}
