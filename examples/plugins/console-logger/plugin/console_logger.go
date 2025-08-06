package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/ports/plugins"
)

// Build-time embedded security credentials
// In production, these would be generated per customer during the build process
const (
	// Embedded customer authentication token (encrypted in production)
	EmbeddedCustomerToken = "km_customer_demo_token_12345"

	// Plugin binary signature (generated during build)
	PluginSignature = "sha256:a1b2c3d4e5f6789..."

	// Public key for signature verification (embedded in plugin)
	PublicKeyFingerprint = "km_public_key_fingerprint_xyz"

	// Plugin metadata
	PluginName    = "console-logger"
	PluginVersion = "1.0.0"
	RequiredTier  = "Free"
)

// ConsoleLoggerPlugin implements a secure console logging plugin
type ConsoleLoggerPlugin struct {
	authenticated bool
	config        plugins.PluginConfig
	authToken     string
	lastVerified  time.Time
}

// Metadata methods
func (p *ConsoleLoggerPlugin) Name() string {
	return PluginName
}

func (p *ConsoleLoggerPlugin) Version() string {
	return PluginVersion
}

func (p *ConsoleLoggerPlugin) RequiredTier() string {
	return RequiredTier
}

// Authenticate performs multi-layer authentication
func (p *ConsoleLoggerPlugin) Authenticate(ctx context.Context, apiKey string) (*plugins.AuthResponse, error) {
	// Layer 1: Validate binary integrity
	if err := p.validateBinaryIntegrity(); err != nil {
		return nil, fmt.Errorf("binary integrity validation failed: %w", err)
	}

	// Layer 2: Validate embedded customer token
	if err := p.validateEmbeddedToken(apiKey); err != nil {
		return nil, fmt.Errorf("embedded token validation failed: %w", err)
	}

	// Layer 3: Authenticate with kilometers-api (in production)
	authResponse, err := p.authenticateWithAPI(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("API authentication failed: %w", err)
	}

	// Layer 4: Store authentication state
	p.authenticated = true
	p.authToken = authResponse.Token
	p.lastVerified = time.Now()

	return authResponse, nil
}

// Initialize initializes the plugin with configuration
func (p *ConsoleLoggerPlugin) Initialize(ctx context.Context, config plugins.PluginConfig) error {
	if !p.authenticated {
		return fmt.Errorf("plugin not authenticated")
	}

	p.config = config

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[ConsoleLogger] Plugin initialized (Free tier)\n")
		fmt.Fprintf(os.Stderr, "[ConsoleLogger] Features: %v\n", config.Features)
	}

	return nil
}

// Shutdown gracefully shuts down the plugin
func (p *ConsoleLoggerPlugin) Shutdown(ctx context.Context) error {
	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[ConsoleLogger] Plugin shutdown\n")
	}

	p.authenticated = false
	p.authToken = ""

	return nil
}

// HandleMessage processes an intercepted JSON-RPC message
func (p *ConsoleLoggerPlugin) HandleMessage(ctx context.Context, data []byte, direction string, correlationID string) error {
	// Security check: Ensure still authenticated
	if !p.isAuthenticationValid() {
		// Silent failure - plugin becomes inactive
		return nil
	}

	// Periodic re-verification (every 5 minutes)
	if time.Since(p.lastVerified) > 5*time.Minute {
		// In production, this would re-authenticate with the API
		p.lastVerified = time.Now()
	}

	// Process the message
	timestamp := time.Now().Format("15:04:05.000")
	directionIcon := "→"
	if direction == "response" {
		directionIcon = "←"
	}

	fmt.Fprintf(os.Stderr, "[%s] %s %s JSON-RPC (%d bytes) [%s]\n",
		timestamp, directionIcon, direction, len(data), correlationID)

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[ConsoleLogger] Data: %s\n", string(data))
	}

	return nil
}

// HandleError processes an error
func (p *ConsoleLoggerPlugin) HandleError(ctx context.Context, err error) error {
	if !p.isAuthenticationValid() {
		return nil
	}

	fmt.Fprintf(os.Stderr, "[ConsoleLogger] Error: %v\n", err)
	return nil
}

// HandleStreamEvent processes a stream lifecycle event
func (p *ConsoleLoggerPlugin) HandleStreamEvent(ctx context.Context, event plugins.StreamEvent) error {
	if !p.isAuthenticationValid() {
		return nil
	}

	if p.config.Debug {
		fmt.Fprintf(os.Stderr, "[ConsoleLogger] Stream Event: %s at %s\n",
			event.Type, event.Timestamp.Format("15:04:05.000"))
	}

	return nil
}

// Security validation methods

// validateBinaryIntegrity checks if the plugin binary has been tampered with
func (p *ConsoleLoggerPlugin) validateBinaryIntegrity() error {
	// In production, this would:
	// 1. Calculate the current binary's hash
	// 2. Compare with the embedded signature
	// 3. Verify signature with public key

	// For demo purposes, we'll do a simple check
	if PluginSignature == "" {
		return fmt.Errorf("missing plugin signature")
	}

	if PublicKeyFingerprint == "" {
		return fmt.Errorf("missing public key fingerprint")
	}

	// Simulate binary hash validation
	expectedHash := "a1b2c3d4e5f6789..."
	if !validateHash(expectedHash, PluginSignature) {
		return fmt.Errorf("binary integrity check failed")
	}

	return nil
}

// validateEmbeddedToken validates the embedded customer token against the API key
func (p *ConsoleLoggerPlugin) validateEmbeddedToken(apiKey string) error {
	// In production, this would decrypt and validate the embedded token
	// For demo purposes, we'll do a simple validation

	if EmbeddedCustomerToken == "" {
		return fmt.Errorf("missing embedded customer token")
	}

	// Simulate token validation
	if len(apiKey) < 10 {
		return fmt.Errorf("invalid API key format")
	}

	return nil
}

// authenticateWithAPI performs authentication with the kilometers-api
func (p *ConsoleLoggerPlugin) authenticateWithAPI(ctx context.Context, apiKey string) (*plugins.AuthResponse, error) {
	// In production, this would:
	// 1. Create authentication request with plugin metadata
	// 2. Send request to kilometers-api /api/plugins/authenticate
	// 3. Receive and validate authentication response

	// For demo purposes, return a mock successful response
	return &plugins.AuthResponse{
		Success:            true,
		Token:              "mock_plugin_token_" + hex.EncodeToString([]byte(apiKey))[:8],
		ExpiresAt:          time.Now().Add(5 * time.Minute),
		AuthorizedFeatures: []string{"console_logging"},
		SubscriptionTier:   RequiredTier,
		CustomerName:       "Demo Customer",
		PluginVersion:      PluginVersion,
	}, nil
}

// isAuthenticationValid checks if the current authentication is still valid
func (p *ConsoleLoggerPlugin) isAuthenticationValid() bool {
	if !p.authenticated {
		return false
	}

	// Check if token has expired (in production, would validate JWT)
	if p.authToken == "" {
		return false
	}

	// In production, would check JWT expiration
	// For demo, assume valid for 5 minutes from last verification
	return time.Since(p.lastVerified) <= 6*time.Minute
}

// Helper functions

// validateHash simulates cryptographic hash validation
func validateHash(expected, actual string) bool {
	// In production, this would perform actual cryptographic validation
	return expected != "" && actual != ""
}

// generatePluginHash creates a hash of the plugin for integrity checking
func generatePluginHash() string {
	// In production, this would hash the actual binary
	hash := sha256.Sum256([]byte(PluginName + PluginVersion + RequiredTier))
	return hex.EncodeToString(hash[:])
}
