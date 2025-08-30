package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
	httpinfra "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
)

// AuthenticatedHTTPClient wraps an HTTP client with automatic authentication
type AuthenticatedHTTPClient struct {
	baseClient  *http.Client
	authManager auth.AuthManager
	scope       string
}

// NewAuthenticatedHTTPClient creates a new authenticated HTTP client
func NewAuthenticatedHTTPClient(authManager auth.AuthManager, scope string) *AuthenticatedHTTPClient {
	return &AuthenticatedHTTPClient{
		baseClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authManager: authManager,
		scope:       scope,
	}
}

// Do performs an HTTP request with automatic authentication
func (c *AuthenticatedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Get valid token
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	token, err := c.authManager.GetValidToken(ctx, c.scope)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Set authorization header
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))

	// Perform request
	resp, err := c.baseClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check if token was rejected
	if resp.StatusCode == http.StatusUnauthorized {
		// Force refresh and retry once
		token, err = c.authManager.ForceRefresh(ctx, c.scope)
		if err != nil {
			return resp, nil // Return original response
		}

		// Update authorization header
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))

		// Retry request
		resp.Body.Close()
		return c.baseClient.Do(req)
	}

	return resp, nil
}

// RoundTripper implementation for use with http.Client

// AuthenticatedRoundTripper implements http.RoundTripper with automatic authentication
type AuthenticatedRoundTripper struct {
	base        http.RoundTripper
	authManager auth.AuthManager
	scope       string
}

// NewAuthenticatedRoundTripper creates a new authenticated round tripper
func NewAuthenticatedRoundTripper(authManager auth.AuthManager, scope string) *AuthenticatedRoundTripper {
	return &AuthenticatedRoundTripper{
		base:        http.DefaultTransport,
		authManager: authManager,
		scope:       scope,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (t *AuthenticatedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Get valid token
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	token, err := t.authManager.GetValidToken(ctx, t.scope)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Clone request to avoid modifying the original
	newReq := req.Clone(ctx)

	// Set authorization header
	newReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))

	// Perform request
	resp, err := t.base.RoundTrip(newReq)
	if err != nil {
		return nil, err
	}

	// Check if token was rejected
	if resp.StatusCode == http.StatusUnauthorized {
		// Force refresh and retry once
		token, err = t.authManager.ForceRefresh(ctx, t.scope)
		if err != nil {
			return resp, nil // Return original response
		}

		// Update authorization header
		newReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))

		// Retry request
		resp.Body.Close()
		return t.base.RoundTrip(newReq)
	}

	return resp, nil
}

// CreateAuthenticatedClient creates an http.Client with automatic authentication
func CreateAuthenticatedClient(authManager auth.AuthManager, scope string) *http.Client {
	// Delegate to infrastructure round tripper; legacy kept above for rollback
	return &http.Client{
		Transport: httpinfra.NewRoundTripperWithAuth(authManager, scope),
		Timeout:   30 * time.Second,
	}
}
