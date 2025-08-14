package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/config"
)

// ApiClient handles HTTP communication with kilometers-api
type ApiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// McpEventDto represents an MCP event for the API
type McpEventDto struct {
	Id            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	Direction     string `json:"direction"`
	Method        string `json:"method,omitempty"`
	Payload       string `json:"payload"` // base64 encoded
	Size          int    `json:"size"`
	CorrelationId string `json:"correlationId,omitempty"`
}

// BatchEventDto represents an individual event within a batch
type BatchEventDto struct {
	Id            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	Direction     string `json:"direction"`
	Method        string `json:"method,omitempty"`
	Payload       string `json:"payload"` // base64 encoded
	Size          int    `json:"size"`
	CorrelationId string `json:"correlationId,omitempty"`
}

// BatchRequest represents a batch of events to send to the API
type BatchRequest struct {
	Events         []BatchEventDto `json:"events"`
	CorrelationId  string          `json:"correlationId"`
	CliVersion     string          `json:"cliVersion"`
	BatchTimestamp string          `json:"batchTimestamp"`
}

// NewApiClient creates a new API client using unified configuration
func NewApiClient() *ApiClient {
	loader, storage, err := config.CreateConfigServiceFromDefaults()
	if err != nil {
		return nil // Failed to create config service
	}
	configService := config.NewConfigService(loader, storage)

	ctx := context.Background()
	unifiedConfig, err := configService.Load(ctx)
	if err != nil {
		return nil // Failed to load config
	}

	if !unifiedConfig.HasAPIKey() {
		return nil // No API key, no client
	}

	return &ApiClient{
		baseURL: unifiedConfig.APIEndpoint,
		apiKey:  unifiedConfig.APIKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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

// GetUserFeatures retrieves the user's subscription features from the API
func (c *ApiClient) GetUserFeatures(ctx context.Context) (*UserFeaturesResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("no API client configured")
	}

	url := fmt.Sprintf("%s/api/user/features", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var features UserFeaturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&features); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &features, nil
}

// UserFeaturesResponse represents the API response for user features
type UserFeaturesResponse struct {
	Tier      string   `json:"tier"`
	Features  []string `json:"features"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

// SendBatchEvents sends a batch of events to the API
func (c *ApiClient) SendBatchEvents(ctx context.Context, batch BatchRequest) error {
	if c == nil {
		return nil // No client, no-op
	}

	url := fmt.Sprintf("%s/api/events/batch", c.baseURL)

	batchData, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(batchData))
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
