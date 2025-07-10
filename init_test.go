package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Test helper function to create a mock scanner from input string
func createMockScanner(input string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(input))
}

// Test helper function to backup and restore config file
func withTempConfig(t *testing.T, testFunc func()) {
	configPath := getConfigPath()

	// Backup existing config if it exists
	var configExists bool
	var backupData []byte
	if data, err := os.ReadFile(configPath); err == nil {
		configExists = true
		backupData = data
	}

	// Clean up after test
	defer func() {
		if configExists {
			os.WriteFile(configPath, backupData, 0644)
		} else {
			os.Remove(configPath)
		}
	}()

	testFunc()
}

// Refactored version of handleInit for testing (doesn't call os.Exit)
func handleInitForTesting(scanner *bufio.Scanner) (*Config, error) {
	config := DefaultConfig()

	// API Key (required)
	scanner.Scan()
	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		return nil, fmt.Errorf("API Key is required")
	}
	config.APIKey = apiKey

	// API URL (optional, default to production)
	scanner.Scan()
	apiURL := strings.TrimSpace(scanner.Text())
	if apiURL == "" {
		apiURL = "https://api.dev.kilometers.ai"
	}
	config.APIEndpoint = apiURL

	// Customer ID (optional, default to "default")
	scanner.Scan()
	customerID := strings.TrimSpace(scanner.Text())
	if customerID == "" {
		customerID = "default"
	}
	// Note: Customer ID is not currently in the Config struct

	// Debug mode (optional)
	scanner.Scan()
	debugResponse := strings.TrimSpace(strings.ToLower(scanner.Text()))
	config.Debug = debugResponse == "y" || debugResponse == "yes"

	// Batch size (optional)
	scanner.Scan()
	batchSizeStr := strings.TrimSpace(scanner.Text())
	if batchSizeStr != "" {
		if batchSize, err := strconv.Atoi(batchSizeStr); err == nil && batchSize > 0 {
			config.BatchSize = batchSize
		}
	}

	return config, nil
}

func TestHandleInit_WithValidInput_CreatesCorrectConfig(t *testing.T) {
	withTempConfig(t, func() {
		// Simulate user input: API key, custom API URL, custom customer ID, debug=yes, batch size=20
		input := "test-api-key-123\nhttps://custom-api.example.com\ncustom-customer\nyes\n20\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err != nil {
			t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
		}

		// Verify configuration values
		tests := []struct {
			name     string
			actual   interface{}
			expected interface{}
		}{
			{"APIKey", config.APIKey, "test-api-key-123"},
			{"APIEndpoint", config.APIEndpoint, "https://custom-api.example.com"},
			{"Debug", config.Debug, true},
			{"BatchSize", config.BatchSize, 20},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !reflect.DeepEqual(tt.actual, tt.expected) {
					t.Errorf("Config.%s = %v, want %v", tt.name, tt.actual, tt.expected)
				}
			})
		}
	})
}

func TestHandleInit_WithDefaultValues_UsesCorrectDefaults(t *testing.T) {
	withTempConfig(t, func() {
		// Simulate user input: API key, then all defaults (empty responses)
		input := "test-api-key-123\n\n\nn\n\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err != nil {
			t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
		}

		// Verify default values are used
		tests := []struct {
			name     string
			actual   interface{}
			expected interface{}
		}{
			{"APIKey", config.APIKey, "test-api-key-123"},
			{"APIEndpoint", config.APIEndpoint, "https://api.dev.kilometers.ai"},
			{"Debug", config.Debug, false},
			{"BatchSize", config.BatchSize, 10}, // Default batch size
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if !reflect.DeepEqual(tt.actual, tt.expected) {
					t.Errorf("Config.%s = %v, want %v", tt.name, tt.actual, tt.expected)
				}
			})
		}
	})
}

func TestHandleInit_WithEmptyAPIKey_ReturnsError(t *testing.T) {
	withTempConfig(t, func() {
		// Simulate user input: empty API key
		input := "\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err == nil {
			t.Error("handleInitForTesting() should return error when API key is empty")
		}

		if config != nil {
			t.Error("handleInitForTesting() should return nil config when API key is empty")
		}

		expectedError := "API Key is required"
		if err.Error() != expectedError {
			t.Errorf("handleInitForTesting() error = %v, want %v", err.Error(), expectedError)
		}
	})
}

func TestHandleInit_WithWhitespaceAPIKey_ReturnsError(t *testing.T) {
	withTempConfig(t, func() {
		// Simulate user input: whitespace-only API key
		input := "   \n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err == nil {
			t.Error("handleInitForTesting() should return error when API key is whitespace only")
		}

		if config != nil {
			t.Error("handleInitForTesting() should return nil config when API key is whitespace only")
		}
	})
}

func TestHandleInit_WithDebugVariations_HandlesCorrectly(t *testing.T) {
	withTempConfig(t, func() {
		tests := []struct {
			name          string
			debugInput    string
			expectedDebug bool
		}{
			{"Debug_Y", "y", true},
			{"Debug_Yes", "yes", true},
			{"Debug_YES", "YES", true},
			{"Debug_Yes_WithSpaces", "  yes  ", true},
			{"Debug_N", "n", false},
			{"Debug_No", "no", false},
			{"Debug_Empty", "", false},
			{"Debug_Invalid", "maybe", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// API key, default URL, default customer, debug response, default batch size
				input := fmt.Sprintf("test-key\n\n\n%s\n\n", tt.debugInput)
				scanner := createMockScanner(input)

				config, err := handleInitForTesting(scanner)

				if err != nil {
					t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
				}

				if config.Debug != tt.expectedDebug {
					t.Errorf("Config.Debug = %v, want %v for input %q", config.Debug, tt.expectedDebug, tt.debugInput)
				}
			})
		}
	})
}

func TestHandleInit_WithBatchSizeVariations_HandlesCorrectly(t *testing.T) {
	withTempConfig(t, func() {
		tests := []struct {
			name              string
			batchSizeInput    string
			expectedBatchSize int
		}{
			{"BatchSize_Valid", "25", 25},
			{"BatchSize_ValidLarge", "100", 100},
			{"BatchSize_Empty", "", 10},      // Default
			{"BatchSize_Zero", "0", 10},      // Invalid, should use default
			{"BatchSize_Negative", "-5", 10}, // Invalid, should use default
			{"BatchSize_Invalid", "abc", 10}, // Invalid, should use default
			{"BatchSize_WithSpaces", "  15  ", 15},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// API key, default URL, default customer, no debug, batch size input
				input := fmt.Sprintf("test-key\n\n\nn\n%s\n", tt.batchSizeInput)
				scanner := createMockScanner(input)

				config, err := handleInitForTesting(scanner)

				if err != nil {
					t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
				}

				if config.BatchSize != tt.expectedBatchSize {
					t.Errorf("Config.BatchSize = %v, want %v for input %q", config.BatchSize, tt.expectedBatchSize, tt.batchSizeInput)
				}
			})
		}
	})
}

func TestHandleInit_SavesConfigurationToFile(t *testing.T) {
	withTempConfig(t, func() {
		// Simulate user input
		input := "test-api-key-123\nhttps://api.test.com\ntest-customer\nyes\n15\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)
		if err != nil {
			t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
		}

		// Save the config
		if err := SaveConfig(config); err != nil {
			t.Fatalf("SaveConfig() returned unexpected error: %v", err)
		}

		// Verify file was created
		configPath := getConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Verify file contents by loading it back
		loadedConfig, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() returned unexpected error: %v", err)
		}

		// Verify loaded config matches saved config
		if loadedConfig.APIKey != config.APIKey {
			t.Errorf("Loaded config APIKey = %v, want %v", loadedConfig.APIKey, config.APIKey)
		}
		if loadedConfig.APIEndpoint != config.APIEndpoint {
			t.Errorf("Loaded config APIEndpoint = %v, want %v", loadedConfig.APIEndpoint, config.APIEndpoint)
		}
		if loadedConfig.Debug != config.Debug {
			t.Errorf("Loaded config Debug = %v, want %v", loadedConfig.Debug, config.Debug)
		}
		if loadedConfig.BatchSize != config.BatchSize {
			t.Errorf("Loaded config BatchSize = %v, want %v", loadedConfig.BatchSize, config.BatchSize)
		}
	})
}

func TestHandleInit_WithMinimalInput_CreatesValidConfig(t *testing.T) {
	withTempConfig(t, func() {
		// Only provide API key, use defaults for everything else
		input := "minimal-api-key\n\n\n\n\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err != nil {
			t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
		}

		// Verify the configuration is valid
		if err := ValidateConfig(config); err != nil {
			t.Errorf("ValidateConfig() returned error for minimal config: %v", err)
		}

		// Verify essential values
		if config.APIKey != "minimal-api-key" {
			t.Errorf("Config.APIKey = %v, want %v", config.APIKey, "minimal-api-key")
		}
		if config.APIEndpoint != "https://api.dev.kilometers.ai" {
			t.Errorf("Config.APIEndpoint = %v, want %v", config.APIEndpoint, "https://api.dev.kilometers.ai")
		}
	})
}

// Integration test that verifies the entire flow works together
func TestHandleInit_Integration_FullFlow(t *testing.T) {
	withTempConfig(t, func() {
		// Test with a complete, realistic input
		input := "sk-12345abcdef67890\nhttps://api.production.kilometers.ai\ncompany-123\ny\n50\n"
		scanner := createMockScanner(input)

		config, err := handleInitForTesting(scanner)

		if err != nil {
			t.Fatalf("handleInitForTesting() returned unexpected error: %v", err)
		}

		// Validate the configuration
		if err := ValidateConfig(config); err != nil {
			t.Errorf("ValidateConfig() returned error: %v", err)
		}

		// Save and reload to verify persistence
		if err := SaveConfig(config); err != nil {
			t.Fatalf("SaveConfig() returned unexpected error: %v", err)
		}

		reloadedConfig, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() returned unexpected error: %v", err)
		}

		// Verify all values persisted correctly
		if !reflect.DeepEqual(config.APIKey, reloadedConfig.APIKey) {
			t.Errorf("APIKey not persisted correctly: got %v, want %v", reloadedConfig.APIKey, config.APIKey)
		}
		if !reflect.DeepEqual(config.APIEndpoint, reloadedConfig.APIEndpoint) {
			t.Errorf("APIEndpoint not persisted correctly: got %v, want %v", reloadedConfig.APIEndpoint, config.APIEndpoint)
		}
		if !reflect.DeepEqual(config.Debug, reloadedConfig.Debug) {
			t.Errorf("Debug not persisted correctly: got %v, want %v", reloadedConfig.Debug, config.Debug)
		}
		if !reflect.DeepEqual(config.BatchSize, reloadedConfig.BatchSize) {
			t.Errorf("BatchSize not persisted correctly: got %v, want %v", reloadedConfig.BatchSize, config.BatchSize)
		}
	})
}
