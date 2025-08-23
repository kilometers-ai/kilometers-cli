package cli

import (
	"context"
	"fmt"
	"time"

	configpkg "github.com/kilometers-ai/kilometers-cli/internal/config"
	"github.com/kilometers-ai/kilometers-cli/internal/plugins"
	kmsdk "github.com/kilometers-ai/kilometers-plugins-sdk"
	"github.com/spf13/cobra"
)

// newDevCommand creates the dev command for development utilities
func newDevCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Development utilities for Kilometers CLI",
		Long: `Development utilities for testing and debugging Kilometers CLI functionality.

These commands are designed for development and testing purposes and should not be 
used in production environments.`,
		Example: `  # Test a plugin with mock MCP communication
  km dev plugin-test console-logger

  # Test a plugin with verbose debugging
  km dev plugin-test console-logger --verbose`,
	}

	// Add dev subcommands
	cmd.AddCommand(newDevPluginTestCommand(version))

	return cmd
}

// newDevPluginTestCommand creates the plugin-test subcommand
func newDevPluginTestCommand(version string) *cobra.Command {
	var verbose bool
	var noAuth bool
	var pluginDebug bool
	var timeout string

	cmd := &cobra.Command{
		Use:   "plugin-test <plugin-name>",
		Short: "Test plugin integration with mock MCP communication",
		Long: `Test plugin integration using mock MCP client-server communication.

This command loads the specified plugin and simulates JSON-RPC message flow 
without requiring an actual MCP server. This is useful for:

- Testing plugin loading and authentication
- Debugging plugin message handling
- Validating plugin behavior in isolation
- Quick development iteration

The plugin must be installed in ~/.km/plugins directory.`,
		Args:    cobra.ExactArgs(1),
		Example: `  # Test console-logger plugin
  km dev plugin-test console-logger

  # Test with verbose output
  km dev plugin-test console-logger --verbose

  # Test without authentication (for development)
  km dev plugin-test console-logger --no-auth

  # Test with plugin-specific debugging
  km dev plugin-test console-logger --plugin-debug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]
			return runDevPluginTest(cmd, pluginName, version, verbose, noAuth, pluginDebug, timeout)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output showing plugin loading and message flow")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Skip authentication, run plugin in development mode")
	cmd.Flags().BoolVar(&pluginDebug, "plugin-debug", false, "Enable plugin-specific debug logging")
	cmd.Flags().StringVar(&timeout, "timeout", "30s", "Set plugin initialization timeout")

	return cmd
}

// runDevPluginTest executes the plugin test with mock MCP communication
func runDevPluginTest(cmd *cobra.Command, pluginName, version string, verbose, noAuth, pluginDebug bool, timeout string) error {
	ctx := context.Background()
	
	fmt.Printf("🔧 Development Plugin Test\n")
	fmt.Printf("📦 Plugin: %s\n", pluginName)
	fmt.Printf("⚙️  CLI Version: %s\n", version)
	
	if verbose {
		fmt.Printf("🔍 Verbose output enabled\n")
	}
	if noAuth {
		fmt.Printf("🚫 Authentication disabled\n")
	}
	if pluginDebug {
		fmt.Printf("🐛 Plugin debug enabled\n")
	}
	
	fmt.Printf("⏱️  Timeout: %s\n\n", timeout)
	
	// Load configuration to get plugins directory
	loader, storage, err := configpkg.CreateConfigServiceFromDefaults()
	if err != nil {
		return fmt.Errorf("failed to create config service: %w", err)
	}
	configService := configpkg.NewConfigService(loader, storage)
	
	config, err := configService.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Determine plugins directory
	pluginsDir := config.PluginsDir
	if pluginsDir == "" {
		pluginsDir = "~/.km/plugins"
	}
	
	if verbose {
		fmt.Printf("🔍 Looking for plugins in: %s\n", pluginsDir)
	}
	
	// Step 1: Plugin Discovery
	fmt.Printf("🔍 Discovering plugin: %s\n", pluginName)
	
	pluginInfo, err := plugins.FindInstalledPlugin(pluginName, pluginsDir)
	if err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}
	
	fmt.Printf("✅ Found plugin binary: %s\n", pluginInfo.Path)
	if verbose {
		fmt.Printf("   📊 Size: %d bytes\n", pluginInfo.Size)
		fmt.Printf("   🔐 Executable: %t\n", pluginInfo.Executable)
	}
	
	// Step 2: Plugin Loading
	fmt.Printf("\n🚀 Loading plugin via go-plugin...\n")
	
	loadResult, err := plugins.LoadSinglePlugin(ctx, pluginName, pluginInfo.Path, verbose)
	if err != nil {
		return fmt.Errorf("plugin loading failed: %w", err)
	}
	
	// Ensure plugin is cleaned up on exit
	defer func() {
		if verbose {
			fmt.Printf("🛑 Shutting down plugin: %s\n", pluginName)
		}
		loadResult.Client.Kill()
	}()
	
	fmt.Printf("✅ Plugin loaded successfully: %s v%s\n", loadResult.PluginInfo.Name, loadResult.PluginInfo.Version)
	if verbose {
		fmt.Printf("   📋 Description: %s\n", loadResult.PluginInfo.Description)
		fmt.Printf("   🏛️  Required Tier: %s\n", loadResult.PluginInfo.RequiredTier)
	}
	
	// Step 3: Plugin Authentication
	fmt.Printf("\n🔑 Authenticating plugin...\n")
	
	var apiKey string
	if !noAuth {
		// Use API key from config if available
		if config.HasAPIKey() {
			apiKey = config.APIKey
			if verbose {
				fmt.Printf("   🔑 Using API key from configuration\n")
			}
		} else {
			if verbose {
				fmt.Printf("   ℹ️  No API key configured - Free tier authentication\n")
			}
		}
	} else {
		if verbose {
			fmt.Printf("   🚫 Authentication disabled via --no-auth flag\n")
		}
	}
	
	err = plugins.AuthenticateLoadedPluginWithJWT(ctx, loadResult.Plugin, apiKey, pluginName, config.APIEndpoint, verbose)
	if err != nil {
		return fmt.Errorf("plugin authentication failed: %w", err)
	}
	
	// Step 4: Mock MCP Communication
	fmt.Printf("\n📨 Testing plugin with mock MCP messages...\n")
	
	err = testPluginWithMockMessages(ctx, loadResult.Plugin, pluginName, verbose)
	if err != nil {
		return fmt.Errorf("mock MCP communication failed: %w", err)
	}
	
	fmt.Printf("\n✅ Plugin testing completed successfully!\n")
	fmt.Printf("💡 Check ~/.km/logs/ for any log files created by the plugin\n")
	
	return nil
}

// testPluginWithMockMessages tests the plugin's ProcessMessage method with mock MCP requests
func testPluginWithMockMessages(ctx context.Context, plugin kmsdk.KilometersPlugin, pluginName string, verbose bool) error {
	// Mock MCP request messages to test the plugin
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
			name:      "Initialize Response",
			direction: "outbound", 
			message: `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "logging": {},
      "tools": [
        {
          "name": "list_files",
          "description": "List files in a directory"
        }
      ]
    },
    "serverInfo": {
      "name": "filesystem-server",
      "version": "0.1.0"
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
	}

	if verbose {
		fmt.Printf("📋 Testing with %d mock MCP messages:\n", len(mockMessages))
	}

	// Process each mock message through the plugin
	for i, mock := range mockMessages {
		if verbose {
			fmt.Printf("   %d. %s (%s)\n", i+1, mock.name, mock.direction)
		} else {
			fmt.Printf("📨 Processing: %s\n", mock.name)
		}

		// Call the plugin's ProcessMessage method
		events, err := plugin.ProcessMessage(ctx, []byte(mock.message), mock.direction)
		if err != nil {
			return fmt.Errorf("plugin failed to process message '%s': %w", mock.name, err)
		}

		if verbose {
			fmt.Printf("     ✅ Processed successfully, generated %d events\n", len(events))
			if len(events) > 0 {
				for j, event := range events {
					fmt.Printf("     📝 Event %d: ID=%s, Type=%s\n", j+1, event.ID, event.Type)
				}
			}
		}

		// Small delay between messages for realistic timing
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("✅ Successfully processed %d mock MCP messages\n", len(mockMessages))
	
	return nil
}