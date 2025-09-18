package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
	"github.com/redis/go-redis/v9"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client         redis.UniversalClient
	config         *config.RedisConfig
	keyPrefix      string
	circuitBreaker CircuitBreaker
	metrics        Metrics
	mu             sync.RWMutex
	cleanupStop    chan struct{}
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg *config.RedisConfig) (*RedisCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis configuration is required")
	}

	// Create Redis client options
	opts := &redis.UniversalOptions{
		Addrs:           cfg.Addrs,
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: cfg.MaxIdleTime,
		PoolTimeout:     cfg.PoolTimeout,
	}

	// Create Redis client
	client := redis.NewUniversalClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisCache{
		client:      client,
		config:      cfg,
		keyPrefix:   "mcp:",
		cleanupStop: make(chan struct{}),
	}

	// Start session cleanup routine
	go cache.cleanupRoutine()

	return cache, nil
}

// formatKey adds the key prefix to a key
func (c *RedisCache) formatKey(key string) string {
	if strings.HasPrefix(key, c.keyPrefix) {
		return key
	}
	return c.keyPrefix + key
}

// Get retrieves a value from the cache
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	key = c.formatKey(key)

	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to get value",
			"GET",
			key,
			err,
		)
	}

	return val, nil
}

// Set stores a value in the cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	key = c.formatKey(key)

	if ttl <= 0 {
		ttl = c.config.SessionTTL // Use session TTL as default
	}

	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return NewCacheErrorWithKey(ErrorCodeServerError, "failed to set value", "SET", key, err)
	}

	return nil
}

// Delete removes a value from the cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	key = c.formatKey(key)

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to delete value",
			"DELETE",
			key,
			err,
		)
	}

	return nil
}

// Exists checks if a key exists in the cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	key = c.formatKey(key)

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to check existence",
			"EXISTS",
			key,
			err,
		)
	}

	return count > 0, nil
}

// MultiGet retrieves multiple values from the cache
func (c *RedisCache) MultiGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Format keys
	formattedKeys := make([]string, len(keys))
	keyMap := make(map[string]string) // formatted -> original
	for i, key := range keys {
		formatted := c.formatKey(key)
		formattedKeys[i] = formatted
		keyMap[formatted] = key
	}

	// Get values
	values, err := c.client.MGet(ctx, formattedKeys...).Result()
	if err != nil {
		return nil, NewCacheError(
			ErrorCodeServerError,
			"failed to get multiple values",
			"MGET",
			err,
		)
	}

	// Build result map
	result := make(map[string][]byte)
	for i, val := range values {
		if val != nil {
			originalKey := keyMap[formattedKeys[i]]
			if str, ok := val.(string); ok {
				result[originalKey] = []byte(str)
			}
		}
	}

	return result, nil
}

// MultiSet stores multiple values in the cache
func (c *RedisCache) MultiSet(ctx context.Context, items map[string]CacheItem) error {
	if len(items) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for key, item := range items {
		key = c.formatKey(key)
		ttl := item.TTL
		if ttl <= 0 {
			ttl = c.config.SessionTTL
		}
		pipe.Set(ctx, key, item.Value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return NewCacheError(ErrorCodeServerError, "failed to set multiple values", "MSET", err)
	}

	return nil
}

// MultiDelete removes multiple values from the cache
func (c *RedisCache) MultiDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Format keys
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	err := c.client.Del(ctx, formattedKeys...).Err()
	if err != nil {
		return NewCacheError(ErrorCodeServerError, "failed to delete multiple values", "MDEL", err)
	}

	return nil
}

// Keys returns all keys matching a pattern
func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	pattern = c.formatKey(pattern)

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, NewCacheError(ErrorCodeServerError, "failed to get keys", "KEYS", err)
	}

	// Remove prefix from keys
	prefix := c.keyPrefix
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, strings.TrimPrefix(key, prefix))
	}

	return result, nil
}

// DeletePattern deletes all keys matching a pattern
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) (int, error) {
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	err = c.MultiDelete(ctx, keys)
	if err != nil {
		return 0, err
	}

	return len(keys), nil
}

// Increment increments a counter
func (c *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	key = c.formatKey(key)

	val, err := c.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to increment",
			"INCR",
			key,
			err,
		)
	}

	return val, nil
}

// Decrement decrements a counter
func (c *RedisCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	key = c.formatKey(key)

	val, err := c.client.DecrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to decrement",
			"DECR",
			key,
			err,
		)
	}

	return val, nil
}

// TTL returns the TTL of a key
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	key = c.formatKey(key)

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, NewCacheErrorWithKey(ErrorCodeServerError, "failed to get TTL", "TTL", key, err)
	}

	if ttl < 0 {
		return 0, ErrKeyNotFound
	}

	return ttl, nil
}

// Expire sets the expiration of a key
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	key = c.formatKey(key)

	ok, err := c.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return NewCacheErrorWithKey(
			ErrorCodeServerError,
			"failed to set expiration",
			"EXPIRE",
			key,
			err,
		)
	}

	if !ok {
		return ErrKeyNotFound
	}

	return nil
}

// Health checks the health of the cache
func (c *RedisCache) Health(ctx context.Context) error {
	err := c.client.Ping(ctx).Err()
	if err != nil {
		return NewCacheError(ErrorCodeConnectionFailed, "cache health check failed", "PING", err)
	}
	return nil
}

// Info returns cache server information
func (c *RedisCache) Info(ctx context.Context) (*CacheInfo, error) {
	infoStr, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, NewCacheError(ErrorCodeServerError, "failed to get info", "INFO", err)
	}

	// Parse info string into CacheInfo struct
	info := &CacheInfo{
		Metadata: make(map[string]interface{}),
	}

	// Simple parsing of Redis INFO output
	lines := strings.Split(infoStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info.Metadata[key] = value
			}
		}
	}

	return info, nil
}

// FlushDB flushes the current database
func (c *RedisCache) FlushDB(ctx context.Context) error {
	err := c.client.FlushDB(ctx).Err()
	if err != nil {
		return NewCacheError(ErrorCodeServerError, "failed to flush database", "FLUSHDB", err)
	}
	return nil
}

// Close closes the cache connection
func (c *RedisCache) Close() error {
	close(c.cleanupStop)
	return c.client.Close()
}

// cleanupRoutine periodically cleans up expired sessions
func (c *RedisCache) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			c.DeleteExpiredSessions(ctx)
			cancel()
		case <-c.cleanupStop:
			return
		}
	}
}

// GetSession retrieves a session from cache
func (c *RedisCache) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, NewCacheErrorWithKey(
			ErrorCodeSerializationFailed,
			"failed to unmarshal session",
			"GET_SESSION",
			key,
			err,
		)
	}

	return &session, nil
}

// SetSession stores a session in cache
func (c *RedisCache) SetSession(ctx context.Context, session *Session) error {
	if session == nil || session.ID == uuid.Nil {
		return NewCacheError(ErrorCodeInvalidArgument, "invalid session", "SET_SESSION", nil)
	}

	key := fmt.Sprintf("session:%s", session.ID.String())

	data, err := json.Marshal(session)
	if err != nil {
		return NewCacheErrorWithKey(
			ErrorCodeSerializationFailed,
			"failed to marshal session",
			"SET_SESSION",
			key,
			err,
		)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = c.config.SessionTTL
	}

	return c.Set(ctx, key, data, ttl)
}

// DeleteSession removes a session from cache
func (c *RedisCache) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.Delete(ctx, key)
}

// RefreshSession extends the TTL of a session
func (c *RedisCache) RefreshSession(
	ctx context.Context,
	sessionID string,
	ttl time.Duration,
) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.Expire(ctx, key, ttl)
}

// GetUserSessions retrieves all sessions for a user
func (c *RedisCache) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	pattern := fmt.Sprintf("session:*")
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0)
	for _, key := range keys {
		// Extract session ID from key
		parts := strings.Split(key, ":")
		if len(parts) < 2 {
			continue
		}

		session, err := c.GetSession(ctx, parts[1])
		if err != nil {
			continue
		}

		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// DeleteUserSessions removes all sessions for a user
func (c *RedisCache) DeleteUserSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	sessions, err := c.GetUserSessions(ctx, userID)
	if err != nil {
		return 0, err
	}

	for _, session := range sessions {
		if err := c.DeleteSession(ctx, session.ID.String()); err != nil {
			return len(sessions), err
		}
	}

	return len(sessions), nil
}

// GetActiveSessions retrieves active sessions up to limit
func (c *RedisCache) GetActiveSessions(ctx context.Context, limit int) ([]*Session, error) {
	pattern := fmt.Sprintf("session:*")
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0, limit)
	for i, key := range keys {
		if i >= limit {
			break
		}

		// Extract session ID from key
		parts := strings.Split(key, ":")
		if len(parts) < 2 {
			continue
		}

		session, err := c.GetSession(ctx, parts[1])
		if err != nil {
			continue
		}

		if session.IsActive && session.ExpiresAt.After(time.Now()) {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// DeleteExpiredSessions removes expired sessions
func (c *RedisCache) DeleteExpiredSessions(ctx context.Context) (int, error) {
	pattern := fmt.Sprintf("session:*")
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, key := range keys {
		// Extract session ID from key
		parts := strings.Split(key, ":")
		if len(parts) < 2 {
			continue
		}

		session, err := c.GetSession(ctx, parts[1])
		if err != nil {
			continue
		}

		if session.ExpiresAt.Before(time.Now()) {
			if err := c.DeleteSession(ctx, session.ID.String()); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// GetSessionCount returns the total number of sessions
func (c *RedisCache) GetSessionCount(ctx context.Context) (int64, error) {
	pattern := fmt.Sprintf("session:*")
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return 0, err
	}

	return int64(len(keys)), nil
}

// GetUserSessionCount returns the number of sessions for a user
func (c *RedisCache) GetUserSessionCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	sessions, err := c.GetUserSessions(ctx, userID)
	if err != nil {
		return 0, err
	}

	return int64(len(sessions)), nil
}
