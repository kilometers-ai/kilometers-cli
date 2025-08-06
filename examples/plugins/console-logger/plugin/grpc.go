package plugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
	pb "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/proto/generated"
)

// KilometersPluginGRPC implements the plugin interface over gRPC
type KilometersPluginGRPC struct {
	// This is the real implementation
	Impl plugins.KilometersPlugin

	plugin.NetRPCUnsupportedPlugin
}

func (p *KilometersPluginGRPC) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// Register the gRPC service implementation
	pb.RegisterKilometersPluginServer(s, &KilometersPluginGRPCServer{Impl: p.Impl})
	return nil
}

func (p *KilometersPluginGRPC) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// This is called by the CLI - not needed for plugin binaries
	return nil, nil
}

// KilometersPluginGRPCServer implements the GRPC server side for plugins
type KilometersPluginGRPCServer struct {
	Impl plugins.KilometersPlugin
	pb.UnimplementedKilometersPluginServer
}

func (s *KilometersPluginGRPCServer) GetName(ctx context.Context, req *pb.Empty) (*pb.StringResponse, error) {
	return &pb.StringResponse{Value: s.Impl.Name()}, nil
}

func (s *KilometersPluginGRPCServer) GetVersion(ctx context.Context, req *pb.Empty) (*pb.StringResponse, error) {
	return &pb.StringResponse{Value: s.Impl.Version()}, nil
}

func (s *KilometersPluginGRPCServer) GetRequiredTier(ctx context.Context, req *pb.Empty) (*pb.StringResponse, error) {
	return &pb.StringResponse{Value: s.Impl.RequiredTier()}, nil
}

func (s *KilometersPluginGRPCServer) Authenticate(ctx context.Context, req *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	authResp, err := s.Impl.Authenticate(ctx, req.ApiKey)
	if err != nil {
		return &pb.AuthenticateResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.AuthenticateResponse{
		Success: true,
		AuthResponse: &pb.AuthResponse{
			Authorized: authResp.Authorized,
			UserTier:   authResp.UserTier,
			Features:   authResp.Features,
			ExpiresAt: func() string {
				if authResp.ExpiresAt != nil {
					return *authResp.ExpiresAt
				}
				return ""
			}(),
		},
	}, nil
}

func (s *KilometersPluginGRPCServer) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
	config := plugins.PluginConfig{
		ApiEndpoint: req.Config.ApiEndpoint,
		Debug:       req.Config.Debug,
		ApiKey:      req.Config.ApiKey,
	}

	err := s.Impl.Initialize(ctx, config)
	if err != nil {
		return &pb.InitializeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.InitializeResponse{Success: true}, nil
}

func (s *KilometersPluginGRPCServer) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	err := s.Impl.Shutdown(ctx)
	if err != nil {
		return &pb.ShutdownResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.ShutdownResponse{Success: true}, nil
}

func (s *KilometersPluginGRPCServer) HandleMessage(ctx context.Context, req *pb.HandleMessageRequest) (*pb.HandleMessageResponse, error) {
	err := s.Impl.HandleMessage(ctx, req.Data, req.Direction, req.CorrelationId)
	if err != nil {
		return &pb.HandleMessageResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.HandleMessageResponse{Success: true}, nil
}

func (s *KilometersPluginGRPCServer) HandleError(ctx context.Context, req *pb.HandleErrorRequest) (*pb.HandleErrorResponse, error) {
	err := s.Impl.HandleError(ctx, plugins.NewPluginError(req.ErrorMessage))
	if err != nil {
		return &pb.HandleErrorResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.HandleErrorResponse{Success: true}, nil
}

func (s *KilometersPluginGRPCServer) HandleStreamEvent(ctx context.Context, req *pb.HandleStreamEventRequest) (*pb.HandleStreamEventResponse, error) {
	event := plugins.StreamEvent{
		Type:      plugins.StreamEventType(req.Event.Type),
		Timestamp: req.Event.Timestamp.AsTime(),
		Data:      req.Event.Data,
	}

	err := s.Impl.HandleStreamEvent(ctx, event)
	if err != nil {
		return &pb.HandleStreamEventResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.HandleStreamEventResponse{Success: true}, nil
}
