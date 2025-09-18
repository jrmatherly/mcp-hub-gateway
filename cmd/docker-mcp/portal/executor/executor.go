package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SecureCLIExecutor provides secure execution of CLI commands
type SecureCLIExecutor struct {
	whitelist      []CommandWhitelist
	auditLogger    AuditLogger
	rateLimiter    RateLimiter
	validator      Validator
	processManager ProcessManager
	maxTimeout     time.Duration
	mu             sync.RWMutex
}

// NewSecureCLIExecutor creates a new secure CLI executor
func NewSecureCLIExecutor(
	auditLogger AuditLogger,
	rateLimiter RateLimiter,
	validator Validator,
	processManager ProcessManager,
) *SecureCLIExecutor {
	return &SecureCLIExecutor{
		whitelist:      initWhitelist(),
		auditLogger:    auditLogger,
		rateLimiter:    rateLimiter,
		validator:      validator,
		processManager: processManager,
		maxTimeout:     5 * time.Minute,
	}
}

// Execute runs a CLI command with full security validation
func (e *SecureCLIExecutor) Execute(
	ctx context.Context,
	req *ExecutionRequest,
) (*ExecutionResult, error) {
	// 1. Validate command
	if errs := e.ValidateCommand(req); len(errs) > 0 {
		if e.auditLogger != nil {
			_ = e.auditLogger.LogValidationFailure(ctx, req, errs)
		}
		return nil, errs[0]
	}

	// 2. Check rate limits
	if err := e.rateLimiter.Allow(req.UserID, req.Command); err != nil {
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			if e.auditLogger != nil {
				_ = e.auditLogger.LogRateLimitExceeded(ctx, req, rateLimitErr)
			}
		}
		return nil, err
	}

	// 3. Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if timeout > e.maxTimeout {
		timeout = e.maxTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 4. Build command
	cmd := e.buildCommand(req)

	// 5. Execute command
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// 6. Build result
	result := &ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  duration,
		Success:   err == nil,
		ExitCode:  0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Stderr = string(exitErr.Stderr)
		}
		result.Error = err.Error()
		result.ErrorType = fmt.Sprintf("%T", err)
	} else {
		result.Stdout = string(output)
	}

	// 7. Log execution
	if e.auditLogger != nil {
		_ = e.auditLogger.LogExecution(ctx, req, result)
	}

	return result, nil
}

// ExecuteStream runs a CLI command with streaming output
func (e *SecureCLIExecutor) ExecuteStream(
	ctx context.Context,
	req *ExecutionRequest,
	outputChan chan<- string,
) (*ExecutionResult, error) {
	// Similar to Execute but with streaming support
	// Implementation would involve reading from cmd.StdoutPipe() and cmd.StderrPipe()
	// and sending chunks to outputChan
	return e.Execute(ctx, req)
}

// ValidateCommand validates a command request without executing it
func (e *SecureCLIExecutor) ValidateCommand(req *ExecutionRequest) []ValidationError {
	var errors []ValidationError

	// 1. Check if command is whitelisted
	whitelist := e.getCommandWhitelist(req.Command)
	if whitelist == nil {
		errors = append(errors, ValidationError{
			Field:   "command",
			Value:   string(req.Command),
			Message: "command not allowed",
			Code:    "COMMAND_NOT_WHITELISTED",
		})
		return errors
	}

	// 2. Check user role
	if !e.hasRequiredRole(req.UserRole, whitelist.MinRole) {
		errors = append(errors, ValidationError{
			Field:   "user_role",
			Value:   string(req.UserRole),
			Message: fmt.Sprintf("insufficient privileges, requires %s role", whitelist.MinRole),
			Code:    "INSUFFICIENT_PRIVILEGES",
		})
	}

	// 3. Validate arguments
	for _, arg := range req.Args {
		if containsDangerousPattern(arg) {
			errors = append(errors, ValidationError{
				Field:   "args",
				Value:   arg,
				Message: "argument contains dangerous pattern",
				Code:    "DANGEROUS_PATTERN",
			})
		}
	}

	// 4. Check required arguments
	for _, required := range whitelist.RequiredArgs {
		if !contains(req.Args, required) {
			errors = append(errors, ValidationError{
				Field:   "args",
				Value:   required,
				Message: fmt.Sprintf("required argument missing: %s", required),
				Code:    "REQUIRED_ARG_MISSING",
			})
		}
	}

	// 5. Check forbidden arguments
	for _, forbidden := range whitelist.ForbiddenArgs {
		if contains(req.Args, forbidden) {
			errors = append(errors, ValidationError{
				Field:   "args",
				Value:   forbidden,
				Message: fmt.Sprintf("forbidden argument used: %s", forbidden),
				Code:    "FORBIDDEN_ARG_USED",
			})
		}
	}

	return errors
}

// GetWhitelist returns the current command whitelist
func (e *SecureCLIExecutor) GetWhitelist() []CommandWhitelist {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.whitelist
}

// GetRateLimit returns current rate limit status for a user/command
func (e *SecureCLIExecutor) GetRateLimit(
	userID string,
	command CommandType,
) (remaining int, resetTime time.Time, err error) {
	if e.rateLimiter != nil {
		return e.rateLimiter.GetLimit(userID, command)
	}
	return 100, time.Now().Add(time.Hour), nil
}

// Health returns executor health status
func (e *SecureCLIExecutor) Health(ctx context.Context) error {
	// Check if docker CLI is available
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker CLI not available: %w", err)
	}
	return nil
}

// buildCommand builds the exec.Cmd for the request
func (e *SecureCLIExecutor) buildCommand(req *ExecutionRequest) *exec.Cmd {
	// Map command type to actual CLI command
	var args []string

	switch req.Command {
	case CommandTypeServerList:
		args = append([]string{"mcp", "server", "list", "--json"}, req.Args...)
	case CommandTypeServerEnable:
		args = append([]string{"mcp", "server", "enable"}, req.Args...)
	case CommandTypeServerDisable:
		args = append([]string{"mcp", "server", "disable"}, req.Args...)
	case CommandTypeServerInspect:
		args = append([]string{"mcp", "server", "inspect", "--json"}, req.Args...)
	case CommandTypeGatewayRun:
		args = append([]string{"mcp", "gateway", "run"}, req.Args...)
	case CommandTypeConfigRead:
		args = append([]string{"mcp", "config", "read", "--json"}, req.Args...)
	case CommandTypeConfigWrite:
		args = append([]string{"mcp", "config", "write"}, req.Args...)
	default:
		args = append([]string{"mcp"}, req.Args...)
	}

	cmd := exec.Command("docker", args...)

	// Set environment variables
	for key, value := range req.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	return cmd
}

// getCommandWhitelist returns the whitelist entry for a command
func (e *SecureCLIExecutor) getCommandWhitelist(command CommandType) *CommandWhitelist {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, w := range e.whitelist {
		if w.Command == command {
			return &w
		}
	}
	return nil
}

// hasRequiredRole checks if the user has the required role
func (e *SecureCLIExecutor) hasRequiredRole(userRole, minRole UserRole) bool {
	roleHierarchy := map[UserRole]int{
		RoleStandardUser: 1,
		RoleTeamAdmin:    2,
		RoleSuperAdmin:   3,
	}

	userLevel, ok1 := roleHierarchy[userRole]
	minLevel, ok2 := roleHierarchy[minRole]

	if !ok1 || !ok2 {
		return false
	}

	return userLevel >= minLevel
}

// initWhitelist initializes the command whitelist
func initWhitelist() []CommandWhitelist {
	return []CommandWhitelist{
		{
			Command:     CommandTypeServerList,
			MinRole:     RoleStandardUser,
			MaxTimeout:  30 * time.Second,
			Description: "List available MCP servers",
			RateLimit: RateLimitConfig{
				UserRequests: 60,
				UserWindow:   time.Minute,
			},
		},
		{
			Command:      CommandTypeServerEnable,
			MinRole:      RoleTeamAdmin,
			MaxTimeout:   60 * time.Second,
			Description:  "Enable an MCP server",
			RequiredArgs: []string{},
			RateLimit: RateLimitConfig{
				UserRequests: 10,
				UserWindow:   time.Minute,
			},
		},
		{
			Command:      CommandTypeServerDisable,
			MinRole:      RoleTeamAdmin,
			MaxTimeout:   60 * time.Second,
			Description:  "Disable an MCP server",
			RequiredArgs: []string{},
			RateLimit: RateLimitConfig{
				UserRequests: 10,
				UserWindow:   time.Minute,
			},
		},
		{
			Command:      CommandTypeServerInspect,
			MinRole:      RoleStandardUser,
			MaxTimeout:   30 * time.Second,
			Description:  "Inspect MCP server details",
			RequiredArgs: []string{},
			RateLimit: RateLimitConfig{
				UserRequests: 30,
				UserWindow:   time.Minute,
			},
		},
		{
			Command:     CommandTypeConfigRead,
			MinRole:     RoleStandardUser,
			MaxTimeout:  30 * time.Second,
			Description: "Read configuration",
			RateLimit: RateLimitConfig{
				UserRequests: 30,
				UserWindow:   time.Minute,
			},
		},
		{
			Command:       CommandTypeConfigWrite,
			MinRole:       RoleTeamAdmin,
			MaxTimeout:    60 * time.Second,
			Description:   "Write configuration",
			ForbiddenArgs: []string{"--force", "--bypass-validation"},
			RateLimit: RateLimitConfig{
				UserRequests: 5,
				UserWindow:   time.Minute,
			},
		},
	}
}

// Helper functions

func containsDangerousPattern(s string) bool {
	dangerousPatterns := []string{
		"..",           // Path traversal
		"~/",           // Home directory access
		"/etc/",        // System config access
		"/proc/",       // Process info access
		"--privileged", // Docker privileged mode
		"--cap-add",    // Docker capabilities
		"--security",   // Security options
		"${",           // Variable expansion
		"$(",           // Command substitution
		"`",            // Backtick substitution
		";",            // Command separator
		"&&",           // Command chaining
		"||",           // Command chaining
		"|",            // Pipe
		">",            // Redirect
		"<",            // Redirect
		"\n",           // Newline
		"\r",           // Carriage return
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Ensure interface compliance
var _ Executor = (*SecureCLIExecutor)(nil)
