package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// flagManager implements the FlagManager interface
type flagManager struct {
	// Dependencies
	store     FlagStore
	engine    EvaluationEngine
	cache     CacheProvider
	metrics   MetricsCollector
	loader    ConfigurationLoader
	auditor   audit.Logger

	// Configuration
	config      *FlagConfiguration
	configMu    sync.RWMutex
	lastReload  time.Time
	autoReload  bool
	reloadInterval time.Duration

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Event handlers
	handlers []EventHandler
	handlersMu sync.RWMutex

	// Circuit breaker for evaluation safety
	circuitBreaker *CircuitBreaker
}

// CircuitBreaker provides safety mechanism for flag evaluation
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures         int
	lastFailureTime  time.Time
	state           string // "closed", "open", "half-open"
	mu              sync.RWMutex
}

// CreateFlagManager creates a new feature flag manager
func CreateFlagManager(
	store FlagStore,
	engine EvaluationEngine,
	cacheProvider CacheProvider,
	metricsCollector MetricsCollector,
	configLoader ConfigurationLoader,
	auditor audit.Logger,
) (FlagManager, error) {
	if store == nil {
		return nil, fmt.Errorf("flag store is required")
	}
	if engine == nil {
		return nil, fmt.Errorf("evaluation engine is required")
	}
	if cacheProvider == nil {
		return nil, fmt.Errorf("cache provider is required")
	}
	if metricsCollector == nil {
		return nil, fmt.Errorf("metrics collector is required")
	}
	if configLoader == nil {
		return nil, fmt.Errorf("configuration loader is required")
	}
	if auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}

	manager := &flagManager{
		store:          store,
		engine:         engine,
		cache:          cacheProvider,
		metrics:        metricsCollector,
		loader:         configLoader,
		auditor:        auditor,
		autoReload:     true,
		reloadInterval: time.Minute * 5,
		stopChan:       make(chan struct{}),
		handlers:       make([]EventHandler, 0),
		circuitBreaker: &CircuitBreaker{
			failureThreshold: 10,
			resetTimeout:     time.Minute * 2,
			state:           "closed",
		},
	}

	// Load initial configuration
	if err := manager.loadInitialConfiguration(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}

	// Start background workers
	manager.startBackgroundWorkers()

	return manager, nil
}

// EvaluateFlag evaluates a single feature flag
func (m *flagManager) EvaluateFlag(
	ctx context.Context,
	flag FlagName,
	evalCtx *EvaluationContext,
) (*FlagValue, error) {
	startTime := time.Now()
	evaluationID := uuid.New().String()

	// Check circuit breaker
	if !m.circuitBreaker.CanEvaluate() {
		return m.getFailsafeValue(flag), fmt.Errorf("circuit breaker is open")
	}

	// Try cache first
	cacheKey := m.buildCacheKey(flag, evalCtx)
	if cachedValue, err := m.cache.Get(ctx, cacheKey); err == nil {
		cachedValue.CacheHit = true
		m.recordEvaluation(ctx, flag, cachedValue, evalCtx, evaluationID, time.Since(startTime), true)
		return cachedValue, nil
	}

	// Get flag definition
	flagDef, err := m.getFlag(ctx, flag)
	if err != nil {
		m.circuitBreaker.RecordFailure()
		m.metrics.RecordError(ctx, flag, err)
		return m.getFailsafeValue(flag), fmt.Errorf("failed to get flag definition: %w", err)
	}

	// Evaluate flag
	value, err := m.engine.Evaluate(ctx, flagDef, evalCtx)
	if err != nil {
		m.circuitBreaker.RecordFailure()
		m.metrics.RecordError(ctx, flag, err)
		return m.getFailsafeValue(flag), fmt.Errorf("failed to evaluate flag: %w", err)
	}

	// Cache the result
	cacheTTL := m.getCacheTTL(flagDef)
	if cacheErr := m.cache.Set(ctx, cacheKey, value, cacheTTL); cacheErr != nil {
		// Log cache error but don't fail evaluation
		m.auditor.Log(ctx, audit.ActionUpdate, "flag_cache", cacheKey, "system", map[string]any{
			"error": cacheErr.Error(),
		})
	}

	// Record successful evaluation
	m.circuitBreaker.RecordSuccess()
	duration := time.Since(startTime)
	m.recordEvaluation(ctx, flag, value, evalCtx, evaluationID, duration, false)

	// Trigger events
	m.notifyFlagEvaluated(ctx, &FlagEvaluation{
		Flag:         flag,
		Value:        *value,
		Context:      *evalCtx,
		EvaluationID: evaluationID,
		Duration:     duration,
		CacheHit:     false,
	})

	return value, nil
}

// EvaluateAllFlags evaluates all flags for the given context
func (m *flagManager) EvaluateAllFlags(
	ctx context.Context,
	evalCtx *EvaluationContext,
) (map[FlagName]*FlagValue, error) {
	m.configMu.RLock()
	flags := make(map[FlagName]*FlagDefinition)
	for name, flag := range m.config.Flags {
		flags[name] = flag
	}
	m.configMu.RUnlock()

	results := make(map[FlagName]*FlagValue)
	var errors []error

	// Evaluate flags in parallel with limited concurrency
	semaphore := make(chan struct{}, 10) // Limit to 10 concurrent evaluations
	var wg sync.WaitGroup
	var mu sync.Mutex

	for flagName := range flags {
		wg.Add(1)
		go func(name FlagName) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			value, err := m.EvaluateFlag(ctx, name, evalCtx)
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("flag %s: %w", name, err))
			} else {
				results[name] = value
			}
			mu.Unlock()
		}(flagName)
	}

	wg.Wait()

	if len(errors) > 0 {
		// Return partial results with error
		return results, fmt.Errorf("failed to evaluate %d flags: %v", len(errors), errors[0])
	}

	return results, nil
}

// EvaluateBooleanFlag evaluates a flag and returns boolean result
func (m *flagManager) EvaluateBooleanFlag(
	ctx context.Context,
	flag FlagName,
	evalCtx *EvaluationContext,
) (bool, error) {
	value, err := m.EvaluateFlag(ctx, flag, evalCtx)
	if err != nil {
		return false, err
	}
	return value.Enabled, nil
}

// GetFlag retrieves a flag definition
func (m *flagManager) GetFlag(ctx context.Context, name FlagName) (*FlagDefinition, error) {
	return m.getFlag(ctx, name)
}

// CreateFlag creates a new flag definition
func (m *flagManager) CreateFlag(ctx context.Context, flag *FlagDefinition) error {
	if flag == nil {
		return fmt.Errorf("flag is required")
	}

	// Validate flag
	if err := m.validateFlag(flag); err != nil {
		return fmt.Errorf("invalid flag: %w", err)
	}

	// Set metadata
	now := time.Now()
	flag.CreatedAt = now
	flag.UpdatedAt = now
	flag.Version = 1

	// Save to store
	if err := m.store.SaveFlag(ctx, flag); err != nil {
		return fmt.Errorf("failed to save flag: %w", err)
	}

	// Update local configuration
	m.configMu.Lock()
	if m.config.Flags == nil {
		m.config.Flags = make(map[FlagName]*FlagDefinition)
	}
	m.config.Flags[flag.Name] = flag
	m.configMu.Unlock()

	// Invalidate cache
	m.invalidateFlagCache(ctx, flag.Name)

	// Audit log
	m.auditor.Log(ctx, audit.ActionCreate, "feature_flag", string(flag.Name), "system", map[string]any{
		"type":        flag.Type,
		"enabled":     flag.Enabled,
		"description": flag.Description,
	})

	// Trigger events
	m.notifyFlagChanged(ctx, flag, nil)

	return nil
}

// UpdateFlag updates an existing flag definition
func (m *flagManager) UpdateFlag(ctx context.Context, flag *FlagDefinition) error {
	if flag == nil {
		return fmt.Errorf("flag is required")
	}

	// Get existing flag
	oldFlag, err := m.getFlag(ctx, flag.Name)
	if err != nil {
		return fmt.Errorf("flag not found: %w", err)
	}

	// Validate flag
	if err := m.validateFlag(flag); err != nil {
		return fmt.Errorf("invalid flag: %w", err)
	}

	// Update metadata
	flag.UpdatedAt = time.Now()
	flag.Version = oldFlag.Version + 1
	flag.CreatedAt = oldFlag.CreatedAt
	flag.CreatedBy = oldFlag.CreatedBy

	// Save to store
	if err := m.store.SaveFlag(ctx, flag); err != nil {
		return fmt.Errorf("failed to save flag: %w", err)
	}

	// Update local configuration
	m.configMu.Lock()
	m.config.Flags[flag.Name] = flag
	m.configMu.Unlock()

	// Invalidate cache
	m.invalidateFlagCache(ctx, flag.Name)

	// Audit log
	m.auditor.Log(ctx, audit.ActionUpdate, "feature_flag", string(flag.Name), "system", map[string]any{
		"old_enabled": oldFlag.Enabled,
		"new_enabled": flag.Enabled,
		"version":     flag.Version,
	})

	// Trigger events
	m.notifyFlagChanged(ctx, flag, oldFlag)

	return nil
}

// DeleteFlag deletes a flag definition
func (m *flagManager) DeleteFlag(ctx context.Context, name FlagName) error {
	// Get existing flag for audit
	oldFlag, err := m.getFlag(ctx, name)
	if err != nil {
		return fmt.Errorf("flag not found: %w", err)
	}

	// Delete from store
	if err := m.store.DeleteFlag(ctx, name); err != nil {
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	// Remove from local configuration
	m.configMu.Lock()
	delete(m.config.Flags, name)
	m.configMu.Unlock()

	// Invalidate cache
	m.invalidateFlagCache(ctx, name)

	// Audit log
	m.auditor.Log(ctx, audit.ActionDelete, "feature_flag", string(name), "system", map[string]any{
		"type":    oldFlag.Type,
		"enabled": oldFlag.Enabled,
	})

	return nil
}

// ListFlags returns all flag definitions
func (m *flagManager) ListFlags(ctx context.Context) ([]*FlagDefinition, error) {
	m.configMu.RLock()
	defer m.configMu.RUnlock()

	flags := make([]*FlagDefinition, 0, len(m.config.Flags))
	for _, flag := range m.config.Flags {
		flags = append(flags, flag)
	}

	return flags, nil
}

// LoadConfiguration loads configuration from specified source
func (m *flagManager) LoadConfiguration(
	ctx context.Context,
	source ConfigurationSource,
	location string,
) error {
	var config *FlagConfiguration
	var err error

	switch source {
	case SourceFile:
		config, err = m.loader.LoadFromFile(ctx, location)
	case SourceEnvironment:
		config, err = m.loader.LoadFromEnvironment(ctx)
	case SourceDatabase:
		config, err = m.loader.LoadFromDatabase(ctx)
	case SourceHTTP:
		config, err = m.loader.LoadFromHTTP(ctx, location)
	default:
		return fmt.Errorf("unsupported configuration source: %s", source)
	}

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := m.ValidateConfiguration(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update configuration
	m.configMu.Lock()
	oldConfig := m.config
	m.config = config
	m.lastReload = time.Now()
	m.configMu.Unlock()

	// Clear cache after configuration change
	m.cache.Clear(ctx)

	// Audit log
	m.auditor.Log(ctx, audit.ActionUpdate, "flag_configuration", "config", "system", map[string]any{
		"source":     source,
		"location":   location,
		"flag_count": len(config.Flags),
		"version":    config.Version,
	})

	// Trigger events
	m.notifyConfigurationChanged(ctx, config)

	// If we had an old config, check for flag changes
	if oldConfig != nil {
		m.detectAndNotifyFlagChanges(ctx, oldConfig, config)
	}

	return nil
}

// ReloadConfiguration reloads the current configuration
func (m *flagManager) ReloadConfiguration(ctx context.Context) error {
	// Reload from store
	config, err := m.store.GetConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Validate configuration
	if err := m.ValidateConfiguration(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update configuration
	m.configMu.Lock()
	oldConfig := m.config
	m.config = config
	m.lastReload = time.Now()
	m.configMu.Unlock()

	// Clear cache
	m.cache.Clear(ctx)

	// Trigger events
	m.notifyConfigurationChanged(ctx, config)
	if oldConfig != nil {
		m.detectAndNotifyFlagChanges(ctx, oldConfig, config)
	}

	return nil
}

// GetConfiguration returns the current configuration
func (m *flagManager) GetConfiguration(ctx context.Context) (*FlagConfiguration, error) {
	m.configMu.RLock()
	defer m.configMu.RUnlock()

	// Return deep copy to prevent modification
	return m.copyConfiguration(m.config), nil
}

// ValidateConfiguration validates a flag configuration
func (m *flagManager) ValidateConfiguration(config *FlagConfiguration) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	var errors []string

	// Validate flags
	for name, flag := range config.Flags {
		if flag.Name != name {
			errors = append(errors, fmt.Sprintf("flag %s: name mismatch", name))
		}

		if err := m.validateFlag(flag); err != nil {
			errors = append(errors, fmt.Sprintf("flag %s: %v", name, err))
		}
	}

	// Validate experiments
	for id, experiment := range config.Experiments {
		if experiment.ID != id {
			errors = append(errors, fmt.Sprintf("experiment %s: ID mismatch", id))
		}

		if err := m.validateExperiment(experiment); err != nil {
			errors = append(errors, fmt.Sprintf("experiment %s: %v", id, err))
		}
	}

	if len(errors) > 0 {
		config.Valid = false
		config.Errors = errors
		return fmt.Errorf("configuration validation failed: %v", errors)
	}

	config.Valid = true
	config.Errors = nil
	return nil
}

// GetMetrics returns current metrics
func (m *flagManager) GetMetrics(ctx context.Context) (*FlagMetrics, error) {
	return m.metrics.GetMetrics(ctx)
}

// GetFlagMetrics returns metrics for a specific flag
func (m *flagManager) GetFlagMetrics(ctx context.Context, flag FlagName) (*FlagMetric, error) {
	return m.metrics.GetFlagMetrics(ctx, flag)
}

// ResetMetrics resets all metrics
func (m *flagManager) ResetMetrics(ctx context.Context) error {
	return m.metrics.Reset(ctx)
}

// InvalidateCache invalidates cache for a specific flag
func (m *flagManager) InvalidateCache(ctx context.Context, flag FlagName) error {
	return m.invalidateFlagCache(ctx, flag)
}

// ClearCache clears the entire cache
func (m *flagManager) ClearCache(ctx context.Context) error {
	return m.cache.Clear(ctx)
}

// Health checks the health of the flag manager
func (m *flagManager) Health(ctx context.Context) error {
	// Check store health
	if err := m.store.Health(ctx); err != nil {
		return fmt.Errorf("store health check failed: %w", err)
	}

	// Check cache stats
	if _, err := m.cache.Stats(ctx); err != nil {
		return fmt.Errorf("cache health check failed: %w", err)
	}

	// Check configuration validity
	m.configMu.RLock()
	configValid := m.config != nil && m.config.Valid
	m.configMu.RUnlock()

	if !configValid {
		return fmt.Errorf("configuration is invalid")
	}

	// Check circuit breaker state
	if m.circuitBreaker.state == "open" {
		return fmt.Errorf("circuit breaker is open")
	}

	return nil
}

// Helper methods

func (m *flagManager) getFlag(ctx context.Context, name FlagName) (*FlagDefinition, error) {
	m.configMu.RLock()
	flag, exists := m.config.Flags[name]
	m.configMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("flag %s not found", name)
	}

	return flag, nil
}

func (m *flagManager) validateFlag(flag *FlagDefinition) error {
	if flag.Name == "" {
		return fmt.Errorf("flag name is required")
	}

	if flag.Type == "" {
		return fmt.Errorf("flag type is required")
	}

	// Validate rollout percentage
	if flag.RolloutPercentage < 0 || flag.RolloutPercentage > 100 {
		return fmt.Errorf("rollout percentage must be between 0 and 100")
	}

	// Validate rules
	for i, rule := range flag.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}

		for j, condition := range rule.Conditions {
			if condition.Attribute == "" {
				return fmt.Errorf("rule %d, condition %d: attribute is required", i, j)
			}
			if condition.Operator == "" {
				return fmt.Errorf("rule %d, condition %d: operator is required", i, j)
			}
		}
	}

	// Validate variants
	totalWeight := 0
	for i, variant := range flag.Variants {
		if variant.Name == "" {
			return fmt.Errorf("variant %d: name is required", i)
		}
		if variant.Weight < 0 || variant.Weight > 100 {
			return fmt.Errorf("variant %d: weight must be between 0 and 100", i)
		}
		totalWeight += variant.Weight
	}

	if len(flag.Variants) > 0 && totalWeight != 100 {
		return fmt.Errorf("total variant weight must equal 100, got %d", totalWeight)
	}

	return nil
}

func (m *flagManager) validateExperiment(experiment *Experiment) error {
	if experiment.ID == "" {
		return fmt.Errorf("experiment ID is required")
	}

	if experiment.Name == "" {
		return fmt.Errorf("experiment name is required")
	}

	if experiment.Flag == "" {
		return fmt.Errorf("experiment flag is required")
	}

	// Validate that the referenced flag exists
	m.configMu.RLock()
	_, exists := m.config.Flags[experiment.Flag]
	m.configMu.RUnlock()

	if !exists {
		return fmt.Errorf("experiment references non-existent flag: %s", experiment.Flag)
	}

	if experiment.TrafficAllocation < 0 || experiment.TrafficAllocation > 100 {
		return fmt.Errorf("traffic allocation must be between 0 and 100")
	}

	// Validate variants
	totalWeight := 0
	for i, variant := range experiment.Variants {
		if variant.Name == "" {
			return fmt.Errorf("variant %d: name is required", i)
		}
		if variant.Weight < 0 || variant.Weight > 100 {
			return fmt.Errorf("variant %d: weight must be between 0 and 100", i)
		}
		totalWeight += variant.Weight
	}

	if len(experiment.Variants) > 0 && totalWeight != 100 {
		return fmt.Errorf("total variant weight must equal 100, got %d", totalWeight)
	}

	return nil
}

func (m *flagManager) buildCacheKey(flag FlagName, evalCtx *EvaluationContext) string {
	// Create a deterministic cache key based on flag name and context
	key := fmt.Sprintf("flag:%s:user:%s", flag, evalCtx.UserID.String())

	if evalCtx.ServerName != "" {
		key += fmt.Sprintf(":server:%s", evalCtx.ServerName)
	}

	if evalCtx.TenantID != "" {
		key += fmt.Sprintf(":tenant:%s", evalCtx.TenantID)
	}

	return key
}

func (m *flagManager) getCacheTTL(flag *FlagDefinition) time.Duration {
	// Default TTL
	defaultTTL := time.Minute * 5

	m.configMu.RLock()
	if m.config.GlobalSettings != nil && m.config.GlobalSettings.DefaultCacheTTL > 0 {
		defaultTTL = m.config.GlobalSettings.DefaultCacheTTL
	}
	m.configMu.RUnlock()

	return defaultTTL
}

func (m *flagManager) getFailsafeValue(flag FlagName) *FlagValue {
	// Return safe default value when evaluation fails
	failureMode := "fail_closed" // Default to conservative approach

	m.configMu.RLock()
	if m.config.GlobalSettings != nil && m.config.GlobalSettings.FailureMode != "" {
		failureMode = m.config.GlobalSettings.FailureMode
	}
	m.configMu.RUnlock()

	enabled := failureMode == "fail_open"

	return &FlagValue{
		Name:        flag,
		Type:        FlagTypeBoolean,
		Enabled:     enabled,
		Value:       enabled,
		Reason:      "failsafe_mode",
		EvaluatedAt: time.Now(),
		CacheHit:    false,
	}
}

func (m *flagManager) invalidateFlagCache(ctx context.Context, flag FlagName) error {
	// Build pattern to match all cache keys for this flag
	pattern := fmt.Sprintf("flag:%s:*", flag)

	// Note: This is a simplified implementation
	// In practice, you might need to implement pattern-based deletion
	// or store cache keys to iterate over them
	return m.cache.Delete(ctx, pattern)
}

func (m *flagManager) recordEvaluation(
	ctx context.Context,
	flag FlagName,
	value *FlagValue,
	evalCtx *EvaluationContext,
	evaluationID string,
	duration time.Duration,
	cacheHit bool,
) {
	evaluation := &FlagEvaluation{
		Flag:         flag,
		Value:        *value,
		Context:      *evalCtx,
		EvaluationID: evaluationID,
		Duration:     duration,
		CacheHit:     cacheHit,
	}

	m.metrics.RecordEvaluation(ctx, evaluation)
}

func (m *flagManager) loadInitialConfiguration(ctx context.Context) error {
	// Try to load from store first
	config, err := m.store.GetConfiguration(ctx)
	if err != nil {
		// If store fails, create default configuration
		config = m.createDefaultConfiguration()
	}

	// Validate configuration
	if err := m.ValidateConfiguration(config); err != nil {
		return fmt.Errorf("invalid initial configuration: %w", err)
	}

	m.config = config
	m.lastReload = time.Now()

	return nil
}

func (m *flagManager) createDefaultConfiguration() *FlagConfiguration {
	now := time.Now()

	return &FlagConfiguration{
		Version:     1,
		LoadedAt:    now,
		LoadedFrom:  SourceDatabase,
		Environment: "development",
		GlobalSettings: &GlobalFlagSettings{
			DefaultEnabled:           false,
			DefaultCacheTTL:          time.Minute * 5,
			EvaluationTimeout:        time.Second * 1,
			DefaultRolloutPercentage: 0,
			DefaultRolloutStrategy:   RolloutPercentage,
			MetricsEnabled:          true,
			MetricsInterval:         time.Minute * 1,
			TrackingEnabled:         true,
			FailureMode:             "fail_closed",
			MaxEvaluationTime:       time.Second * 2,
			CircuitBreakerEnabled:   true,
		},
		Flags:       m.createOAuthFlags(),
		Groups:      make(map[string]*FlagGroup),
		Experiments: make(map[string]*Experiment),
		Valid:       true,
	}
}

func (m *flagManager) createOAuthFlags() map[FlagName]*FlagDefinition {
	now := time.Now()
	flags := make(map[FlagName]*FlagDefinition)

	// OAuth master flag
	flags[FlagOAuthEnabled] = &FlagDefinition{
		Name:              FlagOAuthEnabled,
		Type:              FlagTypeBoolean,
		Description:       "Master switch for OAuth functionality",
		Enabled:           false,
		DefaultValue:      false,
		RolloutPercentage: 0,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           1,
		Tags:              []string{"oauth", "security"},
	}

	// OAuth auto 401 handling
	flags[FlagOAuthAuto401] = &FlagDefinition{
		Name:              FlagOAuthAuto401,
		Type:              FlagTypeBoolean,
		Description:       "Automatic 401 detection and token refresh",
		Enabled:           false,
		DefaultValue:      false,
		RolloutPercentage: 0,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           1,
		Tags:              []string{"oauth", "automation"},
	}

	// Dynamic Client Registration
	flags[FlagOAuthDCR] = &FlagDefinition{
		Name:              FlagOAuthDCR,
		Type:              FlagTypeBoolean,
		Description:       "Dynamic Client Registration support",
		Enabled:           false,
		DefaultValue:      false,
		RolloutPercentage: 0,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           1,
		Tags:              []string{"oauth", "dcr"},
	}

	// Provider flags
	providerFlags := []struct {
		name        FlagName
		description string
	}{
		{FlagOAuthProviderGitHub, "GitHub OAuth provider support"},
		{FlagOAuthProviderGoogle, "Google OAuth provider support"},
		{FlagOAuthProviderMicrosoft, "Microsoft OAuth provider support"},
	}

	for _, pf := range providerFlags {
		flags[pf.name] = &FlagDefinition{
			Name:              pf.name,
			Type:              FlagTypeBoolean,
			Description:       pf.description,
			Enabled:           false,
			DefaultValue:      false,
			RolloutPercentage: 0,
			CreatedAt:         now,
			UpdatedAt:         now,
			Version:           1,
			Tags:              []string{"oauth", "provider"},
		}
	}

	return flags
}

func (m *flagManager) copyConfiguration(config *FlagConfiguration) *FlagConfiguration {
	// In a real implementation, you would do a deep copy
	// For now, we'll return the same reference with a note
	// that this should be implemented properly
	return config
}

func (m *flagManager) startBackgroundWorkers() {
	// Configuration reload worker
	if m.autoReload {
		m.wg.Add(1)
		go m.configReloadWorker()
	}

	// Metrics flush worker
	m.wg.Add(1)
	go m.metricsWorker()
}

func (m *flagManager) configReloadWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			if err := m.ReloadConfiguration(ctx); err != nil {
				m.auditor.Log(ctx, audit.ActionUpdate, "flag_configuration", "reload", "system", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}
}

func (m *flagManager) metricsWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			// Metrics are handled by the MetricsCollector
			// This worker could be used for periodic cleanup or aggregation
		}
	}
}

// Event handling methods

func (m *flagManager) AddEventHandler(handler EventHandler) {
	m.handlersMu.Lock()
	defer m.handlersMu.Unlock()
	m.handlers = append(m.handlers, handler)
}

func (m *flagManager) notifyFlagEvaluated(ctx context.Context, eval *FlagEvaluation) {
	m.handlersMu.RLock()
	handlers := make([]EventHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.handlersMu.RUnlock()

	for _, handler := range handlers {
		go handler.OnFlagEvaluated(ctx, eval)
	}
}

func (m *flagManager) notifyFlagChanged(ctx context.Context, flag, oldFlag *FlagDefinition) {
	m.handlersMu.RLock()
	handlers := make([]EventHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.handlersMu.RUnlock()

	for _, handler := range handlers {
		go handler.OnFlagChanged(ctx, flag, oldFlag)
	}
}

func (m *flagManager) notifyConfigurationChanged(ctx context.Context, config *FlagConfiguration) {
	m.handlersMu.RLock()
	handlers := make([]EventHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.handlersMu.RUnlock()

	for _, handler := range handlers {
		go handler.OnConfigurationChanged(ctx, config)
	}
}

func (m *flagManager) detectAndNotifyFlagChanges(
	ctx context.Context,
	oldConfig, newConfig *FlagConfiguration,
) {
	// Check for changed flags
	for name, newFlag := range newConfig.Flags {
		if oldFlag, exists := oldConfig.Flags[name]; exists {
			// Flag was updated
			if oldFlag.Version != newFlag.Version {
				m.notifyFlagChanged(ctx, newFlag, oldFlag)
			}
		} else {
			// Flag was added
			m.notifyFlagChanged(ctx, newFlag, nil)
		}
	}

	// Check for deleted flags
	for name, oldFlag := range oldConfig.Flags {
		if _, exists := newConfig.Flags[name]; !exists {
			// Flag was deleted
			m.notifyFlagChanged(ctx, nil, oldFlag)
		}
	}
}

// Circuit breaker methods

func (cb *CircuitBreaker) CanEvaluate() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case "closed":
		return true
	case "open":
		// Check if we should try half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == "open" && time.Since(cb.lastFailureTime) > cb.resetTimeout {
				cb.state = "half-open"
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == "half-open"
		}
		return false
	case "half-open":
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == "half-open" {
		cb.state = "closed"
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = "open"
	}
}

// Stop shuts down the flag manager
func (m *flagManager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}