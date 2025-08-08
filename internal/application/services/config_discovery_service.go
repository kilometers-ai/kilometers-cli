package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	"github.com/kilometers-ai/kilometers-cli/internal/infrastructure/config"
)

// ConfigDiscoveryService implements configuration discovery
type ConfigDiscoveryService struct {
	scanners          []ports.ConfigScanner
	apiDiscoverer     *config.APIEndpointDiscoverer
	credentialLocator *config.CredentialLocator
	validator         *config.ConfigValidator
	showProgress      bool
}

// NewConfigDiscoveryService creates a new configuration discovery service
func NewConfigDiscoveryService() (*ConfigDiscoveryService, error) {
	credentialLocator, err := config.NewCredentialLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential locator: %w", err)
	}

	return &ConfigDiscoveryService{
		scanners: []ports.ConfigScanner{
			config.NewEnvironmentScanner(),
			config.NewFileSystemScanner(),
		},
		apiDiscoverer:     config.NewAPIEndpointDiscoverer(),
		credentialLocator: credentialLocator,
		validator:         config.NewConfigValidator(),
		showProgress:      true,
	}, nil
}

// SetShowProgress controls whether to show progress messages
func (s *ConfigDiscoveryService) SetShowProgress(show bool) {
	s.showProgress = show
}

// DiscoverConfig searches for configuration from all available sources
func (s *ConfigDiscoveryService) DiscoverConfig(ctx context.Context) (*domain.DiscoveredConfig, error) {
	if s.showProgress {
		fmt.Println("🔍 Scanning for configuration...")
	}

	// Run all scanners in parallel
	var wg sync.WaitGroup
	configs := make([]*domain.DiscoveredConfig, len(s.scanners))
	errors := make([]error, len(s.scanners))

	for i, scanner := range s.scanners {
		wg.Add(1)
		go func(idx int, sc ports.ConfigScanner) {
			defer wg.Done()

			if s.showProgress {
				fmt.Printf("  • Scanning %s...\n", sc.Name())
			}

			cfg, err := sc.Scan(ctx)
			configs[idx] = cfg
			errors[idx] = err
		}(i, scanner)
	}

	wg.Wait()

	// Merge all discovered configs
	merged := &domain.DiscoveredConfig{
		DiscoveredAt: time.Now(),
		TotalSources: 0,
		Warnings:     []string{},
	}

	for i, cfg := range configs {
		if errors[i] != nil {
			merged.Warnings = append(merged.Warnings,
				fmt.Sprintf("Scanner %s failed: %v", s.scanners[i].Name(), errors[i]))
			continue
		}
		if cfg != nil {
			merged.Merge(cfg)
		}
	}

	// Discover API endpoints if not already found
	if merged.APIEndpoint == nil || merged.APIEndpoint.Value == nil {
		if s.showProgress {
			fmt.Println("  • Discovering API endpoints...")
		}

		if endpoints, err := s.apiDiscoverer.DiscoverEndpoints(ctx); err == nil && len(endpoints) > 0 {
			// Use the first valid endpoint
			for _, endpoint := range endpoints {
				if err := s.validator.ValidateAPIEndpoint(endpoint); err == nil {
					merged.APIEndpoint = &domain.ConfigValue{
						Value:      endpoint,
						Source:     domain.SourceAutoDiscovered,
						SourcePath: "auto-discovered",
						Confidence: 0.8,
					}
					merged.TotalSources++
					if s.showProgress {
						fmt.Printf("  ✓ Found API endpoint: %s\n", endpoint)
					}
					break
				}
			}
		}
	}

	// Discover credentials if not already found
	if merged.APIKey == nil || merged.APIKey.Value == nil {
		if s.showProgress {
			fmt.Println("  • Looking for API credentials...")
		}

		if key, source, err := s.credentialLocator.LocateAPIKey(ctx); err == nil && key != "" {
			if err := s.validator.ValidateAPIKey(key); err == nil {
				merged.APIKey = &domain.ConfigValue{
					Value:      key,
					Source:     domain.SourceAutoDiscovered,
					SourcePath: source,
					Confidence: 0.9,
				}
				merged.TotalSources++
				if s.showProgress {
					// Mask the key for display
					maskedKey := maskAPIKey(key)
					fmt.Printf("  ✓ Found API key: %s (from %s)\n", maskedKey, source)
				}
			}
		}
	}

	// Apply defaults for missing values
	s.applyDefaults(merged)

	if s.showProgress {
		fmt.Printf("\n✅ Configuration discovery complete! Found %d sources.\n", merged.TotalSources)
		if len(merged.Warnings) > 0 {
			fmt.Printf("⚠️  %d warnings encountered during discovery.\n", len(merged.Warnings))
		}
	}

	return merged, nil
}

// ValidateConfig validates a discovered configuration
func (s *ConfigDiscoveryService) ValidateConfig(config *domain.DiscoveredConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	errors := make(map[string]error)

	// Validate required fields
	if config.APIEndpoint == nil || config.APIEndpoint.Value == nil {
		errors["api_endpoint"] = fmt.Errorf("API endpoint is required")
	} else if endpoint, ok := config.APIEndpoint.Value.(string); ok {
		if err := s.validator.ValidateAPIEndpoint(endpoint); err != nil {
			errors["api_endpoint"] = err
		}
	}

	// API key is optional but validate if present
	if config.APIKey != nil && config.APIKey.Value != nil {
		if key, ok := config.APIKey.Value.(string); ok {
			if err := s.validator.ValidateAPIKey(key); err != nil {
				errors["api_key"] = err
			}
		}
	}

	// Validate other fields if present
	if config.BufferSize != nil && config.BufferSize.Value != nil {
		if err := s.validator.ValidateBufferSize(config.BufferSize.Value); err != nil {
			errors["buffer_size"] = err
		}
	}

	if config.LogLevel != nil && config.LogLevel.Value != nil {
		if level, ok := config.LogLevel.Value.(string); ok {
			if err := s.validator.ValidateLogLevel(level); err != nil {
				errors["log_level"] = err
			}
		}
	}

	if config.PluginsDir != nil && config.PluginsDir.Value != nil {
		if dir, ok := config.PluginsDir.Value.(string); ok {
			if err := s.validator.ValidatePluginsDir(dir); err != nil {
				errors["plugins_dir"] = err
			}
		}
	}

	if config.DefaultTimeout != nil && config.DefaultTimeout.Value != nil {
		if err := s.validator.ValidateTimeout(config.DefaultTimeout.Value); err != nil {
			errors["default_timeout"] = err
		}
	}

	// Return combined error if any validation failed
	if len(errors) > 0 {
		var errStrs []string
		for field, err := range errors {
			errStrs = append(errStrs, fmt.Sprintf("%s: %v", field, err))
		}
		sort.Strings(errStrs) // For consistent error messages
		return fmt.Errorf("validation failed:\n  %s", strings.Join(errStrs, "\n  "))
	}

	return nil
}

// MigrateConfig migrates configuration from old formats
func (s *ConfigDiscoveryService) MigrateConfig(oldConfig map[string]interface{}) (*domain.ConfigMigration, error) {
	migration := &domain.ConfigMigration{
		FromVersion: "legacy",
		ToVersion:   "1.0",
		MigratedAt:  time.Now(),
		Changes:     make(map[string]string),
	}

	// Map old field names to new ones
	mappings := map[string]string{
		"apiKey":        "api_key",
		"apiEndpoint":   "api_endpoint",
		"api_url":       "api_endpoint",
		"bufferSize":    "buffer_size",
		"logLevel":      "log_level",
		"pluginsDir":    "plugins_dir",
		"autoProvision": "auto_provision",
		"timeout":       "default_timeout",
	}

	// Apply mappings
	newConfig := make(map[string]interface{})
	for oldKey, value := range oldConfig {
		if newKey, ok := mappings[oldKey]; ok {
			newConfig[newKey] = value
			migration.Changes[oldKey] = newKey
		} else {
			// Keep unmapped fields as-is
			newConfig[oldKey] = value
		}
	}

	// Special case: convert debug flag to log level
	if debug, ok := oldConfig["debug"].(bool); ok && debug {
		newConfig["log_level"] = "debug"
		migration.Changes["debug"] = "log_level=debug"
	}

	// Update the old config map in place
	for k := range oldConfig {
		delete(oldConfig, k)
	}
	for k, v := range newConfig {
		oldConfig[k] = v
	}

	if s.showProgress && len(migration.Changes) > 0 {
		fmt.Printf("📝 Migrated %d configuration fields to new format.\n", len(migration.Changes))
	}

	return migration, nil
}

// applyDefaults applies default values for missing configuration
func (s *ConfigDiscoveryService) applyDefaults(config *domain.DiscoveredConfig) {
	// Apply default buffer size if not set
	if config.BufferSize == nil {
		config.BufferSize = &domain.ConfigValue{
			Value:      1024 * 1024, // 1MB
			Source:     domain.SourceAutoDiscovered,
			SourcePath: "default",
			Confidence: 1.0,
		}
	}

	// Apply default log level if not set
	if config.LogLevel == nil {
		config.LogLevel = &domain.ConfigValue{
			Value:      "info",
			Source:     domain.SourceAutoDiscovered,
			SourcePath: "default",
			Confidence: 1.0,
		}
	}

	// Apply default timeout if not set
	if config.DefaultTimeout == nil {
		config.DefaultTimeout = &domain.ConfigValue{
			Value:      30 * time.Second,
			Source:     domain.SourceAutoDiscovered,
			SourcePath: "default",
			Confidence: 1.0,
		}
	}
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return fmt.Sprintf("%s****%s", key[:4], key[len(key)-4:])
}

// PrintDiscoveredConfig prints a human-readable summary of discovered configuration
func PrintDiscoveredConfig(config *domain.DiscoveredConfig) {
	fmt.Println("\nDiscovered configuration:")
	fmt.Println("------------------------")

	// Helper to print a config value
	printValue := func(name string, value *domain.ConfigValue) {
		if value == nil || value.Value == nil {
			return
		}

		displayValue := fmt.Sprintf("%v", value.Value)
		if name == "API Key" {
			displayValue = maskAPIKey(displayValue)
		}

		confidence := ""
		if value.Confidence < 1.0 {
			confidence = fmt.Sprintf(" (confidence: %.0f%%)", value.Confidence*100)
		}

		fmt.Printf("• %-15s: %s\n", name, displayValue)
		fmt.Printf("  %-15s  Source: %s (%s)%s\n", "", value.Source, value.SourcePath, confidence)
	}

	printValue("API Endpoint", config.APIEndpoint)
	printValue("API Key", config.APIKey)
	printValue("Buffer Size", config.BufferSize)
	printValue("Log Level", config.LogLevel)
	printValue("Plugins Dir", config.PluginsDir)
	printValue("Auto Provision", config.AutoProvision)
	printValue("Default Timeout", config.DefaultTimeout)

	if len(config.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range config.Warnings {
			fmt.Printf("⚠️  %s\n", warning)
		}
	}
}
