package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionService provides AES-256-GCM encryption/decryption operations
type EncryptionService interface {
	Encrypt(plaintext []byte, key []byte) (*EncryptedData, error)
	Decrypt(encrypted *EncryptedData, key []byte) ([]byte, error)
	EncryptString(plaintext string, key []byte) (*EncryptedData, error)
	DecryptString(encrypted *EncryptedData, key []byte) (string, error)
	DeriveKey(password []byte, salt []byte) []byte
	GenerateSalt() ([]byte, error)
	GenerateKey() ([]byte, error)
	EncryptBulk(data [][]byte, key []byte) ([]*EncryptedData, error)
	DecryptBulk(encrypted []*EncryptedData, key []byte) ([][]byte, error)
}

// EncryptedData represents encrypted data with metadata
type EncryptedData struct {
	Ciphertext []byte    `json:"ciphertext"`
	Nonce      []byte    `json:"nonce"`
	Salt       []byte    `json:"salt,omitempty"`
	KeySize    int       `json:"key_size"`
	Algorithm  string    `json:"algorithm"`
	Timestamp  time.Time `json:"timestamp"`
	Version    int       `json:"version"`
}

// Config holds encryption service configuration
type Config struct {
	KeySize      int    // Key size in bytes (32 for AES-256)
	NonceSize    int    // Nonce size in bytes (12 for GCM)
	SaltSize     int    // Salt size in bytes (32 recommended)
	PBKDF2Rounds int    // PBKDF2 iteration count
	Algorithm    string // Algorithm identifier
	Version      int    // Version for future compatibility
	MaxDataSize  int64  // Maximum data size to encrypt (anti-DoS)
	MaxBulkSize  int    // Maximum bulk operation size
}

// DefaultConfig returns a secure default configuration
func DefaultConfig() *Config {
	return &Config{
		KeySize:      32,     // AES-256
		NonceSize:    12,     // GCM standard
		SaltSize:     32,     // 256-bit salt
		PBKDF2Rounds: 100000, // OWASP recommended minimum
		Algorithm:    "AES-256-GCM",
		Version:      1,
		MaxDataSize:  10 * 1024 * 1024, // 10MB max
		MaxBulkSize:  100,              // Max 100 items in bulk operations
	}
}

// AESGCMService implements EncryptionService using AES-256-GCM
type AESGCMService struct {
	config   *Config
	mu       sync.RWMutex
	randPool sync.Pool
}

// NewAESGCMService creates a new AES-GCM encryption service
func NewAESGCMService(config *Config) (*AESGCMService, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	service := &AESGCMService{
		config: config,
		randPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 32) // Reusable buffer for random data
			},
		},
	}

	return service, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (s *AESGCMService) Encrypt(plaintext []byte, key []byte) (*EncryptedData, error) {
	// Validate inputs
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}
	if len(key) != s.config.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", s.config.KeySize, len(key))
	}
	if int64(len(plaintext)) > s.config.MaxDataSize {
		return nil, fmt.Errorf(
			"data too large: %d bytes exceeds limit of %d",
			len(plaintext),
			s.config.MaxDataSize,
		)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce, err := s.generateNonce(gcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		KeySize:    s.config.KeySize,
		Algorithm:  s.config.Algorithm,
		Timestamp:  time.Now().UTC(),
		Version:    s.config.Version,
	}, nil
}

// Decrypt decrypts data using AES-256-GCM
func (s *AESGCMService) Decrypt(encrypted *EncryptedData, key []byte) ([]byte, error) {
	// Validate inputs
	if encrypted == nil {
		return nil, fmt.Errorf("encrypted data cannot be nil")
	}
	if len(key) != s.config.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", s.config.KeySize, len(key))
	}
	if len(encrypted.Ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}
	if len(encrypted.Nonce) != s.config.NonceSize {
		return nil, fmt.Errorf(
			"invalid nonce size: expected %d, got %d",
			s.config.NonceSize,
			len(encrypted.Nonce),
		)
	}

	// Validate compatibility
	if encrypted.Algorithm != s.config.Algorithm {
		return nil, fmt.Errorf("unsupported algorithm: %s", encrypted.Algorithm)
	}
	if encrypted.Version > s.config.Version {
		return nil, fmt.Errorf("unsupported version: %d", encrypted.Version)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt the data
	plaintext, err := gcm.Open(nil, encrypted.Nonce, encrypted.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns encrypted data
func (s *AESGCMService) EncryptString(plaintext string, key []byte) (*EncryptedData, error) {
	return s.Encrypt([]byte(plaintext), key)
}

// DecryptString decrypts data and returns a string
func (s *AESGCMService) DecryptString(encrypted *EncryptedData, key []byte) (string, error) {
	plaintext, err := s.Decrypt(encrypted, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// DeriveKey derives a key from password and salt using PBKDF2
func (s *AESGCMService) DeriveKey(password []byte, salt []byte) []byte {
	if len(salt) != s.config.SaltSize {
		// This is a programming error, but we'll handle it gracefully
		salt = s.padOrTruncate(salt, s.config.SaltSize)
	}

	return pbkdf2.Key(password, salt, s.config.PBKDF2Rounds, s.config.KeySize, sha256.New)
}

// GenerateSalt generates a cryptographically secure random salt
func (s *AESGCMService) GenerateSalt() ([]byte, error) {
	salt := make([]byte, s.config.SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateKey generates a cryptographically secure random key
func (s *AESGCMService) GenerateKey() ([]byte, error) {
	key := make([]byte, s.config.KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// EncryptBulk encrypts multiple data items efficiently
func (s *AESGCMService) EncryptBulk(data [][]byte, key []byte) ([]*EncryptedData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to encrypt")
	}
	if len(data) > s.config.MaxBulkSize {
		return nil, fmt.Errorf(
			"bulk operation too large: %d items exceeds limit of %d",
			len(data),
			s.config.MaxBulkSize,
		)
	}
	if len(key) != s.config.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", s.config.KeySize, len(key))
	}

	// Pre-allocate result slice
	results := make([]*EncryptedData, 0, len(data))

	// Create cipher once for all operations
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt each item
	for i, plaintext := range data {
		if len(plaintext) == 0 {
			return nil, fmt.Errorf("plaintext at index %d cannot be empty", i)
		}
		if int64(len(plaintext)) > s.config.MaxDataSize {
			return nil, fmt.Errorf(
				"data at index %d too large: %d bytes exceeds limit of %d",
				i,
				len(plaintext),
				s.config.MaxDataSize,
			)
		}

		// Generate unique nonce for each item
		nonce, err := s.generateNonce(gcm.NonceSize())
		if err != nil {
			return nil, fmt.Errorf("failed to generate nonce for index %d: %w", i, err)
		}

		// Encrypt
		ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

		results = append(results, &EncryptedData{
			Ciphertext: ciphertext,
			Nonce:      nonce,
			KeySize:    s.config.KeySize,
			Algorithm:  s.config.Algorithm,
			Timestamp:  time.Now().UTC(),
			Version:    s.config.Version,
		})
	}

	return results, nil
}

// DecryptBulk decrypts multiple encrypted data items efficiently
func (s *AESGCMService) DecryptBulk(encrypted []*EncryptedData, key []byte) ([][]byte, error) {
	if len(encrypted) == 0 {
		return nil, fmt.Errorf("no data to decrypt")
	}
	if len(encrypted) > s.config.MaxBulkSize {
		return nil, fmt.Errorf(
			"bulk operation too large: %d items exceeds limit of %d",
			len(encrypted),
			s.config.MaxBulkSize,
		)
	}
	if len(key) != s.config.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", s.config.KeySize, len(key))
	}

	// Pre-allocate result slice
	results := make([][]byte, 0, len(encrypted))

	// Create cipher once for all operations
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt each item
	for i, enc := range encrypted {
		if enc == nil {
			return nil, fmt.Errorf("encrypted data at index %d cannot be nil", i)
		}

		// Validate compatibility
		if enc.Algorithm != s.config.Algorithm {
			return nil, fmt.Errorf("unsupported algorithm at index %d: %s", i, enc.Algorithm)
		}
		if enc.Version > s.config.Version {
			return nil, fmt.Errorf("unsupported version at index %d: %d", i, enc.Version)
		}
		if len(enc.Nonce) != s.config.NonceSize {
			return nil, fmt.Errorf(
				"invalid nonce size at index %d: expected %d, got %d",
				i,
				s.config.NonceSize,
				len(enc.Nonce),
			)
		}

		// Decrypt
		plaintext, err := gcm.Open(nil, enc.Nonce, enc.Ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("decryption failed at index %d: %w", i, err)
		}

		results = append(results, plaintext)
	}

	return results, nil
}

// ToBase64 converts EncryptedData to a base64-encoded string for storage/transmission
func (e *EncryptedData) ToBase64() (string, error) {
	if e == nil {
		return "", fmt.Errorf("encrypted data cannot be nil")
	}

	// Create a compact representation
	// Format: version(1)|keysize(1)|algorithm_len(1)|algorithm|nonce_len(2)|nonce|ciphertext
	var data []byte

	// Version (1 byte)
	data = append(data, byte(e.Version))

	// Key size (1 byte)
	data = append(data, byte(e.KeySize))

	// Algorithm length and algorithm
	algBytes := []byte(e.Algorithm)
	data = append(data, byte(len(algBytes)))
	data = append(data, algBytes...)

	// Nonce length (2 bytes) and nonce
	nonceLen := len(e.Nonce)
	data = append(data, byte(nonceLen>>8), byte(nonceLen))
	data = append(data, e.Nonce...)

	// Ciphertext
	data = append(data, e.Ciphertext...)

	return base64.StdEncoding.EncodeToString(data), nil
}

// FromBase64 creates EncryptedData from a base64-encoded string
func FromBase64(encoded string) (*EncryptedData, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(data) < 5 { // Minimum: version(1) + keysize(1) + alg_len(1) + nonce_len(2)
		return nil, fmt.Errorf("encoded data too short")
	}

	pos := 0

	// Parse version
	version := int(data[pos])
	pos++

	// Parse key size
	keySize := int(data[pos])
	pos++

	// Parse algorithm
	algLen := int(data[pos])
	pos++
	if pos+algLen > len(data) {
		return nil, fmt.Errorf("invalid algorithm length")
	}
	algorithm := string(data[pos : pos+algLen])
	pos += algLen

	// Parse nonce
	if pos+2 > len(data) {
		return nil, fmt.Errorf("invalid nonce length field")
	}
	nonceLen := int(data[pos])<<8 | int(data[pos+1])
	pos += 2
	if pos+nonceLen > len(data) {
		return nil, fmt.Errorf("invalid nonce length")
	}
	nonce := make([]byte, nonceLen)
	copy(nonce, data[pos:pos+nonceLen])
	pos += nonceLen

	// Parse ciphertext
	ciphertext := make([]byte, len(data)-pos)
	copy(ciphertext, data[pos:])

	return &EncryptedData{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		KeySize:    keySize,
		Algorithm:  algorithm,
		Version:    version,
		Timestamp:  time.Now().UTC(), // We don't encode timestamp to save space
	}, nil
}

// generateNonce generates a cryptographically secure random nonce
func (s *AESGCMService) generateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// padOrTruncate pads or truncates data to the specified length
func (s *AESGCMService) padOrTruncate(data []byte, length int) []byte {
	if len(data) == length {
		return data
	}

	result := make([]byte, length)
	if len(data) > length {
		copy(result, data[:length])
	} else {
		copy(result, data)
		// Rest is zero-padded by make()
	}
	return result
}

// validateConfig validates the encryption service configuration
func validateConfig(config *Config) error {
	if config.KeySize != 16 && config.KeySize != 24 && config.KeySize != 32 {
		return fmt.Errorf("invalid key size: %d (must be 16, 24, or 32)", config.KeySize)
	}
	if config.NonceSize != 12 {
		return fmt.Errorf("invalid nonce size: %d (GCM requires 12)", config.NonceSize)
	}
	if config.SaltSize < 16 {
		return fmt.Errorf("salt size too small: %d (minimum 16)", config.SaltSize)
	}
	if config.PBKDF2Rounds < 10000 {
		return fmt.Errorf("PBKDF2 rounds too low: %d (minimum 10000)", config.PBKDF2Rounds)
	}
	if config.MaxDataSize <= 0 {
		return fmt.Errorf("max data size must be positive: %d", config.MaxDataSize)
	}
	if config.MaxBulkSize <= 0 {
		return fmt.Errorf("max bulk size must be positive: %d", config.MaxBulkSize)
	}
	if config.Algorithm == "" {
		return fmt.Errorf("algorithm cannot be empty")
	}
	if config.Version <= 0 {
		return fmt.Errorf("version must be positive: %d", config.Version)
	}

	return nil
}

// Encryption is an alias for EncryptionService to maintain compatibility
type Encryption = EncryptionService

// MemorySecurityService provides additional security features for key management
type MemorySecurityService struct {
	encryptionService EncryptionService
}

// NewMemorySecurityService creates a service that helps with secure memory management
func NewMemorySecurityService(encryptionService EncryptionService) *MemorySecurityService {
	return &MemorySecurityService{
		encryptionService: encryptionService,
	}
}

// SecureWipe overwrites sensitive data in memory with random bytes
func (m *MemorySecurityService) SecureWipe(data []byte) {
	if len(data) == 0 {
		return
	}

	// Overwrite with random data
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		// Fallback: overwrite with zeros
		for i := range data {
			data[i] = 0
		}
	}
}

// SecureKeyDerivation derives keys with automatic secure cleanup
func (m *MemorySecurityService) SecureKeyDerivation(password string, salt []byte) ([]byte, func()) {
	passwordBytes := []byte(password)
	defer m.SecureWipe(passwordBytes)

	key := m.encryptionService.DeriveKey(passwordBytes, salt)

	// Return cleanup function
	cleanup := func() {
		m.SecureWipe(key)
	}

	return key, cleanup
}
