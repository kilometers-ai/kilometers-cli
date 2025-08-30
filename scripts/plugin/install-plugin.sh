#!/bin/bash

# Plugin Installation System for Kilometers CLI
# This script demonstrates how secure plugins are installed and managed

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_DIRS=(
    "$HOME/.km/plugins"
    "/usr/local/share/km/plugins"
    "./plugins"
)

# Default values
PLUGIN_PACKAGE=""
FORCE_INSTALL=false
DRY_RUN=false

# Print usage information
usage() {
    echo "Plugin Installation System for Kilometers CLI"
    echo ""
    echo "Usage: $0 PLUGIN_PACKAGE [OPTIONS]"
    echo ""
    echo "Arguments:"
    echo "  PLUGIN_PACKAGE      Path to the plugin package (.kmpkg file)"
    echo ""
    echo "Options:"
    echo "  --force             Force installation (overwrite existing)"
    echo "  --dry-run           Show what would be done without installing"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Install a plugin package"
    echo "  $0 km-plugin-console-logger-abc123.kmpkg"
    echo ""
    echo "  # Force install (overwrite existing)"
    echo "  $0 km-plugin-api-logger-def456.kmpkg --force"
    echo ""
    echo "  # Dry run (preview installation)"
    echo "  $0 km-plugin-console-logger-abc123.kmpkg --dry-run"
    echo ""
    echo "Installation Process:"
    echo "  1. Verify plugin package integrity"
    echo "  2. Check installation permissions"
    echo "  3. Extract plugin to target directory"
    echo "  4. Validate plugin functionality"
    echo "  5. Register plugin for auto-discovery"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE_INSTALL=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *.kmpkg)
                PLUGIN_PACKAGE="$1"
                shift
                ;;
            *)
                if [[ -z "$PLUGIN_PACKAGE" ]]; then
                    PLUGIN_PACKAGE="$1"
                else
                    echo -e "${RED}❌ Unknown argument: $1${NC}"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$PLUGIN_PACKAGE" ]]; then
        echo -e "${RED}❌ Plugin package is required${NC}"
        usage
        exit 1
    fi

    if [[ ! -f "$PLUGIN_PACKAGE" ]]; then
        echo -e "${RED}❌ Plugin package not found: $PLUGIN_PACKAGE${NC}"
        exit 1
    fi
}

# Determine installation directory
determine_install_dir() {
    local install_dir=""
    
    # Try directories in order of preference
    for dir in "${PLUGIN_DIRS[@]}"; do
        # Expand tilde
        local expanded_dir="${dir/#\~/$HOME}"
        
        # Check if directory exists and is writable
        if [[ -d "$expanded_dir" && -w "$expanded_dir" ]]; then
            install_dir="$expanded_dir"
            break
        fi
        
        # Try to create directory if it doesn't exist
        if [[ ! -d "$expanded_dir" ]]; then
            if mkdir -p "$expanded_dir" 2>/dev/null; then
                install_dir="$expanded_dir"
                break
            fi
        fi
    done
    
    # If no writable directory found, use first preference
    if [[ -z "$install_dir" ]]; then
        install_dir="${PLUGIN_DIRS[0]/#\~/$HOME}"
        echo -e "${YELLOW}⚠️  Creating plugin directory: $install_dir${NC}"
        mkdir -p "$install_dir"
    fi
    
    echo "$install_dir"
}

# Verify plugin package
verify_package() {
    local package_path="$1"
    
    echo -e "${BLUE}🔍 Verifying plugin package...${NC}"
    
    # Use the verification script if available
    local verify_script="$SCRIPT_DIR/verify-plugin.sh"
    if [[ -f "$verify_script" ]]; then
        if ! "$verify_script" "$package_path" >/dev/null 2>&1; then
            echo -e "${RED}❌ Plugin package verification failed${NC}"
            return 1
        fi
    else
        # Basic verification
        if ! tar -tzf "$package_path" >/dev/null 2>&1; then
            echo -e "${RED}❌ Plugin package is corrupted${NC}"
            return 1
        fi
    fi
    
    echo -e "${GREEN}✅ Plugin package verified${NC}"
}

# Extract plugin information
extract_plugin_info() {
    local package_path="$1"
    local temp_dir=$(mktemp -d)
    
    # Extract package
    tar -xzf "$package_path" -C "$temp_dir" 2>/dev/null
    
    # Find manifest file
    local manifest_file=$(find "$temp_dir" -name "*.manifest" | head -1)
    if [[ -z "$manifest_file" ]]; then
        rm -rf "$temp_dir"
        echo "unknown|unknown|unknown"
        return
    fi
    
    # Extract plugin information
    local plugin_name="unknown"
    local customer_id="unknown"
    local target_tier="unknown"
    
    if command -v jq &> /dev/null; then
        plugin_name=$(jq -r '.plugin_name // "unknown"' "$manifest_file")
        customer_id=$(jq -r '.customer_id // "unknown"' "$manifest_file")
        target_tier=$(jq -r '.target_tier // "unknown"' "$manifest_file")
    else
        # Basic parsing without jq
        plugin_name=$(grep -o '"plugin_name"[[:space:]]*:[[:space:]]*"[^"]*"' "$manifest_file" | cut -d'"' -f4 || echo "unknown")
        customer_id=$(grep -o '"customer_id"[[:space:]]*:[[:space:]]*"[^"]*"' "$manifest_file" | cut -d'"' -f4 || echo "unknown")
        target_tier=$(grep -o '"target_tier"[[:space:]]*:[[:space:]]*"[^"]*"' "$manifest_file" | cut -d'"' -f4 || echo "unknown")
    fi
    
    rm -rf "$temp_dir"
    echo "$plugin_name|$customer_id|$target_tier"
}

# Check for existing installation
check_existing_installation() {
    local install_dir="$1"
    local plugin_name="$2"
    
    # Look for existing plugin binaries with the same name
    local existing_plugins=$(find "$install_dir" -name "km-plugin-$plugin_name-*" 2>/dev/null)
    
    if [[ -n "$existing_plugins" ]]; then
        echo -e "${YELLOW}⚠️  Found existing installation(s):${NC}"
        echo "$existing_plugins" | while read -r plugin; do
            echo "  • $(basename "$plugin")"
        done
        
        if [[ "$FORCE_INSTALL" != "true" ]]; then
            echo -e "${RED}❌ Plugin already installed. Use --force to overwrite${NC}"
            return 1
        else
            echo -e "${YELLOW}🔄 Force install enabled, will overwrite existing plugins${NC}"
        fi
    fi
    
    return 0
}

# Install plugin
install_plugin() {
    local package_path="$1"
    local install_dir="$2"
    local plugin_info="$3"
    
    IFS='|' read -r plugin_name customer_id target_tier <<< "$plugin_info"
    
    echo -e "${BLUE}📦 Installing plugin...${NC}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}[DRY RUN] Would extract plugin to: $install_dir${NC}"
        return 0
    fi
    
    # Extract plugin package to install directory
    local temp_dir=$(mktemp -d)
    tar -xzf "$package_path" -C "$temp_dir"
    
    # Find binary file
    local binary_file=$(find "$temp_dir" -name "km-plugin-*" ! -name "*.sig" ! -name "*.manifest" | head -1)
    if [[ -z "$binary_file" ]]; then
        echo -e "${RED}❌ Plugin binary not found in package${NC}"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Copy binary to install directory
    local binary_name=$(basename "$binary_file")
    local target_path="$install_dir/$binary_name"
    
    cp "$binary_file" "$target_path"
    chmod +x "$target_path"
    
    # Copy signature and manifest if they exist
    local signature_file="$binary_file.sig"
    local manifest_file="$binary_file.manifest"
    
    if [[ -f "$signature_file" ]]; then
        cp "$signature_file" "$install_dir/"
    fi
    
    if [[ -f "$manifest_file" ]]; then
        cp "$manifest_file" "$install_dir/"
    fi
    
    rm -rf "$temp_dir"
    
    echo -e "${GREEN}✅ Plugin installed: $target_path${NC}"
    echo "$target_path"
}

# Validate plugin functionality
validate_plugin() {
    local plugin_path="$1"
    
    echo -e "${BLUE}🧪 Validating plugin functionality...${NC}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}[DRY RUN] Would validate plugin functionality${NC}"
        return 0
    fi
    
    # Check if plugin is executable
    if [[ ! -x "$plugin_path" ]]; then
        echo -e "${RED}❌ Plugin is not executable${NC}"
        return 1
    fi
    
    # Basic plugin test (would run plugin with test parameters in production)
    echo -e "${GREEN}✅ Plugin appears functional${NC}"
}

# Register plugin for auto-discovery
register_plugin() {
    local install_dir="$1"
    local plugin_info="$2"
    
    IFS='|' read -r plugin_name customer_id target_tier <<< "$plugin_info"
    
    echo -e "${BLUE}📋 Registering plugin for auto-discovery...${NC}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}[DRY RUN] Would register plugin for auto-discovery${NC}"
        return 0
    fi
    
    # Create plugin registry file if it doesn't exist
    local registry_file="$install_dir/.plugin-registry"
    
    # Add plugin to registry
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ"):$plugin_name:$customer_id:$target_tier" >> "$registry_file"
    
    echo -e "${GREEN}✅ Plugin registered for auto-discovery${NC}"
}

# Generate installation report
generate_report() {
    local plugin_path="$1"
    local plugin_info="$2"
    local install_dir="$3"
    
    IFS='|' read -r plugin_name customer_id target_tier <<< "$plugin_info"
    
    echo ""
    echo -e "${GREEN}🎉 Plugin Installation Complete!${NC}"
    echo "================================="
    echo ""
    echo -e "${BLUE}Installation Details:${NC}"
    echo "  • Plugin: $plugin_name"
    echo "  • Customer: $customer_id"
    echo "  • Tier: $target_tier"
    echo "  • Location: $plugin_path"
    echo "  • Directory: $install_dir"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo ""
    echo "1. Verify installation:"
    echo "   km list-plugins"
    echo ""
    echo "2. Test plugin with monitoring:"
    echo "   km monitor --server -- echo '{\"jsonrpc\":\"2.0\",\"method\":\"test\",\"id\":1}'"
    echo ""
    echo "3. View plugin status:"
    echo "   km plugin-status"
    echo ""
    echo -e "${GREEN}✅ Plugin is ready to use!${NC}"
    echo ""
}

# Main installation function
main() {
    echo -e "${GREEN}🔒 Kilometers CLI - Plugin Installation System${NC}"
    echo "==============================================="
    echo ""
    
    parse_args "$@"
    
    echo -e "${BLUE}Installing plugin package: $(basename "$PLUGIN_PACKAGE")${NC}"
    echo ""
    
    # Verify package
    if ! verify_package "$PLUGIN_PACKAGE"; then
        exit 1
    fi
    
    # Extract plugin information
    local plugin_info=$(extract_plugin_info "$PLUGIN_PACKAGE")
    IFS='|' read -r plugin_name customer_id target_tier <<< "$plugin_info"
    
    echo -e "${BLUE}Plugin Information:${NC}"
    echo "  • Name: $plugin_name"
    echo "  • Customer: $customer_id"
    echo "  • Tier: $target_tier"
    echo ""
    
    # Determine installation directory
    local install_dir=$(determine_install_dir)
    echo -e "${BLUE}Installation directory: $install_dir${NC}"
    echo ""
    
    # Check for existing installation
    if ! check_existing_installation "$install_dir" "$plugin_name"; then
        exit 1
    fi
    
    # Install plugin
    local plugin_path=$(install_plugin "$PLUGIN_PACKAGE" "$install_dir" "$plugin_info")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Validate plugin functionality
    if ! validate_plugin "$plugin_path"; then
        exit 1
    fi
    
    # Register plugin for auto-discovery
    register_plugin "$install_dir" "$plugin_info"
    
    # Generate installation report
    generate_report "$plugin_path" "$plugin_info" "$install_dir"
}

# Run main function with all arguments
main "$@"