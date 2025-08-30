package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// LoadPluginResult contains the result of loading a plugin
type LoadPluginResult struct {
	Plugin     kmsdk.KilometersPlugin
	Client     *plugin.Client
	PluginInfo kmsdk.PluginInfo
	PID        int
}

// LoadSinglePlugin loads a plugin binary using go-plugin (extracted from PluginManager.loadPlugin)
// This is the core plugin loading logic that can be reused by both PluginManager and dev commands
func LoadSinglePlugin(ctx context.Context, pluginName, pluginPath string, debug bool, useDelve bool, delveAddr string) (*LoadPluginResult, error) {
	// Validate plugin file exists and is executable
	info, err := os.Stat(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin file check failed: %w", err)
	}

	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("plugin is not executable")
	}

	if debug {
		fmt.Printf("🔌 Loading plugin: %s\n", pluginPath)
		fmt.Printf("   📊 Size: %d bytes\n", info.Size())
	}

	// Create plugin client (same as PluginManager)
	// Always start the real plugin binary directly to preserve the go-plugin handshake.
	// When Delve debugging is requested, callers should attach to the spawned PID.
	if useDelve && debug {
		fmt.Printf("🐞 Delve debugging requested; starting plugin normally and will attach to PID after launch\n")
	}
	cmd := exec.Command(pluginPath)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  GetHandshakeConfig(),
		Plugins:          GetPluginMap(),
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		Logger:           CreatePluginLogger(debug),
	})

	// Connect to plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Get plugin instance
	raw, err := rpcClient.Dispense("kilometers")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	// Cast to KilometersPlugin interface
	kilometersPlugin, ok := raw.(kmsdk.KilometersPlugin)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin does not implement KilometersPlugin interface")
	}

	// Get plugin info
	//TODO, this did not return the plugin info implemented in the plugin, but rather the default kmsdk.PluginInfo
	pluginInfo := kilometersPlugin.GetInfo()

	if debug {
		fmt.Printf("✅ Plugin loaded: %s v%s\n", pluginInfo.Name, pluginInfo.Version)
		fmt.Printf("   🏛️  Tier: %s\n", pluginInfo.RequiredTier)
	}

	pid := 0
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	return &LoadPluginResult{
		Plugin:     kilometersPlugin,
		Client:     client,
		PluginInfo: pluginInfo,
		PID:        pid,
	}, nil
}

// AuthenticateLoadedPlugin authenticates a loaded plugin (extracted authentication logic)
func AuthenticateLoadedPlugin(ctx context.Context, plugin kmsdk.KilometersPlugin, apiKey, pluginName string, debug bool) error {
	return AuthenticateLoadedPluginWithJWT(ctx, plugin, apiKey, pluginName, "", debug)
}

// AuthenticateLoadedPluginWithJWT authenticates a loaded plugin using JWT authentication flow
// This function delegates to the PluginJWTAuthenticator for single responsibility
func AuthenticateLoadedPluginWithJWT(ctx context.Context, plugin kmsdk.KilometersPlugin, apiKey, pluginName, apiEndpoint string, debug bool) error {
	authenticator := NewPluginJWTAuthenticator(apiKey, apiEndpoint, debug)
	return authenticator.AuthenticatePlugin(ctx, plugin, pluginName)
}
