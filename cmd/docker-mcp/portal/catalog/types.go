// Package catalog provides server catalog management for the MCP Portal.
// It implements catalog data models, CLI integration, and repository patterns
// for managing MCP server catalogs with proper security and validation.
package catalog

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CatalogType represents the type of catalog
type CatalogType string

const (
	CatalogTypeOfficial      CatalogType = "official"       // Docker official catalog
	CatalogTypeTeam          CatalogType = "team"           // Team/organization catalog
	CatalogTypePersonal      CatalogType = "personal"       // User personal catalog
	CatalogTypeImported      CatalogType = "imported"       // Imported from external source
	CatalogTypeCustom        CatalogType = "custom"         // Custom user-created catalog
	CatalogTypeAdminBase     CatalogType = "admin_base"     // Admin-controlled base catalog
	CatalogTypeSystemDefault CatalogType = "system_default" // System default catalog
)

// CatalogStatus represents the status of a catalog
type CatalogStatus string

const (
	CatalogStatusActive       CatalogStatus = "active"       // Actively maintained
	CatalogStatusDeprecated   CatalogStatus = "deprecated"   // No longer maintained
	CatalogStatusExperimental CatalogStatus = "experimental" // Beta/testing
	CatalogStatusArchived     CatalogStatus = "archived"     // Historical/read-only
)

// ServerType represents the type of MCP server
type ServerType string

const (
	ServerTypeFilesystem   ServerType = "filesystem"   // File system operations
	ServerTypeDatabase     ServerType = "database"     // Database connections
	ServerTypeAPI          ServerType = "api"          // API integrations
	ServerTypeDevelopment  ServerType = "development"  // Development tools
	ServerTypeMonitoring   ServerType = "monitoring"   // Monitoring and observability
	ServerTypeAutomation   ServerType = "automation"   // Automation and workflows
	ServerTypeMLAI         ServerType = "ml_ai"        // Machine Learning and AI
	ServerTypeProductivity ServerType = "productivity" // Productivity tools
	ServerTypeOther        ServerType = "other"        // Other/uncategorized
)

// Catalog represents a collection of MCP servers
type Catalog struct {
	ID          uuid.UUID     `json:"id"           db:"id"`
	Name        string        `json:"name"         db:"name"`
	DisplayName string        `json:"display_name" db:"display_name"`
	Description string        `json:"description"  db:"description"`
	Type        CatalogType   `json:"type"         db:"type"`
	Status      CatalogStatus `json:"status"       db:"status"`
	Version     string        `json:"version"      db:"version"`

	// Ownership and access
	OwnerID   uuid.UUID `json:"owner_id"   db:"owner_id"`
	TenantID  string    `json:"tenant_id"  db:"tenant_id"`
	IsPublic  bool      `json:"is_public"  db:"is_public"`
	IsDefault bool      `json:"is_default" db:"is_default"`

	// External source information
	SourceURL    string            `json:"source_url,omitempty"    db:"source_url"`
	SourceType   string            `json:"source_type,omitempty"   db:"source_type"`
	SourceConfig map[string]string `json:"source_config,omitempty" db:"source_config"`

	// Server registry for file-based catalogs
	Registry        map[string]*ServerConfig `json:"registry,omitempty"         yaml:"registry,omitempty"`
	DisabledServers map[string]bool          `json:"disabled_servers,omitempty" yaml:"disabled_servers,omitempty"`

	// Metadata - flexible key-value storage
	Metadata   map[string]interface{} `json:"metadata,omitempty"   yaml:"metadata,omitempty"`
	Tags       []string               `json:"tags,omitempty"                                 db:"tags"`
	Homepage   string                 `json:"homepage,omitempty"                             db:"homepage"`
	Repository string                 `json:"repository,omitempty"                           db:"repository"`
	License    string                 `json:"license,omitempty"                              db:"license"`
	Maintainer string                 `json:"maintainer,omitempty"                           db:"maintainer"`

	// Statistics
	ServerCount   int        `json:"server_count"   db:"server_count"`
	DownloadCount int64      `json:"download_count" db:"download_count"`
	LastSyncedAt  *time.Time `json:"last_synced_at" db:"last_synced_at"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"           db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"           db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Relations (populated on demand)
	Servers []Server `json:"servers,omitempty"`
	Owner   *User    `json:"owner,omitempty"`
}

// Server represents an MCP server definition within a catalog
type Server struct {
	ID          uuid.UUID  `json:"id"           db:"id"`
	CatalogID   uuid.UUID  `json:"catalog_id"   db:"catalog_id"`
	Name        string     `json:"name"         db:"name"`
	DisplayName string     `json:"display_name" db:"display_name"`
	Description string     `json:"description"  db:"description"`
	Type        ServerType `json:"type"         db:"type"`
	Version     string     `json:"version"      db:"version"`

	// Server configuration
	Command     []string          `json:"command"               db:"command"`
	Environment map[string]string `json:"environment,omitempty" db:"environment"`
	WorkingDir  string            `json:"working_dir,omitempty" db:"working_dir"`

	// Docker configuration
	Image      string            `json:"image,omitempty"      db:"image"`
	Dockerfile string            `json:"dockerfile,omitempty" db:"dockerfile"`
	BuildArgs  map[string]string `json:"build_args,omitempty" db:"build_args"`

	// Resource requirements
	CPULimit    string `json:"cpu_limit,omitempty"    db:"cpu_limit"`
	MemoryLimit string `json:"memory_limit,omitempty" db:"memory_limit"`

	// Network and security
	Ports       []PortMapping   `json:"ports,omitempty"       db:"ports"`
	Volumes     []VolumeMapping `json:"volumes,omitempty"     db:"volumes"`
	Secrets     []string        `json:"secrets,omitempty"     db:"secrets"`
	Permissions []string        `json:"permissions,omitempty" db:"permissions"`

	// Metadata
	Tags       []string `json:"tags,omitempty"       db:"tags"`
	Homepage   string   `json:"homepage,omitempty"   db:"homepage"`
	Repository string   `json:"repository,omitempty" db:"repository"`
	License    string   `json:"license,omitempty"    db:"license"`
	Author     string   `json:"author,omitempty"     db:"author"`

	// Validation and requirements
	RequiredTools []string          `json:"required_tools,omitempty" db:"required_tools"`
	MinVersion    string            `json:"min_version,omitempty"    db:"min_version"`
	Config        map[string]string `json:"config,omitempty"         db:"config"`

	// Status
	IsEnabled          bool   `json:"is_enabled"                    db:"is_enabled"`
	IsDeprecated       bool   `json:"is_deprecated"                 db:"is_deprecated"`
	DeprecationMessage string `json:"deprecation_message,omitempty" db:"deprecation_message"`

	// Statistics
	UsageCount  int64   `json:"usage_count"  db:"usage_count"`
	RatingAvg   float64 `json:"rating_avg"   db:"rating_avg"`
	RatingCount int     `json:"rating_count" db:"rating_count"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"           db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"           db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Relations
	Catalog *Catalog `json:"catalog,omitempty"`
}

// PortMapping represents a port mapping for Docker containers
type PortMapping struct {
	Host      string `json:"host"      db:"host"`      // Host port
	Container string `json:"container" db:"container"` // Container port
	Protocol  string `json:"protocol"  db:"protocol"`  // tcp/udp
}

// VolumeMapping represents a volume mapping for Docker containers
type VolumeMapping struct {
	Host      string `json:"host"      db:"host"`      // Host path
	Container string `json:"container" db:"container"` // Container path
	Mode      string `json:"mode"      db:"mode"`      // ro/rw
}

// User represents a minimal user reference for catalog ownership
type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
}

// ServerConfig represents a server configuration in the catalog registry
type ServerConfig struct {
	Name        string                 `json:"name"                   yaml:"name"`
	DisplayName string                 `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Description string                 `json:"description,omitempty"  yaml:"description,omitempty"`
	Image       string                 `json:"image"                  yaml:"image"`
	Tag         string                 `json:"tag,omitempty"          yaml:"tag,omitempty"`
	Command     []string               `json:"command,omitempty"      yaml:"command,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"  yaml:"environment,omitempty"`
	Volumes     []VolumeMapping        `json:"volumes,omitempty"      yaml:"volumes,omitempty"`
	Ports       []PortMapping          `json:"ports,omitempty"        yaml:"ports,omitempty"`
	WorkingDir  string                 `json:"working_dir,omitempty"  yaml:"working_dir,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"     yaml:"metadata,omitempty"`
	IsEnabled   bool                   `json:"is_enabled"             yaml:"is_enabled"`
	IsMandatory bool                   `json:"is_mandatory,omitempty" yaml:"is_mandatory,omitempty"` // Cannot be disabled by users
}

// CatalogFilter represents query filters for catalogs
type CatalogFilter struct {
	Type      []CatalogType   `json:"type,omitempty"`
	Status    []CatalogStatus `json:"status,omitempty"`
	OwnerID   *uuid.UUID      `json:"owner_id,omitempty"`
	TenantID  string          `json:"tenant_id,omitempty"`
	IsPublic  *bool           `json:"is_public,omitempty"`
	IsDefault *bool           `json:"is_default,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	Search    string          `json:"search,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// ServerFilter represents query filters for servers
type ServerFilter struct {
	CatalogID    *uuid.UUID   `json:"catalog_id,omitempty"`
	Type         []ServerType `json:"type,omitempty"`
	IsEnabled    *bool        `json:"is_enabled,omitempty"`
	IsDeprecated *bool        `json:"is_deprecated,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Search       string       `json:"search,omitempty"`

	// Version filtering
	MinVersion string `json:"min_version,omitempty"`
	MaxVersion string `json:"max_version,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// CatalogSyncRequest represents a request to sync a catalog from external source
type CatalogSyncRequest struct {
	CatalogID uuid.UUID `json:"catalog_id"`
	Force     bool      `json:"force,omitempty"`
	DryRun    bool      `json:"dry_run,omitempty"`
}

// CatalogSyncResult represents the result of a catalog sync operation
type CatalogSyncResult struct {
	CatalogID      uuid.UUID `json:"catalog_id"`
	Success        bool      `json:"success"`
	ServersAdded   int       `json:"servers_added"`
	ServersUpdated int       `json:"servers_updated"`
	ServersRemoved int       `json:"servers_removed"`
	Errors         []string  `json:"errors,omitempty"`
	Warnings       []string  `json:"warnings,omitempty"`
	Duration       string    `json:"duration"`
	SyncedAt       time.Time `json:"synced_at"`
}

// CatalogImportRequest represents a request to import a catalog
type CatalogImportRequest struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	SourceURL   string            `json:"source_url"`
	SourceType  string            `json:"source_type,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
	Force       bool              `json:"force,omitempty"`
}

// CatalogExportRequest represents a request to export a catalog
type CatalogExportRequest struct {
	CatalogID    uuid.UUID `json:"catalog_id"`
	Format       string    `json:"format,omitempty"` // yaml, json
	IncludeStats bool      `json:"include_stats,omitempty"`
	Minify       bool      `json:"minify,omitempty"`
}

// ValidationError represents a catalog validation error
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
	ErrCatalogNotFound = &ValidationError{
		Field:   "catalog_id",
		Message: "Catalog not found",
		Code:    "not_found",
	}

	ErrServerNotFound = &ValidationError{
		Field:   "server_id",
		Message: "Server not found",
		Code:    "not_found",
	}
)

// CatalogRepository defines the interface for catalog data access
type CatalogRepository interface {
	// Catalog CRUD operations
	CreateCatalog(ctx context.Context, userID string, catalog *Catalog) error
	GetCatalog(ctx context.Context, userID string, id uuid.UUID) (*Catalog, error)
	GetCatalogByName(ctx context.Context, userID string, name string) (*Catalog, error)
	UpdateCatalog(ctx context.Context, userID string, catalog *Catalog) error
	DeleteCatalog(ctx context.Context, userID string, id uuid.UUID) error

	// Catalog queries
	ListCatalogs(ctx context.Context, userID string, filter CatalogFilter) ([]*Catalog, error)
	CountCatalogs(ctx context.Context, userID string, filter CatalogFilter) (int64, error)
	GetDefaultCatalog(ctx context.Context, userID string) (*Catalog, error)

	// Server CRUD operations
	CreateServer(ctx context.Context, userID string, server *Server) error
	GetServer(ctx context.Context, userID string, id uuid.UUID) (*Server, error)
	GetServerByName(
		ctx context.Context,
		userID string,
		catalogID uuid.UUID,
		name string,
	) (*Server, error)
	UpdateServer(ctx context.Context, userID string, server *Server) error
	DeleteServer(ctx context.Context, userID string, id uuid.UUID) error

	// Server queries
	ListServers(ctx context.Context, userID string, filter ServerFilter) ([]*Server, error)
	CountServers(ctx context.Context, userID string, filter ServerFilter) (int64, error)
	ListServersByCatalog(ctx context.Context, userID string, catalogID uuid.UUID) ([]*Server, error)

	// Bulk operations
	CreateServersBatch(ctx context.Context, userID string, servers []*Server) error
	UpdateServersBatch(ctx context.Context, userID string, servers []*Server) error
	DeleteServersBatch(ctx context.Context, userID string, ids []uuid.UUID) error

	// Search operations
	SearchServers(ctx context.Context, userID string, query string, limit int) ([]*Server, error)
	SearchCatalogs(ctx context.Context, userID string, query string, limit int) ([]*Catalog, error)

	// Statistics
	GetCatalogStats(ctx context.Context, userID string, catalogID uuid.UUID) (*CatalogStats, error)
	GetServerStats(ctx context.Context, userID string, serverID uuid.UUID) (*ServerStats, error)
}

// CatalogService defines the business logic interface for catalog management
type CatalogService interface {
	// Catalog management
	CreateCatalog(ctx context.Context, userID string, req *CreateCatalogRequest) (*Catalog, error)
	GetCatalog(ctx context.Context, userID string, id uuid.UUID) (*Catalog, error)
	UpdateCatalog(
		ctx context.Context,
		userID string,
		id uuid.UUID,
		req *UpdateCatalogRequest,
	) (*Catalog, error)
	DeleteCatalog(ctx context.Context, userID string, id uuid.UUID) error
	ListCatalogs(
		ctx context.Context,
		userID string,
		filter CatalogFilter,
	) ([]*Catalog, int64, error)

	// Server management
	CreateServer(ctx context.Context, userID string, req *CreateServerRequest) (*Server, error)
	GetServer(ctx context.Context, userID string, id uuid.UUID) (*Server, error)
	UpdateServer(
		ctx context.Context,
		userID string,
		id uuid.UUID,
		req *UpdateServerRequest,
	) (*Server, error)
	DeleteServer(ctx context.Context, userID string, id uuid.UUID) error
	ListServers(ctx context.Context, userID string, filter ServerFilter) ([]*Server, int64, error)

	// Catalog operations
	SyncCatalog(
		ctx context.Context,
		userID string,
		req *CatalogSyncRequest,
	) (*CatalogSyncResult, error)
	ImportCatalog(ctx context.Context, userID string, req *CatalogImportRequest) (*Catalog, error)
	ExportCatalog(ctx context.Context, userID string, req *CatalogExportRequest) ([]byte, error)
	ForkCatalog(
		ctx context.Context,
		userID string,
		sourceID uuid.UUID,
		name string,
	) (*Catalog, error)

	// CLI integration
	ExecuteCatalogCommand(
		ctx context.Context,
		userID string,
		command string,
		args []string,
	) (*CLIResult, error)

	// Validation
	ValidateCatalog(ctx context.Context, catalog *Catalog) []ValidationError
	ValidateServer(ctx context.Context, server *Server) []ValidationError
}

// CLIExecutor defines the interface for executing CLI commands
type CLIExecutor interface {
	ExecuteCommand(
		ctx context.Context,
		userID string,
		command string,
		args []string,
	) (*CLIResult, error)
	ValidateCommand(command string, args []string) error
	GetCommandHelp(command string) (string, error)
}

// CLIResult represents the result of a CLI command execution
type CLIResult struct {
	Command   string        `json:"command"`
	Args      []string      `json:"args"`
	Success   bool          `json:"success"`
	ExitCode  int           `json:"exit_code"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// CreateCatalogRequest represents a request to create a new catalog
type CreateCatalogRequest struct {
	Name        string            `json:"name"                   validate:"required,min=1,max=100"`
	DisplayName string            `json:"display_name,omitempty" validate:"max=200"`
	Description string            `json:"description,omitempty"  validate:"max=1000"`
	Type        CatalogType       `json:"type"                   validate:"required"`
	IsPublic    bool              `json:"is_public,omitempty"`
	IsDefault   bool              `json:"is_default,omitempty"`
	Tags        []string          `json:"tags,omitempty"         validate:"max=10,dive,max=50"`
	SourceURL   string            `json:"source_url,omitempty"   validate:"omitempty,url"`
	SourceType  string            `json:"source_type,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
}

// UpdateCatalogRequest represents a request to update a catalog
type UpdateCatalogRequest struct {
	DisplayName *string           `json:"display_name,omitempty" validate:"omitempty,max=200"`
	Description *string           `json:"description,omitempty"  validate:"omitempty,max=1000"`
	IsPublic    *bool             `json:"is_public,omitempty"`
	Tags        []string          `json:"tags,omitempty"         validate:"max=10,dive,max=50"`
	SourceURL   *string           `json:"source_url,omitempty"   validate:"omitempty,url"`
	Config      map[string]string `json:"config,omitempty"`
}

// CreateServerRequest represents a request to create a new server
type CreateServerRequest struct {
	CatalogID   uuid.UUID         `json:"catalog_id"             validate:"required"`
	Name        string            `json:"name"                   validate:"required,min=1,max=100"`
	DisplayName string            `json:"display_name,omitempty" validate:"max=200"`
	Description string            `json:"description,omitempty"  validate:"max=1000"`
	Type        ServerType        `json:"type"                   validate:"required"`
	Version     string            `json:"version,omitempty"      validate:"omitempty,semver"`
	Command     []string          `json:"command"                validate:"required,min=1"`
	Environment map[string]string `json:"environment,omitempty"`
	Image       string            `json:"image,omitempty"`
	Tags        []string          `json:"tags,omitempty"         validate:"max=10,dive,max=50"`
	Config      map[string]string `json:"config,omitempty"`
}

// UpdateServerRequest represents a request to update a server
type UpdateServerRequest struct {
	DisplayName *string           `json:"display_name,omitempty" validate:"omitempty,max=200"`
	Description *string           `json:"description,omitempty"  validate:"omitempty,max=1000"`
	Version     *string           `json:"version,omitempty"      validate:"omitempty,semver"`
	Command     []string          `json:"command,omitempty"      validate:"omitempty,min=1"`
	Environment map[string]string `json:"environment,omitempty"`
	Image       *string           `json:"image,omitempty"`
	Tags        []string          `json:"tags,omitempty"         validate:"max=10,dive,max=50"`
	Config      map[string]string `json:"config,omitempty"`
	IsEnabled   *bool             `json:"is_enabled,omitempty"`
}

// CatalogStats represents catalog statistics
type CatalogStats struct {
	ServerCount    int                `json:"server_count"`
	EnabledServers int                `json:"enabled_servers"`
	TypeCounts     map[ServerType]int `json:"type_counts"`
	TagCounts      map[string]int     `json:"tag_counts"`
	DownloadCount  int64              `json:"download_count"`
	AverageRating  float64            `json:"average_rating"`
	TotalRatings   int                `json:"total_ratings"`
	LastSyncedAt   *time.Time         `json:"last_synced_at"`
}

// ServerStats represents server statistics
type ServerStats struct {
	UsageCount      int64      `json:"usage_count"`
	RatingAvg       float64    `json:"rating_avg"`
	RatingCount     int        `json:"rating_count"`
	LastUsedAt      *time.Time `json:"last_used_at"`
	ConfigCount     int        `json:"config_count"`
	DependencyCount int        `json:"dependency_count"`
}
