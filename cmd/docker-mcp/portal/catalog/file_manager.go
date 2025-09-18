package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// FileCatalogManager manages file-based catalog storage without Docker Desktop dependency
type FileCatalogManager struct {
	basePath    string
	userPath    string
	cacheTTL    time.Duration
	mu          sync.RWMutex
	cache       map[string]*cachedCatalog
	encryptor   EncryptionService
	auditLogger AuditLogger
}

// cachedCatalog represents a catalog with cache metadata
type cachedCatalog struct {
	catalog  *Catalog
	loadedAt time.Time
	checksum string
}

// FileCatalogConfig configuration for FileCatalogManager
type FileCatalogConfig struct {
	BasePath    string
	UserPath    string
	CacheTTL    time.Duration
	Encryptor   EncryptionService
	AuditLogger AuditLogger
}

// EncryptionService interface for encrypting sensitive catalog data
type EncryptionService interface {
	Encrypt(ctx context.Context, data []byte) ([]byte, error)
	Decrypt(ctx context.Context, data []byte) ([]byte, error)
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	LogCatalogOperation(
		ctx context.Context,
		userID, operation string,
		details map[string]interface{},
	) error
}

// NewFileCatalogManager creates a new file-based catalog manager
func NewFileCatalogManager(config FileCatalogConfig) (*FileCatalogManager, error) {
	if config.BasePath == "" {
		config.BasePath = "/app/data/catalogs/base"
	}
	if config.UserPath == "" {
		config.UserPath = "/app/user-catalogs"
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	// Ensure directories exist
	if err := os.MkdirAll(config.BasePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base catalog directory: %w", err)
	}
	if err := os.MkdirAll(config.UserPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create user catalog directory: %w", err)
	}

	return &FileCatalogManager{
		basePath:    config.BasePath,
		userPath:    config.UserPath,
		cacheTTL:    config.CacheTTL,
		cache:       make(map[string]*cachedCatalog),
		encryptor:   config.Encryptor,
		auditLogger: config.AuditLogger,
	}, nil
}

// CreateFileCatalogManager creates a new file-based catalog manager (constructor alias)
func CreateFileCatalogManager(
	basePath, userPath string,
	encryptor EncryptionService,
	cacheTTL time.Duration,
) (*FileCatalogManager, error) {
	return NewFileCatalogManager(FileCatalogConfig{
		BasePath:  basePath,
		UserPath:  userPath,
		CacheTTL:  cacheTTL,
		Encryptor: encryptor,
	})
}

// LoadBaseCatalog loads all admin-controlled base catalogs
func (m *FileCatalogManager) LoadBaseCatalog(ctx context.Context) (*Catalog, error) {
	m.mu.RLock()
	cacheKey := "base:all"
	if cached, ok := m.cache[cacheKey]; ok && !m.isCacheExpired(cached) {
		m.mu.RUnlock()
		return cached.catalog, nil
	}
	m.mu.RUnlock()

	// Load all YAML files from base path
	baseCatalog := &Catalog{
		Name:        "organization-base",
		DisplayName: "Organization Base Catalog",
		Registry:    make(map[string]*ServerConfig),
		Metadata: map[string]interface{}{
			"type":      "admin_base",
			"loaded_at": time.Now().UTC(),
			"source":    m.basePath,
		},
	}

	files, err := filepath.Glob(filepath.Join(m.basePath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list base catalog files: %w", err)
	}

	for _, file := range files {
		catalog, err := m.loadCatalogFile(ctx, file)
		if err != nil {
			// Log error but continue loading other files
			if m.auditLogger != nil {
				m.auditLogger.LogCatalogOperation(
					ctx,
					"system",
					"load_base_failed",
					map[string]interface{}{
						"file":  file,
						"error": err.Error(),
					},
				)
			}
			continue
		}

		// Merge into base catalog
		for name, server := range catalog.Registry {
			baseCatalog.Registry[name] = server
		}
	}

	// Cache the result
	m.mu.Lock()
	m.cache[cacheKey] = &cachedCatalog{
		catalog:  baseCatalog,
		loadedAt: time.Now(),
	}
	m.mu.Unlock()

	return baseCatalog, nil
}

// LoadUserCatalog loads a specific user's personal catalog
func (m *FileCatalogManager) LoadUserCatalog(ctx context.Context, userID string) (*Catalog, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	m.mu.RLock()
	cacheKey := fmt.Sprintf("user:%s", userID)
	if cached, ok := m.cache[cacheKey]; ok && !m.isCacheExpired(cached) {
		m.mu.RUnlock()
		return cached.catalog, nil
	}
	m.mu.RUnlock()

	userFile := filepath.Join(m.userPath, userID, "personal.yaml")

	// Check if user catalog exists
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		// Return empty catalog if doesn't exist
		return &Catalog{
			Name:        fmt.Sprintf("user-%s", userID),
			DisplayName: "Personal Catalog",
			Registry:    make(map[string]*ServerConfig),
			Metadata: map[string]interface{}{
				"type":    "user_personal",
				"user_id": userID,
			},
		}, nil
	}

	catalog, err := m.loadCatalogFile(ctx, userFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load user catalog: %w", err)
	}

	// Add metadata
	if catalog.Metadata == nil {
		catalog.Metadata = make(map[string]interface{})
	}
	catalog.Metadata["type"] = "user_personal"
	catalog.Metadata["user_id"] = userID

	// Cache the result
	m.mu.Lock()
	m.cache[cacheKey] = &cachedCatalog{
		catalog:  catalog,
		loadedAt: time.Now(),
	}
	m.mu.Unlock()

	return catalog, nil
}

// SaveUserCatalog saves a user's personal catalog
func (m *FileCatalogManager) SaveUserCatalog(
	ctx context.Context,
	userID string,
	catalog *Catalog,
) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if catalog == nil {
		return fmt.Errorf("catalog is required")
	}

	// Create user directory if it doesn't exist
	userDir := filepath.Join(m.userPath, userID)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	userFile := filepath.Join(userDir, "personal.yaml")

	// Convert catalog to YAML
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Encrypt sensitive data if encryptor is available
	if m.encryptor != nil {
		encrypted, err := m.encryptor.Encrypt(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to encrypt catalog: %w", err)
		}
		data = encrypted
	}

	// Write to file atomically
	tmpFile := userFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write catalog file: %w", err)
	}

	if err := os.Rename(tmpFile, userFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	// Invalidate cache
	m.mu.Lock()
	delete(m.cache, fmt.Sprintf("user:%s", userID))
	m.mu.Unlock()

	// Audit log
	if m.auditLogger != nil {
		m.auditLogger.LogCatalogOperation(ctx, userID, "save_user_catalog", map[string]interface{}{
			"catalog_name": catalog.Name,
			"server_count": len(catalog.Registry),
		})
	}

	return nil
}

// MergeCatalogs merges base and user catalogs according to precedence rules
func (m *FileCatalogManager) MergeCatalogs(
	ctx context.Context,
	base, user *Catalog,
) (*Catalog, error) {
	if base == nil {
		return user, nil
	}
	if user == nil {
		return base, nil
	}

	merged := &Catalog{
		Name:        "merged",
		DisplayName: "Merged Catalog",
		Registry:    make(map[string]*ServerConfig),
		Metadata: map[string]interface{}{
			"type":      "merged",
			"merged_at": time.Now().UTC(),
		},
	}

	// Start with base catalog servers
	for name, server := range base.Registry {
		// Check if server is mandatory (cannot be disabled)
		if mandatory, ok := server.Metadata["mandatory"].(bool); ok && mandatory {
			merged.Registry[name] = server
			continue
		}

		// Check if user has disabled this server
		if user.DisabledServers != nil {
			if _, disabled := user.DisabledServers[name]; disabled {
				continue // Skip disabled servers
			}
		}

		// Check if user has overridden this server
		if userServer, exists := user.Registry[name]; exists {
			merged.Registry[name] = userServer // User override takes precedence
		} else {
			merged.Registry[name] = server // Use base server
		}
	}

	// Add user's custom servers (not in base)
	for name, server := range user.Registry {
		if _, exists := merged.Registry[name]; !exists {
			merged.Registry[name] = server
		}
	}

	return merged, nil
}

// DeleteUserCatalog deletes a user's personal catalog
func (m *FileCatalogManager) DeleteUserCatalog(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	userDir := filepath.Join(m.userPath, userID)

	// Remove the entire user directory
	if err := os.RemoveAll(userDir); err != nil {
		return fmt.Errorf("failed to delete user catalog: %w", err)
	}

	// Invalidate cache
	m.mu.Lock()
	delete(m.cache, fmt.Sprintf("user:%s", userID))
	m.mu.Unlock()

	// Audit log
	if m.auditLogger != nil {
		m.auditLogger.LogCatalogOperation(ctx, userID, "delete_user_catalog", nil)
	}

	return nil
}

// ImportCatalog imports a catalog from YAML or JSON data
func (m *FileCatalogManager) ImportCatalog(
	ctx context.Context,
	data []byte,
	format string,
) (*Catalog, error) {
	var catalog Catalog

	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Validate the imported catalog
	if err := m.validateCatalog(&catalog); err != nil {
		return nil, fmt.Errorf("catalog validation failed: %w", err)
	}

	return &catalog, nil
}

// ExportCatalog exports a catalog to YAML or JSON format
func (m *FileCatalogManager) ExportCatalog(
	ctx context.Context,
	catalog *Catalog,
	format string,
) ([]byte, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is required")
	}

	switch format {
	case "yaml", "yml":
		return yaml.Marshal(catalog)
	case "json":
		return json.MarshalIndent(catalog, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// SaveBaseCatalog saves a base catalog to the file system
func (m *FileCatalogManager) SaveBaseCatalog(ctx context.Context, catalog *Catalog) error {
	if catalog == nil {
		return fmt.Errorf("catalog is nil")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Create file path
	filename := fmt.Sprintf("%s.yaml", catalog.Name)
	path := filepath.Join(m.basePath, filename)

	// Write atomically
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write catalog file: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	// Clear cache
	m.mu.Lock()
	delete(m.cache, "base:all")
	m.mu.Unlock()

	// Audit log
	if m.auditLogger != nil {
		m.auditLogger.LogCatalogOperation(
			ctx,
			"system",
			"save_base_catalog",
			map[string]interface{}{
				"catalog_name": catalog.Name,
				"path":         path,
			},
		)
	}

	return nil
}

// Helper methods

func (m *FileCatalogManager) loadCatalogFile(ctx context.Context, path string) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open catalog file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog file: %w", err)
	}

	// Try to decrypt if data appears to be encrypted
	if m.encryptor != nil && m.isEncrypted(data) {
		decrypted, err := m.encryptor.Decrypt(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt catalog: %w", err)
		}
		data = decrypted
	}

	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal catalog: %w", err)
	}

	return &catalog, nil
}

func (m *FileCatalogManager) isCacheExpired(cached *cachedCatalog) bool {
	return time.Since(cached.loadedAt) > m.cacheTTL
}

func (m *FileCatalogManager) isEncrypted(data []byte) bool {
	// Simple heuristic: encrypted data typically doesn't start with valid YAML/JSON
	if len(data) < 4 {
		return false
	}
	// Check for YAML document start
	if string(data[:3]) == "---" || data[0] == '#' {
		return false
	}
	// Check for JSON object/array start
	if data[0] == '{' || data[0] == '[' {
		return false
	}
	return true
}

func (m *FileCatalogManager) validateCatalog(catalog *Catalog) error {
	if catalog.Name == "" {
		return fmt.Errorf("catalog name is required")
	}

	for name, server := range catalog.Registry {
		if name == "" {
			return fmt.Errorf("server name cannot be empty")
		}
		if server.Image == "" {
			return fmt.Errorf("server %s: image is required", name)
		}
	}

	return nil
}

// ClearCache clears the in-memory cache
func (m *FileCatalogManager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]*cachedCatalog)
}
