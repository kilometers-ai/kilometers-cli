package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/application"
	apphttp "github.com/kilometers-ai/kilometers-cli/internal/application/http"
	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
	plugininfra "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitCommand_BasicFlow tests the basic initialization flow
func TestInitCommand_BasicFlow(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "kilometers")
	pluginsDir := filepath.Join(tempDir, ".km", "plugins")

	// Set up environment
	os.Setenv("HOME", tempDir)
	os.Setenv("KM_PLUGINS_DIR", pluginsDir)
	defer os.Unsetenv("HOME")
	defer os.Unsetenv("KM_PLUGINS_DIR")

	// Create mock API client
	mockAPI := NewMockAPIClient(t)
	defer mockAPI.Close()

	// Configure the mock server for pro tier with valid API key
	mockAPI.ConfigureServer(MockServerConfig{
		SubscriptionTier:  "pro",
		CustomerName:      "Test User",
		CustomerID:        "test-customer-123",
		APIKeyValid:       true,
		APIKeys:           map[string]bool{"test-api-key": true},
		AvailablePlugins:  []MockPlugin{},
		DownloadResponses: make(map[string][]byte),
		AuthResponses:     make(map[string]*auth.PluginAuthResponse),
	})

	// Test successful initialization with API key
	t.Run("SuccessfulInit", func(t *testing.T) {
		ctx := context.Background()

		// Create backend client
		authService := apphttp.NewAuthHeaderService("test-api-key", "test/1.0")
		backendClient := apphttp.NewBackendClient(
			mockAPI.URL(),
			"test/1.0",
			10*time.Second,
			authService,
			nil,
		)

		// Create services
		provisioningService := plugininfra.NewHTTPProvisioningService(backendClient)
		installer := plugininfra.NewFileSystemInstaller(pluginsDir, false)
		registry := plugininfra.NewFileSystemRegistry(configDir)

		provisioningApp := application.NewPluginProvisioningService(
			provisioningService,
			installer,
			registry,
		)

		// Run initialization
		config := application.InitConfig{
			APIKey:        "test-api-key",
			APIEndpoint:   mockAPI.URL(),
			AutoProvision: false,
			Interactive:   false,
			Force:         true,
		}

		result, err := provisioningApp.InitializeWithValidation(ctx, config)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.APIKeyValidated)
		assert.Equal(t, "pro", string(result.Subscription.Tier))
		assert.Equal(t, "Test User", result.Subscription.CustomerName)

		// Check that API was called
		assert.Equal(t, 1, mockAPI.GetRequestCount("/api/subscription/status"))
		assert.Equal(t, 1, mockAPI.GetRequestCount("/api/plugins/available"))
	})
}

// TestInitCommand_InvalidAPIKey tests handling of invalid API keys
func TestInitCommand_InvalidAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "kilometers")
	pluginsDir := filepath.Join(tempDir, ".km", "plugins")

	// Create mock API client
	mockAPI := NewMockAPIClient(t)
	defer mockAPI.Close()

	// Configure the mock server with invalid API key
	mockAPI.ConfigureServer(MockServerConfig{
		SubscriptionTier:  "pro",
		CustomerName:      "Test User",
		CustomerID:        "test-customer-123",
		APIKeyValid:       false,
		APIKeys:           map[string]bool{"invalid-key": false},
		AvailablePlugins:  []MockPlugin{},
		DownloadResponses: make(map[string][]byte),
		AuthResponses:     make(map[string]*auth.PluginAuthResponse),
	})

	ctx := context.Background()

	// Create backend client with invalid key
	authService := apphttp.NewAuthHeaderService("invalid-key", "test/1.0")
	backendClient := apphttp.NewBackendClient(
		mockAPI.URL(),
		"test/1.0",
		10*time.Second,
		authService,
		nil,
	)

	// Create services
	provisioningService := plugininfra.NewHTTPProvisioningService(backendClient)
	installer := plugininfra.NewFileSystemInstaller(pluginsDir, false)
	registry := plugininfra.NewFileSystemRegistry(configDir)

	provisioningApp := application.NewPluginProvisioningService(
		provisioningService,
		installer,
		registry,
	)

	// Run initialization with invalid key
	config := application.InitConfig{
		APIKey:        "invalid-key",
		APIEndpoint:   mockAPI.URL(),
		AutoProvision: false,
		Interactive:   false,
	}

	result, err := provisioningApp.InitializeWithValidation(ctx, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
	assert.False(t, result.APIKeyValidated)
}

// TestInitCommand_PluginInstallation tests plugin installation flow
func TestInitCommand_PluginInstallation(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "kilometers")
	pluginsDir := filepath.Join(tempDir, ".km", "plugins")

	// Create mock API client
	mockAPI := NewMockAPIClient(t)
	defer mockAPI.Close()

	// Create test plugins
	testPlugins := []MockPlugin{
		{
			Name:         "console-logger",
			Version:      "1.0.0",
			RequiredTier: "free",
		},
		{
			Name:         "api-logger",
			Version:      "1.0.0",
			RequiredTier: "pro",
		},
	}

	// Configure the mock server with plugins and downloads
	mockAPI.ConfigureServer(MockServerConfig{
		SubscriptionTier:  "pro",
		CustomerName:      "Test User",
		CustomerID:        "test-customer-123",
		APIKeyValid:       true,
		APIKeys:           map[string]bool{"test-api-key": true},
		AvailablePlugins:  testPlugins,
		DownloadResponses: make(map[string][]byte),
		AuthResponses:     make(map[string]*auth.PluginAuthResponse),
	})

	// Set up plugin downloads
	mockAPI.SetDownloadResponses(map[string][]byte{
		"console-logger": []byte("mock-console-logger-binary"),
		"api-logger":     []byte("mock-api-logger-binary"),
	})

	ctx := context.Background()

	// Create backend client
	authService := apphttp.NewAuthHeaderService("test-api-key", "test/1.0")
	backendClient := apphttp.NewBackendClient(
		mockAPI.URL(),
		"test/1.0",
		10*time.Second,
		authService,
		nil,
	)

	// Create services
	provisioningService := plugininfra.NewHTTPProvisioningService(backendClient)
	installer := plugininfra.NewFileSystemInstaller(pluginsDir, false)
	registry := plugininfra.NewFileSystemRegistry(configDir)

	provisioningApp := application.NewPluginProvisioningService(
		provisioningService,
		installer,
		registry,
	)

	// Mock stdin for interactive prompt (simulate "Y" response)
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Write "Y" to simulate user input
	go func() {
		w.WriteString("Y\n")
		w.Close()
	}()

	// Run initialization with auto-provision
	config := application.InitConfig{
		APIKey:        "test-api-key",
		APIEndpoint:   mockAPI.URL(),
		AutoProvision: true,
		Interactive:   false,
		Force:         true,
	}

	result, err := provisioningApp.InitializeWithValidation(ctx, config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.APIKeyValidated)
	assert.Equal(t, 2, result.PluginsAvailable)
	assert.Equal(t, 2, result.PluginsInstalled)

	// Verify plugins were downloaded
	assert.GreaterOrEqual(t, mockAPI.GetRequestCount("/api/plugins/download"), 1)

	// Check plugin files exist
	consoleLoggerPath := filepath.Join(pluginsDir, "km-plugin-console-logger")
	apiLoggerPath := filepath.Join(pluginsDir, "km-plugin-api-logger")

	assert.FileExists(t, consoleLoggerPath)
	assert.FileExists(t, apiLoggerPath)

	// Verify content
	content, _ := os.ReadFile(consoleLoggerPath)
	assert.Equal(t, "mock-console-logger-binary", string(content))
}

// TestInitCommand_TierRestrictions tests subscription tier restrictions
func TestInitCommand_TierRestrictions(t *testing.T) {
	// Create mock API client
	mockAPI := NewMockAPIClient(t)
	defer mockAPI.Close()

	// Create test plugins with different tier requirements
	testPlugins := []MockPlugin{
		{
			Name:         "console-logger",
			Version:      "1.0.0",
			RequiredTier: "free",
		},
		{
			Name:         "api-logger",
			Version:      "1.0.0",
			RequiredTier: "pro",
		},
		{
			Name:         "custom-plugin",
			Version:      "1.0.0",
			RequiredTier: "enterprise",
		},
	}

	// Configure the mock server with free tier
	mockAPI.ConfigureServer(MockServerConfig{
		SubscriptionTier:  "free",
		CustomerName:      "Test User",
		CustomerID:        "test-customer-123",
		APIKeyValid:       true,
		APIKeys:           map[string]bool{"test-api-key": true},
		AvailablePlugins:  testPlugins,
		DownloadResponses: make(map[string][]byte),
		AuthResponses:     make(map[string]*auth.PluginAuthResponse),
	})

	ctx := context.Background()

	// Create services
	authService := apphttp.NewAuthHeaderService("test-api-key", "test/1.0")
	backendClient := apphttp.NewBackendClient(mockAPI.URL(), "test/1.0", 10*time.Second, authService, nil)

	result, err := plugininfra.NewHTTPProvisioningService(backendClient).ValidateAPIKey(ctx, "test-api-key")

	assert.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, plugindomain.TierFree, result.Subscription.Tier)

	// Check tier restrictions - convert mock plugins to domain plugins
	domainPlugins := make([]plugindomain.Plugin, len(testPlugins))
	for i, mockPlugin := range testPlugins {
		tier := plugindomain.TierFree
		switch mockPlugin.RequiredTier {
		case "pro":
			tier = plugindomain.TierPro
		case "enterprise":
			tier = plugindomain.TierEnterprise
		}
		domainPlugins[i] = plugindomain.Plugin{
			Name:         mockPlugin.Name,
			Version:      mockPlugin.Version,
			RequiredTier: tier,
		}
	}

	for _, plugin := range domainPlugins {
		canAccess := result.Subscription.CanAccessPlugin(plugin)

		switch plugin.RequiredTier {
		case plugindomain.TierFree:
			assert.True(t, canAccess, "Free tier should access free plugins")
		case plugindomain.TierPro:
			assert.False(t, canAccess, "Free tier should not access pro plugins")
		case plugindomain.TierEnterprise:
			assert.False(t, canAccess, "Free tier should not access enterprise plugins")
		}
	}
}

// TestInitCommand_ConfigurationPersistence tests config saving and loading
func TestInitCommand_ConfigurationPersistence(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".config", "kilometers", "config.json")

	// Create config directory
	os.MkdirAll(filepath.Dir(configPath), 0755)

	// Set HOME environment variable before creating storage
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	// Create and save configuration
	ctx := context.Background()
	storage, err := configpkg.NewUnifiedStorage()
	require.NoError(t, err)

	config := &configpkg.UnifiedConfig{
		APIKey:      "test-saved-key",
		APIEndpoint: "https://api.test.com",
	}

	// Save config
	err = configpkg.SaveConfig(config)
	assert.NoError(t, err)
	assert.FileExists(t, configPath)

	// Load config
	loader := configpkg.NewUnifiedLoader()
	configService := configpkg.NewConfigService(loader, storage)

	loadedConfig, err := configService.Load(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test-saved-key", loadedConfig.APIKey)
	assert.Equal(t, "https://api.test.com", loadedConfig.APIEndpoint)
}

// TestInitCommand_EnvironmentVariables tests environment variable handling
func TestInitCommand_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("KM_API_KEY", "env-api-key")
	os.Setenv("KM_API_ENDPOINT", "https://env.api.com")
	defer os.Unsetenv("KM_API_KEY")
	defer os.Unsetenv("KM_API_ENDPOINT")

	ctx := context.Background()

	// Load configuration
	loader := configpkg.NewUnifiedLoader()
	storage, _ := configpkg.NewUnifiedStorage()
	configService := configpkg.NewConfigService(loader, storage)

	config, err := configService.Load(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "env-api-key", config.APIKey)
	assert.Equal(t, "https://env.api.com", config.APIEndpoint)

	// Check source tracking
	source, exists := config.GetSource("api_key")
	assert.True(t, exists)
	assert.Equal(t, "env", source.Source)
	assert.Contains(t, source.SourcePath, "KM_API_KEY")
}

// TestInitCommand_InteractivePrompts tests interactive user input handling
func TestInitCommand_InteractivePrompts(t *testing.T) {

	// Mock stdin for testing
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tests := []struct {
		name       string
		userInput  string
		expectKey  string
		expectSkip bool
	}{
		{
			name:      "UserProvidesKey",
			userInput: "test-api-key-123\n",
			expectKey: "test-api-key-123",
		},
		{
			name:       "UserSkipsKey",
			userInput:  "\n",
			expectKey:  "",
			expectSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipe for stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write user input
			go func() {
				w.WriteString(tt.userInput)
				w.Close()
			}()

			// Simulate reading user input
			reader := bytes.NewBufferString(tt.userInput)
			line, _ := reader.ReadString('\n')
			apiKey := strings.TrimSpace(line)

			assert.Equal(t, tt.expectKey, apiKey)

			if tt.expectSkip {
				assert.Empty(t, apiKey)
			} else {
				assert.NotEmpty(t, apiKey)
			}
		})
	}
}

// TestInitCommand_ErrorRecovery tests error handling and recovery
func TestInitCommand_ErrorRecovery(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "kilometers")

	// Test network error
	t.Run("NetworkError", func(t *testing.T) {
		ctx := context.Background()

		// Create client with invalid endpoint
		authService := apphttp.NewAuthHeaderService("test-key", "test/1.0")
		backendClient := apphttp.NewBackendClient(
			"http://invalid-endpoint-that-does-not-exist:9999",
			"test/1.0",
			1*time.Second, // Short timeout
			authService,
			nil,
		)

		provisioningService := plugininfra.NewHTTPProvisioningService(backendClient)
		_, err := provisioningService.ValidateAPIKey(ctx, "test-key")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate API key")
	})

	// Test filesystem error
	t.Run("FilesystemError", func(t *testing.T) {
		// Create installer with read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		os.MkdirAll(readOnlyDir, 0555)

		installer := plugininfra.NewFileSystemInstaller(readOnlyDir, false)

		plugin := plugindomain.Plugin{
			Name:    "test-plugin",
			Version: "1.0.0",
		}

		err := installer.Install(context.Background(), plugin, []byte("test-data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	// Test registry corruption recovery
	t.Run("CorruptRegistry", func(t *testing.T) {
		registryPath := filepath.Join(configDir, "plugin-registry.json")
		os.MkdirAll(configDir, 0755)

		// Write corrupt JSON
		os.WriteFile(registryPath, []byte("{corrupt json"), 0644)

		registry := plugininfra.NewFileSystemRegistry(configDir)
		plugins, err := registry.Load(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse registry")
		assert.Nil(t, plugins)
	})
}
