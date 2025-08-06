package domain

import (
	"time"
)

// DiscoverySource represents where a configuration value was discovered
type DiscoverySource string

const (
	SourceCommandLine    DiscoverySource = "command_line"
	SourceEnvironment    DiscoverySource = "environment"
	SourceProjectConfig  DiscoverySource = "project_config"
	SourceUserConfig     DiscoverySource = "user_config"
	SourceSystemConfig   DiscoverySource = "system_config"
	SourceAutoDiscovered DiscoverySource = "auto_discovered"
)

// ConfigValue represents a discovered configuration value with its source
type ConfigValue struct {
	Value      interface{}
	Source     DiscoverySource
	SourcePath string  // File path or env var name
	Confidence float32 // 0.0 to 1.0, how confident we are in this value
}

// DiscoveredConfig represents configuration values discovered from various sources
type DiscoveredConfig struct {
	APIEndpoint    *ConfigValue
	APIKey         *ConfigValue
	BufferSize     *ConfigValue
	LogLevel       *ConfigValue
	PluginsDir     *ConfigValue
	AutoProvision  *ConfigValue
	DefaultTimeout *ConfigValue

	// Discovery metadata
	DiscoveredAt time.Time
	TotalSources int
	Warnings     []string
}

// ConfigMigration represents a migration from an old config format
type ConfigMigration struct {
	FromVersion string
	ToVersion   string
	MigratedAt  time.Time
	Changes     map[string]string // old_key -> new_key mapping
}

// ValidationRule represents a rule for validating configuration values
type ValidationRule struct {
	Field       string
	Required    bool
	Validator   func(value interface{}) error
	Description string
}

// GetPriority returns the priority of a discovery source (lower is higher priority)
func (s DiscoverySource) GetPriority() int {
	priorities := map[DiscoverySource]int{
		SourceCommandLine:    1,
		SourceEnvironment:    2,
		SourceProjectConfig:  3,
		SourceUserConfig:     4,
		SourceSystemConfig:   5,
		SourceAutoDiscovered: 6,
	}

	if priority, ok := priorities[s]; ok {
		return priority
	}
	return 99 // Unknown source has lowest priority
}

// Merge combines multiple discovered configs, respecting source priority
func (dc *DiscoveredConfig) Merge(other *DiscoveredConfig) {
	if other == nil {
		return
	}

	// Helper function to merge individual config values
	mergeValue := func(current, new *ConfigValue) *ConfigValue {
		if new == nil {
			return current
		}
		if current == nil {
			return new
		}
		// Keep the value with higher priority (lower number)
		if new.Source.GetPriority() < current.Source.GetPriority() {
			return new
		}
		return current
	}

	dc.APIEndpoint = mergeValue(dc.APIEndpoint, other.APIEndpoint)
	dc.APIKey = mergeValue(dc.APIKey, other.APIKey)
	dc.BufferSize = mergeValue(dc.BufferSize, other.BufferSize)
	dc.LogLevel = mergeValue(dc.LogLevel, other.LogLevel)
	dc.PluginsDir = mergeValue(dc.PluginsDir, other.PluginsDir)
	dc.AutoProvision = mergeValue(dc.AutoProvision, other.AutoProvision)
	dc.DefaultTimeout = mergeValue(dc.DefaultTimeout, other.DefaultTimeout)

	// Merge warnings
	dc.Warnings = append(dc.Warnings, other.Warnings...)
	dc.TotalSources += other.TotalSources
}

// ToConfig converts discovered config to the standard Config struct
func (dc *DiscoveredConfig) ToConfig() *Config {
	cfg := DefaultConfig()

	if dc.APIEndpoint != nil && dc.APIEndpoint.Value != nil {
		if v, ok := dc.APIEndpoint.Value.(string); ok {
			cfg.ApiEndpoint = v
		}
	}

	if dc.APIKey != nil && dc.APIKey.Value != nil {
		if v, ok := dc.APIKey.Value.(string); ok {
			cfg.ApiKey = v
		}
	}

	if dc.BufferSize != nil && dc.BufferSize.Value != nil {
		if v, ok := dc.BufferSize.Value.(int); ok {
			cfg.BatchSize = v // Map buffer size to batch size for now
		}
	}

	if dc.LogLevel != nil && dc.LogLevel.Value != nil {
		if v, ok := dc.LogLevel.Value.(string); ok && v == "debug" {
			cfg.Debug = true
		}
	}

	// Note: Additional fields like PluginsDir, AutoProvision, DefaultTimeout
	// are not in the current Config struct but will be used by the discovery service

	return &cfg
}

// ToExtendedConfig creates an extended configuration map with all discovered values
func (dc *DiscoveredConfig) ToExtendedConfig() map[string]interface{} {
	config := make(map[string]interface{})

	if dc.APIEndpoint != nil && dc.APIEndpoint.Value != nil {
		config["api_endpoint"] = dc.APIEndpoint.Value
	}

	if dc.APIKey != nil && dc.APIKey.Value != nil {
		config["api_key"] = dc.APIKey.Value
	}

	if dc.BufferSize != nil && dc.BufferSize.Value != nil {
		config["buffer_size"] = dc.BufferSize.Value
	}

	if dc.LogLevel != nil && dc.LogLevel.Value != nil {
		config["log_level"] = dc.LogLevel.Value
	}

	if dc.PluginsDir != nil && dc.PluginsDir.Value != nil {
		config["plugins_dir"] = dc.PluginsDir.Value
	}

	if dc.AutoProvision != nil && dc.AutoProvision.Value != nil {
		config["auto_provision"] = dc.AutoProvision.Value
	}

	if dc.DefaultTimeout != nil && dc.DefaultTimeout.Value != nil {
		config["default_timeout"] = dc.DefaultTimeout.Value
	}

	return config
}
