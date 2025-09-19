// Package oauth provides OAuth interceptor middleware for MCP Portal.
// It handles automatic token injection, 401 detection/retry, and multi-provider support
// for MCP servers requiring OAuth authentication.
package oauth

import (
	"context"
	"crypto"
	"time"

	"github.com/google/uuid"
)

// ProviderType represents the OAuth provider type
type ProviderType string

const (
	ProviderTypeGitHub    ProviderType = "github"
	ProviderTypeGoogle    ProviderType = "google"
	ProviderTypeMicrosoft ProviderType = "microsoft"
	ProviderTypeCustom    ProviderType = "custom"
)

// TokenType represents the type of OAuth token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access_token"
	TokenTypeRefresh TokenType = "refresh_token"
	TokenTypeIDToken TokenType = "id_token"
)

// StorageTier represents the hierarchical storage priority
type StorageTier int

const (
	StorageTierKeyVault      StorageTier = 1 // Highest priority
	StorageTierDockerDesktop StorageTier = 2 // Middle priority
	StorageTierEnvironment   StorageTier = 3 // Lowest priority
)

// ServerConfig represents OAuth configuration for an MCP server
type ServerConfig struct {
	// Server identification
	ServerName   string       `json:"server_name"         yaml:"server_name"`
	ProviderType ProviderType `json:"provider_type"       yaml:"provider_type"`
	TenantID     string       `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty"`

	// OAuth configuration
	ClientID     string   `json:"client_id"               yaml:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`
	Scopes       []string `json:"scopes"                  yaml:"scopes"`
	RedirectURI  string   `json:"redirect_uri"            yaml:"redirect_uri"`

	// Provider-specific configuration
	AuthURL  string            `json:"auth_url,omitempty"  yaml:"auth_url,omitempty"`
	TokenURL string            `json:"token_url,omitempty" yaml:"token_url,omitempty"`
	JWKSURL  string            `json:"jwks_url,omitempty"  yaml:"jwks_url,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"     yaml:"extra,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by" yaml:"created_by"`
	IsActive  bool      `json:"is_active"  yaml:"is_active"`
}

// TokenData represents OAuth token with metadata
type TokenData struct {
	// Token identification
	ServerName   string       `json:"server_name"`
	UserID       uuid.UUID    `json:"user_id"`
	TenantID     string       `json:"tenant_id,omitempty"`
	ProviderType ProviderType `json:"provider_type"`

	// Token values
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`

	// Timing
	ExpiresAt time.Time `json:"expires_at"`
	RefreshAt time.Time `json:"refresh_at"` // When to proactively refresh
	IssuedAt  time.Time `json:"issued_at"`

	// Metadata
	Scopes      []string    `json:"scopes"`
	StorageTier StorageTier `json:"storage_tier"`
	LastUsed    time.Time   `json:"last_used"`
	UsageCount  int64       `json:"usage_count"`
}

// AuthRequest represents an OAuth authentication request context
type AuthRequest struct {
	// Request identification
	RequestID  string    `json:"request_id"`
	ServerName string    `json:"server_name"`
	UserID     uuid.UUID `json:"user_id"`
	TenantID   string    `json:"tenant_id,omitempty"`

	// Request details
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`

	// Timing and retries
	Timestamp    time.Time `json:"timestamp"`
	AttemptCount int       `json:"attempt_count"`
	MaxRetries   int       `json:"max_retries"`

	// Context
	UserAgent  string `json:"user_agent,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
}

// AuthResponse represents the response from an OAuth-protected request
type AuthResponse struct {
	// Response identification
	RequestID  string `json:"request_id"`
	StatusCode int    `json:"status_code"`

	// Response data
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`

	// Timing
	Duration       time.Duration `json:"duration"`
	TokenRefreshed bool          `json:"token_refreshed"`

	// Error information
	Error        string            `json:"error,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorDetails map[string]string `json:"error_details,omitempty"`
}

// RetryPolicy defines retry behavior for failed OAuth requests
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"      yaml:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval" yaml:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"     yaml:"max_interval"`
	Multiplier      float64       `json:"multiplier"       yaml:"multiplier"`
	Jitter          bool          `json:"jitter"           yaml:"jitter"`

	// Retry conditions
	RetryOn401 bool `json:"retry_on_401" yaml:"retry_on_401"`
	RetryOn403 bool `json:"retry_on_403" yaml:"retry_on_403"`
	RetryOn429 bool `json:"retry_on_429" yaml:"retry_on_429"`
	RetryOn5xx bool `json:"retry_on_5xx" yaml:"retry_on_5xx"`
}

// InterceptorConfig represents configuration for the OAuth interceptor
type InterceptorConfig struct {
	// General settings
	Enabled          bool          `json:"enabled"           yaml:"enabled"`
	DefaultTimeout   time.Duration `json:"default_timeout"   yaml:"default_timeout"`
	RefreshThreshold time.Duration `json:"refresh_threshold" yaml:"refresh_threshold"`

	// Retry configuration
	RetryPolicy RetryPolicy `json:"retry_policy" yaml:"retry_policy"`

	// Storage configuration
	StorageTiers  []StorageTier `json:"storage_tiers"  yaml:"storage_tiers"`
	EncryptTokens bool          `json:"encrypt_tokens" yaml:"encrypt_tokens"`

	// Security settings
	ValidateJWTs   bool     `json:"validate_jwts"             yaml:"validate_jwts"`
	RequireHTTPS   bool     `json:"require_https"             yaml:"require_https"`
	AllowedDomains []string `json:"allowed_domains,omitempty" yaml:"allowed_domains,omitempty"`

	// Feature flags
	EnableDCRBridge bool `json:"enable_dcr_bridge" yaml:"enable_dcr_bridge"`
	EnableMetrics   bool `json:"enable_metrics"    yaml:"enable_metrics"`
	EnableAuditLog  bool `json:"enable_audit_log"  yaml:"enable_audit_log"`
}

// DCRRequest represents a Dynamic Client Registration request (RFC 7591)
type DCRRequest struct {
	// Required fields
	RedirectURIs []string `json:"redirect_uris"`

	// Optional metadata
	ClientName      string   `json:"client_name,omitempty"`
	ClientURI       string   `json:"client_uri,omitempty"`
	LogoURI         string   `json:"logo_uri,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	Contacts        []string `json:"contacts,omitempty"`
	TosURI          string   `json:"tos_uri,omitempty"`
	PolicyURI       string   `json:"policy_uri,omitempty"`
	JWKSURI         string   `json:"jwks_uri,omitempty"`
	SoftwareID      string   `json:"software_id,omitempty"`
	SoftwareVersion string   `json:"software_version,omitempty"`

	// Technical settings
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`

	// Extension fields for Azure AD bridge
	TenantID        string `json:"tenant_id,omitempty"`
	ApplicationType string `json:"application_type,omitempty"`
}

// DCRResponse represents a Dynamic Client Registration response
type DCRResponse struct {
	// Client credentials
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`

	// Registration metadata
	ClientIDIssuedAt        int64  `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt   int64  `json:"client_secret_expires_at,omitempty"`
	RegistrationClientURI   string `json:"registration_client_uri,omitempty"`
	RegistrationAccessToken string `json:"registration_access_token,omitempty"`

	// Echo back the request fields
	DCRRequest

	// Azure AD specific fields
	ApplicationID string `json:"application_id,omitempty"`
	ObjectID      string `json:"object_id,omitempty"`
}

// OAuthInterceptor defines the main OAuth interceptor interface
type OAuthInterceptor interface {
	// Request interception
	InterceptRequest(ctx context.Context, req *AuthRequest) (*AuthResponse, error)

	// Token management
	GetToken(ctx context.Context, serverName string, userID uuid.UUID) (*TokenData, error)
	RefreshToken(ctx context.Context, serverName string, userID uuid.UUID) (*TokenData, error)
	StoreToken(ctx context.Context, token *TokenData) error
	RevokeToken(ctx context.Context, serverName string, userID uuid.UUID) error

	// Server configuration
	RegisterServer(ctx context.Context, config *ServerConfig) error
	GetServerConfig(ctx context.Context, serverName string) (*ServerConfig, error)
	UpdateServerConfig(ctx context.Context, config *ServerConfig) error
	RemoveServerConfig(ctx context.Context, serverName string) error

	// Health and monitoring
	Health(ctx context.Context) error
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
}

// TokenStorage defines the interface for hierarchical token storage
type TokenStorage interface {
	// Storage operations
	StoreToken(ctx context.Context, token *TokenData, tier StorageTier) error
	GetToken(ctx context.Context, serverName string, userID uuid.UUID) (*TokenData, error)
	RefreshToken(ctx context.Context, serverName string, userID uuid.UUID) (*TokenData, error)
	DeleteToken(ctx context.Context, serverName string, userID uuid.UUID) error

	// Metadata operations
	GetStorageTier(ctx context.Context, serverName string, userID uuid.UUID) (StorageTier, error)
	ListTokens(ctx context.Context, userID uuid.UUID) ([]*TokenData, error)

	// Cleanup operations
	CleanupExpiredTokens(ctx context.Context) (int, error)
	MigrateTokens(ctx context.Context, fromTier, toTier StorageTier) (int, error)

	// Health
	Health(ctx context.Context) error
}

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	// Provider identification
	GetProviderType() ProviderType
	GetProviderName() string

	// OAuth flow
	GetAuthURL(config *ServerConfig, state string) (string, error)
	ExchangeCode(ctx context.Context, config *ServerConfig, code string) (*TokenData, error)
	RefreshToken(ctx context.Context, config *ServerConfig, refreshToken string) (*TokenData, error)
	RevokeToken(ctx context.Context, config *ServerConfig, token string) error

	// Token validation
	ValidateToken(ctx context.Context, config *ServerConfig, token string) (*TokenClaims, error)
	GetUserInfo(
		ctx context.Context,
		config *ServerConfig,
		token string,
	) (map[string]interface{}, error)

	// Provider-specific features
	SupportsRefresh() bool
	SupportsRevocation() bool
	GetDefaultScopes() []string
	GetTokenExpiry(token string) (time.Time, error)
}

// ProviderRegistry manages OAuth providers
type ProviderRegistry interface {
	// Provider management
	RegisterProvider(provider OAuthProvider) error
	GetProvider(providerType ProviderType) (OAuthProvider, error)
	ListProviders() []ProviderType

	// Dynamic Client Registration support
	SupportsDCR(providerType ProviderType) bool
	RegisterDynamicClient(
		ctx context.Context,
		providerType ProviderType,
		req *DCRRequest,
	) (*DCRResponse, error)
}

// DCRBridge defines the interface for Dynamic Client Registration bridge
type DCRBridge interface {
	// RFC 7591 Dynamic Client Registration
	RegisterClient(ctx context.Context, req *DCRRequest) (*DCRResponse, error)
	GetClient(ctx context.Context, clientID string) (*DCRResponse, error)
	UpdateClient(ctx context.Context, clientID string, req *DCRRequest) (*DCRResponse, error)
	DeleteClient(ctx context.Context, clientID string) error

	// Provider-specific operations
	SupportsProvider(providerType ProviderType) bool
	GetProviderEndpoints(providerType ProviderType) (authURL, tokenURL, jwksURL string, err error)
}

// TokenClaims represents decoded token claims
type TokenClaims struct {
	// Standard claims
	Issuer    string    `json:"iss"`
	Subject   string    `json:"sub"`
	Audience  []string  `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
	NotBefore time.Time `json:"nbf"`
	IssuedAt  time.Time `json:"iat"`
	JWTID     string    `json:"jti"`

	// OAuth-specific claims
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	TokenType string `json:"token_type,omitempty"`

	// Provider-specific claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`

	// Custom claims
	Custom map[string]interface{} `json:"-"`
}

// AuditEvent represents an OAuth-related audit event
type AuditEvent struct {
	// Event identification
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`

	// Context
	UserID     uuid.UUID `json:"user_id"`
	TenantID   string    `json:"tenant_id,omitempty"`
	ServerName string    `json:"server_name"`
	RequestID  string    `json:"request_id,omitempty"`

	// Event details
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`

	// Metadata
	Provider  ProviderType           `json:"provider"`
	Operation string                 `json:"operation"`
	Details   map[string]interface{} `json:"details,omitempty"`

	// Security context
	RemoteAddr     string `json:"remote_addr,omitempty"`
	UserAgent      string `json:"user_agent,omitempty"`
	TokenRefreshed bool   `json:"token_refreshed,omitempty"`
}

// AuditLogger defines the interface for OAuth audit logging
type AuditLogger interface {
	// Log OAuth events
	LogOAuthEvent(ctx context.Context, event *AuditEvent) error
	LogTokenRefresh(ctx context.Context, serverName string, userID uuid.UUID, success bool) error
	LogAuthorizationFlow(
		ctx context.Context,
		serverName string,
		userID uuid.UUID,
		provider ProviderType,
		success bool,
	) error
	LogTokenRevocation(ctx context.Context, serverName string, userID uuid.UUID, success bool) error

	// Query audit logs
	GetUserActivity(ctx context.Context, userID uuid.UUID, since time.Time) ([]*AuditEvent, error)
	GetServerActivity(
		ctx context.Context,
		serverName string,
		since time.Time,
	) ([]*AuditEvent, error)
	GetFailedAttempts(ctx context.Context, since time.Time) ([]*AuditEvent, error)
}

// Metrics defines OAuth metrics
type Metrics struct {
	// Request metrics
	TotalRequests      int64 `json:"total_requests"`
	SuccessfulRequests int64 `json:"successful_requests"`
	FailedRequests     int64 `json:"failed_requests"`
	TokenRefreshCount  int64 `json:"token_refresh_count"`

	// Timing metrics
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`

	// Provider metrics
	ProviderCounts    map[ProviderType]int64         `json:"provider_counts"`
	ProviderLatencies map[ProviderType]time.Duration `json:"provider_latencies"`

	// Error metrics
	ErrorCounts   map[string]int64 `json:"error_counts"`
	RetryAttempts int64            `json:"retry_attempts"`

	// Token metrics
	ActiveTokens     int64                 `json:"active_tokens"`
	ExpiredTokens    int64                 `json:"expired_tokens"`
	StorageTierUsage map[StorageTier]int64 `json:"storage_tier_usage"`

	// System metrics
	LastUpdated   time.Time `json:"last_updated"`
	UptimeSeconds int64     `json:"uptime_seconds"`
}

// MetricsCollector defines the interface for OAuth metrics collection
type MetricsCollector interface {
	// Request tracking
	RecordRequest(
		ctx context.Context,
		serverName string,
		provider ProviderType,
		duration time.Duration,
		success bool,
	)
	RecordTokenRefresh(ctx context.Context, serverName string, provider ProviderType, success bool)
	RecordError(ctx context.Context, errorType string, serverName string, provider ProviderType)

	// Current state
	GetMetrics(ctx context.Context) (*Metrics, error)
	Reset(ctx context.Context) error
}

// ConfigValidator defines validation for OAuth configurations
type ConfigValidator interface {
	// Server configuration validation
	ValidateServerConfig(config *ServerConfig) []ValidationError
	ValidateProviderConfig(providerType ProviderType, config map[string]string) []ValidationError

	// Token validation
	ValidateToken(token *TokenData) []ValidationError
	ValidateTokenClaims(claims *TokenClaims) []ValidationError

	// DCR validation
	ValidateDCRRequest(req *DCRRequest) []ValidationError
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// HTTPClient defines the interface for making HTTP requests with OAuth
type HTTPClient interface {
	// OAuth-aware HTTP methods
	Get(ctx context.Context, serverName string, userID uuid.UUID, url string) (*AuthResponse, error)
	Post(
		ctx context.Context,
		serverName string,
		userID uuid.UUID,
		url string,
		body []byte,
	) (*AuthResponse, error)
	Put(
		ctx context.Context,
		serverName string,
		userID uuid.UUID,
		url string,
		body []byte,
	) (*AuthResponse, error)
	Delete(
		ctx context.Context,
		serverName string,
		userID uuid.UUID,
		url string,
	) (*AuthResponse, error)

	// Raw HTTP with custom headers
	Do(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
}

// KeyRotator handles automatic key rotation for OAuth clients
type KeyRotator interface {
	// Key management
	RotateClientSecret(ctx context.Context, serverName string) error
	RotateSigningKey(ctx context.Context, serverName string) error

	// Key retrieval
	GetCurrentKey(ctx context.Context, serverName string) (crypto.PublicKey, error)
	GetKeyHistory(ctx context.Context, serverName string) ([]crypto.PublicKey, error)

	// Scheduled rotation
	ScheduleRotation(ctx context.Context, serverName string, interval time.Duration) error
	CancelRotation(ctx context.Context, serverName string) error
}
