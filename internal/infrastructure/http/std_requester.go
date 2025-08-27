package httpinfra

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	httpdomain "github.com/kilometers-ai/kilometers-cli/internal/core/domain/http"
	httpports "github.com/kilometers-ai/kilometers-cli/internal/core/ports/http"
)

type StdHttpRequester struct {
	client *http.Client
	retry  httpports.RetryPolicy
}

func NewStdHttpRequester(timeout time.Duration, retry httpports.RetryPolicy) *StdHttpRequester {
	return &StdHttpRequester{client: &http.Client{Timeout: timeout}, retry: retry}
}

func (r *StdHttpRequester) Do(ctx context.Context, endpoint httpdomain.BackendEndpoint, req httpdomain.RequestContext) (int, map[string][]string, []byte, error) {
	fullURL, err := joinURL(endpoint.BaseURL, req.Path, req.Query)
	if err != nil {
		return 0, nil, nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, req.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if endpoint.UserAgent != "" {
		httpReq.Header.Set("User-Agent", endpoint.UserAgent)
	}

	attempt := 0
	for {
		resp, err := r.client.Do(httpReq)
		if err != nil {
			if r.retry != nil {
				retry, backoff := r.retry.ShouldRetry(0, err, attempt)
				if retry {
					time.Sleep(time.Duration(backoff) * time.Millisecond)
					attempt++
					continue
				}
			}
			return 0, nil, nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if r.retry != nil {
			retry, backoff := r.retry.ShouldRetry(resp.StatusCode, nil, attempt)
			if retry {
				time.Sleep(time.Duration(backoff) * time.Millisecond)
				attempt++
				// recreate request body if needed
				if httpReq.Body != nil {
					httpReq.Body = io.NopCloser(bytes.NewBuffer(body))
				}
				continue
			}
		}
		return resp.StatusCode, resp.Header, body, nil
	}
}

func joinURL(base, p string, q map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = joinPath(u.Path, p)
	if len(q) > 0 {
		vals := u.Query()
		for k, v := range q {
			vals.Set(k, v)
		}
		u.RawQuery = vals.Encode()
	}
	return u.String(), nil
}

func joinPath(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a[len(a)-1] == '/' {
		a = a[:len(a)-1]
	}
	if b[0] != '/' {
		b = "/" + b
	}
	return a + b
}

var _ httpports.HttpRequester = (*StdHttpRequester)(nil)
