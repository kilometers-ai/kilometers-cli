//go:build debug
// +build debug

package cli

import (
	"context"
	"fmt"
	"time"

	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
	//kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
)

// runInProcessPluginTest - debug version that supports in-process debugging
func runInProcessPluginTest(ctx context.Context, pluginName, version string, verbose, noAuth, pluginDebug bool, config *configpkg.UnifiedConfig) error {
	fmt.Printf("🔬 Starting in-process plugin test for %s\n", pluginName)
	fmt.Printf("✅ Debug build detected - in-process debugging available\n")
	fmt.Printf("🎯 You can set breakpoints in the shared plugin core!\n\n")

	// Create debug plugin manager
	debugManager := plugins.NewDebugPluginManager(true)

	// Set debug environment if requested
	if pluginDebug {
		// This would be picked up by the shared core
		fmt.Printf("🐛 Plugin debug logging enabled\n")
	}

	// Load plugin in-process
	var apiKey string
	var apiEndpoint string

	if !noAuth {
		if config.HasAPIKey() {
			apiKey = config.APIKey
			apiEndpoint = config.APIEndpoint
			if verbose {
				fmt.Printf("🔑 Using API key from configuration\n")
			}
		} else if verbose {
			fmt.Printf("ℹ️ No API key configured - Free tier authentication\n")
		}
	} else if verbose {
		fmt.Printf("🚫 Authentication disabled via --no-auth flag\n")
	}

	fmt.Printf("🚀 Loading plugin in-process...\n")

	// Load the plugin - this calls the shared core directly
	err := debugManager.LoadDebugPlugin(pluginName, apiKey, apiEndpoint)
	if err != nil {
		return fmt.Errorf("failed to load debug plugin: %w", err)
	}

	// Ensure cleanup on exit
	defer func() {
		if verbose {
			fmt.Printf("🛑 Shutting down in-process plugin\n")
		}
		debugManager.Shutdown()
	}()

	fmt.Printf("✅ Plugin loaded in-process successfully\n")
	fmt.Printf("🚨 BREAKPOINT READY: Set breakpoints in shared/api_logger_core.go now!\n\n")

	// Test with mock messages - this is where breakpoints will hit
	fmt.Printf("📨 Testing plugin with mock MCP messages...\n")
	err = testInProcessPluginWithMockMessages(ctx, debugManager, pluginName, verbose)
	if err != nil {
		return fmt.Errorf("in-process plugin testing failed: %w", err)
	}

	fmt.Printf("\n🎉 In-process plugin testing completed successfully!\n")
	fmt.Printf("💡 All your breakpoints should have been triggered!\n")

	return nil
}

// testInProcessPluginWithMockMessages tests in-process plugins with mock messages
func testInProcessPluginWithMockMessages(ctx context.Context, debugManager *plugins.DebugPluginManager, pluginName string, verbose bool) error {
	// Mock MCP messages that will trigger the plugin logic
	mockMessages := []struct {
		name      string
		message   string
		direction string
	}{
		{
			name:      "Initialize Request",
			direction: "inbound",
			message: `{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "roots": {
        "listChanged": true
      }
    },
    "clientInfo": {
      "name": "Claude Desktop",
      "version": "0.7.0"
    }
  }
}`,
		},
		{
			name:      "Tools List Request",
			direction: "inbound",
			message: `{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}`,
		},
		{
			name:      "Tool Call",
			direction: "inbound",
			message: `{
  "jsonrpc": "2.0", 
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "list_files",
    "arguments": {
      "path": "/tmp"
    }
  }
}`,
		},
		{
			name:      "Server Response",
			direction: "outbound",
			message: `{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Files in /tmp: file1.txt, file2.log"
      }
    ]
  }
}`,
		},
	}

	if verbose {
		fmt.Printf("📋 Testing with %d mock MCP messages (in-process):\n", len(mockMessages))
		fmt.Printf("🎯 Each message will trigger your breakpoints in the shared core!\n\n")
	}

	// Process each mock message through the in-process plugin
	for i, mock := range mockMessages {
		if verbose {
			fmt.Printf("   %d. %s (%s)\n", i+1, mock.name, mock.direction)
			fmt.Printf("      🚨 BREAKPOINT OPPORTUNITY: Check your IDE now!\n")
		} else {
			fmt.Printf("📨 Processing: %s\n", mock.name)
		}

		// This call goes directly to the shared core - perfect for breakpoints!
		// Set breakpoints in: kilometers-cli-plugins/shared/api_logger_core.go
		//   - Authenticate() method
		//   - ProcessMessage() method
		//   - sendEventBatch() method
		//   - Any HTTP request logic
		events, err := debugManager.ProcessMessage(ctx, pluginName, []byte(mock.message), mock.direction)
		if err != nil {
			return fmt.Errorf("in-process plugin failed to process message '%s': %w", mock.name, err)
		}

		if verbose {
			fmt.Printf("     ✅ Processed successfully, generated %d events\n", len(events))
			if len(events) > 0 {
				for j, event := range events {
					fmt.Printf("     📝 Event %d: ID=%s, Type=%s\n", j+1, event.ID, event.Type)

					// Show some event data for debugging
					if verbose {
						fmt.Printf("         📊 Data keys: ")
						keys := make([]string, 0, len(event.Data))
						for k := range event.Data {
							keys = append(keys, k)
						}
						fmt.Printf("%v\n", keys)
					}
				}
			}
		}

		// Small delay between messages for realistic timing
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("✅ Successfully processed %d mock MCP messages (in-process)\n", len(mockMessages))

	if verbose {
		fmt.Printf("\n🎯 Debugging Summary:\n")
		fmt.Printf("   - Plugin ran in same process as CLI (debuggable!)\n")
		fmt.Printf("   - All business logic executed in shared core\n")
		fmt.Printf("   - Same code path as production GRPC plugin\n")
		fmt.Printf("   - Full IDE debugging capabilities were available\n")
	}

	return nil
}
