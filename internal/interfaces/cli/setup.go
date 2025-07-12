package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// NewSetupCommand creates the setup command
func NewSetupCommand(container *CLIContainer) *cobra.Command {
	var setupCmd = &cobra.Command{
		Use:   "setup [assistant]",
		Short: "Set up integration with AI assistants",
		Long: `Set up integration with AI assistants by configuring
their MCP settings to work with Kilometers CLI.

Supported assistants:
- claude-desktop
- vscode
- chatgpt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assistant := args[0]
			return runSetup(container, assistant)
		},
	}

	return setupCmd
}

// runSetup handles the setup process for different AI assistants
func runSetup(container *CLIContainer, assistant string) error {
	// Load configuration to get API key
	config, err := container.ConfigRepo.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration. Run 'km init' first: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key not configured. Run 'km init' first")
	}

	switch assistant {
	case "claude-desktop":
		return setupClaudeDesktop(config.APIKey)
	case "vscode":
		return setupVSCode(config.APIKey)
	case "chatgpt":
		return setupChatGPT(config.APIKey)
	default:
		return fmt.Errorf("unknown AI assistant: %s\n\nSupported assistants:\n- claude-desktop\n- vscode\n- chatgpt", assistant)
	}
}

// setupClaudeDesktop sets up Claude Desktop integration
func setupClaudeDesktop(apiKey string) error {
	fmt.Println("🤖 Setting up Claude Desktop integration")
	fmt.Println("")

	configPath := getClaudeDesktopConfigPath()
	if configPath == "" {
		return fmt.Errorf("could not determine Claude Desktop configuration path")
	}

	fmt.Printf("Claude Desktop config file: %s\n", configPath)
	fmt.Println("")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("⚠️  Claude Desktop config file does not exist.")
		fmt.Println("Please install Claude Desktop first: https://claude.ai/download")
		return nil
	}

	fmt.Println("📝 Manual setup required:")
	fmt.Println("")
	fmt.Println("1. Open Claude Desktop")
	fmt.Println("2. Go to Settings > MCP Servers")
	fmt.Println("3. Add a new server with these settings:")
	fmt.Println("")
	fmt.Printf("   Name: kilometers-cli\n")
	fmt.Printf("   Command: km\n")
	fmt.Printf("   Arguments: [\"monitor\"]\n")
	fmt.Printf("   Environment Variables:\n")
	fmt.Printf("     KILOMETERS_API_KEY: %s\n", apiKey)
	fmt.Println("")
	fmt.Println("4. Save and restart Claude Desktop")
	fmt.Println("")
	fmt.Println("✅ Claude Desktop setup instructions provided")

	return nil
}

// setupVSCode sets up VS Code integration
func setupVSCode(apiKey string) error {
	fmt.Println("🔧 Setting up VS Code integration")
	fmt.Println("")

	fmt.Println("📝 Manual setup required:")
	fmt.Println("")
	fmt.Println("1. Install the MCP extension for VS Code")
	fmt.Println("2. Open VS Code Settings (Ctrl/Cmd + ,)")
	fmt.Println("3. Search for 'MCP' and configure:")
	fmt.Println("")
	fmt.Printf("   MCP Server Path: km\n")
	fmt.Printf("   MCP Server Args: [\"monitor\"]\n")
	fmt.Printf("   Environment Variables:\n")
	fmt.Printf("     KILOMETERS_API_KEY: %s\n", apiKey)
	fmt.Println("")
	fmt.Println("4. Restart VS Code")
	fmt.Println("")
	fmt.Println("✅ VS Code setup instructions provided")

	return nil
}

// setupChatGPT sets up ChatGPT integration
func setupChatGPT(apiKey string) error {
	fmt.Println("💬 Setting up ChatGPT integration")
	fmt.Println("")

	fmt.Println("📝 Manual setup required:")
	fmt.Println("")
	fmt.Println("1. ChatGPT integration requires custom configuration")
	fmt.Println("2. Contact support for specific setup instructions")
	fmt.Println("")
	fmt.Printf("   Your API Key: %s\n", apiKey)
	fmt.Println("")
	fmt.Println("✅ ChatGPT setup information provided")

	return nil
}

// getClaudeDesktopConfigPath returns the path to Claude Desktop config
func getClaudeDesktopConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(homeDir, ".config", "claude", "claude_desktop_config.json")
	default:
		return ""
	}
}
