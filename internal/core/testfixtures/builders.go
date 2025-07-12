package testfixtures

import (
	"math/rand"
	"time"

	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/filtering"
	"kilometers.ai/cli/internal/core/risk"
	"kilometers.ai/cli/internal/core/session"
)

// EventBuilder provides a builder pattern for creating test events
type EventBuilder struct {
	id        *event.EventID
	timestamp *time.Time
	direction *event.Direction
	method    *event.Method
	payload   []byte
	riskScore *event.RiskScore
}

// NewEventBuilder creates a new EventBuilder with sensible defaults
func NewEventBuilder() *EventBuilder {
	defaultDirection, _ := event.NewDirection("inbound")
	defaultMethod, _ := event.NewMethod("test/method")
	defaultRiskScore, _ := event.NewRiskScore(25)

	return &EventBuilder{
		timestamp: timePtr(time.Now()),
		direction: &defaultDirection,
		method:    &defaultMethod,
		payload:   []byte(`{"test": "payload"}`),
		riskScore: &defaultRiskScore,
	}
}

// WithID sets a specific event ID
func (b *EventBuilder) WithID(id event.EventID) *EventBuilder {
	b.id = &id
	return b
}

// WithGeneratedID generates a new random ID
func (b *EventBuilder) WithGeneratedID() *EventBuilder {
	id := event.GenerateEventID()
	b.id = &id
	return b
}

// WithTimestamp sets a specific timestamp
func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.timestamp = &t
	return b
}

// WithDirection sets the event direction
func (b *EventBuilder) WithDirection(direction string) *EventBuilder {
	d, _ := event.NewDirection(direction)
	b.direction = &d
	return b
}

// WithMethod sets the event method
func (b *EventBuilder) WithMethod(method string) *EventBuilder {
	m, _ := event.NewMethod(method)
	b.method = &m
	return b
}

// WithPayload sets the event payload
func (b *EventBuilder) WithPayload(payload []byte) *EventBuilder {
	b.payload = payload
	return b
}

// WithJSONPayload sets a JSON string payload
func (b *EventBuilder) WithJSONPayload(jsonStr string) *EventBuilder {
	b.payload = []byte(jsonStr)
	return b
}

// WithRiskScore sets the risk score
func (b *EventBuilder) WithRiskScore(score int) *EventBuilder {
	rs, _ := event.NewRiskScore(score)
	b.riskScore = &rs
	return b
}

// WithHighRisk sets a high risk score (80)
func (b *EventBuilder) WithHighRisk() *EventBuilder {
	return b.WithRiskScore(80)
}

// WithMediumRisk sets a medium risk score (50)
func (b *EventBuilder) WithMediumRisk() *EventBuilder {
	return b.WithRiskScore(50)
}

// WithLowRisk sets a low risk score (10)
func (b *EventBuilder) WithLowRisk() *EventBuilder {
	return b.WithRiskScore(10)
}

// Build creates the event
func (b *EventBuilder) Build() (*event.Event, error) {
	id := b.id
	if id == nil {
		generated := event.GenerateEventID()
		id = &generated
	}

	return event.NewEvent(
		*id,
		*b.timestamp,
		*b.direction,
		*b.method,
		b.payload,
		*b.riskScore,
	)
}

// MustBuild creates the event and panics on error (for test convenience)
func (b *EventBuilder) MustBuild() *event.Event {
	evt, err := b.Build()
	if err != nil {
		panic(err)
	}
	return evt
}

// SessionBuilder provides a builder pattern for creating test sessions
type SessionBuilder struct {
	id     *session.SessionID
	config *session.SessionConfig
}

// NewSessionBuilder creates a new SessionBuilder with defaults
func NewSessionBuilder() *SessionBuilder {
	config := session.DefaultSessionConfig()
	return &SessionBuilder{
		config: &config,
	}
}

// WithID sets a specific session ID
func (b *SessionBuilder) WithID(id session.SessionID) *SessionBuilder {
	b.id = &id
	return b
}

// WithGeneratedID generates a new random ID
func (b *SessionBuilder) WithGeneratedID() *SessionBuilder {
	id := session.GenerateSessionID()
	b.id = &id
	return b
}

// WithConfig sets the session configuration
func (b *SessionBuilder) WithConfig(config session.SessionConfig) *SessionBuilder {
	b.config = &config
	return b
}

// WithBatchSize sets the batch size
func (b *SessionBuilder) WithBatchSize(size int) *SessionBuilder {
	b.config.BatchSize = size
	return b
}

// WithFlushInterval sets the flush interval
func (b *SessionBuilder) WithFlushInterval(interval time.Duration) *SessionBuilder {
	b.config.FlushInterval = interval
	return b
}

// WithMaxSessionSize sets the maximum session size
func (b *SessionBuilder) WithMaxSessionSize(size int) *SessionBuilder {
	b.config.MaxSessionSize = size
	return b
}

// WithRiskFiltering enables or disables risk filtering
func (b *SessionBuilder) WithRiskFiltering(enabled bool) *SessionBuilder {
	b.config.EnableRiskFiltering = enabled
	return b
}

// Build creates the session
func (b *SessionBuilder) Build() *session.Session {
	if b.id != nil {
		return session.NewSessionWithID(*b.id, *b.config)
	}
	return session.NewSession(*b.config)
}

// RiskAnalyzerBuilder provides a builder for risk analyzers
type RiskAnalyzerBuilder struct {
	config risk.RiskAnalyzerConfig
}

// NewRiskAnalyzerBuilder creates a new builder with defaults
func NewRiskAnalyzerBuilder() *RiskAnalyzerBuilder {
	return &RiskAnalyzerBuilder{
		config: risk.RiskAnalyzerConfig{
			HighRiskMethodsOnly: false,
			PayloadSizeLimit:    0,
			CustomPatterns:      []risk.CustomRiskPattern{},
			EnabledCategories:   []string{},
		},
	}
}

// WithHighRiskMethodsOnly sets the high risk methods only flag
func (b *RiskAnalyzerBuilder) WithHighRiskMethodsOnly(enabled bool) *RiskAnalyzerBuilder {
	b.config.HighRiskMethodsOnly = enabled
	return b
}

// WithPayloadSizeLimit sets the payload size limit
func (b *RiskAnalyzerBuilder) WithPayloadSizeLimit(limit int) *RiskAnalyzerBuilder {
	b.config.PayloadSizeLimit = limit
	return b
}

// WithCustomPattern adds a custom risk pattern
func (b *RiskAnalyzerBuilder) WithCustomPattern(pattern, level, description, category string) *RiskAnalyzerBuilder {
	riskLevel := risk.RiskLevel(level)
	b.config.CustomPatterns = append(b.config.CustomPatterns, risk.CustomRiskPattern{
		Pattern:     pattern,
		Level:       riskLevel,
		Description: description,
		Category:    category,
	})
	return b
}

// Build creates the risk analyzer
func (b *RiskAnalyzerBuilder) Build() *risk.PatternBasedRiskAnalyzer {
	return risk.NewPatternBasedRiskAnalyzer(b.config)
}

// FilterBuilder provides a builder for event filters
type FilterBuilder struct {
	rules filtering.FilteringRules
}

// NewFilterBuilder creates a new builder with defaults
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		rules: filtering.DefaultFilteringRules(),
	}
}

// WithMethodWhitelist sets the method whitelist
func (b *FilterBuilder) WithMethodWhitelist(methods []string) *FilterBuilder {
	b.rules.MethodWhitelist = methods
	return b
}

// WithMethodBlacklist sets the method blacklist
func (b *FilterBuilder) WithMethodBlacklist(methods []string) *FilterBuilder {
	b.rules.MethodBlacklist = methods
	return b
}

// WithPayloadSizeLimit sets the payload size limit
func (b *FilterBuilder) WithPayloadSizeLimit(limit int) *FilterBuilder {
	b.rules.PayloadSizeLimit = limit
	return b
}

// WithMinimumRiskLevel sets the minimum risk level
func (b *FilterBuilder) WithMinimumRiskLevel(level risk.RiskLevel) *FilterBuilder {
	b.rules.MinimumRiskLevel = level
	return b
}

// WithPingExclusion enables or disables ping message exclusion
func (b *FilterBuilder) WithPingExclusion(exclude bool) *FilterBuilder {
	b.rules.ExcludePingMessages = exclude
	return b
}

// WithHighRiskOnly enables or disables high risk only mode
func (b *FilterBuilder) WithHighRiskOnly(enabled bool) *FilterBuilder {
	b.rules.OnlyHighRiskMethods = enabled
	return b
}

// WithDirectionFilter sets the direction filter
func (b *FilterBuilder) WithDirectionFilter(directions []event.Direction) *FilterBuilder {
	b.rules.DirectionFilter = directions
	return b
}

// WithContentFiltering enables or disables content filtering
func (b *FilterBuilder) WithContentFiltering(enabled bool) *FilterBuilder {
	b.rules.EnableContentFiltering = enabled
	return b
}

// WithContentBlacklist sets the content blacklist
func (b *FilterBuilder) WithContentBlacklist(patterns []string) *FilterBuilder {
	b.rules.ContentBlacklist = patterns
	return b
}

// Build creates the composite filter
func (b *FilterBuilder) Build(riskAnalyzer risk.RiskAnalyzer) *filtering.CompositeFilter {
	return filtering.NewCompositeFilter(b.rules, riskAnalyzer)
}

// Common test data and helper functions

// SampleEvents returns a slice of sample events for testing
func SampleEvents() []*event.Event {
	return []*event.Event{
		NewEventBuilder().WithMethod("initialize").WithLowRisk().MustBuild(),
		NewEventBuilder().WithMethod("tools/call").WithHighRisk().WithJSONPayload(`{"name": "test_tool", "arguments": {}}`).MustBuild(),
		NewEventBuilder().WithMethod("resources/read").WithMediumRisk().WithJSONPayload(`{"uri": "file:///etc/passwd"}`).MustBuild(),
		NewEventBuilder().WithMethod("ping").WithLowRisk().WithJSONPayload(`{}`).MustBuild(),
		NewEventBuilder().WithMethod("completion/complete").WithLowRisk().WithJSONPayload(`{"text": "hello world"}`).MustBuild(),
	}
}

// HighRiskEvents returns events that should be considered high risk
func HighRiskEvents() []*event.Event {
	return []*event.Event{
		NewEventBuilder().WithMethod("tools/call").WithHighRisk().WithJSONPayload(`{"name": "shell", "arguments": {"command": "rm -rf /"}}`).MustBuild(),
		NewEventBuilder().WithMethod("resources/read").WithHighRisk().WithJSONPayload(`{"uri": "file:///etc/shadow"}`).MustBuild(),
		NewEventBuilder().WithMethod("filesystem/write").WithHighRisk().WithJSONPayload(`{"path": "/etc/passwd", "content": "malicious"}`).MustBuild(),
	}
}

// LowRiskEvents returns events that should be considered low risk
func LowRiskEvents() []*event.Event {
	return []*event.Event{
		NewEventBuilder().WithMethod("ping").WithLowRisk().WithJSONPayload(`{}`).MustBuild(),
		NewEventBuilder().WithMethod("initialize").WithLowRisk().WithJSONPayload(`{"capabilities": {}}`).MustBuild(),
		NewEventBuilder().WithMethod("logging/log").WithLowRisk().WithJSONPayload(`{"level": "info", "message": "test"}`).MustBuild(),
	}
}

// LargePayloadEvent returns an event with a large payload for testing
func LargePayloadEvent(sizeBytes int) *event.Event {
	payload := make([]byte, sizeBytes)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	return NewEventBuilder().
		WithMethod("large/payload").
		WithPayload(payload).
		WithMediumRisk().
		MustBuild()
}

// RandomEvent generates a random event for property-based testing
func RandomEvent(rng *rand.Rand) *event.Event {
	methods := []string{"initialize", "tools/call", "resources/read", "ping", "completion/complete", "filesystem/write"}
	directions := []string{"inbound", "outbound"}

	method := methods[rng.Intn(len(methods))]
	direction := directions[rng.Intn(len(directions))]

	payloadSize := rng.Intn(1000) + 10
	payload := make([]byte, payloadSize)
	rng.Read(payload)

	riskScore := rng.Intn(101)

	return NewEventBuilder().
		WithMethod(method).
		WithDirection(direction).
		WithPayload(payload).
		WithRiskScore(riskScore).
		MustBuild()
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

// ValidEventIDs returns a slice of valid event IDs for testing
func ValidEventIDs() []string {
	return []string{
		"test-id-1",
		"a1b2c3d4e5f6",
		"event_123",
		"very-long-event-identifier-with-many-characters",
	}
}

// InvalidEventIDs returns a slice of invalid event IDs for testing
func InvalidEventIDs() []string {
	return []string{
		"", // empty
	}
}

// ValidMethods returns a slice of valid method names for testing
func ValidMethods() []string {
	return []string{
		"initialize",
		"tools/call",
		"resources/read",
		"completion/complete",
		"ping",
		"method-with-dashes",
		"method_with_underscores",
		"method.with.dots",
		"a",
		"very/long/method/name/with/many/parts",
	}
}

// InvalidMethods returns a slice of invalid method names for testing
func InvalidMethods() []string {
	return []string{
		"", // empty
	}
}

// ValidDirections returns valid directions for testing
func ValidDirections() []string {
	return []string{"inbound", "outbound", "request", "response"}
}

// InvalidDirections returns invalid directions for testing
func InvalidDirections() []string {
	return []string{"", "invalid", "up", "down", "left", "right"}
}

// ValidRiskScores returns valid risk scores for testing
func ValidRiskScores() []int {
	return []int{0, 1, 25, 34, 35, 50, 74, 75, 99, 100}
}

// InvalidRiskScores returns invalid risk scores for testing
func InvalidRiskScores() []int {
	return []int{-1, -100, 101, 1000, -999}
}
