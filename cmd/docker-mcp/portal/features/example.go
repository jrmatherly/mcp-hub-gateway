package features

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/oauth"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// ExampleUsage demonstrates how to use the feature flag system for OAuth rollout
func ExampleUsage() {
	ctx := context.Background()

	// 1. Create dependencies (these would be injected in real usage)
	dbPool, _ := createExampleDatabasePool()
	cacheProvider, _ := createExampleCache()
	auditor, _ := createExampleAuditor()
	oauthInterceptor, _ := createExampleOAuthInterceptor()

	// 2. Create feature flag components
	store, err := CreateFlagStore(dbPool)
	if err != nil {
		log.Fatalf("Failed to create flag store: %v", err)
	}

	engine, err := CreateEvaluationEngine()
	if err != nil {
		log.Fatalf("Failed to create evaluation engine: %v", err)
	}

	// Create cache adapter
	cacheAdapter := CreateSimpleCacheAdapter()

	metrics, err := CreateMetricsCollector(cacheProvider)
	if err != nil {
		log.Fatalf("Failed to create metrics collector: %v", err)
	}

	loader, err := CreateConfigurationLoader(dbPool)
	if err != nil {
		log.Fatalf("Failed to create configuration loader: %v", err)
	}

	// 3. Create the main flag manager
	flagManager, err := CreateFlagManager(store, engine, cacheAdapter, metrics, loader, auditor)
	if err != nil {
		log.Fatalf("Failed to create flag manager: %v", err)
	}

	// 4. Create OAuth integration
	oauthIntegration, err := CreateOAuthFeatureIntegration(flagManager, oauthInterceptor)
	if err != nil {
		log.Fatalf("Failed to create OAuth integration: %v", err)
	}

	// 5. Create feature-flagged OAuth interceptor
	flaggedInterceptor := CreateFeatureFlaggedOAuthInterceptor(oauthInterceptor, oauthIntegration)

	// 6. Example: Check if OAuth is enabled for a user
	userID := uuid.New()
	tenantID := "example-tenant"

	evalCtx := &EvaluationContext{
		UserID:      userID,
		TenantID:    tenantID,
		ServerName:  "github-server",
		Environment: "production",
		Timestamp:   time.Now(),
		Attributes: map[string]interface{}{
			"user_type": "enterprise",
			"region":    "us-west-2",
		},
	}

	// Check OAuth enabled flag
	oauthEnabled, err := flagManager.EvaluateBooleanFlag(ctx, FlagOAuthEnabled, evalCtx)
	if err != nil {
		log.Printf("Failed to evaluate OAuth flag: %v", err)
		return
	}

	log.Printf("OAuth enabled for user %s: %v", userID, oauthEnabled)

	// 7. Example: Check specific OAuth features
	if oauthEnabled {
		auto401, _ := oauthIntegration.IsAuto401Enabled(ctx, evalCtx)
		log.Printf("Auto 401 handling enabled: %v", auto401)

		githubEnabled, _ := oauthIntegration.IsProviderEnabled(
			ctx,
			oauth.ProviderTypeGitHub,
			evalCtx,
		)
		log.Printf("GitHub provider enabled: %v", githubEnabled)

		config, _ := oauthIntegration.GetOAuthConfiguration(ctx, evalCtx)
		log.Printf("OAuth configuration: %+v", config)
	}

	// 8. Example: Handle an OAuth request with feature flags
	authRequest := &oauth.AuthRequest{
		RequestID:    uuid.New().String(),
		ServerName:   "github-server",
		UserID:       userID,
		Method:       "GET",
		URL:          "https://api.github.com/user",
		Headers:      map[string]string{"Authorization": "Bearer token"},
		Timestamp:    time.Now(),
		AttemptCount: 1,
		MaxRetries:   3,
	}

	response, err := flaggedInterceptor.InterceptRequest(ctx, authRequest)
	if err != nil {
		log.Printf("OAuth interception failed: %v", err)
	} else {
		log.Printf("OAuth response status: %d", response.StatusCode)
	}

	// 9. Example: Gradual rollout by updating flag percentage
	err = updateOAuthRolloutPercentage(ctx, flagManager, 25) // Enable for 25% of users
	if err != nil {
		log.Printf("Failed to update rollout: %v", err)
	}

	// 10. Example: Create an A/B test for OAuth providers
	err = createOAuthProviderExperiment(ctx, flagManager)
	if err != nil {
		log.Printf("Failed to create experiment: %v", err)
	}

	// 11. Example: Get metrics
	globalMetrics, err := flagManager.GetMetrics(ctx)
	if err == nil {
		log.Printf("Total flag evaluations: %d", globalMetrics.TotalEvaluations)
		log.Printf("Cache hit rate: %.2f%%", globalMetrics.CacheHitRate)
	}

	oauthFlagMetrics, err := flagManager.GetFlagMetrics(ctx, FlagOAuthEnabled)
	if err == nil {
		log.Printf("OAuth flag true rate: %.2f%%", oauthFlagMetrics.TrueRate)
	}

	// 12. Example: Health check
	if err := flagManager.Health(ctx); err != nil {
		log.Printf("Feature flag system is unhealthy: %v", err)
	} else {
		log.Println("Feature flag system is healthy")
	}
}

// updateOAuthRolloutPercentage demonstrates updating a flag's rollout percentage
func updateOAuthRolloutPercentage(
	ctx context.Context,
	flagManager FlagManager,
	percentage int,
) error {
	// Get the current flag definition
	flag, err := flagManager.GetFlag(ctx, FlagOAuthEnabled)
	if err != nil {
		return fmt.Errorf("failed to get OAuth flag: %w", err)
	}

	// Update the rollout percentage
	flag.RolloutPercentage = percentage
	flag.UpdatedAt = time.Now()
	flag.Version++

	// Save the updated flag
	err = flagManager.UpdateFlag(ctx, flag)
	if err != nil {
		return fmt.Errorf("failed to update OAuth flag: %w", err)
	}

	log.Printf("Updated OAuth rollout to %d%%", percentage)
	return nil
}

// createOAuthProviderExperiment demonstrates creating an A/B test
func createOAuthProviderExperiment(ctx context.Context, flagManager FlagManager) error {
	// This would typically use an experiment manager
	// For this example, we'll just create a flag with variants

	experiment := &FlagDefinition{
		Name:              FlagOAuthProviderGitHub,
		Type:              FlagTypeVariant,
		Description:       "A/B test GitHub vs Google OAuth providers",
		Enabled:           true,
		DefaultValue:      false,
		RolloutPercentage: 50, // 50% of users in experiment
		Variants: []FlagVariant{
			{
				Name:    "github",
				Weight:  50,
				Value:   true,
				Enabled: true,
			},
			{
				Name:    "google",
				Weight:  50,
				Value:   false, // Use Google instead
				Enabled: true,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
		Tags:      []string{"experiment", "oauth", "provider"},
	}

	err := flagManager.UpdateFlag(ctx, experiment)
	if err != nil {
		return fmt.Errorf("failed to create OAuth provider experiment: %w", err)
	}

	log.Println("Created OAuth provider A/B test")
	return nil
}

// Example helper functions (these would be real implementations in production)

func createExampleDatabasePool() (*mockDatabasePool, error) {
	// This would create a real database pool
	// For the example, we'll return a mock
	return &mockDatabasePool{}, nil
}

func createExampleCache() (cache.Cache, error) {
	// This would create a real cache (Redis, etc.)
	// For the example, we'll return a mock
	return &mockCache{}, nil
}

func createExampleAuditor() (audit.Logger, error) {
	// This would create a real audit logger
	// For the example, we'll return a mock
	return &mockAuditor{}, nil
}

func createExampleOAuthInterceptor() (oauth.OAuthInterceptor, error) {
	// This would create a real OAuth interceptor
	// For the example, we'll return a mock
	return &mockOAuthInterceptor{}, nil
}

// Mock implementations for example (in production, these would be real)

type mockDatabasePool struct{}

func (m *mockDatabasePool) GetPool() interface{} {
	return nil
}

func (m *mockDatabasePool) Health(ctx context.Context) error {
	return nil
}

func (m *mockDatabasePool) Stats(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *mockDatabasePool) Get(ctx context.Context) (*mockConnection, error) {
	return &mockConnection{}, nil
}

func (m *mockDatabasePool) Close() error {
	return nil
}

type mockConnection struct{}

func (m *mockConnection) Query(
	ctx context.Context,
	sql string,
	args ...interface{},
) (*mockRows, error) {
	return &mockRows{}, nil
}

func (m *mockConnection) QueryRow(
	ctx context.Context,
	sql string,
	args ...interface{},
) *mockRow {
	return &mockRow{}
}

func (m *mockConnection) Exec(
	ctx context.Context,
	sql string,
	args ...interface{},
) (*mockResult, error) {
	return &mockResult{}, nil
}

func (m *mockConnection) Release() {}

type mockRows struct{}

func (m *mockRows) Next() bool                     { return false }
func (m *mockRows) Scan(dest ...interface{}) error { return nil }
func (m *mockRows) Close()                         {}
func (m *mockRows) Err() error                     { return nil }

type mockRow struct{}

func (m *mockRow) Scan(dest ...interface{}) error { return nil }

type mockResult struct{}

func (m *mockResult) RowsAffected() int64 { return 1 }

type mockCache struct{}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockCache) Set(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (m *mockCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) DeletePattern(ctx context.Context, pattern string) (int, error) {
	return 0, nil
}

func (m *mockCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return []string{}, nil
}

func (m *mockCache) MultiGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

func (m *mockCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return delta, nil
}

func (m *mockCache) MultiSet(ctx context.Context, items map[string]cache.CacheItem) error {
	return nil
}

func (m *mockCache) MultiDelete(ctx context.Context, keys []string) error {
	return nil
}

func (m *mockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *mockCache) Health(ctx context.Context) error {
	return nil
}

func (m *mockCache) Info(ctx context.Context) (*cache.CacheInfo, error) {
	return nil, nil
}

func (m *mockCache) FlushDB(ctx context.Context) error {
	return nil
}

type mockAuditor struct{}

func (m *mockAuditor) Log(
	ctx context.Context,
	action audit.Action,
	entityType, entityID, userID string,
	metadata map[string]interface{},
) error {
	log.Printf("Audit: %s %s %s by %s", action, entityType, entityID, userID)
	return nil
}

func (m *mockAuditor) GetLogs(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]audit.AuditEntry, error) {
	return []audit.AuditEntry{}, nil
}

func (m *mockAuditor) LogCommand(
	ctx context.Context,
	userID uuid.UUID,
	command string,
	args []string,
) uuid.UUID {
	log.Printf("Command: %s by %s", command, userID)
	return uuid.New()
}

func (m *mockAuditor) LogCommandResult(
	ctx context.Context,
	auditID uuid.UUID,
	result string,
	err error,
	duration time.Duration,
) {
	log.Printf("Command result: %s", result)
}

func (m *mockAuditor) LogSecurityEvent(
	ctx context.Context,
	userID uuid.UUID,
	event audit.EventType,
	details map[string]interface{},
) {
	log.Printf("Security event: %s by %s", event, userID)
}

func (m *mockAuditor) LogAccessDenied(
	ctx context.Context,
	userID uuid.UUID,
	resource string,
	reason string,
) {
	log.Printf("Access denied: %s to %s (%s)", userID, resource, reason)
}

func (m *mockAuditor) LogRateLimitExceeded(ctx context.Context, userID uuid.UUID, command string) {
	log.Printf("Rate limit exceeded: %s for %s", userID, command)
}

type mockOAuthInterceptor struct{}

func (m *mockOAuthInterceptor) InterceptRequest(
	ctx context.Context,
	req *oauth.AuthRequest,
) (*oauth.AuthResponse, error) {
	return &oauth.AuthResponse{
		RequestID:  req.RequestID,
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"user": "test"}`),
		Duration:   time.Millisecond * 100,
	}, nil
}

func (m *mockOAuthInterceptor) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*oauth.TokenData, error) {
	return &oauth.TokenData{
		ServerName:  serverName,
		UserID:      userID,
		AccessToken: "mock-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour),
		IssuedAt:    time.Now(),
		Scopes:      []string{"read", "write"},
		StorageTier: oauth.StorageTierEnvironment,
	}, nil
}

func (m *mockOAuthInterceptor) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*oauth.TokenData, error) {
	return m.GetToken(ctx, serverName, userID)
}

func (m *mockOAuthInterceptor) StoreToken(ctx context.Context, token *oauth.TokenData) error {
	return nil
}

func (m *mockOAuthInterceptor) RevokeToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	return nil
}

func (m *mockOAuthInterceptor) RegisterServer(
	ctx context.Context,
	config *oauth.ServerConfig,
) error {
	return nil
}

func (m *mockOAuthInterceptor) GetServerConfig(
	ctx context.Context,
	serverName string,
) (*oauth.ServerConfig, error) {
	return &oauth.ServerConfig{
		ServerName:   serverName,
		ProviderType: oauth.ProviderTypeGitHub,
		ClientID:     "mock-client-id",
		Scopes:       []string{"read", "write"},
		RedirectURI:  "http://localhost:3000/callback",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}, nil
}

func (m *mockOAuthInterceptor) UpdateServerConfig(
	ctx context.Context,
	config *oauth.ServerConfig,
) error {
	return nil
}

func (m *mockOAuthInterceptor) RemoveServerConfig(ctx context.Context, serverName string) error {
	return nil
}

func (m *mockOAuthInterceptor) Health(ctx context.Context) error {
	return nil
}

func (m *mockOAuthInterceptor) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_requests": 100,
		"success_rate":   0.95,
	}, nil
}
