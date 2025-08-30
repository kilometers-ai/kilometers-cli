package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	APIEndpoint      string `json:"api_endpoint"`
	APIKey           string `json:"api_key"`
	SubscriptionTier string `json:"subscription_tier"`
	LogLevel         string `json:"log_level"`
	PluginsDir       string `json:"plugins_dir"`

	// Plugin configuration
	Plugins map[string]map[string]interface{} `json:"plugins"`
}

func Load(configPath string) (*Config, error) {
	cfg := &Config{
		APIEndpoint:      "https://api.kilometers.ai",
		APIKey:           os.Getenv("KM_API_KEY"),
		SubscriptionTier: "free",
		LogLevel:         "info",
		PluginsDir:       "./plugins",
		Plugins:          make(map[string]map[string]interface{}),
	}

	if configPath == "" {
		configPath = os.Getenv("KM_CONFIG_PATH")
		if configPath == "" {
			configPath = "km-config.json"
		}
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
