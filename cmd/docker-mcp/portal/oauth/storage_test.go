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
	"github.com/stretchr/testify/require"
)

// MockEncryptionService for testing encryption functionality
type MockEncryptionService struct {
	encryptFunc func([]byte) ([]byte, error)
	decryptFunc func([]byte) ([]byte, error)
}

func (m *MockEncryptionService) Encrypt(data []byte) ([]byte, error) {
	if m.encryptFunc != nil {
		return m.encryptFunc(data)
	}
	// Simple mock encryption - just prefix with "encrypted:"
	return append([]byte("encrypted:"), data...), nil
}

func (m *MockEncryptionService) Decrypt(data []byte) ([]byte, error) {
	if m.decryptFunc != nil {
		return m.decryptFunc(data)
	}
	// Simple mock decryption - remove "encrypted:" prefix
	if len(data) > 10 && string(data[:10]) == "encrypted:" {
		return data[10:], nil
	}
	return data, nil
}

func TestCreateHierarchicalTokenStorage(t *testing.T) {
	tests := []struct {
		name        string
		config      *InterceptorConfig
		keyVaultURL string
		expectError bool
	}{
		{
			name: "valid configuration with Key Vault",
			config: &InterceptorConfig{
				StorageTiers: []StorageTier{
					StorageTierKeyVault,
					StorageTierDockerDesktop,
					StorageTierEnvironment,
				},
				EncryptTokens: false,
			},
			keyVaultURL: "https://test-vault.vault.azure.net/",
			expectError: false,
		},
		{
			name: "valid configuration without Key Vault",
			config: &InterceptorConfig{
				StorageTiers:  []StorageTier{StorageTierDockerDesktop, StorageTierEnvironment},
				EncryptTokens: true,
			},
			keyVaultURL: "",
			expectError: false,
		},
		{
			name: "invalid Key Vault URL",
			config: &InterceptorConfig{
				StorageTiers:  []StorageTier{StorageTierKeyVault},
				EncryptTokens: false,
			},
			keyVaultURL: "invalid-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encSvc := &MockEncryptionService{}

			storage, err := CreateHierarchicalTokenStorage(tt.config, encSvc, tt.keyVaultURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, storage)
			} else {
				// Key Vault client creation might fail without proper Azure credentials
				if err != nil && tt.keyVaultURL != "" {
					t.Skipf("Skipping test due to Azure credential requirements: %v", err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, storage)
				assert.Equal(t, tt.config, storage.config)
				assert.Equal(t, encSvc, storage.encryptionSvc)
			}
		})
	}
}

func TestHierarchicalTokenStorage_StorageTierAvailability(t *testing.T) {
	config := &InterceptorConfig{
		StorageTiers: []StorageTier{StorageTierDockerDesktop, StorageTierEnvironment},
	}
	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: "/tmp/test-docker",
	}

	tests := []struct {
		name     string
		tier     StorageTier
		expected bool
	}{
		{
			name:     "Key Vault not available",
			tier:     StorageTierKeyVault,
			expected: false,
		},
		{
			name:     "Docker Desktop available",
			tier:     StorageTierDockerDesktop,
			expected: true,
		},
		{
			name:     "Environment always available",
			tier:     StorageTierEnvironment,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.isStorageTierAvailable(tt.tier)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHierarchicalTokenStorage_DockerDesktopOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	config := &InterceptorConfig{
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
		EncryptTokens: false,
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()
	userID := uuid.New()
	serverName := "test-server"

	// Create test token
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeMicrosoft,
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
		Scopes:       []string{"openid", "profile"},
		StorageTier:  StorageTierDockerDesktop,
	}

	t.Run("store token in Docker Desktop", func(t *testing.T) {
		err := storage.storeTokenInDockerDesktop(ctx, token)
		assert.NoError(t, err)

		// Verify file exists
		filename := filepath.Join(tempDir, "oauth-tokens", serverName+"-"+userID.String()+".json")
		assert.FileExists(t, filename)

		// Verify file content
		data, err := os.ReadFile(filename)
		assert.NoError(t, err)

		var storedToken TokenData
		err = json.Unmarshal(data, &storedToken)
		assert.NoError(t, err)
		assert.Equal(t, token.AccessToken, storedToken.AccessToken)
		assert.Equal(t, token.ServerName, storedToken.ServerName)
	})

	t.Run("get token from Docker Desktop", func(t *testing.T) {
		retrievedToken, err := storage.getTokenFromDockerDesktop(ctx, serverName, userID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedToken)
		assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)
		assert.Equal(t, token.ServerName, retrievedToken.ServerName)
		assert.Equal(t, token.UserID, retrievedToken.UserID)
	})

	t.Run("list tokens from Docker Desktop", func(t *testing.T) {
		tokens, err := storage.listTokensFromDockerDesktop(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, token.AccessToken, tokens[0].AccessToken)
	})

	t.Run("delete token from Docker Desktop", func(t *testing.T) {
		err := storage.deleteTokenFromDockerDesktop(ctx, serverName, userID)
		assert.NoError(t, err)

		// Verify file is deleted
		filename := filepath.Join(tempDir, "oauth-tokens", serverName+"-"+userID.String()+".json")
		assert.NoFileExists(t, filename)
	})

	t.Run("get non-existent token", func(t *testing.T) {
		_, err := storage.getTokenFromDockerDesktop(ctx, "non-existent", userID)
		assert.Error(t, err)
	})
}

func TestHierarchicalTokenStorage_EnvironmentOperations(t *testing.T) {
	config := &InterceptorConfig{
		StorageTiers:  []StorageTier{StorageTierEnvironment},
		EncryptTokens: false,
	}

	storage := &HierarchicalTokenStorage{
		config: config,
	}

	ctx := context.Background()
	userID := uuid.New()
	serverName := "test-server"

	// Create test token JSON
	token := &TokenData{
		ServerName:   serverName,
		UserID:       userID,
		ProviderType: ProviderTypeMicrosoft,
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		StorageTier:  StorageTierEnvironment,
	}

	tokenJSON, err := json.Marshal(token)
	require.NoError(t, err)

	// Set environment variable
	envKey := "OAUTH_TOKEN_TEST_SERVER_" + userID.String()
	envKey = "OAUTH_TOKEN_TEST_SERVER_" + userID.String()[:8] + "_" + userID.String()[9:13] + "_" + userID.String()[14:18] + "_" + userID.String()[19:23] + "_" + userID.String()[24:]
	envKey = "OAUTH_TOKEN_TEST_SERVER_" + userID.String()
	envKey = envKey[:len(envKey)-4] + userID.String()[len(userID.String())-4:]

	// Clean format for environment variable
	userIDStr := userID.String()
	userIDStr = userIDStr[:8] + userIDStr[9:13] + userIDStr[14:18] + userIDStr[19:23] + userIDStr[24:]
	envKey = "OAUTH_TOKEN_TEST_SERVER_" + userIDStr

	t.Cleanup(func() {
		os.Unsetenv(envKey)
	})

	t.Run("store token in environment (should fail)", func(t *testing.T) {
		err := storage.storeTokenInEnvironment(ctx, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("get token from environment", func(t *testing.T) {
		// Set the environment variable manually
		err := os.Setenv(envKey, string(tokenJSON))
		require.NoError(t, err)

		retrievedToken, err := storage.getTokenFromEnvironment(ctx, serverName, userID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedToken)
		assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)
		assert.Equal(t, StorageTierEnvironment, retrievedToken.StorageTier)
	})

	t.Run("list tokens from environment", func(t *testing.T) {
		tokens, err := storage.listTokensFromEnvironment(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, token.AccessToken, tokens[0].AccessToken)
	})

	t.Run("delete token from environment (should fail)", func(t *testing.T) {
		err := storage.deleteTokenFromEnvironment(ctx, serverName, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("get non-existent token from environment", func(t *testing.T) {
		_, err := storage.getTokenFromEnvironment(ctx, "non-existent", userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestHierarchicalTokenStorage_Encryption(t *testing.T) {
	encSvc := &MockEncryptionService{}

	config := &InterceptorConfig{
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
		EncryptTokens: true,
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		encryptionSvc:     encSvc,
		dockerDesktopPath: t.TempDir(),
	}

	token := &TokenData{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		IDToken:      "test-id-token",
	}

	t.Run("encrypt token", func(t *testing.T) {
		encryptedToken, err := storage.encryptToken(token)
		assert.NoError(t, err)
		assert.NotNil(t, encryptedToken)
		assert.Equal(t, "encrypted:test-access-token", encryptedToken.AccessToken)
		assert.Equal(t, "encrypted:test-refresh-token", encryptedToken.RefreshToken)
		assert.Equal(t, "encrypted:test-id-token", encryptedToken.IDToken)
	})

	t.Run("decrypt token", func(t *testing.T) {
		encryptedToken := &TokenData{
			AccessToken:  "encrypted:test-access-token",
			RefreshToken: "encrypted:test-refresh-token",
			IDToken:      "encrypted:test-id-token",
		}

		decryptedToken, err := storage.decryptToken(encryptedToken)
		assert.NoError(t, err)
		assert.NotNil(t, decryptedToken)
		assert.Equal(t, "test-access-token", decryptedToken.AccessToken)
		assert.Equal(t, "test-refresh-token", decryptedToken.RefreshToken)
		assert.Equal(t, "test-id-token", decryptedToken.IDToken)
	})

	t.Run("encryption with nil service", func(t *testing.T) {
		storage.encryptionSvc = nil

		encryptedToken, err := storage.encryptToken(token)
		assert.NoError(t, err)
		assert.Equal(t, token, encryptedToken) // Should return unchanged
	})
}

func TestHierarchicalTokenStorage_TierFallback(t *testing.T) {
	tempDir := t.TempDir()

	config := &InterceptorConfig{
		StorageTiers: []StorageTier{
			StorageTierKeyVault,
			StorageTierDockerDesktop,
			StorageTierEnvironment,
		},
		EncryptTokens: false,
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
		// keyVaultClient is nil - simulating Key Vault unavailable
	}

	ctx := context.Background()
	userID := uuid.New()
	token := &TokenData{
		ServerName:   "test-server",
		UserID:       userID,
		ProviderType: ProviderTypeMicrosoft,
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
		IssuedAt:     time.Now(),
	}

	t.Run("store token with tier fallback", func(t *testing.T) {
		// Try to store at Key Vault tier, should fall back to Docker Desktop
		err := storage.StoreToken(ctx, token, StorageTierKeyVault)
		assert.NoError(t, err)
		assert.Equal(t, StorageTierDockerDesktop, token.StorageTier)
	})

	t.Run("get token from highest available tier", func(t *testing.T) {
		retrievedToken, err := storage.GetToken(ctx, "test-server", userID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedToken)
		assert.Equal(t, token.AccessToken, retrievedToken.AccessToken)
	})
}

func TestHierarchicalTokenStorage_CleanupExpiredTokens(t *testing.T) {
	tempDir := t.TempDir()

	config := &InterceptorConfig{
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
		EncryptTokens: false,
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()

	t.Run("cleanup expired tokens", func(t *testing.T) {
		count, err := storage.CleanupExpiredTokens(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

func TestHierarchicalTokenStorage_Health(t *testing.T) {
	tempDir := t.TempDir()

	config := &InterceptorConfig{
		StorageTiers: []StorageTier{StorageTierDockerDesktop, StorageTierEnvironment},
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()

	t.Run("health check passes", func(t *testing.T) {
		err := storage.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("health check fails with no available tiers", func(t *testing.T) {
		badStorage := &HierarchicalTokenStorage{
			config: &InterceptorConfig{
				StorageTiers: []StorageTier{StorageTierKeyVault},
			},
			// No Key Vault client
		}

		err := badStorage.Health(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no storage tiers are healthy")
	})
}

func TestGetDockerDesktopConfigPath(t *testing.T) {
	path := getDockerDesktopConfigPath()
	// Path might be empty if Docker Desktop is not installed
	// This is acceptable behavior
	t.Logf("Docker Desktop config path: %s", path)
}

// Benchmark tests
func BenchmarkStoreTokenDockerDesktop(b *testing.B) {
	tempDir := b.TempDir()
	storage := &HierarchicalTokenStorage{
		config: &InterceptorConfig{
			StorageTiers:  []StorageTier{StorageTierDockerDesktop},
			EncryptTokens: false,
		},
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()
	token := &TokenData{
		ServerName:  "benchmark-server",
		UserID:      uuid.New(),
		AccessToken: "benchmark-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour),
		IssuedAt:    time.Now(),
		StorageTier: StorageTierDockerDesktop,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token.UserID = uuid.New() // Unique token for each iteration
		_ = storage.storeTokenInDockerDesktop(ctx, token)
	}
}

func BenchmarkEncryptToken(b *testing.B) {
	encSvc := &MockEncryptionService{}
	storage := &HierarchicalTokenStorage{
		encryptionSvc: encSvc,
	}

	token := &TokenData{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		IDToken:      "test-id-token",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.encryptToken(token)
	}
}

// Test concurrent access
func TestHierarchicalTokenStorage_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()

	config := &InterceptorConfig{
		StorageTiers:  []StorageTier{StorageTierDockerDesktop},
		EncryptTokens: false,
	}

	storage := &HierarchicalTokenStorage{
		config:            config,
		dockerDesktopPath: tempDir,
	}

	ctx := context.Background()
	userID := uuid.New()

	// Test concurrent store operations
	t.Run("concurrent store operations", func(t *testing.T) {
		const numGoroutines = 10

		done := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				token := &TokenData{
					ServerName:  "test-server-" + string(rune('0'+index)),
					UserID:      userID,
					AccessToken: "test-token-" + string(rune('0'+index)),
					TokenType:   "Bearer",
					ExpiresAt:   time.Now().Add(time.Hour),
					IssuedAt:    time.Now(),
					StorageTier: StorageTierDockerDesktop,
				}

				err := storage.StoreToken(ctx, token, StorageTierDockerDesktop)
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			err := <-done
			assert.NoError(t, err)
		}
	})
}
