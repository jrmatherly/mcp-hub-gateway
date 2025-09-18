package catalog

import (
	"context"
	"fmt"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

// EncryptionAdapter adapts the crypto.EncryptionService to the catalog.EncryptionService interface
type EncryptionAdapter struct {
	service crypto.EncryptionService
	key     []byte
}

// NewEncryptionAdapter creates a new encryption adapter
func NewEncryptionAdapter(service crypto.EncryptionService, key []byte) *EncryptionAdapter {
	return &EncryptionAdapter{
		service: service,
		key:     key,
	}
}

// Encrypt implements catalog.EncryptionService.Encrypt
func (a *EncryptionAdapter) Encrypt(ctx context.Context, data []byte) ([]byte, error) {
	if a.service == nil {
		return data, nil // No encryption if service is nil
	}

	encrypted, err := a.service.Encrypt(data, a.key)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Convert EncryptedData to bytes (simplified - in production you'd use proper serialization)
	// For now, just return the ciphertext
	return encrypted.Ciphertext, nil
}

// Decrypt implements catalog.EncryptionService.Decrypt
func (a *EncryptionAdapter) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	if a.service == nil {
		return data, nil // No decryption if service is nil
	}

	// Convert bytes back to EncryptedData (simplified)
	// In production, you'd need proper serialization/deserialization
	encrypted := &crypto.EncryptedData{
		Ciphertext: data,
		// Note: This is simplified - you'd need to store/retrieve IV and salt
	}

	decrypted, err := a.service.Decrypt(encrypted, a.key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return decrypted, nil
}
