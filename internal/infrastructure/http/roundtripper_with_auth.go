package httpinfra

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kilometers-ai/kilometers-cli/internal/auth"
)

// RoundTripperWithAuth mirrors existing AuthenticatedRoundTripper behavior.
type RoundTripperWithAuth struct {
	base        http.RoundTripper
	authManager auth.AuthManager
	scope       string
}

func NewRoundTripperWithAuth(authManager auth.AuthManager, scope string) *RoundTripperWithAuth {
	return &RoundTripperWithAuth{base: http.DefaultTransport, authManager: authManager, scope: scope}
}

func (t *RoundTripperWithAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	token, err := t.authManager.GetValidToken(ctx, t.scope)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	newReq := req.Clone(ctx)
	newReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))
	resp, err := t.base.RoundTrip(newReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		token, err = t.authManager.ForceRefresh(ctx, t.scope)
		if err != nil {
			return resp, nil
		}
		newReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Token))
		resp.Body.Close()
		return t.base.RoundTrip(newReq)
	}
	return resp, nil
}
