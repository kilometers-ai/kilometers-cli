package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// KilometersAPIGateway implements the APIGateway interface
type KilometersAPIGateway struct {
	endpoint    string
	apiKey      string
	httpClient  *http.Client
	retryPolicy *RetryPolicy
	breaker     *CircuitBreaker
	logger      ports.LoggingGateway
	stats       *APIStats
	mutex       sync.RWMutex
}

// APIStats tracks API usage statistics
type APIStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalEvents        int64         `json:"total_events"`
	TotalPayloadSize   int64         `json:"total_payload_size"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastRequestTime    time.Time     `json:"last_request_time"`
	LastError          string        `json:"last_error,omitempty"`
	connectionStatus   ports.ConnectionStatus
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	Multiplier  float64       `json:"multiplier"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state           CircuitBreakerState
	mutex           sync.RWMutex
}

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// CanExecute returns true if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful execution
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	cb.state = StateClosed
}

// RecordFailure records a failed execution
func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = StateOpen
	}
}

// NewKilometersAPIGateway creates a new API gateway
func NewKilometersAPIGateway(endpoint, apiKey string, logger ports.LoggingGateway) *KilometersAPIGateway {
	return &KilometersAPIGateway{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryPolicy: DefaultRetryPolicy(),
		breaker:     NewCircuitBreaker(5, 60*time.Second),
		logger:      logger,
		stats: &APIStats{
			connectionStatus: ports.ConnectionStatus{
				IsConnected: false,
				Latency:     0,
				RetryCount:  0,
			},
		},
	}
}

// NewTestAPIGateway creates a new API gateway with test-friendly settings
func NewTestAPIGateway(endpoint, apiKey string, logger ports.LoggingGateway) *KilometersAPIGateway {
	return &KilometersAPIGateway{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Shorter timeout for tests
		},
		retryPolicy: &RetryPolicy{
			MaxAttempts: 2,                      // Fewer retries for tests
			BaseDelay:   100 * time.Millisecond, // Shorter delays
			MaxDelay:    1 * time.Second,
			Multiplier:  2.0,
		},
		breaker: NewCircuitBreaker(3, 5*time.Second), // Faster recovery for tests
		logger:  logger,
		stats: &APIStats{
			connectionStatus: ports.ConnectionStatus{
				IsConnected: false,
				Latency:     0,
				RetryCount:  0,
			},
		},
	}
}

// UpdateEndpoint safely updates the API endpoint at runtime
func (g *KilometersAPIGateway) UpdateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.logger.Log(ports.LogLevelInfo, "Updating API endpoint", map[string]interface{}{
		"old_endpoint": g.endpoint,
		"new_endpoint": endpoint,
	})

	g.endpoint = endpoint

	// Reset connection status since we're changing endpoints
	g.stats.connectionStatus = ports.ConnectionStatus{
		IsConnected: false,
		Latency:     0,
		RetryCount:  0,
	}

	return nil
}

// getEndpoint safely retrieves the current endpoint
func (g *KilometersAPIGateway) getEndpoint() string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.endpoint
}

// isDebugEnabled checks if debug logging is enabled
func (g *KilometersAPIGateway) isDebugEnabled() bool {
	// Check if the logger is set to debug level
	if g.logger != nil && g.logger.GetLogLevel() == ports.LogLevelDebug {
		return true
	}
	// Fallback to environment variable check
	return os.Getenv("KM_DEBUG") == "true"
}

// logHTTPRequest logs HTTP request details for debugging
func (g *KilometersAPIGateway) logHTTPRequest(req *http.Request, body []byte) {
	if !g.isDebugEnabled() {
		return
	}

	// Truncate body if too large for logging
	bodyPreview := string(body)
	if len(bodyPreview) > 1000 {
		bodyPreview = bodyPreview[:1000] + "... (truncated)"
	}

	g.logger.Log(ports.LogLevelDebug, "HTTP Request", map[string]interface{}{
		"method":       req.Method,
		"url":          req.URL.String(),
		"headers":      req.Header,
		"body_size":    len(body),
		"body_preview": bodyPreview,
	})
}

// logHTTPResponse logs HTTP response details for debugging
func (g *KilometersAPIGateway) logHTTPResponse(resp *http.Response, body []byte, latency time.Duration) {
	if !g.isDebugEnabled() {
		return
	}

	// Truncate response body if too large for logging
	bodyPreview := string(body)
	if len(bodyPreview) > 1000 {
		bodyPreview = bodyPreview[:1000] + "... (truncated)"
	}

	g.logger.Log(ports.LogLevelDebug, "HTTP Response", map[string]interface{}{
		"status_code":  resp.StatusCode,
		"status":       resp.Status,
		"headers":      resp.Header,
		"body_size":    len(body),
		"body_preview": bodyPreview,
		"latency_ms":   latency.Milliseconds(),
	})
}

// SendEventBatch sends a batch of events to the API
func (g *KilometersAPIGateway) SendEventBatch(batch *session.EventBatch) error {
	if batch == nil || len(batch.Events) == 0 {
		return fmt.Errorf("cannot send empty batch")
	}

	g.logger.Log(ports.LogLevelInfo, "Sending event batch", map[string]interface{}{
		"batch_id":   batch.ID,
		"batch_size": batch.Size,
		"session_id": batch.SessionID.Value(),
	})

	// Convert events to DTOs
	eventDtos := make([]EventDto, len(batch.Events))
	for i, evt := range batch.Events {
		eventDtos[i] = g.eventToDTO(evt)
	}

	// Create batch DTO
	batchDto := EventBatchDto{
		Events:         eventDtos,
		CliVersion:     "2.0.0", // TODO: Get from build info
		BatchTimestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Execute with retry and circuit breaker
	return g.executeWithRetry(func() error {
		return g.sendBatchRequest(batchDto, len(batch.Events))
	})
}

// SendEvent sends a single event to the API
func (g *KilometersAPIGateway) SendEvent(evt *event.Event) error {
	if evt == nil {
		return fmt.Errorf("cannot send nil event")
	}

	g.logger.Log(ports.LogLevelDebug, "Sending single event", map[string]interface{}{
		"event_id": evt.ID().Value(),
		"method":   evt.Method().Value(),
	})

	dto := g.eventToDTO(evt)

	// Execute with retry and circuit breaker
	return g.executeWithRetry(func() error {
		return g.sendEventRequest(dto)
	})
}

// CreateSession creates a new session on the server
func (g *KilometersAPIGateway) CreateSession(sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("cannot create nil session")
	}

	g.logger.Log(ports.LogLevelDebug, "Creating session", map[string]interface{}{
		"session_id": sess.ID().Value(),
		"state":      string(sess.State()),
	})

	sessionDto := SessionDto{
		ID:        sess.ID().Value(),
		CreatedAt: sess.StartTime().Unix(),
		Status:    string(sess.State()),
	}

	// Execute with retry and circuit breaker
	return g.executeWithRetry(func() error {
		return g.sendSessionRequest(sessionDto)
	})
}

// TestConnection tests the API connection and authentication
func (g *KilometersAPIGateway) TestConnection() error {
	endpoint := g.getEndpoint()
	g.logger.Log(ports.LogLevelInfo, "Testing API connection", map[string]interface{}{
		"endpoint": endpoint,
	})

	start := time.Now()

	// Test health endpoint
	err := g.executeWithRetry(func() error {
		req, err := http.NewRequest("GET", endpoint+"/health", nil)
		if err != nil {
			return fmt.Errorf("failed to create health request: %w", err)
		}

		req.Header.Set("X-Version", "1.0")

		// Debug log the health check request
		g.logHTTPRequest(req, []byte{})

		requestStart := time.Now()
		resp, err := g.httpClient.Do(req)
		requestLatency := time.Since(requestStart)

		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body for debug logging
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte{}
		}

		// Debug log the health check response
		g.logHTTPResponse(resp, body, requestLatency)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("health check failed with status %d", resp.StatusCode)
		}

		return nil
	})

	if err != nil {
		g.updateConnectionStatus(false, time.Since(start), err.Error())
		return err
	}

	// Test authentication if API key is provided
	if g.apiKey != "" {
		err = g.executeWithRetry(func() error {
			req, err := http.NewRequest("GET", endpoint+"/api/customer", nil)
			if err != nil {
				return fmt.Errorf("failed to create auth request: %w", err)
			}

			req.Header.Set("Authorization", "Bearer "+g.apiKey)
			req.Header.Set("X-Version", "1.0")

			// Debug log the auth check request
			g.logHTTPRequest(req, []byte{})

			requestStart := time.Now()
			resp, err := g.httpClient.Do(req)
			requestLatency := time.Since(requestStart)

			if err != nil {
				return fmt.Errorf("authentication test failed: %w", err)
			}
			defer resp.Body.Close()

			// Read response body for debug logging
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				body = []byte{}
			}

			// Debug log the auth check response
			g.logHTTPResponse(resp, body, requestLatency)

			if resp.StatusCode == 401 {
				return fmt.Errorf("API key authentication failed")
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("authentication test failed with status %d", resp.StatusCode)
			}

			return nil
		})

		if err != nil {
			g.updateConnectionStatus(false, time.Since(start), err.Error())
			return err
		}
	}

	latency := time.Since(start)
	g.updateConnectionStatus(true, latency, "")
	g.logger.Log(ports.LogLevelInfo, "API connection test successful", map[string]interface{}{
		"latency_ms": latency.Milliseconds(),
	})

	return nil
}

// GetConnectionStatus returns the current connection status
func (g *KilometersAPIGateway) GetConnectionStatus() ports.ConnectionStatus {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.stats.connectionStatus
}

// GetAPIInfo returns information about the API
func (g *KilometersAPIGateway) GetAPIInfo() (*ports.APIInfo, error) {
	// This would typically fetch from an API info endpoint
	endpoint := g.getEndpoint()
	return &ports.APIInfo{
		Version:     "2.0.0",
		Environment: "production",
		Region:      "us-east-1",
		Endpoints:   []string{endpoint},
		RateLimit: ports.RateLimit{
			RequestsPerMinute: 1000,
			RequestsPerHour:   10000,
			RequestsPerDay:    100000,
			CurrentUsage:      int(g.stats.TotalRequests),
		},
	}, nil
}

// ValidateAPIKey validates the provided API key
func (g *KilometersAPIGateway) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Store original key and test with provided key
	originalKey := g.apiKey
	g.apiKey = apiKey

	err := g.TestConnection()

	// Restore original key
	g.apiKey = originalKey

	return err
}

// GetUsageStats returns API usage statistics
func (g *KilometersAPIGateway) GetUsageStats() (*ports.APIUsageStats, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return &ports.APIUsageStats{
		TotalRequests:      g.stats.TotalRequests,
		SuccessfulRequests: g.stats.SuccessfulRequests,
		FailedRequests:     g.stats.FailedRequests,
		TotalEvents:        g.stats.TotalEvents,
		TotalPayloadSize:   g.stats.TotalPayloadSize,
		AverageLatency:     g.stats.AverageLatency,
		LastRequestTime:    g.stats.LastRequestTime,
	}, nil
}

// executeWithRetry executes a function with retry logic and circuit breaker
func (g *KilometersAPIGateway) executeWithRetry(fn func() error) error {
	if !g.breaker.CanExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	var lastErr error
	for attempt := 0; attempt < g.retryPolicy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := g.calculateDelay(attempt)
			g.logger.Log(ports.LogLevelDebug, "Retrying request", map[string]interface{}{
				"attempt": attempt + 1,
				"delay":   delay,
			})
			time.Sleep(delay)
		}

		g.updateStats(true, false, 0, 0, "")

		err := fn()
		if err == nil {
			g.breaker.RecordSuccess()
			g.updateStats(false, true, 0, 0, "")
			return nil
		}

		lastErr = err
		g.breaker.RecordFailure()
		g.updateStats(false, false, 0, 0, err.Error())

		// Don't retry on certain errors
		if !g.shouldRetry(err) {
			break
		}
	}

	return fmt.Errorf("request failed after %d attempts: %w", g.retryPolicy.MaxAttempts, lastErr)
}

// sendBatchRequest sends a batch request to the API
func (g *KilometersAPIGateway) sendBatchRequest(batchDto EventBatchDto, eventCount int) error {
	jsonData, err := json.Marshal(batchDto)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	endpoint := g.getEndpoint()
	req, err := http.NewRequest("POST", endpoint+"/api/events/batch", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	g.setRequestHeaders(req)

	// Debug log the request
	g.logHTTPRequest(req, jsonData)

	start := time.Now()
	resp, err := g.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once for both logging and error handling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug log the response
	g.logHTTPResponse(resp, body, latency)

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	g.updateStats(false, true, int64(eventCount), int64(len(jsonData)), "")
	g.updateLatency(latency)

	g.logger.Log(ports.LogLevelInfo, "Batch sent successfully", map[string]interface{}{
		"event_count": eventCount,
		"latency_ms":  latency.Milliseconds(),
	})

	return nil
}

// sendEventRequest sends a single event request to the API
func (g *KilometersAPIGateway) sendEventRequest(dto EventDto) error {
	jsonData, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	endpoint := g.getEndpoint()
	req, err := http.NewRequest("POST", endpoint+"/api/events", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	g.setRequestHeaders(req)

	// Debug log the request
	g.logHTTPRequest(req, jsonData)

	start := time.Now()
	resp, err := g.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once for both logging and error handling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug log the response
	g.logHTTPResponse(resp, body, latency)

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	g.updateStats(false, true, 1, int64(len(jsonData)), "")
	g.updateLatency(latency)

	return nil
}

// sendSessionRequest sends a session creation request to the API
func (g *KilometersAPIGateway) sendSessionRequest(dto SessionDto) error {
	jsonData, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	endpoint := g.getEndpoint()
	req, err := http.NewRequest("POST", endpoint+"/api/v1/sessions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}

	g.setRequestHeaders(req)

	// Debug log the request
	g.logHTTPRequest(req, jsonData)

	start := time.Now()
	resp, err := g.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to send session request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once for both logging and error handling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug log the response
	g.logHTTPResponse(resp, body, latency)

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed - check your API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("session creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	g.updateStats(false, true, 1, int64(len(jsonData)), "")
	g.updateLatency(latency)

	return nil
}

// setRequestHeaders sets common request headers
func (g *KilometersAPIGateway) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "kilometers-cli/2.0.0")
	req.Header.Set("X-Version", "1.0")

	if g.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.apiKey)
	}
}

// eventToDTO converts an Event to EventDto
func (g *KilometersAPIGateway) eventToDTO(evt *event.Event) EventDto {
	return EventDto{
		ID:        evt.ID().Value(),
		Timestamp: evt.Timestamp().UTC().Format(time.RFC3339),
		Direction: evt.Direction().String(),
		Method:    evt.Method().Value(),
		Payload:   base64.StdEncoding.EncodeToString(evt.Payload()),
		Size:      evt.Size(),
	}
}

// calculateDelay calculates the delay for retry attempts
func (g *KilometersAPIGateway) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(g.retryPolicy.BaseDelay) *
		float64(attempt) * g.retryPolicy.Multiplier)

	if delay > g.retryPolicy.MaxDelay {
		delay = g.retryPolicy.MaxDelay
	}

	return delay
}

// shouldRetry determines if an error should trigger a retry
func (g *KilometersAPIGateway) shouldRetry(err error) bool {
	// Don't retry on authentication errors
	if err.Error() == "authentication failed - check your API key" {
		return false
	}

	// Retry on network errors and 5xx status codes
	return true
}

// updateStats updates API statistics
func (g *KilometersAPIGateway) updateStats(isAttempt, isSuccess bool, events, payloadSize int64, errorMsg string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if isAttempt {
		g.stats.TotalRequests++
		g.stats.LastRequestTime = time.Now()
	}

	if isSuccess {
		g.stats.SuccessfulRequests++
		g.stats.TotalEvents += events
		g.stats.TotalPayloadSize += payloadSize
		g.stats.LastError = ""
	} else if !isAttempt {
		g.stats.FailedRequests++
		g.stats.LastError = errorMsg
	}
}

// updateLatency updates average latency
func (g *KilometersAPIGateway) updateLatency(latency time.Duration) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.stats.AverageLatency == 0 {
		g.stats.AverageLatency = latency
	} else {
		// Simple moving average
		g.stats.AverageLatency = (g.stats.AverageLatency + latency) / 2
	}
}

// updateConnectionStatus updates the connection status
func (g *KilometersAPIGateway) updateConnectionStatus(connected bool, latency time.Duration, errorMsg string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.stats.connectionStatus.IsConnected = connected
	g.stats.connectionStatus.Latency = latency
	g.stats.connectionStatus.LastError = errorMsg

	if connected {
		g.stats.connectionStatus.LastConnected = time.Now()
		g.stats.connectionStatus.RetryCount = 0
	} else {
		g.stats.connectionStatus.RetryCount++
	}
}

// EventDto represents the structure expected by the API
type EventDto struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Direction string `json:"direction"`
	Method    string `json:"method,omitempty"`
	Payload   string `json:"payload"` // Base64 encoded
	Size      int    `json:"size"`
}

// EventBatchDto represents a batch of events for the API
type EventBatchDto struct {
	Events         []EventDto `json:"events"`
	CliVersion     string     `json:"cliVersion"`
	BatchTimestamp string     `json:"batchTimestamp"`
}

// SessionDto represents a session for API requests
type SessionDto struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at"`
	Status    string `json:"status"`
}
