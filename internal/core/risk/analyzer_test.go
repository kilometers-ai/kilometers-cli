package risk

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"kilometers.ai/cli/internal/core/event"
)

// Local test helpers

// createTestEvent creates a test event for risk analysis testing
func createTestEvent(method string, payload []byte) *event.Event {
	id := event.GenerateEventID()
	direction, _ := event.NewDirection("inbound")
	methodObj, _ := event.NewMethod(method)
	riskScore, _ := event.NewRiskScore(0) // Start with zero, will be updated by analyzer

	evt, _ := event.NewEvent(id, time.Now(), direction, methodObj, payload, riskScore)
	return evt
}

// TestRiskPattern_Creation_ValidatesInput tests RiskPattern creation with various inputs
func TestRiskPattern_Creation_ValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		level       RiskLevel
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidPattern_ShouldSucceed",
			pattern:     `\.ssh/id_rsa`,
			level:       RiskLevelHigh,
			description: "SSH private key pattern",
			expectError: false,
		},
		{
			name:        "InvalidRegex_ShouldFail",
			pattern:     `[unclosed`,
			level:       RiskLevelHigh,
			description: "Invalid regex pattern",
			expectError: true,
			errorMsg:    "invalid regex pattern",
		},
		{
			name:        "SimplePattern_ShouldSucceed",
			pattern:     `password`,
			level:       RiskLevelMedium,
			description: "Simple password pattern",
			expectError: false,
		},
		{
			name:        "ComplexPattern_ShouldSucceed",
			pattern:     `(?i)BEGIN.*PRIVATE.*KEY`,
			level:       RiskLevelHigh,
			description: "Case-insensitive private key pattern",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := NewRiskPattern(tt.pattern, tt.level, tt.description)

			if tt.expectError {
				assert.Error(t, err, "Should reject invalid pattern")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error should contain expected message")
				assert.Nil(t, pattern, "Should return nil pattern on error")
			} else {
				assert.NoError(t, err, "Should accept valid pattern")
				assert.NotNil(t, pattern, "Should return non-nil pattern")
				assert.Equal(t, tt.level, pattern.Level, "Pattern should have correct level")
				assert.Equal(t, tt.description, pattern.Description, "Pattern should have correct description")
			}
		})
	}
}

// TestRiskPattern_Matches_DetectsCorrectly tests pattern matching functionality
func TestRiskPattern_Matches_DetectsCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		content     string
		shouldMatch bool
	}{
		{
			name:        "SSHKey_ShouldMatch",
			pattern:     `\.ssh/id_rsa`,
			content:     "file:///home/user/.ssh/id_rsa",
			shouldMatch: true,
		},
		{
			name:        "SSHKey_ShouldNotMatch",
			pattern:     `\.ssh/id_rsa`,
			content:     "file:///home/user/documents/file.txt",
			shouldMatch: false,
		},
		{
			name:        "Password_ShouldMatch",
			pattern:     `password.*:`,
			content:     `{"config": {"password": "secret123"}}`,
			shouldMatch: true,
		},
		{
			name:        "CaseInsensitive_ShouldMatch",
			pattern:     `(?i)SECRET`,
			content:     "api_key=Secret123",
			shouldMatch: true,
		},
		{
			name:        "BeginPrivateKey_ShouldMatch",
			pattern:     `BEGIN.*PRIVATE.*KEY`,
			content:     "-----BEGIN RSA PRIVATE KEY-----",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := NewRiskPattern(tt.pattern, RiskLevelHigh, "test pattern")
			require.NoError(t, err)

			matches := pattern.Matches(tt.content)
			assert.Equal(t, tt.shouldMatch, matches, "Pattern matching should be correct")
		})
	}
}

// TestPatternBasedRiskAnalyzer_Creation_InitializesCorrectly tests analyzer creation
func TestPatternBasedRiskAnalyzer_Creation_InitializesCorrectly(t *testing.T) {
	config := RiskAnalyzerConfig{
		HighRiskMethodsOnly: false,
		PayloadSizeLimit:    0,
		CustomPatterns:      []CustomRiskPattern{},
		EnabledCategories:   []string{},
	}

	analyzer := NewPatternBasedRiskAnalyzer(config)

	assert.NotNil(t, analyzer, "Should create non-nil analyzer")

	// Test that default patterns are initialized
	// We can't directly access private fields, but we can test through AnalyzeMethod
	methodRisk := analyzer.AnalyzeMethod("resources/read")
	assert.Equal(t, RiskLevelHigh, methodRisk, "Should recognize high-risk method")

	lowRisk := analyzer.AnalyzeMethod("ping")
	assert.Equal(t, RiskLevelLow, lowRisk, "Should recognize low-risk method")
}

// TestRiskAnalyzer_AnalyzeMethod_FollowsRules tests method-based risk analysis
func TestRiskAnalyzer_AnalyzeMethod_FollowsRules(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	tests := []struct {
		method       string
		expectedRisk RiskLevel
		description  string
	}{
		// High-risk methods
		{"resources/read", RiskLevelHigh, "File system read is high risk"},
		{"tools/call", RiskLevelHigh, "Tool execution is high risk"},
		{"filesystem/write", RiskLevelHigh, "File system write is high risk"},
		{"filesystem/delete", RiskLevelHigh, "File system delete is high risk"},
		{"process/execute", RiskLevelHigh, "Process execution is high risk"},
		{"shell/execute", RiskLevelHigh, "Shell execution is high risk"},

		// Medium-risk methods
		{"prompts/get", RiskLevelMedium, "Prompt access is medium risk"},
		{"resources/write", RiskLevelMedium, "Resource write is medium risk"},
		{"tools/execute", RiskLevelMedium, "Tool execution is medium risk"},
		{"database/query", RiskLevelMedium, "Database query is medium risk"},
		{"api/call", RiskLevelMedium, "API call is medium risk"},
		{"filesystem/read", RiskLevelMedium, "File system read is medium risk"},

		// Low-risk methods
		{"ping", RiskLevelLow, "Ping is low risk"},
		{"initialize", RiskLevelLow, "Initialize is low risk"},
		{"completion/complete", RiskLevelLow, "Completion is low risk"},
		{"logging/log", RiskLevelLow, "Logging is low risk"},

		// Unknown methods should be categorized by pattern
		{"unknown/method", RiskLevelLow, "Unknown method defaults to low risk"},
		{"admin/access", RiskLevelMedium, "Admin methods are medium risk"},
		{"system/execute", RiskLevelHigh, "Execute methods are high risk"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Method_%s", tt.method), func(t *testing.T) {
			risk := analyzer.AnalyzeMethod(tt.method)
			assert.Equal(t, tt.expectedRisk, risk, tt.description)
		})
	}
}

// TestRiskAnalyzer_AnalyzeMethod_EmptyMethod tests edge case handling
func TestRiskAnalyzer_AnalyzeMethod_EmptyMethod(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	risk := analyzer.AnalyzeMethod("")
	assert.Equal(t, RiskLevelLow, risk, "Empty method should be low risk")
}

// TestRiskAnalyzer_AnalyzeContent_HighRiskPatterns tests content analysis for high-risk patterns
func TestRiskAnalyzer_AnalyzeContent_HighRiskPatterns(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	tests := []struct {
		name         string
		content      string
		expectedRisk RiskLevel
		description  string
	}{
		// High-risk file paths
		{
			name:         "EtcPasswd_ShouldBeHighRisk",
			content:      `{"uri": "file:///etc/passwd"}`,
			expectedRisk: RiskLevelHigh,
			description:  "Access to /etc/passwd should be high risk",
		},
		{
			name:         "EtcShadow_ShouldBeHighRisk",
			content:      `{"uri": "file:///etc/shadow"}`,
			expectedRisk: RiskLevelHigh,
			description:  "Access to /etc/shadow should be high risk",
		},
		{
			name:         "SSHPrivateKey_ShouldBeHighRisk",
			content:      `{"path": "/home/user/.ssh/id_rsa"}`,
			expectedRisk: RiskLevelHigh,
			description:  "SSH private key access should be high risk",
		},
		{
			name:         "PrivateKeyContent_ShouldBeHighRisk",
			content:      "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC7",
			expectedRisk: RiskLevelHigh,
			description:  "Private key content should be high risk",
		},
		{
			name:         "RootDirectory_ShouldBeHighRisk",
			content:      `{"path": "/root/secret.txt"}`,
			expectedRisk: RiskLevelHigh,
			description:  "Root directory access should be high risk",
		},
		{
			name:         "ProcAccess_ShouldBeHighRisk",
			content:      `{"uri": "file:///proc/meminfo"}`,
			expectedRisk: RiskLevelHigh,
			description:  "Proc filesystem access should be high risk",
		},

		// Medium-risk patterns
		{
			name:         "EnvFile_ShouldBeMediumRisk",
			content:      `{"path": "/app/config.json"}`,
			expectedRisk: RiskLevelMedium,
			description:  "Config file access should be medium risk",
		},
		{
			name:         "ConfigFile_ShouldBeMediumRisk",
			content:      `{"path": "/app/config.json"}`,
			expectedRisk: RiskLevelMedium,
			description:  "Config file access should be medium risk",
		},
		{
			name:         "SQLQuery_ShouldBeMediumRisk",
			content:      `{"query": "SELECT * FROM users WHERE admin = 1"}`,
			expectedRisk: RiskLevelMedium,
			description:  "SQL query with admin check should be medium risk",
		},
		{
			name:         "PasswordField_ShouldBeMediumRisk",
			content:      `{"password": "secret123"}`,
			expectedRisk: RiskLevelMedium,
			description:  "Password field should be medium risk",
		},

		// Low-risk content
		{
			name:         "RegularFile_ShouldBeLowRisk",
			content:      `{"path": "/home/user/documents/file.txt"}`,
			expectedRisk: RiskLevelLow,
			description:  "Regular file access should be low risk",
		},
		{
			name:         "SimpleMessage_ShouldBeLowRisk",
			content:      `{"message": "Hello world"}`,
			expectedRisk: RiskLevelLow,
			description:  "Simple message should be low risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := analyzer.AnalyzeContent([]byte(tt.content))
			assert.Equal(t, tt.expectedRisk, risk, tt.description)
		})
	}
}

// TestRiskAnalyzer_AnalyzePayloadSize_FollowsLimits tests payload size analysis
func TestRiskAnalyzer_AnalyzePayloadSize_FollowsLimits(t *testing.T) {
	tests := []struct {
		name         string
		sizeLimit    int
		payloadSize  int
		expectedRisk RiskLevel
		description  string
	}{
		{
			name:         "NoLimit_ShouldBeLowRisk",
			sizeLimit:    0,
			payloadSize:  10000,
			expectedRisk: RiskLevelLow,
			description:  "No size limit should always be low risk",
		},
		{
			name:         "UnderLimit_ShouldBeLowRisk",
			sizeLimit:    1000,
			payloadSize:  500,
			expectedRisk: RiskLevelLow,
			description:  "Payload under limit should be low risk",
		},
		{
			name:         "AtLimit_ShouldBeLowRisk",
			sizeLimit:    1000,
			payloadSize:  1000,
			expectedRisk: RiskLevelLow,
			description:  "Payload at limit should be low risk",
		},
		{
			name:         "OverLimit_ShouldBeMediumRisk",
			sizeLimit:    1000,
			payloadSize:  1001,
			expectedRisk: RiskLevelMedium,
			description:  "Payload over limit should be medium risk",
		},
		{
			name:         "WayOverLimit_ShouldBeMediumRisk",
			sizeLimit:    1000,
			payloadSize:  10000,
			expectedRisk: RiskLevelMedium,
			description:  "Payload way over limit should be medium risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RiskAnalyzerConfig{
				PayloadSizeLimit: tt.sizeLimit,
			}
			analyzer := NewPatternBasedRiskAnalyzer(config)

			risk := analyzer.AnalyzePayloadSize(tt.payloadSize)
			assert.Equal(t, tt.expectedRisk, risk, tt.description)
		})
	}
}

// TestRiskAnalyzer_AnalyzeEvent_ComprehensiveAnalysis tests comprehensive event analysis
func TestRiskAnalyzer_AnalyzeEvent_ComprehensiveAnalysis(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	tests := []struct {
		name        string
		method      string
		payload     string
		expectedMin int
		expectedMax int
		description string
	}{
		{
			name:        "HighRiskMethodAndContent_ShouldBeHighScore",
			method:      "tools/call",
			payload:     `{"name": "shell", "arguments": {"command": "rm -rf /etc/passwd"}}`,
			expectedMin: 75,
			expectedMax: 100,
			description: "High-risk method with dangerous content should have high score",
		},
		{
			name:        "MediumRiskMethodAndContent_ShouldBeMediumScore",
			method:      "resources/write",
			payload:     `{"path": "/app/.env", "content": "API_KEY=secret123"}`,
			expectedMin: 35,
			expectedMax: 74,
			description: "Medium-risk method with sensitive content should have medium score",
		},
		{
			name:        "LowRiskMethodAndContent_ShouldBeLowScore",
			method:      "ping",
			payload:     `{}`,
			expectedMin: 0,
			expectedMax: 34,
			description: "Low-risk method with safe content should have low score",
		},
		{
			name:        "HighRiskMethod_LowRiskContent_ShouldBeHigh",
			method:      "filesystem/delete",
			payload:     `{"message": "Hello world"}`,
			expectedMin: 75,
			expectedMax: 100,
			description: "High-risk method should dominate the score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := createTestEvent(tt.method, []byte(tt.payload))

			score, err := analyzer.AnalyzeEvent(evt)
			assert.NoError(t, err, "Analysis should succeed")
			assert.GreaterOrEqual(t, score.Value(), tt.expectedMin, "Score should be at least %d", tt.expectedMin)
			assert.LessOrEqual(t, score.Value(), tt.expectedMax, "Score should be at most %d", tt.expectedMax)
		})
	}
}

// TestRiskAnalyzer_AnalyzeEvent_HandlesNilEvent tests error handling
func TestRiskAnalyzer_AnalyzeEvent_HandlesNilEvent(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	score, err := analyzer.AnalyzeEvent(nil)
	assert.Error(t, err, "Should return error for nil event")
	assert.Contains(t, err.Error(), "cannot be nil", "Error should mention nil event")
	assert.Zero(t, score.Value(), "Score should be zero on error")
}

// TestRiskAnalyzer_CustomPatterns_WorkCorrectly tests custom pattern functionality
func TestRiskAnalyzer_CustomPatterns_WorkCorrectly(t *testing.T) {
	customPatterns := []CustomRiskPattern{
		{
			Pattern:     "CUSTOM_SECRET",
			Level:       RiskLevelHigh,
			Description: "Custom secret pattern",
			Category:    "secrets",
		},
		{
			Pattern:     "TEST_API_KEY",
			Level:       RiskLevelMedium,
			Description: "Test API key pattern",
			Category:    "api",
		},
	}

	config := RiskAnalyzerConfig{
		CustomPatterns: customPatterns,
	}
	analyzer := NewPatternBasedRiskAnalyzer(config)

	tests := []struct {
		content      string
		expectedRisk RiskLevel
		description  string
	}{
		{
			content:      `{"secret": "CUSTOM_SECRET=abc123"}`,
			expectedRisk: RiskLevelHigh,
			description:  "Custom high-risk pattern should be detected",
		},
		{
			content:      `{"api_key": "TEST_API_KEY=xyz789"}`,
			expectedRisk: RiskLevelMedium,
			description:  "Custom medium-risk pattern should be detected",
		},
		{
			content:      `{"message": "nothing suspicious here"}`,
			expectedRisk: RiskLevelLow,
			description:  "Safe content should remain low risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			risk := analyzer.AnalyzeContent([]byte(tt.content))
			assert.Equal(t, tt.expectedRisk, risk, tt.description)
		})
	}
}

// TestRiskAnalyzer_Performance_WithLargePayloads tests performance with large payloads
func TestRiskAnalyzer_Performance_WithLargePayloads(t *testing.T) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	// Create a large payload (1MB)
	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte('A' + (i % 26))
	}

	evt := createTestEvent("performance/test", largePayload)

	// Should complete within reasonable time
	score, err := analyzer.AnalyzeEvent(evt)
	assert.NoError(t, err, "Should handle large payload without error")
	assert.GreaterOrEqual(t, score.Value(), 0, "Score should be valid")
	assert.LessOrEqual(t, score.Value(), 100, "Score should be valid")
}

// Property-based tests using rapid

// TestRiskAnalyzer_PropertyBased_ScoreConsistency tests score consistency properties
func TestRiskAnalyzer_PropertyBased_ScoreConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

		// Generate random method and payload
		methods := []string{"ping", "tools/call", "resources/read", "completion/complete", "filesystem/write"}
		method := rapid.SampledFrom(methods).Draw(t, "method")

		payloadSize := rapid.IntRange(1, 1000).Draw(t, "payloadSize")
		payload := make([]byte, payloadSize)
		for i := range payload {
			payload[i] = byte(rapid.IntRange(32, 126).Draw(t, "char")) // Printable ASCII
		}

		evt := createTestEvent(method, payload)

		score, err := analyzer.AnalyzeEvent(evt)
		assert.NoError(t, err, "Analysis should always succeed for valid events")
		assert.GreaterOrEqual(t, score.Value(), 0, "Score should be >= 0")
		assert.LessOrEqual(t, score.Value(), 100, "Score should be <= 100")

		// Score should be deterministic - same input should give same output
		score2, err := analyzer.AnalyzeEvent(evt)
		assert.NoError(t, err, "Second analysis should succeed")
		assert.Equal(t, score.Value(), score2.Value(), "Score should be deterministic")
	})
}

// TestRiskAnalyzer_PropertyBased_MethodRiskConsistency tests method risk level consistency
func TestRiskAnalyzer_PropertyBased_MethodRiskConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

		// Generate random method name
		methodLen := rapid.IntRange(1, 50).Draw(t, "methodLen")
		method := ""
		for i := 0; i < methodLen; i++ {
			if i > 0 && rapid.Float64().Draw(t, "slash") < 0.2 {
				method += "/"
			} else {
				method += string(rune(rapid.IntRange(97, 122).Draw(t, "char"))) // lowercase letter
			}
		}

		risk := analyzer.AnalyzeMethod(method)

		// Risk level should always be valid
		assert.Contains(t, []RiskLevel{RiskLevelLow, RiskLevelMedium, RiskLevelHigh}, risk, "Risk level should be valid")

		// Same method should always return same risk level
		risk2 := analyzer.AnalyzeMethod(method)
		assert.Equal(t, risk, risk2, "Method risk assessment should be deterministic")
	})
}

// Benchmark tests for performance validation

func BenchmarkRiskAnalyzer_AnalyzeMethod(b *testing.B) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeMethod("tools/call")
	}
}

func BenchmarkRiskAnalyzer_AnalyzeContent_Small(b *testing.B) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})
	content := []byte(`{"path": "/etc/passwd", "action": "read"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeContent(content)
	}
}

func BenchmarkRiskAnalyzer_AnalyzeContent_Large(b *testing.B) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})

	// Create 10KB payload
	content := make([]byte, 10*1024)
	for i := range content {
		content[i] = byte('A' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeContent(content)
	}
}

func BenchmarkRiskAnalyzer_AnalyzeEvent_Complete(b *testing.B) {
	analyzer := NewPatternBasedRiskAnalyzer(RiskAnalyzerConfig{})
	evt := createTestEvent("tools/call", []byte(`{"name": "test", "args": {}}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeEvent(evt)
	}
}
