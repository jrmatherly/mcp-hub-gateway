package catalog

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

// MultiUserCatalogService extends the catalog service with multi-user support
type MultiUserCatalogService struct {
	*catalogService                          // Embed existing service
	FileManager     *FileCatalogManager      // File-based catalog manager (exported)
	Inheritance     *InheritanceEngine       // Catalog inheritance engine (exported)
	encryption      crypto.EncryptionService // Encryption for sensitive data
	mu              sync.RWMutex
}

// CreateMultiUserCatalogService creates a new multi-user catalog service
func CreateMultiUserCatalogService(
	repo CatalogRepository,
	exec executor.Executor,
	auditLogger audit.Logger,
	cacheStore cache.Cache,
	encryption crypto.EncryptionService,
) (*MultiUserCatalogService, error) {
	// Create base service
	baseService := CreateCatalogService(repo, exec, auditLogger, cacheStore)

	// Create encryption adapter to bridge interface differences
	var encryptionAdapter EncryptionService
	if encryption != nil {
		// Generate a key for the adapter (in production, this would come from configuration)
		key := make([]byte, 32) // 256-bit key
		encryptionAdapter = NewEncryptionAdapter(encryption, key)
	}

	// Create file manager
	fileManager, err := CreateFileCatalogManager(
		"/app/data/catalogs/base",
		"/app/user-catalogs",
		encryptionAdapter,
		5*time.Minute, // Cache TTL
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file manager: %w", err)
	}

	// Create inheritance engine
	inheritance := CreateInheritanceEngine(
		fileManager,
		repo,
		10*time.Minute, // Cache TTL
	)

	return &MultiUserCatalogService{
		catalogService: baseService,
		FileManager:    fileManager,
		Inheritance:    inheritance,
		encryption:     encryption,
	}, nil
}

// GetResolvedCatalogForUser returns the merged catalog for a specific user
// This includes admin base catalogs + user customizations
func (s *MultiUserCatalogService) GetResolvedCatalogForUser(
	ctx context.Context,
	userID string,
) (*ResolvedCatalog, error) {
	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Use inheritance engine to resolve catalog
	resolved, err := s.Inheritance.ResolveForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve catalog for user: %w", err)
	}

	// Log the resolution
	uidLog, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uidLog, audit.EventTypeConfiguration, map[string]interface{}{
		"action":         "catalog.resolved",
		"user_id":        userID,
		"server_count":   len(resolved.MergedCatalog.Registry),
		"admin_servers":  resolved.AdminServers,
		"user_overrides": resolved.UserOverrides,
	})

	return resolved, nil
}

// CreateAdminBaseCatalog creates an admin-controlled base catalog
func (s *MultiUserCatalogService) CreateAdminBaseCatalog(
	ctx context.Context,
	adminID string,
	req *CreateCatalogRequest,
) (*Catalog, error) {
	// Validate admin permissions (would check against admin role in production)
	if !s.isAdmin(ctx, adminID) {
		return nil, fmt.Errorf("unauthorized: admin access required")
	}

	// Create catalog with admin base type
	req.Type = CatalogTypeAdminBase
	catalog, err := s.CreateCatalog(ctx, adminID, req)
	if err != nil {
		return nil, err
	}

	// Save to file system for distribution
	if err := s.FileManager.SaveBaseCatalog(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to save base catalog: %w", err)
	}

	// Invalidate all user catalog caches
	s.Inheritance.ClearCache()

	return catalog, nil
}

// UpdateUserCatalogCustomization allows users to customize their catalog
func (s *MultiUserCatalogService) UpdateUserCatalogCustomization(
	ctx context.Context,
	userID string,
	req *UserCatalogCustomizationRequest,
) (*Catalog, error) {
	// Get current user catalog
	userCatalog, err := s.FileManager.LoadUserCatalog(ctx, userID)
	if err != nil {
		// Create new user catalog if it doesn't exist
		userCatalog = &Catalog{
			ID:              uuid.New(),
			Name:            fmt.Sprintf("user-%s", userID),
			DisplayName:     "Personal Catalog",
			Type:            CatalogTypePersonal,
			Status:          CatalogStatusActive,
			Registry:        make(map[string]*ServerConfig),
			DisabledServers: make(map[string]bool),
		}

		// Parse user ID
		if uid, err := uuid.Parse(userID); err == nil {
			userCatalog.OwnerID = uid
		}
	}

	// Apply customizations
	if req.EnabledServers != nil {
		// Reset disabled servers
		userCatalog.DisabledServers = make(map[string]bool)

		// Mark servers not in the enabled list as disabled
		uid, _ := uuid.Parse(userID)
		resolved, _ := s.Inheritance.ResolveForUser(ctx, uid)
		if resolved != nil {
			for serverName := range resolved.MergedCatalog.Registry {
				enabled := false
				for _, enabledName := range req.EnabledServers {
					if serverName == enabledName {
						enabled = true
						break
					}
				}
				if !enabled {
					userCatalog.DisabledServers[serverName] = true
				}
			}
		}
	}

	if req.CustomServers != nil {
		// Add or update custom servers
		for _, serverConfig := range req.CustomServers {
			userCatalog.Registry[serverConfig.Name] = serverConfig
		}
	}

	// Save user catalog
	if err := s.FileManager.SaveUserCatalog(ctx, userID, userCatalog); err != nil {
		return nil, fmt.Errorf("failed to save user catalog: %w", err)
	}

	// Save to database for persistence
	if err := s.repo.UpdateCatalog(ctx, userID, userCatalog); err != nil {
		return nil, fmt.Errorf("failed to persist user catalog: %w", err)
	}

	// Clear cache for this user
	s.Inheritance.clearUserCache(userID)

	// Log the customization
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]interface{}{
		"action":          "catalog.customized",
		"enabled_servers": len(req.EnabledServers),
		"custom_servers":  len(req.CustomServers),
	})

	return userCatalog, nil
}

// GetAdminBaseCatalogs returns all admin-controlled base catalogs
func (s *MultiUserCatalogService) GetAdminBaseCatalogs(
	ctx context.Context,
	adminID string,
) ([]*Catalog, error) {
	// Validate admin permissions
	if !s.isAdmin(ctx, adminID) {
		return nil, fmt.Errorf("unauthorized: admin access required")
	}

	// List catalogs with admin base type
	filter := CatalogFilter{
		Type: []CatalogType{CatalogTypeAdminBase},
	}

	return s.repo.ListCatalogs(ctx, adminID, filter)
}

// GetUserCustomizations returns user-specific catalog customizations
func (s *MultiUserCatalogService) GetUserCustomizations(
	ctx context.Context,
	userID string,
) (*UserCatalogCustomizations, error) {
	// Load user catalog
	userCatalog, err := s.FileManager.LoadUserCatalog(ctx, userID)
	if err != nil {
		// No customizations yet
		return &UserCatalogCustomizations{
			UserID:          userID,
			EnabledServers:  []string{},
			DisabledServers: []string{},
			CustomServers:   []*ServerConfig{},
		}, nil
	}

	// Build customizations response
	customizations := &UserCatalogCustomizations{
		UserID:          userID,
		EnabledServers:  []string{},
		DisabledServers: []string{},
		CustomServers:   []*ServerConfig{},
	}

	// Get disabled servers
	for serverName, disabled := range userCatalog.DisabledServers {
		if disabled {
			customizations.DisabledServers = append(customizations.DisabledServers, serverName)
		}
	}

	// Get custom servers
	for _, serverConfig := range userCatalog.Registry {
		customizations.CustomServers = append(customizations.CustomServers, serverConfig)
	}

	// Get enabled servers (all servers not explicitly disabled)
	uid, _ := uuid.Parse(userID)
	resolved, err := s.Inheritance.ResolveForUser(ctx, uid)
	if err == nil && resolved != nil {
		for serverName := range resolved.MergedCatalog.Registry {
			if !userCatalog.DisabledServers[serverName] {
				customizations.EnabledServers = append(customizations.EnabledServers, serverName)
			}
		}
	}

	return customizations, nil
}

// ImportAdminCatalog imports a catalog as an admin base catalog
func (s *MultiUserCatalogService) ImportAdminCatalog(
	ctx context.Context,
	adminID string,
	data []byte,
	format string,
) (*Catalog, error) {
	// Validate admin permissions
	if !s.isAdmin(ctx, adminID) {
		return nil, fmt.Errorf("unauthorized: admin access required")
	}

	// Import catalog using file manager
	catalog, err := s.FileManager.ImportCatalog(ctx, data, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import catalog: %w", err)
	}

	// Set as admin base type
	catalog.Type = CatalogTypeAdminBase

	// Save as base catalog
	if err := s.FileManager.SaveBaseCatalog(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to save imported catalog: %w", err)
	}

	// Store in database
	if err := s.repo.CreateCatalog(ctx, adminID, catalog); err != nil {
		return nil, fmt.Errorf("failed to persist imported catalog: %w", err)
	}

	// Invalidate all user catalog caches
	s.Inheritance.ClearCache()

	return catalog, nil
}

// ExportUserCatalog exports a user's resolved catalog
func (s *MultiUserCatalogService) ExportUserCatalog(
	ctx context.Context,
	userID string,
	format string,
) ([]byte, error) {
	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get resolved catalog for user
	resolved, err := s.Inheritance.ResolveForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve catalog: %w", err)
	}

	// Export using file manager
	data, err := s.FileManager.ExportCatalog(ctx, resolved.MergedCatalog, format)
	if err != nil {
		return nil, fmt.Errorf("failed to export catalog: %w", err)
	}

	return data, nil
}

// isAdmin checks if a user has admin privileges
func (s *MultiUserCatalogService) isAdmin(ctx context.Context, userID string) bool {
	// In production, this would check against the user's role in the database
	// For now, we'll use a simple check (could be enhanced with proper RBAC)

	// Check if user has admin role in context
	if role, ok := ctx.Value("user_role").(string); ok {
		isAdmin := role == "admin" || role == "super_admin"

		// Audit log admin access checks
		if s.audit != nil {
			uid, _ := uuid.Parse(userID)
			s.audit.LogSecurityEvent(
				ctx,
				uid,
				audit.EventTypeAuthentication,
				map[string]interface{}{
					"action":   "admin_check",
					"user_id":  userID,
					"role":     role,
					"is_admin": isAdmin,
				},
			)
		}

		return isAdmin
	}

	// Default to false for safety
	return false
}

// clearUserCache is a helper to clear cache for a specific user
func (ie *InheritanceEngine) clearUserCache(userID string) {
	ie.cache.mu.Lock()
	defer ie.cache.mu.Unlock()
	delete(ie.cache.resolved, userID)
}

// Request/Response types for multi-user operations

// UserCatalogCustomizationRequest represents a user's catalog customization request
type UserCatalogCustomizationRequest struct {
	EnabledServers []string        `json:"enabled_servers,omitempty"`
	CustomServers  []*ServerConfig `json:"custom_servers,omitempty"`
}

// UserCatalogCustomizations represents a user's current customizations
type UserCatalogCustomizations struct {
	UserID          string          `json:"user_id"`
	EnabledServers  []string        `json:"enabled_servers"`
	DisabledServers []string        `json:"disabled_servers"`
	CustomServers   []*ServerConfig `json:"custom_servers"`
	LastUpdated     time.Time       `json:"last_updated"`
}
