package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
)

// Error definitions
var (
	ErrTokenExpired = errors.New("token has expired")
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	audience   []string
	ttl        time.Duration
}

// CreateJWTService creates a new JWT service
func CreateJWTService(cfg *config.SecurityConfig) (*JWTService, error) {
	// Parse the RSA private key
	privateKey, err := parsePrivateKey(cfg.JWTSigningKey)
	if err != nil {
		// If parsing fails, generate a new key pair (development mode)
		privateKey, err = generateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %w", err)
		}
	}

	return &JWTService{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		issuer:     cfg.JWTIssuer,
		audience:   cfg.JWTAudience,
		ttl:        cfg.AccessTokenTTL,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (s *JWTService) GenerateToken(user *User, sessionID string) (string, error) {
	now := time.Now()

	// Parse session ID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid session ID: %w", err)
	}

	// Create claims
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
			Issuer:    s.issuer,
			Audience:  s.audience,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
		UserID:      user.ID,
		Email:       user.Email,
		Name:        user.Name,
		TenantID:    user.TenantID,
		Role:        user.Role,
		Permissions: user.Permissions,
		SessionID:   sessionUUID,
		AzureClaims: user.Claims,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign token
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Validate claims
	if err := s.validateClaims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateClaims validates JWT claims
func (s *JWTService) validateClaims(claims *Claims) error {
	// Validate issuer
	if claims.Issuer != s.issuer {
		return fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Validate audience
	found := false
	for _, aud := range s.audience {
		if claims.VerifyAudience(aud, true) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid audience")
	}

	// Validate expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	// Validate not before
	if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
		return fmt.Errorf("token not yet valid")
	}

	return nil
}

// GenerateRefreshToken generates a refresh token
func (s *JWTService) GenerateRefreshToken(user *User, sessionID string) (string, error) {
	now := time.Now()

	// Parse session ID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid session ID: %w", err)
	}

	// Create claims for refresh token (longer TTL, minimal info)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
			Issuer:    s.issuer,
			Audience:  s.audience,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)), // 7 days
		},
		UserID:    user.ID,
		SessionID: sessionUUID,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign token
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// GetPublicKey returns the public key for JWKS
func (s *JWTService) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}

// parsePrivateKey parses an RSA private key from PEM format
func parsePrivateKey(keyStr string) (*rsa.PrivateKey, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("private key is empty")
	}

	block, _ := pem.Decode([]byte(keyStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Try to parse as PKCS1
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// Try to parse as PKCS8
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("not an RSA private key")
	}

	return nil, fmt.Errorf("failed to parse private key")
}

// generateKeyPair generates a new RSA key pair (for development)
func generateKeyPair() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Claims type is defined in types.go
