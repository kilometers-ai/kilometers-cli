package di

import (
	"testing"
)

func TestApplyAPIURLOverride(t *testing.T) {
	tests := []struct {
		name          string
		apiURL        string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid URL override",
			apiURL:      "http://localhost:5149",
			expectError: false,
		},
		{
			name:          "empty URL should fail",
			apiURL:        "",
			expectError:   true,
			expectedError: "API URL cannot be empty",
		},
		{
			name:        "HTTPS URL override",
			apiURL:      "https://staging.api.kilometers.ai",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create container
			container, err := NewContainer()
			if err != nil {
				t.Fatalf("Failed to create container: %v", err)
			}

			// Apply URL override
			err = container.ApplyAPIURLOverride(tt.apiURL)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("ApplyAPIURLOverride() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ApplyAPIURLOverride() unexpected error: %v", err)
			}
			if tt.expectError && err != nil && err.Error() != tt.expectedError {
				t.Errorf("ApplyAPIURLOverride() error = %v, want %v", err.Error(), tt.expectedError)
			}

			// Verify the endpoint was updated (only if no error expected)
			if !tt.expectError {
				apiInfo, err := container.APIGateway.GetAPIInfo()
				if err != nil {
					t.Errorf("Failed to get API info: %v", err)
				} else if len(apiInfo.Endpoints) == 0 || apiInfo.Endpoints[0] != tt.apiURL {
					t.Errorf("API endpoint = %v, want %v", apiInfo.Endpoints, []string{tt.apiURL})
				}
			}
		})
	}
}

func TestApplyAPIKeyOverride(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid API key override",
			apiKey:      "test-key-123",
			expectError: false,
		},
		{
			name:          "empty API key should fail",
			apiKey:        "",
			expectError:   true,
			expectedError: "API key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create container
			container, err := NewContainer()
			if err != nil {
				t.Fatalf("Failed to create container: %v", err)
			}

			// Apply API key override
			err = container.ApplyAPIKeyOverride(tt.apiKey)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("ApplyAPIKeyOverride() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ApplyAPIKeyOverride() unexpected error: %v", err)
			}
			if tt.expectError && err != nil && err.Error() != tt.expectedError {
				t.Errorf("ApplyAPIKeyOverride() error = %v, want %v", err.Error(), tt.expectedError)
			}

			// For successful cases, we mainly verify no error occurred
			// The actual API key verification would require more complex testing
		})
	}
}
