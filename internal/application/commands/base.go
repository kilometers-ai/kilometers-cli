package commands

import (
	"context"
	"fmt"
	"time"
)

// Command defines the base interface for all commands
type Command interface {
	// Validate validates the command parameters
	Validate() error

	// GetType returns the command type
	GetType() string

	// GetID returns the command ID
	GetID() string
}

// CommandHandler defines the interface for command handlers
type CommandHandler[T Command] interface {
	// Handle processes the command and returns a result
	Handle(ctx context.Context, command T) (*CommandResult, error)

	// CanHandle returns true if this handler can process the command
	CanHandle(command Command) bool
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success       bool                   `json:"success"`
	Message       string                 `json:"message"`
	Data          interface{}            `json:"data,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewSuccessResult creates a successful command result
func NewSuccessResult(message string, data interface{}) *CommandResult {
	return &CommandResult{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResult creates an error command result
func NewErrorResult(message string, errors []string) *CommandResult {
	return &CommandResult{
		Success: false,
		Message: message,
		Errors:  errors,
	}
}

// AddWarning adds a warning to the command result
func (r *CommandResult) AddWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// AddError adds an error to the command result
func (r *CommandResult) AddError(error string) {
	r.Errors = append(r.Errors, error)
	r.Success = false
}

// SetMetadata adds metadata to the command result
func (r *CommandResult) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
}

// BaseCommand provides common functionality for all commands
type BaseCommand struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UserID    string    `json:"user_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
}

// NewBaseCommand creates a new base command
func NewBaseCommand(commandType string) BaseCommand {
	return BaseCommand{
		ID:        generateCommandID(),
		Type:      commandType,
		CreatedAt: time.Now(),
	}
}

// GetID returns the command ID
func (c BaseCommand) GetID() string {
	return c.ID
}

// GetType returns the command type
func (c BaseCommand) GetType() string {
	return c.Type
}

// Validate provides default validation (can be overridden)
func (c BaseCommand) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("command ID is required")
	}
	if c.Type == "" {
		return fmt.Errorf("command type is required")
	}
	return nil
}

// CommandBus manages command routing and execution
type CommandBus interface {
	// RegisterHandler registers a command handler
	RegisterHandler(handler interface{}) error

	// Execute executes a command
	Execute(ctx context.Context, command Command) (*CommandResult, error)

	// GetHandlers returns all registered handlers
	GetHandlers() []interface{}
}

// CommandMiddleware defines middleware for command processing
type CommandMiddleware interface {
	// Process processes the command before/after execution
	Process(ctx context.Context, command Command, next func(ctx context.Context, command Command) (*CommandResult, error)) (*CommandResult, error)
}

// CommandValidator provides validation for commands
type CommandValidator interface {
	// Validate validates a command
	Validate(command Command) error
}

// CommandAuditor provides auditing for commands
type CommandAuditor interface {
	// AuditCommand logs command execution
	AuditCommand(ctx context.Context, command Command, result *CommandResult) error
}

// CommandMetrics provides metrics for command execution
type CommandMetrics interface {
	// RecordCommandExecution records command execution metrics
	RecordCommandExecution(commandType string, duration time.Duration, success bool)

	// GetCommandStats returns command execution statistics
	GetCommandStats() map[string]CommandStats
}

// CommandStats contains statistics for command execution
type CommandStats struct {
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecutionTime    time.Time     `json:"last_execution_time"`
}

// CommandError represents a command-specific error
type CommandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e CommandError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common command error codes
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeTimeout      = "TIMEOUT"
	ErrCodeRateLimit    = "RATE_LIMIT"
)

// NewCommandError creates a new command error
func NewCommandError(code, message string) CommandError {
	return CommandError{
		Code:    code,
		Message: message,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) CommandError {
	return CommandError{
		Code:    ErrCodeValidation,
		Message: message,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) CommandError {
	return CommandError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// NewInternalError creates an internal error
func NewInternalError(message string) CommandError {
	return CommandError{
		Code:    ErrCodeInternal,
		Message: message,
	}
}

// generateCommandID generates a unique command ID
func generateCommandID() string {
	// Simple implementation - in production, use UUID or similar
	return fmt.Sprintf("cmd_%d", time.Now().UnixNano())
}

// CommandContext provides context for command execution
type CommandContext struct {
	Context   context.Context
	Command   Command
	StartTime time.Time
	Metadata  map[string]interface{}
}

// NewCommandContext creates a new command context
func NewCommandContext(ctx context.Context, command Command) *CommandContext {
	return &CommandContext{
		Context:   ctx,
		Command:   command,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// GetExecutionTime returns the execution time
func (c *CommandContext) GetExecutionTime() time.Duration {
	return time.Since(c.StartTime)
}

// SetMetadata sets metadata in the context
func (c *CommandContext) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}

// GetMetadata gets metadata from the context
func (c *CommandContext) GetMetadata(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}
