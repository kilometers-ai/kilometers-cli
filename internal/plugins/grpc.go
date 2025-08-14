package plugins

import (
	"context"
	"io"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	pb "github.com/kilometers-ai/kilometers-cli/internal/plugins/proto/generated"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// GetHandshakeConfig returns the handshake configuration for go-plugin
func GetHandshakeConfig() plugin.HandshakeConfig {
	return plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "KILOMETERS_PLUGIN",
		MagicCookieValue: "kilometers_monitoring_plugin",
	}
}

// GetPluginMap returns the plugin map for go-plugin
func GetPluginMap() map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		"kilometers": &KilometersPluginGRPC{},
	}
}

// CreatePluginLogger creates a logger for go-plugin
func CreatePluginLogger(debug bool) hclog.Logger {
	level := hclog.Error
	output := io.Discard

	if debug {
		level = hclog.Debug
		output = os.Stderr
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:   "km-plugin",
		Level:  level,
		Output: output,
	})
}

// KilometersPluginGRPC implements the go-plugin Plugin interface for GRPC
type KilometersPluginGRPC struct {
	plugin.NetRPCUnsupportedPlugin
}

// GRPCServer returns a GRPC server implementation for the plugin
func (p *KilometersPluginGRPC) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// This would be implemented by the plugin binary itself
	// The CLI only needs the client side
	return nil
}

// GRPCClient returns a GRPC client implementation for the plugin
func (p *KilometersPluginGRPC) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// Return the GRPC client that implements the plugin interface
	return &PluginGRPCClient{
		client: pb.NewKilometersPluginClient(c),
	}, nil
}

// PluginGRPCClient implements the KilometersPlugin interface over GRPC
type PluginGRPCClient struct {
	client pb.KilometersPluginClient
}

// Name returns the plugin name via GRPC
func (c *PluginGRPCClient) Name() string {
	resp, err := c.client.GetName(context.Background(), &pb.Empty{})
	if err != nil {
		return "unknown"
	}
	return resp.Value
}

// Version returns the plugin version via GRPC
func (c *PluginGRPCClient) Version() string {
	resp, err := c.client.GetVersion(context.Background(), &pb.Empty{})
	if err != nil {
		return "unknown"
	}
	return resp.Value
}

// RequiredTier returns the required subscription tier via GRPC
func (c *PluginGRPCClient) RequiredTier() string {
	resp, err := c.client.GetRequiredTier(context.Background(), &pb.Empty{})
	if err != nil {
		return "Free"
	}
	return resp.Value
}

// Authenticate performs plugin authentication via GRPC
func (c *PluginGRPCClient) Authenticate(ctx context.Context, apiKey string) (*auth.PluginAuthResponse, error) {
	req := &pb.AuthenticateRequest{
		ApiKey: apiKey,
	}

	resp, err := c.client.Authenticate(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, NewPluginError("authentication failed: " + resp.Error)
	}

	authResp := resp.AuthResponse
	return &auth.PluginAuthResponse{
		Authorized: authResp.Authorized,
		UserTier:   authResp.UserTier,
		Features:   authResp.Features,
		ExpiresAt:  &authResp.ExpiresAt,
	}, nil
}

// Initialize initializes the plugin via GRPC
func (c *PluginGRPCClient) Initialize(ctx context.Context, config PluginConfig) error {
	req := &pb.InitializeRequest{
		Config: &pb.PluginConfig{
			ApiEndpoint: config.ApiEndpoint,
			Debug:       config.Debug,
			ApiKey:      config.ApiKey,
		},
	}

	resp, err := c.client.Initialize(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return NewPluginError("initialization failed: " + resp.Error)
	}

	return nil
}

// Shutdown shuts down the plugin via GRPC
func (c *PluginGRPCClient) Shutdown(ctx context.Context) error {
	req := &pb.ShutdownRequest{}

	resp, err := c.client.Shutdown(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return NewPluginError("shutdown failed: " + resp.Error)
	}

	return nil
}

// HandleMessage handles a message via GRPC
func (c *PluginGRPCClient) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	req := &pb.HandleMessageRequest{
		Data:          data,
		Direction:     direction,
		CorrelationId: correlationID,
	}

	resp, err := c.client.HandleMessage(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return NewPluginError("handle message failed: " + resp.Error)
	}

	return nil
}

// HandleError handles an error via GRPC
func (c *PluginGRPCClient) HandleError(ctx context.Context, err error) error {
	req := &pb.HandleErrorRequest{
		ErrorMessage: err.Error(),
	}

	resp, err := c.client.HandleError(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return NewPluginError("handle error failed: " + resp.Error)
	}

	return nil
}

// HandleStreamEvent handles a stream event via GRPC
func (c *PluginGRPCClient) HandleStreamEvent(ctx context.Context, event PluginStreamEvent) error {
	req := &pb.HandleStreamEventRequest{
		Event: &pb.StreamEvent{
			Type:      string(event.Type),
			Timestamp: timestamppb.New(event.Timestamp),
			Data:      event.Data,
		},
	}

	resp, err := c.client.HandleStreamEvent(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return NewPluginError("handle stream event failed: " + resp.Error)
	}

	return nil
}

// SDKClientAdapter adapts PluginGRPCClient to the SDK Plugin interface
type SDKClientAdapter struct {
	client      *PluginGRPCClient
	apiEndpoint string
	apiKey      string
}

// NewSDKClientAdapter creates a new SDK client adapter
func NewSDKClientAdapter(client *PluginGRPCClient, apiEndpoint string, apiKey string) *SDKClientAdapter {
	return &SDKClientAdapter{
		client:      client,
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
	}
}

// GetInfo returns plugin information (SDK interface)
func (a *SDKClientAdapter) GetInfo() kmsdk.PluginInfo {
	return kmsdk.PluginInfo{
		Name:    a.client.Name(),
		Version: a.client.Version(),
	}
}

// Initialize initializes the plugin (SDK interface)
func (a *SDKClientAdapter) Initialize(ctx context.Context, config kmsdk.Config) error {
	return a.client.Initialize(ctx, PluginConfig{
		ApiEndpoint: config.ApiEndpoint,
		Debug:       config.Debug,
		ApiKey:      a.apiKey,
	})
}

// HandleMessage handles a message (SDK interface)
func (a *SDKClientAdapter) HandleMessage(ctx context.Context, data []byte, direction kmsdk.Direction, correlationID string) error {
	return a.client.HandleMessage(ctx, data, string(direction), correlationID)
}

// HandleError handles an error (SDK interface)
func (a *SDKClientAdapter) HandleError(ctx context.Context, err error) {
	// SDK interface doesn't return error for HandleError
	_ = a.client.HandleError(ctx, err)
}

// HandleStreamEvent handles a stream event (SDK interface)
func (a *SDKClientAdapter) HandleStreamEvent(ctx context.Context, event kmsdk.StreamEvent) {
	// SDK interface doesn't return error for HandleStreamEvent
	_ = a.client.HandleStreamEvent(ctx, PluginStreamEvent{
		Type:      PluginStreamEventType(event.Type),
		Timestamp: event.Timestamp,
		Data:      map[string]string{"message": event.Message},
	})
}

// Authenticate performs plugin authentication (SDK interface)
func (a *SDKClientAdapter) Authenticate(ctx context.Context, apiKey string) error {
	_, err := a.client.Authenticate(ctx, apiKey)
	return err
}

// Shutdown shuts down the plugin (SDK interface)
func (a *SDKClientAdapter) Shutdown(ctx context.Context) error {
	return a.client.Shutdown(ctx)
}
