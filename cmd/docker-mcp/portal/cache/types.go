// Package cache provides Redis-based caching and session management for the MCP Portal.
// It implements connection pooling, circuit breaker patterns, and comprehensive session
// management with JWT storage, user context, and refresh token handling.
package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Cache defines the main cache interface for Redis operations
type Cache interface {
	// Basic cache operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Batch operations
	MultiGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MultiSet(ctx context.Context, items map[string]CacheItem) error
	MultiDelete(ctx context.Context, keys []string) error

	// Pattern operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	DeletePattern(ctx context.Context, pattern string) (int, error)

	// Counter operations
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	// Expiration operations
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Health and monitoring
	Health(ctx context.Context) error
	Info(ctx context.Context) (*CacheInfo, error)
	FlushDB(ctx context.Context) error
}

// SessionCache defines session-specific cache operations
type SessionCache interface {
	// Session CRUD operations
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	SetSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error
	RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error

	// Session queries
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) (int, error)
	GetActiveSessions(ctx context.Context, limit int) ([]*Session, error)

	// Session cleanup
	DeleteExpiredSessions(ctx context.Context) (int, error)
	GetSessionCount(ctx context.Context) (int64, error)
	GetUserSessionCount(ctx context.Context, userID uuid.UUID) (int64, error)
}

// RateLimitCache defines rate limiting cache operations
type RateLimitCache interface {
	// Rate limiting
	GetRateLimit(ctx context.Context, key string) (*RateLimitState, error)
	IncrementRateLimit(
		ctx context.Context,
		key string,
		window time.Duration,
		limit int,
	) (*RateLimitState, error)
	ResetRateLimit(ctx context.Context, key string) error

	// Token bucket rate limiting
	ConsumeTokens(
		ctx context.Context,
		key string,
		tokens int,
		capacity int,
		refillRate time.Duration,
	) (*TokenBucketState, error)
	GetTokenBucket(ctx context.Context, key string) (*TokenBucketState, error)
}

// CacheItem represents an item to be stored in cache
type CacheItem struct {
	Key   string
	Value []byte
	TTL   time.Duration
}

// Session represents a user session stored in cache
type Session struct {
	// Core session data
	ID           uuid.UUID `json:"id"            redis:"id"`
	UserID       uuid.UUID `json:"user_id"       redis:"user_id"`
	TenantID     string    `json:"tenant_id"     redis:"tenant_id"`
	AccessToken  string    `json:"access_token"  redis:"access_token"`
	RefreshToken string    `json:"refresh_token" redis:"refresh_token"`

	// Timing information
	CreatedAt    time.Time `json:"created_at"    redis:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"    redis:"expires_at"`
	LastActivity time.Time `json:"last_activity" redis:"last_activity"`

	// Client information
	IPAddress string `json:"ip_address" redis:"ip_address"`
	UserAgent string `json:"user_agent" redis:"user_agent"`

	// Status and metadata
	IsActive bool              `json:"is_active" redis:"is_active"`
	Metadata map[string]string `json:"metadata"  redis:"metadata"`

	// Security attributes
	RequiresMFA  bool   `json:"requires_mfa"  redis:"requires_mfa"`
	SecurityTier string `json:"security_tier" redis:"security_tier"`
}

// RateLimitState represents the current state of a rate limit
type RateLimitState struct {
	Key        string        `json:"key"`
	Count      int           `json:"count"`
	Limit      int           `json:"limit"`
	Window     time.Duration `json:"window"`
	ResetTime  time.Time     `json:"reset_time"`
	Remaining  int           `json:"remaining"`
	IsExceeded bool          `json:"is_exceeded"`
}

// TokenBucketState represents the current state of a token bucket rate limiter
type TokenBucketState struct {
	Key         string        `json:"key"`
	Tokens      int           `json:"tokens"`
	Capacity    int           `json:"capacity"`
	LastRefill  time.Time     `json:"last_refill"`
	RefillRate  time.Duration `json:"refill_rate"`
	IsAvailable bool          `json:"is_available"`
}

// CacheInfo represents cache server information and statistics
type CacheInfo struct {
	// Server information
	Version string        `json:"version"`
	Mode    string        `json:"mode"`
	Uptime  time.Duration `json:"uptime"`

	// Connection information
	ConnectedClients int `json:"connected_clients"`
	MaxClients       int `json:"max_clients"`

	// Memory information
	UsedMemory  int64   `json:"used_memory"`
	MaxMemory   int64   `json:"max_memory"`
	MemoryUsage float64 `json:"memory_usage_percent"`

	// Operation statistics
	TotalCommands  int64   `json:"total_commands"`
	CommandsPerSec float64 `json:"commands_per_sec"`
	HitRate        float64 `json:"hit_rate"`
	MissRate       float64 `json:"miss_rate"`

	// Key information
	TotalKeys   int64 `json:"total_keys"`
	ExpiredKeys int64 `json:"expired_keys"`
	EvictedKeys int64 `json:"evicted_keys"`

	// Performance metrics
	AvgLatency time.Duration `json:"avg_latency"`
	P99Latency time.Duration `json:"p99_latency"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata"`
}

// Config represents Redis configuration
type Config struct {
	// Connection settings
	Address  string `json:"address"  yaml:"address"`
	Password string `json:"password" yaml:"password"`
	Database int    `json:"database" yaml:"database"`

	// Pool settings
	PoolSize      int           `json:"pool_size"       yaml:"pool_size"`
	MinIdleConns  int           `json:"min_idle_conns"  yaml:"min_idle_conns"`
	MaxConnAge    time.Duration `json:"max_conn_age"    yaml:"max_conn_age"`
	PoolTimeout   time.Duration `json:"pool_timeout"    yaml:"pool_timeout"`
	IdleTimeout   time.Duration `json:"idle_timeout"    yaml:"idle_timeout"`
	IdleCheckFreq time.Duration `json:"idle_check_freq" yaml:"idle_check_freq"`

	// Timeout settings
	DialTimeout  time.Duration `json:"dial_timeout"  yaml:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"  yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// Retry settings
	MaxRetries      int           `json:"max_retries"       yaml:"max_retries"`
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"`

	// Circuit breaker settings
	CircuitBreakerEnabled          bool          `json:"circuit_breaker_enabled"           yaml:"circuit_breaker_enabled"`
	CircuitBreakerFailureThreshold int           `json:"circuit_breaker_failure_threshold" yaml:"circuit_breaker_failure_threshold"`
	CircuitBreakerTimeout          time.Duration `json:"circuit_breaker_timeout"           yaml:"circuit_breaker_timeout"`
	CircuitBreakerMaxRequests      int           `json:"circuit_breaker_max_requests"      yaml:"circuit_breaker_max_requests"`

	// Session settings
	SessionTTL             time.Duration `json:"session_ttl"              yaml:"session_ttl"`
	SessionCleanupInterval time.Duration `json:"session_cleanup_interval" yaml:"session_cleanup_interval"`
	MaxSessionsPerUser     int           `json:"max_sessions_per_user"    yaml:"max_sessions_per_user"`

	// Cache settings
	DefaultTTL           time.Duration `json:"default_ttl"           yaml:"default_ttl"`
	KeyPrefix            string        `json:"key_prefix"            yaml:"key_prefix"`
	CompressionEnabled   bool          `json:"compression_enabled"   yaml:"compression_enabled"`
	CompressionThreshold int           `json:"compression_threshold" yaml:"compression_threshold"`

	// Monitoring settings
	MetricsEnabled      bool          `json:"metrics_enabled"       yaml:"metrics_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	SlowLogThreshold    time.Duration `json:"slow_log_threshold"    yaml:"slow_log_threshold"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerStateClosed CircuitBreakerState = iota
	CircuitBreakerStateHalfOpen
	CircuitBreakerStateOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerStateClosed:
		return "closed"
	case CircuitBreakerStateHalfOpen:
		return "half-open"
	case CircuitBreakerStateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker defines circuit breaker functionality
type CircuitBreaker interface {
	// Execute wraps an operation with circuit breaker protection
	Execute(ctx context.Context, operation func() error) error

	// State returns current circuit breaker state
	State() CircuitBreakerState

	// Metrics returns circuit breaker metrics
	Metrics() *CircuitBreakerMetrics

	// Reset manually resets the circuit breaker to closed state
	Reset()
}

// CircuitBreakerMetrics contains circuit breaker statistics
type CircuitBreakerMetrics struct {
	State               CircuitBreakerState `json:"state"`
	TotalRequests       int64               `json:"total_requests"`
	SuccessfulReqs      int64               `json:"successful_requests"`
	FailedRequests      int64               `json:"failed_requests"`
	ConsecutiveFailures int64               `json:"consecutive_failures"`
	LastFailure         *time.Time          `json:"last_failure,omitempty"`
	LastSuccess         *time.Time          `json:"last_success,omitempty"`
	StateChangedAt      time.Time           `json:"state_changed_at"`
}

// CacheError represents cache-related errors
type CacheError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Operation string    `json:"operation"`
	Key       string    `json:"key,omitempty"`
	Retryable bool      `json:"retryable"`
	Timestamp time.Time `json:"timestamp"`
	Cause     error     `json:"-"`
}

func (e *CacheError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *CacheError) Unwrap() error {
	return e.Cause
}

// Common cache error codes
const (
	ErrorCodeConnectionFailed    = "connection_failed"
	ErrorCodeTimeout             = "timeout"
	ErrorCodeKeyNotFound         = "key_not_found"
	ErrorCodeSerializationFailed = "serialization_failed"
	ErrorCodeCircuitBreakerOpen  = "circuit_breaker_open"
	ErrorCodeRateLimitExceeded   = "rate_limit_exceeded"
	ErrorCodeInvalidArgument     = "invalid_argument"
	ErrorCodeServerError         = "server_error"
	ErrorCodeMemoryFull          = "memory_full"
	ErrorCodeReadOnly            = "read_only"
)

// Predefined cache errors
var (
	ErrConnectionFailed = &CacheError{
		Code:      ErrorCodeConnectionFailed,
		Message:   "failed to connect to cache server",
		Retryable: true,
		Timestamp: time.Now(),
	}

	ErrTimeout = &CacheError{
		Code:      ErrorCodeTimeout,
		Message:   "cache operation timed out",
		Retryable: true,
		Timestamp: time.Now(),
	}

	ErrKeyNotFound = &CacheError{
		Code:      ErrorCodeKeyNotFound,
		Message:   "key not found in cache",
		Retryable: false,
		Timestamp: time.Now(),
	}

	ErrCircuitBreakerOpen = &CacheError{
		Code:      ErrorCodeCircuitBreakerOpen,
		Message:   "circuit breaker is open",
		Retryable: false,
		Timestamp: time.Now(),
	}

	ErrRateLimitExceeded = &CacheError{
		Code:      ErrorCodeRateLimitExceeded,
		Message:   "rate limit exceeded",
		Retryable: false,
		Timestamp: time.Now(),
	}
)

// NewCacheError creates a new cache error
func NewCacheError(code, message, operation string, cause error) *CacheError {
	return &CacheError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Retryable: isRetryableError(code),
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// NewCacheErrorWithKey creates a new cache error with a key
func NewCacheErrorWithKey(code, message, operation, key string, cause error) *CacheError {
	return &CacheError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Key:       key,
		Retryable: isRetryableError(code),
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// isRetryableError determines if an error code represents a retryable error
func isRetryableError(code string) bool {
	switch code {
	case ErrorCodeConnectionFailed, ErrorCodeTimeout, ErrorCodeServerError:
		return true
	default:
		return false
	}
}

// Metrics represents cache metrics
type Metrics interface {
	// Operation counters
	IncrementOperations(operation string, status string)
	IncrementConnectionAttempts()
	IncrementConnectionFailures()

	// Timing metrics
	RecordOperationDuration(operation string, duration time.Duration)
	RecordConnectionDuration(duration time.Duration)

	// Gauge metrics
	SetConnectedClients(count int)
	SetMemoryUsage(bytes int64)
	SetActiveConnections(count int)

	// Cache-specific metrics
	IncrementCacheHits()
	IncrementCacheMisses()
	IncrementCacheEvictions()

	// Session metrics
	IncrementSessionCreated()
	IncrementSessionDeleted()
	SetActiveSessions(count int64)

	// Rate limit metrics
	IncrementRateLimitHits()
	IncrementRateLimitMisses()

	// Circuit breaker metrics
	IncrementCircuitBreakerEvents(event string)
}
