package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockCache is a mock implementation of the Cache interface
type MockCache struct {
	mock.Mock
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string][]byte),
	}
}

// Get retrieves a value from the cache
func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

// Set stores a value in the cache
func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Delete removes a value from the cache
func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Exists checks if a key exists in the cache
func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// MultiGet retrieves multiple values from the cache
func (m *MockCache) MultiGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

// MultiSet stores multiple values in the cache
func (m *MockCache) MultiSet(ctx context.Context, items map[string]CacheItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

// MultiDelete removes multiple values from the cache
func (m *MockCache) MultiDelete(ctx context.Context, keys []string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

// Keys returns keys matching a pattern
func (m *MockCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// DeletePattern removes keys matching a pattern
func (m *MockCache) DeletePattern(ctx context.Context, pattern string) (int, error) {
	args := m.Called(ctx, pattern)
	return args.Int(0), args.Error(1)
}

// Increment increments a counter value
func (m *MockCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	args := m.Called(ctx, key, delta)
	return args.Get(0).(int64), args.Error(1)
}

// Decrement decrements a counter value
func (m *MockCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	args := m.Called(ctx, key, delta)
	return args.Get(0).(int64), args.Error(1)
}

// TTL returns the time to live for a key
func (m *MockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

// Expire sets the expiration for a key
func (m *MockCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

// Health checks cache health
func (m *MockCache) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Info returns cache information
func (m *MockCache) Info(ctx context.Context) (*CacheInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CacheInfo), args.Error(1)
}

// FlushDB flushes the cache database
func (m *MockCache) FlushDB(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSessionCache is a mock implementation of the SessionCache interface
type MockSessionCache struct {
	mock.Mock
}

// GetSession retrieves a session
func (m *MockSessionCache) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

// SetSession stores a session
func (m *MockSessionCache) SetSession(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

// DeleteSession removes a session
func (m *MockSessionCache) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// RefreshSession refreshes a session's TTL
func (m *MockSessionCache) RefreshSession(
	ctx context.Context,
	sessionID string,
	ttl time.Duration,
) error {
	args := m.Called(ctx, sessionID, ttl)
	return args.Error(0)
}

// GetUserSessions retrieves all sessions for a user
func (m *MockSessionCache) GetUserSessions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

// DeleteUserSessions removes all sessions for a user
func (m *MockSessionCache) DeleteUserSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// GetActiveSessions retrieves active sessions
func (m *MockSessionCache) GetActiveSessions(ctx context.Context, limit int) ([]*Session, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

// DeleteExpiredSessions removes expired sessions
func (m *MockSessionCache) DeleteExpiredSessions(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// GetSessionCount returns the total number of sessions
func (m *MockSessionCache) GetSessionCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// GetUserSessionCount returns the number of sessions for a user
func (m *MockSessionCache) GetUserSessionCount(
	ctx context.Context,
	userID uuid.UUID,
) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockRateLimitCache is a mock implementation of the RateLimitCache interface
type MockRateLimitCache struct {
	mock.Mock
}

// GetRateLimit retrieves rate limit state
func (m *MockRateLimitCache) GetRateLimit(
	ctx context.Context,
	key string,
) (*RateLimitState, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RateLimitState), args.Error(1)
}

// IncrementRateLimit increments rate limit counter
func (m *MockRateLimitCache) IncrementRateLimit(
	ctx context.Context,
	key string,
	window time.Duration,
	limit int,
) (*RateLimitState, error) {
	args := m.Called(ctx, key, window, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RateLimitState), args.Error(1)
}

// ResetRateLimit resets rate limit counter
func (m *MockRateLimitCache) ResetRateLimit(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// ConsumeTokens consumes tokens from token bucket
func (m *MockRateLimitCache) ConsumeTokens(
	ctx context.Context,
	key string,
	tokens int,
	capacity int,
	refillRate time.Duration,
) (*TokenBucketState, error) {
	args := m.Called(ctx, key, tokens, capacity, refillRate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenBucketState), args.Error(1)
}

// GetTokenBucket retrieves token bucket state
func (m *MockRateLimitCache) GetTokenBucket(
	ctx context.Context,
	key string,
) (*TokenBucketState, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenBucketState), args.Error(1)
}

// Common cache error for testing
var (
	ErrNotFound       = errors.New("key not found")
	ErrConnectionFail = errors.New("connection failed")
)

// DefaultTTL is the default cache TTL for testing
const DefaultTTL = 15 * time.Minute

// CreateRedisCache creates a mock Redis cache for testing
func CreateRedisCache(url string, ttl time.Duration) (Cache, error) {
	return NewMockCache(), nil
}
