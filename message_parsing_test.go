package main

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"testing"
)

// Helper function to create a minimal ProcessWrapper for testing parseMCPMessage
func createTestProcessWrapper() *ProcessWrapper {
	config := DefaultConfig()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)
	
	return &ProcessWrapper{
		config: config,
		logger: logger,
	}
}

func TestParseMCPMessage_ValidJSONRPCRequest(t *testing.T) {
	pw := createTestProcessWrapper()
	
	validRequest := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`
	
	result := pw.parseMCPMessage([]byte(validRequest))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid JSON-RPC request")
	}
	
	if result.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version '2.0', got '%s'", result.JSONRPC)
	}
	
	if result.Method != "initialize" {
		t.Errorf("Expected method 'initialize', got '%s'", result.Method)
	}
	
	if result.ID != float64(1) { // JSON unmarshaling converts numbers to float64
		t.Errorf("Expected ID 1, got %v", result.ID)
	}
}

func TestParseMCPMessage_ValidJSONRPCResponse(t *testing.T) {
	pw := createTestProcessWrapper()
	
	validResponse := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}}}}`
	
	result := pw.parseMCPMessage([]byte(validResponse))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid JSON-RPC response")
	}
	
	if result.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version '2.0', got '%s'", result.JSONRPC)
	}
	
	if result.ID != float64(1) {
		t.Errorf("Expected ID 1, got %v", result.ID)
	}
	
	if result.Result == nil {
		t.Error("Expected Result to be present in response")
	}
}

func TestParseMCPMessage_ValidJSONRPCNotification(t *testing.T) {
	pw := createTestProcessWrapper()
	
	validNotification := `{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"info","message":"Hello"}}`
	
	result := pw.parseMCPMessage([]byte(validNotification))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid JSON-RPC notification")
	}
	
	if result.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version '2.0', got '%s'", result.JSONRPC)
	}
	
	if result.Method != "notifications/message" {
		t.Errorf("Expected method 'notifications/message', got '%s'", result.Method)
	}
	
	// Notifications don't have ID, should be nil
	if result.ID != nil {
		t.Errorf("Expected ID to be nil for notification, got %v", result.ID)
	}
}

func TestParseMCPMessage_ErrorResponse(t *testing.T) {
	pw := createTestProcessWrapper()
	
	errorResponse := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`
	
	result := pw.parseMCPMessage([]byte(errorResponse))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid error response")
	}
	
	if result.Error == nil {
		t.Fatal("Expected Error to be present in error response")
	}
	
	if result.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", result.Error.Code)
	}
	
	if result.Error.Message != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got '%s'", result.Error.Message)
	}
}

func TestParseMCPMessage_DifferentIDTypes(t *testing.T) {
	pw := createTestProcessWrapper()
	
	tests := []struct {
		name        string
		jsonMessage string
		expectedID  interface{}
	}{
		{
			name:        "StringID",
			jsonMessage: `{"jsonrpc":"2.0","id":"abc-123","method":"test"}`,
			expectedID:  "abc-123",
		},
		{
			name:        "NumberID",
			jsonMessage: `{"jsonrpc":"2.0","id":42,"method":"test"}`,
			expectedID:  float64(42),
		},
		{
			name:        "NullID",
			jsonMessage: `{"jsonrpc":"2.0","id":null,"method":"test"}`,
			expectedID:  nil,
		},
		{
			name:        "NoID_Notification",
			jsonMessage: `{"jsonrpc":"2.0","method":"test"}`,
			expectedID:  nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pw.parseMCPMessage([]byte(tt.jsonMessage))
			
			if result == nil {
				t.Fatal("parseMCPMessage() returned nil for valid message")
			}
			
			if !reflect.DeepEqual(result.ID, tt.expectedID) {
				t.Errorf("Expected ID %v (%T), got %v (%T)", tt.expectedID, tt.expectedID, result.ID, result.ID)
			}
		})
	}
}

func TestParseMCPMessage_InvalidJSON(t *testing.T) {
	pw := createTestProcessWrapper()
	
	tests := []struct {
		name    string
		input   string
		reason  string
	}{
		{
			name:   "MalformedJSON",
			input:  `{"jsonrpc":"2.0","id":1,"method":"test"`,
			reason: "Missing closing brace",
		},
		{
			name:   "EmptyString",
			input:  "",
			reason: "Empty input",
		},
		{
			name:   "InvalidJSONSyntax",
			input:  `{jsonrpc:"2.0",id:1}`,
			reason: "Unquoted keys",
		},
		{
			name:   "NotAnObject",
			input:  `["not", "an", "object"]`,
			reason: "JSON array instead of object",
		},
		{
			name:   "PlainString",
			input:  "not json at all",
			reason: "Plain text",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pw.parseMCPMessage([]byte(tt.input))
			
			if result != nil {
				t.Errorf("parseMCPMessage() should return nil for invalid JSON (%s), got: %+v", tt.reason, result)
			}
		})
	}
}

func TestParseMCPMessage_InvalidJSONRPCVersion(t *testing.T) {
	pw := createTestProcessWrapper()
	
	tests := []struct {
		name    string
		input   string
		reason  string
	}{
		{
			name:   "MissingJSONRPCField",
			input:  `{"id":1,"method":"test"}`,
			reason: "Missing jsonrpc field",
		},
		{
			name:   "WrongVersion_1_0",
			input:  `{"jsonrpc":"1.0","id":1,"method":"test"}`,
			reason: "JSON-RPC 1.0 instead of 2.0",
		},
		{
			name:   "EmptyVersion",
			input:  `{"jsonrpc":"","id":1,"method":"test"}`,
			reason: "Empty jsonrpc version",
		},
		{
			name:   "NullVersion",
			input:  `{"jsonrpc":null,"id":1,"method":"test"}`,
			reason: "Null jsonrpc version",
		},
		{
			name:   "NumberVersion",
			input:  `{"jsonrpc":2.0,"id":1,"method":"test"}`,
			reason: "Number instead of string version",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pw.parseMCPMessage([]byte(tt.input))
			
			if result != nil {
				t.Errorf("parseMCPMessage() should return nil for invalid JSON-RPC version (%s), got: %+v", tt.reason, result)
			}
		})
	}
}

func TestParseMCPMessage_RealWorldMCPExamples(t *testing.T) {
	pw := createTestProcessWrapper()
	
	tests := []struct {
		name           string
		jsonMessage    string
		expectedMethod string
	}{
		{
			name: "InitializeRequest",
			jsonMessage: `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "initialize",
				"params": {
					"protocolVersion": "2024-11-05",
					"capabilities": {
						"tools": {}
					},
					"clientInfo": {
						"name": "test-client",
						"version": "1.0.0"
					}
				}
			}`,
			expectedMethod: "initialize",
		},
		{
			name: "ToolsListRequest",
			jsonMessage: `{
				"jsonrpc": "2.0",
				"id": 2,
				"method": "tools/list"
			}`,
			expectedMethod: "tools/list",
		},
		{
			name: "ToolsCallRequest",
			jsonMessage: `{
				"jsonrpc": "2.0",
				"id": 3,
				"method": "tools/call",
				"params": {
					"name": "filesystem_read",
					"arguments": {
						"path": "/path/to/file.txt"
					}
				}
			}`,
			expectedMethod: "tools/call",
		},
		{
			name: "ResourcesListRequest",
			jsonMessage: `{
				"jsonrpc": "2.0",
				"id": 4,
				"method": "resources/list"
			}`,
			expectedMethod: "resources/list",
		},
		{
			name: "NotificationMessage",
			jsonMessage: `{
				"jsonrpc": "2.0",
				"method": "notifications/message",
				"params": {
					"level": "info",
					"message": "Operation completed successfully"
				}
			}`,
			expectedMethod: "notifications/message",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pw.parseMCPMessage([]byte(tt.jsonMessage))
			
			if result == nil {
				t.Fatal("parseMCPMessage() returned nil for valid real-world MCP message")
			}
			
			if result.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC version '2.0', got '%s'", result.JSONRPC)
			}
			
			if result.Method != tt.expectedMethod {
				t.Errorf("Expected method '%s', got '%s'", tt.expectedMethod, result.Method)
			}
		})
	}
}

func TestParseMCPMessage_LargePayload(t *testing.T) {
	pw := createTestProcessWrapper()
	
	// Create a large params object
	largeData := make(map[string]string)
	for i := 0; i < 1000; i++ {
		largeData[string(rune('a'+i%26))+string(rune('A'+i%26))] = "large data content that repeats many times to create a substantial payload size"
	}
	
	messageStruct := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "test/large",
		"params":  largeData,
	}
	
	messageBytes, err := json.Marshal(messageStruct)
	if err != nil {
		t.Fatalf("Failed to marshal large test message: %v", err)
	}
	
	result := pw.parseMCPMessage(messageBytes)
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for large but valid message")
	}
	
	if result.Method != "test/large" {
		t.Errorf("Expected method 'test/large', got '%s'", result.Method)
	}
	
	// Verify params were parsed correctly
	if result.Params == nil {
		t.Error("Expected Params to be present in large message")
	}
}

func TestParseMCPMessage_PreservesRawParams(t *testing.T) {
	pw := createTestProcessWrapper()
	
	originalMessage := `{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value","number":42}}`
	
	result := pw.parseMCPMessage([]byte(originalMessage))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid message")
	}
	
	// Verify params are preserved as RawMessage
	if result.Params == nil {
		t.Fatal("Expected Params to be present")
	}
	
	// Parse the raw params to verify content
	var params map[string]interface{}
	if err := json.Unmarshal(result.Params, &params); err != nil {
		t.Fatalf("Failed to parse raw params: %v", err)
	}
	
	if params["key"] != "value" {
		t.Errorf("Expected params.key to be 'value', got %v", params["key"])
	}
	
	if params["number"] != float64(42) {
		t.Errorf("Expected params.number to be 42, got %v", params["number"])
	}
}

func TestParseMCPMessage_PreservesRawResult(t *testing.T) {
	pw := createTestProcessWrapper()
	
	originalMessage := `{"jsonrpc":"2.0","id":1,"result":{"status":"success","data":[1,2,3]}}`
	
	result := pw.parseMCPMessage([]byte(originalMessage))
	
	if result == nil {
		t.Fatal("parseMCPMessage() returned nil for valid message")
	}
	
	// Verify result is preserved as RawMessage
	if result.Result == nil {
		t.Fatal("Expected Result to be present")
	}
	
	// Parse the raw result to verify content
	var resultData map[string]interface{}
	if err := json.Unmarshal(result.Result, &resultData); err != nil {
		t.Fatalf("Failed to parse raw result: %v", err)
	}
	
	if resultData["status"] != "success" {
		t.Errorf("Expected result.status to be 'success', got %v", resultData["status"])
	}
}
