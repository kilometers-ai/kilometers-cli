package domain

import "time"

// DebugSession represents a single plugin debugging session lifecycle.
type DebugSession struct {
	ID             string
	PluginName     string
	StartedAt      time.Time
	TranscriptPath string
}
