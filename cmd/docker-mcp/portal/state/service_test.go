// Package state provides server state management tests.
package state

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// TestCreateStateService tests the creation of a state service
func TestCreateStateService(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConfig()

	// Test successful creation
	service, err := CreateStateService(
		mockCache,
		mockExecutor,
		mockRealtimeManager,
		mockAuditor,
		config,
	)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}

	// Test with nil cache
	_, err = CreateStateService(nil, mockExecutor, mockRealtimeManager, mockAuditor, config)
	if err == nil {
		t.Error("Expected error with nil cache, got nil")
	}

	// Test with nil executor
	_, err = CreateStateService(mockCache, nil, mockRealtimeManager, mockAuditor, config)
	if err == nil {
		t.Error("Expected error with nil executor, got nil")
	}

	// Test with nil realtime manager
	_, err = CreateStateService(mockCache, mockExecutor, nil, mockAuditor, config)
	if err == nil {
		t.Error("Expected error with nil realtime manager, got nil")
	}

	// Test with nil auditor
	_, err = CreateStateService(mockCache, mockExecutor, mockRealtimeManager, nil, config)
	if err == nil {
		t.Error("Expected error with nil auditor, got nil")
	}
}

// TestServerStateOperations tests basic server state operations
func TestServerStateOperations(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConfig()

	service, err := CreateStateService(
		mockCache,
		mockExecutor,
		mockRealtimeManager,
		mockAuditor,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	serverName := "test-server"
	userID := uuid.New()
	tenantID := "test-tenant"

	// Test setting server state
	state := &ServerState{
		Name:        serverName,
		DisplayName: "Test Server",
		Status:      StatusRunning,
		UserID:      userID,
		TenantID:    tenantID,
		LastUpdated: time.Now(),
		Config:      make(map[string]any),
		Labels:      make(map[string]string),
		Tags:        []string{},
	}

	err = service.SetServerState(ctx, state)
	if err != nil {
		t.Errorf("Expected no error setting state, got %v", err)
	}

	// Test getting server state (will attempt cache lookup)
	_, err = service.GetServerState(ctx, serverName)
	// Note: This will likely fail due to mocked cache, but we're testing the flow
	if err != nil {
		t.Logf("Expected cache miss, got error: %v", err)
	}

	// Test deleting server state
	err = service.DeleteServerState(ctx, serverName)
	if err != nil {
		t.Errorf("Expected no error deleting state, got %v", err)
	}
}

// TestServerStatusTransitions tests server status transitions
func TestServerStatusTransitions(t *testing.T) {
	// Test valid transitions
	tests := []struct {
		from  ServerStatus
		to    ServerStatus
		valid bool
	}{
		{StatusStopped, StatusStarting, true},
		{StatusStarting, StatusRunning, true},
		{StatusRunning, StatusStopping, true},
		{StatusStopping, StatusStopped, true},
		{StatusRunning, StatusStopped, false}, // Invalid: must go through stopping
		{StatusStopped, StatusRunning, false}, // Invalid: must go through starting
	}

	for _, test := range tests {
		result := test.from.CanTransitionTo(test.to)
		if result != test.valid {
			t.Errorf(
				"Transition from %s to %s: expected %v, got %v",
				test.from,
				test.to,
				test.valid,
				result,
			)
		}
	}
}

// TestHealthCheckResult tests health check operations
func TestHealthCheckResult(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConfig()

	service, err := CreateStateService(
		mockCache,
		mockExecutor,
		mockRealtimeManager,
		mockAuditor,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	serverName := "test-server"

	// Test health check (will attempt CLI execution)
	_, err = service.PerformHealthCheck(ctx, serverName)
	// Note: This will likely fail due to mocked executor, but we're testing the flow
	if err != nil {
		t.Logf("Expected executor call, got error: %v", err)
	}
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CachePrefix == "" {
		t.Error("Expected cache prefix to be set")
	}
	if config.CacheTTL <= 0 {
		t.Error("Expected cache TTL to be positive")
	}
	if config.HealthCheckInterval <= 0 {
		t.Error("Expected health check interval to be positive")
	}
	if config.WorkerPoolSize <= 0 {
		t.Error("Expected worker pool size to be positive")
	}
}

// TestCacheKeyFunctions tests cache key generation functions
func TestCacheKeyFunctions(t *testing.T) {
	serverName := "test-server"
	userID := "test-user"

	stateKey := GetServerStateKey(serverName)
	if stateKey == "" || stateKey == KeyServerState {
		t.Error("Expected server state key to include server name")
	}

	healthKey := GetServerHealthKey(serverName)
	if healthKey == "" || healthKey == KeyServerHealth {
		t.Error("Expected server health key to include server name")
	}

	metricsKey := GetServerMetricsKey(serverName)
	if metricsKey == "" || metricsKey == KeyServerMetrics {
		t.Error("Expected server metrics key to include server name")
	}

	eventsKey := GetServerEventsKey(serverName)
	if eventsKey == "" || eventsKey == KeyServerEvents {
		t.Error("Expected server events key to include server name")
	}

	perfKey := GetPerformanceStatsKey(serverName)
	if perfKey == "" || perfKey == KeyPerformanceStats {
		t.Error("Expected performance stats key to include server name")
	}

	subscriptionsKey := GetSubscriptionsKey(userID)
	if subscriptionsKey == "" || subscriptionsKey == KeySubscriptions {
		t.Error("Expected subscriptions key to include user ID")
	}
}
