package userconfig

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserConfigService defines the interface for user configuration management
type UserConfigService interface {
	// Configuration CRUD operations
	CreateConfig(ctx context.Context, userID string, req *CreateConfigRequest) (*UserConfig, error)
	GetConfig(ctx context.Context, userID string, id uuid.UUID) (*UserConfig, error)
	UpdateConfig(
		ctx context.Context,
		userID string,
		id uuid.UUID,
		req *UpdateConfigRequest,
	) (*UserConfig, error)
	DeleteConfig(ctx context.Context, userID string, id uuid.UUID) error
	ListConfigs(
		ctx context.Context,
		userID string,
		filter ConfigFilter,
	) ([]*UserConfig, int64, error)

	// Server configuration operations
	EnableServer(ctx context.Context, userID string, serverName string) error
	DisableServer(ctx context.Context, userID string, serverName string) error
	ConfigureServer(ctx context.Context, userID string, req *ServerConfigRequest) error
	GetServerStatus(ctx context.Context, userID string, serverName string) (*ServerStatus, error)

	// Bulk operations
	ImportConfig(ctx context.Context, userID string, req *ConfigImportRequest) (*UserConfig, error)
	ExportConfig(ctx context.Context, userID string, req *ConfigExportRequest) ([]byte, error)

	// CLI command execution
	ExecuteConfigCommand(
		ctx context.Context,
		userID string,
		command string,
		args []string,
	) (*CLIResult, error)

	// Validation
	ValidateConfig(ctx context.Context, config *UserConfig) []ValidationError
}

// UserConfigRepository defines the database interface for user configurations
type UserConfigRepository interface {
	CreateConfig(ctx context.Context, userID string, config *UserConfig) error
	GetConfig(ctx context.Context, userID string, id uuid.UUID) (*UserConfig, error)
	UpdateConfig(ctx context.Context, userID string, config *UserConfig) error
	DeleteConfig(ctx context.Context, userID string, id uuid.UUID) error
	ListConfigs(ctx context.Context, userID string, filter ConfigFilter) ([]*UserConfig, error)
	CountConfigs(ctx context.Context, userID string, filter ConfigFilter) (int64, error)

	// Server configuration queries
	GetServerConfig(ctx context.Context, userID string, serverName string) (*ServerConfig, error)
	SaveServerConfig(ctx context.Context, userID string, config *ServerConfig) error
	DeleteServerConfig(ctx context.Context, userID string, serverName string) error
	ListServerConfigs(ctx context.Context, userID string) ([]*ServerConfig, error)
}

// UserConfig represents a user's MCP configuration
type UserConfig struct {
	ID          uuid.UUID      `json:"id"                     db:"id"`
	Name        string         `json:"name"                   db:"name"`
	DisplayName string         `json:"display_name"           db:"display_name"`
	Description string         `json:"description"            db:"description"`
	Type        ConfigType     `json:"type"                   db:"type"`
	Status      ConfigStatus   `json:"status"                 db:"status"`
	OwnerID     uuid.UUID      `json:"owner_id"               db:"owner_id"`
	TenantID    string         `json:"tenant_id"              db:"tenant_id"`
	IsDefault   bool           `json:"is_default"             db:"is_default"`
	IsActive    bool           `json:"is_active"              db:"is_active"`
	Version     string         `json:"version"                db:"version"`
	Settings    map[string]any `json:"settings"               db:"settings"` // Encrypted JSON
	Servers     []ServerConfig `json:"servers,omitempty"`
	CreatedAt   time.Time      `json:"created_at"             db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"             db:"updated_at"`
	LastUsedAt  *time.Time     `json:"last_used_at,omitempty" db:"last_used_at"`
}

// ServerConfig represents configuration for a specific MCP server
type ServerConfig struct {
	ID          uuid.UUID         `json:"id"           db:"id"`
	ConfigID    uuid.UUID         `json:"config_id"    db:"config_id"`
	ServerName  string            `json:"server_name"  db:"server_name"`
	DisplayName string            `json:"display_name" db:"display_name"`
	IsEnabled   bool              `json:"is_enabled"   db:"is_enabled"`
	Command     []string          `json:"command"      db:"command"`
	Environment map[string]string `json:"environment"  db:"environment"` // Encrypted
	Args        []string          `json:"args"         db:"args"`
	WorkingDir  string            `json:"working_dir"  db:"working_dir"`
	Image       string            `json:"image"        db:"image"`
	Transport   TransportType     `json:"transport"    db:"transport"`
	Settings    map[string]any    `json:"settings"     db:"settings"` // Encrypted JSON
	CreatedAt   time.Time         `json:"created_at"   db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"   db:"updated_at"`

	// Compatibility fields for implementations expecting different field names
	ServerID string         `json:"server_id,omitempty"` // Alias for ServerName
	Config   map[string]any `json:"config,omitempty"`    // Alias for Settings
	Metadata map[string]any `json:"metadata,omitempty"`  // Additional metadata
	Status   string         `json:"status,omitempty"`    // Server status
}

// GetServerID returns the server identifier (alias for ServerName)
func (sc *ServerConfig) GetServerID() string {
	if sc.ServerID != "" {
		return sc.ServerID
	}
	return sc.ServerName
}

// GetConfig returns the configuration (alias for Settings)
func (sc *ServerConfig) GetConfig() map[string]any {
	if sc.Config != nil {
		return sc.Config
	}
	return sc.Settings
}

// SetCompatibilityFields sets the compatibility fields for cross-compatibility
func (sc *ServerConfig) SetCompatibilityFields() {
	if sc.ServerID == "" {
		sc.ServerID = sc.ServerName
	}
	if sc.Config == nil && sc.Settings != nil {
		sc.Config = sc.Settings
	}
	if sc.Status == "" && sc.IsEnabled {
		sc.Status = "enabled"
	} else if sc.Status == "" {
		sc.Status = "disabled"
	}
}

// ServerStatus represents the current status of an MCP server
type ServerStatus struct {
	ServerName  string         `json:"server_name"`
	IsEnabled   bool           `json:"is_enabled"`
	IsRunning   bool           `json:"is_running"`
	Status      string         `json:"status"`
	ProcessID   int            `json:"process_id,omitempty"`
	StartTime   *time.Time     `json:"start_time,omitempty"`
	Uptime      string         `json:"uptime,omitempty"`
	MemoryUsage int64          `json:"memory_usage,omitempty"`
	CPUUsage    float64        `json:"cpu_usage,omitempty"`
	Connections int            `json:"connections"`
	LastError   string         `json:"last_error,omitempty"`
	LastPing    *time.Time     `json:"last_ping,omitempty"`
	Config      *ServerConfig  `json:"config,omitempty"`
	Metrics     map[string]any `json:"metrics,omitempty"`
}

// ConfigType represents the type of configuration
type ConfigType string

const (
	ConfigTypeDefault     ConfigType = "default"
	ConfigTypePersonal    ConfigType = "personal"
	ConfigTypeTeam        ConfigType = "team"
	ConfigTypeEnvironment ConfigType = "environment"
	ConfigTypeProject     ConfigType = "project"
)

// ConfigStatus represents the status of a configuration
type ConfigStatus string

const (
	ConfigStatusActive   ConfigStatus = "active"
	ConfigStatusInactive ConfigStatus = "inactive"
	ConfigStatusDraft    ConfigStatus = "draft"
	ConfigStatusArchived ConfigStatus = "archived"
)

// TransportType represents the transport protocol for MCP communication
type TransportType string

const (
	TransportStdio     TransportType = "stdio"
	TransportStreaming TransportType = "streaming"
	TransportSSE       TransportType = "sse"
	TransportWebSocket TransportType = "websocket"
)

// Request/Response types

// CreateConfigRequest represents a request to create a new configuration
type CreateConfigRequest struct {
	Name        string                      `json:"name"              binding:"required,min=1,max=100"`
	DisplayName string                      `json:"display_name"      binding:"required,min=1,max=200"`
	Description string                      `json:"description"       binding:"max=1000"`
	Type        ConfigType                  `json:"type"              binding:"required"`
	IsDefault   bool                        `json:"is_default"`
	Settings    map[string]any              `json:"settings"`
	Servers     []CreateServerConfigRequest `json:"servers,omitempty"`
}

// UpdateConfigRequest represents a request to update a configuration
type UpdateConfigRequest struct {
	DisplayName *string        `json:"display_name,omitempty" binding:"omitempty,min=1,max=200"`
	Description *string        `json:"description,omitempty"  binding:"omitempty,max=1000"`
	IsDefault   *bool          `json:"is_default,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
	Status      *ConfigStatus  `json:"status,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}

// CreateServerConfigRequest represents a request to create server configuration
type CreateServerConfigRequest struct {
	ServerName  string            `json:"server_name"           binding:"required"`
	DisplayName string            `json:"display_name"          binding:"required"`
	IsEnabled   bool              `json:"is_enabled"`
	Command     []string          `json:"command"               binding:"required,min=1"`
	Environment map[string]string `json:"environment,omitempty"`
	Args        []string          `json:"args,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Image       string            `json:"image,omitempty"`
	Transport   TransportType     `json:"transport"             binding:"required"`
	Settings    map[string]any    `json:"settings,omitempty"`
}

// UpdateServerConfigRequest represents a request to update server configuration
type UpdateServerConfigRequest struct {
	DisplayName *string           `json:"display_name,omitempty"`
	IsEnabled   *bool             `json:"is_enabled,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Args        []string          `json:"args,omitempty"`
	WorkingDir  *string           `json:"working_dir,omitempty"`
	Image       *string           `json:"image,omitempty"`
	Transport   *TransportType    `json:"transport,omitempty"`
	Settings    map[string]any    `json:"settings,omitempty"`
}

// ServerConfigRequest represents a request to configure a server
type ServerConfigRequest struct {
	ServerName string                     `json:"server_name"      binding:"required"`
	Action     ServerConfigAction         `json:"action"           binding:"required"`
	Config     *UpdateServerConfigRequest `json:"config,omitempty"`
}

// ServerConfigAction represents an action to perform on server configuration
type ServerConfigAction string

const (
	ServerActionEnable    ServerConfigAction = "enable"
	ServerActionDisable   ServerConfigAction = "disable"
	ServerActionConfigure ServerConfigAction = "configure"
	ServerActionRestart   ServerConfigAction = "restart"
	ServerActionStop      ServerConfigAction = "stop"
	ServerActionStart     ServerConfigAction = "start"
)

// ConfigImportRequest represents a request to import configuration
type ConfigImportRequest struct {
	Name        string    `json:"name"         binding:"required"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Source      string    `json:"source"       binding:"required"` // file path or URL
	SourceType  string    `json:"source_type"`                     // json, yaml, toml
	Data        []byte    `json:"data"`
	EncryptKeys []string  `json:"encrypt_keys"`
	MergeMode   MergeMode `json:"merge_mode"`
	Force       bool      `json:"force"`
	DryRun      bool      `json:"dry_run"`
}

// ConfigExportRequest represents a request to export configuration
type ConfigExportRequest struct {
	ConfigID       uuid.UUID   `json:"config_id"       binding:"required"`
	ConfigIDs      []uuid.UUID `json:"config_ids"`
	Format         string      `json:"format"`          // json, yaml, toml
	IncludeSecrets bool        `json:"include_secrets"` // Whether to include encrypted secrets
	IncludeServers bool        `json:"include_servers"` // Whether to include server configs
	Minify         bool        `json:"minify"`          // Whether to minify output
}

// MergeMode represents how to merge imported configuration
type MergeMode string

const (
	MergeModeReplace MergeMode = "replace"
	MergeModeOverlay MergeMode = "overlay"
	MergeModeAppend  MergeMode = "append"
)

// ConfigFilter represents filtering options for listing configurations
type ConfigFilter struct {
	Type        ConfigType   `form:"type"`
	Status      ConfigStatus `form:"status"`
	IsDefault   *bool        `form:"is_default"`
	IsActive    *bool        `form:"is_active"`
	Search      string       `form:"search"`
	NamePattern string       `form:"name_pattern"` // Add for search functionality
	Limit       int          `form:"limit"        binding:"min=1,max=100"`
	Offset      int          `form:"offset"       binding:"min=0"`
	SortBy      string       `form:"sort_by"`
	SortOrder   string       `form:"sort_order"   binding:"oneof=asc desc"`
}

// CLIResult represents the result of a CLI command execution
type CLIResult struct {
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	Success    bool      `json:"success"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	Output     string    `json:"output"`
	ErrorMsg   string    `json:"error_msg"`
	Duration   string    `json:"duration"`
	Timestamp  time.Time `json:"timestamp"`
	ExecutedAt time.Time `json:"executed_at"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ConfigStats represents statistics about user configurations
type ConfigStats struct {
	TotalConfigs   int64      `json:"total_configs"`
	ActiveConfigs  int64      `json:"active_configs"`
	DefaultConfigs int64      `json:"default_configs"`
	TotalServers   int64      `json:"total_servers"`
	EnabledServers int64      `json:"enabled_servers"`
	RunningServers int64      `json:"running_servers"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
}
