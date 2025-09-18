// Package executor provides a secure CLI command execution framework for the MCP Portal.
// It implements comprehensive security measures including command whitelisting, parameter
// validation, rate limiting, audit logging, and timeout management to prevent command
// injection attacks and ensure safe CLI execution.
package executor

import (
	"context"
	"time"
)

// CommandType represents the type of MCP CLI command being executed
type CommandType string

const (
	// Server management commands
	CommandTypeServerList    CommandType = "server.list"
	CommandTypeServerEnable  CommandType = "server.enable"
	CommandTypeServerDisable CommandType = "server.disable"
	CommandTypeServerInspect CommandType = "server.inspect"
	CommandTypeServerStatus  CommandType = "server.status"

	// Gateway management commands
	CommandTypeGatewayRun    CommandType = "gateway.run"
	CommandTypeGatewayStop   CommandType = "gateway.stop"
	CommandTypeGatewayStatus CommandType = "gateway.status"
	CommandTypeGatewayLogs   CommandType = "gateway.logs"

	// Catalog management commands
	CommandTypeCatalogInit CommandType = "catalog.init"
	CommandTypeCatalogList CommandType = "catalog.list"
	CommandTypeCatalogShow CommandType = "catalog.show"
	CommandTypeCatalogSync CommandType = "catalog.sync"

	// Configuration commands
	CommandTypeConfigRead  CommandType = "config.read"
	CommandTypeConfigWrite CommandType = "config.write"

	// Secret management commands
	CommandTypeSecretSet  CommandType = "secret.set"
	CommandTypeSecretGet  CommandType = "secret.get"
	CommandTypeSecretList CommandType = "secret.list"
	CommandTypeSecretDel  CommandType = "secret.delete"

	// System commands
	CommandTypeVersion CommandType = "version"
	CommandTypeHealth  CommandType = "health"
)

// UserRole represents the role of the user making the request
type UserRole string

const (
	RoleStandardUser UserRole = "standard"
	RoleTeamAdmin    UserRole = "team_admin"
	RoleSuperAdmin   UserRole = "super_admin"
)

// ExecutionRequest represents a request to execute a CLI command
type ExecutionRequest struct {
	// Command identification
	Command CommandType `json:"command"`
	Args    []string    `json:"args"`

	// User context
	UserID   string   `json:"user_id"`
	UserRole UserRole `json:"user_role"`
	TenantID string   `json:"tenant_id,omitempty"`

	// Execution context
	RequestID   string            `json:"request_id"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`

	// Options
	StreamOutput bool `json:"stream_output,omitempty"`
	JSONOutput   bool `json:"json_output,omitempty"`
}

// ExecutionResult represents the result of a CLI command execution
type ExecutionResult struct {
	// Execution metadata
	RequestID string        `json:"request_id"`
	Command   CommandType   `json:"command"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`

	// Output
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`

	// Error information
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"error_type,omitempty"`

	// Resource usage
	CPUTime     time.Duration `json:"cpu_time,omitempty"`
	MemoryUsage int64         `json:"memory_usage,omitempty"`
}

// ValidationError represents a command validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// RateLimitError represents a rate limiting error
type RateLimitError struct {
	UserID        string        `json:"user_id"`
	Command       CommandType   `json:"command"`
	Limit         int           `json:"limit"`
	Window        time.Duration `json:"window"`
	ResetTime     time.Time     `json:"reset_time"`
	RemainingTime time.Duration `json:"remaining_time"`
}

func (e RateLimitError) Error() string {
	return "rate limit exceeded"
}

// TimeoutError represents a command timeout error
type TimeoutError struct {
	RequestID string        `json:"request_id"`
	Command   CommandType   `json:"command"`
	Timeout   time.Duration `json:"timeout"`
	Duration  time.Duration `json:"duration"`
}

func (e TimeoutError) Error() string {
	return "command execution timeout"
}

// SecurityEvent represents a security-related event for audit logging
type SecurityEvent struct {
	// Event metadata
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`

	// User context
	UserID   string   `json:"user_id"`
	UserRole UserRole `json:"user_role"`
	TenantID string   `json:"tenant_id,omitempty"`

	// Request context
	RequestID  string      `json:"request_id"`
	Command    CommandType `json:"command"`
	Args       []string    `json:"args"`
	RemoteAddr string      `json:"remote_addr,omitempty"`
	UserAgent  string      `json:"user_agent,omitempty"`

	// Event details
	Message     string            `json:"message"`
	Details     map[string]string `json:"details,omitempty"`
	Success     bool              `json:"success"`
	ErrorReason string            `json:"error_reason,omitempty"`

	// Resource impact
	Duration    time.Duration `json:"duration,omitempty"`
	CPUTime     time.Duration `json:"cpu_time,omitempty"`
	MemoryUsage int64         `json:"memory_usage,omitempty"`
}

// CommandWhitelist defines allowed commands and their parameter constraints
type CommandWhitelist struct {
	// Command configuration
	Command     CommandType   `json:"command"`
	MinRole     UserRole      `json:"min_role"`
	MaxTimeout  time.Duration `json:"max_timeout"`
	Description string        `json:"description"`

	// Rate limiting
	RateLimit RateLimitConfig `json:"rate_limit"`

	// Parameter validation
	AllowedArgs    []string          `json:"allowed_args,omitempty"`
	RequiredArgs   []string          `json:"required_args,omitempty"`
	ForbiddenArgs  []string          `json:"forbidden_args,omitempty"`
	ArgValidation  map[string]string `json:"arg_validation,omitempty"`  // regex patterns
	ArgConstraints map[string]int    `json:"arg_constraints,omitempty"` // max lengths
}

// RateLimitConfig defines rate limiting configuration for a command
type RateLimitConfig struct {
	// Per-user limits
	UserRequests int           `json:"user_requests"`
	UserWindow   time.Duration `json:"user_window"`

	// Per-command limits
	CommandRequests int           `json:"command_requests"`
	CommandWindow   time.Duration `json:"command_window"`

	// Global limits
	GlobalRequests int           `json:"global_requests"`
	GlobalWindow   time.Duration `json:"global_window"`

	// Burst allowance
	BurstSize int `json:"burst_size,omitempty"`
}

// Executor defines the interface for secure CLI command execution
type Executor interface {
	// Execute runs a CLI command with full security validation
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error)

	// ExecuteStream runs a CLI command with streaming output
	ExecuteStream(
		ctx context.Context,
		req *ExecutionRequest,
		outputChan chan<- string,
	) (*ExecutionResult, error)

	// ValidateCommand validates a command request without executing it
	ValidateCommand(req *ExecutionRequest) []ValidationError

	// GetWhitelist returns the current command whitelist
	GetWhitelist() []CommandWhitelist

	// GetRateLimit returns current rate limit status for a user/command
	GetRateLimit(userID string, command CommandType) (remaining int, resetTime time.Time, err error)

	// Health returns executor health status
	Health(ctx context.Context) error
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	// LogSecurityEvent logs a security-related event
	LogSecurityEvent(ctx context.Context, event *SecurityEvent) error

	// LogExecution logs a command execution
	LogExecution(ctx context.Context, req *ExecutionRequest, result *ExecutionResult) error

	// LogValidationFailure logs command validation failures
	LogValidationFailure(ctx context.Context, req *ExecutionRequest, errors []ValidationError) error

	// LogRateLimitExceeded logs rate limit violations
	LogRateLimitExceeded(ctx context.Context, req *ExecutionRequest, err *RateLimitError) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow checks if a request is allowed under current rate limits
	Allow(userID string, command CommandType) error

	// GetLimit returns current limit status
	GetLimit(userID string, command CommandType) (remaining int, resetTime time.Time, err error)

	// Reset resets rate limits for a user (admin function)
	Reset(userID string, command CommandType) error
}

// Validator defines the interface for command validation
type Validator interface {
	// ValidateCommand validates a command request
	ValidateCommand(req *ExecutionRequest) []ValidationError

	// ValidateArgs validates command arguments
	ValidateArgs(command CommandType, args []string) []ValidationError

	// ValidateUser validates user permissions for a command
	ValidateUser(userID string, role UserRole, command CommandType) error

	// SanitizeArgs sanitizes command arguments
	SanitizeArgs(args []string) []string
}

// MockExecutor defines the interface for mocking CLI execution in tests
type MockExecutor interface {
	Executor

	// SetMockResult sets the result for a specific command
	SetMockResult(command CommandType, result *ExecutionResult)

	// SetMockError sets an error for a specific command
	SetMockError(command CommandType, err error)

	// GetExecutionHistory returns the history of executed commands
	GetExecutionHistory() []*ExecutionRequest

	// Reset clears all mock state
	Reset()
}

// ProcessManager defines the interface for managing OS processes
type ProcessManager interface {
	// StartProcess starts a new process with the given command and args
	StartProcess(
		ctx context.Context,
		command string,
		args []string,
		env map[string]string,
	) (Process, error)
}

// Process defines the interface for a running process
type Process interface {
	// Wait waits for the process to complete
	Wait() error

	// Kill terminates the process
	Kill() error

	// PID returns the process ID
	PID() int

	// ExitCode returns the exit code (only valid after Wait)
	ExitCode() int

	// Stdout returns the stdout reader
	Stdout() <-chan string

	// Stderr returns the stderr reader
	Stderr() <-chan string
}
