package features

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"gopkg.in/yaml.v3"
)

// configurationLoader implements the ConfigurationLoader interface
type configurationLoader struct {
	dbPool    interface{} // Compatible with different pool types
	client    *http.Client
	envPrefix string
}

// CreateConfigurationLoader creates a new configuration loader
func CreateConfigurationLoader(dbPool interface{}) (ConfigurationLoader, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database pool is required")
	}

	return &configurationLoader{
		dbPool: dbPool,
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		envPrefix: "MCP_PORTAL_FEATURE_",
	}, nil
}

// LoadFromFile loads configuration from a YAML or JSON file
func (l *configurationLoader) LoadFromFile(
	ctx context.Context,
	path string,
) (*FlagConfiguration, error) {
	if path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(path))
	var config FlagConfiguration

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON configuration: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	// Set metadata
	config.LoadedAt = time.Now()
	config.LoadedFrom = SourceFile

	// Initialize maps if nil
	if config.Flags == nil {
		config.Flags = make(map[FlagName]*FlagDefinition)
	}
	if config.Groups == nil {
		config.Groups = make(map[string]*FlagGroup)
	}
	if config.Experiments == nil {
		config.Experiments = make(map[string]*Experiment)
	}

	return &config, nil
}

// LoadFromEnvironment loads configuration from environment variables
func (l *configurationLoader) LoadFromEnvironment(ctx context.Context) (*FlagConfiguration, error) {
	config := &FlagConfiguration{
		Version:     1,
		LoadedAt:    time.Now(),
		LoadedFrom:  SourceEnvironment,
		Environment: os.Getenv("MCP_PORTAL_ENV"),
		Flags:       make(map[FlagName]*FlagDefinition),
		Groups:      make(map[string]*FlagGroup),
		Experiments: make(map[string]*Experiment),
		Valid:       true,
	}

	if config.Environment == "" {
		config.Environment = "development"
	}

	// Load global settings from environment
	config.GlobalSettings = l.loadGlobalSettingsFromEnv()

	// Load OAuth flags from environment
	oauthFlags := l.loadOAuthFlagsFromEnv()
	for name, flag := range oauthFlags {
		config.Flags[name] = flag
	}

	// Load custom flags from environment (MCP_PORTAL_FEATURE_CUSTOM_*)
	customFlags := l.loadCustomFlagsFromEnv()
	for name, flag := range customFlags {
		config.Flags[name] = flag
	}

	return config, nil
}

// LoadFromDatabase loads configuration from the database
func (l *configurationLoader) LoadFromDatabase(ctx context.Context) (*FlagConfiguration, error) {
	// For now, return a default configuration since we're using mock adapters
	// In a real implementation, this would use the actual database connection
	return l.createDefaultDatabaseConfiguration(ctx)

	// Load configuration from database
	query := `
		SELECT
			version,
			configuration,
			created_at,
			updated_at
		FROM feature_flag_configuration
		WHERE active = true
		ORDER BY version DESC
		LIMIT 1
	`

	var version int
	var configData []byte
	var createdAt, updatedAt time.Time

	err = conn.QueryRow(ctx, query).Scan(&version, &configData, &createdAt, &updatedAt)
	if err != nil {
		// If no configuration exists, create a default one
		if err.Error() == "no rows in result set" {
			return l.createDefaultDatabaseConfiguration(ctx)
		}
		return nil, fmt.Errorf("failed to load configuration from database: %w", err)
	}

	// Parse configuration
	var config FlagConfiguration
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Set metadata
	config.Version = version
	config.LoadedAt = time.Now()
	config.LoadedFrom = SourceDatabase

	// Load individual flags from database
	flags, err := l.loadFlagsFromDatabase(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load flags from database: %w", err)
	}

	if config.Flags == nil {
		config.Flags = make(map[FlagName]*FlagDefinition)
	}

	for _, flag := range flags {
		config.Flags[flag.Name] = flag
	}

	// Load experiments from database
	experiments, err := l.loadExperimentsFromDatabase(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to load experiments from database: %w", err)
	}

	if config.Experiments == nil {
		config.Experiments = make(map[string]*Experiment)
	}

	for _, experiment := range experiments {
		config.Experiments[experiment.ID] = experiment
	}

	return &config, nil
}

// LoadFromHTTP loads configuration from an HTTP endpoint
func (l *configurationLoader) LoadFromHTTP(
	ctx context.Context,
	url string,
) (*FlagConfiguration, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MCP-Portal-FeatureFlags/1.0")

	// Make request
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Read response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse configuration
	var config FlagConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse HTTP response: %w", err)
	}

	// Set metadata
	config.LoadedAt = time.Now()
	config.LoadedFrom = SourceHTTP

	// Initialize maps if nil
	if config.Flags == nil {
		config.Flags = make(map[FlagName]*FlagDefinition)
	}
	if config.Groups == nil {
		config.Groups = make(map[string]*FlagGroup)
	}
	if config.Experiments == nil {
		config.Experiments = make(map[string]*Experiment)
	}

	return &config, nil
}

// Watch for configuration changes (placeholder implementation)
func (l *configurationLoader) Watch(ctx context.Context, callback func(*FlagConfiguration)) error {
	// This would implement file watching, database change notifications, etc.
	// For now, we'll return not implemented
	return fmt.Errorf("configuration watching not yet implemented")
}

// StopWatching stops watching for configuration changes
func (l *configurationLoader) StopWatching(ctx context.Context) error {
	// Stop any active watchers
	return nil
}

// Helper methods

func (l *configurationLoader) loadGlobalSettingsFromEnv() *GlobalFlagSettings {
	settings := &GlobalFlagSettings{
		DefaultEnabled: parseBoolEnv("MCP_PORTAL_FLAGS_DEFAULT_ENABLED", false),
		DefaultCacheTTL: parseDurationEnv(
			"MCP_PORTAL_FLAGS_DEFAULT_CACHE_TTL",
			time.Minute*5,
		),
		EvaluationTimeout: parseDurationEnv(
			"MCP_PORTAL_FLAGS_EVALUATION_TIMEOUT",
			time.Second*1,
		),
		DefaultRolloutPercentage: parseIntEnv("MCP_PORTAL_FLAGS_DEFAULT_ROLLOUT_PERCENTAGE", 0),
		DefaultRolloutStrategy: RolloutStrategy(
			parseStringEnv("MCP_PORTAL_FLAGS_DEFAULT_ROLLOUT_STRATEGY", string(RolloutPercentage)),
		),
		MetricsEnabled: parseBoolEnv("MCP_PORTAL_FLAGS_METRICS_ENABLED", true),
		MetricsInterval: parseDurationEnv(
			"MCP_PORTAL_FLAGS_METRICS_INTERVAL",
			time.Minute*1,
		),
		TrackingEnabled: parseBoolEnv("MCP_PORTAL_FLAGS_TRACKING_ENABLED", true),
		FailureMode:     parseStringEnv("MCP_PORTAL_FLAGS_FAILURE_MODE", "fail_closed"),
		MaxEvaluationTime: parseDurationEnv(
			"MCP_PORTAL_FLAGS_MAX_EVALUATION_TIME",
			time.Second*2,
		),
		CircuitBreakerEnabled: parseBoolEnv("MCP_PORTAL_FLAGS_CIRCUIT_BREAKER_ENABLED", true),
	}

	return settings
}

func (l *configurationLoader) loadOAuthFlagsFromEnv() map[FlagName]*FlagDefinition {
	now := time.Now()
	flags := make(map[FlagName]*FlagDefinition)

	// Define OAuth flags with their environment variable mappings
	oauthFlags := map[FlagName]struct {
		envKey      string
		description string
		tags        []string
	}{
		FlagOAuthEnabled: {
			"OAUTH_ENABLED",
			"Master switch for OAuth functionality",
			[]string{"oauth", "security"},
		},
		FlagOAuthAuto401: {
			"OAUTH_AUTO_401",
			"Automatic 401 detection and token refresh",
			[]string{"oauth", "automation"},
		},
		FlagOAuthDCR: {
			"OAUTH_DCR",
			"Dynamic Client Registration support",
			[]string{"oauth", "dcr"},
		},
		FlagOAuthProviderGitHub: {
			"OAUTH_PROVIDER_GITHUB",
			"GitHub OAuth provider support",
			[]string{"oauth", "provider", "github"},
		},
		FlagOAuthProviderGoogle: {
			"OAUTH_PROVIDER_GOOGLE",
			"Google OAuth provider support",
			[]string{"oauth", "provider", "google"},
		},
		FlagOAuthProviderMicrosoft: {
			"OAUTH_PROVIDER_MICROSOFT",
			"Microsoft OAuth provider support",
			[]string{"oauth", "provider", "microsoft"},
		},
		FlagOAuthDockerSecrets: {
			"OAUTH_DOCKER_SECRETS",
			"Docker Desktop secrets integration",
			[]string{"oauth", "docker", "secrets"},
		},
		FlagOAuthTokenRefresh: {
			"OAUTH_TOKEN_REFRESH",
			"Automatic token refresh",
			[]string{"oauth", "tokens"},
		},
		FlagOAuthTokenStorage: {
			"OAUTH_TOKEN_STORAGE",
			"Token storage functionality",
			[]string{"oauth", "tokens", "storage"},
		},
		FlagOAuthJWTValidation: {
			"OAUTH_JWT_VALIDATION",
			"JWT token validation",
			[]string{"oauth", "jwt", "security"},
		},
		FlagOAuthHTTPSRequired: {
			"OAUTH_HTTPS_REQUIRED",
			"Require HTTPS for OAuth",
			[]string{"oauth", "security", "https"},
		},
		FlagOAuthAuditLogging: {
			"OAUTH_AUDIT_LOGGING",
			"OAuth audit logging",
			[]string{"oauth", "audit", "logging"},
		},
		FlagOAuthMetrics: {
			"OAUTH_METRICS",
			"OAuth metrics collection",
			[]string{"oauth", "metrics"},
		},
		FlagOAuthKeyRotation: {
			"OAUTH_KEY_ROTATION",
			"Automatic key rotation",
			[]string{"oauth", "security", "rotation"},
		},
	}

	for flagName, config := range oauthFlags {
		envKey := l.envPrefix + config.envKey

		// Check if flag is enabled via environment
		enabled := parseBoolEnv(envKey, false)

		// Check for rollout percentage
		rolloutKey := envKey + "_ROLLOUT_PERCENTAGE"
		rolloutPercentage := parseIntEnv(rolloutKey, 0)

		// Check for user overrides
		userOverridesKey := envKey + "_USER_OVERRIDES"
		userOverrides := parseMapEnv(userOverridesKey)

		// Check for server overrides
		serverOverridesKey := envKey + "_SERVER_OVERRIDES"
		serverOverrides := parseMapEnv(serverOverridesKey)

		flags[flagName] = &FlagDefinition{
			Name:              flagName,
			Type:              FlagTypeBoolean,
			Description:       config.description,
			Enabled:           enabled,
			DefaultValue:      enabled,
			RolloutPercentage: rolloutPercentage,
			UserOverrides:     userOverrides,
			ServerOverrides:   serverOverrides,
			CreatedAt:         now,
			UpdatedAt:         now,
			Version:           1,
			Tags:              config.tags,
		}
	}

	return flags
}

func (l *configurationLoader) loadCustomFlagsFromEnv() map[FlagName]*FlagDefinition {
	now := time.Now()
	flags := make(map[FlagName]*FlagDefinition)

	// Look for custom flags with prefix MCP_PORTAL_FEATURE_CUSTOM_
	customPrefix := l.envPrefix + "CUSTOM_"

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, customPrefix) {
			continue
		}

		// Parse environment variable
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey := parts[0]
		value := parts[1]

		// Extract flag name
		flagNameStr := strings.TrimPrefix(envKey, customPrefix)
		flagNameStr = strings.ToLower(flagNameStr)

		// Skip if this is a modifier (like _ROLLOUT_PERCENTAGE)
		if strings.Contains(flagNameStr, "_rollout_percentage") ||
			strings.Contains(flagNameStr, "_user_overrides") ||
			strings.Contains(flagNameStr, "_server_overrides") {
			continue
		}

		flagName := FlagName(flagNameStr)

		// Determine flag type and value
		var flagType FlagType
		var defaultValue interface{}

		if val, err := strconv.ParseBool(value); err == nil {
			flagType = FlagTypeBoolean
			defaultValue = val
		} else if val, err := strconv.ParseInt(value, 10, 64); err == nil {
			flagType = FlagTypePercentage
			defaultValue = val
		} else {
			// Treat as string/variant
			flagType = FlagTypeVariant
			defaultValue = value
		}

		// Check for additional configuration
		rolloutPercentage := parseIntEnv(envKey+"_ROLLOUT_PERCENTAGE", 0)
		userOverrides := parseMapEnv(envKey + "_USER_OVERRIDES")
		serverOverrides := parseMapEnv(envKey + "_SERVER_OVERRIDES")

		flags[flagName] = &FlagDefinition{
			Name:              flagName,
			Type:              flagType,
			Description:       fmt.Sprintf("Custom flag: %s", flagName),
			Enabled:           true,
			DefaultValue:      defaultValue,
			RolloutPercentage: rolloutPercentage,
			UserOverrides:     userOverrides,
			ServerOverrides:   serverOverrides,
			CreatedAt:         now,
			UpdatedAt:         now,
			Version:           1,
			Tags:              []string{"custom"},
		}
	}

	return flags
}

func (l *configurationLoader) createDefaultDatabaseConfiguration(
	ctx context.Context,
) (*FlagConfiguration, error) {
	config := &FlagConfiguration{
		Version:     1,
		LoadedAt:    time.Now(),
		LoadedFrom:  SourceDatabase,
		Environment: os.Getenv("MCP_PORTAL_ENV"),
		GlobalSettings: &GlobalFlagSettings{
			DefaultEnabled:           false,
			DefaultCacheTTL:          time.Minute * 5,
			EvaluationTimeout:        time.Second * 1,
			DefaultRolloutPercentage: 0,
			DefaultRolloutStrategy:   RolloutPercentage,
			MetricsEnabled:           true,
			MetricsInterval:          time.Minute * 1,
			TrackingEnabled:          true,
			FailureMode:              "fail_closed",
			MaxEvaluationTime:        time.Second * 2,
			CircuitBreakerEnabled:    true,
		},
		Flags:       make(map[FlagName]*FlagDefinition),
		Groups:      make(map[string]*FlagGroup),
		Experiments: make(map[string]*Experiment),
		Valid:       true,
	}

	if config.Environment == "" {
		config.Environment = "development"
	}

	return config, nil
}

func (l *configurationLoader) loadFlagsFromDatabase(
	ctx context.Context,
	conn interface{},
) ([]*FlagDefinition, error) {
	query := `
		SELECT
			name,
			type,
			description,
			enabled,
			default_value,
			rollout_percentage,
			user_overrides,
			server_overrides,
			variants,
			rules,
			created_at,
			updated_at,
			created_by,
			version,
			tags,
			deprecated
		FROM feature_flags
		WHERE deleted_at IS NULL
		ORDER BY name
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flags: %w", err)
	}
	defer rows.Close()

	var flags []*FlagDefinition

	for rows.Next() {
		var flag FlagDefinition
		var defaultValueJSON, userOverridesJSON, serverOverridesJSON, variantsJSON, rulesJSON, tagsJSON []byte

		err := rows.Scan(
			&flag.Name,
			&flag.Type,
			&flag.Description,
			&flag.Enabled,
			&defaultValueJSON,
			&flag.RolloutPercentage,
			&userOverridesJSON,
			&serverOverridesJSON,
			&variantsJSON,
			&rulesJSON,
			&flag.CreatedAt,
			&flag.UpdatedAt,
			&flag.CreatedBy,
			&flag.Version,
			&tagsJSON,
			&flag.Deprecated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flag row: %w", err)
		}

		// Parse JSON fields
		if len(defaultValueJSON) > 0 {
			if err := json.Unmarshal(defaultValueJSON, &flag.DefaultValue); err != nil {
				return nil, fmt.Errorf(
					"failed to parse default value for flag %s: %w",
					flag.Name,
					err,
				)
			}
		}

		if len(userOverridesJSON) > 0 {
			if err := json.Unmarshal(userOverridesJSON, &flag.UserOverrides); err != nil {
				return nil, fmt.Errorf(
					"failed to parse user overrides for flag %s: %w",
					flag.Name,
					err,
				)
			}
		}

		if len(serverOverridesJSON) > 0 {
			if err := json.Unmarshal(serverOverridesJSON, &flag.ServerOverrides); err != nil {
				return nil, fmt.Errorf(
					"failed to parse server overrides for flag %s: %w",
					flag.Name,
					err,
				)
			}
		}

		if len(variantsJSON) > 0 {
			if err := json.Unmarshal(variantsJSON, &flag.Variants); err != nil {
				return nil, fmt.Errorf("failed to parse variants for flag %s: %w", flag.Name, err)
			}
		}

		if len(rulesJSON) > 0 {
			if err := json.Unmarshal(rulesJSON, &flag.Rules); err != nil {
				return nil, fmt.Errorf("failed to parse rules for flag %s: %w", flag.Name, err)
			}
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &flag.Tags); err != nil {
				return nil, fmt.Errorf("failed to parse tags for flag %s: %w", flag.Name, err)
			}
		}

		flags = append(flags, &flag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate flag rows: %w", err)
	}

	return flags, nil
}

func (l *configurationLoader) loadExperimentsFromDatabase(
	ctx context.Context,
	conn database.Connection,
) ([]*Experiment, error) {
	query := `
		SELECT
			id,
			name,
			description,
			status,
			flag_name,
			variants,
			audience_filter,
			traffic_allocation,
			start_time,
			end_time,
			duration_seconds,
			primary_metric,
			secondary_metrics,
			results,
			created_at,
			updated_at,
			created_by
		FROM feature_flag_experiments
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query experiments: %w", err)
	}
	defer rows.Close()

	var experiments []*Experiment

	for rows.Next() {
		var experiment Experiment
		var variantsJSON, audienceFilterJSON, secondaryMetricsJSON, resultsJSON []byte
		var endTime *time.Time
		var durationSeconds *int64

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan experiment row: %w", err)
		}

		// Handle optional fields
		experiment.EndTime = endTime
		if durationSeconds != nil {
			duration := time.Duration(*durationSeconds) * time.Second
			experiment.Duration = &duration
		}

		// Parse JSON fields
		if len(variantsJSON) > 0 {
			if err := json.Unmarshal(variantsJSON, &experiment.Variants); err != nil {
				return nil, fmt.Errorf(
					"failed to parse variants for experiment %s: %w",
					experiment.ID,
					err,
				)
			}
		}

		if len(audienceFilterJSON) > 0 {
			if err := json.Unmarshal(audienceFilterJSON, &experiment.AudienceFilter); err != nil {
				return nil, fmt.Errorf(
					"failed to parse audience filter for experiment %s: %w",
					experiment.ID,
					err,
				)
			}
		}

		if len(secondaryMetricsJSON) > 0 {
			if err := json.Unmarshal(secondaryMetricsJSON, &experiment.SecondaryMetrics); err != nil {
				return nil, fmt.Errorf(
					"failed to parse secondary metrics for experiment %s: %w",
					experiment.ID,
					err,
				)
			}
		}

		if len(resultsJSON) > 0 {
			if err := json.Unmarshal(resultsJSON, &experiment.Results); err != nil {
				return nil, fmt.Errorf(
					"failed to parse results for experiment %s: %w",
					experiment.ID,
					err,
				)
			}
		}

		experiments = append(experiments, &experiment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate experiment rows: %w", err)
	}

	return experiments, nil
}

// Utility functions for parsing environment variables

func parseBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func parseIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func parseStringEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func parseMapEnv(key string) map[string]interface{} {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}

	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		// If JSON parsing fails, treat as simple key=value pairs
		result = make(map[string]interface{})
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		}
	}

	return result
}
