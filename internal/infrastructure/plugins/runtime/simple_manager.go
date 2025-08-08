package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// SimpleExternalPluginManager provides a basic external plugin manager for POC
type SimpleExternalPluginManager struct {
	config  *PluginManagerConfig
	apiKey  string
	debug   bool
	plugins map[string]*SimplePluginInstance
}

// SimplePluginInstance represents a simple plugin instance
type SimplePluginInstance struct {
	Name         string
	Version      string
	RequiredTier string
	Path         string
	LastAuth     time.Time
}

// NewSimpleExternalPluginManager creates a simple external plugin manager
func NewSimpleExternalPluginManager(config *PluginManagerConfig) *SimpleExternalPluginManager {
	return &SimpleExternalPluginManager{
		config:  config,
		plugins: make(map[string]*SimplePluginInstance),
	}
}

// Start initializes the simple plugin manager
func (m *SimpleExternalPluginManager) Start(ctx context.Context) error {
	if m.config.Debug {
		fmt.Println("[SimplePluginManager] Starting plugin manager...")
	}
	return nil
}

// Stop shuts down the simple plugin manager
func (m *SimpleExternalPluginManager) Stop(ctx context.Context) error {
	if m.config.Debug {
		fmt.Println("[SimplePluginManager] Stopping plugin manager...")
	}
	return nil
}

// DiscoverAndLoadPlugins discovers and loads plugins (simplified for POC)
func (m *SimpleExternalPluginManager) DiscoverAndLoadPlugins(ctx context.Context, apiKey string) error {
	m.apiKey = apiKey

	if m.config.Debug {
		fmt.Println("[SimplePluginManager] Discovering ports...")
	}

	// For POC, simulate discovering some plugins
	m.loadSimulatedPlugins()

	if m.config.Debug {
		fmt.Printf("[SimplePluginManager] Loaded %d plugins\n", len(m.plugins))
	}

	return nil
}

// loadSimulatedPlugins loads some simulated plugins for testing
func (m *SimpleExternalPluginManager) loadSimulatedPlugins() {
	// Simulate console logger (Free tier)
	if m.hasAPIKey() || true { // Always available for POC
		m.plugins["console-logger"] = &SimplePluginInstance{
			Name:         "console-logger",
			Version:      "1.0.0",
			RequiredTier: "Free",
			Path:         "~/.km/plugins/km-plugin-console-logger",
			LastAuth:     time.Now(),
		}
	}

	// Simulate API logger (Pro tier)
	if m.hasAPIKey() {
		m.plugins["api-logger"] = &SimplePluginInstance{
			Name:         "api-logger",
			Version:      "1.0.0",
			RequiredTier: "Pro",
			Path:         "~/.km/plugins/km-plugin-api-logger",
			LastAuth:     time.Now(),
		}
	}
}

// hasAPIKey checks if an API key is configured
func (m *SimpleExternalPluginManager) hasAPIKey() bool {
	return m.apiKey != "" || os.Getenv("KM_API_KEY") != ""
}

// GetLoadedPlugins returns information about loaded plugins
func (m *SimpleExternalPluginManager) GetLoadedPlugins() interface{} {
	result := make(map[string]*SimplePluginInstance)
	for name, plugin := range m.plugins {
		result[name] = plugin
	}
	return result
}

// HandleMessage forwards a message to plugins (simplified for POC)
func (m *SimpleExternalPluginManager) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	if m.config.Debug {
		fmt.Printf("[SimplePluginManager] Handling message: direction=%s, size=%d bytes\n", direction, len(data))

		// Try to parse as JSON-RPC to show more realistic plugin behavior
		if jsonMsg := parseJSONRPCMessage(data); jsonMsg != nil {
			fmt.Printf("[SimplePluginManager] JSON-RPC: method=%s, id=%v\n", jsonMsg["method"], jsonMsg["id"])

			// Simulate plugin processing
			for name, plugin := range m.plugins {
				m.simulatePluginProcessing(name, plugin, jsonMsg, direction, correlationID)
			}
		}
	}

	return nil
}

// parseJSONRPCMessage attempts to parse a JSON-RPC message
func parseJSONRPCMessage(data []byte) map[string]interface{} {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	// Check if it looks like JSON-RPC
	if _, hasVersion := msg["jsonrpc"]; hasVersion {
		return msg
	}

	return nil
}

// simulatePluginProcessing simulates realistic plugin processing
func (m *SimpleExternalPluginManager) simulatePluginProcessing(name string, plugin *SimplePluginInstance, msg map[string]interface{}, direction string, correlationID string) {
	if !m.config.Debug {
		return
	}

	switch name {
	case "console-logger":
		// Simulate console logging behavior
		fmt.Printf("[Plugin:%s] Logging to console: %s %v\n", name, direction, msg["method"])

	case "api-logger":
		// Simulate API logging behavior (only if we have API key)
		if m.hasAPIKey() {
			fmt.Printf("[Plugin:%s] Sending to API: correlation=%s, method=%v\n", name, correlationID, msg["method"])
		} else {
			fmt.Printf("[Plugin:%s] Skipping API log (no API key)\n", name)
		}
	}
}

// HandleError forwards an error to plugins (simplified for POC)
func (m *SimpleExternalPluginManager) HandleError(ctx context.Context, err error) error {
	if m.config.Debug {
		fmt.Printf("[SimplePluginManager] Handling error: %v\n", err)
	}

	// For POC, just log the error
	// In a real implementation, this would forward to actual plugin processes

	return nil
}

// HandleStreamEvent forwards a stream event to plugins (simplified for POC)
func (m *SimpleExternalPluginManager) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) error {
	if m.config.Debug {
		fmt.Printf("[SimplePluginManager] Handling stream event: type=%s\n", event.Type)
	}

	// For POC, just log the event
	// In a real implementation, this would forward to actual plugin processes

	return nil
}
