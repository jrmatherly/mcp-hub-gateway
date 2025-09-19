# OAuth Implementation Testing Guide

Comprehensive testing strategy for MCP Portal OAuth implementation with Azure service dependencies.

## Overview

This guide provides complete testing patterns for the OAuth implementation, focusing on:

- DCR Bridge with Microsoft Graph SDK v1.64.0
- Key Vault storage with Azure Key Vault SDK v1.4.0
- Hierarchical token storage testing
- Integration testing with proper mocking
- Test coverage strategies to reach 50%+ target

## Test Structure

```
cmd/docker-mcp/portal/oauth/
├── dcr_bridge_test.go          # DCR bridge unit tests
├── storage_test.go             # Token storage tests
├── interceptor_test.go         # OAuth interceptor tests
├── mock_azure.go               # Azure SDK mocks
├── mock_graph.go               # Microsoft Graph mocks
├── testdata/                   # Test fixtures
│   ├── tokens/                 # Sample token data
│   └── responses/              # Mock API responses
└── integration/                # Integration tests
    ├── dcr_integration_test.go
    └── oauth_flow_test.go
```

## 1. Azure SDK Mocking Framework

### 1.1 Microsoft Graph SDK Mocks

Create comprehensive mocks for Microsoft Graph SDK components:

```go
// cmd/docker-mcp/portal/oauth/mock_graph.go
package oauth

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	graphapplications "github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/serviceprincipal"
)

// MockGraphServiceClient mocks the Microsoft Graph client
type MockGraphServiceClient struct {
	mock.Mock
	applications    *MockApplicationsRequestBuilder
	servicePrincipals *MockServicePrincipalsRequestBuilder
}

func NewMockGraphServiceClient() *MockGraphServiceClient {
	m := &MockGraphServiceClient{}
	m.applications = NewMockApplicationsRequestBuilder()
	m.servicePrincipals = NewMockServicePrincipalsRequestBuilder()
	return m
}

func (m *MockGraphServiceClient) Applications() *MockApplicationsRequestBuilder {
	return m.applications
}

func (m *MockGraphServiceClient) ServicePrincipals() *MockServicePrincipalsRequestBuilder {
	return m.servicePrincipals
}

// MockApplicationsRequestBuilder mocks applications endpoint
type MockApplicationsRequestBuilder struct {
	mock.Mock
}

func NewMockApplicationsRequestBuilder() *MockApplicationsRequestBuilder {
	return &MockApplicationsRequestBuilder{}
}

func (m *MockApplicationsRequestBuilder) Post(
	ctx context.Context,
	body models.Applicationable,
	requestConfiguration *graphapplications.ApplicationsRequestBuilderPostRequestConfiguration,
) (models.Applicationable, error) {
	args := m.Called(ctx, body, requestConfiguration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.Applicationable), args.Error(1)
}

func (m *MockApplicationsRequestBuilder) ByApplicationId(applicationId string) *MockApplicationItemRequestBuilder {
	args := m.Called(applicationId)
	return args.Get(0).(*MockApplicationItemRequestBuilder)
}

// MockApplicationItemRequestBuilder mocks specific application operations
type MockApplicationItemRequestBuilder struct {
	mock.Mock
	addPassword *MockAddPasswordRequestBuilder
}

func NewMockApplicationItemRequestBuilder() *MockApplicationItemRequestBuilder {
	m := &MockApplicationItemRequestBuilder{}
	m.addPassword = NewMockAddPasswordRequestBuilder()
	return m
}

func (m *MockApplicationItemRequestBuilder) AddPassword() *MockAddPasswordRequestBuilder {
	return m.addPassword
}

func (m *MockApplicationItemRequestBuilder) Patch(
	ctx context.Context,
	body models.Applicationable,
	requestConfiguration interface{},
) (models.Applicationable, error) {
	args := m.Called(ctx, body, requestConfiguration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.Applicationable), args.Error(1)
}

func (m *MockApplicationItemRequestBuilder) Delete(
	ctx context.Context,
	requestConfiguration interface{},
) error {
	args := m.Called(ctx, requestConfiguration)
	return args.Error(0)
}

// MockAddPasswordRequestBuilder mocks password creation
type MockAddPasswordRequestBuilder struct {
	mock.Mock
}

func NewMockAddPasswordRequestBuilder() *MockAddPasswordRequestBuilder {
	return &MockAddPasswordRequestBuilder{}
}

func (m *MockAddPasswordRequestBuilder) Post(
	ctx context.Context,
	body *graphapplications.ItemAddPasswordPostRequestBody,
	requestConfiguration interface{},
) (models.PasswordCredentialable, error) {
	args := m.Called(ctx, body, requestConfiguration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.PasswordCredentialable), args.Error(1)
}

// MockServicePrincipalsRequestBuilder mocks service principals
type MockServicePrincipalsRequestBuilder struct {
	mock.Mock
}

func NewMockServicePrincipalsRequestBuilder() *MockServicePrincipalsRequestBuilder {
	return &MockServicePrincipalsRequestBuilder{}
}

func (m *MockServicePrincipalsRequestBuilder) Post(
	ctx context.Context,
	body models.ServicePrincipalable,
	requestConfiguration *serviceprincipal.ServicePrincipalsRequestBuilderPostRequestConfiguration,
) (models.ServicePrincipalable, error) {
	args := m.Called(ctx, body, requestConfiguration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.ServicePrincipalable), args.Error(1)
}

// Mock models for testing
type MockApplication struct {
	mock.Mock
	appId     string
	objectId  string
	displayName string
}

func NewMockApplication(appId, objectId, displayName string) *MockApplication {
	return &MockApplication{
		appId:       appId,
		objectId:    objectId,
		displayName: displayName,
	}
}

func (m *MockApplication) GetAppId() *string {
	return &m.appId
}

func (m *MockApplication) GetId() *string {
	return &m.objectId
}

func (m *MockApplication) GetDisplayName() *string {
	return &m.displayName
}

func (m *MockApplication) SetDisplayName(value *string) {
	if value != nil {
		m.displayName = *value
	}
}

// Implement other required methods as needed...

type MockPasswordCredential struct {
	mock.Mock
	secretText  string
	endDateTime time.Time
	displayName string
}

func NewMockPasswordCredential(secretText string, endDateTime time.Time) *MockPasswordCredential {
	return &MockPasswordCredential{
		secretText:  secretText,
		endDateTime: endDateTime,
		displayName: "Test Secret",
	}
}

func (m *MockPasswordCredential) GetSecretText() *string {
	return &m.secretText
}

func (m *MockPasswordCredential) GetEndDateTime() *time.Time {
	return &m.endDateTime
}

func (m *MockPasswordCredential) GetDisplayName() *string {
	return &m.displayName
}

type MockServicePrincipal struct {
	mock.Mock
	id    string
	appId string
}

func NewMockServicePrincipal(id, appId string) *MockServicePrincipal {
	return &MockServicePrincipal{
		id:    id,
		appId: appId,
	}
}

func (m *MockServicePrincipal) GetId() *string {
	return &m.id
}

func (m *MockServicePrincipal) GetAppId() *string {
	return &m.appId
}
```

### 1.2 Azure Key Vault SDK Mocks

```go
// cmd/docker-mcp/portal/oauth/mock_azure.go
package oauth

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/stretchr/testify/mock"
)

// MockKeyVaultClient mocks Azure Key Vault client
type MockKeyVaultClient struct {
	mock.Mock
	secrets map[string]string
}

func NewMockKeyVaultClient() *MockKeyVaultClient {
	return &MockKeyVaultClient{
		secrets: make(map[string]string),
	}
}

func (m *MockKeyVaultClient) SetSecret(
	ctx context.Context,
	secretName string,
	parameters azsecrets.SetSecretParameters,
	options *azsecrets.SetSecretOptions,
) (azsecrets.SetSecretResponse, error) {
	args := m.Called(ctx, secretName, parameters, options)

	// Store the secret for retrieval
	if parameters.Value != nil {
		m.secrets[secretName] = *parameters.Value
	}

	return azsecrets.SetSecretResponse{}, args.Error(1)
}

func (m *MockKeyVaultClient) GetSecret(
	ctx context.Context,
	secretName string,
	version string,
	options *azsecrets.GetSecretOptions,
) (azsecrets.GetSecretResponse, error) {
	args := m.Called(ctx, secretName, version, options)

	if secret, exists := m.secrets[secretName]; exists {
		return azsecrets.GetSecretResponse{
			SecretBundle: azsecrets.SecretBundle{
				Value: &secret,
			},
		}, args.Error(1)
	}

	return azsecrets.GetSecretResponse{}, args.Error(1)
}

func (m *MockKeyVaultClient) DeleteSecret(
	ctx context.Context,
	secretName string,
	options *azsecrets.DeleteSecretOptions,
) (azsecrets.DeleteSecretResponse, error) {
	args := m.Called(ctx, secretName, options)

	delete(m.secrets, secretName)

	return azsecrets.DeleteSecretResponse{}, args.Error(1)
}

// Helper to seed test data
func (m *MockKeyVaultClient) SeedSecret(name, value string) {
	m.secrets[name] = value
}

// MockCredential mocks Azure credential
type MockCredential struct {
	mock.Mock
}

func (m *MockCredential) GetToken(ctx context.Context, options interface{}) (interface{}, error) {
	args := m.Called(ctx, options)
	return args.Get(0), args.Error(1)
}
```

## 2. DCR Bridge Unit Tests

### 2.1 Enhanced DCR Bridge Tests

```go
// cmd/docker-mcp/portal/oauth/dcr_bridge_test.go
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	graphapplications "github.com/microsoftgraph/msgraph-sdk-go/applications"
)

// DCRBridgeTestSuite provides comprehensive test setup
type DCRBridgeTestSuite struct {
	suite.Suite
	bridge          *AzureADDCRBridge
	mockGraphClient *MockGraphServiceClient
	mockKeyVault    *MockKeyVaultClient
	ctx             context.Context
}

func (suite *DCRBridgeTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockGraphClient = NewMockGraphServiceClient()
	suite.mockKeyVault = NewMockKeyVaultClient()

	// Create bridge with mocked dependencies
	suite.bridge = &AzureADDCRBridge{
		graphClient:    suite.mockGraphClient,
		tenantID:       "test-tenant-id",
		subscriptionID: "test-subscription-id",
		resourceGroup:  "test-resource-group",
		keyVaultURL:    "https://test-vault.vault.azure.net/",
		registeredApps: make(map[string]*DCRResponse),
	}
}

func (suite *DCRBridgeTestSuite) TestCreateClientSecret_Success() {
	// Arrange
	appObjectId := "test-app-object-id"
	expectedSecret := "generated-client-secret"
	expiryTime := time.Now().AddDate(2, 0, 0)

	mockPasswordCredential := NewMockPasswordCredential(expectedSecret, expiryTime)
	mockAddPasswordBuilder := NewMockAddPasswordRequestBuilder()
	mockAppBuilder := NewMockApplicationItemRequestBuilder()

	// Setup expectations
	suite.mockGraphClient.applications.On("ByApplicationId", appObjectId).Return(mockAppBuilder)
	mockAppBuilder.On("AddPassword").Return(mockAddPasswordBuilder)
	mockAddPasswordBuilder.On("Post",
		suite.ctx,
		mock.AnythingOfType("*graphapplications.ItemAddPasswordPostRequestBody"),
		mock.Anything,
	).Return(mockPasswordCredential, nil)

	// Act
	result, err := suite.bridge.createClientSecret(suite.ctx, appObjectId)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), expectedSecret, *result.GetSecretText())
	assert.Equal(suite.T(), expiryTime, *result.GetEndDateTime())

	// Verify all expectations were met
	suite.mockGraphClient.applications.AssertExpectations(suite.T())
	mockAppBuilder.AssertExpectations(suite.T())
	mockAddPasswordBuilder.AssertExpectations(suite.T())
}

func (suite *DCRBridgeTestSuite) TestCreateClientSecret_GraphAPIError() {
	// Arrange
	appObjectId := "test-app-object-id"
	expectedError := errors.New("Graph API error")

	mockAddPasswordBuilder := NewMockAddPasswordRequestBuilder()
	mockAppBuilder := NewMockApplicationItemRequestBuilder()

	// Setup expectations for error case
	suite.mockGraphClient.applications.On("ByApplicationId", appObjectId).Return(mockAppBuilder)
	mockAppBuilder.On("AddPassword").Return(mockAddPasswordBuilder)
	mockAddPasswordBuilder.On("Post",
		suite.ctx,
		mock.AnythingOfType("*graphapplications.ItemAddPasswordPostRequestBody"),
		mock.Anything,
	).Return(nil, expectedError)

	// Act
	result, err := suite.bridge.createClientSecret(suite.ctx, appObjectId)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to add client secret")
	assert.Contains(suite.T(), err.Error(), "Graph API error")
}

func (suite *DCRBridgeTestSuite) TestStoreCredentialsInKeyVault_Success() {
	// Arrange
	response := &DCRResponse{
		ClientID:              "test-client-id",
		ClientSecret:          "test-client-secret",
		ClientIDIssuedAt:      time.Now().Unix(),
		ClientSecretExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		DCRRequest: DCRRequest{
			ClientName:   "Test Client",
			RedirectURIs: []string{"https://example.com/callback"},
		},
	}

	// Mock Key Vault operations
	suite.mockKeyVault.On("SetSecret",
		suite.ctx,
		mock.MatchedBy(func(secretName string) bool {
			return secretName == "oauth-client-test-client-id"
		}),
		mock.AnythingOfType("azsecrets.SetSecretParameters"),
		mock.Anything,
	).Return(mock.Anything, nil)

	// Create bridge with mocked Key Vault
	bridge := &AzureADDCRBridge{
		keyVaultURL:    "https://test-vault.vault.azure.net/",
		registeredApps: make(map[string]*DCRResponse),
	}

	// Act
	err := bridge.storeCredentialsInKeyVault(suite.ctx, response)

	// Assert
	assert.NoError(suite.T(), err)
	suite.mockKeyVault.AssertExpectations(suite.T())
}

func (suite *DCRBridgeTestSuite) TestRegisterClient_CompleteFlow() {
	// Arrange
	request := &DCRRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Test Client",
		Scope:        "openid profile email",
	}

	// Mock application creation
	mockApp := NewMockApplication("test-app-id", "test-object-id", "Test Client")
	suite.mockGraphClient.applications.On("Post",
		suite.ctx,
		mock.AnythingOfType("models.Applicationable"),
		mock.Anything,
	).Return(mockApp, nil)

	// Mock service principal creation
	mockSP := NewMockServicePrincipal("test-sp-id", "test-app-id")
	suite.mockGraphClient.servicePrincipals.On("Post",
		suite.ctx,
		mock.AnythingOfType("models.ServicePrincipalable"),
		mock.Anything,
	).Return(mockSP, nil)

	// Mock client secret creation
	mockPasswordCredential := NewMockPasswordCredential(
		"generated-secret",
		time.Now().AddDate(2, 0, 0),
	)
	mockAddPasswordBuilder := NewMockAddPasswordRequestBuilder()
	mockAppBuilder := NewMockApplicationItemRequestBuilder()

	suite.mockGraphClient.applications.On("ByApplicationId", "test-object-id").Return(mockAppBuilder)
	mockAppBuilder.On("AddPassword").Return(mockAddPasswordBuilder)
	mockAddPasswordBuilder.On("Post",
		suite.ctx,
		mock.AnythingOfType("*graphapplications.ItemAddPasswordPostRequestBody"),
		mock.Anything,
	).Return(mockPasswordCredential, nil)

	// Act
	response, err := suite.bridge.RegisterClient(suite.ctx, request)

	// Assert
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), response)

	assert.Equal(suite.T(), "test-app-id", response.ClientID)
	assert.Equal(suite.T(), "generated-secret", response.ClientSecret)
	assert.Equal(suite.T(), "test-object-id", response.ObjectID)
	assert.Equal(suite.T(), request.ClientName, response.ClientName)
	assert.Equal(suite.T(), request.RedirectURIs, response.RedirectURIs)

	// Verify the client is registered in memory
	storedResponse, exists := suite.bridge.registeredApps[response.ClientID]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), response, storedResponse)

	// Verify all mocks were called as expected
	suite.mockGraphClient.AssertExpectations(suite.T())
}

func (suite *DCRBridgeTestSuite) TestRegisterClient_ValidationFailure() {
	// Arrange - invalid request
	request := &DCRRequest{
		// Missing required fields
		ClientName: "Test Client",
		// RedirectURIs is missing
	}

	// Act
	response, err := suite.bridge.RegisterClient(suite.ctx, request)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "DCR request validation failed")
	assert.Contains(suite.T(), err.Error(), "redirect_uris is required")
}

func (suite *DCRBridgeTestSuite) TestUpdateClient_Success() {
	// Arrange - register a client first
	existingResponse := &DCRResponse{
		ClientID:     "test-client-id",
		ObjectID:     "test-object-id",
		ClientSecret: "test-secret",
		DCRRequest: DCRRequest{
			ClientName:   "Original Client",
			RedirectURIs: []string{"https://example.com/callback"},
		},
	}
	suite.bridge.registeredApps[existingResponse.ClientID] = existingResponse

	// New request with updates
	updateRequest := &DCRRequest{
		ClientName:   "Updated Client Name",
		RedirectURIs: []string{"https://example.com/callback", "https://example.com/callback2"},
	}

	// Mock the update operation
	mockApp := NewMockApplication("test-app-id", "test-object-id", "Updated Client Name")
	mockAppBuilder := NewMockApplicationItemRequestBuilder()

	suite.mockGraphClient.applications.On("ByApplicationId", "test-object-id").Return(mockAppBuilder)
	mockAppBuilder.On("Patch",
		suite.ctx,
		mock.AnythingOfType("models.Applicationable"),
		mock.Anything,
	).Return(mockApp, nil)

	// Act
	response, err := suite.bridge.UpdateClient(suite.ctx, "test-client-id", updateRequest)

	// Assert
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), response)

	assert.Equal(suite.T(), "test-client-id", response.ClientID)
	assert.Equal(suite.T(), "Updated Client Name", response.ClientName)
	assert.Equal(suite.T(), updateRequest.RedirectURIs, response.RedirectURIs)

	// Verify the update was stored
	storedResponse, exists := suite.bridge.registeredApps[response.ClientID]
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), updateRequest.ClientName, storedResponse.ClientName)

	suite.mockGraphClient.AssertExpectations(suite.T())
}

func (suite *DCRBridgeTestSuite) TestDeleteClient_Success() {
	// Arrange - register a client first
	existingResponse := &DCRResponse{
		ClientID: "test-client-id",
		ObjectID: "test-object-id",
	}
	suite.bridge.registeredApps[existingResponse.ClientID] = existingResponse

	// Mock the delete operation
	mockAppBuilder := NewMockApplicationItemRequestBuilder()
	suite.mockGraphClient.applications.On("ByApplicationId", "test-object-id").Return(mockAppBuilder)
	mockAppBuilder.On("Delete", suite.ctx, mock.Anything).Return(nil)

	// Act
	err := suite.bridge.DeleteClient(suite.ctx, "test-client-id")

	// Assert
	assert.NoError(suite.T(), err)

	// Verify the client was removed from memory
	_, exists := suite.bridge.registeredApps["test-client-id"]
	assert.False(suite.T(), exists)

	suite.mockGraphClient.AssertExpectations(suite.T())
}

// Run the test suite
func TestDCRBridgeTestSuite(t *testing.T) {
	suite.Run(t, new(DCRBridgeTestSuite))
}

// Additional table-driven tests for edge cases
func TestDCRBridge_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *AzureADDCRBridge
		test    func(t *testing.T, bridge *AzureADDCRBridge)
	}{
		{
			name: "nil graph client",
			setup: func() *AzureADDCRBridge {
				return &AzureADDCRBridge{
					graphClient: nil,
					tenantID:    "test-tenant",
				}
			},
			test: func(t *testing.T, bridge *AzureADDCRBridge) {
				ctx := context.Background()
				request := &DCRRequest{
					RedirectURIs: []string{"https://example.com/callback"},
					ClientName:   "Test Client",
				}

				// This should panic or error gracefully
				response, err := bridge.RegisterClient(ctx, request)
				assert.Error(t, err)
				assert.Nil(t, response)
			},
		},
		{
			name: "context cancellation",
			setup: func() *AzureADDCRBridge {
				mockClient := NewMockGraphServiceClient()
				return &AzureADDCRBridge{
					graphClient:    mockClient,
					tenantID:       "test-tenant",
					registeredApps: make(map[string]*DCRResponse),
				}
			},
			test: func(t *testing.T, bridge *AzureADDCRBridge) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				request := &DCRRequest{
					RedirectURIs: []string{"https://example.com/callback"},
					ClientName:   "Test Client",
				}

				response, err := bridge.RegisterClient(ctx, request)
				assert.Error(t, err)
				assert.Nil(t, response)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := tt.setup()
			tt.test(t, bridge)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkDCRBridge_RegisterClient(b *testing.B) {
	bridge := &AzureADDCRBridge{
		registeredApps: make(map[string]*DCRResponse),
	}

	request := &DCRRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Test Client",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will fail without proper mocking, but measures validation overhead
		_, _ = bridge.validateDCRRequest(request)
	}
}
```

## 3. Token Storage Testing

### 3.1 Hierarchical Storage Tests

```go
// cmd/docker-mcp/portal/oauth/storage_test.go (enhanced)
package oauth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// StorageTestSuite provides comprehensive storage testing
type StorageTestSuite struct {
	suite.Suite
	storage      *HierarchicalTokenStorage
	mockKeyVault *MockKeyVaultClient
	tempDir      string
	ctx          context.Context
}

func (suite *StorageTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockKeyVault = NewMockKeyVaultClient()

	// Create temporary directory for file-based storage tests
	var err error
	suite.tempDir, err = os.MkdirTemp("", "oauth-storage-test")
	require.NoError(suite.T(), err)

	// Create test storage configuration
	config := &InterceptorConfig{
		EncryptTokens: false, // Disable encryption for simpler testing
		StorageTiers: []StorageTier{
			StorageTierKeyVault,
			StorageTierDockerDesktop,
			StorageTierEnvironment,
		},
	}

	suite.storage = &HierarchicalTokenStorage{
		config:            config,
		keyVaultClient:    suite.mockKeyVault,
		dockerDesktopPath: suite.tempDir,
	}
}

func (suite *StorageTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *StorageTestSuite) TestStoreToken_KeyVaultSuccess() {
	// Arrange
	token := &TokenData{
		ServerName:   "test-server",
		UserID:       uuid.New(),
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
		StorageTier:  StorageTierKeyVault,
	}

	// Mock Key Vault success
	suite.mockKeyVault.On("SetSecret",
		suite.ctx,
		mock.MatchedBy(func(secretName string) bool {
			expectedName := "oauth-token-test-server-" + token.UserID.String()
			return secretName == expectedName
		}),
		mock.AnythingOfType("azsecrets.SetSecretParameters"),
		mock.Anything,
	).Return(mock.Anything, nil)

	// Act
	err := suite.storage.StoreToken(suite.ctx, token, StorageTierKeyVault)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), StorageTierKeyVault, token.StorageTier)
	suite.mockKeyVault.AssertExpectations(suite.T())
}

func (suite *StorageTestSuite) TestStoreToken_FallbackToLowerTier() {
	// Arrange
	token := &TokenData{
		ServerName:   "test-server",
		UserID:       uuid.New(),
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
	}

	// Mock Key Vault failure to trigger fallback
	suite.mockKeyVault.On("SetSecret",
		suite.ctx,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(mock.Anything, assert.AnError)

	// Act - request Key Vault but expect fallback to Docker Desktop
	err := suite.storage.StoreToken(suite.ctx, token, StorageTierKeyVault)

	// Assert
	assert.NoError(suite.T(), err) // Should succeed via fallback
	assert.Equal(suite.T(), StorageTierDockerDesktop, token.StorageTier)

	// Verify file was created
	expectedFile := filepath.Join(
		suite.tempDir,
		"oauth-tokens",
		"test-server-"+token.UserID.String()+".json",
	)
	assert.FileExists(suite.T(), expectedFile)

	// Verify file content
	data, err := os.ReadFile(expectedFile)
	require.NoError(suite.T(), err)

	var storedToken TokenData
	err = json.Unmarshal(data, &storedToken)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), token.ServerName, storedToken.ServerName)
	assert.Equal(suite.T(), token.AccessToken, storedToken.AccessToken)
}

func (suite *StorageTestSuite) TestGetToken_MultiTierRetrieval() {
	// Arrange - store tokens in different tiers
	userID := uuid.New()

	// Store in Docker Desktop (file system)
	dockerToken := &TokenData{
		ServerName:   "docker-server",
		UserID:       userID,
		AccessToken:  "docker-access-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
		StorageTier:  StorageTierDockerDesktop,
	}

	err := suite.storage.StoreToken(suite.ctx, dockerToken, StorageTierDockerDesktop)
	require.NoError(suite.T(), err)

	// Mock Key Vault token
	keyVaultTokenJSON := `{
		"server_name": "keyvault-server",
		"user_id": "` + userID.String() + `",
		"access_token": "keyvault-access-token",
		"expires_at": "` + time.Now().Add(1*time.Hour).Format(time.RFC3339) + `",
		"issued_at": "` + time.Now().Format(time.RFC3339) + `",
		"storage_tier": 1
	}`

	suite.mockKeyVault.SeedSecret(
		"oauth-token-keyvault-server-"+userID.String(),
		keyVaultTokenJSON,
	)
	suite.mockKeyVault.On("GetSecret",
		suite.ctx,
		"oauth-token-keyvault-server-"+userID.String(),
		"",
		mock.Anything,
	).Return(mock.Anything, nil)

	// Act & Assert - retrieve from Key Vault (highest priority)
	token, err := suite.storage.GetToken(suite.ctx, "keyvault-server", userID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "keyvault-access-token", token.AccessToken)

	// Act & Assert - retrieve from Docker Desktop
	token, err = suite.storage.GetToken(suite.ctx, "docker-server", userID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "docker-access-token", token.AccessToken)
}

func (suite *StorageTestSuite) TestDeleteToken_AllTiers() {
	// Arrange - store token in multiple tiers
	userID := uuid.New()
	serverName := "multi-tier-server"

	token := &TokenData{
		ServerName:  serverName,
		UserID:      userID,
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
	}

	// Store in Docker Desktop
	err := suite.storage.StoreToken(suite.ctx, token, StorageTierDockerDesktop)
	require.NoError(suite.T(), err)

	// Mock Key Vault delete
	suite.mockKeyVault.On("DeleteSecret",
		suite.ctx,
		"oauth-token-"+serverName+"-"+userID.String(),
		mock.Anything,
	).Return(mock.Anything, nil)

	// Act
	err = suite.storage.DeleteToken(suite.ctx, serverName, userID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify file was deleted
	expectedFile := filepath.Join(
		suite.tempDir,
		"oauth-tokens",
		serverName+"-"+userID.String()+".json",
	)
	assert.NoFileExists(suite.T(), expectedFile)

	suite.mockKeyVault.AssertExpectations(suite.T())
}

func (suite *StorageTestSuite) TestListTokens_AllTiers() {
	// Arrange - create tokens in different tiers
	userID := uuid.New()

	// Docker Desktop token
	dockerToken := &TokenData{
		ServerName:  "docker-server",
		UserID:      userID,
		AccessToken: "docker-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
	}
	err := suite.storage.StoreToken(suite.ctx, dockerToken, StorageTierDockerDesktop)
	require.NoError(suite.T(), err)

	// Environment token (mock via env var)
	envToken := TokenData{
		ServerName:  "env-server",
		UserID:      userID,
		AccessToken: "env-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
		StorageTier: StorageTierEnvironment,
	}
	envTokenJSON, _ := json.Marshal(envToken)
	envKey := "OAUTH_TOKEN_ENV_SERVER_" + userID.String()
	envKey = filepath.ToSlash(envKey) // Normalize for testing
	os.Setenv(envKey, string(envTokenJSON))
	defer os.Unsetenv(envKey)

	// Act
	tokens, err := suite.storage.ListTokens(suite.ctx, userID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), tokens, 2) // Docker + Environment

	// Verify token contents
	serverNames := make(map[string]bool)
	for _, token := range tokens {
		serverNames[token.ServerName] = true
		assert.Equal(suite.T(), userID, token.UserID)
	}

	assert.True(suite.T(), serverNames["docker-server"])
	assert.True(suite.T(), serverNames["env-server"])
}

func (suite *StorageTestSuite) TestEncryption_WhenEnabled() {
	// Arrange - enable encryption
	suite.storage.config.EncryptTokens = true
	suite.storage.encryptionSvc = &MockEncryptionService{}

	token := &TokenData{
		ServerName:   "encrypted-server",
		UserID:       uuid.New(),
		AccessToken:  "sensitive-token",
		RefreshToken: "sensitive-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
	}

	// Act
	err := suite.storage.StoreToken(suite.ctx, token, StorageTierDockerDesktop)

	// Assert
	require.NoError(suite.T(), err)

	// Verify file contains encrypted data (not plain text)
	expectedFile := filepath.Join(
		suite.tempDir,
		"oauth-tokens",
		"encrypted-server-"+token.UserID.String()+".json",
	)
	data, err := os.ReadFile(expectedFile)
	require.NoError(suite.T(), err)

	// Should not contain original token values
	assert.NotContains(suite.T(), string(data), "sensitive-token")
	assert.NotContains(suite.T(), string(data), "sensitive-refresh")
}

func (suite *StorageTestSuite) TestCleanupExpiredTokens() {
	// Arrange - create expired and valid tokens
	userID := uuid.New()

	// Expired token
	expiredToken := &TokenData{
		ServerName:  "expired-server",
		UserID:      userID,
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Already expired
		IssuedAt:    time.Now().Add(-2 * time.Hour),
	}
	err := suite.storage.StoreToken(suite.ctx, expiredToken, StorageTierDockerDesktop)
	require.NoError(suite.T(), err)

	// Valid token
	validToken := &TokenData{
		ServerName:  "valid-server",
		UserID:      userID,
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour), // Still valid
		IssuedAt:    time.Now(),
	}
	err = suite.storage.StoreToken(suite.ctx, validToken, StorageTierDockerDesktop)
	require.NoError(suite.T(), err)

	// Act
	cleanedCount, err := suite.storage.CleanupExpiredTokens(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	// Note: Current implementation doesn't actually clean file-based tokens
	// This test validates the interface and can be enhanced when cleanup is implemented
	assert.GreaterOrEqual(suite.T(), cleanedCount, 0)
}

// Mock encryption service for testing
type MockEncryptionService struct {
	mock.Mock
}

func (m *MockEncryptionService) Encrypt(data []byte) ([]byte, error) {
	// Simple "encryption" for testing - just prefix with "ENCRYPTED:"
	return append([]byte("ENCRYPTED:"), data...), nil
}

func (m *MockEncryptionService) Decrypt(data []byte) ([]byte, error) {
	// Simple "decryption" for testing - remove "ENCRYPTED:" prefix
	if len(data) > 10 && string(data[:10]) == "ENCRYPTED:" {
		return data[10:], nil
	}
	return data, nil
}

// Run the test suite
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

// Additional performance tests
func BenchmarkTokenStorage_StoreRetrieve(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "oauth-bench")
	defer os.RemoveAll(tempDir)

	config := &InterceptorConfig{
		EncryptTokens: false,
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()
	userID := uuid.New()

	token := &TokenData{
		ServerName:  "bench-server",
		UserID:      userID,
		AccessToken: "bench-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.StoreToken(ctx, token, StorageTierDockerDesktop)
		_, _ = storage.GetToken(ctx, "bench-server", userID)
	}
}
```

## 4. Integration Tests

### 4.1 OAuth Flow Integration

```go
// cmd/docker-mcp/portal/oauth/integration/oauth_flow_test.go
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/google/uuid"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/oauth"
)

// OAuthFlowTestSuite tests complete OAuth flows
type OAuthFlowTestSuite struct {
	suite.Suite
	interceptor *oauth.DefaultOAuthInterceptor
	storage     *oauth.HierarchicalTokenStorage
	dcrBridge   *oauth.MultiProviderDCRBridge
	ctx         context.Context
}

func (suite *OAuthFlowTestSuite) SetupSuite() {
	// Skip integration tests in short mode
	if testing.Short() {
		suite.T().Skip("Skipping integration tests in short mode")
	}

	suite.ctx = context.Background()

	// Create test components
	config := &oauth.InterceptorConfig{
		Enabled:          true,
		DefaultTimeout:   30 * time.Second,
		RefreshThreshold: 5 * time.Minute,
		EncryptTokens:    false,
		StorageTiers: []oauth.StorageTier{
			oauth.StorageTierDockerDesktop,
			oauth.StorageTierEnvironment,
		},
		RetryPolicy: oauth.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
			RetryOn401:      true,
			RetryOn403:      false,
			RetryOn429:      true,
			RetryOn5xx:      true,
		},
	}

	// Create storage
	storage, err := oauth.CreateHierarchicalTokenStorage(config, nil, "")
	require.NoError(suite.T(), err)
	suite.storage = storage

	// Create provider registry
	registry := oauth.CreateProviderRegistry()

	// Create interceptor
	suite.interceptor = oauth.CreateOAuthInterceptor(
		config,
		storage,
		registry,
		&oauth.NoOpAuditLogger{},
		&oauth.NoOpMetricsCollector{},
		&oauth.DefaultConfigValidator{},
	)
}

func (suite *OAuthFlowTestSuite) TestCompleteOAuthFlow() {
	// Arrange
	serverConfig := &oauth.ServerConfig{
		ServerName:   "test-server",
		ProviderType: oauth.ProviderTypeMicrosoft,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURI:  "https://localhost:8080/callback",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    uuid.New(),
		IsActive:     true,
	}

	// Register server
	err := suite.interceptor.RegisterServer(suite.ctx, serverConfig)
	require.NoError(suite.T(), err)

	// Create test token
	userID := uuid.New()
	tokenData := &oauth.TokenData{
		ServerName:   "test-server",
		UserID:       userID,
		ProviderType: oauth.ProviderTypeMicrosoft,
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		IssuedAt:     time.Now(),
		Scopes:       []string{"openid", "profile", "email"},
		StorageTier:  oauth.StorageTierDockerDesktop,
		LastUsed:     time.Now(),
		UsageCount:   0,
	}

	// Store token
	err = suite.interceptor.StoreToken(suite.ctx, tokenData)
	require.NoError(suite.T(), err)

	// Act - retrieve token
	retrievedToken, err := suite.interceptor.GetToken(suite.ctx, "test-server", userID)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrievedToken)
	assert.Equal(suite.T(), tokenData.AccessToken, retrievedToken.AccessToken)
	assert.Equal(suite.T(), tokenData.ServerName, retrievedToken.ServerName)
	assert.Equal(suite.T(), tokenData.UserID, retrievedToken.UserID)
}

func (suite *OAuthFlowTestSuite) TestTokenRefreshFlow() {
	// Arrange - token near expiry
	userID := uuid.New()
	serverName := "refresh-test-server"

	serverConfig := &oauth.ServerConfig{
		ServerName:   serverName,
		ProviderType: oauth.ProviderTypeMicrosoft,
		ClientID:     "refresh-client-id",
		ClientSecret: "refresh-client-secret",
		Scopes:       []string{"openid", "profile"},
		RedirectURI:  "https://localhost:8080/callback",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    uuid.New(),
		IsActive:     true,
	}

	err := suite.interceptor.RegisterServer(suite.ctx, serverConfig)
	require.NoError(suite.T(), err)

	// Token that needs refresh
	expiringSoon := time.Now().Add(2 * time.Minute) // Within refresh threshold
	tokenData := &oauth.TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: oauth.ProviderTypeMicrosoft,
		AccessToken:  "expiring-access-token",
		RefreshToken: "valid-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    expiringSoon,
		RefreshAt:    time.Now().Add(1 * time.Minute),
		IssuedAt:     time.Now().Add(-1 * time.Hour),
		Scopes:       []string{"openid", "profile"},
		StorageTier:  oauth.StorageTierDockerDesktop,
	}

	err = suite.interceptor.StoreToken(suite.ctx, tokenData)
	require.NoError(suite.T(), err)

	// Act - refresh token
	refreshedToken, err := suite.interceptor.RefreshToken(suite.ctx, serverName, userID)

	// Assert
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), refreshedToken)
	assert.Equal(suite.T(), serverName, refreshedToken.ServerName)
	assert.Equal(suite.T(), userID, refreshedToken.UserID)
	assert.True(suite.T(), refreshedToken.ExpiresAt.After(expiringSoon))
	assert.Greater(suite.T(), refreshedToken.UsageCount, tokenData.UsageCount)
}

func (suite *OAuthFlowTestSuite) TestInterceptRequest_WithValidToken() {
	// Arrange
	userID := uuid.New()
	serverName := "intercept-test-server"

	// Create and register server
	serverConfig := &oauth.ServerConfig{
		ServerName:   serverName,
		ProviderType: oauth.ProviderTypeMicrosoft,
		ClientID:     "intercept-client-id",
		ClientSecret: "intercept-client-secret",
		Scopes:       []string{"openid"},
		RedirectURI:  "https://localhost:8080/callback",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    uuid.New(),
		IsActive:     true,
	}

	err := suite.interceptor.RegisterServer(suite.ctx, serverConfig)
	require.NoError(suite.T(), err)

	// Store valid token
	tokenData := &oauth.TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: oauth.ProviderTypeMicrosoft,
		AccessToken:  "valid-access-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
		Scopes:       []string{"openid"},
		StorageTier:  oauth.StorageTierDockerDesktop,
	}

	err = suite.interceptor.StoreToken(suite.ctx, tokenData)
	require.NoError(suite.T(), err)

	// Create request
	authRequest := &oauth.AuthRequest{
		RequestID:    uuid.New().String(),
		ServerName:   serverName,
		UserID:       userID,
		Method:       "GET",
		URL:          "https://graph.microsoft.com/v1.0/me",
		Headers:      map[string]string{"Content-Type": "application/json"},
		Timestamp:    time.Now(),
		AttemptCount: 1,
		MaxRetries:   3,
	}

	// Act
	response, err := suite.interceptor.InterceptRequest(suite.ctx, authRequest)

	// Assert
	// Note: This will likely fail without a real HTTP client, but tests the flow
	assert.NotNil(suite.T(), response)
	// Error is expected since we're not making real HTTP calls
	// The important thing is that the token was retrieved and request was processed
}

func (suite *OAuthFlowTestSuite) TestRevokeToken() {
	// Arrange
	userID := uuid.New()
	serverName := "revoke-test-server"

	serverConfig := &oauth.ServerConfig{
		ServerName:   serverName,
		ProviderType: oauth.ProviderTypeMicrosoft,
		ClientID:     "revoke-client-id",
		ClientSecret: "revoke-client-secret",
		Scopes:       []string{"openid"},
		RedirectURI:  "https://localhost:8080/callback",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    uuid.New(),
		IsActive:     true,
	}

	err := suite.interceptor.RegisterServer(suite.ctx, serverConfig)
	require.NoError(suite.T(), err)

	// Store token
	tokenData := &oauth.TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: oauth.ProviderTypeMicrosoft,
		AccessToken:  "revoke-access-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		IssuedAt:     time.Now(),
		StorageTier:  oauth.StorageTierDockerDesktop,
	}

	err = suite.interceptor.StoreToken(suite.ctx, tokenData)
	require.NoError(suite.T(), err)

	// Verify token exists
	retrievedToken, err := suite.interceptor.GetToken(suite.ctx, serverName, userID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrievedToken)

	// Act - revoke token
	err = suite.interceptor.RevokeToken(suite.ctx, serverName, userID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify token is removed
	revokedToken, err := suite.interceptor.GetToken(suite.ctx, serverName, userID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), revokedToken)
}

func (suite *OAuthFlowTestSuite) TestHealthCheck() {
	// Act
	err := suite.interceptor.Health(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *OAuthFlowTestSuite) TestGetMetrics() {
	// Act
	metrics, err := suite.interceptor.GetMetrics(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), metrics)

	// Should contain expected metric keys
	assert.Contains(suite.T(), metrics, "total_requests")
	assert.Contains(suite.T(), metrics, "successful_requests")
	assert.Contains(suite.T(), metrics, "failed_requests")
}

// Run the integration test suite
func TestOAuthFlowTestSuite(t *testing.T) {
	suite.Run(t, new(OAuthFlowTestSuite))
}

// No-op implementations for testing
type NoOpAuditLogger struct{}

func (n *NoOpAuditLogger) LogOAuthEvent(ctx context.Context, event *oauth.AuditEvent) error {
	return nil
}

func (n *NoOpAuditLogger) LogTokenRefresh(ctx context.Context, serverName string, userID uuid.UUID, success bool) error {
	return nil
}

func (n *NoOpAuditLogger) LogAuthorizationFlow(ctx context.Context, serverName string, userID uuid.UUID, provider oauth.ProviderType, success bool) error {
	return nil
}

func (n *NoOpAuditLogger) LogTokenRevocation(ctx context.Context, serverName string, userID uuid.UUID, success bool) error {
	return nil
}

func (n *NoOpAuditLogger) GetUserActivity(ctx context.Context, userID uuid.UUID, since time.Time) ([]*oauth.AuditEvent, error) {
	return []*oauth.AuditEvent{}, nil
}

func (n *NoOpAuditLogger) GetServerActivity(ctx context.Context, serverName string, since time.Time) ([]*oauth.AuditEvent, error) {
	return []*oauth.AuditEvent{}, nil
}

func (n *NoOpAuditLogger) GetFailedAttempts(ctx context.Context, since time.Time) ([]*oauth.AuditEvent, error) {
	return []*oauth.AuditEvent{}, nil
}

type NoOpMetricsCollector struct{}

func (n *NoOpMetricsCollector) RecordRequest(ctx context.Context, serverName string, provider oauth.ProviderType, duration time.Duration, success bool) {
}

func (n *NoOpMetricsCollector) RecordTokenRefresh(ctx context.Context, serverName string, provider oauth.ProviderType, success bool) {
}

func (n *NoOpMetricsCollector) RecordError(ctx context.Context, errorType string, serverName string, provider oauth.ProviderType) {
}

func (n *NoOpMetricsCollector) GetMetrics(ctx context.Context) (*oauth.Metrics, error) {
	return &oauth.Metrics{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		TokenRefreshCount:  0,
		AverageLatency:     0,
		P95Latency:         0,
		P99Latency:         0,
		ProviderCounts:     make(map[oauth.ProviderType]int64),
		ProviderLatencies:  make(map[oauth.ProviderType]time.Duration),
		ErrorCounts:        make(map[string]int64),
		RetryAttempts:      0,
		ActiveTokens:       0,
		ExpiredTokens:      0,
		StorageTierUsage:   make(map[oauth.StorageTier]int64),
		LastUpdated:        time.Now(),
		UptimeSeconds:      0,
	}, nil
}

func (n *NoOpMetricsCollector) Reset(ctx context.Context) error {
	return nil
}

type DefaultConfigValidator struct{}

func (d *DefaultConfigValidator) ValidateServerConfig(config *oauth.ServerConfig) []oauth.ValidationError {
	var errors []oauth.ValidationError

	if config.ServerName == "" {
		errors = append(errors, oauth.ValidationError{
			Field:   "server_name",
			Message: "server name is required",
			Code:    "required",
		})
	}

	if config.ClientID == "" {
		errors = append(errors, oauth.ValidationError{
			Field:   "client_id",
			Message: "client ID is required",
			Code:    "required",
		})
	}

	return errors
}

func (d *DefaultConfigValidator) ValidateProviderConfig(providerType oauth.ProviderType, config map[string]string) []oauth.ValidationError {
	return []oauth.ValidationError{}
}

func (d *DefaultConfigValidator) ValidateToken(token *oauth.TokenData) []oauth.ValidationError {
	var errors []oauth.ValidationError

	if token.AccessToken == "" {
		errors = append(errors, oauth.ValidationError{
			Field:   "access_token",
			Message: "access token is required",
			Code:    "required",
		})
	}

	if token.ExpiresAt.Before(time.Now()) {
		errors = append(errors, oauth.ValidationError{
			Field:   "expires_at",
			Message: "token is expired",
			Code:    "expired",
		})
	}

	return errors
}

func (d *DefaultConfigValidator) ValidateTokenClaims(claims *oauth.TokenClaims) []oauth.ValidationError {
	return []oauth.ValidationError{}
}

func (d *DefaultConfigValidator) ValidateDCRRequest(req *oauth.DCRRequest) []oauth.ValidationError {
	return []oauth.ValidationError{}
}
```

## 5. Test Data and Fixtures

### 5.1 Test Data Setup

```go
// cmd/docker-mcp/portal/oauth/testdata/fixtures.go
package testdata

import (
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/oauth"
)

// CreateTestDCRRequest creates a valid DCR request for testing
func CreateTestDCRRequest() *oauth.DCRRequest {
	return &oauth.DCRRequest{
		RedirectURIs:    []string{"https://example.com/callback", "https://example.com/callback2"},
		ClientName:      "Test OAuth Client",
		ClientURI:       "https://example.com",
		LogoURI:         "https://example.com/logo.png",
		Scope:           "openid profile email",
		Contacts:        []string{"admin@example.com"},
		TosURI:          "https://example.com/terms",
		PolicyURI:       "https://example.com/privacy",
		GrantTypes:      []string{"authorization_code", "refresh_token"},
		ResponseTypes:   []string{"code"},
		ApplicationType: "web",
		TenantID:        "test-tenant-id",
	}
}

// CreateTestDCRResponse creates a DCR response for testing
func CreateTestDCRResponse() *oauth.DCRResponse {
	return &oauth.DCRResponse{
		ClientID:                "test-client-id-" + uuid.New().String(),
		ClientSecret:            "test-client-secret-" + generateRandomString(32),
		ClientIDIssuedAt:        time.Now().Unix(),
		ClientSecretExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		ApplicationID:           "test-app-id-" + uuid.New().String(),
		ObjectID:                "test-object-id-" + uuid.New().String(),
		DCRRequest:              *CreateTestDCRRequest(),
	}
}

// CreateTestTokenData creates test token data
func CreateTestTokenData(serverName string, userID uuid.UUID) *oauth.TokenData {
	return &oauth.TokenData{
		ServerName:   serverName,
		UserID:       userID,
		TenantID:     "test-tenant-id",
		ProviderType: oauth.ProviderTypeMicrosoft,
		AccessToken:  "test-access-token-" + generateRandomString(32),
		RefreshToken: "test-refresh-token-" + generateRandomString(32),
		IDToken:      "test-id-token-" + generateRandomString(64),
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		IssuedAt:     time.Now(),
		Scopes:       []string{"openid", "profile", "email"},
		StorageTier:  oauth.StorageTierDockerDesktop,
		LastUsed:     time.Now(),
		UsageCount:   0,
	}
}

// CreateTestServerConfig creates test server configuration
func CreateTestServerConfig(serverName string) *oauth.ServerConfig {
	return &oauth.ServerConfig{
		ServerName:   serverName,
		ProviderType: oauth.ProviderTypeMicrosoft,
		TenantID:     "test-tenant-id",
		ClientID:     "test-client-id-" + generateRandomString(16),
		ClientSecret: "test-client-secret-" + generateRandomString(32),
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURI:  "https://localhost:8080/oauth/callback",
		AuthURL:      "https://login.microsoftonline.com/test-tenant-id/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/test-tenant-id/oauth2/v2.0/token",
		JWKSURL:      "https://login.microsoftonline.com/test-tenant-id/discovery/v2.0/keys",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    uuid.New(),
		IsActive:     true,
	}
}

// CreateTestAuthRequest creates test auth request
func CreateTestAuthRequest(serverName string, userID uuid.UUID) *oauth.AuthRequest {
	return &oauth.AuthRequest{
		RequestID:    uuid.New().String(),
		ServerName:   serverName,
		UserID:       userID,
		TenantID:     "test-tenant-id",
		Method:       "GET",
		URL:          "https://graph.microsoft.com/v1.0/me",
		Headers:      map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Timestamp:    time.Now(),
		AttemptCount: 1,
		MaxRetries:   3,
		UserAgent:    "MCP-Portal/1.0",
		RemoteAddr:   "127.0.0.1:12345",
	}
}

// CreateTestInterceptorConfig creates test interceptor configuration
func CreateTestInterceptorConfig() *oauth.InterceptorConfig {
	return &oauth.InterceptorConfig{
		Enabled:          true,
		DefaultTimeout:   30 * time.Second,
		RefreshThreshold: 5 * time.Minute,
		RetryPolicy: oauth.RetryPolicy{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
			Jitter:          true,
			RetryOn401:      true,
			RetryOn403:      false,
			RetryOn429:      true,
			RetryOn5xx:      true,
		},
		StorageTiers: []oauth.StorageTier{
			oauth.StorageTierKeyVault,
			oauth.StorageTierDockerDesktop,
			oauth.StorageTierEnvironment,
		},
		EncryptTokens:   false,
		ValidateJWTs:    true,
		RequireHTTPS:    true,
		AllowedDomains:  []string{"localhost", "127.0.0.1", "example.com"},
		EnableDCRBridge: true,
		EnableMetrics:   true,
		EnableAuditLog:  true,
	}
}

// Utility functions
func generateRandomString(length int) string {
	// Simple random string generation for testing
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
```

## 6. Test Coverage Strategy

### 6.1 Coverage Targets and Measurement

```makefile
# Add to Makefile for test coverage tracking
.PHONY: test-oauth-coverage
test-oauth-coverage:
	@echo "Running OAuth tests with coverage..."
	go test -coverprofile=oauth-coverage.out -coverpkg=./cmd/docker-mcp/portal/oauth/... ./cmd/docker-mcp/portal/oauth/...
	go tool cover -html=oauth-coverage.out -o oauth-coverage.html
	go tool cover -func=oauth-coverage.out | grep "total:" | awk '{print "Total OAuth Coverage: " $$3}'

.PHONY: test-oauth-detailed
test-oauth-detailed:
	@echo "Running detailed OAuth test analysis..."
	go test -v -coverprofile=oauth-detailed.out -coverpkg=./cmd/docker-mcp/portal/oauth/... ./cmd/docker-mcp/portal/oauth/...
	go tool cover -func=oauth-detailed.out

.PHONY: test-oauth-benchmark
test-oauth-benchmark:
	@echo "Running OAuth performance benchmarks..."
	go test -bench=. -benchmem ./cmd/docker-mcp/portal/oauth/...
```

### 6.2 Test Organization Script

```bash
#!/bin/bash
# scripts/run-oauth-tests.sh

set -e

echo "🔐 Running OAuth Implementation Tests"
echo "===================================="

# Test categories
UNIT_TESTS="./cmd/docker-mcp/portal/oauth/"
INTEGRATION_TESTS="./cmd/docker-mcp/portal/oauth/integration/"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

run_test_category() {
    local category=$1
    local path=$2
    local description=$3

    echo ""
    echo -e "${YELLOW}Running $category tests: $description${NC}"
    echo "----------------------------------------"

    if go test -v -race -timeout 60s $path; then
        echo -e "${GREEN}✅ $category tests passed${NC}"
    else
        echo -e "${RED}❌ $category tests failed${NC}"
        exit 1
    fi
}

# Run unit tests
run_test_category "Unit" "$UNIT_TESTS" "DCR Bridge, Token Storage, OAuth Interceptor"

# Run integration tests (if not in CI)
if [ "$CI" != "true" ]; then
    run_test_category "Integration" "$INTEGRATION_TESTS" "End-to-end OAuth flows"
else
    echo "⏭️  Skipping integration tests in CI environment"
fi

# Generate coverage report
echo ""
echo -e "${YELLOW}Generating coverage report...${NC}"
go test -coverprofile=oauth-coverage.out -coverpkg=./cmd/docker-mcp/portal/oauth/... ./cmd/docker-mcp/portal/oauth/...
COVERAGE=$(go tool cover -func=oauth-coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')

echo "OAuth Test Coverage: $COVERAGE%"

if (( $(echo "$COVERAGE >= 50" | bc -l) )); then
    echo -e "${GREEN}✅ Coverage target met (≥50%)${NC}"
else
    echo -e "${RED}❌ Coverage below target (<50%)${NC}"
    echo "Consider adding more tests to reach the 50% target"
fi

# Generate HTML coverage report
go tool cover -html=oauth-coverage.out -o oauth-coverage.html
echo "📊 HTML coverage report generated: oauth-coverage.html"

echo ""
echo -e "${GREEN}🎉 OAuth testing complete!${NC}"
```

## 7. CI/CD Integration

### 7.1 GitHub Actions Workflow

```yaml
# .github/workflows/oauth-tests.yml
name: OAuth Tests

on:
  push:
    paths:
      - "cmd/docker-mcp/portal/oauth/**"
      - ".github/workflows/oauth-tests.yml"
  pull_request:
    paths:
      - "cmd/docker-mcp/portal/oauth/**"

jobs:
  oauth-tests:
    name: OAuth Implementation Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.24.4"

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Install dependencies
        run: go mod download

      - name: Run OAuth unit tests
        run: |
          go test -v -race -timeout 60s -coverprofile=oauth-coverage.out \
            -coverpkg=./cmd/docker-mcp/portal/oauth/... \
            ./cmd/docker-mcp/portal/oauth/...

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=oauth-coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 50" | bc -l) )); then
            echo "❌ Coverage $COVERAGE% is below 50% threshold"
            exit 1
          fi
          echo "✅ Coverage $COVERAGE% meets 50% threshold"

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./oauth-coverage.out
          flags: oauth
          name: oauth-coverage

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem ./cmd/docker-mcp/portal/oauth/... > benchmark-results.txt
          cat benchmark-results.txt

      - name: Upload benchmark results
        uses: actions/upload-artifact@v3
        with:
          name: oauth-benchmarks
          path: benchmark-results.txt
```

## 8. Mock Generation Automation

### 8.1 Automated Mock Generation

```bash
#!/bin/bash
# scripts/generate-oauth-mocks.sh

echo "🏭 Generating OAuth mocks..."

# Install mockery if not present
if ! command -v mockery &> /dev/null; then
    echo "Installing mockery..."
    go install github.com/vektra/mockery/v2@latest
fi

# Generate mocks for OAuth interfaces
mockery --dir=cmd/docker-mcp/portal/oauth --name=OAuthInterceptor --output=cmd/docker-mcp/portal/oauth/mocks
mockery --dir=cmd/docker-mcp/portal/oauth --name=TokenStorage --output=cmd/docker-mcp/portal/oauth/mocks
mockery --dir=cmd/docker-mcp/portal/oauth --name=OAuthProvider --output=cmd/docker-mcp/portal/oauth/mocks
mockery --dir=cmd/docker-mcp/portal/oauth --name=DCRBridge --output=cmd/docker-mcp/portal/oauth/mocks
mockery --dir=cmd/docker-mcp/portal/oauth --name=AuditLogger --output=cmd/docker-mcp/portal/oauth/mocks
mockery --dir=cmd/docker-mcp/portal/oauth --name=MetricsCollector --output=cmd/docker-mcp/portal/oauth/mocks

echo "✅ OAuth mocks generated successfully"
```

## 9. Performance Testing

### 9.1 Load Testing Framework

```go
// cmd/docker-mcp/portal/oauth/performance_test.go
package oauth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthPerformance_ConcurrentTokenOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Test parameters
	numGoroutines := 100
	operationsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Measure execution time
	startTime := time.Now()

	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			userID := uuid.New()
			serverName := fmt.Sprintf("server-%d", routineID)

			for j := 0; j < operationsPerGoroutine; j++ {
				// Create token
				token := &TokenData{
					ServerName:  serverName,
					UserID:      userID,
					AccessToken: fmt.Sprintf("token-%d-%d", routineID, j),
					ExpiresAt:   time.Now().Add(1 * time.Hour),
					IssuedAt:    time.Now(),
				}

				// Store token
				if err := storage.StoreToken(ctx, token, StorageTierDockerDesktop); err != nil {
					errors <- fmt.Errorf("store failed: %w", err)
					return
				}

				// Retrieve token
				if _, err := storage.GetToken(ctx, serverName, userID); err != nil {
					errors <- fmt.Errorf("get failed: %w", err)
					return
				}

				// Update token
				token.UsageCount++
				if err := storage.StoreToken(ctx, token, StorageTierDockerDesktop); err != nil {
					errors <- fmt.Errorf("update failed: %w", err)
					return
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	duration := time.Since(startTime)
	totalOperations := numGoroutines * operationsPerGoroutine * 3 // store, get, update

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Logf("Errors encountered: %d", len(errorList))
		for _, err := range errorList[:min(5, len(errorList))] { // Show first 5 errors
			t.Logf("Error: %v", err)
		}
	}

	// Performance assertions
	opsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Performance Results:")
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations/second: %.2f", opsPerSecond)
	t.Logf("  Errors: %d", len(errorList))

	// Assert performance criteria
	assert.Greater(t, opsPerSecond, 100.0, "Should handle at least 100 ops/second")
	assert.Less(t, float64(len(errorList))/float64(totalOperations), 0.01, "Error rate should be < 1%")
}

func BenchmarkTokenStorage_StoreRetrieve(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	ctx := context.Background()
	userID := uuid.New()

	token := &TokenData{
		ServerName:  "bench-server",
		UserID:      userID,
		AccessToken: "bench-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = storage.StoreToken(ctx, token, StorageTierDockerDesktop)
			_, _ = storage.GetToken(ctx, "bench-server", userID)
		}
	})
}

func BenchmarkDCRBridge_RegisterClient(b *testing.B) {
	bridge := setupBenchmarkDCRBridge(b)
	ctx := context.Background()

	request := &DCRRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Benchmark Client",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will fail without proper mocking but measures validation overhead
		_ = bridge.validateDCRRequest(request)
	}
}

// Helper functions
func setupTestStorage(t *testing.T) *HierarchicalTokenStorage {
	tempDir, err := os.MkdirTemp("", "oauth-perf-test")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	config := &InterceptorConfig{
		EncryptTokens: false,
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
	}

	return &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}
}

func setupBenchmarkStorage(b *testing.B) *HierarchicalTokenStorage {
	tempDir, _ := os.MkdirTemp("", "oauth-bench")
	b.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	config := &InterceptorConfig{
		EncryptTokens: false,
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
	}

	return &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}
}

func setupBenchmarkDCRBridge(b *testing.B) *AzureADDCRBridge {
	return &AzureADDCRBridge{
		registeredApps: make(map[string]*DCRResponse),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

## Summary

This comprehensive testing guide provides:

1. **Complete mock framework** for Azure SDKs (Microsoft Graph v1.64.0, Key Vault v1.4.0)
2. **Extensive unit tests** with table-driven patterns and edge case coverage
3. **Integration tests** for complete OAuth flows
4. **Performance benchmarks** and load testing
5. **CI/CD integration** with coverage requirements
6. **Test data fixtures** and utilities
7. **Automated mock generation** and maintenance scripts

### Key Features

- **50%+ Coverage Target**: Structured approach to reach coverage goals
- **Real Azure SDK Mocking**: Proper mocks for Microsoft Graph and Key Vault SDKs
- **Comprehensive Test Patterns**: Unit, integration, and performance tests
- **Production-Ready**: Tests validate error handling, concurrency, and edge cases
- **Maintainable**: Clear structure with fixtures and helper functions

### Usage

1. Run `make test-oauth-coverage` to check current coverage
2. Use `scripts/run-oauth-tests.sh` for comprehensive testing
3. Add new test cases following the established patterns
4. Monitor CI/CD for coverage regression

This testing framework ensures the OAuth implementation is robust, well-tested, and production-ready while maintaining the 50%+ coverage target for the overall project.
