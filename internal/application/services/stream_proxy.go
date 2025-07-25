package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// StreamProxy handles bidirectional communication between client and server
type StreamProxy struct {
	process       ports.Process
	session       *domain.MonitoringSession
	messageLogger ports.MessageHandler

	// Synchronization
	mu     sync.RWMutex
	active bool
	done   chan struct{}
}

// NewStreamProxy creates a new stream proxy
func NewStreamProxy(
	process ports.Process,
	session *domain.MonitoringSession,
	messageLogger ports.MessageHandler,
) *StreamProxy {
	return &StreamProxy{
		process:       process,
		session:       session,
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
	bufferSize := p.session.Config().BufferSize
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
			if err := p.handleMessage(ctx, line, domain.DirectionInbound); err != nil {
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
	bufferSize := p.session.Config().BufferSize
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
			if err := p.handleMessage(ctx, line, domain.DirectionOutbound); err != nil {
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
func (p *StreamProxy) handleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Try to parse as JSON-RPC message
	message, err := domain.NewJSONRPCMessageFromRaw(data, direction, p.session.ID())
	if err != nil {
		// Not a valid JSON-RPC message, ignore
		return nil
	}

	// Add message to session
	if err := p.session.AddMessage(*message); err != nil {
		return fmt.Errorf("failed to add message to session: %w", err)
	}

	// Log the message
	if p.messageLogger != nil {
		return p.messageLogger.HandleMessage(ctx, data, direction)
	}

	return nil
}
