package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
)

// RedisSessionManager implements SessionManager interface using Redis
type RedisSessionManager struct {
	cache      cache.Cache
	sessionTTL time.Duration
}

// CreateRedisSessionManager creates a new Redis-based session manager
func CreateRedisSessionManager(cache cache.Cache, sessionTTL time.Duration) SessionManager {
	return &RedisSessionManager{
		cache:      cache,
		sessionTTL: sessionTTL,
	}
}

// CreateSession creates a new session
func (m *RedisSessionManager) CreateSession(
	ctx context.Context,
	user *User,
	metadata map[string]string,
) (*Session, error) {
	session := &Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		TenantID:     user.TenantID,
		ExpiresAt:    time.Now().Add(m.sessionTTL),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
		Metadata:     metadata,
	}

	// Convert to cache session
	cacheSession := &cache.Session{
		ID:           session.ID,
		UserID:       session.UserID,
		TenantID:     session.TenantID,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		CreatedAt:    session.CreatedAt,
		ExpiresAt:    session.ExpiresAt,
		LastActivity: time.Now(),
		IsActive:     session.IsActive,
		Metadata:     session.Metadata,
	}

	// Set session in cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		if err := redisCache.SetSession(ctx, cacheSession); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	} else {
		return nil, fmt.Errorf("cache does not support session operations")
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (m *RedisSessionManager) GetSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (*Session, error) {
	// Get from cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		cacheSession, err := redisCache.GetSession(ctx, sessionID.String())
		if err != nil {
			if err == cache.ErrKeyNotFound {
				return nil, ErrSessionExpired
			}
			return nil, fmt.Errorf("failed to get session: %w", err)
		}

		// Convert from cache session
		session := &Session{
			ID:           cacheSession.ID,
			UserID:       cacheSession.UserID,
			TenantID:     cacheSession.TenantID,
			AccessToken:  cacheSession.AccessToken,
			RefreshToken: cacheSession.RefreshToken,
			ExpiresAt:    cacheSession.ExpiresAt,
			CreatedAt:    cacheSession.CreatedAt,
			LastActivity: cacheSession.LastActivity,
			IsActive:     cacheSession.IsActive,
			Metadata:     cacheSession.Metadata,
		}

		// Check if expired
		if session.ExpiresAt.Before(time.Now()) {
			session.IsActive = false
			return nil, ErrSessionExpired
		}

		return session, nil
	}

	return nil, fmt.Errorf("cache does not support session operations")
}

// UpdateSession updates a session
func (m *RedisSessionManager) UpdateSession(ctx context.Context, session *Session) error {
	session.LastActivity = time.Now()

	// Convert to cache session
	cacheSession := &cache.Session{
		ID:           session.ID,
		UserID:       session.UserID,
		TenantID:     session.TenantID,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		CreatedAt:    session.CreatedAt,
		ExpiresAt:    session.ExpiresAt,
		LastActivity: session.LastActivity,
		IsActive:     session.IsActive,
		Metadata:     session.Metadata,
	}

	// Update in cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		if err := redisCache.SetSession(ctx, cacheSession); err != nil {
			return fmt.Errorf("failed to update session: %w", err)
		}
		return nil
	}

	return fmt.Errorf("cache does not support session operations")
}

// DeleteSession deletes a session
func (m *RedisSessionManager) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	// Delete from cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		if err := redisCache.DeleteSession(ctx, sessionID.String()); err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}
		return nil
	}

	return fmt.Errorf("cache does not support session operations")
}

// ValidateSession validates and refreshes a session
func (m *RedisSessionManager) ValidateSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (*Session, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if !session.IsActive || session.ExpiresAt.Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	// Update last activity
	session.LastActivity = time.Now()
	if err := m.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session activity: %w", err)
	}

	return session, nil
}

// RefreshSession extends the session TTL
func (m *RedisSessionManager) RefreshSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (*Session, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Extend expiration
	session.ExpiresAt = time.Now().Add(m.sessionTTL)
	session.LastActivity = time.Now()

	if err := m.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to refresh session: %w", err)
	}

	return session, nil
}

// GetUserSessions retrieves all sessions for a user
func (m *RedisSessionManager) GetUserSessions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*Session, error) {
	// Get from cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		cacheSessions, err := redisCache.GetUserSessions(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user sessions: %w", err)
		}

		// Convert sessions
		sessions := make([]*Session, len(cacheSessions))
		for i, cs := range cacheSessions {
			sessions[i] = &Session{
				ID:           cs.ID,
				UserID:       cs.UserID,
				TenantID:     cs.TenantID,
				AccessToken:  cs.AccessToken,
				RefreshToken: cs.RefreshToken,
				ExpiresAt:    cs.ExpiresAt,
				CreatedAt:    cs.CreatedAt,
				LastActivity: cs.LastActivity,
				IsActive:     cs.IsActive && cs.ExpiresAt.After(time.Now()),
				Metadata:     cs.Metadata,
			}
		}

		return sessions, nil
	}

	return nil, fmt.Errorf("cache does not support session operations")
}

// DeleteUserSessions deletes all sessions for a user
func (m *RedisSessionManager) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	// Delete from cache
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		_, err := redisCache.DeleteUserSessions(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to delete user sessions: %w", err)
		}
		return nil
	}

	return fmt.Errorf("cache does not support session operations")
}

// DeleteExpiredSessions removes expired sessions
func (m *RedisSessionManager) DeleteExpiredSessions(ctx context.Context) (int, error) {
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		return redisCache.DeleteExpiredSessions(ctx)
	}

	return 0, fmt.Errorf("cache does not support session operations")
}

// InvalidateByRefreshToken invalidates a session by refresh token
func (m *RedisSessionManager) InvalidateByRefreshToken(
	ctx context.Context,
	refreshToken string,
) error {
	if redisCache, ok := m.cache.(*cache.RedisCache); ok {
		// Find all sessions and check refresh tokens
		pattern := "session:*"
		keys, err := redisCache.Keys(ctx, pattern)
		if err != nil {
			return fmt.Errorf("failed to find sessions: %w", err)
		}

		for _, key := range keys {
			// Extract session ID from key
			sessionID := key
			if len(key) > 8 && key[:8] == "session:" {
				sessionID = key[8:] // Remove "session:" prefix
			}

			session, err := redisCache.GetSession(ctx, sessionID)
			if err != nil {
				continue // Skip invalid sessions
			}

			if session.RefreshToken == refreshToken {
				// Found the session, mark as inactive
				session.IsActive = false
				if err := redisCache.SetSession(ctx, session); err != nil {
					return fmt.Errorf("failed to invalidate session: %w", err)
				}
				return nil
			}
		}

		return ErrSessionExpired
	}

	return fmt.Errorf("cache does not support session operations")
}
