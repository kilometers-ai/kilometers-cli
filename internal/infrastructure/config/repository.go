package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"kilometers.ai/cli/internal/application/ports"
)

// CompositeConfigRepository implements the ConfigurationRepository interface
type CompositeConfigRepository struct {
	sources    []ConfigSource
	cache      *ConfigCache
	configPath string
}

// ConfigSource defines the interface for configuration sources
type ConfigSource interface {
	Load() (*ports.Configuration, error)
	Priority() int
	Name() string
}

// ConfigCache provides caching for configuration
type ConfigCache struct {
	config    *ports.Configuration
	timestamp time.Time
	ttl       time.Duration
}

// NewCompositeConfigRepository creates a new configuration repository
func NewCompositeConfigRepository() *CompositeConfigRepository {
	// Check for config file from environment variable first
	configPath := os.Getenv("KM_CONFIG_FILE")
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	repo := &CompositeConfigRepository{
		sources: make([]ConfigSource, 0),
		cache: &ConfigCache{
			ttl: 5 * time.Minute,
		},
		configPath: configPath,
	}

	// Add default sources in priority order
	repo.AddSource(NewEnvironmentConfigSource())
	repo.AddSource(NewFileConfigSource(repo.configPath))

	return repo
}

// AddSource adds a configuration source
func (r *CompositeConfigRepository) AddSource(source ConfigSource) {
	r.sources = append(r.sources, source)
}

// Load retrieves the current configuration
func (r *CompositeConfigRepository) Load() (*ports.Configuration, error) {
	// Check cache first
	if r.cache.config != nil && time.Since(r.cache.timestamp) < r.cache.ttl {
		return r.cache.config, nil
	}

	// Start with default configuration
	config := r.LoadDefault()

	// Sort sources by priority (lower number = higher priority)
	sortedSources := make([]ConfigSource, len(r.sources))
	copy(sortedSources, r.sources)

	// Simple bubble sort by priority
	for i := 0; i < len(sortedSources)-1; i++ {
		for j := 0; j < len(sortedSources)-i-1; j++ {
			if sortedSources[j].Priority() > sortedSources[j+1].Priority() {
				sortedSources[j], sortedSources[j+1] = sortedSources[j+1], sortedSources[j]
			}
		}
	}

	// Check if we have an explicitly set config file that we should fail on
	explicitConfigFile := os.Getenv("KM_CONFIG_FILE") != ""

	// Apply sources in priority order (higher priority overwrites lower)
	for _, source := range sortedSources {
		sourceConfig, err := source.Load()
		if err != nil {
			// If this is a file source and we have an explicitly set config file,
			// propagate the error instead of silently ignoring it
			if explicitConfigFile && source.Name() == "file" {
				return nil, fmt.Errorf("failed to load explicitly set config file: %w", err)
			}
			// Otherwise, log error but continue with other sources
			continue
		}

		if sourceConfig != nil {
			// Merge configurations
			config = r.mergeConfigurations(config, sourceConfig)
		}
	}

	// Validate final configuration
	if err := r.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Cache the result
	r.cache.config = config
	r.cache.timestamp = time.Now()

	return config, nil
}

// Save persists the configuration
func (r *CompositeConfigRepository) Save(config *ports.Configuration) error {
	if err := r.Validate(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(r.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	// Invalidate cache
	r.cache.config = nil

	return nil
}

// LoadDefault returns the default configuration
func (r *CompositeConfigRepository) LoadDefault() *ports.Configuration {
	return &ports.Configuration{
		APIHost:   "https://api.kilometers.ai",
		APIKey:    "",
		BatchSize: 10,
		Debug:     false,
	}
}

// Validate validates the configuration
func (r *CompositeConfigRepository) Validate(config *ports.Configuration) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if config.APIHost == "" {
		return fmt.Errorf("API host is required")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (r *CompositeConfigRepository) GetConfigPath() string {
	return r.configPath
}

// ClearCache invalidates the configuration cache
func (r *CompositeConfigRepository) ClearCache() {
	r.cache.config = nil
	r.cache.timestamp = time.Time{}
}

// BackupConfig creates a backup of the current configuration
func (r *CompositeConfigRepository) BackupConfig() error {
	if _, err := os.Stat(r.configPath); os.IsNotExist(err) {
		return nil // No config file to backup
	}

	backupPath := r.configPath + ".backup." + time.Now().Format("20060102-150405")

	data, err := os.ReadFile(r.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}

	return nil
}

// RestoreConfig restores configuration from backup
func (r *CompositeConfigRepository) RestoreConfig() error {
	backupPattern := r.configPath + ".backup.*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		return fmt.Errorf("failed to find backup files: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no backup files found")
	}

	// Use the most recent backup
	latestBackup := matches[len(matches)-1]

	data, err := os.ReadFile(latestBackup)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(r.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config file: %w", err)
	}

	// Invalidate cache
	r.cache.config = nil

	return nil
}

// mergeConfigurations merges two configurations (source overwrites target)
func (r *CompositeConfigRepository) mergeConfigurations(target, source *ports.Configuration) *ports.Configuration {
	if source == nil {
		return target
	}
	if target == nil {
		return source
	}

	result := *target // Copy target

	// Override with source values if they are not zero values
	if source.APIHost != "" {
		result.APIHost = source.APIHost
	}
	if source.APIKey != "" {
		result.APIKey = source.APIKey
	}
	if source.BatchSize != 0 {
		result.BatchSize = source.BatchSize
	}
	// Boolean fields - always override
	result.Debug = source.Debug

	return &result
}

// FileConfigSource loads configuration from a JSON file
type FileConfigSource struct {
	filePath string
}

// NewFileConfigSource creates a new file configuration source
func NewFileConfigSource(filePath string) *FileConfigSource {
	return &FileConfigSource{
		filePath: filePath,
	}
}

// Load loads configuration from file
func (f *FileConfigSource) Load() (*ports.Configuration, error) {
	// Always check for current KM_CONFIG_FILE environment variable
	configPath := f.filePath
	if envPath := os.Getenv("KM_CONFIG_FILE"); envPath != "" {
		configPath = envPath
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If the config file is explicitly set via environment variable,
		// we should return an error for missing files
		if os.Getenv("KM_CONFIG_FILE") != "" {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		// Otherwise, file doesn't exist, return nil config (no error)
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ports.Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid config file format: %w", err)
	}

	return &config, nil
}

// Priority returns the priority of this source (lower number = higher priority)
func (f *FileConfigSource) Priority() int {
	return 100 // Low priority
}

// Name returns the name of this source
func (f *FileConfigSource) Name() string {
	return "file"
}

// EnvironmentConfigSource loads configuration from environment variables
type EnvironmentConfigSource struct{}

// NewEnvironmentConfigSource creates a new environment configuration source
func NewEnvironmentConfigSource() *EnvironmentConfigSource {
	return &EnvironmentConfigSource{}
}

// Load loads configuration from environment variables
func (e *EnvironmentConfigSource) Load() (*ports.Configuration, error) {
	config := &ports.Configuration{}

	// API Configuration
	if val := os.Getenv("KM_API_HOST"); val != "" {
		config.APIHost = val
	}
	if val := os.Getenv("KM_API_URL"); val != "" {
		config.APIHost = val
	}
	if val := os.Getenv("KILOMETERS_API_HOST"); val != "" {
		config.APIHost = val
	}
	if val := os.Getenv("KM_API_KEY"); val != "" {
		config.APIKey = val
	}
	if val := os.Getenv("KILOMETERS_API_KEY"); val != "" {
		config.APIKey = val
	}

	// Basic Configuration
	if val := os.Getenv("KM_BATCH_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.BatchSize = size
		}
	}
	if val := os.Getenv("KM_DEBUG"); val == "true" {
		config.Debug = true
	}

	return config, nil
}

// Priority returns the priority of this source (lower number = higher priority)
func (e *EnvironmentConfigSource) Priority() int {
	return 10 // High priority
}

// Name returns the name of this source
func (e *EnvironmentConfigSource) Name() string {
	return "environment"
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".kilometers-config.json"
	}

	return filepath.Join(homeDir, ".config", "kilometers", "config.json")
}

// ApplicationConfig represents the main application configuration
type ApplicationConfig struct {
	APIHost   string `json:"api_host"`
	APIKey    string `json:"api_key"`
	BatchSize int    `json:"batch_size"`
	Debug     bool   `json:"debug"`
}

// DefaultApplicationConfig returns the default application configuration
func DefaultApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		APIHost:   "https://api.kilometers.ai",
		APIKey:    "",
		BatchSize: 10,
		Debug:     false,
	}
}

// Validate validates the application configuration
func (config *ApplicationConfig) Validate() error {
	if config.APIHost == "" {
		return fmt.Errorf("API host is required")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	return nil
}
