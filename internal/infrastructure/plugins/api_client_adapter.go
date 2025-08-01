package plugins

import (
	"context"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain"
	"github.com/kilometers-ai/kilometers-cli/internal/core/ports"
	httpClient "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
)

// apiClientAdapter adapts the http.ApiClient to the ports.APIClient interface
type apiClientAdapter struct {
	client *httpClient.ApiClient
}

// NewAPIClientAdapter creates a new API client adapter
func NewAPIClientAdapter() ports.APIClient {
	return &apiClientAdapter{
		client: httpClient.NewApiClient(),
	}
}

// GetUserFeatures retrieves the user's subscription features
func (a *apiClientAdapter) GetUserFeatures(ctx context.Context) (*ports.UserFeaturesResponse, error) {
	if a.client == nil {
		// No client configured, return free tier
		return &ports.UserFeaturesResponse{
			Tier:     domain.TierFree,
			Features: domain.DefaultFreeFeatures(),
		}, nil
	}

	resp, err := a.client.GetUserFeatures(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to domain types
	return &ports.UserFeaturesResponse{
		Tier:      domain.SubscriptionTier(resp.Tier),
		Features:  resp.Features,
		ExpiresAt: resp.ExpiresAt,
	}, nil
}

// SendBatchEvents sends a batch of events to the API
func (a *apiClientAdapter) SendBatchEvents(ctx context.Context, batch interface{}) error {
	if a.client == nil {
		return nil // No client, no-op
	}

	// Type assert to the expected batch type
	batchRequest, ok := batch.(httpClient.BatchRequest)
	if !ok {
		return nil // Wrong type, skip
	}

	return a.client.SendBatchEvents(ctx, batchRequest)
}
