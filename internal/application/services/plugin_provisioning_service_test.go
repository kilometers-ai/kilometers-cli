package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations

type MockProvisioningService struct {
	mock.Mock
}

func (m *MockProvisioningService) ProvisionPlugins(ctx context.Context, apiKey string) (*domain.PluginProvisionResponse, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PluginProvisionResponse), args.Error(1)
}

func (m *MockProvisioningService) GetSubscriptionStatus(ctx context.Context, apiKey string) (string, error) {
	args := m.Called(ctx, apiKey)
	return args.String(0), args.Error(1)
}

type MockDownloader struct {
	mock.Mock
}

func (m *MockDownloader) DownloadPlugin(ctx context.Context, plugin domain.ProvisionedPlugin) (io.ReadCloser, error) {
	args := m.Called(ctx, plugin)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDownloader) VerifySignature(pluginData []byte, signature string) error {
	args := m.Called(pluginData, signature)
	return args.Error(0)
}

type MockInstaller struct {
	mock.Mock
}

func (m *MockInstaller) InstallPlugin(ctx context.Context, pluginData io.Reader, plugin domain.ProvisionedPlugin) error {
	args := m.Called(ctx, pluginData, plugin)
	return args.Error(0)
}

func (m *MockInstaller) UninstallPlugin(ctx context.Context, pluginName string) error {
	args := m.Called(ctx, pluginName)
	return args.Error(0)
}

func (m *MockInstaller) GetInstalledPlugins(ctx context.Context) ([]domain.InstalledPlugin, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.InstalledPlugin), args.Error(1)
}

type MockRegistryStore struct {
	mock.Mock
	registry *domain.PluginRegistry
}

func (m *MockRegistryStore) SaveRegistry(registry *domain.PluginRegistry) error {
	args := m.Called(registry)
	m.registry = registry
	return args.Error(0)
}

func (m *MockRegistryStore) LoadRegistry() (*domain.PluginRegistry, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PluginRegistry), args.Error(1)
}

func (m *MockRegistryStore) UpdatePlugin(plugin domain.InstalledPlugin) error {
	args := m.Called(plugin)
	return args.Error(0)
}

// Tests

func TestAutoProvisionPlugins(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		config         *domain.UnifiedConfig
		setupMocks     func(*MockProvisioningService, *MockDownloader, *MockInstaller, *MockRegistryStore)
		expectedError  string
		expectedOutput []string
	}{
		{
			name: "successful provisioning",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				// Mock provisioning response
				ps.On("ProvisionPlugins", ctx, "test-api-key").Return(&domain.PluginProvisionResponse{
					CustomerID:       "cust_123",
					SubscriptionTier: "Pro",
					Plugins: []domain.ProvisionedPlugin{
						{
							Name:         "console-logger",
							Version:      "1.0.0",
							DownloadURL:  "http://test.com/console-logger.kmpkg",
							Signature:    "sig1",
							ExpiresAt:    time.Now().Add(1 * time.Hour),
							RequiredTier: "Free",
						},
						{
							Name:         "api-logger",
							Version:      "2.0.0",
							DownloadURL:  "http://test.com/api-logger.kmpkg",
							Signature:    "sig2",
							ExpiresAt:    time.Now().Add(1 * time.Hour),
							RequiredTier: "Pro",
						},
					},
				}, nil)

				// Mock registry operations
				r.On("LoadRegistry").Return(&domain.PluginRegistry{
					CustomerID:  "cust_123",
					CurrentTier: "Pro",
					Plugins:     make(map[string]domain.InstalledPlugin),
				}, nil)
				r.On("SaveRegistry", mock.Anything).Return(nil)

				// Mock download and install for each plugin
				for range []string{"console-logger", "api-logger"} {
					reader := io.NopCloser(strings.NewReader("plugin-data"))
					d.On("DownloadPlugin", ctx, mock.Anything).Return(reader, nil).Once()
					i.On("InstallPlugin", ctx, mock.Anything, mock.Anything).Return(nil).Once()
				}
			},
			expectedError: "",
			expectedOutput: []string{
				"Installed console-logger plugin",
				"Installed api-logger plugin",
				"Successfully installed 2/2 plugins",
			},
		},
		{
			name: "no api key",
			config: &domain.UnifiedConfig{
				APIKey:      "",
				APIEndpoint: "http://test.com",
			},
			setupMocks:    func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {},
			expectedError: "API key is required for plugin provisioning",
		},
		{
			name: "provisioning API error",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				ps.On("ProvisionPlugins", ctx, "test-api-key").Return(nil, fmt.Errorf("API error"))
			},
			expectedError: "failed to provision plugins: API error",
		},
		{
			name: "partial installation failure",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				ps.On("ProvisionPlugins", ctx, "test-api-key").Return(&domain.PluginProvisionResponse{
					CustomerID:       "cust_123",
					SubscriptionTier: "Pro",
					Plugins: []domain.ProvisionedPlugin{
						{
							Name:         "console-logger",
							Version:      "1.0.0",
							DownloadURL:  "http://test.com/console-logger.kmpkg",
							RequiredTier: "Free",
						},
						{
							Name:         "api-logger",
							Version:      "2.0.0",
							DownloadURL:  "http://test.com/api-logger.kmpkg",
							RequiredTier: "Pro",
						},
					},
				}, nil)

				r.On("LoadRegistry").Return(&domain.PluginRegistry{
					CustomerID:  "cust_123",
					CurrentTier: "Pro",
					Plugins:     make(map[string]domain.InstalledPlugin),
				}, nil)
				r.On("SaveRegistry", mock.Anything).Return(nil)

				// First plugin succeeds
				reader := io.NopCloser(strings.NewReader("plugin-data"))
				d.On("DownloadPlugin", ctx, mock.Anything).Return(reader, nil).Once()
				i.On("InstallPlugin", ctx, mock.Anything, mock.Anything).Return(nil).Once()

				// Second plugin fails
				d.On("DownloadPlugin", ctx, mock.Anything).Return(nil, fmt.Errorf("download failed")).Once()
			},
			expectedError: "",
			expectedOutput: []string{
				"Installed console-logger plugin",
				"Failed to install api-logger",
				"Successfully installed 1/2 plugins",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockPS := new(MockProvisioningService)
			mockD := new(MockDownloader)
			mockI := new(MockInstaller)
			mockR := new(MockRegistryStore)

			// Setup mocks
			tc.setupMocks(mockPS, mockD, mockI, mockR)

			// Create manager
			manager := NewPluginProvisioningManager(mockPS, mockD, mockI, mockR)

			// Execute
			err := manager.AutoProvisionPlugins(ctx, tc.config)

			// Verify error
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockPS.AssertExpectations(t)
			mockD.AssertExpectations(t)
			mockI.AssertExpectations(t)
			mockR.AssertExpectations(t)
		})
	}
}

func TestRefreshPlugins(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		config        *domain.UnifiedConfig
		currentTier   string
		newTier       string
		setupMocks    func(*MockProvisioningService, *MockDownloader, *MockInstaller, *MockRegistryStore)
		expectedError string
	}{
		{
			name: "tier upgrade",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			currentTier: "Free",
			newTier:     "Pro",
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				// Mock subscription check
				ps.On("GetSubscriptionStatus", ctx, "test-api-key").Return("Pro", nil)

				// Mock registry with Free tier
				r.On("LoadRegistry").Return(&domain.PluginRegistry{
					CustomerID:  "cust_123",
					CurrentTier: "Free",
					Plugins: map[string]domain.InstalledPlugin{
						"console-logger": {
							Name:         "console-logger",
							RequiredTier: "Free",
							Enabled:      true,
						},
					},
				}, nil)

				// Expect registry save with updated tier
				r.On("SaveRegistry", mock.Anything).Return(nil)

				// Expect auto-provision to be called for new plugins
				ps.On("ProvisionPlugins", ctx, "test-api-key").Return(&domain.PluginProvisionResponse{
					CustomerID:       "cust_123",
					SubscriptionTier: "Pro",
					Plugins: []domain.ProvisionedPlugin{
						{
							Name:         "api-logger",
							Version:      "2.0.0",
							DownloadURL:  "http://test.com/api-logger.kmpkg",
							RequiredTier: "Pro",
						},
					},
				}, nil)

				// Mock download and install for new Pro plugin
				reader := io.NopCloser(strings.NewReader("plugin-data"))
				d.On("DownloadPlugin", ctx, mock.Anything).Return(reader, nil)
				i.On("InstallPlugin", ctx, mock.Anything, mock.Anything).Return(nil)

				// Second save after installing new plugins
				r.On("SaveRegistry", mock.Anything).Return(nil)
			},
		},
		{
			name: "tier downgrade",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			currentTier: "Pro",
			newTier:     "Free",
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				// Mock subscription check
				ps.On("GetSubscriptionStatus", ctx, "test-api-key").Return("Free", nil)

				// Mock registry with Pro tier
				r.On("LoadRegistry").Return(&domain.PluginRegistry{
					CustomerID:  "cust_123",
					CurrentTier: "Pro",
					Plugins: map[string]domain.InstalledPlugin{
						"console-logger": {
							Name:         "console-logger",
							RequiredTier: "Free",
							Enabled:      true,
						},
						"api-logger": {
							Name:         "api-logger",
							RequiredTier: "Pro",
							Enabled:      true,
						},
					},
				}, nil)

				// Expect registry save with disabled Pro plugins
				r.On("SaveRegistry", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					registry := args.Get(0).(*domain.PluginRegistry)
					assert.Equal(t, "Free", registry.CurrentTier)
					assert.False(t, registry.Plugins["api-logger"].Enabled)
					assert.True(t, registry.Plugins["console-logger"].Enabled)
				})
			},
		},
		{
			name: "no tier change no refresh needed",
			config: &domain.UnifiedConfig{
				APIKey:      "test-api-key",
				APIEndpoint: "http://test.com",
			},
			currentTier: "Pro",
			newTier:     "Pro",
			setupMocks: func(ps *MockProvisioningService, d *MockDownloader, i *MockInstaller, r *MockRegistryStore) {
				// Mock subscription check
				ps.On("GetSubscriptionStatus", ctx, "test-api-key").Return("Pro", nil)

				// Mock registry - recent update
				r.On("LoadRegistry").Return(&domain.PluginRegistry{
					CustomerID:  "cust_123",
					CurrentTier: "Pro",
					LastUpdated: time.Now().Add(-1 * time.Hour), // Recently updated
					Plugins:     map[string]domain.InstalledPlugin{},
				}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockPS := new(MockProvisioningService)
			mockD := new(MockDownloader)
			mockI := new(MockInstaller)
			mockR := new(MockRegistryStore)

			// Setup mocks
			tc.setupMocks(mockPS, mockD, mockI, mockR)

			// Create manager
			manager := NewPluginProvisioningManager(mockPS, mockD, mockI, mockR)

			// Execute
			err := manager.RefreshPlugins(ctx, tc.config)

			// Verify error
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockPS.AssertExpectations(t)
			mockD.AssertExpectations(t)
			mockI.AssertExpectations(t)
			mockR.AssertExpectations(t)
		})
	}
}

func TestIsTierSufficient(t *testing.T) {
	tests := []struct {
		currentTier  string
		requiredTier string
		expected     bool
	}{
		{"Free", "Free", true},
		{"Pro", "Free", true},
		{"Enterprise", "Free", true},
		{"Free", "Pro", false},
		{"Pro", "Pro", true},
		{"Enterprise", "Pro", true},
		{"Free", "Enterprise", false},
		{"Pro", "Enterprise", false},
		{"Enterprise", "Enterprise", true},
		{"Invalid", "Free", false},
		{"Pro", "Invalid", false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s >= %s", tc.currentTier, tc.requiredTier), func(t *testing.T) {
			result := domain.IsTierSufficient(tc.currentTier, tc.requiredTier)
			assert.Equal(t, tc.expected, result)
		})
	}
}
