# Kilometers CLI

The Kilometers CLI (`km`) is a transparent wrapper for MCP (Model Context Protocol) servers that monitors AI agent activity with zero configuration changes.

## 🚀 Quick Start

```bash
# Instead of running your MCP server directly:
npx @modelcontextprotocol/server-github

# Run it through the Kilometers wrapper:
km npx @modelcontextprotocol/server-github
```

That's it! Your MCP server runs exactly the same, but now all interactions are monitored and sent to the Kilometers.ai dashboard.

## 📦 Installation

### Option 1: Direct Download (Recommended)
```bash
# macOS/Linux
curl -sSL https://get.kilometers.ai/install.sh | sh

# Windows (PowerShell)
iwr -useb https://get.kilometers.ai/install.ps1 | iex
```

### Option 2: Manual Download
1. Download the latest release for your platform from [GitHub Releases](https://github.com/kilometers-ai/kilometers/releases)
2. Extract the binary to your PATH
3. Rename to `km` (or `km.exe` on Windows)

### Option 3: Build from Source
```bash
git clone https://github.com/kilometers-ai/kilometers
cd kilometers/cli
go build -o km .
```

### Verify Installation
```bash
km --help
```

## ⚙️ Configuration

The CLI supports multiple configuration methods (in order of precedence):

### Environment Variables
```bash
export KILOMETERS_API_URL="https://api.kilometers.ai"
export KILOMETERS_API_KEY="your-api-key-here"
export KM_DEBUG="true"                    # Enable debug logging
export KM_BATCH_SIZE="20"                 # Events per batch (default: 10)
```

### Configuration File
Create `~/.config/kilometers/config.json`:
```json
{
  "api_endpoint": "https://api.kilometers.ai",
  "api_key": "your-api-key-here",
  "batch_size": 10,
  "debug": false
}
```

### Default Values
- **API Endpoint**: `http://localhost:5194` (for local development)
- **Batch Size**: `10` events per API call
- **Debug**: `false`

## 🎯 Usage Examples

### Basic MCP Server Wrapping
```bash
# GitHub MCP Server
km npx @modelcontextprotocol/server-github

# Slack MCP Server  
km python -m slack_mcp_server

# File System MCP Server
km npx @modelcontextprotocol/server-filesystem --path /path/to/workspace

# Custom MCP Server
km ./my-custom-mcp-server --config config.json
```

### With Different AI Tools

#### Cursor
```bash
# Configure in Cursor settings.json:
{
  "mcp.servers": {
    "github": {
      "command": "km",
      "args": ["npx", "@modelcontextprotocol/server-github"]
    }
  }
}
```

#### Claude Desktop
```bash
# Configure in claude_desktop_config.json:
{
  "mcpServers": {
    "github": {
      "command": "km",
      "args": ["npx", "@modelcontextprotocol/server-github"]
    }
  }
}
```

### Debug Mode
```bash
# Enable verbose logging
KM_DEBUG=true km npx @modelcontextprotocol/server-github

# View logs
km npx @modelcontextprotocol/server-github 2>&1 | grep "\[km\]"
```

### Local Development
```bash
# Point to local API during development
KILOMETERS_API_URL="http://localhost:5194" km your-mcp-server

# Disable API (local logging only)
KILOMETERS_API_URL="" km your-mcp-server
```

## 📊 What Gets Monitored

The CLI transparently captures:

- **All MCP JSON-RPC Messages**: Requests and responses
- **Method Calls**: `tools/call`, `resources/read`, `prompts/get`, etc.
- **Performance Data**: Response times, payload sizes
- **Metadata**: Timestamps, event IDs, error conditions

### Example Captured Event
```json
{
  "id": "evt_1234567890abcdef",
      "timestamp": "2025-06-27T10:30:45Z",
  "direction": "request",
  "method": "tools/call",
  "payload": "eyJ0b29sIjogImdpdGh1Yi1zZWFyY2giLCAiYXJncyI6IHt9fQ==",
  "size": 156
}
```

## 🔧 CLI Commands

### Basic Usage
```bash
km [MCP_SERVER_COMMAND] [ARGS...]
```

### Help and Version
```bash
km --help          # Show usage information
km --version       # Show version information
```

### Configuration Management
```bash
# Test API connection
km --test-connection

# Show current configuration
km --show-config

# Reset configuration to defaults
km --reset-config
```

## 🚨 Troubleshooting

### Common Issues

#### CLI Not Found
```bash
# Check if km is in PATH
which km

# Add to PATH if needed (macOS/Linux)
export PATH=$PATH:/path/to/km

# Add to PATH if needed (Windows)
set PATH=%PATH%;C:\path\to\km.exe
```

#### API Connection Failed
```bash
# Test connectivity
curl https://api.kilometers.ai/health

# Check configuration
echo $KILOMETERS_API_URL
echo $KILOMETERS_API_KEY

# Enable debug mode
KM_DEBUG=true km your-mcp-server
```

#### MCP Server Won't Start
```bash
# Test MCP server directly first
npx @modelcontextprotocol/server-github

# Then test with wrapper
km npx @modelcontextprotocol/server-github

# Check for conflicting processes
ps aux | grep mcp
```

#### No Events in Dashboard
1. **Check API Key**: Ensure `KILOMETERS_API_KEY` is set correctly
2. **Verify API URL**: Default is localhost; use production URL for dashboard
3. **Test Connection**: Use `KM_DEBUG=true` to see API communication
4. **Check Firewall**: Ensure outbound HTTPS access to api.kilometers.ai

### Debug Information

Enable debug logging to see detailed information:
```bash
KM_DEBUG=true km your-mcp-server 2>&1 | tee km-debug.log
```

Debug output includes:
- Configuration loaded
- MCP server startup
- Event capture details  
- API communication status
- Error messages and stack traces

### Performance Monitoring

Check CLI overhead:
```bash
# Without wrapper
time npx @modelcontextprotocol/server-github

# With wrapper  
time km npx @modelcontextprotocol/server-github

# The difference should be <5ms
```

### Local Fallback

If the API is unavailable, the CLI continues to work:
- Events are logged locally to stderr
- MCP server functions normally
- No data is lost (events queued for retry)

## 🔒 Security & Privacy

### Data Collection
- **Only MCP Protocol Data**: We capture JSON-RPC messages between AI tools and MCP servers
- **No Personal Files**: File contents are not transmitted unless explicitly requested by AI
- **Configurable Retention**: Data retention periods are configurable (7-90 days)

### Authentication
- **API Keys**: Secure bearer token authentication
- **TLS Encryption**: All data transmitted over HTTPS/TLS 1.3
- **Local Storage**: No sensitive data stored locally

### Privacy Controls
- **Opt-out**: Remove wrapper to stop all monitoring
- **Data Export**: Request data exports anytime
- **Data Deletion**: Request data deletion anytime

## 🛠️ Advanced Configuration

### Custom Batching
```bash
# High-frequency environments (more API calls, less latency)
KM_BATCH_SIZE=5 km your-mcp-server

# Low-frequency environments (fewer API calls, more latency)
KM_BATCH_SIZE=50 km your-mcp-server
```

### Custom API Endpoints
```bash
# Enterprise installations
KILOMETERS_API_URL="https://kilometers.your-company.com" km your-mcp-server

# Local development
KILOMETERS_API_URL="http://localhost:5194" km your-mcp-server
```

### Multiple Configurations
```bash
# Development environment
export KM_ENV=dev
export KILOMETERS_API_URL="https://dev-api.kilometers.ai"

# Production environment  
export KM_ENV=prod
export KILOMETERS_API_URL="https://api.kilometers.ai"
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `go test ./...`
6. Submit a pull request

### Development Setup
```bash
git clone https://github.com/kilometers-ai/kilometers
cd kilometers/cli
go mod tidy
go build -o km .
```

### Running Tests
```bash
go test -v ./...
go test -race ./...
go test -cover ./...
```

## 📄 License

MIT License - see [LICENSE](../LICENSE) for details.

## 🆘 Support

- **Documentation**: [https://docs.kilometers.ai](https://docs.kilometers.ai)
- **Issues**: [GitHub Issues](https://github.com/kilometers-ai/kilometers/issues)
- **Email**: support@kilometers.ai
- **Discord**: [Join our community](https://discord.gg/kilometers)

---

**Kilometers.ai** - Your AI's digital odometer. Track every interaction, understand patterns, control costs. # Trigger deployment
