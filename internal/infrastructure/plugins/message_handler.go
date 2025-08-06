package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	pluginPorts "github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// PluginMessageHandler implements the MessageHandler interface using the PluginManager
// This bridges the current monitoring system with the new go-plugins architecture
type PluginMessageHandler struct {
	pluginManager *PluginManager
	correlationID string
	initialized   bool
}

// NewPluginMessageHandler creates a new plugin-based message handler
func NewPluginMessageHandler(pluginManager *PluginManager) *PluginMessageHandler {
	return &PluginMessageHandler{
		pluginManager: pluginManager,
		initialized:   false,
	}
}

// SetCorrelationID sets the correlation ID for message tracking
// This method is called by the MonitoringService
func (h *PluginMessageHandler) SetCorrelationID(correlationID string) {
	h.correlationID = correlationID
}

// HandleMessage processes a JSON-RPC message through all loaded plugins
func (h *PluginMessageHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	if !h.initialized {
		fmt.Fprintf(os.Stderr, "[PluginHandler] Warning: Handler not properly initialized\n")
		return nil
	}

	// Forward message to all loaded and authenticated plugins
	if h.pluginManager != nil {
		return h.pluginManager.HandleMessage(ctx, data, direction, h.correlationID)
	}

	return nil
}

// HandleError processes an error through all loaded plugins
func (h *PluginMessageHandler) HandleError(ctx context.Context, err error) {
	if !h.initialized {
		return
	}

	// Forward error to all loaded plugins
	if h.pluginManager != nil {
		h.pluginManager.HandleError(ctx, err)
	}
}

// HandleStreamEvent processes a stream event through all loaded plugins
func (h *PluginMessageHandler) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	if !h.initialized {
		return
	}

	// Convert ports.StreamEvent to plugin StreamEvent
	pluginEvent := pluginPorts.StreamEvent{
		Type:      pluginPorts.StreamEventType(event.Type),
		Timestamp: time.Unix(0, event.Timestamp),
		Data:      map[string]string{"message": event.Message},
	}

	// Forward stream event to all loaded plugins
	if h.pluginManager != nil {
		h.pluginManager.HandleStreamEvent(ctx, pluginEvent)
	}
}

// Flush flushes any pending events (for compatibility with API handler)
func (h *PluginMessageHandler) Flush(ctx context.Context) error {
	if !h.initialized {
		return nil
	}

	// Plugins handle their own flushing through their Shutdown methods
	// This is mainly for compatibility with the existing API handler interface
	return nil
}

// Initialize initializes the plugin handler with API key and configuration
func (h *PluginMessageHandler) Initialize(ctx context.Context, apiKey string) error {
	if h.pluginManager == nil {
		return fmt.Errorf("plugin manager not set")
	}

	// Start the plugin manager
	if err := h.pluginManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start plugin manager: %w", err)
	}

	// Discover and load plugins with authentication
	if err := h.pluginManager.DiscoverAndLoadPlugins(ctx, apiKey); err != nil {
		return fmt.Errorf("failed to discover and load plugins: %w", err)
	}

	h.initialized = true

	// Report loaded plugins
	loadedPlugins := h.pluginManager.GetLoadedPlugins()
	if len(loadedPlugins) > 0 {
		fmt.Fprintf(os.Stderr, "[PluginHandler] Loaded %d plugins:\n", len(loadedPlugins))
		for name, plugin := range loadedPlugins {
			fmt.Fprintf(os.Stderr, "  ✓ %s v%s (%s tier)\n",
				name, plugin.Info.Version, plugin.Info.RequiredTier)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[PluginHandler] No plugins loaded (running in basic mode)\n")
	}

	return nil
}

// Shutdown gracefully shuts down the plugin handler and all loaded plugins
func (h *PluginMessageHandler) Shutdown(ctx context.Context) error {
	if !h.initialized || h.pluginManager == nil {
		return nil
	}

	fmt.Fprintf(os.Stderr, "[PluginHandler] Shutting down plugins...\n")

	// Stop the plugin manager (this will unload all plugins)
	if err := h.pluginManager.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop plugin manager: %w", err)
	}

	h.initialized = false
	fmt.Fprintf(os.Stderr, "[PluginHandler] Plugins shut down successfully\n")

	return nil
}

// GetLoadedPluginNames returns the names of currently loaded plugins
func (h *PluginMessageHandler) GetLoadedPluginNames() []string {
	if !h.initialized || h.pluginManager == nil {
		return nil
	}

	loadedPlugins := h.pluginManager.GetLoadedPlugins()
	names := make([]string, 0, len(loadedPlugins))
	for name := range loadedPlugins {
		names = append(names, name)
	}
	return names
}

// IsInitialized returns whether the handler has been properly initialized
func (h *PluginMessageHandler) IsInitialized() bool {
	return h.initialized
}

// PluginManagerFactory creates and configures a PluginManager for the CLI
type PluginManagerFactory struct{}

// NewPluginManagerFactory creates a new plugin manager factory
func NewPluginManagerFactory() *PluginManagerFactory {
	return &PluginManagerFactory{}
}

// CreatePluginManager creates a fully configured plugin manager
func (f *PluginManagerFactory) CreatePluginManager(apiEndpoint string, debug bool) (*PluginManager, error) {
	// Create plugin manager configuration
	config := &PluginManagerConfig{
		PluginDirectories:   []string{"~/.km/plugins/", "/usr/local/share/km/plugins/", "./plugins/"},
		AuthRefreshInterval: 5 * time.Minute,
		ApiEndpoint:         apiEndpoint,
		Debug:               debug,
		MaxPlugins:          10,
		LoadTimeout:         30 * time.Second,
	}

	// Create discovery service
	validator := NewSignaturePluginValidator([]byte("km-public-key")) // TODO: Real public key
	discovery := NewFileSystemPluginDiscovery(config.PluginDirectories, validator)

	// Create authenticator
	authenticator := NewHTTPPluginAuthenticator(apiEndpoint)

	// Create authentication cache
	authCache := NewMemoryAuthenticationCache(5 * time.Minute)

	// Create and return plugin manager
	return NewPluginManager(config, discovery, validator, authenticator, authCache), nil
}

// CreatePluginMessageHandler creates a complete plugin-based message handler
func (f *PluginManagerFactory) CreatePluginMessageHandler(apiEndpoint string, debug bool) (*PluginMessageHandler, error) {
	// Create plugin manager
	pluginManager, err := f.CreatePluginManager(apiEndpoint, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Create and return message handler
	return NewPluginMessageHandler(pluginManager), nil
}
