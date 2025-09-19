// Package features provides feature flag management for gradual OAuth rollout
// and A/B testing capabilities in the MCP Portal.
package features

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FlagName represents a feature flag identifier
type FlagName string

// OAuth feature flags for gradual rollout
const (
	// Master OAuth switch
	FlagOAuthEnabled FlagName = "oauth_enabled"

	// Automatic 401 handling
	FlagOAuthAuto401 FlagName = "oauth_auto_401"

	// Dynamic Client Registration
	FlagOAuthDCR FlagName = "oauth_dcr"

	// Provider-specific flags
	FlagOAuthProviderGitHub    FlagName = "oauth_provider_github"
	FlagOAuthProviderGoogle    FlagName = "oauth_provider_google"
	FlagOAuthProviderMicrosoft FlagName = "oauth_provider_microsoft"

	// Docker Desktop secrets integration
	FlagOAuthDockerSecrets FlagName = "oauth_docker_secrets"

	// Token management features
	FlagOAuthTokenRefresh FlagName = "oauth_token_refresh"
	FlagOAuthTokenStorage FlagName = "oauth_token_storage"

	// Security features
	FlagOAuthJWTValidation FlagName = "oauth_jwt_validation"
	FlagOAuthHTTPSRequired FlagName = "oauth_https_required"

	// Advanced features
	FlagOAuthAuditLogging FlagName = "oauth_audit_logging"
	FlagOAuthMetrics      FlagName = "oauth_metrics"
	FlagOAuthKeyRotation  FlagName = "oauth_key_rotation"
)

// FlagType represents the type of feature flag evaluation
type FlagType string

const (
	// Boolean flag - simple on/off
	FlagTypeBoolean FlagType = "boolean"

	// Percentage rollout - 0-100% of users
	FlagTypePercentage FlagType = "percentage"

	// User-specific targeting
	FlagTypeUser FlagType = "user"

	// Server-specific targeting
	FlagTypeServer FlagType = "server"

	// A/B testing variant
	FlagTypeVariant FlagType = "variant"
)

// EvaluationContext contains context for flag evaluation
type EvaluationContext struct {
	// User context
	UserID   uuid.UUID `json:"user_id"`
	TenantID string    `json:"tenant_id,omitempty"`

	// Server context
	ServerName string   `json:"server_name,omitempty"`
	ServerTags []string `json:"server_tags,omitempty"`

	// Request context
	RequestID  string            `json:"request_id,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`

	// Environment context
	Environment string    `json:"environment,omitempty"`
	Timestamp   time.Time `json:"timestamp"`

	// Custom attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// FlagValue represents the result of flag evaluation
type FlagValue struct {
	// Flag identification
	Name FlagName `json:"name"`
	Type FlagType `json:"type"`

	// Evaluation result
	Enabled bool        `json:"enabled"`
	Value   interface{} `json:"value,omitempty"`
	Variant string      `json:"variant,omitempty"`

	// Evaluation metadata
	Reason      string                 `json:"reason"`
	RuleMatched string                 `json:"rule_matched,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// Timing
	EvaluatedAt time.Time `json:"evaluated_at"`
	CacheHit    bool      `json:"cache_hit"`
}

// FlagDefinition defines a feature flag configuration
type FlagDefinition struct {
	// Basic properties
	Name        FlagName `json:"name"        yaml:"name"`
	Type        FlagType `json:"type"        yaml:"type"`
	Description string   `json:"description" yaml:"description"`
	Enabled     bool     `json:"enabled"     yaml:"enabled"`

	// Default value when no rules match
	DefaultValue interface{} `json:"default_value" yaml:"default_value"`

	// Targeting rules
	Rules []FlagRule `json:"rules" yaml:"rules"`

	// Percentage rollout (0-100)
	RolloutPercentage int `json:"rollout_percentage" yaml:"rollout_percentage"`

	// User/Server overrides
	UserOverrides   map[string]interface{} `json:"user_overrides,omitempty"   yaml:"user_overrides,omitempty"`
	ServerOverrides map[string]interface{} `json:"server_overrides,omitempty" yaml:"server_overrides,omitempty"`

	// A/B testing variants
	Variants []FlagVariant `json:"variants,omitempty" yaml:"variants,omitempty"`

	// Metadata
	CreatedAt  time.Time `json:"created_at"     yaml:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"     yaml:"updated_at"`
	CreatedBy  uuid.UUID `json:"created_by"     yaml:"created_by"`
	Version    int       `json:"version"        yaml:"version"`
	Tags       []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Deprecated bool      `json:"deprecated"     yaml:"deprecated"`

	// Rollout configuration
	RolloutConfig *RolloutConfig `json:"rollout_config,omitempty" yaml:"rollout_config,omitempty"`
}

// FlagRule defines targeting rules for flag evaluation
type FlagRule struct {
	// Rule identification
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Priority    int    `json:"priority"              yaml:"priority"`
	Enabled     bool   `json:"enabled"               yaml:"enabled"`

	// Conditions that must be met
	Conditions []FlagCondition `json:"conditions" yaml:"conditions"`

	// Value to return when rule matches
	Value   interface{} `json:"value"             yaml:"value"`
	Variant string      `json:"variant,omitempty" yaml:"variant,omitempty"`

	// Rule metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// FlagCondition defines a single condition within a rule
type FlagCondition struct {
	// Attribute to evaluate
	Attribute string `json:"attribute" yaml:"attribute"`

	// Comparison operator
	Operator ConditionOperator `json:"operator" yaml:"operator"`

	// Value(s) to compare against
	Value  interface{}   `json:"value,omitempty"  yaml:"value,omitempty"`
	Values []interface{} `json:"values,omitempty" yaml:"values,omitempty"`

	// Negation
	Negate bool `json:"negate,omitempty" yaml:"negate,omitempty"`
}

// ConditionOperator defines comparison operators for conditions
type ConditionOperator string

const (
	OpEquals       ConditionOperator = "equals"
	OpNotEquals    ConditionOperator = "not_equals"
	OpContains     ConditionOperator = "contains"
	OpNotContains  ConditionOperator = "not_contains"
	OpStartsWith   ConditionOperator = "starts_with"
	OpEndsWith     ConditionOperator = "ends_with"
	OpGreaterThan  ConditionOperator = "greater_than"
	OpLessThan     ConditionOperator = "less_than"
	OpGreaterEqual ConditionOperator = "greater_equal"
	OpLessEqual    ConditionOperator = "less_equal"
	OpIn           ConditionOperator = "in"
	OpNotIn        ConditionOperator = "not_in"
	OpRegexMatch   ConditionOperator = "regex_match"
	OpPercentage   ConditionOperator = "percentage"
	OpVersionMatch ConditionOperator = "version_match"
	OpDateAfter    ConditionOperator = "date_after"
	OpDateBefore   ConditionOperator = "date_before"
)

// FlagVariant represents an A/B testing variant
type FlagVariant struct {
	// Variant identification
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Variant configuration
	Value   interface{} `json:"value"   yaml:"value"`
	Weight  int         `json:"weight"  yaml:"weight"` // Percentage weight (0-100)
	Enabled bool        `json:"enabled" yaml:"enabled"`

	// Tracking
	TrackingKey string `json:"tracking_key,omitempty" yaml:"tracking_key,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// RolloutConfig defines gradual rollout configuration
type RolloutConfig struct {
	// Rollout strategy
	Strategy RolloutStrategy `json:"strategy" yaml:"strategy"`

	// Percentage-based rollout
	StartPercentage int       `json:"start_percentage"       yaml:"start_percentage"`
	EndPercentage   int       `json:"end_percentage"         yaml:"end_percentage"`
	StepSize        int       `json:"step_size"              yaml:"step_size"`
	StepInterval    string    `json:"step_interval"          yaml:"step_interval"` // e.g., "1h", "1d"
	NextStepAt      time.Time `json:"next_step_at,omitempty" yaml:"next_step_at,omitempty"`

	// Canary deployment
	CanaryGroups []string `json:"canary_groups,omitempty" yaml:"canary_groups,omitempty"`

	// Monitoring and safety
	HealthChecks     []HealthCheck `json:"health_checks,omitempty" yaml:"health_checks,omitempty"`
	FailureThreshold float64       `json:"failure_threshold"       yaml:"failure_threshold"`
	AutoRollback     bool          `json:"auto_rollback"           yaml:"auto_rollback"`

	// Schedule
	ScheduledRollout *ScheduledRollout `json:"scheduled_rollout,omitempty" yaml:"scheduled_rollout,omitempty"`
}

// RolloutStrategy defines the rollout approach
type RolloutStrategy string

const (
	RolloutPercentage RolloutStrategy = "percentage"
	RolloutCanary     RolloutStrategy = "canary"
	RolloutScheduled  RolloutStrategy = "scheduled"
	RolloutManual     RolloutStrategy = "manual"
)

// HealthCheck defines a health check for rollout monitoring
type HealthCheck struct {
	Name         string        `json:"name"          yaml:"name"`
	Type         string        `json:"type"          yaml:"type"` // "metric", "endpoint", "custom"
	Target       string        `json:"target"        yaml:"target"`
	Threshold    float64       `json:"threshold"     yaml:"threshold"`
	Operator     string        `json:"operator"      yaml:"operator"` // "gt", "lt", "eq"
	Interval     time.Duration `json:"interval"      yaml:"interval"`
	Timeout      time.Duration `json:"timeout"       yaml:"timeout"`
	FailureLimit int           `json:"failure_limit" yaml:"failure_limit"`
}

// ScheduledRollout defines time-based rollout schedule
type ScheduledRollout struct {
	StartTime     time.Time                `json:"start_time"               yaml:"start_time"`
	EndTime       time.Time                `json:"end_time,omitempty"       yaml:"end_time,omitempty"`
	Milestones    []RolloutMilestone       `json:"milestones"               yaml:"milestones"`
	TimeZone      string                   `json:"timezone"                 yaml:"timezone"`
	BusinessHours *BusinessHoursConstraint `json:"business_hours,omitempty" yaml:"business_hours,omitempty"`
}

// RolloutMilestone defines a scheduled rollout step
type RolloutMilestone struct {
	Time       time.Time `json:"time"             yaml:"time"`
	Percentage int       `json:"percentage"       yaml:"percentage"`
	Groups     []string  `json:"groups,omitempty" yaml:"groups,omitempty"`
}

// BusinessHoursConstraint limits rollout to business hours
type BusinessHoursConstraint struct {
	StartHour int      `json:"start_hour" yaml:"start_hour"` // 0-23
	EndHour   int      `json:"end_hour"   yaml:"end_hour"`   // 0-23
	Days      []string `json:"days"       yaml:"days"`       // ["monday", "tuesday", ...]
	TimeZone  string   `json:"timezone"   yaml:"timezone"`
}

// FlagEvaluation contains the result and metadata of flag evaluation
type FlagEvaluation struct {
	// Evaluation result
	Flag    FlagName          `json:"flag"`
	Value   FlagValue         `json:"value"`
	Context EvaluationContext `json:"context"`

	// Evaluation metadata
	EvaluationID string        `json:"evaluation_id"`
	RuleTrace    []RuleTrace   `json:"rule_trace,omitempty"`
	Duration     time.Duration `json:"duration"`
	CacheHit     bool          `json:"cache_hit"`
	Error        string        `json:"error,omitempty"`

	// Tracking
	TrackingData map[string]interface{} `json:"tracking_data,omitempty"`
	ExperimentID string                 `json:"experiment_id,omitempty"`
	VariantID    string                 `json:"variant_id,omitempty"`
}

// RuleTrace represents evaluation trace for debugging
type RuleTrace struct {
	RuleName   string           `json:"rule_name"`
	Matched    bool             `json:"matched"`
	Conditions []ConditionTrace `json:"conditions"`
	Value      interface{}      `json:"value,omitempty"`
	Duration   time.Duration    `json:"duration"`
}

// ConditionTrace represents condition evaluation trace
type ConditionTrace struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"`
	Expected  interface{} `json:"expected"`
	Actual    interface{} `json:"actual"`
	Matched   bool        `json:"matched"`
	Error     string      `json:"error,omitempty"`
}

// FlagMetrics contains metrics for flag usage and performance
type FlagMetrics struct {
	// Basic counters
	TotalEvaluations      int64   `json:"total_evaluations"`
	TotalEvaluationTime   int64   `json:"total_evaluation_time_ms"`
	AverageEvaluationTime float64 `json:"average_evaluation_time_ms"`

	// Cache metrics
	CacheHits    int64   `json:"cache_hits"`
	CacheMisses  int64   `json:"cache_misses"`
	CacheHitRate float64 `json:"cache_hit_rate"`

	// Flag-specific metrics
	FlagEvaluations map[FlagName]*FlagMetric `json:"flag_evaluations"`

	// Error metrics
	ErrorCount   int64            `json:"error_count"`
	ErrorRate    float64          `json:"error_rate"`
	ErrorsByType map[string]int64 `json:"errors_by_type"`

	// Timing
	LastUpdated time.Time `json:"last_updated"`
	StartTime   time.Time `json:"start_time"`
}

// FlagMetric contains metrics for a specific flag
type FlagMetric struct {
	Name FlagName `json:"name"`

	// Evaluation counts
	TotalEvaluations int64   `json:"total_evaluations"`
	TrueEvaluations  int64   `json:"true_evaluations"`
	FalseEvaluations int64   `json:"false_evaluations"`
	TrueRate         float64 `json:"true_rate"`

	// Rule metrics
	RuleMatches map[string]int64 `json:"rule_matches"`

	// Variant metrics (for A/B testing)
	VariantCounts map[string]int64 `json:"variant_counts,omitempty"`

	// Performance
	AverageEvaluationTime time.Duration `json:"average_evaluation_time"`
	MaxEvaluationTime     time.Duration `json:"max_evaluation_time"`

	// Timing
	FirstEvaluation time.Time `json:"first_evaluation"`
	LastEvaluation  time.Time `json:"last_evaluation"`
}

// ConfigurationSource represents where flags are loaded from
type ConfigurationSource string

const (
	SourceEnvironment ConfigurationSource = "environment"
	SourceFile        ConfigurationSource = "file"
	SourceDatabase    ConfigurationSource = "database"
	SourceRedis       ConfigurationSource = "redis"
	SourceHTTP        ConfigurationSource = "http"
)

// FlagConfiguration contains the complete flag configuration
type FlagConfiguration struct {
	// Configuration metadata
	Version     int                 `json:"version"     yaml:"version"`
	LoadedAt    time.Time           `json:"loaded_at"   yaml:"loaded_at"`
	LoadedFrom  ConfigurationSource `json:"loaded_from" yaml:"loaded_from"`
	Environment string              `json:"environment" yaml:"environment"`

	// Global settings
	GlobalSettings *GlobalFlagSettings `json:"global_settings,omitempty" yaml:"global_settings,omitempty"`

	// Flag definitions
	Flags map[FlagName]*FlagDefinition `json:"flags" yaml:"flags"`

	// Flag groups for organization
	Groups map[string]*FlagGroup `json:"groups,omitempty" yaml:"groups,omitempty"`

	// Experiments for A/B testing
	Experiments map[string]*Experiment `json:"experiments,omitempty" yaml:"experiments,omitempty"`

	// Configuration validation
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// GlobalFlagSettings contains global flag behavior settings
type GlobalFlagSettings struct {
	// Default behavior
	DefaultEnabled    bool          `json:"default_enabled"    yaml:"default_enabled"`
	DefaultCacheTTL   time.Duration `json:"default_cache_ttl"  yaml:"default_cache_ttl"`
	EvaluationTimeout time.Duration `json:"evaluation_timeout" yaml:"evaluation_timeout"`

	// Rollout defaults
	DefaultRolloutPercentage int             `json:"default_rollout_percentage" yaml:"default_rollout_percentage"`
	DefaultRolloutStrategy   RolloutStrategy `json:"default_rollout_strategy"   yaml:"default_rollout_strategy"`

	// Monitoring
	MetricsEnabled  bool          `json:"metrics_enabled"  yaml:"metrics_enabled"`
	MetricsInterval time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	TrackingEnabled bool          `json:"tracking_enabled" yaml:"tracking_enabled"`

	// Safety
	FailureMode           string        `json:"failure_mode"            yaml:"failure_mode"` // "fail_open" or "fail_closed"
	MaxEvaluationTime     time.Duration `json:"max_evaluation_time"     yaml:"max_evaluation_time"`
	CircuitBreakerEnabled bool          `json:"circuit_breaker_enabled" yaml:"circuit_breaker_enabled"`
}

// FlagGroup organizes related flags
type FlagGroup struct {
	Name        string     `json:"name"                  yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Flags       []FlagName `json:"flags"                 yaml:"flags"`
	Enabled     bool       `json:"enabled"               yaml:"enabled"`
	CreatedAt   time.Time  `json:"created_at"            yaml:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"            yaml:"updated_at"`
}

// Experiment represents an A/B testing experiment
type Experiment struct {
	// Experiment metadata
	ID          string `json:"id"                    yaml:"id"`
	Name        string `json:"name"                  yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Status      string `json:"status"                yaml:"status"` // "draft", "running", "paused", "completed"

	// Experiment configuration
	Flag     FlagName      `json:"flag"     yaml:"flag"`
	Variants []FlagVariant `json:"variants" yaml:"variants"`

	// Targeting
	AudienceFilter    *AudienceFilter `json:"audience_filter,omitempty" yaml:"audience_filter,omitempty"`
	TrafficAllocation int             `json:"traffic_allocation"        yaml:"traffic_allocation"` // Percentage 0-100

	// Timeline
	StartTime time.Time      `json:"start_time"         yaml:"start_time"`
	EndTime   *time.Time     `json:"end_time,omitempty" yaml:"end_time,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`

	// Metrics
	PrimaryMetric    string   `json:"primary_metric"              yaml:"primary_metric"`
	SecondaryMetrics []string `json:"secondary_metrics,omitempty" yaml:"secondary_metrics,omitempty"`

	// Results
	Results *ExperimentResults `json:"results,omitempty" yaml:"results,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by" yaml:"created_by"`
}

// AudienceFilter defines experiment audience targeting
type AudienceFilter struct {
	// User attributes
	UserAttributes map[string]interface{} `json:"user_attributes,omitempty" yaml:"user_attributes,omitempty"`

	// Geographic targeting
	Countries []string `json:"countries,omitempty" yaml:"countries,omitempty"`
	Regions   []string `json:"regions,omitempty"   yaml:"regions,omitempty"`

	// Platform targeting
	Platforms []string `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	Browsers  []string `json:"browsers,omitempty"  yaml:"browsers,omitempty"`

	// Custom conditions
	Conditions []FlagCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ExperimentResults contains experiment outcome data
type ExperimentResults struct {
	// Participant counts
	TotalParticipants int64            `json:"total_participants"`
	VariantCounts     map[string]int64 `json:"variant_counts"`

	// Metric results
	MetricResults map[string]*MetricResult `json:"metric_results"`

	// Statistical significance
	ConfidenceLevel   float64 `json:"confidence_level"`
	PValue            float64 `json:"p_value"`
	SignificantWinner string  `json:"significant_winner,omitempty"`

	// Timing
	AnalyzedAt time.Time `json:"analyzed_at"`
	DataWindow struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"data_window"`
}

// MetricResult contains results for a specific metric
type MetricResult struct {
	MetricName     string                    `json:"metric_name"`
	VariantResults map[string]*VariantResult `json:"variant_results"`
	Winner         string                    `json:"winner,omitempty"`
	Confidence     float64                   `json:"confidence"`
}

// VariantResult contains results for a specific variant
type VariantResult struct {
	Variant string  `json:"variant"`
	Count   int64   `json:"count"`
	Rate    float64 `json:"rate"`
	Mean    float64 `json:"mean,omitempty"`
	StdDev  float64 `json:"std_dev,omitempty"`
}

// FlagManager defines the main interface for feature flag management
type FlagManager interface {
	// Flag evaluation
	EvaluateFlag(ctx context.Context, flag FlagName, evalCtx *EvaluationContext) (*FlagValue, error)
	EvaluateAllFlags(
		ctx context.Context,
		evalCtx *EvaluationContext,
	) (map[FlagName]*FlagValue, error)
	EvaluateBooleanFlag(
		ctx context.Context,
		flag FlagName,
		evalCtx *EvaluationContext,
	) (bool, error)

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
	ValidateConfiguration(config *FlagConfiguration) error

	// Metrics and monitoring
	GetMetrics(ctx context.Context) (*FlagMetrics, error)
	GetFlagMetrics(ctx context.Context, flag FlagName) (*FlagMetric, error)
	ResetMetrics(ctx context.Context) error

	// Cache management
	InvalidateCache(ctx context.Context, flag FlagName) error
	ClearCache(ctx context.Context) error

	// Health check
	Health(ctx context.Context) error
}

// FlagStore defines the interface for flag persistence
type FlagStore interface {
	// Flag CRUD operations
	GetFlag(ctx context.Context, name FlagName) (*FlagDefinition, error)
	SaveFlag(ctx context.Context, flag *FlagDefinition) error
	DeleteFlag(ctx context.Context, name FlagName) error
	ListFlags(ctx context.Context) ([]*FlagDefinition, error)

	// Configuration operations
	GetConfiguration(ctx context.Context) (*FlagConfiguration, error)
	SaveConfiguration(ctx context.Context, config *FlagConfiguration) error

	// Experiment operations
	GetExperiment(ctx context.Context, id string) (*Experiment, error)
	SaveExperiment(ctx context.Context, experiment *Experiment) error
	ListExperiments(ctx context.Context, status string) ([]*Experiment, error)

	// Health check
	Health(ctx context.Context) error
}

// EvaluationEngine defines the interface for flag evaluation logic
type EvaluationEngine interface {
	// Core evaluation
	Evaluate(
		ctx context.Context,
		flag *FlagDefinition,
		evalCtx *EvaluationContext,
	) (*FlagValue, error)

	// Rule evaluation
	EvaluateRules(
		ctx context.Context,
		rules []FlagRule,
		evalCtx *EvaluationContext,
	) (*FlagRule, error)
	EvaluateConditions(
		ctx context.Context,
		conditions []FlagCondition,
		evalCtx *EvaluationContext,
	) (bool, error)

	// Rollout evaluation
	EvaluateRollout(
		ctx context.Context,
		config *RolloutConfig,
		evalCtx *EvaluationContext,
	) (bool, error)

	// Variant selection
	SelectVariant(
		ctx context.Context,
		variants []FlagVariant,
		evalCtx *EvaluationContext,
	) (*FlagVariant, error)
}

// MetricsCollector defines the interface for collecting flag metrics
type MetricsCollector interface {
	// Record evaluation
	RecordEvaluation(ctx context.Context, eval *FlagEvaluation)
	RecordError(ctx context.Context, flag FlagName, err error)

	// Get metrics
	GetMetrics(ctx context.Context) (*FlagMetrics, error)
	GetFlagMetrics(ctx context.Context, flag FlagName) (*FlagMetric, error)

	// Reset metrics
	Reset(ctx context.Context) error
}

// CacheProvider defines the interface for flag evaluation caching
type CacheProvider interface {
	// Cache operations
	Get(ctx context.Context, key string) (*FlagValue, error)
	Set(ctx context.Context, key string, value *FlagValue, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error

	// Cache statistics
	Stats(ctx context.Context) (map[string]interface{}, error)
}

// ConfigurationLoader defines the interface for loading flag configuration
type ConfigurationLoader interface {
	// Load configuration from various sources
	LoadFromFile(ctx context.Context, path string) (*FlagConfiguration, error)
	LoadFromEnvironment(ctx context.Context) (*FlagConfiguration, error)
	LoadFromDatabase(ctx context.Context) (*FlagConfiguration, error)
	LoadFromHTTP(ctx context.Context, url string) (*FlagConfiguration, error)

	// Watch for changes
	Watch(ctx context.Context, callback func(*FlagConfiguration)) error
	StopWatching(ctx context.Context) error
}

// EventHandler defines the interface for flag event handling
type EventHandler interface {
	// Handle flag events
	OnFlagEvaluated(ctx context.Context, eval *FlagEvaluation)
	OnFlagChanged(ctx context.Context, flag *FlagDefinition, oldFlag *FlagDefinition)
	OnConfigurationChanged(ctx context.Context, config *FlagConfiguration)
	OnExperimentStarted(ctx context.Context, experiment *Experiment)
	OnExperimentEnded(ctx context.Context, experiment *Experiment, results *ExperimentResults)
}
