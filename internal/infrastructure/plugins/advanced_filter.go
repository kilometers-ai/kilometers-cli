package plugins

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// AdvancedFilterPlugin provides advanced filtering capabilities for Pro+ tiers
type AdvancedFilterPlugin struct {
	deps     ports.PluginDependencies
	patterns []*regexp.Regexp
	rules    []FilterRule
}

// FilterRule represents a complex filtering rule
type FilterRule struct {
	Name        string
	Pattern     string
	Action      FilterAction
	Condition   FilterCondition
	Enabled     bool
}

// FilterAction defines what to do when a rule matches
type FilterAction string

const (
	ActionAllow  FilterAction = "allow"
	ActionBlock  FilterAction = "block"
	ActionRedact FilterAction = "redact"
	ActionWarn   FilterAction = "warn"
)

// FilterCondition defines when a rule should apply
type FilterCondition struct {
	MessageType string
	Method      string
	Direction   string
	MinSize     int
	MaxSize     int
}

// NewAdvancedFilterPlugin creates a new advanced filter plugin
func NewAdvancedFilterPlugin() *AdvancedFilterPlugin {
	return &AdvancedFilterPlugin{
		patterns: make([]*regexp.Regexp, 0),
		rules:    getDefaultFilterRules(),
	}
}

// Name returns the plugin name
func (p *AdvancedFilterPlugin) Name() string {
	return "advanced-filters"
}

// RequiredFeature returns the required feature flag
func (p *AdvancedFilterPlugin) RequiredFeature() string {
	return domain.FeatureAdvancedFilters
}

// RequiredTier returns the minimum subscription tier
func (p *AdvancedFilterPlugin) RequiredTier() domain.SubscriptionTier {
	return domain.TierPro
}

// Initialize sets up the plugin
func (p *AdvancedFilterPlugin) Initialize(deps ports.PluginDependencies) error {
	p.deps = deps
	
	// Compile regex patterns
	for _, rule := range p.rules {
		if rule.Enabled && rule.Pattern != "" {
			compiled, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile pattern '%s': %w", rule.Pattern, err)
			}
			p.patterns = append(p.patterns, compiled)
		}
	}
	
	return nil
}

// IsAvailable checks if plugin can be used
func (p *AdvancedFilterPlugin) IsAvailable(ctx context.Context) bool {
	return p.deps.AuthManager.IsFeatureEnabled(domain.FeatureAdvancedFilters)
}

// Execute runs the plugin
func (p *AdvancedFilterPlugin) Execute(ctx context.Context, params ports.PluginParams) (ports.PluginResult, error) {
	command := params.Command
	
	switch command {
	case "add-rule":
		return p.addRule(params)
	case "remove-rule":
		return p.removeRule(params)
	case "list-rules":
		return p.listRules(params)
	default:
		return ports.PluginResult{}, fmt.Errorf("unknown command: %s", command)
	}
}

// Cleanup performs cleanup
func (p *AdvancedFilterPlugin) Cleanup() error {
	p.patterns = nil
	p.rules = nil
	return nil
}

// FilterMessage processes an MCP message
func (p *AdvancedFilterPlugin) FilterMessage(ctx context.Context, message ports.MCPMessage) (ports.MCPMessage, error) {
	for _, rule := range p.rules {
		if !rule.Enabled {
			continue
		}
		
		if p.ruleMatches(rule, message) {
			switch rule.Action {
			case ActionBlock:
				return nil, fmt.Errorf("message blocked by rule: %s", rule.Name)
			case ActionRedact:
				return p.redactMessage(message), nil
			case ActionWarn:
				p.deps.MessageLogger.LogWarning(fmt.Sprintf("Warning: Message matched rule '%s'", rule.Name))
			}
		}
	}
	
	return message, nil
}

// ShouldFilter determines if message should be processed
func (p *AdvancedFilterPlugin) ShouldFilter(ctx context.Context, message ports.MCPMessage) bool {
	// Always process messages for Pro+ users
	return p.deps.AuthManager.IsFeatureEnabled(domain.FeatureAdvancedFilters)
}

// Helper methods

func (p *AdvancedFilterPlugin) ruleMatches(rule FilterRule, message ports.MCPMessage) bool {
	// Check message type
	if rule.Condition.MessageType != "" && string(message.Type()) != rule.Condition.MessageType {
		return false
	}
	
	// Check method
	if rule.Condition.Method != "" && message.Method() != rule.Condition.Method {
		return false
	}
	
	// Check direction
	if rule.Condition.Direction != "" && string(message.Direction()) != rule.Condition.Direction {
		return false
	}
	
	// Check size constraints
	size := message.Size()
	if rule.Condition.MinSize > 0 && size < rule.Condition.MinSize {
		return false
	}
	if rule.Condition.MaxSize > 0 && size > rule.Condition.MaxSize {
		return false
	}
	
	// Check pattern match
	if rule.Pattern != "" {
		for _, pattern := range p.patterns {
			if pattern.MatchString(string(message.Payload())) {
				return true
			}
		}
	}
	
	return rule.Pattern == "" // Match if no pattern specified
}

func (p *AdvancedFilterPlugin) redactMessage(message ports.MCPMessage) ports.MCPMessage {
	// Create a redacted version of the message
	redacted := []byte(`{"jsonrpc":"2.0","method":"redacted","params":{"reason":"filtered"}}`)
	
	// This would need to properly reconstruct the message with redacted content
	// For demo purposes, we'll return the original message
	return message
}

func (p *AdvancedFilterPlugin) addRule(params ports.PluginParams) (ports.PluginResult, error) {
	ruleName, ok := params.Data["name"].(string)
	if !ok {
		return ports.PluginResult{}, fmt.Errorf("rule name is required")
	}
	
	pattern, _ := params.Data["pattern"].(string)
	action, _ := params.Data["action"].(string)
	
	rule := FilterRule{
		Name:    ruleName,
		Pattern: pattern,
		Action:  FilterAction(action),
		Enabled: true,
	}
	
	p.rules = append(p.rules, rule)
	
	return ports.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"message": fmt.Sprintf("Added rule '%s'", ruleName),
		},
	}, nil
}

func (p *AdvancedFilterPlugin) removeRule(params ports.PluginParams) (ports.PluginResult, error) {
	ruleName, ok := params.Data["name"].(string)
	if !ok {
		return ports.PluginResult{}, fmt.Errorf("rule name is required")
	}
	
	for i, rule := range p.rules {
		if rule.Name == ruleName {
			p.rules = append(p.rules[:i], p.rules[i+1:]...)
			return ports.PluginResult{
				Success: true,
				Data: map[string]interface{}{
					"message": fmt.Sprintf("Removed rule '%s'", ruleName),
				},
			}, nil
		}
	}
	
	return ports.PluginResult{}, fmt.Errorf("rule '%s' not found", ruleName)
}

func (p *AdvancedFilterPlugin) listRules(params ports.PluginParams) (ports.PluginResult, error) {
	ruleList := make([]map[string]interface{}, len(p.rules))
	for i, rule := range p.rules {
		ruleList[i] = map[string]interface{}{
			"name":    rule.Name,
			"pattern": rule.Pattern,
			"action":  rule.Action,
			"enabled": rule.Enabled,
		}
	}
	
	return ports.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"rules": ruleList,
		},
	}, nil
}

func getDefaultFilterRules() []FilterRule {
	return []FilterRule{
		{
			Name:    "block-sensitive-data",
			Pattern: `(?i)(password|secret|token|key)\s*[:=]\s*["']?[\w-]+["']?`,
			Action:  ActionRedact,
			Enabled: true,
			Condition: FilterCondition{
				MessageType: "request",
			},
		},
		{
			Name:    "warn-large-payloads",
			Pattern: "",
			Action:  ActionWarn,
			Enabled: true,
			Condition: FilterCondition{
				MinSize: 1024 * 1024, // 1MB
			},
		},
	}
}
