package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UnifiedStorage implements the ConfigStorage interface
type UnifiedStorage struct {
	configPath string
}

// NewUnifiedStorage creates a new unified configuration storage
func NewUnifiedStorage() (*UnifiedStorage, error) {
	configPath, err := getDefaultConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	return &UnifiedStorage{
		configPath: configPath,
	}, nil
}

// NewUnifiedStorageWithPath creates storage with a specific path
func NewUnifiedStorageWithPath(path string) *UnifiedStorage {
	return &UnifiedStorage{
		configPath: path,
	}
}

// Save saves configuration to persistent storage
func (s *UnifiedStorage) Save(ctx context.Context, config *UnifiedConfig) error {
	return s.SaveToPath(ctx, config, s.configPath)
}

// SaveToPath saves configuration to a specific path
func (s *UnifiedStorage) SaveToPath(ctx context.Context, config *UnifiedConfig, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create a clean config for saving (without internal metadata)
	saveConfig := &SaveableConfig{
		APIKey:         config.APIKey,
		APIEndpoint:    config.APIEndpoint,
		BufferSize:     config.BufferSize,
		BatchSize:      config.BatchSize,
		LogLevel:       config.LogLevel,
		Debug:          config.Debug,
		PluginsDir:     config.PluginsDir,
		AutoProvision:  config.AutoProvision,
		DefaultTimeout: config.DefaultTimeout,
		SavedAt:        time.Now(),
		Version:        "2.0", // Mark as new unified config format
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(saveConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with secure permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Delete removes configuration from persistent storage
func (s *UnifiedStorage) Delete(ctx context.Context) error {
	if err := os.Remove(s.configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %w", err)
	}
	return nil
}

// Exists checks if configuration exists in persistent storage
func (s *UnifiedStorage) Exists(ctx context.Context) (bool, error) {
	_, err := os.Stat(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check config file: %w", err)
	}
	return true, nil
}

// GetConfigPath returns the path where configuration is stored
func (s *UnifiedStorage) GetConfigPath() (string, error) {
	return s.configPath, nil
}

// LoadFromStorage loads configuration from storage (used by the unified loader)
func (s *UnifiedStorage) LoadFromStorage(ctx context.Context) (*UnifiedConfig, error) {
	// Check if file exists
	exists, err := s.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return DefaultUnifiedConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to parse as new unified config format first
	var saveableConfig SaveableConfig
	if err := json.Unmarshal(data, &saveableConfig); err == nil && saveableConfig.Version == "2.0" {
		return s.fromSaveableConfig(&saveableConfig), nil
	}

	// If we can't parse as new format, return default config
	// Legacy config files are no longer supported - use unified config format
	return DefaultUnifiedConfig(), nil
}

// SaveableConfig represents the configuration format saved to disk
type SaveableConfig struct {
	APIKey         string        `json:"api_key,omitempty"`
	APIEndpoint    string        `json:"api_endpoint,omitempty"`
	BufferSize     int           `json:"buffer_size,omitempty"`
	BatchSize      int           `json:"batch_size,omitempty"`
	LogLevel       string        `json:"log_level,omitempty"`
	Debug          bool          `json:"debug,omitempty"`
	PluginsDir     string        `json:"plugins_dir,omitempty"`
	AutoProvision  bool          `json:"auto_provision,omitempty"`
	DefaultTimeout time.Duration `json:"default_timeout,omitempty"`
	SavedAt        time.Time     `json:"saved_at"`
	Version        string        `json:"version"` // Format version
}

// fromSaveableConfig converts SaveableConfig to UnifiedConfig
func (s *UnifiedStorage) fromSaveableConfig(sc *SaveableConfig) *UnifiedConfig {
	config := DefaultUnifiedConfig()

	if sc.APIKey != "" {
		config.SetValue("api_key", "file", s.configPath, sc.APIKey, 4)
	}
	if sc.APIEndpoint != "" {
		config.SetValue("api_endpoint", "file", s.configPath, sc.APIEndpoint, 4)
	}
	if sc.BufferSize != 0 {
		config.SetValue("buffer_size", "file", s.configPath, sc.BufferSize, 4)
	}
	if sc.BatchSize != 0 {
		config.SetValue("batch_size", "file", s.configPath, sc.BatchSize, 4)
	}
	if sc.LogLevel != "" {
		config.SetValue("log_level", "file", s.configPath, sc.LogLevel, 4)
	}
	if sc.Debug {
		config.SetValue("debug", "file", s.configPath, sc.Debug, 4)
	}
	if sc.PluginsDir != "" {
		config.SetValue("plugins_dir", "file", s.configPath, sc.PluginsDir, 4)
	}
	if sc.AutoProvision {
		config.SetValue("auto_provision", "file", s.configPath, sc.AutoProvision, 4)
	}
	if sc.DefaultTimeout != 0 {
		config.SetValue("default_timeout", "file", s.configPath, sc.DefaultTimeout, 4)
	}

	return config
}

// getDefaultConfigPath returns the default path for configuration storage
func getDefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "kilometers", "config.json"), nil
}

// CreateConfigServiceFromDefaults creates default config loader and storage
// Returns (loader, storage, error)
func CreateConfigServiceFromDefaults() (*UnifiedLoader, *UnifiedStorage, error) {
	loader := NewUnifiedLoader()

	storage, err := NewUnifiedStorage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return loader, storage, nil
}
