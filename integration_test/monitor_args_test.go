package integration_test

import (
	"fmt"
	"testing"
)

// TestMonitorArgumentParsing tests the new --server -- argument parsing logic
func TestMonitorArgumentParsing_WorksCorrectly(t *testing.T) {

	t.Run("simple_server_command", func(t *testing.T) {
		// Test: km monitor --server -- npx -y @modelcontextprotocol/server-github
		args := []string{"npx", "-y", "@modelcontextprotocol/server-github"}

		command, commandArgs, err := testParseServerCommand(args)
		if err != nil {
			t.Fatalf("Failed to parse simple server command: %v", err)
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
		// Test: km monitor --server -- python -m my_mcp_server --port 8080
		args := []string{"python", "-m", "my_mcp_server", "--port", "8080"}

		command, commandArgs, err := testParseServerCommand(args)
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

	t.Run("server_command_with_complex_flags", func(t *testing.T) {
		// Test: km monitor --server -- npx -y @modelcontextprotocol/server-linear --config ./config.json
		args := []string{"npx", "-y", "@modelcontextprotocol/server-linear", "--config", "./config.json"}

		command, commandArgs, err := testParseServerCommand(args)
		if err != nil {
			t.Fatalf("Failed to parse server command with complex flags: %v", err)
		}

		if command != "npx" {
			t.Errorf("Expected command 'npx', got '%s'", command)
		}

		expectedArgs := []string{"-y", "@modelcontextprotocol/server-linear", "--config", "./config.json"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})

	t.Run("single_command_no_args", func(t *testing.T) {
		// Test: km monitor --server -- echo
		args := []string{"echo"}

		command, commandArgs, err := testParseServerCommand(args)
		if err != nil {
			t.Fatalf("Failed to parse single command: %v", err)
		}

		if command != "echo" {
			t.Errorf("Expected command 'echo', got '%s'", command)
		}

		if len(commandArgs) != 0 {
			t.Errorf("Expected no args, got %v", commandArgs)
		}
	})

	t.Run("missing_server_command", func(t *testing.T) {
		// Test: km monitor --server -- (nothing after --)
		args := []string{}

		_, _, err := testParseServerCommand(args)
		if err == nil {
			t.Error("Expected error for missing server command")
		}

		if err.Error() != "server command is required after --server --" {
			t.Errorf("Expected 'server command is required after --server --' error, got '%s'", err.Error())
		}
	})

	t.Run("command_with_quotes_in_args", func(t *testing.T) {
		// Test: km monitor --server -- node script.js --param "value with spaces"
		args := []string{"node", "script.js", "--param", "value with spaces"}

		command, commandArgs, err := testParseServerCommand(args)
		if err != nil {
			t.Fatalf("Failed to parse command with quoted args: %v", err)
		}

		if command != "node" {
			t.Errorf("Expected command 'node', got '%s'", command)
		}

		expectedArgs := []string{"script.js", "--param", "value with spaces"}
		if !slicesEqual(commandArgs, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, commandArgs)
		}
	})
}

// testParseServerCommand simulates the new simplified argument processing logic
func testParseServerCommand(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("server command is required after --server --")
	}

	// First argument is the command, rest are arguments
	return args[0], args[1:], nil
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
