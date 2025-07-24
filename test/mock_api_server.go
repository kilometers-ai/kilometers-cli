package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MockAPIServer implements a mock Kilometers API server for testing
type MockAPIServer struct {
	server              *http.Server
	mux                 *http.ServeMux
	mu                  sync.RWMutex
	isRunning           bool
	requestCount        int64
	failureRate         float64
	latency             time.Duration
	authTokens          map[string]bool
	refreshTokens       map[string]string    // Maps refresh tokens to access tokens
	accessTokens        map[string]time.Time // Maps access tokens to expiration times
	responseOverrides   map[string]MockResponse
	requestLog          []APIRequest
	circuitBreakerOpen  bool
	consecutiveFailures int
	maxFailures         int
	resetTime           time.Time
	ctx                 context.Context
	cancel              context.CancelFunc
}

// APIRequest represents a logged API request
type APIRequest struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Timestamp time.Time         `json:"timestamp"`
	ClientIP  string            `json:"client_ip"`
}

// MockResponse represents a configured response
type MockResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string]string      `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	Delay      time.Duration          `json:"delay"`
}

// EventBatch represents a batch of events for the API
type EventBatch struct {
	SessionID string                   `json:"session_id"`
	Events    []map[string]interface{} `json:"events"`
	Metadata  map[string]interface{}   `json:"metadata"`
}

// NewMockAPIServer creates a new mock API server
func NewMockAPIServer() *MockAPIServer {
	ctx, cancel := context.WithCancel(context.Background())

	server := &MockAPIServer{
		mux:               http.NewServeMux(),
		authTokens:        make(map[string]bool),
		refreshTokens:     make(map[string]string),
		accessTokens:      make(map[string]time.Time),
		responseOverrides: make(map[string]MockResponse),
		requestLog:        make([]APIRequest, 0),
		failureRate:       0.0,
		latency:           0,
		maxFailures:       5,
		ctx:               ctx,
		cancel:            cancel,
	}

	server.setupRoutes()

	return server
}

// Start starts the mock API server on the specified port
func (s *MockAPIServer) Start(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.mux,
	}

	s.isRunning = true

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the mock API server
func (s *MockAPIServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	s.cancel()
	s.isRunning = false

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	return nil
}

// setupRoutes sets up the API routes
func (s *MockAPIServer) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleWithMiddleware(s.handleHealth))
	// Accept both /auth/token and /api/auth/token
	s.mux.HandleFunc("/auth/token", s.handleWithMiddleware(s.handleAuthToken))
	s.mux.HandleFunc("/api/auth/token", s.handleWithMiddleware(s.handleAuthToken))
	s.mux.HandleFunc("/auth/refresh", s.handleWithMiddleware(s.handleAuthRefresh))
	s.mux.HandleFunc("/api/events/batch", s.handleWithMiddleware(s.handleEventBatch))
	s.mux.HandleFunc("/api/sessions", s.handleWithMiddleware(s.handleSessions))
	s.mux.HandleFunc("/api/sessions/", s.handleWithMiddleware(s.handleSessionByID))
	s.mux.HandleFunc("/api/config", s.handleWithMiddleware(s.handleConfig))
	s.mux.HandleFunc("/api/customer", s.handleWithMiddleware(s.handleCustomer))
}

// handleWithMiddleware wraps handlers with common middleware
func (s *MockAPIServer) handleWithMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log request
		s.logRequest(r)

		// Increment request count
		atomic.AddInt64(&s.requestCount, 1)

		// Add artificial latency (read with lock)
		s.mu.RLock()
		latency := s.latency
		s.mu.RUnlock()

		if latency > 0 {
			time.Sleep(latency)
		}

		// Check circuit breaker
		if s.isCircuitOpen() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Service temporarily unavailable (circuit breaker open)",
			})
			return
		}

		// Simulate failures
		if s.shouldFail() {
			s.mu.Lock()
			s.consecutiveFailures++
			shouldOpenCircuit := s.consecutiveFailures >= s.maxFailures
			s.mu.Unlock()

			if shouldOpenCircuit {
				s.openCircuitBreaker()
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Simulated server error",
			})
			return
		}

		// Reset failure count on success
		s.mu.Lock()
		s.consecutiveFailures = 0
		s.mu.Unlock()

		// Check for response override (read with lock)
		s.mu.RLock()
		override, exists := s.responseOverrides[r.URL.Path]
		s.mu.RUnlock()

		if exists {
			if override.Delay > 0 {
				time.Sleep(override.Delay)
			}

			for key, value := range override.Headers {
				w.Header().Set(key, value)
			}

			w.WriteHeader(override.StatusCode)
			json.NewEncoder(w).Encode(override.Body)
			return
		}

		// Call actual handler
		handler(w, r)
	}
}

// logRequest logs an API request
func (s *MockAPIServer) logRequest(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}

	request := APIRequest{
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   headers,
		Body:      string(body),
		Timestamp: time.Now(),
		ClientIP:  r.RemoteAddr,
	}

	s.mu.Lock()
	s.requestLog = append(s.requestLog, request)
	s.mu.Unlock()
}

// shouldFail determines if the request should fail based on failure rate
func (s *MockAPIServer) shouldFail() bool {
	s.mu.RLock()
	failureRate := s.failureRate
	s.mu.RUnlock()

	if failureRate <= 0 {
		return false
	}
	return rand.Float64() < failureRate
}

// isCircuitOpen checks if the circuit breaker is open
func (s *MockAPIServer) isCircuitOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.circuitBreakerOpen {
		return false
	}

	// Check if reset time has passed
	if time.Now().After(s.resetTime) {
		s.circuitBreakerOpen = false
		s.consecutiveFailures = 0
		return false
	}

	return true
}

// openCircuitBreaker opens the circuit breaker
func (s *MockAPIServer) openCircuitBreaker() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.circuitBreakerOpen = true
	s.resetTime = time.Now().Add(30 * time.Second) // 30 second reset
}

// Handler implementations

func (s *MockAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "mock-1.0.0",
	})
}

func (s *MockAPIServer) handleAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var authRequest map[string]string
	if err := json.NewDecoder(r.Body).Decode(&authRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	// Accept both 'apiKey' and 'api_key' for compatibility
	apiKey := authRequest["api_key"]
	if apiKey == "" {
		apiKey = authRequest["apiKey"]
	}

	// Simple mock authentication
	if apiKey == "test_key" {
		accessToken := fmt.Sprintf("mock_access_token_%d", time.Now().Unix())
		refreshToken := fmt.Sprintf("mock_refresh_token_%d", time.Now().Unix())
		expiry := time.Now().Add(time.Hour * 24) // Mock expiry

		s.authTokens[accessToken] = true
		s.accessTokens[accessToken] = expiry
		s.refreshTokens[refreshToken] = accessToken

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"token": map[string]interface{}{
				"accessToken":                accessToken,
				"refreshToken":               refreshToken,
				"accessTokenExpiresAt":       time.Now().Add(time.Hour).Format(time.RFC3339),
				"refreshTokenExpiresAt":      time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
				"tokenType":                  "Bearer",
				"accessTokenLifetimeMinutes": 60,
				"refreshTokenLifetimeDays":   7,
			},
		})
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key"})
	}
}

func (s *MockAPIServer) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var refreshRequest map[string]string
	if err := json.NewDecoder(r.Body).Decode(&refreshRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	refreshToken := refreshRequest["refresh_token"]
	if _, exists := s.refreshTokens[refreshToken]; !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid refresh token"})
		return
	}

	accessToken := fmt.Sprintf("mock_access_token_%d", time.Now().Unix())
	expiry := time.Now().Add(time.Hour * 24) // Mock expiry
	s.accessTokens[accessToken] = expiry
	s.refreshTokens[refreshToken] = accessToken

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": accessToken,
		"expires_in":   3600,
		"token_type":   "Bearer",
	})
}

func (s *MockAPIServer) handleEventBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if !s.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	var batch EventBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	// Validate batch
	if batch.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing session_id"})
		return
	}

	if len(batch.Events) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Empty events array"})
		return
	}

	// Success response
	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"batch_id":     batchID,
		"session_id":   batch.SessionID,
		"events_count": len(batch.Events),
		"status":       "accepted",
		"processed_at": time.Now().Unix(),
	})
}

func (s *MockAPIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List sessions
		sessions := []map[string]interface{}{
			{
				"id":          "session_123",
				"created_at":  time.Now().Add(-1 * time.Hour).Unix(),
				"status":      "active",
				"event_count": 42,
			},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessions,
			"total":    len(sessions),
		})

	case http.MethodPost:
		// Create session
		sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         sessionID,
			"created_at": time.Now().Unix(),
			"status":     "created",
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *MockAPIServer) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Extract session ID from path
	sessionID := r.URL.Path[len("/api/v1/sessions/"):]

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          sessionID,
			"created_at":  time.Now().Add(-1 * time.Hour).Unix(),
			"status":      "active",
			"event_count": 42,
			"metadata": map[string]interface{}{
				"client_version": "1.0.0",
				"platform":       "test",
			},
		})

	case http.MethodPatch:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         sessionID,
			"updated_at": time.Now().Unix(),
			"status":     "updated",
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *MockAPIServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	config := map[string]interface{}{
		"batch_size":                100,
		"batch_timeout":             "30s",
		"retry_attempts":            3,
		"retry_delay":               "1s",
		"circuit_breaker_threshold": 5,
		"rate_limit":                1000,
		"features": map[string]bool{
			"risk_analysis":     true,
			"content_filtering": true,
			"real_time_alerts":  false,
		},
	}

	json.NewEncoder(w).Encode(config)
}

func (s *MockAPIServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":     "1.0.0",
		"api_version": "v1",
		"build_time":  time.Now().Format(time.RFC3339),
		"status":      "stable",
	})
}

func (s *MockAPIServer) handleCustomer(w http.ResponseWriter, r *http.Request) {
	if !s.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"customer_id": "test_customer_123",
		"name":        "Test Customer",
		"email":       "test@example.com",
		"plan":        "developer",
		"status":      "active",
	})
}

// isAuthenticated checks if the request is authenticated
func (s *MockAPIServer) isAuthenticated(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	prefix := "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}

	token := authHeader[len(prefix):]

	// Check if token exists and is not expired
	if expiry, exists := s.accessTokens[token]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		// Token expired, remove it
		delete(s.accessTokens, token)
	}

	// Fallback to old auth tokens for backward compatibility
	return s.authTokens[token]
}

// Configuration methods for testing

// SetFailureRate sets the failure rate (0.0 to 1.0)
func (s *MockAPIServer) SetFailureRate(rate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureRate = rate
}

// SetLatency sets artificial latency for all requests
func (s *MockAPIServer) SetLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = latency
}

// SetResponseOverride sets a custom response for a specific path
func (s *MockAPIServer) SetResponseOverride(path string, response MockResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseOverrides[path] = response
}

// AddAuthToken adds a valid authentication token
func (s *MockAPIServer) AddAuthToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authTokens[token] = true
}

// RemoveAuthToken removes an authentication token
func (s *MockAPIServer) RemoveAuthToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.authTokens, token)
}

// GetStats returns server statistics
func (s *MockAPIServer) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"is_running":           s.isRunning,
		"request_count":        atomic.LoadInt64(&s.requestCount),
		"failure_rate":         s.failureRate,
		"latency_ms":           s.latency.Milliseconds(),
		"circuit_breaker_open": s.circuitBreakerOpen,
		"consecutive_failures": s.consecutiveFailures,
		"auth_tokens_count":    len(s.authTokens),
		"request_log_size":     len(s.requestLog),
	}
}

// GetRequestLog returns the request log
func (s *MockAPIServer) GetRequestLog() []APIRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logCopy := make([]APIRequest, len(s.requestLog))
	copy(logCopy, s.requestLog)
	return logCopy
}

// ClearRequestLog clears the request log
func (s *MockAPIServer) ClearRequestLog() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestLog = s.requestLog[:0]
}

// ResetCircuitBreaker manually resets the circuit breaker
func (s *MockAPIServer) ResetCircuitBreaker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.circuitBreakerOpen = false
	s.consecutiveFailures = 0
}
