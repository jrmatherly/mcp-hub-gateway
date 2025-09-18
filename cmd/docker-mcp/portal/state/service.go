// Package state provides server state management implementation with Redis-based caching.
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// stateService implements the StateManager interface
type stateService struct {
	cache           cache.Cache
	executor        executor.Executor
	realtimeManager realtime.ConnectionManager
	auditor         audit.Logger
	config          Config

	// Internal state management
	subscribers   map[string]chan StateChangeEvent
	subscribersMu sync.RWMutex
	workerPool    chan struct{}
	stopChan      chan struct{}
	wg            sync.WaitGroup

	// Metrics tracking
	metrics   *StateMetrics
	metricsMu sync.RWMutex
}

// CreateStateService creates a new state management service
func CreateStateService(
	cache cache.Cache,
	executor executor.Executor,
	realtimeManager realtime.ConnectionManager,
	auditor audit.Logger,
	config Config,
) (StateManager, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("executor is required")
	}
	if realtimeManager == nil {
		return nil, fmt.Errorf("realtime manager is required")
	}
	if auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}

	service := &stateService{
		cache:           cache,
		executor:        executor,
		realtimeManager: realtimeManager,
		auditor:         auditor,
		config:          config,
		subscribers:     make(map[string]chan StateChangeEvent),
		workerPool:      make(chan struct{}, config.WorkerPoolSize),
		stopChan:        make(chan struct{}),
		metrics: &StateMetrics{
			LastUpdated: time.Now(),
		},
	}

	// Initialize worker pool
	for i := 0; i < config.WorkerPoolSize; i++ {
		service.workerPool <- struct{}{}
	}

	// Start background workers
	service.startBackgroundWorkers()

	return service, nil
}

// GetServerState retrieves the current state of a server
func (s *stateService) GetServerState(
	ctx context.Context,
	serverName string,
) (*ServerState, error) {
	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}

	// Try cache first
	key := GetServerStateKey(serverName)
	data, err := s.cache.Get(ctx, key)
	if err == nil && data != nil {
		var state ServerState
		if err := json.Unmarshal(data, &state); err == nil {
			s.recordCacheHit()
			return &state, nil
		}
	}

	// Cache miss - refresh from CLI
	s.recordCacheMiss()
	return s.refreshServerStateFromCLI(ctx, serverName)
}

// SetServerState stores or updates a server state
func (s *stateService) SetServerState(ctx context.Context, state *ServerState) error {
	if state == nil {
		return fmt.Errorf("state is required")
	}
	if state.Name == "" {
		return fmt.Errorf("server name is required")
	}

	// Get current state for comparison
	oldState, _ := s.GetServerState(ctx, state.Name)

	// Update state metadata
	state.LastUpdated = time.Now()
	state.StateVersion++
	state.CacheExpiry = time.Now().Add(s.config.CacheTTL)

	// Serialize and store in cache
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	key := GetServerStateKey(state.Name)
	if err := s.cache.Set(ctx, key, data, s.config.CacheTTL); err != nil {
		return fmt.Errorf("failed to store state in cache: %w", err)
	}

	// Record state change event
	if oldState != nil && oldState.Status != state.Status {
		event := &StateEvent{
			ID:         uuid.New(),
			ServerName: state.Name,
			EventType:  EventTypeStateChange,
			FromStatus: oldState.Status,
			ToStatus:   state.Status,
			Reason:     "State updated",
			UserID:     state.UserID,
			TenantID:   state.TenantID,
			Timestamp:  time.Now(),
		}
		s.recordStateEvent(ctx, event)
	}

	// Broadcast real-time update
	if s.config.EnableRealTimeUpdates {
		changeEvent := StateChangeEvent{
			Type:         realtime.EventTypeServerStatusUpdate,
			ServerName:   state.Name,
			OldState:     oldState,
			NewState:     state,
			ChangeReason: "State updated",
			UserID:       state.UserID,
			TenantID:     state.TenantID,
			Timestamp:    time.Now(),
		}
		go s.BroadcastStateChange(ctx, changeEvent)
	}

	// Audit log
	s.auditor.Log(
		ctx,
		audit.ActionUpdate,
		"server_state",
		state.Name,
		state.UserID.String(),
		map[string]any{
			"status":        state.Status,
			"state_version": state.StateVersion,
		},
	)

	s.recordStateChange()
	return nil
}

// DeleteServerState removes a server state from cache
func (s *stateService) DeleteServerState(ctx context.Context, serverName string) error {
	if serverName == "" {
		return fmt.Errorf("server name is required")
	}

	// Remove from cache
	key := GetServerStateKey(serverName)
	if err := s.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete state from cache: %w", err)
	}

	// Also clean up related cache entries
	healthKey := GetServerHealthKey(serverName)
	metricsKey := GetServerMetricsKey(serverName)
	eventsKey := GetServerEventsKey(serverName)

	keys := []string{healthKey, metricsKey, eventsKey}
	s.cache.MultiDelete(ctx, keys)

	return nil
}

// ListServerStates returns a list of server states based on filter criteria
func (s *stateService) ListServerStates(
	ctx context.Context,
	filter *StateFilter,
) ([]*ServerState, error) {
	// Get all server state keys
	pattern := GetServerStateKey("*")
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get state keys: %w", err)
	}

	// Batch get all states
	stateData, err := s.cache.MultiGet(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get states: %w", err)
	}

	var states []*ServerState
	for _, data := range stateData {
		var state ServerState
		if err := json.Unmarshal(data, &state); err != nil {
			continue // Skip malformed entries
		}

		// Apply filters
		if s.matchesFilter(&state, filter) {
			states = append(states, &state)
		}
	}

	// Sort and limit results
	s.sortStates(states, filter)
	if filter != nil && filter.Limit > 0 && len(states) > filter.Limit {
		start := filter.Offset
		if start < 0 || start >= len(states) {
			start = 0
		}
		end := start + filter.Limit
		if end > len(states) {
			end = len(states)
		}
		states = states[start:end]
	}

	return states, nil
}

// GetMultipleStates retrieves states for multiple servers
func (s *stateService) GetMultipleStates(
	ctx context.Context,
	serverNames []string,
) (map[string]*ServerState, error) {
	if len(serverNames) == 0 {
		return make(map[string]*ServerState), nil
	}

	// Build cache keys
	keys := make([]string, len(serverNames))
	for i, name := range serverNames {
		keys[i] = GetServerStateKey(name)
	}

	// Batch get from cache
	stateData, err := s.cache.MultiGet(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple states: %w", err)
	}

	// Parse results
	result := make(map[string]*ServerState, len(serverNames))
	for i, name := range serverNames {
		key := keys[i]
		if data, exists := stateData[key]; exists {
			var state ServerState
			if err := json.Unmarshal(data, &state); err == nil {
				result[name] = &state
			}
		}
	}

	return result, nil
}

// SetMultipleStates updates states for multiple servers
func (s *stateService) SetMultipleStates(ctx context.Context, states []*ServerState) error {
	if len(states) == 0 {
		return nil
	}

	// Prepare cache items
	items := make(map[string]cache.CacheItem, len(states))
	for _, state := range states {
		if state.Name == "" {
			continue
		}

		state.LastUpdated = time.Now()
		state.StateVersion++
		state.CacheExpiry = time.Now().Add(s.config.CacheTTL)

		data, err := json.Marshal(state)
		if err != nil {
			continue
		}

		key := GetServerStateKey(state.Name)
		items[key] = cache.CacheItem{
			Key:   key,
			Value: data,
			TTL:   s.config.CacheTTL,
		}
	}

	// Batch set in cache
	if err := s.cache.MultiSet(ctx, items); err != nil {
		return fmt.Errorf("failed to set multiple states: %w", err)
	}

	s.recordStateChange()
	return nil
}

// RefreshAllStates refreshes all server states from CLI
func (s *stateService) RefreshAllStates(ctx context.Context) error {
	// Get list of enabled servers from CLI
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerList,
		Args:    []string{"--json"},
		UserID:  "system",
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Parse server list
	var serverList []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &serverList); err != nil {
		return fmt.Errorf("failed to parse server list: %w", err)
	}

	// Refresh each enabled server state
	for _, server := range serverList {
		if server.Enabled {
			go func(serverName string) {
				if _, err := s.RefreshServerState(ctx, serverName); err != nil {
					// Log error but don't fail the entire operation
					fmt.Printf("Failed to refresh state for server %s: %v\n", serverName, err)
				}
			}(server.Name)
		}
	}

	return nil
}

// RefreshServerState refreshes a single server's state from CLI
func (s *stateService) RefreshServerState(
	ctx context.Context,
	serverName string,
) (*ServerState, error) {
	return s.refreshServerStateFromCLI(ctx, serverName)
}

// PerformHealthCheck performs a health check on a server
func (s *stateService) PerformHealthCheck(
	ctx context.Context,
	serverName string,
) (*HealthCheckResult, error) {
	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}

	startTime := time.Now()

	// Get server info from CLI
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerInspect,
		Args:    []string{serverName},
		UserID:  "system",
	}
	result, err := s.executor.Execute(ctx, req)

	responseTime := time.Since(startTime)

	healthResult := &HealthCheckResult{
		CheckedAt:    startTime,
		ResponseTime: responseTime,
		Endpoint:     serverName,
		Method:       "CLI inspect",
	}

	if err != nil {
		healthResult.Status = HealthStatusUnhealthy
		healthResult.ErrorMessage = err.Error()
		healthResult.Message = "Health check failed"
	} else if result.ExitCode != 0 {
		healthResult.Status = HealthStatusUnhealthy
		healthResult.ErrorMessage = result.Error
		healthResult.Message = "Health check failed"
		healthResult.StatusCode = result.ExitCode
	} else {
		healthResult.Status = HealthStatusHealthy
		healthResult.Message = "Health check passed"
		healthResult.StatusCode = 200
	}

	// Store health check result in cache
	data, _ := json.Marshal(healthResult)
	key := GetServerHealthKey(serverName)
	s.cache.Set(ctx, key, data, s.config.CacheTTL)

	s.recordHealthCheck(err == nil)
	return healthResult, nil
}

// GetHealthSummary returns overall health summary
func (s *stateService) GetHealthSummary(ctx context.Context) (*HealthSummary, error) {
	// Check cache first
	data, err := s.cache.Get(ctx, KeyHealthSummary)
	if err == nil && data != nil {
		var summary HealthSummary
		if err := json.Unmarshal(data, &summary); err == nil {
			return &summary, nil
		}
	}

	// Generate fresh summary
	states, err := s.ListServerStates(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get server states: %w", err)
	}

	summary := &HealthSummary{
		TotalServers:    len(states),
		StatusBreakdown: make(map[ServerStatus]int),
		HealthBreakdown: make(map[HealthStatus]int),
		LastUpdated:     time.Now(),
	}

	for _, state := range states {
		summary.StatusBreakdown[state.Status]++

		if state.HealthCheck != nil {
			summary.HealthBreakdown[state.HealthCheck.Status]++

			switch state.HealthCheck.Status {
			case HealthStatusHealthy:
				summary.HealthyServers++
			case HealthStatusUnhealthy:
				summary.UnhealthyServers++
			case HealthStatusDegraded:
				summary.DegradedServers++
			default:
				summary.UnknownServers++
			}
		} else {
			summary.UnknownServers++
		}
	}

	// Store in cache
	data, _ = json.Marshal(summary)
	s.cache.Set(ctx, KeyHealthSummary, data, time.Minute*5)

	return summary, nil
}

// TransitionServerState transitions a server to a new state
func (s *stateService) TransitionServerState(
	ctx context.Context,
	serverName string,
	targetState ServerStatus,
	reason string,
) error {
	if serverName == "" {
		return fmt.Errorf("server name is required")
	}

	// Get current state
	currentState, err := s.GetServerState(ctx, serverName)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Validate transition
	if !currentState.Status.CanTransitionTo(targetState) {
		return fmt.Errorf("invalid transition from %s to %s", currentState.Status, targetState)
	}

	// Update state
	currentState.Status = targetState
	currentState.LastUpdated = time.Now()

	if err := s.SetServerState(ctx, currentState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Record transition event
	event := &StateEvent{
		ID:         uuid.New(),
		ServerName: serverName,
		EventType:  EventTypeStateChange,
		FromStatus: currentState.Status,
		ToStatus:   targetState,
		Reason:     reason,
		UserID:     currentState.UserID,
		TenantID:   currentState.TenantID,
		Timestamp:  time.Now(),
	}

	return s.RecordStateEvent(ctx, event)
}

// RecordStateEvent records a state event
func (s *stateService) RecordStateEvent(ctx context.Context, event *StateEvent) error {
	return s.recordStateEvent(ctx, event)
}

// GetStateHistory returns the state history for a server
func (s *stateService) GetStateHistory(
	ctx context.Context,
	serverName string,
	limit int,
) ([]*StateEvent, error) {
	if serverName == "" {
		return nil, fmt.Errorf("server name is required")
	}

	key := GetServerEventsKey(serverName)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return []*StateEvent{}, nil // Return empty slice if not found
	}

	var events []*StateEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse state events: %w", err)
	}

	// Apply limit
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

// Helper functions

func (s *stateService) refreshServerStateFromCLI(
	ctx context.Context,
	serverName string,
) (*ServerState, error) {
	// Get server inspection data
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerInspect,
		Args:    []string{serverName},
		UserID:  "system",
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect server: %w", err)
	}

	// Parse server info
	var serverInfo map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &serverInfo); err != nil {
		return nil, fmt.Errorf("failed to parse server info: %w", err)
	}

	// Build server state
	state := &ServerState{
		Name:        serverName,
		LastUpdated: time.Now(),
		LastCheck:   time.Now(),
		LastSeen:    time.Now(),
		Status:      s.determineServerStatus(serverInfo),
		Config:      serverInfo,
	}

	// Determine status based on server info
	if enabled, ok := serverInfo["enabled"].(bool); ok && enabled {
		if running, ok := serverInfo["running"].(bool); ok && running {
			state.Status = StatusRunning
		} else {
			state.Status = StatusStopped
		}
	} else {
		state.Status = StatusStopped
	}

	// Store in cache
	if err := s.SetServerState(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to cache server state: %w", err)
	}

	return state, nil
}

func (s *stateService) determineServerStatus(serverInfo map[string]any) ServerStatus {
	if enabled, ok := serverInfo["enabled"].(bool); ok && enabled {
		if running, ok := serverInfo["running"].(bool); ok && running {
			return StatusRunning
		}
		return StatusStopped
	}
	return StatusStopped
}

func (s *stateService) matchesFilter(state *ServerState, filter *StateFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by status
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if state.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by category
	if len(filter.Categories) > 0 {
		found := false
		for _, category := range filter.Categories {
			if state.Category == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by tags
	if len(filter.Tags) > 0 {
		for _, filterTag := range filter.Tags {
			found := false
			for _, stateTag := range state.Tags {
				if stateTag == filterTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Filter by user ID
	if filter.UserID != nil && state.UserID != *filter.UserID {
		return false
	}

	// Filter by tenant ID
	if filter.TenantID != "" && state.TenantID != filter.TenantID {
		return false
	}

	// Health only filter
	if filter.HealthOnly &&
		(state.HealthCheck == nil || state.HealthCheck.Status != HealthStatusHealthy) {
		return false
	}

	return true
}

func (s *stateService) sortStates(states []*ServerState, filter *StateFilter) {
	if filter == nil || filter.SortBy == "" {
		// Default sort by name
		sort.Slice(states, func(i, j int) bool {
			return states[i].Name < states[j].Name
		})
		return
	}

	sortBy := filter.SortBy
	sortOrder := filter.SortOrder
	ascending := sortOrder != "desc"

	sort.Slice(states, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = states[i].Name < states[j].Name
		case "status":
			less = states[i].Status < states[j].Status
		case "last_updated":
			less = states[i].LastUpdated.Before(states[j].LastUpdated)
		case "priority":
			less = states[i].Priority < states[j].Priority
		default:
			less = states[i].Name < states[j].Name
		}

		if ascending {
			return less
		}
		return !less
	})
}

func (s *stateService) recordStateEvent(ctx context.Context, event *StateEvent) error {
	// Get existing events
	key := GetServerEventsKey(event.ServerName)
	data, _ := s.cache.Get(ctx, key)

	var events []*StateEvent
	if data != nil {
		json.Unmarshal(data, &events)
	}

	// Add new event
	events = append([]*StateEvent{event}, events...)

	// Limit history size
	if len(events) > s.config.MaxStateHistory {
		events = events[:s.config.MaxStateHistory]
	}

	// Store back to cache
	data, _ = json.Marshal(events)
	return s.cache.Set(ctx, key, data, s.config.StateEventTTL)
}

// Metrics tracking methods

func (s *stateService) recordCacheHit() {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	// Implementation would increment cache hit counters
}

func (s *stateService) recordCacheMiss() {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	// Implementation would increment cache miss counters
}

func (s *stateService) recordStateChange() {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metrics.TotalStateChanges++
	s.metrics.EventsProcessed++
}

func (s *stateService) recordHealthCheck(success bool) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metrics.HealthCheckCount++
	if !success {
		s.metrics.FailedHealthChecks++
	}
}

// Background workers

func (s *stateService) startBackgroundWorkers() {
	// Health check worker
	s.wg.Add(1)
	go s.healthCheckWorker()

	// Cache cleanup worker
	s.wg.Add(1)
	go s.cacheCleanupWorker()

	// Metrics update worker
	s.wg.Add(1)
	go s.metricsWorker()
}

func (s *stateService) healthCheckWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Perform periodic health checks
			ctx := context.Background()
			states, err := s.ListServerStates(ctx, nil)
			if err != nil {
				continue
			}

			for _, state := range states {
				if state.Status.IsRunning() {
					go func(serverName string) {
						s.PerformHealthCheck(ctx, serverName)
					}(state.Name)
				}
			}
		}
	}
}

func (s *stateService) cacheCleanupWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.CacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Clean up expired cache entries
			ctx := context.Background()
			s.CleanupExpiredStates(ctx, s.config.StaleStateThreshold)
		}
	}
}

func (s *stateService) metricsWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Update metrics
			s.updateMetrics()
		}
	}
}

func (s *stateService) updateMetrics() {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	s.metrics.LastUpdated = time.Now()
	// Additional metrics calculations would go here
}

// Cleanup and shutdown

func (s *stateService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// Remaining interface methods (implementation stubs for now)

func (s *stateService) GetStateMetrics(ctx context.Context) (*StateMetrics, error) {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	// Return copy of current metrics
	metrics := *s.metrics
	return &metrics, nil
}

func (s *stateService) GetServerMetrics(
	ctx context.Context,
	serverName string,
) (*ServerMetrics, error) {
	// Implementation would gather server-specific metrics
	return &ServerMetrics{
		ServerName:  serverName,
		LastUpdated: time.Now(),
	}, nil
}

func (s *stateService) UpdatePerformanceStats(
	ctx context.Context,
	serverName string,
	stats *PerformanceStats,
) error {
	key := GetPerformanceStatsKey(serverName)
	data, _ := json.Marshal(stats)
	return s.cache.Set(ctx, key, data, s.config.CacheTTL)
}

func (s *stateService) InvalidateCache(ctx context.Context, serverName string) error {
	return s.DeleteServerState(ctx, serverName)
}

func (s *stateService) WarmupCache(ctx context.Context, serverNames []string) error {
	for _, name := range serverNames {
		go s.RefreshServerState(ctx, name)
	}
	return nil
}

func (s *stateService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	// Implementation would gather cache statistics
	return &CacheStats{
		LastCleanup: time.Now(),
	}, nil
}

func (s *stateService) SubscribeToStateChanges(
	ctx context.Context,
	userID string,
) (<-chan StateChangeEvent, error) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	ch := make(chan StateChangeEvent, 100)
	s.subscribers[userID] = ch
	return ch, nil
}

func (s *stateService) UnsubscribeFromStateChanges(ctx context.Context, userID string) error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if ch, exists := s.subscribers[userID]; exists {
		close(ch)
		delete(s.subscribers, userID)
	}
	return nil
}

func (s *stateService) BroadcastStateChange(ctx context.Context, event StateChangeEvent) error {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip this subscriber
		}
	}

	// Also broadcast through realtime manager
	if s.realtimeManager != nil {
		realtimeEvent := realtime.Event{
			ID:        uuid.New(),
			Type:      event.Type,
			Channel:   realtime.ChannelServers,
			Data:      event,
			Timestamp: event.Timestamp,
		}
		return s.realtimeManager.BroadcastToChannel(realtime.ChannelServers, realtimeEvent)
	}

	return nil
}

func (s *stateService) CleanupExpiredStates(
	ctx context.Context,
	maxAge time.Duration,
) (int, error) {
	// Implementation would clean up old state entries
	return 0, nil
}

func (s *stateService) CompactStateHistory(ctx context.Context, maxEntries int) error {
	// Implementation would compact state history
	return nil
}

func (s *stateService) ExportStates(
	ctx context.Context,
	filter *StateFilter,
) ([]*ServerState, error) {
	return s.ListServerStates(ctx, filter)
}
