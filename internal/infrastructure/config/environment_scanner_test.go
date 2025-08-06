package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentScanner_Scan(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantConfig func(*domain.DiscoveredConfig) bool
		wantErr    bool
	}{
		{
			name: "scan_kilometers_prefixed_vars",
			envVars: map[string]string{
				"KILOMETERS_API_KEY":        "test-api-key-123456789012345",
				"KILOMETERS_API_ENDPOINT":   "https://api.test.kilometers.ai",
				"KILOMETERS_BUFFER_SIZE":    "4MB",
				"KILOMETERS_LOG_LEVEL":      "debug",
				"KILOMETERS_PLUGINS_DIR":    "~/custom/plugins",
				"KILOMETERS_AUTO_PROVISION": "true",
				"KILOMETERS_TIMEOUT":        "45s",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "test-api-key-123456789012345" &&
					cfg.APIKey.Source == domain.SourceEnvironment &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "https://api.test.kilometers.ai" &&
					cfg.BufferSize != nil &&
					cfg.BufferSize.Value.(int) == 4*1024*1024 &&
					cfg.LogLevel != nil &&
					cfg.LogLevel.Value.(string) == "debug" &&
					cfg.PluginsDir != nil &&
					cfg.AutoProvision != nil &&
					cfg.AutoProvision.Value.(bool) == true &&
					cfg.DefaultTimeout != nil &&
					cfg.DefaultTimeout.Value.(time.Duration) == 45*time.Second
			},
			wantErr: false,
		},
		{
			name: "scan_legacy_km_prefixed_vars",
			envVars: map[string]string{
				"KM_API_KEY":      "legacy-key-123456789012345",
				"KM_API_ENDPOINT": "http://localhost:8080",
				"KM_DEBUG":        "true",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "legacy-key-123456789012345" &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "http://localhost:8080" &&
					cfg.LogLevel != nil &&
					cfg.LogLevel.Value.(string) == "debug"
			},
			wantErr: false,
		},
		{
			name: "scan_special_env_vars_without_prefix",
			envVars: map[string]string{
				"API_KEY":      "special-key-123456789012345",
				"CI":           "true",
				"API_ENDPOINT": "https://ci.api.test.com",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Should find API_KEY with lower confidence
				return cfg.APIKey != nil &&
					cfg.APIKey.Value.(string) == "special-key-123456789012345" &&
					cfg.APIKey.Confidence < 1.0 &&
					cfg.APIEndpoint != nil &&
					cfg.APIEndpoint.Value.(string) == "https://ci.api.test.com"
			},
			wantErr: false,
		},
		{
			name: "handle_various_size_formats",
			envVars: map[string]string{
				"KILOMETERS_BUFFER_SIZE": "1024KB",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				return cfg.BufferSize != nil &&
					cfg.BufferSize.Value.(int) == 1024*1024
			},
			wantErr: false,
		},
		{
			name: "handle_invalid_values_with_warnings",
			envVars: map[string]string{
				"KILOMETERS_BUFFER_SIZE": "invalid-size",
				"KILOMETERS_TIMEOUT":     "not-a-duration",
				"KILOMETERS_API_KEY":     "valid-key-123456789012345",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Should still find valid values and add warnings
				return cfg.APIKey != nil &&
					cfg.BufferSize == nil && // Invalid, so not set
					cfg.DefaultTimeout == nil && // Invalid, so not set
					len(cfg.Warnings) >= 2 // Should have warnings for invalid values
			},
			wantErr: false,
		},
		{
			name: "handle_unknown_env_vars",
			envVars: map[string]string{
				"KILOMETERS_UNKNOWN_VAR": "some-value",
				"KM_ANOTHER_UNKNOWN":     "another-value",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Should add warnings for unknown vars
				warningCount := 0
				for _, w := range cfg.Warnings {
					if contains(w, "Unknown environment variable") {
						warningCount++
					}
				}
				return warningCount >= 2
			},
			wantErr: false,
		},
		{
			name: "priority_of_newer_prefix_over_legacy",
			envVars: map[string]string{
				"KILOMETERS_API_KEY": "new-key-123456789012345",
				"KM_API_KEY":         "old-key-123456789012345",
			},
			wantConfig: func(cfg *domain.DiscoveredConfig) bool {
				// Both should be found, but merge logic in service should prefer newer
				return cfg.APIKey != nil &&
					cfg.TotalSources == 2 // Found both
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current environment
			savedEnv := make(map[string]string)
			for k := range tt.envVars {
				if v, exists := os.LookupEnv(k); exists {
					savedEnv[k] = v
				}
			}

			// Set test environment
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Clean up after test
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
				for k, v := range savedEnv {
					os.Setenv(k, v)
				}
			}()

			scanner := NewEnvironmentScanner()
			ctx := context.Background()

			config, err := scanner.Scan(ctx)

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

func TestEnvironmentScanner_ParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1kb", 1024, false},
		{"2MB", 2 * 1024 * 1024, false},
		{"1.5MB", int(1.5 * 1024 * 1024), false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"100B", 100, false},
		{"  10KB  ", 10 * 1024, false},
		{"invalid", 0, true},
		{"10XB", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnvironmentScanner_CustomPrefixes(t *testing.T) {
	scanner := NewEnvironmentScanner()
	scanner.SetPrefixes([]string{"MYAPP_", "APP_"})

	// Set custom prefix env vars
	os.Setenv("MYAPP_API_KEY", "custom-key-123456789012345")
	os.Setenv("APP_API_ENDPOINT", "https://custom.api.com")
	defer os.Unsetenv("MYAPP_API_KEY")
	defer os.Unsetenv("APP_API_ENDPOINT")

	ctx := context.Background()
	config, err := scanner.Scan(ctx)

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.NotNil(t, config.APIKey)
	assert.Equal(t, "custom-key-123456789012345", config.APIKey.Value.(string))
	assert.NotNil(t, config.APIEndpoint)
	assert.Equal(t, "https://custom.api.com", config.APIEndpoint.Value.(string))
}

func TestEnvironmentScanner_Name(t *testing.T) {
	scanner := NewEnvironmentScanner()
	assert.Equal(t, "environment", scanner.Name())
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && contains(s[1:], substr)
}

