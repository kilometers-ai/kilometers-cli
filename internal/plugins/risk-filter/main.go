package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
)

type RiskFilterPlugin struct {
	apiEndpoint string
	apiKey      string
	httpClient  *http.Client
}

type BackendRiskRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	UserID  string      `json:"user_id"`
	Context interface{} `json:"context"`
}

type BackendRiskResponse struct {
	Action string `json:"action"` // "allow", "block"
	Reason string `json:"reason"`
	Risk   struct {
		Level      string   `json:"level"`      // "low", "medium", "high"
		Score      float64  `json:"score"`      // 0.0 - 1.0
		Categories []string `json:"categories"` // ["data_access", "destructive"]
	} `json:"risk"`
}

func (p *RiskFilterPlugin) OnRequest(ctx context.Context, request *plugins.MCPRequest) (*plugins.MCPResponse, error) {
	// Check if this is a potentially risky operation
	if !p.isRiskyOperation(request) {
		return &plugins.MCPResponse{
			ID:     request.ID,
			Action: "allow",
		}, nil
	}

	// Call backend API for risk assessment
	backendResp, err := p.callBackendAPI(ctx, request)
	if err != nil {
		// If backend is unavailable, default to allow (graceful degradation)
		fmt.Printf("Backend API error, allowing request: %v\n", err)
		return &plugins.MCPResponse{
			ID:     request.ID,
			Action: "allow",
		}, nil
	}

	return &plugins.MCPResponse{
		ID:     request.ID,
		Action: backendResp.Action,
		Reason: backendResp.Reason,
	}, nil
}

func (p *RiskFilterPlugin) OnResponse(ctx context.Context, response *plugins.MCPResponse) error {
	// Could log risk decisions or update user context
	return nil
}

func (p *RiskFilterPlugin) GetName() string {
	return "risk-filter"
}

func (p *RiskFilterPlugin) RequiresPremium() bool {
	return true // Premium tier plugin
}

func (p *RiskFilterPlugin) Shutdown() {
	// No cleanup needed for this plugin
}

func (p *RiskFilterPlugin) isRiskyOperation(request *plugins.MCPRequest) bool {
	// Check for potentially destructive operations
	riskyMethods := []string{
		"sql/execute",
		"filesystem/write",
		"filesystem/delete",
		"system/exec",
	}

	for _, risky := range riskyMethods {
		if strings.Contains(request.Method, risky) {
			return true
		}
	}

	// Check for risky SQL keywords in params
	if params, ok := request.Params.(map[string]interface{}); ok {
		if query, ok := params["query"].(string); ok {
			riskySQL := []string{"DELETE", "DROP", "TRUNCATE", "UPDATE"}
			upperQuery := strings.ToUpper(query)
			for _, keyword := range riskySQL {
				if strings.Contains(upperQuery, keyword) {
					return true
				}
			}
		}
	}

	return false
}

func (p *RiskFilterPlugin) callBackendAPI(ctx context.Context, request *plugins.MCPRequest) (*BackendRiskResponse, error) {
	backendReq := BackendRiskRequest{
		Method: request.Method,
		Params: request.Params,
		UserID: "user_123", // Would come from auth context
		Context: map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"source":    "km-cli",
		},
	}

	jsonData, err := json.Marshal(backendReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/risk/evaluate", p.apiEndpoint),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend API error: %d", resp.StatusCode)
	}

	var backendResp BackendRiskResponse
	if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
		return nil, err
	}

	return &backendResp, nil
}

func main() {
	apiEndpoint := os.Getenv("KM_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = "https://api.kilometers.ai"
	}

	apiKey := os.Getenv("KM_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "KM_API_KEY environment variable required\n")
		os.Exit(1)
	}

	riskFilter := &RiskFilterPlugin{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"mcp": &plugins.KmPluginImpl{Impl: riskFilter},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
