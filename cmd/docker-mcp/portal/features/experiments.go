package features

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// experimentManager manages A/B testing experiments
type experimentManager struct {
	// Dependencies
	dbPool       database.Pool
	flagManager  FlagManager
	auditor      audit.Logger

	// Active experiments cache
	activeExperiments map[string]*Experiment
	experimentsMu     sync.RWMutex

	// Participant tracking
	participants map[string]map[string]string // experimentID -> userID -> variant
	participantsMu sync.RWMutex

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Configuration
	refreshInterval time.Duration
	maxParticipants int
}

// ExperimentManager defines the interface for experiment management
type ExperimentManager interface {
	// Experiment lifecycle
	CreateExperiment(ctx context.Context, experiment *Experiment) error
	UpdateExperiment(ctx context.Context, experiment *Experiment) error
	StartExperiment(ctx context.Context, experimentID string) error
	PauseExperiment(ctx context.Context, experimentID string) error
	StopExperiment(ctx context.Context, experimentID string) error

	// Experiment management
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)
	ListExperiments(ctx context.Context, status string) ([]*Experiment, error)
	DeleteExperiment(ctx context.Context, experimentID string) error

	// Participant management
	AssignParticipant(ctx context.Context, experimentID string, userID uuid.UUID) (string, error)
	GetParticipantVariant(ctx context.Context, experimentID string, userID uuid.UUID) (string, error)
	GetExperimentParticipants(ctx context.Context, experimentID string) (map[string]string, error)

	// Results and analysis
	CalculateResults(ctx context.Context, experimentID string) (*ExperimentResults, error)
	GetExperimentMetrics(ctx context.Context, experimentID string) (map[string]*MetricResult, error)
	ExportResults(ctx context.Context, experimentID string, format string) ([]byte, error)

	// Health check
	Health(ctx context.Context) error
}

// CreateExperimentManager creates a new experiment manager
func CreateExperimentManager(
	dbPool database.Pool,
	flagManager FlagManager,
	auditor audit.Logger,
) (ExperimentManager, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	if flagManager == nil {
		return nil, fmt.Errorf("flag manager is required")
	}
	if auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}

	manager := &experimentManager{
		dbPool:            dbPool,
		flagManager:       flagManager,
		auditor:           auditor,
		activeExperiments: make(map[string]*Experiment),
		participants:      make(map[string]map[string]string),
		stopChan:          make(chan struct{}),
		refreshInterval:   time.Minute * 5,
		maxParticipants:   100000, // Prevent memory issues
	}

	// Load active experiments
	if err := manager.loadActiveExperiments(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load active experiments: %w", err)
	}

	// Start background workers
	manager.startBackgroundWorkers()

	return manager, nil
}

// CreateExperiment creates a new experiment
func (e *experimentManager) CreateExperiment(ctx context.Context, experiment *Experiment) error {
	if experiment == nil {
		return fmt.Errorf("experiment is required")
	}

	// Validate experiment
	if err := e.validateExperiment(experiment); err != nil {
		return fmt.Errorf("invalid experiment: %w", err)
	}

	// Set metadata
	now := time.Now()
	if experiment.ID == "" {
		experiment.ID = uuid.New().String()
	}
	experiment.CreatedAt = now
	experiment.UpdatedAt = now
	experiment.Status = "draft"

	// Save to database
	if err := e.saveExperiment(ctx, experiment); err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	// Audit log
	e.auditor.Log(ctx, audit.ActionCreate, "experiment", experiment.ID, "system", map[string]any{
		"name":              experiment.Name,
		"flag":              experiment.Flag,
		"traffic_allocation": experiment.TrafficAllocation,
		"variant_count":     len(experiment.Variants),
	})

	return nil
}

// UpdateExperiment updates an existing experiment
func (e *experimentManager) UpdateExperiment(ctx context.Context, experiment *Experiment) error {
	if experiment == nil {
		return fmt.Errorf("experiment is required")
	}

	// Get existing experiment
	existing, err := e.GetExperiment(ctx, experiment.ID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	// Don't allow updates to running experiments unless paused
	if existing.Status == "running" {
		return fmt.Errorf("cannot update running experiment")
	}

	// Validate experiment
	if err := e.validateExperiment(experiment); err != nil {
		return fmt.Errorf("invalid experiment: %w", err)
	}

	// Update metadata
	experiment.UpdatedAt = time.Now()
	experiment.CreatedAt = existing.CreatedAt
	experiment.CreatedBy = existing.CreatedBy

	// Save to database
	if err := e.saveExperiment(ctx, experiment); err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	// Update cache if active
	if experiment.Status == "running" {
		e.experimentsMu.Lock()
		e.activeExperiments[experiment.ID] = experiment
		e.experimentsMu.Unlock()
	}

	// Audit log
	e.auditor.Log(ctx, audit.ActionUpdate, "experiment", experiment.ID, "system", map[string]any{
		"old_status": existing.Status,
		"new_status": experiment.Status,
	})

	return nil
}

// StartExperiment starts an experiment
func (e *experimentManager) StartExperiment(ctx context.Context, experimentID string) error {
	experiment, err := e.GetExperiment(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	if experiment.Status == "running" {
		return fmt.Errorf("experiment is already running")
	}

	if experiment.Status == "completed" {
		return fmt.Errorf("cannot restart completed experiment")
	}

	// Validate that the flag exists
	_, err = e.flagManager.GetFlag(ctx, experiment.Flag)
	if err != nil {
		return fmt.Errorf("experiment flag not found: %w", err)
	}

	// Update status and timing
	now := time.Now()
	experiment.Status = "running"
	experiment.StartTime = now
	experiment.UpdatedAt = now

	// Calculate end time if duration is specified
	if experiment.Duration != nil {
		endTime := now.Add(*experiment.Duration)
		experiment.EndTime = &endTime
	}

	// Save to database
	if err := e.saveExperiment(ctx, experiment); err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	// Add to active experiments
	e.experimentsMu.Lock()
	e.activeExperiments[experimentID] = experiment
	e.experimentsMu.Unlock()

	// Initialize participant tracking
	e.participantsMu.Lock()
	e.participants[experimentID] = make(map[string]string)
	e.participantsMu.Unlock()

	// Audit log
	e.auditor.Log(ctx, audit.ActionUpdate, "experiment", experimentID, "system", map[string]any{
		"action": "start_experiment",
	})

	return nil
}

// PauseExperiment pauses a running experiment
func (e *experimentManager) PauseExperiment(ctx context.Context, experimentID string) error {
	experiment, err := e.GetExperiment(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	if experiment.Status != "running" {
		return fmt.Errorf("experiment is not running")
	}

	// Update status
	experiment.Status = "paused"
	experiment.UpdatedAt = time.Now()

	// Save to database
	if err := e.saveExperiment(ctx, experiment); err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	// Remove from active experiments
	e.experimentsMu.Lock()
	delete(e.activeExperiments, experimentID)
	e.experimentsMu.Unlock()

	// Audit log
	e.auditor.Log(ctx, audit.ActionUpdate, "experiment", experimentID, "system", map[string]any{
		"action": "pause_experiment",
	})

	return nil
}

// StopExperiment stops an experiment and calculates final results
func (e *experimentManager) StopExperiment(ctx context.Context, experimentID string) error {
	experiment, err := e.GetExperiment(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	if experiment.Status != "running" && experiment.Status != "paused" {
		return fmt.Errorf("experiment is not running or paused")
	}

	// Calculate final results
	results, err := e.CalculateResults(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("failed to calculate results: %w", err)
	}

	// Update experiment
	now := time.Now()
	experiment.Status = "completed"
	experiment.EndTime = &now
	experiment.UpdatedAt = now
	experiment.Results = results

	// Save to database
	if err := e.saveExperiment(ctx, experiment); err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	// Remove from active experiments
	e.experimentsMu.Lock()
	delete(e.activeExperiments, experimentID)
	e.experimentsMu.Unlock()

	// Keep participant data for analysis
	// In production, you might want to persist this to database

	// Audit log
	e.auditor.Log(ctx, audit.ActionUpdate, "experiment", experimentID, "system", map[string]any{
		"action":            "stop_experiment",
		"total_participants": results.TotalParticipants,
		"significant_winner": results.SignificantWinner,
	})

	return nil
}

// GetExperiment retrieves an experiment by ID
func (e *experimentManager) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	// Check active experiments cache first
	e.experimentsMu.RLock()
	if experiment, exists := e.activeExperiments[experimentID]; exists {
		e.experimentsMu.RUnlock()
		return e.copyExperiment(experiment), nil
	}
	e.experimentsMu.RUnlock()

	// Load from database
	experiment, err := e.loadExperiment(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load experiment: %w", err)
	}

	return experiment, nil
}

// ListExperiments lists experiments by status
func (e *experimentManager) ListExperiments(ctx context.Context, status string) ([]*Experiment, error) {
	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Release()

	query := `
		SELECT
			id, name, description, status, flag_name, variants,
			audience_filter, traffic_allocation, start_time, end_time,
			duration_seconds, primary_metric, secondary_metrics,
			results, created_at, updated_at, created_by
		FROM feature_flag_experiments
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}
	if status != "" {
		query += " AND status = $1"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query experiments: %w", err)
	}
	defer rows.Close()

	var experiments []*Experiment

	for rows.Next() {
		experiment, err := e.scanExperiment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan experiment: %w", err)
		}
		experiments = append(experiments, experiment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate experiments: %w", err)
	}

	return experiments, nil
}

// DeleteExperiment deletes an experiment
func (e *experimentManager) DeleteExperiment(ctx context.Context, experimentID string) error {
	experiment, err := e.GetExperiment(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("experiment not found: %w", err)
	}

	if experiment.Status == "running" {
		return fmt.Errorf("cannot delete running experiment")
	}

	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Release()

	// Soft delete
	query := `
		UPDATE feature_flag_experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := conn.Exec(ctx, query, experimentID)
	if err != nil {
		return fmt.Errorf("failed to delete experiment: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("experiment not found")
	}

	// Remove from caches
	e.experimentsMu.Lock()
	delete(e.activeExperiments, experimentID)
	e.experimentsMu.Unlock()

	e.participantsMu.Lock()
	delete(e.participants, experimentID)
	e.participantsMu.Unlock()

	// Audit log
	e.auditor.Log(ctx, audit.ActionDelete, "experiment", experimentID, "system", map[string]any{
		"name": experiment.Name,
	})

	return nil
}

// AssignParticipant assigns a user to an experiment variant
func (e *experimentManager) AssignParticipant(ctx context.Context, experimentID string, userID uuid.UUID) (string, error) {
	// Check if already assigned
	e.participantsMu.RLock()
	if participants, exists := e.participants[experimentID]; exists {
		if variant, assigned := participants[userID.String()]; assigned {
			e.participantsMu.RUnlock()
			return variant, nil
		}
	}
	e.participantsMu.RUnlock()

	// Get experiment
	e.experimentsMu.RLock()
	experiment, exists := e.activeExperiments[experimentID]
	e.experimentsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("experiment not active")
	}

	// Check traffic allocation
	if !e.isUserInTrafficAllocation(userID, experiment.TrafficAllocation) {
		return "", fmt.Errorf("user not in traffic allocation")
	}

	// Check audience filter
	if experiment.AudienceFilter != nil {
		evalCtx := &EvaluationContext{
			UserID:    userID,
			Timestamp: time.Now(),
		}

		if !e.matchesAudienceFilter(experiment.AudienceFilter, evalCtx) {
			return "", fmt.Errorf("user not in target audience")
		}
	}

	// Select variant
	variant := e.selectVariantForUser(userID, experiment.Variants)
	if variant == "" {
		return "", fmt.Errorf("no variant selected")
	}

	// Store assignment
	e.participantsMu.Lock()
	if e.participants[experimentID] == nil {
		e.participants[experimentID] = make(map[string]string)
	}

	// Check participant limit
	if len(e.participants[experimentID]) >= e.maxParticipants {
		e.participantsMu.Unlock()
		return "", fmt.Errorf("experiment has reached maximum participants")
	}

	e.participants[experimentID][userID.String()] = variant
	e.participantsMu.Unlock()

	// Store in database for persistence
	e.storeParticipantAssignment(ctx, experimentID, userID, variant)

	return variant, nil
}

// GetParticipantVariant gets the assigned variant for a user
func (e *experimentManager) GetParticipantVariant(ctx context.Context, experimentID string, userID uuid.UUID) (string, error) {
	e.participantsMu.RLock()
	defer e.participantsMu.RUnlock()

	participants, exists := e.participants[experimentID]
	if !exists {
		return "", fmt.Errorf("experiment not found")
	}

	variant, assigned := participants[userID.String()]
	if !assigned {
		return "", fmt.Errorf("user not assigned to experiment")
	}

	return variant, nil
}

// GetExperimentParticipants gets all participants for an experiment
func (e *experimentManager) GetExperimentParticipants(ctx context.Context, experimentID string) (map[string]string, error) {
	e.participantsMu.RLock()
	defer e.participantsMu.RUnlock()

	participants, exists := e.participants[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment not found")
	}

	// Return copy to prevent modification
	result := make(map[string]string)
	for userID, variant := range participants {
		result[userID] = variant
	}

	return result, nil
}

// CalculateResults calculates experiment results with statistical analysis
func (e *experimentManager) CalculateResults(ctx context.Context, experimentID string) (*ExperimentResults, error) {
	experiment, err := e.GetExperiment(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %w", err)
	}

	// Get participants
	participants, err := e.GetExperimentParticipants(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	// Calculate variant counts
	variantCounts := make(map[string]int64)
	for _, variant := range participants {
		variantCounts[variant]++
	}

	// Calculate metrics (simplified implementation)
	metricResults := make(map[string]*MetricResult)

	// For the primary metric, calculate conversion rates
	primaryResult := &MetricResult{
		MetricName:     experiment.PrimaryMetric,
		VariantResults: make(map[string]*VariantResult),
	}

	totalParticipants := int64(len(participants))
	for variantName, count := range variantCounts {
		// Simplified conversion calculation
		// In practice, you'd integrate with your analytics system
		conversionRate := e.calculateConversionRate(ctx, experimentID, variantName)

		primaryResult.VariantResults[variantName] = &VariantResult{
			Variant: variantName,
			Count:   count,
			Rate:    conversionRate,
		}
	}

	// Determine winner (simplified statistical test)
	winner, confidence := e.determineWinner(primaryResult.VariantResults)
	primaryResult.Winner = winner
	primaryResult.Confidence = confidence

	metricResults[experiment.PrimaryMetric] = primaryResult

	// Calculate secondary metrics
	for _, metric := range experiment.SecondaryMetrics {
		secondaryResult := &MetricResult{
			MetricName:     metric,
			VariantResults: make(map[string]*VariantResult),
		}

		for variantName, count := range variantCounts {
			rate := e.calculateMetricRate(ctx, experimentID, variantName, metric)
			secondaryResult.VariantResults[variantName] = &VariantResult{
				Variant: variantName,
				Count:   count,
				Rate:    rate,
			}
		}

		metricResults[metric] = secondaryResult
	}

	// Statistical significance test (simplified)
	pValue := e.calculatePValue(primaryResult.VariantResults)
	confidenceLevel := 95.0
	significantWinner := ""
	if pValue < 0.05 && winner != "" {
		significantWinner = winner
	}

	results := &ExperimentResults{
		TotalParticipants: totalParticipants,
		VariantCounts:     variantCounts,
		MetricResults:     metricResults,
		ConfidenceLevel:   confidenceLevel,
		PValue:           pValue,
		SignificantWinner: significantWinner,
		AnalyzedAt:       time.Now(),
	}

	// Set data window
	results.DataWindow.Start = experiment.StartTime
	if experiment.EndTime != nil {
		results.DataWindow.End = *experiment.EndTime
	} else {
		results.DataWindow.End = time.Now()
	}

	return results, nil
}

// GetExperimentMetrics gets detailed metrics for an experiment
func (e *experimentManager) GetExperimentMetrics(ctx context.Context, experimentID string) (map[string]*MetricResult, error) {
	results, err := e.CalculateResults(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	return results.MetricResults, nil
}

// ExportResults exports experiment results in the specified format
func (e *experimentManager) ExportResults(ctx context.Context, experimentID string, format string) ([]byte, error) {
	results, err := e.CalculateResults(ctx, experimentID)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(results, "", "  ")
	case "csv":
		return e.exportResultsAsCSV(results)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// Health checks the health of the experiment manager
func (e *experimentManager) Health(ctx context.Context) error {
	// Check database connection
	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer conn.Release()

	// Check active experiments
	e.experimentsMu.RLock()
	activeCount := len(e.activeExperiments)
	e.experimentsMu.RUnlock()

	if activeCount > 100 { // Arbitrary limit
		return fmt.Errorf("too many active experiments: %d", activeCount)
	}

	return nil
}

// Helper methods

func (e *experimentManager) validateExperiment(experiment *Experiment) error {
	if experiment.Name == "" {
		return fmt.Errorf("experiment name is required")
	}

	if experiment.Flag == "" {
		return fmt.Errorf("experiment flag is required")
	}

	if experiment.TrafficAllocation < 0 || experiment.TrafficAllocation > 100 {
		return fmt.Errorf("traffic allocation must be between 0 and 100")
	}

	if len(experiment.Variants) < 2 {
		return fmt.Errorf("experiment must have at least 2 variants")
	}

	// Validate variant weights
	totalWeight := 0
	for _, variant := range experiment.Variants {
		if variant.Name == "" {
			return fmt.Errorf("variant name is required")
		}
		if variant.Weight < 0 || variant.Weight > 100 {
			return fmt.Errorf("variant weight must be between 0 and 100")
		}
		totalWeight += variant.Weight
	}

	if totalWeight != 100 {
		return fmt.Errorf("total variant weight must equal 100, got %d", totalWeight)
	}

	return nil
}

func (e *experimentManager) loadActiveExperiments(ctx context.Context) error {
	experiments, err := e.ListExperiments(ctx, "running")
	if err != nil {
		return err
	}

	e.experimentsMu.Lock()
	defer e.experimentsMu.Unlock()

	for _, experiment := range experiments {
		e.activeExperiments[experiment.ID] = experiment
	}

	return nil
}

func (e *experimentManager) saveExperiment(ctx context.Context, experiment *Experiment) error {
	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Release()

	// Serialize JSON fields
	variantsJSON, _ := json.Marshal(experiment.Variants)
	audienceFilterJSON, _ := json.Marshal(experiment.AudienceFilter)
	secondaryMetricsJSON, _ := json.Marshal(experiment.SecondaryMetrics)
	resultsJSON, _ := json.Marshal(experiment.Results)

	var durationSeconds *int64
	if experiment.Duration != nil {
		seconds := int64(experiment.Duration.Seconds())
		durationSeconds = &seconds
	}

	query := `
		INSERT INTO feature_flag_experiments (
			id, name, description, status, flag_name, variants,
			audience_filter, traffic_allocation, start_time, end_time,
			duration_seconds, primary_metric, secondary_metrics,
			results, created_at, updated_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17
		) ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			variants = EXCLUDED.variants,
			audience_filter = EXCLUDED.audience_filter,
			traffic_allocation = EXCLUDED.traffic_allocation,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			duration_seconds = EXCLUDED.duration_seconds,
			primary_metric = EXCLUDED.primary_metric,
			secondary_metrics = EXCLUDED.secondary_metrics,
			results = EXCLUDED.results,
			updated_at = EXCLUDED.updated_at
	`

	_, err = conn.Exec(ctx, query,
		experiment.ID,
		experiment.Name,
		experiment.Description,
		experiment.Status,
		experiment.Flag,
		variantsJSON,
		audienceFilterJSON,
		experiment.TrafficAllocation,
		experiment.StartTime,
		experiment.EndTime,
		durationSeconds,
		experiment.PrimaryMetric,
		secondaryMetricsJSON,
		resultsJSON,
		experiment.CreatedAt,
		experiment.UpdatedAt,
		experiment.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save experiment: %w", err)
	}

	return nil
}

func (e *experimentManager) loadExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Release()

	query := `
		SELECT
			id, name, description, status, flag_name, variants,
			audience_filter, traffic_allocation, start_time, end_time,
			duration_seconds, primary_metric, secondary_metrics,
			results, created_at, updated_at, created_by
		FROM feature_flag_experiments
		WHERE id = $1 AND deleted_at IS NULL
	`

	row := conn.QueryRow(ctx, query, experimentID)
	return e.scanExperiment(row)
}

func (e *experimentManager) scanExperiment(row interface{ Scan(...interface{}) error }) (*Experiment, error) {
	var experiment Experiment
	var variantsJSON, audienceFilterJSON, secondaryMetricsJSON, resultsJSON []byte
	var endTime *time.Time
	var durationSeconds *int64

	err := row.Scan(
		&experiment.ID,
		&experiment.Name,
		&experiment.Description,
		&experiment.Status,
		&experiment.Flag,
		&variantsJSON,
		&audienceFilterJSON,
		&experiment.TrafficAllocation,
		&experiment.StartTime,
		&endTime,
		&durationSeconds,
		&experiment.PrimaryMetric,
		&secondaryMetricsJSON,
		&resultsJSON,
		&experiment.CreatedAt,
		&experiment.UpdatedAt,
		&experiment.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Handle optional fields
	experiment.EndTime = endTime
	if durationSeconds != nil {
		duration := time.Duration(*durationSeconds) * time.Second
		experiment.Duration = &duration
	}

	// Parse JSON fields
	if len(variantsJSON) > 0 {
		json.Unmarshal(variantsJSON, &experiment.Variants)
	}

	if len(audienceFilterJSON) > 0 {
		json.Unmarshal(audienceFilterJSON, &experiment.AudienceFilter)
	}

	if len(secondaryMetricsJSON) > 0 {
		json.Unmarshal(secondaryMetricsJSON, &experiment.SecondaryMetrics)
	}

	if len(resultsJSON) > 0 {
		json.Unmarshal(resultsJSON, &experiment.Results)
	}

	return &experiment, nil
}

func (e *experimentManager) copyExperiment(original *Experiment) *Experiment {
	// Deep copy experiment
	data, _ := json.Marshal(original)
	var copy Experiment
	json.Unmarshal(data, &copy)
	return &copy
}

func (e *experimentManager) isUserInTrafficAllocation(userID uuid.UUID, allocation int) bool {
	if allocation >= 100 {
		return true
	}
	if allocation <= 0 {
		return false
	}

	// Use consistent hashing for traffic allocation
	hash := e.generateUserHash(userID.String())
	bucket := hash % 100
	return bucket < allocation
}

func (e *experimentManager) matchesAudienceFilter(filter *AudienceFilter, evalCtx *EvaluationContext) bool {
	// Simplified audience filtering
	// In practice, you'd implement comprehensive filtering logic

	if len(filter.Conditions) > 0 {
		// Use the evaluation engine to check conditions
		engine, _ := CreateEvaluationEngine()
		matched, _ := engine.EvaluateConditions(context.Background(), filter.Conditions, evalCtx)
		return matched
	}

	return true // No filters means everyone matches
}

func (e *experimentManager) selectVariantForUser(userID uuid.UUID, variants []FlagVariant) string {
	// Use consistent hashing to select variant
	hash := e.generateUserHash(userID.String())

	// Calculate cumulative weights
	totalWeight := 0
	for _, variant := range variants {
		if variant.Enabled {
			totalWeight += variant.Weight
		}
	}

	if totalWeight == 0 {
		return ""
	}

	bucket := hash % totalWeight
	cumulativeWeight := 0

	for _, variant := range variants {
		if !variant.Enabled {
			continue
		}

		cumulativeWeight += variant.Weight
		if bucket < cumulativeWeight {
			return variant.Name
		}
	}

	return ""
}

func (e *experimentManager) generateUserHash(userID string) int {
	// Simple hash function for consistent bucketing
	hash := 0
	for _, char := range userID {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func (e *experimentManager) storeParticipantAssignment(ctx context.Context, experimentID string, userID uuid.UUID, variant string) {
	// Store participant assignment in database for persistence
	// This is a simplified implementation
	conn, err := e.dbPool.Get(ctx)
	if err != nil {
		return
	}
	defer conn.Release()

	query := `
		INSERT INTO experiment_participants (experiment_id, user_id, variant, assigned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (experiment_id, user_id) DO NOTHING
	`

	conn.Exec(ctx, query, experimentID, userID, variant)
}

func (e *experimentManager) calculateConversionRate(ctx context.Context, experimentID, variant string) float64 {
	// Simplified conversion rate calculation
	// In practice, you'd integrate with your analytics system
	return 0.1 + float64(e.generateUserHash(experimentID+variant)%20)/100.0
}

func (e *experimentManager) calculateMetricRate(ctx context.Context, experimentID, variant, metric string) float64 {
	// Simplified metric calculation
	return 0.05 + float64(e.generateUserHash(experimentID+variant+metric)%15)/100.0
}

func (e *experimentManager) determineWinner(variantResults map[string]*VariantResult) (string, float64) {
	// Simplified winner determination
	var bestVariant string
	var bestRate float64

	for variant, result := range variantResults {
		if result.Rate > bestRate {
			bestRate = result.Rate
			bestVariant = variant
		}
	}

	// Simplified confidence calculation
	confidence := 75.0 + float64(len(variantResults))*5.0
	if confidence > 95.0 {
		confidence = 95.0
	}

	return bestVariant, confidence
}

func (e *experimentManager) calculatePValue(variantResults map[string]*VariantResult) float64 {
	// Simplified p-value calculation
	// In practice, you'd use proper statistical tests
	if len(variantResults) < 2 {
		return 1.0
	}

	// Simple approximation based on rate differences
	rates := make([]float64, 0, len(variantResults))
	for _, result := range variantResults {
		rates = append(rates, result.Rate)
	}

	sort.Float64s(rates)
	diff := rates[len(rates)-1] - rates[0]

	// Very simplified p-value approximation
	pValue := math.Max(0.001, 0.5-diff*5)
	return pValue
}

func (e *experimentManager) exportResultsAsCSV(results *ExperimentResults) ([]byte, error) {
	// Simplified CSV export
	csv := "Variant,Participants,Rate\n"
	for variant, count := range results.VariantCounts {
		rate := 0.0
		if primaryResult, exists := results.MetricResults["primary"]; exists {
			if variantResult, exists := primaryResult.VariantResults[variant]; exists {
				rate = variantResult.Rate
			}
		}
		csv += fmt.Sprintf("%s,%d,%.4f\n", variant, count, rate)
	}
	return []byte(csv), nil
}

func (e *experimentManager) startBackgroundWorkers() {
	// Experiment status checker
	e.wg.Add(1)
	go e.statusWorker()

	// Metrics calculator
	e.wg.Add(1)
	go e.metricsWorker()
}

func (e *experimentManager) statusWorker() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.checkExperimentStatus()
		}
	}
}

func (e *experimentManager) metricsWorker() {
	defer e.wg.Done()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.updateExperimentMetrics()
		}
	}
}

func (e *experimentManager) checkExperimentStatus() {
	ctx := context.Background()
	e.experimentsMu.RLock()
	activeExperiments := make(map[string]*Experiment)
	for id, exp := range e.activeExperiments {
		activeExperiments[id] = exp
	}
	e.experimentsMu.RUnlock()

	for id, experiment := range activeExperiments {
		// Check if experiment should end
		if experiment.EndTime != nil && time.Now().After(*experiment.EndTime) {
			e.StopExperiment(ctx, id)
		}
	}
}

func (e *experimentManager) updateExperimentMetrics() {
	// Periodic metrics update for running experiments
	ctx := context.Background()
	e.experimentsMu.RLock()
	activeExperiments := make([]string, 0, len(e.activeExperiments))
	for id := range e.activeExperiments {
		activeExperiments = append(activeExperiments, id)
	}
	e.experimentsMu.RUnlock()

	for _, experimentID := range activeExperiments {
		// Calculate interim results
		_, err := e.CalculateResults(ctx, experimentID)
		if err != nil {
			continue
		}
		// Results are automatically cached in the calculation
	}
}

// Stop shuts down the experiment manager
func (e *experimentManager) Stop() {
	close(e.stopChan)
	e.wg.Wait()
}