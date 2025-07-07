package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// EventDto represents the structure expected by the API
type EventDto struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Direction string `json:"direction"`
	Method    string `json:"method,omitempty"`
	Payload   string `json:"payload"` // Base64 encoded
	Size      int    `json:"size"`
	RiskScore int    `json:"riskScore,omitempty"` // Client-side risk assessment
}

// EventBatchDto represents a batch of events for the API
type EventBatchDto struct {
	Events         []EventDto `json:"events"`
	CliVersion     string     `json:"cliVersion"`
	BatchTimestamp string     `json:"batchTimestamp"`
}

// APIClient handles communication with the Kilometers API
type APIClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
	logger     *log.Logger
}

// NewAPIClient creates a new API client
func NewAPIClient(config *Config, logger *log.Logger) *APIClient {
	return &APIClient{
		endpoint: config.APIEndpoint,
		apiKey:   config.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// SendEvent sends a single event to the API
func (c *APIClient) SendEvent(event Event) error {
	dto := c.eventToDTO(event)

	jsonData, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", c.endpoint+"/api/events", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else {
		return fmt.Errorf("API key is required for authentication")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	c.logger.Printf("Successfully sent event %s to API", event.ID)
	return nil
}

// SendEventBatch sends multiple events to the API in a single batch
func (c *APIClient) SendEventBatch(events []Event) error {
	if len(events) == 0 {
		return nil
	}

	c.logger.Printf("Attempting to send batch of %d events to %s/api/events/batch", len(events), c.endpoint)

	// Convert events to DTOs
	eventDtos := make([]EventDto, len(events))
	for i, event := range events {
		eventDtos[i] = c.eventToDTO(event)
	}

	// Create batch DTO
	batchDto := EventBatchDto{
		Events:         eventDtos,
		CliVersion:     "1.0.0", // TODO: Make this configurable
		BatchTimestamp: time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(batchDto)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	c.logger.Printf("Sending JSON payload: %s", string(jsonData))

	req, err := http.NewRequest("POST", c.endpoint+"/api/events/batch", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else {
		return fmt.Errorf("API key is required for authentication")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("HTTP request failed: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Printf("API response: HTTP %d", resp.StatusCode)

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	c.logger.Printf("✅ Successfully sent batch of %d events to API", len(events))
	return nil
}

// TestConnection tests the API connection
func (c *APIClient) TestConnection() error {
	req, err := http.NewRequest("GET", c.endpoint+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		// Test authentication with the customer endpoint
		authReq, err := http.NewRequest("GET", c.endpoint+"/api/customer", nil)
		if err != nil {
			return fmt.Errorf("failed to create auth test request: %w", err)
		}

		authReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		authResp, err := c.httpClient.Do(authReq)
		if err != nil {
			return fmt.Errorf("failed to test authentication: %w", err)
		}
		defer authResp.Body.Close()

		if authResp.StatusCode == 401 {
			return fmt.Errorf("API key authentication failed")
		}
		if authResp.StatusCode < 200 || authResp.StatusCode >= 300 {
			return fmt.Errorf("authentication test failed with status %d", authResp.StatusCode)
		}

		c.logger.Printf("✅ API key authentication successful")
	}

	// Test basic health endpoint
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// eventToDTO converts an Event to EventDto
func (c *APIClient) eventToDTO(event Event) EventDto {
	return EventDto{
		ID:        event.ID,
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339),
		Direction: event.Direction,
		Method:    event.Method,
		Payload:   base64.StdEncoding.EncodeToString(event.Payload),
		Size:      event.Size,
		RiskScore: event.RiskScore,
	}
}
