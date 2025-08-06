package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock token provider
type MockTokenProvider struct {
	mock.Mock
}

func (m *MockTokenProvider) GetToken(ctx context.Context, request *domain.TokenRequest) (*domain.AuthToken, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthToken), args.Error(1)
}

func (m *MockTokenProvider) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthToken, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthToken), args.Error(1)
}

func (m *MockTokenProvider) ValidateToken(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

// Test token cache implementation
type testMemoryTokenCache struct {
	tokens map[string]*domain.AuthToken
	mu     sync.RWMutex
}

func newTestMemoryTokenCache() *testMemoryTokenCache {
	return &testMemoryTokenCache{
		tokens: make(map[string]*domain.AuthToken),
	}
}

func (c *testMemoryTokenCache) GetToken(scope string) (*domain.AuthToken, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	token, exists := c.tokens[scope]
	if !exists {
		return nil, nil
	}

	// Return the token even if expired - the caller will check expiration
	return token, nil
}

func (c *testMemoryTokenCache) SetToken(scope string, token *domain.AuthToken) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[scope] = token
	return nil
}

func (c *testMemoryTokenCache) RemoveToken(scope string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, scope)
	return nil
}

func (c *testMemoryTokenCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens = make(map[string]*domain.AuthToken)
	return nil
}

// Helper functions
func createTestToken(expiresIn time.Duration) *domain.AuthToken {
	now := time.Now()
	return &domain.AuthToken{
		Token:        "test-token",
		Type:         "Bearer",
		ExpiresAt:    now.Add(expiresIn),
		IssuedAt:     now,
		RefreshToken: "test-refresh-token",
	}
}

func TestAutoRefreshAuthManager_GetValidToken(t *testing.T) {
	tests := []struct {
		name               string
		cachedToken        *domain.AuthToken
		providerResponse   *domain.AuthToken
		providerError      error
		expectedToken      string
		expectProviderCall bool
		expectError        bool
	}{
		{
			name:               "valid cached token",
			cachedToken:        createTestToken(10 * time.Minute), // Valid for 10 minutes
			expectedToken:      "test-token",
			expectProviderCall: false,
		},
		{
			name:               "expired cached token with refresh",
			cachedToken:        createTestToken(-1 * time.Minute), // Expired 1 minute ago
			providerResponse:   createTestToken(30 * time.Minute),
			expectedToken:      "test-token",
			expectProviderCall: true,
		},
		{
			name:               "no cached token",
			cachedToken:        nil,
			providerResponse:   createTestToken(30 * time.Minute),
			expectedToken:      "test-token",
			expectProviderCall: true,
		},
		{
			name:               "token needs refresh",
			cachedToken:        createTestToken(3 * time.Minute), // Expires in 3 minutes (< 5 minute threshold)
			providerResponse:   createTestToken(30 * time.Minute),
			expectedToken:      "test-token",
			expectProviderCall: true,
		},
		{
			name:               "refresh fails",
			cachedToken:        createTestToken(-1 * time.Minute),
			providerError:      errors.New("refresh failed"),
			expectProviderCall: true,
			expectError:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockProvider := new(MockTokenProvider)
			cache := newTestMemoryTokenCache()

			config := &AutoRefreshConfig{
				RefreshAhead:  5 * time.Minute,
				RetryInterval: 10 * time.Millisecond,
				MaxRetries:    1,
			}

			manager := NewAutoRefreshAuthManager(mockProvider, cache, "test-api-key", config)

			// Set cached token if provided
			if tc.cachedToken != nil {
				cache.SetToken("test-scope", tc.cachedToken)
			}

			// Setup mock expectations
			if tc.expectProviderCall {
				if tc.cachedToken != nil && tc.cachedToken.RefreshToken != "" {
					mockProvider.On("RefreshToken", mock.Anything, tc.cachedToken.RefreshToken).
						Return(tc.providerResponse, tc.providerError).Once()
				}

				// If refresh fails or no refresh token, expect API key call
				if tc.providerError != nil || tc.cachedToken == nil || tc.cachedToken.RefreshToken == "" {
					// For refresh failures, GetToken should succeed unless we want total failure
					getTokenError := tc.providerError
					getTokenResponse := tc.providerResponse
					if tc.cachedToken != nil && tc.cachedToken.RefreshToken != "" && tc.providerError != nil {
						// Refresh failed, but GetToken might succeed
						if tc.expectError {
							// Both should fail for total failure
							getTokenError = tc.providerError
						} else {
							// Refresh failed but GetToken succeeds
							getTokenError = nil
						}
					}
					mockProvider.On("GetToken", mock.Anything, mock.MatchedBy(func(req *domain.TokenRequest) bool {
						return req.APIKey == "test-api-key" && req.GrantType == "api_key"
					})).Return(getTokenResponse, getTokenError).Once()
				}
			}

			// Execute
			ctx := context.Background()
			token, err := manager.GetValidToken(ctx, "test-scope")

			// Verify
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tc.expectedToken, token.Token)
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestAutoRefreshAuthManager_ForceRefresh(t *testing.T) {
	// Setup
	mockProvider := new(MockTokenProvider)
	cache := newTestMemoryTokenCache()

	config := &AutoRefreshConfig{
		RefreshAhead:  5 * time.Minute,
		RetryInterval: 10 * time.Millisecond,
		MaxRetries:    1,
	}

	manager := NewAutoRefreshAuthManager(mockProvider, cache, "test-api-key", config)

	// Set a valid cached token
	validToken := createTestToken(30 * time.Minute)
	cache.SetToken("test-scope", validToken)

	// Setup mock - expect API key call (force refresh ignores cached token)
	newToken := createTestToken(60 * time.Minute)
	newToken.Token = "new-token"

	mockProvider.On("GetToken", mock.Anything, mock.MatchedBy(func(req *domain.TokenRequest) bool {
		return req.APIKey == "test-api-key" && req.GrantType == "api_key"
	})).Return(newToken, nil).Once()

	// Execute
	ctx := context.Background()
	token, err := manager.ForceRefresh(ctx, "test-scope")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "new-token", token.Token)

	// Verify cache was updated
	cachedToken, _ := cache.GetToken("test-scope")
	assert.Equal(t, "new-token", cachedToken.Token)

	mockProvider.AssertExpectations(t)
}

func TestAutoRefreshAuthManager_AutoRefreshLoop(t *testing.T) {
	// This test verifies the background refresh functionality

	// Setup
	mockProvider := new(MockTokenProvider)
	cache := newTestMemoryTokenCache()

	config := &AutoRefreshConfig{
		RefreshAhead:  5 * time.Minute,
		RetryInterval: 10 * time.Millisecond,
		MaxRetries:    1,
		CheckInterval: 50 * time.Millisecond, // Fast check for testing
	}

	manager := NewAutoRefreshAuthManager(mockProvider, cache, "test-api-key", config)

	// Set a token that will need refresh soon
	almostExpiredToken := createTestToken(4 * time.Minute)
	cache.SetToken("default", almostExpiredToken)

	// Setup mock - expect refresh to be called
	newToken := createTestToken(30 * time.Minute)
	newToken.Token = "refreshed-token"

	mockProvider.On("RefreshToken", mock.Anything, almostExpiredToken.RefreshToken).
		Return(newToken, nil).Once()

	// Start auto-refresh
	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	// Wait for refresh to happen
	time.Sleep(100 * time.Millisecond)

	// Stop auto-refresh
	cancel()
	manager.Stop()

	// Verify token was refreshed
	cachedToken, _ := cache.GetToken("default")
	assert.Equal(t, "refreshed-token", cachedToken.Token)

	mockProvider.AssertExpectations(t)
}

func TestAutoRefreshAuthManager_ConcurrentRefresh(t *testing.T) {
	// Test that concurrent refresh requests are handled properly

	// Setup
	mockProvider := new(MockTokenProvider)
	cache := newTestMemoryTokenCache()

	config := DefaultAutoRefreshConfig()
	manager := NewAutoRefreshAuthManager(mockProvider, cache, "test-api-key", config)

	// Set expired token
	expiredToken := createTestToken(-1 * time.Minute)
	cache.SetToken("test-scope", expiredToken)

	// Setup mock - should only be called once despite concurrent requests
	newToken := createTestToken(30 * time.Minute)

	// Add delay to simulate slow refresh
	mockProvider.On("RefreshToken", mock.Anything, expiredToken.RefreshToken).
		Run(func(args mock.Arguments) {
			time.Sleep(50 * time.Millisecond)
		}).
		Return(newToken, nil).Once()

	// Execute concurrent requests
	ctx := context.Background()
	results := make(chan *domain.AuthToken, 3)
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			token, err := manager.GetValidToken(ctx, "test-scope")
			if err != nil {
				errors <- err
			} else {
				results <- token
			}
		}()
	}

	// Wait for all requests
	for i := 0; i < 3; i++ {
		select {
		case token := <-results:
			assert.NotNil(t, token)
			assert.Equal(t, "test-token", token.Token)
		case err := <-errors:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for results")
		}
	}

	// Verify refresh was only called once
	mockProvider.AssertExpectations(t)
}

func TestAutoRefreshAuthManager_RetryLogic(t *testing.T) {
	// Test that retry logic works correctly

	// Setup
	mockProvider := new(MockTokenProvider)
	cache := newTestMemoryTokenCache()

	config := &AutoRefreshConfig{
		RefreshAhead:  5 * time.Minute,
		RetryInterval: 10 * time.Millisecond,
		MaxRetries:    3,
	}

	manager := NewAutoRefreshAuthManager(mockProvider, cache, "test-api-key", config)

	// Setup mock - fail twice, then succeed
	mockProvider.On("GetToken", mock.Anything, mock.Anything).
		Return(nil, errors.New("temporary error")).Twice()

	newToken := createTestToken(30 * time.Minute)
	mockProvider.On("GetToken", mock.Anything, mock.Anything).
		Return(newToken, nil).Once()

	// Execute
	ctx := context.Background()
	token, err := manager.GetValidToken(ctx, "test-scope")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test-token", token.Token)

	mockProvider.AssertExpectations(t)
}

func TestAuthToken_ShouldRefresh(t *testing.T) {
	tests := []struct {
		name         string
		expiresIn    time.Duration
		refreshAhead time.Duration
		expected     bool
	}{
		{
			name:         "token expired",
			expiresIn:    -1 * time.Minute,
			refreshAhead: 5 * time.Minute,
			expected:     true,
		},
		{
			name:         "token expires soon",
			expiresIn:    3 * time.Minute,
			refreshAhead: 5 * time.Minute,
			expected:     true,
		},
		{
			name:         "token still valid",
			expiresIn:    10 * time.Minute,
			refreshAhead: 5 * time.Minute,
			expected:     false,
		},
		{
			name:         "token at refresh threshold",
			expiresIn:    5 * time.Minute,
			refreshAhead: 5 * time.Minute,
			expected:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := createTestToken(tc.expiresIn)
			result := token.ShouldRefresh(tc.refreshAhead)
			assert.Equal(t, tc.expected, result)
		})
	}
}
