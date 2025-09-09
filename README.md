# Kilometers CLI (`km`)

A Rust-based MCP (Model Context Protocol) proxy that intercepts, logs, and forwards requests between MCP clients and servers while sending telemetry to the Kilometers.ai API.

## Installation

### Automatic Installation (Recommended)

#### Linux/macOS
```bash
curl -fsSL https://raw.githubusercontent.com/kilometers-ai/kilometers-cli/main/install.sh | bash
```

#### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/kilometers-ai/kilometers-cli/main/install.ps1 | iex
```

### Manual Installation

1. Download the appropriate binary for your system from the [latest release](https://github.com/kilometers-ai/kilometers-cli/releases/latest):
   - **Linux x64**: `km-linux-amd64.tar.gz`
   - **Linux ARM64**: `km-linux-arm64.tar.gz`
   - **macOS Intel**: `km-darwin-amd64.tar.gz`
   - **macOS Apple Silicon**: `km-darwin-arm64.tar.gz`
   - **Windows x64**: `km-windows-amd64.exe.zip`

2. Extract the binary and place it in your PATH
3. Make it executable (Linux/macOS): `chmod +x km`

### Build from Source

```bash
git clone https://github.com/kilometers-ai/kilometers-cli.git
cd kilometers-cli
cargo build --release
```

The binary will be available at `target/release/km`.

## Quick Start

1. **Initialize configuration**:
   ```bash
   km init
   ```
   This will prompt you to enter your Kilometers.ai API key.

2. **Start monitoring an MCP server**:
   ```bash
   km monitor -- npx -y @modelcontextprotocol/server-filesystem ~/Documents
   ```

3. **View logs**:
   ```bash
   # Logs are automatically written to your system's data directory:
   # - Linux/macOS: ~/.local/share/km/mcp_proxy.log
   # - Windows: %LOCALAPPDATA%\km\mcp_proxy.log
   ```

4. **Clear logs**:
   ```bash
   km clear-logs
   ```

## Usage

### Commands

- `km init` - Initialize configuration with your API key
- `km monitor -- <command>` - Proxy and monitor an MCP server
- `km clear-logs` - Clear all log files

### Examples

```bash
# Monitor a filesystem MCP server
km monitor -- npx -y @modelcontextprotocol/server-filesystem ~/Documents

# Monitor with custom arguments
km monitor -- python -m my_mcp_server --port 3000

# Monitor a local MCP server binary
km monitor -- ./my-mcp-server
```

## Configuration

Configuration is stored in `~/.config/km/config.json` (or equivalent on Windows).

Example configuration:
```json
{
  "api_key": "your-api-key-here"
}
```

## Architecture

The Kilometers CLI follows a layered architecture:

- **Domain Layer**: Core business models and types
- **Application Layer**: Command implementations and business logic
- **Infrastructure Layer**: External integrations (API, file system, processes)

### Key Components

- **Proxy Service**: Intercepts and forwards MCP JSON-RPC messages
- **Authentication Service**: Manages API key validation
- **Event Sender**: Sends telemetry data to Kilometers.ai
- **Process Manager**: Manages spawned MCP server processes
- **Log Repository**: Handles persistent logging in JSON Lines format

## Development

### Prerequisites

- Rust 1.70+ (install via [rustup](https://rustup.rs/))

### Commands

```bash
# Build
cargo build              # Debug build
cargo build --release    # Release build

# Test
cargo test               # Run all tests
cargo test <test_name>   # Run specific test

# Lint and format
cargo clippy             # Run linter
cargo fmt                # Format code

# Run locally
cargo run -- init                    # Initialize config
cargo run -- monitor -- <command>    # Monitor a command
cargo run -- clear-logs              # Clear logs
```

### Cross-compilation

The project is configured for cross-compilation to multiple targets. See `.github/workflows/release.yml` for the full list of supported platforms.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `cargo clippy` and `cargo fmt`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

For issues and support:
- GitHub Issues: [https://github.com/kilometers-ai/kilometers-cli/issues](https://github.com/kilometers-ai/kilometers-cli/issues)
- Documentation: [https://kilometers.ai/docs](https://kilometers.ai/docs)