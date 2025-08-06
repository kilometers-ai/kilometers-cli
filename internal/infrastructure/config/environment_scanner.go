package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// EnvironmentScanner scans environment variables for configuration
type EnvironmentScanner struct {
	prefixes []string
	mappings map[string]string // env var suffix -> config field mapping
}

// NewEnvironmentScanner creates a new environment scanner
func NewEnvironmentScanner() *EnvironmentScanner {
	return &EnvironmentScanner{
		prefixes: []string{"KILOMETERS_", "KM_"}, // Support both new and legacy prefixes
		mappings: map[string]string{
			"API_KEY":        "api_key",
			"API_ENDPOINT":   "api_endpoint",
			"API_URL":        "api_endpoint", // Alternative name
			"BUFFER_SIZE":    "buffer_size",
			"LOG_LEVEL":      "log_level",
			"DEBUG":          "debug",
			"PLUGINS_DIR":    "plugins_dir",
			"AUTO_PROVISION": "auto_provision",
			"TIMEOUT":        "default_timeout",
		},
	}
}

// SetPrefixes sets the environment variable prefixes to scan
func (s *EnvironmentScanner) SetPrefixes(prefixes []string) {
	s.prefixes = prefixes
}

// Name returns the name of this scanner
func (s *EnvironmentScanner) Name() string {
	return "environment"
}

// Scan searches for configuration values in environment variables
func (s *EnvironmentScanner) Scan(ctx context.Context) (*domain.DiscoveredConfig, error) {
	config := &domain.DiscoveredConfig{
		DiscoveredAt: time.Now(),
		TotalSources: 0,
		Warnings:     []string{},
	}

	// Get all environment variables
	environ := os.Environ()

	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if this env var matches our prefixes
		for _, prefix := range s.prefixes {
			if strings.HasPrefix(key, prefix) {
				suffix := strings.TrimPrefix(key, prefix)
				s.processEnvVar(config, key, suffix, value)
				config.TotalSources++
			}
		}
	}

	// Check for special cases without prefixes
	s.checkSpecialEnvVars(config)

	return config, nil
}

// processEnvVar processes a single environment variable
func (s *EnvironmentScanner) processEnvVar(config *domain.DiscoveredConfig, fullKey, suffix, value string) {
	// Map environment variable to config field
	configField, ok := s.mappings[suffix]
	if !ok {
		// Try lowercase mapping
		configField, ok = s.mappings[strings.ToUpper(suffix)]
		if !ok {
			config.Warnings = append(config.Warnings,
				fmt.Sprintf("Unknown environment variable: %s", fullKey))
			return
		}
	}

	configValue := &domain.ConfigValue{
		Source:     domain.SourceEnvironment,
		SourcePath: fullKey,
		Confidence: 1.0, // Environment variables have high confidence
	}

	switch configField {
	case "api_key":
		configValue.Value = value
		config.APIKey = configValue

	case "api_endpoint":
		configValue.Value = value
		config.APIEndpoint = configValue

	case "buffer_size":
		if size, err := parseSize(value); err == nil {
			configValue.Value = size
			config.BufferSize = configValue
		} else {
			config.Warnings = append(config.Warnings,
				fmt.Sprintf("Invalid buffer size in %s: %v", fullKey, err))
		}

	case "log_level":
		configValue.Value = strings.ToLower(value)
		config.LogLevel = configValue

	case "debug":
		if debug, err := strconv.ParseBool(value); err == nil {
			if debug {
				configValue.Value = "debug"
				config.LogLevel = configValue
			}
		}

	case "plugins_dir":
		configValue.Value = expandPath(value)
		config.PluginsDir = configValue

	case "auto_provision":
		if autoProvision, err := strconv.ParseBool(value); err == nil {
			configValue.Value = autoProvision
			config.AutoProvision = configValue
		}

	case "default_timeout":
		if timeout, err := time.ParseDuration(value); err == nil {
			configValue.Value = timeout
			config.DefaultTimeout = configValue
		} else {
			config.Warnings = append(config.Warnings,
				fmt.Sprintf("Invalid timeout in %s: %v", fullKey, err))
		}
	}
}

// checkSpecialEnvVars checks for environment variables without our standard prefixes
func (s *EnvironmentScanner) checkSpecialEnvVars(config *domain.DiscoveredConfig) {
	// Check for common API key patterns
	apiKeyPatterns := []string{
		"API_KEY",
		"APIKEY",
		"AUTH_TOKEN",
		"ACCESS_TOKEN",
	}

	for _, pattern := range apiKeyPatterns {
		if value := os.Getenv(pattern); value != "" && config.APIKey == nil {
			config.APIKey = &domain.ConfigValue{
				Value:      value,
				Source:     domain.SourceEnvironment,
				SourcePath: pattern,
				Confidence: 0.7, // Lower confidence for non-standard env vars
			}
			config.TotalSources++
			config.Warnings = append(config.Warnings,
				fmt.Sprintf("Found API key in non-standard environment variable: %s", pattern))
		}
	}

	// Check for CI/CD environment indicators
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("JENKINS_HOME") != "" {
		// In CI environment, check for different patterns
		if endpoint := os.Getenv("API_ENDPOINT"); endpoint != "" && config.APIEndpoint == nil {
			config.APIEndpoint = &domain.ConfigValue{
				Value:      endpoint,
				Source:     domain.SourceEnvironment,
				SourcePath: "API_ENDPOINT",
				Confidence: 0.8,
			}
			config.TotalSources++
		}
	}
}

// parseSize parses size strings like "1MB", "2048", "1024KB"
func parseSize(s string) (int, error) {
	s = strings.TrimSpace(strings.ToUpper(s))

	// Try to parse as plain number first
	if size, err := strconv.Atoi(s); err == nil {
		return size, nil
	}

	// Parse with units
	units := map[string]int{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	for unit, multiplier := range units {
		if strings.HasSuffix(s, unit) {
			numStr := strings.TrimSuffix(s, unit)
			if num, err := strconv.ParseFloat(numStr, 64); err == nil {
				return int(num * float64(multiplier)), nil
			}
		}
	}

	return 0, fmt.Errorf("invalid size format: %s", s)
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}
	return os.ExpandEnv(path)
}

