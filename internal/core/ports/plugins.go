package ports

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// MCPMessage is an alias for JSONRPCMessage for plugin compatibility
type MCPMessage = domain.JSONRPCMessage

// Plugin represents a feature plugin that can be loaded and executed
type Plugin interface {
	// Name returns the plugin name
	Name() string
	
	// RequiredFeature returns the feature flag required to use this plugin
	RequiredFeature() string
	
	// RequiredTier returns the minimum subscription tier required
	RequiredTier() domain.SubscriptionTier
	
	// Initialize sets up the plugin with dependencies
	Initialize(deps PluginDependencies) error
	
	// IsAvailable checks if the plugin can be used
	IsAvailable(ctx context.Context) bool
	
	// Execute runs the plugin functionality
	Execute(ctx context.Context, params PluginParams) (PluginResult, error)
	
	// Cleanup performs any necessary cleanup
	Cleanup() error
}

// PluginDependencies provides common dependencies to plugins
type PluginDependencies struct {
	AuthManager   *domain.AuthenticationManager
	MessageLogger MessageHandler
	Config        domain.Config
}

// PluginParams contains parameters passed to plugin execution
type PluginParams struct {
	Command string
	Args    []string
	Data    map[string]interface{}
}

// PluginResult contains the result of plugin execution
type PluginResult struct {
	Success bool
	Data    map[string]interface{}
	Error   error
}

// PluginManager manages the lifecycle of plugins
type PluginManager interface {
	// LoadPlugins discovers and loads available plugins
	LoadPlugins(ctx context.Context) error
	
	// GetPlugin retrieves a plugin by name
	GetPlugin(name string) (Plugin, error)
	
	// GetAvailablePlugins returns plugins available for current subscription
	GetAvailablePlugins(ctx context.Context) []Plugin
	
	// ExecutePlugin runs a plugin with given parameters
	ExecutePlugin(ctx context.Context, name string, params PluginParams) (PluginResult, error)
	
	// ListFeatures returns all available features for current subscription
	ListFeatures(ctx context.Context) []string
}

// FilterPlugin represents plugins that can filter and modify MCP messages
type FilterPlugin interface {
	Plugin
	
	// FilterMessage processes an MCP message and returns modified version
	FilterMessage(ctx context.Context, message MCPMessage) (MCPMessage, error)
	
	// ShouldFilter determines if this message should be processed by this filter
	ShouldFilter(ctx context.Context, message MCPMessage) bool
}

// AnalyticsPlugin represents plugins that can analyze MCP communications
type AnalyticsPlugin interface {
	Plugin
	
	// AnalyzeMessage processes a message for analytics
	AnalyzeMessage(ctx context.Context, message MCPMessage) error
	
	// GetAnalytics returns analytics data
	GetAnalytics(ctx context.Context) (map[string]interface{}, error)
	
	// ResetAnalytics clears analytics data
	ResetAnalytics(ctx context.Context) error
}

// SecurityPlugin represents plugins that can detect security issues
type SecurityPlugin interface {
	Plugin
	
	// CheckSecurity analyzes a message for security concerns
	CheckSecurity(ctx context.Context, message MCPMessage) (SecurityResult, error)
	
	// GetSecurityReport returns a security analysis report
	GetSecurityReport(ctx context.Context) (SecurityReport, error)
}

// SecurityResult contains the result of security analysis
type SecurityResult struct {
	IsSecure     bool
	RiskLevel    string // low, medium, high, critical
	Issues       []SecurityIssue
	Confidence   float64
}

// SecurityIssue represents a detected security concern
type SecurityIssue struct {
	Type        string
	Description string
	Severity    string
	Mitigation  string
}

// SecurityReport contains overall security analysis
type SecurityReport struct {
	TotalMessages    int
	SecurityIssues   []SecurityIssue
	RiskDistribution map[string]int
	Recommendations  []string
}
