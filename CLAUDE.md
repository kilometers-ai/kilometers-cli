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

## Configuration

The CLI supports configuration through multiple sources in order of precedence:

1. **Environment Variables** (highest priority)
2. **Configuration File** (`km_config.json`) - for settings only
3. **OS Keyring** - for secure credential storage
4. **Default Values** (lowest priority)

### Environment Variables

The following environment variables can be used:

```bash
KM_API_KEY=your_api_key_here           # Required: Your Kilometers.ai API key
KM_API_URL=https://api.kilometers.ai   # Optional: API URL (defaults to https://api.kilometers.ai)
KM_DEFAULT_TIER=enterprise             # Optional: Default tier for requests
```

### .env File Support

You can also use a `.env` file in the project root for local development:

```bash
# Copy .env.example to .env and fill in your values
cp .env.example .env
```

### Configuration File

JSON configuration via `km_config.json` stores application settings:

```json
{
  "api_url": "https://api.kilometers.ai",
  "default_tier": "enterprise"
}
```

**Note:** API keys and authentication tokens are stored securely in your OS keyring (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux), not in the configuration file.

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
- **keyring_token_store.rs**: Securely stores JWT tokens in OS keyring

## Key Files

- **km_config.json**: Stores application settings (not credentials)
- **OS Keyring**: Stores JWT tokens and authentication credentials securely
- **mcp_proxy.log**: JSON Lines format log of all MCP requests/responses
- **src/main.rs**: CLI entry point with clap-based command parsing

## MCP Protocol Flow

1. Client sends JSON-RPC request to km proxy via stdin
2. Proxy logs request to mcp_proxy.log
3. Proxy forwards request to target MCP server process
4. Server responds with JSON-RPC response
5. Proxy logs response and sends telemetry to Kilometers.ai
6. Proxy forwards response back to client via stdout
