package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of the Logger interface
type MockLogger struct {
	mock.Mock
}

// LogCommand logs the start of a command execution
func (m *MockLogger) LogCommand(
	ctx context.Context,
	userID uuid.UUID,
	command string,
	args []string,
) uuid.UUID {
	ret := m.Called(ctx, userID, command, args)
	return ret.Get(0).(uuid.UUID)
}

// LogCommandResult logs the result of a command execution
func (m *MockLogger) LogCommandResult(
	ctx context.Context,
	auditID uuid.UUID,
	result string,
	err error,
	duration time.Duration,
) {
	m.Called(ctx, auditID, result, err, duration)
}

// LogSecurityEvent logs a security-related event
func (m *MockLogger) LogSecurityEvent(
	ctx context.Context,
	userID uuid.UUID,
	event EventType,
	details map[string]interface{},
) {
	m.Called(ctx, userID, event, details)
}

// LogAccessDenied logs an access denied event
func (m *MockLogger) LogAccessDenied(
	ctx context.Context,
	userID uuid.UUID,
	resource string,
	reason string,
) {
	m.Called(ctx, userID, resource, reason)
}

// LogRateLimitExceeded logs a rate limit exceeded event
func (m *MockLogger) LogRateLimitExceeded(ctx context.Context, userID uuid.UUID, command string) {
	m.Called(ctx, userID, command)
}

// GetLogs retrieves audit logs for a user
func (m *MockLogger) GetLogs(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]AuditEntry, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]AuditEntry), args.Error(1)
}

// Log provides a general-purpose audit logging method
func (m *MockLogger) Log(
	ctx context.Context,
	action Action,
	resource string,
	resourceID string,
	userID string,
	metadata map[string]interface{},
) error {
	args := m.Called(ctx, action, resource, resourceID, userID, metadata)
	return args.Error(0)
}

// MockStorage is a mock implementation of the Storage interface
type MockStorage struct {
	mock.Mock
}

// Store saves an entry
func (m *MockStorage) Store(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

// Retrieve gets entries
func (m *MockStorage) Retrieve(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]AuditEntry, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]AuditEntry), args.Error(1)
}

// RetrieveByID gets a specific entry
func (m *MockStorage) RetrieveByID(ctx context.Context, id uuid.UUID) (*AuditEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AuditEntry), args.Error(1)
}

// CreateAuditLogger creates a mock audit logger for testing
func CreateAuditLogger(config Config) (Logger, error) {
	mockLogger := &MockLogger{}

	// Set up default behaviors
	mockLogger.On("LogCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(uuid.New())
	mockLogger.On("LogCommandResult", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return()
	mockLogger.On("LogSecurityEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return()
	mockLogger.On("LogAccessDenied", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return()
	mockLogger.On("LogRateLimitExceeded", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Log", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	mockLogger.On("GetLogs", mock.Anything, mock.Anything, mock.Anything).
		Return([]AuditEntry{}, nil)

	return mockLogger, nil
}

// Config represents audit logger configuration for testing
type Config struct {
	Enabled    bool   `json:"enabled"`
	LogLevel   string `json:"log_level"`
	OutputPath string `json:"output_path"`
}

// NewMockLogger creates a new mock logger instance
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// NewMockStorage creates a new mock storage instance
func NewMockStorage() *MockStorage {
	return &MockStorage{}
}
