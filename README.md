# Kilometers CLI (`km`)

A Rust-based MCP (Model Context Protocol) proxy that intercepts, logs, and forwards requests between MCP clients and servers while sending telemetry to the Kilometers.ai API.

## Quick Start

### Installation

#### From Source
```bash
# Clone the repository
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli

# Build and install
cargo build --release
cargo install --path .
```

#### Pre-built Binaries
Download the latest release for your platform from the [releases page](https://github.com/kilometers-ai/kilometers-cli/releases).

### Initial Setup

1. **Initialize with your API key:**
```bash
km init
```
You'll be prompted to enter your Kilometers.ai API key. The key will be validated and stored in `~/.config/km/config.json`.

2. **Test the proxy with an MCP server:**
```bash
# Example: Proxy to the filesystem MCP server
km monitor -- npx -y @modelcontextprotocol/server-filesystem ~/Documents
```

## Usage

### Commands

#### `km init`
Initialize or update your Kilometers.ai API key configuration.
- Prompts for API key input
- Validates the key with the Kilometers.ai API
- Stores configuration in `~/.config/km/config.json`

#### `km monitor -- <command> [args...]`
Start monitoring and proxying MCP requests to a target server.
- Intercepts JSON-RPC requests/responses
- Logs all traffic to `~/.local/share/km/mcp_proxy.log` (platform-specific)
- Sends telemetry to Kilometers.ai
- Forwards all communication transparently

**Examples:**
```bash
# Proxy to filesystem server
km monitor -- npx -y @modelcontextprotocol/server-filesystem ~/Documents

# Proxy to GitHub server
km monitor -- npx -y @modelcontextprotocol/server-github

# Proxy to a custom MCP server
km monitor -- python my-mcp-server.py --arg1 value1
```

#### `km clear-logs`
Clear all logged MCP requests and responses from the local log file.

## Development

### Prerequisites
- Rust 1.70+ (install via [rustup](https://rustup.rs/))
- Git

### Building from Source

```bash
# Debug build (faster compilation, larger binary)
cargo build

# Release build (optimized, smaller binary)
cargo build --release
```

### Running During Development

```bash
# Run directly with cargo
cargo run -- init
cargo run -- monitor -- <command>
cargo run -- clear-logs

# Run with verbose output for debugging
RUST_LOG=debug cargo run -- monitor -- <command>
```

### Testing

```bash
# Run all tests
cargo test

# Run specific test
cargo test test_successful_initialization

# Run tests with output displayed
cargo test -- --nocapture

# Run integration tests only
cargo test --test '*'
```

### Code Quality

```bash
# Run linter (clippy)
cargo clippy

# Run linter with all targets
cargo clippy --all-targets --all-features

# Format code
cargo fmt

# Check formatting without modifying
cargo fmt -- --check
```

### Release Process

#### Version Management
The project uses date-based versioning with the `scripts/version.sh` script:

```bash
# Generate a new version number (vYYYY.M.D.BUILD)
./scripts/version.sh generate

# Create a new version (updates Cargo.toml, commits, and tags)
./scripts/version.sh create

# Push the latest tag to remote
./scripts/version.sh push

# List recent versions
./scripts/version.sh list

# Show current version
./scripts/version.sh current
```

#### Building Releases

```bash
# Build optimized binary for current platform
cargo build --release --locked

# The binary will be at: target/release/km
```

For cross-platform builds, use [cross](https://github.com/cross-rs/cross):
```bash
# Install cross
cargo install cross

# Build for different targets
cross build --release --target x86_64-pc-windows-gnu
cross build --release --target x86_64-apple-darwin
cross build --release --target x86_64-unknown-linux-gnu
```

## Architecture

### Project Structure
```
kilometers-cli/
├── src/
│   ├── main.rs                 # CLI entry point
│   ├── domain/                 # Business logic and entities
│   │   ├── auth.rs            # Authentication models
│   │   └── proxy.rs           # MCP protocol types
│   ├── application/            # Use cases and workflows
│   │   ├── commands.rs        # CLI command implementations
│   │   └── services.rs        # Business logic services
│   └── infrastructure/         # External dependencies
│       ├── api_client.rs      # Kilometers.ai API client
│       ├── configuration_repository.rs  # Config persistence
│       ├── log_repository.rs  # Log file management
│       ├── process_manager.rs # Process spawning
│       └── event_sender.rs    # Telemetry sending
├── tests/
│   └── init_command_tests.rs  # Integration tests
└── scripts/
    └── version.sh              # Version management
```

### Data Flow

1. **MCP Client** → sends JSON-RPC request to `km` via stdin
2. **km proxy** → logs request to `mcp_proxy.log`
3. **km proxy** → forwards request to target MCP server
4. **MCP Server** → processes request and returns response
5. **km proxy** → logs response and sends telemetry to Kilometers.ai
6. **km proxy** → forwards response back to client via stdout

### File Locations

- **Configuration:** `~/.config/km/config.json`
  ```json
  {
    "api_key": "km_live_..."
  }
  ```

- **Logs:** Platform-specific data directory
  - Linux/macOS: `~/.local/share/km/mcp_proxy.log`
  - Windows: `%APPDATA%\km\mcp_proxy.log`
  
  Log format (JSON Lines):
  ```jsonl
  {"timestamp":"2025-01-09T12:00:00Z","direction":"request","data":{...}}
  {"timestamp":"2025-01-09T12:00:01Z","direction":"response","data":{...}}
  ```

## API Integration

The CLI communicates with the Kilometers.ai API for:
- API key validation during `km init`
- Sending telemetry data during `km monitor`

### Environment Variables

- `KILOMETERS_API_URL`: Override the default API endpoint (default: `https://api.kilometers.ai`)
- `RUST_LOG`: Set logging level (`error`, `warn`, `info`, `debug`, `trace`)
- `NO_COLOR`: Disable colored output

## Troubleshooting

### Common Issues

#### "API key not found" error
```bash
# Re-initialize with your API key
km init
```

#### Permission denied when accessing config
```bash
# Ensure config directory exists and has proper permissions
mkdir -p ~/.config/km
chmod 700 ~/.config/km
```

#### MCP server not responding
```bash
# Test the MCP server directly
npx -y @modelcontextprotocol/server-filesystem ~/Documents

# Check if the server requires specific environment variables
export MCP_SERVER_VAR=value
km monitor -- <command>
```

#### Logs growing too large
```bash
# Clear the logs
km clear-logs

# Or manually remove the log file
rm ~/.local/share/km/mcp_proxy.log
```

### Debug Mode

Enable detailed logging for troubleshooting:
```bash
RUST_LOG=debug km monitor -- <command>
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes
4. Run tests (`cargo test`)
5. Run linter (`cargo clippy`)
6. Format code (`cargo fmt`)
7. Commit your changes
8. Push to your fork
9. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Support

- **Documentation:** [https://kilometers.ai/docs](https://kilometers.ai/docs)
- **Issues:** [GitHub Issues](https://github.com/kilometers-ai/kilometers-cli/issues)
- **Discord:** [Join our community](https://discord.gg/kilometers)