package plugins

// This file now serves as a bridge to the public SDK types
// All shared types have been moved to github.com/kilometers-ai/kilometers-plugins-sdk

import "github.com/kilometers-ai/kilometers-plugins-sdk"

// Type aliases for compatibility with existing code
type SDKPlugin = kmsdk.Plugin
type SDKPluginInfo = kmsdk.PluginInfo
type SDKConfig = kmsdk.Config
type SDKDirection = kmsdk.Direction
type SDKStreamEvent = kmsdk.StreamEvent

// Direction constants for compatibility
const (
	SDKDirectionInbound  = kmsdk.DirectionInbound
	SDKDirectionOutbound = kmsdk.DirectionOutbound
)
