package domain

// Re-export domain types that plugins need
// This allows external modules to import domain types without accessing internal packages

import "github.com/kilometers-ai/kilometers-cli/internal/core/domain"

// Re-export commonly used domain types
type Direction = domain.Direction
type SubscriptionTier = domain.SubscriptionTier
type UnifiedConfig = domain.UnifiedConfig
type JSONRPCMessage = domain.JSONRPCMessage
type MessageType = domain.MessageType
type MessageID = domain.MessageID

// Re-export constants
const (
	DirectionInbound  = domain.DirectionInbound
	DirectionOutbound = domain.DirectionOutbound

	TierFree       = domain.TierFree
	TierPro        = domain.TierPro
	TierEnterprise = domain.TierEnterprise

	MessageTypeRequest      = domain.MessageTypeRequest
	MessageTypeResponse     = domain.MessageTypeResponse
	MessageTypeNotification = domain.MessageTypeNotification

	// Feature constants
	FeatureBasicMonitoring     = domain.FeatureBasicMonitoring
	FeatureConsoleLogging      = domain.FeatureConsoleLogging
	FeatureAPILogging          = domain.FeatureAPILogging
	FeatureAdvancedFilters     = domain.FeatureAdvancedFilters
	FeaturePoisonDetection     = domain.FeaturePoisonDetection
	FeatureMLAnalytics         = domain.FeatureMLAnalytics
	FeatureComplianceReporting = domain.FeatureComplianceReporting
	FeatureTeamCollaboration   = domain.FeatureTeamCollaboration
)

// Re-export helper functions
func NewJSONRPCMessageFromRaw(data []byte, direction Direction, correlationID string) (*JSONRPCMessage, error) {
	return domain.NewJSONRPCMessageFromRaw(data, direction, correlationID)
}
