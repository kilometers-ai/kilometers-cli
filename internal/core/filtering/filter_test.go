package filtering

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/risk"
)

// Local test helpers

// createTestEvent creates a test event for filtering testing
func createTestEvent(method string, direction string, payload []byte, riskScore int) *event.Event {
	id := event.GenerateEventID()
	dir, _ := event.NewDirection(direction)
	methodObj, _ := event.NewMethod(method)
	risk, _ := event.NewRiskScore(riskScore)

	evt, _ := event.NewEvent(id, time.Now(), dir, methodObj, payload, risk)
	return evt
}

// createTestRiskAnalyzer creates a test risk analyzer
func createTestRiskAnalyzer() risk.RiskAnalyzer {
	return risk.NewPatternBasedRiskAnalyzer(risk.RiskAnalyzerConfig{})
}

// TestFilteringRules_DefaultValues tests default filtering rules
func TestFilteringRules_DefaultValues(t *testing.T) {
	rules := DefaultFilteringRules()

	assert.Empty(t, rules.MethodWhitelist, "Default method whitelist should be empty")
	assert.Empty(t, rules.MethodBlacklist, "Default method blacklist should be empty")
	assert.Equal(t, 0, rules.PayloadSizeLimit, "Default payload size limit should be 0")
	assert.Equal(t, risk.RiskLevelLow, rules.MinimumRiskLevel, "Default minimum risk level should be low")
	assert.True(t, rules.ExcludePingMessages, "Default should exclude ping messages")
	assert.False(t, rules.OnlyHighRiskMethods, "Default should not be high risk only")
	assert.Empty(t, rules.DirectionFilter, "Default direction filter should be empty")
	assert.False(t, rules.EnableContentFiltering, "Default content filtering should be disabled")
	assert.Empty(t, rules.ContentBlacklist, "Default content blacklist should be empty")
}

// TestMethodFilter_WildcardMatching_WorksCorrectly tests wildcard matching in method filters
func TestMethodFilter_WildcardMatching_WorksCorrectly(t *testing.T) {
	tests := []struct {
		name          string
		whitelist     []string
		blacklist     []string
		excludePing   bool
		method        string
		shouldCapture bool
		description   string
	}{
		// Whitelist tests
		{
			name:          "ExactWhitelistMatch_ShouldCapture",
			whitelist:     []string{"tools/call"},
			method:        "tools/call",
			shouldCapture: true,
			description:   "Exact whitelist match should be captured",
		},
		{
			name:          "WildcardWhitelistMatch_ShouldCapture",
			whitelist:     []string{"tools/*"},
			method:        "tools/call",
			shouldCapture: true,
			description:   "Wildcard whitelist match should be captured",
		},
		{
			name:          "MultipleWildcardMatch_ShouldCapture",
			whitelist:     []string{"*/read", "tools/*"},
			method:        "tools/execute",
			shouldCapture: true,
			description:   "Multiple wildcard patterns should work",
		},
		{
			name:          "WhitelistNoMatch_ShouldNotCapture",
			whitelist:     []string{"tools/*"},
			method:        "resources/read",
			shouldCapture: false,
			description:   "Method not in whitelist should not be captured",
		},

		// Blacklist tests
		{
			name:          "ExactBlacklistMatch_ShouldNotCapture",
			blacklist:     []string{"tools/call"},
			method:        "tools/call",
			shouldCapture: false,
			description:   "Exact blacklist match should not be captured",
		},
		{
			name:          "WildcardBlacklistMatch_ShouldNotCapture",
			blacklist:     []string{"tools/*"},
			method:        "tools/execute",
			shouldCapture: false,
			description:   "Wildcard blacklist match should not be captured",
		},
		{
			name:          "BlacklistNoMatch_ShouldCapture",
			blacklist:     []string{"tools/*"},
			method:        "resources/read",
			shouldCapture: true,
			description:   "Method not in blacklist should be captured",
		},

		// Ping exclusion tests
		{
			name:          "PingExclusion_Enabled_ShouldNotCapture",
			excludePing:   true,
			method:        "ping",
			shouldCapture: false,
			description:   "Ping should be excluded when ping exclusion is enabled",
		},
		{
			name:          "PingExclusion_Disabled_ShouldCapture",
			excludePing:   false,
			method:        "ping",
			shouldCapture: true,
			description:   "Ping should be captured when ping exclusion is disabled",
		},
		{
			name:          "PingCaseInsensitive_ShouldNotCapture",
			excludePing:   true,
			method:        "PING",
			shouldCapture: false,
			description:   "Ping exclusion should be case insensitive",
		},

		// Combined whitelist and blacklist (blacklist takes precedence)
		{
			name:          "WhitelistAndBlacklist_BlacklistWins",
			whitelist:     []string{"tools/*"},
			blacklist:     []string{"tools/call"},
			method:        "tools/call",
			shouldCapture: false,
			description:   "Blacklist should take precedence over whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewMethodFilter(tt.whitelist, tt.blacklist, tt.excludePing)
			evt := createTestEvent(tt.method, "inbound", []byte(`{}`), 25)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			if !tt.shouldCapture {
				reason := filter.GetFilterReason(evt)
				assert.NotEmpty(t, reason, "Should provide filter reason when not capturing")
			}
		})
	}
}

// TestSizeFilter_EnforcesLimits tests size filter enforcement
func TestSizeFilter_EnforcesLimits(t *testing.T) {
	tests := []struct {
		name          string
		maxSize       int
		payloadSize   int
		shouldCapture bool
		description   string
	}{
		{
			name:          "NoLimit_ShouldAlwaysCapture",
			maxSize:       0,
			payloadSize:   10000,
			shouldCapture: true,
			description:   "No size limit should always capture",
		},
		{
			name:          "UnderLimit_ShouldCapture",
			maxSize:       1000,
			payloadSize:   500,
			shouldCapture: true,
			description:   "Payload under limit should be captured",
		},
		{
			name:          "AtLimit_ShouldCapture",
			maxSize:       1000,
			payloadSize:   1000,
			shouldCapture: true,
			description:   "Payload at exact limit should be captured",
		},
		{
			name:          "OverLimit_ShouldNotCapture",
			maxSize:       1000,
			payloadSize:   1001,
			shouldCapture: false,
			description:   "Payload over limit should not be captured",
		},
		{
			name:          "WayOverLimit_ShouldNotCapture",
			maxSize:       1000,
			payloadSize:   10000,
			shouldCapture: false,
			description:   "Payload way over limit should not be captured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSizeFilter(tt.maxSize)

			// Create payload of specified size
			payload := make([]byte, tt.payloadSize)
			evt := createTestEvent("test/method", "inbound", payload, 25)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			if !tt.shouldCapture {
				reason := filter.GetFilterReason(evt)
				assert.Contains(t, reason, "size", "Filter reason should mention size")
			}
		})
	}
}

// TestRiskFilter_EnforcesRiskLevels tests risk-based filtering
func TestRiskFilter_EnforcesRiskLevels(t *testing.T) {
	tests := []struct {
		name          string
		minimumLevel  risk.RiskLevel
		onlyHighRisk  bool
		eventRisk     int
		shouldCapture bool
		description   string
	}{
		// Minimum risk level tests
		{
			name:          "LowMinimum_LowRisk_ShouldCapture",
			minimumLevel:  risk.RiskLevelLow,
			eventRisk:     10,
			shouldCapture: true,
			description:   "Low risk event should be captured with low minimum",
		},
		{
			name:          "MediumMinimum_LowRisk_ShouldNotCapture",
			minimumLevel:  risk.RiskLevelMedium,
			eventRisk:     10,
			shouldCapture: false,
			description:   "Low risk event should not be captured with medium minimum",
		},
		{
			name:          "MediumMinimum_MediumRisk_ShouldCapture",
			minimumLevel:  risk.RiskLevelMedium,
			eventRisk:     50,
			shouldCapture: true,
			description:   "Medium risk event should be captured with medium minimum",
		},
		{
			name:          "HighMinimum_MediumRisk_ShouldNotCapture",
			minimumLevel:  risk.RiskLevelHigh,
			eventRisk:     50,
			shouldCapture: false,
			description:   "Medium risk event should not be captured with high minimum",
		},
		{
			name:          "HighMinimum_HighRisk_ShouldCapture",
			minimumLevel:  risk.RiskLevelHigh,
			eventRisk:     85,
			shouldCapture: true,
			description:   "High risk event should be captured with high minimum",
		},

		// High risk only tests
		{
			name:          "HighRiskOnly_LowRisk_ShouldNotCapture",
			onlyHighRisk:  true,
			eventRisk:     10,
			shouldCapture: false,
			description:   "Low risk event should not be captured in high risk only mode",
		},
		{
			name:          "HighRiskOnly_MediumRisk_ShouldNotCapture",
			onlyHighRisk:  true,
			eventRisk:     50,
			shouldCapture: false,
			description:   "Medium risk event should not be captured in high risk only mode",
		},
		{
			name:          "HighRiskOnly_HighRisk_ShouldCapture",
			onlyHighRisk:  true,
			eventRisk:     85,
			shouldCapture: true,
			description:   "High risk event should be captured in high risk only mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewRiskFilter(tt.minimumLevel, tt.onlyHighRisk)
			evt := createTestEvent("test/method", "inbound", []byte(`{}`), tt.eventRisk)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			if !tt.shouldCapture {
				reason := filter.GetFilterReason(evt)
				assert.Contains(t, reason, "risk", "Filter reason should mention risk")
			}
		})
	}
}

// TestContentFilter_DetectsBlacklistedContent tests content-based filtering
func TestContentFilter_DetectsBlacklistedContent(t *testing.T) {
	tests := []struct {
		name          string
		blacklist     []string
		enabled       bool
		payload       string
		shouldCapture bool
		description   string
	}{
		// Content filtering disabled
		{
			name:          "Disabled_ShouldAlwaysCapture",
			blacklist:     []string{"secret", "password"},
			enabled:       false,
			payload:       `{"password": "secret123"}`,
			shouldCapture: true,
			description:   "Disabled content filter should always capture",
		},

		// Content filtering enabled
		{
			name:          "Enabled_NoMatch_ShouldCapture",
			blacklist:     []string{"secret", "password"},
			enabled:       true,
			payload:       `{"message": "hello world"}`,
			shouldCapture: true,
			description:   "Content without blacklisted terms should be captured",
		},
		{
			name:          "Enabled_ExactMatch_ShouldNotCapture",
			blacklist:     []string{"secret", "password"},
			enabled:       true,
			payload:       `{"password": "test123"}`,
			shouldCapture: false,
			description:   "Content with blacklisted term should not be captured",
		},
		{
			name:          "Enabled_PartialMatch_ShouldNotCapture",
			blacklist:     []string{"secret"},
			enabled:       true,
			payload:       `{"api_secret": "abc123"}`,
			shouldCapture: false,
			description:   "Content with partial blacklisted term should not be captured",
		},
		{
			name:          "Enabled_CaseInsensitive_ShouldNotCapture",
			blacklist:     []string{"SECRET"},
			enabled:       true,
			payload:       `{"value": "secret123"}`,
			shouldCapture: false,
			description:   "Content filtering should be case insensitive",
		},
		{
			name:          "Enabled_MultiplePatterns_ShouldNotCapture",
			blacklist:     []string{"admin", "root", "password"},
			enabled:       true,
			payload:       `{"user": "admin", "level": "normal"}`,
			shouldCapture: false,
			description:   "Any blacklisted pattern should trigger filtering",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewContentFilter(tt.blacklist, tt.enabled)
			evt := createTestEvent("test/method", "inbound", []byte(tt.payload), 25)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			if !tt.shouldCapture {
				reason := filter.GetFilterReason(evt)
				assert.Contains(t, reason, "content", "Filter reason should mention content")
			}
		})
	}
}

// TestDirectionFilter_FiltersCorrectly tests direction-based filtering
func TestDirectionFilter_FiltersCorrectly(t *testing.T) {
	tests := []struct {
		name              string
		allowedDirections []event.Direction
		eventDirection    string
		shouldCapture     bool
		description       string
	}{
		{
			name:              "EmptyFilter_ShouldCaptureAll",
			allowedDirections: []event.Direction{},
			eventDirection:    "inbound",
			shouldCapture:     true,
			description:       "Empty direction filter should capture all directions",
		},
		{
			name:              "InboundOnly_Inbound_ShouldCapture",
			allowedDirections: []event.Direction{event.DirectionInbound},
			eventDirection:    "inbound",
			shouldCapture:     true,
			description:       "Inbound-only filter should capture inbound events",
		},
		{
			name:              "InboundOnly_Outbound_ShouldNotCapture",
			allowedDirections: []event.Direction{event.DirectionInbound},
			eventDirection:    "outbound",
			shouldCapture:     false,
			description:       "Inbound-only filter should not capture outbound events",
		},
		{
			name:              "OutboundOnly_Outbound_ShouldCapture",
			allowedDirections: []event.Direction{event.DirectionOutbound},
			eventDirection:    "outbound",
			shouldCapture:     true,
			description:       "Outbound-only filter should capture outbound events",
		},
		{
			name:              "Both_Either_ShouldCapture",
			allowedDirections: []event.Direction{event.DirectionInbound, event.DirectionOutbound},
			eventDirection:    "inbound",
			shouldCapture:     true,
			description:       "Both directions filter should capture either direction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewDirectionFilter(tt.allowedDirections)
			evt := createTestEvent("test/method", tt.eventDirection, []byte(`{}`), 25)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			if !tt.shouldCapture {
				reason := filter.GetFilterReason(evt)
				assert.Contains(t, reason, "direction", "Filter reason should mention direction")
			}
		})
	}
}

// TestCompositeFilter_CombinesFiltersLogically tests composite filter logic
func TestCompositeFilter_CombinesFiltersLogically(t *testing.T) {
	riskAnalyzer := createTestRiskAnalyzer()

	tests := []struct {
		name          string
		rules         FilteringRules
		method        string
		direction     string
		payload       string
		riskScore     int
		shouldCapture bool
		description   string
	}{
		{
			name: "AllFiltersPass_ShouldCapture",
			rules: FilteringRules{
				MethodWhitelist:        []string{"tools/*"},
				PayloadSizeLimit:       1000,
				MinimumRiskLevel:       risk.RiskLevelLow,
				ExcludePingMessages:    false,
				DirectionFilter:        []event.Direction{event.DirectionInbound},
				EnableContentFiltering: false,
			},
			method:        "tools/call",
			direction:     "inbound",
			payload:       `{"test": "data"}`,
			riskScore:     25,
			shouldCapture: true,
			description:   "Event passing all filters should be captured",
		},
		{
			name: "MethodFilter_Fails_ShouldNotCapture",
			rules: FilteringRules{
				MethodWhitelist:  []string{"tools/*"},
				PayloadSizeLimit: 1000,
				MinimumRiskLevel: risk.RiskLevelLow,
			},
			method:        "resources/read",
			direction:     "inbound",
			payload:       `{"test": "data"}`,
			riskScore:     25,
			shouldCapture: false,
			description:   "Event failing method filter should not be captured",
		},
		{
			name: "SizeFilter_Fails_ShouldNotCapture",
			rules: FilteringRules{
				MethodWhitelist:  []string{"tools/*"},
				PayloadSizeLimit: 10,
				MinimumRiskLevel: risk.RiskLevelLow,
			},
			method:        "tools/call",
			direction:     "inbound",
			payload:       `{"test": "very long data that exceeds the limit"}`,
			riskScore:     25,
			shouldCapture: false,
			description:   "Event failing size filter should not be captured",
		},
		{
			name: "RiskFilter_Fails_ShouldNotCapture",
			rules: FilteringRules{
				MethodWhitelist:  []string{"ping", "initialize"},
				PayloadSizeLimit: 1000,
				MinimumRiskLevel: risk.RiskLevelHigh,
			},
			method:        "ping", // This will be analyzed as low risk
			direction:     "inbound",
			payload:       `{}`,
			riskScore:     25, // Low risk
			shouldCapture: false,
			description:   "Event failing risk filter should not be captured",
		},
		{
			name: "DirectionFilter_Fails_ShouldNotCapture",
			rules: FilteringRules{
				MethodWhitelist:  []string{"tools/*"},
				PayloadSizeLimit: 1000,
				MinimumRiskLevel: risk.RiskLevelLow,
				DirectionFilter:  []event.Direction{event.DirectionInbound},
			},
			method:        "tools/call",
			direction:     "outbound",
			payload:       `{"test": "data"}`,
			riskScore:     25,
			shouldCapture: false,
			description:   "Event failing direction filter should not be captured",
		},
		{
			name: "ContentFilter_Fails_ShouldNotCapture",
			rules: FilteringRules{
				MethodWhitelist:        []string{"tools/*"},
				PayloadSizeLimit:       1000,
				MinimumRiskLevel:       risk.RiskLevelLow,
				EnableContentFiltering: true,
				ContentBlacklist:       []string{"secret"},
			},
			method:        "tools/call",
			direction:     "inbound",
			payload:       `{"secret": "password123"}`,
			riskScore:     25,
			shouldCapture: false,
			description:   "Event failing content filter should not be captured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewCompositeFilter(tt.rules, riskAnalyzer)
			evt := createTestEvent(tt.method, tt.direction, []byte(tt.payload), tt.riskScore)

			result := filter.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)

			// Test filter reason
			reason := filter.GetFilterReason(evt)
			if tt.shouldCapture {
				assert.Contains(t, reason, "passed", "Should indicate event passed filters")
			} else {
				assert.NotEmpty(t, reason, "Should provide specific filter reason")
			}

			// Check statistics
			stats := filter.GetFilterStatistics()
			assert.Equal(t, 1, stats.TotalEvaluated, "Should track evaluations")
			if tt.shouldCapture {
				assert.Equal(t, 1, stats.TotalCaptured, "Should track captures")
				assert.Equal(t, 0, stats.TotalFiltered, "Should not count as filtered")
			} else {
				assert.Equal(t, 0, stats.TotalCaptured, "Should not count as captured")
				assert.Equal(t, 1, stats.TotalFiltered, "Should track filtered events")
			}
		})
	}
}

// TestCompositeFilter_UpdateRules tests dynamic rule updates
func TestCompositeFilter_UpdateRules(t *testing.T) {
	riskAnalyzer := createTestRiskAnalyzer()

	// Start with permissive rules
	initialRules := FilteringRules{
		MethodWhitelist:  []string{},
		PayloadSizeLimit: 0,
		MinimumRiskLevel: risk.RiskLevelLow,
	}

	filter := NewCompositeFilter(initialRules, riskAnalyzer)
	evt := createTestEvent("test/method", "inbound", []byte(`{"test": "data"}`), 25)

	// Should capture with initial rules
	result := filter.ShouldCapture(evt)
	assert.True(t, result, "Should capture with permissive rules")

	// Update to restrictive rules
	restrictiveRules := FilteringRules{
		MethodWhitelist:  []string{"allowed/*"},
		PayloadSizeLimit: 10,
		MinimumRiskLevel: risk.RiskLevelHigh,
	}

	filter.UpdateRules(restrictiveRules)

	// Should not capture with restrictive rules
	result = filter.ShouldCapture(evt)
	assert.False(t, result, "Should not capture with restrictive rules")
}

// TestFilterChain_CombinesMultipleFilters tests filter chain functionality
func TestFilterChain_CombinesMultipleFilters(t *testing.T) {
	riskAnalyzer := createTestRiskAnalyzer()

	// Create composite filters for the chain
	methodFilterRules := FilteringRules{
		MethodWhitelist: []string{"tools/*", "ping"},
	}
	methodFilter := NewCompositeFilter(methodFilterRules, riskAnalyzer)

	sizeFilterRules := FilteringRules{
		PayloadSizeLimit: 100,
	}
	sizeFilter := NewCompositeFilter(sizeFilterRules, riskAnalyzer)

	riskFilterRules := FilteringRules{
		MinimumRiskLevel: risk.RiskLevelMedium,
	}
	riskFilter := NewCompositeFilter(riskFilterRules, riskAnalyzer)

	chain := NewFilterChain(methodFilter, sizeFilter, riskFilter)

	tests := []struct {
		name          string
		method        string
		payloadSize   int
		riskScore     int
		shouldCapture bool
		description   string
	}{
		{
			name:          "AllPass_ShouldCapture",
			method:        "tools/call",
			payloadSize:   50,
			riskScore:     60,
			shouldCapture: true,
			description:   "Event passing all filters in chain should be captured",
		},
		{
			name:          "FirstFails_ShouldNotCapture",
			method:        "resources/read",
			payloadSize:   50,
			riskScore:     60,
			shouldCapture: false,
			description:   "Event failing first filter should not be captured",
		},
		{
			name:          "SecondFails_ShouldNotCapture",
			method:        "tools/call",
			payloadSize:   150,
			riskScore:     60,
			shouldCapture: false,
			description:   "Event failing second filter should not be captured",
		},
		{
			name:          "ThirdFails_ShouldNotCapture",
			method:        "ping", // ping is low risk
			payloadSize:   50,
			riskScore:     20,
			shouldCapture: false,
			description:   "Event failing third filter (risk) should not be captured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, tt.payloadSize)
			evt := createTestEvent(tt.method, "inbound", payload, tt.riskScore)

			result := chain.ShouldCapture(evt)
			assert.Equal(t, tt.shouldCapture, result, tt.description)
		})
	}
}

// TestFilterChain_ManageFilters tests filter chain management
func TestFilterChain_ManageFilters(t *testing.T) {
	riskAnalyzer := createTestRiskAnalyzer()

	methodFilterRules := FilteringRules{}
	methodFilter := NewCompositeFilter(methodFilterRules, riskAnalyzer)

	sizeFilterRules := FilteringRules{PayloadSizeLimit: 0}
	sizeFilter := NewCompositeFilter(sizeFilterRules, riskAnalyzer)

	chain := NewFilterChain(methodFilter)

	// Add filter
	chain.AddFilter(sizeFilter)

	// Remove filter
	err := chain.RemoveFilter(0)
	assert.NoError(t, err, "Should remove filter successfully")

	// Try to remove invalid index
	err = chain.RemoveFilter(10)
	assert.Error(t, err, "Should return error for invalid index")
}

// Property-based tests using rapid

// TestCompositeFilter_PropertyBased_ConsistentBehavior tests consistent filtering behavior
func TestCompositeFilter_PropertyBased_ConsistentBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		riskAnalyzer := createTestRiskAnalyzer()

		// Generate random filtering rules
		rules := FilteringRules{
			PayloadSizeLimit: rapid.IntRange(0, 2000).Draw(t, "sizeLimit"),
			MinimumRiskLevel: rapid.SampledFrom([]risk.RiskLevel{
				risk.RiskLevelLow, risk.RiskLevelMedium, risk.RiskLevelHigh,
			}).Draw(t, "riskLevel"),
			ExcludePingMessages: rapid.Bool().Draw(t, "excludePing"),
		}

		filter := NewCompositeFilter(rules, riskAnalyzer)

		// Generate random event
		methods := []string{"ping", "tools/call", "resources/read", "completion/complete"}
		method := rapid.SampledFrom(methods).Draw(t, "method")

		payloadSize := rapid.IntRange(1, 3000).Draw(t, "payloadSize")
		payload := make([]byte, payloadSize)

		riskScore := rapid.IntRange(0, 100).Draw(t, "riskScore")

		evt := createTestEvent(method, "inbound", payload, riskScore)

		// Filter should be deterministic
		result1 := filter.ShouldCapture(evt)
		result2 := filter.ShouldCapture(evt)
		assert.Equal(t, result1, result2, "Filter should be deterministic")

		// Statistics should be consistent
		stats := filter.GetFilterStatistics()
		assert.Equal(t, 2, stats.TotalEvaluated, "Should track all evaluations")

		if result1 {
			assert.Equal(t, 2, stats.TotalCaptured, "Should track captures")
		} else {
			assert.Equal(t, 2, stats.TotalFiltered, "Should track filtered events")
		}
	})
}

// Benchmark tests for performance validation

func BenchmarkMethodFilter_WildcardMatching(b *testing.B) {
	filter := NewMethodFilter([]string{"tools/*", "resources/*", "completion/*"}, []string{}, false)
	evt := createTestEvent("tools/call", "inbound", []byte(`{}`), 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.ShouldCapture(evt)
	}
}

func BenchmarkCompositeFilter_AllFilters(b *testing.B) {
	rules := FilteringRules{
		MethodWhitelist:        []string{"tools/*", "resources/*"},
		MethodBlacklist:        []string{"tools/dangerous"},
		PayloadSizeLimit:       1000,
		MinimumRiskLevel:       risk.RiskLevelLow,
		ExcludePingMessages:    true,
		DirectionFilter:        []event.Direction{event.DirectionInbound},
		EnableContentFiltering: true,
		ContentBlacklist:       []string{"secret", "password"},
	}

	riskAnalyzer := createTestRiskAnalyzer()
	filter := NewCompositeFilter(rules, riskAnalyzer)
	evt := createTestEvent("tools/call", "inbound", []byte(`{"action": "safe"}`), 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.ShouldCapture(evt)
	}
}

func BenchmarkContentFilter_LargePayload(b *testing.B) {
	blacklist := []string{"secret", "password", "token", "key", "credential"}
	filter := NewContentFilter(blacklist, true)

	// Create 10KB payload
	payload := make([]byte, 10*1024)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	evt := createTestEvent("test/method", "inbound", payload, 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.ShouldCapture(evt)
	}
}
