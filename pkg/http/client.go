package http

// Re-export HTTP client functionality that plugins need
// This allows external modules to use HTTP client without accessing internal packages

import "github.com/kilometers-ai/kilometers-cli/internal/infrastructure/http"

// Re-export HTTP client types
type ApiClient = http.ApiClient
type McpEventDto = http.McpEventDto
type BatchEventDto = http.BatchEventDto
type BatchRequest = http.BatchRequest

// NewApiClient creates a new HTTP API client instance
func NewApiClient() *ApiClient {
	return http.NewApiClient()
}