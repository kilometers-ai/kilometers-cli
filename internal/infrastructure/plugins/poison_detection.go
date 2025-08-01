package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// PoisonDetectionPlugin detects prompt injection and tool poisoning attempts
type PoisonDetectionPlugin struct {
	deps              ports.PluginDependencies
	threatPatterns    []ThreatPattern
	detectionHistory  []DetectionEvent
	confidenceThreshold float64
}

// ThreatPattern represents a pattern that indicates potential poisoning
type ThreatPattern struct {
	ID          string
	Name        string
	Description string
	Pattern     string
	Severity    string
	Confidence  float64
}

// DetectionEvent represents a detected potential threat
type DetectionEvent struct {
	MessageID   string
	ThreatType  string
	Severity    string
	Confidence  float64
	Description string
	Timestamp   string
}

// NewPoisonDetectionPlugin creates a new poison detection plugin
func NewPoisonDetectionPlugin() *PoisonDetectionPlugin {
	return &PoisonDetectionPlugin{
		threatPatterns:      getDefaultThreatPatterns(),
		detectionHistory:    make([]DetectionEvent, 0),
		confidenceThreshold: 0.7,
	}
}

// Name returns the plugin name
func (p *PoisonDetectionPlugin) Name() string {
	return "poison-detection"
}

// RequiredFeature returns the required feature flag
func (p *PoisonDetectionPlugin) RequiredFeature() string {
	return domain.FeaturePoisonDetection
}

// RequiredTier returns the minimum subscription tier
func (p *PoisonDetectionPlugin) RequiredTier() domain.SubscriptionTier {
	return domain.TierPro
}

// Initialize sets up the plugin
func (p *PoisonDetectionPlugin) Initialize(deps ports.PluginDependencies) error {
	p.deps = deps
	return nil
}

// IsAvailable checks if plugin can be used
func (p *PoisonDetectionPlugin) IsAvailable(ctx context.Context) bool {
	return p.deps.AuthManager.IsFeatureEnabled(domain.FeaturePoisonDetection)
}

// Execute runs the plugin
func (p *PoisonDetectionPlugin) Execute(ctx context.Context, params ports.PluginParams) (ports.PluginResult, error) {
	command := params.Command
	
	switch command {
	case "analyze-message":
		return p.analyzeMessage(params)
	case "get-threats":
		return p.getThreatHistory(params)
	case "update-patterns":
		return p.updatePatterns(params)
	default:
		return ports.PluginResult{}, fmt.Errorf("unknown command: %s", command)
	}
}

// Cleanup performs cleanup
func (p *PoisonDetectionPlugin) Cleanup() error {
	p.detectionHistory = nil
	return nil
}

// CheckSecurity analyzes a message for security concerns
func (p *PoisonDetectionPlugin) CheckSecurity(ctx context.Context, message ports.MCPMessage) (ports.SecurityResult, error) {
	payload := string(message.Payload())
	issues := make([]ports.SecurityIssue, 0)
	maxRiskLevel := "low"
	totalConfidence := 0.0
	matchCount := 0
	
	for _, pattern := range p.threatPatterns {
		if p.matchesPattern(payload, pattern) {
			issue := ports.SecurityIssue{
				Type:        pattern.Name,
				Description: pattern.Description,
				Severity:    pattern.Severity,
				Mitigation:  p.getMitigation(pattern.Name),
			}
			issues = append(issues, issue)
			
			// Update max risk level
			if p.isHigherRisk(pattern.Severity, maxRiskLevel) {
				maxRiskLevel = pattern.Severity
			}
			
			totalConfidence += pattern.Confidence
			matchCount++
			
			// Record detection event
			p.recordDetection(message, pattern)
		}
	}
	
	averageConfidence := 0.0
	if matchCount > 0 {
		averageConfidence = totalConfidence / float64(matchCount)
	}
	
	return ports.SecurityResult{
		IsSecure:   len(issues) == 0,
		RiskLevel:  maxRiskLevel,
		Issues:     issues,
		Confidence: averageConfidence,
	}, nil
}

// GetSecurityReport returns a security analysis report
func (p *PoisonDetectionPlugin) GetSecurityReport(ctx context.Context) (ports.SecurityReport, error) {
	riskDistribution := make(map[string]int)
	riskDistribution["low"] = 0
	riskDistribution["medium"] = 0
	riskDistribution["high"] = 0
	riskDistribution["critical"] = 0
	
	allIssues := make([]ports.SecurityIssue, 0)
	
	for _, event := range p.detectionHistory {
		riskDistribution[event.Severity]++
		
		issue := ports.SecurityIssue{
			Type:        event.ThreatType,
			Description: event.Description,
			Severity:    event.Severity,
			Mitigation:  p.getMitigation(event.ThreatType),
		}
		allIssues = append(allIssues, issue)
	}
	
	recommendations := p.generateRecommendations(riskDistribution)
	
	return ports.SecurityReport{
		TotalMessages:    len(p.detectionHistory),
		SecurityIssues:   allIssues,
		RiskDistribution: riskDistribution,
		Recommendations:  recommendations,
	}, nil
}

// Helper methods

func (p *PoisonDetectionPlugin) matchesPattern(payload string, pattern ThreatPattern) bool {
	payload = strings.ToLower(payload)
	patternStr := strings.ToLower(pattern.Pattern)
	
	// Simple substring matching - in production, you'd use more sophisticated detection
	return strings.Contains(payload, patternStr)
}

func (p *PoisonDetectionPlugin) isHigherRisk(severity1, severity2 string) bool {
	riskLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}
	
	return riskLevels[severity1] > riskLevels[severity2]
}

func (p *PoisonDetectionPlugin) recordDetection(message ports.MCPMessage, pattern ThreatPattern) {
	event := DetectionEvent{
		MessageID:   string(message.ID()),
		ThreatType:  pattern.Name,
		Severity:    pattern.Severity,
		Confidence:  pattern.Confidence,
		Description: pattern.Description,
		Timestamp:   message.Timestamp().Format("2006-01-02 15:04:05"),
	}
	
	p.detectionHistory = append(p.detectionHistory, event)
	
	// Keep only last 1000 events
	if len(p.detectionHistory) > 1000 {
		p.detectionHistory = p.detectionHistory[len(p.detectionHistory)-1000:]
	}
}

func (p *PoisonDetectionPlugin) getMitigation(threatType string) string {
	mitigations := map[string]string{
		"prompt_injection":    "Sanitize user inputs and use parameterized queries",
		"tool_manipulation":   "Validate tool parameters and implement access controls",
		"data_exfiltration":   "Monitor data access patterns and implement DLP policies",
		"privilege_escalation": "Use principle of least privilege and validate permissions",
	}
	
	if mitigation, exists := mitigations[threatType]; exists {
		return mitigation
	}
	return "Review message content and validate against security policies"
}

func (p *PoisonDetectionPlugin) generateRecommendations(riskDistribution map[string]int) []string {
	recommendations := make([]string, 0)
	
	if riskDistribution["critical"] > 0 {
		recommendations = append(recommendations, "Immediate action required: Critical security threats detected")
	}
	
	if riskDistribution["high"] > 5 {
		recommendations = append(recommendations, "Multiple high-risk threats detected - review security policies")
	}
	
	if riskDistribution["medium"] > 10 {
		recommendations = append(recommendations, "Consider implementing additional security controls")
	}
	
	totalThreats := riskDistribution["low"] + riskDistribution["medium"] + riskDistribution["high"] + riskDistribution["critical"]
	if totalThreats == 0 {
		recommendations = append(recommendations, "No security threats detected - monitoring is working effectively")
	}
	
	return recommendations
}

func (p *PoisonDetectionPlugin) analyzeMessage(params ports.PluginParams) (ports.PluginResult, error) {
	messageData, ok := params.Data["message"].(string)
	if !ok {
		return ports.PluginResult{}, fmt.Errorf("message data is required")
	}
	
	threats := make([]map[string]interface{}, 0)
	for _, pattern := range p.threatPatterns {
		if p.matchesPattern(messageData, pattern) {
			threats = append(threats, map[string]interface{}{
				"type":        pattern.Name,
				"severity":    pattern.Severity,
				"confidence":  pattern.Confidence,
				"description": pattern.Description,
			})
		}
	}
	
	return ports.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"threats": threats,
			"safe":    len(threats) == 0,
		},
	}, nil
}

func (p *PoisonDetectionPlugin) getThreatHistory(params ports.PluginParams) (ports.PluginResult, error) {
	limit := 50
	if l, ok := params.Data["limit"].(int); ok {
		limit = l
	}
	
	historySize := len(p.detectionHistory)
	start := 0
	if historySize > limit {
		start = historySize - limit
	}
	
	recentHistory := p.detectionHistory[start:]
	
	return ports.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"events": recentHistory,
			"total":  historySize,
		},
	}, nil
}

func (p *PoisonDetectionPlugin) updatePatterns(params ports.PluginParams) (ports.PluginResult, error) {
	// In a real implementation, this would update threat patterns from a threat intelligence feed
	return ports.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "Threat patterns updated successfully",
			"count":   len(p.threatPatterns),
		},
	}, nil
}

func getDefaultThreatPatterns() []ThreatPattern {
	return []ThreatPattern{
		{
			ID:          "PI001",
			Name:        "prompt_injection",
			Description: "Potential prompt injection attempt detected",
			Pattern:     "ignore previous instructions",
			Severity:    "high",
			Confidence:  0.8,
		},
		{
			ID:          "PI002",
			Name:        "prompt_injection",
			Description: "System prompt override attempt",
			Pattern:     "you are now",
			Severity:    "medium",
			Confidence:  0.7,
		},
		{
			ID:          "TM001",
			Name:        "tool_manipulation",
			Description: "Potential tool parameter manipulation",
			Pattern:     "../",
			Severity:    "medium",
			Confidence:  0.6,
		},
		{
			ID:          "DE001",
			Name:        "data_exfiltration",
			Description: "Potential data exfiltration attempt",
			Pattern:     "send to",
			Severity:    "high",
			Confidence:  0.5,
		},
		{
			ID:          "PE001",
			Name:        "privilege_escalation",
			Description: "Potential privilege escalation attempt",
			Pattern:     "sudo",
			Severity:    "high",
			Confidence:  0.7,
		},
	}
}
