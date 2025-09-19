package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultOAuthInterceptor implements the OAuthInterceptor interface
type DefaultOAuthInterceptor struct {
	config           *InterceptorConfig
	tokenStorage     TokenStorage
	providerRegistry ProviderRegistry
	auditLogger      AuditLogger
	metricsCollector MetricsCollector
	configValidator  ConfigValidator
	serverConfigs    map[string]*ServerConfig
	httpClient       *http.Client
	mu               sync.RWMutex
}

// CreateOAuthInterceptor creates a new OAuth interceptor instance
func CreateOAuthInterceptor(
	config *InterceptorConfig,
	tokenStorage TokenStorage,
	providerRegistry ProviderRegistry,
	auditLogger AuditLogger,
	metricsCollector MetricsCollector,
	configValidator ConfigValidator,
) *DefaultOAuthInterceptor {
	return &DefaultOAuthInterceptor{
		config:           config,
		tokenStorage:     tokenStorage,
		providerRegistry: providerRegistry,
		auditLogger:      auditLogger,
		metricsCollector: metricsCollector,
		configValidator:  configValidator,
		serverConfigs:    make(map[string]*ServerConfig),
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
		},
	}
}

// InterceptRequest intercepts an HTTP request and adds OAuth authentication
func (i *DefaultOAuthInterceptor) InterceptRequest(
	ctx context.Context,
	req *AuthRequest,
) (*AuthResponse, error) {
	if !i.config.Enabled {
		return nil, fmt.Errorf("OAuth interceptor is disabled")
	}

	startTime := time.Now()

	// Get server configuration
	serverConfig, err := i.GetServerConfig(ctx, req.ServerName)
	if err != nil {
		return nil, fmt.Errorf("server configuration not found: %w", err)
	}

	// Get provider
	provider, err := i.providerRegistry.GetProvider(serverConfig.ProviderType)
	if err != nil {
		return nil, fmt.Errorf("OAuth provider not found: %w", err)
	}

	// Attempt request with retry logic
	response, err := i.attemptRequestWithRetry(ctx, req, serverConfig, provider)

	// Record metrics
	duration := time.Since(startTime)
	success := err == nil && response != nil && response.StatusCode < 400
	i.metricsCollector.RecordRequest(
		ctx,
		req.ServerName,
		serverConfig.ProviderType,
		duration,
		success,
	)

	// Log audit event
	if i.auditLogger != nil {
		auditEvent := &AuditEvent{
			EventID:    uuid.New().String(),
			EventType:  "oauth_request",
			Timestamp:  startTime,
			UserID:     req.UserID,
			TenantID:   req.TenantID,
			ServerName: req.ServerName,
			RequestID:  req.RequestID,
			Success:    success,
			Duration:   duration,
			Provider:   serverConfig.ProviderType,
			Operation:  "intercept_request",
			RemoteAddr: req.RemoteAddr,
			UserAgent:  req.UserAgent,
		}

		if err != nil {
			auditEvent.Error = err.Error()
		}

		if response != nil {
			auditEvent.TokenRefreshed = response.TokenRefreshed
			auditEvent.Details = map[string]interface{}{
				"status_code":   response.StatusCode,
				"attempt_count": req.AttemptCount,
			}
		}

		_ = i.auditLogger.LogOAuthEvent(ctx, auditEvent)
	}

	return response, err
}

// attemptRequestWithRetry performs the request with automatic retry and token refresh
func (i *DefaultOAuthInterceptor) attemptRequestWithRetry(
	ctx context.Context,
	req *AuthRequest,
	serverConfig *ServerConfig,
	provider OAuthProvider,
) (*AuthResponse, error) {
	var lastErr error
	var lastResponse *AuthResponse

	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = i.config.RetryPolicy.MaxRetries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req.AttemptCount = attempt + 1

		// Get current token
		token, err := i.GetToken(ctx, req.ServerName, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get OAuth token: %w", err)
		}

		// Check if token needs refresh
		if i.shouldRefreshToken(token) {
			refreshedToken, refreshErr := i.refreshTokenIfNeeded(
				ctx,
				req.ServerName,
				req.UserID,
				provider,
				serverConfig,
			)
			if refreshErr != nil {
				// Log refresh failure but continue with existing token
				i.metricsCollector.RecordTokenRefresh(
					ctx,
					req.ServerName,
					serverConfig.ProviderType,
					false,
				)
			} else {
				token = refreshedToken
				i.metricsCollector.RecordTokenRefresh(ctx, req.ServerName, serverConfig.ProviderType, true)
			}
		}

		// Make the HTTP request
		response, err := i.makeHTTPRequest(ctx, req, token)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				break
			}
			time.Sleep(i.calculateBackoffDelay(attempt))
			continue
		}

		lastResponse = response

		// Check if we should retry based on status code
		shouldRetry := i.shouldRetryOnStatusCode(response.StatusCode)
		if !shouldRetry {
			return response, nil
		}

		// Handle 401 specifically - token might be invalid
		if response.StatusCode == http.StatusUnauthorized {
			// Try to refresh token
			if provider.SupportsRefresh() && token.RefreshToken != "" {
				refreshedToken, refreshErr := i.RefreshToken(ctx, req.ServerName, req.UserID)
				if refreshErr == nil {
					// Mark that token was refreshed for this response
					response.TokenRefreshed = true

					// Retry with refreshed token
					retryResponse, retryErr := i.makeHTTPRequest(ctx, req, refreshedToken)
					if retryErr == nil && retryResponse.StatusCode != http.StatusUnauthorized {
						retryResponse.TokenRefreshed = true
						return retryResponse, nil
					}
				}
			}
		}

		// If we've reached max retries, return the last response
		if attempt == maxRetries {
			break
		}

		// Wait before retrying
		time.Sleep(i.calculateBackoffDelay(attempt))
	}

	if lastResponse != nil {
		return lastResponse, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// makeHTTPRequest makes the actual HTTP request with OAuth token
func (i *DefaultOAuthInterceptor) makeHTTPRequest(
	ctx context.Context,
	req *AuthRequest,
	token *TokenData,
) (*AuthResponse, error) {
	startTime := time.Now()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add OAuth token to Authorization header
	httpReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add request body if present
	if len(req.Body) > 0 {
		httpReq.Body = io.NopCloser(strings.NewReader(string(req.Body)))
		httpReq.ContentLength = int64(len(req.Body))
	}

	// Set User-Agent
	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	// Make the request
	resp, err := i.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Update token usage statistics
	token.LastUsed = time.Now()
	token.UsageCount++
	_ = i.tokenStorage.StoreToken(ctx, token, token.StorageTier) // Update usage stats

	// Build response
	response := &AuthResponse{
		RequestID:  req.RequestID,
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		Body:       body,
		Duration:   time.Since(startTime),
	}

	// Copy response headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	// Handle error status codes
	if resp.StatusCode >= 400 {
		response.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		response.ErrorCode = fmt.Sprintf("HTTP_%d", resp.StatusCode)

		// Try to extract error details from response body
		if len(body) > 0 {
			var errorDetails map[string]interface{}
			if err := json.Unmarshal(body, &errorDetails); err == nil {
				response.ErrorDetails = make(map[string]string)
				for key, value := range errorDetails {
					if str, ok := value.(string); ok {
						response.ErrorDetails[key] = str
					}
				}
			}
		}
	}

	return response, nil
}

// shouldRefreshToken determines if a token should be refreshed proactively
func (i *DefaultOAuthInterceptor) shouldRefreshToken(token *TokenData) bool {
	if token.RefreshToken == "" {
		return false
	}

	now := time.Now()

	// Check if token is already expired
	if now.After(token.ExpiresAt) {
		return true
	}

	// Check if token should be refreshed proactively
	if now.After(token.RefreshAt) {
		return true
	}

	// Check against configured refresh threshold
	timeToExpiry := token.ExpiresAt.Sub(now)
	if timeToExpiry < i.config.RefreshThreshold {
		return true
	}

	return false
}

// refreshTokenIfNeeded refreshes a token if needed and supported
func (i *DefaultOAuthInterceptor) refreshTokenIfNeeded(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	provider OAuthProvider,
	serverConfig *ServerConfig,
) (*TokenData, error) {
	if !provider.SupportsRefresh() {
		return nil, fmt.Errorf("provider does not support token refresh")
	}

	currentToken, err := i.GetToken(ctx, serverName, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current token: %w", err)
	}

	if currentToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Refresh the token
	refreshedToken, err := provider.RefreshToken(ctx, serverConfig, currentToken.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update token metadata
	refreshedToken.ServerName = serverName
	refreshedToken.UserID = userID
	refreshedToken.TenantID = currentToken.TenantID
	refreshedToken.StorageTier = currentToken.StorageTier

	// Store the refreshed token
	if err := i.tokenStorage.StoreToken(ctx, refreshedToken, refreshedToken.StorageTier); err != nil {
		return nil, fmt.Errorf("failed to store refreshed token: %w", err)
	}

	return refreshedToken, nil
}

// shouldRetryOnStatusCode determines if a request should be retried based on status code
func (i *DefaultOAuthInterceptor) shouldRetryOnStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized:
		return i.config.RetryPolicy.RetryOn401
	case http.StatusForbidden:
		return i.config.RetryPolicy.RetryOn403
	case http.StatusTooManyRequests:
		return i.config.RetryPolicy.RetryOn429
	default:
		if statusCode >= 500 && statusCode < 600 {
			return i.config.RetryPolicy.RetryOn5xx
		}
		return false
	}
}

// calculateBackoffDelay calculates delay for exponential backoff
func (i *DefaultOAuthInterceptor) calculateBackoffDelay(attempt int) time.Duration {
	policy := i.config.RetryPolicy

	delay := time.Duration(
		float64(policy.InitialInterval) * math.Pow(policy.Multiplier, float64(attempt)),
	)

	if delay > policy.MaxInterval {
		delay = policy.MaxInterval
	}

	// Add jitter if enabled
	if policy.Jitter {
		jitterRange := float64(delay) * 0.1 // 10% jitter
		jitter := time.Duration(rand.Float64() * jitterRange)
		if rand.Intn(2) == 0 {
			delay += jitter
		} else {
			delay -= jitter
		}
	}

	return delay
}

// GetToken retrieves a token for a server and user
func (i *DefaultOAuthInterceptor) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	return i.tokenStorage.GetToken(ctx, serverName, userID)
}

// RefreshToken refreshes a token for a server and user
func (i *DefaultOAuthInterceptor) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*TokenData, error) {
	return i.tokenStorage.RefreshToken(ctx, serverName, userID)
}

// StoreToken stores a token
func (i *DefaultOAuthInterceptor) StoreToken(ctx context.Context, token *TokenData) error {
	// Validate token before storing
	if i.configValidator != nil {
		if errors := i.configValidator.ValidateToken(token); len(errors) > 0 {
			return fmt.Errorf("token validation failed: %s", errors[0].Message)
		}
	}

	// Use highest priority storage tier by default
	tier := StorageTierKeyVault
	if len(i.config.StorageTiers) > 0 {
		tier = i.config.StorageTiers[0]
	}

	return i.tokenStorage.StoreToken(ctx, token, tier)
}

// RevokeToken revokes a token
func (i *DefaultOAuthInterceptor) RevokeToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	// Get current token
	token, err := i.GetToken(ctx, serverName, userID)
	if err != nil {
		return fmt.Errorf("failed to get token for revocation: %w", err)
	}

	// Get server config and provider
	serverConfig, err := i.GetServerConfig(ctx, serverName)
	if err != nil {
		return fmt.Errorf("server configuration not found: %w", err)
	}

	provider, err := i.providerRegistry.GetProvider(serverConfig.ProviderType)
	if err != nil {
		return fmt.Errorf("OAuth provider not found: %w", err)
	}

	// Revoke with provider if supported
	if provider.SupportsRevocation() {
		if err := provider.RevokeToken(ctx, serverConfig, token.AccessToken); err != nil {
			// Log error but continue with local deletion
			if i.auditLogger != nil {
				_ = i.auditLogger.LogTokenRevocation(ctx, serverName, userID, false)
			}
		} else {
			if i.auditLogger != nil {
				_ = i.auditLogger.LogTokenRevocation(ctx, serverName, userID, true)
			}
		}
	}

	// Delete from local storage
	return i.tokenStorage.DeleteToken(ctx, serverName, userID)
}

// RegisterServer registers an OAuth server configuration
func (i *DefaultOAuthInterceptor) RegisterServer(ctx context.Context, config *ServerConfig) error {
	// Validate configuration
	if i.configValidator != nil {
		if errors := i.configValidator.ValidateServerConfig(config); len(errors) > 0 {
			return fmt.Errorf("server configuration validation failed: %s", errors[0].Message)
		}
	}

	// Check if provider is supported
	_, err := i.providerRegistry.GetProvider(config.ProviderType)
	if err != nil {
		return fmt.Errorf("unsupported OAuth provider: %w", err)
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	i.serverConfigs[config.ServerName] = config
	return nil
}

// GetServerConfig retrieves server configuration
func (i *DefaultOAuthInterceptor) GetServerConfig(
	ctx context.Context,
	serverName string,
) (*ServerConfig, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	config, exists := i.serverConfigs[serverName]
	if !exists {
		return nil, fmt.Errorf("server configuration not found: %s", serverName)
	}

	if !config.IsActive {
		return nil, fmt.Errorf("server configuration is disabled: %s", serverName)
	}

	return config, nil
}

// UpdateServerConfig updates server configuration
func (i *DefaultOAuthInterceptor) UpdateServerConfig(
	ctx context.Context,
	config *ServerConfig,
) error {
	// Validate configuration
	if i.configValidator != nil {
		if errors := i.configValidator.ValidateServerConfig(config); len(errors) > 0 {
			return fmt.Errorf("server configuration validation failed: %s", errors[0].Message)
		}
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	existingConfig, exists := i.serverConfigs[config.ServerName]
	if !exists {
		return fmt.Errorf("server configuration not found: %s", config.ServerName)
	}

	// Preserve creation metadata
	config.CreatedAt = existingConfig.CreatedAt
	config.CreatedBy = existingConfig.CreatedBy
	config.UpdatedAt = time.Now()

	i.serverConfigs[config.ServerName] = config
	return nil
}

// RemoveServerConfig removes server configuration
func (i *DefaultOAuthInterceptor) RemoveServerConfig(ctx context.Context, serverName string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.serverConfigs[serverName]; !exists {
		return fmt.Errorf("server configuration not found: %s", serverName)
	}

	delete(i.serverConfigs, serverName)
	return nil
}

// Health checks the health of the OAuth interceptor
func (i *DefaultOAuthInterceptor) Health(ctx context.Context) error {
	// Check token storage health
	if err := i.tokenStorage.Health(ctx); err != nil {
		return fmt.Errorf("token storage unhealthy: %w", err)
	}

	// Check provider registry
	providers := i.providerRegistry.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no OAuth providers registered")
	}

	// Test HTTP client
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	testReq, err := http.NewRequestWithContext(testCtx, "GET", "https://httpbin.org/get", nil)
	if err == nil {
		_, err = i.httpClient.Do(testReq)
		if err != nil {
			return fmt.Errorf("HTTP client unhealthy: %w", err)
		}
	}

	return nil
}

// GetMetrics returns current OAuth metrics
func (i *DefaultOAuthInterceptor) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	if i.metricsCollector == nil {
		return map[string]interface{}{
			"metrics_disabled": true,
		}, nil
	}

	metrics, err := i.metricsCollector.GetMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Add interceptor-specific metrics
	result := map[string]interface{}{
		"oauth_metrics":       metrics,
		"server_configs":      len(i.serverConfigs),
		"supported_providers": len(i.providerRegistry.ListProviders()),
		"config": map[string]interface{}{
			"enabled":           i.config.Enabled,
			"default_timeout":   i.config.DefaultTimeout.String(),
			"refresh_threshold": i.config.RefreshThreshold.String(),
			"storage_tiers":     i.config.StorageTiers,
		},
	}

	return result, nil
}

// HTTPClientAdapter adapts the OAuth interceptor for HTTP client usage
type HTTPClientAdapter struct {
	interceptor OAuthInterceptor
}

// CreateHTTPClient creates an HTTP client that uses OAuth interceptor
func CreateHTTPClient(interceptor OAuthInterceptor) HTTPClient {
	return &HTTPClientAdapter{
		interceptor: interceptor,
	}
}

// Get performs an OAuth-authenticated GET request
func (h *HTTPClientAdapter) Get(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	url string,
) (*AuthResponse, error) {
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "GET",
		URL:        url,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	return h.interceptor.InterceptRequest(ctx, req)
}

// Post performs an OAuth-authenticated POST request
func (h *HTTPClientAdapter) Post(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	url string,
	body []byte,
) (*AuthResponse, error) {
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "POST",
		URL:        url,
		Body:       body,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	return h.interceptor.InterceptRequest(ctx, req)
}

// Put performs an OAuth-authenticated PUT request
func (h *HTTPClientAdapter) Put(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	url string,
	body []byte,
) (*AuthResponse, error) {
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "PUT",
		URL:        url,
		Body:       body,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	return h.interceptor.InterceptRequest(ctx, req)
}

// Delete performs an OAuth-authenticated DELETE request
func (h *HTTPClientAdapter) Delete(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
	url string,
) (*AuthResponse, error) {
	req := &AuthRequest{
		RequestID:  uuid.New().String(),
		ServerName: serverName,
		UserID:     userID,
		Method:     "DELETE",
		URL:        url,
		Timestamp:  time.Now(),
		MaxRetries: 3,
	}

	return h.interceptor.InterceptRequest(ctx, req)
}

// Do performs an OAuth-authenticated HTTP request with custom parameters
func (h *HTTPClientAdapter) Do(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}

	return h.interceptor.InterceptRequest(ctx, req)
}
