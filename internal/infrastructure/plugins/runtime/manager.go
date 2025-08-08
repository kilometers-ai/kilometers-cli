package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugins/grpc"
    kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// PluginManagerConfig configures the external plugin manager
type PluginManagerConfig struct {
	PluginDirectories   []string
	AuthRefreshInterval time.Duration
	ApiEndpoint         string
	Debug               bool
	MaxPlugins          int
	LoadTimeout         time.Duration
}

// PluginManager manages external go-plugins binaries
type PluginManager struct {
	config        *PluginManagerConfig
	discovery     ports.PluginDiscovery
	validator     ports.PluginValidator
	authenticator ports.PluginAuthenticator
	authCache     ports.AuthenticationCache

	// Plugin instances
	loadedPlugins map[string]*PluginInstance
	clients       map[string]*plugin.Client
	mutex         sync.RWMutex

	// Lifecycle
	started  bool
	shutdown chan struct{}
}

// PluginInstance represents a loaded plugin
type PluginInstance struct {
	Info     ports.PluginInfo
	Plugin   ports.KilometersPlugin
	Client   *plugin.Client
	LastAuth time.Time
    SDK      kmsdk.Plugin
}

// NewExternalPluginManager creates a new external plugin manager
func NewExternalPluginManager(
	config *PluginManagerConfig,
	discovery ports.PluginDiscovery,
	validator ports.PluginValidator,
	authenticator ports.PluginAuthenticator,
	authCache ports.AuthenticationCache,
) *PluginManager {
	return &PluginManager{
		config:        config,
		discovery:     discovery,
		validator:     validator,
		authenticator: authenticator,
		authCache:     authCache,
		loadedPlugins: make(map[string]*PluginInstance),
		clients:       make(map[string]*plugin.Client),
		shutdown:      make(chan struct{}),
	}
}

// Start initializes the plugin manager
func (pm *PluginManager) Start(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.started {
		return nil
	}

	pm.started = true

	// Start background authentication refresh
	go pm.backgroundAuthRefresh()

	return nil
}

// Stop shuts down the plugin manager and all loaded plugins
func (pm *PluginManager) Stop(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.started {
		return nil
	}

	// Signal shutdown
	close(pm.shutdown)

	// Shutdown all loaded plugins
	for name, instance := range pm.loadedPlugins {
		if err := pm.shutdownPlugin(ctx, name, instance); err != nil {
			// Log error but continue shutting down other plugins
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Error shutting down plugin %s: %v\n", name, err)
			}
		}
	}

	pm.loadedPlugins = make(map[string]*PluginInstance)
	pm.clients = make(map[string]*plugin.Client)
	pm.started = false

	return nil
}

// DiscoverAndLoadPlugins discovers available plugins and loads authorized ones
func (pm *PluginManager) DiscoverAndLoadPlugins(ctx context.Context, apiKey string) error {
	// Discover available plugins
	discoveredPlugins, err := pm.discovery.DiscoverPlugins(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Discovered %d plugins\n", len(discoveredPlugins))
	}

	// Load each discovered plugin
	for _, pluginInfo := range discoveredPlugins {
		if err := pm.loadPlugin(ctx, pluginInfo, apiKey); err != nil {
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Failed to load plugin %s: %v\n", pluginInfo.Name, err)
			}
			// Continue loading other plugins
			continue
		}
	}

	return nil
}

// loadPlugin loads and authenticates a single plugin
func (pm *PluginManager) loadPlugin(ctx context.Context, info ports.PluginInfo, apiKey string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if already loaded
	if _, exists := pm.loadedPlugins[info.Name]; exists {
		return nil
	}

	// Validate plugin binary
	if err := pm.validator.ValidateSignature(ctx, info.Path, info.Signature); err != nil {
		return fmt.Errorf("plugin signature validation failed: %w", err)
	}

	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  grpc.GetHandshakeConfig(),
		Plugins:          grpc.GetPluginMap(),
		Cmd:              exec.Command(info.Path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           grpc.CreatePluginLogger(pm.config.Debug),
	})

	// Connect to plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Get plugin instance
    raw, err := rpcClient.Dispense("kilometers")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

    kilometersPlugin, ok := raw.(ports.KilometersPlugin)
	if !ok {
		client.Kill()
		return fmt.Errorf("plugin does not implement KilometersPlugin interface")
	}

	// Authenticate plugin
	authResponse, err := pm.authenticatePlugin(ctx, kilometersPlugin, apiKey)
	if err != nil {
		client.Kill()
		return fmt.Errorf("plugin authentication failed: %w", err)
	}

	// Verify plugin is authorized for this tier
	if !pm.isPluginAuthorized(authResponse, info.RequiredTier) {
		client.Kill()
		if pm.config.Debug {
			fmt.Printf("[PluginManager] Plugin %s not authorized for current tier\n", info.Name)
		}
		return nil // Not an error, just not authorized
	}

	// Initialize plugin
	config := ports.PluginConfig{
		ApiEndpoint: pm.config.ApiEndpoint,
		Debug:       pm.config.Debug,
		ApiKey:      apiKey,
	}

	if err := kilometersPlugin.Initialize(ctx, config); err != nil {
		client.Kill()
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	// Store loaded plugin
	instance := &PluginInstance{
		Info:     info,
		Plugin:   kilometersPlugin,
		Client:   client,
		LastAuth: time.Now(),
	}

    // Try to create SDK adapter for future SDK-based interactions
    if clientImpl, ok := raw.(*grpc.PluginGRPCClient); ok {
        instance.SDK = grpc.NewSDKClientAdapter(clientImpl, pm.config.ApiEndpoint, apiKey)
    }

	pm.loadedPlugins[info.Name] = instance
	pm.clients[info.Name] = client

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Successfully loaded plugin: %s v%s\n", info.Name, info.Version)
	}

	return nil
}

// authenticatePlugin handles plugin authentication with the API
func (pm *PluginManager) authenticatePlugin(ctx context.Context, plugin ports.KilometersPlugin, apiKey string) (*ports.AuthResponse, error) {
	// Check cache first
	if cachedAuth := pm.authCache.Get(plugin.Name(), apiKey); cachedAuth != nil {
		return cachedAuth, nil
	}

	// Call the plugin's authenticate method directly
	authResponse, err := plugin.Authenticate(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	// Cache authentication result
	pm.authCache.Set(plugin.Name(), apiKey, authResponse)

	return authResponse, nil
}

// isPluginAuthorized checks if plugin is authorized for the given tier
func (pm *PluginManager) isPluginAuthorized(authResponse *ports.AuthResponse, requiredTier string) bool {
	if !authResponse.Authorized {
		return false
	}

	// Check tier authorization
	tierLevels := map[string]int{
		"Free":       0,
		"Pro":        1,
		"Enterprise": 2,
	}

	userLevel, userExists := tierLevels[authResponse.UserTier]
	requiredLevel, requiredExists := tierLevels[requiredTier]

	if !userExists || !requiredExists {
		return false
	}

	return userLevel >= requiredLevel
}

// shutdownPlugin gracefully shuts down a single plugin
func (pm *PluginManager) shutdownPlugin(ctx context.Context, name string, instance *PluginInstance) error {
	// Shutdown plugin
	if err := instance.Plugin.Shutdown(ctx); err != nil {
		// Log but don't fail - we still want to kill the process
		if pm.config.Debug {
			fmt.Printf("[PluginManager] Plugin %s shutdown error: %v\n", name, err)
		}
	}

	// Kill client process
	instance.Client.Kill()

	return nil
}

// backgroundAuthRefresh periodically refreshes plugin authentication
func (pm *PluginManager) backgroundAuthRefresh() {
	ticker := time.NewTicker(pm.config.AuthRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.refreshAuthentication()
		case <-pm.shutdown:
			return
		}
	}
}

// refreshAuthentication refreshes authentication for all loaded plugins
func (pm *PluginManager) refreshAuthentication() {
	pm.mutex.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.loadedPlugins))
	for _, instance := range pm.loadedPlugins {
		plugins = append(plugins, instance)
	}
	pm.mutex.RUnlock()

	// Refresh authentication for each plugin
	for _, instance := range plugins {
		// Check if authentication needs refresh (refresh 1 minute before expiry)
		if time.Since(instance.LastAuth) > pm.config.AuthRefreshInterval-time.Minute {
			// TODO: Get API key from auth manager
			// For now, skip refresh - this will be connected to the auth manager
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Skipping auth refresh for %s (API key needed)\n", instance.Info.Name)
			}
		}
	}
}

// GetLoadedPlugins returns all currently loaded plugins
func (pm *PluginManager) GetLoadedPlugins() interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Return copy to prevent external mutation
	result := make(map[string]*PluginInstance)
	for name, instance := range pm.loadedPlugins {
		result[name] = instance
	}

	return result
}

// HandleMessage forwards a message to all loaded plugins
func (pm *PluginManager) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	pm.mutex.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.loadedPlugins))
	for _, instance := range pm.loadedPlugins {
		plugins = append(plugins, instance)
	}
	pm.mutex.RUnlock()

    // Forward message to all plugins (prefer SDK adapter if available)
    for _, instance := range plugins {
        var err error
        if instance.SDK != nil {
            err = instance.SDK.HandleMessage(ctx, data, kmsdk.Direction(direction), correlationID)
        } else {
            err = instance.Plugin.HandleMessage(ctx, data, direction, correlationID)
        }
        if err != nil {
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Plugin %s message handling error: %v\n", instance.Info.Name, err)
			}
			// Continue processing other plugins
		}
	}

	return nil
}

// HandleError forwards an error to all loaded plugins
func (pm *PluginManager) HandleError(ctx context.Context, err error) error {
	pm.mutex.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.loadedPlugins))
	for _, instance := range pm.loadedPlugins {
		plugins = append(plugins, instance)
	}
	pm.mutex.RUnlock()

    // Forward error to all plugins (prefer SDK adapter if available)
    for _, instance := range plugins {
        var pluginErr error
        if instance.SDK != nil {
            // SDK interface doesn't return error for HandleError
            instance.SDK.HandleError(ctx, err)
            pluginErr = nil
        } else {
            pluginErr = instance.Plugin.HandleError(ctx, err)
        }
        if pluginErr != nil {
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Plugin %s error handling error: %v\n", instance.Info.Name, pluginErr)
			}
			// Continue processing other plugins
		}
	}

	return nil
}

// HandleStreamEvent forwards a stream event to all loaded plugins
func (pm *PluginManager) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) error {
	pm.mutex.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.loadedPlugins))
	for _, instance := range pm.loadedPlugins {
		plugins = append(plugins, instance)
	}
	pm.mutex.RUnlock()

    // Convert to PluginStreamEvent and forward (prefer SDK if available)
    pluginEvent := ports.PluginStreamEvent{
        Type:      ports.PluginStreamEventType(event.Type),
        Timestamp: time.Unix(0, event.Timestamp),
        Data:      map[string]string{"message": event.Message},
    }

    for _, instance := range plugins {
        var fwdErr error
        if instance.SDK != nil {
            instance.SDK.HandleStreamEvent(ctx, kmsdk.StreamEvent{Type: string(event.Type), Timestamp: time.Unix(0, event.Timestamp), Message: event.Message})
            fwdErr = nil
        } else {
            fwdErr = instance.Plugin.HandleStreamEvent(ctx, pluginEvent)
        }
        if fwdErr != nil {
			if pm.config.Debug {
                fmt.Printf("[PluginManager] Plugin %s stream event handling error: %v\n", instance.Info.Name, fwdErr)
			}
			// Continue processing other plugins
		}
	}

	return nil
}
