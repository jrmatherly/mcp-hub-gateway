// Package auth provides Azure EntraID authentication for the MCP Portal.
// It implements OAuth2 flow with MSAL, JWT generation, refresh tokens, and JWKS validation.
package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserRole represents the role of an authenticated user
type UserRole string

const (
	RoleGuest       UserRole = "guest"
	RoleUser        UserRole = "user"
	RoleAdmin       UserRole = "admin"
	RoleSuperAdmin  UserRole = "super_admin"
	RoleSystemAdmin UserRole = "system_admin"
)

// User represents an authenticated user from Azure AD
type User struct {
	ID          uuid.UUID      `json:"id"                    db:"id"`
	TenantID    string         `json:"tenant_id"             db:"tenant_id"`
	Email       string         `json:"email"                 db:"email"`
	Name        string         `json:"name"                  db:"name"`
	GivenName   string         `json:"given_name"            db:"given_name"`
	FamilyName  string         `json:"family_name"           db:"family_name"`
	Role        UserRole       `json:"role"                  db:"role"`
	AzureID     string         `json:"azure_id"              db:"azure_id"`
	IsActive    bool           `json:"is_active"             db:"is_active"`
	LastLogin   *time.Time     `json:"last_login"            db:"last_login"`
	CreatedAt   time.Time      `json:"created_at"            db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"            db:"updated_at"`
	Permissions []Permission   `json:"permissions,omitempty"`
	Claims      map[string]any `json:"claims,omitempty"`
}

// Permission represents a user permission
type Permission string

const (
	PermissionServerList    Permission = "server:list"
	PermissionServerEnable  Permission = "server:enable"
	PermissionServerDisable Permission = "server:disable"
	PermissionServerInspect Permission = "server:inspect"
	PermissionGatewayRun    Permission = "gateway:run"
	PermissionGatewayStop   Permission = "gateway:stop"
	PermissionGatewayLogs   Permission = "gateway:logs"
	PermissionGatewayConfig Permission = "gateway:config"
	PermissionConfigRead    Permission = "config:read"
	PermissionConfigWrite   Permission = "config:write"
	PermissionSecretRead    Permission = "secret:read"
	PermissionSecretWrite   Permission = "secret:write"
	PermissionSecretSet     Permission = "secret:set"
	PermissionSecretRemove  Permission = "secret:remove"
	PermissionAuditRead     Permission = "audit:read"
	PermissionUserManage    Permission = "user:manage"
	PermissionSystemManage  Permission = "system:manage"
)

// Session represents an active user session
type Session struct {
	ID           uuid.UUID         `json:"id"            redis:"id"`
	UserID       uuid.UUID         `json:"user_id"       redis:"user_id"`
	TenantID     string            `json:"tenant_id"     redis:"tenant_id"`
	AccessToken  string            `json:"access_token"  redis:"access_token"`
	RefreshToken string            `json:"refresh_token" redis:"refresh_token"`
	ExpiresAt    time.Time         `json:"expires_at"    redis:"expires_at"`
	CreatedAt    time.Time         `json:"created_at"    redis:"created_at"`
	LastActivity time.Time         `json:"last_activity" redis:"last_activity"`
	IPAddress    string            `json:"ip_address"    redis:"ip_address"`
	UserAgent    string            `json:"user_agent"    redis:"user_agent"`
	IsActive     bool              `json:"is_active"     redis:"is_active"`
	Metadata     map[string]string `json:"metadata"      redis:"metadata"`
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// TokenResponse represents the response from Azure AD token endpoint
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthContext represents authentication context passed through requests
type AuthContext struct {
	User      *User     `json:"user"`
	Session   *Session  `json:"session"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// Claims represents JWT claims structure
type Claims struct {
	UserID      uuid.UUID      `json:"user_id"`
	TenantID    string         `json:"tenant_id"`
	Email       string         `json:"email"`
	Name        string         `json:"name"`
	Role        UserRole       `json:"role"`
	Permissions []Permission   `json:"permissions"`
	SessionID   uuid.UUID      `json:"session_id"`
	AzureClaims map[string]any `json:"azure_claims"`
	jwt.RegisteredClaims
}

// VerifyAudience verifies the audience claim in the JWT
func (c *Claims) VerifyAudience(audience string, required bool) bool {
	if c.Audience == nil || len(c.Audience) == 0 {
		return !required
	}

	for _, aud := range c.Audience {
		if aud == audience {
			return true
		}
	}
	return false
}

// AuthError represents authentication-related errors
type AuthError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	HTTPStatus  int    `json:"http_status"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common authentication error codes
const (
	ErrorCodeInvalidToken       = "invalid_token"
	ErrorCodeExpiredToken       = "expired_token"
	ErrorCodeInvalidCredentials = "invalid_credentials"
	ErrorCodeAccessDenied       = "access_denied"
	ErrorCodeInsufficientPerms  = "insufficient_permissions"
	ErrorCodeSessionExpired     = "session_expired"
	ErrorCodeInvalidState       = "invalid_state"
	ErrorCodeInvalidRequest     = "invalid_request"
	ErrorCodeServerError        = "server_error"
	ErrorCodeRateLimited        = "rate_limited"
)

// Authentication errors
var (
	ErrInvalidToken = &AuthError{
		Code:       ErrorCodeInvalidToken,
		Message:    "Invalid authentication token",
		HTTPStatus: 401,
	}
	ErrExpiredToken = &AuthError{
		Code:       ErrorCodeExpiredToken,
		Message:    "Authentication token has expired",
		HTTPStatus: 401,
	}
	ErrAccessDenied = &AuthError{
		Code:       ErrorCodeAccessDenied,
		Message:    "Access denied",
		HTTPStatus: 403,
	}
	ErrInsufficientPermissions = &AuthError{
		Code:       ErrorCodeInsufficientPerms,
		Message:    "Insufficient permissions",
		HTTPStatus: 403,
	}
	ErrSessionExpired = &AuthError{
		Code:       ErrorCodeSessionExpired,
		Message:    "Session has expired",
		HTTPStatus: 401,
	}
)

// Authenticator defines the main authentication interface
type Authenticator interface {
	// OAuth2 flow
	GetAuthURL(state string, redirectURI string) (string, error)
	HandleCallback(ctx context.Context, code, state, redirectURI string) (*TokenPair, *User, error)

	// Token management
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	RevokeToken(ctx context.Context, token string) error

	// User management
	GetUser(ctx context.Context, userID uuid.UUID) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeactivateUser(ctx context.Context, userID uuid.UUID) error

	// Permission checking
	HasPermission(ctx context.Context, userID uuid.UUID, permission Permission) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error)
}

// SessionManager defines session management interface
type SessionManager interface {
	// Session lifecycle
	CreateSession(ctx context.Context, user *User, metadata map[string]string) (*Session, error)
	GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error

	// Session validation
	ValidateSession(ctx context.Context, sessionID uuid.UUID) (*Session, error)
	RefreshSession(ctx context.Context, sessionID uuid.UUID) (*Session, error)

	// Bulk operations
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) (int, error)

	// Token-based invalidation
	InvalidateByRefreshToken(ctx context.Context, refreshToken string) error
}

// JWTManager defines JWT token management interface
type JWTManager interface {
	// Token generation
	GenerateToken(ctx context.Context, claims *Claims) (string, error)
	GenerateRefreshToken(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (string, error)

	// Token validation
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	ValidateRefreshToken(ctx context.Context, refreshToken string) (*Claims, error)

	// JWKS operations
	GetPublicKey(ctx context.Context, keyID string) (any, error)
	GetJWKS(ctx context.Context) (any, error)
	RotateKeys(ctx context.Context) error
}

// UserRepository defines user data access interface
type UserRepository interface {
	// CRUD operations
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByAzureID(ctx context.Context, azureID string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Queries
	List(ctx context.Context, limit, offset int) ([]*User, error)
	ListByRole(ctx context.Context, role UserRole) ([]*User, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*User, error)

	// Permissions
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error)
	SetUserPermissions(ctx context.Context, userID uuid.UUID, permissions []Permission) error
	AddUserPermission(ctx context.Context, userID uuid.UUID, permission Permission) error
	RemoveUserPermission(ctx context.Context, userID uuid.UUID, permission Permission) error
}

// RolePermissions defines default permissions for each role
var RolePermissions = map[UserRole][]Permission{
	RoleGuest: {},
	RoleUser: {
		PermissionServerList,
		PermissionServerInspect,
		PermissionGatewayLogs,
		PermissionConfigRead,
	},
	RoleAdmin: {
		PermissionServerList,
		PermissionServerEnable,
		PermissionServerDisable,
		PermissionServerInspect,
		PermissionGatewayRun,
		PermissionGatewayStop,
		PermissionGatewayLogs,
		PermissionConfigRead,
		PermissionConfigWrite,
		PermissionSecretRead,
		PermissionAuditRead,
	},
	RoleSuperAdmin: {
		PermissionServerList,
		PermissionServerEnable,
		PermissionServerDisable,
		PermissionServerInspect,
		PermissionGatewayRun,
		PermissionGatewayStop,
		PermissionGatewayLogs,
		PermissionConfigRead,
		PermissionConfigWrite,
		PermissionSecretRead,
		PermissionSecretWrite,
		PermissionAuditRead,
		PermissionUserManage,
	},
	RoleSystemAdmin: {
		PermissionServerList,
		PermissionServerEnable,
		PermissionServerDisable,
		PermissionServerInspect,
		PermissionGatewayRun,
		PermissionGatewayStop,
		PermissionGatewayLogs,
		PermissionConfigRead,
		PermissionConfigWrite,
		PermissionSecretRead,
		PermissionSecretWrite,
		PermissionAuditRead,
		PermissionUserManage,
		PermissionSystemManage,
	},
}

// GetRolePermissions returns default permissions for a role
func GetRolePermissions(role UserRole) []Permission {
	if perms, exists := RolePermissions[role]; exists {
		return perms
	}
	return []Permission{}
}

// HasRolePermission checks if a role has a specific permission
func HasRolePermission(role UserRole, permission Permission) bool {
	perms := GetRolePermissions(role)
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// AuthContextKey is used for context values
type AuthContextKey string

const (
	// Context keys
	ContextKeyUser      AuthContextKey = "auth_user"
	ContextKeySession   AuthContextKey = "auth_session"
	ContextKeyRequestID AuthContextKey = "auth_request_id"
	ContextKeyIPAddress AuthContextKey = "auth_ip_address"
	ContextKeyUserAgent AuthContextKey = "auth_user_agent"
)

// GetUserFromContext extracts user from context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(ContextKeyUser).(*User)
	return user, ok
}

// GetSessionFromContext extracts session from context
func GetSessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(ContextKeySession).(*Session)
	return session, ok
}

// WithUser adds user to context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, ContextKeyUser, user)
}

// WithSession adds session to context
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, ContextKeySession, session)
}

// WithAuthContext adds complete auth context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	ctx = WithUser(ctx, authCtx.User)
	ctx = WithSession(ctx, authCtx.Session)
	ctx = context.WithValue(ctx, ContextKeyRequestID, authCtx.RequestID)
	ctx = context.WithValue(ctx, ContextKeyIPAddress, authCtx.IPAddress)
	ctx = context.WithValue(ctx, ContextKeyUserAgent, authCtx.UserAgent)
	return ctx
}

// Valid validates the claims (implements jwt.Claims interface)
func (c *Claims) Valid() error {
	// Additional validation specific to our claims
	if c.UserID == uuid.Nil {
		return &AuthError{
			Code:       ErrorCodeInvalidToken,
			Message:    "Invalid user ID in token",
			HTTPStatus: 401,
		}
	}

	if c.Email == "" {
		return &AuthError{
			Code:       ErrorCodeInvalidToken,
			Message:    "Email is required in token",
			HTTPStatus: 401,
		}
	}

	return nil
}
