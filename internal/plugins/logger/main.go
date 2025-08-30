package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
)

type LoggerPlugin struct {
	logFile *os.File
}

func (p *LoggerPlugin) OnRequest(ctx context.Context, request *plugins.MCPRequest) (*plugins.MCPResponse, error) {
	// Log all requests
	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"type":      "request",
		"method":    request.Method,
		"id":        request.ID,
		"params":    request.Params,
	}

	if jsonData, err := json.Marshal(logEntry); err == nil {
		fmt.Fprintf(p.logFile, "%s\n", jsonData)
		p.logFile.Sync()
	}

	// Always allow requests (this is just a logger)
	return &plugins.MCPResponse{
		ID:     request.ID,
		Action: "allow",
	}, nil
}

func (p *LoggerPlugin) OnResponse(ctx context.Context, response *plugins.MCPResponse) error {
	// Log all responses
	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"type":      "response",
		"id":        response.ID,
		"result":    response.Result,
		"error":     response.Error,
	}

	if jsonData, err := json.Marshal(logEntry); err == nil {
		fmt.Fprintf(p.logFile, "%s\n", jsonData)
		p.logFile.Sync()
	}

	return nil
}

func (p *LoggerPlugin) GetName() string {
	return "km-plugin-logger"
}

func (p *LoggerPlugin) RequiresPremium() bool {
	return false
}

func (p *LoggerPlugin) Shutdown() {
	if p.logFile != nil {
		p.logFile.Close()
	}
}

func main() {
	// Open log file
	logFile, err := os.OpenFile("km-mcp-log.jsonl", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	logger := &LoggerPlugin{logFile: logFile}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"mcp": &plugins.KmPluginImpl{Impl: logger},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
