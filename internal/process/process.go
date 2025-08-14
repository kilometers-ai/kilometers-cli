package process

import (
	"io"
	"os"
	"syscall"
)

// Process represents a running MCP server process
type Process interface {
	// PID returns the process ID
	PID() int

	// Stdin returns the stdin writer for sending data to the process
	Stdin() io.WriteCloser

	// Stdout returns the stdout reader for receiving data from the process
	Stdout() io.ReadCloser

	// Stderr returns the stderr reader for receiving error output
	Stderr() io.ReadCloser

	// Wait waits for the process to complete and returns the exit code
	Wait() error

	// Signal sends a signal to the process (for graceful shutdown)
	Signal(signal ProcessSignal) error

	// Kill forcefully terminates the process
	Kill() error

	// IsRunning returns true if the process is still running
	IsRunning() bool

	// ExitCode returns the exit code if the process has finished
	ExitCode() int
}

// ProcessSignal represents signals that can be sent to processes
type ProcessSignal int

const (
	SignalTerminate ProcessSignal = iota // SIGTERM
	SignalInterrupt                      // SIGINT
	SignalKill                           // SIGKILL
)

// ConvertSignal converts ProcessSignal to os.Signal
func ConvertSignal(signal ProcessSignal) os.Signal {
	switch signal {
	case SignalTerminate:
		return syscall.SIGTERM
	case SignalInterrupt:
		return syscall.SIGINT
	case SignalKill:
		return syscall.SIGKILL
	default:
		return syscall.SIGTERM
	}
}
