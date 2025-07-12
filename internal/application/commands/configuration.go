package commands

import (
	"fmt"
	"strings"
)

// InitializeConfigurationCommand represents the initialize configuration command
type InitializeConfigurationCommand struct {
	BaseCommand
	APIEndpoint           string   `json:"api_endpoint"`
	APIKey                string   `json:"api_key"`
	BatchSize             int      `json:"batch_size"`
	FlushInterval         int      `json:"flush_interval"`
	Debug                 bool     `json:"debug"`
	EnableRiskDetection   bool     `json:"enable_risk_detection"`
	MethodWhitelist       []string `json:"method_whitelist"`
	MethodBlacklist       []string `json:"method_blacklist"`
	PayloadSizeLimit      int      `json:"payload_size_limit"`
	HighRiskMethodsOnly   bool     `json:"high_risk_methods_only"`
	ExcludePingMessages   bool     `json:"exclude_ping_messages"`
	MinimumRiskLevel      string   `json:"minimum_risk_level"`
	EnableLocalStorage    bool     `json:"enable_local_storage"`
	StoragePath           string   `json:"storage_path"`
	MaxStorageSize        int64    `json:"max_storage_size"`
	RetentionDays         int      `json:"retention_days"`
	MaxConcurrentRequests int      `json:"max_concurrent_requests"`
	RequestTimeout        int      `json:"request_timeout"`
	RetryAttempts         int      `json:"retry_attempts"`
	RetryDelay            int      `json:"retry_delay"`
}

// NewInitializeConfigurationCommand creates a new initialize configuration command
func NewInitializeConfigurationCommand(apiEndpoint, apiKey string) *InitializeConfigurationCommand {
	return &InitializeConfigurationCommand{
		BaseCommand:           NewBaseCommand("initialize_configuration"),
		APIEndpoint:           apiEndpoint,
		APIKey:                apiKey,
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

// Validate validates the initialize configuration command
func (c *InitializeConfigurationCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	if c.APIEndpoint == "" {
		return NewValidationError("API endpoint is required")
	}

	if c.APIKey == "" {
		return NewValidationError("API key is required")
	}

	if c.BatchSize <= 0 {
		return NewValidationError("batch size must be greater than 0")
	}

	if c.FlushInterval < 0 {
		return NewValidationError("flush interval cannot be negative")
	}

	if c.PayloadSizeLimit < 0 {
		return NewValidationError("payload size limit cannot be negative")
	}

	if c.RetentionDays < 0 {
		return NewValidationError("retention days cannot be negative")
	}

	if c.MaxConcurrentRequests <= 0 {
		return NewValidationError("max concurrent requests must be greater than 0")
	}

	if c.RequestTimeout <= 0 {
		return NewValidationError("request timeout must be greater than 0")
	}

	if c.RetryAttempts < 0 {
		return NewValidationError("retry attempts cannot be negative")
	}

	if c.RetryDelay < 0 {
		return NewValidationError("retry delay cannot be negative")
	}

	// Validate minimum risk level
	validRiskLevels := []string{"low", "medium", "high"}
	isValid := false
	for _, level := range validRiskLevels {
		if c.MinimumRiskLevel == level {
			isValid = true
			break
		}
	}
	if !isValid {
		return NewValidationError(fmt.Sprintf("minimum risk level must be one of: %s", strings.Join(validRiskLevels, ", ")))
	}

	return nil
}

// GetConfigurationCommand retrieves the current configuration
type GetConfigurationCommand struct {
	BaseCommand
	IncludeDefaults bool `json:"include_defaults"`
	MaskSecrets     bool `json:"mask_secrets"`
}

// NewGetConfigurationCommand creates a new get configuration command
func NewGetConfigurationCommand() *GetConfigurationCommand {
	return &GetConfigurationCommand{
		BaseCommand:     NewBaseCommand("get_configuration"),
		IncludeDefaults: false,
		MaskSecrets:     true,
	}
}

// Validate validates the get configuration command
func (c *GetConfigurationCommand) Validate() error {
	return c.BaseCommand.Validate()
}

// UpdateConfigurationCommand updates the configuration
type UpdateConfigurationCommand struct {
	BaseCommand
	APIEndpoint           *string   `json:"api_endpoint,omitempty"`
	APIKey                *string   `json:"api_key,omitempty"`
	BatchSize             *int      `json:"batch_size,omitempty"`
	FlushInterval         *int      `json:"flush_interval,omitempty"`
	Debug                 *bool     `json:"debug,omitempty"`
	EnableRiskDetection   *bool     `json:"enable_risk_detection,omitempty"`
	MethodWhitelist       *[]string `json:"method_whitelist,omitempty"`
	MethodBlacklist       *[]string `json:"method_blacklist,omitempty"`
	PayloadSizeLimit      *int      `json:"payload_size_limit,omitempty"`
	HighRiskMethodsOnly   *bool     `json:"high_risk_methods_only,omitempty"`
	ExcludePingMessages   *bool     `json:"exclude_ping_messages,omitempty"`
	MinimumRiskLevel      *string   `json:"minimum_risk_level,omitempty"`
	EnableLocalStorage    *bool     `json:"enable_local_storage,omitempty"`
	StoragePath           *string   `json:"storage_path,omitempty"`
	MaxStorageSize        *int64    `json:"max_storage_size,omitempty"`
	RetentionDays         *int      `json:"retention_days,omitempty"`
	MaxConcurrentRequests *int      `json:"max_concurrent_requests,omitempty"`
	RequestTimeout        *int      `json:"request_timeout,omitempty"`
	RetryAttempts         *int      `json:"retry_attempts,omitempty"`
	RetryDelay            *int      `json:"retry_delay,omitempty"`
}

// NewUpdateConfigurationCommand creates a new update configuration command
func NewUpdateConfigurationCommand() *UpdateConfigurationCommand {
	return &UpdateConfigurationCommand{
		BaseCommand: NewBaseCommand("update_configuration"),
	}
}

// Validate validates the update configuration command
func (c *UpdateConfigurationCommand) Validate() error {
	if err := c.BaseCommand.Validate(); err != nil {
		return err
	}

	// Validate individual fields if provided
	if c.BatchSize != nil && *c.BatchSize <= 0 {
		return NewValidationError("batch size must be greater than 0")
	}

	if c.FlushInterval != nil && *c.FlushInterval < 0 {
		return NewValidationError("flush interval cannot be negative")
	}

	if c.PayloadSizeLimit != nil && *c.PayloadSizeLimit < 0 {
		return NewValidationError("payload size limit cannot be negative")
	}

	if c.RetentionDays != nil && *c.RetentionDays < 0 {
		return NewValidationError("retention days cannot be negative")
	}

	if c.MaxConcurrentRequests != nil && *c.MaxConcurrentRequests <= 0 {
		return NewValidationError("max concurrent requests must be greater than 0")
	}

	if c.RequestTimeout != nil && *c.RequestTimeout <= 0 {
		return NewValidationError("request timeout must be greater than 0")
	}

	if c.RetryAttempts != nil && *c.RetryAttempts < 0 {
		return NewValidationError("retry attempts cannot be negative")
	}

	if c.RetryDelay != nil && *c.RetryDelay < 0 {
		return NewValidationError("retry delay cannot be negative")
	}

	// Validate minimum risk level
	if c.MinimumRiskLevel != nil {
		validRiskLevels := []string{"low", "medium", "high"}
		isValid := false
		for _, level := range validRiskLevels {
			if *c.MinimumRiskLevel == level {
				isValid = true
				break
			}
		}
		if !isValid {
			return NewValidationError(fmt.Sprintf("minimum risk level must be one of: %s", strings.Join(validRiskLevels, ", ")))
		}
	}

	return nil
}

// ValidateConfigurationCommand validates a configuration
type ValidateConfigurationCommand struct {
	BaseCommand
	ConfigPath string `json:"config_path,omitempty"`
}

// NewValidateConfigurationCommand creates a new validate configuration command
func NewValidateConfigurationCommand() *ValidateConfigurationCommand {
	return &ValidateConfigurationCommand{
		BaseCommand: NewBaseCommand("validate_configuration"),
	}
}

// Validate validates the validate configuration command
func (c *ValidateConfigurationCommand) Validate() error {
	return c.BaseCommand.Validate()
}

// BackupConfigurationCommand creates a backup of the configuration
type BackupConfigurationCommand struct {
	BaseCommand
	BackupPath string `json:"backup_path,omitempty"`
}

// NewBackupConfigurationCommand creates a new backup configuration command
func NewBackupConfigurationCommand() *BackupConfigurationCommand {
	return &BackupConfigurationCommand{
		BaseCommand: NewBaseCommand("backup_configuration"),
	}
}

// Validate validates the backup configuration command
func (c *BackupConfigurationCommand) Validate() error {
	return c.BaseCommand.Validate()
}

// RestoreConfigurationCommand restores configuration from backup
type RestoreConfigurationCommand struct {
	BaseCommand
	BackupPath string `json:"backup_path,omitempty"`
}

// NewRestoreConfigurationCommand creates a new restore configuration command
func NewRestoreConfigurationCommand() *RestoreConfigurationCommand {
	return &RestoreConfigurationCommand{
		BaseCommand: NewBaseCommand("restore_configuration"),
	}
}

// Validate validates the restore configuration command
func (c *RestoreConfigurationCommand) Validate() error {
	return c.BaseCommand.Validate()
}
