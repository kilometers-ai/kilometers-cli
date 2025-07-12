package main

import (
	"encoding/json"
	"fmt"
)

// ClaudeDesktopConfig represents the structure of Claude Desktop's config file
type ClaudeDesktopConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents an MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// GetClaudeDesktopTemplate returns a template configuration for Claude Desktop
func GetClaudeDesktopTemplate() *ClaudeDesktopConfig {
	return &ClaudeDesktopConfig{
		MCPServers: map[string]MCPServerConfig{
			"github": {
				Command: "km",
				Args:    []string{"npx", "@modelcontextprotocol/server-github"},
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": "your_github_token_here",
				},
			},
			"filesystem": {
				Command: "km",
				Args:    []string{"npx", "@modelcontextprotocol/server-filesystem", "/Users/your-username/Documents"},
			},
			"brave-search": {
				Command: "km",
				Args:    []string{"npx", "@modelcontextprotocol/server-brave-search"},
				Env: map[string]string{
					"BRAVE_API_KEY": "your_brave_api_key_here",
				},
			},
		},
	}
}

// GetClaudeDesktopTemplateJSON returns the template as formatted JSON
func GetClaudeDesktopTemplateJSON() string {
	template := GetClaudeDesktopTemplate()
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// GetVSCodeTemplate returns configuration instructions for VS Code
func GetVSCodeTemplate() string {
	return `{
  "mcp.servers": {
    "github": {
      "command": "km",
      "args": ["npx", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "your_github_token_here"
      }
    },
    "filesystem": {
      "command": "km",
      "args": ["npx", "@modelcontextprotocol/server-filesystem", "/path/to/your/documents"],
      "env": {}
    }
  }
}`
}

// GetCommonMCPServers returns a list of popular MCP servers with their configurations
func GetCommonMCPServers() []MCPServerExample {
	return []MCPServerExample{
		{
			Name:        "GitHub",
			Description: "Access GitHub repositories, issues, and pull requests",
			Package:     "@modelcontextprotocol/server-github",
			EnvVars: map[string]string{
				"GITHUB_PERSONAL_ACCESS_TOKEN": "your_github_token_here",
			},
		},
		{
			Name:        "Filesystem",
			Description: "Access local files and directories",
			Package:     "@modelcontextprotocol/server-filesystem",
			Args:        []string{"/path/to/your/documents"},
		},
		{
			Name:        "Brave Search",
			Description: "Search the web using Brave Search API",
			Package:     "@modelcontextprotocol/server-brave-search",
			EnvVars: map[string]string{
				"BRAVE_API_KEY": "your_brave_api_key_here",
			},
		},
		{
			Name:        "Slack",
			Description: "Access Slack channels and messages",
			Package:     "slack-mcp-server",
			EnvVars: map[string]string{
				"SLACK_BOT_TOKEN": "your_slack_bot_token_here",
			},
		},
		{
			Name:        "PostgreSQL",
			Description: "Query PostgreSQL databases",
			Package:     "@modelcontextprotocol/server-postgres",
			EnvVars: map[string]string{
				"POSTGRES_CONNECTION_STRING": "postgresql://user:password@localhost:5432/database",
			},
		},
	}
}

// MCPServerExample represents an example MCP server configuration
type MCPServerExample struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Package     string            `json:"package"`
	Args        []string          `json:"args,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

// GenerateClaudeDesktopConfig generates a Claude Desktop config with selected MCP servers
func GenerateClaudeDesktopConfig(serverNames []string) *ClaudeDesktopConfig {
	config := &ClaudeDesktopConfig{
		MCPServers: make(map[string]MCPServerConfig),
	}

	servers := GetCommonMCPServers()
	serverMap := make(map[string]MCPServerExample)
	for _, server := range servers {
		serverMap[server.Name] = server
	}

	for _, name := range serverNames {
		if server, exists := serverMap[name]; exists {
			args := []string{"npx", server.Package}
			args = append(args, server.Args...)

			config.MCPServers[name] = MCPServerConfig{
				Command: "km",
				Args:    args,
				Env:     server.EnvVars,
			}
		}
	}

	return config
}

// PrintMCPServerList prints a list of available MCP servers
func PrintMCPServerList() {
	fmt.Println("📦 Popular MCP Servers:")
	fmt.Println("")

	servers := GetCommonMCPServers()
	for i, server := range servers {
		fmt.Printf("%d. %s\n", i+1, server.Name)
		fmt.Printf("   %s\n", server.Description)
		fmt.Printf("   Package: %s\n", server.Package)

		if len(server.EnvVars) > 0 {
			fmt.Println("   Environment variables required:")
			for key, value := range server.EnvVars {
				fmt.Printf("     %s=%s\n", key, value)
			}
		}

		if len(server.Args) > 0 {
			fmt.Printf("   Arguments: %v\n", server.Args)
		}

		fmt.Println("")
	}
}
