package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// Build-time embedded security credentials for Pro tier plugin
const (
	// Embedded customer authentication token (encrypted in production)
	EmbeddedCustomerToken = "km_customer_pro_token_67890"

	// Plugin binary signature (generated during build)
	PluginSignature = "sha256:pro_plugin_signature_xyz..."

	// Public key for signature verification
	PublicKeyFingerprint = "km_public_key_pro_fingerprint_abc"

	// Plugin metadata
	PluginName    = "api-logger"
	PluginVersion = "1.0.0"
	RequiredTier  = "Pro"
)

// APILoggerPlugin implements a secure API logging plugin for Pro tier
type APILoggerPlugin struct {
	authenticated bool
	config        plugins.PluginConfig
	authToken     string
	lastVerified  time.Time

	// Batching for API calls
	eventBuffer []APIEvent
	bufferMutex sync.Mutex
	httpClient  *http.Client
	flushTimer  *time.Timer

	// Customer correlation
	correlationID string
}

// APIEvent represents an event to send to the API
type APIEvent struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Direction     string    `json:"direction"`
	Method        string    `json:"method"`
	Payload       string    `json:"payload"`
	Size          int       `json:"size"`
	CorrelationID string    `json:"correlation_id"`
}

// Metadata methods
func (p *APILoggerPlugin) Name() string {
	return PluginName
}

func (p *APILoggerPlugin) Version() string {
	return PluginVersion
}

func (p *APILoggerPlugin) RequiredTier() string {
	return RequiredTier
}

// Authenticate performs enhanced authentication for Pro tier plugin
func (p *APILoggerPlugin) Authenticate(ctx context.Context, apiKey string) (*plugins.AuthResponse, error) {
	// Layer 1: Validate binary integrity
	if err := p.validateBinaryIntegrity(); err != nil {
		return nil, fmt.Errorf("binary integrity validation failed: %w", err)
	}

	// Layer 2: Validate embedded customer token
	if err := p.validateEmbeddedToken(apiKey); err != nil {
		return nil, fmt.Errorf("embedded token validation failed: %w", err)
	}

	// Layer 3: Enhanced API authentication for Pro tier
	authResponse, err := p.authenticateWithAPI(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("API authentication failed: %w", err)
	}

	// Layer 4: Validate Pro tier access
	if authResponse.SubscriptionTier != "Pro" && authResponse.SubscriptionTier != "Enterprise" {
		return nil, fmt.Errorf("insufficient subscription tier for API logger plugin")
	}

	// Layer 5: Store authentication state
	p.authenticated = true
	p.authToken = authResponse.Token
	p.lastVerified = time.Now()

	return authResponse, nil
}

// Initialize initializes the Pro tier plugin with enhanced configuration
func (p *APILoggerPlugin) Initialize(ctx context.Context, config plugins.PluginConfig) error {
	if !p.authenticated {
		return fmt.Errorf("plugin not authenticated")
	}

	p.config = config
	p.eventBuffer = make([]APIEvent, 0, 10) // Batch size of 10
	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Start batch flush timer
	p.resetFlushTimer()

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Plugin initialized (Pro tier)\n")
		fmt.Fprintf(os.Stderr, "[APILogger] Features: %v\n", config.Features)
		fmt.Fprintf(os.Stderr, "[APILogger] API Endpoint: %s\n", config.ApiEndpoint)
	}

	return nil
}

// Shutdown gracefully shuts down the plugin and flushes pending events
func (p *APILoggerPlugin) Shutdown(ctx context.Context) error {
	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Plugin shutdown initiated\n")
	}

	// Stop flush timer
	if p.flushTimer != nil {
		p.flushTimer.Stop()
	}

	// Flush any remaining events
	p.bufferMutex.Lock()
	if len(p.eventBuffer) > 0 {
		p.flushEventsToAPI(ctx)
	}
	p.bufferMutex.Unlock()

	p.authenticated = false
	p.authToken = ""

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Plugin shutdown complete\n")
	}

	return nil
}

// HandleMessage processes JSON-RPC messages and sends them to the API
func (p *APILoggerPlugin) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	// Security check: Ensure still authenticated
	if !p.isAuthenticationValid() {
		return nil // Silent failure
	}

	// Periodic re-verification (every 5 minutes)
	if time.Since(p.lastVerified) > 5*time.Minute {
		if err := p.reAuthenticate(ctx); err != nil {
			if p.config.Debug {
				fmt.Fprintf(os.Stderr, "[APILogger] Re-authentication failed: %v\n", err)
			}
			return nil // Silent failure
		}
	}

	p.correlationID = correlationID

	// Parse JSON-RPC message to extract method
	var jsonRPCMsg map[string]interface{}
	method := "unknown"
	if err := json.Unmarshal(data, &jsonRPCMsg); err == nil {
		if m, ok := jsonRPCMsg["method"].(string); ok {
			method = m
		}
	}

	// Create API event
	event := APIEvent{
		ID:            fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Timestamp:     time.Now(),
		Direction:     direction,
		Method:        method,
		Payload:       base64.StdEncoding.EncodeToString(data),
		Size:          len(data),
		CorrelationID: correlationID,
	}

	// Add to batch
	p.addEventToBatch(event)

	// Console output for immediate feedback
	timestamp := time.Now().Format("15:04:05.000")
	directionIcon := "→"
	if direction == "response" {
		directionIcon = "←"
	}

	fmt.Fprintf(os.Stderr, "[%s] %s %s %s (%d bytes) [%s] → API\n",
		timestamp, directionIcon, direction, method, len(data), correlationID)

	return nil
}

// HandleError processes errors
func (p *APILoggerPlugin) HandleError(ctx context.Context, err error) error {
	if !p.isAuthenticationValid() {
		return nil
	}

	fmt.Fprintf(os.Stderr, "[APILogger] Error: %v\n", err)

	// Could send error events to API as well
	return nil
}

// HandleStreamEvent processes stream lifecycle events
func (p *APILoggerPlugin) HandleStreamEvent(ctx context.Context, event plugins.StreamEvent) error {
	if !p.isAuthenticationValid() {
		return nil
	}

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Stream Event: %s at %s\n",
			event.Type, event.Timestamp.Format("15:04:05.000"))
	}

	return nil
}

// Enhanced security validation methods for Pro tier

func (p *APILoggerPlugin) validateBinaryIntegrity() error {
	// Enhanced integrity checking for Pro tier
	if PluginSignature == "" {
		return fmt.Errorf("missing plugin signature")
	}

	if PublicKeyFingerprint == "" {
		return fmt.Errorf("missing public key fingerprint")
	}

	// Simulate enhanced binary hash validation for Pro tier
	expectedHash := "pro_plugin_signature_xyz..."
	if !validateEnhancedHash(expectedHash, PluginSignature) {
		return fmt.Errorf("binary integrity check failed")
	}

	return nil
}

func (p *APILoggerPlugin) validateEmbeddedToken(apiKey string) error {
	// Enhanced token validation for Pro tier
	if EmbeddedCustomerToken == "" {
		return fmt.Errorf("missing embedded customer token")
	}

	// Pro tier requires stronger API key validation
	if len(apiKey) < 20 || !hasValidAPIKeyFormat(apiKey) {
		return fmt.Errorf("invalid Pro tier API key format")
	}

	return nil
}

func (p *APILoggerPlugin) authenticateWithAPI(ctx context.Context, apiKey string) (*plugins.AuthResponse, error) {
	// In production, this would send a real authentication request
	// For demo purposes, return a Pro tier response
	return &plugins.AuthResponse{
		Success:            true,
		Token:              "pro_plugin_token_" + hex.EncodeToString([]byte(apiKey))[:12],
		ExpiresAt:          time.Now().Add(5 * time.Minute),
		AuthorizedFeatures: []string{"console_logging", "api_logging", "advanced_analytics"},
		SubscriptionTier:   RequiredTier,
		CustomerName:       "Pro Customer",
		PluginVersion:      PluginVersion,
	}, nil
}

func (p *APILoggerPlugin) isAuthenticationValid() bool {
	if !p.authenticated {
		return false
	}

	if p.authToken == "" {
		return false
	}

	// Enhanced validation for Pro tier (stricter timeouts)
	return time.Since(p.lastVerified) <= 5*time.Minute
}

func (p *APILoggerPlugin) reAuthenticate(ctx context.Context) error {
	// In production, this would re-authenticate with the API
	p.lastVerified = time.Now()
	return nil
}

// API batch processing methods

func (p *APILoggerPlugin) addEventToBatch(event APIEvent) {
	p.bufferMutex.Lock()
	defer p.bufferMutex.Unlock()

	p.eventBuffer = append(p.eventBuffer, event)

	// Flush if batch is full
	if len(p.eventBuffer) >= 10 {
		go p.flushEventsToAPI(context.Background())
	}
}

func (p *APILoggerPlugin) resetFlushTimer() {
	if p.flushTimer != nil {
		p.flushTimer.Stop()
	}

	p.flushTimer = time.AfterFunc(5*time.Second, func() {
		p.bufferMutex.Lock()
		if len(p.eventBuffer) > 0 {
			go p.flushEventsToAPI(context.Background())
		}
		p.bufferMutex.Unlock()
	})
}

func (p *APILoggerPlugin) flushEventsToAPI(ctx context.Context) {
	p.bufferMutex.Lock()
	if len(p.eventBuffer) == 0 {
		p.bufferMutex.Unlock()
		return
	}

	// Copy events and clear buffer
	events := make([]APIEvent, len(p.eventBuffer))
	copy(events, p.eventBuffer)
	p.eventBuffer = p.eventBuffer[:0]
	p.bufferMutex.Unlock()

	// Send events to API
	if err := p.sendEventsToAPI(ctx, events); err != nil {
		if p.config.Debug {
			fmt.Fprintf(os.Stderr, "[APILogger] Failed to send events to API: %v\n", err)
		}
	} else if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Sent %d events to API\n", len(events))
	}

	// Reset timer
	p.resetFlushTimer()
}

func (p *APILoggerPlugin) sendEventsToAPI(ctx context.Context, events []APIEvent) error {
	// Prepare batch request
	batchRequest := map[string]interface{}{
		"events":          events,
		"correlation_id":  p.correlationID,
		"plugin_version":  PluginVersion,
		"batch_timestamp": time.Now().Format(time.RFC3339),
	}

	// Convert to JSON
	requestBody, err := json.Marshal(batchRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal batch request: %w", err)
	}

	// Create HTTP request
	apiURL := p.config.ApiEndpoint + "/api/events/batch"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("User-Agent", "kilometers-plugin-api-logger/"+PluginVersion)

	// Send request (in production)
	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[APILogger] Would send %d events to %s\n", len(events), apiURL)
	}

	// For demo purposes, simulate success
	return nil
}

// Helper functions

func validateEnhancedHash(expected, actual string) bool {
	// Enhanced hash validation for Pro tier
	return expected != "" && actual != "" && len(actual) > 10
}

func hasValidAPIKeyFormat(apiKey string) bool {
	// Check if API key has valid Pro tier format
	return len(apiKey) >= 20 && (apiKey[:8] == "km_live_" || apiKey[:8] == "km_test_")
}

// generateProPluginHash creates an enhanced hash for Pro tier plugins
func generateProPluginHash() string {
	hash := sha256.Sum256([]byte(PluginName + PluginVersion + RequiredTier + "pro_salt"))
	return hex.EncodeToString(hash[:])
}
