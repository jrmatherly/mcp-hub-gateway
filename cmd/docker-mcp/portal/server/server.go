// Package server provides the HTTP server implementation for the MCP Portal.
// It implements a production-ready REST API with authentication, rate limiting,
// CORS, and comprehensive middleware for secure CLI command execution.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/auth"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/catalog"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/ratelimit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/server/handlers"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/server/middleware"
)

// AuditLoggerAdapter adapts audit.Logger to executor.AuditLogger interface
type AuditLoggerAdapter struct {
	logger audit.Logger
}

func (a *AuditLoggerAdapter) LogSecurityEvent(
	ctx context.Context,
	event *executor.SecurityEvent,
) error {
	details := map[string]any{
		"event_id":     event.EventID,
		"event_type":   event.EventType,
		"severity":     event.Severity,
		"request_id":   event.RequestID,
		"command":      string(event.Command),
		"args":         event.Args,
		"remote_addr":  event.RemoteAddr,
		"user_agent":   event.UserAgent,
		"message":      event.Message,
		"details":      event.Details,
		"success":      event.Success,
		"error_reason": event.ErrorReason,
		"duration":     event.Duration,
		"cpu_time":     event.CPUTime,
		"memory_usage": event.MemoryUsage,
	}
	a.logger.LogSecurityEvent(
		ctx,
		uuid.MustParse(event.UserID),
		audit.EventType(event.EventType),
		details,
	)
	return nil
}

func (a *AuditLoggerAdapter) LogExecution(
	ctx context.Context,
	req *executor.ExecutionRequest,
	result *executor.ExecutionResult,
) error {
	auditID := a.logger.LogCommand(ctx, uuid.MustParse(req.UserID), string(req.Command), req.Args)
	var err error
	if !result.Success {
		err = fmt.Errorf("%s", result.Error)
	}
	a.logger.LogCommandResult(ctx, auditID, result.Stdout, err, result.Duration)
	return nil
}

func (a *AuditLoggerAdapter) LogValidationFailure(
	ctx context.Context,
	req *executor.ExecutionRequest,
	errors []executor.ValidationError,
) error {
	details := map[string]any{
		"command":    string(req.Command),
		"args":       req.Args,
		"errors":     errors,
		"request_id": req.RequestID,
	}
	a.logger.LogSecurityEvent(
		ctx,
		uuid.MustParse(req.UserID),
		audit.EventTypeSecurityAlert,
		details,
	)
	return nil
}

func (a *AuditLoggerAdapter) LogRateLimitExceeded(
	ctx context.Context,
	req *executor.ExecutionRequest,
	err *executor.RateLimitError,
) error {
	a.logger.LogRateLimitExceeded(ctx, uuid.MustParse(req.UserID), string(req.Command))
	return nil
}

// RateLimiterAdapter adapts ratelimit.RateLimiter to executor.RateLimiter interface
type RateLimiterAdapter struct {
	limiter ratelimit.RateLimiter
}

func (r *RateLimiterAdapter) Allow(userID string, command executor.CommandType) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	ctx := context.Background()
	if !r.limiter.Allow(ctx, uid, string(command)) {
		return &executor.RateLimitError{
			UserID:  userID,
			Command: command,
		}
	}
	return nil
}

func (r *RateLimiterAdapter) GetLimit(
	userID string,
	command executor.CommandType,
) (remaining int, resetTime time.Time, err error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid user ID: %w", err)
	}

	ctx := context.Background()
	status := r.limiter.GetStatus(ctx, uid)
	return int(status.RequestsLimit - status.RequestsUsed), status.WindowResetTime, nil
}

func (r *RateLimiterAdapter) Reset(userID string, command executor.CommandType) error {
	// Not implemented in underlying rate limiter
	return nil
}

// Server represents the MCP Portal HTTP server
type Server struct {
	config          *config.Config
	httpServer      *http.Server
	ginEngine       *gin.Engine
	executor        executor.Executor
	authService     *auth.AzureADService
	sessionMgr      auth.SessionManager
	cache           cache.Cache
	db              *database.Pool
	rateLimiter     ratelimit.RateLimiter
	auditLogger     audit.Logger
	catalogService  catalog.CatalogService
	realtimeManager realtime.ConnectionManager
}

// Config holds server configuration
type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Host:         "localhost",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize cache
	redisCache, err := cache.NewRedisCache(&cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize database pool
	dbPool, err := database.NewPool(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize session manager
	sessionMgr := auth.CreateRedisSessionManager(redisCache, cfg.Security.AccessTokenTTL)

	// Initialize auth service
	authService, err := auth.CreateAzureADService(&cfg.Azure, sessionMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth service: %w", err)
	}

	// Initialize audit logger
	auditLogger := audit.NewLogger(audit.NewMemoryStorage())

	// Initialize rate limiter
	rateLimiter := ratelimit.NewRateLimiter(&ratelimit.Config{
		RequestsPerMinute: 60,
		BurstSize:         10,
	})

	// Create adapter for audit logger to match executor interface
	auditAdapter := &AuditLoggerAdapter{logger: auditLogger}

	// Create adapter for rate limiter to match executor interface
	rateLimiterAdapter := &RateLimiterAdapter{limiter: rateLimiter}

	// Initialize CLI executor with proper dependencies
	cliExecutor := executor.NewSecureCLIExecutor(
		auditAdapter,
		rateLimiterAdapter,
		nil, // validator - will use default
		nil, // process manager - will use default
	)

	// Initialize catalog repository
	catalogRepo := catalog.CreatePostgresRepository(dbPool)

	// Initialize catalog service
	catalogService := catalog.CreateCatalogService(
		catalogRepo,
		cliExecutor,
		auditLogger,
		redisCache,
	)

	// Initialize realtime connection manager
	realtimeConfig := realtime.DefaultConnectionConfig()
	realtimeManager, err := realtime.CreateConnectionManager(auditLogger, realtimeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize realtime manager: %w", err)
	}

	// Create Gin engine
	ginEngine := gin.New()

	// Create server
	server := &Server{
		config:          cfg,
		ginEngine:       ginEngine,
		executor:        cliExecutor,
		authService:     authService,
		sessionMgr:      sessionMgr,
		cache:           redisCache,
		db:              dbPool,
		rateLimiter:     rateLimiter,
		auditLogger:     auditLogger,
		catalogService:  catalogService,
		realtimeManager: realtimeManager,
	}

	// Setup middleware and routes
	server.setupMiddleware()
	server.setupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server.httpServer = &http.Server{
		Addr:         addr,
		Handler:      ginEngine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.ShutdownTimeout,
	}

	return server, nil
}

// setupMiddleware configures all middleware for the Gin engine
func (s *Server) setupMiddleware() {
	// Apply middleware in order
	s.ginEngine.Use(middleware.RequestID())
	s.ginEngine.Use(middleware.Logger())
	s.ginEngine.Use(middleware.Recovery(s.auditLogger))
	s.ginEngine.Use(middleware.SecurityHeaders())
	s.ginEngine.Use(middleware.RateLimit(s.rateLimiter, s.auditLogger))
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health and status endpoints (no auth required)
	s.ginEngine.GET("/health", s.handleHealth)
	s.ginEngine.GET("/status", s.handleStatus)

	// API v1 routes
	v1 := s.ginEngine.Group("/api/v1")

	// Authentication routes (no auth required)
	auth := v1.Group("/auth")
	{
		auth.GET("/login", s.handleLogin)
		auth.GET("/callback", s.handleAuthCallback)
		auth.POST("/logout", s.handleLogout)
		auth.POST("/refresh", s.handleRefreshToken)
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.authService, s.auditLogger))

	// Create catalog handler
	catalogHandler := handlers.CreateCatalogHandler(s.catalogService)

	// Register catalog routes
	handlers.RegisterCatalogRoutes(protected, catalogHandler)

	// Legacy server management endpoints (for backward compatibility)
	servers := protected.Group("/servers")
	{
		servers.GET("", s.handleServers)
		servers.GET("/:name", s.handleServerInspect)
		servers.POST("/:name/:action", s.handleServerAction)
	}

	// Gateway management endpoints
	gateway := protected.Group("/gateway")
	{
		gateway.POST("/start", s.handleGatewayStart)
		gateway.POST("/stop", s.handleGatewayStop)
		gateway.GET("/status", s.handleGatewayStatus)
	}

	// Configuration endpoints
	config := protected.Group("/config")
	{
		config.GET("", s.handleGetConfig)
		config.PUT("", s.handleUpdateConfig)
	}

	// Real-time communication endpoints
	realtime := protected.Group("")
	realtime.Use(
		s.realtimeAuthMiddleware(),
	) // Custom middleware to set user_id for realtime service
	{
		realtime.GET("/ws", s.handleWebSocket)
		realtime.GET("/sse", s.handleSSE)
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting MCP Portal server on %s", s.httpServer.Addr)
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	// Wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Printf("Received shutdown signal: %v", sig)

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown realtime manager first
		if s.realtimeManager != nil {
			s.realtimeManager.Stop()
		}

		// Shutdown server gracefully
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
			return s.httpServer.Close()
		}

		log.Println("Server stopped gracefully")
		return nil
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down server")
		// Shutdown realtime manager first
		if s.realtimeManager != nil {
			s.realtimeManager.Stop()
		}
		return s.httpServer.Shutdown(ctx)
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	// Check dependencies
	health := map[string]string{
		"status": "healthy",
	}

	// Check cache
	if err := s.cache.Health(c.Request.Context()); err != nil {
		health["cache"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["cache"] = "healthy"
	}

	// Check database
	if err := s.db.Health(c.Request.Context()); err != nil {
		health["database"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["database"] = "healthy"
	}

	status := http.StatusOK
	if health["status"] == "degraded" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}

// handleStatus handles status requests
func (s *Server) handleStatus(c *gin.Context) {
	status := map[string]any{
		"service": "mcp-portal",
		"version": "1.0.0",
		"uptime":  time.Since(time.Now().Add(-time.Hour)), // Placeholder
	}

	handlers.SuccessResponse(c, status)
}

// handleLogin handles login initiation
func (s *Server) handleLogin(c *gin.Context) {
	// Generate state for CSRF protection
	state := fmt.Sprintf("state_%d", time.Now().Unix())

	// Get auth URL
	authURL, err := s.authService.GetAuthURL(state, "")
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to generate auth URL")
		return
	}

	handlers.SuccessResponse(c, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// handleAuthCallback handles OAuth callback
func (s *Server) handleAuthCallback(c *gin.Context) {
	// Extract code and state from query parameters
	code := c.Query("code")
	_ = c.Query("state") // TODO: Validate state for CSRF protection

	if code == "" {
		handlers.ErrorResponse(
			c,
			http.StatusBadRequest,
			"missing_code",
			"Missing authorization code",
		)
		return
	}

	// Exchange code for tokens
	tokens, err := s.authService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		handlers.ErrorResponse(
			c,
			http.StatusUnauthorized,
			"exchange_failed",
			"Failed to exchange code",
		)
		return
	}

	// Get user info
	user, err := s.authService.GetUserInfo(c.Request.Context(), tokens.AccessToken)
	if err != nil {
		handlers.ErrorResponse(
			c,
			http.StatusUnauthorized,
			"user_info_failed",
			"Failed to get user info",
		)
		return
	}

	// Create session
	session, err := s.sessionMgr.CreateSession(c.Request.Context(), user, map[string]string{
		"ip_address": c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to create session")
		return
	}

	handlers.SuccessResponse(c, map[string]any{
		"user":    user,
		"session": session.ID,
	})
}

// handleLogout handles logout
func (s *Server) handleLogout(c *gin.Context) {
	// Extract session from context (added by auth middleware)
	session, ok := c.Request.Context().Value("session").(*auth.Session)
	if !ok {
		handlers.UnauthorizedResponse(c, "No active session")
		return
	}

	// Delete session
	if err := s.sessionMgr.DeleteSession(c.Request.Context(), session.ID); err != nil {
		handlers.InternalErrorResponse(c, "Failed to logout")
		return
	}

	handlers.SuccessResponse(c, nil)
}

// handleRefreshToken handles token refresh
func (s *Server) handleRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.ValidationErrorResponse(c, map[string]string{
			"refresh_token": "refresh_token is required",
		})
		return
	}

	// Refresh token
	tokens, err := s.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handlers.ErrorResponse(
			c,
			http.StatusUnauthorized,
			"refresh_failed",
			"Failed to refresh token",
		)
		return
	}

	handlers.SuccessResponse(c, tokens)
}

// handleServers handles server listing (legacy endpoint)
func (s *Server) handleServers(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	// Execute CLI command to list servers
	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeServerList,
		Args:    []string{"--json"},
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to list servers")
		return
	}

	// Parse JSON output
	var servers any
	if err := json.Unmarshal([]byte(result.Stdout), &servers); err != nil {
		// If not JSON, return raw output
		handlers.SuccessResponse(c, map[string]string{"output": result.Stdout})
		return
	}

	handlers.SuccessResponse(c, servers)
}

// handleServerAction handles individual server actions (POST)
func (s *Server) handleServerAction(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	// Extract server name and action from path
	serverName := c.Param("name")
	action := c.Param("action")

	if serverName == "" {
		handlers.ValidationErrorResponse(c, map[string]string{
			"name": "Server name is required",
		})
		return
	}

	if action == "" {
		handlers.ValidationErrorResponse(c, map[string]string{
			"action": "Action is required",
		})
		return
	}

	s.performServerAction(c, user.ID.String(), serverName, action)
}

// handleServerInspect inspects a specific server (GET)
func (s *Server) handleServerInspect(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	serverName := c.Param("name")
	if serverName == "" {
		handlers.ValidationErrorResponse(c, map[string]string{
			"name": "Server name is required",
		})
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeServerInspect,
		Args:    []string{serverName, "--json"},
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to inspect server")
		return
	}

	var serverInfo any
	if err := json.Unmarshal([]byte(result.Stdout), &serverInfo); err != nil {
		handlers.SuccessResponse(c, map[string]string{"output": result.Stdout})
		return
	}

	handlers.SuccessResponse(c, serverInfo)
}

// performServerAction performs an action on a server
func (s *Server) performServerAction(c *gin.Context, userID, serverName, action string) {
	var cmdType executor.CommandType
	switch action {
	case "enable":
		cmdType = executor.CommandTypeServerEnable
	case "disable":
		cmdType = executor.CommandTypeServerDisable
	default:
		handlers.ValidationErrorResponse(c, map[string]string{
			"action": "Invalid action. Valid actions are: enable, disable",
		})
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: cmdType,
		Args:    []string{serverName},
		UserID:  userID,
	})
	if err != nil {
		handlers.InternalErrorResponse(c, fmt.Sprintf("Failed to %s server", action))
		return
	}

	handlers.SuccessResponse(c, map[string]string{
		"output":  result.Stdout,
		"message": fmt.Sprintf("Server %s %sd successfully", serverName, action),
	})
}

// handleGatewayStart handles gateway start
func (s *Server) handleGatewayStart(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeGatewayRun,
		Args:    []string{"--detach"},
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to start gateway")
		return
	}

	handlers.SuccessResponse(c, map[string]string{
		"output":  result.Stdout,
		"message": "Gateway started successfully",
	})
}

// handleGatewayStop handles gateway stop
func (s *Server) handleGatewayStop(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeGatewayStop,
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to stop gateway")
		return
	}

	handlers.SuccessResponse(c, map[string]string{
		"output":  result.Stdout,
		"message": "Gateway stopped successfully",
	})
}

// handleGatewayStatus handles gateway status
func (s *Server) handleGatewayStatus(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeGatewayStatus,
		Args:    []string{"--json"},
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to get gateway status")
		return
	}

	var status any
	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		handlers.SuccessResponse(c, map[string]string{"output": result.Stdout})
		return
	}

	handlers.SuccessResponse(c, status)
}

// handleGetConfig handles configuration retrieval
func (s *Server) handleGetConfig(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	result, err := s.executor.Execute(c.Request.Context(), &executor.ExecutionRequest{
		Command: executor.CommandTypeConfigRead,
		Args:    []string{"--json"},
		UserID:  user.ID.String(),
	})
	if err != nil {
		handlers.InternalErrorResponse(c, "Failed to get configuration")
		return
	}

	var config any
	if err := json.Unmarshal([]byte(result.Stdout), &config); err != nil {
		handlers.SuccessResponse(c, map[string]string{"output": result.Stdout})
		return
	}

	handlers.SuccessResponse(c, config)
}

// handleUpdateConfig handles configuration updates
func (s *Server) handleUpdateConfig(c *gin.Context) {
	// Get user from context
	user, exists := handlers.GetUserFromContext(c)
	if !exists {
		handlers.UnauthorizedResponse(c, "Authentication required")
		return
	}

	// Use user variable to avoid unused variable warning
	_ = user

	// This would require more complex handling to write config
	// For now, return not implemented
	handlers.ErrorResponse(
		c,
		http.StatusNotImplemented,
		"not_implemented",
		"Configuration update not yet implemented",
	)
}

// realtimeAuthMiddleware creates middleware that extracts user ID for realtime connections
func (s *Server) realtimeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by Auth middleware)
		user, exists := handlers.GetUserFromContext(c)
		if !exists {
			handlers.UnauthorizedResponse(c, "Authentication required")
			c.Abort()
			return
		}

		// Set user_id for realtime service
		c.Set("user_id", user.ID.String())
		c.Next()
	}
}

// handleWebSocket handles WebSocket connection requests
func (s *Server) handleWebSocket(c *gin.Context) {
	// Delegate to realtime connection manager
	s.realtimeManager.HandleWebSocket(c)
}

// handleSSE handles Server-Sent Events connection requests
func (s *Server) handleSSE(c *gin.Context) {
	// Delegate to realtime connection manager
	s.realtimeManager.HandleSSE(c)
}
