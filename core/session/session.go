package session

import (
	"time"
)

type McpSession struct {
	id           SessionId
	config       SessionConfig
	events       []Event
	startTime    time.Time
	isActive     bool
	eventCounter int
}

type SessionId struct {
	value string
}
