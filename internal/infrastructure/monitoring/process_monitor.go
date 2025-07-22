package monitoring

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"kilometers.ai/cli/internal/application/ports"
)

// MCPProcessMonitor implements the ProcessMonitor interface and provides robust process management
// and transparent forwarding of MCP protocol messages. It launches a subprocess, separates stdout/stderr,
// extracts complete JSON protocol messages from stdout, and forwards them to a user-defined handler.
//
// Usage:
//
//	monitor := NewMCPProcessMonitor(logger)
//	monitor.SetProtocolHandler(func(msg []byte) error {
//	    // Forward to network/socket/SSE here
//	    return nil
//	})
//	monitor.Start(command, args)
//
// The protocolHandler is called for each complete JSON object detected on stdout.
// Non-JSON output is logged for diagnostics and not forwarded.
type MCPProcessMonitor struct {
	command         string
	args            []string
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	stderr          io.ReadCloser
	stdinChan       chan []byte
	stdoutChan      chan []byte
	stderrChan      chan []byte
	errorChan       chan error
	done            chan struct{}
	cancel          context.CancelFunc
	isRunning       bool
	exitCode        int
	startTime       time.Time
	logger          ports.LoggingGateway
	stats           *MonitoringStats
	mu              sync.RWMutex
	workingDir      string
	environment     map[string]string
	strictMCPMode   bool               // If true, only forward valid JSON lines to stdout
	protocolChan    chan []byte        // Channel for complete JSON protocol messages extracted from stdout
	protocolHandler func([]byte) error // Callback for forwarding protocol messages (e.g., to a network socket)
}

// MonitoringStats tracks process monitoring statistics
type MonitoringStats struct {
	TotalBytesRead    int64         `json:"total_bytes_read"`
	TotalBytesWritten int64         `json:"total_bytes_written"`
	MessagesProcessed int64         `json:"messages_processed"`
	ErrorCount        int64         `json:"error_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	UptimeSeconds     int64         `json:"uptime_seconds"`
	LastActivityTime  time.Time     `json:"last_activity_time"`
}

// NewMCPProcessMonitor creates a new process monitor
func NewMCPProcessMonitor(logger ports.LoggingGateway) *MCPProcessMonitor {
	return &MCPProcessMonitor{
		stdinChan:     make(chan []byte, 1000),
		stdoutChan:    make(chan []byte, 1000),
		stderrChan:    make(chan []byte, 1000),
		errorChan:     make(chan error, 100),
		done:          make(chan struct{}),
		logger:        logger,
		stats:         &MonitoringStats{},
		isRunning:     false,
		exitCode:      -1,
		strictMCPMode: true,                    // Enable strict MCP mode by default
		protocolChan:  make(chan []byte, 1000), // New: protocol channel
	}
}

// Start starts monitoring a process with the given command and arguments
func (m *MCPProcessMonitor) Start(command string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("process is already running")
	}

	m.command = command
	m.args = args
	m.startTime = time.Now()
	m.stats.LastActivityTime = time.Now()

	m.logger.Log(ports.LogLevelInfo, "Starting process", map[string]interface{}{
		"command": command,
		"args":    args,
	})

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// Create or recreate done channel
	m.done = make(chan struct{})

	// Create command
	m.cmd = exec.CommandContext(ctx, command, args...)

	// Set working directory if specified
	if m.workingDir != "" {
		m.cmd.Dir = m.workingDir
	}

	// Set environment variables
	if len(m.environment) > 0 {
		env := os.Environ()
		for key, value := range m.environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		m.cmd.Env = env
	}

	// Create pipes
	var err error

	m.stdin, err = m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	m.stdout, err = m.cmd.StdoutPipe()
	if err != nil {
		m.stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	m.stderr, err = m.cmd.StderrPipe()
	if err != nil {
		m.stdin.Close()
		m.stdout.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := m.cmd.Start(); err != nil {
		m.cleanup()
		return fmt.Errorf("failed to start process: %w", err)
	}

	m.isRunning = true

	// Start monitoring goroutines
	go m.monitorStdout(ctx)
	go m.monitorStderr(ctx)
	go m.monitorStdin(ctx)
	go m.waitForProcess(ctx)
	go m.proxyProtocolForwarder(ctx) // New: start protocol forwarder

	m.logger.Log(ports.LogLevelInfo, "Process started successfully", map[string]interface{}{
		"pid":     m.cmd.Process.Pid,
		"command": command,
	})

	return nil
}

// Stop stops the monitoring process
func (m *MCPProcessMonitor) Stop() error {
	m.mu.Lock()

	if !m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("process is not running")
	}

	m.logger.Log(ports.LogLevelInfo, "Stopping process", map[string]interface{}{
		"pid": m.cmd.Process.Pid,
	})

	// Cancel context to stop goroutines
	if m.cancel != nil {
		m.cancel()
	}

	// Get process reference before releasing lock
	process := m.cmd.Process
	done := m.done

	// Release lock before waiting to avoid deadlock
	m.mu.Unlock()

	// Try graceful shutdown first
	if process != nil {
		if err := process.Signal(syscall.SIGTERM); err != nil {
			m.logger.LogError(err, "Failed to send SIGTERM", nil)
		}

		// Wait for graceful shutdown using the done channel
		select {
		case <-done:
			m.logger.Log(ports.LogLevelInfo, "Process stopped gracefully", nil)
		case <-time.After(10 * time.Second):
			// Force kill if graceful shutdown takes too long
			m.logger.Log(ports.LogLevelWarn, "Forcing process termination", nil)
			if err := process.Kill(); err != nil {
				m.logger.LogError(err, "Failed to kill process", nil)
			}
			// Wait for the waitForProcess goroutine to finish
			<-done
		}
	} else {
		// If no process, wait for done channel anyway
		<-done
	}

	return nil
}

// IsRunning returns true if the process is currently running
func (m *MCPProcessMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetProcessInfo returns information about the monitored process
func (m *MCPProcessMonitor) GetProcessInfo() (*ports.ProcessInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cmd == nil || m.cmd.Process == nil {
		return nil, fmt.Errorf("no process information available")
	}

	status := ports.ProcessStatusUnknown
	if m.isRunning {
		status = ports.ProcessStatusRunning
	} else {
		status = ports.ProcessStatusStopped
	}

	return &ports.ProcessInfo{
		PID:        m.cmd.Process.Pid,
		Command:    m.command,
		Args:       m.args,
		StartTime:  m.startTime,
		Status:     status,
		ExitCode:   m.exitCode,
		CPUPercent: 0.0, // TODO: Implement CPU monitoring
		MemoryMB:   0,   // TODO: Implement memory monitoring
	}, nil
}

// ReadStdin returns a channel for reading stdin data
func (m *MCPProcessMonitor) ReadStdin() <-chan []byte {
	return m.stdinChan
}

// ReadStdout returns a channel for reading stdout data
func (m *MCPProcessMonitor) ReadStdout() <-chan []byte {
	return m.stdoutChan
}

// ReadStderr returns a channel for reading stderr data
func (m *MCPProcessMonitor) ReadStderr() <-chan []byte {
	return m.stderrChan
}

// WriteStdin writes data to the process stdin
func (m *MCPProcessMonitor) WriteStdin(data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning || m.stdin == nil {
		return fmt.Errorf("process is not running or stdin is not available")
	}

	n, err := m.stdin.Write(data)
	if err != nil {
		m.logger.LogError(err, "Failed to write to stdin", map[string]interface{}{
			"data_size": len(data),
		})
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	m.updateStats(0, int64(n), 0, 0, false)

	m.logger.Log(ports.LogLevelDebug, "Data written to stdin", map[string]interface{}{
		"bytes_written": n,
	})

	return nil
}

// Wait waits for the process to complete
func (m *MCPProcessMonitor) Wait() error {
	if !m.IsRunning() {
		return fmt.Errorf("process is not running")
	}

	err := m.cmd.Wait()
	m.logger.Log(ports.LogLevelInfo, "🔚 Child process exited", map[string]interface{}{"pid": m.cmd.Process.Pid})
	return err
}

// GetExitCode returns the exit code of the process
func (m *MCPProcessMonitor) GetExitCode() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exitCode
}

// GetMonitoringStats returns monitoring statistics
func (m *MCPProcessMonitor) GetMonitoringStats() (*ports.MonitoringStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := int64(0)
	if m.isRunning {
		uptime = int64(time.Since(m.startTime).Seconds())
	}

	return &ports.MonitoringStats{
		TotalBytesRead:    m.stats.TotalBytesRead,
		TotalBytesWritten: m.stats.TotalBytesWritten,
		MessagesProcessed: m.stats.MessagesProcessed,
		ErrorCount:        m.stats.ErrorCount,
		AverageLatency:    m.stats.AverageLatency,
		UptimeSeconds:     uptime,
	}, nil
}

// SetWorkingDirectory sets the working directory for the process
func (m *MCPProcessMonitor) SetWorkingDirectory(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workingDir = dir
}

// SetEnvironment sets environment variables for the process
func (m *MCPProcessMonitor) SetEnvironment(env map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environment = env
}

// SetStrictMCPMode enables or disables strict MCP protocol filtering
func (m *MCPProcessMonitor) SetStrictMCPMode(strict bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strictMCPMode = strict
}

// SetProtocolHandler sets the callback for forwarding protocol messages.
// The handler is called for each complete JSON object detected on stdout.
// This enables transparent proxying of MCP protocol messages to a client/server connection.
func (m *MCPProcessMonitor) SetProtocolHandler(handler func([]byte) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.protocolHandler = handler
}

// monitorStdout monitors stdout and forwards data to channel
func (m *MCPProcessMonitor) monitorStdout(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Log(ports.LogLevelError, "Stdout monitor panicked", map[string]interface{}{
				"error": r,
			})
		}
		m.logger.Log(ports.LogLevelDebug, "\U0001F51A monitorStdout goroutine exiting", nil)
	}()

	m.mu.RLock()
	stdout := m.stdout
	m.mu.RUnlock()

	if stdout == nil {
		return
	}

	reader := bufio.NewReaderSize(stdout, 1024*1024)
	var accumulator []byte

	// --- MCP JSON OBJECT FRAMING (brace-matching) ---
	var braceCount int
	var inString, escapeNext bool

	for {
		select {
		case <-ctx.Done():
			return
		default:
			chunk := make([]byte, 8192)
			n, err := reader.Read(chunk)
			if err != nil {
				if err != io.EOF {
					m.logger.LogError(err, "Error reading stdout", nil)
					m.updateStats(0, 0, 0, 1, true)
				}
				return
			}

			if n > 0 {
				accumulator = append(accumulator, chunk[:n]...)
				start := 0
				for i := 0; i < len(accumulator); i++ {
					b := accumulator[i]
					if inString {
						if escapeNext {
							escapeNext = false
						} else if b == '\\' {
							escapeNext = true
						} else if b == '"' {
							inString = false
						}
					} else {
						if b == '"' {
							inString = true
						} else if b == '{' {
							braceCount++
						} else if b == '}' {
							braceCount--
							if braceCount == 0 {
								// Found a complete JSON object
								obj := accumulator[start : i+1]
								select {
								case m.protocolChan <- obj:
									// Success
								case <-ctx.Done():
									return
								default:
									m.logger.Log(ports.LogLevelWarn, "protocolChan full, dropping message", map[string]interface{}{"bytes": len(obj)})
								}
								start = i + 1
							}
						}
					}
				}
				// Log any non-JSON output (bytes before start)
				if start > 0 && start < len(accumulator) {
					nonJSON := accumulator[:start]
					if len(bytes.TrimSpace(nonJSON)) > 0 {
						m.logger.Log(ports.LogLevelWarn, "Non-JSON output detected on stdout", map[string]interface{}{"output": string(nonJSON)})
						// Optionally increment a counter for diagnostics
					}
				}
				// Remove processed bytes from accumulator
				if start > 0 {
					accumulator = accumulator[start:]
				}
				// If accumulator gets too large, reset to avoid memory issues
				if len(accumulator) > 10*1024*1024 {
					m.logger.Log(ports.LogLevelWarn, "Stdout accumulator too large, resetting", map[string]interface{}{
						"size": len(accumulator),
					})
					accumulator = nil
					braceCount = 0
					inString = false
					escapeNext = false
				}
			}
		}
	}
}

// monitorStderr monitors stderr and forwards data to channel
func (m *MCPProcessMonitor) monitorStderr(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Log(ports.LogLevelError, "Stderr monitor panicked", map[string]interface{}{
				"error": r,
			})
		}
	}()

	// Get stderr reference with proper locking to avoid race with cleanup
	m.mu.RLock()
	stderr := m.stderr
	m.mu.RUnlock()

	if stderr == nil {
		return
	}

	reader := bufio.NewReaderSize(stderr, 1024*1024) // 1MB buffer for large MCP messages
	var accumulator []byte

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read data in chunks
			chunk := make([]byte, 8192)
			n, err := reader.Read(chunk)
			if err != nil {
				if err != io.EOF {
					m.logger.LogError(err, "Error reading stderr", nil)
					m.updateStats(0, 0, 0, 1, true)
				}
				return
			}

			if n > 0 {
				// Append to accumulator
				accumulator = append(accumulator, chunk[:n]...)

				// Try to extract complete JSON-RPC messages (newline-delimited)
				for {
					newlineIdx := bytes.IndexByte(accumulator, '\n')
					if newlineIdx == -1 {
						break // No complete message yet
					}

					// Extract complete message (including newline)
					message := accumulator[:newlineIdx+1]
					accumulator = accumulator[newlineIdx+1:]

					// Send complete message
					if len(bytes.TrimSpace(message)) > 0 {
						select {
						case m.stderrChan <- message:
							m.logger.Log(ports.LogLevelDebug, "Stderr message received", map[string]interface{}{
								"bytes": len(message),
							})
						case <-ctx.Done():
							return
						default:
							m.logger.Log(ports.LogLevelWarn, "Stderr channel full, dropping message", map[string]interface{}{
								"bytes": len(message),
							})
						}
					}
				}

				// If accumulator gets too large, it might indicate a problem
				if len(accumulator) > 10*1024*1024 { // 10MB limit
					m.logger.Log(ports.LogLevelWarn, "Stderr accumulator too large, resetting", map[string]interface{}{
						"size": len(accumulator),
					})
					accumulator = nil
				}
			}
		}
	}
}

// monitorStdin monitors stdin channel and forwards data to process
func (m *MCPProcessMonitor) monitorStdin(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Log(ports.LogLevelError, "Stdin monitor panicked", map[string]interface{}{
				"error": r,
			})
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-m.stdinChan:
			// Use read lock to safely access stdin field
			m.mu.RLock()
			stdin := m.stdin
			m.mu.RUnlock()

			if stdin != nil {
				_, err := stdin.Write(data)
				if err != nil {
					m.logger.LogError(err, "Failed to write stdin data to process", nil)
					m.updateStats(0, 0, 0, 1, true)
				} else {
					m.updateStats(0, int64(len(data)), 0, 0, false)
				}
			}
		}
	}
}

// waitForProcess waits for the process to exit
func (m *MCPProcessMonitor) waitForProcess(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Log(ports.LogLevelError, "Process waiter panicked", map[string]interface{}{
				"error": r,
			})
		}
	}()

	err := m.cmd.Wait()

	// Get exit code before acquiring lock to avoid potential deadlock
	var exitCode int
	if m.cmd.ProcessState != nil {
		exitCode = m.cmd.ProcessState.ExitCode()
	}

	m.mu.Lock()
	m.isRunning = false
	m.exitCode = exitCode
	m.mu.Unlock()

	if err != nil {
		m.logger.LogError(err, "Process exited with error", map[string]interface{}{
			"exit_code": m.exitCode,
		})
	} else {
		m.logger.Log(ports.LogLevelInfo, "Process exited normally", map[string]interface{}{
			"exit_code": m.exitCode,
		})
	}

	// Clean up resources
	m.cleanup()

	// Signal completion by closing done channel
	close(m.done)
}

// proxyProtocolForwarder reads from protocolChan and calls the protocolHandler for each message.
// This decouples protocol extraction from the actual forwarding mechanism, enabling flexible proxying.
func (m *MCPProcessMonitor) proxyProtocolForwarder(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-m.protocolChan:
			m.mu.RLock()
			handler := m.protocolHandler
			m.mu.RUnlock()
			if handler != nil {
				if err := handler(msg); err != nil {
					m.logger.LogError(err, "Failed to forward protocol message", nil)
				}
			}
		}
	}
}

// cleanup closes all pipes and channels
func (m *MCPProcessMonitor) cleanup() {
	// Use write lock to safely modify IO fields
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stdin != nil {
		m.stdin.Close()
		m.stdin = nil
	}
	if m.stdout != nil {
		m.stdout.Close()
		m.stdout = nil
	}
	if m.stderr != nil {
		m.stderr.Close()
		m.stderr = nil
	}
}

// updateStats updates monitoring statistics
func (m *MCPProcessMonitor) updateStats(bytesRead, bytesWritten, messages, errors int64, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalBytesRead += bytesRead
	m.stats.TotalBytesWritten += bytesWritten
	m.stats.MessagesProcessed += messages
	m.stats.ErrorCount += errors
	m.stats.LastActivityTime = time.Now()

	if isError {
		m.stats.ErrorCount++
	}
}

// GetErrorChannel returns the error channel for external monitoring
func (m *MCPProcessMonitor) GetErrorChannel() <-chan error {
	return m.errorChan
}

// SendError sends an error to the error channel
func (m *MCPProcessMonitor) SendError(err error) {
	select {
	case m.errorChan <- err:
	default:
		// Error channel is full, log directly
		m.logger.LogError(err, "Error channel full, logging directly", nil)
	}
}

// New: Getter for protocolChan
func (m *MCPProcessMonitor) ReadProtocol() <-chan []byte {
	return m.protocolChan
}
