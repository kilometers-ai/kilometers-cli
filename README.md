# Kilometers CLI (`km`)

**MCP Server Monitoring Proxy for Debugging and Development**

Kilometers CLI is a command-line tool that acts as a transparent proxy for Model Context Protocol (MCP) servers, capturing and logging JSON-RPC communication for debugging, analysis, and development purposes.

## 🚀 Quick Start

```bash
# Install and configure
go build -o build/km cmd/main.go
./build/km init --api-key YOUR_API_KEY

# Monitor any MCP server
./build/km monitor --server -- npx -y @modelcontextprotocol/server-github
```

## 📋 Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Cursor MCP Integration](#cursor-mcp-integration)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## 🛠 Installation

### Prerequisites
- Go 1.21 or later
- Access to a Kilometers API endpoint

### Build from Source
```bash
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli
go build -o build/km cmd/main.go
```

### Verify Installation
```bash
./build/km --version
./build/km --help
```

## ⚙️ Configuration

### Initial Setup
```bash
# Interactive configuration
./build/km init

# Direct configuration
./build/km init --api-key km_live_your_api_key_here

# Force overwrite existing config
./build/km init --api-key km_live_your_api_key_here --force
```

### Environment Variables
The CLI supports these environment variables (precedence: env > config file > defaults):

- `KILOMETERS_API_KEY`: Your Kilometers API key
- `KILOMETERS_API_ENDPOINT`: API endpoint (default: `http://localhost:5194`)

## 🎯 Usage

### Basic Monitoring
```bash
# Monitor any MCP server command
./build/km monitor --server -- [server-command]
```

### Common Examples
```bash
# GitHub MCP Server
./build/km monitor --server -- npx -y @modelcontextprotocol/server-github

# Linear MCP Server  
./build/km monitor --server -- npx -y @tacticlaunch/mcp-linear

# Python MCP Server
./build/km monitor --server -- python -m my_mcp_server

# Docker MCP Server
./build/km monitor --server -- docker run my-mcp-server

# Custom executable
./build/km monitor --server -- /path/to/custom-mcp-server
```

### Debug Mode
```bash
# Enable detailed debug output
./build/km monitor --debug --server -- npx -y @modelcontextprotocol/server-github
```

### Output Formats
```bash
# JSON output format
./build/km monitor --output-format json --server -- npx -y @modelcontextprotocol/server-linear

# Console output (default)
./build/km monitor --output-format console --server -- python -m my_server
```

## 🔌 Cursor MCP Integration

To use the Kilometers CLI with Cursor's MCP system, add the following configuration to your `~/.cursor/mcp.json` file:

### Basic Configuration

```json
{
  "mcpServers": {
    "km-linear": {
      "command": "/path/to/kilometers-cli/build/km",
      "args": [
        "monitor",
        "--server",
        "--",
        "npx",
        "-y",
        "@tacticlaunch/mcp-linear"
      ],
      "env": {
        "KILOMETERS_API_KEY": "km_live_your_api_key_here",
        "KILOMETERS_API_ENDPOINT": "http://localhost:5194",
        "LINEAR_API_TOKEN": "lin_api_your_linear_token_here"
      }
    }
  }
}
```

### Multiple MCP Servers

```json
{
  "mcpServers": {
    "km-linear": {
      "command": "/path/to/kilometers-cli/build/km",
      "args": ["monitor", "--server", "--", "npx", "-y", "@tacticlaunch/mcp-linear"],
      "env": {
        "KILOMETERS_API_KEY": "km_live_your_api_key_here",
        "KILOMETERS_API_ENDPOINT": "http://localhost:5194",
        "LINEAR_API_TOKEN": "lin_api_your_linear_token_here"
      }
    },
    "km-github": {
      "command": "/path/to/kilometers-cli/build/km", 
      "args": ["monitor", "--server", "--", "npx", "-y", "@modelcontextprotocol/server-github"],
      "env": {
        "KILOMETERS_API_KEY": "km_live_your_api_key_here",
        "KILOMETERS_API_ENDPOINT": "http://localhost:5194",
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_your_github_token_here"
      }
    },
    "km-playwright": {
      "command": "/path/to/kilometers-cli/build/km",
      "args": ["monitor", "--server", "--", "npx", "@playwright/mcp@latest"],
      "env": {
        "KILOMETERS_API_KEY": "km_live_your_api_key_here",
        "KILOMETERS_API_ENDPOINT": "http://localhost:5194"
      }
    }
  }
}
```

### Setup Steps

1. **Build the CLI tool** (see [Installation](#installation))

2. **Get your Kilometers API key**:
   - Contact your Kilometers administrator or
   - Generate one from your Kilometers dashboard

3. **Update the command path** in mcp.json:
   - Replace `/path/to/kilometers-cli/build/km` with your actual build path
   - Example: `/Users/yourusername/Source/kilometers-cli/build/km`

4. **Configure environment variables**:
   - `KILOMETERS_API_KEY`: Your Kilometers API key (required)
   - `KILOMETERS_API_ENDPOINT`: Your Kilometers API endpoint (required)
   - Additional tokens for specific MCP servers (LINEAR_API_TOKEN, GITHUB_PERSONAL_ACCESS_TOKEN, etc.)

5. **Restart Cursor** to load the new MCP configuration

### Troubleshooting Cursor Integration

- **Command not found**: Verify the `command` path points to your built km binary
- **Permission denied**: Ensure the km binary has execute permissions (`chmod +x build/km`)
- **API connection failed**: Check your `KILOMETERS_API_KEY` and `KILOMETERS_API_ENDPOINT`
- **MCP server fails**: Verify the underlying MCP server works without km first

## 🔧 Advanced Usage

### Configuration Options
```bash
# Large message support (for 1MB+ payloads)
./build/km monitor --buffer-size 2MB --server -- python -m large_mcp_server

# Batch processing
./build/km monitor --batch-size 50 --server -- npx -y @modelcontextprotocol/server-github

# Custom output directory  
./build/km monitor --output-dir ./logs --server -- docker run my-mcp-server
```

### All Available Flags
```bash
./build/km monitor --help

Flags:
      --batch-size int        Batch size for processing messages (default 10)
      --buffer-size string    Buffer size for large messages (default "1MB")
      --debug                 Enable debug output
  -h, --help                  help for monitor
      --output-dir string     Output directory for logs (default "./")
      --output-format string  Output format: console, json (default "console")
      --server                Indicates server command follows
```

### Large Message Handling
The CLI automatically handles large JSON-RPC messages (1MB+) that can cause "token too long" errors in other tools:

```bash
# Configure larger buffer for very large messages
./build/km monitor --buffer-size 5MB --server -- python -m large_data_server
```

## 🔍 Troubleshooting

### Common Issues

**"Command not found" error**
```bash
# Ensure km is built and executable
go build -o build/km cmd/main.go
chmod +x build/km
./build/km --version
```

**"API key not configured" error**
```bash
# Configure your API key
./build/km init --api-key km_live_your_api_key_here
```

**"Connection refused" to Kilometers API**
- Verify `KILOMETERS_API_ENDPOINT` is correct
- Ensure the Kilometers API server is running
- Check network connectivity and firewall settings

**MCP server fails to start**
- Test the MCP server command directly without km first
- Check that all required environment variables are set
- Verify the MCP server binary/package is installed

**"Token too long" errors**
```bash
# Increase buffer size for large messages
./build/km monitor --buffer-size 2MB --server -- your-mcp-server
```

### Debug Mode
Enable debug mode for detailed troubleshooting:
```bash
./build/km monitor --debug --server -- npx -y @modelcontextprotocol/server-github
```

This will show:
- Process execution details
- Stream handling information  
- JSON-RPC message parsing
- Error details and stack traces

## 🏗 Development

### Architecture
The CLI follows Domain-Driven Design (DDD) and Clean Architecture principles:

- **Domain Layer**: Core business logic (MonitoringSession, JSONRPCMessage)
- **Application Layer**: Use cases and services (MonitoringService)
- **Infrastructure Layer**: External concerns (ProcessExecutor, StreamProxy, Logging)
- **Interface Layer**: CLI commands and user interaction (Cobra CLI)

### Building
```bash
# Development build
go build -o build/km cmd/main.go

# Cross-platform builds (see build-releases.sh)
./build-releases.sh
```

### Testing
```bash
# Run unit tests
go test ./...

# Integration tests
./test-mcp-monitoring.sh

# Test with mock MCP server
echo '{"jsonrpc":"2.0","method":"initialize","params":{"capabilities":{}},"id":1}' | \
  ./build/km monitor --debug --server -- cat
```

### Project Structure
```
├── cmd/main.go                 # CLI entry point
├── internal/
│   ├── core/domain/           # Domain models and business logic
│   ├── core/ports/            # Interface definitions  
│   ├── application/services/  # Application services
│   ├── infrastructure/        # External adapters
│   └── interfaces/cli/        # CLI commands and parsing
├── memory-bank/               # Project documentation
└── scripts/                   # Build and install scripts
```

## 📝 License

[Add your license information here]

## 🤝 Contributing

[Add contributing guidelines here]

## 📞 Support

For support and questions:
- [Add support contact information]
- [Add issue tracker link]
- [Add documentation links]

---

**Kilometers CLI** - Transparent MCP server monitoring for better debugging and development.
