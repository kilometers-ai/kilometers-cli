package services

import (
	"context"
	"fmt"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// ConfigService orchestrates configuration loading, validation, and storage
type ConfigService struct {
	loader  ports.ConfigLoader
	storage ports.ConfigStorage
}

// NewConfigService creates a new configuration service
func NewConfigService(loader ports.ConfigLoader, storage ports.ConfigStorage) *ConfigService {
	return &ConfigService{
		loader:  loader,
		storage: storage,
	}
}

// Load loads configuration from all available sources
func (s *ConfigService) Load(ctx context.Context) (*domain.UnifiedConfig, error) {
	config, err := s.loader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadWithOptions loads configuration with specific options
func (s *ConfigService) LoadWithOptions(ctx context.Context, opts ports.LoadOptions) (*domain.UnifiedConfig, error) {
	config, err := s.loader.LoadWithOptions(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate if required
	if opts.FailOnValidation {
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return config, nil
}

// Save saves configuration to persistent storage
func (s *ConfigService) Save(ctx context.Context, config *domain.UnifiedConfig) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	if err := s.storage.Save(ctx, config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// UpdateAPIKey updates the API key and saves the configuration
func (s *ConfigService) UpdateAPIKey(ctx context.Context, apiKey string) error {
	// Load current config
	config, err := s.Load(ctx)
	if err != nil {
		// If no config exists, start with defaults
		config = domain.DefaultUnifiedConfig()
	}

	// Update API key
	err = config.SetValue("api_key", "user_update", "config_service", apiKey, 1)
	if err != nil {
		return fmt.Errorf("failed to set API key: %w", err)
	}

	// Save updated config
	return s.Save(ctx, config)
}

// ClearAPIKey removes the API key from configuration
func (s *ConfigService) ClearAPIKey(ctx context.Context) error {
	// Load current config
	config, err := s.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load current configuration: %w", err)
	}

	// Clear API key
	err = config.SetValue("api_key", "user_update", "config_service", "", 1)
	if err != nil {
		return fmt.Errorf("failed to clear API key: %w", err)
	}

	// Save updated config
	return s.Save(ctx, config)
}

// GetConfigStatus returns information about the current configuration
func (s *ConfigService) GetConfigStatus(ctx context.Context) (*ConfigStatus, error) {
	config, err := s.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	exists, _ := s.storage.Exists(ctx)
	configPath, _ := s.storage.GetConfigPath()

	status := &ConfigStatus{
		HasConfig:   exists,
		ConfigPath:  configPath,
		HasAPIKey:   config.HasAPIKey(),
		APIEndpoint: config.APIEndpoint,
		DebugMode:   config.IsDebugMode(),
		Sources:     make(map[string]SourceInfo),
	}

	// Add source information for transparency
	for field, source := range config.Sources {
		status.Sources[field] = SourceInfo{
			Value:      source.Value,
			Source:     source.Source,
			SourcePath: source.SourcePath,
			Priority:   source.Priority,
		}
	}

	return status, nil
}

// ConfigStatus provides information about the current configuration state
type ConfigStatus struct {
	HasConfig   bool                  `json:"has_config"`
	ConfigPath  string                `json:"config_path"`
	HasAPIKey   bool                  `json:"has_api_key"`
	APIEndpoint string                `json:"api_endpoint"`
	DebugMode   bool                  `json:"debug_mode"`
	Sources     map[string]SourceInfo `json:"sources"`
}

// SourceInfo provides information about where a config value came from
type SourceInfo struct {
	Value      interface{} `json:"value"`
	Source     string      `json:"source"`
	SourcePath string      `json:"source_path"`
	Priority   int         `json:"priority"`
}

// Migrate migrates configuration from legacy formats
func (s *ConfigService) Migrate(ctx context.Context) (*MigrationResult, error) {
	result := &MigrationResult{
		LegacyConfigFound: false,
		MigrationApplied:  false,
		Changes:           make(map[string]string),
	}

	// Legacy migration is no longer needed with unified config
	// All configuration is now handled through the unified system
	// which automatically loads from all available sources

	return result, nil
}

// MigrationResult provides information about configuration migration
type MigrationResult struct {
	LegacyConfigFound bool              `json:"legacy_config_found"`
	MigrationApplied  bool              `json:"migration_applied"`
	Changes           map[string]string `json:"changes"`
}
