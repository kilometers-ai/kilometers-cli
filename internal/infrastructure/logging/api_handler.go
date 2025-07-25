package logging

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	httpClient "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
)

// ApiHandler wraps another MessageHandler and sends events to the kilometers-api
type ApiHandler struct {
	baseHandler   ports.MessageHandler
	apiClient     *httpClient.ApiClient
	correlationID string
}

// NewApiHandler creates a new API handler that wraps another handler
func NewApiHandler(baseHandler ports.MessageHandler) *ApiHandler {
	return &ApiHandler{
		baseHandler: baseHandler,
		apiClient:   httpClient.NewApiClient(),
	}
}

// SetCorrelationID sets the correlation ID for linking events
func (h *ApiHandler) SetCorrelationID(correlationID string) {
	h.correlationID = correlationID
}

// HandleMessage processes a message by forwarding to the base handler and sending to API
func (h *ApiHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Forward to base handler first
	if h.baseHandler != nil {
		if err := h.baseHandler.HandleMessage(ctx, data, direction); err != nil {
			return err
		}
	}

	// Send to API if client is available and correlation ID is set
	if h.apiClient != nil && h.correlationID != "" {
		go h.sendToApi(ctx, data, direction)
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

// sendToApi sends the message data to the kilometers-api
func (h *ApiHandler) sendToApi(ctx context.Context, data []byte, direction domain.Direction) {
	// Parse the JSON-RPC message to extract metadata
	message, err := domain.NewJSONRPCMessageFromRaw(data, direction, h.correlationID)
	if err != nil {
		// Not a valid JSON-RPC message, but still send raw data
		h.sendRawEvent(ctx, data, direction)
		return
	}

	// Create DTO for API
	event := httpClient.McpEventDto{
		Id:        string(message.ID()),
		Timestamp: message.Timestamp().Format(time.RFC3339Nano),
		Direction: h.mapDirection(direction),
		Method:    message.Method(),
		Payload:   base64.StdEncoding.EncodeToString(data),
		Size:      len(data),
		SessionId: h.correlationID,
	}

	// Send to API
	if err := h.apiClient.SendEvent(ctx, event); err != nil {
		// Log error but don't block
		fmt.Fprintf(os.Stderr, "[API] Failed to send event: %v\n", err)
	}
}

// sendRawEvent sends non-JSON-RPC data as a generic event
func (h *ApiHandler) sendRawEvent(ctx context.Context, data []byte, direction domain.Direction) {
	event := httpClient.McpEventDto{
		Id:        fmt.Sprintf("raw-%d", time.Now().UnixNano()),
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Direction: h.mapDirection(direction),
		Payload:   base64.StdEncoding.EncodeToString(data),
		Size:      len(data),
		SessionId: h.correlationID,
	}

	if err := h.apiClient.SendEvent(ctx, event); err != nil {
		fmt.Fprintf(os.Stderr, "[API] Failed to send raw event: %v\n", err)
	}
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
