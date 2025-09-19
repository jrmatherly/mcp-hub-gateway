package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockOAuthInterceptor provides a mock implementation for testing
type MockOAuthInterceptor struct {
	tokens        map[string]*TokenData
	serverConfigs map[string]*ServerConfig
	requestCount  int64
	mu            sync.RWMutex
}

// CreateMockOAuthInterceptor creates a new mock OAuth interceptor
func CreateMockOAuthInterceptor() *MockOAuthInterceptor {
	return &MockOAuthInterceptor{
		tokens:        make(map[string]*TokenData),
		serverConfigs: make(map[string]*ServerConfig),
	}
}

// InterceptRequest mocks request interception
func (m *MockOAuthInterceptor) InterceptRequest(
	ctx context.Context,
	req *AuthRequest,
) (*AuthResponse, error) {
	m.mu.Lock()
	m.requestCount++
	count := m.requestCount
	m.mu.Unlock()

	// Simulate processing delay
	time.Sleep(10 * time.Millisecond)

	// Check if server is configured
	_, exists := m.serverConfigs[req.ServerName]
	if !exists {
		return nil, fmt.Errorf("server not configured: %s", req.ServerName)
	}

	// Check if token exists
	tokenKey := fmt.Sprintf("%s:%s", req.ServerName, req.UserID.String())
	token, exists := m.tokens[tokenKey]
	if !exists {
		return &AuthResponse{
			RequestID:  req.RequestID,
			StatusCode: 401,
			Error:      "no token available",
			ErrorCode:  "NO_TOKEN",
		}, nil
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return &AuthResponse{
			RequestID:  req.RequestID,
			StatusCode: 401,
			Error:      "token expired",
			ErrorCode:  "EXPIRED_TOKEN",
		}, nil
	}

	// Simulate successful response
	responseBody := map[string]interface{}{
		"success":     true,
		"request_id":  req.RequestID,
		"server_name": req.ServerName,
		"method":      req.Method,
		"url":         req.URL,
		"mock_count":  count,
	}

	bodyBytes, _ := json.Marshal(responseBody)

	return &AuthResponse{
		RequestID:  req.RequestID,
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:     bodyBytes,
		Duration: 10 * time.Millisecond,
	}, nil
}

// GetToken mocks token retrieval
func (m *MockOAuthInterceptor) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	token, exists := m.tokens[tokenKey]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	return token, nil
}

// RefreshToken mocks token refresh
func (m *MockOAuthInterceptor) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	token, exists := m.tokens[tokenKey]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Create new token with extended expiry
	newToken := &TokenData{
		ServerName:   token.ServerName,
		UserID:       token.UserID,
		TenantID:     token.TenantID,
		ProviderType: token.ProviderType,
		AccessToken:  fmt.Sprintf("refreshed-%s-%d", token.AccessToken, time.Now().Unix()),
		RefreshToken: fmt.Sprintf("new-refresh-%d", time.Now().Unix()),
		IDToken:      token.IDToken,
		TokenType:    token.TokenType,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		IssuedAt:     time.Now(),
		Scopes:       token.Scopes,
		StorageTier:  token.StorageTier,
		LastUsed:     time.Now(),
		UsageCount:   token.UsageCount + 1,
	}

	m.tokens[tokenKey] = newToken
	return newToken, nil
}

// StoreToken mocks token storage
func (m *MockOAuthInterceptor) StoreToken(ctx context.Context, token *TokenData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", token.ServerName, token.UserID.String())
	m.tokens[tokenKey] = token
	return nil
}

// RevokeToken mocks token revocation
func (m *MockOAuthInterceptor) RevokeToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	delete(m.tokens, tokenKey)
	return nil
}

// RegisterServer mocks server registration
func (m *MockOAuthInterceptor) RegisterServer(ctx context.Context, config *ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.serverConfigs[config.ServerName] = config
	return nil
}

// GetServerConfig mocks server config retrieval
func (m *MockOAuthInterceptor) GetServerConfig(
	ctx context.Context,
	serverName string,
) (*ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.serverConfigs[serverName]
	if !exists {
		return nil, fmt.Errorf("server configuration not found: %s", serverName)
	}

	return config, nil
}

// UpdateServerConfig mocks server config update
func (m *MockOAuthInterceptor) UpdateServerConfig(
	ctx context.Context,
	config *ServerConfig,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.serverConfigs[config.ServerName]
	if !exists {
		return fmt.Errorf("server configuration not found: %s", config.ServerName)
	}

	m.serverConfigs[config.ServerName] = config
	return nil
}

// RemoveServerConfig mocks server config removal
func (m *MockOAuthInterceptor) RemoveServerConfig(
	ctx context.Context,
	serverName string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.serverConfigs[serverName]
	if !exists {
		return fmt.Errorf("server configuration not found: %s", serverName)
	}

	delete(m.serverConfigs, serverName)
	return nil
}

// Health mocks health check
func (m *MockOAuthInterceptor) Health(ctx context.Context) error {
	return nil // Always healthy
}

// GetMetrics mocks metrics retrieval
func (m *MockOAuthInterceptor) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"mock_enabled":       true,
		"total_requests":     m.requestCount,
		"tokens_stored":      len(m.tokens),
		"servers_configured": len(m.serverConfigs),
		"timestamp":          time.Now().UTC(),
	}, nil
}

// Helper methods for testing

// SetMockToken adds a token to the mock storage for testing
func (m *MockOAuthInterceptor) SetMockToken(serverName string, userID uuid.UUID, token *TokenData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	m.tokens[tokenKey] = token
}

// GetRequestCount returns the number of requests processed
func (m *MockOAuthInterceptor) GetRequestCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount
}

// Reset clears all mock state
func (m *MockOAuthInterceptor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = make(map[string]*TokenData)
	m.serverConfigs = make(map[string]*ServerConfig)
	m.requestCount = 0
}

// MockTokenStorage provides an in-memory token storage for testing
type MockTokenStorage struct {
	tokens map[string]*TokenData
	mu     sync.RWMutex
}

// CreateMockTokenStorage creates a new mock token storage
func CreateMockTokenStorage() *MockTokenStorage {
	return &MockTokenStorage{
		tokens: make(map[string]*TokenData),
	}
}

// StoreToken stores a token in memory
func (m *MockTokenStorage) StoreToken(
	ctx context.Context,
	token *TokenData,
	tier StorageTier,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", token.ServerName, token.UserID.String())
	token.StorageTier = tier
	m.tokens[tokenKey] = token
	return nil
}

// GetToken retrieves a token from memory
func (m *MockTokenStorage) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	token, exists := m.tokens[tokenKey]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	return token, nil
}

// RefreshToken simulates token refresh
func (m *MockTokenStorage) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	token, exists := m.tokens[tokenKey]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	// Create refreshed token
	refreshedToken := &TokenData{
		ServerName:   token.ServerName,
		UserID:       token.UserID,
		TenantID:     token.TenantID,
		ProviderType: token.ProviderType,
		AccessToken:  fmt.Sprintf("refreshed-%s", token.AccessToken),
		RefreshToken: fmt.Sprintf("new-refresh-%d", time.Now().Unix()),
		IDToken:      token.IDToken,
		TokenType:    token.TokenType,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshAt:    time.Now().Add(45 * time.Minute),
		IssuedAt:     time.Now(),
		Scopes:       token.Scopes,
		StorageTier:  token.StorageTier,
		LastUsed:     time.Now(),
		UsageCount:   token.UsageCount + 1,
	}

	m.tokens[tokenKey] = refreshedToken
	return refreshedToken, nil
}

// DeleteToken removes a token from memory
func (m *MockTokenStorage) DeleteToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", serverName, userID.String())
	delete(m.tokens, tokenKey)
	return nil
}

// GetStorageTier returns the storage tier for a token
func (m *MockTokenStorage) GetStorageTier(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (StorageTier, error) {
	token, err := m.GetToken(ctx, serverName, userID)
	if err != nil {
		return 0, err
	}
	return token.StorageTier, nil
}

// ListTokens returns all tokens for a user
func (m *MockTokenStorage) ListTokens(
	ctx context.Context,
	userID uuid.UUID,
) ([]*TokenData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tokens []*TokenData
	userIDStr := userID.String()

	for key, token := range m.tokens {
		if fmt.Sprintf("%s:%s", token.ServerName, userIDStr) == key {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

// CleanupExpiredTokens removes expired tokens
func (m *MockTokenStorage) CleanupExpiredTokens(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for key, token := range m.tokens {
		if now.After(token.ExpiresAt) {
			delete(m.tokens, key)
			cleaned++
		}
	}

	return cleaned, nil
}

// MigrateTokens simulates token migration
func (m *MockTokenStorage) MigrateTokens(
	ctx context.Context,
	fromTier, toTier StorageTier,
) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	migrated := 0
	for _, token := range m.tokens {
		if token.StorageTier == fromTier {
			token.StorageTier = toTier
			migrated++
		}
	}

	return migrated, nil
}

// Health always returns healthy
func (m *MockTokenStorage) Health(ctx context.Context) error {
	return nil
}

// Helper methods

// SetToken adds a token directly for testing
func (m *MockTokenStorage) SetToken(token *TokenData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokenKey := fmt.Sprintf("%s:%s", token.ServerName, token.UserID.String())
	m.tokens[tokenKey] = token
}

// GetTokenCount returns the number of stored tokens
func (m *MockTokenStorage) GetTokenCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tokens)
}

// Reset clears all tokens
func (m *MockTokenStorage) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = make(map[string]*TokenData)
}

// MockAuditLogger provides a simple audit logger for testing
type MockAuditLogger struct {
	events []AuditEvent
	mu     sync.RWMutex
}

// CreateMockAuditLogger creates a new mock audit logger
func CreateMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{
		events: make([]AuditEvent, 0),
	}
}

// LogOAuthEvent logs an OAuth event
func (m *MockAuditLogger) LogOAuthEvent(ctx context.Context, event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, *event)
	return nil
}

// LogTokenRefresh logs a token refresh event
func (m *MockAuditLogger) LogTokenRefresh(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	success bool,
) error {
	event := &AuditEvent{
		EventID:    uuid.New().String(),
		EventType:  "token_refresh",
		Timestamp:  time.Now(),
		UserID:     userID,
		ServerName: serverName,
		Success:    success,
		Operation:  "refresh_token",
	}

	return m.LogOAuthEvent(ctx, event)
}

// LogAuthorizationFlow logs an authorization flow event
func (m *MockAuditLogger) LogAuthorizationFlow(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	provider ProviderType,
	success bool,
) error {
	event := &AuditEvent{
		EventID:    uuid.New().String(),
		EventType:  "authorization_flow",
		Timestamp:  time.Now(),
		UserID:     userID,
		ServerName: serverName,
		Provider:   provider,
		Success:    success,
		Operation:  "oauth_authorization",
	}

	return m.LogOAuthEvent(ctx, event)
}

// LogTokenRevocation logs a token revocation event
func (m *MockAuditLogger) LogTokenRevocation(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	success bool,
) error {
	event := &AuditEvent{
		EventID:    uuid.New().String(),
		EventType:  "token_revocation",
		Timestamp:  time.Now(),
		UserID:     userID,
		ServerName: serverName,
		Success:    success,
		Operation:  "revoke_token",
	}

	return m.LogOAuthEvent(ctx, event)
}

// GetUserActivity returns audit events for a user
func (m *MockAuditLogger) GetUserActivity(
	ctx context.Context,
	userID uuid.UUID,
	since time.Time,
) ([]*AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var userEvents []*AuditEvent
	for _, event := range m.events {
		if event.UserID == userID && event.Timestamp.After(since) {
			eventCopy := event
			userEvents = append(userEvents, &eventCopy)
		}
	}

	return userEvents, nil
}

// GetServerActivity returns audit events for a server
func (m *MockAuditLogger) GetServerActivity(
	ctx context.Context,
	serverName string,
	since time.Time,
) ([]*AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var serverEvents []*AuditEvent
	for _, event := range m.events {
		if event.ServerName == serverName && event.Timestamp.After(since) {
			eventCopy := event
			serverEvents = append(serverEvents, &eventCopy)
		}
	}

	return serverEvents, nil
}

// GetFailedAttempts returns failed audit events
func (m *MockAuditLogger) GetFailedAttempts(
	ctx context.Context,
	since time.Time,
) ([]*AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var failedEvents []*AuditEvent
	for _, event := range m.events {
		if !event.Success && event.Timestamp.After(since) {
			eventCopy := event
			failedEvents = append(failedEvents, &eventCopy)
		}
	}

	return failedEvents, nil
}

// GetEvents returns all audit events
func (m *MockAuditLogger) GetEvents() []AuditEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]AuditEvent, len(m.events))
	copy(events, m.events)
	return events
}

// GetEventCount returns the number of logged events
func (m *MockAuditLogger) GetEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events)
}

// Reset clears all events
func (m *MockAuditLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]AuditEvent, 0)
}

// MockMetricsCollector provides basic metrics collection for testing
type MockMetricsCollector struct {
	metrics *Metrics
	mu      sync.RWMutex
}

// CreateMockMetricsCollector creates a new mock metrics collector
func CreateMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: &Metrics{
			ProviderCounts:    make(map[ProviderType]int64),
			ProviderLatencies: make(map[ProviderType]time.Duration),
			ErrorCounts:       make(map[string]int64),
			StorageTierUsage:  make(map[StorageTier]int64),
			LastUpdated:       time.Now(),
		},
	}
}

// RecordRequest records a request metric
func (m *MockMetricsCollector) RecordRequest(
	ctx context.Context,
	serverName string,
	provider ProviderType,
	duration time.Duration,
	success bool,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalRequests++
	if success {
		m.metrics.SuccessfulRequests++
	} else {
		m.metrics.FailedRequests++
	}

	m.metrics.ProviderCounts[provider]++
	m.metrics.LastUpdated = time.Now()

	// Update latency (simple average)
	currentLatency := m.metrics.ProviderLatencies[provider]
	count := m.metrics.ProviderCounts[provider]
	newAverage := (currentLatency*time.Duration(count-1) + duration) / time.Duration(count)
	m.metrics.ProviderLatencies[provider] = newAverage
}

// RecordTokenRefresh records a token refresh metric
func (m *MockMetricsCollector) RecordTokenRefresh(
	ctx context.Context,
	serverName string,
	provider ProviderType,
	success bool,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TokenRefreshCount++
	if !success {
		m.metrics.ErrorCounts["refresh_failed"]++
	}
	m.metrics.LastUpdated = time.Now()
}

// RecordError records an error metric
func (m *MockMetricsCollector) RecordError(
	ctx context.Context,
	errorType string,
	serverName string,
	provider ProviderType,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.ErrorCounts[errorType]++
	m.metrics.LastUpdated = time.Now()
}

// GetMetrics returns current metrics
func (m *MockMetricsCollector) GetMetrics(ctx context.Context) (*Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	metricsCopy := *m.metrics
	metricsCopy.ProviderCounts = make(map[ProviderType]int64)
	metricsCopy.ProviderLatencies = make(map[ProviderType]time.Duration)
	metricsCopy.ErrorCounts = make(map[string]int64)
	metricsCopy.StorageTierUsage = make(map[StorageTier]int64)

	for k, v := range m.metrics.ProviderCounts {
		metricsCopy.ProviderCounts[k] = v
	}
	for k, v := range m.metrics.ProviderLatencies {
		metricsCopy.ProviderLatencies[k] = v
	}
	for k, v := range m.metrics.ErrorCounts {
		metricsCopy.ErrorCounts[k] = v
	}
	for k, v := range m.metrics.StorageTierUsage {
		metricsCopy.StorageTierUsage[k] = v
	}

	return &metricsCopy, nil
}

// Reset resets all metrics
func (m *MockMetricsCollector) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = &Metrics{
		ProviderCounts:    make(map[ProviderType]int64),
		ProviderLatencies: make(map[ProviderType]time.Duration),
		ErrorCounts:       make(map[string]int64),
		StorageTierUsage:  make(map[StorageTier]int64),
		LastUpdated:       time.Now(),
	}

	return nil
}
