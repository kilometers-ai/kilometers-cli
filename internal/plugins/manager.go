package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	"github.com/kilometers-ai/kilometers-cli/internal/http"
	"github.com/kilometers-ai/kilometers-cli/internal/streaming"
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

// PluginManager manages external go-plugins binaries with optional API capabilities
type PluginManager struct {
	config        *PluginManagerConfig
	discovery     PluginDiscovery
	validator     PluginValidator
	authenticator PluginAuthenticator
	authCache     AuthenticationCache

	// API capabilities (optional - nil when API not available)
	apiClient  *CachedPluginApiClient
	downloader *SecurePluginDownloader
	manifest   *http.PluginManifestResponse
	manifestMu sync.RWMutex
	lastFetch  time.Time

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
	Info     PluginInfo
	Plugin   KilometersPlugin
	Client   *plugin.Client
	LastAuth time.Time
	SDK      SDKPlugin
}

// PluginUpdateInfo contains information about an available plugin update
type PluginUpdateInfo struct {
	Name           string
	CurrentVersion string
	NewVersion     string
	Tier           string
}

// NewPluginManager creates a new plugin manager with optional API capabilities
func NewPluginManager(
	config *PluginManagerConfig,
	discovery PluginDiscovery,
	validator PluginValidator,
	authenticator PluginAuthenticator,
	authCache AuthenticationCache,
) (*PluginManager, error) {
	pm := &PluginManager{
		config:        config,
		discovery:     discovery,
		validator:     validator,
		authenticator: authenticator,
		authCache:     authCache,
		loadedPlugins: make(map[string]*PluginInstance),
		clients:       make(map[string]*plugin.Client),
		shutdown:      make(chan struct{}),
	}

	// Try to initialize API capabilities if API endpoint is configured
	if config.ApiEndpoint != "" && len(config.PluginDirectories) > 0 {
		pluginsDir := expandPath(config.PluginDirectories[0])

		// Try to create secure downloader
		downloader, err := NewSecurePluginDownloader(pluginsDir, config.Debug)
		if err != nil {
			// Downloader is optional, continue without it
			if config.Debug {
				fmt.Printf("[PluginManager] Warning: Could not create downloader: %v\n", err)
			}
		} else {
			pm.downloader = downloader
		}

		// Try to create cached API client
		// Note: This will only work if API key is configured in the environment
		// For testing, pass a mock API client instead
		cacheDir := filepath.Join(pluginsDir, ".cache")
		cacheTTL := 5 * time.Minute
		cachedClient, err := NewCachedPluginApiClient(cacheDir, cacheTTL, config.Debug)
		if err != nil {
			// API client is optional
			if config.Debug {
				fmt.Printf("[PluginManager] Warning: Could not create API client: %v\n", err)
			}
		} else {
			pm.apiClient = cachedClient
		}
	}

	return pm, nil
}

// SetAPIClient sets a custom API client (mainly for testing)
func (pm *PluginManager) SetAPIClient(client *CachedPluginApiClient) {
	pm.apiClient = client
}

// SetDownloader sets a custom downloader (mainly for testing)
func (pm *PluginManager) SetDownloader(downloader *SecurePluginDownloader) {
	pm.downloader = downloader
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
	// First, refresh manifest from API if available
	if pm.apiClient != nil {
		if err := pm.refreshManifest(ctx); err != nil {
			if pm.config.Debug {
				fmt.Printf("[PluginManager] Warning: Could not refresh manifest: %v\n", err)
			}
		}
	}

	// Discover available plugins (both local and API)
	discoveredPlugins, err := pm.discovery.DiscoverPlugins(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Discovered %d plugins\n", len(discoveredPlugins))
	}

	// Process each discovered plugin
	for _, pluginInfo := range discoveredPlugins {
		// Check if this is an API plugin that needs downloading
		if pm.needsDownload(pluginInfo) {
			if err := pm.downloadAndLoadPlugin(ctx, pluginInfo, apiKey); err != nil {
				if pm.config.Debug {
					fmt.Printf("[PluginManager] Failed to download plugin %s: %v\n",
						pluginInfo.Name, err)
				}
				continue
			}
		} else {
			// Load local plugin normally
			if err := pm.loadPlugin(ctx, pluginInfo, apiKey); err != nil {
				if pm.config.Debug {
					fmt.Printf("[PluginManager] Failed to load plugin %s: %v\n",
						pluginInfo.Name, err)
				}
				continue
			}
		}
	}

	return nil
}

// loadPlugin loads and authenticates a single plugin
func (pm *PluginManager) loadPlugin(ctx context.Context, info PluginInfo, apiKey string) error {
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
		HandshakeConfig:  GetHandshakeConfig(),
		Plugins:          GetPluginMap(),
		Cmd:              exec.Command(info.Path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           CreatePluginLogger(pm.config.Debug),
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

	kilometersPlugin, ok := raw.(KilometersPlugin)
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
	config := PluginConfig{
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
	if clientImpl, ok := raw.(*PluginGRPCClient); ok {
		instance.SDK = NewSDKClientAdapter(clientImpl, pm.config.ApiEndpoint, apiKey)
	}

	pm.loadedPlugins[info.Name] = instance
	pm.clients[info.Name] = client

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Successfully loaded plugin: %s v%s\n", info.Name, info.Version)
	}

	return nil
}

// API-related methods (only work when API client is available)

// refreshManifest fetches the latest plugin manifest from the API
func (pm *PluginManager) refreshManifest(ctx context.Context) error {
	if pm.apiClient == nil {
		return fmt.Errorf("API client not available")
	}

	pm.manifestMu.Lock()
	defer pm.manifestMu.Unlock()

	// Check if we've fetched recently (cache for 5 minutes)
	if time.Since(pm.lastFetch) < 5*time.Minute && pm.manifest != nil {
		return nil
	}

	manifest, err := pm.apiClient.GetPluginManifest(ctx)
	if err != nil {
		return err
	}

	pm.manifest = manifest
	pm.lastFetch = time.Now()

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Refreshed manifest with %d plugins\n", len(manifest.Plugins))
	}

	return nil
}

// needsDownload checks if a plugin needs to be downloaded from the API
func (pm *PluginManager) needsDownload(info PluginInfo) bool {
	// Check if the path indicates an API plugin
	if strings.HasPrefix(info.Path, "api://") {
		return true
	}

	// Check if local file exists
	if info.Path != "" {
		if _, err := os.Stat(info.Path); err == nil {
			return false // Local file exists
		}
	}

	// Check if this plugin is in the API manifest
	pm.manifestMu.RLock()
	defer pm.manifestMu.RUnlock()

	if pm.manifest != nil {
		for _, entry := range pm.manifest.Plugins {
			if entry.Name == info.Name {
				// Plugin is available from API and not installed locally
				return true
			}
		}
	}

	return false
}

// downloadAndLoadPlugin downloads a plugin from the API and loads it
func (pm *PluginManager) downloadAndLoadPlugin(ctx context.Context, info PluginInfo, apiKey string) error {
	if pm.downloader == nil {
		return fmt.Errorf("plugin downloader not available")
	}

	// Find plugin in manifest
	pm.manifestMu.RLock()
	var pluginEntry *http.PluginManifestEntry
	if pm.manifest != nil {
		for _, entry := range pm.manifest.Plugins {
			if entry.Name == info.Name {
				e := entry // Create a copy
				pluginEntry = &e
				break
			}
		}
	}
	pm.manifestMu.RUnlock()

	if pluginEntry == nil {
		return fmt.Errorf("plugin %s not found in manifest", info.Name)
	}

	// Download and verify plugin
	result, err := pm.downloader.DownloadAndVerifyPlugin(ctx, pluginEntry)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Update plugin info with local path
	info.Path = result.LocalPath

	// Load the downloaded plugin
	return pm.loadPlugin(ctx, info, apiKey)
}

// ListAvailablePlugins returns all plugins available from the API
func (pm *PluginManager) ListAvailablePlugins(ctx context.Context) ([]http.PluginManifestEntry, error) {
	if pm.apiClient == nil {
		return nil, fmt.Errorf("API client not available")
	}

	// Refresh manifest
	if err := pm.refreshManifest(ctx); err != nil {
		return nil, err
	}

	pm.manifestMu.RLock()
	defer pm.manifestMu.RUnlock()

	if pm.manifest == nil {
		return nil, fmt.Errorf("no manifest available")
	}

	return pm.manifest.Plugins, nil
}

// InstallPlugin installs a new plugin from the API
func (pm *PluginManager) InstallPlugin(ctx context.Context, pluginName string, apiKey string) error {
	if pm.downloader == nil {
		return fmt.Errorf("plugin downloader not available")
	}

	// Check if already installed
	if installed, _ := pm.IsPluginInstalled(pluginName); installed {
		if pm.config.Debug {
			fmt.Printf("[PluginManager] Plugin %s is already installed\n", pluginName)
		}
		// Try to load it if not already loaded
		pm.mutex.RLock()
		_, loaded := pm.loadedPlugins[pluginName]
		pm.mutex.RUnlock()

		if !loaded {
			// Create plugin info and load
			localPath := pm.getLocalPluginPath(pluginName)
			info := PluginInfo{
				Name: pluginName,
				Path: localPath,
			}
			return pm.loadPlugin(ctx, info, apiKey)
		}

		return nil
	}

	// Refresh manifest to get latest plugin info
	if err := pm.refreshManifest(ctx); err != nil {
		return fmt.Errorf("failed to fetch plugin manifest: %w", err)
	}

	// Find plugin in manifest
	pm.manifestMu.RLock()
	var pluginEntry *http.PluginManifestEntry
	if pm.manifest != nil {
		for _, entry := range pm.manifest.Plugins {
			if entry.Name == pluginName {
				e := entry
				pluginEntry = &e
				break
			}
		}
	}
	pm.manifestMu.RUnlock()

	if pluginEntry == nil {
		return fmt.Errorf("plugin %s not found in available plugins", pluginName)
	}

	// Download and verify plugin
	result, err := pm.downloader.DownloadAndVerifyPlugin(ctx, pluginEntry)
	if err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	// Create plugin info for loading
	info := PluginInfo{
		Name:         pluginEntry.Name,
		Version:      pluginEntry.Version,
		Path:         result.LocalPath,
		RequiredTier: pluginEntry.Tier,
		Signature:    []byte(pluginEntry.Signature),
	}

	// Load the installed plugin
	return pm.loadPlugin(ctx, info, apiKey)
}

// UpdatePlugin updates a specific plugin to the latest version
func (pm *PluginManager) UpdatePlugin(ctx context.Context, pluginName string, apiKey string) error {
	if pm.downloader == nil {
		return fmt.Errorf("plugin downloader not available")
	}

	// Find plugin in manifest
	pm.manifestMu.RLock()
	var pluginEntry *http.PluginManifestEntry
	if pm.manifest != nil {
		for _, entry := range pm.manifest.Plugins {
			if entry.Name == pluginName {
				e := entry
				pluginEntry = &e
				break
			}
		}
	}
	pm.manifestMu.RUnlock()

	if pluginEntry == nil {
		return fmt.Errorf("plugin %s not found in manifest", pluginName)
	}

	// Shutdown existing plugin if loaded
	pm.mutex.Lock()
	if instance, exists := pm.loadedPlugins[pluginName]; exists {
		pm.shutdownPlugin(ctx, pluginName, instance)
		delete(pm.loadedPlugins, pluginName)
		delete(pm.clients, pluginName)
	}
	pm.mutex.Unlock()

	// Download and verify updated plugin
	result, err := pm.downloader.DownloadAndVerifyPlugin(ctx, pluginEntry)
	if err != nil {
		return fmt.Errorf("failed to download plugin update: %w", err)
	}

	// Create plugin info for loading
	info := PluginInfo{
		Name:         pluginEntry.Name,
		Version:      pluginEntry.Version,
		Path:         result.LocalPath,
		RequiredTier: pluginEntry.Tier,
		Signature:    []byte(pluginEntry.Signature),
	}

	// Load the updated plugin
	return pm.loadPlugin(ctx, info, apiKey)
}

// UninstallPlugin removes a plugin from the system
func (pm *PluginManager) UninstallPlugin(ctx context.Context, pluginName string) error {
	// Shutdown and unload plugin if it's running
	pm.mutex.Lock()
	if instance, exists := pm.loadedPlugins[pluginName]; exists {
		pm.shutdownPlugin(ctx, pluginName, instance)
		delete(pm.loadedPlugins, pluginName)
		delete(pm.clients, pluginName)
	}
	pm.mutex.Unlock()

	// Remove plugin files
	return pm.removePluginFiles(pluginName)
}

// CheckForUpdates checks if any loaded plugins have updates available
func (pm *PluginManager) CheckForUpdates(ctx context.Context) ([]PluginUpdateInfo, error) {
	if pm.apiClient == nil {
		return nil, fmt.Errorf("API client not available")
	}

	// Refresh manifest
	if err := pm.refreshManifest(ctx); err != nil {
		return nil, err
	}

	pm.mutex.RLock()
	loadedPlugins := make(map[string]*PluginInstance)
	for name, instance := range pm.loadedPlugins {
		loadedPlugins[name] = instance
	}
	pm.mutex.RUnlock()

	pm.manifestMu.RLock()
	defer pm.manifestMu.RUnlock()

	var updates []PluginUpdateInfo

	if pm.manifest != nil {
		for _, entry := range pm.manifest.Plugins {
			if instance, loaded := loadedPlugins[entry.Name]; loaded {
				// Compare versions
				if entry.Version > instance.Info.Version {
					updates = append(updates, PluginUpdateInfo{
						Name:           entry.Name,
						CurrentVersion: instance.Info.Version,
						NewVersion:     entry.Version,
						Tier:           entry.Tier,
					})
				}
			}
		}
	}

	return updates, nil
}

// IsPluginInstalled checks if a plugin is installed locally
func (pm *PluginManager) IsPluginInstalled(pluginName string) (bool, string) {
	localPath := pm.getLocalPluginPath(pluginName)

	info, err := os.Stat(localPath)
	if err != nil {
		return false, ""
	}

	// Check if it's a regular file and executable
	if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
		return true, localPath
	}

	return false, ""
}

// Helper methods

// getLocalPluginPath returns the local path for a plugin
func (pm *PluginManager) getLocalPluginPath(pluginName string) string {
	pluginsDir := ""
	if len(pm.config.PluginDirectories) > 0 {
		pluginsDir = expandPath(pm.config.PluginDirectories[0])
	}
	return filepath.Join(pluginsDir, fmt.Sprintf("km-plugin-%s", pluginName))
}

// removePluginFiles removes plugin files from the filesystem
func (pm *PluginManager) removePluginFiles(pluginName string) error {
	localPath := pm.getLocalPluginPath(pluginName)

	if err := os.Remove(localPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plugin %s is not installed", pluginName)
		}
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	// Also remove any associated files (manifest, signature, etc.)
	manifestPath := localPath + ".manifest.json"
	os.Remove(manifestPath) // Ignore error if doesn't exist

	signaturePath := localPath + ".sig"
	os.Remove(signaturePath) // Ignore error if doesn't exist

	if pm.config.Debug {
		fmt.Printf("[PluginManager] Removed plugin %s from %s\n", pluginName, localPath)
	}

	return nil
}

// authenticatePlugin handles plugin authentication with the API
func (pm *PluginManager) authenticatePlugin(ctx context.Context, plugin KilometersPlugin, apiKey string) (*auth.PluginAuthResponse, error) {
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
func (pm *PluginManager) isPluginAuthorized(authResponse *auth.PluginAuthResponse, requiredTier string) bool {
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
			err = instance.SDK.HandleMessage(ctx, data, SDKDirection(direction), correlationID)
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
func (pm *PluginManager) HandleStreamEvent(ctx context.Context, event streaming.StreamEvent) error {
	pm.mutex.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.loadedPlugins))
	for _, instance := range pm.loadedPlugins {
		plugins = append(plugins, instance)
	}
	pm.mutex.RUnlock()

	// Convert to PluginStreamEvent and forward (prefer SDK if available)
	pluginEvent := PluginStreamEvent{
		Type:      PluginStreamEventType(event.Type),
		Timestamp: time.Unix(0, event.Timestamp/1e9), // Convert nanoseconds to seconds
		Data:      map[string]string{"message": event.Message},
	}

	for _, instance := range plugins {
		var fwdErr error
		if instance.SDK != nil {
			instance.SDK.HandleStreamEvent(ctx, SDKStreamEvent{Type: string(event.Type), Timestamp: time.Unix(0, event.Timestamp/1e9), Message: event.Message})
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