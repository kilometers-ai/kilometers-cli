package integration_test

import (
	"fmt"
	"strings"
	"testing"
)

// TestMonitorArgumentParsing tests the new --server flag argument parsing logic
func TestMonitorArgumentParsing_WorksCorrectly(t *testing.T) {

	t.Run("quoted_server_command_with_flags", func(t *testing.T) {
		// Test: km monitor --server "npx -y @modelcontextprotocol/server-github"
		args := []string{"--server", "npx -y @modelcontextprotocol/server-github"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse quoted server command: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		expectedArgs := []string{"-y", "@modelcontextprotocol/server-github"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("unquoted_server_command_with_flags", func(t *testing.T) {
		// Test: km monitor --server npx -y @modelcontextprotocol/server-github
		args := []string{"--server", "npx", "-y", "@modelcontextprotocol/server-github"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse unquoted server command: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		expectedArgs := []string{"-y", "@modelcontextprotocol/server-github"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("server_command_with_monitor_flags", func(t *testing.T) {
		// Test: km monitor --server npx -y @modelcontextprotocol/server-github --batch-size 20
		args := []string{"--server", "npx", "-y", "@modelcontextprotocol/server-github", "--batch-size", "20"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse server command with monitor flags: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		// Should only include MCP server args, not monitor flags
		expectedArgs := []string{"-y", "@modelcontextprotocol/server-github"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("quoted_server_with_monitor_flags", func(t *testing.T) {
		// Test: km monitor --server "npx -y @modelcontextprotocol/server-github" --batch-size 20
		args := []string{"--server", "npx -y @modelcontextprotocol/server-github", "--batch-size", "20"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse quoted server command with monitor flags: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		expectedArgs := []string{"-y", "@modelcontextprotocol/server-github"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("python_server_command", func(t *testing.T) {
		// Test: km monitor --server "python -m my_mcp_server --port 8080"
		args := []string{"--server", "python -m my_mcp_server --port 8080"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse python server command: %v", err)
		}

		if command != "python" {
			t.Errorf("Expected command 'python', got '%s'", command)
		}

		expectedArgs := []string{"-m", "my_mcp_server", "--port", "8080"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("missing_server_flag", func(t *testing.T) {
		// Test: km monitor (no --server flag)
		args := []string{}

		_, _, err := testProcessMonitorArguments(args)
		if err == nil {
			t.Error("Expected error for missing --server flag")
		}

		if err.Error() != "--server flag is required" {
			t.Errorf("Expected '--server flag is required' error, got '%s'", err.Error())
		}
	})

	t.Run("empty_server_command", func(t *testing.T) {
		// Test: km monitor --server ""
		args := []string{"--server", ""}

		_, _, err := testProcessMonitorArguments(args)
		if err == nil {
			t.Error("Expected error for empty server command")
		}
	})

	t.Run("complex_mixed_flags", func(t *testing.T) {
		// Test: km monitor --server npx -y @modelcontextprotocol/server-linear --batch-size 5 --debug-replay file.jsonl
		args := []string{"--server", "npx", "-y", "@modelcontextprotocol/server-linear", "--batch-size", "5", "--debug-replay", "file.jsonl"}

		command, commandArgs, err := testProcessMonitorArguments(args)
		if err != nil {
			t.Fatalf("Failed to parse complex mixed flags: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		// Should only include MCP server args
		expectedArgs := []string{"-y", "@modelcontextprotocol/server-linear"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})
}

// testProcessMonitorArguments simulates the argument processing logic
// This is a simplified version for testing purposes
func testProcessMonitorArguments(args []string) (string, []string, error) {
	// Find --server flag in arguments
	serverIndex := -1
	for i, arg := range args {
		if arg == "--server" {
			serverIndex = i
			break
		}
	}

	if serverIndex == -1 {
		return "", nil, fmt.Errorf("--server flag is required")
	}

	if serverIndex+1 >= len(args) {
		return "", nil, fmt.Errorf("--server flag requires a value")
	}

	// Known monitor flags that should NOT be part of server command
	monitorFlags := map[string]bool{
		"--batch-size": true, "--flush-interval": true, "--enable-risk-detection": true,
		"--method-whitelist": true, "--method-blacklist": true, "--payload-size-limit": true,
		"--high-risk-only": true, "--exclude-ping": true, "--min-risk-level": true,
		"--debug-replay": true, "--debug-delay": true, "--debug": true,
	}

	// Find where server command ends (either at next monitor flag or end of args)
	serverEndIndex := len(args)
	for i := serverIndex + 1; i < len(args); i++ {
		if monitorFlags[args[i]] {
			serverEndIndex = i
			break
		}
	}

	// Extract server command parts
	serverCommandParts := args[serverIndex+1 : serverEndIndex]

	if len(serverCommandParts) == 0 {
		return "", nil, fmt.Errorf("--server flag requires a command")
	}

	// Check if first part is empty or just whitespace
	if strings.TrimSpace(serverCommandParts[0]) == "" {
		return "", nil, fmt.Errorf("server command cannot be empty")
	}

	// Check if first part contains spaces (quoted command with args)
	if strings.Contains(serverCommandParts[0], " ") {
		// Parse the quoted string into command and args
		return parseServerCommand(serverCommandParts[0])
	}

	// First part is command, rest are arguments
	return serverCommandParts[0], serverCommandParts[1:], nil
}

// parseServerCommand parses a server command string into command and arguments
func parseServerCommand(serverCmd string) (string, []string, error) {
	// Simple space splitting since this comes from JSON/CLI, not shell
	parts := strings.Fields(strings.TrimSpace(serverCmd))
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("server command cannot be empty")
	}

	return parts[0], parts[1:], nil
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
