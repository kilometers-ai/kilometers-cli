package event

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// EventID is a value object representing a unique event identifier
type EventID struct {
	value string
}

// NewEventID creates a new EventID with validation
func NewEventID(value string) (EventID, error) {
	if value == "" {
		return EventID{}, fmt.Errorf("event ID cannot be empty")
	}
	return EventID{value: value}, nil
}

// GenerateEventID creates a new unique EventID
func GenerateEventID() EventID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return EventID{value: hex.EncodeToString(bytes)}
}

// Value returns the string value of the EventID
func (e EventID) Value() string {
	return e.value
}

// String implements the Stringer interface
func (e EventID) String() string {
	return e.value
}

// Direction represents the direction of an MCP message
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// NewDirection creates a Direction with validation
func NewDirection(value string) (Direction, error) {
	switch value {
	case "inbound", "outbound", "request", "response":
		// Support legacy values for backward compatibility
		if value == "request" {
			return DirectionInbound, nil
		}
		if value == "response" {
			return DirectionOutbound, nil
		}
		return Direction(value), nil
	default:
		return "", fmt.Errorf("invalid direction: %s", value)
	}
}

// String returns the string representation of Direction
func (d Direction) String() string {
	return string(d)
}

// Method represents an MCP method name
type Method struct {
	value string
}

// NewMethod creates a Method with validation
func NewMethod(value string) (Method, error) {
	if value == "" {
		return Method{}, fmt.Errorf("method cannot be empty")
	}
	return Method{value: value}, nil
}

// Value returns the string value of the Method
func (m Method) Value() string {
	return m.value
}

// String implements the Stringer interface
func (m Method) String() string {
	return m.value
}

// RiskScore represents the risk assessment score for an event
type RiskScore struct {
	value int
}

// NewRiskScore creates a RiskScore with validation
func NewRiskScore(value int) (RiskScore, error) {
	if value < 0 || value > 100 {
		return RiskScore{}, fmt.Errorf("risk score must be between 0 and 100, got %d", value)
	}
	return RiskScore{value: value}, nil
}

// Value returns the integer value of the RiskScore
func (r RiskScore) Value() int {
	return r.value
}

// Level returns the risk level based on the score
func (r RiskScore) Level() string {
	switch {
	case r.value >= 75:
		return "high"
	case r.value >= 35:
		return "medium"
	default:
		return "low"
	}
}

// IsHigh returns true if the risk score is high (>= 75)
func (r RiskScore) IsHigh() bool {
	return r.value >= 75
}

// IsMedium returns true if the risk score is medium (35-74)
func (r RiskScore) IsMedium() bool {
	return r.value >= 35 && r.value < 75
}

// IsLow returns true if the risk score is low (< 35)
func (r RiskScore) IsLow() bool {
	return r.value < 35
}

// Event represents an MCP event with proper domain encapsulation
type Event struct {
	id        EventID
	timestamp time.Time
	direction Direction
	method    Method
	payload   []byte
	size      int
	riskScore RiskScore
}

// NewEvent creates a new Event with validation
func NewEvent(
	id EventID,
	timestamp time.Time,
	direction Direction,
	method Method,
	payload []byte,
	riskScore RiskScore,
) (*Event, error) {
	if timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("payload cannot be empty")
	}

	return &Event{
		id:        id,
		timestamp: timestamp,
		direction: direction,
		method:    method,
		payload:   payload,
		size:      len(payload),
		riskScore: riskScore,
	}, nil
}

// CreateEvent is a factory method for creating events with generated ID
func CreateEvent(
	direction Direction,
	method Method,
	payload []byte,
	riskScore RiskScore,
) (*Event, error) {
	return NewEvent(
		GenerateEventID(),
		time.Now(),
		direction,
		method,
		payload,
		riskScore,
	)
}

// ID returns the event ID
func (e *Event) ID() EventID {
	return e.id
}

// Timestamp returns the event timestamp
func (e *Event) Timestamp() time.Time {
	return e.timestamp
}

// Direction returns the event direction
func (e *Event) Direction() Direction {
	return e.direction
}

// Method returns the event method
func (e *Event) Method() Method {
	return e.method
}

// Payload returns a copy of the event payload
func (e *Event) Payload() []byte {
	// Return a copy to maintain immutability
	payload := make([]byte, len(e.payload))
	copy(payload, e.payload)
	return payload
}

// Size returns the size of the payload
func (e *Event) Size() int {
	return e.size
}

// RiskScore returns the risk score
func (e *Event) RiskScore() RiskScore {
	return e.riskScore
}

// IsInbound returns true if the event is inbound
func (e *Event) IsInbound() bool {
	return e.direction == DirectionInbound
}

// IsOutbound returns true if the event is outbound
func (e *Event) IsOutbound() bool {
	return e.direction == DirectionOutbound
}

// IsHighRisk returns true if the event has a high risk score
func (e *Event) IsHighRisk() bool {
	return e.riskScore.IsHigh()
}

// UpdateRiskScore updates the risk score (used by risk analysis service)
func (e *Event) UpdateRiskScore(score RiskScore) {
	e.riskScore = score
}

// String returns a string representation of the event
func (e *Event) String() string {
	return fmt.Sprintf("Event{ID: %s, Method: %s, Direction: %s, Size: %d, Risk: %d}",
		e.id.Value(),
		e.method.Value(),
		e.direction.String(),
		e.size,
		e.riskScore.Value(),
	)
}
