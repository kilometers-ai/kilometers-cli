package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// loadSubscriptionConfig loads subscription from file
func loadSubscriptionConfig() (*SubscriptionConfig, error) {
	configPath, err := getSubscriptionConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config SubscriptionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse subscription config: %w", err)
	}

	return &config, nil
}

// saveSubscriptionConfig saves subscription to file
func saveSubscriptionConfig(config *SubscriptionConfig) error {
	configPath, err := getSubscriptionConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get subscription config path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscription config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write subscription config file: %w", err)
	}

	return nil
}

// getSubscriptionConfigPath returns the path to the subscription config file
func getSubscriptionConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "kilometers", "subscription.json"), nil
}
