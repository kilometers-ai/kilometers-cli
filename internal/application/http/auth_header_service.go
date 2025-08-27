package apphttp

import (
	"context"
)

// AuthHeaderService emits headers according to existing behavior:
// - For plugin endpoints, X-API-Key is used (caller provides apiKey)
// - Authorization could be added later if token flow is used elsewhere
type AuthHeaderService struct {
	apiKey    string
	userAgent string
}

func NewAuthHeaderService(apiKey, userAgent string) *AuthHeaderService {
	return &AuthHeaderService{apiKey: apiKey, userAgent: userAgent}
}

func (s *AuthHeaderService) Headers(ctx context.Context) (map[string]string, error) {
	h := map[string]string{}
	if s.userAgent != "" {
		h["User-Agent"] = s.userAgent
	}
	if s.apiKey != "" {
		h["X-API-Key"] = s.apiKey
	}
	return h, nil
}
