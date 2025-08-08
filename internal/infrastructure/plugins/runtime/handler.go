package runtime

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/discovery"
)

// PluginManagerInterface defines the common interface for plugin managers
type PluginManagerInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	DiscoverAndLoadPlugins(ctx context.Context, apiKey string) error
	HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error
	HandleError(ctx context.Context, err error) error
	HandleStreamEvent(ctx context.Context, event ports.StreamEvent) error
	GetLoadedPlugins() interface{} // Returns either map[string]*PluginInstance or map[string]*SimplePluginInstance
}

// PluginMessageHandler implements the MessageHandler interface using the PluginManager
// This bridges the current monitoring system with the new go-plugins architecture
type PluginMessageHandler struct {
	pluginManager PluginManagerInterface
	correlationID string
	initialized   bool
}

// NewPluginMessageHandler creates a new plugin-based message handler
func NewPluginMessageHandler(pluginManager PluginManagerInterface) *PluginMessageHandler {
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
		return h.pluginManager.HandleMessage(ctx, data, string(direction), h.correlationID)
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

	// Forward stream event to all loaded plugins
	if h.pluginManager != nil {
		h.pluginManager.HandleStreamEvent(ctx, event)
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
	pluginCount, pluginInfo := h.extractPluginInfo(loadedPlugins)

	if pluginCount > 0 {
		fmt.Fprintf(os.Stderr, "[PluginHandler] Loaded %d plugins:\n", pluginCount)
		for _, info := range pluginInfo {
			fmt.Fprintf(os.Stderr, "  ✓ %s v%s (%s tier)\n",
				info.Name, info.Version, info.RequiredTier)
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

	fmt.Fprintf(os.Stderr, "[PluginHandler] Shutting down ports...\n")

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
	pluginCount, pluginInfo := h.extractPluginInfo(loadedPlugins)

	names := make([]string, 0, pluginCount)
	for _, info := range pluginInfo {
		names = append(names, info.Name)
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

	// Create dependency implementations
	pluginDiscovery := discovery.NewFileSystemPluginDiscovery(config.PluginDirectories, debug)
	validator := discovery.NewBasicPluginValidator(debug)
	authenticator := auth.NewHTTPPluginAuthenticator(apiEndpoint, debug)
	authCache := auth.NewMemoryAuthenticationCache(debug)

	// Create and return real plugin manager
	return NewExternalPluginManager(config, pluginDiscovery, validator, authenticator, authCache), nil
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

// PluginInfo represents common plugin information
type PluginInfo struct {
	Name         string
	Version      string
	RequiredTier string
}

// extractPluginInfo extracts plugin information from different plugin manager types
func (h *PluginMessageHandler) extractPluginInfo(loadedPlugins interface{}) (int, []PluginInfo) {
	switch plugins := loadedPlugins.(type) {
	case map[string]*SimplePluginInstance:
		count := len(plugins)
		info := make([]PluginInfo, 0, count)
		for _, plugin := range plugins {
			info = append(info, PluginInfo{
				Name:         plugin.Name,
				Version:      plugin.Version,
				RequiredTier: plugin.RequiredTier,
			})
		}
		return count, info

	case map[string]*PluginInstance:
		count := len(plugins)
		info := make([]PluginInfo, 0, count)
		for _, plugin := range plugins {
			info = append(info, PluginInfo{
				Name:         plugin.Info.Name,
				Version:      plugin.Info.Version,
				RequiredTier: plugin.Info.RequiredTier,
			})
		}
		return count, info

	default:
		return 0, nil
	}
}
