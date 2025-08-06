package grpc

import (
	"context"
	"io"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
	pb "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/proto/generated"
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

// PluginGRPCClient implements the plugins.KilometersPlugin interface over GRPC
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
func (c *PluginGRPCClient) Authenticate(ctx context.Context, apiKey string) (*plugins.AuthResponse, error) {
	req := &pb.AuthenticateRequest{
		ApiKey: apiKey,
	}

	resp, err := c.client.Authenticate(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, plugins.NewPluginError("authentication failed: " + resp.Error)
	}

	authResp := resp.AuthResponse
	return &plugins.AuthResponse{
		Authorized: authResp.Authorized,
		UserTier:   authResp.UserTier,
		Features:   authResp.Features,
		ExpiresAt:  &authResp.ExpiresAt,
	}, nil
}

// Initialize initializes the plugin via GRPC
func (c *PluginGRPCClient) Initialize(ctx context.Context, config plugins.PluginConfig) error {
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
		return plugins.NewPluginError("initialization failed: " + resp.Error)
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
		return plugins.NewPluginError("shutdown failed: " + resp.Error)
	}

	return nil
}

// HandleMessage forwards a message to the plugin via GRPC
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
		return plugins.NewPluginError("message handling failed: " + resp.Error)
	}

	return nil
}

// HandleError forwards an error to the plugin via GRPC
func (c *PluginGRPCClient) HandleError(ctx context.Context, err error) error {
	req := &pb.HandleErrorRequest{
		ErrorMessage: err.Error(),
		ErrorType:    "general",
	}

	resp, errResp := c.client.HandleError(ctx, req)
	if errResp != nil {
		return errResp
	}

	if !resp.Success {
		return plugins.NewPluginError("error handling failed: " + resp.Error)
	}

	return nil
}

// HandleStreamEvent forwards a stream event to the plugin via GRPC
func (c *PluginGRPCClient) HandleStreamEvent(ctx context.Context, event plugins.StreamEvent) error {
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
		return plugins.NewPluginError("stream event handling failed: " + resp.Error)
	}

	return nil
}
