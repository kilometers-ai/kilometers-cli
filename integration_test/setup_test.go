package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"kilometers.ai/cli/test"
)

// TestMain provides setup and teardown for the integration test suite
func TestMain(m *testing.M) {
	// Set test mode
	os.Setenv("KM_TEST_MODE", "true")
	os.Setenv("KM_LOG_LEVEL", "debug")

	// Run tests
	exitCode := m.Run()

	// Cleanup
	os.Unsetenv("KM_TEST_MODE")
	os.Unsetenv("KM_LOG_LEVEL")

	os.Exit(exitCode)
}

// createBasicTestEnvironment creates a basic test environment for most tests
func createBasicTestEnvironment(t *testing.T) *test.TestEnvironment {
	env := test.NewTestEnvironment(t)

	// Configure mock API server with basic auth token
	env.MockAPIServer.AddAuthToken("test_token_123")

	// Start the environment
	if err := env.Start(t); err != nil {
		t.Fatalf("Failed to start test environment: %v", err)
	}

	return env
}

// createFullTestEnvironment creates a complete test environment with DI container
func createFullTestEnvironment(t *testing.T) *test.TestEnvironment {
	env := test.NewTestEnvironment(t)

	// Configure mock servers
	env.MockAPIServer.AddAuthToken("test_token_123")

	// Setup MCP server with realistic handlers
	setupRealisticMCPServer(env.MockMCPServer)

	// Start with container
	if err := env.StartWithContainer(t); err != nil {
		t.Fatalf("Failed to start test environment with container: %v", err)
	}

	return env
}

// setupRealisticMCPServer configures the mock MCP server with realistic responses
func setupRealisticMCPServer(server *test.MockMCPServer) {
	// Simulate tool responses
	server.SetHandler("tools/call", func(params interface{}) interface{} {
		return map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Successfully executed tool call",
				},
			},
			"isError": false,
		}
	})

	// Simulate resource responses
	server.SetHandler("resources/read", func(params interface{}) interface{} {
		return map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"uri":      "file://test.txt",
					"mimeType": "text/plain",
					"text":     "Test file content",
				},
			},
		}
	})

	// Simulate prompt responses
	server.SetHandler("prompts/get", func(params interface{}) interface{} {
		return map[string]interface{}{
			"description": "Test prompt",
			"messages": []interface{}{
				map[string]interface{}{
					"role": "user",
					"content": map[string]interface{}{
						"type": "text",
						"text": "Test prompt message",
					},
				},
			},
		}
	})

	// Add latency for some methods to test timeout handling
	server.SetLatency("slow/operation", 100*time.Millisecond)
}

// waitForServerReady waits for servers to be ready
func waitForServerReady(env *test.TestEnvironment) bool {
	return test.WaitForCondition(func() bool {
		mcpStats := env.MockMCPServer.GetStats()
		apiStats := env.MockAPIServer.GetStats()

		mcpRunning, _ := mcpStats["is_running"].(bool)
		apiRunning, _ := apiStats["is_running"].(bool)

		return mcpRunning && apiRunning
	}, 5*time.Second, 100*time.Millisecond)
}

// setupTestContext creates a test context with timeout
func setupTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Common test constants
const (
	TestTimeout        = 30 * time.Second
	ShortTestTimeout   = 5 * time.Second
	TestSessionID      = "test-session-123"
	TestAPIKey         = "test_key"
	TestBatchSize      = 10
	MaxTestLatency     = 100 * time.Millisecond
	MaxAPIResponseTime = 500 * time.Millisecond
)
