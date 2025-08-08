package grpc

import (
	"context"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// SDKClientAdapter adapts the GRPC client to the kmsdk.Plugin interface.
// It allows the host to interact with plugins using the SDK contract while reusing the existing GRPC client.
type SDKClientAdapter struct {
	client      *PluginGRPCClient
	apiEndpoint string
	apiKey      string
}

// NewSDKClientAdapter creates a new adapter for the given GRPC client.
func NewSDKClientAdapter(client *PluginGRPCClient, apiEndpoint, apiKey string) *SDKClientAdapter {
	return &SDKClientAdapter{client: client, apiEndpoint: apiEndpoint, apiKey: apiKey}
}

// Authenticate delegates to the underlying client and returns error on failure.
func (a *SDKClientAdapter) Authenticate(ctx context.Context, token string) error {
	_, err := a.client.Authenticate(ctx, token)
	return err
}

// Initialize maps the SDK config to the existing ports.PluginConfig and initializes the plugin.
func (a *SDKClientAdapter) Initialize(ctx context.Context, cfg kmsdk.Config) error {
	pc := ports.PluginConfig{
		ApiEndpoint: a.apiEndpoint,
		Debug:       cfg.Debug,
		ApiKey:      a.apiKey,
	}
	return a.client.Initialize(ctx, pc)
}

// Shutdown delegates to the underlying client.
func (a *SDKClientAdapter) Shutdown(ctx context.Context) error {
	return a.client.Shutdown(ctx)
}

// HandleMessage forwards a message to the plugin using the SDK direction.
func (a *SDKClientAdapter) HandleMessage(ctx context.Context, data []byte, direction kmsdk.Direction, correlationID string) error {
	return a.client.HandleMessage(ctx, data, string(direction), correlationID)
}

// HandleError forwards an error to the plugin.
func (a *SDKClientAdapter) HandleError(ctx context.Context, err error) {
	_ = a.client.HandleError(ctx, err)
}

// HandleStreamEvent converts the SDK event to ports.StreamEvent and forwards it.
func (a *SDKClientAdapter) HandleStreamEvent(ctx context.Context, event kmsdk.StreamEvent) {
	_ = a.client.HandleStreamEvent(ctx, ports.StreamEvent{
		Type:      ports.StreamEventType(event.Type),
		Timestamp: time.Now().UnixNano(),
		Data:      map[string]interface{}{"message": event.Message},
	})
}

// GetInfo constructs kmsdk.PluginInfo using the client's metadata methods.
func (a *SDKClientAdapter) GetInfo() kmsdk.PluginInfo {
	return kmsdk.PluginInfo{
		Name:         a.client.Name(),
		Version:      a.client.Version(),
		Description:  "",
		RequiredTier: kmsdk.SubscriptionTier(a.client.RequiredTier()),
	}
}
