package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
)

// ApiClient handles HTTP communication with kilometers-api
type ApiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// SessionResponse represents the response from creating a session
type SessionResponse struct {
	SessionId string `json:"sessionId"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"`
}

// McpEventDto represents an MCP event for the API
type McpEventDto struct {
	Id        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Direction string `json:"direction"`
	Method    string `json:"method,omitempty"`
	Payload   string `json:"payload"` // base64 encoded
	Size      int    `json:"size"`
	SessionId string `json:"sessionId,omitempty"`
}

// NewApiClient creates a new API client using configuration
func NewApiClient() *ApiClient {
	config := domain.LoadConfig()

	if config.ApiKey == "" {
		return nil // No API key, no client
	}

	return &ApiClient{
		baseURL: config.ApiEndpoint,
		apiKey:  config.ApiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateSession creates a new session in the API
func (c *ApiClient) CreateSession(ctx context.Context) (*SessionResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("API client not initialized")
	}

	url := fmt.Sprintf("%s/api/sessions", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &sessionResp, nil
}

// SendEvent sends an MCP event to the API
func (c *ApiClient) SendEvent(ctx context.Context, event McpEventDto) error {
	if c == nil {
		return nil // No client, no-op
	}

	url := fmt.Sprintf("%s/api/events", c.baseURL)

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
