#!/bin/sh
set -e

# Kilometers CLI installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/kilometers-ai/kilometers-cli/main/scripts/install.sh | sh

REPO="kilometers-ai/kilometers-cli"
BINARY_NAME="km"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    darwin)
        OS="apple-darwin"
        ;;
    linux)
        OS="unknown-linux-gnu"
        ;;
    *)
        echo "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH="x86_64"
        ;;
    arm64|aarch64)
        ARCH="aarch64"
        ;;
    *)
        echo "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

PLATFORM="${ARCH}-${OS}"

# Get latest release
echo "Fetching latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo "${RED}Failed to fetch latest release${NC}"
    exit 1
fi

echo "Latest release: ${GREEN}$LATEST_RELEASE${NC}"

# Download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/km-$PLATFORM.tar.gz"

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download binary
echo "Downloading $BINARY_NAME for $PLATFORM..."
if ! curl -L -o "$TMP_DIR/$BINARY_NAME.tar.gz" "$DOWNLOAD_URL"; then
    echo "${RED}Failed to download binary${NC}"
    echo "URL: $DOWNLOAD_URL"
    exit 1
fi

# Extract binary
echo "Extracting binary..."
tar -xzf "$TMP_DIR/$BINARY_NAME.tar.gz" -C "$TMP_DIR"

# Check if binary exists
if [ ! -f "$TMP_DIR/$BINARY_NAME" ]; then
    echo "${RED}Binary not found in archive${NC}"
    exit 1
fi

# Make binary executable
chmod +x "$TMP_DIR/$BINARY_NAME"

# Install to /usr/local/bin or ~/.local/bin
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing $BINARY_NAME to $INSTALL_DIR..."
mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

# Add to PATH if necessary
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "${YELLOW}Note: $INSTALL_DIR is not in your PATH${NC}"
    echo "Add the following to your shell configuration file:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

# Verify installation
if command -v km >/dev/null 2>&1; then
    echo "${GREEN}✓ Kilometers CLI installed successfully!${NC}"
    km --version
else
    echo "${GREEN}✓ Kilometers CLI installed to $INSTALL_DIR${NC}"
    echo "Run the following to verify:"
    echo "  $INSTALL_DIR/km --version"
fi

echo ""
echo "Get started with:"
echo "  km init        # Initialize with your API key"
echo "  km --help      # Show available commands"