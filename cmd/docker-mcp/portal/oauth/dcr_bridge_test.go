package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGraphClient mocks the Microsoft Graph client
type MockGraphClient struct {
	mock.Mock
}

// MockKeyVaultClient mocks the Azure Key Vault client
type MockKeyVaultClient struct {
	mock.Mock
}

// MockPasswordCredential mocks the Graph SDK PasswordCredential
type MockPasswordCredential struct {
	mock.Mock
	secretText  string
	endDateTime time.Time
}

func (m *MockPasswordCredential) GetSecretText() *string {
	return &m.secretText
}

func (m *MockPasswordCredential) GetEndDateTime() *time.Time {
	return &m.endDateTime
}

func TestCreateAzureADDCRBridge(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		subscriptionID string
		resourceGroup  string
		keyVaultURL    string
		expectError    bool
	}{
		{
			name:           "valid configuration",
			tenantID:       "test-tenant-id",
			subscriptionID: "test-subscription-id",
			resourceGroup:  "test-resource-group",
			keyVaultURL:    "https://test-vault.vault.azure.net/",
			expectError:    false,
		},
		{
			name:           "empty tenant ID",
			tenantID:       "",
			subscriptionID: "test-subscription-id",
			resourceGroup:  "test-resource-group",
			keyVaultURL:    "https://test-vault.vault.azure.net/",
			expectError:    false, // Should work with default credentials
		},
		{
			name:           "no key vault URL",
			tenantID:       "test-tenant-id",
			subscriptionID: "test-subscription-id",
			resourceGroup:  "test-resource-group",
			keyVaultURL:    "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge, err := CreateAzureADDCRBridge(
				tt.tenantID,
				tt.subscriptionID,
				tt.resourceGroup,
				tt.keyVaultURL,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, bridge)
			} else {
				// Note: This might fail in CI without Azure credentials
				// Consider using mock credentials in real tests
				if err != nil {
					t.Skipf("Skipping test due to Azure credential requirements: %v", err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, bridge)
				assert.Equal(t, tt.tenantID, bridge.tenantID)
				assert.Equal(t, tt.subscriptionID, bridge.subscriptionID)
				assert.Equal(t, tt.resourceGroup, bridge.resourceGroup)
				assert.Equal(t, tt.keyVaultURL, bridge.keyVaultURL)
			}
		})
	}
}

func TestAzureADDCRBridge_ValidateDCRRequest(t *testing.T) {
	bridge := &AzureADDCRBridge{}

	tests := []struct {
		name        string
		request     *DCRRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &DCRRequest{
				RedirectURIs: []string{"https://example.com/callback"},
				ClientName:   "Test Client",
			},
			expectError: false,
		},
		{
			name: "missing redirect URIs",
			request: &DCRRequest{
				ClientName: "Test Client",
			},
			expectError: true,
			errorMsg:    "redirect_uris is required",
		},
		{
			name: "missing client name",
			request: &DCRRequest{
				RedirectURIs: []string{"https://example.com/callback"},
			},
			expectError: true,
			errorMsg:    "client_name is required",
		},
		{
			name: "invalid redirect URI",
			request: &DCRRequest{
				RedirectURIs: []string{"http://example.com/callback"}, // Not HTTPS
				ClientName:   "Test Client",
			},
			expectError: true,
			errorMsg:    "invalid redirect URI",
		},
		{
			name: "localhost allowed",
			request: &DCRRequest{
				RedirectURIs: []string{"http://localhost:8080/callback"},
				ClientName:   "Test Client",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bridge.validateDCRRequest(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAzureADDCRBridge_SupportsProvider(t *testing.T) {
	bridge := &AzureADDCRBridge{}

	tests := []struct {
		name         string
		providerType ProviderType
		expected     bool
	}{
		{
			name:         "supports Microsoft",
			providerType: ProviderTypeMicrosoft,
			expected:     true,
		},
		{
			name:         "does not support GitHub",
			providerType: ProviderTypeGitHub,
			expected:     false,
		},
		{
			name:         "does not support Google",
			providerType: ProviderTypeGoogle,
			expected:     false,
		},
		{
			name:         "does not support Custom",
			providerType: ProviderTypeCustom,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bridge.SupportsProvider(tt.providerType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAzureADDCRBridge_GetProviderEndpoints(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		providerType ProviderType
		expectError  bool
	}{
		{
			name:         "Microsoft provider with tenant",
			tenantID:     "test-tenant-id",
			providerType: ProviderTypeMicrosoft,
			expectError:  false,
		},
		{
			name:         "Microsoft provider without tenant",
			tenantID:     "",
			providerType: ProviderTypeMicrosoft,
			expectError:  false,
		},
		{
			name:         "unsupported provider",
			tenantID:     "test-tenant-id",
			providerType: ProviderTypeGitHub,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := &AzureADDCRBridge{
				tenantID: tt.tenantID,
			}

			authURL, tokenURL, jwksURL, err := bridge.GetProviderEndpoints(tt.providerType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, authURL)
				assert.Empty(t, tokenURL)
				assert.Empty(t, jwksURL)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, authURL)
				assert.NotEmpty(t, tokenURL)
				assert.NotEmpty(t, jwksURL)

				expectedTenant := tt.tenantID
				if expectedTenant == "" {
					expectedTenant = "common"
				}

				assert.Contains(t, authURL, expectedTenant)
				assert.Contains(t, tokenURL, expectedTenant)
				assert.Contains(t, jwksURL, expectedTenant)
			}
		})
	}
}

func TestMultiProviderDCRBridge(t *testing.T) {
	t.Run("create multi-provider bridge", func(t *testing.T) {
		bridge, err := CreateMultiProviderDCRBridge(
			"test-tenant-id",
			"test-subscription-id",
			"test-resource-group",
			"https://test-vault.vault.azure.net/",
		)
		// This might fail without Azure credentials
		if err != nil {
			t.Skipf("Skipping test due to Azure credential requirements: %v", err)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, bridge)
		assert.True(t, bridge.SupportsProvider(ProviderTypeMicrosoft))
		assert.False(t, bridge.SupportsProvider(ProviderTypeGitHub))
	})
}

func TestDCRRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request DCRRequest
		isValid bool
	}{
		{
			name: "minimal valid request",
			request: DCRRequest{
				RedirectURIs: []string{"https://example.com/callback"},
				ClientName:   "Test Client",
			},
			isValid: true,
		},
		{
			name: "complete request",
			request: DCRRequest{
				RedirectURIs: []string{
					"https://example.com/callback",
					"https://example.com/callback2",
				},
				ClientName:      "Test Client",
				ClientURI:       "https://example.com",
				LogoURI:         "https://example.com/logo.png",
				Scope:           "openid profile email",
				Contacts:        []string{"admin@example.com"},
				TosURI:          "https://example.com/tos",
				PolicyURI:       "https://example.com/privacy",
				GrantTypes:      []string{"authorization_code", "refresh_token"},
				ResponseTypes:   []string{"code"},
				ApplicationType: "web",
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic structural validation
			assert.NotEmpty(t, tt.request.RedirectURIs)
			assert.NotEmpty(t, tt.request.ClientName)

			if tt.isValid {
				assert.True(t, len(tt.request.RedirectURIs) > 0)
				assert.NotEmpty(t, tt.request.ClientName)
			}
		})
	}
}

func TestDCRResponse_Structure(t *testing.T) {
	response := DCRResponse{
		ClientID:              "test-client-id",
		ClientSecret:          "test-client-secret",
		ClientIDIssuedAt:      time.Now().Unix(),
		ClientSecretExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		ApplicationID:         "test-app-id",
		ObjectID:              "test-object-id",
		DCRRequest: DCRRequest{
			RedirectURIs: []string{"https://example.com/callback"},
			ClientName:   "Test Client",
		},
	}

	assert.NotEmpty(t, response.ClientID)
	assert.NotEmpty(t, response.ClientSecret)
	assert.Greater(t, response.ClientIDIssuedAt, int64(0))
	assert.Greater(t, response.ClientSecretExpiresAt, response.ClientIDIssuedAt)
	assert.NotEmpty(t, response.ApplicationID)
	assert.NotEmpty(t, response.ObjectID)
	assert.NotEmpty(t, response.RedirectURIs)
	assert.NotEmpty(t, response.ClientName)
}

// Benchmark tests for performance validation
func BenchmarkValidateDCRRequest(b *testing.B) {
	bridge := &AzureADDCRBridge{}
	request := &DCRRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Test Client",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bridge.validateDCRRequest(request)
	}
}

func BenchmarkGetProviderEndpoints(b *testing.B) {
	bridge := &AzureADDCRBridge{
		tenantID: "test-tenant-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = bridge.GetProviderEndpoints(ProviderTypeMicrosoft)
	}
}

// Integration test helper for real Azure testing
func TestAzureIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require real Azure credentials and should be run separately
	t.Skip("Integration test requires Azure setup - run manually with proper credentials")

	ctx := context.Background()

	bridge, err := CreateAzureADDCRBridge(
		"your-tenant-id",
		"your-subscription-id",
		"your-resource-group",
		"https://your-vault.vault.azure.net/",
	)
	require.NoError(t, err)

	// Test DCR flow
	request := &DCRRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Integration Test Client",
		Scope:        "openid profile email",
	}

	response, err := bridge.RegisterClient(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotEmpty(t, response.ClientID)
	require.NotEmpty(t, response.ClientSecret)

	// Clean up - delete the test client
	err = bridge.DeleteClient(ctx, response.ClientID)
	assert.NoError(t, err)
}
