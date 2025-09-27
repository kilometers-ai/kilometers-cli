#!/bin/bash

# Interactive MCP Test Script
# Provides a step-by-step manual test interface for MCP via km proxy

echo "🎯 Interactive MCP Test via km proxy"
echo "===================================="

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
echo ""

# Create properly formatted JSON requests
cat << 'EOF' > mcp_requests.json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}},"clientInfo":{"name":"km-test-client","version":"1.0.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello from km proxy!"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_numbers","arguments":{"a":42,"b":58}}}
{"jsonrpc":"2.0","id":5,"method":"resources/list"}
{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"everything://resource/1"}}
{"jsonrpc":"2.0","id":7,"method":"prompts/list"}
EOF

echo -e "${GREEN}Created mcp_requests.json with properly formatted MCP requests${NC}"
echo ""
echo -e "${YELLOW}Manual test options:${NC}"
echo ""
echo -e "${BLUE}Option 1: Send all requests at once${NC}"
echo "cat mcp_requests.json | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"
echo ""
echo -e "${BLUE}Option 2: Send individual requests (copy-paste each line):${NC}"
echo ""

# Show each request with instructions
requests=(
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}},"clientInfo":{"name":"km-test-client","version":"1.0.0"}}}'
    '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello from km proxy!"}}}'
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"add_numbers","arguments":{"a":42,"b":58}}}'
    '{"jsonrpc":"2.0","id":5,"method":"resources/list"}'
    '{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"everything://resource/1"}}'
    '{"jsonrpc":"2.0","id":7,"method":"prompts/list"}'
)

descriptions=(
    "Initialize MCP connection"
    "Send initialized notification"
    "List available tools"
    "Call echo tool with test message"
    "Call add_numbers tool (42 + 58)"
    "List available resources"
    "Read resource #1"
    "List available prompts"
)

for i in "${!requests[@]}"; do
    echo -e "${YELLOW}Step $((i+1)): ${descriptions[$i]}${NC}"
    echo "echo '${requests[$i]}' | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"
    echo ""
done

echo -e "${GREEN}Test the km proxy by running any of the above commands!${NC}"
echo ""
echo -e "${BLUE}Tips:${NC}"
echo "• The km proxy will log all requests/responses to mcp_proxy.log"
echo "• You can monitor the log in real-time with: tail -f mcp_proxy.log"
echo "• Each request should return a JSON-RPC response"
echo "• The proxy sends telemetry to the Kilometers.ai API"
echo ""
echo -e "${YELLOW}To run a quick automated test:${NC}"
echo "cat mcp_requests.json | $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything"