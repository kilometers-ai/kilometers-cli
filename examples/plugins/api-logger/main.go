package main

import (
	"github.com/hashicorp/go-plugin"

	kilomePlugin "github.com/kilometers-ai/kilometers-cli/examples/plugins/api-logger/plugin"
)

// handshakeConfig is used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "KILOMETERS_PLUGIN",
	MagicCookieValue: "kilometers_monitoring_plugin",
}

// pluginMap is the map of plugins we can dispense
var pluginMap = map[string]plugin.Plugin{
	"kilometers": &kilomePlugin.KilometersPluginGRPC{Impl: &kilomePlugin.APILoggerPlugin{}},
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
