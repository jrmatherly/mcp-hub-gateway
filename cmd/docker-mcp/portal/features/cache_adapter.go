package features

import (
	"context"
	"fmt"
	"time"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
)

// CacheAdapter adapts the portal cache interface to the features cache interface
type CacheAdapter struct {
	cache cache.Cache
}

// CreateCacheAdapter creates a new cache adapter
func CreateCacheAdapter(c cache.Cache) CacheProvider {
	return &CacheAdapter{cache: c}
}

// Get retrieves a flag value from cache
func (a *CacheAdapter) Get(ctx context.Context, key string) (*FlagValue, error) {
	// Try to get as bytes first
	_, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// In a real implementation, we would deserialize the bytes to FlagValue
	// For now, return a mock value
	return &FlagValue{
		Name:        FlagOAuthEnabled,
		Type:        FlagTypeBoolean,
		Enabled:     true,
		Value:       true,
		Reason:      "cache_hit",
		EvaluatedAt: time.Now(),
		CacheHit:    true,
	}, nil
}

// Set stores a flag value in cache
func (a *CacheAdapter) Set(
	ctx context.Context,
	key string,
	value *FlagValue,
	ttl time.Duration,
) error {
	// In a real implementation, we would serialize the FlagValue to bytes
	// For now, just call the underlying cache with mock data
	mockData := []byte(`{"name":"mock","enabled":true}`)
	return a.cache.Set(ctx, key, mockData, ttl)
}

// Delete removes a flag value from cache
func (a *CacheAdapter) Delete(ctx context.Context, key string) error {
	return a.cache.Delete(ctx, key)
}

// Clear clears the entire cache
func (a *CacheAdapter) Clear(ctx context.Context) error {
	return a.cache.FlushDB(ctx)
}

// Stats returns cache statistics
func (a *CacheAdapter) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := a.cache.Info(ctx)
	if err != nil {
		return nil, err
	}

	// Convert CacheInfo to generic map
	stats := map[string]interface{}{
		"version":           info.Version,
		"uptime":            info.Uptime,
		"used_memory":       info.UsedMemory,
		"connected_clients": info.ConnectedClients,
		"hit_rate":          info.HitRate,
		"total_keys":        info.TotalKeys,
	}

	return stats, nil
}

// SimpleCacheAdapter provides a minimal cache implementation for testing
type SimpleCacheAdapter struct {
	data map[string]*FlagValue
}

// CreateSimpleCacheAdapter creates a simple in-memory cache adapter
func CreateSimpleCacheAdapter() CacheProvider {
	return &SimpleCacheAdapter{
		data: make(map[string]*FlagValue),
	}
}

// Get retrieves a flag value from memory
func (s *SimpleCacheAdapter) Get(ctx context.Context, key string) (*FlagValue, error) {
	if value, exists := s.data[key]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

// Set stores a flag value in memory
func (s *SimpleCacheAdapter) Set(
	ctx context.Context,
	key string,
	value *FlagValue,
	ttl time.Duration,
) error {
	s.data[key] = value
	return nil
}

// Delete removes a flag value from memory
func (s *SimpleCacheAdapter) Delete(ctx context.Context, key string) error {
	delete(s.data, key)
	return nil
}

// Clear clears all cached values
func (s *SimpleCacheAdapter) Clear(ctx context.Context) error {
	s.data = make(map[string]*FlagValue)
	return nil
}

// Stats returns simple cache statistics
func (s *SimpleCacheAdapter) Stats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"keys_count": len(s.data),
		"type":       "simple_memory",
	}, nil
}
