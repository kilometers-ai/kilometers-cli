# Kilometers CLI

**The first purpose-built monitoring tool for Model Context Protocol (MCP) communications**

Kilometers CLI (`km`) enables real-time monitoring and insights into AI assistant interactions by intercepting and analyzing MCP JSON-RPC 2.0 messages. Built with Domain-Driven Design and Hexagonal Architecture principles.

## 🚀 Quick Start

```bash
# Install (macOS/Linux)
curl -fsSL https://get.kilometers.ai/install.sh | sh

# Initialize configuration
km init

# Monitor an MCP server
km monitor --server -- npx -y @modelcontextprotocol/server-linear
```

## ✨ Key Features

- **🔍 Real-time MCP Monitoring**: Intercept and analyze all JSON-RPC messages with clean command separation
- **🤖 AI Agent Ready**: Drop-in replacement for MCP servers in JSON configurations
- **📊 Session Management**: Group events into logical monitoring sessions with intelligent batching
- **☁️ Platform Integration**: Send insights to Kilometers platform for analysis and visualization
- **🐛 Debug Tools**: Advanced debugging with message replay capabilities
- **⚡ Simple & Fast**: Minimal configuration, maximum visibility

## 🖥️ Monitor Command Usage

The `km monitor` command is the core functionality for monitoring MCP servers. It uses clean `--server --` syntax to separate monitor flags from server commands.

### Basic Usage

```bash
# Standard syntax with clean separation
km monitor --server -- npx -y @modelcontextprotocol/server-github

# With additional monitor flags
km monitor --batch-size 20 --server -- npx -y @modelcontextprotocol/server-github

# With debug replay
km monitor --debug-replay events.jsonl --server -- python -m my_server
```

### More Examples

```bash
# Monitor Linear MCP server
km monitor --server -- npx -y @modelcontextprotocol/server-linear

# Monitor Python MCP server with custom port
km monitor --server -- python -m my_mcp_server --port 8080

# Monitor with custom batch size
km monitor --batch-size 5 --server -- npx @modelcontextprotocol/server-filesystem
```

### 🤖 AI Agent Integration

For AI agents using MCP configuration files, add Kilometers monitoring by wrapping your server commands:

**Before (direct MCP server):**
```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}
```

**After (with Kilometers monitoring):**
```json
{
  "mcpServers": {
    "github": {
      "command": "km",
      "args": ["monitor", "--server", "--", "npx", "-y", "@modelcontextprotocol/server-github"]
    },
    "linear": {
      "command": "km", 
      "args": ["monitor", "--server", "--", "npx", "-y", "@modelcontextprotocol/server-linear"]
    }
  }
}
```

### Monitor Flags

- `--batch-size int` - Number of events to batch before sending (default: 10)
- `--debug-replay string` - Path to debug replay file for testing
- `--server` - Required flag followed by `--` and the MCP server command
- `--help, -h` - Show detailed help with examples

### Notes

- **Press Ctrl+C** to stop monitoring gracefully
- The `--server --` syntax cleanly separates monitor flags from server command and arguments
- Works seamlessly in both terminal usage and JSON configurations
- All monitor flags must come before the `--server --` separator

## 📚 Documentation

### 🎯 Project Understanding
Start here to understand what Kilometers CLI does and why it exists:

- **[📋 Project Overview](docs/project/overview.md)** - Comprehensive project knowledge and strategic positioning
- **[🎯 Project Brief](memory-bank/projectbrief.md)** - Core mission, goals, and requirements  
- **[🏗️ Why This Exists](memory-bank/productContext.md)** - Problem space, user scenarios, and market context

### 🛠️ Development & Contributing
Essential resources for developers working on the project:

- **[🛠️ Developer Guide](docs/development/guide.md)** - Setup, building, testing, and development workflow
- **[🐛 Debug & Troubleshooting](docs/development/debug-replay.md)** - Debug replay feature and troubleshooting
- **[🔧 Current Issues & Fixes](docs/development/mcp-fixes.md)** - Critical MCP processing issues that need attention

### 🏗️ Architecture & Design
Deep technical documentation for understanding the system design:

- **[🏛️ System Patterns](memory-bank/systemPatterns.md)** - DDD/Hexagonal architecture implementation
- **[⚙️ Technical Stack](memory-bank/techContext.md)** - Technologies, constraints, and development environment

### 📊 Current Status & Progress  
Track the current state of development and active work:

- **[📊 Implementation Progress](memory-bank/progress.md)** - What works, what's broken, known issues
- **[🎯 Active Work](memory-bank/activeContext.md)** - Current development focus and priorities

### 🧠 Memory Bank System
Special documentation system for AI-assisted development:

- **[🧠 Memory Bank](memory-bank/)** - Cursor's persistent memory system for project context

## 🏗️ Architecture

Kilometers CLI follows **Domain-Driven Design** and **Hexagonal Architecture** principles:

```
┌─ Interface Layer ─────────────────────────┐
│  CLI Commands (Cobra) + Dependency Injection │
├─ Application Layer ───────────────────────┤
│  Services + Command Handlers (CQRS)      │
├─ Domain Layer ────────────────────────────┤  
│  • Event Management (MCP message lifecycle) │
│  • Session Aggregates (monitoring sessions) │
│  • Core Business Rules                    │
├─ Infrastructure Layer ────────────────────┤
│  • API Gateway (Kilometers platform)     │
│  • Process Monitor (MCP server wrapping) │
│  • Configuration Repository              │
└────────────────────────────────────────────┘
```

### Key Design Principles

- **Ports & Adapters**: Clean separation between domain logic and external dependencies
- **Event-Driven**: MCP messages flow as domain events through the system
- **Session-Centric**: All monitoring organized around session lifecycles
- **Configuration-Light**: Minimal setup, smart defaults, environment-aware

## 🛠️ Development

### Prerequisites
- Go 1.24.4 or later
- Make (for build automation)

### Installation
```bash
# Clone the repository
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli

# Install dependencies  
go mod download

# Build and test
make build
make test
```

### First Use
```bash
# Initialize configuration
./km init

# Test with a simple command
./km monitor --server -- echo "Hello MCP"

# Monitor a real MCP server
./km monitor --server -- npx -y @modelcontextprotocol/server-linear
```

## 🤝 Contributing

1. Read the [Developer Guide](docs/development/guide.md) for setup and workflow
2. Check [Active Work](memory-bank/activeContext.md) for current priorities  
3. Review [Architecture Patterns](memory-bank/systemPatterns.md) for design principles
4. See [Current Issues](docs/development/mcp-fixes.md) for critical fixes needed

## 📄 License

[License information - to be added]

## 🔗 Links

- **Platform**: [Kilometers.ai](https://kilometers.ai)
- **Issues**: [GitHub Issues](https://github.com/kilometers-ai/kilometers-cli/issues) 
- **Linear Project**: [Kilometers CLI Development](https://linear.app/kilometers-ai)

---

> **Note**: This project is production-ready for core MCP monitoring functionality. The CLI provides real-time monitoring and session management for AI agents and direct CLI usage. See [Active Work Context](memory-bank/activeContext.md) for current development focus. 