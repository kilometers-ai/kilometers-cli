package plugins

import (
	"context"
	"time"

	config2 "github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/jsonrpc"
	"github.com/kilometers-ai/kilometers-cli/internal/streaming"
)

// NewPluginManagerFactory creates a new plugin manager factory
func NewPluginManagerFactory() *PluginManagerFactory {
	return &PluginManagerFactory{}
}

// PluginManagerFactory creates plugin managers
type PluginManagerFactory struct{}

// CreatePluginManager creates a plugin manager from configuration
func (f *PluginManagerFactory) CreatePluginManager(config interface{}) (PluginManagerInterface, error) {
	// Extract configuration from UnifiedConfig if provided
	var apiEndpoint string = "http://localhost:5194"
	var hasAPIKey bool
	var debug bool = true
	var pluginDirs []string = StandardPluginDirectories
	
	if unifiedConfig, ok := config.(*config2.UnifiedConfig); ok {
		apiEndpoint = unifiedConfig.APIEndpoint
		hasAPIKey = unifiedConfig.HasAPIKey()
		debug = unifiedConfig.Debug
		if unifiedConfig.PluginsDir != "" {
			pluginDirs = []string{unifiedConfig.PluginsDir}
		}
	}
	
	// Create real plugin manager with proper configuration
	managerConfig := &PluginManagerConfig{
		PluginDirectories:   pluginDirs,
		AuthRefreshInterval: 5 * time.Minute,
		ApiEndpoint:         apiEndpoint,
		Debug:               debug,
		MaxPlugins:          10,
		LoadTimeout:         30 * time.Second,
	}

	// Create discovery sources
	var discovery PluginDiscovery
	
	// Always include filesystem discovery
	fsDiscovery := NewFileSystemPluginDiscovery(managerConfig.PluginDirectories, managerConfig.Debug)
	
	// If API key is available, create composite discovery with API source
	if hasAPIKey {
		apiDiscovery, err := NewAPIPluginDiscovery(pluginDirs[0], managerConfig.Debug)
		if err != nil {
			// Fall back to filesystem-only if API discovery fails
			if managerConfig.Debug {
				println("[PluginManagerFactory] Failed to create API discovery, using filesystem only:", err.Error())
			}
			discovery = fsDiscovery
		} else {
			// Create composite discovery with API taking precedence
			discovery = NewCompositePluginDiscovery([]PluginDiscovery{
				fsDiscovery, // Check filesystem first
				apiDiscovery, // Then check API (takes precedence for same plugin)
			}, managerConfig.Debug)
		}
	} else {
		discovery = fsDiscovery
	}
	
	// Create validator with enhanced verification
	enhancedValidator, err := NewEnhancedPluginValidator(managerConfig.Debug)
	var validator PluginValidator
	if err != nil {
		// Fall back to simple validator
		validator = NewSimplePluginValidator(managerConfig.Debug)
	} else {
		validator = enhancedValidator
	}
	
	authenticator := NewHTTPPluginAuthenticator(managerConfig.ApiEndpoint, managerConfig.Debug)
	authCache := NewInMemoryAuthCache(5 * time.Minute)

	// Create and return real plugin manager
	return NewPluginManager(managerConfig, discovery, validator, authenticator, authCache)
}

// SimplePluginManager is a simplified plugin manager
type SimplePluginManager struct {
	started bool
}

// NewSimplePluginManager creates a simple plugin manager
func NewSimplePluginManager() *SimplePluginManager {
	return &SimplePluginManager{}
}

// Start starts the simple plugin manager
func (sm *SimplePluginManager) Start(ctx context.Context) error {
	sm.started = true
	return nil
}

// Stop stops the simple plugin manager
func (sm *SimplePluginManager) Stop(ctx context.Context) error {
	sm.started = false
	return nil
}

// DiscoverAndLoadPlugins discovers and loads plugins
func (sm *SimplePluginManager) DiscoverAndLoadPlugins(ctx context.Context, apiKey string) error {
	return nil
}

// GetLoadedPlugins returns loaded plugins
func (sm *SimplePluginManager) GetLoadedPlugins() interface{} {
	return make(map[string]*SimplePluginInstance)
}

// HandleMessage handles a message
func (sm *SimplePluginManager) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	return nil
}

// HandleError handles an error
func (sm *SimplePluginManager) HandleError(ctx context.Context, err error) error {
	return nil
}

// HandleStreamEvent handles a stream event
func (sm *SimplePluginManager) HandleStreamEvent(ctx context.Context, event streaming.StreamEvent) error {
	return nil
}

// SimplePluginInstance represents a simple plugin instance
type SimplePluginInstance struct {
	Name         string
	Version      string
	RequiredTier string
	LastAuth     time.Time
	Path         string
}

// PluginMessageHandler handles plugin messages
type PluginMessageHandler struct {
	pluginManager PluginManagerInterface
}

// CreatePluginMessageHandler creates a plugin message handler
func (f *PluginManagerFactory) CreatePluginMessageHandler(config interface{}) (*PluginMessageHandler, error) {
	// Create real plugin manager
	pluginManager, err := f.CreatePluginManager(config)
	if err != nil {
		return nil, err
	}

	// Start the plugin manager
	ctx := context.Background()
	if err := pluginManager.Start(ctx); err != nil {
		return nil, err
	}

	// Try to discover and load plugins if we have config with API key
	if appConfig, ok := config.(*config2.UnifiedConfig); ok && appConfig.HasAPIKey() {
		if err := pluginManager.DiscoverAndLoadPlugins(ctx, appConfig.APIKey); err != nil {
			// Log error but don't fail - plugins are optional
			// TODO: Add proper logging
		}
	}

	return NewPluginMessageHandlerWithManager(pluginManager), nil
}

// NewPluginMessageHandler creates a new plugin message handler
func NewPluginMessageHandler() *PluginMessageHandler {
	return &PluginMessageHandler{}
}

// NewPluginMessageHandlerWithManager creates a new plugin message handler with a plugin manager
func NewPluginMessageHandlerWithManager(pluginManager PluginManagerInterface) *PluginMessageHandler {
	return &PluginMessageHandler{
		pluginManager: pluginManager,
	}
}

// HandleError handles an error (MessageHandler interface)
func (h *PluginMessageHandler) HandleError(ctx context.Context, err error) {
	if h.pluginManager != nil {
		h.pluginManager.HandleError(ctx, err)
	}
}

// HandleMessage handles a message (MessageHandler interface)
func (h *PluginMessageHandler) HandleMessage(ctx context.Context, data []byte, direction jsonrpc.Direction) error {
	if h.pluginManager != nil {
		// Convert direction to string for plugin manager
		directionStr := string(direction)
		correlationID := "msg_" + time.Now().Format("150405000")
		return h.pluginManager.HandleMessage(ctx, data, directionStr, correlationID)
	}
	return nil
}

// HandleStreamEvent handles a stream event (MessageHandler interface)
func (h *PluginMessageHandler) HandleStreamEvent(ctx context.Context, event streaming.StreamEvent) {
	if h.pluginManager != nil {
		h.pluginManager.HandleStreamEvent(ctx, event)
	}
}

// Initialize initializes the handler
func (h *PluginMessageHandler) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// Shutdown shuts down the handler
func (h *PluginMessageHandler) Shutdown(ctx context.Context) error {
	return nil
}
