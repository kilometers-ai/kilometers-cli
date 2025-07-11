#!/bin/sh
# Kilometers.ai CLI Installer
# 
# Installation:
#   curl -sSL https://get.kilometers.ai/install.sh | sh
#
# Downloads binaries from Kilometers.ai CDN for maximum reliability

set -e

# Configuration
BINARY_NAME="km"
GITHUB_REPO="kilometers-ai/kilometers-cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
    exit 1
}

success() {
    printf "${BLUE}[SUCCESS]${NC} %s\n" "$1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
    
    case $ARCH in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get user-friendly installation options based on OS
get_install_options() {
    case $OS in
        darwin)
            # macOS options - prioritize existing directories that are in PATH
            if [ -d "$HOME/bin" ] && case ":$PATH:" in *":$HOME/bin:"*) true;; *) false;; esac; then
                # ~/bin exists and is in PATH - perfect!
                OPTION_1="$HOME/bin (recommended - no sudo required, already in PATH)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="Custom location"
                VALUE_1="$HOME/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="custom"
                OPTION_COUNT=4
            elif [ -d "$HOME/.local/bin" ] && case ":$PATH:" in *":$HOME/.local/bin:"*) true;; *) false;; esac; then
                # ~/.local/bin exists and is in PATH - great!
                OPTION_1="$HOME/.local/bin (recommended - no sudo required, already in PATH)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/bin (no sudo required)"
                OPTION_4="Custom location"
                VALUE_1="$HOME/.local/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/bin"
                VALUE_4="custom"
                OPTION_COUNT=4
            elif [ -d "$HOME/bin" ]; then
                # ~/bin exists but not in PATH
                OPTION_1="$HOME/bin (recommended - no sudo required)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="Custom location"
                VALUE_1="$HOME/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="custom"
                OPTION_COUNT=4
            elif [ -d "$HOME/.local/bin" ]; then
                # ~/.local/bin exists but not in PATH
                OPTION_1="$HOME/.local/bin (recommended - no sudo required)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/bin (no sudo required)"
                OPTION_4="Custom location"
                VALUE_1="$HOME/.local/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/bin"
                VALUE_4="custom"
                OPTION_COUNT=4
            else
                # Default macOS - /usr/local/bin is standard and usually exists
                OPTION_1="/usr/local/bin (recommended - requires sudo)"
                OPTION_2="$HOME/bin (no sudo required)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="Custom location"
                VALUE_1="/usr/local/bin"
                VALUE_2="$HOME/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="custom"
                OPTION_COUNT=4
            fi
            ;;
        linux)
            # Linux options - prioritize existing directories that are in PATH
            if [ -d "$HOME/bin" ] && case ":$PATH:" in *":$HOME/bin:"*) true;; *) false;; esac; then
                # ~/bin exists and is in PATH - perfect!
                OPTION_1="$HOME/bin (recommended - no sudo required, already in PATH)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="/opt/kilometers/bin (requires sudo)"
                OPTION_5="Custom location"
                VALUE_1="$HOME/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="/opt/kilometers/bin"
                VALUE_5="custom"
                OPTION_COUNT=5
            elif [ -d "$HOME/.local/bin" ] && case ":$PATH:" in *":$HOME/.local/bin:"*) true;; *) false;; esac; then
                # ~/.local/bin exists and is in PATH - great!
                OPTION_1="$HOME/.local/bin (recommended - no sudo required, already in PATH)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/bin (no sudo required)"
                OPTION_4="/opt/kilometers/bin (requires sudo)"
                OPTION_5="Custom location"
                VALUE_1="$HOME/.local/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/bin"
                VALUE_4="/opt/kilometers/bin"
                VALUE_5="custom"
                OPTION_COUNT=5
            elif [ -d "$HOME/bin" ]; then
                # ~/bin exists but not in PATH
                OPTION_1="$HOME/bin (recommended - no sudo required)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="/opt/kilometers/bin (requires sudo)"
                OPTION_5="Custom location"
                VALUE_1="$HOME/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="/opt/kilometers/bin"
                VALUE_5="custom"
                OPTION_COUNT=5
            elif [ -d "$HOME/.local/bin" ]; then
                # ~/.local/bin exists but not in PATH
                OPTION_1="$HOME/.local/bin (recommended - no sudo required)"
                OPTION_2="/usr/local/bin (requires sudo)"
                OPTION_3="$HOME/bin (no sudo required)"
                OPTION_4="/opt/kilometers/bin (requires sudo)"
                OPTION_5="Custom location"
                VALUE_1="$HOME/.local/bin"
                VALUE_2="/usr/local/bin"
                VALUE_3="$HOME/bin"
                VALUE_4="/opt/kilometers/bin"
                VALUE_5="custom"
                OPTION_COUNT=5
            else
                # Default Linux - /usr/local/bin is most common and usually exists
                OPTION_1="/usr/local/bin (recommended - requires sudo)"
                OPTION_2="$HOME/bin (no sudo required)"
                OPTION_3="$HOME/.local/bin (no sudo required)"
                OPTION_4="/opt/kilometers/bin (requires sudo)"
                OPTION_5="Custom location"
                VALUE_1="/usr/local/bin"
                VALUE_2="$HOME/bin"
                VALUE_3="$HOME/.local/bin"
                VALUE_4="/opt/kilometers/bin"
                VALUE_5="custom"
                OPTION_COUNT=5
            fi
            ;;
        windows)
            # Windows options (though this script is primarily for Unix-like systems)
            OPTION_1="$HOME/.local/bin (recommended)"
            OPTION_2="C:\\Program Files\\Kilometers\\bin (requires admin)"
            OPTION_3="Custom location"
            VALUE_1="$HOME/.local/bin"
            VALUE_2="C:\\Program Files\\Kilometers\\bin"
            VALUE_3="custom"
            OPTION_COUNT=3
            ;;
    esac
}

# Select installation location with user choice
select_install_location() {
    get_install_options
    
    echo ""
    printf "${BLUE}Choose installation location:${NC}\n"
    echo ""
    
    # Display options
    i=1
    while [ $i -le $OPTION_COUNT ]; do
        eval "option=\$OPTION_$i"
        printf "  %d) %s\n" $i "$option"
        i=$((i + 1))
    done
    echo ""
    
    # Get user input
    while true; do
        printf "Enter your choice (1-%d): " $OPTION_COUNT
        read -r choice
        
        # Validate input - POSIX compatible number check
        case "$choice" in
            *[!0-9]*)
                printf "${RED}Invalid choice. Please enter a number between 1 and %d.${NC}\n" $OPTION_COUNT
                continue
                ;;
        esac
        
        if [ "$choice" -ge 1 ] && [ "$choice" -le $OPTION_COUNT ]; then
            eval "INSTALL_DIR=\$VALUE_$choice"
            
            # Handle custom location
            if [ "$INSTALL_DIR" = "custom" ]; then
                echo ""
                printf "Enter custom installation path: "
                read -r INSTALL_DIR
                
                # Validate custom path
                if [ -z "$INSTALL_DIR" ]; then
                    error "Installation path cannot be empty"
                fi
            fi
            
            # Check if directory exists, create if needed
            if [ ! -d "$INSTALL_DIR" ]; then
                printf "Directory %s does not exist. Create it? (y/N): " "$INSTALL_DIR"
                read -r create_dir
                case "$create_dir" in
                    [Yy]*)
                        mkdir -p "$INSTALL_DIR" || error "Failed to create directory $INSTALL_DIR"
                        ;;
                    *)
                        error "Installation cancelled"
                        ;;
                esac
            fi
            
            # Check write permissions
            if [ -w "$INSTALL_DIR" ]; then
                SUDO=""
                info "Installing to $INSTALL_DIR (no sudo required)"
            else
                if command -v sudo >/dev/null 2>&1; then
                    printf "Directory %s requires sudo. Continue? (y/N): " "$INSTALL_DIR"
                    read -r use_sudo
                    case "$use_sudo" in
                        [Yy]*)
                            SUDO="sudo"
                            info "Installing to $INSTALL_DIR (requires sudo)"
                            ;;
                        *)
                            echo "Please choose a different location."
                            continue
                            ;;
                    esac
                else
                    error "Cannot write to $INSTALL_DIR and sudo is not available. Please choose a different location."
                fi
            fi
            
            break
        else
            printf "${RED}Invalid choice. Please enter a number between 1 and %d.${NC}\n" $OPTION_COUNT
        fi
    done
}

# Download the binary from GitHub Releases (most reliable)
download_binary() {
    BINARY_FILE="km-${PLATFORM}"
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_FILE}.exe"
    fi
    
    TEMP_FILE="/tmp/km-download-$$"
    
    # Use GitHub releases as primary source (most reliable)
    GITHUB_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_FILE}"
    CDN_URL="https://get.kilometers.ai/releases/latest/${BINARY_FILE}"
    
    info "Downloading Kilometers CLI..."
    
    # Try CDN first, fallback to GitHub
    if command -v curl >/dev/null 2>&1; then
        info "Source: ${CDN_URL} (trying CDN first)"
        if curl -L --fail -o "$TEMP_FILE" "$CDN_URL" 2>/dev/null; then
            success "Download completed successfully from CDN"
        else
            warn "CDN download failed, trying GitHub releases..."
            info "Source: ${GITHUB_URL}"
            if curl -L --fail -o "$TEMP_FILE" "$GITHUB_URL" 2>/dev/null; then
                success "Download completed successfully from GitHub"
            else
                error "Download failed from both CDN and GitHub. Please check your internet connection or try again later."
            fi
        fi
    elif command -v wget >/dev/null 2>&1; then
        info "Source: ${CDN_URL} (trying CDN first)"
        if wget -O "$TEMP_FILE" "$CDN_URL" 2>/dev/null; then
            success "Download completed successfully from CDN"
        else
            warn "CDN download failed, trying GitHub releases..."
            info "Source: ${GITHUB_URL}"
            if wget -O "$TEMP_FILE" "$GITHUB_URL" 2>/dev/null; then
                success "Download completed successfully from GitHub"
            else
                error "Download failed from both CDN and GitHub. Please check your internet connection or try again later."
            fi
        fi
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
    
    # Verify download
    if [ ! -f "$TEMP_FILE" ] || [ ! -s "$TEMP_FILE" ]; then
        error "Downloaded file is missing or empty"
    fi
}

# Install the binary
install_binary() {
    info "Installing Kilometers CLI..."
    
    chmod +x "$TEMP_FILE"
    $SUDO mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        success "Installation completed successfully!"
    else
        warn "Binary installed but not found in PATH"
        warn "You may need to add $INSTALL_DIR to your PATH"
    fi
}

# Post-installation setup and info
post_install() {
    echo ""
    success "🎉 Kilometers CLI installed successfully!"
    echo ""
    
    # Show version if available
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        VERSION=$("$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        info "Installed version: $VERSION"
        echo ""
    fi

    # Check if INSTALL_DIR is in PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*)
            # Already in PATH, do nothing
            ;;
        *)
            # Not in PATH
            printf "${YELLOW}Warning: $INSTALL_DIR is not in your PATH.${NC}\n"
            echo ""
            # Only prompt if not in non-interactive mode
            if [ "${NON_INTERACTIVE:-false}" != "true" ]; then
                # Detect shell and rc file
                DETECTED_SHELL="$(basename "$SHELL")"
                case "$DETECTED_SHELL" in
                    zsh)
                        RC_FILE="$HOME/.zshrc"
                        ;;
                    bash)
                        RC_FILE="$HOME/.bashrc"
                        # On macOS, .bash_profile is sometimes used
                        if [ "$OS" = "darwin" ] && [ -f "$HOME/.bash_profile" ]; then
                            RC_FILE="$HOME/.bash_profile"
                        fi
                        ;;
                    ksh)
                        RC_FILE="$HOME/.kshrc"
                        ;;
                    fish)
                        RC_FILE="$HOME/.config/fish/config.fish"
                        ;;
                    *)
                        RC_FILE="$HOME/.profile"
                        ;;
                esac
                printf "Would you like to add $INSTALL_DIR to your PATH in $RC_FILE? (y/N): "
                read -r add_path
                case "$add_path" in
                    [Yy]*)
                        # Only add if not already present
                        if ! grep -q "export PATH=\"$INSTALL_DIR:\$PATH\"" "$RC_FILE" 2>/dev/null; then
                            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$RC_FILE"
                            printf "${GREEN}Added to PATH in $RC_FILE. Please restart your terminal or run: source $RC_FILE${NC}\n"
                        else
                            printf "${YELLOW}$RC_FILE already contains the export line for $INSTALL_DIR${NC}\n"
                        fi
                        ;;
                    *)
                        printf "${YELLOW}You can add it manually by adding this line to $RC_FILE:${NC}\n"
                        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
                        ;;
                esac
                echo ""
            else
                printf "${YELLOW}To add to PATH, add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):${NC}\n"
                echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
                echo ""
            fi
            ;;
    esac
    
    # Show quick start guide
    printf "${BLUE}Quick Start:${NC}\n"
    echo "  1. Set your API key:"
    echo "     export KILOMETERS_API_KEY=\"your-api-key-here\""
    echo ""
    echo "  2. Wrap your MCP server:"
    echo "     km npx @modelcontextprotocol/server-github"
    echo ""
    echo "  3. View debug logs:"
    echo "     export KM_DEBUG=true"
    echo ""
    
    printf "${BLUE}Resources:${NC}\n"
    echo "  • Dashboard: https://app.kilometers.ai"
    echo "  • Documentation: https://docs.kilometers.ai"
    echo "  • Support: support@kilometers.ai"
    echo ""
    
    printf "${GREEN}✓ Ready to monitor your AI tools!${NC}\n"
}

# Main installation flow
main() {
    echo ""
    printf "${BLUE}Kilometers.ai CLI Installer${NC}\n"
    echo "==============================="
    echo ""
    
    # Handle help first
    case "${1:-}" in
        --help)
            echo "Usage: $0 [--help] [--non-interactive] [--install-dir DIR]"
            echo ""
            echo "Install Kilometers.ai CLI:"
            echo "  curl -sSL https://get.kilometers.ai/install.sh | sh"
            echo ""
            echo "Options:"
            echo "  --non-interactive    Install to default location without prompts"
            echo "  --install-dir DIR   Install to specific directory"
            echo ""
            echo "Downloads binaries from Kilometers.ai CDN for reliability."
            exit 0
            ;;
    esac
    
    # Detect platform first (needed for non-interactive mode)
    detect_platform
    
    # Handle other options after platform detection
    case "${1:-}" in
        --non-interactive)
            NON_INTERACTIVE=true
            # Use default user-local directory
            case $OS in
                darwin|linux)
                    INSTALL_DIR="$HOME/.local/bin"
                    ;;
                windows)
                    INSTALL_DIR="$HOME/.local/bin"
                    ;;
            esac
            # Create directory if it doesn't exist
            mkdir -p "$INSTALL_DIR" || error "Failed to create directory $INSTALL_DIR"
            SUDO=""
            info "Installing to $INSTALL_DIR (non-interactive mode)"
            ;;
        --install-dir)
            if [ -z "${2:-}" ]; then
                error "Directory path required for --install-dir option"
            fi
            INSTALL_DIR="$2"
            # Create directory if it doesn't exist
            mkdir -p "$INSTALL_DIR" || error "Failed to create directory $INSTALL_DIR"
            # Check write permissions
            if [ -w "$INSTALL_DIR" ]; then
                SUDO=""
                info "Installing to $INSTALL_DIR (no sudo required)"
            else
                if command -v sudo >/dev/null 2>&1; then
                    SUDO="sudo"
                    info "Installing to $INSTALL_DIR (requires sudo)"
                else
                    error "Cannot write to $INSTALL_DIR and sudo is not available"
                fi
            fi
            # Skip the next argument (the directory path)
            shift
            ;;
    esac
    
    # Auto-detect non-interactive mode (e.g., when piped from curl)
    if [ -t 0 ] && [ "${NON_INTERACTIVE:-false}" != "true" ] && [ -z "${INSTALL_DIR:-}" ]; then
        # Interactive mode - show options
        select_install_location
    elif [ -z "${INSTALL_DIR:-}" ]; then
        # Non-interactive mode - use smart defaults
        NON_INTERACTIVE=true
        case $OS in
            darwin)
                # macOS: prefer ~/.local/bin, fallback to ~/bin
                if [ -d "$HOME/.local/bin" ]; then
                    INSTALL_DIR="$HOME/.local/bin"
                elif [ -d "$HOME/bin" ]; then
                    INSTALL_DIR="$HOME/bin"
                else
                    INSTALL_DIR="$HOME/.local/bin"
                fi
                ;;
            linux)
                # Linux: prefer existing directories in PATH, fallback to ~/.local/bin
                if [ -d "$HOME/bin" ] && case ":$PATH:" in *":$HOME/bin:"*) true;; *) false;; esac; then
                    INSTALL_DIR="$HOME/bin"
                elif [ -d "$HOME/.local/bin" ] && case ":$PATH:" in *":$HOME/.local/bin:"*) true;; *) false;; esac; then
                    INSTALL_DIR="$HOME/.local/bin"
                elif [ -d "$HOME/bin" ]; then
                    INSTALL_DIR="$HOME/bin"
                elif [ -d "$HOME/.local/bin" ]; then
                    INSTALL_DIR="$HOME/.local/bin"
                else
                    INSTALL_DIR="$HOME/.local/bin"
                fi
                ;;
            windows)
                INSTALL_DIR="$HOME/.local/bin"
                ;;
        esac
        # Create directory if it doesn't exist
        mkdir -p "$INSTALL_DIR" || error "Failed to create directory $INSTALL_DIR"
        SUDO=""
        info "Installing to $INSTALL_DIR (auto-detected non-interactive mode)"
    fi
    
    download_binary
    install_binary
    post_install
}

# Run main function
main "$@" 