package domain

// DebugOptions control runtime debugging behavior when starting a plugin
// during development/testing flows.
type DebugOptions struct {
	Enabled bool

	// When set, runtime logs and protocol transcripts are written to this path
	TranscriptPath string

	// If true, enable kmsdk API logger integration for richer debugging output
	EnableAPILogger bool

	// Simple redaction rules for transcript/log persistence
	RedactSecrets bool
}
