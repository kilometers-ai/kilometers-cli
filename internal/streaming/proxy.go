package streaming

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/jsonrpc"
	processlib "github.com/kilometers-ai/kilometers-cli/internal/process"
)

// MessageHandler defines the interface for handling intercepted messages
type MessageHandler interface {
	// HandleMessage processes an intercepted JSON-RPC message
	HandleMessage(ctx context.Context, data []byte, direction jsonrpc.Direction) error

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

// StreamProxy handles bidirectional communication between client and server
type StreamProxy struct {
	process       processlib.Process
	correlationID string
	config        config.MonitorConfig
	messageLogger MessageHandler

	// Synchronization
	mu     sync.RWMutex
	active bool
	done   chan struct{}
}

// NewStreamProxy creates a new stream proxy
func NewStreamProxy(
	process processlib.Process,
	correlationID string,
	config config.MonitorConfig,
	messageLogger MessageHandler,
) *StreamProxy {
	return &StreamProxy{
		process:       process,
		correlationID: correlationID,
		config:        config,
		messageLogger: messageLogger,
		done:          make(chan struct{}),
	}
}

// Start begins proxying data between client and server
func (p *StreamProxy) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return fmt.Errorf("proxy already active")
	}
	p.active = true
	p.mu.Unlock()

	// Create a wait group to coordinate goroutines
	var wg sync.WaitGroup

	// Handle stdin (client -> server)
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.handleStdinProxy(ctx)
	}()

	// Handle stdout (server -> client)
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.handleStdoutProxy(ctx)
	}()

	// Handle stderr (server errors)
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.handleStderrProxy(ctx)
	}()

	// Wait for context cancellation or completion
	go func() {
		wg.Wait()
		close(p.done)
	}()

	// Wait for either context cancellation or all goroutines to complete
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return nil
	}
}

// Stop gracefully stops the proxy
func (p *StreamProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return nil
	}

	p.active = false
	return nil
}

// handleStdinProxy forwards data from client stdin to server stdin
func (p *StreamProxy) handleStdinProxy(ctx context.Context) {
	defer p.process.Stdin().Close()

	// Create a scanner with large buffer to handle big JSON-RPC messages
	scanner := bufio.NewScanner(os.Stdin)
	bufferSize := p.config.BufferSize
	scanner.Buffer(make([]byte, 0, 64*1024), bufferSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Log the message going to the server
		if p.messageLogger != nil {
			if err := p.handleMessage(ctx, line, jsonrpc.DirectionInbound); err != nil {
				p.messageLogger.HandleError(ctx, fmt.Errorf("failed to handle inbound message: %w", err))
			}
		}

		// Forward to server
		if _, err := p.process.Stdin().Write(append(line, '\n')); err != nil {
			if p.messageLogger != nil {
				p.messageLogger.HandleError(ctx, fmt.Errorf("failed to write to server stdin: %w", err))
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		if p.messageLogger != nil {
			p.messageLogger.HandleError(ctx, fmt.Errorf("stdin scanner error: %w", err))
		}
	}
}

// handleStdoutProxy forwards data from server stdout to client stdout
func (p *StreamProxy) handleStdoutProxy(ctx context.Context) {
	// Create a scanner with large buffer for server responses
	scanner := bufio.NewScanner(p.process.Stdout())
	bufferSize := p.config.BufferSize
	scanner.Buffer(make([]byte, 0, 64*1024), bufferSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Log the message coming from the server
		if p.messageLogger != nil {
			if err := p.handleMessage(ctx, line, jsonrpc.DirectionOutbound); err != nil {
				p.messageLogger.HandleError(ctx, fmt.Errorf("failed to handle outbound message: %w", err))
			}
		}

		// Forward to client
		if _, err := os.Stdout.Write(append(line, '\n')); err != nil {
			if p.messageLogger != nil {
				p.messageLogger.HandleError(ctx, fmt.Errorf("failed to write to client stdout: %w", err))
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		if p.messageLogger != nil {
			p.messageLogger.HandleError(ctx, fmt.Errorf("stdout scanner error: %w", err))
		}
	}
}

// handleStderrProxy forwards server stderr to client stderr
func (p *StreamProxy) handleStderrProxy(ctx context.Context) {
	// Simply copy stderr without parsing (not JSON-RPC)
	if _, err := io.Copy(os.Stderr, p.process.Stderr()); err != nil {
		if p.messageLogger != nil {
			p.messageLogger.HandleError(ctx, fmt.Errorf("stderr copy error: %w", err))
		}
	}
}

// handleMessage processes a captured message
func (p *StreamProxy) handleMessage(ctx context.Context, data []byte, direction jsonrpc.Direction) error {
	// Try to parse as JSON-RPC message to validate and extract metadata
	_, err := jsonrpc.NewJSONRPCMessageFromRaw(data, direction, p.correlationID)
	if err != nil {
		// Not a valid JSON-RPC message, ignore parsing but still log raw data
		return nil
	}

	// Log the message (this will handle both console output and API events)
	if p.messageLogger != nil {
		return p.messageLogger.HandleMessage(ctx, data, direction)
	}

	return nil
}
