package main

import (
	"os"
	"reflect"
	"testing"
)

func TestDefaultConfig_ReturnsExpectedValues(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test all default values
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
	}{
		{"APIEndpoint", config.APIEndpoint, "http://localhost:5194"},
		{"APIKey", config.APIKey, ""},
		{"BatchSize", config.BatchSize, 10},
		{"Debug", config.Debug, false},
		{"EnableRiskDetection", config.EnableRiskDetection, false},
		{"MethodWhitelist", len(config.MethodWhitelist), 0},
		{"PayloadSizeLimit", config.PayloadSizeLimit, 0},
		{"HighRiskMethodsOnly", config.HighRiskMethodsOnly, false},
		{"ExcludePingMessages", config.ExcludePingMessages, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.actual, tt.expected) {
				t.Errorf("DefaultConfig().%s = %v, want %v", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

func TestValidateConfig_HandlesInvalidBatchSize(t *testing.T) {
	tests := []struct {
		name              string
		inputBatchSize    int
		expectedBatchSize int
	}{
		{"ZeroBatchSize_ShouldBeSetToDefault", 0, 10},
		{"NegativeBatchSize_ShouldBeSetToDefault", -5, 10},
		{"ValidBatchSize_ShouldRemainUnchanged", 25, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{BatchSize: tt.inputBatchSize}

			err := ValidateConfig(config)

			if err != nil {
				t.Errorf("ValidateConfig() returned unexpected error: %v", err)
			}

			if config.BatchSize != tt.expectedBatchSize {
				t.Errorf("ValidateConfig() BatchSize = %v, want %v", config.BatchSize, tt.expectedBatchSize)
			}
		})
	}
}

func TestValidateConfig_HandlesInvalidPayloadSizeLimit(t *testing.T) {
	tests := []struct {
		name                     string
		inputPayloadSizeLimit    int
		expectedPayloadSizeLimit int
	}{
		{"NegativePayloadSizeLimit_ShouldBeSetToZero", -100, 0},
		{"ZeroPayloadSizeLimit_ShouldRemainZero", 0, 0},
		{"ValidPayloadSizeLimit_ShouldRemainUnchanged", 1024, 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{PayloadSizeLimit: tt.inputPayloadSizeLimit}

			err := ValidateConfig(config)

			if err != nil {
				t.Errorf("ValidateConfig() returned unexpected error: %v", err)
			}

			if config.PayloadSizeLimit != tt.expectedPayloadSizeLimit {
				t.Errorf("ValidateConfig() PayloadSizeLimit = %v, want %v", config.PayloadSizeLimit, tt.expectedPayloadSizeLimit)
			}
		})
	}
}

func TestValidateConfig_AutoEnablesRiskDetectionWhenHighRiskOnlyIsTrue(t *testing.T) {
	config := &Config{
		HighRiskMethodsOnly: true,
		EnableRiskDetection: false, // Initially disabled
	}

	err := ValidateConfig(config)

	if err != nil {
		t.Errorf("ValidateConfig() returned unexpected error: %v", err)
	}

	if !config.EnableRiskDetection {
		t.Error("ValidateConfig() should auto-enable risk detection when HighRiskMethodsOnly is true")
	}
}

func TestValidateConfig_PreservesRiskDetectionWhenHighRiskOnlyIsFalse(t *testing.T) {
	config := &Config{
		HighRiskMethodsOnly: false,
		EnableRiskDetection: true, // Should remain enabled
	}

	err := ValidateConfig(config)

	if err != nil {
		t.Errorf("ValidateConfig() returned unexpected error: %v", err)
	}

	if !config.EnableRiskDetection {
		t.Error("ValidateConfig() should preserve EnableRiskDetection when HighRiskMethodsOnly is false")
	}
}

func TestValidateConfig_WithValidConfiguration_MakesNoChanges(t *testing.T) {
	originalConfig := &Config{
		APIEndpoint:         "https://api.example.com",
		APIKey:              "test-key",
		BatchSize:           50,
		Debug:               true,
		EnableRiskDetection: true,
		MethodWhitelist:     []string{"method1", "method2"},
		PayloadSizeLimit:    2048,
		HighRiskMethodsOnly: false,
		ExcludePingMessages: false,
	}

	// Create a copy to compare against
	configCopy := *originalConfig
	configCopy.MethodWhitelist = make([]string, len(originalConfig.MethodWhitelist))
	copy(configCopy.MethodWhitelist, originalConfig.MethodWhitelist)

	err := ValidateConfig(originalConfig)

	if err != nil {
		t.Errorf("ValidateConfig() returned unexpected error: %v", err)
	}

	// Verify no changes were made
	if !reflect.DeepEqual(*originalConfig, configCopy) {
		t.Error("ValidateConfig() modified a valid configuration when it should not have")
	}
}

func TestLoadConfig_RespectsEnvironmentVariables(t *testing.T) {
	// Save original environment values
	originalEndpoint := os.Getenv("KILOMETERS_API_URL")
	originalAPIKey := os.Getenv("KILOMETERS_API_KEY")
	originalDebug := os.Getenv("KM_DEBUG")
	originalBatchSize := os.Getenv("KM_BATCH_SIZE")

	// Handle config file that might override environment variables
	configPath := getConfigPath()
	var configExists bool

	// Check if config file exists and back it up
	if _, err := os.ReadFile(configPath); err == nil {
		configExists = true
		// Temporarily rename the config file so it doesn't interfere with the test
		os.Rename(configPath, configPath+".test-backup")
	}

	// Clean up after test - restore original values or unset if they were empty
	defer func() {
		if originalEndpoint == "" {
			os.Unsetenv("KILOMETERS_API_URL")
		} else {
			os.Setenv("KILOMETERS_API_URL", originalEndpoint)
		}
		if originalAPIKey == "" {
			os.Unsetenv("KILOMETERS_API_KEY")
		} else {
			os.Setenv("KILOMETERS_API_KEY", originalAPIKey)
		}
		if originalDebug == "" {
			os.Unsetenv("KM_DEBUG")
		} else {
			os.Setenv("KM_DEBUG", originalDebug)
		}
		if originalBatchSize == "" {
			os.Unsetenv("KM_BATCH_SIZE")
		} else {
			os.Setenv("KM_BATCH_SIZE", originalBatchSize)
		}

		// Restore config file if it existed
		if configExists {
			os.Rename(configPath+".test-backup", configPath)
		}
	}()

	// Set test environment variables
	os.Setenv("KILOMETERS_API_URL", "https://test.api.com")
	os.Setenv("KILOMETERS_API_KEY", "test-api-key")
	os.Setenv("KM_DEBUG", "true")
	os.Setenv("KM_BATCH_SIZE", "25")

	config, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
	}{
		{"APIEndpoint_FromEnvironment", config.APIEndpoint, "https://test.api.com"},
		{"APIKey_FromEnvironment", config.APIKey, "test-api-key"},
		{"Debug_FromEnvironment", config.Debug, true},
		{"BatchSize_FromEnvironment", config.BatchSize, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.actual, tt.expected) {
				t.Errorf("LoadConfig() %s = %v, want %v", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

func TestLoadConfig_HandlesInvalidBatchSizeEnvironmentVariable(t *testing.T) {
	// Save original environment value
	originalBatchSize := os.Getenv("KM_BATCH_SIZE")
	defer func() {
		if originalBatchSize == "" {
			os.Unsetenv("KM_BATCH_SIZE")
		} else {
			os.Setenv("KM_BATCH_SIZE", originalBatchSize)
		}
	}()

	// Set invalid batch size
	os.Setenv("KM_BATCH_SIZE", "invalid-number")

	config, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	// Should fall back to default value
	if config.BatchSize != 10 {
		t.Errorf("LoadConfig() with invalid KM_BATCH_SIZE should use default value 10, got %v", config.BatchSize)
	}
}

func TestLoadConfig_HandlesZeroBatchSizeEnvironmentVariable(t *testing.T) {
	// Save original environment value
	originalBatchSize := os.Getenv("KM_BATCH_SIZE")
	defer func() {
		if originalBatchSize == "" {
			os.Unsetenv("KM_BATCH_SIZE")
		} else {
			os.Setenv("KM_BATCH_SIZE", originalBatchSize)
		}
	}()

	// Set zero batch size (invalid)
	os.Setenv("KM_BATCH_SIZE", "0")

	config, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}

	// Should fall back to default value since 0 is invalid
	if config.BatchSize != 10 {
		t.Errorf("LoadConfig() with zero KM_BATCH_SIZE should use default value 10, got %v", config.BatchSize)
	}
}
