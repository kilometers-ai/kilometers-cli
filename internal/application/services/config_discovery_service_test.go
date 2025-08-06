package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDiscoveryService_DiscoverConfig(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() func()
		wantConfig func(*domain.DiscoveredConfig) bool
		wantErr    bool
	}{
		{
			name: "discovers_from_environment_variables",
			setup: func() func() {
				os.Setenv("KILOMETERS_API_KEY", "test-api-key-12345")
				os.Setenv("KILOMETERS_API_ENDPOINT", "https://api.test.com")
				os.Setenv("KILOMETERS_BUFFER_SIZE", "2MB")
				os.Setenv("KILOMETERS_LOG_LEVEL", "debug")

				return func() {
					os.Unsetenv("KILOMETERS_API_KEY")
					os.Unsetenv("KILOMETERS_API_ENDPOINT")
					os.Unsetenv("KILOMETERS_BUFFER_SIZE")
					os.Unsetenv("KILOMETERS_LOG_LEVEL")
				}
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "test-api-key-12345" &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "https://api.test.com" &&
					cfg.BufferSize != nil &&
					cfg.BufferSize.Value.(int) == 2*1024*1024 &&
					cfg.LogLevel != nil &&
					cfg.LogLevel.Value.(string) == "debug"
			},
			wantErr: false,
		},
		{
			name: "discovers_from_legacy_environment_variables",
			setup: func() func() {
				os.Setenv("KM_API_KEY", "legacy-key-12345")
				os.Setenv("KM_API_ENDPOINT", "http://localhost:8080")

				return func() {
					os.Unsetenv("KM_API_KEY")
					os.Unsetenv("KM_API_ENDPOINT")
				}
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "legacy-key-12345" &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "http://localhost:8080"
			},
			wantErr: false,
		},
		{
			name: "applies_defaults_for_missing_values",
			setup: func() func() {
				// No setup - rely on defaults
				return func() {}
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Should have default values
				return cfg.BufferSize != nil &&
					cfg.BufferSize.Value.(int) == 1024*1024 && // 1MB default
					cfg.LogLevel != nil &&
					cfg.LogLevel.Value.(string) == "info" && // info default
					cfg.DefaultTimeout != nil &&
					cfg.DefaultTimeout.Value.(time.Duration) == 30*time.Second // 30s default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			service, err := NewConfigDiscoveryService()
			require.NoError(t, err)
			service.SetShowProgress(false) // Disable progress output in tests

			ctx := context.Background()
			config, err := service.DiscoverConfig(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.True(t, tt.wantConfig(config), "Config does not match expected values")
		})
	}
}

func TestConfigDiscoveryService_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *domain.DiscoveredConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_configuration",
			config: &domain.DiscoveredConfig{
				APIEndpoint: &domain.ConfigValue{
					Value: "https://api.kilometers.ai",
				},
				APIKey: &domain.ConfigValue{
					Value: "valid-api-key-with-20-chars-minimum",
				},
				BufferSize: &domain.ConfigValue{
					Value: 1024 * 1024, // 1MB
				},
				LogLevel: &domain.ConfigValue{
					Value: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "missing_required_api_endpoint",
			config: &domain.DiscoveredConfig{
				APIKey: &domain.ConfigValue{
					Value: "valid-api-key-with-20-chars-minimum",
				},
			},
			wantErr: true,
			errMsg:  "api_endpoint: API endpoint is required",
		},
		{
			name: "invalid_api_endpoint_scheme",
			config: &domain.DiscoveredConfig{
				APIEndpoint: &domain.ConfigValue{
					Value: "ftp://invalid.scheme.com",
				},
			},
			wantErr: true,
			errMsg:  "unsupported URL scheme",
		},
		{
			name: "invalid_api_key_too_short",
			config: &domain.DiscoveredConfig{
				APIEndpoint: &domain.ConfigValue{
					Value: "https://api.kilometers.ai",
				},
				APIKey: &domain.ConfigValue{
					Value: "short-key",
				},
			},
			wantErr: true,
			errMsg:  "API key too short",
		},
		{
			name: "invalid_buffer_size",
			config: &domain.DiscoveredConfig{
				APIEndpoint: &domain.ConfigValue{
					Value: "https://api.kilometers.ai",
				},
				BufferSize: &domain.ConfigValue{
					Value: 500, // Less than 1KB
				},
			},
			wantErr: true,
			errMsg:  "buffer size too small",
		},
		{
			name: "invalid_log_level",
			config: &domain.DiscoveredConfig{
				APIEndpoint: &domain.ConfigValue{
					Value: "https://api.kilometers.ai",
				},
				LogLevel: &domain.ConfigValue{
					Value: "invalid-level",
				},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewConfigDiscoveryService()
			require.NoError(t, err)

			err = service.ValidateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigDiscoveryService_MigrateConfig(t *testing.T) {
	tests := []struct {
		name          string
		oldConfig     map[string]interface{}
		wantMigration map[string]string
		wantConfig    map[string]interface{}
	}{
		{
			name: "migrate_camelCase_to_snake_case",
			oldConfig: map[string]interface{}{
				"apiKey":      "test-key",
				"apiEndpoint": "https://api.test.com",
				"bufferSize":  2048,
				"logLevel":    "debug",
			},
			wantMigration: map[string]string{
				"apiKey":      "api_key",
				"apiEndpoint": "api_endpoint",
				"bufferSize":  "buffer_size",
				"logLevel":    "log_level",
			},
			wantConfig: map[string]interface{}{
				"api_key":      "test-key",
				"api_endpoint": "https://api.test.com",
				"buffer_size":  2048,
				"log_level":    "debug",
			},
		},
		{
			name: "migrate_debug_flag_to_log_level",
			oldConfig: map[string]interface{}{
				"apiKey": "test-key",
				"debug":  true,
			},
			wantMigration: map[string]string{
				"apiKey": "api_key",
				"debug":  "log_level=debug",
			},
			wantConfig: map[string]interface{}{
				"api_key":   "test-key",
				"log_level": "debug",
			},
		},
		{
			name: "migrate_api_url_to_api_endpoint",
			oldConfig: map[string]interface{}{
				"api_url": "https://old.api.com",
			},
			wantMigration: map[string]string{
				"api_url": "api_endpoint",
			},
			wantConfig: map[string]interface{}{
				"api_endpoint": "https://old.api.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewConfigDiscoveryService()
			require.NoError(t, err)
			service.SetShowProgress(false)

			// Make a copy of old config to test mutation
			configCopy := make(map[string]interface{})
			for k, v := range tt.oldConfig {
				configCopy[k] = v
			}

			migration, err := service.MigrateConfig(configCopy)

			require.NoError(t, err)
			require.NotNil(t, migration)

			// Check migration record
			assert.Equal(t, "legacy", migration.FromVersion)
			assert.Equal(t, "1.0", migration.ToVersion)
			assert.Equal(t, tt.wantMigration, migration.Changes)

			// Check that config was modified in place
			assert.Equal(t, tt.wantConfig, configCopy)
		})
	}
}

func TestConfigDiscoveryService_FileSystemDiscovery(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setupFiles func() error
		wantConfig func(*domain.DiscoveredConfig) bool
	}{
		{
			name: "discovers_from_yaml_config",
			setupFiles: func() error {
				configContent := `
api_key: yaml-test-key-12345
api_endpoint: https://yaml.test.com
buffer_size: 4MB
log_level: warn
plugins_dir: ~/.km/plugins
auto_provision: true
default_timeout: 60s
`
				return os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "yaml-test-key-12345" &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "https://yaml.test.com"
			},
		},
		{
			name: "discovers_from_json_config",
			setupFiles: func() error {
				configContent := `{
  "api_key": "json-test-key-12345",
  "api_endpoint": "https://json.test.com",
  "buffer_size": 2097152,
  "log_level": "error"
}`
				return os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configContent), 0644)
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "json-test-key-12345" &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "https://json.test.com"
			},
		},
		{
			name: "discovers_from_env_file",
			setupFiles: func() error {
				envContent := `
# API Configuration
KILOMETERS_API_KEY=env-file-key-12345
KILOMETERS_API_ENDPOINT=https://env.test.com

# Other settings
API_KEY=backup-key-12345
`
				return os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Should find the KILOMETERS_API_KEY
				return cfg.APIKey != nil &&
					cfg.APIEndpoint != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup files
			require.NoError(t, tt.setupFiles())

			// Create service and override search paths
			service, err := NewConfigDiscoveryService()
			require.NoError(t, err)
			service.SetShowProgress(false)

			// Temporarily change working directory
			oldWd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldWd)

			ctx := context.Background()
			config, err := service.DiscoverConfig(ctx)

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.True(t, tt.wantConfig(config), "Config does not match expected values")

			// Clean up files for next test
			files, _ := filepath.Glob(filepath.Join(tmpDir, "*"))
			for _, file := range files {
				os.Remove(file)
			}
		})
	}
}

// TestPrintDiscoveredConfig tests the output formatting
func TestPrintDiscoveredConfig(t *testing.T) {
	config := &domain.DiscoveredConfig{
		APIEndpoint: &domain.ConfigValue{
			Value:      "https://api.test.com",
			Source:     domain.SourceEnvironment,
			SourcePath: "KILOMETERS_API_ENDPOINT",
			Confidence: 1.0,
		},
		APIKey: &domain.ConfigValue{
			Value:      "test-key-12345-67890-abcdef",
			Source:     domain.SourceUserConfig,
			SourcePath: "~/.km/config.yaml",
			Confidence: 0.9,
		},
		BufferSize: &domain.ConfigValue{
			Value:      2097152,
			Source:     domain.SourceProjectConfig,
			SourcePath: "./km.config.yaml",
			Confidence: 1.0,
		},
		LogLevel: &domain.ConfigValue{
			Value:      "debug",
			Source:     domain.SourceAutoDiscovered,
			SourcePath: "default",
			Confidence: 0.7,
		},
		Warnings: []string{
			"Found API key in non-standard location",
			"Multiple config files found",
		},
	}

	// This test just verifies the function doesn't panic
	// In a real test, we might capture stdout
	PrintDiscoveredConfig(config)
}

