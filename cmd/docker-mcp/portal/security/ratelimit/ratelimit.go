package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RateLimiter interface for rate limiting operations
type RateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID, command string) bool
	AllowUser(ctx context.Context, userID uuid.UUID) bool
	AllowCommand(ctx context.Context, command string) bool
	Reset(ctx context.Context, userID uuid.UUID)
	GetStatus(ctx context.Context, userID uuid.UUID) *LimitStatus
}

// LimitStatus represents the current rate limit status
type LimitStatus struct {
	UserID          uuid.UUID      `json:"user_id"`
	RequestsUsed    int            `json:"requests_used"`
	RequestsLimit   int            `json:"requests_limit"`
	WindowDuration  time.Duration  `json:"window_duration"`
	WindowResetTime time.Time      `json:"window_reset_time"`
	CommandLimits   map[string]int `json:"command_limits,omitempty"`
	IsBlocked       bool           `json:"is_blocked"`
	BlockedUntil    *time.Time     `json:"blocked_until,omitempty"`
}

// Config holds rate limiter configuration
type Config struct {
	// Global limits
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	BurstSize         int           `json:"burst_size"`
	WindowDuration    time.Duration `json:"window_duration"`

	// Per-command limits
	CommandLimits map[string]int `json:"command_limits"`

	// User-specific settings
	PremiumUserMultiplier float64       `json:"premium_user_multiplier"`
	BlockDuration         time.Duration `json:"block_duration"`
}

// DefaultConfig returns a default rate limiter configuration
func DefaultConfig() *Config {
	return &Config{
		RequestsPerMinute: 30,
		RequestsPerHour:   1000,
		BurstSize:         10,
		WindowDuration:    time.Minute,
		CommandLimits: map[string]int{
			"server_enable":  10, // Per minute
			"server_disable": 10,
			"bulk_operation": 5,
			"config_update":  20,
			"secret_set":     10,
		},
		PremiumUserMultiplier: 2.0,
		BlockDuration:         time.Minute * 15,
	}
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	config         *Config
	buckets        map[uuid.UUID]*tokenBucket
	commandBuckets map[string]*tokenBucket
	blockedUsers   map[uuid.UUID]time.Time
	mu             sync.RWMutex
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *Config) *TokenBucketLimiter {
	if config == nil {
		config = DefaultConfig()
	}

	limiter := &TokenBucketLimiter{
		config:         config,
		buckets:        make(map[uuid.UUID]*tokenBucket),
		commandBuckets: make(map[string]*tokenBucket),
		blockedUsers:   make(map[uuid.UUID]time.Time),
	}

	// Initialize command buckets
	for command, limit := range config.CommandLimits {
		limiter.commandBuckets[command] = &tokenBucket{
			tokens:     float64(limit),
			maxTokens:  float64(limit),
			refillRate: float64(limit) / 60.0, // Per second
			lastRefill: time.Now(),
		}
	}

	// Start cleanup goroutine
	go limiter.cleanupBlocked()

	return limiter
}

// Allow checks if a user can execute a specific command
func (l *TokenBucketLimiter) Allow(ctx context.Context, userID uuid.UUID, command string) bool {
	// Check if user is blocked
	if l.isBlocked(userID) {
		return false
	}

	// Check user rate limit
	if !l.AllowUser(ctx, userID) {
		l.blockUser(userID)
		return false
	}

	// Check command-specific rate limit
	if !l.AllowCommand(ctx, command) {
		return false
	}

	return true
}

// AllowUser checks if a user has available tokens
func (l *TokenBucketLimiter) AllowUser(ctx context.Context, userID uuid.UUID) bool {
	l.mu.Lock()
	bucket, exists := l.buckets[userID]
	if !exists {
		bucket = l.createUserBucket()
		l.buckets[userID] = bucket
	}
	l.mu.Unlock()

	return bucket.consume(1)
}

// AllowCommand checks if a command can be executed
func (l *TokenBucketLimiter) AllowCommand(ctx context.Context, command string) bool {
	l.mu.RLock()
	bucket, exists := l.commandBuckets[command]
	l.mu.RUnlock()

	if !exists {
		// No specific limit for this command
		return true
	}

	return bucket.consume(1)
}

// Reset resets the rate limit for a user
func (l *TokenBucketLimiter) Reset(ctx context.Context, userID uuid.UUID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.buckets, userID)
	delete(l.blockedUsers, userID)
}

// GetStatus returns the current rate limit status for a user
func (l *TokenBucketLimiter) GetStatus(ctx context.Context, userID uuid.UUID) *LimitStatus {
	l.mu.RLock()
	bucket, exists := l.buckets[userID]
	blockedUntil, isBlocked := l.blockedUsers[userID]
	l.mu.RUnlock()

	status := &LimitStatus{
		UserID:          userID,
		RequestsLimit:   l.config.RequestsPerMinute,
		WindowDuration:  l.config.WindowDuration,
		WindowResetTime: time.Now().Add(l.config.WindowDuration),
		CommandLimits:   l.config.CommandLimits,
		IsBlocked:       isBlocked && time.Now().Before(blockedUntil),
	}

	if status.IsBlocked {
		status.BlockedUntil = &blockedUntil
	}

	if exists {
		bucket.mu.Lock()
		status.RequestsUsed = int(bucket.maxTokens - bucket.tokens)
		bucket.mu.Unlock()
	}

	return status
}

// createUserBucket creates a new token bucket for a user
func (l *TokenBucketLimiter) createUserBucket() *tokenBucket {
	return &tokenBucket{
		tokens:     float64(l.config.BurstSize),
		maxTokens:  float64(l.config.RequestsPerMinute),
		refillRate: float64(l.config.RequestsPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

// consume attempts to consume tokens from the bucket
func (b *tokenBucket) consume(tokens float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	// Check if we have enough tokens
	if b.tokens >= tokens {
		b.tokens -= tokens
		return true
	}

	return false
}

// isBlocked checks if a user is currently blocked
func (l *TokenBucketLimiter) isBlocked(userID uuid.UUID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	blockedUntil, exists := l.blockedUsers[userID]
	if !exists {
		return false
	}

	return time.Now().Before(blockedUntil)
}

// blockUser blocks a user for the configured duration
func (l *TokenBucketLimiter) blockUser(userID uuid.UUID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.blockedUsers[userID] = time.Now().Add(l.config.BlockDuration)
}

// cleanupBlocked periodically removes expired blocks
func (l *TokenBucketLimiter) cleanupBlocked() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for userID, blockedUntil := range l.blockedUsers {
			if now.After(blockedUntil) {
				delete(l.blockedUsers, userID)
			}
		}
		l.mu.Unlock()
	}
}

// FixedWindowLimiter implements fixed window rate limiting
type FixedWindowLimiter struct {
	config  *Config
	windows map[uuid.UUID]*fixedWindow
	mu      sync.RWMutex
}

// fixedWindow represents a fixed time window for rate limiting
type fixedWindow struct {
	count       int
	windowStart time.Time
	limit       int
	duration    time.Duration
	mu          sync.Mutex
}

// NewFixedWindowLimiter creates a new fixed window rate limiter
func NewFixedWindowLimiter(config *Config) *FixedWindowLimiter {
	if config == nil {
		config = DefaultConfig()
	}

	return &FixedWindowLimiter{
		config:  config,
		windows: make(map[uuid.UUID]*fixedWindow),
	}
}

// Allow checks if a request is allowed
func (f *FixedWindowLimiter) Allow(ctx context.Context, userID uuid.UUID, command string) bool {
	f.mu.Lock()
	window, exists := f.windows[userID]
	if !exists {
		window = &fixedWindow{
			count:       0,
			windowStart: time.Now(),
			limit:       f.config.RequestsPerMinute,
			duration:    f.config.WindowDuration,
		}
		f.windows[userID] = window
	}
	f.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	// Check if we need to reset the window
	if time.Now().Sub(window.windowStart) > window.duration {
		window.count = 0
		window.windowStart = time.Now()
	}

	// Check if we're within the limit
	if window.count < window.limit {
		window.count++
		return true
	}

	return false
}

// AllowUser checks user-specific rate limit
func (f *FixedWindowLimiter) AllowUser(ctx context.Context, userID uuid.UUID) bool {
	return f.Allow(ctx, userID, "")
}

// AllowCommand checks command-specific rate limit
func (f *FixedWindowLimiter) AllowCommand(ctx context.Context, command string) bool {
	// Fixed window limiter doesn't have per-command limits
	return true
}

// Reset resets the window for a user
func (f *FixedWindowLimiter) Reset(ctx context.Context, userID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.windows, userID)
}

// GetStatus returns the current status
func (f *FixedWindowLimiter) GetStatus(ctx context.Context, userID uuid.UUID) *LimitStatus {
	f.mu.RLock()
	window, exists := f.windows[userID]
	f.mu.RUnlock()

	status := &LimitStatus{
		UserID:          userID,
		RequestsLimit:   f.config.RequestsPerMinute,
		WindowDuration:  f.config.WindowDuration,
		WindowResetTime: time.Now().Add(f.config.WindowDuration),
	}

	if exists {
		window.mu.Lock()
		status.RequestsUsed = window.count
		status.WindowResetTime = window.windowStart.Add(window.duration)
		window.mu.Unlock()
	}

	return status
}

// MockLimiter provides a mock rate limiter for testing
type MockLimiter struct {
	AllowResponse  bool
	StatusResponse *LimitStatus
	mu             sync.Mutex
	callCount      map[string]int
}

// NewMockLimiter creates a new mock limiter
func NewMockLimiter(allowAll bool) *MockLimiter {
	return &MockLimiter{
		AllowResponse: allowAll,
		callCount:     make(map[string]int),
		StatusResponse: &LimitStatus{
			RequestsUsed:  0,
			RequestsLimit: 100,
		},
	}
}

// Allow always returns the configured response
func (m *MockLimiter) Allow(ctx context.Context, userID uuid.UUID, command string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["Allow"]++
	return m.AllowResponse
}

// AllowUser always returns the configured response
func (m *MockLimiter) AllowUser(ctx context.Context, userID uuid.UUID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["AllowUser"]++
	return m.AllowResponse
}

// AllowCommand always returns the configured response
func (m *MockLimiter) AllowCommand(ctx context.Context, command string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["AllowCommand"]++
	return m.AllowResponse
}

// Reset does nothing in the mock
func (m *MockLimiter) Reset(ctx context.Context, userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["Reset"]++
}

// GetStatus returns the configured status
func (m *MockLimiter) GetStatus(ctx context.Context, userID uuid.UUID) *LimitStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount["GetStatus"]++
	return m.StatusResponse
}

// GetCallCount returns the number of times a method was called
func (m *MockLimiter) GetCallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount[method]
}
