package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Tests for Configuration Loading
// These tests verify the complete configuration loading flow in real scenarios
// =============================================================================

func TestConfigLoading_EndToEndIntegration(t *testing.T) {
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

	t.Run("config file loaded correctly with real file system", func(t *testing.T) {
		// Create temporary directory structure mimicking real config
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_test_integration_key",
			"api_endpoint": "http://localhost:9999",
			"batch_size": 25,
			"debug": true
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Temporarily override the config path function for this test
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables to test file loading
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should read from file
		config := LoadConfig()
		
		assert.Equal(t, "km_live_test_integration_key", config.ApiKey)
		assert.Equal(t, "http://localhost:9999", config.ApiEndpoint)
		assert.Equal(t, 25, config.BatchSize)
		assert.True(t, config.Debug)
	})

	t.Run("environment variables override file configuration", func(t *testing.T) {
		// Create config file with specific values
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_file_key",
			"api_endpoint": "http://file-endpoint:5194",
			"batch_size": 15,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Set environment variables that should override file
		os.Setenv("KILOMETERS_API_KEY", "km_live_env_override_key")
		os.Setenv("KILOMETERS_API_ENDPOINT", "http://env-endpoint:8080")
		
		// Load config - env vars should override file values
		config := LoadConfig()
		
		assert.Equal(t, "km_live_env_override_key", config.ApiKey)
		assert.Equal(t, "http://env-endpoint:8080", config.ApiEndpoint)
		// These should remain from file since no env override
		assert.Equal(t, 15, config.BatchSize)
		assert.False(t, config.Debug)
	})

	t.Run("graceful fallback when config file missing", func(t *testing.T) {
		// Point to non-existent config file
		nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist", "config.json")
		
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return nonExistentPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should use defaults
		config := LoadConfig()
		
		assert.Empty(t, config.ApiKey) // Default is empty
		assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
		assert.Equal(t, 10, config.BatchSize)
		assert.False(t, config.Debug)
	})
}

func TestConfigLoading_ConfigFileValidation(t *testing.T) {
	t.Run("corrupted JSON config file handled gracefully", func(t *testing.T) {
		// Create config file with invalid JSON
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		invalidJSON := `{
			"api_key": "km_live_test_key",
			"api_endpoint": "http://localhost:5194"
			// Missing closing brace and has comment
		`
		
		require.NoError(t, os.WriteFile(configPath, []byte(invalidJSON), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should fall back to defaults when JSON is invalid
		config := LoadConfig()
		
		// Should use default values when file is corrupted
		assert.Empty(t, config.ApiKey)
		assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
		assert.Equal(t, 10, config.BatchSize)
		assert.False(t, config.Debug)
	})

	t.Run("config file with extra fields handled gracefully", func(t *testing.T) {
		// Create config file with extra unknown fields
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configWithExtras := `{
			"api_key": "km_live_test_key",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 20,
			"debug": true,
			"unknown_field": "should_be_ignored",
			"another_unknown": 42
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configWithExtras), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should ignore unknown fields
		config := LoadConfig()
		
		assert.Equal(t, "km_live_test_key", config.ApiKey)
		assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
		assert.Equal(t, 20, config.BatchSize)
		assert.True(t, config.Debug)
	})
}

func TestConfigLoading_RealWorldScenarios(t *testing.T) {
	t.Run("user has existing config like current setup", func(t *testing.T) {
		// Simulate user's actual config file content
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		realWorldConfig := `{
			"api_key": "km_live_CcGhXn8RH8WvgYkU9yq4yrcIFDt5JtPxjyUCFDjLk",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(realWorldConfig), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables to test pure file loading
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config
		config := LoadConfig()
		
		// Should load exactly like user's current setup
		assert.Equal(t, "km_live_CcGhXn8RH8WvgYkU9yq4yrcIFDt5JtPxjyUCFDjLk", config.ApiKey)
		assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
		assert.Equal(t, 10, config.BatchSize)
		assert.False(t, config.Debug)
		
		// Verify that config indicates API handler should be created
		assert.NotEmpty(t, config.ApiKey, "Config should have API key for API handler creation")
	})

	t.Run("user removes env vars from mcp.json and relies on config file", func(t *testing.T) {
		// This simulates the target scenario after our changes
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_user_key_from_file",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Explicitly clear environment variables (simulating clean mcp.json)
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config multiple times to ensure consistency
		config1 := LoadConfig()
		config2 := LoadConfig()
		
		// Both should be identical and loaded from file
		assert.Equal(t, config1, config2)
		assert.Equal(t, "km_live_user_key_from_file", config1.ApiKey)
		assert.Equal(t, "http://localhost:5194", config1.ApiEndpoint)
		
		// This should enable API integration
		assert.NotEmpty(t, config1.ApiKey, "File-based config should enable API integration")
	})
}

func TestConfigLoading_RealFileSystemIntegration(t *testing.T) {
	t.Run("config directory created with proper permissions", func(t *testing.T) {
		// Use temporary directory to test directory creation
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		configPath := filepath.Join(configDir, "config.json")
		
		// Override config path to point to temp directory
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Save a config which should create the directory structure
		testConfig := Config{
			ApiKey:      "km_live_test_key",
			ApiEndpoint: "http://localhost:5194",
			BatchSize:   10,
			Debug:       false,
		}
		
		err := SaveConfig(testConfig)
		require.NoError(t, err)
		
		// Verify directory was created
		dirInfo, err := os.Stat(configDir)
		require.NoError(t, err)
		assert.True(t, dirInfo.IsDir())
		
		// Verify file was created with proper permissions  
		fileInfo, err := os.Stat(configPath)
		require.NoError(t, err)
		assert.False(t, fileInfo.IsDir())
		assert.Equal(t, os.FileMode(0644), fileInfo.Mode().Perm())
	})

	t.Run("concurrent config loading is safe", func(t *testing.T) {
		// Create config file
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_concurrent_test_key",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 20,
			"debug": true
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config concurrently from multiple goroutines
		const numGoroutines = 10
		configs := make([]Config, numGoroutines)
		done := make(chan int, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				configs[index] = LoadConfig()
				done <- index
			}(i)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
		
		// All configs should be identical
		expectedConfig := Config{
			ApiKey:      "km_live_concurrent_test_key",
			ApiEndpoint: "http://localhost:5194",
			BatchSize:   20,
			Debug:       true,
		}
		
		for i, config := range configs {
			assert.Equal(t, expectedConfig, config, "Config %d should match expected", i)
		}
	})

	t.Run("config file handles special characters and unicode", func(t *testing.T) {
		// Create config file with special characters
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		// Config with special characters and unicode
		configContent := `{
			"api_key": "km_live_测试_key_with_émojis_🚀",
			"api_endpoint": "http://localhost:5194/api/v1/测试",
			"batch_size": 25,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should handle unicode properly
		config := LoadConfig()
		
		assert.Equal(t, "km_live_测试_key_with_émojis_🚀", config.ApiKey)
		assert.Equal(t, "http://localhost:5194/api/v1/测试", config.ApiEndpoint)
		assert.Equal(t, 25, config.BatchSize)
		assert.False(t, config.Debug)
	})

	t.Run("config loading performance is acceptable", func(t *testing.T) {
		// Create config file
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		configContent := `{
			"api_key": "km_live_performance_test_key",
			"api_endpoint": "http://localhost:5194",
			"batch_size": 10,
			"debug": false
		}`
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Measure config loading performance
		const iterations = 100
		start := time.Now()
		
		for i := 0; i < iterations; i++ {
			config := LoadConfig()
			// Verify config is loaded correctly
			assert.Equal(t, "km_live_performance_test_key", config.ApiKey)
		}
		
		duration := time.Since(start)
		avgDuration := duration / iterations
		
		// Config loading should be fast (less than 1ms per load on average)
		assert.Less(t, avgDuration, time.Millisecond, 
			"Config loading should be fast, got %v per load", avgDuration)
		
		t.Logf("Config loading performance: %v per load (total: %v for %d iterations)", 
			avgDuration, duration, iterations)
	})

	t.Run("config handles very long values", func(t *testing.T) {
		// Create config file with very long values
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "kilometers")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		
		configPath := filepath.Join(configDir, "config.json")
		
		// Generate very long API key and endpoint
		longApiKey := "km_live_" + strings.Repeat("a", 1000)
		longEndpoint := "http://localhost:5194/" + strings.Repeat("path/", 100)
		
		configContent := fmt.Sprintf(`{
			"api_key": "%s",
			"api_endpoint": "%s",
			"batch_size": 30,
			"debug": true
		}`, longApiKey, longEndpoint)
		
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))
		
		// Override config path
		originalGetConfigPath := getConfigPath
		getConfigPath = func() (string, error) {
			return configPath, nil
		}
		defer func() {
			getConfigPath = originalGetConfigPath
		}()
		
		// Clear environment variables
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")
		
		// Load config - should handle long values
		config := LoadConfig()
		
		assert.Equal(t, longApiKey, config.ApiKey)
		assert.Equal(t, longEndpoint, config.ApiEndpoint)
		assert.Equal(t, 30, config.BatchSize)
		assert.True(t, config.Debug)
		
		// Verify the loaded config makes sense for API usage
		assert.True(t, len(config.ApiKey) > 1000)
		assert.True(t, len(config.ApiEndpoint) > 500)
	})
} 