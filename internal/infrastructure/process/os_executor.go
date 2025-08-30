package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/kilometers-ai/kilometers-cli/internal/core/domain/process"
	procp "github.com/kilometers-ai/kilometers-cli/internal/core/ports/process"
)

// Executor implements the ProcessExecutor interface
type Executor struct {
	// Configuration for process execution
	timeout time.Duration
	workDir string
	env     []string
}

// NewExecutor creates a new process executor
func NewExecutor() *Executor {
	return &Executor{
		timeout: 30 * time.Second, // Default timeout
		workDir: "",               // Use current directory
		env:     os.Environ(),     // Use current environment
	}
}

// NewExecutorWithOptions creates a new process executor with custom options
func NewExecutorWithOptions(timeout time.Duration, workDir string, env []string) *Executor {
	if env == nil {
		env = os.Environ()
	}

	return &Executor{
		timeout: timeout,
		workDir: workDir,
		env:     env,
	}
}

// Execute starts a new process and returns a Process handle
func (e *Executor) Execute(ctx context.Context, cmd process.Command) (procp.Process, error) {
	// Create the OS command
	execCmd := exec.CommandContext(ctx, cmd.Executable(), cmd.Args()...)

	// Set working directory
	if cmd.WorkingDir() != "" {
		execCmd.Dir = cmd.WorkingDir()
	} else if e.workDir != "" {
		execCmd.Dir = e.workDir
	}

	// Set environment variables
	execCmd.Env = e.buildEnvironment(cmd.Env())

	// Create pipes for stdin, stdout, stderr
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := execCmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Create and return the process wrapper
	processImpl := &processImpl{
		cmd:    execCmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}

	// Start monitoring the process in a goroutine
	go processImpl.monitor()

	return processImpl, nil
}

// buildEnvironment combines current environment with command-specific environment
func (e *Executor) buildEnvironment(cmdEnv map[string]string) []string {
	env := append([]string(nil), e.env...) // Copy base environment

	// Add command-specific environment variables
	for key, value := range cmdEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// processImpl implements the Process interface
type processImpl struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// Process state
	mu       sync.RWMutex
	running  bool
	exitCode int
	done     chan struct{}
	waitErr  error
}

// PID returns the process ID
func (p *processImpl) PID() int {
	if p.cmd == nil || p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

// Stdin returns the stdin writer for sending data to the process
func (p *processImpl) Stdin() io.WriteCloser {
	return p.stdin
}

// Stdout returns the stdout reader for receiving data from the process
func (p *processImpl) Stdout() io.ReadCloser {
	return p.stdout
}

// Stderr returns the stderr reader for receiving error output
func (p *processImpl) Stderr() io.ReadCloser {
	return p.stderr
}

// Wait waits for the process to complete and returns the exit code
func (p *processImpl) Wait() error {
	<-p.done
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.waitErr
}

func (p *processImpl) Signal(signal process.ProcessSignal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	sig := ConvertSignal(signal)
	return p.cmd.Process.Signal(sig)
}

func (p *processImpl) Kill() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	if p.stdin != nil {
		p.stdin.Close()
	}

	return p.cmd.Process.Kill()
}

func (p *processImpl) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *processImpl) ExitCode() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitCode
}

// ConvertSignal converts domain signal to OS signal
func ConvertSignal(signal process.ProcessSignal) os.Signal {
	switch signal {
	case process.SignalTerminate:
		return syscall.SIGTERM
	case process.SignalInterrupt:
		return syscall.SIGINT
	case process.SignalKill:
		return syscall.SIGKILL
	default:
		return syscall.SIGTERM
	}
}

func (p *processImpl) monitor() {
	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	err := p.cmd.Wait()

	p.mu.Lock()
	p.running = false
	p.waitErr = err

	if exitError, ok := err.(*exec.ExitError); ok {
		p.exitCode = exitError.ExitCode()
	} else if err == nil {
		p.exitCode = 0
	} else {
		p.exitCode = -1
	}
	p.mu.Unlock()

	close(p.done)

	if p.stdout != nil {
		p.stdout.Close()
	}
	if p.stderr != nil {
		p.stderr.Close()
	}
}
