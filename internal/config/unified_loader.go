package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LoadOptions provides configuration for how config should be loaded
type LoadOptions struct {
	// IncludeSources specifies which sources to include (empty = all)
	IncludeSources []string

	// ExcludeSources specifies which sources to exclude
	ExcludeSources []string

	// OverrideValues allows direct value overrides (typically from CLI flags)
	OverrideValues map[string]interface{}

	// ShowProgress indicates whether to show loading progress
	ShowProgress bool

	// FailOnValidation indicates whether to fail on validation errors
	FailOnValidation bool
}

// UnifiedLoader implements configuration loading from various sources
type UnifiedLoader struct {
	filesystemScanner *SimpleFileSystemScanner
}

// NewUnifiedLoader creates a new unified configuration loader
func NewUnifiedLoader() *UnifiedLoader {
	return &UnifiedLoader{
		filesystemScanner: NewSimpleFileSystemScanner(),
	}
}

// Load loads configuration from all available sources with proper precedence
func (l *UnifiedLoader) Load(ctx context.Context) (*UnifiedConfig, error) {
	return l.LoadWithOptions(ctx, LoadOptions{})
}

// LoadWithOptions loads configuration with specific loading options
func (l *UnifiedLoader) LoadWithOptions(ctx context.Context, opts LoadOptions) (*UnifiedConfig, error) {
	config := DefaultUnifiedConfig()

	// Apply CLI overrides first (highest priority)
	if opts.OverrideValues != nil {
		for field, value := range opts.OverrideValues {
			err := config.SetValue(field, "cli", "command_line_flag", value, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to set CLI override for %s: %w", field, err)
			}
		}
	}

	// Process environment variables with standardized naming
	l.loadEnvironmentVariables(config)

	// Scan filesystem configuration unless excluded
	if !l.isExcluded("filesystem", opts) && l.isIncluded("filesystem", opts) {
		scanResult, err := l.filesystemScanner.Scan(ctx)
		if err == nil {
			// Merge results with proper precedence
			l.mergeConfigs(config, scanResult, l.filesystemScanner.Name(), l.filesystemScanner.Priority())
		}
	}

	return config, nil
}

// loadEnvironmentVariables loads configuration from standardized environment variables
func (l *UnifiedLoader) loadEnvironmentVariables(config *UnifiedConfig) {
	envMappings := map[string]string{
		"KM_API_KEY":        "api_key",
		"KM_API_ENDPOINT":   "api_endpoint",
		"KM_BUFFER_SIZE":    "buffer_size",
		"KM_BATCH_SIZE":     "batch_size",
		"KM_LOG_LEVEL":      "log_level",
		"KM_DEBUG":          "debug",
		"KM_PLUGINS_DIR":    "plugins_dir",
		"KM_AUTO_PROVISION": "auto_provision",
		"KM_TIMEOUT":        "default_timeout",
	}

	for envVar, configField := range envMappings {
		if value := os.Getenv(envVar); value != "" {
			var convertedValue interface{} = value

			// Convert values based on field type
			switch configField {
			case "buffer_size", "batch_size":
				if intVal, err := strconv.Atoi(value); err == nil {
					convertedValue = intVal
				}
			case "debug", "auto_provision":
				if boolVal, err := strconv.ParseBool(value); err == nil {
					convertedValue = boolVal
				}
			case "default_timeout":
				if duration, err := time.ParseDuration(value); err == nil {
					convertedValue = duration
				}
			}

			// All KM_* environment variables have priority 2
			priority := 2

			config.SetValue(configField, "env", envVar, convertedValue, priority)
		}
	}
}

// isExcluded checks if a source is excluded
func (l *UnifiedLoader) isExcluded(source string, opts LoadOptions) bool {
	for _, excluded := range opts.ExcludeSources {
		if source == excluded {
			return true
		}
	}
	return false
}

// isIncluded checks if a source is included (or if no include list is specified)
func (l *UnifiedLoader) isIncluded(source string, opts LoadOptions) bool {
	if len(opts.IncludeSources) == 0 {
		return true
	}
	for _, included := range opts.IncludeSources {
		if source == included {
			return true
		}
	}
	return false
}

// mergeConfigs merges configuration from a scanner into the main config
func (l *UnifiedLoader) mergeConfigs(main, scanned *UnifiedConfig, scannerName string, basePriority int) {
	if scanned == nil {
		return
	}

	// Merge each field from the scanned config
	fieldMappings := map[string]interface{}{
		"api_key":         scanned.APIKey,
		"api_endpoint":    scanned.APIEndpoint,
		"buffer_size":     scanned.BufferSize,
		"batch_size":      scanned.BatchSize,
		"log_level":       scanned.LogLevel,
		"debug":           scanned.Debug,
		"plugins_dir":     scanned.PluginsDir,
		"auto_provision":  scanned.AutoProvision,
		"default_timeout": scanned.DefaultTimeout,
	}

	for field, value := range fieldMappings {
		// Skip empty/zero values
		if l.isEmptyValue(value) {
			continue
		}

		sourcePath := fmt.Sprintf("%s:%s", scannerName, field)
		if source, exists := scanned.Sources[field]; exists {
			sourcePath = source.SourcePath
		}

		main.SetValue(field, scannerName, sourcePath, value, basePriority)
	}
}

// isEmptyValue checks if a value is considered empty for configuration purposes
func (l *UnifiedLoader) isEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case int:
		return v == 0
	case bool:
		return false // false is a valid value
	case time.Duration:
		return v == 0
	default:
		return value == nil
	}
}

// SimpleFileSystemScanner implements basic file system configuration scanning
type SimpleFileSystemScanner struct{}

// NewSimpleFileSystemScanner creates a new filesystem scanner
func NewSimpleFileSystemScanner() *SimpleFileSystemScanner {
	return &SimpleFileSystemScanner{}
}

// Name returns the scanner name
func (s *SimpleFileSystemScanner) Name() string {
	return "filesystem"
}

// Priority returns the priority for filesystem configuration
func (s *SimpleFileSystemScanner) Priority() int {
	return 4 // Lower priority than environment
}

// Scan discovers configuration from filesystem (.env files and saved config)
func (s *SimpleFileSystemScanner) Scan(ctx context.Context) (*UnifiedConfig, error) {
	config := DefaultUnifiedConfig()
	homeDir, _ := os.UserHomeDir()
	workDir, _ := os.Getwd()

	searchPaths := []string{
		workDir, // Current directory (.env files)
		filepath.Join(homeDir, ".config", "kilometers"), // User config directory
	}

	for _, searchPath := range searchPaths {
		// Check for .env files
		envPath := filepath.Join(searchPath, ".env")
		if _, err := os.Stat(envPath); err == nil {
			s.loadEnvFile(envPath, config)
		}
	}

	// Load saved configuration from unified storage
	storage, err := NewUnifiedStorage()
	if err == nil {
		if savedConfig, err := storage.LoadFromStorage(ctx); err == nil && savedConfig != nil {
			// Merge saved config values (priority 3)
			for field, source := range savedConfig.Sources {
				config.SetValue(field, source.Source, source.SourcePath, source.Value, 3)
			}
		}
	}

	return config, nil
}

// loadEnvFile loads configuration from a .env file
func (s *SimpleFileSystemScanner) loadEnvFile(path string, config *UnifiedConfig) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
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
		case "KM_API_KEY":
			config.SetValue("api_key", "file", fmt.Sprintf("%s:%s", path, key), value, 4)
		case "KM_API_ENDPOINT":
			config.SetValue("api_endpoint", "file", fmt.Sprintf("%s:%s", path, key), value, 4)
		case "KM_BUFFER_SIZE":
			if intVal, err := strconv.Atoi(value); err == nil {
				config.SetValue("buffer_size", "file", fmt.Sprintf("%s:%s", path, key), intVal, 4)
			}
		case "KM_LOG_LEVEL":
			config.SetValue("log_level", "file", fmt.Sprintf("%s:%s", path, key), value, 4)
		case "KM_DEBUG":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				config.SetValue("debug", "file", fmt.Sprintf("%s:%s", path, key), boolVal, 4)
			}
		case "KM_PLUGINS_DIR":
			config.SetValue("plugins_dir", "file", fmt.Sprintf("%s:%s", path, key), value, 4)
		}
	}
}
