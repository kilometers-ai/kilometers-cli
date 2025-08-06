package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
)

// AutoRefreshConfig configures the auto-refresh behavior
type AutoRefreshConfig struct {
	RefreshAhead  time.Duration // How long before expiry to refresh (default: 5 minutes)
	RetryInterval time.Duration // How long to wait between retry attempts (default: 30 seconds)
	MaxRetries    int           // Maximum number of refresh retries (default: 3)
	CheckInterval time.Duration // How often to check tokens (default: 1 minute)
}

// DefaultAutoRefreshConfig returns sensible defaults
func DefaultAutoRefreshConfig() *AutoRefreshConfig {
	return &AutoRefreshConfig{
		RefreshAhead:  5 * time.Minute,
		RetryInterval: 30 * time.Second,
		MaxRetries:    3,
		CheckInterval: 1 * time.Minute,
	}
}

// AutoRefreshAuthManager implements automatic token refresh
type AutoRefreshAuthManager struct {
	tokenProvider ports.TokenProvider
	tokenCache    ports.TokenCache
	config        *AutoRefreshConfig
	apiKey        string

	mu              sync.RWMutex
	refreshChannels map[string]chan struct{}
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

// NewAutoRefreshAuthManager creates a new auto-refresh auth manager
func NewAutoRefreshAuthManager(
	tokenProvider ports.TokenProvider,
	tokenCache ports.TokenCache,
	apiKey string,
	config *AutoRefreshConfig,
) *AutoRefreshAuthManager {
	if config == nil {
		config = DefaultAutoRefreshConfig()
	}

	return &AutoRefreshAuthManager{
		tokenProvider:   tokenProvider,
		tokenCache:      tokenCache,
		config:          config,
		apiKey:          apiKey,
		refreshChannels: make(map[string]chan struct{}),
		shutdown:        make(chan struct{}),
	}
}

// Start begins the auto-refresh background process
func (m *AutoRefreshAuthManager) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.autoRefreshLoop(ctx)
}

// Stop gracefully shuts down the auto-refresh process
func (m *AutoRefreshAuthManager) Stop() {
	close(m.shutdown)
	m.wg.Wait()
}

// GetValidToken returns a valid token, refreshing if necessary
func (m *AutoRefreshAuthManager) GetValidToken(ctx context.Context, scope string) (*domain.AuthToken, error) {
	// Try to get from cache first
	token, err := m.tokenCache.GetToken(scope)
	if err == nil && token != nil && !token.ShouldRefresh(m.config.RefreshAhead) {
		return token, nil
	}

	// Token needs refresh or doesn't exist
	return m.refreshToken(ctx, scope, token)
}

// ForceRefresh forces a token refresh regardless of expiration
func (m *AutoRefreshAuthManager) ForceRefresh(ctx context.Context, scope string) (*domain.AuthToken, error) {
	return m.refreshToken(ctx, scope, nil)
}

// ClearCache clears all cached authentication data
func (m *AutoRefreshAuthManager) ClearCache() error {
	return m.tokenCache.Clear()
}

// Private methods

func (m *AutoRefreshAuthManager) autoRefreshLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdown:
			return
		case <-ticker.C:
			m.checkAndRefreshTokens(ctx)
		}
	}
}

func (m *AutoRefreshAuthManager) checkAndRefreshTokens(ctx context.Context) {
	// Get all scopes that need checking
	// In a real implementation, this would iterate through known scopes
	scopes := []string{"default", "plugins", "monitoring"}

	for _, scope := range scopes {
		token, err := m.tokenCache.GetToken(scope)
		if err != nil || token == nil {
			continue
		}

		if token.ShouldRefresh(m.config.RefreshAhead) {
			// Refresh in background
			go func(s string, t *domain.AuthToken) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if _, err := m.refreshToken(ctx, s, t); err != nil {
					fmt.Printf("⚠️  Failed to auto-refresh token for scope %s: %v\n", s, err)
				}
			}(scope, token)
		}
	}
}

func (m *AutoRefreshAuthManager) refreshToken(ctx context.Context, scope string, existingToken *domain.AuthToken) (*domain.AuthToken, error) {
	// Ensure only one refresh per scope at a time
	m.mu.Lock()
	refreshChan, refreshing := m.refreshChannels[scope]
	if !refreshing {
		refreshChan = make(chan struct{})
		m.refreshChannels[scope] = refreshChan
	}
	m.mu.Unlock()

	if refreshing {
		// Wait for ongoing refresh to complete
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-refreshChan:
			// Refresh completed, try to get from cache
			return m.tokenCache.GetToken(scope)
		}
	}

	// Perform the refresh
	defer func() {
		m.mu.Lock()
		delete(m.refreshChannels, scope)
		close(refreshChan)
		m.mu.Unlock()
	}()

	var newToken *domain.AuthToken
	var err error

	// Try refresh token if available
	if existingToken != nil && existingToken.RefreshToken != "" {
		newToken, err = m.attemptRefreshWithRetry(ctx, existingToken.RefreshToken)
		// If refresh succeeded, use the new token
		if err == nil && newToken != nil {
			// Cache the new token
			if err := m.tokenCache.SetToken(scope, newToken); err != nil {
				// Log but don't fail - token is still valid
				fmt.Printf("⚠️  Failed to cache token: %v\n", err)
			}
			return newToken, nil
		}
		// Otherwise fall back to API key
	}

	// Fall back to API key if refresh fails or unavailable
	if newToken == nil {
		request := &domain.TokenRequest{
			APIKey:    m.apiKey,
			Scope:     []string{scope},
			GrantType: "api_key",
		}

		newToken, err = m.attemptGetTokenWithRetry(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain token: %w", err)
		}

		// Cache the new token
		if err := m.tokenCache.SetToken(scope, newToken); err != nil {
			// Log but don't fail - token is still valid
			fmt.Printf("⚠️  Failed to cache token: %v\n", err)
		}
	}

	return newToken, nil
}

func (m *AutoRefreshAuthManager) attemptRefreshWithRetry(ctx context.Context, refreshToken string) (*domain.AuthToken, error) {
	var lastErr error

	for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(m.config.RetryInterval):
			}
		}

		token, err := m.tokenProvider.RefreshToken(ctx, refreshToken)
		if err == nil {
			return token, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("refresh failed after %d attempts: %w", m.config.MaxRetries, lastErr)
}

func (m *AutoRefreshAuthManager) attemptGetTokenWithRetry(ctx context.Context, request *domain.TokenRequest) (*domain.AuthToken, error) {
	var lastErr error

	for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(m.config.RetryInterval):
			}
		}

		token, err := m.tokenProvider.GetToken(ctx, request)
		if err == nil {
			return token, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("token acquisition failed after %d attempts: %w", m.config.MaxRetries, lastErr)
}
