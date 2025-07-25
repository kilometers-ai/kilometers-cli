package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// MonitoringService orchestrates monitoring operations
type MonitoringService struct {
	sessionRepo    ports.SessionRepository
	eventStore     ports.EventStore
	apiGateway     ports.APIGateway
	processMonitor ports.ProcessMonitor
	logger         ports.LoggingGateway
	config         *ports.Configuration
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(
	sessionRepo ports.SessionRepository,
	eventStore ports.EventStore,
	apiGateway ports.APIGateway,
	processMonitor ports.ProcessMonitor,
	logger ports.LoggingGateway,
	config *ports.Configuration,
) *MonitoringService {
	return &MonitoringService{
		sessionRepo:    sessionRepo,
		eventStore:     eventStore,
		apiGateway:     apiGateway,
		processMonitor: processMonitor,
		logger:         logger,
		config:         config,
	}
}

// StartMonitoring starts a new monitoring session
func (s *MonitoringService) StartMonitoring(ctx context.Context, cmd *commands.StartMonitoringCommand) (*commands.MonitoringResult, error) {
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Use configuration defaults if not specified in session config
	sessionConfig := cmd.SessionConfig
	if sessionConfig.BatchSize == 0 {
		sessionConfig.BatchSize = s.config.BatchSize
	}

	// Create new session
	newSession := session.NewSession(sessionConfig)

	// Debug mode: handle replay file if specified
	if cmd.DebugReplayFile != "" {
		return s.handleDebugReplay(ctx, cmd, newSession)
	}

	// Store the session
	if err := s.sessionRepo.Save(newSession); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Start the session to transition it to active state
	if err := newSession.Start(); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Start the process monitor
	if err := s.processMonitor.Start(cmd.Command, cmd.Arguments); err != nil {
		return nil, fmt.Errorf("failed to start process monitor: %w", err)
	}

	// Start monitoring in background
	go s.processEvents(ctx, newSession)

	result := &commands.MonitoringResult{
		Success:   true,
		SessionID: newSession.ID(),
		Status:    "active",
		Message:   "Monitoring started successfully",
		StartTime: time.Now(),
		Metadata: map[string]interface{}{
			"session_id": newSession.ID().String(),
		},
	}

	return result, nil
}

// StopMonitoring stops an active monitoring session
func (s *MonitoringService) StopMonitoring(ctx context.Context, cmd *commands.StopMonitoringCommand) (*commands.MonitoringResult, error) {
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Find the session
	activeSession, err := s.sessionRepo.FindByID(cmd.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	// End the session
	_, err = activeSession.End()
	if err != nil {
		return nil, fmt.Errorf("failed to end session: %w", err)
	}

	// Stop the process monitor only if it's running
	if s.processMonitor.IsRunning() {
		if err := s.processMonitor.Stop(); err != nil {
			return nil, fmt.Errorf("failed to stop process monitor: %w", err)
		}
	}

	// Save the updated session
	if err := s.sessionRepo.Save(activeSession); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	result := &commands.MonitoringResult{
		Success:   true,
		SessionID: activeSession.ID(),
		Status:    "stopped",
		Message:   "Monitoring stopped successfully",
	}

	return result, nil
}

// GetMonitoringStatus returns the status of monitoring sessions
func (s *MonitoringService) GetMonitoringStatus(ctx context.Context, cmd *commands.GetMonitoringStatusCommand) (*commands.MonitoringResult, error) {
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	var activeSession *session.Session
	var err error

	if cmd.SessionID != nil {
		// Get specific session
		activeSession, err = s.sessionRepo.FindByID(*cmd.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to find session: %w", err)
		}
	} else {
		// Get active session
		activeSession, err = s.sessionRepo.FindActive()
		if err != nil {
			return nil, fmt.Errorf("failed to find active session: %w", err)
		}
	}

	if activeSession == nil {
		return &commands.MonitoringResult{
			Success: true,
			Status:  "inactive",
			Message: "No active monitoring session",
		}, nil
	}

	result := &commands.MonitoringResult{
		Success:   true,
		SessionID: activeSession.ID(),
		Status:    "active",
		Message:   "Monitoring status retrieved",
		Metadata: map[string]interface{}{
			"session_id": activeSession.ID().String(),
			"state":      string(activeSession.State()),
		},
	}

	return result, nil
}

// ListActiveSessions handles the list active sessions command
func (s *MonitoringService) ListActiveSessions(ctx context.Context, cmd *commands.ListActiveSessionsCommand) (*commands.CommandResult, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return commands.NewErrorResult("Validation failed", []string{err.Error()}), nil
	}

	// Get active sessions
	sessions, err := s.sessionRepo.FindByState(session.SessionStateActive)
	if err != nil {
		s.logger.LogError(err, "Failed to find active sessions", nil)
		return commands.NewErrorResult("Failed to find active sessions", []string{err.Error()}), nil
	}

	// Apply pagination
	start := cmd.Offset
	end := cmd.Offset + cmd.Limit
	if start > len(sessions) {
		start = len(sessions)
	}
	if end > len(sessions) {
		end = len(sessions)
	}

	paginatedSessions := sessions[start:end]
	sessionStatuses := make([]commands.SessionStatus, 0, len(paginatedSessions))

	for _, sessionObj := range paginatedSessions {
		status := commands.SessionStatus{
			SessionID:    sessionObj.ID(),
			State:        string(sessionObj.State()),
			LastActivity: sessionObj.StartTime(),
			CreatedAt:    sessionObj.StartTime(),
			UpdatedAt:    time.Now(),
		}

		// Add statistics if requested
		if cmd.IncludeStats {
			status.Statistics = &commands.SessionStats{
				TotalEvents:     sessionObj.TotalEvents(),
				CapturedEvents:  sessionObj.BatchedEvents(),
				TotalBatches:    sessionObj.BatchedEvents() / sessionObj.Config().BatchSize,
				SessionDuration: sessionObj.Duration(),
				EventsPerSecond: float64(sessionObj.TotalEvents()) / sessionObj.Duration().Seconds(),
			}
		}

		sessionStatuses = append(sessionStatuses, status)
	}

	result := commands.NewSuccessResult("Active sessions retrieved", sessionStatuses)
	result.SetMetadata("total_sessions", len(sessions))
	result.SetMetadata("returned_sessions", len(sessionStatuses))
	result.SetMetadata("offset", cmd.Offset)
	result.SetMetadata("limit", cmd.Limit)

	return result, nil
}

// FlushEvents handles the flush events command
func (s *MonitoringService) FlushEvents(ctx context.Context, cmd *commands.FlushEventsCommand) (*commands.CommandResult, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return commands.NewErrorResult("Validation failed", []string{err.Error()}), nil
	}

	// Find session
	sessionObj, err := s.sessionRepo.FindByID(cmd.SessionID)
	if err != nil {
		s.logger.LogError(err, "Failed to find session", map[string]interface{}{
			"session_id": cmd.SessionID.Value(),
		})
		return commands.NewErrorResult("Session not found", []string{err.Error()}), nil
	}

	if !sessionObj.IsActive() {
		return commands.NewErrorResult("Session is not active", []string{
			fmt.Sprintf("Session state: %s", sessionObj.State()),
		}), nil
	}

	// Force flush events
	batch, err := sessionObj.ForceFlush()
	if err != nil {
		s.logger.LogError(err, "Failed to flush events", nil)
		return commands.NewErrorResult("Failed to flush events", []string{err.Error()}), nil
	}

	var flushedCount int
	if batch != nil {
		flushedCount = len(batch.Events)
		// Send the batch
		if err := s.sendEventBatch(ctx, batch); err != nil {
			s.logger.LogError(err, "Failed to send flushed batch", nil)
			return commands.NewErrorResult("Failed to send flushed batch", []string{err.Error()}), nil
		}
	}

	// Update session in repository
	if err := s.sessionRepo.Update(sessionObj); err != nil {
		s.logger.LogError(err, "Failed to update session", nil)
		return commands.NewErrorResult("Failed to update session", []string{err.Error()}), nil
	}

	result := commands.NewSuccessResult("Events flushed successfully", map[string]interface{}{
		"session_id":     sessionObj.ID().Value(),
		"flushed_events": flushedCount,
	})

	result.SetMetadata("flushed_events", flushedCount)
	return result, nil
}

// processEvents processes events from the monitoring session
func (s *MonitoringService) processEvents(ctx context.Context, sessionObj *session.Session) {
	s.logger.Log(ports.LogLevelInfo, "Starting event processing", map[string]interface{}{
		"session_id": sessionObj.ID().Value(),
	})

	// Create channels for event processing
	eventChan := make(chan *event.Event, 100)
	errorChan := make(chan error, 10)

	// Start reading from process monitor (this also handles stdout/stderr forwarding)
	go s.readProcessOutput(ctx, eventChan, errorChan)

	// Start forwarding stdin from parent process (Cursor) to MCP server
	go s.forwardStdin(ctx, eventChan)

	// Process events
	for {
		select {
		case <-ctx.Done():
			s.logger.Log(ports.LogLevelInfo, "Context cancelled, stopping event processing", nil)
			return
		case evt := <-eventChan:
			if err := s.processEvent(ctx, sessionObj, evt); err != nil {
				s.logger.LogError(err, "Failed to process event", map[string]interface{}{
					"event_id":   evt.ID().Value(),
					"session_id": sessionObj.ID().Value(),
				})
			}
		case err := <-errorChan:
			s.logger.LogError(err, "Error in event processing", map[string]interface{}{
				"session_id": sessionObj.ID().Value(),
			})
		}
	}
}

// forwardStdin forwards stdin from parent process (Cursor) to MCP server
func (s *MonitoringService) forwardStdin(ctx context.Context, eventChan chan<- *event.Event) {
	buffer := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read from parent process stdin
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				if err != io.EOF {
					s.logger.LogError(err, "Failed to read from parent stdin", nil)
				}
				return
			}
			if n > 0 {
				// Forward to MCP server stdin
				if err := s.processMonitor.WriteStdin(buffer[:n]); err != nil {
					s.logger.LogError(err, "Failed to forward stdin to MCP server", map[string]interface{}{
						"data_size": n,
					})
				}

				// Also create events for inbound messages for monitoring
				if isValidJSONRPC(buffer[:n]) {
					if evt, err := s.parseEventFromData(buffer[:n], event.DirectionInbound); err == nil {
						// Try to send to event channel non-blocking
						select {
						case eventChan <- evt:
							// Event sent successfully
						default:
							// Channel full, skip this event
						}
					}
				}
			}
		}
	}
}

// processEvent processes a single event (simplified without filtering)
func (s *MonitoringService) processEvent(ctx context.Context, sessionObj *session.Session, evt *event.Event) error {
	// Add event to session (no filtering applied)
	batch, err := sessionObj.AddEvent(evt)
	if err != nil {
		return fmt.Errorf("failed to add event to session: %w", err)
	}

	// If batch is ready, send it
	if batch != nil {
		if err := s.sendEventBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to send event batch: %w", err)
		}
	}

	// Update session in repository
	if err := s.sessionRepo.Update(sessionObj); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// readProcessOutput reads output from the process monitor and creates events
func (s *MonitoringService) readProcessOutput(ctx context.Context, eventChan chan<- *event.Event, errorChan chan<- error) {
	// This implementation both monitors AND forwards data for transparent MCP wrapper functionality

	stdoutChan := s.processMonitor.ReadStdout()
	stderrChan := s.processMonitor.ReadStderr()

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-stdoutChan:
			// Forward to parent process stdout FIRST (for Cursor to receive MCP responses)
			if _, err := os.Stdout.Write(data); err != nil {
				s.logger.LogError(err, "Failed to forward stdout to parent", map[string]interface{}{
					"data_size": len(data),
				})
			}

			// Then process for monitoring/events
			if isValidJSONRPC(data) {
				if evt, err := s.parseEventFromData(data, event.DirectionOutbound); err == nil {
					eventChan <- evt
				}
			}
		case data := <-stderrChan:
			// Forward stderr to parent process stderr
			if _, err := os.Stderr.Write(data); err != nil {
				s.logger.LogError(err, "Failed to forward stderr to parent", map[string]interface{}{
					"data_size": len(data),
				})
			}

			// Log stderr for debugging
			s.logger.Log(ports.LogLevelDebug, "Received stderr data", map[string]interface{}{
				"data_size": len(data),
			})
		}
	}
}

func isValidJSONRPC(data []byte) bool {
	var msg struct {
		JSONRPC string `json:"jsonrpc"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}

	return msg.JSONRPC == "2.0"
}

// parseEventFromData parses event data from process output
func (s *MonitoringService) parseEventFromData(data []byte, direction event.Direction) (*event.Event, error) {
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	var msg struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method,omitempty"`
		ID      json.RawMessage `json:"id,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   json.RawMessage `json:"error,omitempty"`
	}

	if err := json.Unmarshal(trimmedData, &msg); err != nil {
		s.logger.LogError(err, "Failed to parse JSON-RPC message", map[string]interface{}{
			"data_preview": string(trimmedData[:min(len(trimmedData), 200)]),
			"data_size":    len(trimmedData),
		})
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	methodName := msg.Method
	if methodName == "" {
		if msg.Error != nil {
			methodName = "error_response"
		} else {
			methodName = "response"
		}
	}

	method, err := event.NewMethod(methodName)
	if err != nil {
		return nil, err
	}

	riskScore := 10
	if strings.Contains(methodName, "write") ||
		strings.Contains(methodName, "delete") ||
		strings.Contains(methodName, "create") {
		riskScore = 50
	}

	score, err := event.NewRiskScore(riskScore)
	if err != nil {
		return nil, err
	}

	return event.CreateEvent(direction, method, trimmedData, score)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sendEventBatch sends an event batch to the API
func (s *MonitoringService) sendEventBatch(ctx context.Context, batch *session.EventBatch) error {
	// Store the batch locally first
	if err := s.eventStore.Store(batch); err != nil {
		s.logger.LogError(err, "Failed to store event batch locally", nil)
		// Continue with API send even if local storage fails
	}

	// Send to API
	if err := s.apiGateway.SendEventBatch(batch); err != nil {
		s.logger.LogError(err, "Failed to send event batch to API", map[string]interface{}{
			"batch_id":   batch.ID,
			"batch_size": batch.Size,
			"session_id": batch.SessionID.Value(),
		})
		return err
	}

	s.logger.Log(ports.LogLevelInfo, "Event batch sent successfully", map[string]interface{}{
		"batch_id":   batch.ID,
		"batch_size": batch.Size,
		"session_id": batch.SessionID.Value(),
	})

	return nil
}

// handleDebugReplay handles debug replay mode for testing
func (s *MonitoringService) handleDebugReplay(ctx context.Context, cmd *commands.StartMonitoringCommand, session *session.Session) (*commands.MonitoringResult, error) {
	// Validate replay file exists
	if _, err := os.Stat(cmd.DebugReplayFile); os.IsNotExist(err) {
		return &commands.MonitoringResult{
			Success: false,
			Message: fmt.Sprintf("Debug replay file not found: %s", cmd.DebugReplayFile),
		}, nil
	}

	// Store the session
	if err := s.sessionRepo.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Start the session to transition it to active state
	if err := session.Start(); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Start debug replay processing
	go s.processDebugReplay(ctx, session, cmd.DebugReplayFile)

	result := &commands.MonitoringResult{
		Success:   true,
		SessionID: session.ID(),
		Status:    "active",
		Message:   "Debug replay started successfully",
		StartTime: time.Now(),
		Metadata: map[string]interface{}{
			"session_id":  session.ID().String(),
			"debug_mode":  true,
			"replay_file": cmd.DebugReplayFile,
		},
	}

	return result, nil
}

// processDebugReplay processes events from a debug replay file
func (s *MonitoringService) processDebugReplay(ctx context.Context, session *session.Session, replayFile string) {
	// TODO: Implement actual debug replay processing
	// This would read JSON-RPC events from the replay file and process them
	// For now, this is a placeholder that simulates processing
	fmt.Printf("Debug replay processing would happen here for file: %s\n", replayFile)

	// Simulate some processing time for realistic testing
	time.Sleep(100 * time.Millisecond)
}

// GetProcessMonitor returns the process monitor for direct configuration
func (s *MonitoringService) GetProcessMonitor() ports.ProcessMonitor {
	return s.processMonitor
}
