package plugins

import (
	"context"
	"encoding/gob"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	// "github.com/hashicorp/go-plugin"
)

func init() {
	// Register types that will be passed via gob encoding
	gob.RegisterName("map[string]interface {}", map[string]interface{}{})
	gob.RegisterName("[]interface {}", []interface{}{})
}

type KmPlugin interface {
	// OnRequest is called when an MCP request is received
	OnRequest(ctx context.Context, request *MCPRequest) (*MCPResponse, error)

	// OnResponse is called when an MCP response is received
	OnResponse(ctx context.Context, response *MCPResponse) error

	// GetName returns the plugin name
	GetName() string

	// RequiresPremium returns true if this plugin requires premium subscription
	RequiresPremium() bool

	// Shutdown is called when the plugin is shutting down
	Shutdown()
}

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`

	// Action tells the proxy what to do
	Action string `json:"-"` // "allow", "block", "modify"
	Reason string `json:"-"` // Human-readable reason
}

type KmPluginRPC struct {
	client *rpc.Client
}

func (p *KmPluginRPC) OnRequest(ctx context.Context, request *MCPRequest) (*MCPResponse, error) {
	var response MCPResponse
	err := p.client.Call("Plugin.OnRequest", request, &response)
	return &response, err
}

func (p *KmPluginRPC) OnResponse(ctx context.Context, response *MCPResponse) error {
	var ack bool
	return p.client.Call("Plugin.OnResponse", response, &ack)
}

func (p *KmPluginRPC) GetName() string {
	var name string
	p.client.Call("Plugin.GetName", new(interface{}), &name)
	return name
}

func (p *KmPluginRPC) RequiresPremium() bool {
	var premium bool
	p.client.Call("Plugin.RequiresPremium", new(interface{}), &premium)
	return premium
}

func (p *KmPluginRPC) Shutdown() {
	var ack bool
	p.client.Call("Plugin.Shutdown", new(interface{}), &ack)
}

// MCPPluginRPCServer is the RPC server
type KmPluginRPCServer struct {
	Impl KmPlugin
}

func (s *KmPluginRPCServer) OnRequest(request *MCPRequest, response *MCPResponse) error {
	ctx := context.Background()
	resp, err := s.Impl.OnRequest(ctx, request)
	if err != nil {
		return err
	}
	*response = *resp
	return nil
}

func (s *KmPluginRPCServer) OnResponse(response *MCPResponse, ack *bool) error {
	ctx := context.Background()
	err := s.Impl.OnResponse(ctx, response)
	*ack = err == nil
	return err
}

func (s *KmPluginRPCServer) GetName(args interface{}, name *string) error {
	*name = s.Impl.GetName()
	return nil
}

func (s *KmPluginRPCServer) RequiresPremium(args interface{}, premium *bool) error {
	*premium = s.Impl.RequiresPremium()
	return nil
}

func (s *KmPluginRPCServer) Shutdown(args interface{}, ack *bool) error {
	s.Impl.Shutdown()
	*ack = true
	return nil
}

// Plugin configuration for go-plugin
type KmPluginImpl struct {
	plugin.Plugin
	Impl KmPlugin
}

func (p *KmPluginImpl) Server(*plugin.MuxBroker) (interface{}, error) {
	return &KmPluginRPCServer{Impl: p.Impl}, nil
}

func (p *KmPluginImpl) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &KmPluginRPC{client: c}, nil
}

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "KM_PLUGIN",
	MagicCookieValue: "kilometers_mcp_plugin",
}

var PluginMap = map[string]plugin.Plugin{
	"mcp": &KmPluginImpl{},
}
