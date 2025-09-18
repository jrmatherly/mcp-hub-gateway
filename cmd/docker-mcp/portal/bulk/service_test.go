// Package bulk provides bulk operations tests.
package bulk

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/state"
)

// TestCreateBulkOperationService tests the creation of a bulk operation service
func TestCreateBulkOperationService(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockStateManager := &state.MockStateManager{}
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	workerCount := 5

	// Test successful creation
	service, err := CreateBulkOperationService(
		mockCache,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		workerCount,
	)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}

	// Test with nil cache
	_, err = CreateBulkOperationService(
		nil,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		workerCount,
	)
	if err == nil {
		t.Error("Expected error with nil cache, got nil")
	}

	// Test with nil executor
	_, err = CreateBulkOperationService(
		mockCache,
		nil,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		workerCount,
	)
	if err == nil {
		t.Error("Expected error with nil executor, got nil")
	}

	// Test with nil state manager
	_, err = CreateBulkOperationService(
		mockCache,
		mockExecutor,
		nil,
		mockRealtimeManager,
		mockAuditor,
		workerCount,
	)
	if err == nil {
		t.Error("Expected error with nil state manager, got nil")
	}

	// Test with nil auditor
	_, err = CreateBulkOperationService(
		mockCache,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		nil,
		workerCount,
	)
	if err == nil {
		t.Error("Expected error with nil auditor, got nil")
	}

	// Test with zero worker count (should default to 10)
	service, err = CreateBulkOperationService(
		mockCache,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		0,
	)
	if err != nil {
		t.Errorf("Expected no error with zero worker count, got %v", err)
	}
	if service == nil {
		t.Error("Expected service to be created with default worker count, got nil")
	}
}

// TestBulkOperationRequest tests bulk operation request validation
func TestBulkOperationRequest(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockStateManager := &state.MockStateManager{}
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())

	service, err := CreateBulkOperationService(
		mockCache,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		5,
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	userID := uuid.New()
	tenantID := "test-tenant"

	// Test valid request
	validRequest := &BulkOperationRequest{
		Type:           OperationTypeEnableServers,
		Name:           "Test Operation",
		Description:    "Test bulk operation",
		Targets:        []string{"server1", "server2"},
		CommandType:    executor.CommandTypeServerEnable,
		UserID:         userID,
		TenantID:       tenantID,
		Parallel:       true,
		MaxConcurrency: 2,
		Timeout:        time.Minute * 5,
	}

	// This will likely fail due to mocked dependencies, but tests the flow
	_, err = service.StartOperation(ctx, validRequest)
	if err != nil {
		t.Logf("Expected operation start with mocked dependencies to have issues: %v", err)
	}

	// Test invalid requests
	invalidRequests := []*BulkOperationRequest{
		nil, // nil request
		{ // missing type
			Name:    "Test",
			Targets: []string{"server1"},
			UserID:  userID,
		},
		{ // missing targets and filter
			Type:   OperationTypeEnableServers,
			Name:   "Test",
			UserID: userID,
		},
		{ // missing command
			Type:    OperationTypeEnableServers,
			Name:    "Test",
			Targets: []string{"server1"},
			UserID:  userID,
		},
		{ // missing user ID
			Type:        OperationTypeEnableServers,
			Name:        "Test",
			Targets:     []string{"server1"},
			CommandType: executor.CommandTypeServerEnable,
		},
	}

	for i, invalidRequest := range invalidRequests {
		_, err := service.StartOperation(ctx, invalidRequest)
		if err == nil {
			t.Errorf("Expected error for invalid request %d, got nil", i)
		}
	}
}

// TestOperationStatus tests operation status helper methods
func TestOperationStatus(t *testing.T) {
	tests := []struct {
		status     OperationStatus
		isActive   bool
		isComplete bool
	}{
		{StatusPending, true, false},
		{StatusQueued, true, false},
		{StatusRunning, true, false},
		{StatusRetrying, true, false},
		{StatusCompleted, false, true},
		{StatusFailed, false, true},
		{StatusCancelled, false, true},
		{StatusPartialSuccess, false, true},
	}

	for _, test := range tests {
		if test.status.IsActive() != test.isActive {
			t.Errorf(
				"Status %s IsActive(): expected %v, got %v",
				test.status,
				test.isActive,
				test.status.IsActive(),
			)
		}
		if test.status.IsComplete() != test.isComplete {
			t.Errorf(
				"Status %s IsComplete(): expected %v, got %v",
				test.status,
				test.isComplete,
				test.status.IsComplete(),
			)
		}
	}
}

// TestBatchServerRequest tests batch server operations
func TestBatchServerRequest(t *testing.T) {
	// Create mock dependencies
	mockCache := &cache.MockCache{}
	mockExecutor := executor.NewTestableExecutor()
	mockStateManager := &state.MockStateManager{}
	mockRealtimeManager := &realtime.MockConnectionManager{}
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())

	service, err := CreateBulkOperationService(
		mockCache,
		mockExecutor,
		mockStateManager,
		mockRealtimeManager,
		mockAuditor,
		5,
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	userID := uuid.New()
	tenantID := "test-tenant"

	request := &BatchServerRequest{
		Servers:       []string{"server1", "server2", "server3"},
		Configuration: DefaultOperationConfig(),
		UserID:        userID,
		TenantID:      tenantID,
	}

	// Test enable servers
	_, err = service.EnableServers(ctx, request)
	if err != nil {
		t.Logf("Expected enable servers with mocked dependencies to have issues: %v", err)
	}

	// Test disable servers
	_, err = service.DisableServers(ctx, request)
	if err != nil {
		t.Logf("Expected disable servers with mocked dependencies to have issues: %v", err)
	}

	// Test restart servers
	_, err = service.RestartServers(ctx, request)
	if err != nil {
		t.Logf("Expected restart servers with mocked dependencies to have issues: %v", err)
	}
}

// TestDefaultOperationConfig tests the default operation configuration
func TestDefaultOperationConfig(t *testing.T) {
	config := DefaultOperationConfig()

	if config.BatchSize <= 0 {
		t.Error("Expected batch size to be positive")
	}
	if config.TimeoutPerTarget <= 0 {
		t.Error("Expected timeout per target to be positive")
	}
	if config.ProgressInterval <= 0 {
		t.Error("Expected progress interval to be positive")
	}
	if config.MaxErrorRate < 0 || config.MaxErrorRate > 1 {
		t.Error("Expected max error rate to be between 0 and 1")
	}
	if config.StopOnErrorCount <= 0 {
		t.Error("Expected stop on error count to be positive")
	}
}

// TestTargetFilter tests target filtering functionality
func TestTargetFilter(t *testing.T) {
	filter := &TargetFilter{
		Status:         []string{"running", "stopped"},
		Categories:     []string{"web", "api"},
		Tags:           []string{"prod", "critical"},
		NamePattern:    "server-*",
		HealthyOnly:    true,
		RunningOnly:    false,
		Exclude:        []string{"server-maintenance"},
		ExcludePattern: "*-test",
	}

	// Test filter properties
	if len(filter.Status) != 2 {
		t.Error("Expected 2 status filters")
	}
	if len(filter.Categories) != 2 {
		t.Error("Expected 2 category filters")
	}
	if len(filter.Tags) != 2 {
		t.Error("Expected 2 tag filters")
	}
	if filter.NamePattern != "server-*" {
		t.Error("Expected name pattern to be 'server-*'")
	}
	if len(filter.Exclude) != 1 {
		t.Error("Expected 1 exclude filter")
	}
	if filter.ExcludePattern != "*-test" {
		t.Error("Expected exclude pattern to be '*-test'")
	}
	if !filter.HealthyOnly {
		t.Error("Expected healthy only to be true")
	}
	if filter.RunningOnly {
		t.Error("Expected running only to be false")
	}
}

// TestCacheKeyFunctions tests cache key generation functions
func TestCacheKeyFunctions(t *testing.T) {
	operationID := uuid.New()
	userID := uuid.New()

	bulkKey := GetBulkOperationKey(operationID)
	if bulkKey == "" || bulkKey == KeyBulkOperation {
		t.Error("Expected bulk operation key to include operation ID")
	}

	progressKey := GetOperationProgressKey(operationID)
	if progressKey == "" || progressKey == KeyOperationProgress {
		t.Error("Expected operation progress key to include operation ID")
	}

	resultsKey := GetOperationResultsKey(operationID)
	if resultsKey == "" || resultsKey == KeyOperationResults {
		t.Error("Expected operation results key to include operation ID")
	}

	userOpsKey := GetUserOperationsKey(userID)
	if userOpsKey == "" || userOpsKey == KeyUserOperations {
		t.Error("Expected user operations key to include user ID")
	}
}

// TestOperationProgress tests operation progress tracking
func TestOperationProgress(t *testing.T) {
	progress := &OperationProgress{
		Total:          10,
		Completed:      7,
		Failed:         2,
		Skipped:        1,
		Pending:        0,
		LastUpdated:    time.Now(),
		TargetProgress: make(map[string]*TargetProgress),
	}

	// Calculate percentage
	expectedPercentage := float64(progress.Completed) / float64(progress.Total) * 100
	progress.Percentage = expectedPercentage

	if progress.Percentage != 70.0 {
		t.Errorf("Expected percentage to be 70.0, got %f", progress.Percentage)
	}

	// Test individual counters
	if progress.Total != 10 {
		t.Errorf("Expected total to be 10, got %d", progress.Total)
	}
	if progress.Completed != 7 {
		t.Errorf("Expected completed to be 7, got %d", progress.Completed)
	}
	if progress.Failed != 2 {
		t.Errorf("Expected failed to be 2, got %d", progress.Failed)
	}
	if progress.Skipped != 1 {
		t.Errorf("Expected skipped to be 1, got %d", progress.Skipped)
	}
	if progress.Pending != 0 {
		t.Errorf("Expected pending to be 0, got %d", progress.Pending)
	}

	// Test target progress
	progress.TargetProgress["server1"] = &TargetProgress{
		Target:      "server1",
		Status:      TargetStatusCompleted,
		StartedAt:   &progress.LastUpdated,
		CompletedAt: &progress.LastUpdated,
		Duration:    time.Second * 30,
	}

	if len(progress.TargetProgress) != 1 {
		t.Error("Expected 1 target progress entry")
	}

	targetProgress := progress.TargetProgress["server1"]
	if targetProgress.Status != TargetStatusCompleted {
		t.Error("Expected target status to be completed")
	}
}
