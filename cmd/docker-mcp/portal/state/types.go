// Package state provides server state management with Redis-based caching for the MCP Portal.
// It implements comprehensive state tracking, health monitoring, and real-time state synchronization
// across all MCP servers and gateway components with audit logging and performance optimization.
package state

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/docker"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/realtime"
)

// StateManager defines the interface for managing server states
type StateManager interface {
	// Server state operations
	GetServerState(ctx context.Context, serverName string) (*ServerState, error)
	SetServerState(ctx context.Context, state *ServerState) error
	DeleteServerState(ctx context.Context, serverName string) error
	ListServerStates(ctx context.Context, filter *StateFilter) ([]*ServerState, error)

	// Batch operations
	GetMultipleStates(ctx context.Context, serverNames []string) (map[string]*ServerState, error)
	SetMultipleStates(ctx context.Context, states []*ServerState) error
	RefreshAllStates(ctx context.Context) error

	// State monitoring and health checks
	RefreshServerState(ctx context.Context, serverName string) (*ServerState, error)
	PerformHealthCheck(ctx context.Context, serverName string) (*HealthCheckResult, error)
	GetHealthSummary(ctx context.Context) (*HealthSummary, error)

	// State transitions and events
	TransitionServerState(
		ctx context.Context,
		serverName string,
		targetState ServerStatus,
		reason string,
	) error
	RecordStateEvent(ctx context.Context, event *StateEvent) error
	GetStateHistory(ctx context.Context, serverName string, limit int) ([]*StateEvent, error)

	// Performance and metrics
	GetStateMetrics(ctx context.Context) (*StateMetrics, error)
	GetServerMetrics(ctx context.Context, serverName string) (*ServerMetrics, error)
	UpdatePerformanceStats(ctx context.Context, serverName string, stats *PerformanceStats) error

	// Cache management
	InvalidateCache(ctx context.Context, serverName string) error
	WarmupCache(ctx context.Context, serverNames []string) error
	GetCacheStats(ctx context.Context) (*CacheStats, error)

	// Real-time updates
	SubscribeToStateChanges(ctx context.Context, userID string) (<-chan StateChangeEvent, error)
	UnsubscribeFromStateChanges(ctx context.Context, userID string) error
	BroadcastStateChange(ctx context.Context, event StateChangeEvent) error

	// Cleanup and maintenance
	CleanupExpiredStates(ctx context.Context, maxAge time.Duration) (int, error)
	CompactStateHistory(ctx context.Context, maxEntries int) error
	ExportStates(ctx context.Context, filter *StateFilter) ([]*ServerState, error)
}

// ServerState represents the complete state of an MCP server
type ServerState struct {
	// Basic information
	Name        string       `json:"name"         redis:"name"`
	DisplayName string       `json:"display_name" redis:"display_name"`
	Version     string       `json:"version"      redis:"version"`
	Status      ServerStatus `json:"status"       redis:"status"`

	// Container information (if containerized)
	ContainerID    string                       `json:"container_id,omitempty"    redis:"container_id"`
	ContainerState docker.ContainerState        `json:"container_state,omitempty" redis:"container_state"`
	HealthStatus   docker.ContainerHealthStatus `json:"health_status,omitempty"   redis:"health_status"`

	// Configuration
	Config     map[string]any `json:"config"      redis:"config"`
	Arguments  []string       `json:"arguments"   redis:"arguments"`
	WorkingDir string         `json:"working_dir" redis:"working_dir"`

	// Network and connectivity
	Port      int    `json:"port,omitempty" redis:"port"`
	Protocol  string `json:"protocol"       redis:"protocol"`
	Endpoint  string `json:"endpoint"       redis:"endpoint"`
	Transport string `json:"transport"      redis:"transport"`

	// Timing information
	StartedAt   *time.Time `json:"started_at,omitempty" redis:"started_at"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty" redis:"stopped_at"`
	LastSeen    time.Time  `json:"last_seen"            redis:"last_seen"`
	LastUpdated time.Time  `json:"last_updated"         redis:"last_updated"`
	LastCheck   time.Time  `json:"last_check"           redis:"last_check"`

	// Health and performance
	HealthCheck      *HealthCheckResult `json:"health_check,omitempty"      redis:"health_check"`
	PerformanceStats *PerformanceStats  `json:"performance_stats,omitempty" redis:"performance_stats"`
	ErrorCount       int                `json:"error_count"                 redis:"error_count"`
	LastError        string             `json:"last_error,omitempty"        redis:"last_error"`

	// Metadata and tags
	Labels   map[string]string `json:"labels"   redis:"labels"`
	Tags     []string          `json:"tags"     redis:"tags"`
	Category string            `json:"category" redis:"category"`
	Priority int               `json:"priority" redis:"priority"`

	// User and tenant context
	UserID   uuid.UUID `json:"user_id"   redis:"user_id"`
	TenantID string    `json:"tenant_id" redis:"tenant_id"`

	// State tracking
	StateVersion int64     `json:"state_version"         redis:"state_version"`
	CacheExpiry  time.Time `json:"cache_expiry"          redis:"cache_expiry"`
	IsStale      bool      `json:"is_stale"              redis:"is_stale"`
	StaleSince   time.Time `json:"stale_since,omitempty" redis:"stale_since"`
}

// ServerStatus represents the operational status of a server
type ServerStatus string

const (
	StatusUnknown      ServerStatus = "unknown"
	StatusInitializing ServerStatus = "initializing"
	StatusStarting     ServerStatus = "starting"
	StatusRunning      ServerStatus = "running"
	StatusStopping     ServerStatus = "stopping"
	StatusStopped      ServerStatus = "stopped"
	StatusError        ServerStatus = "error"
	StatusMaintenance  ServerStatus = "maintenance"
	StatusUpdating     ServerStatus = "updating"
	StatusPaused       ServerStatus = "paused"
	StatusRestarting   ServerStatus = "restarting"
)

// IsRunning returns true if the server is in a running state
func (s ServerStatus) IsRunning() bool {
	return s == StatusRunning
}

// IsHealthy returns true if the server is in a healthy state
func (s ServerStatus) IsHealthy() bool {
	return s == StatusRunning || s == StatusStarting
}

// CanTransitionTo checks if transition to target status is valid
func (s ServerStatus) CanTransitionTo(target ServerStatus) bool {
	validTransitions := map[ServerStatus][]ServerStatus{
		StatusUnknown:      {StatusInitializing, StatusStopped, StatusError},
		StatusInitializing: {StatusStarting, StatusError, StatusStopped},
		StatusStarting:     {StatusRunning, StatusError, StatusStopped},
		StatusRunning: {
			StatusStopping,
			StatusPaused,
			StatusRestarting,
			StatusError,
			StatusMaintenance,
		},
		StatusStopping:    {StatusStopped, StatusError},
		StatusStopped:     {StatusStarting, StatusError},
		StatusError:       {StatusStarting, StatusStopped, StatusMaintenance},
		StatusMaintenance: {StatusStarting, StatusStopped},
		StatusUpdating:    {StatusRunning, StatusError, StatusStopped},
		StatusPaused:      {StatusRunning, StatusStopped, StatusError},
		StatusRestarting:  {StatusRunning, StatusError, StatusStopped},
	}

	allowed, exists := validTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowed {
		if allowed == target {
			return true
		}
	}
	return false
}

// StateEvent represents a state change event
type StateEvent struct {
	ID         uuid.UUID      `json:"id"                 redis:"id"`
	ServerName string         `json:"server_name"        redis:"server_name"`
	EventType  StateEventType `json:"event_type"         redis:"event_type"`
	FromStatus ServerStatus   `json:"from_status"        redis:"from_status"`
	ToStatus   ServerStatus   `json:"to_status"          redis:"to_status"`
	Reason     string         `json:"reason"             redis:"reason"`
	Message    string         `json:"message"            redis:"message"`
	Metadata   map[string]any `json:"metadata"           redis:"metadata"`
	UserID     uuid.UUID      `json:"user_id"            redis:"user_id"`
	TenantID   string         `json:"tenant_id"          redis:"tenant_id"`
	Timestamp  time.Time      `json:"timestamp"          redis:"timestamp"`
	Duration   time.Duration  `json:"duration,omitempty" redis:"duration"`
}

// StateEventType represents the type of state event
type StateEventType string

const (
	EventTypeStateChange   StateEventType = "state_change"
	EventTypeHealthCheck   StateEventType = "health_check"
	EventTypePerformance   StateEventType = "performance"
	EventTypeError         StateEventType = "error"
	EventTypeConfiguration StateEventType = "configuration"
	EventTypeStartup       StateEventType = "startup"
	EventTypeShutdown      StateEventType = "shutdown"
	EventTypeMaintenance   StateEventType = "maintenance"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status       HealthStatus   `json:"status"                  redis:"status"`
	CheckedAt    time.Time      `json:"checked_at"              redis:"checked_at"`
	ResponseTime time.Duration  `json:"response_time"           redis:"response_time"`
	Message      string         `json:"message"                 redis:"message"`
	Details      map[string]any `json:"details"                 redis:"details"`
	Endpoint     string         `json:"endpoint"                redis:"endpoint"`
	Method       string         `json:"method"                  redis:"method"`
	StatusCode   int            `json:"status_code,omitempty"   redis:"status_code"`
	ErrorMessage string         `json:"error_message,omitempty" redis:"error_message"`
}

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// PerformanceStats represents performance statistics for a server
type PerformanceStats struct {
	CPUUsage       float64       `json:"cpu_usage"        redis:"cpu_usage"`
	MemoryUsage    float64       `json:"memory_usage"     redis:"memory_usage"`
	MemoryLimit    int64         `json:"memory_limit"     redis:"memory_limit"`
	NetworkRxBytes int64         `json:"network_rx_bytes" redis:"network_rx_bytes"`
	NetworkTxBytes int64         `json:"network_tx_bytes" redis:"network_tx_bytes"`
	DiskRead       int64         `json:"disk_read"        redis:"disk_read"`
	DiskWrite      int64         `json:"disk_write"       redis:"disk_write"`
	RequestCount   int64         `json:"request_count"    redis:"request_count"`
	ErrorRate      float64       `json:"error_rate"       redis:"error_rate"`
	ResponseTime   time.Duration `json:"response_time"    redis:"response_time"`
	Uptime         time.Duration `json:"uptime"           redis:"uptime"`
	LastUpdated    time.Time     `json:"last_updated"     redis:"last_updated"`
}

// StateFilter represents filtering options for server states
type StateFilter struct {
	Statuses   []ServerStatus `json:"statuses,omitempty"`
	Categories []string       `json:"categories,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	UserID     *uuid.UUID     `json:"user_id,omitempty"`
	TenantID   string         `json:"tenant_id,omitempty"`
	HealthOnly bool           `json:"health_only"`
	Limit      int            `json:"limit,omitempty"`
	Offset     int            `json:"offset,omitempty"`
	SortBy     string         `json:"sort_by,omitempty"`
	SortOrder  string         `json:"sort_order,omitempty"`
}

// HealthSummary represents overall health summary
type HealthSummary struct {
	TotalServers     int                  `json:"total_servers"`
	HealthyServers   int                  `json:"healthy_servers"`
	UnhealthyServers int                  `json:"unhealthy_servers"`
	DegradedServers  int                  `json:"degraded_servers"`
	UnknownServers   int                  `json:"unknown_servers"`
	StatusBreakdown  map[ServerStatus]int `json:"status_breakdown"`
	HealthBreakdown  map[HealthStatus]int `json:"health_breakdown"`
	LastUpdated      time.Time            `json:"last_updated"`
	AlertCount       int                  `json:"alert_count"`
	CriticalIssues   []string             `json:"critical_issues"`
	Recommendations  []string             `json:"recommendations"`
}

// StateMetrics represents overall state management metrics
type StateMetrics struct {
	TotalStateChanges   int64         `json:"total_state_changes"`
	StateChangesPerHour float64       `json:"state_changes_per_hour"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	ErrorRate           float64       `json:"error_rate"`
	HealthCheckCount    int64         `json:"health_check_count"`
	FailedHealthChecks  int64         `json:"failed_health_checks"`
	LastUpdated         time.Time     `json:"last_updated"`
	ActiveSubscriptions int           `json:"active_subscriptions"`
	EventsProcessed     int64         `json:"events_processed"`
}

// ServerMetrics represents metrics for a specific server
type ServerMetrics struct {
	ServerName          string              `json:"server_name"`
	StateChanges        int64               `json:"state_changes"`
	HealthChecks        int64               `json:"health_checks"`
	FailedHealthChecks  int64               `json:"failed_health_checks"`
	AverageResponseTime time.Duration       `json:"average_response_time"`
	ErrorRate           float64             `json:"error_rate"`
	UptimePercentage    float64             `json:"uptime_percentage"`
	LastHealthCheck     *time.Time          `json:"last_health_check,omitempty"`
	PerformanceHistory  []*PerformanceStats `json:"performance_history"`
	RecentEvents        []*StateEvent       `json:"recent_events"`
	LastUpdated         time.Time           `json:"last_updated"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	HitRate           float64       `json:"hit_rate"`
	MissRate          float64       `json:"miss_rate"`
	TotalOperations   int64         `json:"total_operations"`
	AverageLatency    time.Duration `json:"average_latency"`
	StoredKeys        int64         `json:"stored_keys"`
	ExpiredKeys       int64         `json:"expired_keys"`
	MemoryUsage       int64         `json:"memory_usage"`
	LastCleanup       time.Time     `json:"last_cleanup"`
	CleanupOperations int64         `json:"cleanup_operations"`
}

// StateChangeEvent represents a real-time state change event
type StateChangeEvent struct {
	Type         realtime.EventType `json:"type"`
	ServerName   string             `json:"server_name"`
	OldState     *ServerState       `json:"old_state,omitempty"`
	NewState     *ServerState       `json:"new_state"`
	ChangeReason string             `json:"change_reason"`
	UserID       uuid.UUID          `json:"user_id"`
	TenantID     string             `json:"tenant_id"`
	Timestamp    time.Time          `json:"timestamp"`
	Metadata     map[string]any     `json:"metadata,omitempty"`
}

// Config represents configuration for state management
type Config struct {
	// Cache settings
	CachePrefix          string        `json:"cache_prefix"           yaml:"cache_prefix"`
	CacheTTL             time.Duration `json:"cache_ttl"              yaml:"cache_ttl"`
	CacheCleanupInterval time.Duration `json:"cache_cleanup_interval" yaml:"cache_cleanup_interval"`

	// Health check settings
	HealthCheckInterval    time.Duration `json:"health_check_interval"    yaml:"health_check_interval"`
	HealthCheckTimeout     time.Duration `json:"health_check_timeout"     yaml:"health_check_timeout"`
	HealthCheckRetries     int           `json:"health_check_retries"     yaml:"health_check_retries"`
	HealthCheckConcurrency int           `json:"health_check_concurrency" yaml:"health_check_concurrency"`

	// State management settings
	StateRefreshInterval time.Duration `json:"state_refresh_interval" yaml:"state_refresh_interval"`
	MaxStateHistory      int           `json:"max_state_history"      yaml:"max_state_history"`
	StateEventTTL        time.Duration `json:"state_event_ttl"        yaml:"state_event_ttl"`
	StaleStateThreshold  time.Duration `json:"stale_state_threshold"  yaml:"stale_state_threshold"`

	// Performance settings
	MaxConcurrentOperations int `json:"max_concurrent_operations" yaml:"max_concurrent_operations"`
	BatchSize               int `json:"batch_size"                yaml:"batch_size"`
	WorkerPoolSize          int `json:"worker_pool_size"          yaml:"worker_pool_size"`

	// Real-time settings
	EnableRealTimeUpdates bool          `json:"enable_real_time_updates" yaml:"enable_real_time_updates"`
	MaxSubscriptions      int           `json:"max_subscriptions"        yaml:"max_subscriptions"`
	SubscriptionTTL       time.Duration `json:"subscription_ttl"         yaml:"subscription_ttl"`

	// Monitoring settings
	MetricsEnabled         bool          `json:"metrics_enabled"          yaml:"metrics_enabled"`
	MetricsInterval        time.Duration `json:"metrics_interval"         yaml:"metrics_interval"`
	PerformanceHistorySize int           `json:"performance_history_size" yaml:"performance_history_size"`
}

// DefaultConfig returns the default configuration for state management
func DefaultConfig() Config {
	return Config{
		CachePrefix:             "mcp:state:",
		CacheTTL:                15 * time.Minute,
		CacheCleanupInterval:    5 * time.Minute,
		HealthCheckInterval:     30 * time.Second,
		HealthCheckTimeout:      10 * time.Second,
		HealthCheckRetries:      3,
		HealthCheckConcurrency:  5,
		StateRefreshInterval:    1 * time.Minute,
		MaxStateHistory:         100,
		StateEventTTL:           24 * time.Hour,
		StaleStateThreshold:     5 * time.Minute,
		MaxConcurrentOperations: 10,
		BatchSize:               50,
		WorkerPoolSize:          5,
		EnableRealTimeUpdates:   true,
		MaxSubscriptions:        1000,
		SubscriptionTTL:         1 * time.Hour,
		MetricsEnabled:          true,
		MetricsInterval:         1 * time.Minute,
		PerformanceHistorySize:  100,
	}
}

// Cache key constants
const (
	KeyServerState      = "server:state:"
	KeyServerHealth     = "server:health:"
	KeyServerMetrics    = "server:metrics:"
	KeyServerEvents     = "server:events:"
	KeyStateMetrics     = "metrics:state"
	KeyHealthSummary    = "health:summary"
	KeySubscriptions    = "subscriptions:"
	KeyPerformanceStats = "performance:"
)

// GetServerStateKey returns the cache key for a server state
func GetServerStateKey(serverName string) string {
	return KeyServerState + serverName
}

// GetServerHealthKey returns the cache key for server health
func GetServerHealthKey(serverName string) string {
	return KeyServerHealth + serverName
}

// GetServerMetricsKey returns the cache key for server metrics
func GetServerMetricsKey(serverName string) string {
	return KeyServerMetrics + serverName
}

// GetServerEventsKey returns the cache key for server events
func GetServerEventsKey(serverName string) string {
	return KeyServerEvents + serverName
}

// GetPerformanceStatsKey returns the cache key for performance stats
func GetPerformanceStatsKey(serverName string) string {
	return KeyPerformanceStats + serverName
}

// GetSubscriptionsKey returns the cache key for user subscriptions
func GetSubscriptionsKey(userID string) string {
	return KeySubscriptions + userID
}
