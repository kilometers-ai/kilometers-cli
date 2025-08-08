package ports

import (
	"context"
	"io"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// ProcessExecutor defines the interface for executing MCP server processes
type ProcessExecutor interface {
	// Execute starts a new process and returns a Process handle
	Execute(ctx context.Context, cmd domain.Command) (Process, error)
}

// Process represents a running MCP server process
type Process interface {
	// PID returns the process ID
	PID() int

	// Stdin returns the stdin writer for sending data to the process
	Stdin() io.WriteCloser

	// Stdout returns the stdout reader for receiving data from the process
	Stdout() io.ReadCloser

	// Stderr returns the stderr reader for receiving error output
	Stderr() io.ReadCloser

	// Wait waits for the process to complete and returns the exit code
	Wait() error

	// Signal sends a signal to the process (for graceful shutdown)
	Signal(signal ProcessSignal) error

	// Kill forcefully terminates the process
	Kill() error

	// IsRunning returns true if the process is still running
	IsRunning() bool

	// ExitCode returns the exit code if the process has finished
	ExitCode() int
}

// ProcessSignal represents signals that can be sent to processes
type ProcessSignal int

const (
	SignalTerminate ProcessSignal = iota // SIGTERM
	SignalInterrupt                      // SIGINT
	SignalKill                           // SIGKILL
)

// StreamProxy defines the interface for proxying data between client and server
type StreamProxy interface {
	// Start begins proxying data between the streams
	Start(ctx context.Context) error

	// Stop gracefully stops the proxy
	Stop() error

	// AddMessageHandler adds a handler for intercepted messages
	AddMessageHandler(handler MessageHandler)
}

// MessageHandler defines the interface for handling intercepted messages
type MessageHandler interface {
	// HandleMessage processes an intercepted JSON-RPC message
	HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error

	// HandleError processes an error that occurred during message handling
	HandleError(ctx context.Context, err error)

	// HandleStreamEvent processes stream lifecycle events
	HandleStreamEvent(ctx context.Context, event StreamEvent)
}

// StreamEvent represents events in the stream lifecycle
type StreamEvent struct {
	Type      StreamEventType
	Message   string
	Timestamp int64
	Data      interface{}
}

// StreamEventType represents different types of stream events
type StreamEventType string

const (
	StreamEventConnected    StreamEventType = "connected"
	StreamEventDisconnected StreamEventType = "disconnected"
	StreamEventError        StreamEventType = "error"
	StreamEventDataSent     StreamEventType = "data_sent"
	StreamEventDataReceived StreamEventType = "data_received"
)
