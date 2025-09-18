// Package bulk provides bulk operations implementation for batch command execution.
package bulk

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/state"
)

// bulkOperationService implements the BulkOperationManager interface
type bulkOperationService struct {
	cache           cache.Cache
	executor        executor.Executor
	stateManager    state.StateManager
	realtimeManager realtime.ConnectionManager
	auditor         audit.Logger

	// Active operations tracking
	activeOperations map[uuid.UUID]*BulkOperation
	operationsMu     sync.RWMutex

	// Progress subscribers
	progressSubscribers map[uuid.UUID]map[string]chan OperationProgress
	subscribersMu       sync.RWMutex

	// Worker pool for concurrent execution
	workerPool  chan struct{}
	workerCount int

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Statistics
	statistics *OperationStatistics
	statsMu    sync.RWMutex
}

// CreateBulkOperationService creates a new bulk operation service
func CreateBulkOperationService(
	cache cache.Cache,
	executor executor.Executor,
	stateManager state.StateManager,
	realtimeManager realtime.ConnectionManager,
	auditor audit.Logger,
	workerCount int,
) (BulkOperationManager, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("executor is required")
	}
	if stateManager == nil {
		return nil, fmt.Errorf("state manager is required")
	}
	if auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	service := &bulkOperationService{
		cache:               cache,
		executor:            executor,
		stateManager:        stateManager,
		realtimeManager:     realtimeManager,
		auditor:             auditor,
		activeOperations:    make(map[uuid.UUID]*BulkOperation),
		progressSubscribers: make(map[uuid.UUID]map[string]chan OperationProgress),
		workerPool:          make(chan struct{}, workerCount),
		workerCount:         workerCount,
		stopChan:            make(chan struct{}),
		statistics: &OperationStatistics{
			OperationsByType:   make(map[OperationType]int64),
			OperationsByStatus: make(map[OperationStatus]int64),
			LastUpdated:        time.Now(),
		},
	}

	// Initialize worker pool
	for i := 0; i < workerCount; i++ {
		service.workerPool <- struct{}{}
	}

	// Start background workers
	service.startBackgroundWorkers()

	return service, nil
}

// StartOperation starts a new bulk operation
func (s *bulkOperationService) StartOperation(
	ctx context.Context,
	request *BulkOperationRequest,
) (*BulkOperation, error) {
	if request == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Validate request
	if err := s.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Resolve targets
	targets, err := s.resolveTargets(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets resolved")
	}

	// Create operation
	operation := &BulkOperation{
		ID:          uuid.New(),
		Type:        request.Type,
		Status:      StatusPending,
		Name:        request.Name,
		Description: request.Description,
		Request:     request,
		TargetCount: len(targets),
		Targets:     targets,
		Progress: &OperationProgress{
			Total:          len(targets),
			Pending:        len(targets),
			LastUpdated:    time.Now(),
			TargetProgress: make(map[string]*TargetProgress),
		},
		Results: &OperationResults{
			TargetResults:  make(map[string]*TargetResult),
			OutputByTarget: make(map[string]string),
			ErrorSummary:   make(map[string]int),
		},
		CreatedAt:         time.Now(),
		UserID:            request.UserID,
		TenantID:          request.TenantID,
		RequestID:         request.RequestID,
		Configuration:     request.Configuration,
		Metadata:          request.Metadata,
		FailOnError:       request.FailOnError,
		MaxRetries:        request.MaxRetries,
		EstimatedDuration: s.estimateOperationDuration(request.Type, len(targets)),
	}

	// Apply default configuration if not provided
	if operation.Configuration == nil {
		operation.Configuration = DefaultOperationConfig()
	}

	// Initialize target progress
	for _, target := range targets {
		operation.Progress.TargetProgress[target] = &TargetProgress{
			Target: target,
			Status: TargetStatusPending,
		}
	}

	// Store operation in cache and active operations
	if err := s.storeOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to store operation: %w", err)
	}

	s.operationsMu.Lock()
	s.activeOperations[operation.ID] = operation
	s.operationsMu.Unlock()

	// Update statistics
	s.updateStatistics(operation, true)

	// Audit log
	s.auditor.Log(
		ctx,
		audit.ActionCreate,
		"bulk_operation",
		operation.ID.String(),
		request.UserID.String(),
		map[string]any{
			"type":         operation.Type,
			"target_count": operation.TargetCount,
		},
	)

	// Start execution asynchronously
	go s.executeOperation(ctx, operation)

	return operation, nil
}

// GetOperation retrieves an operation by ID
func (s *bulkOperationService) GetOperation(
	ctx context.Context,
	operationID uuid.UUID,
) (*BulkOperation, error) {
	// Check active operations first
	s.operationsMu.RLock()
	if operation, exists := s.activeOperations[operationID]; exists {
		s.operationsMu.RUnlock()
		return s.cloneOperation(operation), nil
	}
	s.operationsMu.RUnlock()

	// Check cache
	key := GetBulkOperationKey(operationID)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}

	var operation BulkOperation
	if err := json.Unmarshal(data, &operation); err != nil {
		return nil, fmt.Errorf("failed to parse operation: %w", err)
	}

	return &operation, nil
}

// CancelOperation cancels a running operation
func (s *bulkOperationService) CancelOperation(ctx context.Context, operationID uuid.UUID) error {
	s.operationsMu.Lock()
	defer s.operationsMu.Unlock()

	operation, exists := s.activeOperations[operationID]
	if !exists {
		return fmt.Errorf("operation not found or not active")
	}

	if operation.Status.IsComplete() {
		return fmt.Errorf("operation already completed")
	}

	// Mark as cancelled
	now := time.Now()
	operation.Status = StatusCancelled
	operation.CancelledAt = &now
	operation.CancelReason = "User cancelled"
	operation.CompletedAt = &now
	operation.Duration = now.Sub(operation.CreatedAt)

	// Update progress
	operation.Progress.LastUpdated = now

	// Store updated operation
	s.storeOperation(ctx, operation)

	// Remove from active operations
	delete(s.activeOperations, operationID)

	// Audit log
	s.auditor.Log(
		ctx,
		audit.ActionUpdate,
		"bulk_operation",
		operationID.String(),
		operation.UserID.String(),
		map[string]any{
			"action":    "cancel_operation",
			"tenant_id": operation.TenantID,
		},
	)

	return nil
}

// RetryOperation retries a failed operation
func (s *bulkOperationService) RetryOperation(
	ctx context.Context,
	operationID uuid.UUID,
) (*BulkOperation, error) {
	operation, err := s.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}

	if operation.Status != StatusFailed && operation.Status != StatusPartialSuccess {
		return nil, fmt.Errorf("operation cannot be retried (status: %s)", operation.Status)
	}

	if operation.RetryCount >= operation.MaxRetries {
		return nil, fmt.Errorf("maximum retry attempts reached")
	}

	// Increment retry count
	operation.RetryCount++
	now := time.Now()
	operation.LastRetryAt = &now
	operation.Status = StatusRetrying

	// Reset failed targets for retry
	for _, progress := range operation.Progress.TargetProgress {
		if progress.Status == TargetStatusFailed {
			progress.Status = TargetStatusPending
			progress.Error = ""
			progress.RetryCount++
		}
	}

	// Update counters
	operation.Progress.Failed = 0
	operation.Progress.Pending = 0
	for _, progress := range operation.Progress.TargetProgress {
		if progress.Status == TargetStatusPending {
			operation.Progress.Pending++
		}
	}

	// Store updated operation
	s.storeOperation(ctx, operation)

	// Add back to active operations
	s.operationsMu.Lock()
	s.activeOperations[operation.ID] = operation
	s.operationsMu.Unlock()

	// Start execution
	go s.executeOperation(ctx, operation)

	return operation, nil
}

// ListOperations lists operations based on filter criteria
func (s *bulkOperationService) ListOperations(
	ctx context.Context,
	filter *OperationFilter,
) ([]*BulkOperation, error) {
	// Get all operation keys
	pattern := KeyBulkOperation + "*"
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation keys: %w", err)
	}

	// Batch get operations
	operationData, err := s.cache.MultiGet(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}

	var operations []*BulkOperation
	for _, data := range operationData {
		var operation BulkOperation
		if err := json.Unmarshal(data, &operation); err != nil {
			continue // Skip malformed entries
		}

		// Apply filters
		if s.matchesFilter(&operation, filter) {
			operations = append(operations, &operation)
		}
	}

	// Sort and limit results
	s.sortOperations(operations, filter)
	if filter != nil && filter.Limit > 0 && len(operations) > filter.Limit {
		start := filter.Offset
		if start < 0 || start >= len(operations) {
			start = 0
		}
		end := start + filter.Limit
		if end > len(operations) {
			end = len(operations)
		}
		operations = operations[start:end]
	}

	return operations, nil
}

// GetUserOperations gets operations for a specific user
func (s *bulkOperationService) GetUserOperations(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*BulkOperation, error) {
	filter := &OperationFilter{
		UserID:    &userID,
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	return s.ListOperations(ctx, filter)
}

// GetActiveOperations gets all currently active operations
func (s *bulkOperationService) GetActiveOperations(ctx context.Context) ([]*BulkOperation, error) {
	s.operationsMu.RLock()
	defer s.operationsMu.RUnlock()

	operations := make([]*BulkOperation, 0, len(s.activeOperations))
	for _, operation := range s.activeOperations {
		operations = append(operations, s.cloneOperation(operation))
	}

	return operations, nil
}

// GetOperationProgress gets the current progress of an operation
func (s *bulkOperationService) GetOperationProgress(
	ctx context.Context,
	operationID uuid.UUID,
) (*OperationProgress, error) {
	operation, err := s.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}

	// Calculate current progress percentage
	if operation.Progress.Total > 0 {
		operation.Progress.Percentage = float64(
			operation.Progress.Completed,
		) / float64(
			operation.Progress.Total,
		) * 100
	}

	// Calculate timing estimates
	if operation.StartedAt != nil && operation.Progress.Completed > 0 {
		elapsed := time.Since(*operation.StartedAt)
		operation.Progress.ElapsedTime = elapsed
		operation.Progress.AverageTimePerTarget = elapsed / time.Duration(
			operation.Progress.Completed,
		)

		if operation.Progress.Pending > 0 {
			operation.Progress.EstimatedRemaining = operation.Progress.AverageTimePerTarget * time.Duration(
				operation.Progress.Pending,
			)
		}
	}

	return operation.Progress, nil
}

// Batch operation methods

// EnableServers enables multiple servers
func (s *bulkOperationService) EnableServers(
	ctx context.Context,
	request *BatchServerRequest,
) (*BulkOperation, error) {
	bulkRequest := &BulkOperationRequest{
		Type:           OperationTypeEnableServers,
		Name:           "Enable Servers",
		Description:    fmt.Sprintf("Enable %d servers", len(request.Servers)),
		Targets:        request.Servers,
		CommandType:    executor.CommandTypeServerEnable,
		Configuration:  request.Configuration,
		Metadata:       request.Metadata,
		UserID:         request.UserID,
		TenantID:       request.TenantID,
		Parallel:       true,
		MaxConcurrency: 5,
		Timeout:        time.Minute * 10,
	}

	return s.StartOperation(ctx, bulkRequest)
}

// DisableServers disables multiple servers
func (s *bulkOperationService) DisableServers(
	ctx context.Context,
	request *BatchServerRequest,
) (*BulkOperation, error) {
	bulkRequest := &BulkOperationRequest{
		Type:           OperationTypeDisableServers,
		Name:           "Disable Servers",
		Description:    fmt.Sprintf("Disable %d servers", len(request.Servers)),
		Targets:        request.Servers,
		CommandType:    executor.CommandTypeServerDisable,
		Configuration:  request.Configuration,
		Metadata:       request.Metadata,
		UserID:         request.UserID,
		TenantID:       request.TenantID,
		Parallel:       true,
		MaxConcurrency: 5,
		Timeout:        time.Minute * 10,
	}

	return s.StartOperation(ctx, bulkRequest)
}

// RestartServers restarts multiple servers
func (s *bulkOperationService) RestartServers(
	ctx context.Context,
	request *BatchServerRequest,
) (*BulkOperation, error) {
	bulkRequest := &BulkOperationRequest{
		Type:           OperationTypeRestartServers,
		Name:           "Restart Servers",
		Description:    fmt.Sprintf("Restart %d servers", len(request.Servers)),
		Targets:        request.Servers,
		CommandType:    executor.CommandTypeServerDisable, // Will be handled as restart sequence
		Configuration:  request.Configuration,
		Metadata:       request.Metadata,
		UserID:         request.UserID,
		TenantID:       request.TenantID,
		Parallel:       false, // Sequential for restart
		MaxConcurrency: 1,
		Timeout:        time.Minute * 15,
	}

	return s.StartOperation(ctx, bulkRequest)
}

// UpdateServers performs updates on multiple servers
func (s *bulkOperationService) UpdateServers(
	ctx context.Context,
	request *BatchUpdateRequest,
) (*BulkOperation, error) {
	bulkRequest := &BulkOperationRequest{
		Type: OperationTypeUpdateServers,
		Name: "Update Servers",
		Description: fmt.Sprintf(
			"Update %d servers (%s)",
			len(request.Servers),
			request.UpdateType,
		),
		Targets:        request.Servers,
		CommandType:    executor.CommandTypeCatalogSync,
		Args:           []string{request.UpdateType},
		Configuration:  request.Configuration,
		Metadata:       request.UpdateData,
		UserID:         request.UserID,
		TenantID:       request.TenantID,
		Parallel:       true,
		MaxConcurrency: 3,
		Timeout:        time.Minute * 20,
	}

	return s.StartOperation(ctx, bulkRequest)
}

// Core execution logic

func (s *bulkOperationService) executeOperation(ctx context.Context, operation *BulkOperation) {
	// Mark as started
	now := time.Now()
	operation.StartedAt = &now
	operation.Status = StatusRunning
	operation.Progress.LastUpdated = now

	s.storeOperation(ctx, operation)
	s.broadcastProgress(operation)

	// Execute based on operation type and configuration
	if operation.Request.Parallel && operation.Request.MaxConcurrency > 1 {
		s.executeParallel(ctx, operation)
	} else {
		s.executeSequential(ctx, operation)
	}

	// Mark as completed
	now = time.Now()
	operation.CompletedAt = &now
	operation.Duration = now.Sub(operation.CreatedAt)

	// Determine final status
	if operation.Progress.Failed == 0 {
		operation.Status = StatusCompleted
	} else if operation.Progress.Completed > 0 {
		operation.Status = StatusPartialSuccess
	} else {
		operation.Status = StatusFailed
	}

	// Finalize results
	s.finalizeResults(operation)

	// Store final state
	s.storeOperation(ctx, operation)
	s.broadcastProgress(operation)

	// Remove from active operations
	s.operationsMu.Lock()
	delete(s.activeOperations, operation.ID)
	s.operationsMu.Unlock()

	// Update statistics
	s.updateStatistics(operation, false)
}

func (s *bulkOperationService) executeParallel(ctx context.Context, operation *BulkOperation) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, operation.Request.MaxConcurrency)

	for _, target := range operation.Targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			s.executeTarget(ctx, operation, target)
		}(target)
	}

	wg.Wait()
}

func (s *bulkOperationService) executeSequential(ctx context.Context, operation *BulkOperation) {
	for _, target := range operation.Targets {
		// Check for cancellation
		if operation.Status == StatusCancelled {
			break
		}

		s.executeTarget(ctx, operation, target)

		// Add delay between targets if configured
		if operation.Configuration.DelayBetweenBatches > 0 {
			time.Sleep(operation.Configuration.DelayBetweenBatches)
		}
	}
}

func (s *bulkOperationService) executeTarget(
	ctx context.Context,
	operation *BulkOperation,
	target string,
) {
	// Get target progress
	targetProgress := operation.Progress.TargetProgress[target]
	if targetProgress == nil {
		return
	}

	// Mark as running
	now := time.Now()
	targetProgress.Status = TargetStatusRunning
	targetProgress.StartedAt = &now

	operation.Progress.Pending--
	operation.Progress.LastUpdated = now

	// Periodic progress broadcast
	s.broadcastProgress(operation)

	// Execute command
	var result *TargetResult
	if operation.Type == OperationTypeRestartServers {
		result = s.executeRestart(ctx, target)
	} else {
		result = s.executeSingleCommand(ctx, operation, target)
	}

	// Update progress
	completedAt := time.Now()
	targetProgress.CompletedAt = &completedAt
	targetProgress.Duration = completedAt.Sub(*targetProgress.StartedAt)
	targetProgress.Result = result

	if result.Success {
		targetProgress.Status = TargetStatusCompleted
		operation.Progress.Completed++
		operation.Results.Successful++
		operation.Results.SuccessfulTargets = append(operation.Results.SuccessfulTargets, target)
	} else {
		targetProgress.Status = TargetStatusFailed
		targetProgress.Error = result.Error
		operation.Progress.Failed++
		operation.Results.Failed++
		operation.Results.FailedTargets = append(operation.Results.FailedTargets, target)
		operation.ErrorCount++
		operation.Errors = append(operation.Errors, fmt.Sprintf("%s: %s", target, result.Error))

		// Update error summary
		operation.Results.ErrorSummary[result.Error]++
	}

	// Store result
	operation.Results.TargetResults[target] = result
	if result.Output != "" {
		operation.Results.OutputByTarget[target] = result.Output
	}

	operation.Progress.LastUpdated = time.Now()

	// Store updated operation
	s.storeOperation(ctx, operation)

	// Check if we should stop on error
	if !operation.Configuration.ContinueOnError && !result.Success {
		operation.Status = StatusFailed
	}

	// Check error rate threshold
	if operation.Progress.Total > 0 {
		currentErrorRate := float64(
			operation.Progress.Failed,
		) / float64(
			operation.Progress.Completed+operation.Progress.Failed,
		)
		if currentErrorRate > operation.Configuration.MaxErrorRate {
			operation.Status = StatusFailed
		}
	}
}

func (s *bulkOperationService) executeSingleCommand(
	ctx context.Context,
	operation *BulkOperation,
	target string,
) *TargetResult {
	startTime := time.Now()

	// Create execution request
	req := &executor.ExecutionRequest{
		Command: operation.CommandType,
		Args:    append(operation.Args, target),
		UserID:  "system",
	}

	// Execute command
	result, err := s.executor.Execute(ctx, req)

	duration := time.Since(startTime)

	targetResult := &TargetResult{
		Target:    target,
		Success:   err == nil && result.ExitCode == 0,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	if err != nil {
		targetResult.Error = err.Error()
	} else {
		targetResult.Output = result.Stdout
		targetResult.ExitCode = result.ExitCode
		if result.ExitCode != 0 && result.Error != "" {
			targetResult.Error = result.Error
		}
	}

	return targetResult
}

func (s *bulkOperationService) executeRestart(
	ctx context.Context,
	target string,
) *TargetResult {
	startTime := time.Now()
	var outputs []string
	var lastError string

	// This function is now only used for restart operations
	// First disable the server
	disableReq := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerDisable,
		Args:    []string{target},
		UserID:  "system",
	}

	result, err := s.executor.Execute(ctx, disableReq)
	if err != nil {
		return &TargetResult{
			Target:    target,
			Success:   false,
			Duration:  time.Since(startTime),
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
	}

	if result.ExitCode != 0 {
		return &TargetResult{
			Target:    target,
			Success:   false,
			Duration:  time.Since(startTime),
			Error:     result.Error,
			Timestamp: time.Now(),
		}
	}

	outputs = append(outputs, fmt.Sprintf("Disabled: %s", result.Stdout))

	// Then enable the server
	enableReq := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerEnable,
		Args:    []string{target},
		UserID:  "system",
	}

	result, err = s.executor.Execute(ctx, enableReq)
	success := true
	if err != nil {
		success = false
		lastError = err.Error()
	} else if result.ExitCode != 0 {
		success = false
		lastError = result.Error
	} else {
		outputs = append(outputs, fmt.Sprintf("Enabled: %s", result.Stdout))
	}

	duration := time.Since(startTime)

	return &TargetResult{
		Target:    target,
		Success:   success,
		Output:    strings.Join(outputs, "\n"),
		Error:     lastError,
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

// Helper methods

func (s *bulkOperationService) validateRequest(request *BulkOperationRequest) error {
	if request.Type == "" {
		return fmt.Errorf("operation type is required")
	}

	if len(request.Targets) == 0 && request.TargetFilter == nil {
		return fmt.Errorf("targets or target filter is required")
	}

	if request.CommandType == "" {
		return fmt.Errorf("command type is required")
	}

	if request.UserID == uuid.Nil {
		return fmt.Errorf("user ID is required")
	}

	return nil
}

func (s *bulkOperationService) resolveTargets(
	ctx context.Context,
	request *BulkOperationRequest,
) ([]string, error) {
	targets := make(map[string]bool)

	// Add explicit targets
	for _, target := range request.Targets {
		targets[target] = true
	}

	// Resolve filtered targets
	if request.TargetFilter != nil {
		filteredTargets, err := s.resolveFilteredTargets(ctx, request.TargetFilter)
		if err != nil {
			return nil, err
		}
		for _, target := range filteredTargets {
			targets[target] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(targets))
	for target := range targets {
		result = append(result, target)
	}

	return result, nil
}

func (s *bulkOperationService) resolveFilteredTargets(
	ctx context.Context,
	filter *TargetFilter,
) ([]string, error) {
	// Get all server states
	stateFilter := &state.StateFilter{}

	// Apply status filter
	if len(filter.Status) > 0 {
		statuses := make([]state.ServerStatus, len(filter.Status))
		for i, status := range filter.Status {
			statuses[i] = state.ServerStatus(status)
		}
		stateFilter.Statuses = statuses
	}

	// Apply category filter
	if len(filter.Categories) > 0 {
		stateFilter.Categories = filter.Categories
	}

	// Apply tag filter
	if len(filter.Tags) > 0 {
		stateFilter.Tags = filter.Tags
	}

	// Apply health filter
	if filter.HealthyOnly {
		stateFilter.HealthOnly = true
	}

	states, err := s.stateManager.ListServerStates(ctx, stateFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get server states: %w", err)
	}

	var targets []string
	for _, state := range states {
		// Apply running filter
		if filter.RunningOnly && !state.Status.IsRunning() {
			continue
		}

		// Apply name pattern filter
		if filter.NamePattern != "" {
			if !strings.Contains(state.Name, filter.NamePattern) {
				continue
			}
		}

		// Apply exclusions
		excluded := false
		for _, exclude := range filter.Exclude {
			if state.Name == exclude {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Apply exclude pattern
		if filter.ExcludePattern != "" {
			if strings.Contains(state.Name, filter.ExcludePattern) {
				continue
			}
		}

		targets = append(targets, state.Name)
	}

	return targets, nil
}

func (s *bulkOperationService) estimateOperationDuration(
	opType OperationType,
	targetCount int,
) time.Duration {
	// Base estimates per operation type
	baseEstimates := map[OperationType]time.Duration{
		OperationTypeEnableServers:  time.Second * 30,
		OperationTypeDisableServers: time.Second * 15,
		OperationTypeRestartServers: time.Minute * 2,
		OperationTypeUpdateServers:  time.Minute * 5,
		OperationTypeHealthCheck:    time.Second * 10,
		OperationTypeCollectMetrics: time.Second * 20,
	}

	baseTime, exists := baseEstimates[opType]
	if !exists {
		baseTime = time.Minute * 2 // Default estimate
	}

	// Scale by target count with diminishing returns for parallel operations
	scaleFactor := float64(targetCount)
	if targetCount > 10 {
		scaleFactor = 10 + (float64(targetCount-10) * 0.5) // Parallel efficiency
	}

	return time.Duration(float64(baseTime) * scaleFactor)
}

func (s *bulkOperationService) storeOperation(ctx context.Context, operation *BulkOperation) error {
	data, err := json.Marshal(operation)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	key := GetBulkOperationKey(operation.ID)
	return s.cache.Set(ctx, key, data, time.Hour*24) // Store for 24 hours
}

func (s *bulkOperationService) cloneOperation(operation *BulkOperation) *BulkOperation {
	data, _ := json.Marshal(operation)
	var clone BulkOperation
	json.Unmarshal(data, &clone)
	return &clone
}

func (s *bulkOperationService) matchesFilter(
	operation *BulkOperation,
	filter *OperationFilter,
) bool {
	if filter == nil {
		return true
	}

	// Filter by types
	if len(filter.Types) > 0 {
		found := false
		for _, opType := range filter.Types {
			if operation.Type == opType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by statuses
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if operation.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by user ID
	if filter.UserID != nil && operation.UserID != *filter.UserID {
		return false
	}

	// Filter by tenant ID
	if filter.TenantID != "" && operation.TenantID != filter.TenantID {
		return false
	}

	// Filter by time range
	if filter.StartTime != nil && operation.CreatedAt.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && operation.CreatedAt.After(*filter.EndTime) {
		return false
	}

	return true
}

func (s *bulkOperationService) sortOperations(
	operations []*BulkOperation,
	filter *OperationFilter,
) {
	if filter == nil || filter.SortBy == "" {
		// Default sort by created time descending
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].CreatedAt.After(operations[j].CreatedAt)
		})
		return
	}

	sortBy := filter.SortBy
	sortOrder := filter.SortOrder
	ascending := sortOrder != "desc"

	sort.Slice(operations, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "created_at":
			less = operations[i].CreatedAt.Before(operations[j].CreatedAt)
		case "name":
			less = operations[i].Name < operations[j].Name
		case "type":
			less = operations[i].Type < operations[j].Type
		case "status":
			less = operations[i].Status < operations[j].Status
		case "target_count":
			less = operations[i].TargetCount < operations[j].TargetCount
		default:
			less = operations[i].CreatedAt.Before(operations[j].CreatedAt)
		}

		if ascending {
			return less
		}
		return !less
	})
}

func (s *bulkOperationService) finalizeResults(operation *BulkOperation) {
	results := operation.Results
	results.Total = operation.TargetCount
	results.CompletedAt = time.Now()
	results.GeneratedAt = time.Now()

	if results.Total > 0 {
		results.SuccessRate = float64(results.Successful) / float64(results.Total)
	}

	// Calculate timing statistics
	if operation.StartedAt != nil && operation.CompletedAt != nil {
		results.TotalDuration = operation.CompletedAt.Sub(*operation.StartedAt)
		if results.Total > 0 {
			results.AverageDuration = results.TotalDuration / time.Duration(results.Total)
		}
	}

	// Find fastest and slowest targets
	var fastestDuration, slowestDuration time.Duration
	for target, result := range results.TargetResults {
		if fastestDuration == 0 || result.Duration < fastestDuration {
			fastestDuration = result.Duration
			results.FastestTarget = target
		}
		if result.Duration > slowestDuration {
			slowestDuration = result.Duration
			results.SlowestTarget = target
		}
	}

	// Aggregate common errors
	errorCounts := make(map[string]int)
	for _, result := range results.TargetResults {
		if !result.Success && result.Error != "" {
			errorCounts[result.Error]++
		}
	}

	// Find most common errors
	type errorCount struct {
		error string
		count int
	}
	var errors []errorCount
	for err, count := range errorCounts {
		errors = append(errors, errorCount{err, count})
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})

	// Take top 5 common errors
	for i, err := range errors {
		if i >= 5 {
			break
		}
		results.CommonErrors = append(
			results.CommonErrors,
			fmt.Sprintf("%s (%d times)", err.error, err.count),
		)
	}
}

func (s *bulkOperationService) broadcastProgress(operation *BulkOperation) {
	// Broadcast to progress subscribers
	s.subscribersMu.RLock()
	if subscribers, exists := s.progressSubscribers[operation.ID]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- *operation.Progress:
			default:
				// Channel full, skip
			}
		}
	}
	s.subscribersMu.RUnlock()

	// Broadcast via realtime manager
	if s.realtimeManager != nil {
		event := realtime.Event{
			ID:        uuid.New(),
			Type:      realtime.EventTypeServerStatusUpdate,
			Channel:   realtime.ChannelServers,
			Data:      operation.Progress,
			Timestamp: time.Now(),
		}
		s.realtimeManager.BroadcastToChannel(realtime.ChannelServers, event)
	}
}

func (s *bulkOperationService) updateStatistics(operation *BulkOperation, starting bool) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	if starting {
		s.statistics.TotalOperations++
		s.statistics.ActiveOperations++
		s.statistics.OperationsByType[operation.Type]++
	} else {
		s.statistics.ActiveOperations--
		s.statistics.OperationsByStatus[operation.Status]++
		s.statistics.TotalTargetsProcessed += int64(operation.TargetCount)

		// Update success rate
		if s.statistics.TotalOperations > 0 {
			completed := s.statistics.OperationsByStatus[StatusCompleted] + s.statistics.OperationsByStatus[StatusPartialSuccess]
			s.statistics.SuccessRate = float64(completed) / float64(s.statistics.TotalOperations)
		}

		// Update average execution time
		if operation.Duration > 0 {
			// Simple moving average approximation
			if s.statistics.AverageExecutionTime == 0 {
				s.statistics.AverageExecutionTime = operation.Duration
			} else {
				s.statistics.AverageExecutionTime = (s.statistics.AverageExecutionTime + operation.Duration) / 2
			}
		}

		now := time.Now()
		s.statistics.LastOperationTime = &now
	}

	s.statistics.LastUpdated = time.Now()
}

// Background workers

func (s *bulkOperationService) startBackgroundWorkers() {
	// Cleanup worker
	s.wg.Add(1)
	go s.cleanupWorker()

	// Statistics update worker
	s.wg.Add(1)
	go s.statisticsWorker()
}

func (s *bulkOperationService) cleanupWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Clean up old operations
			ctx := context.Background()
			s.CleanupCompletedOperations(
				ctx,
				time.Hour*48,
			) // Clean up operations older than 48 hours
		}
	}
}

func (s *bulkOperationService) statisticsWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Store current statistics in cache
			ctx := context.Background()
			data, _ := json.Marshal(s.statistics)
			s.cache.Set(ctx, KeyOperationStats, data, time.Hour)
		}
	}
}

// Remaining interface methods (stubs for now)

func (s *bulkOperationService) SubscribeToProgress(
	ctx context.Context,
	operationID uuid.UUID,
) (<-chan OperationProgress, error) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if s.progressSubscribers[operationID] == nil {
		s.progressSubscribers[operationID] = make(map[string]chan OperationProgress)
	}

	userID := uuid.New().String() // Generate unique subscriber ID
	ch := make(chan OperationProgress, 100)
	s.progressSubscribers[operationID][userID] = ch

	return ch, nil
}

func (s *bulkOperationService) UnsubscribeFromProgress(
	ctx context.Context,
	operationID uuid.UUID,
	userID string,
) error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if subscribers, exists := s.progressSubscribers[operationID]; exists {
		if ch, exists := subscribers[userID]; exists {
			close(ch)
			delete(subscribers, userID)
		}
	}

	return nil
}

func (s *bulkOperationService) GetOperationResults(
	ctx context.Context,
	operationID uuid.UUID,
) (*OperationResults, error) {
	operation, err := s.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}
	return operation.Results, nil
}

func (s *bulkOperationService) ExportResults(
	ctx context.Context,
	operationID uuid.UUID,
	format string,
) ([]byte, error) {
	results, err := s.GetOperationResults(ctx, operationID)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(results, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *bulkOperationService) GetOperationSummary(
	ctx context.Context,
	operationID uuid.UUID,
) (*OperationSummary, error) {
	operation, err := s.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}

	summary := &OperationSummary{
		ID:           operation.ID,
		Type:         operation.Type,
		Status:       operation.Status,
		Name:         operation.Name,
		TargetCount:  operation.TargetCount,
		SuccessCount: operation.Results.Successful,
		FailureCount: operation.Results.Failed,
		Duration:     operation.Duration,
		CreatedAt:    operation.CreatedAt,
		CompletedAt:  operation.CompletedAt,
		UserID:       operation.UserID,
		TenantID:     operation.TenantID,
	}

	if operation.Progress.Total > 0 {
		summary.ProgressPercent = float64(
			operation.Progress.Completed,
		) / float64(
			operation.Progress.Total,
		) * 100
	}

	return summary, nil
}

func (s *bulkOperationService) ApplyConfiguration(
	ctx context.Context,
	request *ConfigurationBatchRequest,
) (*BulkOperation, error) {
	// Implementation would handle configuration application
	return nil, fmt.Errorf("not implemented")
}

func (s *bulkOperationService) BackupConfigurations(
	ctx context.Context,
	request *BackupRequest,
) (*BulkOperation, error) {
	// Implementation would handle configuration backup
	return nil, fmt.Errorf("not implemented")
}

func (s *bulkOperationService) RestoreConfigurations(
	ctx context.Context,
	request *RestoreRequest,
) (*BulkOperation, error) {
	// Implementation would handle configuration restore
	return nil, fmt.Errorf("not implemented")
}

func (s *bulkOperationService) PerformHealthChecks(
	ctx context.Context,
	request *HealthCheckBatchRequest,
) (*BulkOperation, error) {
	// Implementation would handle batch health checks
	return nil, fmt.Errorf("not implemented")
}

func (s *bulkOperationService) CollectMetrics(
	ctx context.Context,
	request *MetricsBatchRequest,
) (*BulkOperation, error) {
	// Implementation would handle metrics collection
	return nil, fmt.Errorf("not implemented")
}

func (s *bulkOperationService) CleanupCompletedOperations(
	ctx context.Context,
	olderThan time.Duration,
) (int, error) {
	// Implementation would clean up old operations
	return 0, nil
}

func (s *bulkOperationService) GetOperationStatistics(
	ctx context.Context,
) (*OperationStatistics, error) {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	// Return copy of current statistics
	stats := *s.statistics
	return &stats, nil
}

// Shutdown stops the service and cleans up resources
func (s *bulkOperationService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}
