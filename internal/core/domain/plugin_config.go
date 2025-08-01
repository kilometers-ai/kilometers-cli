package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PluginConfig represents configuration for a specific plugin
type PluginConfig struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
	Version  string                 `json:"version"`
}

// PluginConfigs represents all plugin configurations
type PluginConfigs struct {
	Plugins map[string]PluginConfig `json:"plugins"`
	Version string                  `json:"version"`
}

// DefaultPluginConfigs returns default plugin configurations
func DefaultPluginConfigs() PluginConfigs {
	return PluginConfigs{
		Plugins: make(map[string]PluginConfig),
		Version: "1.0",
	}
}

// LoadPluginConfigs loads plugin configurations from file
func LoadPluginConfigs() (PluginConfigs, error) {
	configPath, err := getPluginConfigPath()
	if err != nil {
		return DefaultPluginConfigs(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return defaults if file doesn't exist
		if os.IsNotExist(err) {
			return DefaultPluginConfigs(), nil
		}
		return PluginConfigs{}, err
	}

	var configs PluginConfigs
	if err := json.Unmarshal(data, &configs); err != nil {
		return PluginConfigs{}, fmt.Errorf("failed to parse plugin config file: %w", err)
	}

	// Ensure plugins map is initialized
	if configs.Plugins == nil {
		configs.Plugins = make(map[string]PluginConfig)
	}

	return configs, nil
}

// SavePluginConfigs saves plugin configurations to file
func SavePluginConfigs(configs PluginConfigs) error {
	configPath, err := getPluginConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get plugin config path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugin configs: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plugin config file: %w", err)
	}

	return nil
}

// GetPluginConfig retrieves configuration for a specific plugin
func (pc *PluginConfigs) GetPluginConfig(pluginName string) PluginConfig {
	if config, exists := pc.Plugins[pluginName]; exists {
		return config
	}

	// Return default config if not found
	return PluginConfig{
		Name:     pluginName,
		Enabled:  true,
		Settings: make(map[string]interface{}),
		Version:  "1.0",
	}
}

// SetPluginConfig sets configuration for a specific plugin
func (pc *PluginConfigs) SetPluginConfig(pluginName string, config PluginConfig) {
	if pc.Plugins == nil {
		pc.Plugins = make(map[string]PluginConfig)
	}
	
	config.Name = pluginName
	pc.Plugins[pluginName] = config
}

// UpdatePluginSetting updates a specific setting for a plugin
func (pc *PluginConfigs) UpdatePluginSetting(pluginName, key string, value interface{}) {
	config := pc.GetPluginConfig(pluginName)
	
	if config.Settings == nil {
		config.Settings = make(map[string]interface{})
	}
	
	config.Settings[key] = value
	pc.SetPluginConfig(pluginName, config)
}

// EnablePlugin enables a specific plugin
func (pc *PluginConfigs) EnablePlugin(pluginName string) {
	config := pc.GetPluginConfig(pluginName)
	config.Enabled = true
	pc.SetPluginConfig(pluginName, config)
}

// DisablePlugin disables a specific plugin
func (pc *PluginConfigs) DisablePlugin(pluginName string) {
	config := pc.GetPluginConfig(pluginName)
	config.Enabled = false
	pc.SetPluginConfig(pluginName, config)
}

// IsPluginEnabled checks if a plugin is enabled
func (pc *PluginConfigs) IsPluginEnabled(pluginName string) bool {
	config := pc.GetPluginConfig(pluginName)
	return config.Enabled
}

// ListEnabledPlugins returns all enabled plugin names
func (pc *PluginConfigs) ListEnabledPlugins() []string {
	var enabled []string
	for name, config := range pc.Plugins {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// ExportConfig exports plugin configuration for sharing
func (pc *PluginConfigs) ExportConfig(pluginName string) (map[string]interface{}, error) {
	config := pc.GetPluginConfig(pluginName)
	
	export := map[string]interface{}{
		"name":     config.Name,
		"enabled":  config.Enabled,
		"settings": config.Settings,
		"version":  config.Version,
	}
	
	return export, nil
}

// ImportConfig imports plugin configuration from export
func (pc *PluginConfigs) ImportConfig(pluginName string, data map[string]interface{}) error {
	config := PluginConfig{
		Name:     pluginName,
		Enabled:  true,
		Settings: make(map[string]interface{}),
		Version:  "1.0",
	}
	
	if enabled, ok := data["enabled"].(bool); ok {
		config.Enabled = enabled
	}
	
	if settings, ok := data["settings"].(map[string]interface{}); ok {
		config.Settings = settings
	}
	
	if version, ok := data["version"].(string); ok {
		config.Version = version
	}
	
	pc.SetPluginConfig(pluginName, config)
	return nil
}

// getPluginConfigPath returns the path to the plugin config file
func getPluginConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "kilometers", "plugins.json"), nil
}

// GetPluginConfigPath returns the plugin config file path (public helper)
func GetPluginConfigPath() (string, error) {
	return getPluginConfigPath()
}
