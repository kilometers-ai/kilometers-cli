package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"kilometers.ai/cli/internal/application/commands"
	"kilometers.ai/cli/internal/application/ports"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/filtering"
	"kilometers.ai/cli/internal/core/risk"
	"kilometers.ai/cli/internal/core/session"
)

// MonitoringService orchestrates monitoring operations
type MonitoringService struct {
	sessionRepo    ports.SessionRepository
	eventStore     ports.EventStore
	apiGateway     ports.APIGateway
	processMonitor ports.ProcessMonitor
	riskAnalyzer   risk.RiskAnalyzer
	eventFilter    filtering.EventFilter
	logger         ports.LoggingGateway
	config         *ports.Configuration
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(
	sessionRepo ports.SessionRepository,
	eventStore ports.EventStore,
	apiGateway ports.APIGateway,
	processMonitor ports.ProcessMonitor,
	riskAnalyzer risk.RiskAnalyzer,
	eventFilter filtering.EventFilter,
	logger ports.LoggingGateway,
	config *ports.Configuration,
) *MonitoringService {
	return &MonitoringService{
		sessionRepo:    sessionRepo,
		eventStore:     eventStore,
		apiGateway:     apiGateway,
		processMonitor: processMonitor,
		riskAnalyzer:   riskAnalyzer,
		eventFilter:    eventFilter,
		logger:         logger,
		config:         config,
	}
}

// StartMonitoring handles the start monitoring command
func (s *MonitoringService) StartMonitoring(ctx context.Context, cmd *commands.StartMonitoringCommand) (*commands.CommandResult, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return commands.NewErrorResult("Validation failed", []string{err.Error()}), nil
	}

	// Check if there's already an active session
	activeSession, err := s.sessionRepo.FindActive()
	if err != nil {
		s.logger.LogError(err, "Failed to check for active session", nil)
		return commands.NewErrorResult("Failed to check for active session", []string{err.Error()}), nil
	}

	if activeSession != nil {
		return commands.NewErrorResult("Active monitoring session already exists", []string{
			fmt.Sprintf("Session ID: %s", activeSession.ID().Value()),
		}), nil
	}

	// Create new session
	sessionConfig := cmd.SessionConfig
	if sessionConfig.BatchSize == 0 {
		sessionConfig.BatchSize = s.config.BatchSize
	}
	if sessionConfig.FlushInterval == 0 {
		sessionConfig.FlushInterval = time.Duration(s.config.FlushInterval) * time.Second
	}

	newSession := session.NewSession(sessionConfig)
	if err := newSession.Start(); err != nil {
		s.logger.LogError(err, "Failed to start session", nil)
		return commands.NewErrorResult("Failed to start session", []string{err.Error()}), nil
	}

	// Save session to repository
	if err := s.sessionRepo.Save(newSession); err != nil {
		s.logger.LogError(err, "Failed to save session", nil)
		return commands.NewErrorResult("Failed to save session", []string{err.Error()}), nil
	}

	// Create session on the server via API
	if err := s.apiGateway.CreateSession(newSession); err != nil {
		s.logger.LogError(err, "Failed to create session on server", nil)
		// Don't fail the operation, just log the error - session can work locally
	}

	// Start process monitoring
	if err := s.processMonitor.Start(cmd.ProcessCommand, cmd.ProcessArgs); err != nil {
		s.logger.LogError(err, "Failed to start process monitoring", nil)
		// Clean up session
		newSession.End()
		s.sessionRepo.Update(newSession)
		return commands.NewErrorResult("Failed to start process monitoring", []string{err.Error()}), nil
	}

	// Start event processing in background
	go s.processEvents(ctx, newSession)

	// Create result
	result := commands.NewSuccessResult("Monitoring started successfully", commands.MonitoringResult{
		SessionID: newSession.ID(),
		Status:    "active",
		Message:   "Monitoring session started",
		StartTime: newSession.StartTime(),
		Duration:  newSession.Duration(),
	})

	result.SetMetadata("session_id", newSession.ID().Value())
	result.SetMetadata("process_command", cmd.ProcessCommand)
	result.SetMetadata("process_args", cmd.ProcessArgs)

	s.logger.LogSession(newSession, "Monitoring session started")
	return result, nil
}

// StopMonitoring handles the stop monitoring command
func (s *MonitoringService) StopMonitoring(ctx context.Context, cmd *commands.StopMonitoringCommand) (*commands.CommandResult, error) {
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

	// Stop process monitoring
	if s.processMonitor.IsRunning() {
		if err := s.processMonitor.Stop(); err != nil {
			s.logger.LogError(err, "Failed to stop process monitoring", nil)
			if !cmd.ForceStop {
				return commands.NewErrorResult("Failed to stop process monitoring", []string{err.Error()}), nil
			}
		}
	}

	// End session and flush any remaining events
	finalBatch, err := sessionObj.End()
	if err != nil {
		s.logger.LogError(err, "Failed to end session", nil)
		return commands.NewErrorResult("Failed to end session", []string{err.Error()}), nil
	}

	// Send final batch if exists
	if finalBatch != nil && len(finalBatch.Events) > 0 {
		if err := s.sendEventBatch(ctx, finalBatch); err != nil {
			s.logger.LogError(err, "Failed to send final batch", nil)
			// Don't fail the command for this - log warning instead
		}
	}

	// Update session in repository
	if err := s.sessionRepo.Update(sessionObj); err != nil {
		s.logger.LogError(err, "Failed to update session", nil)
		return commands.NewErrorResult("Failed to update session", []string{err.Error()}), nil
	}

	// Create result
	result := commands.NewSuccessResult("Monitoring stopped successfully", commands.MonitoringResult{
		SessionID:    sessionObj.ID(),
		Status:       "stopped",
		Message:      "Monitoring session stopped",
		EventsCount:  sessionObj.TotalEvents(),
		BatchesCount: sessionObj.BatchedEvents() / sessionObj.Config().BatchSize,
		StartTime:    sessionObj.StartTime(),
		EndTime:      sessionObj.EndTime(),
		Duration:     sessionObj.Duration(),
	})

	result.SetMetadata("session_id", sessionObj.ID().Value())
	result.SetMetadata("total_events", sessionObj.TotalEvents())
	result.SetMetadata("duration_seconds", sessionObj.Duration().Seconds())

	s.logger.LogSession(sessionObj, "Monitoring session stopped")
	return result, nil
}

// GetSessionStatus handles the get session status command
func (s *MonitoringService) GetSessionStatus(ctx context.Context, cmd *commands.GetSessionStatusCommand) (*commands.CommandResult, error) {
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

	// Build session status
	status := &commands.SessionStatus{
		SessionID:    sessionObj.ID(),
		State:        string(sessionObj.State()),
		LastActivity: sessionObj.StartTime(), // TODO: Track last activity
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

	// Add recent events if requested
	if cmd.IncludeEvents {
		events := sessionObj.GetEventHistory()
		limit := cmd.EventLimit
		if limit > len(events) {
			limit = len(events)
		}

		recentEvents := make([]commands.EventSummary, 0, limit)
		for i := len(events) - limit; i < len(events); i++ {
			evt := events[i]
			recentEvents = append(recentEvents, commands.EventSummary{
				ID:        evt.ID().Value(),
				Method:    evt.Method().Value(),
				Direction: evt.Direction().String(),
				Size:      evt.Size(),
				RiskScore: evt.RiskScore().Value(),
				Timestamp: evt.Timestamp(),
			})
		}
		status.RecentEvents = recentEvents
	}

	// Add process info if available
	if s.processMonitor.IsRunning() {
		if processInfo, err := s.processMonitor.GetProcessInfo(); err == nil {
			status.ProcessInfo = &commands.ProcessInfo{
				PID:       processInfo.PID,
				Command:   processInfo.Command,
				Args:      processInfo.Args,
				Status:    string(processInfo.Status),
				ExitCode:  processInfo.ExitCode,
				StartTime: processInfo.StartTime,
			}
		}
	}

	return commands.NewSuccessResult("Session status retrieved", status), nil
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

	// Start reading from process monitor
	go s.readProcessOutput(ctx, sessionObj, eventChan, errorChan)

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

// processEvent processes a single event
func (s *MonitoringService) processEvent(ctx context.Context, sessionObj *session.Session, evt *event.Event) error {
	// Apply risk analysis
	riskScore, err := s.riskAnalyzer.AnalyzeEvent(evt)
	if err != nil {
		s.logger.LogError(err, "Failed to analyze event risk", nil)
		// Continue processing with existing risk score
	} else {
		evt.UpdateRiskScore(riskScore)
	}

	// Apply filtering
	if !s.eventFilter.ShouldCapture(evt) {
		// Event was filtered out
		return nil
	}

	// Add event to session
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
func (s *MonitoringService) readProcessOutput(ctx context.Context, sessionObj *session.Session, eventChan chan<- *event.Event, errorChan chan<- error) {
	// This is a simplified implementation
	// In reality, you would parse MCP messages from the process output

	stdoutChan := s.processMonitor.ReadStdout()
	stderrChan := s.processMonitor.ReadStderr()

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-stdoutChan:
			// Forward to stdout for transparent proxy mode (IMPORTANT: Do this first!)
			os.Stdout.Write(data)

			if evt, err := s.parseEventFromData(data, event.DirectionOutbound); err == nil {
				eventChan <- evt
			} else {
				errorChan <- err
			}
		case data := <-stderrChan:
			// Handle stderr data if needed
			s.logger.Log(ports.LogLevelDebug, "Received stderr data", map[string]interface{}{
				"data_size": len(data),
			})
		}
	}
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
