// Package docker provides Docker container lifecycle management for the MCP Portal.
// It implements container operations, monitoring, and state management through secure
// CLI command execution with comprehensive validation and audit logging.
package docker

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ContainerState represents the state of a Docker container
type ContainerState string

const (
	ContainerStateCreated    ContainerState = "created"
	ContainerStateRunning    ContainerState = "running"
	ContainerStatePaused     ContainerState = "paused"
	ContainerStateRestarting ContainerState = "restarting"
	ContainerStateRemoving   ContainerState = "removing"
	ContainerStateExited     ContainerState = "exited"
	ContainerStateDead       ContainerState = "dead"
)

// ContainerHealthStatus represents the health status of a container
type ContainerHealthStatus string

const (
	HealthStatusHealthy   ContainerHealthStatus = "healthy"
	HealthStatusUnhealthy ContainerHealthStatus = "unhealthy"
	HealthStatusStarting  ContainerHealthStatus = "starting"
	HealthStatusNone      ContainerHealthStatus = "none"
)

// RestartPolicy represents container restart policy
type RestartPolicy string

const (
	RestartPolicyNo            RestartPolicy = "no"
	RestartPolicyAlways        RestartPolicy = "always"
	RestartPolicyOnFailure     RestartPolicy = "on-failure"
	RestartPolicyUnlessStopped RestartPolicy = "unless-stopped"
)

// Container represents a Docker container managed by the MCP Portal
type Container struct {
	// Basic information
	ID      string `json:"id"       db:"id"`
	Name    string `json:"name"     db:"name"`
	Image   string `json:"image"    db:"image"`
	ImageID string `json:"image_id" db:"image_id"`

	// State information
	State        ContainerState        `json:"state"         db:"state"`
	Status       string                `json:"status"        db:"status"`
	HealthStatus ContainerHealthStatus `json:"health_status" db:"health_status"`
	ExitCode     int                   `json:"exit_code"     db:"exit_code"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"  db:"created_at"`
	StartedAt  *time.Time `json:"started_at"  db:"started_at"`
	FinishedAt *time.Time `json:"finished_at" db:"finished_at"`

	// Configuration
	Command     []string          `json:"command"     db:"command"`
	Args        []string          `json:"args"        db:"args"`
	Environment map[string]string `json:"environment" db:"environment"`
	WorkingDir  string            `json:"working_dir" db:"working_dir"`

	// Network and ports
	Ports       []PortBinding `json:"ports"        db:"ports"`
	NetworkMode string        `json:"network_mode" db:"network_mode"`
	IPAddress   string        `json:"ip_address"   db:"ip_address"`

	// Resource limits
	CPUShares   int64  `json:"cpu_shares"   db:"cpu_shares"`
	Memory      int64  `json:"memory"       db:"memory"`
	MemorySwap  int64  `json:"memory_swap"  db:"memory_swap"`
	CPUQuota    int64  `json:"cpu_quota"    db:"cpu_quota"`
	CPUPeriod   int64  `json:"cpu_period"   db:"cpu_period"`
	BlkioWeight uint16 `json:"blkio_weight" db:"blkio_weight"`

	// Volume mounts
	Mounts []MountPoint `json:"mounts" db:"mounts"`

	// Labels and metadata
	Labels map[string]string `json:"labels" db:"labels"`

	// Restart policy
	RestartPolicy RestartPolicy `json:"restart_policy" db:"restart_policy"`
	MaxRetries    int           `json:"max_retries"    db:"max_retries"`

	// MCP specific
	ServerID  *uuid.UUID `json:"server_id"  db:"server_id"`
	CatalogID *uuid.UUID `json:"catalog_id" db:"catalog_id"`
	UserID    uuid.UUID  `json:"user_id"    db:"user_id"`
	TenantID  string     `json:"tenant_id"  db:"tenant_id"`
	IsManaged bool       `json:"is_managed" db:"is_managed"`

	// Runtime stats
	Stats *ContainerStats `json:"stats,omitempty"`
}

// PortBinding represents a port binding between host and container
type PortBinding struct {
	HostIP        string `json:"host_ip"        db:"host_ip"`
	HostPort      string `json:"host_port"      db:"host_port"`
	ContainerPort string `json:"container_port" db:"container_port"`
	Protocol      string `json:"protocol"       db:"protocol"` // tcp, udp
}

// MountPoint represents a volume mount
type MountPoint struct {
	Type        string `json:"type"        db:"type"`        // bind, volume, tmpfs
	Source      string `json:"source"      db:"source"`      // host path or volume name
	Destination string `json:"destination" db:"destination"` // container path
	Mode        string `json:"mode"        db:"mode"`        // ro, rw
	Propagation string `json:"propagation" db:"propagation"` // shared, slave, private
}

// ContainerStats represents runtime statistics for a container
type ContainerStats struct {
	// CPU stats
	CPUUsage   float64 `json:"cpu_usage"`
	CPUPercent float64 `json:"cpu_percent"`
	SystemCPU  uint64  `json:"system_cpu"`
	OnlineCPUs uint32  `json:"online_cpus"`

	// Memory stats
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryCache   uint64  `json:"memory_cache"`
	MemorySwap    uint64  `json:"memory_swap"`

	// Network stats
	NetworkRx uint64 `json:"network_rx"`
	NetworkTx uint64 `json:"network_tx"`

	// Block I/O stats
	BlockRead  uint64 `json:"block_read"`
	BlockWrite uint64 `json:"block_write"`

	// Process stats
	PIDs uint64 `json:"pids"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// ContainerCreateRequest represents a request to create a new container
type ContainerCreateRequest struct {
	// Basic configuration
	Name        string            `json:"name"                  validate:"required,min=1,max=255"`
	Image       string            `json:"image"                 validate:"required"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`

	// Port mappings
	Ports []PortBinding `json:"ports,omitempty"`

	// Volume mounts
	Mounts []MountPoint `json:"mounts,omitempty"`

	// Resource limits
	CPUShares  int64 `json:"cpu_shares,omitempty"`
	Memory     int64 `json:"memory,omitempty"`
	MemorySwap int64 `json:"memory_swap,omitempty"`
	CPUQuota   int64 `json:"cpu_quota,omitempty"`
	CPUPeriod  int64 `json:"cpu_period,omitempty"`

	// Network configuration
	NetworkMode string `json:"network_mode,omitempty"`

	// Restart policy
	RestartPolicy RestartPolicy `json:"restart_policy,omitempty"`
	MaxRetries    int           `json:"max_retries,omitempty"`

	// Labels
	Labels map[string]string `json:"labels,omitempty"`

	// MCP specific
	ServerID  *uuid.UUID `json:"server_id,omitempty"`
	CatalogID *uuid.UUID `json:"catalog_id,omitempty"`

	// Runtime options
	AutoStart  bool `json:"auto_start,omitempty"`
	AutoRemove bool `json:"auto_remove,omitempty"`
}

// ContainerUpdateRequest represents a request to update container configuration
type ContainerUpdateRequest struct {
	// Resource limits
	CPUShares  *int64 `json:"cpu_shares,omitempty"`
	Memory     *int64 `json:"memory,omitempty"`
	MemorySwap *int64 `json:"memory_swap,omitempty"`
	CPUQuota   *int64 `json:"cpu_quota,omitempty"`
	CPUPeriod  *int64 `json:"cpu_period,omitempty"`

	// Restart policy
	RestartPolicy *RestartPolicy `json:"restart_policy,omitempty"`
	MaxRetries    *int           `json:"max_retries,omitempty"`
}

// ContainerFilter represents query filters for containers
type ContainerFilter struct {
	// Identity filters
	Names  []string `json:"names,omitempty"`
	IDs    []string `json:"ids,omitempty"`
	Images []string `json:"images,omitempty"`

	// State filters
	States       []ContainerState        `json:"states,omitempty"`
	HealthStatus []ContainerHealthStatus `json:"health_status,omitempty"`

	// MCP specific filters
	ServerIDs  []uuid.UUID `json:"server_ids,omitempty"`
	CatalogIDs []uuid.UUID `json:"catalog_ids,omitempty"`
	UserID     *uuid.UUID  `json:"user_id,omitempty"`
	TenantID   string      `json:"tenant_id,omitempty"`
	IsManaged  *bool       `json:"is_managed,omitempty"`

	// Label filters
	Labels map[string]string `json:"labels,omitempty"`

	// Time filters
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// ContainerAction represents actions that can be performed on containers
type ContainerAction string

const (
	ActionStart   ContainerAction = "start"
	ActionStop    ContainerAction = "stop"
	ActionRestart ContainerAction = "restart"
	ActionPause   ContainerAction = "pause"
	ActionUnpause ContainerAction = "unpause"
	ActionKill    ContainerAction = "kill"
	ActionRemove  ContainerAction = "remove"
	ActionUpdate  ContainerAction = "update"
)

// ContainerActionRequest represents a request to perform an action on containers
type ContainerActionRequest struct {
	Action       ContainerAction `json:"action"        validate:"required"`
	ContainerIDs []string        `json:"container_ids" validate:"required,min=1"`

	// Action-specific options
	Force   bool          `json:"force,omitempty"`   // For stop, kill, remove
	Signal  string        `json:"signal,omitempty"`  // For kill
	Timeout time.Duration `json:"timeout,omitempty"` // For stop

	// Update options (for update action)
	UpdateConfig *ContainerUpdateRequest `json:"update_config,omitempty"`
}

// ContainerActionResult represents the result of a container action
type ContainerActionResult struct {
	ContainerID string          `json:"container_id"`
	Action      ContainerAction `json:"action"`
	Success     bool            `json:"success"`
	Error       string          `json:"error,omitempty"`
	Duration    time.Duration   `json:"duration"`
	Timestamp   time.Time       `json:"timestamp"`
}

// BulkActionResult represents the result of a bulk container action
type BulkActionResult struct {
	Action       ContainerAction         `json:"action"`
	TotalCount   int                     `json:"total_count"`
	SuccessCount int                     `json:"success_count"`
	FailureCount int                     `json:"failure_count"`
	Results      []ContainerActionResult `json:"results"`
	Duration     time.Duration           `json:"duration"`
	Timestamp    time.Time               `json:"timestamp"`
}

// ContainerLogs represents container log entry
type ContainerLogs struct {
	ContainerID string    `json:"container_id"`
	Stream      string    `json:"stream"` // stdout, stderr
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
}

// LogsRequest represents a request to retrieve container logs
type LogsRequest struct {
	ContainerID string     `json:"container_id"         validate:"required"`
	Follow      bool       `json:"follow,omitempty"`
	Tail        string     `json:"tail,omitempty"` // number of lines or "all"
	Since       *time.Time `json:"since,omitempty"`
	Until       *time.Time `json:"until,omitempty"`
	Timestamps  bool       `json:"timestamps,omitempty"`
	Details     bool       `json:"details,omitempty"`
}

// ExecRequest represents a request to execute a command in a container
type ExecRequest struct {
	ContainerID  string   `json:"container_id"            validate:"required"`
	Command      []string `json:"command"                 validate:"required,min=1"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	User         string   `json:"user,omitempty"`
	Privileged   bool     `json:"privileged,omitempty"`
	TTY          bool     `json:"tty,omitempty"`
	AttachStdin  bool     `json:"attach_stdin,omitempty"`
	AttachStdout bool     `json:"attach_stdout,omitempty"`
	AttachStderr bool     `json:"attach_stderr,omitempty"`
	Environment  []string `json:"environment,omitempty"`
}

// ExecResult represents the result of command execution in a container
type ExecResult struct {
	ExecID      string        `json:"exec_id"`
	ContainerID string        `json:"container_id"`
	Command     []string      `json:"command"`
	ExitCode    int           `json:"exit_code"`
	Stdout      string        `json:"stdout,omitempty"`
	Stderr      string        `json:"stderr,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

// SystemInfo represents Docker system information
type SystemInfo struct {
	// Docker daemon info
	DockerVersion string `json:"docker_version"`
	APIVersion    string `json:"api_version"`
	GitCommit     string `json:"git_commit"`
	GoVersion     string `json:"go_version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	KernelVersion string `json:"kernel_version"`

	// System resources
	NCPU          int    `json:"ncpu"`
	MemTotal      int64  `json:"mem_total"`
	DockerRootDir string `json:"docker_root_dir"`

	// Container stats
	ContainersRunning int `json:"containers_running"`
	ContainersPaused  int `json:"containers_paused"`
	ContainersStopped int `json:"containers_stopped"`
	Images            int `json:"images"`

	// Storage driver
	Driver       string            `json:"driver"`
	DriverStatus map[string]string `json:"driver_status,omitempty"`

	// Registry info
	RegistryConfig map[string]any `json:"registry_config,omitempty"`

	// Timestamps
	ServerTime time.Time `json:"server_time"`
}

// ValidationError represents a Docker operation validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	return ve.Errors[0].Message
}

// Common errors
var (
	ErrContainerNotFound = &ValidationError{
		Field:   "container_id",
		Message: "Container not found",
		Code:    "not_found",
	}

	ErrContainerNotRunning = &ValidationError{
		Field:   "container_state",
		Message: "Container is not running",
		Code:    "invalid_state",
	}

	ErrInvalidContainerName = &ValidationError{
		Field:   "name",
		Message: "Invalid container name",
		Code:    "invalid_format",
	}

	ErrImageNotFound = &ValidationError{
		Field:   "image",
		Message: "Docker image not found",
		Code:    "image_not_found",
	}
)

// ContainerService defines the business logic interface for container management
type ContainerService interface {
	// Container lifecycle
	CreateContainer(
		ctx context.Context,
		userID string,
		req *ContainerCreateRequest,
	) (*Container, error)
	StartContainer(ctx context.Context, userID string, containerID string) error
	StopContainer(
		ctx context.Context,
		userID string,
		containerID string,
		timeout time.Duration,
	) error
	RestartContainer(
		ctx context.Context,
		userID string,
		containerID string,
		timeout time.Duration,
	) error
	PauseContainer(ctx context.Context, userID string, containerID string) error
	UnpauseContainer(ctx context.Context, userID string, containerID string) error
	KillContainer(ctx context.Context, userID string, containerID string, signal string) error
	RemoveContainer(ctx context.Context, userID string, containerID string, force bool) error

	// Container information
	GetContainer(ctx context.Context, userID string, containerID string) (*Container, error)
	ListContainers(
		ctx context.Context,
		userID string,
		filter ContainerFilter,
	) ([]*Container, int64, error)
	GetContainerStats(
		ctx context.Context,
		userID string,
		containerID string,
	) (*ContainerStats, error)
	GetContainerLogs(ctx context.Context, userID string, req *LogsRequest) ([]ContainerLogs, error)

	// Container operations
	UpdateContainer(
		ctx context.Context,
		userID string,
		containerID string,
		req *ContainerUpdateRequest,
	) error
	ExecInContainer(ctx context.Context, userID string, req *ExecRequest) (*ExecResult, error)

	// Bulk operations
	BulkContainerAction(
		ctx context.Context,
		userID string,
		req *ContainerActionRequest,
	) (*BulkActionResult, error)

	// System information
	GetSystemInfo(ctx context.Context, userID string) (*SystemInfo, error)

	// Health and monitoring
	HealthCheck(ctx context.Context) error
	GetContainerHealth(
		ctx context.Context,
		userID string,
		containerID string,
	) (*ContainerHealthStatus, error)
}

// ContainerRepository defines the interface for container data access
type ContainerRepository interface {
	// Container CRUD operations
	CreateContainer(ctx context.Context, userID string, container *Container) error
	GetContainer(ctx context.Context, userID string, containerID string) (*Container, error)
	UpdateContainer(ctx context.Context, userID string, container *Container) error
	DeleteContainer(ctx context.Context, userID string, containerID string) error

	// Container queries
	ListContainers(ctx context.Context, userID string, filter ContainerFilter) ([]*Container, error)
	CountContainers(ctx context.Context, userID string, filter ContainerFilter) (int64, error)
	ListContainersByServer(
		ctx context.Context,
		userID string,
		serverID uuid.UUID,
	) ([]*Container, error)

	// Container state tracking
	UpdateContainerState(
		ctx context.Context,
		userID string,
		containerID string,
		state ContainerState,
	) error
	UpdateContainerStats(
		ctx context.Context,
		userID string,
		containerID string,
		stats *ContainerStats,
	) error

	// Bulk operations
	UpdateContainersBatch(ctx context.Context, userID string, containers []*Container) error
	DeleteContainersBatch(ctx context.Context, userID string, containerIDs []string) error
}
