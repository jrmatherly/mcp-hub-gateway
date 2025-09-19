package features

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// evaluationEngine implements the EvaluationEngine interface
type evaluationEngine struct {
	// Configuration
	maxEvaluationTime time.Duration
	debugEnabled      bool
}

// CreateEvaluationEngine creates a new flag evaluation engine
func CreateEvaluationEngine() (EvaluationEngine, error) {
	return &evaluationEngine{
		maxEvaluationTime: time.Second * 2,
		debugEnabled:      false,
	}, nil
}

// Evaluate evaluates a flag definition against an evaluation context
func (e *evaluationEngine) Evaluate(
	ctx context.Context,
	flag *FlagDefinition,
	evalCtx *EvaluationContext,
) (*FlagValue, error) {
	startTime := time.Now()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.maxEvaluationTime)
	defer cancel()

	// Check if flag is disabled
	if !flag.Enabled {
		return &FlagValue{
			Name:        flag.Name,
			Type:        flag.Type,
			Enabled:     false,
			Value:       flag.DefaultValue,
			Reason:      "flag_disabled",
			EvaluatedAt: time.Now(),
		}, nil
	}

	var result *FlagValue
	var err error

	// Check for user-specific overrides first
	if override, exists := e.checkUserOverride(flag, evalCtx); exists {
		result = &FlagValue{
			Name:        flag.Name,
			Type:        flag.Type,
			Enabled:     e.toBool(override),
			Value:       override,
			Reason:      "user_override",
			EvaluatedAt: time.Now(),
		}
	} else if override, exists := e.checkServerOverride(flag, evalCtx); exists {
		// Check for server-specific overrides
		result = &FlagValue{
			Name:        flag.Name,
			Type:        flag.Type,
			Enabled:     e.toBool(override),
			Value:       override,
			Reason:      "server_override",
			EvaluatedAt: time.Now(),
		}
	} else {
		// Evaluate rules and rollout
		result, err = e.evaluateRulesAndRollout(timeoutCtx, flag, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rules and rollout: %w", err)
		}
	}

	// Check for timeout
	select {
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("evaluation timeout after %v", time.Since(startTime))
	default:
	}

	return result, nil
}

// EvaluateRules evaluates a set of flag rules against the context
func (e *evaluationEngine) EvaluateRules(
	ctx context.Context,
	rules []FlagRule,
	evalCtx *EvaluationContext,
) (*FlagRule, error) {
	// Sort rules by priority (higher priority first)
	sortedRules := make([]FlagRule, len(rules))
	copy(sortedRules, rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	// Evaluate rules in priority order
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		matched, err := e.EvaluateConditions(ctx, rule.Conditions, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %s: %w", rule.Name, err)
		}

		if matched {
			return &rule, nil
		}
	}

	return nil, nil // No rule matched
}

// EvaluateConditions evaluates a set of conditions with AND logic
func (e *evaluationEngine) EvaluateConditions(
	ctx context.Context,
	conditions []FlagCondition,
	evalCtx *EvaluationContext,
) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must be true (AND logic)
	for _, condition := range conditions {
		matched, err := e.evaluateCondition(ctx, &condition, evalCtx)
		if err != nil {
			return false, fmt.Errorf(
				"failed to evaluate condition %s: %w",
				condition.Attribute,
				err,
			)
		}

		// Apply negation if specified
		if condition.Negate {
			matched = !matched
		}

		if !matched {
			return false, nil
		}
	}

	return true, nil
}

// EvaluateRollout evaluates rollout configuration
func (e *evaluationEngine) EvaluateRollout(
	ctx context.Context,
	config *RolloutConfig,
	evalCtx *EvaluationContext,
) (bool, error) {
	if config == nil {
		return true, nil
	}

	switch config.Strategy {
	case RolloutPercentage:
		return e.evaluatePercentageRollout(config, evalCtx)
	case RolloutCanary:
		return e.evaluateCanaryRollout(config, evalCtx)
	case RolloutScheduled:
		return e.evaluateScheduledRollout(config, evalCtx)
	case RolloutManual:
		return false, nil // Manual rollout requires explicit enabling
	default:
		return false, fmt.Errorf("unknown rollout strategy: %s", config.Strategy)
	}
}

// SelectVariant selects a variant based on weights and context
func (e *evaluationEngine) SelectVariant(
	ctx context.Context,
	variants []FlagVariant,
	evalCtx *EvaluationContext,
) (*FlagVariant, error) {
	if len(variants) == 0 {
		return nil, nil
	}

	// Filter enabled variants
	enabledVariants := make([]FlagVariant, 0, len(variants))
	for _, variant := range variants {
		if variant.Enabled {
			enabledVariants = append(enabledVariants, variant)
		}
	}

	if len(enabledVariants) == 0 {
		return nil, nil
	}

	// Generate deterministic hash based on user ID
	hash := e.generateUserHash(evalCtx.UserID.String())

	// Calculate cumulative weights
	totalWeight := 0
	for _, variant := range enabledVariants {
		totalWeight += variant.Weight
	}

	if totalWeight == 0 {
		// If no weights, select first variant
		return &enabledVariants[0], nil
	}

	// Use hash to select variant
	bucket := hash % totalWeight
	cumulativeWeight := 0

	for _, variant := range enabledVariants {
		cumulativeWeight += variant.Weight
		if bucket < cumulativeWeight {
			return &variant, nil
		}
	}

	// Fallback to last variant
	return &enabledVariants[len(enabledVariants)-1], nil
}

// Helper methods

func (e *evaluationEngine) evaluateRulesAndRollout(
	ctx context.Context,
	flag *FlagDefinition,
	evalCtx *EvaluationContext,
) (*FlagValue, error) {
	// First, check if any rules match
	if len(flag.Rules) > 0 {
		matchedRule, err := e.EvaluateRules(ctx, flag.Rules, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("rule evaluation failed: %w", err)
		}

		if matchedRule != nil {
			// Rule matched, use rule value
			value := &FlagValue{
				Name:        flag.Name,
				Type:        flag.Type,
				Enabled:     e.toBool(matchedRule.Value),
				Value:       matchedRule.Value,
				Variant:     matchedRule.Variant,
				Reason:      fmt.Sprintf("rule_match:%s", matchedRule.Name),
				RuleMatched: matchedRule.Name,
				EvaluatedAt: time.Now(),
			}

			return value, nil
		}
	}

	// No rules matched, check rollout
	if flag.RolloutConfig != nil {
		inRollout, err := e.EvaluateRollout(ctx, flag.RolloutConfig, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("rollout evaluation failed: %w", err)
		}

		if !inRollout {
			// Not in rollout, return default value
			return &FlagValue{
				Name:        flag.Name,
				Type:        flag.Type,
				Enabled:     e.toBool(flag.DefaultValue),
				Value:       flag.DefaultValue,
				Reason:      "rollout_excluded",
				EvaluatedAt: time.Now(),
			}, nil
		}
	}

	// Check percentage rollout
	if flag.RolloutPercentage > 0 && flag.RolloutPercentage < 100 {
		inPercentage := e.evaluatePercentageRolloutSimple(flag.RolloutPercentage, evalCtx)
		if !inPercentage {
			return &FlagValue{
				Name:        flag.Name,
				Type:        flag.Type,
				Enabled:     e.toBool(flag.DefaultValue),
				Value:       flag.DefaultValue,
				Reason:      "percentage_excluded",
				EvaluatedAt: time.Now(),
			}, nil
		}
	}

	// Check if we have variants to select from
	if len(flag.Variants) > 0 {
		variant, err := e.SelectVariant(ctx, flag.Variants, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("variant selection failed: %w", err)
		}

		if variant != nil {
			return &FlagValue{
				Name:        flag.Name,
				Type:        flag.Type,
				Enabled:     e.toBool(variant.Value),
				Value:       variant.Value,
				Variant:     variant.Name,
				Reason:      "variant_selected",
				EvaluatedAt: time.Now(),
			}, nil
		}
	}

	// Default case - use flag's default value
	return &FlagValue{
		Name:        flag.Name,
		Type:        flag.Type,
		Enabled:     e.toBool(flag.DefaultValue),
		Value:       flag.DefaultValue,
		Reason:      "default_value",
		EvaluatedAt: time.Now(),
	}, nil
}

func (e *evaluationEngine) checkUserOverride(
	flag *FlagDefinition,
	evalCtx *EvaluationContext,
) (interface{}, bool) {
	if flag.UserOverrides == nil {
		return nil, false
	}

	userID := evalCtx.UserID.String()
	if override, exists := flag.UserOverrides[userID]; exists {
		return override, true
	}

	return nil, false
}

func (e *evaluationEngine) checkServerOverride(
	flag *FlagDefinition,
	evalCtx *EvaluationContext,
) (interface{}, bool) {
	if flag.ServerOverrides == nil || evalCtx.ServerName == "" {
		return nil, false
	}

	if override, exists := flag.ServerOverrides[evalCtx.ServerName]; exists {
		return override, true
	}

	return nil, false
}

func (e *evaluationEngine) evaluateCondition(
	ctx context.Context,
	condition *FlagCondition,
	evalCtx *EvaluationContext,
) (bool, error) {
	_ = ctx // Context reserved for future use (logging, tracing, cancellation)
	// Get the actual value for the attribute
	actualValue, err := e.getAttributeValue(condition.Attribute, evalCtx)
	if err != nil {
		return false, fmt.Errorf("failed to get attribute value: %w", err)
	}

	// Evaluate based on operator
	switch condition.Operator {
	case OpEquals:
		return e.compareValues(actualValue, condition.Value, "equals"), nil
	case OpNotEquals:
		return !e.compareValues(actualValue, condition.Value, "equals"), nil
	case OpContains:
		return e.compareValues(actualValue, condition.Value, "contains"), nil
	case OpNotContains:
		return !e.compareValues(actualValue, condition.Value, "contains"), nil
	case OpStartsWith:
		return e.compareValues(actualValue, condition.Value, "starts_with"), nil
	case OpEndsWith:
		return e.compareValues(actualValue, condition.Value, "ends_with"), nil
	case OpGreaterThan:
		return e.compareValues(actualValue, condition.Value, "greater_than"), nil
	case OpLessThan:
		return e.compareValues(actualValue, condition.Value, "less_than"), nil
	case OpGreaterEqual:
		return e.compareValues(actualValue, condition.Value, "greater_equal"), nil
	case OpLessEqual:
		return e.compareValues(actualValue, condition.Value, "less_equal"), nil
	case OpIn:
		return e.isValueInList(actualValue, condition.Values), nil
	case OpNotIn:
		return !e.isValueInList(actualValue, condition.Values), nil
	case OpRegexMatch:
		return e.matchesRegex(actualValue, condition.Value)
	case OpPercentage:
		return e.matchesPercentage(actualValue, condition.Value, evalCtx), nil
	case OpVersionMatch:
		return e.matchesVersion(actualValue, condition.Value), nil
	case OpDateAfter:
		return e.compareDates(actualValue, condition.Value, "after"), nil
	case OpDateBefore:
		return e.compareDates(actualValue, condition.Value, "before"), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

func (e *evaluationEngine) getAttributeValue(
	attribute string,
	evalCtx *EvaluationContext,
) (interface{}, error) {
	switch attribute {
	case "user_id":
		return evalCtx.UserID.String(), nil
	case "tenant_id":
		return evalCtx.TenantID, nil
	case "server_name":
		return evalCtx.ServerName, nil
	case "server_tags":
		return evalCtx.ServerTags, nil
	case "request_id":
		return evalCtx.RequestID, nil
	case "remote_addr":
		return evalCtx.RemoteAddr, nil
	case "user_agent":
		return evalCtx.UserAgent, nil
	case "environment":
		return evalCtx.Environment, nil
	case "timestamp":
		return evalCtx.Timestamp, nil
	default:
		// Check custom attributes
		if evalCtx.Attributes != nil {
			if value, exists := evalCtx.Attributes[attribute]; exists {
				return value, nil
			}
		}

		// Check headers
		if evalCtx.Headers != nil {
			if value, exists := evalCtx.Headers[attribute]; exists {
				return value, nil
			}
		}

		return nil, fmt.Errorf("unknown attribute: %s", attribute)
	}
}

func (e *evaluationEngine) compareValues(actual, expected interface{}, operator string) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	switch operator {
	case "equals":
		return actualStr == expectedStr
	case "contains":
		return strings.Contains(actualStr, expectedStr)
	case "starts_with":
		return strings.HasPrefix(actualStr, expectedStr)
	case "ends_with":
		return strings.HasSuffix(actualStr, expectedStr)
	case "greater_than":
		return e.compareNumeric(actual, expected, ">")
	case "less_than":
		return e.compareNumeric(actual, expected, "<")
	case "greater_equal":
		return e.compareNumeric(actual, expected, ">=")
	case "less_equal":
		return e.compareNumeric(actual, expected, "<=")
	default:
		return false
	}
}

func (e *evaluationEngine) compareNumeric(actual, expected interface{}, operator string) bool {
	actualFloat, err1 := e.toFloat64(actual)
	expectedFloat, err2 := e.toFloat64(expected)

	if err1 != nil || err2 != nil {
		return false
	}

	switch operator {
	case ">":
		return actualFloat > expectedFloat
	case "<":
		return actualFloat < expectedFloat
	case ">=":
		return actualFloat >= expectedFloat
	case "<=":
		return actualFloat <= expectedFloat
	default:
		return false
	}
}

func (e *evaluationEngine) isValueInList(value interface{}, list []interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	for _, item := range list {
		if fmt.Sprintf("%v", item) == valueStr {
			return true
		}
	}
	return false
}

func (e *evaluationEngine) matchesRegex(value, pattern interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	regex, err := regexp.Compile(patternStr)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return regex.MatchString(valueStr), nil
}

func (e *evaluationEngine) matchesPercentage(
	_ interface{}, percentage interface{},
	evalCtx *EvaluationContext,
) bool {
	percentageInt, err := e.toInt(percentage)
	if err != nil {
		return false
	}

	// Use user ID for consistent bucketing
	hash := e.generateUserHash(evalCtx.UserID.String())
	bucket := hash % 100

	return bucket < percentageInt
}

func (e *evaluationEngine) matchesVersion(actual, expected interface{}) bool {
	// Simple version comparison - in practice, you'd use a proper semver library
	actualStr := strings.TrimSpace(fmt.Sprintf("%v", actual))
	expectedStr := strings.TrimSpace(fmt.Sprintf("%v", expected))

	return actualStr == expectedStr
}

func (e *evaluationEngine) compareDates(actual, expected interface{}, operator string) bool {
	actualTime, err1 := e.toTime(actual)
	expectedTime, err2 := e.toTime(expected)

	if err1 != nil || err2 != nil {
		return false
	}

	switch operator {
	case "after":
		return actualTime.After(expectedTime)
	case "before":
		return actualTime.Before(expectedTime)
	default:
		return false
	}
}

func (e *evaluationEngine) evaluatePercentageRollout(
	config *RolloutConfig,
	evalCtx *EvaluationContext,
) (bool, error) {
	if config.StartPercentage <= 0 {
		return false, nil
	}

	if config.StartPercentage >= 100 {
		return true, nil
	}

	// Use user ID for consistent bucketing
	hash := e.generateUserHash(evalCtx.UserID.String())
	bucket := hash % 100

	return bucket < config.StartPercentage, nil
}

func (e *evaluationEngine) evaluateCanaryRollout(
	config *RolloutConfig,
	evalCtx *EvaluationContext,
) (bool, error) {
	if len(config.CanaryGroups) == 0 {
		return false, nil
	}

	// Check if user is in any canary group
	userID := evalCtx.UserID.String()
	for _, group := range config.CanaryGroups {
		if e.isUserInCanaryGroup(userID, group) {
			return true, nil
		}
	}

	return false, nil
}

func (e *evaluationEngine) evaluateScheduledRollout(
	config *RolloutConfig,
	evalCtx *EvaluationContext,
) (bool, error) {
	if config.ScheduledRollout == nil {
		return false, nil
	}

	now := evalCtx.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	schedule := config.ScheduledRollout

	// Check if we're within the overall schedule window
	if now.Before(schedule.StartTime) {
		return false, nil
	}

	if !schedule.EndTime.IsZero() && now.After(schedule.EndTime) {
		return true, nil // Schedule completed, full rollout
	}

	// Check business hours constraint
	if schedule.BusinessHours != nil {
		if !e.isWithinBusinessHours(now, schedule.BusinessHours) {
			return false, nil
		}
	}

	// Find the current milestone
	currentPercentage := 0
	for _, milestone := range schedule.Milestones {
		if now.After(milestone.Time) || now.Equal(milestone.Time) {
			currentPercentage = milestone.Percentage
		} else {
			break
		}
	}

	// Evaluate percentage rollout
	if currentPercentage <= 0 {
		return false, nil
	}

	if currentPercentage >= 100 {
		return true, nil
	}

	hash := e.generateUserHash(evalCtx.UserID.String())
	bucket := hash % 100

	return bucket < currentPercentage, nil
}

func (e *evaluationEngine) evaluatePercentageRolloutSimple(
	percentage int,
	evalCtx *EvaluationContext,
) bool {
	if percentage <= 0 {
		return false
	}

	if percentage >= 100 {
		return true
	}

	hash := e.generateUserHash(evalCtx.UserID.String())
	bucket := hash % 100

	return bucket < percentage
}

func (e *evaluationEngine) generateUserHash(userID string) int {
	hasher := fnv.New32a()
	hasher.Write([]byte(userID))
	return int(hasher.Sum32())
}

func (e *evaluationEngine) isUserInCanaryGroup(userID, group string) bool {
	// Simple hash-based canary group assignment
	// In practice, you might use a more sophisticated group assignment
	combined := userID + group
	hasher := md5.New()
	hasher.Write([]byte(combined))
	hash := hasher.Sum(nil)

	// Use first byte to determine group membership (1 in 16 chance)
	return hash[0]%16 == 0
}

func (e *evaluationEngine) isWithinBusinessHours(t time.Time, bh *BusinessHoursConstraint) bool {
	// Convert to business hours timezone
	loc, err := time.LoadLocation(bh.TimeZone)
	if err != nil {
		return false // Invalid timezone, fail safe
	}

	localTime := t.In(loc)
	hour := localTime.Hour()
	weekday := localTime.Weekday().String()

	// Check hour range
	if hour < bh.StartHour || hour >= bh.EndHour {
		return false
	}

	// Check day of week
	if len(bh.Days) > 0 {
		for _, day := range bh.Days {
			if strings.EqualFold(day, weekday) {
				return true
			}
		}
		return false
	}

	return true
}

// Utility methods

func (e *evaluationEngine) toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1"
	case int, int32, int64:
		intVal, _ := e.toInt(v)
		return intVal != 0
	case float32, float64:
		f, _ := e.toFloat64(v)
		return f != 0
	default:
		return false
	}
}

func (e *evaluationEngine) toInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func (e *evaluationEngine) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func (e *evaluationEngine) toTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		// Try common time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("invalid time format: %s", v)
	case int64:
		// Assume Unix timestamp
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", value)
	}
}
