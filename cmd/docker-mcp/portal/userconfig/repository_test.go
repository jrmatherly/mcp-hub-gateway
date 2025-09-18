package userconfig

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

type UserConfigRepositoryTestSuite struct {
	suite.Suite

	// Repository under test
	repo UserConfigRepository

	// Test infrastructure
	pgContainer testcontainers.Container
	encryption  crypto.Encryption
	ctx         context.Context

	// Test data
	testUserID   string
	testConfigID uuid.UUID
	testConfig   *UserConfig
}

func TestUserConfigRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserConfigRepositoryTestSuite))
}

func (s *UserConfigRepositoryTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Start PostgreSQL test container
	pgContainer, err := postgres.RunContainer(
		s.ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	// Get connection string and initialize database
	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Initialize database with connection string
	err = database.InitializeWithConnectionString(connStr)
	s.Require().NoError(err)

	// Run migrations
	err = database.RunMigrationsSimple()
	s.Require().NoError(err)

	// Create encryption service
	key := []byte("test-key-32-bytes-for-aes-256!!")
	s.encryption, err = crypto.CreateEncryption(key)
	s.Require().NoError(err)

	// Create repository
	s.repo, err = CreateUserConfigRepository(s.encryption)
	s.Require().NoError(err)
}

func (s *UserConfigRepositoryTestSuite) TearDownSuite() {
	if s.pgContainer != nil {
		err := s.pgContainer.Terminate(s.ctx)
		s.Require().NoError(err)
	}
}

func (s *UserConfigRepositoryTestSuite) SetupTest() {
	s.testUserID = uuid.New().String()
	s.testConfigID = uuid.New()

	s.testConfig = &UserConfig{
		ID:          s.testConfigID,
		Name:        "test-config",
		DisplayName: "Test Configuration",
		Description: "Test configuration for repository tests",
		Type:        ConfigTypePersonal,
		Status:      ConfigStatusActive,
		OwnerID:     uuid.MustParse(s.testUserID),
		TenantID:    "test-tenant",
		IsDefault:   false,
		IsActive:    true,
		Version:     "1.0.0",
		Settings: map[string]any{
			"theme":       "dark",
			"language":    "en",
			"debugMode":   true,
			"secretToken": "secret-123",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (s *UserConfigRepositoryTestSuite) TearDownTest() {
	// Clean up test data
	_ = s.repo.DeleteConfig(s.ctx, s.testUserID, s.testConfigID)
}

// Test CreateConfig - Success
func (s *UserConfigRepositoryTestSuite) TestCreateConfig_Success() {
	// Act
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)

	// Assert
	s.NoError(err)

	// Verify the config was created
	retrievedConfig, err := s.repo.GetConfig(s.ctx, s.testUserID, s.testConfigID)
	s.NoError(err)
	s.NotNil(retrievedConfig)
	s.Equal(s.testConfig.Name, retrievedConfig.Name)
	s.Equal(s.testConfig.Type, retrievedConfig.Type)
	s.Equal(s.testConfig.Settings["theme"], retrievedConfig.Settings["theme"])
}

// Test CreateConfig - Duplicate Name
func (s *UserConfigRepositoryTestSuite) TestCreateConfig_DuplicateName() {
	// Arrange - create first config
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)
	s.NoError(err)

	// Try to create another config with same name
	duplicateConfig := &UserConfig{
		ID:       uuid.New(),
		Name:     s.testConfig.Name, // Same name
		Type:     ConfigTypeTeam,
		OwnerID:  uuid.MustParse(s.testUserID),
		TenantID: "test-tenant",
		Settings: map[string]any{"test": "value"},
	}

	// Act
	err = s.repo.CreateConfig(s.ctx, s.testUserID, duplicateConfig)

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "duplicate key value")
}

// Test GetConfig - Success
func (s *UserConfigRepositoryTestSuite) TestGetConfig_Success() {
	// Arrange
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)
	s.NoError(err)

	// Act
	retrievedConfig, err := s.repo.GetConfig(s.ctx, s.testUserID, s.testConfigID)

	// Assert
	s.NoError(err)
	s.NotNil(retrievedConfig)
	s.Equal(s.testConfig.ID, retrievedConfig.ID)
	s.Equal(s.testConfig.Name, retrievedConfig.Name)
	s.Equal(s.testConfig.DisplayName, retrievedConfig.DisplayName)
	s.Equal(s.testConfig.Type, retrievedConfig.Type)
	s.Equal(s.testConfig.Status, retrievedConfig.Status)

	// Verify settings were encrypted and decrypted correctly
	s.Equal(s.testConfig.Settings["theme"], retrievedConfig.Settings["theme"])
	s.Equal(s.testConfig.Settings["secretToken"], retrievedConfig.Settings["secretToken"])
}

// Test GetConfig - Not Found
func (s *UserConfigRepositoryTestSuite) TestGetConfig_NotFound() {
	// Act
	retrievedConfig, err := s.repo.GetConfig(s.ctx, s.testUserID, uuid.New())

	// Assert
	s.Error(err)
	s.Nil(retrievedConfig)
	s.Contains(err.Error(), "not found")
}

// Test GetConfig - Wrong User (RLS Test)
func (s *UserConfigRepositoryTestSuite) TestGetConfig_WrongUser() {
	// Arrange
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)
	s.NoError(err)

	differentUserID := uuid.New().String()

	// Act
	retrievedConfig, err := s.repo.GetConfig(s.ctx, differentUserID, s.testConfigID)

	// Assert
	s.Error(err)
	s.Nil(retrievedConfig)
}

// Test UpdateConfig - Success
func (s *UserConfigRepositoryTestSuite) TestUpdateConfig_Success() {
	// Arrange
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)
	s.NoError(err)

	// Modify config
	s.testConfig.DisplayName = "Updated Display Name"
	s.testConfig.Description = "Updated description"
	s.testConfig.Settings["theme"] = "light"
	s.testConfig.Settings["newField"] = "new value"
	s.testConfig.UpdatedAt = time.Now().UTC()

	// Act
	err = s.repo.UpdateConfig(s.ctx, s.testUserID, s.testConfig)

	// Assert
	s.NoError(err)

	// Verify update
	updatedConfig, err := s.repo.GetConfig(s.ctx, s.testUserID, s.testConfigID)
	s.NoError(err)
	s.Equal("Updated Display Name", updatedConfig.DisplayName)
	s.Equal("Updated description", updatedConfig.Description)
	s.Equal("light", updatedConfig.Settings["theme"])
	s.Equal("new value", updatedConfig.Settings["newField"])
}

// Test UpdateConfig - Not Found
func (s *UserConfigRepositoryTestSuite) TestUpdateConfig_NotFound() {
	// Arrange
	nonExistentConfig := &UserConfig{
		ID:       uuid.New(),
		OwnerID:  uuid.MustParse(s.testUserID),
		Settings: map[string]any{"test": "value"},
	}

	// Act
	err := s.repo.UpdateConfig(s.ctx, s.testUserID, nonExistentConfig)

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

// Test DeleteConfig - Success
func (s *UserConfigRepositoryTestSuite) TestDeleteConfig_Success() {
	// Arrange
	err := s.repo.CreateConfig(s.ctx, s.testUserID, s.testConfig)
	s.NoError(err)

	// Act
	err = s.repo.DeleteConfig(s.ctx, s.testUserID, s.testConfigID)

	// Assert
	s.NoError(err)

	// Verify deletion
	_, err = s.repo.GetConfig(s.ctx, s.testUserID, s.testConfigID)
	s.Error(err)
}

// Test DeleteConfig - Not Found
func (s *UserConfigRepositoryTestSuite) TestDeleteConfig_NotFound() {
	// Act
	err := s.repo.DeleteConfig(s.ctx, s.testUserID, uuid.New())

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

// Test ListConfigs - No Filter
func (s *UserConfigRepositoryTestSuite) TestListConfigs_NoFilter() {
	// Arrange - create multiple configs
	configs := []*UserConfig{
		{
			ID:       uuid.New(),
			Name:     "config1",
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"test": "value1"},
		},
		{
			ID:       uuid.New(),
			Name:     "config2",
			Type:     ConfigTypeTeam,
			Status:   ConfigStatusDraft,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"test": "value2"},
		},
		{
			ID:       uuid.New(),
			Name:     "config3",
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"test": "value3"},
		},
	}

	for _, config := range configs {
		err := s.repo.CreateConfig(s.ctx, s.testUserID, config)
		s.NoError(err)
	}

	// Act
	results, err := s.repo.ListConfigs(s.ctx, s.testUserID, ConfigFilter{})

	// Assert
	s.NoError(err)
	s.Len(results, 3)

	// Clean up
	for _, config := range configs {
		_ = s.repo.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}

// Test ListConfigs - With Filters
func (s *UserConfigRepositoryTestSuite) TestListConfigs_WithFilters() {
	// Arrange - create configs with different types and statuses
	configs := []*UserConfig{
		{
			ID:       uuid.New(),
			Name:     "personal-active",
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			IsActive: true,
			Settings: map[string]any{"test": "value1"},
		},
		{
			ID:       uuid.New(),
			Name:     "personal-draft",
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusDraft,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			IsActive: false,
			Settings: map[string]any{"test": "value2"},
		},
		{
			ID:       uuid.New(),
			Name:     "team-active",
			Type:     ConfigTypeTeam,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			IsActive: true,
			Settings: map[string]any{"test": "value3"},
		},
	}

	for _, config := range configs {
		err := s.repo.CreateConfig(s.ctx, s.testUserID, config)
		s.NoError(err)
	}

	// Test filter by type
	filter := ConfigFilter{Type: ConfigTypePersonal}
	results, err := s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 2)

	// Test filter by status
	filter = ConfigFilter{Status: ConfigStatusActive}
	results, err = s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 2)

	// Test filter by active status
	isActive := true
	filter = ConfigFilter{IsActive: &isActive}
	results, err = s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 2)

	// Test combined filters
	filter = ConfigFilter{
		Type:     ConfigTypePersonal,
		Status:   ConfigStatusActive,
		IsActive: &isActive,
	}
	results, err = s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal("personal-active", results[0].Name)

	// Clean up
	for _, config := range configs {
		_ = s.repo.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}

// Test ListConfigs - Pagination
func (s *UserConfigRepositoryTestSuite) TestListConfigs_Pagination() {
	// Arrange - create 5 configs
	configs := make([]*UserConfig, 5)
	for i := 0; i < 5; i++ {
		configs[i] = &UserConfig{
			ID:       uuid.New(),
			Name:     fmt.Sprintf("config%d", i),
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"index": i},
		}
		err := s.repo.CreateConfig(s.ctx, s.testUserID, configs[i])
		s.NoError(err)
	}

	// Test first page
	filter := ConfigFilter{Limit: 2, Offset: 0}
	results, err := s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 2)

	// Test second page
	filter = ConfigFilter{Limit: 2, Offset: 2}
	results, err = s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 2)

	// Test last page
	filter = ConfigFilter{Limit: 2, Offset: 4}
	results, err = s.repo.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.Len(results, 1)

	// Clean up
	for _, config := range configs {
		_ = s.repo.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}

// Test CountConfigs
func (s *UserConfigRepositoryTestSuite) TestCountConfigs() {
	// Arrange - create configs
	configs := []*UserConfig{
		{
			ID:       uuid.New(),
			Name:     "count-test1",
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"test": "value1"},
		},
		{
			ID:       uuid.New(),
			Name:     "count-test2",
			Type:     ConfigTypeTeam,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{"test": "value2"},
		},
	}

	for _, config := range configs {
		err := s.repo.CreateConfig(s.ctx, s.testUserID, config)
		s.NoError(err)
	}

	// Test total count
	count, err := s.repo.CountConfigs(s.ctx, s.testUserID, ConfigFilter{})
	s.NoError(err)
	s.Equal(int64(2), count)

	// Test filtered count
	count, err = s.repo.CountConfigs(s.ctx, s.testUserID, ConfigFilter{Type: ConfigTypePersonal})
	s.NoError(err)
	s.Equal(int64(1), count)

	// Clean up
	for _, config := range configs {
		_ = s.repo.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}

// Test Server Configuration Operations
func (s *UserConfigRepositoryTestSuite) TestServerConfig_CRUD() {
	serverName := "test-server"
	serverConfig := &ServerConfig{
		ServerID: serverName,
		Config: map[string]any{
			"endpoint": "http://localhost:8080",
			"apiKey":   "secret-key",
		},
		Metadata: map[string]any{
			"version":     "1.0.0",
			"description": "Test server configuration",
		},
		Status: "configured",
	}

	// Test Create/Save
	err := s.repo.SaveServerConfig(s.ctx, s.testUserID, serverConfig)
	s.NoError(err)

	// Test Get
	retrievedConfig, err := s.repo.GetServerConfig(s.ctx, s.testUserID, serverName)
	s.NoError(err)
	s.NotNil(retrievedConfig)
	s.Equal(serverName, retrievedConfig.ServerID)
	s.Equal("http://localhost:8080", retrievedConfig.Config["endpoint"])
	s.Equal("secret-key", retrievedConfig.Config["apiKey"])

	// Test Update (upsert)
	serverConfig.Config["endpoint"] = "http://localhost:9000"
	serverConfig.Status = "updated"
	err = s.repo.SaveServerConfig(s.ctx, s.testUserID, serverConfig)
	s.NoError(err)

	retrievedConfig, err = s.repo.GetServerConfig(s.ctx, s.testUserID, serverName)
	s.NoError(err)
	s.Equal("http://localhost:9000", retrievedConfig.Config["endpoint"])
	s.Equal("updated", retrievedConfig.Status)

	// Test List
	configs, err := s.repo.ListServerConfigs(s.ctx, s.testUserID)
	s.NoError(err)
	s.Len(configs, 1)
	s.Equal(serverName, configs[0].ServerID)

	// Test Delete
	err = s.repo.DeleteServerConfig(s.ctx, s.testUserID, serverName)
	s.NoError(err)

	// Verify deletion
	_, err = s.repo.GetServerConfig(s.ctx, s.testUserID, serverName)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

// Test Encryption/Decryption
func (s *UserConfigRepositoryTestSuite) TestEncryptionDecryption() {
	// Arrange - config with sensitive data
	sensitiveConfig := &UserConfig{
		ID:       uuid.New(),
		Name:     "sensitive-config",
		Type:     ConfigTypePersonal,
		Status:   ConfigStatusActive,
		OwnerID:  uuid.MustParse(s.testUserID),
		TenantID: "test-tenant",
		Settings: map[string]any{
			"apiKey":      "super-secret-api-key",
			"password":    "user-password-123",
			"token":       "jwt-token-value",
			"normalField": "this is not secret",
		},
	}

	// Act - save and retrieve
	err := s.repo.CreateConfig(s.ctx, s.testUserID, sensitiveConfig)
	s.NoError(err)

	retrievedConfig, err := s.repo.GetConfig(s.ctx, s.testUserID, sensitiveConfig.ID)
	s.NoError(err)

	// Assert - verify all data is correctly decrypted
	s.Equal("super-secret-api-key", retrievedConfig.Settings["apiKey"])
	s.Equal("user-password-123", retrievedConfig.Settings["password"])
	s.Equal("jwt-token-value", retrievedConfig.Settings["token"])
	s.Equal("this is not secret", retrievedConfig.Settings["normalField"])

	// Clean up
	_ = s.repo.DeleteConfig(s.ctx, s.testUserID, sensitiveConfig.ID)
}

// Benchmark tests
func (s *UserConfigRepositoryTestSuite) TestPerformance_CreateAndRetrieve() {
	configsToCreate := 100
	configs := make([]*UserConfig, configsToCreate)

	// Prepare configs
	for i := 0; i < configsToCreate; i++ {
		configs[i] = &UserConfig{
			ID:       uuid.New(),
			Name:     fmt.Sprintf("perf-config-%d", i),
			Type:     ConfigTypePersonal,
			Status:   ConfigStatusActive,
			OwnerID:  uuid.MustParse(s.testUserID),
			TenantID: "test-tenant",
			Settings: map[string]any{
				"index": i,
				"data":  fmt.Sprintf("data-value-%d", i),
			},
		}
	}

	// Test bulk create performance
	start := time.Now()
	for _, config := range configs {
		err := s.repo.CreateConfig(s.ctx, s.testUserID, config)
		s.NoError(err)
	}
	createDuration := time.Since(start)

	s.T().Logf("Created %d configs in %v (avg: %v per config)",
		configsToCreate, createDuration, createDuration/time.Duration(configsToCreate))

	// Test bulk retrieve performance
	start = time.Now()
	for _, config := range configs {
		_, err := s.repo.GetConfig(s.ctx, s.testUserID, config.ID)
		s.NoError(err)
	}
	retrieveDuration := time.Since(start)

	s.T().Logf("Retrieved %d configs in %v (avg: %v per config)",
		configsToCreate, retrieveDuration, retrieveDuration/time.Duration(configsToCreate))

	// Test list performance
	start = time.Now()
	results, err := s.repo.ListConfigs(s.ctx, s.testUserID, ConfigFilter{})
	s.NoError(err)
	listDuration := time.Since(start)

	s.T().Logf("Listed %d configs in %v", len(results), listDuration)
	s.Len(results, configsToCreate)

	// Performance assertions
	avgCreateTime := createDuration / time.Duration(configsToCreate)
	avgRetrieveTime := retrieveDuration / time.Duration(configsToCreate)

	s.Less(avgCreateTime, 50*time.Millisecond, "Create operation should be fast")
	s.Less(avgRetrieveTime, 20*time.Millisecond, "Retrieve operation should be fast")
	s.Less(listDuration, 100*time.Millisecond, "List operation should be fast")

	// Clean up
	for _, config := range configs {
		_ = s.repo.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}
