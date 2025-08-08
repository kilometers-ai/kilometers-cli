package ports

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// Plugin represents a feature plugin that can be enabled/disabled
type Plugin interface {
	// Metadata
	Name() string
	RequiredFeature() string
	RequiredTier() domain.SubscriptionTier

	// Lifecycle
	Initialize(ctx context.Context, deps PluginDependencies) error
	Shutdown(ctx context.Context) error

	// Message handling - replaces the old MessageHandler interface
	HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error
	HandleError(ctx context.Context, err error)
	HandleStreamEvent(ctx context.Context, event StreamEvent)
}

// PluginDependencies provides access to core services
type PluginDependencies struct {
	Config      *domain.UnifiedConfig
	AuthManager AuthenticationManager
	APIClient   APIClient
}

// PluginManager manages plugin lifecycle and access
type PluginManager interface {
	// Registration
	RegisterPlugin(plugin Plugin) error

	// Feature management
	RefreshFeatures(ctx context.Context) error
	GetEnabledPlugins() []Plugin
	IsPluginEnabled(name string) bool

	// Lifecycle
	InitializePlugins(ctx context.Context) error
	ShutdownPlugins(ctx context.Context) error

	// Get a composite message handler for all enabled plugins
	GetMessageHandler() MessageHandler
}

// AuthenticationManager handles feature validation
type AuthenticationManager interface {
	GetAPIKey() string
	GetSubscriptionTier() domain.SubscriptionTier
	GetEnabledFeatures() []string
	IsFeatureEnabled(feature string) bool
	RefreshSubscription(ctx context.Context) error
}

// APIClient interface for making API calls
type APIClient interface {
	GetUserFeatures(ctx context.Context) (*UserFeaturesResponse, error)
	SendBatchEvents(ctx context.Context, batch interface{}) error
}

// UserFeaturesResponse from the API
type UserFeaturesResponse struct {
	Tier      domain.SubscriptionTier `json:"tier"`
	Features  []string                `json:"features"`
	ExpiresAt *string                 `json:"expires_at,omitempty"`
}
