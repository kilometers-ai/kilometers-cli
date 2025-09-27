#!/bin/bash

# Basic MCP Test Script - Direct connection without km proxy for comparison
# Tests the MCP everything server directly, then shows how to use km proxy

set -e

echo "🔧 Basic MCP Server Test"
echo "========================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}First, let's test the MCP server directly (without proxy):${NC}"
echo "--------------------------------------------------------"

# Create a temp file with test requests
TEMP_FILE=$(mktemp)
cat > $TEMP_FILE << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello MCP!"}}}
EOF

echo "Test requests:"
cat $TEMP_FILE
echo ""

echo -e "${BLUE}Testing direct connection to MCP everything server...${NC}"
timeout 15s bash -c 'cat '"$TEMP_FILE"' | npx -y @modelcontextprotocol/server-everything' || echo -e "${YELLOW}Direct test completed/timed out${NC}"

echo ""
echo -e "${YELLOW}Now let's show the km proxy command structure:${NC}"
echo "------------------------------------------------"

# Check if km is built
if [ ! -f "target/debug/km" ] && [ ! -f "target/release/km" ]; then
    echo -e "${YELLOW}Building km...${NC}"
    cargo build
fi

# Determine which binary to use
KM_BINARY="./target/debug/km"
if [ -f "target/release/km" ]; then
    KM_BINARY="./target/release/km"
fi

echo -e "${BLUE}km binary available: $KM_BINARY${NC}"

echo ""
echo -e "${GREEN}How to use km proxy:${NC}"
echo "1. Direct proxy command:"
echo "   $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"
echo ""
echo "2. Send JSON-RPC via stdin:"
echo "   echo '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}' | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"
echo ""
echo "3. Use a file with multiple requests:"
echo "   cat requests.json | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"

echo ""
echo -e "${GREEN}Sample MCP JSON-RPC requests:${NC}"
echo "----------------------------"

echo -e "${BLUE}Initialize:${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}}}'

echo -e "${BLUE}Initialized notification:${NC}"
echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'

echo -e "${BLUE}List tools:${NC}"
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'

echo -e "${BLUE}Call echo tool:${NC}"
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello km!"}}}'

echo -e "${BLUE}Call add_numbers tool:${NC}"
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_numbers","arguments":{"a":15,"b":25}}}'

echo -e "${BLUE}List resources:${NC}"
echo '{"jsonrpc":"2.0","id":5,"method":"resources/list"}'

echo -e "${BLUE}Read resource:${NC}"
echo '{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"everything://resource/1"}}'

echo -e "${BLUE}List prompts:${NC}"
echo '{"jsonrpc":"2.0","id":7,"method":"prompts/list"}'

# Clean up
rm $TEMP_FILE

echo ""
echo -e "${GREEN}✅ Test information provided!${NC}"
echo -e "${YELLOW}💡 The km proxy will log all requests/responses to mcp_proxy.log${NC}"
echo -e "${YELLOW}💡 Authentication issues with km can be resolved by checking API key configuration${NC}"

if [ -f "mcp_proxy.log" ]; then
    echo ""
    echo -e "${BLUE}Current proxy log has $(wc -l < mcp_proxy.log) entries${NC}"
fi
