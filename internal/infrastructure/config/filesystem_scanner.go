package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"gopkg.in/yaml.v3"
)

// FileSystemScanner scans the file system for configuration files
type FileSystemScanner struct {
	searchPaths  []string
	filePatterns []string
}

// NewFileSystemScanner creates a new file system scanner
func NewFileSystemScanner() *FileSystemScanner {
	homeDir, _ := os.UserHomeDir()
	workDir, _ := os.Getwd()

	return &FileSystemScanner{
		searchPaths: []string{
			workDir,                                         // Current directory
			filepath.Join(homeDir, ".km"),                   // ~/.km/
			filepath.Join(homeDir, ".kilometers"),           // ~/.kilometers/
			filepath.Join(homeDir, ".config", "km"),         // ~/.config/km/
			filepath.Join(homeDir, ".config", "kilometers"), // ~/.config/kilometers/
			"/etc/kilometers",                               // System-wide config
		},
		filePatterns: []string{
			"config.yaml",
			"config.yml",
			"config.json",
			"km.config.yaml",
			"km.config.yml",
			"km.config.json",
			"kilometers.yaml",
			"kilometers.yml",
			"kilometers.json",
			".kmrc",
			".kilometersrc",
		},
	}
}

// SetSearchPaths sets the paths to search for config files
func (s *FileSystemScanner) SetSearchPaths(paths []string) {
	s.searchPaths = paths
}

// SetFilePatterns sets the file patterns to match
func (s *FileSystemScanner) SetFilePatterns(patterns []string) {
	s.filePatterns = patterns
}

// Name returns the name of this scanner
func (s *FileSystemScanner) Name() string {
	return "filesystem"
}

// Scan searches for configuration files in the file system
func (s *FileSystemScanner) Scan(ctx context.Context) (*domain.DiscoveredConfig, error) {
	config := &domain.DiscoveredConfig{
		DiscoveredAt: time.Now(),
		TotalSources: 0,
		Warnings:     []string{},
	}

	// Track which config files we've already processed to avoid duplicates
	processedFiles := make(map[string]bool)

	for _, searchPath := range s.searchPaths {
		// Skip if path doesn't exist
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		for _, pattern := range s.filePatterns {
			configPath := filepath.Join(searchPath, pattern)

			// Skip if already processed
			if processedFiles[configPath] {
				continue
			}

			// Check if file exists
			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				processedFiles[configPath] = true

				// Load and merge config
				if fileConfig, err := s.loadConfigFile(configPath); err == nil {
					config.Merge(fileConfig)
					config.TotalSources++
				} else {
					config.Warnings = append(config.Warnings,
						fmt.Sprintf("Failed to load config from %s: %v", configPath, err))
				}
			}
		}

		// Also check for .env files for API configuration
		envPath := filepath.Join(searchPath, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if envConfig, err := s.loadEnvFile(envPath); err == nil {
				config.Merge(envConfig)
				config.TotalSources++
			}
		}
	}

	return config, nil
}

// loadConfigFile loads configuration from a specific file
func (s *FileSystemScanner) loadConfigFile(path string) (*domain.DiscoveredConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine source priority based on path
	source := s.determineSource(path)

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(path))
	var configMap map[string]interface{}

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &configMap); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &configMap); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		// Try JSON first, then YAML
		if err := json.Unmarshal(data, &configMap); err != nil {
			if err := yaml.Unmarshal(data, &configMap); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	return s.mapToDiscoveredConfig(configMap, source, path), nil
}

// loadEnvFile loads configuration from a .env file
func (s *FileSystemScanner) loadEnvFile(path string) (*domain.DiscoveredConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &domain.DiscoveredConfig{
		DiscoveredAt: time.Now(),
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		switch strings.ToUpper(key) {
		case "KILOMETERS_API_KEY", "API_KEY":
			config.APIKey = &domain.ConfigValue{
				Value:      value,
				Source:     s.determineSource(path),
				SourcePath: fmt.Sprintf("%s:%s", path, key),
				Confidence: 0.9,
			}
		case "KILOMETERS_API_ENDPOINT", "API_ENDPOINT", "API_URL":
			config.APIEndpoint = &domain.ConfigValue{
				Value:      value,
				Source:     s.determineSource(path),
				SourcePath: fmt.Sprintf("%s:%s", path, key),
				Confidence: 0.9,
			}
		}
	}

	return config, nil
}

// mapToDiscoveredConfig converts a map to DiscoveredConfig
func (s *FileSystemScanner) mapToDiscoveredConfig(m map[string]interface{}, source domain.DiscoverySource, path string) *domain.DiscoveredConfig {
	config := &domain.DiscoveredConfig{
		DiscoveredAt: time.Now(),
		Warnings:     []string{},
	}

	// Map known fields
	if v, ok := getStringValue(m, "api_key", "apiKey", "ApiKey"); ok {
		config.APIKey = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getStringValue(m, "api_endpoint", "apiEndpoint", "ApiEndpoint", "api_url", "endpoint"); ok {
		config.APIEndpoint = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getIntValue(m, "buffer_size", "bufferSize", "BufferSize"); ok {
		config.BufferSize = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getStringValue(m, "log_level", "logLevel", "LogLevel"); ok {
		config.LogLevel = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	} else if v, ok := getBoolValue(m, "debug", "Debug"); ok && v {
		config.LogLevel = &domain.ConfigValue{
			Value:      "debug",
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getStringValue(m, "plugins_dir", "pluginsDir", "PluginsDir"); ok {
		config.PluginsDir = &domain.ConfigValue{
			Value:      expandPath(v),
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getBoolValue(m, "auto_provision", "autoProvision", "AutoProvision"); ok {
		config.AutoProvision = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	if v, ok := getDurationValue(m, "default_timeout", "defaultTimeout", "DefaultTimeout", "timeout"); ok {
		config.DefaultTimeout = &domain.ConfigValue{
			Value:      v,
			Source:     source,
			SourcePath: path,
			Confidence: 1.0,
		}
	}

	return config
}

// determineSource determines the discovery source based on file path
func (s *FileSystemScanner) determineSource(path string) domain.DiscoverySource {
	absPath, _ := filepath.Abs(path)
	workDir, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	// Check if it's a project-local config
	if strings.HasPrefix(absPath, workDir) {
		return domain.SourceProjectConfig
	}

	// Check if it's a user config
	if strings.HasPrefix(absPath, homeDir) {
		return domain.SourceUserConfig
	}

	// Otherwise it's a system config
	return domain.SourceSystemConfig
}

// Helper functions to extract values from maps with multiple possible keys
func getStringValue(m map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if str, ok := v.(string); ok {
				return str, true
			}
		}
	}
	return "", false
}

func getIntValue(m map[string]interface{}, keys ...string) (int, bool) {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case int:
				return val, true
			case int64:
				return int(val), true
			case float64:
				return int(val), true
			case string:
				if size, err := parseSize(val); err == nil {
					return size, true
				}
			}
		}
	}
	return 0, false
}

func getBoolValue(m map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if b, ok := v.(bool); ok {
				return b, true
			}
		}
	}
	return false, false
}

func getDurationValue(m map[string]interface{}, keys ...string) (time.Duration, bool) {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case string:
				if d, err := time.ParseDuration(val); err == nil {
					return d, true
				}
			case int:
				return time.Duration(val) * time.Second, true
			case float64:
				return time.Duration(val) * time.Second, true
			}
		}
	}
	return 0, false
}

