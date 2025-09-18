// Package state provides mock implementations for testing.
package state

import (
	"context"
	"fmt"
	"time"
)

// MockStateManager is a mock implementation of StateManager for testing
type MockStateManager struct {
	states map[string]*ServerState
}

// NewMockStateManager creates a new mock state manager
func NewMockStateManager() *MockStateManager {
	return &MockStateManager{
		states: make(map[string]*ServerState),
	}
}

func (m *MockStateManager) GetServerState(
	ctx context.Context,
	serverName string,
) (*ServerState, error) {
	if state, exists := m.states[serverName]; exists {
		return state, nil
	}
	return nil, fmt.Errorf("server state not found")
}

func (m *MockStateManager) SetServerState(ctx context.Context, state *ServerState) error {
	m.states[state.Name] = state
	return nil
}

func (m *MockStateManager) DeleteServerState(ctx context.Context, serverName string) error {
	delete(m.states, serverName)
	return nil
}

func (m *MockStateManager) ListServerStates(
	ctx context.Context,
	filter *StateFilter,
) ([]*ServerState, error) {
	states := make([]*ServerState, 0, len(m.states))
	for _, state := range m.states {
		states = append(states, state)
	}
	return states, nil
}

func (m *MockStateManager) GetMultipleStates(
	ctx context.Context,
	serverNames []string,
) (map[string]*ServerState, error) {
	result := make(map[string]*ServerState)
	for _, name := range serverNames {
		if state, exists := m.states[name]; exists {
			result[name] = state
		}
	}
	return result, nil
}

func (m *MockStateManager) SetMultipleStates(ctx context.Context, states []*ServerState) error {
	for _, state := range states {
		m.states[state.Name] = state
	}
	return nil
}

func (m *MockStateManager) RefreshAllStates(ctx context.Context) error {
	return nil // Mock implementation
}

func (m *MockStateManager) RefreshServerState(
	ctx context.Context,
	serverName string,
) (*ServerState, error) {
	return m.GetServerState(ctx, serverName)
}

func (m *MockStateManager) PerformHealthCheck(
	ctx context.Context,
	serverName string,
) (*HealthCheckResult, error) {
	return &HealthCheckResult{
		Status:       HealthStatusHealthy,
		CheckedAt:    time.Now(),
		ResponseTime: time.Millisecond * 100,
		Message:      "Mock health check",
	}, nil
}

func (m *MockStateManager) GetHealthSummary(ctx context.Context) (*HealthSummary, error) {
	return &HealthSummary{
		TotalServers:    len(m.states),
		HealthyServers:  len(m.states),
		StatusBreakdown: make(map[ServerStatus]int),
		HealthBreakdown: make(map[HealthStatus]int),
		LastUpdated:     time.Now(),
	}, nil
}

func (m *MockStateManager) TransitionServerState(
	ctx context.Context,
	serverName string,
	targetState ServerStatus,
	reason string,
) error {
	if state, exists := m.states[serverName]; exists {
		state.Status = targetState
		return nil
	}
	return fmt.Errorf("server not found")
}

func (m *MockStateManager) RecordStateEvent(ctx context.Context, event *StateEvent) error {
	return nil // Mock implementation
}

func (m *MockStateManager) GetStateHistory(
	ctx context.Context,
	serverName string,
	limit int,
) ([]*StateEvent, error) {
	return []*StateEvent{}, nil // Mock implementation
}

func (m *MockStateManager) GetStateMetrics(ctx context.Context) (*StateMetrics, error) {
	return &StateMetrics{
		LastUpdated: time.Now(),
	}, nil
}

func (m *MockStateManager) GetServerMetrics(
	ctx context.Context,
	serverName string,
) (*ServerMetrics, error) {
	return &ServerMetrics{
		ServerName:  serverName,
		LastUpdated: time.Now(),
	}, nil
}

func (m *MockStateManager) UpdatePerformanceStats(
	ctx context.Context,
	serverName string,
	stats *PerformanceStats,
) error {
	return nil // Mock implementation
}

func (m *MockStateManager) InvalidateCache(ctx context.Context, serverName string) error {
	return nil // Mock implementation
}

func (m *MockStateManager) WarmupCache(ctx context.Context, serverNames []string) error {
	return nil // Mock implementation
}

func (m *MockStateManager) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	return &CacheStats{
		LastCleanup: time.Now(),
	}, nil
}

func (m *MockStateManager) SubscribeToStateChanges(
	ctx context.Context,
	userID string,
) (<-chan StateChangeEvent, error) {
	ch := make(chan StateChangeEvent, 1)
	return ch, nil
}

func (m *MockStateManager) UnsubscribeFromStateChanges(ctx context.Context, userID string) error {
	return nil // Mock implementation
}

func (m *MockStateManager) BroadcastStateChange(ctx context.Context, event StateChangeEvent) error {
	return nil // Mock implementation
}

func (m *MockStateManager) CleanupExpiredStates(
	ctx context.Context,
	maxAge time.Duration,
) (int, error) {
	return 0, nil // Mock implementation
}

func (m *MockStateManager) CompactStateHistory(ctx context.Context, maxEntries int) error {
	return nil // Mock implementation
}

func (m *MockStateManager) ExportStates(
	ctx context.Context,
	filter *StateFilter,
) ([]*ServerState, error) {
	return m.ListServerStates(ctx, filter)
}
