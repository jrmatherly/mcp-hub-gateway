package features

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/oauth"
)

// OAuthFeatureIntegration integrates feature flags with OAuth functionality
type OAuthFeatureIntegration struct {
	flagManager      FlagManager
	oauthInterceptor oauth.OAuthInterceptor
}

// CreateOAuthFeatureIntegration creates a new OAuth feature integration
func CreateOAuthFeatureIntegration(
	flagManager FlagManager,
	oauthInterceptor oauth.OAuthInterceptor,
) (*OAuthFeatureIntegration, error) {
	if flagManager == nil {
		return nil, fmt.Errorf("flag manager is required")
	}
	if oauthInterceptor == nil {
		return nil, fmt.Errorf("oauth interceptor is required")
	}

	return &OAuthFeatureIntegration{
		flagManager:      flagManager,
		oauthInterceptor: oauthInterceptor,
	}, nil
}

// OAuthEvaluationContext creates evaluation context from OAuth request
func (o *OAuthFeatureIntegration) OAuthEvaluationContext(
	req *oauth.AuthRequest,
	userID uuid.UUID,
	tenantID string,
) *EvaluationContext {
	return &EvaluationContext{
		UserID:      userID,
		TenantID:    tenantID,
		ServerName:  req.ServerName,
		RequestID:   req.RequestID,
		RemoteAddr:  req.RemoteAddr,
		UserAgent:   req.UserAgent,
		Headers:     req.Headers,
		Environment: "production", // Default, could be configurable
		Timestamp:   time.Now(),
		Attributes: map[string]interface{}{
			"oauth_method":    req.Method,
			"oauth_url":       req.URL,
			"oauth_attempt":   req.AttemptCount,
			"oauth_max_retry": req.MaxRetries,
		},
	}
}

// IsOAuthEnabled checks if OAuth is enabled globally
func (o *OAuthFeatureIntegration) IsOAuthEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthEnabled, evalCtx)
}

// IsAuto401Enabled checks if automatic 401 handling is enabled
func (o *OAuthFeatureIntegration) IsAuto401Enabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthAuto401, evalCtx)
}

// IsDCREnabled checks if Dynamic Client Registration is enabled
func (o *OAuthFeatureIntegration) IsDCREnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthDCR, evalCtx)
}

// IsProviderEnabled checks if a specific OAuth provider is enabled
func (o *OAuthFeatureIntegration) IsProviderEnabled(
	ctx context.Context,
	provider oauth.ProviderType,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	var flagName FlagName
	switch provider {
	case oauth.ProviderTypeGitHub:
		flagName = FlagOAuthProviderGitHub
	case oauth.ProviderTypeGoogle:
		flagName = FlagOAuthProviderGoogle
	case oauth.ProviderTypeMicrosoft:
		flagName = FlagOAuthProviderMicrosoft
	default:
		return false, fmt.Errorf("unsupported provider: %s", provider)
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, flagName, evalCtx)
}

// IsDockerSecretsEnabled checks if Docker Desktop secrets integration is enabled
func (o *OAuthFeatureIntegration) IsDockerSecretsEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthDockerSecrets, evalCtx)
}

// IsTokenRefreshEnabled checks if automatic token refresh is enabled
func (o *OAuthFeatureIntegration) IsTokenRefreshEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthTokenRefresh, evalCtx)
}

// IsTokenStorageEnabled checks if token storage functionality is enabled
func (o *OAuthFeatureIntegration) IsTokenStorageEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthTokenStorage, evalCtx)
}

// IsJWTValidationEnabled checks if JWT token validation is enabled
func (o *OAuthFeatureIntegration) IsJWTValidationEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthJWTValidation, evalCtx)
}

// IsHTTPSRequired checks if HTTPS is required for OAuth operations
func (o *OAuthFeatureIntegration) IsHTTPSRequired(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthHTTPSRequired, evalCtx)
}

// IsAuditLoggingEnabled checks if OAuth audit logging is enabled
func (o *OAuthFeatureIntegration) IsAuditLoggingEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthAuditLogging, evalCtx)
}

// IsMetricsEnabled checks if OAuth metrics collection is enabled
func (o *OAuthFeatureIntegration) IsMetricsEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthMetrics, evalCtx)
}

// IsKeyRotationEnabled checks if automatic key rotation is enabled
func (o *OAuthFeatureIntegration) IsKeyRotationEnabled(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (bool, error) {
	// First check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthKeyRotation, evalCtx)
}

// GetEnabledOAuthFeatures returns all enabled OAuth features for the context
func (o *OAuthFeatureIntegration) GetEnabledOAuthFeatures(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (map[FlagName]bool, error) {
	// Evaluate all OAuth flags
	oauthFlags := []FlagName{
		FlagOAuthEnabled,
		FlagOAuthAuto401,
		FlagOAuthDCR,
		FlagOAuthProviderGitHub,
		FlagOAuthProviderGoogle,
		FlagOAuthProviderMicrosoft,
		FlagOAuthDockerSecrets,
		FlagOAuthTokenRefresh,
		FlagOAuthTokenStorage,
		FlagOAuthJWTValidation,
		FlagOAuthHTTPSRequired,
		FlagOAuthAuditLogging,
		FlagOAuthMetrics,
		FlagOAuthKeyRotation,
	}

	results := make(map[FlagName]bool)

	for _, flag := range oauthFlags {
		enabled, err := o.flagManager.EvaluateBooleanFlag(ctx, flag, evalCtx)
		if err != nil {
			// Log error but continue with other flags
			results[flag] = false
		} else {
			results[flag] = enabled
		}
	}

	return results, nil
}

// GetOAuthConfiguration returns OAuth configuration based on feature flags
func (o *OAuthFeatureIntegration) GetOAuthConfiguration(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (*OAuthFeatureConfig, error) {
	features, err := o.GetEnabledOAuthFeatures(ctx, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled features: %w", err)
	}

	config := &OAuthFeatureConfig{
		Enabled:              features[FlagOAuthEnabled],
		Auto401Handling:      features[FlagOAuthAuto401],
		DCREnabled:           features[FlagOAuthDCR],
		DockerSecretsEnabled: features[FlagOAuthDockerSecrets],
		TokenRefreshEnabled:  features[FlagOAuthTokenRefresh],
		TokenStorageEnabled:  features[FlagOAuthTokenStorage],
		JWTValidationEnabled: features[FlagOAuthJWTValidation],
		HTTPSRequired:        features[FlagOAuthHTTPSRequired],
		AuditLoggingEnabled:  features[FlagOAuthAuditLogging],
		MetricsEnabled:       features[FlagOAuthMetrics],
		KeyRotationEnabled:   features[FlagOAuthKeyRotation],
		EnabledProviders:     make(map[oauth.ProviderType]bool),
	}

	// Map provider flags
	config.EnabledProviders[oauth.ProviderTypeGitHub] = features[FlagOAuthProviderGitHub]
	config.EnabledProviders[oauth.ProviderTypeGoogle] = features[FlagOAuthProviderGoogle]
	config.EnabledProviders[oauth.ProviderTypeMicrosoft] = features[FlagOAuthProviderMicrosoft]

	return config, nil
}

// OAuthFeatureConfig represents the OAuth configuration based on feature flags
type OAuthFeatureConfig struct {
	Enabled              bool                        `json:"enabled"`
	Auto401Handling      bool                        `json:"auto_401_handling"`
	DCREnabled           bool                        `json:"dcr_enabled"`
	DockerSecretsEnabled bool                        `json:"docker_secrets_enabled"`
	TokenRefreshEnabled  bool                        `json:"token_refresh_enabled"`
	TokenStorageEnabled  bool                        `json:"token_storage_enabled"`
	JWTValidationEnabled bool                        `json:"jwt_validation_enabled"`
	HTTPSRequired        bool                        `json:"https_required"`
	AuditLoggingEnabled  bool                        `json:"audit_logging_enabled"`
	MetricsEnabled       bool                        `json:"metrics_enabled"`
	KeyRotationEnabled   bool                        `json:"key_rotation_enabled"`
	EnabledProviders     map[oauth.ProviderType]bool `json:"enabled_providers"`
}

// ShouldInterceptRequest determines if an OAuth request should be intercepted based on feature flags
func (o *OAuthFeatureIntegration) ShouldInterceptRequest(
	ctx context.Context,
	req *oauth.AuthRequest,
	userID uuid.UUID,
	tenantID string,
) (bool, error) {
	evalCtx := o.OAuthEvaluationContext(req, userID, tenantID)

	// Check if OAuth is enabled
	oauthEnabled, err := o.IsOAuthEnabled(ctx, evalCtx)
	if err != nil || !oauthEnabled {
		return false, err
	}

	// Check if the specific server should use OAuth
	evalCtx.ServerName = req.ServerName
	return o.flagManager.EvaluateBooleanFlag(ctx, FlagOAuthEnabled, evalCtx)
}

// ShouldHandle401 determines if a 401 response should be handled automatically
func (o *OAuthFeatureIntegration) ShouldHandle401(
	ctx context.Context,
	req *oauth.AuthRequest,
	userID uuid.UUID,
	tenantID string,
) (bool, error) {
	evalCtx := o.OAuthEvaluationContext(req, userID, tenantID)
	return o.IsAuto401Enabled(ctx, evalCtx)
}

// ShouldUseProvider determines if a specific provider should be used
func (o *OAuthFeatureIntegration) ShouldUseProvider(
	ctx context.Context,
	provider oauth.ProviderType,
	req *oauth.AuthRequest,
	userID uuid.UUID,
	tenantID string,
) (bool, error) {
	evalCtx := o.OAuthEvaluationContext(req, userID, tenantID)
	return o.IsProviderEnabled(ctx, provider, evalCtx)
}

// Enhanced OAuth interceptor that uses feature flags
type FeatureFlaggedOAuthInterceptor struct {
	baseInterceptor oauth.OAuthInterceptor
	integration     *OAuthFeatureIntegration
}

// CreateFeatureFlaggedOAuthInterceptor creates an OAuth interceptor with feature flag integration
func CreateFeatureFlaggedOAuthInterceptor(
	baseInterceptor oauth.OAuthInterceptor,
	integration *OAuthFeatureIntegration,
) *FeatureFlaggedOAuthInterceptor {
	return &FeatureFlaggedOAuthInterceptor{
		baseInterceptor: baseInterceptor,
		integration:     integration,
	}
}

// InterceptRequest intercepts OAuth requests with feature flag checks
func (f *FeatureFlaggedOAuthInterceptor) InterceptRequest(
	ctx context.Context,
	req *oauth.AuthRequest,
) (*oauth.AuthResponse, error) {
	// For this example, we'll use a system user ID
	// In practice, you'd extract this from the request context
	systemUserID := uuid.New()
	tenantID := "" // Could be extracted from request

	// Check if request should be intercepted
	shouldIntercept, err := f.integration.ShouldInterceptRequest(ctx, req, systemUserID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if request should be intercepted: %w", err)
	}

	if !shouldIntercept {
		// OAuth is disabled, return error or pass through
		return &oauth.AuthResponse{
			RequestID:  req.RequestID,
			StatusCode: 501, // Not implemented
			Error:      "OAuth functionality is disabled",
		}, nil
	}

	// Get OAuth configuration
	evalCtx := f.integration.OAuthEvaluationContext(req, systemUserID, tenantID)
	config, err := f.integration.GetOAuthConfiguration(ctx, evalCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth configuration: %w", err)
	}

	// Apply feature-specific logic before delegating to base interceptor
	if config.HTTPSRequired && req.URL != "" {
		// Check if URL is HTTPS
		if len(req.URL) > 8 && req.URL[:8] != "https://" {
			return &oauth.AuthResponse{
				RequestID:  req.RequestID,
				StatusCode: 400,
				Error:      "HTTPS is required for OAuth operations",
			}, nil
		}
	}

	// Delegate to base interceptor
	return f.baseInterceptor.InterceptRequest(ctx, req)
}

// Implement other oauth.OAuthInterceptor methods by delegating to base interceptor

func (f *FeatureFlaggedOAuthInterceptor) GetToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*oauth.TokenData, error) {
	return f.baseInterceptor.GetToken(ctx, serverName, userID)
}

func (f *FeatureFlaggedOAuthInterceptor) RefreshToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) (*oauth.TokenData, error) {
	return f.baseInterceptor.RefreshToken(ctx, serverName, userID)
}

func (f *FeatureFlaggedOAuthInterceptor) StoreToken(
	ctx context.Context,
	token *oauth.TokenData,
) error {
	return f.baseInterceptor.StoreToken(ctx, token)
}

func (f *FeatureFlaggedOAuthInterceptor) RevokeToken(
	ctx context.Context,
	serverName string,
	userID uuid.UUID,
) error {
	return f.baseInterceptor.RevokeToken(ctx, serverName, userID)
}

func (f *FeatureFlaggedOAuthInterceptor) RegisterServer(
	ctx context.Context,
	config *oauth.ServerConfig,
) error {
	return f.baseInterceptor.RegisterServer(ctx, config)
}

func (f *FeatureFlaggedOAuthInterceptor) GetServerConfig(
	ctx context.Context,
	serverName string,
) (*oauth.ServerConfig, error) {
	return f.baseInterceptor.GetServerConfig(ctx, serverName)
}

func (f *FeatureFlaggedOAuthInterceptor) UpdateServerConfig(
	ctx context.Context,
	config *oauth.ServerConfig,
) error {
	return f.baseInterceptor.UpdateServerConfig(ctx, config)
}

func (f *FeatureFlaggedOAuthInterceptor) RemoveServerConfig(
	ctx context.Context,
	serverName string,
) error {
	return f.baseInterceptor.RemoveServerConfig(ctx, serverName)
}

func (f *FeatureFlaggedOAuthInterceptor) Health(ctx context.Context) error {
	return f.baseInterceptor.Health(ctx)
}

func (f *FeatureFlaggedOAuthInterceptor) GetMetrics(
	ctx context.Context,
) (map[string]interface{}, error) {
	return f.baseInterceptor.GetMetrics(ctx)
}

// OAuthFeatureEventHandler handles feature flag events related to OAuth
type OAuthFeatureEventHandler struct {
	oauthInterceptor oauth.OAuthInterceptor
}

// CreateOAuthFeatureEventHandler creates a new OAuth feature event handler
func CreateOAuthFeatureEventHandler(
	oauthInterceptor oauth.OAuthInterceptor,
) *OAuthFeatureEventHandler {
	return &OAuthFeatureEventHandler{
		oauthInterceptor: oauthInterceptor,
	}
}

// OnFlagEvaluated handles flag evaluation events
func (h *OAuthFeatureEventHandler) OnFlagEvaluated(ctx context.Context, eval *FlagEvaluation) {
	// Log OAuth-related flag evaluations for monitoring
	if h.isOAuthFlag(eval.Flag) {
		// You could send this to monitoring systems, audit logs, etc.
		fmt.Printf("OAuth flag evaluated: %s = %v (reason: %s)\n",
			eval.Flag, eval.Value.Enabled, eval.Value.Reason)
	}
}

// OnFlagChanged handles flag change events
func (h *OAuthFeatureEventHandler) OnFlagChanged(
	ctx context.Context,
	flag, oldFlag *FlagDefinition,
) {
	if flag != nil && h.isOAuthFlag(flag.Name) {
		// Handle OAuth flag changes
		if flag.Name == FlagOAuthEnabled {
			// OAuth master switch changed
			if oldFlag != nil && oldFlag.Enabled != flag.Enabled {
				fmt.Printf("OAuth master switch changed: %v -> %v\n", oldFlag.Enabled, flag.Enabled)
			}
		}

		// You could trigger cache invalidation, configuration reloads, etc.
	}
}

// OnConfigurationChanged handles configuration change events
func (h *OAuthFeatureEventHandler) OnConfigurationChanged(
	ctx context.Context,
	config *FlagConfiguration,
) {
	// Handle configuration changes that affect OAuth
	fmt.Printf("Feature flag configuration changed, version: %d\n", config.Version)
}

// OnExperimentStarted handles experiment start events
func (h *OAuthFeatureEventHandler) OnExperimentStarted(
	ctx context.Context,
	experiment *Experiment,
) {
	if h.isOAuthFlag(experiment.Flag) {
		fmt.Printf("OAuth experiment started: %s for flag %s\n", experiment.Name, experiment.Flag)
	}
}

// OnExperimentEnded handles experiment end events
func (h *OAuthFeatureEventHandler) OnExperimentEnded(
	ctx context.Context,
	experiment *Experiment,
	results *ExperimentResults,
) {
	if h.isOAuthFlag(experiment.Flag) {
		fmt.Printf(
			"OAuth experiment ended: %s, winner: %s\n",
			experiment.Name,
			results.SignificantWinner,
		)
	}
}

func (h *OAuthFeatureEventHandler) isOAuthFlag(flag FlagName) bool {
	oauthFlags := map[FlagName]bool{
		FlagOAuthEnabled:           true,
		FlagOAuthAuto401:           true,
		FlagOAuthDCR:               true,
		FlagOAuthProviderGitHub:    true,
		FlagOAuthProviderGoogle:    true,
		FlagOAuthProviderMicrosoft: true,
		FlagOAuthDockerSecrets:     true,
		FlagOAuthTokenRefresh:      true,
		FlagOAuthTokenStorage:      true,
		FlagOAuthJWTValidation:     true,
		FlagOAuthHTTPSRequired:     true,
		FlagOAuthAuditLogging:      true,
		FlagOAuthMetrics:           true,
		FlagOAuthKeyRotation:       true,
	}

	return oauthFlags[flag]
}
