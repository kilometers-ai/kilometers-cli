package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Monitor Command Configuration Integration Tests
// These tests verify that the monitor command correctly integrates with config loading
// =============================================================================

func TestMonitorCommand_ConfigurationIntegration(t *testing.T) {
	// Save original environment
	originalApiKey := os.Getenv("KILOMETERS_API_KEY")
	originalEndpoint := os.Getenv("KILOMETERS_API_ENDPOINT")
	
	// Clean up after test
	defer func() {
		if originalApiKey != "" {
			os.Setenv("KILOMETERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("KILOMETERS_API_KEY")
		}
		if originalEndpoint != "" {
			os.Setenv("KILOMETERS_API_ENDPOINT", originalEndpoint)
		} else {
			os.Unsetenv("KILOMETERS_API_ENDPOINT")
		}
	}()

	t.Run("createMessageLogger uses config file when available", func(t *testing.T) {
		// Create config file with API key
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_test_monitor_key",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(configPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Clear environment variables to force file loading
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Create monitor config
		monitorConfig := domain.DefaultMonitorConfig()
		
		// Call createMessageLogger - should detect API key and create API handler
		messageLogger := createMessageLogger(monitorConfig)
		
		// Should be ApiHandler wrapping ConsoleLogger when API key is present
		assert.IsType(t, &logging.ApiHandler{}, messageLogger)
	})

	t.Run("createMessageLogger falls back to console only when no API key", func(t *testing.T) {
		// Create config file without API key
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(configPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Create monitor config
		monitorConfig := domain.DefaultMonitorConfig()
		
		// Call createMessageLogger - should create console logger only
		messageLogger := createMessageLogger(monitorConfig)
		
		// Should be ConsoleLogger when no API key
		assert.IsType(t, &logging.ConsoleLogger{}, messageLogger)
	})

	t.Run("environment variables override config file in monitor flow", func(t *testing.T) {
		// Create config file with one API key
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_file_key",
			"api_endpoint": "http://file-endpoint:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(configPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Set environment variable that should override file
		os.Setenv("KILOMETERS_API_KEY", "km_live_env_override_key")
		
		// Create monitor config
		monitorConfig := domain.DefaultMonitorConfig()
		
		// Call createMessageLogger - should use env API key
		messageLogger := createMessageLogger(monitorConfig)
		
		// Should be ApiHandler since we have an API key (from env)
		assert.IsType(t, &logging.ApiHandler{}, messageLogger)
		
		// Verify the loaded config used environment override
		config := domain.LoadConfig()
		assert.Equal(t, "km_live_env_override_key", config.ApiKey)
	})

	t.Run("monitor command works with missing config file", func(t *testing.T) {
		// Point to non-existent config file
		nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist", "config.json")
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(nonExistentPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Create monitor config
		monitorConfig := domain.DefaultMonitorConfig()
		
		// Call createMessageLogger - should not fail, use console only
		messageLogger := createMessageLogger(monitorConfig)
		
		// Should be ConsoleLogger when no config and no env vars
		assert.IsType(t, &logging.ConsoleLogger{}, messageLogger)
	})
}

func TestMonitorCommand_RealWorldConfigScenarios(t *testing.T) {
	// Save original environment
	originalApiKey := os.Getenv("KILOMETERS_API_KEY")
	originalEndpoint := os.Getenv("KILOMETERS_API_ENDPOINT")
	
	// Clean up after test
	defer func() {
		if originalApiKey != "" {
			os.Setenv("KILOMETERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("KILOMETERS_API_KEY")
		}
		if originalEndpoint != "" {
			os.Setenv("KILOMETERS_API_ENDPOINT", originalEndpoint)
		} else {
			os.Unsetenv("KILOMETERS_API_ENDPOINT")
		}
	}()

	t.Run("user scenario: config file exists, no env vars needed", func(t *testing.T) {
		// Simulate user's actual current config
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		userConfigContent := `{
			"api_key": "km_live_CcGhXn8RH8WvgYkU9yq4yrcIFDt5JtPxjyUCFDjLk",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(userConfigContent), 0644))
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(configPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Remove environment variables (simulating clean mcp.json)
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Monitor command flow
		monitorConfig := domain.DefaultMonitorConfig()
		messageLogger := createMessageLogger(monitorConfig)
		
		// Should create API handler since config file has API key
		assert.IsType(t, &logging.ApiHandler{}, messageLogger)
		
		// Verify config loads user's settings
		config := domain.LoadConfig()
		assert.Equal(t, "km_live_CcGhXn8RH8WvgYkU9yq4yrcIFDt5JtPxjyUCFDjLk", config.ApiKey)
		assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
		
		// This test proves user can remove env section from mcp.json
		assert.NotEmpty(t, config.ApiKey, "Config file should provide API key without env vars")
	})

	t.Run("monitor command initializes successfully with file config", func(t *testing.T) {
		// Test the complete monitor initialization flow
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_integration_test_key",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 15,
			"debug": true
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override the config path temporarily
		originalGetConfigPath := domain.GetConfigPath
		domain.SetTestConfigPath(configPath)
		defer func() {
			domain.RestoreConfigPath(originalGetConfigPath)
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Test complete monitor service creation flow
		monitorConfig := domain.DefaultMonitorConfig()
		executor := createProcessExecutor()
		logger := createMessageLogger(monitorConfig)
		monitoringService := createMonitoringService(executor, logger)
		
		// All components should initialize successfully
		assert.NotNil(t, executor)
		assert.NotNil(t, logger)
		assert.NotNil(t, monitoringService)
		
		// Logger should be API handler due to config file API key
		assert.IsType(t, &logging.ApiHandler{}, logger)
	})
} 