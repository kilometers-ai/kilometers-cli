package event

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestEventID_Creation_ValidatesInput tests EventID creation with various inputs
func TestEventID_Creation_ValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "ValidID_ShouldSucceed",
			input:       "test-id-123",
			expectError: false,
			description: "Valid ID should be accepted",
		},
		{
			name:        "EmptyID_ShouldFail",
			input:       "",
			expectError: true,
			description: "Empty ID should be rejected",
		},
		{
			name:        "LongID_ShouldSucceed",
			input:       "very-long-identifier-with-many-characters-and-numbers-123456789",
			expectError: false,
			description: "Long ID should be accepted",
		},
		{
			name:        "SpecialCharacters_ShouldSucceed",
			input:       "id_with-special.chars@123",
			expectError: false,
			description: "ID with special characters should be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewEventID(tt.input)

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

// TestEventID_Generation_IsUnique tests that generated IDs are unique
func TestEventID_Generation_IsUnique(t *testing.T) {
	const numIDs = 1000
	ids := make(map[string]bool, numIDs)

	for i := 0; i < numIDs; i++ {
		id := GenerateEventID()

		require.NotEmpty(t, id.Value(), "Generated ID should not be empty")
		require.False(t, ids[id.Value()], "Generated ID should be unique, got duplicate: %s", id.Value())

		ids[id.Value()] = true
	}

	assert.Equal(t, numIDs, len(ids), "Should have generated exactly %d unique IDs", numIDs)
}

// TestEventID_GeneratedLength tests that generated IDs have expected length
func TestEventID_GeneratedLength(t *testing.T) {
	const expectedLength = 32 // 16 bytes = 32 hex characters

	for i := 0; i < 100; i++ {
		id := GenerateEventID()
		assert.Equal(t, expectedLength, len(id.Value()), "Generated ID should have length %d", expectedLength)
	}
}

// TestDirection_Creation_ValidatesInput tests Direction creation with various inputs
func TestDirection_Creation_ValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Direction
		expectError bool
		description string
	}{
		{
			name:        "ValidInbound_ShouldSucceed",
			input:       "inbound",
			expected:    DirectionInbound,
			expectError: false,
			description: "Valid inbound direction",
		},
		{
			name:        "ValidOutbound_ShouldSucceed",
			input:       "outbound",
			expected:    DirectionOutbound,
			expectError: false,
			description: "Valid outbound direction",
		},
		{
			name:        "LegacyRequest_ShouldConvertToInbound",
			input:       "request",
			expected:    DirectionInbound,
			expectError: false,
			description: "Legacy request should convert to inbound",
		},
		{
			name:        "LegacyResponse_ShouldConvertToOutbound",
			input:       "response",
			expected:    DirectionOutbound,
			expectError: false,
			description: "Legacy response should convert to outbound",
		},
		{
			name:        "InvalidDirection_ShouldFail",
			input:       "invalid",
			expectError: true,
			description: "Invalid direction should be rejected",
		},
		{
			name:        "EmptyDirection_ShouldFail",
			input:       "",
			expectError: true,
			description: "Empty direction should be rejected",
		},
		{
			name:        "CaseSensitive_ShouldFail",
			input:       "INBOUND",
			expectError: true,
			description: "Case-sensitive validation should reject uppercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			direction, err := NewDirection(tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "invalid direction", "Error should mention invalid direction")
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expected, direction, "Direction should match expected value")
				assert.Equal(t, string(tt.expected), direction.String(), "String() should return correct value")
			}
		})
	}
}

// TestMethod_Creation_ValidatesInput tests Method creation with various inputs
func TestMethod_Creation_ValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "ValidMethod_ShouldSucceed",
			input:       "tools/call",
			expectError: false,
			description: "Valid method name",
		},
		{
			name:        "EmptyMethod_ShouldFail",
			input:       "",
			expectError: true,
			description: "Empty method should be rejected",
		},
		{
			name:        "SimpleMethod_ShouldSucceed",
			input:       "ping",
			expectError: false,
			description: "Simple method name",
		},
		{
			name:        "ComplexMethod_ShouldSucceed",
			input:       "namespace/subnamespace/action",
			expectError: false,
			description: "Complex nested method name",
		},
		{
			name:        "MethodWithSpecialChars_ShouldSucceed",
			input:       "method-with_special.chars123",
			expectError: false,
			description: "Method with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := NewMethod(tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty method")
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.input, method.Value(), "Method value should match input")
				assert.Equal(t, tt.input, method.String(), "String() should match Value()")
			}
		})
	}
}

// TestRiskScore_Creation_ValidatesRange tests RiskScore creation with boundary values
func TestRiskScore_Creation_ValidatesRange(t *testing.T) {
	tests := []struct {
		name          string
		input         int
		expectError   bool
		expectedLevel string
		description   string
	}{
		{
			name:          "MinimumScore_ShouldSucceed",
			input:         0,
			expectError:   false,
			expectedLevel: "low",
			description:   "Minimum score (0) should be accepted",
		},
		{
			name:          "MaximumScore_ShouldSucceed",
			input:         100,
			expectError:   false,
			expectedLevel: "high",
			description:   "Maximum score (100) should be accepted",
		},
		{
			name:          "LowRiskScore_ShouldSucceed",
			input:         25,
			expectError:   false,
			expectedLevel: "low",
			description:   "Low risk score should be categorized correctly",
		},
		{
			name:          "MediumRiskScore_ShouldSucceed",
			input:         50,
			expectError:   false,
			expectedLevel: "medium",
			description:   "Medium risk score should be categorized correctly",
		},
		{
			name:          "HighRiskScore_ShouldSucceed",
			input:         85,
			expectError:   false,
			expectedLevel: "high",
			description:   "High risk score should be categorized correctly",
		},
		{
			name:        "NegativeScore_ShouldFail",
			input:       -1,
			expectError: true,
			description: "Negative score should be rejected",
		},
		{
			name:        "ScoreTooHigh_ShouldFail",
			input:       101,
			expectError: true,
			description: "Score above 100 should be rejected",
		},
		{
			name:        "ScoreWayTooLow_ShouldFail",
			input:       -999,
			expectError: true,
			description: "Very negative score should be rejected",
		},
		{
			name:        "ScoreWayTooHigh_ShouldFail",
			input:       1000,
			expectError: true,
			description: "Very high score should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewRiskScore(tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "must be between 0 and 100", "Error should mention valid range")
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.input, score.Value(), "Score value should match input")
				assert.Equal(t, tt.expectedLevel, score.Level(), "Score level should be correctly categorized")
			}
		})
	}
}

// TestRiskScore_LevelClassification tests risk score level classification logic
func TestRiskScore_LevelClassification(t *testing.T) {
	tests := []struct {
		score    int
		isLow    bool
		isMedium bool
		isHigh   bool
		level    string
	}{
		{0, true, false, false, "low"},
		{34, true, false, false, "low"},
		{35, false, true, false, "medium"},
		{74, false, true, false, "medium"},
		{75, false, false, true, "high"},
		{100, false, false, true, "high"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score%d", tt.score), func(t *testing.T) {
			score, err := NewRiskScore(tt.score)
			require.NoError(t, err)

			assert.Equal(t, tt.isLow, score.IsLow(), "IsLow() classification for score %d", tt.score)
			assert.Equal(t, tt.isMedium, score.IsMedium(), "IsMedium() classification for score %d", tt.score)
			assert.Equal(t, tt.isHigh, score.IsHigh(), "IsHigh() classification for score %d", tt.score)
			assert.Equal(t, tt.level, score.Level(), "Level() classification for score %d", tt.score)
		})
	}
}

// TestEvent_Creation_ValidatesInput tests Event creation with various inputs
func TestEvent_Creation_ValidatesInput(t *testing.T) {
	validID := GenerateEventID()
	validTimestamp := time.Now()
	validDirection, _ := NewDirection("inbound")
	validMethod, _ := NewMethod("test/method")
	validPayload := []byte(`{"test": "data"}`)
	validRiskScore, _ := NewRiskScore(50)

	tests := []struct {
		name        string
		id          EventID
		timestamp   time.Time
		direction   Direction
		method      Method
		payload     []byte
		riskScore   RiskScore
		expectError bool
		description string
	}{
		{
			name:        "ValidEvent_ShouldSucceed",
			id:          validID,
			timestamp:   validTimestamp,
			direction:   validDirection,
			method:      validMethod,
			payload:     validPayload,
			riskScore:   validRiskScore,
			expectError: false,
			description: "Valid event should be created successfully",
		},
		{
			name:        "ZeroTimestamp_ShouldFail",
			id:          validID,
			timestamp:   time.Time{},
			direction:   validDirection,
			method:      validMethod,
			payload:     validPayload,
			riskScore:   validRiskScore,
			expectError: true,
			description: "Zero timestamp should be rejected",
		},
		{
			name:        "EmptyPayload_ShouldFail",
			id:          validID,
			timestamp:   validTimestamp,
			direction:   validDirection,
			method:      validMethod,
			payload:     []byte{},
			riskScore:   validRiskScore,
			expectError: true,
			description: "Empty payload should be rejected",
		},
		{
			name:        "NilPayload_ShouldFail",
			id:          validID,
			timestamp:   validTimestamp,
			direction:   validDirection,
			method:      validMethod,
			payload:     nil,
			riskScore:   validRiskScore,
			expectError: true,
			description: "Nil payload should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := NewEvent(tt.id, tt.timestamp, tt.direction, tt.method, tt.payload, tt.riskScore)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, event, "Event should be nil on error")
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, event, "Event should not be nil on success")

				// Verify all fields are set correctly
				assert.Equal(t, tt.id, event.ID(), "Event ID should match")
				assert.Equal(t, tt.timestamp, event.Timestamp(), "Event timestamp should match")
				assert.Equal(t, tt.direction, event.Direction(), "Event direction should match")
				assert.Equal(t, tt.method, event.Method(), "Event method should match")
				assert.Equal(t, tt.riskScore, event.RiskScore(), "Event risk score should match")
				assert.Equal(t, len(tt.payload), event.Size(), "Event size should match payload length")

				// Verify payload immutability
				retrievedPayload := event.Payload()
				assert.Equal(t, tt.payload, retrievedPayload, "Payload should match")

				// Modify retrieved payload to ensure immutability
				if len(retrievedPayload) > 0 {
					retrievedPayload[0] = byte('X')
					secondRetrieval := event.Payload()
					assert.NotEqual(t, retrievedPayload[0], secondRetrieval[0], "Payload should be immutable")
				}
			}
		})
	}
}

// TestEvent_DirectionChecks tests direction-related methods
func TestEvent_DirectionChecks(t *testing.T) {
	tests := []struct {
		direction  string
		isInbound  bool
		isOutbound bool
	}{
		{"inbound", true, false},
		{"outbound", false, true},
		{"request", true, false},  // legacy
		{"response", false, true}, // legacy
	}

	for _, tt := range tests {
		t.Run(tt.direction, func(t *testing.T) {
			event, err := CreateEvent(
				must(NewDirection(tt.direction)),
				must(NewMethod("test")),
				[]byte(`{"test": true}`),
				must(NewRiskScore(25)),
			)
			require.NoError(t, err)

			assert.Equal(t, tt.isInbound, event.IsInbound(), "IsInbound() for direction %s", tt.direction)
			assert.Equal(t, tt.isOutbound, event.IsOutbound(), "IsOutbound() for direction %s", tt.direction)
		})
	}
}

// TestEvent_RiskChecks tests risk-related methods
func TestEvent_RiskChecks(t *testing.T) {
	tests := []struct {
		riskScore  int
		isHighRisk bool
	}{
		{10, false},
		{34, false},
		{35, false},
		{74, false},
		{75, true},
		{85, true},
		{100, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score%d", tt.riskScore), func(t *testing.T) {
			event, err := CreateEvent(
				DirectionInbound,
				must(NewMethod("test")),
				[]byte(`{"test": true}`),
				must(NewRiskScore(tt.riskScore)),
			)
			require.NoError(t, err)

			assert.Equal(t, tt.isHighRisk, event.IsHighRisk(), "IsHighRisk() for score %d", tt.riskScore)
		})
	}
}

// TestEvent_UpdateRiskScore tests risk score updates
func TestEvent_UpdateRiskScore(t *testing.T) {
	event, err := CreateEvent(
		DirectionInbound,
		must(NewMethod("test")),
		[]byte(`{"test": true}`),
		must(NewRiskScore(25)),
	)
	require.NoError(t, err)

	originalScore := event.RiskScore()
	assert.Equal(t, 25, originalScore.Value())

	newScore, err := NewRiskScore(75)
	require.NoError(t, err)

	event.UpdateRiskScore(newScore)

	updatedScore := event.RiskScore()
	assert.Equal(t, 75, updatedScore.Value())
	assert.True(t, event.IsHighRisk())
}

// TestCreateEvent_FactoryMethod tests the CreateEvent factory method
func TestCreateEvent_FactoryMethod(t *testing.T) {
	direction := DirectionInbound
	method, _ := NewMethod("test/method")
	payload := []byte(`{"factory": "test"}`)
	riskScore, _ := NewRiskScore(42)

	event, err := CreateEvent(direction, method, payload, riskScore)

	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.NotEmpty(t, event.ID().Value(), "Factory method should generate ID")
	assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second, "Factory method should set recent timestamp")
	assert.Equal(t, direction, event.Direction())
	assert.Equal(t, method, event.Method())
	assert.Equal(t, riskScore, event.RiskScore())
}

// TestEvent_String tests the string representation
func TestEvent_String(t *testing.T) {
	event, err := CreateEvent(
		DirectionInbound,
		must(NewMethod("test/method")),
		[]byte(`{"test": "data"}`),
		must(NewRiskScore(50)),
	)
	require.NoError(t, err)

	str := event.String()
	assert.Contains(t, str, "test/method", "String should contain method")
	assert.Contains(t, str, "inbound", "String should contain direction")
	assert.Contains(t, str, "50", "String should contain risk score")
}

// Property-based tests using rapid

// TestEventID_PropertyBased_NonEmptyGeneration tests that all generated IDs are non-empty
func TestEventID_PropertyBased_NonEmptyGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := GenerateEventID()
		assert.NotEmpty(t, id.Value(), "Generated ID should never be empty")
		assert.True(t, len(id.Value()) > 0, "Generated ID should have positive length")
	})
}

// TestRiskScore_PropertyBased_BoundaryValidation tests risk score validation properties
func TestRiskScore_PropertyBased_BoundaryValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		score := rapid.IntRange(-1000, 1000).Draw(t, "score")

		riskScore, err := NewRiskScore(score)

		if score < 0 || score > 100 {
			assert.Error(t, err, "Score outside valid range should be rejected: %d", score)
		} else {
			assert.NoError(t, err, "Score within valid range should be accepted: %d", score)
			assert.Equal(t, score, riskScore.Value(), "Valid score should preserve value")

			// Test level consistency
			level := riskScore.Level()
			if score < 35 {
				assert.Equal(t, "low", level, "Score %d should be low risk", score)
				assert.True(t, riskScore.IsLow(), "Score %d should satisfy IsLow()", score)
			} else if score < 75 {
				assert.Equal(t, "medium", level, "Score %d should be medium risk", score)
				assert.True(t, riskScore.IsMedium(), "Score %d should satisfy IsMedium()", score)
			} else {
				assert.Equal(t, "high", level, "Score %d should be high risk", score)
				assert.True(t, riskScore.IsHigh(), "Score %d should satisfy IsHigh()", score)
			}
		}
	})
}

// TestEvent_PropertyBased_PayloadImmutability tests that event payloads are immutable
func TestEvent_PropertyBased_PayloadImmutability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random payload
		payloadSize := rapid.IntRange(1, 1000).Draw(t, "payloadSize")
		payload := make([]byte, payloadSize)
		for i := range payload {
			payload[i] = byte(rapid.IntRange(0, 255).Draw(t, "byte"))
		}

		event, err := CreateEvent(
			DirectionInbound,
			must(NewMethod("test")),
			payload,
			must(NewRiskScore(50)),
		)
		require.NoError(t, err)

		// Get payload copy
		retrievedPayload := event.Payload()

		// Verify it's a copy by comparing values but different memory
		assert.Equal(t, payload, retrievedPayload, "Retrieved payload should equal original")

		// Modify retrieved payload
		if len(retrievedPayload) > 0 {
			originalByte := retrievedPayload[0]
			retrievedPayload[0] = byte((int(originalByte) + 1) % 256)

			// Get payload again to ensure immutability
			secondRetrieval := event.Payload()
			assert.Equal(t, originalByte, secondRetrieval[0], "Payload should remain immutable")
		}
	})
}

// Benchmark tests for performance validation

func BenchmarkEventID_Generation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateEventID()
	}
}

func BenchmarkEvent_Creation(b *testing.B) {
	id := GenerateEventID()
	direction, _ := NewDirection("inbound")
	method, _ := NewMethod("benchmark/test")
	payload := []byte(`{"benchmark": "data"}`)
	riskScore, _ := NewRiskScore(50)
	timestamp := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewEvent(id, timestamp, direction, method, payload, riskScore)
	}
}

func BenchmarkEvent_PayloadCopy(b *testing.B) {
	event, _ := CreateEvent(
		DirectionInbound,
		must(NewMethod("benchmark")),
		make([]byte, 1024), // 1KB payload
		must(NewRiskScore(50)),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.Payload()
	}
}

// Helper function for tests that need to unwrap values
func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
