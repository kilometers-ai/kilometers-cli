package application

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// DevPluginTester orchestrates the development plugin test flow.
type DevPluginTester struct {
	runtime *PluginRuntimeService
	auth    *AuthService
}

func NewDevPluginTester(runtime *PluginRuntimeService, auth *AuthService) *DevPluginTester {
	return &DevPluginTester{runtime: runtime, auth: auth}
}

// Run starts the plugin, authenticates, and returns a stop function.
func (t *DevPluginTester) Run(ctx context.Context, desc domain.PluginDescriptor, debug domain.DebugOptions) (kmsdk.KilometersPlugin, func(), error) {
	start, err := t.runtime.Start(ctx, desc, debug)
	if err != nil {
		return nil, nil, err
	}

	stop := func() { t.runtime.Stop(start.Client) }

	// Obtain token (empty means Free tier)
	token, err := t.auth.GetPluginToken(ctx, desc.Name)
	if err != nil {
		stop()
		return nil, nil, err
	}

	if err := t.runtime.Authenticate(ctx, start.Plugin, token); err != nil {
		stop()
		return nil, nil, err
	}

	return start.Plugin, stop, nil
}

// Helper to create a default secure token cache for dev flows
func DefaultDevTokenCache() (auth.TokenCache, error) {
	return auth.NewSecureFileTokenCache("~/.km/cache")
}
