package userconfig

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockUserConfigRepository is a mock implementation of UserConfigRepository
type MockUserConfigRepository struct {
	mock.Mock
}

func (m *MockUserConfigRepository) CreateConfig(
	ctx context.Context,
	userID string,
	config *UserConfig,
) error {
	args := m.Called(ctx, userID, config)
	return args.Error(0)
}

func (m *MockUserConfigRepository) GetConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*UserConfig, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserConfig), args.Error(1)
}

func (m *MockUserConfigRepository) UpdateConfig(
	ctx context.Context,
	userID string,
	config *UserConfig,
) error {
	args := m.Called(ctx, userID, config)
	return args.Error(0)
}

func (m *MockUserConfigRepository) DeleteConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockUserConfigRepository) ListConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) ([]*UserConfig, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserConfig), args.Error(1)
}

func (m *MockUserConfigRepository) CountConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) (int64, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserConfigRepository) GetServerConfig(
	ctx context.Context,
	userID string,
	serverName string,
) (*ServerConfig, error) {
	args := m.Called(ctx, userID, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ServerConfig), args.Error(1)
}

func (m *MockUserConfigRepository) SaveServerConfig(
	ctx context.Context,
	userID string,
	config *ServerConfig,
) error {
	args := m.Called(ctx, userID, config)
	return args.Error(0)
}

func (m *MockUserConfigRepository) DeleteServerConfig(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	args := m.Called(ctx, userID, serverName)
	return args.Error(0)
}

func (m *MockUserConfigRepository) ListServerConfigs(
	ctx context.Context,
	userID string,
) ([]*ServerConfig, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ServerConfig), args.Error(1)
}

// MockUserConfigService is a mock implementation of UserConfigService
type MockUserConfigService struct {
	mock.Mock
}

func (m *MockUserConfigService) CreateConfig(
	ctx context.Context,
	userID string,
	req *CreateConfigRequest,
) (*UserConfig, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserConfig), args.Error(1)
}

func (m *MockUserConfigService) GetConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*UserConfig, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserConfig), args.Error(1)
}

func (m *MockUserConfigService) UpdateConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	req *UpdateConfigRequest,
) (*UserConfig, error) {
	args := m.Called(ctx, userID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserConfig), args.Error(1)
}

func (m *MockUserConfigService) DeleteConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockUserConfigService) ListConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) ([]*UserConfig, int64, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*UserConfig), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserConfigService) EnableServer(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	args := m.Called(ctx, userID, serverName)
	return args.Error(0)
}

func (m *MockUserConfigService) DisableServer(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	args := m.Called(ctx, userID, serverName)
	return args.Error(0)
}

func (m *MockUserConfigService) ConfigureServer(
	ctx context.Context,
	userID string,
	req *ServerConfigRequest,
) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockUserConfigService) GetServerStatus(
	ctx context.Context,
	userID string,
	serverName string,
) (*ServerStatus, error) {
	args := m.Called(ctx, userID, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ServerStatus), args.Error(1)
}

func (m *MockUserConfigService) ImportConfig(
	ctx context.Context,
	userID string,
	req *ConfigImportRequest,
) (*UserConfig, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserConfig), args.Error(1)
}

func (m *MockUserConfigService) ExportConfig(
	ctx context.Context,
	userID string,
	req *ConfigExportRequest,
) ([]byte, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockUserConfigService) ExecuteConfigCommand(
	ctx context.Context,
	userID string,
	command string,
	args []string,
) (*CLIResult, error) {
	callArgs := m.Called(ctx, userID, command, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(*CLIResult), callArgs.Error(1)
}

func (m *MockUserConfigService) ValidateConfig(
	ctx context.Context,
	config *UserConfig,
) []ValidationError {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}
