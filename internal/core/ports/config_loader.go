package ports

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// ConfigLoader defines the interface for loading configuration from various sources
type ConfigLoader interface {
	// Load loads configuration from all available sources with proper precedence
	Load(ctx context.Context) (*domain.UnifiedConfig, error)

	// LoadWithOptions loads configuration with specific loading options
	LoadWithOptions(ctx context.Context, opts LoadOptions) (*domain.UnifiedConfig, error)
}

// ConfigStorage defines the interface for persisting configuration
type ConfigStorage interface {
	// Save saves configuration to persistent storage
	Save(ctx context.Context, config *domain.UnifiedConfig) error

	// SaveToPath saves configuration to a specific path
	SaveToPath(ctx context.Context, config *domain.UnifiedConfig, path string) error

	// Delete removes configuration from persistent storage
	Delete(ctx context.Context) error

	// Exists checks if configuration exists in persistent storage
	Exists(ctx context.Context) (bool, error)

	// GetConfigPath returns the path where configuration is stored
	GetConfigPath() (string, error)
}

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

// UnifiedConfigScanner defines the interface for scanning configuration from a specific source
type UnifiedConfigScanner interface {
	// Name returns the name of this scanner (e.g., "filesystem", "environment")
	Name() string

	// Scan discovers configuration from this source
	Scan(ctx context.Context) (*domain.UnifiedConfig, error)

	// Priority returns the base priority for values from this scanner
	Priority() int
}
