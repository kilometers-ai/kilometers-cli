package logging

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	httpClient "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
)

const (
	defaultBatchSize     = 10
	defaultFlushInterval = 5 * time.Second
	cliVersion           = "1.2.0" // TODO: Extract from build variables
)

// ApiHandler wraps another MessageHandler and sends events to the kilometers-api
type ApiHandler struct {
	baseHandler   ports.MessageHandler
	apiClient     *httpClient.ApiClient
	correlationID string

	// Batching fields
	eventBuffer []httpClient.BatchEventDto
	bufferMutex sync.Mutex
	flushTimer  *time.Timer
	stopChan    chan struct{}
}

// NewApiHandler creates a new API handler that wraps another handler
func NewApiHandler(baseHandler ports.MessageHandler) *ApiHandler {
	handler := &ApiHandler{
		baseHandler: baseHandler,
		apiClient:   httpClient.NewApiClient(),
		eventBuffer: make([]httpClient.BatchEventDto, 0, defaultBatchSize),
		stopChan:    make(chan struct{}),
	}

	// Start the flush timer
	handler.resetFlushTimer()

	return handler
}

// SetCorrelationID sets the correlation ID for linking events
func (h *ApiHandler) SetCorrelationID(correlationID string) {
	h.correlationID = correlationID
}

// HandleMessage processes a message by forwarding to the base handler and adding to batch
func (h *ApiHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Forward to base handler first
	if h.baseHandler != nil {
		if err := h.baseHandler.HandleMessage(ctx, data, direction); err != nil {
			return err
		}
	}

	// Add to batch if client is available and correlation ID is set
	if h.apiClient != nil && h.correlationID != "" {
		h.addToBatch(ctx, data, direction)
	}

	return nil
}

// HandleError forwards error handling to the base handler
func (h *ApiHandler) HandleError(ctx context.Context, err error) {
	if h.baseHandler != nil {
		h.baseHandler.HandleError(ctx, err)
	}
}

// HandleStreamEvent forwards stream event handling to the base handler
func (h *ApiHandler) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	if h.baseHandler != nil {
		h.baseHandler.HandleStreamEvent(ctx, event)
	}
}

// Flush sends any pending events and stops the timer
func (h *ApiHandler) Flush(ctx context.Context) error {
	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	// Stop the timer
	if h.flushTimer != nil {
		h.flushTimer.Stop()
	}

	// Signal stop
	select {
	case h.stopChan <- struct{}{}:
	default:
	}

	// Send any remaining events
	if len(h.eventBuffer) > 0 {
		return h.sendBatch(ctx)
	}

	return nil
}

// addToBatch adds an event to the batch buffer
func (h *ApiHandler) addToBatch(ctx context.Context, data []byte, direction domain.Direction) {
	// Parse the JSON-RPC message to extract metadata
	message, err := domain.NewJSONRPCMessageFromRaw(data, direction, h.correlationID)
	if err != nil {
		// Create a raw event for non-JSON-RPC data
		h.addRawEventToBatch(ctx, data, direction)
		return
	}

	// Create batch event
	event := httpClient.BatchEventDto{
		Id:            string(message.ID()),
		Timestamp:     message.Timestamp().Format(time.RFC3339Nano),
		Direction:     h.mapDirection(direction),
		Method:        message.Method(),
		Payload:       base64.StdEncoding.EncodeToString(data),
		Size:          len(data),
		CorrelationId: h.correlationID,
	}

	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	h.eventBuffer = append(h.eventBuffer, event)

	// Check if we need to flush
	if len(h.eventBuffer) >= defaultBatchSize {
		go h.flushBatch(ctx)
	}
}

// addRawEventToBatch adds non-JSON-RPC data as a generic event
func (h *ApiHandler) addRawEventToBatch(ctx context.Context, data []byte, direction domain.Direction) {
	event := httpClient.BatchEventDto{
		Id:            fmt.Sprintf("raw-%d", time.Now().UnixNano()),
		Timestamp:     time.Now().Format(time.RFC3339Nano),
		Direction:     h.mapDirection(direction),
		Payload:       base64.StdEncoding.EncodeToString(data),
		Size:          len(data),
		CorrelationId: h.correlationID,
	}

	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	h.eventBuffer = append(h.eventBuffer, event)

	// Check if we need to flush
	if len(h.eventBuffer) >= defaultBatchSize {
		go h.flushBatch(ctx)
	}
}

// flushBatch sends the current batch if it has events
func (h *ApiHandler) flushBatch(ctx context.Context) {
	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	if len(h.eventBuffer) == 0 {
		return
	}

	if err := h.sendBatch(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[API] Failed to send batch: %v\n", err)
	}

	// Reset the timer after sending
	h.resetFlushTimer()
}

// sendBatch sends the current buffer contents (must be called with mutex held)
func (h *ApiHandler) sendBatch(ctx context.Context) error {
	if len(h.eventBuffer) == 0 {
		return nil
	}

	// Create batch request
	batch := httpClient.BatchRequest{
		Events:         make([]httpClient.BatchEventDto, len(h.eventBuffer)),
		CorrelationId:  h.correlationID,
		CliVersion:     cliVersion,
		BatchTimestamp: time.Now().Format(time.RFC3339Nano),
	}

	// Copy events to batch
	copy(batch.Events, h.eventBuffer)

	// Clear the buffer
	h.eventBuffer = h.eventBuffer[:0]

	// Send the batch
	if err := h.apiClient.SendBatchEvents(ctx, batch); err != nil {
		return err
	}

	return nil
}

// resetFlushTimer resets the flush timer
func (h *ApiHandler) resetFlushTimer() {
	if h.flushTimer != nil {
		h.flushTimer.Stop()
	}

	h.flushTimer = time.AfterFunc(defaultFlushInterval, func() {
		ctx := context.Background()
		h.flushBatch(ctx)
	})
}

// mapDirection converts domain.Direction to API string format
func (h *ApiHandler) mapDirection(direction domain.Direction) string {
	switch direction {
	case domain.DirectionInbound:
		return "request"
	case domain.DirectionOutbound:
		return "response"
	default:
		return string(direction)
	}
}
