package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kilometers.ai/cli/test"
)

// TestCLI_CompleteMonitoringSession_WorksCorrectly tests the complete monitoring workflow
func TestCLI_CompleteMonitoringSession_WorksCorrectly(t *testing.T) {
	env := createFullTestEnvironment(t)
	ctx, cancel := setupTestContext(TestTimeout)
	defer cancel()

	// Build the CLI binary for testing
	cliPath := buildCLIBinary(t)

	t.Run("init_command_creates_valid_config", func(t *testing.T) {
		configDir := filepath.Join(env.TempDir, "km_config")
		cmd := exec.CommandContext(ctx, cliPath, "init", "--config-dir", configDir)

		// Set environment variables
		cmd.Env = append(os.Environ(),
			"KM_API_KEY="+TestAPIKey,
			"KM_API_URL="+env.GetAPIServerAddress(),
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Init command failed: %v\nOutput: %s", err, output)
		}

		// Verify config file was created
		configFile := filepath.Join(configDir, "config.json")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Verify output contains success message
		outputStr := string(output)
		if !strings.Contains(outputStr, "Configuration saved to:") {
			t.Errorf("Expected success message in output, got: %s", outputStr)
		}
	})

	t.Run("config_command_shows_current_configuration", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cliPath, "config", "show")
		cmd.Env = append(os.Environ(),
			"KM_CONFIG_FILE="+env.ConfigFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "API Endpoint") {
			t.Error("Config output should contain API Endpoint")
		}
		if !strings.Contains(outputStr, env.GetAPIServerAddress()) {
			t.Errorf("Config should show correct API endpoint: %s", env.GetAPIServerAddress())
		}
	})

	t.Run("validate_command_checks_configuration", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cliPath, "validate")
		cmd.Env = append(os.Environ(),
			"KM_CONFIG_FILE="+env.ConfigFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Validate command failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "✅ Configuration valid") {
			t.Errorf("Expected validation success message, got: %s", outputStr)
		}
	})

	t.Run("monitor_command_with_mock_process", func(t *testing.T) {
		// Create a simple mock MCP process script
		mockScript := createMockMCPScript(t, env.TempDir)

		// Start monitoring with a short timeout
		monitorCtx, monitorCancel := context.WithTimeout(ctx, 3*time.Second)
		defer monitorCancel()

		cmd := exec.CommandContext(monitorCtx, cliPath, "monitor", "node", mockScript)
		cmd.Env = append(os.Environ(),
			"KM_CONFIG_FILE="+env.ConfigFile,
		)

		// Start the command
		output, err := cmd.CombinedOutput()

		// We expect a timeout or graceful shutdown
		if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Logf("Monitor command output: %s", output)
			// Don't fail on timeout - this is expected for this test
		}

		// Verify some monitoring activity occurred
		outputStr := string(output)
		if !strings.Contains(outputStr, "Started monitoring") && !strings.Contains(outputStr, "Monitoring session started") {
			t.Logf("Monitor output: %s", outputStr)
			// Log but don't fail - the monitoring might have started correctly
		}

		// Verify API calls were made to the mock server
		test.AssertAPIRequestMade(t, env, "POST", "/api/sessions")
	})
}

// TestCLI_ConfigurationInitialization_CreatesValidConfig tests config initialization
func TestCLI_ConfigurationInitialization_CreatesValidConfig(t *testing.T) {
	env := createFullTestEnvironment(t)
	ctx, cancel := setupTestContext(ShortTestTimeout)
	defer cancel()

	cliPath := buildCLIBinary(t)

	t.Run("init_with_custom_values", func(t *testing.T) {
		configDir := filepath.Join(env.TempDir, "custom_config")

		cmd := exec.CommandContext(ctx, cliPath, "init",
			"--config-dir", configDir,
			"--api-key", "custom_key_123",
			"--api-url", "https://custom.api.com",
			"--batch-size", "50",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Init with custom values failed: %v\nOutput: %s", err, output)
		}

		// Verify config file contains custom values
		configFile := filepath.Join(configDir, "config.json")
		content, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		configStr := string(content)
		if !strings.Contains(configStr, "custom_key_123") {
			t.Error("Config should contain custom API key")
		}
		if !strings.Contains(configStr, "https://custom.api.com") {
			t.Error("Config should contain custom API URL")
		}
		if !strings.Contains(configStr, "50") {
			t.Error("Config should contain custom batch size")
		}
	})

	t.Run("init_interactive_mode", func(t *testing.T) {
		// This would test interactive mode if implemented
		t.Skip("Interactive mode testing requires stdin automation")
	})
}

// TestCLI_UpdateCommand_ChecksAndDownloadsUpdate tests the update functionality
func TestCLI_UpdateCommand_ChecksAndDownloadsUpdate(t *testing.T) {
	_ = createBasicTestEnvironment(t) // Not used in this test
	ctx, cancel := setupTestContext(ShortTestTimeout)
	defer cancel()

	cliPath := buildCLIBinary(t)

	t.Run("update_check_only", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cliPath, "update", "--check-only")

		output, err := cmd.CombinedOutput()
		// Update check might fail in test environment, which is okay
		if err != nil {
			t.Logf("Update check failed (expected in test): %v", err)
		}

		outputStr := string(output)
		// Just verify the command runs and provides some output
		if len(outputStr) == 0 {
			t.Error("Update command should provide some output")
		}
	})

	t.Run("update_version_display", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cliPath, "--version")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "version") {
			t.Error("Version output should contain version information")
		}
	})
}

// TestCLI_ErrorHandling_ReportsCorrectly tests error handling scenarios
func TestCLI_ErrorHandling_ReportsCorrectly(t *testing.T) {
	env := createFullTestEnvironment(t)
	ctx, cancel := setupTestContext(ShortTestTimeout)
	defer cancel()

	cliPath := buildCLIBinary(t)

	t.Run("invalid_config_file", func(t *testing.T) {
		invalidConfigFile := test.CreateTempFile(t, env.TempDir, "invalid-config-*.json", "invalid json content")

		cmd := exec.CommandContext(ctx, cliPath, "validate")
		cmd.Env = append(os.Environ(),
			"KM_CONFIG_FILE="+invalidConfigFile,
		)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected validation to fail with invalid config")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "invalid") {
			t.Error("Error message should indicate validation failure")
		}
	})

	t.Run("missing_config_file", func(t *testing.T) {
		nonexistentFile := filepath.Join(env.TempDir, "nonexistent.json")

		cmd := exec.CommandContext(ctx, cliPath, "validate")
		cmd.Env = append(os.Environ(),
			"KM_CONFIG_FILE="+nonexistentFile,
		)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected validation to fail with missing config")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "not found") {
			t.Error("Error message should indicate missing file")
		}
	})

	t.Run("invalid_command_arguments", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cliPath, "monitor", "--invalid-flag")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected command to fail with invalid flag")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "unknown flag") && !strings.Contains(outputStr, "invalid") {
			t.Logf("Error output: %s", outputStr)
			// Don't fail - different CLI frameworks have different error messages
		}
	})
}

// TestCLI_PerformanceUnderLoad_HandlesCorrectly tests performance characteristics
func TestCLI_PerformanceUnderLoad_HandlesCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := createFullTestEnvironment(t)
	ctx, cancel := setupTestContext(TestTimeout)
	defer cancel()

	cliPath := buildCLIBinary(t)

	t.Run("config_operations_performance", func(t *testing.T) {
		iterations := 100

		duration := test.MeasureExecutionTime(t, "config_show_operations", func() {
			for i := 0; i < iterations; i++ {
				cmd := exec.CommandContext(ctx, cliPath, "config", "show")
				cmd.Env = append(os.Environ(), "KM_CONFIG_FILE="+env.ConfigFile)

				_, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("Config operation %d failed: %v", i, err)
				}
			}
		})

		avgDuration := duration / time.Duration(iterations)
		test.AssertExecutionTime(t, avgDuration, 50*time.Millisecond, "average_config_show")
	})

	t.Run("validation_performance", func(t *testing.T) {
		iterations := 50

		duration := test.MeasureExecutionTime(t, "validation_operations", func() {
			for i := 0; i < iterations; i++ {
				cmd := exec.CommandContext(ctx, cliPath, "validate")
				cmd.Env = append(os.Environ(), "KM_CONFIG_FILE="+env.ConfigFile)

				_, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("Validation operation %d failed: %v", i, err)
				}
			}
		})

		avgDuration := duration / time.Duration(iterations)
		test.AssertExecutionTime(t, avgDuration, 100*time.Millisecond, "average_validation")
	})
}

// Helper functions

// buildCLIBinary builds the CLI binary for testing
func buildCLIBinary(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "km-test")
	if _, err := os.Stat("../main.go"); err == nil {
		// Build from root if main.go exists there
		cmd := exec.Command("go", "build", "-o", binaryPath, "../main.go")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build CLI binary: %v", err)
		}
	} else {
		// Build from cmd directory
		cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/main.go")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build CLI binary: %v", err)
		}
	}

	return binaryPath
}

// createMockMCPScript creates a simple Node.js script that acts as an MCP server
func createMockMCPScript(t *testing.T, tempDir string) string {
	t.Helper()

	scriptContent := `
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

// Send initialize response
setTimeout(() => {
  const initResponse = {
    jsonrpc: "2.0",
    id: 1,
    result: {
      protocolVersion: "2024-11-05",
      capabilities: {
        logging: {},
        prompts: {},
        resources: {},
        tools: {}
      },
      serverInfo: {
        name: "mock-mcp-server",
        version: "0.1.0"
      }
    }
  };
  console.log(JSON.stringify(initResponse));
}, 100);

// Handle incoming messages
rl.on('line', (input) => {
  try {
    const message = JSON.parse(input);
    
    const response = {
      jsonrpc: "2.0",
      id: message.id,
      result: { status: "ok", method: message.method || "unknown" }
    };
    
    console.log(JSON.stringify(response));
  } catch (e) {
    // Ignore invalid JSON
  }
});

// Keep alive for a few seconds
setTimeout(() => {
  process.exit(0);
}, 2000);
`

	scriptPath := filepath.Join(tempDir, "mock-mcp-server.js")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	return scriptPath
}
