package userconfig

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

// IntegrationTestSuite tests the complete UserConfig system end-to-end
type IntegrationTestSuite struct {
	suite.Suite

	// System under test
	service UserConfigService

	// Test infrastructure
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	ctx            context.Context

	// Dependencies
	repo        UserConfigRepository
	executor    executor.Executor
	auditLogger audit.Logger
	cacheStore  cache.Cache
	encryption  crypto.Encryption

	// Test data
	testUserID string
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Start PostgreSQL container
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

	// Start Redis container
	redisContainer, err := redis.RunContainer(s.ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections").
			WithStartupTimeout(10*time.Second)),
	)
	s.Require().NoError(err)
	s.redisContainer = redisContainer

	// Initialize database
	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	s.Require().NoError(err)

	err = database.InitializeWithConnectionString(connStr)
	s.Require().NoError(err)

	err = database.RunMigrationsSimple()
	s.Require().NoError(err)

	// Initialize Redis cache
	redisURL, err := redisContainer.ConnectionString(s.ctx)
	s.Require().NoError(err)

	s.cacheStore, err = cache.CreateRedisCache(redisURL, cache.DefaultTTL)
	s.Require().NoError(err)

	// Create encryption service
	key := []byte("integration-test-key-32-bytes!")
	s.encryption, err = crypto.CreateEncryption(key)
	s.Require().NoError(err)

	// Create audit logger
	s.auditLogger, err = audit.CreateAuditLogger(audit.Config{
		Enabled:    true,
		LogLevel:   "info",
		OutputPath: "stdout",
	})
	s.Require().NoError(err)

	// Create mock executor for CLI commands
	s.executor = executor.NewTestableExecutor()

	// Create repository
	s.repo, err = CreateUserConfigRepository(s.encryption)
	s.Require().NoError(err)

	// Create service
	s.service, err = CreateUserConfigService(
		s.repo,
		s.executor,
		s.auditLogger,
		s.cacheStore,
		s.encryption,
	)
	s.Require().NoError(err)

	s.testUserID = uuid.New().String()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}
	if s.redisContainer != nil {
		_ = s.redisContainer.Terminate(s.ctx)
	}
}

// Test complete configuration lifecycle
func (s *IntegrationTestSuite) TestCompleteConfigurationLifecycle() {
	// Step 1: Create configuration
	createReq := &CreateConfigRequest{
		Name:        "lifecycle-test",
		DisplayName: "Lifecycle Test Configuration",
		Description: "Integration test for complete lifecycle",
		Type:        ConfigTypePersonal,
		Settings: map[string]any{
			"theme":    "dark",
			"language": "en",
			"apiKey":   "secret-api-key-123",
			"features": []string{"feature1", "feature2"},
		},
		IsDefault: false,
	}

	config, err := s.service.CreateConfig(s.ctx, s.testUserID, createReq)
	s.NoError(err)
	s.NotNil(config)
	s.Equal(createReq.Name, config.Name)
	s.Equal(ConfigStatusDraft, config.Status)

	// Step 2: Get configuration (should hit cache after first retrieval)
	retrievedConfig, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	s.Equal(config.ID, retrievedConfig.ID)
	s.Equal("secret-api-key-123", retrievedConfig.Settings["apiKey"])

	// Step 3: Update configuration
	updateReq := &UpdateConfigRequest{
		DisplayName: stringPtr("Updated Lifecycle Test"),
		Description: stringPtr("Updated description"),
		Settings: map[string]any{
			"theme":     "light",
			"language":  "es",
			"apiKey":    "updated-secret-key",
			"features":  []string{"feature1", "feature2", "feature3"},
			"newOption": true,
		},
		Status: configStatusPtr(ConfigStatusActive),
	}

	isActive := true
	updateReq.IsActive = &isActive

	_, err = s.service.UpdateConfig(s.ctx, s.testUserID, config.ID, updateReq)
	s.NoError(err)

	// Get the updated config to verify changes
	updatedConfig, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	s.Equal("Updated Lifecycle Test", updatedConfig.DisplayName)
	s.Equal(ConfigStatusActive, updatedConfig.Status)
	s.Equal("updated-secret-key", updatedConfig.Settings["apiKey"])
	s.True(updatedConfig.IsActive)

	// Step 4: List configurations
	filter := ConfigFilter{
		Type:   ConfigTypePersonal,
		Status: ConfigStatusActive,
		Limit:  10,
	}

	configs, total, err := s.service.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.GreaterOrEqual(len(configs), 1)
	s.GreaterOrEqual(total, int64(1))

	// Find our config in the list
	found := false
	for _, c := range configs {
		if c.ID == config.ID {
			found = true
			s.Equal("Updated Lifecycle Test", c.DisplayName)
			break
		}
	}
	s.True(found, "Updated config should be in the list")

	// Step 5: Export configuration
	exportReq := &ConfigExportRequest{
		ConfigIDs: []uuid.UUID{config.ID},
	}

	exportData, err := s.service.ExportConfig(s.ctx, s.testUserID, exportReq)
	s.NoError(err)
	s.NotEmpty(exportData)

	// Step 6: Import configuration (create new one from export)
	importReq := &ConfigImportRequest{
		Name:        "imported-lifecycle-test",
		DisplayName: "Imported Lifecycle Test",
		Description: "Imported from export",
		Data:        exportData,
		MergeMode:   MergeModeReplace,
		EncryptKeys: []string{"apiKey"},
	}

	importedConfig, err := s.service.ImportConfig(s.ctx, s.testUserID, importReq)
	s.NoError(err)
	s.NotNil(importedConfig)
	s.Equal("imported-lifecycle-test", importedConfig.Name)

	// Step 7: Validate configurations
	validationErrors := s.service.ValidateConfig(s.ctx, updatedConfig)
	s.Empty(validationErrors, "Valid config should have no validation errors")

	// Step 8: Clean up - delete configurations
	err = s.service.DeleteConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)

	err = s.service.DeleteConfig(s.ctx, s.testUserID, importedConfig.ID)
	s.NoError(err)

	// Verify deletion
	_, err = s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.Error(err)
}

// Test server configuration management
func (s *IntegrationTestSuite) TestServerConfigurationManagement() {
	// Setup mock executor expectations
	mockExec := s.executor.(*executor.TestableExecutor)

	serverName := "test-integration-server"

	// Test enable server
	mockExec.On("Execute", s.ctx, mock.MatchedBy(func(req *executor.ExecutionRequest) bool {
		return req.Command == executor.CommandTypeServerEnable
	})).
		Return(&executor.ExecutionResult{
			Success:  true,
			Stdout:   fmt.Sprintf("Server %s enabled successfully", serverName),
			ExitCode: 0,
		}, nil)

	err := s.service.EnableServer(s.ctx, s.testUserID, serverName)
	s.NoError(err)

	// Test configure server
	configReq := &ServerConfigRequest{
		ServerName: serverName,
		Config: &UpdateServerConfigRequest{
			Settings: map[string]any{
				"endpoint": "http://localhost:8080",
				"timeout":  30,
				"retries":  3,
			},
		},
		Action: ServerActionConfigure,
	}

	err = s.service.ConfigureServer(s.ctx, s.testUserID, configReq)
	s.NoError(err)

	// Test get server status
	mockExec.On("Execute", s.ctx, mock.MatchedBy(func(req *executor.ExecutionRequest) bool {
		return req.Command == executor.CommandTypeServerInspect
	})).
		Return(&executor.ExecutionResult{
			Success: true,
			Stdout: `{
				"name": "test-integration-server",
				"status": "running",
				"config": {
					"endpoint": "http://localhost:8080"
				},
				"metadata": {
					"version": "1.0.0"
				}
			}`,
			ExitCode: 0,
		}, nil)

	status, err := s.service.GetServerStatus(s.ctx, s.testUserID, serverName)
	s.NoError(err)
	s.NotNil(status)

	// Test disable server
	mockExec.On("Execute", s.ctx, mock.MatchedBy(func(req *executor.ExecutionRequest) bool {
		return req.Command == executor.CommandTypeServerDisable
	})).
		Return(&executor.ExecutionResult{
			Success:  true,
			Stdout:   fmt.Sprintf("Server %s disabled successfully", serverName),
			ExitCode: 0,
		}, nil)

	err = s.service.DisableServer(s.ctx, s.testUserID, serverName)
	s.NoError(err)
}

// Test caching behavior
func (s *IntegrationTestSuite) TestCachingBehavior() {
	// Create a configuration
	createReq := &CreateConfigRequest{
		Name:        "cache-test",
		DisplayName: "Cache Test Configuration",
		Type:        ConfigTypePersonal,
		Settings: map[string]any{
			"testData": "original",
		},
	}

	config, err := s.service.CreateConfig(s.ctx, s.testUserID, createReq)
	s.NoError(err)

	// First retrieval - should populate cache
	start := time.Now()
	retrievedConfig1, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	firstRetrievalTime := time.Since(start)

	// Second retrieval - should hit cache (faster)
	start = time.Now()
	retrievedConfig2, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	secondRetrievalTime := time.Since(start)

	// Cache hit should be faster
	s.Less(secondRetrievalTime, firstRetrievalTime)
	s.Equal(retrievedConfig1.ID, retrievedConfig2.ID)
	s.Equal(retrievedConfig1.Settings["testData"], retrievedConfig2.Settings["testData"])

	// Update should invalidate cache
	updateReq := &UpdateConfigRequest{
		Settings: map[string]any{
			"testData": "updated",
		},
	}

	_, err = s.service.UpdateConfig(s.ctx, s.testUserID, config.ID, updateReq)
	s.NoError(err)

	// Next retrieval should get updated data
	retrievedAfterUpdate, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	s.Equal("updated", retrievedAfterUpdate.Settings["testData"])

	// Clean up
	err = s.service.DeleteConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
}

// Test concurrent operations
func (s *IntegrationTestSuite) TestConcurrentOperations() {
	numConfigs := 10
	configIDs := make([]uuid.UUID, numConfigs)

	// Create multiple configurations concurrently
	results := make(chan struct {
		config *UserConfig
		err    error
	}, numConfigs)

	for i := 0; i < numConfigs; i++ {
		go func(index int) {
			createReq := &CreateConfigRequest{
				Name:        fmt.Sprintf("concurrent-test-%d", index),
				DisplayName: fmt.Sprintf("Concurrent Test %d", index),
				Type:        ConfigTypePersonal,
				Settings: map[string]any{
					"index": index,
					"data":  fmt.Sprintf("concurrent-data-%d", index),
				},
			}

			config, err := s.service.CreateConfig(s.ctx, s.testUserID, createReq)
			results <- struct {
				config *UserConfig
				err    error
			}{config, err}
		}(i)
	}

	// Collect results
	for i := 0; i < numConfigs; i++ {
		result := <-results
		s.NoError(result.err)
		s.NotNil(result.config)
		configIDs[i] = result.config.ID
	}

	// Verify all configurations were created
	filter := ConfigFilter{
		NamePattern: "concurrent-test-",
		Limit:       20,
	}

	configs, total, err := s.service.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)
	s.GreaterOrEqual(len(configs), numConfigs)
	s.GreaterOrEqual(total, int64(numConfigs))

	// Clean up concurrently
	deleteResults := make(chan error, numConfigs)
	for _, configID := range configIDs {
		go func(id uuid.UUID) {
			deleteResults <- s.service.DeleteConfig(s.ctx, s.testUserID, id)
		}(configID)
	}

	// Verify all deletions
	for i := 0; i < numConfigs; i++ {
		err := <-deleteResults
		s.NoError(err)
	}
}

// Test error handling and recovery
func (s *IntegrationTestSuite) TestErrorHandlingAndRecovery() {
	// Test validation errors
	invalidCreateReq := &CreateConfigRequest{
		Name: "", // Invalid empty name
		Type: ConfigTypePersonal,
	}

	config, err := s.service.CreateConfig(s.ctx, s.testUserID, invalidCreateReq)
	s.Error(err)
	s.Nil(config)
	s.Contains(err.Error(), "validation failed")

	// Test not found errors
	nonExistentID := uuid.New()
	_, err = s.service.GetConfig(s.ctx, s.testUserID, nonExistentID)
	s.Error(err)

	// Test CLI command validation
	result, err := s.service.ExecuteConfigCommand(
		s.ctx,
		s.testUserID,
		"invalid-command",
		[]string{},
	)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "command not allowed")

	// Test command injection prevention
	result, err = s.service.ExecuteConfigCommand(
		s.ctx,
		s.testUserID,
		"docker; rm -rf /",
		[]string{"mcp"},
	)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "invalid command format")
}

// Test security features
func (s *IntegrationTestSuite) TestSecurityFeatures() {
	// Test encryption of sensitive data
	sensitiveConfig := &CreateConfigRequest{
		Name:        "security-test",
		DisplayName: "Security Test Configuration",
		Type:        ConfigTypePersonal,
		Settings: map[string]any{
			"apiKey":      "super-secret-api-key",
			"password":    "user-password-123",
			"publicField": "this-is-public",
		},
	}

	config, err := s.service.CreateConfig(s.ctx, s.testUserID, sensitiveConfig)
	s.NoError(err)

	// Retrieve and verify data is correctly decrypted
	retrievedConfig, err := s.service.GetConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
	s.Equal("super-secret-api-key", retrievedConfig.Settings["apiKey"])
	s.Equal("user-password-123", retrievedConfig.Settings["password"])
	s.Equal("this-is-public", retrievedConfig.Settings["publicField"])

	// Test multi-tenancy (different user can't access config)
	differentUserID := uuid.New().String()
	_, err = s.service.GetConfig(s.ctx, differentUserID, config.ID)
	s.Error(err)

	// Clean up
	err = s.service.DeleteConfig(s.ctx, s.testUserID, config.ID)
	s.NoError(err)
}

// Test performance under load
func (s *IntegrationTestSuite) TestPerformanceUnderLoad() {
	numOperations := 50
	startTime := time.Now()

	// Mixed operations: create, read, update
	for i := 0; i < numOperations; i++ {
		createReq := &CreateConfigRequest{
			Name:        fmt.Sprintf("perf-test-%d", i),
			DisplayName: fmt.Sprintf("Performance Test %d", i),
			Type:        ConfigTypePersonal,
			Settings: map[string]any{
				"index": i,
				"data":  fmt.Sprintf("performance-data-%d", i),
			},
		}

		config, err := s.service.CreateConfig(s.ctx, s.testUserID, createReq)
		s.NoError(err)

		// Read it back
		_, err = s.service.GetConfig(s.ctx, s.testUserID, config.ID)
		s.NoError(err)

		// Update it
		updateReq := &UpdateConfigRequest{
			Settings: map[string]any{
				"index":   i,
				"data":    fmt.Sprintf("updated-performance-data-%d", i),
				"updated": true,
			},
		}

		_, err = s.service.UpdateConfig(s.ctx, s.testUserID, config.ID, updateReq)
		s.NoError(err)
	}

	totalTime := time.Since(startTime)
	avgTimePerOperation := totalTime / time.Duration(numOperations*3) // 3 operations per iteration

	s.T().Logf("Completed %d operations in %v (avg: %v per operation)",
		numOperations*3, totalTime, avgTimePerOperation)

	// Performance assertions
	s.Less(avgTimePerOperation, 100*time.Millisecond, "Average operation time should be reasonable")
	s.Less(totalTime, 30*time.Second, "Total time should be reasonable")

	// Clean up
	filter := ConfigFilter{NamePattern: "perf-test-", Limit: 100}
	configs, _, err := s.service.ListConfigs(s.ctx, s.testUserID, filter)
	s.NoError(err)

	for _, config := range configs {
		_ = s.service.DeleteConfig(s.ctx, s.testUserID, config.ID)
	}
}

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	return &s
}

func configStatusPtr(status ConfigStatus) *ConfigStatus {
	return &status
}
