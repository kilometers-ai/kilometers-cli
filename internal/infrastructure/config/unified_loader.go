package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// UnifiedLoader implements the ConfigLoader interface
type UnifiedLoader struct {
	scanners []ports.UnifiedConfigScanner
}

// NewUnifiedLoader creates a new unified configuration loader
func NewUnifiedLoader() *UnifiedLoader {
	return &UnifiedLoader{
		scanners: []ports.UnifiedConfigScanner{
			NewSimpleFileSystemScanner(), // File system scanner (includes .env and saved config)
		},
	}
}

// Load loads configuration from all available sources with proper precedence
func (l *UnifiedLoader) Load(ctx context.Context) (*domain.UnifiedConfig, error) {
	return l.LoadWithOptions(ctx, ports.LoadOptions{})
}

// LoadWithOptions loads configuration with specific loading options
func (l *UnifiedLoader) LoadWithOptions(ctx context.Context, opts ports.LoadOptions) (*domain.UnifiedConfig, error) {
	config := domain.DefaultUnifiedConfig()

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

	// Run all scanners
	for _, scanner := range l.scanners {
		// Skip excluded sources
		if len(opts.ExcludeSources) > 0 {
			skip := false
			for _, excluded := range opts.ExcludeSources {
				if scanner.Name() == excluded {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		// Skip if not in include list (when specified)
		if len(opts.IncludeSources) > 0 {
			include := false
			for _, included := range opts.IncludeSources {
				if scanner.Name() == included {
					include = true
					break
				}
			}
			if !include {
				continue
			}
		}

		// Scan for configuration
		scanResult, err := scanner.Scan(ctx)
		if err != nil {
			// Log error but continue with other scanners
			continue
		}

		// Merge results with proper precedence
		l.mergeConfigs(config, scanResult, scanner.Name(), scanner.Priority())
	}

	return config, nil
}

// loadEnvironmentVariables loads configuration from standardized environment variables
func (l *UnifiedLoader) loadEnvironmentVariables(config *domain.UnifiedConfig) {
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

// mergeConfigs merges configuration from a scanner into the main config
func (l *UnifiedLoader) mergeConfigs(main, scanned *domain.UnifiedConfig, scannerName string, basePriority int) {
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

// AddScanner adds a custom configuration scanner
func (l *UnifiedLoader) AddScanner(scanner ports.UnifiedConfigScanner) {
	l.scanners = append(l.scanners, scanner)
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
func (s *SimpleFileSystemScanner) Scan(ctx context.Context) (*domain.UnifiedConfig, error) {
	config := domain.DefaultUnifiedConfig()
	homeDir, _ := os.UserHomeDir()
	workDir, _ := os.Getwd()

	searchPaths := []string{
		workDir,                                         // Current directory (.env files)
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
func (s *SimpleFileSystemScanner) loadEnvFile(path string, config *domain.UnifiedConfig) {
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
		}
	}
}