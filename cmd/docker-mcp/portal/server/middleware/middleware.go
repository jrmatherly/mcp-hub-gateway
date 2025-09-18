// Package middleware provides HTTP middleware for the MCP Portal server.
// It includes authentication, rate limiting, logging, recovery, and security middleware.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/auth"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/ratelimit"
)

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// Logger middleware provides structured request logging
func Logger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC3339),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		},
		Output:    gin.DefaultWriter,
		SkipPaths: []string{"/health", "/ready"},
	})
}

// Recovery middleware provides panic recovery with audit logging
func Recovery(auditLogger audit.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Log the panic to audit logger
		if auditLogger != nil {
			// Get user ID from context if available
			var userID uuid.UUID
			if user, exists := auth.GetUserFromContext(c.Request.Context()); exists {
				userID = user.ID
			}

			// Create comprehensive metadata including all relevant fields
			metadata := map[string]interface{}{
				"panic_error": fmt.Sprintf("Panic recovered: %v", recovered),
				"stack_trace": string(debug.Stack()),
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"ip_address":  c.ClientIP(),
				"user_agent":  c.Request.UserAgent(),
				"severity":    "critical",
				"timestamp":   time.Now().UTC(),
				"request_id":  c.GetString("request_id"),
			}

			// Log as security alert with all context
			auditLogger.LogSecurityEvent(
				c.Request.Context(),
				userID,
				audit.EventTypeSecurityAlert,
				metadata,
			)
		}

		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// SecurityHeaders middleware adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent content type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'"
		c.Header("Content-Security-Policy", csp)

		// Strict Transport Security (only for HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// RateLimit middleware provides rate limiting with audit logging
func RateLimit(rateLimiter ratelimit.RateLimiter, auditLogger audit.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from context if available
		var userID uuid.UUID
		if user, exists := auth.GetUserFromContext(c.Request.Context()); exists {
			userID = user.ID
		}

		// Check rate limit
		command := getCommandFromPath(c.Request.URL.Path)
		if !rateLimiter.Allow(c.Request.Context(), userID, command) {
			// Log rate limit exceeded
			if auditLogger != nil {
				auditLogger.LogRateLimitExceeded(c.Request.Context(), userID, command)
			}

			// Get rate limit status for headers
			status := rateLimiter.GetStatus(c.Request.Context(), userID)
			c.Header("X-Rate-Limit-Limit", fmt.Sprintf("%d", status.RequestsLimit))
			c.Header("X-Rate-Limit-Remaining", "0")
			c.Header("X-Rate-Limit-Reset", fmt.Sprintf("%d", status.WindowResetTime.Unix()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please try again later.",
				"retry_after": int(time.Until(status.WindowResetTime).Seconds()),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		status := rateLimiter.GetStatus(c.Request.Context(), userID)
		c.Header("X-Rate-Limit-Limit", fmt.Sprintf("%d", status.RequestsLimit))
		c.Header(
			"X-Rate-Limit-Remaining",
			fmt.Sprintf("%d", status.RequestsLimit-status.RequestsUsed),
		)
		c.Header("X-Rate-Limit-Reset", fmt.Sprintf("%d", status.WindowResetTime.Unix()))

		c.Next()
	}
}

// Auth middleware provides JWT authentication
func Auth(authenticator auth.Authenticator, auditLogger audit.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_authorization",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_authorization",
				"message": "Authorization header must be 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authenticator.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Log authentication failure
			if auditLogger != nil {
				auditLogger.LogSecurityEvent(
					c.Request.Context(),
					uuid.New(),
					audit.EventTypeAuthentication,
					map[string]interface{}{
						"success": false,
						"error":   err.Error(),
						"ip":      c.ClientIP(),
						"agent":   c.Request.UserAgent(),
					},
				)
			}

			var authErr *auth.AuthError
			if errors.As(err, &authErr) {
				// authErr is already set by errors.As
			} else {
				// Use the predefined error
				authErr = &auth.AuthError{
					Code:       auth.ErrorCodeInvalidToken,
					Message:    "Invalid authentication token",
					HTTPStatus: 401,
				}
			}

			c.JSON(authErr.HTTPStatus, gin.H{
				"error":   authErr.Code,
				"message": authErr.Message,
			})
			c.Abort()
			return
		}

		// Get user information
		user, err := authenticator.GetUser(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "user_deactivated",
				"message": "User account is deactivated",
			})
			c.Abort()
			return
		}

		// Add user to context
		ctx := auth.WithUser(c.Request.Context(), user)
		c.Request = c.Request.WithContext(ctx)

		// Log successful authentication
		if auditLogger != nil {
			auditLogger.LogSecurityEvent(
				c.Request.Context(),
				user.ID,
				audit.EventTypeAuthentication,
				map[string]interface{}{
					"success": true,
					"ip":      c.ClientIP(),
					"agent":   c.Request.UserAgent(),
				},
			)
		}

		c.Next()
	}
}

// RequireRole middleware requires a specific minimum role
func RequireRole(minRole auth.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := auth.GetUserFromContext(c.Request.Context())
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "User authentication is required",
			})
			c.Abort()
			return
		}

		if !hasMinimumRole(user.Role, minRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_role",
				"message": fmt.Sprintf("Minimum role required: %s", minRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware requires a specific permission
func RequirePermission(permission auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := auth.GetUserFromContext(c.Request.Context())
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "User authentication is required",
			})
			c.Abort()
			return
		}

		// Check if user has the required permission
		hasPermission := false
		for _, perm := range user.Permissions {
			if perm == permission {
				hasPermission = true
				break
			}
		}

		// Also check role-based permissions
		if !hasPermission {
			hasPermission = auth.HasRolePermission(user.Role, permission)
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("Permission required: %s", permission),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasMinimumRole checks if the user's role meets the minimum requirement
func hasMinimumRole(userRole, minRole auth.UserRole) bool {
	roleHierarchy := map[auth.UserRole]int{
		auth.RoleGuest:       0,
		auth.RoleUser:        1,
		auth.RoleAdmin:       2,
		auth.RoleSuperAdmin:  3,
		auth.RoleSystemAdmin: 4,
	}

	userLevel, userExists := roleHierarchy[userRole]
	minLevel, minExists := roleHierarchy[minRole]

	if !userExists || !minExists {
		return false
	}

	return userLevel >= minLevel
}

// getCommandFromPath extracts the command type from the request path
func getCommandFromPath(path string) string {
	// Remove /api/v1 prefix
	path = strings.TrimPrefix(path, "/api/v1")

	// Map paths to commands for rate limiting
	switch {
	case strings.HasPrefix(path, "/servers"):
		return "server_management"
	case strings.HasPrefix(path, "/gateway"):
		return "gateway_management"
	case strings.HasPrefix(path, "/catalog"):
		return "catalog_management"
	case strings.HasPrefix(path, "/config"):
		return "config_management"
	case strings.HasPrefix(path, "/secrets"):
		return "secret_management"
	case strings.HasPrefix(path, "/users"):
		return "user_management"
	case strings.HasPrefix(path, "/audit"):
		return "audit_access"
	default:
		return "general"
	}
}
