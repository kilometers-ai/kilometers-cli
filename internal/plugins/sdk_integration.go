package plugins

import (
	"io"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// GetHandshakeConfig returns the handshake configuration from SDK
func GetHandshakeConfig() plugin.HandshakeConfig {
	return kmsdk.HandshakeConfig
}

// GetPluginMap returns the plugin map from SDK for the client side
func GetPluginMap() map[string]plugin.Plugin {
	// For the client side, we don't provide an implementation
	return kmsdk.PluginMap(nil)
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