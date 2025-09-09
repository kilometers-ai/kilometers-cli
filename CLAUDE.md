# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Kilometers CLI Proxy (`km`), a Rust-based MCP (Model Context Protocol) proxy that intercepts, logs, and forwards requests between MCP clients and servers while sending telemetry to the Kilometers.ai API.

## Development Commands

### Build
```bash
cargo build              # Debug build
cargo build --release    # Release build
```

### Run
```bash
# Initialize configuration with API key
cargo run -- init

# Monitor and proxy MCP requests
cargo run -- monitor -- <command> [args...]

# Clear logs
cargo run -- clear-logs

# Example: Proxy to an MCP server
cargo run -- monitor -- npx -y @modelcontextprotocol/server-filesystem ~/Documents
```

### Lint & Format
```bash
cargo clippy             # Run linter
cargo fmt                # Format code
```

### Test
```bash
cargo test               # Run all tests
cargo test <test_name>   # Run specific test
```

## Architecture

The codebase follows a layered architecture pattern:

### Domain Layer (`src/domain/`)
- **auth.rs**: Authentication models and API key validation
- **proxy.rs**: Core proxy command structures and MCP protocol types

### Application Layer (`src/application/`)
- **commands.rs**: CLI command implementations (InitCommand, MonitorCommand, ClearLogsCommand)
- **services.rs**: Business logic services (AuthenticationService, ProxyService)

### Infrastructure Layer (`src/infrastructure/`)
- **api_client.rs**: HTTP client for Kilometers.ai API communication
- **configuration_repository.rs**: Manages km_config.json persistence
- **log_repository.rs**: Handles mcp_proxy.log file operations
- **process_manager.rs**: Spawns and manages proxied MCP server processes
- **event_sender.rs**: Sends telemetry events to the API

## Key Files

- **km_config.json**: Stores API key configuration
- **mcp_proxy.log**: JSON Lines format log of all MCP requests/responses
- **src/main.rs**: CLI entry point with clap-based command parsing

## MCP Protocol Flow

1. Client sends JSON-RPC request to km proxy via stdin
2. Proxy logs request to mcp_proxy.log
3. Proxy forwards request to target MCP server process
4. Server responds with JSON-RPC response
5. Proxy logs response and sends telemetry to Kilometers.ai
6. Proxy forwards response back to client via stdout