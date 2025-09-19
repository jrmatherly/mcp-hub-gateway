package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/google/uuid"
)

// HierarchicalTokenStorage implements the TokenStorage interface with tier-based fallback
type HierarchicalTokenStorage struct {
	config            *InterceptorConfig
	encryptionSvc     EncryptionService
	keyVaultClient    *azsecrets.Client
	dockerDesktopPath string
	mu                sync.RWMutex
}

// CreateHierarchicalTokenStorage creates a new hierarchical token storage instance
func CreateHierarchicalTokenStorage(
	config *InterceptorConfig,
	encryptionSvc EncryptionService,
	keyVaultURL string,
) (*HierarchicalTokenStorage, error) {
	storage := &HierarchicalTokenStorage{
		config:        config,
		encryptionSvc: encryptionSvc,
	}

	// Initialize Key Vault client if available
	if keyVaultURL != "" {
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err == nil {
			storage.keyVaultClient, err = azsecrets.NewClient(keyVaultURL, cred, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
			}
		}
	}

	// Initialize Docker Desktop path
	storage.dockerDesktopPath = getDockerDesktopConfigPath()

	return storage, nil
}

// StoreToken stores a token at the specified tier with fallback
func (h *HierarchicalTokenStorage) StoreToken(
	ctx context.Context,
	token *TokenData,
	tier StorageTier,
) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Encrypt token if required
	if h.config.EncryptTokens {
		encryptedToken, err := h.encryptToken(token)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %w", err)
		}
		token = encryptedToken
	}

	// Try to store at requested tier, fall back if necessary
	for currentTier := tier; currentTier <= StorageTierEnvironment; currentTier++ {
		if !h.isStorageTierAvailable(currentTier) {
			continue
		}

		if err := h.storeTokenAtTier(ctx, token, currentTier); err == nil {
			token.StorageTier = currentTier
			return nil
		}
	}

	return fmt.Errorf("failed to store token at any available tier")
}

// GetToken retrieves a token from the highest priority available tier
func (h *HierarchicalTokenStorage) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Try each tier in priority order
	for _, tier := range h.config.StorageTiers {
		if !h.isStorageTierAvailable(tier) {
			continue
		}

		token, err := h.getTokenFromTier(ctx, serverName, userID, tier)
		if err == nil && token != nil {
			// Decrypt if necessary
			if h.config.EncryptTokens {
				return h.decryptToken(token)
			}
			return token, nil
		}
	}

	return nil, fmt.Errorf("token not found for server %s and user %s", serverName, userID.String())
}

// RefreshToken refreshes a token and updates storage
func (h *HierarchicalTokenStorage) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	// Get current token
	currentToken, err := h.GetToken(ctx, serverName, userID)
	if err != nil {
		return nil, fmt.Errorf("current token not found: %w", err)
	}

	// This would typically call the OAuth provider to refresh
	// For now, we'll update the metadata and re-store
	refreshedToken := &TokenData{
		ServerName:   currentToken.ServerName,
		UserID:       currentToken.UserID,
		TenantID:     currentToken.TenantID,
		ProviderType: currentToken.ProviderType,
		AccessToken:  currentToken.AccessToken,  // Would be new token from provider
		RefreshToken: currentToken.RefreshToken, // Would be new refresh token
		IDToken:      currentToken.IDToken,
		TokenType:    currentToken.TokenType,
		ExpiresAt:    time.Now().Add(1 * time.Hour), // Would come from provider
		RefreshAt:    time.Now().Add(45 * time.Minute),
		IssuedAt:     time.Now(),
		Scopes:       currentToken.Scopes,
		StorageTier:  currentToken.StorageTier,
		LastUsed:     time.Now(),
		UsageCount:   currentToken.UsageCount + 1,
	}

	// Store the refreshed token
	if err := h.StoreToken(ctx, refreshedToken, currentToken.StorageTier); err != nil {
		return nil, fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return refreshedToken, nil
}

// DeleteToken removes a token from all storage tiers
func (h *HierarchicalTokenStorage) DeleteToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var lastErr error
	deleted := false

	// Delete from all tiers
	for _, tier := range h.config.StorageTiers {
		if !h.isStorageTierAvailable(tier) {
			continue
		}

		if err := h.deleteTokenFromTier(ctx, serverName, userID, tier); err == nil {
			deleted = true
		} else {
			lastErr = err
		}
	}

	if !deleted && lastErr != nil {
		return fmt.Errorf("failed to delete token from any tier: %w", lastErr)
	}

	return nil
}

// GetStorageTier returns the current storage tier for a token
func (h *HierarchicalTokenStorage) GetStorageTier(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (StorageTier, error) {
	token, err := h.GetToken(ctx, serverName, userID)
	if err != nil {
		return 0, err
	}
	return token.StorageTier, nil
}

// ListTokens returns all tokens for a user across all tiers
func (h *HierarchicalTokenStorage) ListTokens(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	tokenMap := make(map[string]*TokenData) // serverName -> token

	// Collect tokens from all tiers (higher priority overwrites lower)
	for i := len(h.config.StorageTiers) - 1; i >= 0; i-- {
		tier := h.config.StorageTiers[i]
		if !h.isStorageTierAvailable(tier) {
			continue
		}

		tokens, err := h.listTokensFromTier(ctx, userID, tier)
		if err != nil {
			continue // Skip this tier on error
		}

		for _, token := range tokens {
			tokenMap[token.ServerName] = token
		}
	}

	// Convert map to slice
	result := make([]*TokenData, 0, len(tokenMap))
	for _, token := range tokenMap {
		if h.config.EncryptTokens {
			if decrypted, err := h.decryptToken(token); err == nil {
				result = append(result, decrypted)
			}
		} else {
			result = append(result, token)
		}
	}

	return result, nil
}

// CleanupExpiredTokens removes expired tokens from all tiers
func (h *HierarchicalTokenStorage) CleanupExpiredTokens(ctx context.Context) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	totalCleaned := 0
	now := time.Now()

	for _, tier := range h.config.StorageTiers {
		if !h.isStorageTierAvailable(tier) {
			continue
		}

		cleaned, err := h.cleanupExpiredTokensFromTier(ctx, tier, now)
		if err == nil {
			totalCleaned += cleaned
		}
	}

	return totalCleaned, nil
}

// MigrateTokens moves tokens from one tier to another
func (h *HierarchicalTokenStorage) MigrateTokens(
	ctx context.Context,
	fromTier, toTier StorageTier,
) (int, error) {
	if !h.isStorageTierAvailable(fromTier) || !h.isStorageTierAvailable(toTier) {
		return 0, fmt.Errorf("source or destination tier not available")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// This would require listing all users - for now return 0
	// In a real implementation, this would:
	// 1. List all tokens in fromTier
	// 2. Store each in toTier
	// 3. Delete from fromTier
	// 4. Return count of migrated tokens

	return 0, nil
}

// Health checks the health of all storage tiers
func (h *HierarchicalTokenStorage) Health(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	healthyTiers := 0
	for _, tier := range h.config.StorageTiers {
		if h.isStorageTierAvailable(tier) {
			if err := h.healthCheckTier(ctx, tier); err == nil {
				healthyTiers++
			}
		}
	}

	if healthyTiers == 0 {
		return fmt.Errorf("no storage tiers are healthy")
	}

	return nil
}

// Private helper methods

func (h *HierarchicalTokenStorage) isStorageTierAvailable(tier StorageTier) bool {
	switch tier {
	case StorageTierKeyVault:
		return h.keyVaultClient != nil
	case StorageTierDockerDesktop:
		return h.dockerDesktopPath != ""
	case StorageTierEnvironment:
		return true // Always available
	default:
		return false
	}
}

func (h *HierarchicalTokenStorage) storeTokenAtTier(
	ctx context.Context,
	token *TokenData,
	tier StorageTier,
) error {
	switch tier {
	case StorageTierKeyVault:
		return h.storeTokenInKeyVault(ctx, token)
	case StorageTierDockerDesktop:
		return h.storeTokenInDockerDesktop(ctx, token)
	case StorageTierEnvironment:
		return h.storeTokenInEnvironment(ctx, token)
	default:
		return fmt.Errorf("unsupported storage tier: %d", tier)
	}
}

func (h *HierarchicalTokenStorage) getTokenFromTier(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	tier StorageTier,
) (*TokenData, error) {
	switch tier {
	case StorageTierKeyVault:
		return h.getTokenFromKeyVault(ctx, serverName, userID)
	case StorageTierDockerDesktop:
		return h.getTokenFromDockerDesktop(ctx, serverName, userID)
	case StorageTierEnvironment:
		return h.getTokenFromEnvironment(ctx, serverName, userID)
	default:
		return nil, fmt.Errorf("unsupported storage tier: %d", tier)
	}
}

func (h *HierarchicalTokenStorage) deleteTokenFromTier(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	tier StorageTier,
) error {
	switch tier {
	case StorageTierKeyVault:
		return h.deleteTokenFromKeyVault(ctx, serverName, userID)
	case StorageTierDockerDesktop:
		return h.deleteTokenFromDockerDesktop(ctx, serverName, userID)
	case StorageTierEnvironment:
		return h.deleteTokenFromEnvironment(ctx, serverName, userID)
	default:
		return fmt.Errorf("unsupported storage tier: %d", tier)
	}
}

func (h *HierarchicalTokenStorage) listTokensFromTier(
	ctx context.Context,
	userID uuid.UUID,
	tier StorageTier,
) ([]*TokenData, error) {
	switch tier {
	case StorageTierKeyVault:
		return h.listTokensFromKeyVault(ctx, userID)
	case StorageTierDockerDesktop:
		return h.listTokensFromDockerDesktop(ctx, userID)
	case StorageTierEnvironment:
		return h.listTokensFromEnvironment(ctx, userID)
	default:
		return nil, fmt.Errorf("unsupported storage tier: %d", tier)
	}
}

func (h *HierarchicalTokenStorage) cleanupExpiredTokensFromTier(
	ctx context.Context,
	tier StorageTier,
	now time.Time,
) (int, error) {
	// Implementation would depend on tier capabilities
	return 0, nil
}

func (h *HierarchicalTokenStorage) healthCheckTier(ctx context.Context, tier StorageTier) error {
	switch tier {
	case StorageTierKeyVault:
		if h.keyVaultClient == nil {
			return fmt.Errorf("Key Vault client not initialized")
		}
		// Could ping Key Vault here
		return nil
	case StorageTierDockerDesktop:
		if h.dockerDesktopPath == "" {
			return fmt.Errorf("Docker Desktop path not available")
		}
		// Check if path is accessible
		if _, err := os.Stat(h.dockerDesktopPath); err != nil {
			return fmt.Errorf("Docker Desktop path not accessible: %w", err)
		}
		return nil
	case StorageTierEnvironment:
		return nil // Always healthy
	default:
		return fmt.Errorf("unknown storage tier: %d", tier)
	}
}

// Key Vault implementations
func (h *HierarchicalTokenStorage) storeTokenInKeyVault(
	ctx context.Context,
	token *TokenData,
) error {
	if h.keyVaultClient == nil {
		return fmt.Errorf("Key Vault client not available")
	}

	secretName := fmt.Sprintf("oauth-token-%s-%s", token.ServerName, token.UserID.String())
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	secretValue := string(tokenJSON)
	_, err = h.keyVaultClient.SetSecret(ctx, secretName, azsecrets.SetSecretParameters{
		Value: &secretValue,
	}, nil)

	return err
}

func (h *HierarchicalTokenStorage) getTokenFromKeyVault(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	if h.keyVaultClient == nil {
		return nil, fmt.Errorf("Key Vault client not available")
	}

	secretName := fmt.Sprintf("oauth-token-%s-%s", serverName, userID.String())
	resp, err := h.keyVaultClient.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return nil, err
	}

	var token TokenData
	if err := json.Unmarshal([]byte(*resp.Value), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (h *HierarchicalTokenStorage) deleteTokenFromKeyVault(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	if h.keyVaultClient == nil {
		return fmt.Errorf("Key Vault client not available")
	}

	secretName := fmt.Sprintf("oauth-token-%s-%s", serverName, userID.String())
	_, err := h.keyVaultClient.DeleteSecret(ctx, secretName, nil)
	return err
}

func (h *HierarchicalTokenStorage) listTokensFromKeyVault(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	if h.keyVaultClient == nil {
		return nil, fmt.Errorf("Key Vault client not available")
	}

	// This would require listing secrets and filtering by user ID
	// Azure Key Vault doesn't have great filtering capabilities
	// In practice, might need to maintain an index
	return []*TokenData{}, nil
}

// Docker Desktop implementations
func (h *HierarchicalTokenStorage) storeTokenInDockerDesktop(
	ctx context.Context,
	token *TokenData,
) error {
	if h.dockerDesktopPath == "" {
		return fmt.Errorf("Docker Desktop path not available")
	}

	tokenDir := filepath.Join(h.dockerDesktopPath, "oauth-tokens")
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.json", token.ServerName, token.UserID.String())
	filepath := filepath.Join(tokenDir, filename)

	tokenJSON, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	return os.WriteFile(filepath, tokenJSON, 0o600)
}

func (h *HierarchicalTokenStorage) getTokenFromDockerDesktop(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	if h.dockerDesktopPath == "" {
		return nil, fmt.Errorf("Docker Desktop path not available")
	}

	filename := fmt.Sprintf("%s-%s.json", serverName, userID.String())
	filepath := filepath.Join(h.dockerDesktopPath, "oauth-tokens", filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (h *HierarchicalTokenStorage) deleteTokenFromDockerDesktop(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	if h.dockerDesktopPath == "" {
		return fmt.Errorf("Docker Desktop path not available")
	}

	filename := fmt.Sprintf("%s-%s.json", serverName, userID.String())
	filepath := filepath.Join(h.dockerDesktopPath, "oauth-tokens", filename)

	return os.Remove(filepath)
}

func (h *HierarchicalTokenStorage) listTokensFromDockerDesktop(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	if h.dockerDesktopPath == "" {
		return nil, fmt.Errorf("Docker Desktop path not available")
	}

	tokenDir := filepath.Join(h.dockerDesktopPath, "oauth-tokens")
	entries, err := os.ReadDir(tokenDir)
	if err != nil {
		return nil, err
	}

	var tokens []*TokenData
	userIDStr := userID.String()

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), fmt.Sprintf("-%s.json", userIDStr)) {
			continue
		}

		filepath := filepath.Join(tokenDir, entry.Name())
		data, err := os.ReadFile(filepath)
		if err != nil {
			continue
		}

		var token TokenData
		if err := json.Unmarshal(data, &token); err != nil {
			continue
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}

// Environment variable implementations
func (h *HierarchicalTokenStorage) storeTokenInEnvironment(
	ctx context.Context,
	token *TokenData,
) error {
	// Environment variables are read-only in this context
	// This would typically be used only for reading pre-configured tokens
	return fmt.Errorf("storing tokens in environment variables not supported")
}

func (h *HierarchicalTokenStorage) getTokenFromEnvironment(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	// Look for environment variables in the format:
	// OAUTH_TOKEN_{SERVER_NAME}_{USER_ID}
	envKey := fmt.Sprintf("OAUTH_TOKEN_%s_%s",
		strings.ToUpper(strings.ReplaceAll(serverName, "-", "_")),
		strings.ReplaceAll(userID.String(), "-", "_"))

	tokenJSON := os.Getenv(envKey)
	if tokenJSON == "" {
		return nil, fmt.Errorf("environment variable %s not found", envKey)
	}

	var token TokenData
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token from environment: %w", err)
	}

	token.StorageTier = StorageTierEnvironment
	return &token, nil
}

func (h *HierarchicalTokenStorage) deleteTokenFromEnvironment(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	// Can't delete environment variables at runtime
	return fmt.Errorf("deleting tokens from environment variables not supported")
}

func (h *HierarchicalTokenStorage) listTokensFromEnvironment(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	// Scan environment variables for OAuth tokens
	var tokens []*TokenData
	userIDStr := strings.ReplaceAll(userID.String(), "-", "_")

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "OAUTH_TOKEN_") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if this token belongs to the user
		if !strings.HasSuffix(key, "_"+userIDStr) {
			continue
		}

		var token TokenData
		if err := json.Unmarshal([]byte(value), &token); err != nil {
			continue
		}

		token.StorageTier = StorageTierEnvironment
		tokens = append(tokens, &token)
	}

	return tokens, nil
}

// Encryption helpers
func (h *HierarchicalTokenStorage) encryptToken(token *TokenData) (*TokenData, error) {
	if h.encryptionSvc == nil {
		return token, nil
	}

	encryptedToken := *token

	// Encrypt sensitive fields
	if token.AccessToken != "" {
		encrypted, err := h.encryptionSvc.Encrypt([]byte(token.AccessToken))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt access token: %w", err)
		}
		encryptedToken.AccessToken = string(encrypted)
	}

	if token.RefreshToken != "" {
		encrypted, err := h.encryptionSvc.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
		encryptedToken.RefreshToken = string(encrypted)
	}

	if token.IDToken != "" {
		encrypted, err := h.encryptionSvc.Encrypt([]byte(token.IDToken))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt ID token: %w", err)
		}
		encryptedToken.IDToken = string(encrypted)
	}

	return &encryptedToken, nil
}

func (h *HierarchicalTokenStorage) decryptToken(token *TokenData) (*TokenData, error) {
	if h.encryptionSvc == nil {
		return token, nil
	}

	decryptedToken := *token

	// Decrypt sensitive fields
	if token.AccessToken != "" {
		decrypted, err := h.encryptionSvc.Decrypt([]byte(token.AccessToken))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt access token: %w", err)
		}
		decryptedToken.AccessToken = string(decrypted)
	}

	if token.RefreshToken != "" {
		decrypted, err := h.encryptionSvc.Decrypt([]byte(token.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
		}
		decryptedToken.RefreshToken = string(decrypted)
	}

	if token.IDToken != "" {
		decrypted, err := h.encryptionSvc.Decrypt([]byte(token.IDToken))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt ID token: %w", err)
		}
		decryptedToken.IDToken = string(decrypted)
	}

	return &decryptedToken, nil
}

// Utility functions
func getDockerDesktopConfigPath() string {
	// Try to detect Docker Desktop configuration path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check common Docker Desktop paths
	paths := []string{
		filepath.Join(homeDir, ".docker"),
		filepath.Join(homeDir, "Library", "Group Containers", "group.com.docker", "settings"),
		filepath.Join(homeDir, "AppData", "Roaming", "Docker"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// EncryptionService interface for token encryption
type EncryptionService interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}
