package streaming

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/streaming"
	"github.com/kilometers-ai/kilometers-cli/internal/jsonrpc"
)

// MessageHandler defines the interface for handling intercepted messages
type MessageHandler interface {
	// HandleMessage processes an intercepted JSON-RPC message
	HandleMessage(ctx context.Context, data []byte, direction jsonrpc.Direction) error

	// HandleError processes an error that occurred during message handling
	HandleError(ctx context.Context, err error)

	// HandleStreamEvent processes stream lifecycle events
	HandleStreamEvent(ctx context.Context, event streaming.StreamEvent)
}

// Proxy defines the contract for a stream proxy.
type Proxy interface {
	Start(ctx context.Context) error
	Stop() error
}
