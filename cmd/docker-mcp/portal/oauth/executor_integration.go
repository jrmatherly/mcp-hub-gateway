package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
)

// OAuth-enabled command types for MCP servers requiring authentication
const (
	// OAuth server management commands
	CommandTypeOAuthAuthorize executor.CommandType = "oauth.authorize"
	CommandTypeOAuthToken     executor.CommandType = "oauth.token"
	CommandTypeOAuthRefresh   executor.CommandType = "oauth.refresh"
	CommandTypeOAuthRevoke    executor.CommandType = "oauth.revoke"
	CommandTypeOAuthStatus    executor.CommandType = "oauth.status"

	// OAuth server configuration commands
	CommandTypeOAuthRegister executor.CommandType = "oauth.register"
	CommandTypeOAuthUpdate   executor.CommandType = "oauth.update"
	CommandTypeOAuthRemove   executor.CommandType = "oauth.remove"
	CommandTypeOAuthList     executor.CommandType = "oauth.list"

	// OAuth DCR commands
	CommandTypeOAuthDCRRegister executor.CommandType = "oauth.dcr.register"
	CommandTypeOAuthDCRUpdate   executor.CommandType = "oauth.dcr.update"
	CommandTypeOAuthDCRDelete   executor.CommandType = "oauth.dcr.delete"

	// OAuth-protected MCP server commands
	CommandTypeServerOAuthConnect executor.CommandType = "server.oauth.connect"
	CommandTypeServerOAuthRequest executor.CommandType = "server.oauth.request"
	CommandTypeServerOAuthStatus  executor.CommandType = "server.oauth.status"
)

// OAuthExecutorWrapper wraps the existing CLI executor with OAuth capabilities
type OAuthExecutorWrapper struct {
	baseExecutor     executor.Executor
	oauthInterceptor OAuthInterceptor
	httpClient       HTTPClient
	auditLogger      executor.AuditLogger
}

// CreateOAuthExecutorWrapper creates a new OAuth-enabled executor wrapper
func CreateOAuthExecutorWrapper(
	baseExecutor executor.Executor,
	oauthInterceptor OAuthInterceptor,
	httpClient HTTPClient,
	auditLogger executor.AuditLogger,
) *OAuthExecutorWrapper {
	return &OAuthExecutorWrapper{
		baseExecutor:     baseExecutor,
		oauthInterceptor: oauthInterceptor,
		httpClient:       httpClient,
		auditLogger:      auditLogger,
	}
}

// Execute runs a CLI command with OAuth support
func (w *OAuthExecutorWrapper) Execute(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	// Check if this is an OAuth command
	if w.isOAuthCommand(req.Command) {
		return w.executeOAuthCommand(ctx, req)
	}

	// Check if this command requires OAuth for an MCP server
	if w.requiresOAuth(req.Command) {
		return w.executeWithOAuthInterception(ctx, req)
	}

	// For non-OAuth commands, delegate to base executor
	return w.baseExecutor.Execute(ctx, req)
}

// ExecuteStream runs a CLI command with streaming output and OAuth support
func (w *OAuthExecutorWrapper) ExecuteStream(
	ctx context.Context,
	req *executor.ExecutionRequest,
	outputChan chan<- string,
) (*executor.ExecutionResult, error) {
	// OAuth commands typically don't support streaming
	if w.isOAuthCommand(req.Command) {
		result, err := w.executeOAuthCommand(ctx, req)
		if err == nil && result.Stdout != "" {
			outputChan <- result.Stdout
		}
		return result, err
	}

	// For commands requiring OAuth, we need to handle streaming differently
	if w.requiresOAuth(req.Command) {
		// For now, execute normally and stream the final output
		result, err := w.executeWithOAuthInterception(ctx, req)
		if err == nil && result.Stdout != "" {
			outputChan <- result.Stdout
		}
		return result, err
	}

	// Delegate to base executor
	return w.baseExecutor.ExecuteStream(ctx, req, outputChan)
}

// ValidateCommand validates a command request including OAuth-specific validation
func (w *OAuthExecutorWrapper) ValidateCommand(
	req *executor.ExecutionRequest,
) []executor.ValidationError {
	// First validate with base executor
	errors := w.baseExecutor.ValidateCommand(req)

	// Add OAuth-specific validation
	if w.isOAuthCommand(req.Command) {
		oauthErrors := w.validateOAuthCommand(req)
		errors = append(errors, oauthErrors...)
	}

	return errors
}

// GetWhitelist returns the command whitelist including OAuth commands
func (w *OAuthExecutorWrapper) GetWhitelist() []executor.CommandWhitelist {
	baseWhitelist := w.baseExecutor.GetWhitelist()
	oauthWhitelist := w.getOAuthWhitelist()
	return append(baseWhitelist, oauthWhitelist...)
}

// GetRateLimit returns current rate limit status
func (w *OAuthExecutorWrapper) GetRateLimit(
	userID string,
	command executor.CommandType,
) (remaining int, resetTime time.Time, err error) {
	return w.baseExecutor.GetRateLimit(userID, command)
}

// Health returns executor health status including OAuth components
func (w *OAuthExecutorWrapper) Health(ctx context.Context) error {
	// Check base executor health
	if err := w.baseExecutor.Health(ctx); err != nil {
		return fmt.Errorf("base executor unhealthy: %w", err)
	}

	// Check OAuth interceptor health
	if err := w.oauthInterceptor.Health(ctx); err != nil {
		return fmt.Errorf("OAuth interceptor unhealthy: %w", err)
	}

	return nil
}

// Private helper methods

func (w *OAuthExecutorWrapper) isOAuthCommand(command executor.CommandType) bool {
	oauthCommands := map[executor.CommandType]bool{
		CommandTypeOAuthAuthorize:     true,
		CommandTypeOAuthToken:         true,
		CommandTypeOAuthRefresh:       true,
		CommandTypeOAuthRevoke:        true,
		CommandTypeOAuthStatus:        true,
		CommandTypeOAuthRegister:      true,
		CommandTypeOAuthUpdate:        true,
		CommandTypeOAuthRemove:        true,
		CommandTypeOAuthList:          true,
		CommandTypeOAuthDCRRegister:   true,
		CommandTypeOAuthDCRUpdate:     true,
		CommandTypeOAuthDCRDelete:     true,
		CommandTypeServerOAuthConnect: true,
		CommandTypeServerOAuthRequest: true,
		CommandTypeServerOAuthStatus:  true,
	}

	return oauthCommands[command]
}

func (w *OAuthExecutorWrapper) requiresOAuth(command executor.CommandType) bool {
	// Commands that might need OAuth interception for MCP server communication
	oauthRequiredCommands := map[executor.CommandType]bool{
		executor.CommandTypeServerInspect: true,
		executor.CommandTypeServerStatus:  true,
		// Add other commands that communicate with MCP servers
	}

	return oauthRequiredCommands[command]
}

func (w *OAuthExecutorWrapper) executeOAuthCommand(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	startTime := time.Now()

	var result *executor.ExecutionResult
	var err error

	switch req.Command {
	case CommandTypeOAuthAuthorize:
		result, err = w.handleOAuthAuthorize(ctx, req)
	case CommandTypeOAuthToken:
		result, err = w.handleOAuthToken(ctx, req)
	case CommandTypeOAuthRefresh:
		result, err = w.handleOAuthRefresh(ctx, req)
	case CommandTypeOAuthRevoke:
		result, err = w.handleOAuthRevoke(ctx, req)
	case CommandTypeOAuthStatus:
		result, err = w.handleOAuthStatus(ctx, req)
	case CommandTypeOAuthRegister:
		result, err = w.handleOAuthRegister(ctx, req)
	case CommandTypeOAuthUpdate:
		result, err = w.handleOAuthUpdate(ctx, req)
	case CommandTypeOAuthRemove:
		result, err = w.handleOAuthRemove(ctx, req)
	case CommandTypeOAuthList:
		result, err = w.handleOAuthList(ctx, req)
	case CommandTypeServerOAuthConnect:
		result, err = w.handleServerOAuthConnect(ctx, req)
	case CommandTypeServerOAuthRequest:
		result, err = w.handleServerOAuthRequest(ctx, req)
	case CommandTypeServerOAuthStatus:
		result, err = w.handleServerOAuthStatus(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported OAuth command: %s", req.Command)
	}

	if result == nil {
		result = &executor.ExecutionResult{
			RequestID: req.RequestID,
			Command:   req.Command,
			StartTime: startTime,
			EndTime:   time.Now(),
			Success:   false,
		}
	}

	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, err
}

func (w *OAuthExecutorWrapper) executeWithOAuthInterception(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	// For commands that communicate with MCP servers, we need to intercept
	// HTTP requests and add OAuth tokens. This is a simplified implementation.

	// First, execute the command normally
	result, err := w.baseExecutor.Execute(ctx, req)
	if err != nil {
		return result, err
	}

	// If the command failed due to authentication issues, try with OAuth
	if result.ExitCode == 401 || strings.Contains(result.Stderr, "unauthorized") {
		// This would require more sophisticated integration with the CLI
		// For now, we'll just log the event
		if w.auditLogger != nil {
			event := &executor.SecurityEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
				EventType: "oauth_interception_needed",
				UserID:    req.UserID,
				UserRole:  req.UserRole,
				RequestID: req.RequestID,
				Command:   req.Command,
				Args:      req.Args,
				Message:   "Command failed with authentication error, OAuth interception may be needed",
				Success:   false,
			}
			_ = w.auditLogger.LogSecurityEvent(ctx, event)
		}
	}

	return result, err
}

func (w *OAuthExecutorWrapper) validateOAuthCommand(
	req *executor.ExecutionRequest,
) []executor.ValidationError {
	var errors []executor.ValidationError

	switch req.Command {
	case CommandTypeOAuthAuthorize:
		if len(req.Args) < 2 {
			errors = append(errors, executor.ValidationError{
				Field:   "args",
				Message: "OAuth authorize requires server-name and provider arguments",
				Code:    "MISSING_REQUIRED_ARGS",
			})
		}
	case CommandTypeOAuthToken:
		if len(req.Args) < 1 {
			errors = append(errors, executor.ValidationError{
				Field:   "args",
				Message: "OAuth token requires server-name argument",
				Code:    "MISSING_REQUIRED_ARGS",
			})
		}
	case CommandTypeOAuthRegister:
		if len(req.Args) < 3 {
			errors = append(errors, executor.ValidationError{
				Field:   "args",
				Message: "OAuth register requires server-name, provider, and client-id arguments",
				Code:    "MISSING_REQUIRED_ARGS",
			})
		}
	case CommandTypeServerOAuthRequest:
		if len(req.Args) < 3 {
			errors = append(errors, executor.ValidationError{
				Field:   "args",
				Message: "OAuth request requires server-name, method, and URL arguments",
				Code:    "MISSING_REQUIRED_ARGS",
			})
		}
	}

	return errors
}

func (w *OAuthExecutorWrapper) getOAuthWhitelist() []executor.CommandWhitelist {
	return []executor.CommandWhitelist{
		{
			Command:     CommandTypeOAuthAuthorize,
			MinRole:     executor.RoleStandardUser,
			MaxTimeout:  2 * time.Minute,
			Description: "Initiate OAuth authorization flow",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 10,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name", "provider"},
		},
		{
			Command:     CommandTypeOAuthToken,
			MinRole:     executor.RoleStandardUser,
			MaxTimeout:  30 * time.Second,
			Description: "Get OAuth token status",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 100,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name"},
		},
		{
			Command:     CommandTypeOAuthRefresh,
			MinRole:     executor.RoleStandardUser,
			MaxTimeout:  30 * time.Second,
			Description: "Refresh OAuth token",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 50,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name"},
		},
		{
			Command:     CommandTypeOAuthRevoke,
			MinRole:     executor.RoleStandardUser,
			MaxTimeout:  30 * time.Second,
			Description: "Revoke OAuth token",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 20,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name"},
		},
		{
			Command:     CommandTypeOAuthRegister,
			MinRole:     executor.RoleTeamAdmin,
			MaxTimeout:  time.Minute,
			Description: "Register OAuth server configuration",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 10,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name", "provider", "client-id"},
		},
		{
			Command:     CommandTypeServerOAuthRequest,
			MinRole:     executor.RoleStandardUser,
			MaxTimeout:  2 * time.Minute,
			Description: "Make OAuth-authenticated request to MCP server",
			RateLimit: executor.RateLimitConfig{
				UserRequests: 200,
				UserWindow:   time.Hour,
			},
			RequiredArgs: []string{"server-name", "method", "url"},
		},
	}
}

// OAuth command handlers

func (w *OAuthExecutorWrapper) handleOAuthAuthorize(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 2 {
		return w.createErrorResult(req, "Missing required arguments: server-name, provider"), nil
	}

	serverName := req.Args[0]
	providerType := ProviderType(req.Args[1])

	// Get server configuration
	_, err := w.oauthInterceptor.GetServerConfig(ctx, serverName)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Server configuration not found: %v", err)), nil
	}

	// This would typically start a browser flow or return an authorization URL
	// For CLI usage, we'll return the authorization URL
	authURL := fmt.Sprintf(
		"https://example.com/oauth/authorize?server=%s&provider=%s",
		serverName,
		providerType,
	)

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    fmt.Sprintf("Authorization URL: %s\n", authURL),
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthToken(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 1 {
		return w.createErrorResult(req, "Missing required argument: server-name"), nil
	}

	serverName := req.Args[0]
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Invalid user ID: %v", err)), nil
	}

	token, err := w.oauthInterceptor.GetToken(ctx, serverName, userID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Token not found: %v", err)), nil
	}

	// Return token status (not the actual token for security)
	tokenStatus := map[string]interface{}{
		"server_name": token.ServerName,
		"provider":    token.ProviderType,
		"expires_at":  token.ExpiresAt.Format(time.RFC3339),
		"scopes":      token.Scopes,
		"last_used":   token.LastUsed.Format(time.RFC3339),
		"usage_count": token.UsageCount,
	}

	statusJSON, _ := json.MarshalIndent(tokenStatus, "", "  ")

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    string(statusJSON) + "\n",
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthRefresh(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 1 {
		return w.createErrorResult(req, "Missing required argument: server-name"), nil
	}

	serverName := req.Args[0]
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Invalid user ID: %v", err)), nil
	}

	refreshedToken, err := w.oauthInterceptor.RefreshToken(ctx, serverName, userID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Token refresh failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout: fmt.Sprintf(
			"Token refreshed successfully. New expiry: %s\n",
			refreshedToken.ExpiresAt.Format(time.RFC3339),
		),
		ExitCode: 0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthRevoke(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 1 {
		return w.createErrorResult(req, "Missing required argument: server-name"), nil
	}

	serverName := req.Args[0]
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Invalid user ID: %v", err)), nil
	}

	err = w.oauthInterceptor.RevokeToken(ctx, serverName, userID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Token revocation failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    "Token revoked successfully\n",
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthStatus(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	metrics, err := w.oauthInterceptor.GetMetrics(ctx)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Failed to get OAuth status: %v", err)), nil
	}

	statusJSON, _ := json.MarshalIndent(metrics, "", "  ")

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    string(statusJSON) + "\n",
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthRegister(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 3 {
		return w.createErrorResult(
			req,
			"Missing required arguments: server-name, provider, client-id",
		), nil
	}

	serverName := req.Args[0]
	providerType := ProviderType(req.Args[1])
	clientID := req.Args[2]

	config := &ServerConfig{
		ServerName:   serverName,
		ProviderType: providerType,
		ClientID:     clientID,
		Scopes:       []string{"read", "write"}, // Default scopes
		RedirectURI:  fmt.Sprintf("http://localhost:8080/oauth/callback/%s", serverName),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	// Parse additional arguments
	for i := 3; i < len(req.Args); i += 2 {
		if i+1 >= len(req.Args) {
			break
		}
		key := req.Args[i]
		value := req.Args[i+1]

		switch key {
		case "--client-secret":
			config.ClientSecret = value
		case "--scopes":
			config.Scopes = strings.Split(value, ",")
		case "--redirect-uri":
			config.RedirectURI = value
		case "--tenant-id":
			config.TenantID = value
		}
	}

	err := w.oauthInterceptor.RegisterServer(ctx, config)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Server registration failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    fmt.Sprintf("OAuth server '%s' registered successfully\n", serverName),
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthUpdate(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 1 {
		return w.createErrorResult(req, "Missing required argument: server-name"), nil
	}

	serverName := req.Args[0]

	// Get existing configuration
	config, err := w.oauthInterceptor.GetServerConfig(ctx, serverName)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Server configuration not found: %v", err)), nil
	}

	// Update configuration based on arguments
	for i := 1; i < len(req.Args); i += 2 {
		if i+1 >= len(req.Args) {
			break
		}
		key := req.Args[i]
		value := req.Args[i+1]

		switch key {
		case "--client-secret":
			config.ClientSecret = value
		case "--scopes":
			config.Scopes = strings.Split(value, ",")
		case "--redirect-uri":
			config.RedirectURI = value
		case "--active":
			config.IsActive = value == "true"
		}
	}

	err = w.oauthInterceptor.UpdateServerConfig(ctx, config)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Server update failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    fmt.Sprintf("OAuth server '%s' updated successfully\n", serverName),
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthRemove(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 1 {
		return w.createErrorResult(req, "Missing required argument: server-name"), nil
	}

	serverName := req.Args[0]

	err := w.oauthInterceptor.RemoveServerConfig(ctx, serverName)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Server removal failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    fmt.Sprintf("OAuth server '%s' removed successfully\n", serverName),
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleOAuthList(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return w.createErrorResult(req, "request cancelled"), ctx.Err()
	default:
	}

	// TODO: Create method to list all registered servers
	// This would require a method to list all registered servers
	// For now, return a placeholder
	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   true,
		Stdout:    "OAuth server listing not yet implemented\n",
		ExitCode:  0,
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleServerOAuthConnect(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return w.createErrorResult(req, "request cancelled"), ctx.Err()
	default:
	}

	// TODO: Implementation for connecting to OAuth-protected MCP servers
	return w.createErrorResult(req, "OAuth server connect not yet implemented"), nil
}

func (w *OAuthExecutorWrapper) handleServerOAuthRequest(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	if len(req.Args) < 3 {
		return w.createErrorResult(req, "Missing required arguments: server-name, method, url"), nil
	}

	serverName := req.Args[0]
	method := req.Args[1]
	url := req.Args[2]

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("Invalid user ID: %v", err)), nil
	}

	// Make OAuth-authenticated request
	var response *AuthResponse
	switch strings.ToUpper(method) {
	case "GET":
		response, err = w.httpClient.Get(ctx, serverName, userID, url)
	case "POST":
		var body []byte
		if len(req.Args) > 3 {
			body = []byte(req.Args[3])
		}
		response, err = w.httpClient.Post(ctx, serverName, userID, url, body)
	case "PUT":
		var body []byte
		if len(req.Args) > 3 {
			body = []byte(req.Args[3])
		}
		response, err = w.httpClient.Put(ctx, serverName, userID, url, body)
	case "DELETE":
		response, err = w.httpClient.Delete(ctx, serverName, userID, url)
	default:
		return w.createErrorResult(req, fmt.Sprintf("Unsupported HTTP method: %s", method)), nil
	}

	if err != nil {
		return w.createErrorResult(req, fmt.Sprintf("OAuth request failed: %v", err)), nil
	}

	result := &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   response.StatusCode < 400,
		Stdout:    string(response.Body),
		ExitCode:  response.StatusCode,
	}

	if response.Error != "" {
		result.Stderr = response.Error
	}

	return result, nil
}

func (w *OAuthExecutorWrapper) handleServerOAuthStatus(
	ctx context.Context,
	req *executor.ExecutionRequest,
) (*executor.ExecutionResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return w.createErrorResult(req, "request cancelled"), ctx.Err()
	default:
	}

	// TODO: Implementation for checking OAuth status of MCP servers
	return w.createErrorResult(req, "OAuth server status not yet implemented"), nil
}

func (w *OAuthExecutorWrapper) createErrorResult(
	req *executor.ExecutionRequest,
	errorMsg string,
) *executor.ExecutionResult {
	return &executor.ExecutionResult{
		RequestID: req.RequestID,
		Command:   req.Command,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Success:   false,
		Stderr:    errorMsg + "\n",
		Error:     errorMsg,
		ExitCode:  1,
	}
}
