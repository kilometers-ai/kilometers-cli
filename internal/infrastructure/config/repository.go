package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	// Apply sources in priority order (higher priority overwrites lower)
	for _, source := range sortedSources {
		sourceConfig, err := source.Load()
		if err != nil {
			// Log error but continue with other sources
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
		APIEndpoint:           "https://api.dev.kilometers.ai",
		BatchSize:             10,
		FlushInterval:         30,
		Debug:                 false,
		EnableRiskDetection:   false,
		MethodWhitelist:       []string{},
		MethodBlacklist:       []string{},
		PayloadSizeLimit:      0,
		HighRiskMethodsOnly:   false,
		ExcludePingMessages:   true,
		MinimumRiskLevel:      "low",
		EnableLocalStorage:    false,
		StoragePath:           "",
		MaxStorageSize:        0,
		RetentionDays:         30,
		MaxConcurrentRequests: 10,
		RequestTimeout:        30,
		RetryAttempts:         3,
		RetryDelay:            1000,
	}
}

// Validate validates the configuration
func (r *CompositeConfigRepository) Validate(config *ports.Configuration) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if config.APIEndpoint == "" {
		return fmt.Errorf("API endpoint is required")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	if config.FlushInterval < 0 {
		return fmt.Errorf("flush interval cannot be negative")
	}

	if config.PayloadSizeLimit < 0 {
		return fmt.Errorf("payload size limit cannot be negative")
	}

	if config.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}

	if config.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max concurrent requests must be greater than 0")
	}

	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be greater than 0")
	}

	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}

	if config.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative")
	}

	// Validate minimum risk level
	validRiskLevels := []string{"low", "medium", "high"}
	isValid := false
	for _, level := range validRiskLevels {
		if config.MinimumRiskLevel == level {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("minimum risk level must be one of: %s", strings.Join(validRiskLevels, ", "))
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (r *CompositeConfigRepository) GetConfigPath() string {
	return r.configPath
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
	if source.APIEndpoint != "" {
		result.APIEndpoint = source.APIEndpoint
	}
	if source.APIKey != "" {
		result.APIKey = source.APIKey
	}
	if source.BatchSize != 0 {
		result.BatchSize = source.BatchSize
	}
	if source.FlushInterval != 0 {
		result.FlushInterval = source.FlushInterval
	}
	// Boolean fields - always override
	result.Debug = source.Debug
	result.EnableRiskDetection = source.EnableRiskDetection
	result.HighRiskMethodsOnly = source.HighRiskMethodsOnly
	result.ExcludePingMessages = source.ExcludePingMessages
	result.EnableLocalStorage = source.EnableLocalStorage

	// Slice fields - override if not empty
	if len(source.MethodWhitelist) > 0 {
		result.MethodWhitelist = source.MethodWhitelist
	}
	if len(source.MethodBlacklist) > 0 {
		result.MethodBlacklist = source.MethodBlacklist
	}

	// Integer fields - override if not zero
	// Special handling for PayloadSizeLimit: -1 means "not set" in environment config
	if source.PayloadSizeLimit > 0 {
		result.PayloadSizeLimit = source.PayloadSizeLimit
	}
	if source.MaxStorageSize != 0 {
		result.MaxStorageSize = source.MaxStorageSize
	}
	if source.RetentionDays != 0 {
		result.RetentionDays = source.RetentionDays
	}
	if source.MaxConcurrentRequests != 0 {
		result.MaxConcurrentRequests = source.MaxConcurrentRequests
	}
	if source.RequestTimeout != 0 {
		result.RequestTimeout = source.RequestTimeout
	}
	if source.RetryAttempts != 0 {
		result.RetryAttempts = source.RetryAttempts
	}
	if source.RetryDelay != 0 {
		result.RetryDelay = source.RetryDelay
	}

	// String fields - override if not empty
	if source.MinimumRiskLevel != "" {
		result.MinimumRiskLevel = source.MinimumRiskLevel
	}
	if source.StoragePath != "" {
		result.StoragePath = source.StoragePath
	}

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
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		return nil, nil // File doesn't exist, return nil config
	}

	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ports.Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
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
	if val := os.Getenv("KM_API_URL"); val != "" {
		config.APIEndpoint = val
	}
	if val := os.Getenv("KILOMETERS_API_URL"); val != "" {
		config.APIEndpoint = val
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
	if val := os.Getenv("KM_FLUSH_INTERVAL"); val != "" {
		if interval, err := strconv.Atoi(val); err == nil && interval >= 0 {
			config.FlushInterval = interval
		}
	}
	if val := os.Getenv("KM_DEBUG"); val == "true" {
		config.Debug = true
	}

	// Filtering Configuration
	if val := os.Getenv("KM_ENABLE_RISK_DETECTION"); val == "true" {
		config.EnableRiskDetection = true
	}
	if val := os.Getenv("KM_METHOD_WHITELIST"); val != "" {
		config.MethodWhitelist = strings.Split(val, ",")
		for i, method := range config.MethodWhitelist {
			config.MethodWhitelist[i] = strings.TrimSpace(method)
		}
	}
	if val := os.Getenv("KM_METHOD_BLACKLIST"); val != "" {
		config.MethodBlacklist = strings.Split(val, ",")
		for i, method := range config.MethodBlacklist {
			config.MethodBlacklist[i] = strings.TrimSpace(method)
		}
	}
	if val := os.Getenv("KM_PAYLOAD_SIZE_LIMIT"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil && limit >= 0 {
			config.PayloadSizeLimit = limit
		}
	}
	if val := os.Getenv("KM_HIGH_RISK_ONLY"); val == "true" {
		config.HighRiskMethodsOnly = true
		config.EnableRiskDetection = true // Auto-enable risk detection
	}
	if val := os.Getenv("KM_EXCLUDE_PING"); val == "false" {
		config.ExcludePingMessages = false
	} else if val == "" {
		// Default to true if not set
		config.ExcludePingMessages = true
	}
	if val := os.Getenv("KM_MINIMUM_RISK_LEVEL"); val != "" {
		validLevels := []string{"low", "medium", "high"}
		for _, level := range validLevels {
			if val == level {
				config.MinimumRiskLevel = val
				break
			}
		}
	}

	// Storage Configuration
	if val := os.Getenv("KM_ENABLE_LOCAL_STORAGE"); val == "true" {
		config.EnableLocalStorage = true
	}
	if val := os.Getenv("KM_STORAGE_PATH"); val != "" {
		config.StoragePath = val
	}
	if val := os.Getenv("KM_MAX_STORAGE_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil && size >= 0 {
			config.MaxStorageSize = size
		}
	}
	if val := os.Getenv("KM_RETENTION_DAYS"); val != "" {
		if days, err := strconv.Atoi(val); err == nil && days >= 0 {
			config.RetentionDays = days
		}
	}

	// Performance Configuration
	if val := os.Getenv("KM_MAX_CONCURRENT_REQUESTS"); val != "" {
		if max, err := strconv.Atoi(val); err == nil && max > 0 {
			config.MaxConcurrentRequests = max
		}
	}
	if val := os.Getenv("KM_REQUEST_TIMEOUT"); val != "" {
		if timeout, err := strconv.Atoi(val); err == nil && timeout > 0 {
			config.RequestTimeout = timeout
		}
	}
	if val := os.Getenv("KM_RETRY_ATTEMPTS"); val != "" {
		if attempts, err := strconv.Atoi(val); err == nil && attempts >= 0 {
			config.RetryAttempts = attempts
		}
	}
	if val := os.Getenv("KM_RETRY_DELAY"); val != "" {
		if delay, err := strconv.Atoi(val); err == nil && delay >= 0 {
			config.RetryDelay = delay
		}
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
