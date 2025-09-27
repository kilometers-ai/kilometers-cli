#!/bin/bash

# Simple MCP Test Script
# Tests basic MCP communication via km proxy

set -e

echo "🚀 Simple MCP Test via km proxy"
echo "==============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

echo -e "${BLUE}Using km binary: $KM_BINARY${NC}"

# Create a temporary file with MCP requests
TEMP_FILE=$(mktemp)
cat > $TEMP_FILE << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello from km proxy!"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_numbers","arguments":{"a":10,"b":20}}}
EOF

echo -e "${GREEN}Sending MCP requests through km proxy...${NC}"
echo "Requests to send:"
cat $TEMP_FILE
echo ""
echo -e "${YELLOW}Proxying through km monitor...${NC}"

# Send all requests at once
cat $TEMP_FILE | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything

# Clean up
rm $TEMP_FILE

echo ""
echo -e "${GREEN}✅ Test completed!${NC}"
echo -e "${BLUE}Check mcp_proxy.log for detailed logs${NC}"

# Show log summary if available
if [ -f "mcp_proxy.log" ]; then
    echo ""
    echo -e "${YELLOW}📊 Recent log entries:${NC}"
    tail -5 mcp_proxy.log | head -5
fi