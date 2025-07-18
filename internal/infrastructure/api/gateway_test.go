package api

import (
	"fmt"
	"sync"
	"testing"

	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// MockLogger implements the LoggingGateway interface for testing
type MockLogger struct{}

func (m *MockLogger) Log(level ports.LogLevel, message string, fields map[string]interface{}) {}
func (m *MockLogger) LogError(err error, message string, fields map[string]interface{})       {}
func (m *MockLogger) LogEvent(event *event.Event, message string)                             {}
func (m *MockLogger) LogSession(session *session.Session, message string)                     {}
func (m *MockLogger) SetLogLevel(level ports.LogLevel)                                        {}
func (m *MockLogger) GetLogLevel() ports.LogLevel                                             { return ports.LogLevelInfo }
func (m *MockLogger) ConfigureLogging(config *ports.LoggingConfig) error                      { return nil }

func TestUpdateEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		initialURL    string
		newURL        string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid URL update",
			initialURL:  "https://api.kilometers.ai",
			newURL:      "http://localhost:5149",
			expectError: false,
		},
		{
			name:          "empty URL should fail",
			initialURL:    "https://api.kilometers.ai",
			newURL:        "",
			expectError:   true,
			expectedError: "endpoint cannot be empty",
		},
		{
			name:        "update to HTTPS URL",
			initialURL:  "http://localhost:5149",
			newURL:      "https://staging.api.kilometers.ai",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create gateway with initial URL
			gateway := NewKilometersAPIGateway(tt.initialURL, "test-key", &MockLogger{})

			// Verify initial endpoint
			if endpoint := gateway.getEndpoint(); endpoint != tt.initialURL {
				t.Errorf("Initial endpoint = %v, want %v", endpoint, tt.initialURL)
			}

			// Update endpoint
			err := gateway.UpdateEndpoint(tt.newURL)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("UpdateEndpoint() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("UpdateEndpoint() unexpected error: %v", err)
			}
			if tt.expectError && err != nil && err.Error() != tt.expectedError {
				t.Errorf("UpdateEndpoint() error = %v, want %v", err.Error(), tt.expectedError)
			}

			// Verify endpoint was updated (only if no error expected)
			if !tt.expectError {
				if endpoint := gateway.getEndpoint(); endpoint != tt.newURL {
					t.Errorf("Updated endpoint = %v, want %v", endpoint, tt.newURL)
				}

				// Verify connection status was reset
				status := gateway.GetConnectionStatus()
				if status.IsConnected {
					t.Errorf("Connection status should be reset to false after endpoint update")
				}
			}
		})
	}
}

func TestUpdateEndpointConcurrency(t *testing.T) {
	gateway := NewKilometersAPIGateway("https://api.kilometers.ai", "test-key", &MockLogger{})

	// Number of concurrent updates
	numUpdates := 10
	var wg sync.WaitGroup
	wg.Add(numUpdates)

	// Channel to collect results
	results := make(chan string, numUpdates)

	// Perform concurrent updates
	for i := 0; i < numUpdates; i++ {
		go func(index int) {
			defer wg.Done()
			url := fmt.Sprintf("http://localhost:%d", 5000+index)
			err := gateway.UpdateEndpoint(url)
			if err != nil {
				t.Errorf("UpdateEndpoint() error in goroutine %d: %v", index, err)
			}
			// Get the endpoint after update to verify no race condition
			results <- gateway.getEndpoint()
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect all results
	var endpoints []string
	for endpoint := range results {
		endpoints = append(endpoints, endpoint)
	}

	// Verify we got the expected number of results
	if len(endpoints) != numUpdates {
		t.Errorf("Expected %d results, got %d", numUpdates, len(endpoints))
	}

	// Verify the final endpoint is one of the expected values
	finalEndpoint := gateway.getEndpoint()
	found := false
	for i := 0; i < numUpdates; i++ {
		expectedURL := fmt.Sprintf("http://localhost:%d", 5000+i)
		if finalEndpoint == expectedURL {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Final endpoint %s is not one of the expected values", finalEndpoint)
	}
}

func TestGetEndpointThreadSafety(t *testing.T) {
	gateway := NewKilometersAPIGateway("https://api.kilometers.ai", "test-key", &MockLogger{})

	// Number of concurrent reads and writes
	numReads := 50
	numWrites := 10
	var wg sync.WaitGroup
	wg.Add(numReads + numWrites)

	// Channel to collect read results
	readResults := make(chan string, numReads)

	// Start concurrent readers
	for i := 0; i < numReads; i++ {
		go func() {
			defer wg.Done()
			endpoint := gateway.getEndpoint()
			readResults <- endpoint
		}()
	}

	// Start concurrent writers
	for i := 0; i < numWrites; i++ {
		go func(index int) {
			defer wg.Done()
			url := fmt.Sprintf("http://test%d.example.com", index)
			gateway.UpdateEndpoint(url)
		}(i)
	}

	wg.Wait()
	close(readResults)

	// Verify we got all read results without panic or deadlock
	readCount := 0
	for range readResults {
		readCount++
	}

	if readCount != numReads {
		t.Errorf("Expected %d read results, got %d", numReads, readCount)
	}
}

func TestUpdateEndpointLogging(t *testing.T) {
	// Create a logger that captures log messages
	logger := &MockLogger{}
	gateway := NewKilometersAPIGateway("https://api.kilometers.ai", "test-key", logger)

	// Update endpoint
	newURL := "http://localhost:5149"
	err := gateway.UpdateEndpoint(newURL)
	if err != nil {
		t.Errorf("UpdateEndpoint() unexpected error: %v", err)
	}

	// Verify endpoint was updated
	if endpoint := gateway.getEndpoint(); endpoint != newURL {
		t.Errorf("Endpoint = %v, want %v", endpoint, newURL)
	}
}
