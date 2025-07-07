package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds the CLI configuration
type Config struct {
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key,omitempty"`
	BatchSize   int    `json:"batch_size"`
	Debug       bool   `json:"debug"`

	// Advanced filtering features
	EnableRiskDetection bool     `json:"enable_risk_detection"`
	MethodWhitelist     []string `json:"method_whitelist"`
	PayloadSizeLimit    int      `json:"payload_size_limit"` // bytes, 0 = no limit
	HighRiskMethodsOnly bool     `json:"high_risk_methods_only"`
	ExcludePingMessages bool     `json:"exclude_ping_messages"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		APIEndpoint:         "http://localhost:5194",
		BatchSize:           10,
		Debug:               false,
		EnableRiskDetection: false,
		MethodWhitelist:     []string{}, // empty = capture all methods
		PayloadSizeLimit:    0,          // 0 = no limit
		HighRiskMethodsOnly: false,
		ExcludePingMessages: true, // exclude ping by default for noise reduction
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Override with environment variables if present
	if endpoint := os.Getenv("KILOMETERS_API_URL"); endpoint != "" {
		config.APIEndpoint = endpoint
	}

	if apiKey := os.Getenv("KILOMETERS_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	if os.Getenv("KM_DEBUG") == "true" {
		config.Debug = true
	}

	// Read batch size from environment if set
	if batchSize := os.Getenv("KM_BATCH_SIZE"); batchSize != "" {
		if size, err := strconv.Atoi(batchSize); err == nil && size > 0 {
			config.BatchSize = size
		}
	}

	// Advanced filtering environment variables
	if os.Getenv("KM_ENABLE_RISK_DETECTION") == "true" {
		config.EnableRiskDetection = true
	}

	if methodList := os.Getenv("KM_METHOD_WHITELIST"); methodList != "" {
		config.MethodWhitelist = strings.Split(methodList, ",")
		// Trim whitespace from each method
		for i, method := range config.MethodWhitelist {
			config.MethodWhitelist[i] = strings.TrimSpace(method)
		}
	}

	if payloadLimit := os.Getenv("KM_PAYLOAD_SIZE_LIMIT"); payloadLimit != "" {
		if limit, err := strconv.Atoi(payloadLimit); err == nil && limit >= 0 {
			config.PayloadSizeLimit = limit
		}
	}

	if os.Getenv("KM_HIGH_RISK_ONLY") == "true" {
		config.HighRiskMethodsOnly = true
		config.EnableRiskDetection = true // auto-enable risk detection
	}

	if os.Getenv("KM_EXCLUDE_PING") == "false" {
		config.ExcludePingMessages = false
	}

	// Try to load from config file
	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := getConfigPath()

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".kilometers-config.json"
	}

	return filepath.Join(homeDir, ".config", "kilometers", "config.json")
}

// ValidateConfig performs validation on the configuration
func ValidateConfig(config *Config) error {
	// Validate payload size limit
	if config.PayloadSizeLimit < 0 {
		config.PayloadSizeLimit = 0
	}

	// Validate batch size
	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}

	// Auto-enable risk detection if high-risk-only is enabled
	if config.HighRiskMethodsOnly {
		config.EnableRiskDetection = true
	}

	return nil
}
