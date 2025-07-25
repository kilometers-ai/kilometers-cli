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

// ApiHandler implements MessageHandler to send events to kilometers-api
type ApiHandler struct {
	apiClient *httpClient.ApiClient
	sessionID string
	console   ports.MessageHandler
}

// NewApiHandler creates a new API handler
func NewApiHandler(console ports.MessageHandler) *ApiHandler {
	apiClient := httpClient.NewApiClient()

	return &ApiHandler{
		apiClient: apiClient,
		console:   console,
	}
}

// SetSessionID sets the session ID for linking events
func (h *ApiHandler) SetSessionID(sessionID string) {
	h.sessionID = sessionID
}

// HandleMessage processes an intercepted JSON-RPC message
func (h *ApiHandler) HandleMessage(ctx context.Context, data []byte, direction domain.Direction) error {
	// Always log to console first (primary behavior)
	if h.console != nil {
		if err := h.console.HandleMessage(ctx, data, direction); err != nil {
			return err
		}
	}

	// Send to API in background (non-blocking)
	if h.apiClient != nil && h.sessionID != "" {
		go h.sendToApi(context.Background(), data, direction)
	}

	return nil
}

// HandleError processes an error that occurred during message handling
func (h *ApiHandler) HandleError(ctx context.Context, err error) {
	// Forward to console handler
	if h.console != nil {
		h.console.HandleError(ctx, err)
	}

	// Log API errors to stderr
	if h.apiClient != nil {
		fmt.Fprintf(os.Stderr, "[API] Error: %v\n", err)
	}
}

// HandleStreamEvent processes stream lifecycle events
func (h *ApiHandler) HandleStreamEvent(ctx context.Context, event ports.StreamEvent) {
	// Forward to console handler
	if h.console != nil {
		h.console.HandleStreamEvent(ctx, event)
	}
}

// sendToApi sends the event to the kilometers-api (non-blocking)
func (h *ApiHandler) sendToApi(ctx context.Context, data []byte, direction domain.Direction) {
	// Parse the JSON-RPC message to extract metadata
	message, err := domain.NewJSONRPCMessageFromRaw(data, direction, domain.SessionID(h.sessionID))
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
		SessionId: h.sessionID,
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
		SessionId: h.sessionID,
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
