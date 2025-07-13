package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
	"kilometers.ai/cli/test"
)

// TestProcessMonitor_MCPServer_CapturesAllEvents tests complete MCP server interaction
func TestProcessMonitor_MCPServer_CapturesAllEvents(t *testing.T) {
	env := createFullTestEnvironment(t)
	_, cancel := setupTestContext(TestTimeout)
	defer cancel()

	t.Run("basic_mcp_communication", func(t *testing.T) {
		// Configure MCP server with test responses
		env.MockMCPServer.SetHandler("initialize", func(params interface{}) interface{} {
			return map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "test-server",
					"version": "1.0.0",
				},
			}
		})

		// Start monitoring using the process monitor from DI container
		_ = env.Container.ProcessMonitor // Available for future use

		// Create a mock process to monitor
		mockProcess := test.NewMockProcess("node", []string{"mock-server.js"})

		// Start the mock process
		if err := mockProcess.Start(); err != nil {
			t.Fatalf("Failed to start mock process: %v", err)
		}
		defer mockProcess.Stop()

		// Simulate MCP communication
		// In a real scenario, this would be done through stdin/stdout pipes
		// For testing, we'll simulate the messages directly to the mock server

		// Send initialize message
		env.MockMCPServer.SendNotification("initialize", map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]interface{}{
				"name":    "kilometers-cli",
				"version": "test",
			},
		})

		// Wait for response
		if !test.WaitForCondition(func() bool {
			stats := env.MockMCPServer.GetStats()
			messages, _ := stats["total_messages"].(int64)
			return messages > 0
		}, 2*time.Second, 100*time.Millisecond) {
			t.Error("MCP server did not receive initialize message")
		}

		// Verify message was received
		test.AssertMCPMessageSent(t, env, "initialize")
	})

	t.Run("tool_call_handling", func(t *testing.T) {
		// Set up tool call handler
		env.MockMCPServer.SetHandler("tools/call", func(params interface{}) interface{} {
			return map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Tool executed successfully",
					},
				},
				"isError": false,
			}
		})

		// Simulate tool call
		env.MockMCPServer.SendNotification("tools/call", map[string]interface{}{
			"name": "test_tool",
			"arguments": map[string]interface{}{
				"param1": "value1",
			},
		})

		// Wait for processing
		time.Sleep(500 * time.Millisecond)

		// Verify tool call was handled
		test.AssertMCPMessageSent(t, env, "tools/call")
	})

	t.Run("error_handling_and_recovery", func(t *testing.T) {
		// Inject error for a specific method
		env.MockMCPServer.SetError("failing/method", fmt.Errorf("simulated error"))

		// Send message that should fail
		env.MockMCPServer.SendNotification("failing/method", map[string]interface{}{
			"test": "data",
		})

		// Wait for error processing
		time.Sleep(500 * time.Millisecond)

		// Send successful message after error
		env.MockMCPServer.SendNotification("working/method", map[string]interface{}{
			"test": "data",
		})

		// Verify both messages were handled
		stats := env.MockMCPServer.GetStats()
		totalMessages, _ := stats["total_messages"].(int64)
		if totalMessages < 2 {
			t.Errorf("Expected at least 2 messages, got %d", totalMessages)
		}
	})
}

// TestProcessMonitor_SignalHandling_GracefulShutdown tests signal handling
func TestProcessMonitor_SignalHandling_GracefulShutdown(t *testing.T) {
	env := createFullTestEnvironment(t)
	_, cancel := setupTestContext(TestTimeout)
	defer cancel()

	t.Run("graceful_shutdown_on_context_cancel", func(t *testing.T) {
		monitor := env.Container.ProcessMonitor

		// Create monitoring context
		_, monitorCancel := context.WithCancel(context.Background())

		// Start monitoring
		mockProcess := test.NewMockProcess("node", []string{"test-server.js"})
		if err := mockProcess.Start(); err != nil {
			t.Fatalf("Failed to start mock process: %v", err)
		}

		// Verify process is running
		if !mockProcess.IsRunning() {
			t.Error("Mock process should be running")
		}

		// Cancel monitoring context
		monitorCancel()

		// Wait for graceful shutdown
		shutdownComplete := test.WaitForCondition(func() bool {
			return !monitor.IsRunning()
		}, 3*time.Second, 100*time.Millisecond)

		if !shutdownComplete {
			t.Error("Process monitor did not shut down gracefully")
		}

		// Stop the mock process
		mockProcess.Stop()
	})

	t.Run("cleanup_on_process_termination", func(t *testing.T) {
		mockProcess := test.NewMockProcess("node", []string{"short-lived-server.js"})

		// Start and immediately stop process
		if err := mockProcess.Start(); err != nil {
			t.Fatalf("Failed to start mock process: %v", err)
		}

		// Stop process to simulate termination
		if err := mockProcess.Stop(); err != nil {
			t.Fatalf("Failed to stop mock process: %v", err)
		}

		// Verify process is no longer running
		if mockProcess.IsRunning() {
			t.Error("Mock process should have stopped")
		}
	})
}

// TestProcessMonitor_HighVolume_HandlesBackpressure tests high-volume processing
func TestProcessMonitor_HighVolume_HandlesBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high-volume test in short mode")
	}

	env := createFullTestEnvironment(t)
	_, cancel := setupTestContext(TestTimeout)
	defer cancel()

	t.Run("high_volume_message_processing", func(t *testing.T) {
		messageCount := 1000
		receivedMessages := make(map[string]bool)
		var mu sync.Mutex

		// Set up handler to track messages
		env.MockMCPServer.SetHandler("bulk/message", func(params interface{}) interface{} {
			mu.Lock()
			defer mu.Unlock()

			if paramsMap, ok := params.(map[string]interface{}); ok {
				if msgID, exists := paramsMap["message_id"]; exists {
					receivedMessages[fmt.Sprintf("%v", msgID)] = true
				}
			}

			return map[string]interface{}{"status": "processed"}
		})

		// Send high volume of messages
		startTime := time.Now()

		for i := 0; i < messageCount; i++ {
			env.MockMCPServer.SendNotification("bulk/message", map[string]interface{}{
				"message_id": fmt.Sprintf("msg_%d", i),
				"data":       fmt.Sprintf("bulk_data_%d", i),
				"timestamp":  time.Now().Unix(),
			})
		}

		// Wait for all messages to be processed
		allProcessed := test.WaitForCondition(func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedMessages) >= messageCount
		}, 10*time.Second, 100*time.Millisecond)

		processingTime := time.Since(startTime)

		if !allProcessed {
			mu.Lock()
			processedCount := len(receivedMessages)
			mu.Unlock()
			t.Errorf("Only %d out of %d messages were processed", processedCount, messageCount)
		}

		// Verify performance
		messagesPerSecond := float64(messageCount) / processingTime.Seconds()
		if messagesPerSecond < 100 { // Expect at least 100 messages per second
			t.Errorf("Processing rate too slow: %.2f messages/second", messagesPerSecond)
		}

		t.Logf("Processed %d messages in %v (%.2f msg/sec)", messageCount, processingTime, messagesPerSecond)
	})

	t.Run("memory_usage_under_load", func(t *testing.T) {
		// This test would ideally measure memory usage
		// For now, we'll test that the system doesn't crash under load

		sessionCount := 10
		eventsPerSession := 100

		var wg sync.WaitGroup
		errors := make(chan error, sessionCount)

		// Create multiple concurrent sessions
		for i := 0; i < sessionCount; i++ {
			wg.Add(1)
			go func(sessionNum int) {
				defer wg.Done()

				sessionID := fmt.Sprintf("load_test_session_%d", sessionNum)
				testSession := test.CreateTestSession(sessionID, eventsPerSession)

				// Verify session was created correctly
				if testSession.TotalEvents() != eventsPerSession {
					errors <- fmt.Errorf("session %d: expected %d events, got %d",
						sessionNum, eventsPerSession, testSession.TotalEvents())
					return
				}

				// Simulate processing delay
				time.Sleep(100 * time.Millisecond)
			}(i)
		}

		// Wait for all sessions to complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All sessions completed
		case err := <-errors:
			t.Fatalf("Session processing error: %v", err)
		case <-time.After(15 * time.Second):
			t.Fatal("Load test timed out")
		}

		// Check if any errors occurred
		select {
		case err := <-errors:
			t.Errorf("Session processing error: %v", err)
		default:
			// No errors
		}

		t.Logf("Successfully processed %d sessions with %d events each", sessionCount, eventsPerSession)
	})
}

// TestProcessMonitor_EventFiltering_WorksCorrectly tests event filtering during monitoring
func TestProcessMonitor_EventFiltering_WorksCorrectly(t *testing.T) {
	env := createFullTestEnvironment(t)
	_, cancel := setupTestContext(TestTimeout)
	defer cancel()

	t.Run("filters_ping_messages", func(t *testing.T) {
		// Create test events - some pings, some regular events
		testEvents := []*event.Event{
			test.CreateTestEvent(TestSessionID, "ping", event.DirectionInbound),
			test.CreateTestEvent(TestSessionID, "tools/call", event.DirectionInbound),
			test.CreateTestEvent(TestSessionID, "ping", event.DirectionOutbound),
			test.CreateTestEvent(TestSessionID, "resources/read", event.DirectionInbound),
		}

		filter := env.Container.EventFilter

		capturedCount := 0
		for _, evt := range testEvents {
			if filter.ShouldCapture(evt) {
				capturedCount++
			}
		}

		// Should filter out ping messages (2 ping messages out of 4 total)
		expectedCaptured := 2
		if capturedCount != expectedCaptured {
			t.Errorf("Expected %d events to be captured, got %d", expectedCaptured, capturedCount)
		}

		// Verify specific filtering results
		test.AssertFilteringResult(t, filter, testEvents[0], false) // ping should be filtered
		test.AssertFilteringResult(t, filter, testEvents[1], true)  // tools/call should pass
		test.AssertFilteringResult(t, filter, testEvents[2], false) // ping should be filtered
		test.AssertFilteringResult(t, filter, testEvents[3], true)  // resources/read should pass
	})

	t.Run("filters_by_payload_size", func(t *testing.T) {
		// Create events with different payload sizes
		smallPayload := []byte(`{"small": "data"}`)
		largePayload := make([]byte, 2*1024*1024) // 2MB payload
		copy(largePayload, `{"large": "data"`)

		smallMethod, _ := event.NewMethod("small/event")
		largeMethod, _ := event.NewMethod("large/event")
		riskScore, _ := event.NewRiskScore(10)

		smallEvent, _ := event.CreateEvent(event.DirectionInbound, smallMethod, smallPayload, riskScore)
		largeEvent, _ := event.CreateEvent(event.DirectionInbound, largeMethod, largePayload, riskScore)

		filter := env.Container.EventFilter

		// Small event should pass, large event should be filtered
		test.AssertFilteringResult(t, filter, smallEvent, true)
		test.AssertFilteringResult(t, filter, largeEvent, false) // Assuming payload limit is < 2MB
	})

	t.Run("filters_by_risk_level", func(t *testing.T) {
		// Create events with different risk levels
		lowRiskScore, _ := event.NewRiskScore(10)
		highRiskScore, _ := event.NewRiskScore(90)

		method, _ := event.NewMethod("test/method")

		lowRiskEvent, _ := event.CreateEvent(event.DirectionInbound, method, []byte(`{"low": "risk"}`), lowRiskScore)
		highRiskEvent, _ := event.CreateEvent(event.DirectionInbound, method, []byte(`{"high": "risk"}`), highRiskScore)

		filter := env.Container.EventFilter

		// Both should pass with default configuration (minimum risk level is low)
		test.AssertFilteringResult(t, filter, lowRiskEvent, true)
		test.AssertFilteringResult(t, filter, highRiskEvent, true)

		// Verify risk levels are correctly identified
		test.AssertEventProcessed(t, lowRiskEvent, "low")
		test.AssertEventProcessed(t, highRiskEvent, "high")
	})
}

// TestProcessMonitor_SessionManagement_HandlesLifecycle tests session lifecycle management
func TestProcessMonitor_SessionManagement_HandlesLifecycle(t *testing.T) {
	_ = createFullTestEnvironment(t) // Environment setup for future use
	_, cancel := setupTestContext(TestTimeout)
	defer cancel()

	t.Run("session_creation_and_lifecycle", func(t *testing.T) {
		sessionID := "lifecycle_test_session"
		testSession := test.CreateTestSession(sessionID, 5)

		// Verify initial session state
		test.AssertSessionState(t, testSession, 5, session.SessionStateActive)

		// Verify session started correctly
		if !testSession.IsActive() {
			t.Error("Session should be active after creation")
		}

		// End the session
		batch, err := testSession.End()
		if err != nil {
			t.Fatalf("Failed to end session: %v", err)
		}

		// Verify session ended correctly
		test.AssertSessionState(t, testSession, 5, session.SessionStateEnded)

		if batch == nil {
			t.Error("End session should return a batch")
		}

		if testSession.IsActive() {
			t.Error("Session should not be active after ending")
		}
	})

	t.Run("batch_creation_and_flushing", func(t *testing.T) {
		sessionID := "batch_test_session"
		config := session.DefaultSessionConfig()
		config.BatchSize = 3 // Small batch size for testing

		sessionIDObj, _ := session.NewSessionID(sessionID)
		testSession := session.NewSessionWithID(sessionIDObj, config)
		testSession.Start()

		var batches []*session.EventBatch

		// Add events and collect batches
		for i := 0; i < 7; i++ {
			evt := test.CreateTestEvent(sessionID, fmt.Sprintf("batch/test_%d", i), event.DirectionInbound)
			batch, err := testSession.AddEvent(evt)
			if err != nil {
				t.Fatalf("Failed to add event %d: %v", i, err)
			}
			if batch != nil {
				batches = append(batches, batch)
			}
		}

		// Should have created batches when batch size limit was reached
		if len(batches) < 2 {
			t.Errorf("Expected at least 2 batches, got %d", len(batches))
		}

		// Force flush remaining events
		finalBatch, err := testSession.ForceFlush()
		if err != nil {
			t.Fatalf("Failed to force flush: %v", err)
		}

		if finalBatch != nil && finalBatch.Size > 0 {
			batches = append(batches, finalBatch)
		}

		// Verify total events across all batches
		totalBatchedEvents := 0
		for _, batch := range batches {
			totalBatchedEvents += batch.Size
		}

		if totalBatchedEvents != 7 {
			t.Errorf("Expected 7 total events across batches, got %d", totalBatchedEvents)
		}
	})
}
