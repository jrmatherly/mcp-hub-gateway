package userconfig

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

type UserConfigServiceTestSuite struct {
	suite.Suite

	// Service under test
	service UserConfigService

	// Mocks
	mockRepo     *MockUserConfigRepository
	mockExecutor *executor.TestableExecutor
	mockAudit    *audit.MockLogger
	mockCache    *cache.MockCache
	mockCrypto   *crypto.MockEncryption

	// Test data
	testUserID   string
	testConfigID uuid.UUID
	testConfig   *UserConfig
	ctx          context.Context
}

func TestUserConfigServiceSuite(t *testing.T) {
	suite.Run(t, new(UserConfigServiceTestSuite))
}

func (s *UserConfigServiceTestSuite) SetupTest() {
	s.mockRepo = &MockUserConfigRepository{}
	s.mockExecutor = executor.NewTestableExecutor()
	s.mockAudit = &audit.MockLogger{}
	s.mockCache = &cache.MockCache{}
	s.mockCrypto = &crypto.MockEncryption{}

	var err error
	s.service, err = CreateUserConfigService(
		s.mockRepo,
		s.mockExecutor,
		s.mockAudit,
		s.mockCache,
		s.mockCrypto,
	)
	s.Require().NoError(err)

	s.testUserID = "test-user-123"
	s.testConfigID = uuid.New()
	s.ctx = context.Background()

	s.testConfig = &UserConfig{
		ID:          s.testConfigID,
		Name:        "test-config",
		DisplayName: "Test Configuration",
		Description: "Test configuration for unit tests",
		Type:        ConfigTypePersonal,
		Status:      ConfigStatusActive,
		OwnerID:     uuid.MustParse(s.testUserID),
		TenantID:    "test-tenant",
		IsDefault:   false,
		IsActive:    true,
		Version:     "1.0.0",
		Settings: map[string]any{
			"theme":     "dark",
			"language":  "en",
			"debugMode": true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (s *UserConfigServiceTestSuite) TearDownTest() {
	s.mockRepo.AssertExpectations(s.T())
	s.mockExecutor.AssertExpectations(s.T())
	s.mockAudit.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
	s.mockCrypto.AssertExpectations(s.T())
}

// Test CreateConfig - Success Case
func (s *UserConfigServiceTestSuite) TestCreateConfig_Success() {
	// Arrange
	req := &CreateConfigRequest{
		Name:        "new-config",
		DisplayName: "New Configuration",
		Description: "New configuration for testing",
		Type:        ConfigTypePersonal,
		Settings: map[string]any{
			"theme": "light",
		},
	}

	// expectedConfig used for reference but not compared directly

	// Mock expectations
	s.mockRepo.On("CreateConfig", s.ctx, s.testUserID, mock.AnythingOfType("*userconfig.UserConfig")).
		Return(nil).
		Run(func(args mock.Arguments) {
			config := args.Get(2).(*UserConfig)
			require.NotEqual(s.T(), uuid.Nil, config.ID)
			require.Equal(s.T(), req.Name, config.Name)
			require.Equal(s.T(), ConfigStatusDraft, config.Status)
		})

	s.mockAudit.On("Log", s.ctx, audit.ActionCreate, "user_config", mock.AnythingOfType("string"), s.testUserID, mock.Anything).
		Return(nil)

	s.mockCache.On("Delete", mock.MatchedBy(func(key string) bool {
		return key == "userconfig:list:"+s.testUserID+":*"
	})).Return(nil)

	// Act
	result, err := s.service.CreateConfig(s.ctx, s.testUserID, req)

	// Assert
	s.NoError(err)
	s.NotNil(result)
	s.Equal(req.Name, result.Name)
	s.Equal(ConfigStatusDraft, result.Status)
	s.NotEqual(uuid.Nil, result.ID)
}

// Test CreateConfig - Validation Error
func (s *UserConfigServiceTestSuite) TestCreateConfig_ValidationError() {
	// Arrange
	req := &CreateConfigRequest{
		Name: "", // Invalid empty name
		Type: ConfigTypePersonal,
	}

	// Act
	result, err := s.service.CreateConfig(s.ctx, s.testUserID, req)

	// Assert
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "name is required")
}

// Test GetConfig - Success with Cache Hit
func (s *UserConfigServiceTestSuite) TestGetConfig_CacheHit() {
	// Arrange
	cacheKey := "userconfig:" + s.testUserID + ":" + s.testConfigID.String()
	configJSON, _ := json.Marshal(s.testConfig)

	s.mockCache.On("Get", cacheKey).Return(configJSON, nil)

	// Act
	result, err := s.service.GetConfig(s.ctx, s.testUserID, s.testConfigID)

	// Assert
	s.NoError(err)
	s.NotNil(result)
	s.Equal(s.testConfig.ID, result.ID)
	s.Equal(s.testConfig.Name, result.Name)
}

// Test GetConfig - Cache Miss, Database Hit
func (s *UserConfigServiceTestSuite) TestGetConfig_CacheMiss() {
	// Arrange
	cacheKey := "userconfig:" + s.testUserID + ":" + s.testConfigID.String()

	s.mockCache.On("Get", cacheKey).Return(nil, cache.ErrNotFound)
	s.mockRepo.On("GetConfig", s.ctx, s.testUserID, s.testConfigID).Return(s.testConfig, nil)

	configJSON, _ := json.Marshal(s.testConfig)
	s.mockCache.On("Set", cacheKey, configJSON, 15*time.Minute).Return(nil)

	// Act
	result, err := s.service.GetConfig(s.ctx, s.testUserID, s.testConfigID)

	// Assert
	s.NoError(err)
	s.NotNil(result)
	s.Equal(s.testConfig.ID, result.ID)
}

// Test UpdateConfig - Success
func (s *UserConfigServiceTestSuite) TestUpdateConfig_Success() {
	// Arrange
	req := &UpdateConfigRequest{
		DisplayName: stringPtr("Updated Display Name"),
		Description: stringPtr("Updated description"),
		Settings: map[string]any{
			"theme":    "dark",
			"language": "es",
		},
	}

	s.mockRepo.On("GetConfig", s.ctx, s.testUserID, s.testConfigID).Return(s.testConfig, nil)
	s.mockRepo.On("UpdateConfig", s.ctx, s.testUserID, mock.AnythingOfType("*userconfig.UserConfig")).
		Return(nil).
		Run(func(args mock.Arguments) {
			config := args.Get(2).(*UserConfig)
			require.Equal(s.T(), *req.DisplayName, config.DisplayName)
			require.Equal(s.T(), *req.Description, config.Description)
		})

	s.mockAudit.On("Log", s.ctx, audit.ActionUpdate, "user_config", s.testConfigID.String(), s.testUserID, mock.Anything).
		Return(nil)

	cacheKey := "userconfig:" + s.testUserID + ":" + s.testConfigID.String()
	s.mockCache.On("Delete", cacheKey).Return(nil)
	s.mockCache.On("Delete", mock.MatchedBy(func(key string) bool {
		return key == "userconfig:list:"+s.testUserID+":*"
	})).Return(nil)

	// Act
	result, err := s.service.UpdateConfig(s.ctx, s.testUserID, s.testConfigID, req)

	// Assert
	s.NoError(err)
	s.NotNil(result)
	s.Equal(*req.DisplayName, result.DisplayName)
}

// Test EnableServer - CLI Integration
func (s *UserConfigServiceTestSuite) TestEnableServer_Success() {
	// Arrange
	serverName := "test-server"

	s.mockExecutor.On("Execute", s.ctx, &executor.ExecutionRequest{
		Command: executor.CommandTypeServerEnable,
		Args:    []string{serverName},
		UserID:  s.testUserID,
	}).Return(&executor.ExecutionResult{
		Success:  true,
		Stdout:   "Server test-server enabled successfully",
		ExitCode: 0,
	}, nil)

	s.mockAudit.On("Log", s.ctx, audit.ActionExecute, "server_enable", serverName, s.testUserID, mock.Anything).
		Return(nil)

	// Act
	err := s.service.EnableServer(s.ctx, s.testUserID, serverName)

	// Assert
	s.NoError(err)
}

// Test ValidateConfig - Success
func (s *UserConfigServiceTestSuite) TestValidateConfig_Success() {
	// Act
	errors := s.service.ValidateConfig(s.ctx, s.testConfig)

	// Assert
	s.Empty(errors)
}

// Test ValidateConfig - Multiple Errors
func (s *UserConfigServiceTestSuite) TestValidateConfig_MultipleErrors() {
	// Arrange
	invalidConfig := &UserConfig{
		Name:     "",             // Invalid empty name
		Type:     "invalid-type", // Invalid type
		Settings: nil,            // Invalid nil settings
	}

	// Act
	errors := s.service.ValidateConfig(s.ctx, invalidConfig)

	// Assert
	s.NotEmpty(errors)
	s.Len(errors, 3) // name, type, settings
}

// Test ImportConfig with Encryption
func (s *UserConfigServiceTestSuite) TestImportConfig_WithEncryption() {
	// Arrange
	importData := map[string]any{
		"servers": []map[string]any{
			{
				"name": "server1",
				"config": map[string]any{
					"apiKey": "secret-key-123",
				},
			},
		},
	}

	importJSON, _ := json.Marshal(importData)
	req := &ConfigImportRequest{
		Name:        "imported-config",
		Data:        importJSON,
		MergeMode:   MergeModeReplace,
		EncryptKeys: []string{"apiKey", "password", "token"},
	}

	s.mockCrypto.On("Encrypt", mock.AnythingOfType("[]uint8")).
		Return([]byte("encrypted-data"), nil)

	s.mockRepo.On("CreateConfig", s.ctx, s.testUserID, mock.AnythingOfType("*userconfig.UserConfig")).
		Return(nil)

	s.mockAudit.On("Log", s.ctx, audit.ActionImport, "user_config", mock.AnythingOfType("string"), s.testUserID, mock.Anything).
		Return(nil)

	// Act
	result, err := s.service.ImportConfig(s.ctx, s.testUserID, req)

	// Assert
	s.NoError(err)
	s.NotNil(result)
	s.Equal(req.Name, result.Name)
}

// Test ExecuteConfigCommand - Security Validation
func (s *UserConfigServiceTestSuite) TestExecuteConfigCommand_SecurityValidation() {
	// Test cases for command injection prevention
	testCases := []struct {
		name        string
		command     string
		args        []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid command",
			command:     "docker",
			args:        []string{"mcp", "server", "list"},
			shouldError: false,
		},
		{
			name:        "invalid command injection",
			command:     "docker; rm -rf /",
			args:        []string{"mcp"},
			shouldError: true,
			errorMsg:    "invalid command",
		},
		{
			name:        "invalid argument injection",
			command:     "docker",
			args:        []string{"mcp", "server", "list; rm -rf /"},
			shouldError: true,
			errorMsg:    "invalid argument",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if !tc.shouldError {
				s.mockExecutor.On("Execute", s.ctx, mock.MatchedBy(func(req *executor.ExecutionRequest) bool {
					return req.UserID == s.testUserID
				})).
					Return(&executor.ExecutionResult{Success: true}, nil)
				s.mockAudit.On("Log", s.ctx, audit.ActionExecute, "config_command", tc.command, s.testUserID, mock.Anything).
					Return(nil)
			}

			result, err := s.service.ExecuteConfigCommand(s.ctx, s.testUserID, tc.command, tc.args)

			if tc.shouldError {
				s.Error(err)
				s.Contains(err.Error(), tc.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
			}
		})
	}
}

// Benchmark tests for performance optimization
func (s *UserConfigServiceTestSuite) TestListConfigs_Performance() {
	// Arrange
	filter := ConfigFilter{
		Type:   ConfigTypePersonal,
		Status: ConfigStatusActive,
		Limit:  10,
		Offset: 0,
	}

	configs := make([]*UserConfig, 10)
	for i := range configs {
		configs[i] = &UserConfig{
			ID:   uuid.New(),
			Name: "config-" + string(rune(i)),
			Type: ConfigTypePersonal,
		}
	}

	s.mockRepo.On("ListConfigs", s.ctx, s.testUserID, filter).Return(configs, nil)
	s.mockRepo.On("CountConfigs", s.ctx, s.testUserID, filter).Return(int64(100), nil)

	// Act
	start := time.Now()
	result, total, err := s.service.ListConfigs(s.ctx, s.testUserID, filter)
	duration := time.Since(start)

	// Assert
	s.NoError(err)
	s.Len(result, 10)
	s.Equal(int64(100), total)
	s.Less(duration, 100*time.Millisecond) // Performance requirement
}
