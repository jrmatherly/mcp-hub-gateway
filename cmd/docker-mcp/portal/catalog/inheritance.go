package catalog

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InheritanceEngine manages catalog inheritance and merging for multi-user support
type InheritanceEngine struct {
	fileManager *FileCatalogManager
	repository  Repository
	cache       *inheritanceCache
	mu          sync.RWMutex
}

// inheritanceCache caches resolved catalogs for users
type inheritanceCache struct {
	mu       sync.RWMutex
	resolved map[string]*resolvedEntry
	ttl      time.Duration
}

// resolvedEntry represents a cached resolved catalog
type resolvedEntry struct {
	catalog   *ResolvedCatalog
	timestamp time.Time
}

// ResolvedCatalog represents a fully resolved catalog for a user
type ResolvedCatalog struct {
	Catalog         *Catalog
	Sources         []CatalogSource
	ResolutionTime  time.Duration
	ConflictDetails []ConflictDetail
}

// CatalogSource represents a source that contributed to the resolved catalog
type CatalogSource struct {
	Type        CatalogType
	Name        string
	Precedence  int
	ServerCount int
}

// ConflictDetail represents a conflict resolution detail
type ConflictDetail struct {
	ServerName       string
	WinningSource    string
	OverriddenSource string
	Reason           string
}

// InheritanceConfig configuration for the inheritance engine
type InheritanceConfig struct {
	FileManager *FileCatalogManager
	Repository  Repository
	CacheTTL    time.Duration
}

// NewInheritanceEngine creates a new catalog inheritance engine
func NewInheritanceEngine(config InheritanceConfig) (*InheritanceEngine, error) {
	if config.FileManager == nil {
		return nil, fmt.Errorf("file manager is required")
	}
	if config.Repository == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &InheritanceEngine{
		fileManager: config.FileManager,
		repository:  config.Repository,
		cache: &inheritanceCache{
			resolved: make(map[string]*resolvedEntry),
			ttl:      config.CacheTTL,
		},
	}, nil
}

// ResolveForUser resolves the complete catalog for a specific user
func (e *InheritanceEngine) ResolveForUser(
	ctx context.Context,
	userID uuid.UUID,
) (*ResolvedCatalog, error) {
	startTime := time.Now()

	// Check cache
	e.cache.mu.RLock()
	if cached, ok := e.cache.resolved[userID.String()]; ok {
		if time.Since(cached.timestamp) < e.cache.ttl {
			e.cache.mu.RUnlock()
			return cached.catalog, nil
		}
	}
	e.cache.mu.RUnlock()

	// Collect all catalogs in precedence order
	catalogLayers, err := e.collectCatalogLayers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to collect catalog layers: %w", err)
	}

	// Apply precedence rules and merge
	merged, conflicts := e.applyPrecedenceRules(catalogLayers)

	// Validate the merged catalog
	if err := e.validateMergedCatalog(merged); err != nil {
		return nil, fmt.Errorf("catalog validation failed: %w", err)
	}

	// Create resolved catalog
	resolved := &ResolvedCatalog{
		Catalog:         merged,
		Sources:         e.extractSources(catalogLayers),
		ResolutionTime:  time.Since(startTime),
		ConflictDetails: conflicts,
	}

	// Cache the result
	e.cache.mu.Lock()
	e.cache.resolved[userID.String()] = &resolvedEntry{
		catalog:   resolved,
		timestamp: time.Now(),
	}
	e.cache.mu.Unlock()

	return resolved, nil
}

// catalogLayer represents a catalog with its precedence information
type catalogLayer struct {
	catalog    *Catalog
	precedence int
	source     string
	isBase     bool
}

// collectCatalogLayers collects all applicable catalog layers for a user
func (e *InheritanceEngine) collectCatalogLayers(
	ctx context.Context,
	userID uuid.UUID,
) ([]*catalogLayer, error) {
	var layers []*catalogLayer

	// 1. System default catalog (lowest precedence)
	systemCatalog, err := e.repository.GetSystemDefaultCatalog(ctx)
	if err == nil && systemCatalog != nil {
		layers = append(layers, &catalogLayer{
			catalog:    systemCatalog,
			precedence: 1000,
			source:     "system_default",
			isBase:     true,
		})
	}

	// 2. Admin base catalogs (medium precedence)
	baseCatalogs, err := e.repository.GetAdminBaseCatalogs(ctx)
	if err == nil {
		for i, catalog := range baseCatalogs {
			layers = append(layers, &catalogLayer{
				catalog:    catalog,
				precedence: 500 - i, // Higher index = lower precedence
				source:     fmt.Sprintf("admin_base_%d", i),
				isBase:     true,
			})
		}
	}

	// 3. Team catalogs if user belongs to teams (higher precedence)
	teamCatalogs, err := e.repository.GetTeamCatalogsForUser(ctx, userID)
	if err == nil {
		for i, catalog := range teamCatalogs {
			layers = append(layers, &catalogLayer{
				catalog:    catalog,
				precedence: 200 - i,
				source:     fmt.Sprintf("team_%s", catalog.Name),
				isBase:     false,
			})
		}
	}

	// 4. User personal catalog (highest precedence for additions)
	userCatalog, err := e.fileManager.LoadUserCatalog(ctx, userID.String())
	if err == nil && userCatalog != nil && len(userCatalog.Registry) > 0 {
		layers = append(layers, &catalogLayer{
			catalog:    userCatalog,
			precedence: 50,
			source:     "user_personal",
			isBase:     false,
		})
	}

	// 5. User customizations (highest precedence for overrides/disables)
	customizations, err := e.repository.GetUserCustomizations(ctx, userID)
	if err == nil && customizations != nil {
		layers = append(layers, &catalogLayer{
			catalog:    customizations,
			precedence: 10,
			source:     "user_customizations",
			isBase:     false,
		})
	}

	// Sort by precedence (lower number = higher priority)
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].precedence < layers[j].precedence
	})

	return layers, nil
}

// applyPrecedenceRules applies precedence rules to merge catalog layers
func (e *InheritanceEngine) applyPrecedenceRules(
	layers []*catalogLayer,
) (*Catalog, []ConflictDetail) {
	merged := &Catalog{
		Name:        "resolved",
		DisplayName: "Resolved Catalog",
		Type:        CatalogType("resolved"),
		Registry:    make(map[string]*ServerConfig),
		Metadata:    make(map[string]interface{}),
	}

	var conflicts []ConflictDetail
	serverSources := make(map[string]string) // Track which source each server came from

	// Process layers in reverse order (lowest precedence first)
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]

		// Handle disabled servers
		if layer.catalog.DisabledServers != nil {
			for serverName := range layer.catalog.DisabledServers {
				// Check if server is mandatory
				if existing, exists := merged.Registry[serverName]; exists {
					if mandatory, ok := existing.Metadata["mandatory"].(bool); ok && mandatory {
						// Cannot disable mandatory servers
						conflicts = append(conflicts, ConflictDetail{
							ServerName:       serverName,
							WinningSource:    serverSources[serverName],
							OverriddenSource: layer.source,
							Reason:           "Cannot disable mandatory server",
						})
						continue
					}
				}
				// Remove the server
				delete(merged.Registry, serverName)
				delete(serverSources, serverName)
			}
		}

		// Add/override servers
		for serverName, serverConfig := range layer.catalog.Registry {
			if _, exists := merged.Registry[serverName]; exists {
				// Server already exists - check if this layer has higher precedence
				conflicts = append(conflicts, ConflictDetail{
					ServerName:       serverName,
					WinningSource:    layer.source,
					OverriddenSource: serverSources[serverName],
					Reason:           fmt.Sprintf("Higher precedence (%d)", layer.precedence),
				})
			}

			// Add or override the server
			merged.Registry[serverName] = serverConfig
			serverSources[serverName] = layer.source
		}
	}

	// Update server count
	merged.ServerCount = len(merged.Registry)

	return merged, conflicts
}

// validateMergedCatalog validates the merged catalog
func (e *InheritanceEngine) validateMergedCatalog(catalog *Catalog) error {
	if catalog == nil {
		return fmt.Errorf("catalog is nil")
	}

	if len(catalog.Registry) == 0 {
		// Empty catalog is valid but might want to warn
		return nil
	}

	// Validate each server
	for name, server := range catalog.Registry {
		if server.Image == "" {
			return fmt.Errorf("server %s: image is required", name)
		}

		// Check for required metadata
		if server.Metadata == nil {
			server.Metadata = make(map[string]interface{})
		}
	}

	return nil
}

// extractSources extracts source information from catalog layers
func (e *InheritanceEngine) extractSources(layers []*catalogLayer) []CatalogSource {
	var sources []CatalogSource

	for _, layer := range layers {
		serverCount := 0
		if layer.catalog.Registry != nil {
			serverCount = len(layer.catalog.Registry)
		}

		sources = append(sources, CatalogSource{
			Type:        layer.catalog.Type,
			Name:        layer.catalog.Name,
			Precedence:  layer.precedence,
			ServerCount: serverCount,
		})
	}

	return sources
}

// GetInheritanceChain returns the inheritance chain for a user
func (e *InheritanceEngine) GetInheritanceChain(
	ctx context.Context,
	userID uuid.UUID,
) ([]CatalogSource, error) {
	layers, err := e.collectCatalogLayers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to collect catalog layers: %w", err)
	}

	return e.extractSources(layers), nil
}

// ClearUserCache clears the cache for a specific user
func (e *InheritanceEngine) ClearUserCache(userID uuid.UUID) {
	e.cache.mu.Lock()
	defer e.cache.mu.Unlock()
	delete(e.cache.resolved, userID.String())
}

// ClearAllCache clears the entire cache
func (e *InheritanceEngine) ClearAllCache() {
	e.cache.mu.Lock()
	defer e.cache.mu.Unlock()
	e.cache.resolved = make(map[string]*resolvedEntry)
}

// RefreshUserCatalog forces a refresh of a user's catalog
func (e *InheritanceEngine) RefreshUserCatalog(
	ctx context.Context,
	userID uuid.UUID,
) (*ResolvedCatalog, error) {
	// Clear cache first
	e.ClearUserCache(userID)

	// Then resolve
	return e.ResolveForUser(ctx, userID)
}

// Repository interface methods that need to be added to the existing repository
type Repository interface {
	// Existing methods...

	// New methods for inheritance
	GetSystemDefaultCatalog(ctx context.Context) (*Catalog, error)
	GetAdminBaseCatalogs(ctx context.Context) ([]*Catalog, error)
	GetTeamCatalogsForUser(ctx context.Context, userID uuid.UUID) ([]*Catalog, error)
	GetUserCustomizations(ctx context.Context, userID uuid.UUID) (*Catalog, error)
}
