package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvHelper provides utility functions for environment variable handling
type EnvHelper struct {
	prefix string
}

// NewEnvHelper creates a new environment variable helper
func NewEnvHelper(prefix string) *EnvHelper {
	return &EnvHelper{
		prefix: prefix,
	}
}

// GetString retrieves a string environment variable
func (e *EnvHelper) GetString(key string, defaultValue string) string {
	envKey := e.formatKey(key)
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// GetInt retrieves an integer environment variable
func (e *EnvHelper) GetInt(key string, defaultValue int) int {
	envKey := e.formatKey(key)
	if value := os.Getenv(envKey); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetBool retrieves a boolean environment variable
func (e *EnvHelper) GetBool(key string, defaultValue bool) bool {
	envKey := e.formatKey(key)
	if value := os.Getenv(envKey); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDuration retrieves a duration environment variable
func (e *EnvHelper) GetDuration(key string, defaultValue time.Duration) time.Duration {
	envKey := e.formatKey(key)
	if value := os.Getenv(envKey); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetStringSlice retrieves a string slice from environment variable (comma-separated)
func (e *EnvHelper) GetStringSlice(key string, defaultValue []string) []string {
	envKey := e.formatKey(key)
	if value := os.Getenv(envKey); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// formatKey formats the environment variable key with prefix
func (e *EnvHelper) formatKey(key string) string {
	// Convert to uppercase and replace dots with underscores
	key = strings.ToUpper(key)
	key = strings.ReplaceAll(key, ".", "_")

	if e.prefix != "" {
		return e.prefix + "_" + key
	}
	return key
}

// LoadEnvFile loads environment variables from a file
func LoadEnvFile(filepath string) error {
	// This is a simple implementation - for production, consider using godotenv
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .env file is optional
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return nil
}

// MustGetString retrieves a string environment variable or panics
func (e *EnvHelper) MustGetString(key string) string {
	envKey := e.formatKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		panic("required environment variable not set: " + envKey)
	}
	return value
}

// IsSet checks if an environment variable is set
func (e *EnvHelper) IsSet(key string) bool {
	envKey := e.formatKey(key)
	_, exists := os.LookupEnv(envKey)
	return exists
}

// GetEnvOrDefault is a standalone function for quick environment variable access
func GetEnvOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsDevelopment checks if we're running in development mode
func IsDevelopment() bool {
	env := strings.ToLower(os.Getenv("MCP_PORTAL_ENV"))
	return env == "" || env == "development" || env == "dev"
}

// IsProduction checks if we're running in production mode
func IsProduction() bool {
	env := strings.ToLower(os.Getenv("MCP_PORTAL_ENV"))
	return env == "production" || env == "prod"
}

// IsStaging checks if we're running in staging mode
func IsStaging() bool {
	env := strings.ToLower(os.Getenv("MCP_PORTAL_ENV"))
	return env == "staging" || env == "stage"
}

// GetEnvironment returns the current environment name
func GetEnvironment() string {
	if env := os.Getenv("MCP_PORTAL_ENV"); env != "" {
		return strings.ToLower(env)
	}
	return "development"
}
