package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// catalogService implements the CatalogService interface
type catalogService struct {
	repo     CatalogRepository
	executor executor.Executor
	audit    audit.Logger
	cache    cache.Cache
	mu       sync.RWMutex
	syncJobs map[uuid.UUID]*syncJob
}

// syncJob tracks an async catalog sync operation
type syncJob struct {
	ID        uuid.UUID
	CatalogID uuid.UUID
	Status    string
	Progress  int
	StartTime time.Time
	EndTime   *time.Time
	Result    *CatalogSyncResult
	Error     error
	Cancel    context.CancelFunc
}

// CreateCatalogService creates a new catalog service instance
func CreateCatalogService(
	repo CatalogRepository,
	exec executor.Executor,
	auditLogger audit.Logger,
	cacheStore cache.Cache,
) *catalogService {
	return &catalogService{
		repo:     repo,
		executor: exec,
		audit:    auditLogger,
		cache:    cacheStore,
		syncJobs: make(map[uuid.UUID]*syncJob),
	}
}

// CreateCatalog creates a new catalog
func (s *catalogService) CreateCatalog(
	ctx context.Context,
	userID string,
	req *CreateCatalogRequest,
) (*Catalog, error) {
	// Validate request
	if errs := s.validateCreateCatalogRequest(req); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create catalog entity
	catalog := &Catalog{
		ID:           uuid.New(),
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Type:         req.Type,
		Status:       CatalogStatusActive,
		Version:      "1.0.0",
		OwnerID:      uid,
		TenantID:     "", // Will be set from context
		IsPublic:     req.IsPublic,
		IsDefault:    req.IsDefault,
		SourceURL:    req.SourceURL,
		SourceType:   req.SourceType,
		SourceConfig: req.Config,
		Tags:         req.Tags,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Execute CLI command to create catalog
	cliReq := &executor.ExecutionRequest{
		Command:    executor.CommandTypeCatalogInit,
		Args:       []string{"--name", catalog.Name},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    30 * time.Second,
		JSONOutput: true,
	}

	// Execute command
	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeCommandFailure, map[string]interface{}{
			"command": "catalog.create",
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("failed to create catalog via CLI: %w", err)
	}

	// Parse CLI response if needed
	if result.Success {
		// Store in database
		if err := s.repo.CreateCatalog(ctx, userID, catalog); err != nil {
			return nil, fmt.Errorf("failed to store catalog: %w", err)
		}

		// Invalidate cache
		cacheKey := fmt.Sprintf("catalogs:%s", userID)
		s.cache.Delete(ctx, cacheKey)

		// Log success
		s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
			"action":       "catalog.created",
			"catalog_id":   catalog.ID.String(),
			"catalog_name": catalog.Name,
		})

		return catalog, nil
	}

	return nil, fmt.Errorf("CLI command failed: %s", result.Stderr)
}

// GetCatalog retrieves a catalog by ID
func (s *catalogService) GetCatalog(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Catalog, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("catalog:%s:%s", userID, id.String())

	// Try to get from cache
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
		var catalog Catalog
		if err := json.Unmarshal(data, &catalog); err == nil {
			return &catalog, nil
		}
	}

	// Fetch from repository
	cat, err := s.repo.GetCatalog(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	// Cache the result
	if data, err := json.Marshal(cat); err == nil {
		s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return cat, nil
}

// UpdateCatalog updates an existing catalog
func (s *catalogService) UpdateCatalog(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	req *UpdateCatalogRequest,
) (*Catalog, error) {
	// Get existing catalog
	catalog, err := s.GetCatalog(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.DisplayName != nil {
		catalog.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		catalog.Description = *req.Description
	}
	if req.IsPublic != nil {
		catalog.IsPublic = *req.IsPublic
	}
	if req.Tags != nil {
		catalog.Tags = req.Tags
	}
	if req.SourceURL != nil {
		catalog.SourceURL = *req.SourceURL
	}
	if req.Config != nil {
		catalog.SourceConfig = req.Config
	}

	catalog.UpdatedAt = time.Now().UTC()

	// Update in repository
	if err := s.repo.UpdateCatalog(ctx, userID, catalog); err != nil {
		return nil, fmt.Errorf("failed to update catalog: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("catalog:%s:%s", userID, id.String())
	s.cache.Delete(ctx, cacheKey)

	// Log update
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
		"action":     "catalog.updated",
		"catalog_id": catalog.ID.String(),
	})

	return catalog, nil
}

// DeleteCatalog deletes a catalog
func (s *catalogService) DeleteCatalog(ctx context.Context, userID string, id uuid.UUID) error {
	// Execute CLI command to remove catalog
	cliReq := &executor.ExecutionRequest{
		Command:   executor.CommandType("catalog.remove"),
		Args:      []string{id.String()},
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to delete catalog via CLI: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("CLI command failed: %s", result.Stderr)
	}

	// Delete from repository
	if err := s.repo.DeleteCatalog(ctx, userID, id); err != nil {
		return fmt.Errorf("failed to delete catalog from database: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("catalog:%s:%s", userID, id.String())
	s.cache.Delete(ctx, cacheKey)

	// Log deletion
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
		"action":     "catalog.deleted",
		"catalog_id": id.String(),
	})

	return nil
}

// ListCatalogs lists catalogs with filtering
func (s *catalogService) ListCatalogs(
	ctx context.Context,
	userID string,
	filter CatalogFilter,
) ([]*Catalog, int64, error) {
	// Check cache for list
	cacheKey := fmt.Sprintf("catalogs:%s:%v", userID, filter)

	// Try to get from cache
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
		var cached struct {
			Catalogs []*Catalog `json:"catalogs"`
			Count    int64      `json:"count"`
		}
		if err := json.Unmarshal(data, &cached); err == nil {
			return cached.Catalogs, cached.Count, nil
		}
	}

	// Execute CLI command to list catalogs
	cliReq := &executor.ExecutionRequest{
		Command:    executor.CommandTypeCatalogList,
		Args:       []string{"--format", "json"},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    10 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list catalogs via CLI: %w", err)
	}

	// Parse CLI output
	var cliCatalogs []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &cliCatalogs); err != nil {
		// Fall back to database if CLI output parsing fails
		catalogs, err := s.repo.ListCatalogs(ctx, userID, filter)
		if err != nil {
			return nil, 0, err
		}

		count, err := s.repo.CountCatalogs(ctx, userID, filter)
		if err != nil {
			return nil, 0, err
		}

		return catalogs, count, nil
	}

	// Get from repository with filter
	catalogs, err := s.repo.ListCatalogs(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list catalogs: %w", err)
	}

	count, err := s.repo.CountCatalogs(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count catalogs: %w", err)
	}

	// Cache the results
	cacheData := struct {
		Catalogs []*Catalog `json:"catalogs"`
		Count    int64      `json:"count"`
	}{
		Catalogs: catalogs,
		Count:    count,
	}
	if data, err := json.Marshal(cacheData); err == nil {
		s.cache.Set(ctx, cacheKey, data, 2*time.Minute)
	}

	return catalogs, count, nil
}

// CreateServer creates a new server in a catalog
func (s *catalogService) CreateServer(
	ctx context.Context,
	userID string,
	req *CreateServerRequest,
) (*Server, error) {
	// Validate request
	if errs := s.validateCreateServerRequest(req); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	// Create server entity
	server := &Server{
		ID:          uuid.New(),
		CatalogID:   req.CatalogID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        req.Type,
		Version:     req.Version,
		Command:     req.Command,
		Environment: req.Environment,
		Image:       req.Image,
		Tags:        req.Tags,
		Config:      req.Config,
		IsEnabled:   true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Store in repository
	if err := s.repo.CreateServer(ctx, userID, server); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Invalidate catalog cache
	cacheKey := fmt.Sprintf("servers:%s:%s", userID, req.CatalogID.String())
	s.cache.Delete(ctx, cacheKey)

	// Log creation
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
		"action":      "server.created",
		"server_id":   server.ID.String(),
		"server_name": server.Name,
		"catalog_id":  req.CatalogID.String(),
	})

	return server, nil
}

// GetServer retrieves a server by ID
func (s *catalogService) GetServer(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Server, error) {
	// Check cache
	cacheKey := fmt.Sprintf("server:%s:%s", userID, id.String())

	// Try to get from cache
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
		var server Server
		if err := json.Unmarshal(data, &server); err == nil {
			return &server, nil
		}
	}

	// Get from repository
	srv, err := s.repo.GetServer(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	// Cache result
	if data, err := json.Marshal(srv); err == nil {
		s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return srv, nil
}

// UpdateServer updates a server configuration
func (s *catalogService) UpdateServer(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	req *UpdateServerRequest,
) (*Server, error) {
	// Get existing server
	server, err := s.GetServer(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.DisplayName != nil {
		server.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		server.Description = *req.Description
	}
	if req.Version != nil {
		server.Version = *req.Version
	}
	if req.Command != nil {
		server.Command = req.Command
	}
	if req.Environment != nil {
		server.Environment = req.Environment
	}
	if req.Image != nil {
		server.Image = *req.Image
	}
	if req.Tags != nil {
		server.Tags = req.Tags
	}
	if req.Config != nil {
		server.Config = req.Config
	}
	if req.IsEnabled != nil {
		server.IsEnabled = *req.IsEnabled

		// Execute CLI command to enable/disable server
		command := executor.CommandTypeServerEnable
		if !*req.IsEnabled {
			command = executor.CommandTypeServerDisable
		}

		cliReq := &executor.ExecutionRequest{
			Command:   command,
			Args:      []string{server.Name},
			UserID:    userID,
			RequestID: uuid.New().String(),
			Timeout:   30 * time.Second,
		}

		if _, err := s.executor.Execute(ctx, cliReq); err != nil {
			return nil, fmt.Errorf("failed to update server status via CLI: %w", err)
		}
	}

	server.UpdatedAt = time.Now().UTC()

	// Update in repository
	if err := s.repo.UpdateServer(ctx, userID, server); err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("server:%s:%s", userID, id.String())
	s.cache.Delete(ctx, cacheKey)

	return server, nil
}

// DeleteServer removes a server from a catalog
func (s *catalogService) DeleteServer(ctx context.Context, userID string, id uuid.UUID) error {
	// Get server details first
	server, err := s.GetServer(ctx, userID, id)
	if err != nil {
		return err
	}

	// Disable server via CLI if enabled
	if server.IsEnabled {
		cliReq := &executor.ExecutionRequest{
			Command:   executor.CommandTypeServerDisable,
			Args:      []string{server.Name},
			UserID:    userID,
			RequestID: uuid.New().String(),
			Timeout:   30 * time.Second,
		}

		if _, err := s.executor.Execute(ctx, cliReq); err != nil {
			return fmt.Errorf("failed to disable server via CLI: %w", err)
		}
	}

	// Delete from repository
	if err := s.repo.DeleteServer(ctx, userID, id); err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("server:%s:%s", userID, id.String())
	s.cache.Delete(ctx, cacheKey)

	// Log deletion
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
		"action":      "server.deleted",
		"server_id":   id.String(),
		"server_name": server.Name,
	})

	return nil
}

// ListServers lists servers with filtering
func (s *catalogService) ListServers(
	ctx context.Context,
	userID string,
	filter ServerFilter,
) ([]*Server, int64, error) {
	// Execute CLI command to list servers
	cliReq := &executor.ExecutionRequest{
		Command:    executor.CommandTypeServerList,
		Args:       []string{"--format", "json"},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    10 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers via CLI: %w", err)
	}

	// Parse CLI output for enabled servers
	var cliServers []map[string]interface{}
	json.Unmarshal([]byte(result.Stdout), &cliServers)

	// Get from repository with filter
	servers, err := s.repo.ListServers(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list servers: %w", err)
	}

	count, err := s.repo.CountServers(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count servers: %w", err)
	}

	return servers, count, nil
}

// SyncCatalog synchronizes a catalog from its source
func (s *catalogService) SyncCatalog(
	ctx context.Context,
	userID string,
	req *CatalogSyncRequest,
) (*CatalogSyncResult, error) {
	// Create sync job
	jobID := uuid.New()
	syncCtx, cancel := context.WithCancel(ctx)

	job := &syncJob{
		ID:        jobID,
		CatalogID: req.CatalogID,
		Status:    "running",
		Progress:  0,
		StartTime: time.Now(),
		Cancel:    cancel,
	}

	s.mu.Lock()
	s.syncJobs[jobID] = job
	s.mu.Unlock()

	// Run sync in goroutine
	go func() {
		defer cancel()

		// Execute CLI sync command
		cliReq := &executor.ExecutionRequest{
			Command:      executor.CommandTypeCatalogSync,
			Args:         []string{req.CatalogID.String()},
			UserID:       userID,
			RequestID:    jobID.String(),
			Timeout:      5 * time.Minute,
			StreamOutput: true,
		}

		// Update progress periodically
		progressChan := make(chan int)
		go func() {
			for progress := range progressChan {
				s.mu.Lock()
				if j, exists := s.syncJobs[jobID]; exists {
					j.Progress = progress
				}
				s.mu.Unlock()
			}
		}()

		result, err := s.executor.Execute(syncCtx, cliReq)

		endTime := time.Now()
		syncResult := &CatalogSyncResult{
			CatalogID: req.CatalogID,
			Success:   result != nil && result.Success,
			Duration:  endTime.Sub(job.StartTime).String(),
			SyncedAt:  endTime,
		}

		if err != nil {
			syncResult.Success = false
			syncResult.Errors = []string{err.Error()}
			job.Error = err
		} else if result != nil && !result.Success {
			syncResult.Success = false
			syncResult.Errors = []string{result.Stderr}
		}

		// Parse sync results from CLI output if available
		if result != nil && result.Success {
			// Parse JSON output for server counts
			var stats map[string]int
			if err := json.Unmarshal([]byte(result.Stdout), &stats); err == nil {
				syncResult.ServersAdded = stats["added"]
				syncResult.ServersUpdated = stats["updated"]
				syncResult.ServersRemoved = stats["removed"]
			}
		}

		// Update job
		s.mu.Lock()
		job.Status = "completed"
		job.Progress = 100
		job.EndTime = &endTime
		job.Result = syncResult
		s.mu.Unlock()

		// Clean up job after 5 minutes
		time.AfterFunc(5*time.Minute, func() {
			s.mu.Lock()
			delete(s.syncJobs, jobID)
			s.mu.Unlock()
		})

		close(progressChan)
	}()

	// Wait a moment for job to start
	time.Sleep(100 * time.Millisecond)

	// Return initial result
	return &CatalogSyncResult{
		CatalogID: req.CatalogID,
		Success:   false,
		SyncedAt:  time.Now(),
	}, nil
}

// ImportCatalog imports a catalog from an external source
func (s *catalogService) ImportCatalog(
	ctx context.Context,
	userID string,
	req *CatalogImportRequest,
) (*Catalog, error) {
	// Create catalog from import request
	catalog := &Catalog{
		ID:           uuid.New(),
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Type:         CatalogTypeImported,
		Status:       CatalogStatusActive,
		SourceURL:    req.SourceURL,
		SourceType:   req.SourceType,
		SourceConfig: req.Config,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	uid, _ := uuid.Parse(userID)
	catalog.OwnerID = uid

	// Execute CLI import command
	args := []string{"import", "--url", req.SourceURL}
	if req.SourceType != "" {
		args = append(args, "--type", req.SourceType)
	}
	if req.Force {
		args = append(args, "--force")
	}

	cliReq := &executor.ExecutionRequest{
		Command:   executor.CommandType("catalog.import"),
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   2 * time.Minute,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("failed to import catalog: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("import failed: %s", result.Stderr)
	}

	// Store in repository
	if err := s.repo.CreateCatalog(ctx, userID, catalog); err != nil {
		return nil, fmt.Errorf("failed to store imported catalog: %w", err)
	}

	// Trigger initial sync
	go func() {
		syncReq := &CatalogSyncRequest{
			CatalogID: catalog.ID,
			Force:     true,
		}
		s.SyncCatalog(context.Background(), userID, syncReq)
	}()

	return catalog, nil
}

// ExportCatalog exports a catalog to a specified format
func (s *catalogService) ExportCatalog(
	ctx context.Context,
	userID string,
	req *CatalogExportRequest,
) ([]byte, error) {
	// Get catalog details
	catalog, err := s.GetCatalog(ctx, userID, req.CatalogID)
	if err != nil {
		return nil, err
	}

	// Get all servers in catalog
	servers, err := s.repo.ListServersByCatalog(ctx, userID, req.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers for export: %w", err)
	}

	// Build export data
	exportData := map[string]interface{}{
		"catalog": catalog,
		"servers": servers,
	}

	if req.IncludeStats {
		stats, err := s.repo.GetCatalogStats(ctx, userID, req.CatalogID)
		if err == nil {
			exportData["stats"] = stats
		}
	}

	// Marshal based on format
	format := req.Format
	if format == "" {
		format = "json"
	}

	var output []byte
	switch strings.ToLower(format) {
	case "json":
		if req.Minify {
			output, err = json.Marshal(exportData)
		} else {
			output, err = json.MarshalIndent(exportData, "", "  ")
		}
	case "yaml":
		// Would implement YAML marshaling here
		return nil, fmt.Errorf("YAML export not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal export data: %w", err)
	}

	return output, nil
}

// ForkCatalog creates a copy of an existing catalog
func (s *catalogService) ForkCatalog(
	ctx context.Context,
	userID string,
	sourceID uuid.UUID,
	name string,
) (*Catalog, error) {
	// Get source catalog
	source, err := s.GetCatalog(ctx, userID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source catalog: %w", err)
	}

	// Create forked catalog
	uid, _ := uuid.Parse(userID)
	fork := &Catalog{
		ID:          uuid.New(),
		Name:        name,
		DisplayName: fmt.Sprintf("%s (Fork)", source.DisplayName),
		Description: fmt.Sprintf("Forked from %s", source.Name),
		Type:        CatalogTypePersonal,
		Status:      CatalogStatusActive,
		Version:     "1.0.0",
		OwnerID:     uid,
		IsPublic:    false,
		Tags:        source.Tags,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Store fork
	if err := s.repo.CreateCatalog(ctx, userID, fork); err != nil {
		return nil, fmt.Errorf("failed to create fork: %w", err)
	}

	// Copy servers
	servers, err := s.repo.ListServersByCatalog(ctx, userID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source servers: %w", err)
	}

	// Clone servers to fork
	for _, server := range servers {
		clone := &Server{
			ID:          uuid.New(),
			CatalogID:   fork.ID,
			Name:        server.Name,
			DisplayName: server.DisplayName,
			Description: server.Description,
			Type:        server.Type,
			Version:     server.Version,
			Command:     server.Command,
			Environment: server.Environment,
			Image:       server.Image,
			Tags:        server.Tags,
			Config:      server.Config,
			IsEnabled:   false, // Start with servers disabled
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err := s.repo.CreateServer(ctx, userID, clone); err != nil {
			// Log error but continue
			continue
		}
	}

	return fork, nil
}

// ExecuteCatalogCommand executes a catalog-related CLI command
func (s *catalogService) ExecuteCatalogCommand(
	ctx context.Context,
	userID string,
	command string,
	args []string,
) (*CLIResult, error) {
	// Map command to executor CommandType
	var cmdType executor.CommandType
	switch command {
	case "list":
		cmdType = executor.CommandTypeCatalogList
	case "show":
		cmdType = executor.CommandTypeCatalogShow
	case "init":
		cmdType = executor.CommandTypeCatalogInit
	case "sync":
		cmdType = executor.CommandTypeCatalogSync
	default:
		return nil, fmt.Errorf("unsupported catalog command: %s", command)
	}

	// Execute command
	cliReq := &executor.ExecutionRequest{
		Command:    cmdType,
		Args:       args,
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    30 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Convert to CLIResult
	return &CLIResult{
		Command:   command,
		Args:      args,
		Success:   result.Success,
		ExitCode:  result.ExitCode,
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
		Duration:  result.Duration,
		Timestamp: result.StartTime,
	}, nil
}

// ValidateCatalog validates a catalog configuration
func (s *catalogService) ValidateCatalog(ctx context.Context, catalog *Catalog) []ValidationError {
	var errors []ValidationError

	// Validate name
	if catalog.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "catalog name is required",
			Code:    "required",
		})
	} else if len(catalog.Name) > 100 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Value:   catalog.Name,
			Message: "catalog name must be 100 characters or less",
			Code:    "max_length",
		})
	}

	// Validate type
	validTypes := []CatalogType{
		CatalogTypeOfficial,
		CatalogTypeTeam,
		CatalogTypePersonal,
		CatalogTypeImported,
		CatalogTypeCustom,
	}

	valid := false
	for _, t := range validTypes {
		if catalog.Type == t {
			valid = true
			break
		}
	}

	if !valid {
		errors = append(errors, ValidationError{
			Field:   "type",
			Value:   string(catalog.Type),
			Message: "invalid catalog type",
			Code:    "invalid_enum",
		})
	}

	// Validate source URL if provided
	if catalog.SourceURL != "" {
		// Basic URL validation
		if !strings.HasPrefix(catalog.SourceURL, "http://") &&
			!strings.HasPrefix(catalog.SourceURL, "https://") {
			errors = append(errors, ValidationError{
				Field:   "source_url",
				Value:   catalog.SourceURL,
				Message: "source URL must be a valid HTTP/HTTPS URL",
				Code:    "invalid_format",
			})
		}
	}

	return errors
}

// ValidateServer validates a server configuration
func (s *catalogService) ValidateServer(ctx context.Context, server *Server) []ValidationError {
	var errors []ValidationError

	// Validate name
	if server.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "server name is required",
			Code:    "required",
		})
	}

	// Validate command
	if len(server.Command) == 0 {
		errors = append(errors, ValidationError{
			Field:   "command",
			Message: "server command is required",
			Code:    "required",
		})
	}

	// Validate type
	validTypes := []ServerType{
		ServerTypeFilesystem,
		ServerTypeDatabase,
		ServerTypeAPI,
		ServerTypeDevelopment,
		ServerTypeMonitoring,
		ServerTypeAutomation,
		ServerTypeMLAI,
		ServerTypeProductivity,
		ServerTypeOther,
	}

	valid := false
	for _, t := range validTypes {
		if server.Type == t {
			valid = true
			break
		}
	}

	if !valid {
		errors = append(errors, ValidationError{
			Field:   "type",
			Value:   string(server.Type),
			Message: "invalid server type",
			Code:    "invalid_enum",
		})
	}

	// Validate image if provided
	if server.Image != "" && !strings.Contains(server.Image, ":") {
		errors = append(errors, ValidationError{
			Field:   "image",
			Value:   server.Image,
			Message: "Docker image should include a tag",
			Code:    "warning",
		})
	}

	return errors
}

// Helper validation functions

func (s *catalogService) validateCreateCatalogRequest(req *CreateCatalogRequest) []ValidationError {
	catalog := &Catalog{
		Name:      req.Name,
		Type:      req.Type,
		SourceURL: req.SourceURL,
	}
	return s.ValidateCatalog(context.Background(), catalog)
}

func (s *catalogService) validateCreateServerRequest(req *CreateServerRequest) []ValidationError {
	server := &Server{
		Name:    req.Name,
		Command: req.Command,
		Type:    req.Type,
		Image:   req.Image,
	}
	return s.ValidateServer(context.Background(), server)
}
