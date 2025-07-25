#!/bin/bash

# Test Script for Claude AI Client Requests Debug Replay

echo "Testing kilometers CLI with Claude AI client requests debug replay..."
echo ""

# Build the CLI if needed
echo "Building kilometers CLI..."
go build -o km cmd/main.go

echo ""
echo "Running Claude AI client requests debug replay..."
echo "Command: ./km monitor --debug-replay test/claude_ai_client_requests.jsonl --server npx -y @modelcontextprotocol/server-github"
echo ""
echo "This will replay the following sequence with 2-second delays:"
echo "1. MCP Initialize (protocol version 2025-06-18)"
echo "2. Initialized notification"
echo "3. Tools list request"
echo "4. Resources list request"  
echo "5. Prompts list request"
echo ""

# Run the monitor with debug replay
./km monitor --debug-replay test/claude_ai_client_requests.jsonl --server npx -y @modelcontextprotocol/server-github &
MONITOR_PID=$!

# Wait for the replay to complete (approximately 10 seconds for 4 delays + processing time)
echo "Waiting for replay to complete (approximately 10 seconds)..."
sleep 12

# Stop the monitor
echo ""
echo "Stopping monitor..."
kill -INT $MONITOR_PID 2>/dev/null || true

# Wait for it to stop
sleep 1

echo ""
echo "Claude AI client requests debug replay test completed!"
echo ""
echo "To debug in VSCode:"
echo "1. Open VSCode"
echo "2. Go to Run and Debug (Ctrl+Shift+D)"
echo "3. Select 'Debug Monitor with Claude AI Client Replay' from the dropdown"
echo "4. Set breakpoints in the monitoring code"
echo "5. Press F5 to start debugging" 