package services

import (
	"context"
	"fmt"

	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/application/ports"
)

// ConfigurationService handles configuration management
type ConfigurationService struct {
	configRepo ports.ConfigurationRepository
	logger     ports.LoggingGateway
}

// NewConfigurationService creates a new configuration service
func NewConfigurationService(configRepo ports.ConfigurationRepository, logger ports.LoggingGateway) *ConfigurationService {
	return &ConfigurationService{
		configRepo: configRepo,
		logger:     logger,
	}
}

// LoadConfiguration loads the current configuration
func (s *ConfigurationService) LoadConfiguration(ctx context.Context) (*ports.Configuration, error) {
	config, err := s.configRepo.Load()
	if err != nil {
		s.logger.LogError(err, "Failed to load configuration", nil)
		// Return default configuration if loading fails
		defaultConfig := s.configRepo.LoadDefault()
		return defaultConfig, nil
	}

	// Validate configuration
	if err := s.configRepo.Validate(config); err != nil {
		s.logger.LogError(err, "Configuration validation failed", nil)
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// SaveConfiguration saves the configuration
func (s *ConfigurationService) SaveConfiguration(ctx context.Context, config *ports.Configuration) error {
	// Validate configuration before saving
	if err := s.configRepo.Validate(config); err != nil {
		s.logger.LogError(err, "Configuration validation failed", nil)
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create backup before saving
	if err := s.configRepo.BackupConfig(); err != nil {
		s.logger.LogError(err, "Failed to create configuration backup", nil)
		// Continue with save even if backup fails
	}

	// Save configuration
	if err := s.configRepo.Save(config); err != nil {
		s.logger.LogError(err, "Failed to save configuration", nil)
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	s.logger.Log(ports.LogLevelInfo, "Configuration saved successfully", map[string]interface{}{
		"config_path": s.configRepo.GetConfigPath(),
	})

	return nil
}

// GetDefaultConfiguration returns the default configuration
func (s *ConfigurationService) GetDefaultConfiguration(ctx context.Context) *ports.Configuration {
	return s.configRepo.LoadDefault()
}

// ValidateConfiguration validates a configuration
func (s *ConfigurationService) ValidateConfiguration(ctx context.Context, config *ports.Configuration) error {
	if err := s.configRepo.Validate(config); err != nil {
		s.logger.LogError(err, "Configuration validation failed", nil)
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}

// BackupConfiguration creates a backup of the current configuration
func (s *ConfigurationService) BackupConfiguration(ctx context.Context) error {
	if err := s.configRepo.BackupConfig(); err != nil {
		s.logger.LogError(err, "Failed to create configuration backup", nil)
		return fmt.Errorf("failed to create configuration backup: %w", err)
	}

	s.logger.Log(ports.LogLevelInfo, "Configuration backup created successfully", nil)
	return nil
}

// RestoreConfiguration restores configuration from backup
func (s *ConfigurationService) RestoreConfiguration(ctx context.Context) error {
	if err := s.configRepo.RestoreConfig(); err != nil {
		s.logger.LogError(err, "Failed to restore configuration", nil)
		return fmt.Errorf("failed to restore configuration: %w", err)
	}

	s.logger.Log(ports.LogLevelInfo, "Configuration restored successfully", nil)
	return nil
}

// GetConfigurationPath returns the path to the configuration file
func (s *ConfigurationService) GetConfigurationPath(ctx context.Context) string {
	return s.configRepo.GetConfigPath()
}

// InitializeConfiguration handles configuration initialization
func (s *ConfigurationService) InitializeConfiguration(ctx context.Context, cmd *commands.InitializeConfigurationCommand) (*commands.CommandResult, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return commands.NewErrorResult("Validation failed", []string{err.Error()}), nil
	}

	// Create configuration from command
	config := &ports.Configuration{
		APIEndpoint:           cmd.APIEndpoint,
		APIKey:                cmd.APIKey,
		BatchSize:             cmd.BatchSize,
		FlushInterval:         cmd.FlushInterval,
		Debug:                 cmd.Debug,
		EnableRiskDetection:   cmd.EnableRiskDetection,
		MethodWhitelist:       cmd.MethodWhitelist,
		MethodBlacklist:       cmd.MethodBlacklist,
		PayloadSizeLimit:      cmd.PayloadSizeLimit,
		HighRiskMethodsOnly:   cmd.HighRiskMethodsOnly,
		ExcludePingMessages:   cmd.ExcludePingMessages,
		MinimumRiskLevel:      cmd.MinimumRiskLevel,
		EnableLocalStorage:    cmd.EnableLocalStorage,
		StoragePath:           cmd.StoragePath,
		MaxStorageSize:        cmd.MaxStorageSize,
		RetentionDays:         cmd.RetentionDays,
		MaxConcurrentRequests: cmd.MaxConcurrentRequests,
		RequestTimeout:        cmd.RequestTimeout,
		RetryAttempts:         cmd.RetryAttempts,
		RetryDelay:            cmd.RetryDelay,
	}

	// Save configuration
	if err := s.SaveConfiguration(ctx, config); err != nil {
		return commands.NewErrorResult("Failed to save configuration", []string{err.Error()}), nil
	}

	result := commands.NewSuccessResult("Configuration initialized successfully", map[string]interface{}{
		"config_path":  s.configRepo.GetConfigPath(),
		"api_endpoint": config.APIEndpoint,
		"batch_size":   config.BatchSize,
	})

	result.SetMetadata("config_path", s.configRepo.GetConfigPath())
	return result, nil
}
