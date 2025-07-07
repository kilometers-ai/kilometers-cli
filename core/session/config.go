package session

import (
	"errors"
	"time"
)

// SessionConfig - Domain Value Object
// Contains only business rules for session behavior
type SessionConfig struct {
	// Batching behavior (domain rule)
	batchSize     int
	flushInterval time.Duration

	// Event filtering rules (domain rules)
	filteringRules FilteringRules

	// Risk assessment settings (domain rules)
	riskPolicy RiskPolicy
}

// FilteringRules - Value Object for event filtering business rules
type FilteringRules struct {
	methodWhitelist     []string
	payloadSizeLimit    int // bytes, 0 = no limit
	excludePingMessages bool
}

// RiskPolicy - Value Object for risk assessment business rules
type RiskPolicy struct {
	enableRiskDetection bool
	highRiskMethodsOnly bool
	minimumRiskLevel    RiskLevel
}

// Domain errors
var (
	ErrInvalidBatchSize     = errors.New("batch size must be between 1 and 1000")
	ErrInvalidFlushInterval = errors.New("flush interval must be at least 1 second")
	ErrInvalidPayloadLimit  = errors.New("payload size limit cannot be negative")
)

// Constructor with validation (domain rules)
func NewSessionConfig(
	batchSize int,
	flushInterval time.Duration,
	filteringRules FilteringRules,
	riskPolicy RiskPolicy,
) (SessionConfig, error) {

	config := SessionConfig{
		batchSize:      batchSize,
		flushInterval:  flushInterval,
		filteringRules: filteringRules,
		riskPolicy:     riskPolicy,
	}

	if err := config.Validate(); err != nil {
		return SessionConfig{}, err
	}

	return config, nil
}

// Default domain configuration
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		batchSize:     10,
		flushInterval: 5 * time.Second,
		filteringRules: FilteringRules{
			methodWhitelist:     []string{}, // empty = capture all
			payloadSizeLimit:    0,          // 0 = no limit
			excludePingMessages: true,       // exclude noise by default
		},
		riskPolicy: RiskPolicy{
			enableRiskDetection: false,
			highRiskMethodsOnly: false,
			minimumRiskLevel:    Low,
		},
	}
}

// Domain validation rules
func (c SessionConfig) Validate() error {
	if c.batchSize < 1 || c.batchSize > 1000 {
		return ErrInvalidBatchSize
	}

	if c.flushInterval < time.Second {
		return ErrInvalidFlushInterval
	}

	if c.filteringRules.payloadSizeLimit < 0 {
		return ErrInvalidPayloadLimit
	}

	return nil
}

// Domain behavior - should this event be captured?
func (c SessionConfig) ShouldCaptureMethod(method string) bool {
	// Domain rule: ping exclusion
	if c.filteringRules.excludePingMessages && method == "ping" {
		return false
	}

	// Domain rule: method whitelist
	if len(c.filteringRules.methodWhitelist) > 0 {
		return c.isMethodWhitelisted(method)
	}

	return true
}

// Domain behavior - should this payload size be captured?
func (c SessionConfig) ShouldCapturePayloadSize(size int) bool {
	if c.filteringRules.payloadSizeLimit == 0 {
		return true // no limit
	}
	return size <= c.filteringRules.payloadSizeLimit
}

// Domain behavior - should this risk level be captured?
func (c SessionConfig) ShouldCaptureRiskLevel(riskLevel RiskLevel) bool {
	if !c.riskPolicy.enableRiskDetection {
		return true // risk detection disabled, capture all
	}

	if c.riskPolicy.highRiskMethodsOnly {
		return riskLevel == High
	}

	return riskLevel >= c.riskPolicy.minimumRiskLevel
}

// Domain behavior - when should we flush a batch?
func (c SessionConfig) ShouldFlushBatch(eventCount int, timeSinceLastFlush time.Duration) bool {
	// Domain rule: batch size threshold
	if eventCount >= c.batchSize {
		return true
	}

	// Domain rule: time-based flushing
	if timeSinceLastFlush >= c.flushInterval {
		return true
	}

	return false
}

// Getters for immutable access
func (c SessionConfig) BatchSize() int {
	return c.batchSize
}

func (c SessionConfig) FlushInterval() time.Duration {
	return c.flushInterval
}

func (c SessionConfig) FilteringRules() FilteringRules {
	return c.filteringRules
}

func (c SessionConfig) RiskPolicy() RiskPolicy {
	return c.riskPolicy
}

// Private helper methods
func (c SessionConfig) isMethodWhitelisted(method string) bool {
	for _, whitelistedMethod := range c.filteringRules.methodWhitelist {
		if method == whitelistedMethod {
			return true
		}
		// Simple wildcard support
		if c.matchesWildcard(method, whitelistedMethod) {
			return true
		}
	}
	return false
}

func (c SessionConfig) matchesWildcard(method, pattern string) bool {
	// Simple prefix matching for patterns like "tools/*"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(method) >= len(prefix) && method[:len(prefix)] == prefix
	}
	return false
}

// Builder pattern for complex configuration
type SessionConfigBuilder struct {
	batchSize     int
	flushInterval time.Duration
	filtering     FilteringRules
	risk          RiskPolicy
}

func NewSessionConfigBuilder() *SessionConfigBuilder {
	defaults := DefaultSessionConfig()
	return &SessionConfigBuilder{
		batchSize:     defaults.batchSize,
		flushInterval: defaults.flushInterval,
		filtering:     defaults.filteringRules,
		risk:          defaults.riskPolicy,
	}
}

func (b *SessionConfigBuilder) WithBatchSize(size int) *SessionConfigBuilder {
	b.batchSize = size
	return b
}

func (b *SessionConfigBuilder) WithFlushInterval(interval time.Duration) *SessionConfigBuilder {
	b.flushInterval = interval
	return b
}

func (b *SessionConfigBuilder) WithMethodWhitelist(methods []string) *SessionConfigBuilder {
	b.filtering.methodWhitelist = methods
	return b
}

func (b *SessionConfigBuilder) WithPayloadSizeLimit(limit int) *SessionConfigBuilder {
	b.filtering.payloadSizeLimit = limit
	return b
}

func (b *SessionConfigBuilder) WithRiskDetection(enabled bool) *SessionConfigBuilder {
	b.risk.enableRiskDetection = enabled
	return b
}

func (b *SessionConfigBuilder) WithHighRiskOnly(enabled bool) *SessionConfigBuilder {
	b.risk.highRiskMethodsOnly = enabled
	if enabled {
		b.risk.enableRiskDetection = true // auto-enable
	}
	return b
}

func (b *SessionConfigBuilder) Build() (SessionConfig, error) {
	return NewSessionConfig(b.batchSize, b.flushInterval, b.filtering, b.risk)
}

//
//// Conversion from current CLI Config to domain SessionConfig
//func FromCLIConfig(cliConfig *Config) (SessionConfig, error) {
//	return NewSessionConfigBuilder().
//		WithBatchSize(cliConfig.BatchSize).
//		WithMethodWhitelist(cliConfig.MethodWhitelist).
//		WithPayloadSizeLimit(cliConfig.PayloadSizeLimit).
//		WithRiskDetection(cliConfig.EnableRiskDetection).
//		WithHighRiskOnly(cliConfig.HighRiskMethodsOnly).
//		Build()
//}

// Example usage in tests:
/*
func TestSessionConfigValidation(t *testing.T) {
	// Invalid batch size should fail
	_, err := NewSessionConfig(0, time.Second, FilteringRules{}, RiskPolicy{})
	assert.Equal(t, ErrInvalidBatchSize, err)

	// Valid config should succeed
	config, err := NewSessionConfig(10, 5*time.Second, FilteringRules{}, RiskPolicy{})
	assert.NoError(t, err)
	assert.Equal(t, 10, config.BatchSize())
}

func TestSessionConfigBusinessRules(t *testing.T) {
	config := NewSessionConfigBuilder().
		WithMethodWhitelist([]string{"tools/call", "resources/read"}).
		WithRiskDetection(true).
		Build()

	// Should capture whitelisted methods
	assert.True(t, config.ShouldCaptureMethod("tools/call"))

	// Should not capture non-whitelisted methods
	assert.False(t, config.ShouldCaptureMethod("ping"))
}
*/
