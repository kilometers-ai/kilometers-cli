package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidator_ValidateAPIEndpoint(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid_https_endpoint",
			endpoint: "https://api.kilometers.ai",
			wantErr:  false,
		},
		{
			name:     "valid_http_localhost",
			endpoint: "http://localhost:5194",
			wantErr:  false,
		},
		{
			name:     "valid_http_127_0_0_1",
			endpoint: "http://127.0.0.1:8080",
			wantErr:  false,
		},
		{
			name:     "valid_endpoint_with_path",
			endpoint: "https://api.kilometers.ai/v1",
			wantErr:  false,
		},
		{
			name:     "empty_endpoint",
			endpoint: "",
			wantErr:  true,
			errMsg:   "API endpoint cannot be empty",
		},
		{
			name:     "invalid_scheme",
			endpoint: "ftp://api.kilometers.ai",
			wantErr:  true,
			errMsg:   "unsupported URL scheme",
		},
		{
			name:     "missing_scheme",
			endpoint: "api.kilometers.ai",
			wantErr:  true,
			errMsg:   "invalid URL format",
		},
		{
			name:     "missing_host",
			endpoint: "https://",
			wantErr:  true,
			errMsg:   "URL must include host",
		},
		{
			name:     "invalid_url_format",
			endpoint: "not a url at all",
			wantErr:  true,
			errMsg:   "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAPIEndpoint(tt.endpoint)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_ValidateAPIKey(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_api_key",
			key:     "km_1234567890abcdefghij",
			wantErr: false,
		},
		{
			name:    "valid_api_key_with_dashes",
			key:     "12345678-90ab-cdef-ghij-klmnopqrstuv",
			wantErr: false,
		},
		{
			name:    "valid_api_key_with_underscores",
			key:     "api_key_1234567890_abcdef",
			wantErr: false,
		},
		{
			name:    "empty_key",
			key:     "",
			wantErr: true,
			errMsg:  "API key cannot be empty",
		},
		{
			name:    "key_too_short",
			key:     "short",
			wantErr: true,
			errMsg:  "API key too short",
		},
		{
			name:    "key_too_long",
			key:     strings.Repeat("a", 257),
			wantErr: true,
			errMsg:  "API key too long",
		},
		{
			name:    "key_with_spaces",
			key:     "key with spaces is invalid",
			wantErr: true,
			errMsg:  "cannot contain spaces",
		},
		{
			name:    "key_with_newline",
			key:     "key-with-newline\n-is-invalid",
			wantErr: true,
			errMsg:  "cannot contain whitespace",
		},
		{
			name:    "placeholder_key_1",
			key:     "your-api-key-here-12345",
			wantErr: true,
			errMsg:  "placeholder value",
		},
		{
			name:    "placeholder_key_2",
			key:     "xxxx-xxxx-xxxx-xxxx-xxxx",
			wantErr: true,
			errMsg:  "placeholder value",
		},
		{
			name:    "placeholder_key_3",
			key:     "REPLACE_ME_WITH_REAL_KEY",
			wantErr: true,
			errMsg:  "placeholder value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAPIKey(tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_ValidateBufferSize(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name    string
		size    interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_size_int",
			size:    1024 * 1024, // 1MB
			wantErr: false,
		},
		{
			name:    "valid_size_int64",
			size:    int64(2 * 1024 * 1024), // 2MB
			wantErr: false,
		},
		{
			name:    "valid_size_float64",
			size:    float64(1.5 * 1024 * 1024), // 1.5MB
			wantErr: false,
		},
		{
			name:    "valid_size_string",
			size:    "4MB",
			wantErr: false,
		},
		{
			name:    "minimum_valid_size",
			size:    1024, // 1KB
			wantErr: false,
		},
		{
			name:    "maximum_valid_size",
			size:    100 * 1024 * 1024, // 100MB
			wantErr: false,
		},
		{
			name:    "size_too_small",
			size:    512, // Less than 1KB
			wantErr: true,
			errMsg:  "too small",
		},
		{
			name:    "size_too_large",
			size:    101 * 1024 * 1024, // More than 100MB
			wantErr: true,
			errMsg:  "too large",
		},
		{
			name:    "invalid_string_format",
			size:    "invalid",
			wantErr: true,
			errMsg:  "invalid buffer size format",
		},
		{
			name:    "invalid_type",
			size:    true,
			wantErr: true,
			errMsg:  "invalid buffer size type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBufferSize(tt.size)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_ValidateLogLevel(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"valid_debug", "debug", false},
		{"valid_info", "info", false},
		{"valid_warn", "warn", false},
		{"valid_error", "error", false},
		{"valid_fatal", "fatal", false},
		{"valid_panic", "panic", false},
		{"valid_trace", "trace", false},
		{"valid_uppercase", "DEBUG", false},
		{"valid_with_spaces", "  info  ", false},
		{"invalid_level", "invalid", true},
		{"empty_level", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLogLevel(tt.level)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_ValidateTimeout(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name    string
		timeout interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_duration",
			timeout: 30 * time.Second,
			wantErr: false,
		},
		{
			name:    "valid_string_duration",
			timeout: "30s",
			wantErr: false,
		},
		{
			name:    "valid_int_seconds",
			timeout: 60,
			wantErr: false,
		},
		{
			name:    "valid_float_seconds",
			timeout: 45.5,
			wantErr: false,
		},
		{
			name:    "minimum_valid_timeout",
			timeout: "1s",
			wantErr: false,
		},
		{
			name:    "maximum_valid_timeout",
			timeout: "5m",
			wantErr: false,
		},
		{
			name:    "timeout_too_short",
			timeout: "500ms",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "timeout_too_long",
			timeout: "6m",
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "invalid_string_format",
			timeout: "not-a-duration",
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name:    "invalid_type",
			timeout: true,
			wantErr: true,
			errMsg:  "invalid timeout type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTimeout(tt.timeout)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigValidator_ValidateAll(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name       string
		config     map[string]interface{}
		wantErrors map[string]bool // which fields should have errors
	}{
		{
			name: "all_valid",
			config: map[string]interface{}{
				"api_endpoint":    "https://api.kilometers.ai",
				"api_key":         "valid-key-with-20-characters",
				"buffer_size":     "2MB",
				"log_level":       "info",
				"plugins_dir":     "",
				"default_timeout": "30s",
			},
			wantErrors: map[string]bool{},
		},
		{
			name: "multiple_errors",
			config: map[string]interface{}{
				"api_endpoint":    "ftp://invalid.com",
				"api_key":         "short",
				"buffer_size":     100, // Too small
				"log_level":       "invalid",
				"default_timeout": "10m", // Too long
			},
			wantErrors: map[string]bool{
				"api_endpoint":    true,
				"api_key":         true,
				"buffer_size":     true,
				"log_level":       true,
				"default_timeout": true,
			},
		},
		{
			name: "partial_config",
			config: map[string]interface{}{
				"api_endpoint": "https://api.kilometers.ai",
				"log_level":    "debug",
			},
			wantErrors: map[string]bool{}, // Only validates present fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateAll(tt.config)

			// Check that we got errors for the expected fields
			for field, shouldError := range tt.wantErrors {
				if shouldError {
					assert.Contains(t, errors, field, "Expected error for field %s", field)
				} else {
					assert.NotContains(t, errors, field, "Unexpected error for field %s", field)
				}
			}

			// Check that we didn't get unexpected errors
			for field := range errors {
				assert.True(t, tt.wantErrors[field], "Unexpected error for field %s", field)
			}
		})
	}
}

