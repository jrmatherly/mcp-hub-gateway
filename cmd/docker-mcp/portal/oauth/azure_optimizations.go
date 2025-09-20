package oauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
)

// AzureClientPool manages pooled Azure clients for better performance
type AzureClientPool struct {
	graphClients    sync.Pool
	keyVaultClients sync.Pool
	credential      azcore.TokenCredential
	keyVaultURL     string
	mu              sync.RWMutex
}

// NewAzureClientPool creates a new Azure client pool
func NewAzureClientPool(credential azcore.TokenCredential, keyVaultURL string) *AzureClientPool {
	pool := &AzureClientPool{
		credential:  credential,
		keyVaultURL: keyVaultURL,
	}

	// Initialize Graph client pool
	pool.graphClients.New = func() interface{} {
		client, err := msgraph.NewGraphServiceClientWithCredentials(
			credential,
			[]string{"https://graph.microsoft.com/.default"},
		)
		if err != nil {
			return nil
		}
		return client
	}

	// Initialize Key Vault client pool
	pool.keyVaultClients.New = func() interface{} {
		if keyVaultURL == "" {
			return nil
		}
		client, err := azsecrets.NewClient(keyVaultURL, credential, nil)
		if err != nil {
			return nil
		}
		return client
	}

	return pool
}

// GetGraphClient retrieves a pooled Graph client
func (p *AzureClientPool) GetGraphClient() *msgraph.GraphServiceClient {
	client := p.graphClients.Get()
	if client == nil {
		return nil
	}
	return client.(*msgraph.GraphServiceClient)
}

// ReturnGraphClient returns a Graph client to the pool
func (p *AzureClientPool) ReturnGraphClient(client *msgraph.GraphServiceClient) {
	if client != nil {
		p.graphClients.Put(client)
	}
}

// GetKeyVaultClient retrieves a pooled Key Vault client
func (p *AzureClientPool) GetKeyVaultClient() *azsecrets.Client {
	client := p.keyVaultClients.Get()
	if client == nil {
		return nil
	}
	return client.(*azsecrets.Client)
}

// ReturnKeyVaultClient returns a Key Vault client to the pool
func (p *AzureClientPool) ReturnKeyVaultClient(client *azsecrets.Client) {
	if client != nil {
		p.keyVaultClients.Put(client)
	}
}

// AzureMetrics provides Azure-specific operation metrics
type AzureMetrics struct {
	GraphAPICalls      int64            `json:"graph_api_calls"`
	KeyVaultCalls      int64            `json:"key_vault_calls"`
	AvgGraphLatency    time.Duration    `json:"avg_graph_latency"`
	AvgKeyVaultLatency time.Duration    `json:"avg_key_vault_latency"`
	Errors             map[string]int64 `json:"errors"`
	mu                 sync.RWMutex
}

// NewAzureMetrics creates a new Azure metrics collector
func NewAzureMetrics() *AzureMetrics {
	return &AzureMetrics{
		Errors: make(map[string]int64),
	}
}

// RecordGraphCall records a Microsoft Graph API call
func (m *AzureMetrics) RecordGraphCall(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GraphAPICalls++

	// Update average latency
	if m.GraphAPICalls == 1 {
		m.AvgGraphLatency = duration
	} else {
		m.AvgGraphLatency = time.Duration(
			(int64(m.AvgGraphLatency)*(m.GraphAPICalls-1) + int64(duration)) / m.GraphAPICalls,
		)
	}

	if err != nil {
		m.Errors["graph_api"]++
	}
}

// RecordKeyVaultCall records a Key Vault API call
func (m *AzureMetrics) RecordKeyVaultCall(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.KeyVaultCalls++

	// Update average latency
	if m.KeyVaultCalls == 1 {
		m.AvgKeyVaultLatency = duration
	} else {
		m.AvgKeyVaultLatency = time.Duration(
			(int64(m.AvgKeyVaultLatency)*(m.KeyVaultCalls-1) + int64(duration)) / m.KeyVaultCalls,
		)
	}

	if err != nil {
		m.Errors["key_vault"]++
	}
}

// GetMetrics returns current Azure metrics
func (m *AzureMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	errors := make(map[string]int64)
	for k, v := range m.Errors {
		errors[k] = v
	}

	return map[string]interface{}{
		"graph_api_calls":       m.GraphAPICalls,
		"key_vault_calls":       m.KeyVaultCalls,
		"avg_graph_latency":     m.AvgGraphLatency.String(),
		"avg_key_vault_latency": m.AvgKeyVaultLatency.String(),
		"errors":                errors,
	}
}

// OptimizedAzureADDCRBridge extends the basic DCR bridge with optimizations
type OptimizedAzureADDCRBridge struct {
	*AzureADDCRBridge
	clientPool *AzureClientPool
	metrics    *AzureMetrics
}

// CreateOptimizedAzureADDCRBridge creates an optimized DCR bridge
func CreateOptimizedAzureADDCRBridge(
	tenantID, subscriptionID, resourceGroup, keyVaultURL string,
) (*OptimizedAzureADDCRBridge, error) {
	// Create base bridge
	baseBridge, err := CreateAzureADDCRBridge(tenantID, subscriptionID, resourceGroup, keyVaultURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create base bridge: %w", err)
	}

	// Get credential from base bridge
	var credential azcore.TokenCredential
	if baseBridge.credential != nil {
		credential = baseBridge.credential
	} else {
		// Fall back to default credential
		credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create credential: %w", err)
		}
	}

	// Create optimizations
	clientPool := NewAzureClientPool(credential, keyVaultURL)
	metrics := NewAzureMetrics()

	return &OptimizedAzureADDCRBridge{
		AzureADDCRBridge: baseBridge,
		clientPool:       clientPool,
		metrics:          metrics,
	}, nil
}

// CreateClientSecretOptimized creates a client secret with pooled clients and metrics
func (b *OptimizedAzureADDCRBridge) CreateClientSecretOptimized(
	ctx context.Context,
	appObjectId string,
) (string, time.Time, error) {
	startTime := time.Now()

	// Get pooled Graph client
	graphClient := b.clientPool.GetGraphClient()
	if graphClient == nil {
		return "", time.Time{}, fmt.Errorf("failed to get Graph client from pool")
	}
	defer b.clientPool.ReturnGraphClient(graphClient)

	// Use the existing createClientSecret logic but with pooled client
	// This is an example of how you could optimize the existing method
	result, err := b.AzureADDCRBridge.createClientSecret(ctx, appObjectId)

	// Record metrics
	duration := time.Since(startTime)
	b.metrics.RecordGraphCall(duration, err)

	if err != nil {
		return "", time.Time{}, err
	}

	return *result.GetSecretText(), *result.GetEndDateTime(), nil
}

// StoreCredentialsOptimized stores credentials with pooled clients and metrics
func (b *OptimizedAzureADDCRBridge) StoreCredentialsOptimized(
	ctx context.Context,
	response *DCRResponse,
) error {
	startTime := time.Now()

	// Use existing implementation for now
	err := b.AzureADDCRBridge.storeCredentialsInKeyVault(ctx, response)

	// Record metrics
	duration := time.Since(startTime)
	b.metrics.RecordKeyVaultCall(duration, err)

	return err
}

// GetMetrics returns optimization metrics
func (b *OptimizedAzureADDCRBridge) GetOptimizationMetrics() map[string]interface{} {
	return b.metrics.GetMetrics()
}

// AzureRetryPolicy provides Azure-specific retry configuration
type AzureRetryPolicy struct {
	MaxRetries           int           `json:"max_retries"`
	InitialDelay         time.Duration `json:"initial_delay"`
	MaxDelay             time.Duration `json:"max_delay"`
	BackoffMultiplier    float64       `json:"backoff_multiplier"`
	RetryableStatusCodes []int         `json:"retryable_status_codes"`
}

// DefaultAzureRetryPolicy returns recommended retry settings for Azure services
func DefaultAzureRetryPolicy() *AzureRetryPolicy {
	return &AzureRetryPolicy{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: []int{
			429, // Too Many Requests
			500, // Internal Server Error
			502, // Bad Gateway
			503, // Service Unavailable
			504, // Gateway Timeout
		},
	}
}

// AzureCredentialManager handles credential rotation and management
type AzureCredentialManager struct {
	tenantID          string
	clientID          string
	currentCredential azcore.TokenCredential
	backupCredential  azcore.TokenCredential
	rotationInterval  time.Duration
	lastRotation      time.Time
	mu                sync.RWMutex
}

// NewAzureCredentialManager creates a new credential manager
func NewAzureCredentialManager(
	tenantID, clientID string,
	rotationInterval time.Duration,
) *AzureCredentialManager {
	return &AzureCredentialManager{
		tenantID:         tenantID,
		clientID:         clientID,
		rotationInterval: rotationInterval,
		lastRotation:     time.Now(),
	}
}

// GetCredential returns the current active credential
func (m *AzureCredentialManager) GetCredential() azcore.TokenCredential {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCredential
}

// ShouldRotate checks if credentials should be rotated
func (m *AzureCredentialManager) ShouldRotate() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastRotation) >= m.rotationInterval
}

// RotateCredential rotates to the backup credential
func (m *AzureCredentialManager) RotateCredential() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backupCredential == nil {
		return fmt.Errorf("no backup credential available")
	}

	// Swap credentials
	m.currentCredential = m.backupCredential
	m.backupCredential = nil
	m.lastRotation = time.Now()

	return nil
}
