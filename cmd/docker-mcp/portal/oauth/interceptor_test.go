package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Testify-compatible mock implementations for testing

type TestMockTokenStorage struct {
	mock.Mock
}

func (m *TestMockTokenStorage) StoreToken(
	ctx context.Context,
	token *TokenData,
	tier StorageTier,
) error {
	args := m.Called(ctx, token, tier)
	return args.Error(0)
}

func (m *TestMockTokenStorage) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	args := m.Called(ctx, serverName, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenData), args.Error(1)
}

func (m *TestMockTokenStorage) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	args := m.Called(ctx, serverName, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenData), args.Error(1)
}

func (m *TestMockTokenStorage) DeleteToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	args := m.Called(ctx, serverName, userID)
	return args.Error(0)
}

func (m *TestMockTokenStorage) GetStorageTier(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (StorageTier, error) {
	args := m.Called(ctx, serverName, userID)
	return args.Get(0).(StorageTier), args.Error(1)
}

func (m *TestMockTokenStorage) ListTokens(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TokenData), args.Error(1)
}

func (m *TestMockTokenStorage) CleanupExpiredTokens(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *TestMockTokenStorage) MigrateTokens(
	ctx context.Context,
	fromTier, toTier StorageTier,
) (int, error) {
	args := m.Called(ctx, fromTier, toTier)
	return args.Int(0), args.Error(1)
}

func (m *TestMockTokenStorage) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type TestMockOAuthProvider struct {
	mock.Mock
}

func (m *TestMockOAuthProvider) GetProviderType() ProviderType {
	args := m.Called()
	return args.Get(0).(ProviderType)
}

func (m *TestMockOAuthProvider) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

func (m *TestMockOAuthProvider) GetAuthURL(config *ServerConfig, state string) (string, error) {
	args := m.Called(config, state)
	return args.String(0), args.Error(1)
}

func (m *TestMockOAuthProvider) ExchangeCode(
	ctx context.Context,
	config *ServerConfig,
	code string,
) (*TokenData, error) {
	args := m.Called(ctx, config, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenData), args.Error(1)
}

func (m *TestMockOAuthProvider) RefreshToken(
	ctx context.Context,
	config *ServerConfig,
	refreshToken string,
) (*TokenData, error) {
	args := m.Called(ctx, config, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenData), args.Error(1)
}

func (m *TestMockOAuthProvider) RevokeToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) error {
	args := m.Called(ctx, config, token)
	return args.Error(0)
}

func (m *TestMockOAuthProvider) ValidateToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (*TokenClaims, error) {
	args := m.Called(ctx, config, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TokenClaims), args.Error(1)
}

func (m *TestMockOAuthProvider) GetUserInfo(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (map[string]interface{}, error) {
	args := m.Called(ctx, config, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *TestMockOAuthProvider) SupportsRefresh() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *TestMockOAuthProvider) SupportsRevocation() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *TestMockOAuthProvider) GetDefaultScopes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *TestMockOAuthProvider) GetTokenExpiry(token string) (time.Time, error) {
	args := m.Called(token)
	return args.Get(0).(time.Time), args.Error(1)
}

type TestMockProviderRegistry struct {
	mock.Mock
}

func (m *TestMockProviderRegistry) RegisterProvider(provider OAuthProvider) error {
	args := m.Called(provider)
	return args.Error(0)
}

func (m *TestMockProviderRegistry) GetProvider(providerType ProviderType) (OAuthProvider, error) {
	args := m.Called(providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(OAuthProvider), args.Error(1)
}

func (m *TestMockProviderRegistry) ListProviders() []ProviderType {
	args := m.Called()
	return args.Get(0).([]ProviderType)
}

func (m *TestMockProviderRegistry) SupportsDCR(providerType ProviderType) bool {
	args := m.Called(providerType)
	return args.Bool(0)
}

func (m *TestMockProviderRegistry) RegisterDynamicClient(
	ctx context.Context,
	providerType ProviderType,
	req *DCRRequest,
) (*DCRResponse, error) {
	args := m.Called(ctx, providerType, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DCRResponse), args.Error(1)
}

type TestMockAuditLogger struct {
	mock.Mock
}

func (m *TestMockAuditLogger) LogOAuthEvent(ctx context.Context, event *AuditEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *TestMockAuditLogger) LogTokenRefresh(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	success bool,
) error {
	args := m.Called(ctx, serverName, userID, success)
	return args.Error(0)
}

func (m *TestMockAuditLogger) LogAuthorizationFlow(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	provider ProviderType,
	success bool,
) error {
	args := m.Called(ctx, serverName, userID, provider, success)
	return args.Error(0)
}

func (m *TestMockAuditLogger) LogTokenRevocation(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	success bool,
) error {
	args := m.Called(ctx, serverName, userID, success)
	return args.Error(0)
}

func (m *TestMockAuditLogger) GetUserActivity(
	ctx context.Context,
	userID uuid.UUID,
	since time.Time,
) ([]*AuditEvent, error) {
	args := m.Called(ctx, userID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AuditEvent), args.Error(1)
}

func (m *TestMockAuditLogger) GetServerActivity(
	ctx context.Context,
	serverName string,
	since time.Time,
) ([]*AuditEvent, error) {
	args := m.Called(ctx, serverName, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AuditEvent), args.Error(1)
}

func (m *TestMockAuditLogger) GetFailedAttempts(
	ctx context.Context,
	since time.Time,
) ([]*AuditEvent, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AuditEvent), args.Error(1)
}

type TestMockMetricsCollector struct {
	mock.Mock
}

func (m *TestMockMetricsCollector) RecordRequest(
	ctx context.Context,
	serverName string,
	provider ProviderType,
	duration time.Duration,
	success bool,
) {
	m.Called(ctx, serverName, provider, duration, success)
}

func (m *TestMockMetricsCollector) RecordTokenRefresh(
	ctx context.Context,
	serverName string,
	provider ProviderType,
	success bool,
) {
	m.Called(ctx, serverName, provider, success)
}

func (m *TestMockMetricsCollector) RecordError(
	ctx context.Context,
	errorType string,
	serverName string,
	provider ProviderType,
) {
	m.Called(ctx, errorType, serverName, provider)
}

func (m *TestMockMetricsCollector) GetMetrics(ctx context.Context) (*Metrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Metrics), args.Error(1)
}

func (m *TestMockMetricsCollector) Reset(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type TestMockConfigValidator struct {
	mock.Mock
}

func (m *TestMockConfigValidator) ValidateServerConfig(config *ServerConfig) []ValidationError {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

func (m *TestMockConfigValidator) ValidateProviderConfig(
	providerType ProviderType,
	config map[string]string,
) []ValidationError {
	args := m.Called(providerType, config)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

func (m *TestMockConfigValidator) ValidateToken(token *TokenData) []ValidationError {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

func (m *TestMockConfigValidator) ValidateTokenClaims(claims *TokenClaims) []ValidationError {
	args := m.Called(claims)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

func (m *TestMockConfigValidator) ValidateDCRRequest(req *DCRRequest) []ValidationError {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

// Test suite for OAuth Interceptor

func TestOAuthInterceptor_InterceptRequest_Success(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	// Create test data
	userID := uuid.New()
	serverName := "test-server"
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGitHub,
		AccessToken:  "test-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		LastUsed:     time.Now().Add(-5 * time.Minute),
	}

	config := &InterceptorConfig{
		Enabled:        true,
		DefaultTimeout: 30 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
			RetryOn401:      true,
		},
		StorageTiers: []StorageTier{StorageTierKeyVault},
	}

	serverConfig := &ServerConfig{
		ServerName:   serverName,
		ProviderType: ProviderTypeGitHub,
		ClientID:     "test-client-id",
		IsActive:     true,
	}

	// Setup mock expectations
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(token, nil)
	mockStorage.On("StoreToken", mock.Anything, mock.AnythingOfType("*oauth.TokenData"), mock.AnythingOfType("oauth.StorageTier")).
		Return(nil)
	mockRegistry.On("GetProvider", ProviderTypeGitHub).Return(mockProvider, nil)
	mockProvider.On("SupportsRefresh").Return(false)
	mockValidator.On("ValidateServerConfig", mock.AnythingOfType("*oauth.ServerConfig")).
		Return([]ValidationError(nil))
	mockMetrics.On(
		"RecordRequest",
		mock.Anything,
		serverName,
		ProviderTypeGitHub,
		mock.AnythingOfType("time.Duration"),
		true,
	)
	mockAuditLogger.On("LogOAuthEvent", mock.Anything, mock.AnythingOfType("*oauth.AuditEvent")).
		Return(nil)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Register server config
	err := interceptor.RegisterServer(context.Background(), serverConfig)
	require.NoError(t, err)

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create test request
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "GET",
		URL:        server.URL,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	// Execute
	response, err := interceptor.InterceptRequest(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Contains(t, string(response.Body), "success")

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
	mockAuditLogger.AssertExpectations(t)
}

func TestOAuthInterceptor_InterceptRequest_TokenRefresh(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	// Create test data
	userID := uuid.New()
	serverName := "test-server"

	// Expired token
	expiredToken := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGoogle,
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
		RefreshAt:    time.Now().Add(-1 * time.Hour), // Should refresh
		LastUsed:     time.Now().Add(-5 * time.Minute),
	}

	// Refreshed token
	refreshedToken := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGoogle,
		AccessToken:  "new-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		LastUsed:     time.Now(),
	}

	config := &InterceptorConfig{
		Enabled:          true,
		DefaultTimeout:   30 * time.Second,
		RefreshThreshold: 5 * time.Minute,
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
			RetryOn401:      true,
		},
		StorageTiers: []StorageTier{StorageTierKeyVault},
	}

	serverConfig := &ServerConfig{
		ServerName:   serverName,
		ProviderType: ProviderTypeGoogle,
		ClientID:     "test-client-id",
		IsActive:     true,
	}

	// Setup mock expectations
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(expiredToken, nil)
	mockRegistry.On("GetProvider", ProviderTypeGoogle).Return(mockProvider, nil)
	mockProvider.On("SupportsRefresh").Return(true)
	mockProvider.On("RefreshToken", mock.Anything, serverConfig, "refresh-token").
		Return(refreshedToken, nil)
	mockValidator.On("ValidateServerConfig", mock.AnythingOfType("*oauth.ServerConfig")).
		Return([]ValidationError(nil))
	mockStorage.On("StoreToken", mock.Anything, mock.AnythingOfType("*oauth.TokenData"), mock.AnythingOfType("oauth.StorageTier")).
		Return(nil)
	mockMetrics.On("RecordTokenRefresh", mock.Anything, serverName, ProviderTypeGoogle, true)
	mockMetrics.On(
		"RecordRequest",
		mock.Anything,
		serverName,
		ProviderTypeGoogle,
		mock.AnythingOfType("time.Duration"),
		true,
	)
	mockAuditLogger.On("LogOAuthEvent", mock.Anything, mock.AnythingOfType("*oauth.AuditEvent")).
		Return(nil)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Register server config
	err := interceptor.RegisterServer(context.Background(), serverConfig)
	require.NoError(t, err)

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer new-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create test request
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "GET",
		URL:        server.URL,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	// Execute
	response, err := interceptor.InterceptRequest(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.True(t, response.TokenRefreshed)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
	mockAuditLogger.AssertExpectations(t)
}

func TestOAuthInterceptor_InterceptRequest_401Retry(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	// Create test data
	userID := uuid.New()
	serverName := "test-server"
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGoogle,
		AccessToken:  "invalid-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		LastUsed:     time.Now().Add(-5 * time.Minute),
	}

	refreshedToken := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGoogle,
		AccessToken:  "valid-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		LastUsed:     time.Now(),
	}

	config := &InterceptorConfig{
		Enabled:        true,
		DefaultTimeout: 30 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond, // Fast for tests
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			RetryOn401:      true,
		},
		StorageTiers: []StorageTier{StorageTierKeyVault},
	}

	serverConfig := &ServerConfig{
		ServerName:   serverName,
		ProviderType: ProviderTypeGoogle,
		ClientID:     "test-client-id",
		IsActive:     true,
	}

	// Setup mock expectations
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(token, nil).Once()
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(refreshedToken, nil).Once()
	mockRegistry.On("GetProvider", ProviderTypeGoogle).Return(mockProvider, nil)
	mockProvider.On("SupportsRefresh").Return(true)
	mockProvider.On("RefreshToken", mock.Anything, serverConfig, "refresh-token").
		Return(refreshedToken, nil)
	mockValidator.On("ValidateServerConfig", mock.AnythingOfType("*oauth.ServerConfig")).
		Return([]ValidationError(nil))
	mockStorage.On("RefreshToken", mock.Anything, serverName, userID).Return(refreshedToken, nil)
	mockStorage.On("StoreToken", mock.Anything, mock.AnythingOfType("*oauth.TokenData"), mock.AnythingOfType("oauth.StorageTier")).
		Return(nil)
	mockMetrics.On("RecordTokenRefresh", mock.Anything, serverName, ProviderTypeGoogle, true)
	mockMetrics.On(
		"RecordRequest",
		mock.Anything,
		serverName,
		ProviderTypeGoogle,
		mock.AnythingOfType("time.Duration"),
		true,
	)
	mockAuditLogger.On("LogOAuthEvent", mock.Anything, mock.AnythingOfType("*oauth.AuditEvent")).
		Return(nil)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Register server config
	err := interceptor.RegisterServer(context.Background(), serverConfig)
	require.NoError(t, err)

	// Create mock HTTP server that returns 401 for invalid token, 200 for valid token
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		authHeader := r.Header.Get("Authorization")
		if authHeader == "Bearer invalid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid_token"}`))
		} else if authHeader == "Bearer valid-token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		} else {
			t.Errorf("Unexpected authorization header: %s", authHeader)
		}
	}))
	defer server.Close()

	// Create test request
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "GET",
		URL:        server.URL,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	// Execute
	response, err := interceptor.InterceptRequest(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.True(t, response.TokenRefreshed)
	assert.Equal(t, 2, callCount) // First call with invalid token, second with valid

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
	mockAuditLogger.AssertExpectations(t)
}

func TestOAuthInterceptor_RegisterServer(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	config := &InterceptorConfig{
		Enabled: true,
	}

	serverConfig := &ServerConfig{
		ServerName:   "test-server",
		ProviderType: ProviderTypeGitHub,
		ClientID:     "test-client-id",
		Scopes:       []string{"repo", "user"},
		RedirectURI:  "http://localhost:8080/callback",
		IsActive:     true,
	}

	// Setup mock expectations
	mockValidator.On("ValidateServerConfig", serverConfig).Return([]ValidationError{})
	mockRegistry.On("GetProvider", ProviderTypeGitHub).Return(mockProvider, nil)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Execute
	err := interceptor.RegisterServer(context.Background(), serverConfig)

	// Verify
	assert.NoError(t, err)

	// Verify server can be retrieved
	retrievedConfig, err := interceptor.GetServerConfig(context.Background(), "test-server")
	assert.NoError(t, err)
	assert.Equal(t, serverConfig.ServerName, retrievedConfig.ServerName)
	assert.Equal(t, serverConfig.ProviderType, retrievedConfig.ProviderType)
	assert.Equal(t, serverConfig.ClientID, retrievedConfig.ClientID)

	// Verify mock expectations
	mockValidator.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
}

func TestOAuthInterceptor_TokenManagement(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	config := &InterceptorConfig{
		Enabled:      true,
		StorageTiers: []StorageTier{StorageTierKeyVault},
	}

	userID := uuid.New()
	serverName := "test-server"
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGitHub,
		AccessToken:  "test-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	serverConfig := &ServerConfig{
		ServerName:   serverName,
		ProviderType: ProviderTypeGitHub,
		ClientID:     "test-client-id",
		IsActive:     true,
	}

	// Setup mock expectations for token operations
	mockValidator.On("ValidateToken", token).Return([]ValidationError{})
	mockStorage.On("StoreToken", mock.Anything, token, StorageTierKeyVault).Return(nil)
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(token, nil)
	mockStorage.On("RefreshToken", mock.Anything, serverName, userID).Return(token, nil)
	mockStorage.On("DeleteToken", mock.Anything, serverName, userID).Return(nil)
	mockRegistry.On("GetProvider", ProviderTypeGitHub).Return(mockProvider, nil)
	mockProvider.On("SupportsRevocation").Return(true)
	mockProvider.On("RevokeToken", mock.Anything, serverConfig, "test-token").Return(nil)
	mockAuditLogger.On("LogTokenRevocation", mock.Anything, serverName, userID, true).Return(nil)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Register server config
	err := interceptor.RegisterServer(context.Background(), serverConfig)
	require.NoError(t, err)

	// Test StoreToken
	err = interceptor.StoreToken(context.Background(), token)
	assert.NoError(t, err)

	// Test GetToken
	retrievedToken, err := interceptor.GetToken(context.Background(), serverName, userID)
	assert.NoError(t, err)
	assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)

	// Test RefreshToken
	refreshedToken, err := interceptor.RefreshToken(context.Background(), serverName, userID)
	assert.NoError(t, err)
	assert.Equal(t, token.AccessToken, refreshedToken.AccessToken)

	// Test RevokeToken
	err = interceptor.RevokeToken(context.Background(), serverName, userID)
	assert.NoError(t, err)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
	mockAuditLogger.AssertExpectations(t)
}

func TestOAuthInterceptor_Health(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	config := &InterceptorConfig{
		Enabled: true,
	}

	// Setup mock expectations
	mockStorage.On("Health", mock.Anything).Return(nil)
	mockRegistry.On("ListProviders").Return([]ProviderType{ProviderTypeGitHub, ProviderTypeGoogle})

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Execute
	err := interceptor.Health(context.Background())

	// Verify
	assert.NoError(t, err)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
}

func TestOAuthInterceptor_GetMetrics(t *testing.T) {
	// Setup mocks
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockAuditLogger := &TestMockAuditLogger{}
	mockMetrics := &TestMockMetricsCollector{}
	mockValidator := &TestMockConfigValidator{}

	config := &InterceptorConfig{
		Enabled:          true,
		DefaultTimeout:   30 * time.Second,
		RefreshThreshold: 5 * time.Minute,
		StorageTiers:     []StorageTier{StorageTierKeyVault, StorageTierEnvironment},
	}

	expectedMetrics := &Metrics{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		AverageLatency:     100 * time.Millisecond,
		ActiveTokens:       10,
	}

	// Setup mock expectations
	mockMetrics.On("GetMetrics", mock.Anything).Return(expectedMetrics, nil)
	mockRegistry.On("ListProviders").Return([]ProviderType{ProviderTypeGitHub, ProviderTypeGoogle})

	// Create interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		mockStorage,
		mockRegistry,
		mockAuditLogger,
		mockMetrics,
		mockValidator,
	)

	// Execute
	metrics, err := interceptor.GetMetrics(context.Background())

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	// Check that metrics contains our expected data
	oauthMetrics, exists := metrics["oauth_metrics"]
	assert.True(t, exists)
	assert.Equal(t, expectedMetrics, oauthMetrics)

	// Check interceptor-specific metrics
	assert.Equal(t, 0, metrics["server_configs"]) // No servers registered
	assert.Equal(t, 2, metrics["supported_providers"])

	config_section, exists := metrics["config"].(map[string]interface{})
	assert.True(t, exists)
	assert.True(t, config_section["enabled"].(bool))
	assert.Equal(t, "30s", config_section["default_timeout"].(string))

	// Verify mock expectations
	mockMetrics.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
}

func TestOAuthInterceptor_BackoffCalculation(t *testing.T) {
	config := &InterceptorConfig{
		RetryPolicy: RetryPolicy{
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
			Jitter:          false, // Disable jitter for predictable tests
		},
	}

	interceptor := CreateOAuthInterceptor(config, nil, nil, nil, nil, nil)

	// Test exponential backoff
	delay0 := interceptor.calculateBackoffDelay(0)
	delay1 := interceptor.calculateBackoffDelay(1)
	delay2 := interceptor.calculateBackoffDelay(2)
	delay3 := interceptor.calculateBackoffDelay(3)

	assert.Equal(t, 100*time.Millisecond, delay0)
	assert.Equal(t, 200*time.Millisecond, delay1)
	assert.Equal(t, 400*time.Millisecond, delay2)
	assert.Equal(t, 800*time.Millisecond, delay3)

	// Test max interval cap
	delay10 := interceptor.calculateBackoffDelay(10)
	assert.Equal(t, 5*time.Second, delay10)
}

func TestOAuthInterceptor_ShouldRetryOnStatusCode(t *testing.T) {
	config := &InterceptorConfig{
		RetryPolicy: RetryPolicy{
			RetryOn401: true,
			RetryOn403: false,
			RetryOn429: true,
			RetryOn5xx: true,
		},
	}

	interceptor := CreateOAuthInterceptor(config, nil, nil, nil, nil, nil)

	// Test retry conditions
	assert.True(t, interceptor.shouldRetryOnStatusCode(401))  // Unauthorized
	assert.False(t, interceptor.shouldRetryOnStatusCode(403)) // Forbidden (disabled)
	assert.True(t, interceptor.shouldRetryOnStatusCode(429))  // Too Many Requests
	assert.True(t, interceptor.shouldRetryOnStatusCode(500))  // Internal Server Error
	assert.True(t, interceptor.shouldRetryOnStatusCode(502))  // Bad Gateway
	assert.True(t, interceptor.shouldRetryOnStatusCode(503))  // Service Unavailable

	// Test no retry conditions
	assert.False(t, interceptor.shouldRetryOnStatusCode(200)) // OK
	assert.False(t, interceptor.shouldRetryOnStatusCode(400)) // Bad Request
	assert.False(t, interceptor.shouldRetryOnStatusCode(404)) // Not Found
}

func TestOAuthInterceptor_ShouldRefreshToken(t *testing.T) {
	config := &InterceptorConfig{
		RefreshThreshold: 5 * time.Minute,
	}

	interceptor := CreateOAuthInterceptor(config, nil, nil, nil, nil, nil)

	now := time.Now()

	// Token that's already expired
	expiredToken := &TokenData{
		ExpiresAt:    now.Add(-1 * time.Hour),
		RefreshAt:    now.Add(-1 * time.Hour),
		RefreshToken: "refresh-token",
	}
	assert.True(t, interceptor.shouldRefreshToken(expiredToken))

	// Token that should be refreshed proactively (past RefreshAt)
	shouldRefreshToken := &TokenData{
		ExpiresAt:    now.Add(1 * time.Hour),
		RefreshAt:    now.Add(-1 * time.Minute),
		RefreshToken: "refresh-token",
	}
	assert.True(t, interceptor.shouldRefreshToken(shouldRefreshToken))

	// Token that's close to expiry (within refresh threshold)
	closeToExpiryToken := &TokenData{
		ExpiresAt:    now.Add(2 * time.Minute), // Less than 5 minute threshold
		RefreshAt:    now.Add(2 * time.Minute),
		RefreshToken: "refresh-token",
	}
	assert.True(t, interceptor.shouldRefreshToken(closeToExpiryToken))

	// Token that's still valid and not close to expiry
	validToken := &TokenData{
		ExpiresAt:    now.Add(1 * time.Hour),
		RefreshAt:    now.Add(45 * time.Minute),
		RefreshToken: "refresh-token",
	}
	assert.False(t, interceptor.shouldRefreshToken(validToken))

	// Token without refresh token
	noRefreshToken := &TokenData{
		ExpiresAt:    now.Add(-1 * time.Hour), // Even if expired
		RefreshAt:    now.Add(-1 * time.Hour),
		RefreshToken: "", // No refresh token
	}
	assert.False(t, interceptor.shouldRefreshToken(noRefreshToken))
}

// Benchmark tests

func BenchmarkOAuthInterceptor_InterceptRequest(b *testing.B) {
	// Setup minimal mocks for benchmarking
	mockStorage := &TestMockTokenStorage{}
	mockRegistry := &TestMockProviderRegistry{}
	mockProvider := &TestMockOAuthProvider{}

	userID := uuid.New()
	serverName := "bench-server"
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeGitHub,
		AccessToken:  "bench-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
	}

	config := &InterceptorConfig{
		Enabled:        true,
		DefaultTimeout: 30 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxRetries: 0, // No retries for benchmarking
		},
	}

	serverConfig := &ServerConfig{
		ServerName:   serverName,
		ProviderType: ProviderTypeGitHub,
		IsActive:     true,
	}

	// Setup mock expectations
	mockStorage.On("GetToken", mock.Anything, serverName, userID).Return(token, nil)
	mockStorage.On("StoreToken", mock.Anything, mock.AnythingOfType("*oauth.TokenData"), mock.AnythingOfType("oauth.StorageTier")).
		Return(nil)
	mockRegistry.On("GetProvider", ProviderTypeGitHub).Return(mockProvider, nil)
	mockProvider.On("SupportsRefresh").Return(false)

	// Create interceptor
	interceptor := CreateOAuthInterceptor(config, mockStorage, mockRegistry, nil, nil, nil)
	_ = interceptor.RegisterServer(context.Background(), serverConfig)

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create test request
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "GET",
		URL:        server.URL,
		Timestamp:  time.Now(),
		MaxRetries: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.RequestID = fmt.Sprintf("bench-req-%d", i)
		_, err := interceptor.InterceptRequest(context.Background(), req)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
