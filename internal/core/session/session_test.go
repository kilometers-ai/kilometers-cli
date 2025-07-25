package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"kilometers.ai/cli/internal/core/event"
)

// Local test helpers to avoid import cycle

// createTestEvent creates a test event for session testing
func createTestEvent(method string) *event.Event {
	id := event.GenerateEventID()
	direction, _ := event.NewDirection("inbound")
	methodObj, _ := event.NewMethod(method)
	payload := []byte(fmt.Sprintf(`{"method": "%s", "test": "data"}`, method))
	riskScore, _ := event.NewRiskScore(25)

	evt, _ := event.NewEvent(id, time.Now(), direction, methodObj, payload, riskScore)
	return evt
}

// createSessionBuilder provides a builder for test sessions
type sessionBuilder struct {
	config SessionConfig
}

func newSessionBuilder() *sessionBuilder {
	return &sessionBuilder{
		config: DefaultSessionConfig(),
	}
}

func (b *sessionBuilder) withBatchSize(size int) *sessionBuilder {
	b.config.BatchSize = size
	return b
}

func (b *sessionBuilder) withMaxSessionSize(size int) *sessionBuilder {
	b.config.MaxSessionSize = size
	return b
}

func (b *sessionBuilder) build() *Session {
	return NewSession(b.config)
}

// TestSessionID_Creation_ValidatesInput tests SessionID creation with various inputs
func TestSessionID_Creation_ValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "ValidID_ShouldSucceed",
			input:       "session-123",
			expectError: false,
			description: "Valid session ID should be accepted",
		},
		{
			name:        "EmptyID_ShouldFail",
			input:       "",
			expectError: true,
			description: "Empty session ID should be rejected",
		},
		{
			name:        "LongID_ShouldSucceed",
			input:       "very-long-session-identifier-with-many-characters",
			expectError: false,
			description: "Long session ID should be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewSessionID(tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Empty(t, id.Value(), "Invalid ID should have empty value")
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.input, id.Value(), "Valid ID should preserve input value")
				assert.Equal(t, tt.input, id.String(), "String() should match Value()")
			}
		})
	}
}

// TestSessionID_Generation_IsUnique tests that generated session IDs are unique
func TestSessionID_Generation_IsUnique(t *testing.T) {
	const numIDs = 1000
	ids := make(map[string]bool, numIDs)

	for i := 0; i < numIDs; i++ {
		id := GenerateSessionID()

		require.NotEmpty(t, id.Value(), "Generated session ID should not be empty")
		require.False(t, ids[id.Value()], "Generated session ID should be unique, got duplicate: %s", id.Value())

		ids[id.Value()] = true
	}

	assert.Equal(t, numIDs, len(ids), "Should have generated exactly %d unique session IDs", numIDs)
}

// TestSessionConfig_DefaultValues tests default session configuration
func TestSessionConfig_DefaultValues(t *testing.T) {
	config := DefaultSessionConfig()

	assert.Equal(t, 10, config.BatchSize, "Default batch size should be 10")
	assert.Equal(t, 0, config.MaxSessionSize, "Default max session size should be 0 (no limit)")
}

// TestSession_Creation_InitializesCorrectly tests session creation
func TestSession_Creation_InitializesCorrectly(t *testing.T) {
	config := DefaultSessionConfig()

	// Test NewSession
	session := NewSession(config)

	assert.NotNil(t, session, "NewSession should return non-nil session")
	assert.NotEmpty(t, session.ID().Value(), "Session should have generated ID")
	assert.Equal(t, config, session.Config(), "Session should preserve config")
	assert.Equal(t, SessionStateCreated, session.State(), "New session should be in Created state")
	assert.WithinDuration(t, time.Now(), session.StartTime(), time.Second, "Start time should be recent")
	assert.Equal(t, 0, session.TotalEvents(), "New session should have 0 events")
	assert.Equal(t, 0, session.BatchedEvents(), "New session should have 0 batched events")
	assert.Nil(t, session.EndTime(), "New session should not have end time")
	assert.False(t, session.IsActive(), "New session should not be active")
	assert.False(t, session.IsEnded(), "New session should not be ended")

	// Test NewSessionWithID
	id := GenerateSessionID()
	sessionWithID := NewSessionWithID(id, config)

	assert.Equal(t, id, sessionWithID.ID(), "Session should use provided ID")
	assert.Equal(t, config, sessionWithID.Config(), "Session should preserve config")
}

// TestSession_Lifecycle_TransitionsCorrectly tests session state transitions
func TestSession_Lifecycle_TransitionsCorrectly(t *testing.T) {
	session := newSessionBuilder().build()

	// Test Start transition
	err := session.Start()
	assert.NoError(t, err, "Starting created session should succeed")
	assert.Equal(t, SessionStateActive, session.State(), "Started session should be active")
	assert.True(t, session.IsActive(), "Started session should satisfy IsActive()")

	// Test duplicate Start
	err = session.Start()
	assert.Error(t, err, "Starting active session should fail")
	assert.Contains(t, err.Error(), "already active", "Error should mention session is already active")

	// Test End transition
	batch, err := session.End()
	assert.NoError(t, err, "Ending active session should succeed")
	assert.Equal(t, SessionStateEnded, session.State(), "Ended session should be in ended state")
	assert.True(t, session.IsEnded(), "Ended session should satisfy IsEnded()")
	assert.False(t, session.IsActive(), "Ended session should not be active")
	assert.NotNil(t, session.EndTime(), "Ended session should have end time")
	assert.Nil(t, batch, "Empty session should not produce batch on end")

	// Test operations on ended session
	testEvent := createTestEvent("test/method")
	_, err = session.AddEvent(testEvent)
	assert.Error(t, err, "Adding event to ended session should fail")

	_, err = session.ForceFlush()
	assert.Error(t, err, "Flushing ended session should fail")
}

// TestSession_AddEvent_EnforcesBatchSize tests event addition and batch creation
func TestSession_AddEvent_EnforcesBatchSize(t *testing.T) {
	const batchSize = 3
	session := newSessionBuilder().
		withBatchSize(batchSize).
		build()

	err := session.Start()
	require.NoError(t, err)

	var batches []*EventBatch

	// Add events one by one and check batch creation
	for i := 0; i < batchSize*2+1; i++ {
		testEvent := createTestEvent(fmt.Sprintf("test/method%d", i))

		batch, err := session.AddEvent(testEvent)
		assert.NoError(t, err, "Adding event %d should succeed", i)

		if (i+1)%batchSize == 0 {
			// Should create batch when reaching batch size
			assert.NotNil(t, batch, "Should create batch at event %d", i+1)
			assert.Equal(t, batchSize, batch.Size, "Batch should contain %d events", batchSize)
			assert.Equal(t, session.ID(), batch.SessionID, "Batch should have correct session ID")
			batches = append(batches, batch)
		} else {
			// Should not create batch yet
			assert.Nil(t, batch, "Should not create batch at event %d", i+1)
		}
	}

	assert.Equal(t, 2, len(batches), "Should have created 2 batches")
	assert.Equal(t, batchSize*2+1, session.TotalEvents(), "Session should track total events")
	assert.Equal(t, batchSize*2, session.BatchedEvents(), "Session should track batched events")

	// Check domain events
	domainEvents := session.GetDomainEvents()
	batchReadyEvents := 0
	for _, de := range domainEvents {
		if de.EventName() == "batch_ready" {
			batchReadyEvents++
		}
	}
	assert.Equal(t, 2, batchReadyEvents, "Should have emitted 2 batch_ready domain events")
}

// TestSession_ForceFlush_CreatesBatch tests force flush functionality
func TestSession_ForceFlush_CreatesBatch(t *testing.T) {
	session := newSessionBuilder().
		withBatchSize(10).
		build()

	err := session.Start()
	require.NoError(t, err)

	// Add fewer events than batch size
	for i := 0; i < 3; i++ {
		testEvent := createTestEvent(fmt.Sprintf("test/method%d", i))

		batch, err := session.AddEvent(testEvent)
		assert.NoError(t, err)
		assert.Nil(t, batch, "Should not auto-create batch")
	}

	// Force flush should create batch
	batch, err := session.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")
	assert.NotNil(t, batch, "Force flush should create batch")
	assert.Equal(t, 3, batch.Size, "Batch should contain 3 events")

	// Second force flush should return nil (no pending events)
	batch, err = session.ForceFlush()
	assert.NoError(t, err, "Second force flush should succeed")
	assert.Nil(t, batch, "Second force flush should not create batch")
}

// TestSession_MaxSize_EnforcesLimit tests session size limits
func TestSession_MaxSize_EnforcesLimit(t *testing.T) {
	const maxSize = 5
	session := newSessionBuilder().
		withMaxSessionSize(maxSize).
		withBatchSize(10). // Large batch size to prevent auto-batching
		build()

	err := session.Start()
	require.NoError(t, err)

	// Add events up to limit
	for i := 0; i < maxSize; i++ {
		testEvent := createTestEvent(fmt.Sprintf("test/method%d", i))

		batch, err := session.AddEvent(testEvent)
		assert.NoError(t, err, "Adding event %d should succeed", i)
		assert.Nil(t, batch, "Should not create batch due to large batch size")
	}

	// Adding one more should fail
	testEvent := createTestEvent("test/overflow")

	batch, err := session.AddEvent(testEvent)
	assert.Error(t, err, "Adding event beyond max size should fail")
	assert.Nil(t, batch, "Should not create batch when addition fails")
	assert.Contains(t, err.Error(), "maximum size limit", "Error should mention session size limit")
	assert.Equal(t, maxSize, session.TotalEvents(), "Total events should remain at max size")
}

// TestSession_ConcurrentAccess_IsSafe tests thread safety
func TestSession_ConcurrentAccess_IsSafe(t *testing.T) {
	const numGoroutines = 10
	const eventsPerGoroutine = 20

	session := newSessionBuilder().
		withBatchSize(5).
		build()

	err := session.Start()
	require.NoError(t, err)

	var wg sync.WaitGroup
	var batches []*EventBatch
	var batchMutex sync.Mutex

	// Launch concurrent goroutines to add events
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				testEvent := createTestEvent(fmt.Sprintf("test/goroutine%d/event%d", goroutineID, j))

				batch, err := session.AddEvent(testEvent)
				if err != nil {
					t.Errorf("Concurrent AddEvent failed: %v", err)
					return
				}

				if batch != nil {
					batchMutex.Lock()
					batches = append(batches, batch)
					batchMutex.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	totalExpected := numGoroutines * eventsPerGoroutine
	assert.Equal(t, totalExpected, session.TotalEvents(), "Should have added all events")

	// Force flush to get remaining events
	finalBatch, err := session.ForceFlush()
	assert.NoError(t, err)
	if finalBatch != nil {
		batchMutex.Lock()
		batches = append(batches, finalBatch)
		batchMutex.Unlock()
	}

	// Verify all events are batched
	totalBatchedEvents := 0
	for _, batch := range batches {
		totalBatchedEvents += batch.Size
		assert.Equal(t, session.ID(), batch.SessionID, "All batches should have correct session ID")
		assert.True(t, batch.Size > 0, "All batches should have events")
	}
	assert.Equal(t, totalExpected, totalBatchedEvents, "All events should be batched")
}

// TestEventBatch_Creation_SetsCorrectFields tests event batch creation
func TestEventBatch_Creation_SetsCorrectFields(t *testing.T) {
	sessionID := GenerateSessionID()
	events := []*event.Event{
		createTestEvent("initialize"),
		createTestEvent("tools/call"),
		createTestEvent("ping"),
	}

	batch := NewEventBatch(sessionID, events)

	assert.NotNil(t, batch, "NewEventBatch should return non-nil batch")
	assert.NotEmpty(t, batch.ID, "Batch should have generated ID")
	assert.Equal(t, sessionID, batch.SessionID, "Batch should have correct session ID")
	assert.Equal(t, events, batch.Events, "Batch should contain provided events")
	assert.Equal(t, len(events), batch.Size, "Batch size should match events length")
	assert.WithinDuration(t, time.Now(), batch.CreatedAt, time.Second, "Batch creation time should be recent")
}

// TestSession_DomainEvents_EmittedCorrectly tests domain event emission
func TestSession_DomainEvents_EmittedCorrectly(t *testing.T) {
	session := newSessionBuilder().
		withBatchSize(2).
		build()

	err := session.Start()
	require.NoError(t, err)

	// Add events to trigger batch creation
	event1 := createTestEvent("test/event1")
	event2 := createTestEvent("test/event2")

	_, err = session.AddEvent(event1)
	require.NoError(t, err)

	batch, err := session.AddEvent(event2)
	require.NoError(t, err)
	require.NotNil(t, batch, "Should create batch")

	// End session
	_, err = session.End()
	require.NoError(t, err)

	// Check domain events
	domainEvents := session.GetDomainEvents()
	assert.GreaterOrEqual(t, len(domainEvents), 2, "Should have at least 2 domain events")

	// Verify batch ready event
	var batchReadyEvent *BatchReadyEvent
	var sessionEndedEvent *SessionEndedEvent

	for _, de := range domainEvents {
		switch de.EventName() {
		case "batch_ready":
			if bre, ok := de.(BatchReadyEvent); ok {
				batchReadyEvent = &bre
			}
		case "session_ended":
			if see, ok := de.(SessionEndedEvent); ok {
				sessionEndedEvent = &see
			}
		}
	}

	assert.NotNil(t, batchReadyEvent, "Should have batch ready event")
	assert.Equal(t, session.ID(), batchReadyEvent.SessionID(), "Batch ready event should have correct session ID")
	assert.NotNil(t, batchReadyEvent.Batch(), "Batch ready event should have batch")

	assert.NotNil(t, sessionEndedEvent, "Should have session ended event")
	assert.Equal(t, session.ID(), sessionEndedEvent.SessionID(), "Session ended event should have correct session ID")
	assert.Equal(t, 2, sessionEndedEvent.TotalEvents(), "Session ended event should have correct total events")
	assert.Equal(t, 2, sessionEndedEvent.BatchedEvents(), "Session ended event should have correct batched events")
	assert.True(t, sessionEndedEvent.SessionDuration() > 0, "Session ended event should have positive duration")

	// Test clearing domain events
	session.ClearDomainEvents()
	clearedEvents := session.GetDomainEvents()
	assert.Empty(t, clearedEvents, "Domain events should be cleared")
}

// TestSession_GetEventHistory_ReturnsImmutableCopy tests event history access
func TestSession_GetEventHistory_ReturnsImmutableCopy(t *testing.T) {
	session := newSessionBuilder().
		withBatchSize(10). // Large batch size to prevent auto-batching
		build()

	err := session.Start()
	require.NoError(t, err)

	// Add some events
	originalEvents := []*event.Event{
		createTestEvent("test/event1"),
		createTestEvent("test/event2"),
	}

	for _, evt := range originalEvents {
		_, err := session.AddEvent(evt)
		require.NoError(t, err)
	}

	// Get event history
	history := session.GetEventHistory()
	assert.Equal(t, len(originalEvents), len(history), "History should contain all events")

	// Verify it's a copy (modifying returned slice shouldn't affect session)
	if len(history) > 0 {
		history[0] = nil
		history2 := session.GetEventHistory()
		assert.NotNil(t, history2[0], "Original history should be unaffected")
	}
}

// TestSession_Duration_CalculatedCorrectly tests session duration calculation
func TestSession_Duration_CalculatedCorrectly(t *testing.T) {
	session := newSessionBuilder().build()

	// Before ending, duration should be time since start
	startTime := session.StartTime()
	duration1 := session.Duration()
	expectedDuration1 := time.Since(startTime)
	assert.InDelta(t, expectedDuration1.Seconds(), duration1.Seconds(), 0.1, "Duration should be time since start")

	// End session and check duration calculation
	err := session.Start()
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Small delay

	_, err = session.End()
	require.NoError(t, err)

	duration2 := session.Duration()
	endTime := session.EndTime()
	require.NotNil(t, endTime, "Ended session should have end time")

	expectedDuration2 := endTime.Sub(startTime)
	assert.InDelta(t, expectedDuration2.Seconds(), duration2.Seconds(), 0.001, "Duration should be end time minus start time")
}

// TestSession_String_ContainsRelevantInfo tests string representation
func TestSession_String_ContainsRelevantInfo(t *testing.T) {
	session := newSessionBuilder().build()

	str := session.String()
	assert.Contains(t, str, session.ID().Value(), "String should contain session ID")
	assert.Contains(t, str, "created", "String should contain session state")
}

// Property-based tests using rapid

// TestSession_PropertyBased_BatchSizeConsistency tests batch size consistency
func TestSession_PropertyBased_BatchSizeConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		batchSize := rapid.IntRange(1, 50).Draw(t, "batchSize")
		numEvents := rapid.IntRange(1, 200).Draw(t, "numEvents")

		session := newSessionBuilder().
			withBatchSize(batchSize).
			withMaxSessionSize(1000). // Ensure we don't hit size limit
			build()

		err := session.Start()
		require.NoError(t, err)

		var batches []*EventBatch
		for i := 0; i < numEvents; i++ {
			testEvent := createTestEvent(fmt.Sprintf("test/event%d", i))

			batch, err := session.AddEvent(testEvent)
			assert.NoError(t, err, "Adding event %d should succeed", i)

			if batch != nil {
				batches = append(batches, batch)
				assert.Equal(t, batchSize, batch.Size, "Auto-created batch should have correct size")
			}
		}

		// Force flush remaining events
		finalBatch, err := session.ForceFlush()
		assert.NoError(t, err)
		if finalBatch != nil {
			batches = append(batches, finalBatch)
		}

		// Verify all events are accounted for
		totalBatchedEvents := 0
		for _, batch := range batches {
			totalBatchedEvents += batch.Size
		}
		assert.Equal(t, numEvents, totalBatchedEvents, "All events should be batched")
		assert.Equal(t, numEvents, session.TotalEvents(), "Session should track all events")

		expectedFullBatches := numEvents / batchSize
		remainingEvents := numEvents % batchSize
		expectedTotalBatches := expectedFullBatches
		if remainingEvents > 0 {
			expectedTotalBatches++ // Add one for the final partial batch
		}
		assert.Equal(t, expectedTotalBatches, len(batches), "Should have correct total number of batches")
	})
}

// TestSession_PropertyBased_StateTransitions tests valid state transitions
func TestSession_PropertyBased_StateTransitions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		session := newSessionBuilder().build()

		// Initial state should be Created
		assert.Equal(t, SessionStateCreated, session.State())

		// Start should transition to Active
		err := session.Start()
		assert.NoError(t, err)
		assert.Equal(t, SessionStateActive, session.State())

		// Add some random events
		numEvents := rapid.IntRange(0, 10).Draw(t, "numEvents")
		for i := 0; i < numEvents; i++ {
			testEvent := createTestEvent(fmt.Sprintf("test/event%d", i))
			_, err := session.AddEvent(testEvent)
			assert.NoError(t, err)
		}

		// End should transition to Ended
		_, err = session.End()
		assert.NoError(t, err)
		assert.Equal(t, SessionStateEnded, session.State())

		// All operations on ended session should fail
		testEvent := createTestEvent("test/final")
		_, err = session.AddEvent(testEvent)
		assert.Error(t, err, "Operations on ended session should fail")
	})
}

// Benchmark tests for performance validation

func BenchmarkSessionID_Generation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateSessionID()
	}
}

func BenchmarkSession_AddEvent(b *testing.B) {
	session := newSessionBuilder().
		withBatchSize(1000). // Large batch size to avoid frequent batching
		withMaxSessionSize(10000).
		build()

	session.Start()
	testEvent := createTestEvent("benchmark/test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = session.AddEvent(testEvent)
	}
}

func BenchmarkSession_ConcurrentAddEvent(b *testing.B) {
	session := newSessionBuilder().
		withBatchSize(1000).
		withMaxSessionSize(100000).
		build()

	session.Start()
	testEvent := createTestEvent("benchmark/concurrent")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = session.AddEvent(testEvent)
		}
	})
}

func BenchmarkEventBatch_Creation(b *testing.B) {
	sessionID := GenerateSessionID()
	events := []*event.Event{
		createTestEvent("benchmark/event1"),
		createTestEvent("benchmark/event2"),
		createTestEvent("benchmark/event3"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEventBatch(sessionID, events)
	}
}
