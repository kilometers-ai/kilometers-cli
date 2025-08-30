package httpports

import (
	"context"

	httpdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/http"
)

type HttpRequester interface {
	Do(ctx context.Context, endpoint httpdomain.BackendEndpoint, req httpdomain.RequestContext) (status int, headers map[string][]string, body []byte, err error)
}

type AuthHeaderProvider interface {
	Headers(ctx context.Context) (map[string]string, error)
}

type RetryPolicy interface {
	ShouldRetry(status int, err error, attempt int) (bool, int)
}
