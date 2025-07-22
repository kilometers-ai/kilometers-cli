#!/bin/bash

set -e

REQ1='{"jsonrpc": "2.0", "method": "tools/list", "id": 1}'
REQ2='{"jsonrpc": "2.0", "method": "tools/list", "id": 2}'

# Test direct MCP server with two requests

echo "=== Testing Direct MCP Server (two requests) ==="
echo -e "$REQ1\n$REQ2" | npx -y @modelcontextprotocol/server-sequential-thinking

echo ""
echo "=== Testing Through km monitor (two requests) ==="
echo -e "$REQ1\n$REQ2" | ./km monitor -- npx -y @modelcontextprotocol/server-sequential-thinking 2>/dev/null

echo ""
echo "=== Test Complete ===" 