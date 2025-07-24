#!/bin/bash

# Test Debug Replay Feature for km monitor

echo "Testing km monitor with debug replay..."
echo ""

# Build the CLI if needed
echo "Building kilometers CLI..."
go build -o km cmd/main.go

# Create a test replay file
cat > test/replay_test.jsonl << 'EOF'
# Test MCP Debug Replay
{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"0.1.0"},"id":1}

DELAY: 1s

# List tools
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}

DELAY: 500ms

# Call tool with large payload to test buffer handling
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"test_tool","arguments":{"data":"' + $(python3 -c "print('x' * 10000)") + '"}},"id":3}

# Test error handling
{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":4}
EOF

echo ""
echo "Running debug replay test..."
echo "Command: ./km monitor --debug-replay test/replay_test.jsonl --debug-delay 200ms dummy-process"
echo ""

# Run the monitor with debug replay
./km monitor --debug-replay test/replay_test.jsonl --debug-delay 200ms dummy-process &
MONITOR_PID=$!

# Wait for a few seconds to let it process
sleep 5

# Stop the monitor
echo ""
echo "Stopping monitor..."
kill -INT $MONITOR_PID

# Wait for it to stop
sleep 1

echo ""
echo "Test completed!"
echo ""
echo "You can also test with the example file:"
echo "./km monitor --debug-replay test/debug_replay_example.jsonl dummy-process" 