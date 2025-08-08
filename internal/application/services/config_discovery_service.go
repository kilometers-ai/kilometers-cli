package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// ConfigDiscoveryService handles automatic discovery of configuration from various sources
type ConfigDiscoveryService struct {
	// Could add more sophisticated discovery mechanisms later
}

// NewConfigDiscoveryService creates a new configuration discovery service
func NewConfigDiscoveryService() (*ConfigDiscoveryService, error) {
	return &ConfigDiscoveryService{}, nil
}

// DiscoverConfig discovers configuration from environment variables and other sources
func (s *ConfigDiscoveryService) DiscoverConfig(ctx context.Context) (*domain.DiscoveredConfig, error) {
	discovered := &domain.DiscoveredConfig{
		Sources: make([]string, 0),
	}

	// Check for API key from various environment variables
	envVars := []string{"KM_API_KEY", "KILOMETERS_API_KEY", "OPENAI_API_KEY"}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			discovered.APIKey = value
			discovered.Sources = append(discovered.Sources, fmt.Sprintf("env:%s", envVar))
			break // Use first found
		}
	}

	// Check for API endpoint
	endpointVars := []string{"KM_API_ENDPOINT", "KILOMETERS_API_ENDPOINT"}
	for _, envVar := range endpointVars {
		if value := os.Getenv(envVar); value != "" {
			discovered.APIEndpoint = value
			discovered.Sources = append(discovered.Sources, fmt.Sprintf("env:%s", envVar))
			break // Use first found
		}
	}

	// If no endpoint discovered, use default
	if discovered.APIEndpoint == "" {
		discovered.APIEndpoint = "http://localhost:5194"
		discovered.Sources = append(discovered.Sources, "default")
	}

	return discovered, nil
}

// ValidateConfig validates the discovered configuration
func (s *ConfigDiscoveryService) ValidateConfig(config *domain.DiscoveredConfig) error {
	var errors []string

	// Validate API key format if present
	if config.APIKey != "" {
		if len(config.APIKey) < 8 {
			errors = append(errors, "API key appears to be too short")
		}
		if !strings.HasPrefix(config.APIKey, "km_") && !strings.HasPrefix(config.APIKey, "sk-") {
			errors = append(errors, "API key doesn't match expected format (should start with 'km_' or 'sk-')")
		}
	}

	// Validate API endpoint if present
	if config.APIEndpoint != "" {
		if !strings.HasPrefix(config.APIEndpoint, "http://") && !strings.HasPrefix(config.APIEndpoint, "https://") {
			errors = append(errors, "API endpoint should be a valid HTTP(S) URL")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// PrintDiscoveredConfig prints the discovered configuration in a user-friendly format
func PrintDiscoveredConfig(config *domain.DiscoveredConfig) {
	fmt.Println("🔍 Configuration Discovery Results:")
	fmt.Println()

	if config.APIKey != "" {
		maskedKey := config.APIKey
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:4] + strings.Repeat("*", len(maskedKey)-8) + maskedKey[len(maskedKey)-4:]
		}
		fmt.Printf("  🔑 API Key: %s\n", maskedKey)
	} else {
		fmt.Printf("  🔑 API Key: <not found>\n")
	}

	fmt.Printf("  🌐 API Endpoint: %s\n", config.APIEndpoint)

	if len(config.Sources) > 0 {
		fmt.Printf("  📋 Sources: %s\n", strings.Join(config.Sources, ", "))
	}

	fmt.Println()
}