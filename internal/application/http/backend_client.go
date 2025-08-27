package apphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	httpdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/http"
	httpports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/http"
	httpinfra "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"
)

type BackendClient struct {
	endpoint     httpdomain.BackendEndpoint
	requester    httpports.HttpRequester
	authProvider httpports.AuthHeaderProvider
}

func NewBackendClient(baseURL, userAgent string, timeout time.Duration, auth httpports.AuthHeaderProvider, retry httpports.RetryPolicy) *BackendClient {
	return &BackendClient{
		endpoint:     httpdomain.BackendEndpoint{BaseURL: baseURL, UserAgent: userAgent},
		requester:    httpinfra.NewStdHttpRequester(timeout, retry),
		authProvider: auth,
	}
}

func (c *BackendClient) PostJSON(ctx context.Context, path string, payload interface{}, extraHeaders map[string]string) (int, map[string][]string, []byte, error) {
	body, _ := json.Marshal(payload)
	return c.do(ctx, "POST", path, map[string]string{"Content-Type": "application/json"}, bytes.NewBuffer(body), extraHeaders, nil)
}

func (c *BackendClient) GetJSON(ctx context.Context, path string, extraHeaders map[string]string, query map[string]string) (int, map[string][]string, []byte, error) {
	return c.do(ctx, "GET", path, nil, nil, extraHeaders, query)
}

func (c *BackendClient) GetStream(ctx context.Context, path string, extraHeaders map[string]string, query map[string]string) (int, map[string][]string, []byte, error) {
	return c.do(ctx, "GET", path, nil, nil, extraHeaders, query)
}

func (c *BackendClient) do(ctx context.Context, method, path string, baseHeaders map[string]string, body io.Reader, extraHeaders map[string]string, query map[string]string) (int, map[string][]string, []byte, error) {
	h, _ := c.authProvider.Headers(ctx)
	headers := httpinfra.MergeHeaders(baseHeaders, h)
	headers = httpinfra.MergeHeaders(headers, extraHeaders)

	return c.requester.Do(ctx, c.endpoint, httpdomain.RequestContext{
		Method:  method,
		Path:    path,
		Query:   query,
		Headers: headers,
		Body:    body,
	})
}
