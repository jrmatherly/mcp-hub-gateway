package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
)

// metricsCollector implements the MetricsCollector interface
type metricsCollector struct {
	// Dependencies
	cache cache.Cache

	// Metrics storage
	globalMetrics   *FlagMetrics
	flagMetrics     map[FlagName]*FlagMetric
	metricsMu       sync.RWMutex

	// Background aggregation
	stopChan      chan struct{}
	wg            sync.WaitGroup
	flushInterval time.Duration

	// Configuration
	retentionPeriod time.Duration
	maxFlagMetrics  int
}

// CreateMetricsCollector creates a new metrics collector
func CreateMetricsCollector(cacheProvider cache.Cache) (MetricsCollector, error) {
	if cacheProvider == nil {
		return nil, fmt.Errorf("cache provider is required")
	}

	collector := &metricsCollector{
		cache:           cacheProvider,
		globalMetrics:   &FlagMetrics{
			FlagEvaluations: make(map[FlagName]*FlagMetric),
			ErrorsByType:    make(map[string]int64),
			StartTime:       time.Now(),
			LastUpdated:     time.Now(),
		},
		flagMetrics:     make(map[FlagName]*FlagMetric),
		stopChan:        make(chan struct{}),
		flushInterval:   time.Minute * 5,
		retentionPeriod: time.Hour * 24,
		maxFlagMetrics:  10000, // Prevent memory issues
	}

	// Start background workers
	collector.startBackgroundWorkers()

	return collector, nil
}

// RecordEvaluation records a flag evaluation
func (m *metricsCollector) RecordEvaluation(ctx context.Context, eval *FlagEvaluation) {
	if eval == nil {
		return
	}

	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	now := time.Now()

	// Update global metrics
	m.globalMetrics.TotalEvaluations++
	m.globalMetrics.TotalEvaluationTime += eval.Duration.Milliseconds()
	m.globalMetrics.AverageEvaluationTime = float64(
		m.globalMetrics.TotalEvaluationTime,
	) / float64(
		m.globalMetrics.TotalEvaluations,
	)

	if eval.CacheHit {
		m.globalMetrics.CacheHits++
	} else {
		m.globalMetrics.CacheMisses++
	}

	totalCacheOperations := m.globalMetrics.CacheHits + m.globalMetrics.CacheMisses
	if totalCacheOperations > 0 {
		m.globalMetrics.CacheHitRate = float64(
			m.globalMetrics.CacheHits,
		) / float64(
			totalCacheOperations,
		) * 100
	}

	m.globalMetrics.LastUpdated = now

	// Update flag-specific metrics
	flagMetric, exists := m.flagMetrics[eval.Flag]
	if !exists {
		// Check if we've hit the limit
		if len(m.flagMetrics) >= m.maxFlagMetrics {
			// Remove oldest metric
			m.removeOldestFlagMetric()
		}

		flagMetric = &FlagMetric{
			Name:                  eval.Flag,
			RuleMatches:           make(map[string]int64),
			VariantCounts:         make(map[string]int64),
			FirstEvaluation:       now,
			LastEvaluation:        now,
			AverageEvaluationTime: eval.Duration,
			MaxEvaluationTime:     eval.Duration,
		}
		m.flagMetrics[eval.Flag] = flagMetric
		m.globalMetrics.FlagEvaluations[eval.Flag] = flagMetric
	}

	// Update flag metrics
	flagMetric.TotalEvaluations++
	flagMetric.LastEvaluation = now

	if eval.Value.Enabled {
		flagMetric.TrueEvaluations++
	} else {
		flagMetric.FalseEvaluations++
	}

	if flagMetric.TotalEvaluations > 0 {
		flagMetric.TrueRate = float64(
			flagMetric.TrueEvaluations,
		) / float64(
			flagMetric.TotalEvaluations,
		) * 100
	}

	// Update rule matches
	if eval.Value.RuleMatched != "" {
		flagMetric.RuleMatches[eval.Value.RuleMatched]++
	}

	// Update variant counts
	if eval.Value.Variant != "" {
		flagMetric.VariantCounts[eval.Value.Variant]++
	}

	// Update timing
	flagMetric.AverageEvaluationTime = time.Duration(
		(int64(flagMetric.AverageEvaluationTime)*flagMetric.TotalEvaluations + eval.Duration.Nanoseconds()) /
			(flagMetric.TotalEvaluations + 1),
	)

	if eval.Duration > flagMetric.MaxEvaluationTime {
		flagMetric.MaxEvaluationTime = eval.Duration
	}

	// Store evaluation in cache for detailed analysis
	m.storeEvaluationInCache(ctx, eval)
}

// RecordError records an evaluation error
func (m *metricsCollector) RecordError(ctx context.Context, flag FlagName, err error) {
	if err == nil {
		return
	}

	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	m.globalMetrics.ErrorCount++

	// Categorize error type
	errorType := m.categorizeError(err)
	m.globalMetrics.ErrorsByType[errorType]++

	// Calculate error rate
	totalOperations := m.globalMetrics.TotalEvaluations + m.globalMetrics.ErrorCount
	if totalOperations > 0 {
		m.globalMetrics.ErrorRate = float64(
			m.globalMetrics.ErrorCount,
		) / float64(
			totalOperations,
		) * 100
	}

	m.globalMetrics.LastUpdated = time.Now()
}

// GetMetrics returns current global metrics
func (m *metricsCollector) GetMetrics(ctx context.Context) (*FlagMetrics, error) {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// Create a deep copy to avoid race conditions
	metrics := &FlagMetrics{
		TotalEvaluations:      m.globalMetrics.TotalEvaluations,
		TotalEvaluationTime:   m.globalMetrics.TotalEvaluationTime,
		AverageEvaluationTime: m.globalMetrics.AverageEvaluationTime,
		CacheHits:             m.globalMetrics.CacheHits,
		CacheMisses:           m.globalMetrics.CacheMisses,
		CacheHitRate:          m.globalMetrics.CacheHitRate,
		ErrorCount:            m.globalMetrics.ErrorCount,
		ErrorRate:             m.globalMetrics.ErrorRate,
		LastUpdated:           m.globalMetrics.LastUpdated,
		StartTime:             m.globalMetrics.StartTime,
		FlagEvaluations:       make(map[FlagName]*FlagMetric),
		ErrorsByType:          make(map[string]int64),
	}

	// Copy flag evaluations
	for name, flagMetric := range m.globalMetrics.FlagEvaluations {
		metrics.FlagEvaluations[name] = m.copyFlagMetric(flagMetric)
	}

	// Copy error types
	for errorType, count := range m.globalMetrics.ErrorsByType {
		metrics.ErrorsByType[errorType] = count
	}

	return metrics, nil
}

// GetFlagMetrics returns metrics for a specific flag
func (m *metricsCollector) GetFlagMetrics(ctx context.Context, flag FlagName) (*FlagMetric, error) {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	flagMetric, exists := m.flagMetrics[flag]
	if !exists {
		return nil, fmt.Errorf("no metrics found for flag: %s", flag)
	}

	return m.copyFlagMetric(flagMetric), nil
}

// Reset clears all metrics
func (m *metricsCollector) Reset(ctx context.Context) error {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	now := time.Now()

	// Reset global metrics
	m.globalMetrics = &FlagMetrics{
		FlagEvaluations: make(map[FlagName]*FlagMetric),
		ErrorsByType:    make(map[string]int64),
		StartTime:       now,
		LastUpdated:     now,
	}

	// Clear flag metrics
	m.flagMetrics = make(map[FlagName]*FlagMetric)

	// Clear cache
	if err := m.clearMetricsCache(ctx); err != nil {
		return fmt.Errorf("failed to clear metrics cache: %w", err)
	}

	return nil
}

// Helper methods

func (m *metricsCollector) copyFlagMetric(original *FlagMetric) *FlagMetric {
	copy := &FlagMetric{
		Name:                  original.Name,
		TotalEvaluations:      original.TotalEvaluations,
		TrueEvaluations:       original.TrueEvaluations,
		FalseEvaluations:      original.FalseEvaluations,
		TrueRate:              original.TrueRate,
		AverageEvaluationTime: original.AverageEvaluationTime,
		MaxEvaluationTime:     original.MaxEvaluationTime,
		FirstEvaluation:       original.FirstEvaluation,
		LastEvaluation:        original.LastEvaluation,
		RuleMatches:           make(map[string]int64),
		VariantCounts:         make(map[string]int64),
	}

	// Copy maps
	for rule, count := range original.RuleMatches {
		copy.RuleMatches[rule] = count
	}

	for variant, count := range original.VariantCounts {
		copy.VariantCounts[variant] = count
	}

	return copy
}

func (m *metricsCollector) removeOldestFlagMetric() {
	var oldestFlag FlagName
	var oldestTime time.Time

	for flag, metric := range m.flagMetrics {
		if oldestTime.IsZero() || metric.FirstEvaluation.Before(oldestTime) {
			oldestTime = metric.FirstEvaluation
			oldestFlag = flag
		}
	}

	if oldestFlag != "" {
		delete(m.flagMetrics, oldestFlag)
		delete(m.globalMetrics.FlagEvaluations, oldestFlag)
	}
}

func (m *metricsCollector) categorizeError(err error) string {
	errorStr := err.Error()

	switch {
	case strings.Contains(errorStr, "timeout"):
		return "timeout"
	case strings.Contains(errorStr, "not found"):
		return "not_found"
	case strings.Contains(errorStr, "validation"):
		return "validation"
	case strings.Contains(errorStr, "permission"):
		return "permission"
	case strings.Contains(errorStr, "circuit breaker"):
		return "circuit_breaker"
	case strings.Contains(errorStr, "cache"):
		return "cache"
	case strings.Contains(errorStr, "database"):
		return "database"
	case strings.Contains(errorStr, "network"):
		return "network"
	default:
		return "unknown"
	}
}

func (m *metricsCollector) storeEvaluationInCache(ctx context.Context, eval *FlagEvaluation) {
	// Store detailed evaluation for analysis
	key := fmt.Sprintf("flag_eval:%s:%s:%d", eval.Flag, eval.EvaluationID, eval.Context.Timestamp.Unix())

	data, err := json.Marshal(eval)
	if err != nil {
		return // Skip if marshaling fails
	}

	// Store with short TTL for detailed analysis
	m.cache.Set(ctx, key, data, time.Hour)
}

func (m *metricsCollector) clearMetricsCache(ctx context.Context) error {
	// Clear evaluation cache
	pattern := "flag_eval:*"
	keys, err := m.cache.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	for _, key := range keys {
		m.cache.Delete(ctx, key)
	}

	return nil
}

func (m *metricsCollector) startBackgroundWorkers() {
	// Metrics aggregation worker
	m.wg.Add(1)
	go m.aggregationWorker()

	// Cache cleanup worker
	m.wg.Add(1)
	go m.cleanupWorker()
}

func (m *metricsCollector) aggregationWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.aggregateMetrics()
		}
	}
}

func (m *metricsCollector) cleanupWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.cleanupOldMetrics()
		}
	}
}

func (m *metricsCollector) aggregateMetrics() {
	// Periodic aggregation of metrics
	// This could include:
	// - Calculating percentiles
	// - Flushing to persistent storage
	// - Computing derived metrics

	ctx := context.Background()

	// Store current metrics snapshot in cache
	metrics, err := m.GetMetrics(ctx)
	if err != nil {
		return
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return
	}

	key := fmt.Sprintf("metrics_snapshot:%d", time.Now().Unix())
	m.cache.Set(ctx, key, data, time.Hour*24)
}

func (m *metricsCollector) cleanupOldMetrics() {
	// Clean up old evaluation records from cache
	ctx := context.Background()
	cutoff := time.Now().Add(-m.retentionPeriod)

	pattern := "flag_eval:*"
	keys, err := m.cache.Keys(ctx, pattern)
	if err != nil {
		return
	}

	for _, key := range keys {
		// Extract timestamp from key
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			if timestamp, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
				evalTime := time.Unix(timestamp, 0)
				if evalTime.Before(cutoff) {
					m.cache.Delete(ctx, key)
				}
			}
		}
	}

	// Clean up old metric snapshots
	pattern = "metrics_snapshot:*"
	keys, err = m.cache.Keys(ctx, pattern)
	if err != nil {
		return
	}

	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) >= 2 {
			if timestamp, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				snapshotTime := time.Unix(timestamp, 0)
				if snapshotTime.Before(cutoff) {
					m.cache.Delete(ctx, key)
				}
			}
		}
	}
}

// Stop shuts down the metrics collector
func (m *metricsCollector) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// Additional analysis methods

// GetTopFlags returns the most frequently evaluated flags
func (m *metricsCollector) GetTopFlags(ctx context.Context, limit int) ([]*FlagMetric, error) {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// Convert to slice for sorting
	metrics := make([]*FlagMetric, 0, len(m.flagMetrics))
	for _, metric := range m.flagMetrics {
		metrics = append(metrics, m.copyFlagMetric(metric))
	}

	// Sort by total evaluations (descending)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].TotalEvaluations > metrics[j].TotalEvaluations
	})

	// Apply limit
	if limit > 0 && len(metrics) > limit {
		metrics = metrics[:limit]
	}

	return metrics, nil
}

// GetFlagsByTrueRate returns flags sorted by their true rate
func (m *metricsCollector) GetFlagsByTrueRate(ctx context.Context, ascending bool) ([]*FlagMetric, error) {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// Convert to slice for sorting
	metrics := make([]*FlagMetric, 0, len(m.flagMetrics))
	for _, metric := range m.flagMetrics {
		if metric.TotalEvaluations > 0 { // Only include flags with evaluations
			metrics = append(metrics, m.copyFlagMetric(metric))
		}
	}

	// Sort by true rate
	sort.Slice(metrics, func(i, j int) bool {
		if ascending {
			return metrics[i].TrueRate < metrics[j].TrueRate
		}
		return metrics[i].TrueRate > metrics[j].TrueRate
	})

	return metrics, nil
}

// GetSlowestFlags returns flags with the highest evaluation times
func (m *metricsCollector) GetSlowestFlags(ctx context.Context, limit int) ([]*FlagMetric, error) {
	m.metricsMu.RLock()
	defer m.metricsMu.RUnlock()

	// Convert to slice for sorting
	metrics := make([]*FlagMetric, 0, len(m.flagMetrics))
	for _, metric := range m.flagMetrics {
		metrics = append(metrics, m.copyFlagMetric(metric))
	}

	// Sort by average evaluation time (descending)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].AverageEvaluationTime > metrics[j].AverageEvaluationTime
	})

	// Apply limit
	if limit > 0 && len(metrics) > limit {
		metrics = metrics[:limit]
	}

	return metrics, nil
}

// GetEvaluationHistory returns recent evaluations for a flag
func (m *metricsCollector) GetEvaluationHistory(
	ctx context.Context,
	flag FlagName,
	since time.Time,
	limit int,
) ([]*FlagEvaluation, error) {
	pattern := fmt.Sprintf("flag_eval:%s:*", flag)
	keys, err := m.cache.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get evaluation keys: %w", err)
	}

	var evaluations []*FlagEvaluation

	for _, key := range keys {
		data, err := m.cache.Get(ctx, key)
		if err != nil {
			continue
		}

		var eval FlagEvaluation
		if err := json.Unmarshal(data.([]byte), &eval); err != nil {
			continue
		}

		// Filter by time
		if eval.Context.Timestamp.After(since) {
			evaluations = append(evaluations, &eval)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(evaluations, func(i, j int) bool {
		return evaluations[i].Context.Timestamp.After(evaluations[j].Context.Timestamp)
	})

	// Apply limit
	if limit > 0 && len(evaluations) > limit {
		evaluations = evaluations[:limit]
	}

	return evaluations, nil
}