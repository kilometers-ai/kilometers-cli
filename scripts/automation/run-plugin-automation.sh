#!/bin/bash

# Simple runner script for plugin system automation
# Provides easy access to different automation modes

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAIN_SCRIPT="$SCRIPT_DIR/cleanup-and-test-plugins.sh"

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_usage() {
    echo "Kilometers CLI Plugin System Automation Runner"
    echo ""
    echo "Usage: $0 [MODE] [OPTIONS]"
    echo ""
    echo "Modes:"
    echo "  full        Run complete automation suite (default)"
    echo "  quick       Quick test without full cleanup"
    echo "  cleanup     Only remove prebuilt binaries"
    echo "  build       Only build fresh plugins"
    echo "  test        Only run plugin tests"
    echo "  demo        Run demo commands for development"
    echo ""
    echo "Options:"
    echo "  --help      Show this help message"
    echo "  --verbose   Enable verbose output"
    echo "  --dry-run   Show what would be done without executing"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run full automation"
    echo "  $0 quick             # Quick validation"
    echo "  $0 cleanup           # Clean binaries only"
    echo "  $0 demo --verbose    # Demo with verbose output"
}

run_full_automation() {
    print_header "Running Full Plugin Automation"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would execute: $MAIN_SCRIPT"
        return 0
    fi
    
    "$MAIN_SCRIPT"
}

run_quick_test() {
    print_header "Running Quick Plugin Test"
    
    cd "$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would execute quick test commands"
        return 0
    fi
    
    echo -e "${YELLOW}Building CLI binary...${NC}"
    go build -o km ./cmd/main.go
    
    echo -e "${YELLOW}Testing basic functionality...${NC}"
    ./km --help > /dev/null
    ./km plugins --help > /dev/null
    
    echo -e "${YELLOW}Testing plugin commands...${NC}"
    ./km plugins list
    
    if [ -n "$KM_API_KEY" ]; then
        echo -e "${YELLOW}Testing with API key...${NC}"
        ./km plugins status
    else
        echo -e "${YELLOW}Skipping API tests (no KM_API_KEY set)${NC}"
    fi
    
    echo -e "${GREEN}✅ Quick test completed successfully!${NC}"
}

run_cleanup_only() {
    print_header "Running Binary Cleanup Only"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would remove prebuilt binaries from:"
        echo "  - kilometers-cli repository"
        echo "  - kilometers-cli-plugins repository" 
        echo "  - User plugin directory (~/.km/plugins/)"
        return 0
    fi
    
    cd "$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    echo -e "${YELLOW}Removing CLI binaries...${NC}"
    rm -f km km-premium build/km
    
    echo -e "${YELLOW}Removing plugin packages...${NC}"
    PLUGINS_DIR="/Users/milesangelo/Source/active/kilometers.ai/kilometers-cli-plugins"
    if [ -d "$PLUGINS_DIR/dist-standalone" ]; then
        rm -f "$PLUGINS_DIR/dist-standalone"/*.kmpkg
    fi
    
    echo -e "${YELLOW}Checking user plugins...${NC}"
    if [ -d "$HOME/.km/plugins" ]; then
        echo "Found user plugins:"
        ls -la "$HOME/.km/plugins/"
        echo ""
        echo -e "${YELLOW}Remove user plugins? (y/n):${NC}"
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            rm -f "$HOME/.km/plugins"/km-plugin-*
            echo -e "${GREEN}✅ User plugins removed${NC}"
        fi
    fi
    
    echo -e "${GREEN}✅ Cleanup completed!${NC}"
}

run_build_only() {
    print_header "Running Plugin Build Only"
    
    cd "$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would build:"
        echo "  - CLI binary (km)"
        echo "  - Console logger plugin"
        echo "  - Plugin packages (if available)"
        return 0
    fi
    
    echo -e "${YELLOW}Building CLI binary...${NC}"
    go build -o km ./cmd/main.go
    
    echo -e "${YELLOW}Note: Plugin examples moved to kilometers-cli-plugins repository${NC}"
    echo -e "${YELLOW}Use the dedicated plugins repository for plugin development${NC}"
    
    echo -e "${GREEN}✅ Build completed!${NC}"
}

run_test_only() {
    print_header "Running Plugin Tests Only"
    
    cd "$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run plugin command tests"
        return 0
    fi
    
    if [ ! -f "km" ]; then
        echo -e "${YELLOW}CLI binary not found, building...${NC}"
        go build -o km ./cmd/main.go
    fi
    
    echo -e "${YELLOW}Testing plugin commands...${NC}"
    
    echo "• Testing plugins list"
    ./km plugins list
    
    echo "• Testing plugins status"  
    ./km plugins status
    
    if [ -n "$KM_API_KEY" ]; then
        echo "• Testing with API key"
        ./km plugins refresh || echo "  (Expected failure without real API)"
    fi
    
    echo "• Testing monitoring integration"
    timeout 3s ./km monitor --server -- echo '{"jsonrpc":"2.0","method":"test","id":1}' || true
    
    echo -e "${GREEN}✅ Tests completed!${NC}"
}

run_demo() {
    print_header "Running Plugin System Demo"
    
    cd "$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run demo commands"
        return 0
    fi
    
    if [ ! -f "km" ]; then
        echo -e "${YELLOW}Building CLI binary for demo...${NC}"
        go build -o km ./cmd/main.go
    fi
    
    echo -e "${YELLOW}Plugin System Demo Commands:${NC}"
    echo ""
    
    echo -e "${BLUE}1. Basic plugin list (no API key):${NC}"
    ./km plugins list
    echo ""
    
    echo -e "${BLUE}2. Plugin status (no API key):${NC}"
    ./km plugins status
    echo ""
    
    if [ -n "$KM_API_KEY" ]; then
        echo -e "${BLUE}3. Plugin list with API key:${NC}"
        ./km plugins list
        echo ""
        
        echo -e "${BLUE}4. Plugin status with API key:${NC}"
        ./km plugins status
        echo ""
        
        echo -e "${BLUE}5. Monitoring with plugin integration:${NC}"
        timeout 3s ./km monitor --server -- echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' || echo "  → Monitoring completed (timeout expected)"
    else
        echo -e "${BLUE}3. Set KM_API_KEY for API-based tests:${NC}"
        echo "   export KM_API_KEY=your-test-key"
        echo "   $0 demo"
    fi
    
    echo ""
    echo -e "${GREEN}✅ Demo completed!${NC}"
    echo ""
    echo "Next steps:"
    echo "• Try with real API key for full plugin functionality"
    echo "• Test with real MCP servers: npx -y @modelcontextprotocol/server-github"
    echo "• Install plugin packages: km plugins install package.kmpkg"
}

# Parse command line arguments
MODE="full"
VERBOSE=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        full|quick|cleanup|build|test|demo)
            MODE="$1"
            shift
            ;;
        --help|-h)
            print_usage
            exit 0
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            print_usage
            exit 1
            ;;
    esac
done

# Enable verbose output if requested
if [ "$VERBOSE" = "true" ]; then
    set -x
fi

# Check if main script exists
if [ ! -f "$MAIN_SCRIPT" ]; then
    echo -e "${RED}Main automation script not found: $MAIN_SCRIPT${NC}"
    exit 1
fi

# Execute the requested mode
case $MODE in
    full)
        run_full_automation
        ;;
    quick)
        run_quick_test
        ;;
    cleanup)
        run_cleanup_only
        ;;
    build)
        run_build_only
        ;;
    test)
        run_test_only
        ;;
    demo)
        run_demo
        ;;
    *)
        echo -e "${RED}Unknown mode: $MODE${NC}"
        print_usage
        exit 1
        ;;
esac
