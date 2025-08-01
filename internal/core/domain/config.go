package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the CLI configuration
type Config struct {
	ApiKey      string `json:"api_key,omitempty"`
	ApiEndpoint string `json:"api_endpoint"`
	BatchSize   int    `json:"batch_size"`
	Debug       bool   `json:"debug"`
}

// MonitorConfig contains configuration for monitoring behavior
type MonitorConfig struct {
	BufferSize int
}

// DefaultConfig returns a config with default values
func DefaultConfig() Config {
	return Config{
		ApiEndpoint: "http://localhost:5194",
		BatchSize:   10,
		Debug:       false,
	}
}

// DefaultMonitorConfig returns sensible defaults for monitoring
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		BufferSize: 1024 * 1024, // 1MB buffer
	}
}

// LoadConfig loads configuration with precedence: env vars > file > defaults
func LoadConfig() Config {
	config := DefaultConfig()

	// Try to load from file
	if fileConfig, err := loadConfigFile(); err == nil {
		config = fileConfig
	}

	// Override with environment variables
	if apiKey := os.Getenv("KILOMETERS_API_KEY"); apiKey != "" {
		config.ApiKey = apiKey
	}
	if endpoint := os.Getenv("KILOMETERS_API_ENDPOINT"); endpoint != "" {
		config.ApiEndpoint = endpoint
	}

	return config
}

// SaveConfig saves configuration to file
func SaveConfig(config Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadConfigFile loads config from file
func loadConfigFile() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// configPathProvider is a function type for getting config path
type configPathProvider func() (string, error)

// defaultConfigPath is the default implementation
var getConfigPath configPathProvider = defaultGetConfigPath

// defaultGetConfigPath returns the default path to the config file
func defaultGetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "kilometers", "config.json"), nil
}

// GetConfigPath returns the config file path (public helper)
func GetConfigPath() (string, error) {
	return getConfigPath()
}

// Test helpers for overriding config path in tests

// SetTestConfigPath allows tests to override the config path
func SetTestConfigPath(testPath string) {
	getConfigPath = func() (string, error) {
		return testPath, nil
	}
}

// RestoreConfigPath restores the default config path behavior
func RestoreConfigPath(originalFunc func() (string, error)) {
	getConfigPath = defaultGetConfigPath
}
