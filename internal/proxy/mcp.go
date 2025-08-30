package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
)

type MCPProxy struct {
	pluginManager *plugins.PluginManager
}

func NewMCPProxy(pluginManager *plugins.PluginManager) (*MCPProxy, error) {
	return &MCPProxy{
		pluginManager: pluginManager,
	}, nil
}

func (p *MCPProxy) Start(ctx context.Context, mcpProcess *exec.Cmd) error {
	// Create pipes for communication
	stdinPipe, err := mcpProcess.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := mcpProcess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start MCP server
	if err := mcpProcess.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	fmt.Printf("DEBUG: MCP server process started with PID: %d\n", mcpProcess.Process.Pid)

	// Handle requests (stdin -> MCP server)
	go p.handleRequests(ctx, stdinPipe)

	// Handle responses (MCP server -> stdout)
	go p.handleResponses(ctx, stdoutPipe)

	// Wait for context cancellation or process exit
	go func() {
		err := mcpProcess.Wait()
		if err != nil {
			fmt.Printf("DEBUG: MCP process exited with error: %v\n", err)
		} else {
			fmt.Printf("DEBUG: MCP process exited successfully\n")
		}
	}()

	select {
	case <-ctx.Done():
		mcpProcess.Process.Kill()
		return ctx.Err()
	}
}

func (p *MCPProxy) handleRequests(ctx context.Context, stdinPipe io.WriteCloser) {
	defer stdinPipe.Close()

	scanner := bufio.NewScanner(os.Stdin)
	// Set up larger buffer like the working implementation
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		// Parse JSON-RPC request
		var request plugins.MCPRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			// Not a JSON-RPC request, pass through
			fmt.Fprintln(stdinPipe, line)
			continue
		}

		// Process through plugins
		response, err := p.pluginManager.ProcessRequest(ctx, &request)
		if err != nil {
			fmt.Printf("Plugin processing error: %v\n", err)
			continue
		}

		// Debug: Print plugin decision
		fmt.Printf("DEBUG: Plugin response - Action: %s, Reason: %s\n", response.Action, response.Reason)

		// Handle plugin decision
		switch response.Action {
		case "block":
			// Send error response back to client
			errorResponse := map[string]interface{}{
				"id": request.ID,
				"error": map[string]interface{}{
					"code":    -32603,
					"message": fmt.Sprintf("Request blocked: %s", response.Reason),
				},
			}
			errorJSON, _ := json.Marshal(errorResponse)
			fmt.Println(string(errorJSON))

		case "allow", "modify":
			// Forward (possibly modified) request to MCP server
			requestJSON, _ := json.Marshal(request)
			fmt.Printf("DEBUG: Forwarding to MCP server: %s\n", string(requestJSON))
			// Write with newline like the old implementation
			if _, err := stdinPipe.Write(append(requestJSON, '\n')); err != nil {
				fmt.Printf("DEBUG: Error writing to MCP server: %v\n", err)
			}
		}
	}
}

func (p *MCPProxy) handleResponses(ctx context.Context, stdoutPipe io.ReadCloser) {
	defer stdoutPipe.Close()

	scanner := bufio.NewScanner(stdoutPipe)
	// Set up larger buffer like the working implementation
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	fmt.Printf("DEBUG: Starting to read from MCP server stdout...\n")
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		fmt.Printf("DEBUG: Received from MCP server: %s\n", line)

		// Parse JSON-RPC response
		var response plugins.MCPResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			// Not a JSON-RPC response, pass through
			fmt.Printf("DEBUG: Non-JSON response, passing through: %s\n", line)
			fmt.Printf("NON-JSON-RESPONSE: %s\n", line)
			continue
		}

		// Process through plugins
		p.pluginManager.ProcessResponse(ctx, &response)

		// Forward response to client
		fmt.Printf("DEBUG: Forwarding response to client: %s\n", line)
		fmt.Printf("RESPONSE: %s\n", line)
	}
}
