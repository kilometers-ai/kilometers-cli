package commands

import (
	"fmt"
)

// UpdateConfigurationCommand updates application configuration
type UpdateConfigurationCommand struct {
	APIKey    string `json:"api_key"`
	APIHost   string `json:"api_host"`
	BatchSize int    `json:"batch_size"`
}

// NewUpdateConfigurationCommand creates a new update configuration command
func NewUpdateConfigurationCommand() *UpdateConfigurationCommand {
	return &UpdateConfigurationCommand{
		BatchSize: 10,
	}
}

// Validate validates the update configuration command
func (c *UpdateConfigurationCommand) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if c.APIHost == "" {
		return fmt.Errorf("API host is required")
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	return nil
}

// GetConfigurationCommand retrieves current configuration
type GetConfigurationCommand struct {
	// No fields needed for get operation
}

// NewGetConfigurationCommand creates a new get configuration command
func NewGetConfigurationCommand() *GetConfigurationCommand {
	return &GetConfigurationCommand{}
}

// Validate validates the get configuration command
func (c *GetConfigurationCommand) Validate() error {
	// No validation needed for get operation
	return nil
}

// ValidateConfigurationCommand validates configuration without updating
type ValidateConfigurationCommand struct {
	APIKey    *string `json:"api_key,omitempty"`
	APIHost   *string `json:"api_host,omitempty"`
	BatchSize *int    `json:"batch_size,omitempty"`
}

// NewValidateConfigurationCommand creates a new validate configuration command
func NewValidateConfigurationCommand() *ValidateConfigurationCommand {
	return &ValidateConfigurationCommand{}
}

// Validate validates the validate configuration command
func (c *ValidateConfigurationCommand) Validate() error {
	if c.APIKey != nil && *c.APIKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if c.APIHost != nil && *c.APIHost == "" {
		return fmt.Errorf("API host cannot be empty")
	}

	if c.BatchSize != nil && *c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	return nil
}
