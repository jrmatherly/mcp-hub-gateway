package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Basic test to verify types are defined correctly
func TestCatalogTypes(t *testing.T) {
	// Test creating a basic catalog
	catalog := &Catalog{
		ID:          uuid.New(),
		Name:        "test-catalog",
		DisplayName: "Test Catalog",
		Type:        CatalogTypePersonal,
		Description: "Test catalog",
		Status:      CatalogStatusActive,
		Version:     "1.0.0",
		OwnerID:     uuid.New(),
		TenantID:    "default",
		Registry: map[string]*ServerConfig{
			"test-server": {
				Name:        "test-server",
				DisplayName: "Test Server",
				Image:       "test:latest",
				Command:     []string{"echo", "hello"},
				IsEnabled:   true,
			},
		},
		Metadata: map[string]interface{}{
			"created_by": "test",
			"created_at": time.Now().Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.NotNil(t, catalog)
	assert.Equal(t, "test-catalog", catalog.Name)
	assert.Equal(t, CatalogTypePersonal, catalog.Type)
	assert.NotEmpty(t, catalog.Registry)
	assert.Contains(t, catalog.Registry, "test-server")

	// Test server config
	serverConfig := catalog.Registry["test-server"]
	assert.Equal(t, "test-server", serverConfig.Name)
	assert.Equal(t, "test:latest", serverConfig.Image)
	assert.True(t, serverConfig.IsEnabled)
}

// Test InheritanceConfig structure
func TestInheritanceConfig(t *testing.T) {
	config := &InheritanceConfig{
		FileManager: nil, // Would be a real file manager in actual use
		Repository:  nil, // Would be a real repository in actual use
		CacheTTL:    5 * time.Minute,
	}

	assert.NotNil(t, config)
	assert.Equal(t, 5*time.Minute, config.CacheTTL)
}

// Test ResolvedCatalog structure
func TestResolvedCatalog(t *testing.T) {
	catalog := &Catalog{
		ID:       uuid.New(),
		Name:     "merged-catalog",
		Type:     CatalogType("resolved"),
		Registry: make(map[string]*ServerConfig),
	}

	resolved := &ResolvedCatalog{
		MergedCatalog:   catalog,
		AdminServers:    5,
		UserOverrides:   2,
		CustomServers:   3,
		Timestamp:       time.Now(),
		Sources:         []CatalogSource{},
		ResolutionTime:  time.Millisecond * 100,
		ConflictDetails: []ConflictDetail{},
	}

	assert.NotNil(t, resolved)
	assert.Equal(t, catalog, resolved.MergedCatalog)
	assert.Equal(t, 5, resolved.AdminServers)
	assert.Equal(t, 2, resolved.UserOverrides)
	assert.Equal(t, 3, resolved.CustomServers)
}

// Test CatalogSource structure
func TestCatalogSource(t *testing.T) {
	source := CatalogSource{
		Type:        CatalogTypeAdminBase,
		Name:        "admin-base",
		Precedence:  100,
		ServerCount: 10,
	}

	assert.Equal(t, CatalogTypeAdminBase, source.Type)
	assert.Equal(t, "admin-base", source.Name)
	assert.Equal(t, 100, source.Precedence)
	assert.Equal(t, 10, source.ServerCount)
}

// Test ConflictDetail structure
func TestConflictDetail(t *testing.T) {
	conflict := ConflictDetail{
		ServerName:       "test-server",
		WinningSource:    "user_personal",
		OverriddenSource: "admin_base",
		Reason:           "Higher precedence",
	}

	assert.Equal(t, "test-server", conflict.ServerName)
	assert.Equal(t, "user_personal", conflict.WinningSource)
	assert.Equal(t, "admin_base", conflict.OverriddenSource)
	assert.Equal(t, "Higher precedence", conflict.Reason)
}

// Test UserCatalogCustomizationRequest structure
func TestUserCatalogCustomizationRequest(t *testing.T) {
	req := &UserCatalogCustomizationRequest{
		EnabledServers: []string{"server1", "server2"},
		CustomServers: []*ServerConfig{
			{
				Name:        "custom-server",
				DisplayName: "Custom Server",
				Image:       "custom:latest",
				IsEnabled:   true,
			},
		},
	}

	assert.NotNil(t, req)
	assert.Len(t, req.EnabledServers, 2)
	assert.Len(t, req.CustomServers, 1)
	assert.Equal(t, "custom-server", req.CustomServers[0].Name)
}

// Test UserCatalogCustomizations structure
func TestUserCatalogCustomizations(t *testing.T) {
	customizations := &UserCatalogCustomizations{
		UserID:          "user123",
		EnabledServers:  []string{"server1", "server2"},
		DisabledServers: []string{"server3"},
		CustomServers: []*ServerConfig{
			{
				Name:        "custom-server",
				DisplayName: "Custom Server",
				Image:       "custom:latest",
				IsEnabled:   true,
			},
		},
		LastUpdated: time.Now(),
	}

	assert.NotNil(t, customizations)
	assert.Equal(t, "user123", customizations.UserID)
	assert.Len(t, customizations.EnabledServers, 2)
	assert.Len(t, customizations.DisabledServers, 1)
	assert.Len(t, customizations.CustomServers, 1)
}

// Test basic context operations
func TestBasicContextOperations(t *testing.T) {
	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Test context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	assert.NotNil(t, ctxWithTimeout)

	// Test context with value
	ctxWithValue := context.WithValue(ctx, "user_id", "test-user")
	assert.NotNil(t, ctxWithValue)
	assert.Equal(t, "test-user", ctxWithValue.Value("user_id"))
}
