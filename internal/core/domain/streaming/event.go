package streaming

// StreamEvent represents events in the stream lifecycle
type StreamEvent struct {
	Type      StreamEventType
	Message   string
	Timestamp int64
	Data      interface{}
}

// StreamEventType represents different types of stream events
type StreamEventType string

const (
	StreamEventConnected    StreamEventType = "connected"
	StreamEventDisconnected StreamEventType = "disconnected"
	StreamEventError        StreamEventType = "error"
	StreamEventDataSent     StreamEventType = "data_sent"
	StreamEventDataReceived StreamEventType = "data_received"
)
