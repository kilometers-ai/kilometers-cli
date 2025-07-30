package logging

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// ConsoleLogger implements the MessageHandler interface for console output
type ConsoleLogger struct {
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

// HandleMessage processes an intercepted JSON-RPC message
func (l *ConsoleLogger) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Silent - no logging to avoid interfering with MCP communication
	return nil
}

// HandleError processes an error that occurred during message handling
func (l *ConsoleLogger) HandleError(ctx context.Context, err error) {
	// Silent - no logging to avoid interfering with MCP communication
}

// HandleStreamEvent processes stream lifecycle events
func (l *ConsoleLogger) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	// Silent - no logging to avoid interfering with MCP communication
}

// LogMessage logs a parsed JSON-RPC message
func (l *ConsoleLogger) LogMessage(message *domain.JSONRPCMessage) {
	// Silent - no logging to avoid interfering with MCP communication
}

// LogWarning logs a warning message  
func (l *ConsoleLogger) LogWarning(message string) {
	// Silent - no logging to avoid interfering with MCP communication
}

// Note: All logging methods have been removed to ensure complete silence
// and avoid interference with MCP JSON-RPC communication
