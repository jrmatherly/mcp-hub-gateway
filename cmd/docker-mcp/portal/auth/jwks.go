package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSProvider manages JSON Web Key Sets for token validation
type JWKSProvider struct {
	jwksURL    string
	httpClient *http.Client
	cache      *jwksCache
	mu         sync.RWMutex
}

// jwksCache stores the cached JWKS with expiration
type jwksCache struct {
	keySet    *jwt.Keyfunc
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

// CreateJWKSProvider creates a new JWKS provider
func CreateJWKSProvider(jwksURL string, httpClient *http.Client) *JWKSProvider {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &JWKSProvider{
		jwksURL:    jwksURL,
		httpClient: httpClient,
		cache:      &jwksCache{},
	}
}

// GetKeySet retrieves the JWKS, using cache if available
func (p *JWKSProvider) GetKeySet(ctx context.Context) (*JWKS, error) {
	p.mu.RLock()
	if p.cache.keys != nil && p.cache.expiresAt.After(time.Now()) {
		p.mu.RUnlock()
		return nil, nil // Keys are cached
	}
	p.mu.RUnlock()

	// Need to refresh the cache
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if p.cache.keys != nil && p.cache.expiresAt.After(time.Now()) {
		return nil, nil // Keys are cached
	}

	// Fetch new JWKS
	jwks, err := p.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}

	// Parse keys
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty == "RSA" {
			pubKey, err := p.parseRSAPublicKey(key)
			if err != nil {
				continue // Skip invalid keys
			}
			keys[key.Kid] = pubKey
		}
	}

	// Cache for 1 hour
	p.cache.keys = keys
	p.cache.expiresAt = time.Now().Add(time.Hour)

	return jwks, nil
}

// LookupKeyID finds a key by its key ID
func (p *JWKSProvider) LookupKeyID(kid string) any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if key, exists := p.cache.keys[kid]; exists {
		return key
	}
	return nil
}

// parseRSAPublicKey converts a JWK to an RSA public key
func (p *JWKSProvider) parseRSAPublicKey(key JWK) (*rsa.PublicKey, error) {
	// Decode the modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent to int
	var e int
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	// Create the public key
	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}

	return pubKey, nil
}

// fetchJWKS fetches the JWKS from the provider
func (p *JWKSProvider) fetchJWKS(ctx context.Context) (*JWKS, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS request failed with status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	return &jwks, nil
}

// RefreshCache forces a refresh of the JWKS cache
func (p *JWKSProvider) RefreshCache(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	jwks, err := p.fetchJWKS(ctx)
	if err != nil {
		return err
	}

	// Parse keys
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty == "RSA" {
			pubKey, err := p.parseRSAPublicKey(key)
			if err != nil {
				continue // Skip invalid keys
			}
			keys[key.Kid] = pubKey
		}
	}

	p.cache.keys = keys
	p.cache.expiresAt = time.Now().Add(time.Hour)

	return nil
}
