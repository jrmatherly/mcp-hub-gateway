# Feature Flag System for OAuth Rollout

A comprehensive feature flag system designed for gradual OAuth rollout and A/B testing in the MCP Portal.

## Overview

This feature flag system provides:

- **Gradual Rollout**: Percentage-based and scheduled rollouts
- **User/Server Targeting**: Specific targeting for users and servers
- **A/B Testing**: Statistical experiment management
- **Multiple Configuration Sources**: Environment, file, database, HTTP
- **Real-time Metrics**: Performance and usage tracking
- **Circuit Breaker**: Safety mechanisms for production environments
- **OAuth Integration**: Seamless integration with existing OAuth system

## Quick Start

### 1. Initialize the Database

```bash
# Run the database migrations
psql -d mcp_portal -f features/migrations.sql
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/features"
    "github.com/google/uuid"
)

func main() {
    // Create flag manager
    flagManager, err := features.CreateFlagManager(
        store,
        engine,
        cache,
        metrics,
        loader,
        auditor,
    )
    if err != nil {
        panic(err)
    }

    // Create evaluation context
    evalCtx := &features.EvaluationContext{
        UserID:    uuid.New(),
        TenantID:  "tenant-123",
        ServerName: "github-server",
        Environment: "production",
        Timestamp: time.Now(),
    }

    // Check if OAuth is enabled
    enabled, err := flagManager.EvaluateBooleanFlag(
        context.Background(),
        features.FlagOAuthEnabled,
        evalCtx,
    )
    if err != nil {
        log.Printf("Failed to evaluate flag: %v", err)
        return
    }

    if enabled {
        log.Println("OAuth is enabled for this user/server")
    }
}
```

### 3. Environment Configuration

```bash
# OAuth feature flags
export MCP_PORTAL_FEATURE_OAUTH_ENABLED=true
export MCP_PORTAL_FEATURE_OAUTH_ENABLED_ROLLOUT_PERCENTAGE=10
export MCP_PORTAL_FEATURE_OAUTH_PROVIDER_GITHUB=true
export MCP_PORTAL_FEATURE_OAUTH_AUTO_401=false

# Global settings
export MCP_PORTAL_FLAGS_DEFAULT_CACHE_TTL=5m
export MCP_PORTAL_FLAGS_METRICS_ENABLED=true
export MCP_PORTAL_FLAGS_FAILURE_MODE=fail_closed
```

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Flag Manager  │────│ Evaluation      │────│ Configuration   │
│                 │    │ Engine          │    │ Loader          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Metrics         │    │ Cache Provider  │    │ Flag Store      │
│ Collector       │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Database        │    │ Redis Cache     │    │ PostgreSQL      │
│ Metrics         │    │                 │    │ Database        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Evaluation Flow

```
User Request → OAuth Integration → Flag Manager → Evaluation Engine
     ↓                ↓                  ↓              ↓
Evaluation Context → Flag Definition → Rules → Result
     ↓                ↓                  ↓              ↓
Cache Check → User/Server Override → Rollout → Variant Selection
     ↓                ↓                  ↓              ↓
Metrics Collection → Audit Logging → Cache Store → Response
```

## OAuth Feature Flags

### Master Controls

| Flag Name        | Description                           | Default |
| ---------------- | ------------------------------------- | ------- |
| `oauth_enabled`  | Master switch for OAuth functionality | `false` |
| `oauth_auto_401` | Automatic 401 detection and retry     | `false` |
| `oauth_dcr`      | Dynamic Client Registration support   | `false` |

### Provider Flags

| Flag Name                  | Description             | Default |
| -------------------------- | ----------------------- | ------- |
| `oauth_provider_github`    | GitHub OAuth support    | `false` |
| `oauth_provider_google`    | Google OAuth support    | `false` |
| `oauth_provider_microsoft` | Microsoft OAuth support | `false` |

### Security Flags

| Flag Name              | Description             | Default |
| ---------------------- | ----------------------- | ------- |
| `oauth_https_required` | Require HTTPS for OAuth | `true`  |
| `oauth_jwt_validation` | JWT token validation    | `false` |
| `oauth_key_rotation`   | Automatic key rotation  | `false` |

### Infrastructure Flags

| Flag Name              | Description                        | Default |
| ---------------------- | ---------------------------------- | ------- |
| `oauth_docker_secrets` | Docker Desktop secrets integration | `false` |
| `oauth_token_storage`  | Token storage functionality        | `false` |
| `oauth_token_refresh`  | Automatic token refresh            | `false` |
| `oauth_audit_logging`  | OAuth audit logging                | `false` |
| `oauth_metrics`        | OAuth metrics collection           | `false` |

## Configuration Sources

### 1. Environment Variables

```bash
# Flag-specific configuration
MCP_PORTAL_FEATURE_OAUTH_ENABLED=true
MCP_PORTAL_FEATURE_OAUTH_ENABLED_ROLLOUT_PERCENTAGE=25
MCP_PORTAL_FEATURE_OAUTH_ENABLED_USER_OVERRIDES='{"user1":"true","user2":"false"}'

# Global settings
MCP_PORTAL_FLAGS_DEFAULT_ENABLED=false
MCP_PORTAL_FLAGS_DEFAULT_CACHE_TTL=5m
MCP_PORTAL_FLAGS_METRICS_ENABLED=true
```

### 2. YAML Configuration File

```yaml
# portal-flags.yaml
version: 1
environment: production
global_settings:
  default_enabled: false
  default_cache_ttl: 5m
  evaluation_timeout: 1s
  metrics_enabled: true
  failure_mode: fail_closed

flags:
  oauth_enabled:
    name: oauth_enabled
    type: boolean
    description: "Master switch for OAuth functionality"
    enabled: false
    default_value: false
    rollout_percentage: 10
    rules:
      - name: beta_users
        conditions:
          - attribute: user_id
            operator: in
            values: ["user1", "user2", "user3"]
        value: true
        enabled: true
        priority: 100

  oauth_provider_github:
    name: oauth_provider_github
    type: boolean
    description: "GitHub OAuth provider support"
    enabled: true
    default_value: false
    rollout_percentage: 50
```

### 3. Database Configuration

Flags are automatically loaded from the database using the configuration loader. The system supports:

- Hot reloading of configuration
- Version management
- Environment-specific settings
- Audit trails

## Rollout Strategies

### 1. Percentage Rollout

```go
flag := &features.FlagDefinition{
    Name:              features.FlagOAuthEnabled,
    Type:              features.FlagTypeBoolean,
    Enabled:           true,
    RolloutPercentage: 25, // Enable for 25% of users
}
```

### 2. User/Server Targeting

```go
flag := &features.FlagDefinition{
    Name:    features.FlagOAuthProviderGitHub,
    Type:    features.FlagTypeBoolean,
    Enabled: true,
    UserOverrides: map[string]interface{}{
        "user-123": true,
        "user-456": false,
    },
    ServerOverrides: map[string]interface{}{
        "github-server-prod": true,
        "github-server-dev":  false,
    },
}
```

### 3. Rule-Based Targeting

```go
flag := &features.FlagDefinition{
    Name:    features.FlagOAuthDCR,
    Type:    features.FlagTypeBoolean,
    Enabled: true,
    Rules: []features.FlagRule{
        {
            Name:     "enterprise_users",
            Priority: 100,
            Enabled:  true,
            Conditions: []features.FlagCondition{
                {
                    Attribute: "tenant_id",
                    Operator:  features.OpStartsWith,
                    Value:     "enterprise-",
                },
            },
            Value: true,
        },
    },
}
```

### 4. Scheduled Rollout

```go
rolloutConfig := &features.RolloutConfig{
    Strategy:        features.RolloutScheduled,
    StartPercentage: 0,
    EndPercentage:   100,
    ScheduledRollout: &features.ScheduledRollout{
        StartTime: time.Now(),
        EndTime:   &endTime,
        Milestones: []features.RolloutMilestone{
            {Time: time.Now().Add(1 * time.Hour), Percentage: 10},
            {Time: time.Now().Add(6 * time.Hour), Percentage: 50},
            {Time: time.Now().Add(24 * time.Hour), Percentage: 100},
        },
        BusinessHours: &features.BusinessHoursConstraint{
            StartHour: 9,
            EndHour:   17,
            Days:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
            TimeZone:  "UTC",
        },
    },
}
```

## A/B Testing

### Creating an Experiment

```go
experiment := &features.Experiment{
    ID:               "oauth-provider-test",
    Name:             "GitHub OAuth Provider Test",
    Description:      "Test GitHub OAuth vs Google OAuth performance",
    Flag:             features.FlagOAuthProviderGitHub,
    Status:           "draft",
    TrafficAllocation: 50, // 50% of users in experiment
    Variants: []features.FlagVariant{
        {
            Name:    "github",
            Weight:  50,
            Value:   true,
            Enabled: true,
        },
        {
            Name:    "google",
            Weight:  50,
            Value:   false,
            Enabled: true,
        },
    },
    PrimaryMetric:    "conversion_rate",
    SecondaryMetrics: []string{"response_time", "error_rate"},
    StartTime:        time.Now(),
    Duration:         &[]time.Duration{time.Hour * 24 * 7}[0], // 1 week
}

// Create and start experiment
err := experimentManager.CreateExperiment(ctx, experiment)
if err != nil {
    return err
}

err = experimentManager.StartExperiment(ctx, experiment.ID)
if err != nil {
    return err
}
```

### Analyzing Results

```go
// Calculate experiment results
results, err := experimentManager.CalculateResults(ctx, "oauth-provider-test")
if err != nil {
    return err
}

fmt.Printf("Total participants: %d\n", results.TotalParticipants)
fmt.Printf("Significant winner: %s\n", results.SignificantWinner)
fmt.Printf("Confidence level: %.2f%%\n", results.ConfidenceLevel)
fmt.Printf("P-value: %.4f\n", results.PValue)

// Export results
csvData, err := experimentManager.ExportResults(ctx, "oauth-provider-test", "csv")
if err != nil {
    return err
}
```

## Integration Patterns

### 1. OAuth Interceptor Integration

```go
// Create feature flag integration
integration, err := features.CreateOAuthFeatureIntegration(flagManager, oauthInterceptor)
if err != nil {
    return err
}

// Create feature-flagged OAuth interceptor
flaggedInterceptor := features.CreateFeatureFlaggedOAuthInterceptor(
    baseOAuthInterceptor,
    integration,
)

// Use in your request handling
func handleOAuthRequest(req *oauth.AuthRequest) (*oauth.AuthResponse, error) {
    return flaggedInterceptor.InterceptRequest(ctx, req)
}
```

### 2. Middleware Integration

```go
func FeatureFlagMiddleware(flagManager features.FlagManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := getUserID(r) // Extract from auth
            evalCtx := &features.EvaluationContext{
                UserID:     userID,
                RemoteAddr: r.RemoteAddr,
                UserAgent:  r.UserAgent(),
                Headers:    extractHeaders(r),
                Timestamp:  time.Now(),
            }

            // Check if OAuth is enabled
            oauthEnabled, err := flagManager.EvaluateBooleanFlag(
                r.Context(),
                features.FlagOAuthEnabled,
                evalCtx,
            )
            if err != nil {
                http.Error(w, "Flag evaluation failed", 500)
                return
            }

            // Add to request context
            ctx := context.WithValue(r.Context(), "oauth_enabled", oauthEnabled)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## Monitoring and Metrics

### Flag Metrics

```go
// Get global metrics
metrics, err := flagManager.GetMetrics(ctx)
if err != nil {
    return err
}

fmt.Printf("Total evaluations: %d\n", metrics.TotalEvaluations)
fmt.Printf("Cache hit rate: %.2f%%\n", metrics.CacheHitRate)
fmt.Printf("Error rate: %.2f%%\n", metrics.ErrorRate)

// Get flag-specific metrics
flagMetrics, err := flagManager.GetFlagMetrics(ctx, features.FlagOAuthEnabled)
if err != nil {
    return err
}

fmt.Printf("True rate: %.2f%%\n", flagMetrics.TrueRate)
fmt.Printf("Average evaluation time: %v\n", flagMetrics.AverageEvaluationTime)
```

### Performance Monitoring

The system automatically tracks:

- **Evaluation Performance**: Response times, cache hit rates
- **Flag Usage**: Evaluation counts, true/false rates
- **Error Rates**: Failed evaluations, timeout rates
- **Rollout Progress**: Percentage adoption over time
- **Experiment Metrics**: Participant counts, conversion rates

### Dashboards and Alerting

Metrics are stored in both Redis (real-time) and PostgreSQL (historical) for:

- Grafana dashboards
- Prometheus alerts
- Custom monitoring integrations

## Security Considerations

### 1. Failsafe Modes

```go
// Configure failsafe behavior
globalSettings := &features.GlobalFlagSettings{
    FailureMode:           "fail_closed", // or "fail_open"
    MaxEvaluationTime:     time.Second * 2,
    CircuitBreakerEnabled: true,
}
```

### 2. Access Controls

- Database-level permissions for flag management
- API authentication for flag operations
- Audit logging for all changes
- Environment segregation

### 3. Data Privacy

- User ID hashing for consistent bucketing
- No PII storage in evaluation logs
- Configurable data retention policies
- GDPR compliance support

## Production Deployment

### 1. Database Setup

```bash
# Run migrations
psql -d mcp_portal -f features/migrations.sql

# Verify tables
psql -d mcp_portal -c "\dt feature_*"
```

### 2. Environment Configuration

```bash
# Production settings
export MCP_PORTAL_ENV=production
export MCP_PORTAL_FLAGS_FAILURE_MODE=fail_closed
export MCP_PORTAL_FLAGS_CIRCUIT_BREAKER_ENABLED=true
export MCP_PORTAL_FLAGS_METRICS_ENABLED=true
```

### 3. Health Checks

```go
// Add to your health check endpoint
func healthCheck(flagManager features.FlagManager) {
    if err := flagManager.Health(ctx); err != nil {
        log.Printf("Feature flag system unhealthy: %v", err)
        // Handle unhealthy state
    }
}
```

### 4. Scaling Considerations

- **Redis Clustering**: For high-availability caching
- **Database Read Replicas**: For high-read workloads
- **Horizontal Scaling**: Multiple app instances share cache
- **Circuit Breakers**: Prevent cascading failures

## Testing

### Unit Testing

```go
func TestOAuthFlagEvaluation(t *testing.T) {
    // Create test flag manager
    flagManager := createTestFlagManager()

    evalCtx := &features.EvaluationContext{
        UserID:    uuid.New(),
        TenantID:  "test-tenant",
        Timestamp: time.Now(),
    }

    // Test OAuth enabled flag
    enabled, err := flagManager.EvaluateBooleanFlag(
        context.Background(),
        features.FlagOAuthEnabled,
        evalCtx,
    )

    assert.NoError(t, err)
    assert.False(t, enabled) // Should be false by default
}
```

### Integration Testing

```go
func TestOAuthIntegration(t *testing.T) {
    // Test the full OAuth + feature flag integration
    integration := createTestIntegration()

    req := &oauth.AuthRequest{
        ServerName: "test-server",
        UserID:     uuid.New(),
    }

    shouldIntercept, err := integration.ShouldInterceptRequest(
        context.Background(),
        req,
        req.UserID,
        "test-tenant",
    )

    assert.NoError(t, err)
    assert.False(t, shouldIntercept) // OAuth disabled by default
}
```

### Load Testing

```bash
# Test flag evaluation performance
go test -bench=BenchmarkFlagEvaluation -benchmem
```

## Troubleshooting

### Common Issues

1. **High Evaluation Latency**

   - Check cache hit rates
   - Verify database connection pool
   - Review rule complexity

2. **Inconsistent Flag Values**

   - Check cache TTL settings
   - Verify configuration reload
   - Review user ID hashing

3. **Circuit Breaker Triggered**
   - Check error rates
   - Verify database connectivity
   - Review timeout settings

### Debug Mode

```bash
# Enable debug logging
export MCP_PORTAL_FLAGS_DEBUG=true
```

### Metrics and Logs

```go
// Check flag metrics
metrics, _ := flagManager.GetMetrics(ctx)
log.Printf("Cache hit rate: %.2f%%", metrics.CacheHitRate)
log.Printf("Error rate: %.2f%%", metrics.ErrorRate)

// Check specific flag
flagMetrics, _ := flagManager.GetFlagMetrics(ctx, features.FlagOAuthEnabled)
log.Printf("Evaluations: %d", flagMetrics.TotalEvaluations)
log.Printf("True rate: %.2f%%", flagMetrics.TrueRate)
```

## API Reference

### Flag Manager Interface

```go
type FlagManager interface {
    // Flag evaluation
    EvaluateFlag(ctx context.Context, flag FlagName, evalCtx *EvaluationContext) (*FlagValue, error)
    EvaluateAllFlags(ctx context.Context, evalCtx *EvaluationContext) (map[FlagName]*FlagValue, error)
    EvaluateBooleanFlag(ctx context.Context, flag FlagName, evalCtx *EvaluationContext) (bool, error)

    // Flag management
    GetFlag(ctx context.Context, name FlagName) (*FlagDefinition, error)
    CreateFlag(ctx context.Context, flag *FlagDefinition) error
    UpdateFlag(ctx context.Context, flag *FlagDefinition) error
    DeleteFlag(ctx context.Context, name FlagName) error
    ListFlags(ctx context.Context) ([]*FlagDefinition, error)

    // Configuration management
    LoadConfiguration(ctx context.Context, source ConfigurationSource, location string) error
    ReloadConfiguration(ctx context.Context) error
    GetConfiguration(ctx context.Context) (*FlagConfiguration, error)

    // Metrics and monitoring
    GetMetrics(ctx context.Context) (*FlagMetrics, error)
    GetFlagMetrics(ctx context.Context, flag FlagName) (*FlagMetric, error)

    // Health check
    Health(ctx context.Context) error
}
```

### OAuth Integration Interface

```go
type OAuthFeatureIntegration interface {
    IsOAuthEnabled(ctx context.Context, evalCtx *EvaluationContext) (bool, error)
    IsProviderEnabled(ctx context.Context, provider oauth.ProviderType, evalCtx *EvaluationContext) (bool, error)
    GetOAuthConfiguration(ctx context.Context, evalCtx *EvaluationContext) (*OAuthFeatureConfig, error)
    ShouldInterceptRequest(ctx context.Context, req *oauth.AuthRequest, userID uuid.UUID, tenantID string) (bool, error)
}
```

## Contributing

### Adding New OAuth Flags

1. Add the flag constant to `types.go`:

   ```go
   const FlagOAuthNewFeature FlagName = "oauth_new_feature"
   ```

2. Add the flag to the default configuration in `loader.go`

3. Add integration methods to `oauth_integration.go`

4. Update the database migrations

5. Add tests and documentation

### Extending Evaluation Logic

1. Add new operators to `ConditionOperator` in `types.go`
2. Implement the operator logic in `engine.go`
3. Add validation logic
4. Write comprehensive tests

For more details, see the [Contributing Guide](../CONTRIBUTING.md).

## License

This feature flag system is part of the MCP Portal project. See [LICENSE](../LICENSE) for details.
