package plugins

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

type PluginManager struct {
	plugins    []KmPlugin
	pluginsDir string
	config     *config.Config
	clients    []*plugin.Client
}

func NewPluginManager(pluginsDir string, config *config.Config) (*PluginManager, error) {
	m := &PluginManager{
		pluginsDir: pluginsDir,
		config:     config,
		clients:    make([]*plugin.Client, 0),
		plugins:    make([]KmPlugin, 0),
	}

	return m, nil
}

func (m *PluginManager) LoadPlugins(ctx context.Context) error {
	pluginFiles, err := filepath.Glob(filepath.Join(m.pluginsDir, "km-plugin-*"))
	if err != nil {
		return fmt.Errorf("failed to find plugins: %w", err)
	}

	for _, pluginFile := range pluginFiles {
		if err := m.loadPlugin(ctx, pluginFile); err != nil {
			log.Printf("Failed to load plugin %s: %v", pluginFile, err)
			continue
		}
	}

	fmt.Printf("Loaded %d plugins\n", len(m.plugins))
	return nil
}

func (m *PluginManager) loadPlugin(ctx context.Context, pluginPath string) error {
	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(pluginPath),
	})

	// Connect to plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Get plugin interface
	raw, err := rpcClient.Dispense("mcp")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	mcpPlugin := raw.(KmPlugin)

	// Check if premium plugin requires subscription
	if mcpPlugin.RequiresPremium() && m.config.SubscriptionTier == "free" {
		log.Printf("Skipping premium plugin %s (subscription required)", mcpPlugin.GetName())
		client.Kill()
		return nil
	}

	m.clients = append(m.clients, client)
	m.plugins = append(m.plugins, mcpPlugin)

	fmt.Printf("Loaded plugin: %s\n", mcpPlugin.GetName())
	return nil
}

func (m *PluginManager) ProcessRequest(ctx context.Context, request *MCPRequest) (*MCPResponse, error) {
	// Process through all loaded plugins
	for _, plugin := range m.plugins {
		response, err := plugin.OnRequest(ctx, request)
		if err != nil {
			log.Printf("Plugin %s error: %v", plugin.GetName(), err)
			continue
		}

		// If any plugin blocks the request, return immediately
		if response.Action == "block" {
			return response, nil
		}

		// If plugin modifies the request, update it
		if response.Action == "modify" {
			// Update request with modified data
			if response.Result != nil {
				request.Params = response.Result
			}
		}
	}

	// Default: allow the request
	return &MCPResponse{
		ID:     request.ID,
		Action: "allow",
	}, nil
}

func (m *PluginManager) ProcessResponse(ctx context.Context, response *MCPResponse) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnResponse(ctx, response); err != nil {
			log.Printf("Plugin %s response error: %v", plugin.GetName(), err)
		}
	}
	return nil
}

func (m *PluginManager) Shutdown() {
	for _, client := range m.clients {
		client.Kill()
	}
}
