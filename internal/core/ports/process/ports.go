package process

import (
	"context"
	"io"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/process"
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

	// Wait waits for the process to complete and returns its error, if any
	Wait() error

	// Signal sends a signal to the process (for graceful shutdown)
	Signal(signal process.ProcessSignal) error

	// Kill forcefully terminates the process
	Kill() error

	// IsRunning returns true if the process is still running
	IsRunning() bool

	// ExitCode returns the exit code if the process has finished
	ExitCode() int
}

// Executor is responsible for executing commands.
type Executor interface {
	Execute(ctx context.Context, cmd process.Command) (Process, error)
}
