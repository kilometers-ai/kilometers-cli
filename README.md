# Kilometers CLI

**The first purpose-built monitoring tool for Model Context Protocol (MCP) communications**

Kilometers CLI (`km`) enables real-time monitoring, risk analysis, and insights into AI assistant interactions by intercepting and analyzing MCP JSON-RPC 2.0 messages. Built with Domain-Driven Design and Hexagonal Architecture principles.

## 🚀 Quick Start

```bash
# Install (macOS/Linux)
curl -fsSL https://get.kilometers.ai/install.sh | sh

# Initialize configuration
km init

# Monitor an MCP server
km monitor --server "npx -y @modelcontextprotocol/server-linear"
```

## ✨ Key Features

- **🔍 Real-time MCP Monitoring**: Intercept and analyze all JSON-RPC messages with intelligent flag separation
- **🤖 AI Agent Ready**: Drop-in replacement for MCP servers in JSON configurations
- **🛡️ Risk Analysis**: Automated detection of high-risk AI assistant actions  
- **🎯 Intelligent Filtering**: Reduce noise with configurable filtering rules
- **📊 Session Management**: Group events into logical monitoring sessions
- **☁️ Platform Integration**: Send insights to Kilometers platform for analysis
- **🐛 Debug Tools**: Advanced debugging with message replay capabilities

## 🖥️ Monitor Command Usage

The `km monitor` command is the core functionality for monitoring MCP servers. It supports both quoted and unquoted server commands with intelligent flag separation.

### Basic Usage

```bash
# Quoted server command (recommended)
km monitor --server "npx -y @modelcontextprotocol/server-github"

# Unquoted server command (works when no flag conflicts)
km monitor --server npx -y @modelcontextprotocol/server-github

# With additional monitor flags
km monitor --server "npx -y @modelcontextprotocol/server-github" --batch-size 20
```

### More Examples

```bash
# Monitor Linear MCP server
km monitor --server "npx -y @modelcontextprotocol/server-linear"

# Monitor Python MCP server with custom port
km monitor --server "python -m my_mcp_server --port 8080"

# Monitor with debug replay
km monitor --server "npx -y @modelcontextprotocol/server-github" --debug-replay events.jsonl

# Monitor with custom batch size and monitoring flags
km monitor --server npx @modelcontextprotocol/server-filesystem --batch-size 5
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
      "args": ["monitor", "--server", "npx -y @modelcontextprotocol/server-github"]
    },
    "linear": {
      "command": "km", 
      "args": ["monitor", "--server", "npx -y @modelcontextprotocol/server-linear"]
    }
  }
}
```

### Monitor Flags

- `--batch-size int` - Number of events to batch before sending (default: 10)
- `--debug-replay string` - Path to debug replay file  
- `--help, -h` - Show detailed help with examples

### Notes

- **Press Ctrl+C** to stop monitoring gracefully
- The `--server` flag separates km monitor flags from MCP server flags
- Use quotes around server commands with flags to avoid conflicts
- Works seamlessly in both terminal usage and JSON configurations

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
- **[📐 Refactoring Plan](docs/architecture/refactoring-plan.md)** - Architecture transformation from monolith to DDD
- **[📈 Refactoring Summary](docs/architecture/refactoring-summary.md)** - Linear project status and created issues

### 📊 Current Status & Progress  
Track the current state of development and active work:

- **[📋 Status Analysis](docs/status/analysis.md)** - Project status overview and working features
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
├─ Application Layer ──────────────────────────┤  
│  Command Handlers + Application Services      │
├─ Domain Layer (Core Business Logic) ─────────┤
│  • Event Entity + Value Objects              │
│  • Session Aggregate Root                    │  
│  • Risk Analysis + Filtering Services        │
├─ Infrastructure Layer ───────────────────────┤
│  • Process Monitor (MCP wrapping)            │
│  • API Gateway (Platform communication)      │
│  • Configuration (Multi-source config)       │
└───────────────────────────────────────────────┘
```

## 🛠️ Technology Stack

- **Language**: Go 1.24.4+
- **CLI Framework**: Cobra v1.9.1  
- **Architecture**: Domain-Driven Design + Hexagonal Architecture
- **Protocol**: JSON-RPC 2.0 (MCP compliance)
- **Testing**: Testify + Property-based testing with Rapid
- **CI/CD**: GitHub Actions

## 📈 Project Status

**Overall Completion**: ~85% (Architecture complete, core monitoring functionality working)

### ✅ Working
- Complete CLI interface with all commands
- **Monitor command with --server flag**: Supports both quoted and unquoted MCP server commands
- **AI agent integration**: Perfect compatibility with MCP JSON configurations
- **Flag separation**: Intelligent parsing prevents conflicts between km and MCP server flags
- Robust configuration system with validation  
- Full DDD/Hexagonal architecture implementation
- Comprehensive testing infrastructure
- Cross-platform builds and releases

### ⚠️ Needs Attention  
- **Medium**: Buffer size limitations for very large payloads (>1MB)
- **Medium**: JSON-RPC parsing could be more robust
- **Low**: Additional monitor flags implementation

See [Implementation Progress](memory-bank/progress.md) for detailed status.

## 🚀 Getting Started

### Prerequisites
- Go 1.24.4+ 
- Git
- Docker (for testing)

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
./km monitor --server "echo Hello MCP"

# Monitor a real MCP server
./km monitor --server "npx -y @modelcontextprotocol/server-linear"
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

> **Note**: This project is under active development. The core MCP monitoring functionality is working and ready for use with AI agents and direct CLI usage. See [Active Work Context](memory-bank/activeContext.md) for current development focus. 