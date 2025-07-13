#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="km"
VERSION="dev-local"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo -e "${BLUE}🔧 Kilometers CLI - Local Development Setup${NC}"
echo "=================================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go found:${NC} $(go version)"
echo ""

# Step 1: Clean previous builds
echo -e "${YELLOW}🧹 Cleaning previous builds...${NC}"
rm -f ./main
rm -f ./$BINARY_NAME
rm -rf ./dist

# Step 2: Download dependencies
echo -e "${YELLOW}📦 Downloading dependencies...${NC}"
go mod download
go mod tidy

# Step 3: Run tests
echo -e "${YELLOW}🧪 Running tests...${NC}"
if go test ./internal/...; then
    echo -e "${GREEN}✅ All tests passed${NC}"
else
    echo -e "${RED}❌ Tests failed. Please fix them before proceeding.${NC}"
    exit 1
fi

# Step 4: Build the binary
echo -e "${YELLOW}🔨 Building binary...${NC}"
ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

# Build from cmd directory since that's where main.go is
if go build -ldflags="$ldflags" -o "$BINARY_NAME" ./cmd; then
    echo -e "${GREEN}✅ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Step 5: Install the binary
echo -e "${YELLOW}📥 Installing binary...${NC}"

# Try to install in $GOPATH/bin first, then fallback to /usr/local/bin
if [ -n "$GOPATH" ] && [ -d "$GOPATH/bin" ]; then
    INSTALL_DIR="$GOPATH/bin"
elif [ -d "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    # Create local bin directory
    mkdir -p "$HOME/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
fi

# Copy binary to install directory
if cp "$BINARY_NAME" "$INSTALL_DIR/"; then
    echo -e "${GREEN}✅ Binary installed to: $INSTALL_DIR/$BINARY_NAME${NC}"
else
    echo -e "${RED}❌ Failed to install binary. Trying with sudo...${NC}"
    if sudo cp "$BINARY_NAME" "$INSTALL_DIR/"; then
        echo -e "${GREEN}✅ Binary installed to: $INSTALL_DIR/$BINARY_NAME${NC}"
    else
        echo -e "${RED}❌ Failed to install binary${NC}"
        exit 1
    fi
fi

# Step 6: Verify installation
echo -e "${YELLOW}🔍 Verifying installation...${NC}"
if command -v $BINARY_NAME &> /dev/null; then
    echo -e "${GREEN}✅ Binary is accessible in PATH${NC}"
    $BINARY_NAME --version
else
    echo -e "${YELLOW}⚠️  Binary not in PATH. You may need to add $INSTALL_DIR to your PATH${NC}"
    echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"$INSTALL_DIR:\$PATH\""
fi

# Step 7: Create test configuration directory
echo -e "${YELLOW}📁 Setting up configuration directory...${NC}"
CONFIG_DIR="$HOME/.km"
CONFIG_FILE="$CONFIG_DIR/config.json"

mkdir -p "$CONFIG_DIR"

# Create a test configuration if it doesn't exist
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${YELLOW}📝 Creating test configuration...${NC}"
    cat > "$CONFIG_FILE" << EOF
{
  "api_key": "test-api-key-for-local-development",
  "api_endpoint": "https://api.dev.kilometers.ai",
  "batch_size": 10,
  "upload_enabled": false,
  "debug_mode": true,
  "monitoring": {
    "buffer_size": 1048576,
    "timeout_seconds": 30,
    "max_events": 1000
  }
}
EOF
    echo -e "${GREEN}✅ Test configuration created at: $CONFIG_FILE${NC}"
else
    echo -e "${GREEN}✅ Configuration already exists at: $CONFIG_FILE${NC}"
fi

# Step 8: Test the CLI
echo -e "${YELLOW}🧪 Testing CLI functionality...${NC}"
echo ""

echo "Testing basic commands:"
echo "----------------------"

# Test version
echo -n "Version check: "
if $BINARY_NAME --version &> /dev/null; then
    echo -e "${GREEN}✅${NC}"
else
    echo -e "${RED}❌${NC}"
fi

# Test help
echo -n "Help command: "
if $BINARY_NAME --help &> /dev/null; then
    echo -e "${GREEN}✅${NC}"
else
    echo -e "${RED}❌${NC}"
fi

# Test config command
echo -n "Config command: "
if $BINARY_NAME config --help &> /dev/null; then
    echo -e "${GREEN}✅${NC}"
else
    echo -e "${RED}❌${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Setup Complete!${NC}"
echo "=================="
echo ""
echo -e "${BLUE}Available Commands:${NC}"
echo "  $BINARY_NAME --help              # Show all available commands"
echo "  $BINARY_NAME --version           # Show version information"
echo "  $BINARY_NAME config              # Manage configuration"
echo "  $BINARY_NAME init                # Initialize configuration interactively"
echo "  $BINARY_NAME setup <assistant>   # Set up AI assistant integration"
echo "  $BINARY_NAME monitor <path>      # Monitor MCP server process"
echo "  $BINARY_NAME validate            # Validate configuration"
echo ""
echo -e "${BLUE}Testing with Mock Server:${NC}"
echo "  go run test/cmd/run_mock_server.go   # Start mock MCP server"
echo "  $BINARY_NAME monitor --debug     # Monitor with debug output"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "  Config file: $CONFIG_FILE"
echo "  Binary location: $INSTALL_DIR/$BINARY_NAME"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Run '$BINARY_NAME init' to set up your configuration interactively"
echo "2. Or edit $CONFIG_FILE directly"
echo "3. Test with: 'go run test/cmd/run_mock_server.go' in another terminal"
echo "4. Then run: '$BINARY_NAME monitor --debug /path/to/mcp/server'"
echo ""
echo -e "${GREEN}Happy testing! 🚀${NC}" 