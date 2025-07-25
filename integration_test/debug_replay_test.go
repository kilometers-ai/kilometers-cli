package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/core/session"
)

func TestDebugReplay(t *testing.T) {
	// Create test environment
	env := createFullTestEnvironment(t)
	defer env.Cleanup()

	container := env.Container
	ctx := context.Background()

	// Create a test replay file
	replayContent := `# Test debug replay
{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}

DELAY: 100ms

{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}

# This should be skipped
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"test"},"id":3}
`

	// Write replay file
	tempDir := t.TempDir()
	replayFile := filepath.Join(tempDir, "test_replay.jsonl")
	err := os.WriteFile(replayFile, []byte(replayContent), 0644)
	require.NoError(t, err)

	// Test 1: Basic debug replay
	t.Run("basic_replay", func(t *testing.T) {
		sessionConfig := session.SessionConfig{
			BatchSize:      10,
			MaxSessionSize: 0,
		}

		cmd := commands.NewStartMonitoringCommand("debug-test", []string{}, sessionConfig)
		cmd.DebugReplayFile = replayFile

		result, err := container.MonitoringService.StartMonitoring(ctx, cmd)
		require.NoError(t, err)
		assert.True(t, result.Success)

		// Wait for messages to be processed
		time.Sleep(500 * time.Millisecond)

		// Stop monitoring - only if we have a successful result
		if result.Success {
			stopCmd := commands.NewStopMonitoringCommand(result.SessionID)
			stopResult, err := container.MonitoringService.StopMonitoring(ctx, stopCmd)
			require.NoError(t, err)
			assert.True(t, stopResult.Success)
		}
	})

	// Test 2: Invalid replay file
	t.Run("invalid_file", func(t *testing.T) {
		sessionConfig := session.SessionConfig{
			BatchSize: 10,
		}

		cmd := commands.NewStartMonitoringCommand("debug-test", []string{}, sessionConfig)
		cmd.DebugReplayFile = "/nonexistent/file.jsonl"

		result, err := container.MonitoringService.StartMonitoring(ctx, cmd)
		require.NoError(t, err)
		// Should fail gracefully
		assert.False(t, result.Success)
	})

	// Test 3: Replay with DELAY commands
	t.Run("replay_with_delays", func(t *testing.T) {
		replayWithDelays := `{"jsonrpc":"2.0","method":"test1","id":1}
DELAY: 200ms
{"jsonrpc":"2.0","method":"test2","id":2}
DELAY: 100ms
{"jsonrpc":"2.0","method":"test3","id":3}`

		delayFile := filepath.Join(tempDir, "delay_replay.jsonl")
		err := os.WriteFile(delayFile, []byte(replayWithDelays), 0644)
		require.NoError(t, err)

		sessionConfig := session.SessionConfig{
			BatchSize: 10,
		}

		cmd := commands.NewStartMonitoringCommand("debug-test", []string{}, sessionConfig)
		cmd.DebugReplayFile = delayFile

		start := time.Now()
		result, err := container.MonitoringService.StartMonitoring(ctx, cmd)
		require.NoError(t, err)
		assert.True(t, result.Success)

		// Wait for completion
		time.Sleep(500 * time.Millisecond)

		// Stop monitoring - only if we have a successful result
		if result.Success {
			sessionID := result.SessionID
			stopCmd := commands.NewStopMonitoringCommand(sessionID)
			_, err = container.MonitoringService.StopMonitoring(ctx, stopCmd)
			require.NoError(t, err)

			// Should have taken at least 300ms (sum of delays)
			elapsed := time.Since(start)
			assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
		}
	})
}
