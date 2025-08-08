package domain

import (
	"fmt"
	"time"
)

// ConfigSource represents where a configuration value was loaded from
type ConfigSource struct {
	Value      interface{} `json:"value"`
	Source     string      `json:"source"`      // "env", "file", "cli", "default"
	SourcePath string      `json:"source_path"` // specific file path or env var name
	Priority   int         `json:"priority"`    // loading precedence (1=highest)
}

// UnifiedConfig represents the complete configuration for the Kilometers CLI
type UnifiedConfig struct {
	// Core API settings
	APIKey      string `json:"api_key"`
	APIEndpoint string `json:"api_endpoint"`

	// Monitoring settings
	BufferSize int    `json:"buffer_size"`
	BatchSize  int    `json:"batch_size"`
	LogLevel   string `json:"log_level"`
	Debug      bool   `json:"debug"`

	// Plugin settings
	PluginsDir    string `json:"plugins_dir"`
	AutoProvision bool   `json:"auto_provision"`

	// Advanced settings
	DefaultTimeout time.Duration `json:"default_timeout"`

	// Metadata for transparency and debugging
	Sources  map[string]ConfigSource `json:"sources"`
	LoadedAt time.Time               `json:"loaded_at"`
}

// DefaultUnifiedConfig returns a configuration with sensible defaults
func DefaultUnifiedConfig() *UnifiedConfig {
	now := time.Now()
	return &UnifiedConfig{
		APIEndpoint:    "http://localhost:5194",
		BufferSize:     1024 * 1024, // 1MB
		BatchSize:      10,
		LogLevel:       "info",
		Debug:          false,
		AutoProvision:  false,
		DefaultTimeout: 30 * time.Second,
		Sources:        make(map[string]ConfigSource),
		LoadedAt:       now,
	}
}

// SetValue sets a configuration value with its source metadata
func (c *UnifiedConfig) SetValue(field, source, sourcePath string, value interface{}, priority int) error {
	configSource := ConfigSource{
		Value:      value,
		Source:     source,
		SourcePath: sourcePath,
		Priority:   priority,
	}

	// Only update if this source has higher or equal priority
	if existing, exists := c.Sources[field]; !exists || priority <= existing.Priority {
		c.Sources[field] = configSource

		// Set the actual field value
		switch field {
		case "api_key":
			if v, ok := value.(string); ok {
				c.APIKey = v
			}
		case "api_endpoint":
			if v, ok := value.(string); ok {
				c.APIEndpoint = v
			}
		case "buffer_size":
			if v, ok := value.(int); ok {
				c.BufferSize = v
			}
		case "batch_size":
			if v, ok := value.(int); ok {
				c.BatchSize = v
			}
		case "log_level":
			if v, ok := value.(string); ok {
				c.LogLevel = v
			}
		case "debug":
			if v, ok := value.(bool); ok {
				c.Debug = v
			}
		case "plugins_dir":
			if v, ok := value.(string); ok {
				c.PluginsDir = v
			}
		case "auto_provision":
			if v, ok := value.(bool); ok {
				c.AutoProvision = v
			}
		case "default_timeout":
			if v, ok := value.(time.Duration); ok {
				c.DefaultTimeout = v
			} else if v, ok := value.(string); ok {
				if d, err := time.ParseDuration(v); err == nil {
					c.DefaultTimeout = d
				}
			}
		default:
			return fmt.Errorf("unknown config field: %s", field)
		}
	}

	return nil
}

// GetSource returns the source information for a specific field
func (c *UnifiedConfig) GetSource(field string) (ConfigSource, bool) {
	source, exists := c.Sources[field]
	return source, exists
}


// ToMonitorConfig converts UnifiedConfig to MonitorConfig for monitoring operations
func (c *UnifiedConfig) ToMonitorConfig() MonitorConfig {
	return MonitorConfig{
		BufferSize: c.BufferSize,
	}
}

// Validate performs domain-level validation on the configuration
func (c *UnifiedConfig) Validate() error {
	var errors []string

	// Validate API endpoint if provided
	if c.APIEndpoint != "" {
		// Basic URL validation could be added here
	}

	// Validate buffer size
	if c.BufferSize <= 0 {
		errors = append(errors, "buffer_size must be greater than 0")
	}

	// Validate batch size
	if c.BatchSize <= 0 {
		errors = append(errors, "batch_size must be greater than 0")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if c.LogLevel != "" && !validLogLevels[c.LogLevel] {
		errors = append(errors, fmt.Sprintf("invalid log_level: %s (must be one of: debug, info, warn, error, fatal)", c.LogLevel))
	}

	// Validate timeout
	if c.DefaultTimeout < 0 {
		errors = append(errors, "default_timeout must be non-negative")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errors)
	}

	return nil
}

// IsDebugMode returns true if debug logging is enabled
func (c *UnifiedConfig) IsDebugMode() bool {
	return c.Debug || c.LogLevel == "debug"
}

// HasAPIKey returns true if an API key is configured
func (c *UnifiedConfig) HasAPIKey() bool {
	return c.APIKey != ""
}