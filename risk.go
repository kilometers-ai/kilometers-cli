package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Risk levels that match the API scoring system
const (
	RiskLevelLow    = 10
	RiskLevelMedium = 35
	RiskLevelHigh   = 75
)

// RiskDetector handles client-side risk analysis
type RiskDetector struct {
	highRiskPatterns   []*regexp.Regexp
	mediumRiskPatterns []*regexp.Regexp
	highRiskMethods    map[string]bool
	mediumRiskMethods  map[string]bool
}

// NewRiskDetector creates a new risk detector with predefined patterns
func NewRiskDetector() *RiskDetector {
	rd := &RiskDetector{
		highRiskMethods:   make(map[string]bool),
		mediumRiskMethods: make(map[string]bool),
	}

	// High-risk file system paths (from test script patterns)
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
	}

	// Medium-risk file system paths and database patterns
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
		`admin.*=.*1`,
		`password.*=`,
		`token.*=`,
	}

	// Compile patterns
	for _, pattern := range highRiskPaths {
		if regex, err := regexp.Compile(pattern); err == nil {
			rd.highRiskPatterns = append(rd.highRiskPatterns, regex)
		}
	}

	for _, pattern := range mediumRiskPaths {
		if regex, err := regexp.Compile(pattern); err == nil {
			rd.mediumRiskPatterns = append(rd.mediumRiskPatterns, regex)
		}
	}

	// High-risk methods (direct system access)
	rd.highRiskMethods["resources/read"] = true // when accessing sensitive files
	rd.highRiskMethods["tools/call"] = true     // when calling dangerous tools

	// Medium-risk methods (potential data access)
	rd.mediumRiskMethods["prompts/get"] = true     // complex prompts might expose data
	rd.mediumRiskMethods["resources/write"] = true // file modifications
	rd.mediumRiskMethods["tools/execute"] = true   // tool execution

	return rd
}

// AnalyzeEvent calculates the risk score for an MCP event
func (rd *RiskDetector) AnalyzeEvent(msg *MCPMessage, payload []byte) int {
	if msg == nil {
		return RiskLevelLow
	}

	baseScore := rd.getMethodRiskScore(msg.Method)
	contentScore := rd.analyzeContent(payload)
	sizeScore := rd.analyzeSizeRisk(len(payload))

	// Take the highest risk score among all factors
	finalScore := baseScore
	if contentScore > finalScore {
		finalScore = contentScore
	}
	if sizeScore > finalScore {
		finalScore = sizeScore
	}

	return finalScore
}

// getMethodRiskScore returns base risk score for MCP methods
func (rd *RiskDetector) getMethodRiskScore(method string) int {
	// Ping and basic operations are low risk
	if method == "ping" || method == "initialize" {
		return RiskLevelLow
	}

	// List operations are generally low risk
	if strings.HasSuffix(method, "/list") {
		return RiskLevelLow
	}

	// Check high-risk methods
	if rd.highRiskMethods[method] {
		return RiskLevelHigh
	}

	// Check medium-risk methods
	if rd.mediumRiskMethods[method] {
		return RiskLevelMedium
	}

	// Default to low risk for unknown methods
	return RiskLevelLow
}

// analyzeContent examines payload content for risk indicators
func (rd *RiskDetector) analyzeContent(payload []byte) int {
	content := string(payload)

	// Check for high-risk patterns
	for _, pattern := range rd.highRiskPatterns {
		if pattern.MatchString(content) {
			return RiskLevelHigh
		}
	}

	// Check for medium-risk patterns
	for _, pattern := range rd.mediumRiskPatterns {
		if pattern.MatchString(content) {
			return RiskLevelMedium
		}
	}

	// Check for specific high-risk JSON structures
	if rd.containsHighRiskJSON(content) {
		return RiskLevelHigh
	}

	return RiskLevelLow
}

// containsHighRiskJSON checks for dangerous operations in JSON parameters
func (rd *RiskDetector) containsHighRiskJSON(content string) bool {
	// Parse as JSON to check parameters
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonObj); err != nil {
		return false // Not valid JSON
	}

	// Check params for dangerous patterns
	if params, ok := jsonObj["params"].(map[string]interface{}); ok {
		// Check URI parameters for sensitive file access
		if uri, ok := params["uri"].(string); ok {
			lowerURI := strings.ToLower(uri)
			if strings.Contains(lowerURI, "passwd") ||
				strings.Contains(lowerURI, "shadow") ||
				strings.Contains(lowerURI, "id_rsa") ||
				strings.Contains(lowerURI, "/etc/") ||
				strings.Contains(lowerURI, "/root/") {
				return true
			}
		}

		// Check arguments for dangerous database queries
		if args, ok := params["arguments"].(map[string]interface{}); ok {
			if query, ok := args["query"].(string); ok {
				lowerQuery := strings.ToLower(query)
				if strings.Contains(lowerQuery, "select * from users") ||
					strings.Contains(lowerQuery, "admin=1") ||
					strings.Contains(lowerQuery, "delete from") ||
					strings.Contains(lowerQuery, "drop table") {
					return true
				}
			}
		}
	}

	return false
}

// analyzeSizeRisk assigns risk based on payload size
func (rd *RiskDetector) analyzeSizeRisk(size int) int {
	// Very large payloads might indicate data exfiltration
	if size > 100*1024 { // 100KB
		return RiskLevelHigh
	}

	// Medium-large payloads warrant attention
	if size > 10*1024 { // 10KB
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// ShouldCaptureEvent determines if an event should be captured based on risk and filtering rules
func (rd *RiskDetector) ShouldCaptureEvent(msg *MCPMessage, payload []byte, config *Config) bool {
	// Always capture if no filtering is enabled
	if !config.EnableRiskDetection && len(config.MethodWhitelist) == 0 && !config.ExcludePingMessages {
		return true
	}

	// Check ping exclusion first (most common filter)
	if config.ExcludePingMessages && msg != nil && msg.Method == "ping" {
		return false
	}

	// Check method whitelist
	if len(config.MethodWhitelist) > 0 && msg != nil {
		found := false
		for _, whitelistedMethod := range config.MethodWhitelist {
			if msg.Method == whitelistedMethod ||
				(strings.Contains(whitelistedMethod, "*") && rd.matchesPattern(msg.Method, whitelistedMethod)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check risk-based filtering
	if config.EnableRiskDetection {
		riskScore := rd.AnalyzeEvent(msg, payload)

		if config.HighRiskMethodsOnly && riskScore < RiskLevelHigh {
			return false
		}
	}

	// Check payload size limit
	if config.PayloadSizeLimit > 0 && len(payload) > config.PayloadSizeLimit {
		return false
	}

	return true
}

// matchesPattern provides simple wildcard matching for method names
func (rd *RiskDetector) matchesPattern(method, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return method == pattern
	}

	// Simple prefix/suffix matching
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *substring*
		substring := strings.Trim(pattern, "*")
		return strings.Contains(method, substring)
	} else if strings.HasPrefix(pattern, "*") {
		// *suffix
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(method, suffix)
	} else if strings.HasSuffix(pattern, "*") {
		// prefix*
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(method, prefix)
	}

	return method == pattern
}

// GetRiskLabel returns a human-readable risk level
func GetRiskLabel(score int) string {
	switch {
	case score >= RiskLevelHigh:
		return "HIGH"
	case score >= RiskLevelMedium:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
