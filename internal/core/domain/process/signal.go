package process

// ProcessSignal represents signals that can be sent to processes
type ProcessSignal int

const (
	SignalTerminate ProcessSignal = iota // SIGTERM
	SignalInterrupt                      // SIGINT
	SignalKill                           // SIGKILL
)
