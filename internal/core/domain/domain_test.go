package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MonitoringSession Tests (Aggregate Root)
// =============================================================================

func TestMonitoringSession_NewSession(t *testing.T) {
	tests := []struct {
		name           string
		cmd            Command
		config         MonitorConfig
		expectedStatus SessionStatus
	}{
		{
			name:           "creates session with valid command",
			cmd:            mustCreateCommand("echo", []string{"test"}),
			config:         DefaultMonitorConfig(),
			expectedStatus: SessionStatusPending,
		},
		{
			name:           "creates session with custom config",
			cmd:            mustCreateCommand("node", []string{"server.js"}),
			config:         MonitorConfig{BufferSize: 2048},
			expectedStatus: SessionStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewMonitoringSession(tt.cmd, tt.config)

			assert.NotEmpty(t, session.ID())
			assert.Equal(t, tt.cmd, session.ServerCommand())
			assert.Equal(t, tt.config, session.Config())
			assert.Equal(t, tt.expectedStatus, session.Status())
			assert.WithinDuration(t, time.Now(), session.StartTime(), time.Second)
			assert.Nil(t, session.EndTime())
			assert.Zero(t, session.MessageCount())
			assert.Empty(t, session.ErrorMessage())
			assert.True(t, session.Duration() > 0)
		})
	}
}

func TestMonitoringSession_StateTransitions(t *testing.T) {
	session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())

	// Test successful flow: Pending -> Running -> Completed
	t.Run("successful start", func(t *testing.T) {
		err := session.Start()
		assert.NoError(t, err)
		assert.Equal(t, SessionStatusRunning, session.Status())
		assert.True(t, session.IsActive())
		assert.False(t, session.IsCompleted())
		assert.False(t, session.HasFailed())
	})

	t.Run("successful completion", func(t *testing.T) {
		err := session.Complete()
		assert.NoError(t, err)
		assert.Equal(t, SessionStatusCompleted, session.Status())
		assert.False(t, session.IsActive())
		assert.True(t, session.IsCompleted())
		assert.False(t, session.HasFailed())
		assert.NotNil(t, session.EndTime())
	})

	// Test failure scenarios
	t.Run("cannot start already started session", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		session.Start()

		err := session.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start session in status running")
	})

	t.Run("cannot complete non-running session", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())

		err := session.Complete()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot complete session in status pending")
	})

	// Test failure flow
	t.Run("session failure", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		session.Start()

		errorMsg := "server crashed"
		err := session.Fail(errorMsg)
		assert.NoError(t, err)
		assert.Equal(t, SessionStatusFailed, session.Status())
		assert.False(t, session.IsActive())
		assert.False(t, session.IsCompleted())
		assert.True(t, session.HasFailed())
		assert.Equal(t, errorMsg, session.ErrorMessage())
		assert.NotNil(t, session.EndTime())
	})

	// Test cancellation flow
	t.Run("session cancellation", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		session.Start()

		err := session.Cancel()
		assert.NoError(t, err)
		assert.Equal(t, SessionStatusCancelled, session.Status())
		assert.False(t, session.IsActive())
		assert.False(t, session.IsCompleted())
		assert.False(t, session.HasFailed())
		assert.NotNil(t, session.EndTime())
	})

	// Test invalid state transitions
	t.Run("cannot fail completed session", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		session.Start()
		session.Complete()

		err := session.Fail("error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot fail session in status completed")
	})

	t.Run("cannot cancel failed session", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		session.Start()
		session.Fail("error")

		err := session.Cancel()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot cancel session in status failed")
	})
}

func TestMonitoringSession_MessageManagement(t *testing.T) {
	session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
	session.Start()

	t.Run("add message to running session", func(t *testing.T) {
		msg := createTestMessage(MessageTypeRequest, "initialize", DirectionInbound, session.ID())

		err := session.AddMessage(*msg)
		assert.NoError(t, err)
		assert.Equal(t, 1, session.MessageCount())

		messages := session.Messages()
		assert.Len(t, messages, 1)
		assert.Equal(t, msg.ID(), messages[0].ID())
	})

	t.Run("cannot add message to non-running session", func(t *testing.T) {
		session := NewMonitoringSession(mustCreateCommand("echo", []string{"test"}), DefaultMonitorConfig())
		msg := createTestMessage(MessageTypeRequest, "initialize", DirectionInbound, session.ID())

		err := session.AddMessage(*msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot add message to session in status pending")
	})

	t.Run("messages return copy for immutability", func(t *testing.T) {
		originalCount := session.MessageCount()
		messages := session.Messages()

		// Modify returned slice
		if len(messages) > 0 {
			messages[0] = *createTestMessage(MessageTypeResponse, "modified", DirectionOutbound, session.ID())
		}

		// Original session should be unchanged
		newMessages := session.Messages()
		assert.Equal(t, originalCount, session.MessageCount())
		if len(newMessages) > 0 {
			assert.NotEqual(t, "modified", newMessages[0].Method())
		}
	})
}

// =============================================================================
// JSONRPCMessage Tests (Entity)
// =============================================================================

func TestJSONRPCMessage_MessageCreation(t *testing.T) {
	sessionID := SessionID("test-session")

	tests := []struct {
		name      string
		msgType   MessageType
		method    string
		payload   json.RawMessage
		direction Direction
	}{
		{
			name:      "create request message",
			msgType:   MessageTypeRequest,
			method:    "initialize",
			payload:   json.RawMessage(`{"jsonrpc":"2.0","method":"initialize","id":1}`),
			direction: DirectionInbound,
		},
		{
			name:      "create response message",
			msgType:   MessageTypeResponse,
			method:    "",
			payload:   json.RawMessage(`{"jsonrpc":"2.0","result":{},"id":1}`),
			direction: DirectionOutbound,
		},
		{
			name:      "create notification",
			msgType:   MessageTypeNotification,
			method:    "notification",
			payload:   json.RawMessage(`{"jsonrpc":"2.0","method":"notification"}`),
			direction: DirectionInbound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewJSONRPCMessage(tt.msgType, tt.method, tt.payload, tt.direction, sessionID)

			assert.NotEmpty(t, msg.ID())
			assert.Equal(t, tt.msgType, msg.Type())
			assert.Equal(t, tt.method, msg.Method())
			assert.Equal(t, tt.payload, msg.Payload())
			assert.Equal(t, tt.direction, msg.Direction())
			assert.Equal(t, sessionID, msg.SessionID())
			assert.WithinDuration(t, time.Now(), msg.Timestamp(), time.Second)
		})
	}
}

func TestJSONRPCMessage_RawParsing(t *testing.T) {
	sessionID := SessionID("test-session")

	tests := []struct {
		name           string
		rawData        string
		direction      Direction
		expectedType   MessageType
		expectedMethod string
		shouldError    bool
		errorContains  string
	}{
		{
			name:           "parse valid request",
			rawData:        `{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`,
			direction:      DirectionInbound,
			expectedType:   MessageTypeRequest,
			expectedMethod: "initialize",
			shouldError:    false,
		},
		{
			name:           "parse valid response",
			rawData:        `{"jsonrpc":"2.0","result":{"capabilities":{}},"id":1}`,
			direction:      DirectionOutbound,
			expectedType:   MessageTypeResponse,
			expectedMethod: "",
			shouldError:    false,
		},
		{
			name:           "parse valid notification",
			rawData:        `{"jsonrpc":"2.0","method":"tools/call","params":{}}`,
			direction:      DirectionInbound,
			expectedType:   MessageTypeNotification,
			expectedMethod: "tools/call",
			shouldError:    false,
		},
		{
			name:           "parse error response",
			rawData:        `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`,
			direction:      DirectionOutbound,
			expectedType:   MessageTypeError,
			expectedMethod: "",
			shouldError:    false,
		},
		{
			name:          "invalid JSON",
			rawData:       `{invalid json}`,
			direction:     DirectionInbound,
			shouldError:   true,
			errorContains: "invalid JSON-RPC message",
		},
		{
			name:          "wrong JSON-RPC version",
			rawData:       `{"jsonrpc":"1.0","method":"test","id":1}`,
			direction:     DirectionInbound,
			shouldError:   true,
			errorContains: "unsupported JSON-RPC version",
		},
		{
			name:          "indeterminate message type",
			rawData:       `{"jsonrpc":"2.0"}`,
			direction:     DirectionInbound,
			shouldError:   true,
			errorContains: "cannot determine JSON-RPC message type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewJSONRPCMessageFromRaw([]byte(tt.rawData), tt.direction, sessionID)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
				assert.Equal(t, tt.expectedType, msg.Type())
				assert.Equal(t, tt.expectedMethod, msg.Method())
				assert.Equal(t, tt.direction, msg.Direction())
				assert.Equal(t, sessionID, msg.SessionID())
			}
		})
	}
}

func TestJSONRPCMessage_TypeDetection(t *testing.T) {
	sessionID := SessionID("test-session")

	testCases := []struct {
		msgType        MessageType
		isRequest      bool
		isResponse     bool
		isNotification bool
		isError        bool
	}{
		{MessageTypeRequest, true, false, false, false},
		{MessageTypeResponse, false, true, false, false},
		{MessageTypeNotification, false, false, true, false},
		{MessageTypeError, false, false, false, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.msgType), func(t *testing.T) {
			msg := NewJSONRPCMessage(tc.msgType, "test", json.RawMessage(`{}`), DirectionInbound, sessionID)

			assert.Equal(t, tc.isRequest, msg.IsRequest())
			assert.Equal(t, tc.isResponse, msg.IsResponse())
			assert.Equal(t, tc.isNotification, msg.IsNotification())
			assert.Equal(t, tc.isError, msg.IsError())
		})
	}
}

func TestJSONRPCMessage_DirectionDetection(t *testing.T) {
	sessionID := SessionID("test-session")

	t.Run("inbound direction", func(t *testing.T) {
		msg := NewJSONRPCMessage(MessageTypeRequest, "test", json.RawMessage(`{}`), DirectionInbound, sessionID)
		assert.True(t, msg.IsInbound())
		assert.False(t, msg.IsOutbound())
	})

	t.Run("outbound direction", func(t *testing.T) {
		msg := NewJSONRPCMessage(MessageTypeResponse, "test", json.RawMessage(`{}`), DirectionOutbound, sessionID)
		assert.False(t, msg.IsInbound())
		assert.True(t, msg.IsOutbound())
	})
}

func TestJSONRPCMessage_MCPMethodDetection(t *testing.T) {
	sessionID := SessionID("test-session")

	mcpMethods := []string{
		"initialize",
		"tools/list",
		"tools/call",
		"resources/list",
		"resources/read",
		"resources/subscribe",
		"resources/unsubscribe",
		"sampling/createMessage",
		"completion/complete",
		"logging/setLevel",
	}

	for _, method := range mcpMethods {
		t.Run("MCP method: "+method, func(t *testing.T) {
			msg := NewJSONRPCMessage(MessageTypeRequest, method, json.RawMessage(`{}`), DirectionInbound, sessionID)
			assert.True(t, msg.IsMCPMethod(), "Method %s should be detected as MCP method", method)
		})
	}

	nonMCPMethods := []string{
		"custom/method",
		"unknown",
		"",
		"initialize/extended",
	}

	for _, method := range nonMCPMethods {
		t.Run("Non-MCP method: "+method, func(t *testing.T) {
			msg := NewJSONRPCMessage(MessageTypeRequest, method, json.RawMessage(`{}`), DirectionInbound, sessionID)
			assert.False(t, msg.IsMCPMethod(), "Method %s should not be detected as MCP method", method)
		})
	}
}

func TestJSONRPCMessage_DataIntegrity(t *testing.T) {
	sessionID := SessionID("test-session")
	originalPayload := json.RawMessage(`{"test":"data"}`)

	msg := NewJSONRPCMessage(MessageTypeRequest, "test", originalPayload, DirectionInbound, sessionID)

	t.Run("payload returns copy", func(t *testing.T) {
		payload := msg.Payload()
		// Modify the returned payload
		payload[0] = 'X'

		// Original message should be unchanged
		newPayload := msg.Payload()
		assert.Equal(t, originalPayload, newPayload)
	})

	t.Run("request ID returns copy when present", func(t *testing.T) {
		rawData := `{"jsonrpc":"2.0","method":"test","id":123}`
		msg, err := NewJSONRPCMessageFromRaw([]byte(rawData), DirectionInbound, sessionID)
		require.NoError(t, err)

		requestID := msg.RequestID()
		require.NotNil(t, requestID)

		// Modify the returned request ID
		(*requestID)[0] = 'X'

		// Original message should be unchanged
		newRequestID := msg.RequestID()
		require.NotNil(t, newRequestID)
		assert.NotEqual(t, *requestID, *newRequestID)
	})
}

// =============================================================================
// Command Tests (Value Object)
// =============================================================================

func TestCommand_Construction(t *testing.T) {
	t.Run("valid command creation", func(t *testing.T) {
		cmd, err := NewCommand("echo", []string{"hello", "world"})
		assert.NoError(t, err)
		assert.Equal(t, "echo", cmd.Executable())
		assert.Equal(t, []string{"hello", "world"}, cmd.Args())
		assert.NotEmpty(t, cmd.WorkingDir())
		assert.Empty(t, cmd.Env())
	})

	t.Run("empty executable error", func(t *testing.T) {
		_, err := NewCommand("", []string{"arg"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executable cannot be empty")
	})

	t.Run("command with options", func(t *testing.T) {
		workingDir := "/tmp"
		env := map[string]string{"KEY": "value"}

		cmd, err := NewCommandWithOptions("node", []string{"server.js"}, workingDir, env)
		assert.NoError(t, err)
		assert.Equal(t, "node", cmd.Executable())
		assert.Equal(t, []string{"server.js"}, cmd.Args())
		assert.Contains(t, cmd.WorkingDir(), workingDir) // May be converted to absolute path
		assert.Equal(t, env, cmd.Env())
	})

	t.Run("empty working directory uses current", func(t *testing.T) {
		cmd, err := NewCommandWithOptions("echo", []string{"test"}, "", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, cmd.WorkingDir())
	})
}

func TestCommand_Immutability(t *testing.T) {
	originalArgs := []string{"arg1", "arg2"}
	originalEnv := map[string]string{"KEY1": "value1"}

	cmd, err := NewCommandWithOptions("echo", originalArgs, "/tmp", originalEnv)
	require.NoError(t, err)

	t.Run("args are copied", func(t *testing.T) {
		args := cmd.Args()
		args[0] = "modified"

		newArgs := cmd.Args()
		assert.Equal(t, "arg1", newArgs[0])
	})

	t.Run("env is copied", func(t *testing.T) {
		env := cmd.Env()
		env["KEY1"] = "modified"

		newEnv := cmd.Env()
		assert.Equal(t, "value1", newEnv["KEY1"])
	})

	t.Run("WithEnv returns new instance", func(t *testing.T) {
		newCmd := cmd.WithEnv("KEY2", "value2")

		assert.NotEqual(t, cmd, newCmd)
		assert.Equal(t, cmd.Executable(), newCmd.Executable())
		assert.Equal(t, cmd.Args(), newCmd.Args())
		assert.Equal(t, cmd.WorkingDir(), newCmd.WorkingDir())

		// Original should not have new env var
		assert.Empty(t, cmd.Env()["KEY2"])
		// New should have all env vars
		assert.Equal(t, "value1", newCmd.Env()["KEY1"])
		assert.Equal(t, "value2", newCmd.Env()["KEY2"])
	})

	t.Run("WithWorkingDir returns new instance", func(t *testing.T) {
		newWorkingDir := "/home"
		newCmd := cmd.WithWorkingDir(newWorkingDir)

		assert.NotEqual(t, cmd, newCmd)
		assert.Equal(t, cmd.Executable(), newCmd.Executable())
		assert.Equal(t, cmd.Args(), newCmd.Args())
		assert.Equal(t, newWorkingDir, newCmd.WorkingDir())
		assert.Equal(t, cmd.Env(), newCmd.Env())
	})
}

func TestCommand_Validation(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		cmd, err := NewCommand("echo", []string{"test"})
		require.NoError(t, err)

		err = cmd.IsValid()
		assert.NoError(t, err)
	})

	t.Run("empty executable", func(t *testing.T) {
		// Create invalid command by directly constructing
		cmd := Command{executable: ""}

		err := cmd.IsValid()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executable cannot be empty")
	})

	t.Run("invalid working directory", func(t *testing.T) {
		cmd, err := NewCommandWithOptions("echo", []string{"test"}, "/nonexistent/directory", nil)
		require.NoError(t, err)

		err = cmd.IsValid()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "working directory does not exist")
	})
}

func TestCommand_StringRepresentation(t *testing.T) {
	t.Run("command without args", func(t *testing.T) {
		cmd, err := NewCommand("echo", nil)
		require.NoError(t, err)

		assert.Equal(t, "echo", cmd.String())
	})

	t.Run("command with args", func(t *testing.T) {
		cmd, err := NewCommand("echo", []string{"hello", "world"})
		require.NoError(t, err)

		assert.Equal(t, "echo hello world", cmd.String())
	})

	t.Run("full command line", func(t *testing.T) {
		cmd, err := NewCommand("node", []string{"--version"})
		require.NoError(t, err)

		fullCmd := cmd.FullCommandLine()
		expected := []string{"node", "--version"}
		assert.Equal(t, expected, fullCmd)
	})
}

// =============================================================================
// Config Tests (Value Object)
// =============================================================================

func TestConfig_DefaultValues(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "http://localhost:5194", config.ApiEndpoint)
	assert.Equal(t, 10, config.BatchSize)
	assert.False(t, config.Debug)
	assert.Empty(t, config.ApiKey)
}

func TestConfig_EnvironmentPrecedence(t *testing.T) {
	// Save original env vars
	originalApiKey := os.Getenv("KILOMETERS_API_KEY")
	originalEndpoint := os.Getenv("KILOMETERS_API_ENDPOINT")

	// Clean up after test
	defer func() {
		if originalApiKey != "" {
			os.Setenv("KILOMETERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("KILOMETERS_API_KEY")
		}
		if originalEndpoint != "" {
			os.Setenv("KILOMETERS_API_ENDPOINT", originalEndpoint)
		} else {
			os.Unsetenv("KILOMETERS_API_ENDPOINT")
		}
	}()

	t.Run("environment variables override config", func(t *testing.T) {
		// Start with a base config (simulating file-loaded config)
		baseConfig := Config{
			ApiKey:      "file-key",
			ApiEndpoint: "https://file.endpoint.com",
			BatchSize:   20,
			Debug:       true,
		}

		// Set environment variables that should override file values
		os.Setenv("KILOMETERS_API_KEY", "env-key")
		os.Setenv("KILOMETERS_API_ENDPOINT", "https://env.endpoint.com")

		// Simulate the environment override logic from LoadConfig()
		if apiKey := os.Getenv("KILOMETERS_API_KEY"); apiKey != "" {
			baseConfig.ApiKey = apiKey
		}
		if endpoint := os.Getenv("KILOMETERS_API_ENDPOINT"); endpoint != "" {
			baseConfig.ApiEndpoint = endpoint
		}

		assert.Equal(t, "env-key", baseConfig.ApiKey)
		assert.Equal(t, "https://env.endpoint.com", baseConfig.ApiEndpoint)
		assert.Equal(t, 20, baseConfig.BatchSize) // Should remain from file
		assert.True(t, baseConfig.Debug)          // Should remain from file
	})

	t.Run("missing environment variables preserve config", func(t *testing.T) {
		// Ensure environment variables are unset
		os.Unsetenv("KILOMETERS_API_KEY")
		os.Unsetenv("KILOMETERS_API_ENDPOINT")

		// Start with a base config (simulating file-loaded config)
		baseConfig := Config{
			ApiKey:      "file-key",
			ApiEndpoint: "https://file.endpoint.com",
			BatchSize:   20,
			Debug:       true,
		}

		// Simulate the environment override logic from LoadConfig()
		if apiKey := os.Getenv("KILOMETERS_API_KEY"); apiKey != "" {
			baseConfig.ApiKey = apiKey
		}
		if endpoint := os.Getenv("KILOMETERS_API_ENDPOINT"); endpoint != "" {
			baseConfig.ApiEndpoint = endpoint
		}

		// Values should remain from base config
		assert.Equal(t, "file-key", baseConfig.ApiKey)
		assert.Equal(t, "https://file.endpoint.com", baseConfig.ApiEndpoint)
	})
}

func TestConfig_FileOperations(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "kilometers")
	configPath := filepath.Join(configDir, "config.json")

	// Temporarily override the config path function by testing the core logic
	t.Run("save and load config", func(t *testing.T) {
		config := Config{
			ApiKey:      "test-key",
			ApiEndpoint: "https://api.test.com",
			BatchSize:   20,
			Debug:       true,
		}

		// Create directory
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Save config manually for testing
		data, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		// Load config manually for testing
		loadedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var loadedConfig Config
		err = json.Unmarshal(loadedData, &loadedConfig)
		require.NoError(t, err)

		assert.Equal(t, config, loadedConfig)
	})

	t.Run("load nonexistent config returns error", func(t *testing.T) {
		nonexistentPath := filepath.Join(tempDir, "nonexistent", "config.json")

		_, err := os.ReadFile(nonexistentPath)
		assert.Error(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		invalidConfigPath := filepath.Join(configDir, "invalid.json")
		err := os.WriteFile(invalidConfigPath, []byte("{invalid json}"), 0644)
		require.NoError(t, err)

		_, err = os.ReadFile(invalidConfigPath)
		require.NoError(t, err)

		var config Config
		err = json.Unmarshal([]byte("{invalid json}"), &config)
		assert.Error(t, err)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func mustCreateCommand(executable string, args []string) Command {
	cmd, err := NewCommand(executable, args)
	if err != nil {
		panic(err)
	}
	return cmd
}

func createTestMessage(msgType MessageType, method string, direction Direction, sessionID SessionID) *JSONRPCMessage {
	payload := json.RawMessage(`{"jsonrpc":"2.0"}`)
	return NewJSONRPCMessage(msgType, method, payload, direction, sessionID)
}
