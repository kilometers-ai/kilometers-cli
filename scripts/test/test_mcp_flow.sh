#!/bin/bash

# MCP Client-Server Test Script via km proxy
# Tests the typical MCP flow: initialize -> list tools -> call tools
# Uses @modelcontextprotocol/server-everything as the test MCP server

set -e

echo "🚀 Testing MCP Client-Server Flow via km proxy"
echo "=============================================="

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

# Function to send MCP request and get response
send_mcp_request() {
    local request="$1"
    local description="$2"

    echo -e "${YELLOW}📤 Sending: $description${NC}"
    echo "$request"
    echo ""

    # Send the request via km proxy with timeout
    echo "$request" | timeout 10s $KM_BINARY monitor -- npx -y @modelcontextprotocol/server-everything
    local exit_code=$?

    if [ $exit_code -eq 124 ]; then
        echo -e "${RED}⏰ Request timed out after 10 seconds${NC}"
    elif [ $exit_code -ne 0 ]; then
        echo -e "${RED}❌ Request failed with exit code: $exit_code${NC}"
    fi
    echo ""
}

# Function to create a properly formatted JSON-RPC request
create_request() {
    local method="$1"
    local params="$2"
    local id="$3"

    if [ -z "$params" ] || [ "$params" = "null" ]; then
        echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\"}"
    else
        echo "{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\",\"params\":$params}"
    fi
}

echo -e "${GREEN}Step 1: Initialize MCP connection${NC}"
echo "-----------------------------------"
init_request=$(create_request "initialize" '{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}}' 1)
send_mcp_request "$init_request" "Initialize request"

echo -e "${GREEN}Step 2: Send initialized notification${NC}"
echo "------------------------------------"
initialized_notification='{"jsonrpc":"2.0","method":"notifications/initialized"}'
send_mcp_request "$initialized_notification" "Initialized notification"

echo -e "${GREEN}Step 3: List available tools${NC}"
echo "----------------------------"
tools_request=$(create_request "tools/list" "null" 2)
send_mcp_request "$tools_request" "List tools request"

echo -e "${GREEN}Step 4: Call echo tool${NC}"
echo "----------------------"
echo_params='{"name":"echo","arguments":{"text":"Hello from km proxy test!"}}'
echo_request=$(create_request "tools/call" "$echo_params" 3)
send_mcp_request "$echo_request" "Echo tool call"

echo -e "${GREEN}Step 5: Call add_numbers tool${NC}"
echo "-----------------------------"
add_params='{"name":"add_numbers","arguments":{"a":42,"b":58}}'
add_request=$(create_request "tools/call" "$add_params" 4)
send_mcp_request "$add_request" "Add numbers tool call"

echo -e "${GREEN}Step 6: List available resources${NC}"
echo "-------------------------------"
resources_request=$(create_request "resources/list" "null" 5)
send_mcp_request "$resources_request" "List resources request"

echo -e "${GREEN}Step 7: Read a resource${NC}"
echo "------------------------"
read_params='{"uri":"everything://resource/1"}'
read_request=$(create_request "resources/read" "$read_params" 6)
send_mcp_request "$read_request" "Read resource request"

echo -e "${GREEN}Step 8: List available prompts${NC}"
echo "-----------------------------"
prompts_request=$(create_request "prompts/list" "null" 7)
send_mcp_request "$prompts_request" "List prompts request"

echo ""
echo -e "${GREEN}✅ MCP flow test completed!${NC}"
echo -e "${BLUE}Check mcp_proxy.log for detailed request/response logs${NC}"

# Show a summary of the log file
if [ -f "mcp_proxy.log" ]; then
    echo ""
    echo -e "${YELLOW}📊 Log Summary:${NC}"
    echo "Total requests logged: $(wc -l < mcp_proxy.log)"
    echo "Last few log entries:"
    tail -3 mcp_proxy.log | jq -r '"\(.timestamp) - \(.method // "notification") (\(.direction))"' 2>/dev/null || tail -3 mcp_proxy.log
fi
