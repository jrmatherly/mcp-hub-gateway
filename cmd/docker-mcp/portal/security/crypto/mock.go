package crypto

import (
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockEncryption is a mock implementation of the EncryptionService interface
type MockEncryption struct {
	mock.Mock
}

// Encrypt encrypts plaintext data
func (m *MockEncryption) Encrypt(plaintext []byte, key []byte) (*EncryptedData, error) {
	args := m.Called(plaintext, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EncryptedData), args.Error(1)
}

// Decrypt decrypts encrypted data
func (m *MockEncryption) Decrypt(encrypted *EncryptedData, key []byte) ([]byte, error) {
	args := m.Called(encrypted, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// EncryptString encrypts a string
func (m *MockEncryption) EncryptString(plaintext string, key []byte) (*EncryptedData, error) {
	args := m.Called(plaintext, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EncryptedData), args.Error(1)
}

// DecryptString decrypts to a string
func (m *MockEncryption) DecryptString(encrypted *EncryptedData, key []byte) (string, error) {
	args := m.Called(encrypted, key)
	return args.String(0), args.Error(1)
}

// DeriveKey derives a key from password and salt
func (m *MockEncryption) DeriveKey(password []byte, salt []byte) []byte {
	args := m.Called(password, salt)
	return args.Get(0).([]byte)
}

// GenerateSalt generates a random salt
func (m *MockEncryption) GenerateSalt() ([]byte, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// GenerateKey generates a random encryption key
func (m *MockEncryption) GenerateKey() ([]byte, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// EncryptBulk encrypts multiple data items
func (m *MockEncryption) EncryptBulk(data [][]byte, key []byte) ([]*EncryptedData, error) {
	args := m.Called(data, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EncryptedData), args.Error(1)
}

// DecryptBulk decrypts multiple encrypted data items
func (m *MockEncryption) DecryptBulk(encrypted []*EncryptedData, key []byte) ([][]byte, error) {
	args := m.Called(encrypted, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]byte), args.Error(1)
}

// NewMockEncryption creates a new mock encryption service
func NewMockEncryption() *MockEncryption {
	return &MockEncryption{}
}

// CreateEncryption creates a mock encryption service for testing
func CreateEncryption(key []byte) (EncryptionService, error) {
	mockService := NewMockEncryption()

	// Set up default behaviors for common test cases
	mockService.On("Encrypt", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).
		Return(
			&EncryptedData{
				Ciphertext: []byte("encrypted-data"),
				Nonce:      []byte("test-nonce"),
				Salt:       []byte("test-salt"),
				KeySize:    32,
				Algorithm:  "AES-256-GCM",
				Timestamp:  time.Now(),
				Version:    1,
			}, nil)

	mockService.On("Decrypt", mock.AnythingOfType("*crypto.EncryptedData"), mock.AnythingOfType("[]uint8")).
		Return(
			[]byte("decrypted-data"), nil)

	mockService.On("EncryptString", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).
		Return(
			&EncryptedData{
				Ciphertext: []byte("encrypted-string"),
				Nonce:      []byte("test-nonce"),
				Salt:       []byte("test-salt"),
				KeySize:    32,
				Algorithm:  "AES-256-GCM",
				Timestamp:  time.Now(),
				Version:    1,
			}, nil)

	mockService.On("DecryptString", mock.AnythingOfType("*crypto.EncryptedData"), mock.AnythingOfType("[]uint8")).
		Return(
			"decrypted-string", nil)

	mockService.On("GenerateSalt").Return(make([]byte, 32), nil)
	mockService.On("GenerateKey").Return(make([]byte, 32), nil)
	mockService.On("DeriveKey", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).
		Return(make([]byte, 32))

	return mockService, nil
}

// Helper function to create a simple encrypted data for testing
func NewTestEncryptedData(data string) *EncryptedData {
	return &EncryptedData{
		Ciphertext: []byte(data),
		Nonce:      []byte("test-nonce-12"),
		Salt:       []byte("test-salt-32-bytes-for-testing"),
		KeySize:    32,
		Algorithm:  "AES-256-GCM",
		Timestamp:  time.Now(),
		Version:    1,
	}
}

// Helper function to create test encrypted JSON data
func NewTestEncryptedJSON(obj interface{}) (*EncryptedData, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	// Generate a pseudo-random nonce for testing
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return &EncryptedData{
		Ciphertext: data, // In real implementation this would be encrypted
		Nonce:      nonce,
		Salt:       []byte("test-salt-32-bytes-for-testing!!"),
		KeySize:    32,
		Algorithm:  "AES-256-GCM",
		Timestamp:  time.Now(),
		Version:    1,
	}, nil
}
