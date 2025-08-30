package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	plugindomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/interfaces/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginInstallCommand tests the plugin install command workflow
func TestPluginInstallCommand(t *testing.T) {
	// Save original environment
	origHome := os.Getenv("HOME")
	origAPIKey := os.Getenv("KM_API_KEY")
	origEndpoint := os.Getenv("KM_API_ENDPOINT")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("KM_API_KEY", origAPIKey)
		os.Setenv("KM_API_ENDPOINT", origEndpoint)
	}()

	tests := []struct {
		name              string
		args              []string
		pluginName        string
		mockSetup         func(*MockAPIClient)
		expectError       bool
		expectContains    []string
		expectNotContains []string
		setupFiles        func(string) // Setup function to create files in temp directory
	}{
		{
			name:       "InstallConsoleLogger",
			args:       []string{"plugins", "install", "console-logger"},
			pluginName: "console-logger",
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "pro"
				m.AvailablePlugins = []plugindomain.Plugin{
					{
						Name:         "console-logger",
						Version:      "1.2.3",
						Description:  "Logs MCP messages to console",
						RequiredTier: plugindomain.TierFree,
						Size:         1024 * 10, // 10KB
						Checksum:     "sha256:abcdef123456",
						DownloadURL:  "http://localhost:5194/api/plugins/download/console-logger",
					},
				}
				m.DownloadResponses = map[string][]byte{
					"console-logger": []byte("#!/bin/bash\necho 'Mock console-logger plugin binary'\n"),
				}
			},
			expectContains: []string{
				"📦 Installing plugin: console-logger",
				"🔍 Plugin not found locally, checking API registry",
				"✅ Successfully installed plugin from API registry: console-logger",
			},
		},
		{
			name:       "InstallAPILogger",
			args:       []string{"plugins", "install", "api-logger"},
			pluginName: "api-logger",
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "pro"
				m.AvailablePlugins = []plugindomain.Plugin{
					{
						Name:         "api-logger",
						Version:      "2.1.0",
						Description:  "Logs MCP messages to API",
						RequiredTier: plugindomain.TierPro,
						Size:         1024 * 25, // 25KB
						Checksum:     "sha256:xyz789",
						DownloadURL:  "http://localhost:5194/api/plugins/download/api-logger",
					},
				}
				m.DownloadResponses = map[string][]byte{
					"api-logger": []byte("#!/bin/bash\necho 'Mock api-logger plugin binary'\n"),
				}
			},
			expectContains: []string{
				"📦 Installing plugin: api-logger",
				"✅ Successfully installed plugin from API registry: api-logger",
			},
		},
		{
			name:       "InstallWithoutAPIKey",
			args:       []string{"plugins", "install", "console-logger"},
			pluginName: "console-logger",
			mockSetup: func(m *MockAPIClient) {
				// Don't set up any valid API key
			},
			expectError: true,
			expectContains: []string{
				"API key required",
			},
		},
		{
			name:       "InstallPluginNotAvailableForTier",
			args:       []string{"plugins", "install", "enterprise-plugin"},
			pluginName: "enterprise-plugin",
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "free" // Free tier trying to install enterprise plugin
				m.AvailablePlugins = []plugindomain.Plugin{
					{
						Name:         "enterprise-plugin",
						Version:      "1.0.0",
						Description:  "Enterprise-only plugin",
						RequiredTier: plugindomain.TierEnterprise,
						Size:         1024 * 50,
						Checksum:     "sha256:enterprise123",
					},
				}
			},
			expectError: true,
			expectContains: []string{
				"📦 Installing plugin: enterprise-plugin",
				"failed to install plugin",
			},
		},
		{
			name:       "InstallNonExistentPlugin",
			args:       []string{"plugins", "install", "non-existent-plugin"},
			pluginName: "non-existent-plugin",
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "pro"
				// Don't add the plugin to available plugins
			},
			expectError: true,
			expectContains: []string{
				"📦 Installing plugin: non-existent-plugin",
				"failed to install plugin",
			},
		},
		{
			name: "SetupPluginsDirectory",
			args: []string{"plugins", "install"}, // No plugin name = setup directory
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "pro"
			},
			expectContains: []string{
				"plugins directory", // Should contain setup message
			},
		},
		{
			name:       "InstallFromLocalKmpkg",
			args:       []string{"plugins", "install", "local-plugin"},
			pluginName: "local-plugin",
			mockSetup: func(m *MockAPIClient) {
				m.ApiKeyValid = true
				m.SubscriptionTier = "pro"
			},
			setupFiles: func(tempDir string) {
				// Create a mock .kmpkg file in the plugins directory
				pluginsDir := filepath.Join(tempDir, ".km", "plugins")
				os.MkdirAll(pluginsDir, 0755)

				// Create a simple .kmpkg file (mock structure)
				kmpkgPath := filepath.Join(pluginsDir, "local-plugin.kmpkg")
				mockKmpkgContent := `{
					"name": "local-plugin",
					"version": "1.0.0",
					"description": "Local test plugin",
					"binary_path": "local-plugin"
				}`
				os.WriteFile(kmpkgPath, []byte(mockKmpkgContent), 0644)

				// Create the binary file referenced in the .kmpkg
				binaryPath := filepath.Join(pluginsDir, "local-plugin")
				os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'Local plugin binary'\n"), 0755)
			},
			expectContains: []string{
				"📦 Installing plugin: local-plugin",
				"✅ Successfully installed plugin from local .kmpkg: local-plugin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated temp directory for each test
			tempDir := t.TempDir()
			os.Setenv("HOME", tempDir)

			// Clear environment variables
			os.Unsetenv("KM_API_KEY")
			os.Unsetenv("KM_API_ENDPOINT")

			// Setup any files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(tempDir)
			}

			// Create mock API server
			mockAPI := NewMockAPIClient(t).Build()
			if tt.mockSetup != nil {
				tt.mockSetup(mockAPI)
			}
			defer mockAPI.Close()

			// Set environment for API endpoint and key
			os.Setenv("KM_API_ENDPOINT", mockAPI.URL())
			if mockAPI.ApiKeyValid {
				os.Setenv("KM_API_KEY", "test-api-key-123")
			}

			// Capture output
			var output bytes.Buffer
			var errorOutput bytes.Buffer

			// Create root command
			rootCmd := &cobra.Command{
				Use: "km",
			}

			// Add the plugins command
			pluginsCmd := cli.NewPluginsCommand("test-version")
			rootCmd.AddCommand(pluginsCmd)

			// Set output writers
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&errorOutput)

			// Set arguments and execute
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
			} else {
				assert.NoError(t, err, "Expected command to succeed")
			}

			// Get combined output
			combinedOutput := output.String() + errorOutput.String()

			// Check expected content
			for _, expected := range tt.expectContains {
				assert.Contains(t, combinedOutput, expected, "Expected output to contain: %s", expected)
			}

			// Check unexpected content
			for _, unexpected := range tt.expectNotContains {
				assert.NotContains(t, combinedOutput, unexpected, "Expected output NOT to contain: %s", unexpected)
			}

			// Additional assertions for successful installs
			if !tt.expectError && tt.pluginName != "" && tt.name != "SetupPluginsDirectory" {
				// Check that plugin directory was created
				pluginsDir := filepath.Join(tempDir, ".km", "plugins")
				assert.DirExists(t, pluginsDir, "Plugins directory should exist")

				// For local installs, verify the plugin was found locally
				if tt.name == "InstallFromLocalKmpkg" {
					kmpkgPath := filepath.Join(pluginsDir, tt.pluginName+".kmpkg")
					assert.FileExists(t, kmpkgPath, "Local .kmpkg file should exist")
				}
			}
		})
	}
}

// TestPluginInstallWithRealServer tests plugin installation against a real mock server
func TestPluginInstallWithRealServer(t *testing.T) {
	// Skip this test if we don't have a running mock server
	// This test is meant to be run when the mock server is actually running
	if os.Getenv("TEST_WITH_MOCK_SERVER") != "true" {
		t.Skip("Skipping real server test. Set TEST_WITH_MOCK_SERVER=true to run.")
	}

	// Save original environment
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Set up environment for real mock server
	os.Setenv("KM_API_ENDPOINT", "http://localhost:5194")
	os.Setenv("KM_API_KEY", "km_pro_789012") // Use a known test key from mock server

	// Create and configure the mock server via HTTP calls
	mockAPI := NewMockAPIClient(t).Build()
	defer mockAPI.Close()

	// Set up test plugins
	mockAPI.AvailablePlugins = []plugindomain.Plugin{
		{
			Name:         "test-integration-plugin",
			Version:      "1.0.0",
			Description:  "Integration test plugin",
			RequiredTier: plugindomain.TierPro,
			Size:         2048,
			Checksum:     "sha256:integration123",
		},
	}
	mockAPI.DownloadResponses = map[string][]byte{
		"test-integration-plugin": []byte("#!/bin/bash\necho 'Integration test plugin'\n"),
	}

	// Create root command and run install
	var output bytes.Buffer
	rootCmd := &cobra.Command{Use: "km"}
	pluginsCmd := cli.NewPluginsCommand("test")
	rootCmd.AddCommand(pluginsCmd)
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"plugins", "install", "test-integration-plugin"})

	err := rootCmd.Execute()
	require.NoError(t, err, "Plugin install should succeed")

	outputStr := output.String()
	assert.Contains(t, outputStr, "📦 Installing plugin: test-integration-plugin")
	assert.Contains(t, outputStr, "✅ Successfully installed")
}

// TestPluginListCommand tests the plugin list functionality
func TestPluginListCommand(t *testing.T) {
	// Save original environment
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Create mock API server
	mockAPI := NewMockAPIClient(t).Build()
	mockAPI.ApiKeyValid = true
	mockAPI.SubscriptionTier = "pro"
	mockAPI.AvailablePlugins = []plugindomain.Plugin{
		{
			Name:         "console-logger",
			Version:      "1.2.3",
			Description:  "Console logging plugin",
			RequiredTier: plugindomain.TierFree,
			Size:         10240,
		},
		{
			Name:         "api-logger",
			Version:      "2.1.0",
			Description:  "API logging plugin",
			RequiredTier: plugindomain.TierPro,
			Size:         25600,
		},
	}
	defer mockAPI.Close()

	// Set environment
	os.Setenv("KM_API_ENDPOINT", mockAPI.URL())
	os.Setenv("KM_API_KEY", "test-api-key")

	// Create and execute command
	var output bytes.Buffer
	rootCmd := &cobra.Command{Use: "km"}
	pluginsCmd := cli.NewPluginsCommand("test")
	rootCmd.AddCommand(pluginsCmd)
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"plugins", "list"})

	err := rootCmd.Execute()
	require.NoError(t, err, "Plugin list should succeed")

	outputStr := output.String()
	assert.Contains(t, outputStr, "🔍 Fetching available plugins")
	assert.Contains(t, outputStr, "Available Plugins (2)")
	assert.Contains(t, outputStr, "console-logger")
	assert.Contains(t, outputStr, "api-logger")
	assert.Contains(t, outputStr, "1.2.3")
	assert.Contains(t, outputStr, "2.1.0")
}
