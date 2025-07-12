package monitoring

import (
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

// MCPProcessMonitor implements the ProcessMonitor interface
type MCPProcessMonitor struct {
	command     string
	args        []string
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	stdinChan   chan []byte
	stdoutChan  chan []byte
	stderrChan  chan []byte
	errorChan   chan error
	done        chan struct{}
	cancel      context.CancelFunc
	isRunning   bool
	exitCode    int
	startTime   time.Time
	logger      ports.LoggingGateway
	stats       *MonitoringStats
	mu          sync.RWMutex
	workingDir  string
	environment map[string]string
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
		stdinChan:  make(chan []byte, 1000),
		stdoutChan: make(chan []byte, 1000),
		stderrChan: make(chan []byte, 1000),
		errorChan:  make(chan error, 100),
		done:       make(chan struct{}),
		logger:     logger,
		stats:      &MonitoringStats{},
		isRunning:  false,
		exitCode:   -1,
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

	m.logger.Log(ports.LogLevelInfo, "Process started successfully", map[string]interface{}{
		"pid":     m.cmd.Process.Pid,
		"command": command,
	})

	return nil
}

// Stop stops the monitoring process
func (m *MCPProcessMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("process is not running")
	}

	m.logger.Log(ports.LogLevelInfo, "Stopping process", map[string]interface{}{
		"pid": m.cmd.Process.Pid,
	})

	// Cancel context to stop goroutines
	if m.cancel != nil {
		m.cancel()
	}

	// Try graceful shutdown first
	if m.cmd.Process != nil {
		if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			m.logger.LogError(err, "Failed to send SIGTERM", nil)
		}

		// Wait for graceful shutdown
		gracefulDone := make(chan bool, 1)
		go func() {
			m.cmd.Wait()
			gracefulDone <- true
		}()

		select {
		case <-gracefulDone:
			m.logger.Log(ports.LogLevelInfo, "Process stopped gracefully", nil)
		case <-time.After(10 * time.Second):
			// Force kill if graceful shutdown takes too long
			m.logger.Log(ports.LogLevelWarn, "Forcing process termination", nil)
			if err := m.cmd.Process.Kill(); err != nil {
				m.logger.LogError(err, "Failed to kill process", nil)
			}
			m.cmd.Wait()
		}
	}

	m.cleanup()
	m.isRunning = false

	// Close done channel to signal completion
	close(m.done)

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

	// Wait for the done channel to be closed
	<-m.done

	return nil
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

// monitorStdout monitors stdout and forwards data to channel
func (m *MCPProcessMonitor) monitorStdout(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Log(ports.LogLevelError, "Stdout monitor panicked", map[string]interface{}{
				"error": r,
			})
		}
	}()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := m.stdout.Read(buffer)
			if err != nil {
				if err != io.EOF {
					m.logger.LogError(err, "Error reading stdout", nil)
					m.updateStats(0, 0, 0, 1, true)
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case m.stdoutChan <- data:
					m.updateStats(int64(n), 0, 1, 0, false)
					m.logger.Log(ports.LogLevelDebug, "Stdout data received", map[string]interface{}{
						"bytes": n,
					})
				case <-ctx.Done():
					return
				default:
					// Channel is full, drop the data
					m.logger.Log(ports.LogLevelWarn, "Stdout channel full, dropping data", map[string]interface{}{
						"bytes": n,
					})
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

	buffer := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := m.stderr.Read(buffer)
			if err != nil {
				if err != io.EOF {
					m.logger.LogError(err, "Error reading stderr", nil)
					m.updateStats(0, 0, 0, 1, true)
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case m.stderrChan <- data:
					m.logger.Log(ports.LogLevelDebug, "Stderr data received", map[string]interface{}{
						"bytes": n,
					})
				case <-ctx.Done():
					return
				default:
					// Channel is full, drop the data
					m.logger.Log(ports.LogLevelWarn, "Stderr channel full, dropping data", map[string]interface{}{
						"bytes": n,
					})
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
			if m.stdin != nil {
				_, err := m.stdin.Write(data)
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

	m.mu.Lock()
	m.isRunning = false

	if m.cmd.ProcessState != nil {
		m.exitCode = m.cmd.ProcessState.ExitCode()
	}
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
}

// cleanup closes all pipes and channels
func (m *MCPProcessMonitor) cleanup() {
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
