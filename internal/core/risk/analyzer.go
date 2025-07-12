package risk

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"kilometers.ai/cli/internal/core/event"
)

// RiskLevel represents the categorization of risk
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// RiskPattern represents a pattern used for risk detection
type RiskPattern struct {
	Pattern     *regexp.Regexp
	Level       RiskLevel
	Description string
}

// NewRiskPattern creates a new risk pattern with validation
func NewRiskPattern(pattern string, level RiskLevel, description string) (*RiskPattern, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %s: %w", pattern, err)
	}

	return &RiskPattern{
		Pattern:     regex,
		Level:       level,
		Description: description,
	}, nil
}

// Matches checks if the pattern matches the given content
func (rp *RiskPattern) Matches(content string) bool {
	return rp.Pattern.MatchString(content)
}

// RiskAnalyzer defines the interface for risk analysis
type RiskAnalyzer interface {
	AnalyzeEvent(event *event.Event) (event.RiskScore, error)
	AnalyzeContent(content []byte) RiskLevel
	AnalyzeMethod(method string) RiskLevel
	AnalyzePayloadSize(size int) RiskLevel
}

// PatternBasedRiskAnalyzer implements risk analysis using predefined patterns
type PatternBasedRiskAnalyzer struct {
	highRiskPatterns    []*RiskPattern
	mediumRiskPatterns  []*RiskPattern
	lowRiskPatterns     []*RiskPattern
	methodRiskMap       map[string]RiskLevel
	highRiskMethodsOnly bool
	payloadSizeLimit    int
}

// NewPatternBasedRiskAnalyzer creates a new pattern-based risk analyzer
func NewPatternBasedRiskAnalyzer(config RiskAnalyzerConfig) *PatternBasedRiskAnalyzer {
	analyzer := &PatternBasedRiskAnalyzer{
		highRiskPatterns:    make([]*RiskPattern, 0),
		mediumRiskPatterns:  make([]*RiskPattern, 0),
		lowRiskPatterns:     make([]*RiskPattern, 0),
		methodRiskMap:       make(map[string]RiskLevel),
		highRiskMethodsOnly: config.HighRiskMethodsOnly,
		payloadSizeLimit:    config.PayloadSizeLimit,
	}

	// Initialize default patterns
	analyzer.initializeDefaultPatterns()

	return analyzer
}

// RiskAnalyzerConfig holds configuration for the risk analyzer
type RiskAnalyzerConfig struct {
	HighRiskMethodsOnly bool
	PayloadSizeLimit    int // bytes, 0 = no limit
	CustomPatterns      []CustomRiskPattern
	EnabledCategories   []string
}

// CustomRiskPattern allows users to define custom risk patterns
type CustomRiskPattern struct {
	Pattern     string
	Level       RiskLevel
	Description string
	Category    string
}

// initializeDefaultPatterns sets up the default risk detection patterns
func (p *PatternBasedRiskAnalyzer) initializeDefaultPatterns() {
	// High-risk file system paths
	highRiskPaths := []string{
		`/etc/passwd`,
		`/etc/shadow`,
		`\.ssh/id_rsa`,
		`\.ssh/.*_rsa`,
		`/var/log/auth\.log`,
		`/var/log/secure`,
		`/root/`,
		`/etc/sudoers`,
		`/proc/.*`,
		`/sys/.*`,
		`/dev/.*`,
		`\.pem$`,
		`\.key$`,
		`private.*key`,
		`BEGIN.*PRIVATE.*KEY`,
	}

	// Medium-risk patterns
	mediumRiskPaths := []string{
		`\.env$`,
		`\.env\..*`,
		`config\.json$`,
		`database\.json$`,
		`/var/log/.*`,
		`/tmp/.*`,
		`SELECT.*FROM.*users`,
		`DELETE.*FROM`,
		`DROP.*TABLE`,
		`INSERT.*INTO.*users`,
		`UPDATE.*users.*SET`,
		`admin.*=.*1`,
		`password.*=`,
		`token.*=`,
		`api.*key`,
		`secret.*=`,
		`credential`,
		`auth.*token`,
	}

	// Initialize high-risk patterns
	for _, pattern := range highRiskPaths {
		if rp, err := NewRiskPattern(pattern, RiskLevelHigh, "High-risk file system access"); err == nil {
			p.highRiskPatterns = append(p.highRiskPatterns, rp)
		}
	}

	// Initialize medium-risk patterns
	for _, pattern := range mediumRiskPaths {
		if rp, err := NewRiskPattern(pattern, RiskLevelMedium, "Medium-risk data access"); err == nil {
			p.mediumRiskPatterns = append(p.mediumRiskPatterns, rp)
		}
	}

	// Initialize method risk mapping
	p.initializeMethodRiskMap()
}

// initializeMethodRiskMap sets up the method-based risk assessment
func (p *PatternBasedRiskAnalyzer) initializeMethodRiskMap() {
	// High-risk methods (direct system access)
	p.methodRiskMap["resources/read"] = RiskLevelHigh
	p.methodRiskMap["tools/call"] = RiskLevelHigh
	p.methodRiskMap["filesystem/write"] = RiskLevelHigh
	p.methodRiskMap["filesystem/delete"] = RiskLevelHigh
	p.methodRiskMap["process/execute"] = RiskLevelHigh
	p.methodRiskMap["shell/execute"] = RiskLevelHigh

	// Medium-risk methods (potential data access)
	p.methodRiskMap["prompts/get"] = RiskLevelMedium
	p.methodRiskMap["resources/write"] = RiskLevelMedium
	p.methodRiskMap["tools/execute"] = RiskLevelMedium
	p.methodRiskMap["database/query"] = RiskLevelMedium
	p.methodRiskMap["api/call"] = RiskLevelMedium
	p.methodRiskMap["filesystem/read"] = RiskLevelMedium

	// Low-risk methods (basic operations)
	p.methodRiskMap["ping"] = RiskLevelLow
	p.methodRiskMap["initialize"] = RiskLevelLow
	p.methodRiskMap["completion/complete"] = RiskLevelLow
	p.methodRiskMap["logging/log"] = RiskLevelLow
}

// AnalyzeEvent performs comprehensive risk analysis on an event
func (p *PatternBasedRiskAnalyzer) AnalyzeEvent(evt *event.Event) (event.RiskScore, error) {
	if evt == nil {
		return event.RiskScore{}, fmt.Errorf("event cannot be nil")
	}

	// Get risk levels from different analysis methods
	methodRisk := p.AnalyzeMethod(evt.Method().Value())
	contentRisk := p.AnalyzeContent(evt.Payload())
	sizeRisk := p.AnalyzePayloadSize(evt.Size())

	// Determine the highest risk level
	finalRisk := p.combineRiskLevels(methodRisk, contentRisk, sizeRisk)

	// Convert to risk score
	score := p.riskLevelToScore(finalRisk)

	return event.NewRiskScore(score)
}

// AnalyzeContent examines payload content for risk indicators
func (p *PatternBasedRiskAnalyzer) AnalyzeContent(content []byte) RiskLevel {
	contentStr := string(content)

	// Check for high-risk patterns first
	for _, pattern := range p.highRiskPatterns {
		if pattern.Matches(contentStr) {
			return RiskLevelHigh
		}
	}

	// Check for medium-risk patterns
	for _, pattern := range p.mediumRiskPatterns {
		if pattern.Matches(contentStr) {
			return RiskLevelMedium
		}
	}

	// Check for specific high-risk JSON structures
	if p.containsHighRiskJSON(contentStr) {
		return RiskLevelHigh
	}

	// Check for sensitive data patterns
	if p.containsSensitiveData(contentStr) {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// AnalyzeMethod returns the risk level for a specific method
func (p *PatternBasedRiskAnalyzer) AnalyzeMethod(method string) RiskLevel {
	// Handle empty method
	if method == "" {
		return RiskLevelLow
	}

	// Check exact method match
	if level, exists := p.methodRiskMap[method]; exists {
		return level
	}

	// Check pattern-based method matching
	methodLower := strings.ToLower(method)

	// High-risk patterns
	if p.matchesHighRiskMethod(methodLower) {
		return RiskLevelHigh
	}

	// Medium-risk patterns
	if p.matchesMediumRiskMethod(methodLower) {
		return RiskLevelMedium
	}

	// List operations are generally low risk
	if strings.HasSuffix(method, "/list") {
		return RiskLevelLow
	}

	// Default to low risk for unknown methods
	return RiskLevelLow
}

// AnalyzePayloadSize assesses risk based on payload size
func (p *PatternBasedRiskAnalyzer) AnalyzePayloadSize(size int) RiskLevel {
	// If no size limit configured, size is not a risk factor
	if p.payloadSizeLimit <= 0 {
		return RiskLevelLow
	}

	// Large payloads might indicate data exfiltration
	if size > p.payloadSizeLimit {
		return RiskLevelMedium
	}

	// Very large payloads are high risk
	if size > p.payloadSizeLimit*10 {
		return RiskLevelHigh
	}

	return RiskLevelLow
}

// combineRiskLevels determines the highest risk level among multiple assessments
func (p *PatternBasedRiskAnalyzer) combineRiskLevels(levels ...RiskLevel) RiskLevel {
	highest := RiskLevelLow

	for _, level := range levels {
		if level == RiskLevelHigh {
			return RiskLevelHigh
		}
		if level == RiskLevelMedium && highest == RiskLevelLow {
			highest = RiskLevelMedium
		}
	}

	return highest
}

// riskLevelToScore converts a risk level to a numerical score
func (p *PatternBasedRiskAnalyzer) riskLevelToScore(level RiskLevel) int {
	switch level {
	case RiskLevelHigh:
		return 75
	case RiskLevelMedium:
		return 35
	default:
		return 10
	}
}

// containsHighRiskJSON checks for dangerous operations in JSON parameters
func (p *PatternBasedRiskAnalyzer) containsHighRiskJSON(content string) bool {
	// Parse as JSON to check parameters
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonObj); err != nil {
		return false // Not valid JSON
	}

	// Check params for dangerous patterns
	if params, ok := jsonObj["params"].(map[string]interface{}); ok {
		// Check URI parameters for sensitive file access
		if uri, ok := params["uri"].(string); ok {
			if p.isHighRiskURI(uri) {
				return true
			}
		}

		// Check arguments for dangerous operations
		if args, ok := params["arguments"].(map[string]interface{}); ok {
			if p.containsHighRiskArguments(args) {
				return true
			}
		}
	}

	return false
}

// isHighRiskURI checks if a URI points to sensitive resources
func (p *PatternBasedRiskAnalyzer) isHighRiskURI(uri string) bool {
	lowerURI := strings.ToLower(uri)
	sensitivePatterns := []string{
		"passwd", "shadow", "id_rsa", "/etc/", "/root/", "/proc/", "/sys/",
		".ssh/", ".pem", ".key", "private", "credential", "secret",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerURI, pattern) {
			return true
		}
	}

	return false
}

// containsHighRiskArguments checks arguments for dangerous operations
func (p *PatternBasedRiskAnalyzer) containsHighRiskArguments(args map[string]interface{}) bool {
	for key, value := range args {
		if str, ok := value.(string); ok {
			lowerKey := strings.ToLower(key)
			lowerValue := strings.ToLower(str)

			// Check for dangerous SQL operations
			if lowerKey == "query" || lowerKey == "sql" {
				dangerousSQL := []string{
					"select * from users", "admin=1", "delete from",
					"drop table", "union select", "information_schema",
					"show tables", "describe ", "alter table",
				}

				for _, dangerous := range dangerousSQL {
					if strings.Contains(lowerValue, dangerous) {
						return true
					}
				}
			}

			// Check for system commands
			if lowerKey == "command" || lowerKey == "cmd" {
				dangerousCommands := []string{
					"rm -rf", "sudo ", "chmod 777", "passwd", "su ",
					"cat /etc/", "dd if=", "mkfs", "format c:",
				}

				for _, dangerous := range dangerousCommands {
					if strings.Contains(lowerValue, dangerous) {
						return true
					}
				}
			}
		}
	}

	return false
}

// containsSensitiveData checks for patterns that might indicate sensitive data
func (p *PatternBasedRiskAnalyzer) containsSensitiveData(content string) bool {
	lowerContent := strings.ToLower(content)

	// Check for potential credentials or sensitive patterns
	sensitivePatterns := []string{
		"password", "token", "secret", "credential", "api_key",
		"access_token", "private_key", "session_id", "auth_token",
		"bearer ", "basic ", "x-api-key", "authorization:",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	return false
}

// matchesHighRiskMethod checks if a method matches high-risk patterns
func (p *PatternBasedRiskAnalyzer) matchesHighRiskMethod(method string) bool {
	highRiskPatterns := []string{
		"execute", "shell", "system", "eval", "delete", "remove",
		"kill", "terminate", "format", "wipe", "destroy",
	}

	for _, pattern := range highRiskPatterns {
		if strings.Contains(method, pattern) {
			return true
		}
	}

	return false
}

// matchesMediumRiskMethod checks if a method matches medium-risk patterns
func (p *PatternBasedRiskAnalyzer) matchesMediumRiskMethod(method string) bool {
	mediumRiskPatterns := []string{
		"write", "create", "update", "modify", "change", "set",
		"insert", "query", "search", "access", "read", "get",
	}

	for _, pattern := range mediumRiskPatterns {
		if strings.Contains(method, pattern) {
			return true
		}
	}

	return false
}

// AddCustomPattern allows adding custom risk patterns
func (p *PatternBasedRiskAnalyzer) AddCustomPattern(pattern CustomRiskPattern) error {
	riskPattern, err := NewRiskPattern(pattern.Pattern, pattern.Level, pattern.Description)
	if err != nil {
		return err
	}

	switch pattern.Level {
	case RiskLevelHigh:
		p.highRiskPatterns = append(p.highRiskPatterns, riskPattern)
	case RiskLevelMedium:
		p.mediumRiskPatterns = append(p.mediumRiskPatterns, riskPattern)
	case RiskLevelLow:
		p.lowRiskPatterns = append(p.lowRiskPatterns, riskPattern)
	}

	return nil
}

// GetRiskStatistics returns statistics about the risk analyzer
func (p *PatternBasedRiskAnalyzer) GetRiskStatistics() RiskStatistics {
	return RiskStatistics{
		HighRiskPatterns:   len(p.highRiskPatterns),
		MediumRiskPatterns: len(p.mediumRiskPatterns),
		LowRiskPatterns:    len(p.lowRiskPatterns),
		MethodRiskMappings: len(p.methodRiskMap),
	}
}

// RiskStatistics provides information about the risk analyzer configuration
type RiskStatistics struct {
	HighRiskPatterns   int
	MediumRiskPatterns int
	LowRiskPatterns    int
	MethodRiskMappings int
}
