// Package bulk provides bulk operations for batch command execution in the MCP Portal.
// It implements efficient multi-server operations with progress tracking, error handling,
// and real-time status updates for managing multiple MCP servers simultaneously.
package bulk

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
)

// BulkOperationManager defines the interface for bulk operations
type BulkOperationManager interface {
	// Operation lifecycle
	StartOperation(ctx context.Context, request *BulkOperationRequest) (*BulkOperation, error)
	GetOperation(ctx context.Context, operationID uuid.UUID) (*BulkOperation, error)
	CancelOperation(ctx context.Context, operationID uuid.UUID) error
	RetryOperation(ctx context.Context, operationID uuid.UUID) (*BulkOperation, error)

	// Operation queries
	ListOperations(ctx context.Context, filter *OperationFilter) ([]*BulkOperation, error)
	GetUserOperations(ctx context.Context, userID uuid.UUID, limit int) ([]*BulkOperation, error)
	GetActiveOperations(ctx context.Context) ([]*BulkOperation, error)

	// Progress tracking
	GetOperationProgress(ctx context.Context, operationID uuid.UUID) (*OperationProgress, error)
	SubscribeToProgress(
		ctx context.Context,
		operationID uuid.UUID,
	) (<-chan OperationProgress, error)
	UnsubscribeFromProgress(ctx context.Context, operationID uuid.UUID, userID string) error

	// Results and reporting
	GetOperationResults(ctx context.Context, operationID uuid.UUID) (*OperationResults, error)
	ExportResults(ctx context.Context, operationID uuid.UUID, format string) ([]byte, error)
	GetOperationSummary(ctx context.Context, operationID uuid.UUID) (*OperationSummary, error)

	// Batch operations
	EnableServers(ctx context.Context, request *BatchServerRequest) (*BulkOperation, error)
	DisableServers(ctx context.Context, request *BatchServerRequest) (*BulkOperation, error)
	RestartServers(ctx context.Context, request *BatchServerRequest) (*BulkOperation, error)
	UpdateServers(ctx context.Context, request *BatchUpdateRequest) (*BulkOperation, error)

	// Configuration operations
	ApplyConfiguration(
		ctx context.Context,
		request *ConfigurationBatchRequest,
	) (*BulkOperation, error)
	BackupConfigurations(ctx context.Context, request *BackupRequest) (*BulkOperation, error)
	RestoreConfigurations(ctx context.Context, request *RestoreRequest) (*BulkOperation, error)

	// Health and monitoring
	PerformHealthChecks(
		ctx context.Context,
		request *HealthCheckBatchRequest,
	) (*BulkOperation, error)
	CollectMetrics(ctx context.Context, request *MetricsBatchRequest) (*BulkOperation, error)

	// Cleanup and maintenance
	CleanupCompletedOperations(ctx context.Context, olderThan time.Duration) (int, error)
	GetOperationStatistics(ctx context.Context) (*OperationStatistics, error)
}

// BulkOperation represents a bulk operation with its progress and results
type BulkOperation struct {
	// Basic information
	ID          uuid.UUID       `json:"id"          redis:"id"`
	Type        OperationType   `json:"type"        redis:"type"`
	Status      OperationStatus `json:"status"      redis:"status"`
	Name        string          `json:"name"        redis:"name"`
	Description string          `json:"description" redis:"description"`

	// Request information
	Request     *BulkOperationRequest `json:"request"      redis:"request"`
	TargetCount int                   `json:"target_count" redis:"target_count"`
	Targets     []string              `json:"targets"      redis:"targets"`

	// Command information (extracted from request for easy access)
	CommandType executor.CommandType `json:"command_type" redis:"command_type"`
	Args        []string             `json:"args"         redis:"args"`

	// Progress tracking
	Progress *OperationProgress `json:"progress" redis:"progress"`
	Results  *OperationResults  `json:"results"  redis:"results"`

	// Timing information
	CreatedAt         time.Time     `json:"created_at"             redis:"created_at"`
	StartedAt         *time.Time    `json:"started_at,omitempty"   redis:"started_at"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty" redis:"completed_at"`
	Duration          time.Duration `json:"duration"               redis:"duration"`
	EstimatedDuration time.Duration `json:"estimated_duration"     redis:"estimated_duration"`

	// User and tenant context
	UserID    uuid.UUID `json:"user_id"    redis:"user_id"`
	TenantID  string    `json:"tenant_id"  redis:"tenant_id"`
	RequestID string    `json:"request_id" redis:"request_id"`

	// Configuration
	Configuration *OperationConfig `json:"configuration" redis:"configuration"`
	Metadata      map[string]any   `json:"metadata"      redis:"metadata"`

	// Error handling
	ErrorCount  int      `json:"error_count"   redis:"error_count"`
	Errors      []string `json:"errors"        redis:"errors"`
	FailOnError bool     `json:"fail_on_error" redis:"fail_on_error"`

	// Cancellation
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"  redis:"cancelled_at"`
	CancelReason string     `json:"cancel_reason,omitempty" redis:"cancel_reason"`

	// Retry information
	RetryCount  int        `json:"retry_count"             redis:"retry_count"`
	MaxRetries  int        `json:"max_retries"             redis:"max_retries"`
	LastRetryAt *time.Time `json:"last_retry_at,omitempty" redis:"last_retry_at"`
}

// OperationType represents the type of bulk operation
type OperationType string

const (
	// Server operations
	OperationTypeEnableServers  OperationType = "enable_servers"
	OperationTypeDisableServers OperationType = "disable_servers"
	OperationTypeRestartServers OperationType = "restart_servers"
	OperationTypeUpdateServers  OperationType = "update_servers"

	// Configuration operations
	OperationTypeApplyConfiguration   OperationType = "apply_configuration"
	OperationTypeBackupConfiguration  OperationType = "backup_configuration"
	OperationTypeRestoreConfiguration OperationType = "restore_configuration"

	// Health and monitoring
	OperationTypeHealthCheck    OperationType = "health_check"
	OperationTypeCollectMetrics OperationType = "collect_metrics"

	// Generic operations
	OperationTypeCustomCommand OperationType = "custom_command"
	OperationTypeBatchCommand  OperationType = "batch_command"
)

// OperationStatus represents the status of a bulk operation
type OperationStatus string

const (
	StatusPending        OperationStatus = "pending"
	StatusQueued         OperationStatus = "queued"
	StatusRunning        OperationStatus = "running"
	StatusCompleted      OperationStatus = "completed"
	StatusFailed         OperationStatus = "failed"
	StatusCancelled      OperationStatus = "cancelled"
	StatusPartialSuccess OperationStatus = "partial_success"
	StatusRetrying       OperationStatus = "retrying"
)

// IsActive returns true if the operation is in an active state
func (s OperationStatus) IsActive() bool {
	return s == StatusPending || s == StatusQueued || s == StatusRunning || s == StatusRetrying
}

// IsComplete returns true if the operation has finished
func (s OperationStatus) IsComplete() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled ||
		s == StatusPartialSuccess
}

// BulkOperationRequest represents a request for a bulk operation
type BulkOperationRequest struct {
	// Operation details
	Type        OperationType `json:"type"`
	Name        string        `json:"name"`
	Description string        `json:"description"`

	// Target specification
	Targets      []string      `json:"targets"`
	TargetFilter *TargetFilter `json:"target_filter,omitempty"`

	// Command specification
	CommandType executor.CommandType `json:"command_type,omitempty"`
	Args        []string             `json:"args,omitempty"`

	// Configuration
	Configuration *OperationConfig `json:"configuration,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`

	// Execution options
	Parallel       bool          `json:"parallel"`
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
	FailOnError    bool          `json:"fail_on_error"`

	// Retry configuration
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// User context
	UserID    uuid.UUID `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	RequestID string    `json:"request_id"`
}

// TargetFilter represents criteria for selecting operation targets
type TargetFilter struct {
	// Server selection
	Status      []string `json:"status,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	NamePattern string   `json:"name_pattern,omitempty"`

	// Health-based selection
	HealthyOnly bool `json:"healthy_only"`
	RunningOnly bool `json:"running_only"`

	// Exclusions
	Exclude        []string `json:"exclude,omitempty"`
	ExcludePattern string   `json:"exclude_pattern,omitempty"`
}

// OperationConfig represents configuration for bulk operations
type OperationConfig struct {
	// Execution settings
	BatchSize           int           `json:"batch_size"`
	DelayBetweenBatches time.Duration `json:"delay_between_batches"`
	TimeoutPerTarget    time.Duration `json:"timeout_per_target"`

	// Progress reporting
	ProgressInterval      time.Duration `json:"progress_interval"`
	EnableRealTimeUpdates bool          `json:"enable_real_time_updates"`

	// Error handling
	ContinueOnError  bool    `json:"continue_on_error"`
	MaxErrorRate     float64 `json:"max_error_rate"`
	StopOnErrorCount int     `json:"stop_on_error_count"`

	// Resource limits
	MaxMemoryUsage int64   `json:"max_memory_usage"`
	MaxCPUUsage    float64 `json:"max_cpu_usage"`

	// Rollback configuration
	EnableRollback    bool                   `json:"enable_rollback"`
	RollbackOnFailure bool                   `json:"rollback_on_failure"`
	RollbackCommands  []executor.CommandType `json:"rollback_commands,omitempty"`
}

// OperationProgress represents the progress of a bulk operation
type OperationProgress struct {
	// Overall progress
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	Failed     int     `json:"failed"`
	Skipped    int     `json:"skipped"`
	Pending    int     `json:"pending"`
	Percentage float64 `json:"percentage"`

	// Current state
	CurrentTarget string    `json:"current_target,omitempty"`
	CurrentPhase  string    `json:"current_phase"`
	LastUpdated   time.Time `json:"last_updated"`

	// Timing estimates
	ElapsedTime          time.Duration `json:"elapsed_time"`
	EstimatedRemaining   time.Duration `json:"estimated_remaining"`
	AverageTimePerTarget time.Duration `json:"average_time_per_target"`

	// Throughput metrics
	TargetsPerSecond float64 `json:"targets_per_second"`
	SuccessRate      float64 `json:"success_rate"`

	// Detailed progress
	TargetProgress map[string]*TargetProgress `json:"target_progress,omitempty"`
	PhaseProgress  map[string]*PhaseProgress  `json:"phase_progress,omitempty"`
}

// TargetProgress represents progress for a specific target
type TargetProgress struct {
	Target      string        `json:"target"`
	Status      TargetStatus  `json:"status"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	Result      *TargetResult `json:"result,omitempty"`
	Error       string        `json:"error,omitempty"`
	RetryCount  int           `json:"retry_count"`
}

// TargetStatus represents the status of a target in a bulk operation
type TargetStatus string

const (
	TargetStatusPending   TargetStatus = "pending"
	TargetStatusRunning   TargetStatus = "running"
	TargetStatusCompleted TargetStatus = "completed"
	TargetStatusFailed    TargetStatus = "failed"
	TargetStatusSkipped   TargetStatus = "skipped"
	TargetStatusRetrying  TargetStatus = "retrying"
)

// PhaseProgress represents progress for a specific operation phase
type PhaseProgress struct {
	Phase       string        `json:"phase"`
	Status      TargetStatus  `json:"status"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	Message     string        `json:"message,omitempty"`
}

// TargetResult represents the result of an operation on a specific target
type TargetResult struct {
	Target    string         `json:"target"`
	Success   bool           `json:"success"`
	Output    string         `json:"output,omitempty"`
	Error     string         `json:"error,omitempty"`
	ExitCode  int            `json:"exit_code"`
	Duration  time.Duration  `json:"duration"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// OperationResults represents the complete results of a bulk operation
type OperationResults struct {
	// Summary statistics
	Total       int     `json:"total"`
	Successful  int     `json:"successful"`
	Failed      int     `json:"failed"`
	Skipped     int     `json:"skipped"`
	SuccessRate float64 `json:"success_rate"`

	// Timing information
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	FastestTarget   string        `json:"fastest_target,omitempty"`
	SlowestTarget   string        `json:"slowest_target,omitempty"`

	// Results by target
	TargetResults     map[string]*TargetResult `json:"target_results"`
	SuccessfulTargets []string                 `json:"successful_targets"`
	FailedTargets     []string                 `json:"failed_targets"`
	SkippedTargets    []string                 `json:"skipped_targets"`

	// Error summary
	ErrorSummary   map[string]int `json:"error_summary"`
	CommonErrors   []string       `json:"common_errors"`
	CriticalErrors []string       `json:"critical_errors"`

	// Output aggregation
	CombinedOutput string            `json:"combined_output,omitempty"`
	OutputByTarget map[string]string `json:"output_by_target,omitempty"`
	Artifacts      []string          `json:"artifacts,omitempty"`

	// Metadata
	CompletedAt time.Time      `json:"completed_at"`
	GeneratedAt time.Time      `json:"generated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// OperationSummary represents a high-level summary of an operation
type OperationSummary struct {
	ID              uuid.UUID       `json:"id"`
	Type            OperationType   `json:"type"`
	Status          OperationStatus `json:"status"`
	Name            string          `json:"name"`
	TargetCount     int             `json:"target_count"`
	SuccessCount    int             `json:"success_count"`
	FailureCount    int             `json:"failure_count"`
	ProgressPercent float64         `json:"progress_percent"`
	Duration        time.Duration   `json:"duration"`
	CreatedAt       time.Time       `json:"created_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	UserID          uuid.UUID       `json:"user_id"`
	TenantID        string          `json:"tenant_id"`
}

// Batch request types

// BatchServerRequest represents a request for batch server operations
type BatchServerRequest struct {
	Servers       []string         `json:"servers"`
	Filter        *TargetFilter    `json:"filter,omitempty"`
	Configuration *OperationConfig `json:"configuration,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
	UserID        uuid.UUID        `json:"user_id"`
	TenantID      string           `json:"tenant_id"`
}

// BatchUpdateRequest represents a request for batch server updates
type BatchUpdateRequest struct {
	Servers       []string         `json:"servers"`
	UpdateType    string           `json:"update_type"`
	UpdateData    map[string]any   `json:"update_data"`
	Configuration *OperationConfig `json:"configuration,omitempty"`
	UserID        uuid.UUID        `json:"user_id"`
	TenantID      string           `json:"tenant_id"`
}

// ConfigurationBatchRequest represents a request for batch configuration operations
type ConfigurationBatchRequest struct {
	Targets     []string       `json:"targets"`
	Operation   string         `json:"operation"` // apply, backup, restore
	ConfigData  map[string]any `json:"config_data,omitempty"`
	Source      string         `json:"source,omitempty"`
	Destination string         `json:"destination,omitempty"`
	UserID      uuid.UUID      `json:"user_id"`
	TenantID    string         `json:"tenant_id"`
}

// BackupRequest represents a request for configuration backup
type BackupRequest struct {
	Targets      []string  `json:"targets"`
	BackupName   string    `json:"backup_name"`
	Description  string    `json:"description,omitempty"`
	Destination  string    `json:"destination"`
	IncludeState bool      `json:"include_state"`
	UserID       uuid.UUID `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
}

// RestoreRequest represents a request for configuration restore
type RestoreRequest struct {
	Targets      []string  `json:"targets"`
	BackupName   string    `json:"backup_name"`
	Source       string    `json:"source"`
	RestoreState bool      `json:"restore_state"`
	DryRun       bool      `json:"dry_run"`
	UserID       uuid.UUID `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
}

// HealthCheckBatchRequest represents a request for batch health checks
type HealthCheckBatchRequest struct {
	Targets    []string       `json:"targets"`
	CheckType  string         `json:"check_type"` // basic, detailed, custom
	Parameters map[string]any `json:"parameters,omitempty"`
	UserID     uuid.UUID      `json:"user_id"`
	TenantID   string         `json:"tenant_id"`
}

// MetricsBatchRequest represents a request for batch metrics collection
type MetricsBatchRequest struct {
	Targets     []string      `json:"targets"`
	MetricTypes []string      `json:"metric_types"`
	Duration    time.Duration `json:"duration"`
	Interval    time.Duration `json:"interval"`
	UserID      uuid.UUID     `json:"user_id"`
	TenantID    string        `json:"tenant_id"`
}

// Filter and query types

// OperationFilter represents filtering options for operations
type OperationFilter struct {
	Types     []OperationType   `json:"types,omitempty"`
	Statuses  []OperationStatus `json:"statuses,omitempty"`
	UserID    *uuid.UUID        `json:"user_id,omitempty"`
	TenantID  string            `json:"tenant_id,omitempty"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
	SortBy    string            `json:"sort_by,omitempty"`
	SortOrder string            `json:"sort_order,omitempty"`
}

// OperationStatistics represents statistics about bulk operations
type OperationStatistics struct {
	TotalOperations       int64                     `json:"total_operations"`
	ActiveOperations      int64                     `json:"active_operations"`
	CompletedOperations   int64                     `json:"completed_operations"`
	FailedOperations      int64                     `json:"failed_operations"`
	OperationsByType      map[OperationType]int64   `json:"operations_by_type"`
	OperationsByStatus    map[OperationStatus]int64 `json:"operations_by_status"`
	AverageExecutionTime  time.Duration             `json:"average_execution_time"`
	SuccessRate           float64                   `json:"success_rate"`
	TotalTargetsProcessed int64                     `json:"total_targets_processed"`
	LastOperationTime     *time.Time                `json:"last_operation_time,omitempty"`
	LastUpdated           time.Time                 `json:"last_updated"`
}

// Configuration and defaults

// DefaultOperationConfig returns the default operation configuration
func DefaultOperationConfig() *OperationConfig {
	return &OperationConfig{
		BatchSize:             10,
		DelayBetweenBatches:   time.Second * 1,
		TimeoutPerTarget:      time.Minute * 5,
		ProgressInterval:      time.Second * 5,
		EnableRealTimeUpdates: true,
		ContinueOnError:       true,
		MaxErrorRate:          0.5,
		StopOnErrorCount:      10,
		MaxMemoryUsage:        1024 * 1024 * 1024, // 1GB
		MaxCPUUsage:           0.8,                // 80%
		EnableRollback:        false,
		RollbackOnFailure:     false,
	}
}

// Cache key constants
const (
	KeyBulkOperation     = "bulk:operation:"
	KeyOperationProgress = "bulk:progress:"
	KeyOperationResults  = "bulk:results:"
	KeyOperationStats    = "bulk:stats"
	KeyUserOperations    = "bulk:user:"
	KeyActiveOperations  = "bulk:active"
)

// GetBulkOperationKey returns the cache key for a bulk operation
func GetBulkOperationKey(operationID uuid.UUID) string {
	return KeyBulkOperation + operationID.String()
}

// GetOperationProgressKey returns the cache key for operation progress
func GetOperationProgressKey(operationID uuid.UUID) string {
	return KeyOperationProgress + operationID.String()
}

// GetOperationResultsKey returns the cache key for operation results
func GetOperationResultsKey(operationID uuid.UUID) string {
	return KeyOperationResults + operationID.String()
}

// GetUserOperationsKey returns the cache key for user operations
func GetUserOperationsKey(userID uuid.UUID) string {
	return KeyUserOperations + userID.String()
}
