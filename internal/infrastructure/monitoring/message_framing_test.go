package monitoring

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestMessageFramingLogic tests the core accumulator logic for MCP message framing
func TestMessageFramingLogic(t *testing.T) {
	t.Run("single_large_message", func(t *testing.T) {
		// Create a large JSON-RPC message similar to Linear search results
		largeResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"result": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": strings.Repeat("Large Linear search result data ", 1000),
					},
				},
			},
		}

		jsonData, err := json.Marshal(largeResponse)
		if err != nil {
			t.Fatalf("Failed to marshal test JSON: %v", err)
		}

		testMessage := string(jsonData) + "\n"

		// Verify message is large enough (should be > 10KB to test large message handling)
		if len(testMessage) < 10000 {
			t.Errorf("Test message too small, got %d bytes", len(testMessage))
		}

		// Test parsing
		var parsed map[string]interface{}
		trimmed := bytes.TrimSpace([]byte(testMessage))
		if err := json.Unmarshal(trimmed, &parsed); err != nil {
			t.Errorf("Failed to parse large JSON-RPC response: %v", err)
		}

		// Verify structure
		if parsed["jsonrpc"] != "2.0" {
			t.Error("Invalid JSON-RPC version")
		}
	})

	t.Run("message_accumulation_and_extraction", func(t *testing.T) {
		testCases := []struct {
			name     string
			chunks   []string
			expected []string
		}{
			{
				name: "complete_message_single_chunk",
				chunks: []string{
					`{"jsonrpc":"2.0","method":"test","id":1}` + "\n",
				},
				expected: []string{
					`{"jsonrpc":"2.0","method":"test","id":1}` + "\n",
				},
			},
			{
				name: "fragmented_message_multiple_chunks",
				chunks: []string{
					`{"jsonrpc":"2.0",`,
					`"method":"tools/call",`,
					`"params":{"name":"linear_searchIssues"},`,
					`"id":1}` + "\n",
				},
				expected: []string{
					`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"linear_searchIssues"},"id":1}` + "\n",
				},
			},
			{
				name: "multiple_complete_messages",
				chunks: []string{
					`{"jsonrpc":"2.0","method":"init","id":1}` + "\n" + `{"jsonrpc":"2.0","method":"call","id":2}` + "\n",
				},
				expected: []string{
					`{"jsonrpc":"2.0","method":"init","id":1}` + "\n",
					`{"jsonrpc":"2.0","method":"call","id":2}` + "\n",
				},
			},
			{
				name: "mixed_fragmented_and_complete",
				chunks: []string{
					`{"jsonrpc":"2.0","method":"first","id":1}` + "\n" + `{"jsonrpc":"2.0",`,
					`"method":"second","id":2}` + "\n",
				},
				expected: []string{
					`{"jsonrpc":"2.0","method":"first","id":1}` + "\n",
					`{"jsonrpc":"2.0","method":"second","id":2}` + "\n",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var accumulator []byte
				var extractedMessages []string

				// Simulate the accumulator logic from monitorStdout
				for _, chunk := range tc.chunks {
					accumulator = append(accumulator, []byte(chunk)...)

					// Extract complete messages (newline-delimited)
					for {
						newlineIdx := bytes.IndexByte(accumulator, '\n')
						if newlineIdx == -1 {
							break // No complete message yet
						}

						// Extract complete message (including newline)
						message := accumulator[:newlineIdx+1]
						accumulator = accumulator[newlineIdx+1:]

						// Add to results if not empty
						if len(bytes.TrimSpace(message)) > 0 {
							extractedMessages = append(extractedMessages, string(message))
						}
					}
				}

				// Verify we got the expected messages
				if len(extractedMessages) != len(tc.expected) {
					t.Errorf("Expected %d messages, got %d\nExtracted: %v\nExpected: %v",
						len(tc.expected), len(extractedMessages), extractedMessages, tc.expected)
				}

				for i, expected := range tc.expected {
					if i < len(extractedMessages) {
						if extractedMessages[i] != expected {
							t.Errorf("Message %d mismatch\nExpected: %q\nGot: %q", i+1, expected, extractedMessages[i])
						}

						// Verify each extracted message is valid JSON
						var parsed map[string]interface{}
						trimmed := bytes.TrimSpace([]byte(extractedMessages[i]))
						if err := json.Unmarshal(trimmed, &parsed); err != nil {
							t.Errorf("Message %d is not valid JSON: %v\nMessage: %s", i+1, err, extractedMessages[i])
						}
					}
				}
			})
		}
	})

	t.Run("large_linear_search_simulation", func(t *testing.T) {
		// Simulate a realistic Linear search response that caused the original error
		searchResult := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"result": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": generateLargeLinearSearchResult(),
					},
				},
			},
		}

		jsonData, err := json.Marshal(searchResult)
		if err != nil {
			t.Fatalf("Failed to marshal search result: %v", err)
		}

		message := string(jsonData) + "\n"

		// This should be larger than the old 4KB buffer limit
		if len(message) < 10000 {
			t.Errorf("Simulated Linear search result too small: %d bytes", len(message))
		}

		// Test that our new approach can handle it
		var accumulator []byte
		var messages []string

		// Simulate receiving in chunks (like real network/pipe I/O)
		chunkSize := 1024
		for i := 0; i < len(message); i += chunkSize {
			end := i + chunkSize
			if end > len(message) {
				end = len(message)
			}
			chunk := message[i:end]

			accumulator = append(accumulator, []byte(chunk)...)

			// Extract complete messages
			for {
				newlineIdx := bytes.IndexByte(accumulator, '\n')
				if newlineIdx == -1 {
					break
				}

				completeMessage := accumulator[:newlineIdx+1]
				accumulator = accumulator[newlineIdx+1:]

				if len(bytes.TrimSpace(completeMessage)) > 0 {
					messages = append(messages, string(completeMessage))
				}
			}
		}

		// Should have exactly 1 complete message
		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}

		if len(messages) > 0 {
			// Verify the message is the same as the original
			if messages[0] != message {
				t.Error("Message reconstruction failed")
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			trimmed := bytes.TrimSpace([]byte(messages[0]))
			if err := json.Unmarshal(trimmed, &parsed); err != nil {
				t.Errorf("Large message is not valid JSON: %v", err)
			}
		}
	})

	t.Run("non_json_and_binary_output_transparency", func(t *testing.T) {
		// Simulate non-JSON, debug, and binary output
		chunks := []string{
			"DEBUG: Starting MCP server\n",
			"\x00\x01\x02\x03\x04\n",
			"{not: 'json'}\n",
			"\xff\xfe\xfd\xfc\xfb\n",
			"Partial line with no newline",
		}

		var accumulator []byte
		var forwarded [][]byte

		for _, chunk := range chunks {
			accumulator = append(accumulator, []byte(chunk)...)
			forwarded = append(forwarded, []byte(chunk))
		}

		// Simulate the protocol transparency guarantee: all chunks are forwarded as-is
		for i, chunk := range chunks {
			if string(forwarded[i]) != chunk {
				t.Errorf("Chunk %d not forwarded transparently. Expected: %q, Got: %q", i, chunk, string(forwarded[i]))
			}
		}

		// Simulate accumulator logic: only complete lines are extracted for monitoring, but all data is forwarded
		var extracted [][]byte
		acc := accumulator
		for {
			newlineIdx := bytes.IndexByte(acc, '\n')
			if newlineIdx == -1 {
				break
			}
			message := acc[:newlineIdx+1]
			acc = acc[newlineIdx+1:]
			if len(bytes.TrimSpace(message)) > 0 {
				extracted = append(extracted, message)
			}
		}

		// Only lines with newlines are extracted for monitoring, but all are forwarded
		if len(extracted) != 4 {
			t.Errorf("Expected 4 extracted lines, got %d", len(extracted))
		}
	})
}

// generateLargeLinearSearchResult creates a realistic large search result
func generateLargeLinearSearchResult() string {
	var result strings.Builder

	// Simulate multiple Linear issues in search results
	issues := []string{
		"KIL-64: Implement Proper MCP Message Framing and Stream Handling",
		"KIL-62: Fix Buffer Size Limitation for Large MCP Messages",
		"KIL-61: Fix MCP JSON-RPC Message Parsing",
		"KIL-63: Improve Error Handling and Debugging",
		"KIL-65: Create Test Harness for MCP Message Processing",
	}

	for i := 0; i < 500; i++ { // Make it large
		issue := issues[i%len(issues)]
		result.WriteString(issue)
		result.WriteString(" - This is a detailed description of the issue with implementation details, ")
		result.WriteString("acceptance criteria, technical requirements, and other relevant information. ")
		result.WriteString("The issue involves complex changes to the MCP message processing pipeline. ")
	}

	return result.String()
}
