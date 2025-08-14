package jsonrpc

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageID represents a unique message identifier
type MessageID string

// MessageType represents the type of JSON-RPC message
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeNotification MessageType = "notification"
	MessageTypeError        MessageType = "error"
)

// Direction represents the direction of message flow
type Direction string

const (
	DirectionInbound  Direction = "inbound"  // From client to server
	DirectionOutbound Direction = "outbound" // From server to client
)

// JSONRPCMessage represents a JSON-RPC 2.0 message entity
type JSONRPCMessage struct {
	id            MessageID
	msgType       MessageType
	method        string
	payload       json.RawMessage
	timestamp     time.Time
	direction     Direction
	correlationID string
	requestID     *json.RawMessage // For linking responses to requests
	errorInfo     *ErrorInfo       // For error responses
}

// ErrorInfo contains details about JSON-RPC errors
type ErrorInfo struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewJSONRPCMessage creates a new JSON-RPC message
func NewJSONRPCMessage(
	msgType MessageType,
	method string,
	payload json.RawMessage,
	direction Direction,
	correlationID string,
) *JSONRPCMessage {
	messageID := MessageID(fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), string(msgType)))

	return &JSONRPCMessage{
		id:            messageID,
		msgType:       msgType,
		method:        method,
		payload:       payload,
		timestamp:     time.Now(),
		direction:     direction,
		correlationID: correlationID,
	}
}

// NewJSONRPCMessageFromRaw creates a message by parsing raw JSON-RPC data
func NewJSONRPCMessageFromRaw(
	rawData []byte,
	direction Direction,
	correlationID string,
) (*JSONRPCMessage, error) {
	// Parse the JSON to determine message type and extract metadata
	var baseMsg struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method,omitempty"`
		ID      json.RawMessage `json:"id,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *ErrorInfo      `json:"error,omitempty"`
	}

	if err := json.Unmarshal(rawData, &baseMsg); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	// Validate JSON-RPC version
	if baseMsg.JSONRPC != "2.0" {
		return nil, fmt.Errorf("unsupported JSON-RPC version: %s", baseMsg.JSONRPC)
	}

	// Determine message type
	var msgType MessageType
	var method string
	var requestID *json.RawMessage
	var errorInfo *ErrorInfo

	if baseMsg.Error != nil {
		msgType = MessageTypeError
		errorInfo = baseMsg.Error
		if baseMsg.ID != nil {
			requestID = &baseMsg.ID
		}
	} else if baseMsg.Method != "" {
		if baseMsg.ID != nil {
			msgType = MessageTypeRequest
			if baseMsg.ID != nil {
				requestID = &baseMsg.ID
			}
		} else {
			msgType = MessageTypeNotification
		}
		method = baseMsg.Method
	} else if baseMsg.Result != nil || baseMsg.ID != nil {
		msgType = MessageTypeResponse
		if baseMsg.ID != nil {
			requestID = &baseMsg.ID
		}
	} else {
		return nil, fmt.Errorf("cannot determine JSON-RPC message type")
	}

	messageID := MessageID(fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), string(msgType)))

	return &JSONRPCMessage{
		id:            messageID,
		msgType:       msgType,
		method:        method,
		payload:       json.RawMessage(rawData),
		timestamp:     time.Now(),
		direction:     direction,
		correlationID: correlationID,
		requestID:     requestID,
		errorInfo:     errorInfo,
	}, nil
}

// ID returns the message identifier
func (m *JSONRPCMessage) ID() MessageID {
	return m.id
}

// Type returns the message type
func (m *JSONRPCMessage) Type() MessageType {
	return m.msgType
}

// Method returns the JSON-RPC method name
func (m *JSONRPCMessage) Method() string {
	return m.method
}

// Payload returns the raw JSON payload
func (m *JSONRPCMessage) Payload() json.RawMessage {
	return append(json.RawMessage(nil), m.payload...) // Return copy
}

// Timestamp returns when the message was created
func (m *JSONRPCMessage) Timestamp() time.Time {
	return m.timestamp
}

// Direction returns the message direction
func (m *JSONRPCMessage) Direction() Direction {
	return m.direction
}

// CorrelationID returns the associated correlation ID for event tracking
func (m *JSONRPCMessage) CorrelationID() string {
	return m.correlationID
}

// RequestID returns the JSON-RPC request ID (for responses and errors)
func (m *JSONRPCMessage) RequestID() *json.RawMessage {
	if m.requestID == nil {
		return nil
	}
	copy := append(json.RawMessage(nil), *m.requestID...)
	return &copy
}

// ErrorInfo returns error details for error messages
func (m *JSONRPCMessage) ErrorInfo() *ErrorInfo {
	return m.errorInfo
}

// IsRequest returns true if this is a request message
func (m *JSONRPCMessage) IsRequest() bool {
	return m.msgType == MessageTypeRequest
}

// IsResponse returns true if this is a response message
func (m *JSONRPCMessage) IsResponse() bool {
	return m.msgType == MessageTypeResponse
}

// IsNotification returns true if this is a notification message
func (m *JSONRPCMessage) IsNotification() bool {
	return m.msgType == MessageTypeNotification
}

// IsError returns true if this is an error message
func (m *JSONRPCMessage) IsError() bool {
	return m.msgType == MessageTypeError
}

// IsInbound returns true if the message is from client to server
func (m *JSONRPCMessage) IsInbound() bool {
	return m.direction == DirectionInbound
}

// IsOutbound returns true if the message is from server to client
func (m *JSONRPCMessage) IsOutbound() bool {
	return m.direction == DirectionOutbound
}

// String returns a human-readable representation of the message
func (m *JSONRPCMessage) String() string {
	direction := "→"
	if m.direction == DirectionOutbound {
		direction = "←"
	}

	if m.method != "" {
		return fmt.Sprintf("[%s] %s %s %s", m.timestamp.Format("15:04:05.000"), direction, m.msgType, m.method)
	}
	return fmt.Sprintf("[%s] %s %s", m.timestamp.Format("15:04:05.000"), direction, m.msgType)
}

// Size returns the size of the message payload in bytes
func (m *JSONRPCMessage) Size() int {
	return len(m.payload)
}

// IsMCPMethod returns true if this appears to be an MCP-specific method
func (m *JSONRPCMessage) IsMCPMethod() bool {
	mcpMethods := []string{
		"initialize",
		"tools/list",
		"tools/call",
		"resources/list",
		"resources/read",
		"resources/subscribe",
		"resources/unsubscribe",
		"sampling/createMessage",
		"completion/complete",
		"logging/setLevel",
	}

	for _, mcpMethod := range mcpMethods {
		if m.method == mcpMethod {
			return true
		}
	}

	// Check for MCP method patterns
	if len(m.method) > 0 {
		// MCP methods often follow patterns like "tools/*", "resources/*", etc.
		return false // For now, just check explicit methods
	}

	return false
}
