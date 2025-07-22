#!/bin/bash

set -e

FIFO_IN="/tmp/km_monitor_in.$$"
FIFO_OUT="/tmp/km_monitor_out.$$"
mkfifo "$FIFO_IN" "$FIFO_OUT"

# Start km monitor in the background, using the FIFOs
./km monitor -- npx -y @modelcontextprotocol/server-sequential-thinking <"$FIFO_IN" >"$FIFO_OUT" 2>/tmp/km_monitor_test_stderr.$$ &
KM_PID=$!

# Give it a moment to start
sleep 2

send_request() {
  local request="$1"
  echo ">>> Sending: $request"
  echo "$request" > "$FIFO_IN"
  # Use gtimeout to read a single line (response) with a 3s timeout
  local response
  response=$(gtimeout 3s head -n 1 "$FIFO_OUT" || true)
  if [[ -z "$response" ]]; then
    echo "<<< No response received (timeout)"
  else
    echo "<<< Response: $response"
  fi
}

# Send first request
send_request '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}'

# Send second request
send_request '{"jsonrpc": "2.0", "method": "tools/list", "id": 2}'

# Clean up
kill $KM_PID 2>/dev/null || true
rm -f "$FIFO_IN" "$FIFO_OUT" /tmp/km_monitor_test_stderr.$$

echo "Persistent session test complete." 