package database

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
)

// buildConnectionString builds a PostgreSQL connection string from config
func buildConnectionString(cfg *config.DatabaseConfig) string {
	// Start with basic connection parameters
	params := make(url.Values)

	// Add SSL mode
	params.Set("sslmode", cfg.SSLMode)

	// Add application name
	params.Set("application_name", "mcp-portal")

	// Add statement timeout if configured
	if cfg.StatementTimeout > 0 {
		params.Set("statement_timeout", fmt.Sprintf("%dms", cfg.StatementTimeout.Milliseconds()))
	}

	// Add connect timeout
	params.Set("connect_timeout", "10")

	// Build the connection string
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?%s",
		url.QueryEscape(cfg.Username),
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.Database,
		params.Encode(),
	)

	return connString
}

// ConnectionOptions provides additional connection configuration
type ConnectionOptions struct {
	// MaxRetries is the maximum number of connection retries
	MaxRetries int

	// RetryInterval is the interval between connection retries
	RetryInterval int

	// EnableTracing enables query tracing
	EnableTracing bool

	// EnableMetrics enables query metrics collection
	EnableMetrics bool

	// QueryTimeout sets a default query timeout
	QueryTimeout int

	// PreparedStatements enables prepared statement caching
	PreparedStatements bool
}

// DefaultConnectionOptions returns default connection options
func DefaultConnectionOptions() *ConnectionOptions {
	return &ConnectionOptions{
		MaxRetries:         3,
		RetryInterval:      1000, // 1 second
		EnableTracing:      false,
		EnableMetrics:      true,
		QueryTimeout:       30000, // 30 seconds
		PreparedStatements: true,
	}
}

// ValidateDatabaseConnection validates database connection parameters
func ValidateDatabaseConnection(cfg *config.DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database configuration is nil")
	}

	if cfg.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", cfg.Port)
	}

	if cfg.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if cfg.Username == "" {
		return fmt.Errorf("database username is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full", "prefer", "allow"}
	sslModeValid := false
	for _, mode := range validSSLModes {
		if cfg.SSLMode == mode {
			sslModeValid = true
			break
		}
	}
	if !sslModeValid {
		return fmt.Errorf("invalid SSL mode: %s", cfg.SSLMode)
	}

	// Validate connection pool settings
	if cfg.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be greater than 0")
	}

	if cfg.MinConnections < 0 {
		return fmt.Errorf("min connections cannot be negative")
	}

	if cfg.MinConnections > cfg.MaxConnections {
		return fmt.Errorf("min connections (%d) cannot exceed max connections (%d)",
			cfg.MinConnections, cfg.MaxConnections)
	}

	return nil
}

// SanitizeConnectionString removes sensitive information from connection string
func SanitizeConnectionString(connString string) string {
	// Parse the connection string
	if strings.HasPrefix(connString, "postgres://") ||
		strings.HasPrefix(connString, "postgresql://") {
		// Parse as URL
		u, err := url.Parse(connString)
		if err != nil {
			return "[invalid connection string]"
		}

		// Remove password
		if u.User != nil {
			u.User = url.User(u.User.Username())
		}

		return u.String()
	}

	// For key=value format, remove password
	parts := strings.Fields(connString)
	sanitized := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, "password=") {
			sanitized = append(sanitized, "password=****")
		} else {
			sanitized = append(sanitized, part)
		}
	}

	return strings.Join(sanitized, " ")
}

// ParseConnectionString parses a connection string into components
func ParseConnectionString(connString string) (*config.DatabaseConfig, error) {
	cfg := &config.DatabaseConfig{}

	if strings.HasPrefix(connString, "postgres://") ||
		strings.HasPrefix(connString, "postgresql://") {
		// Parse as URL
		u, err := url.Parse(connString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse connection URL: %w", err)
		}

		// Extract host and port
		cfg.Host = u.Hostname()
		port := u.Port()
		if port != "" {
			fmt.Sscanf(port, "%d", &cfg.Port)
		} else {
			cfg.Port = 5432 // Default PostgreSQL port
		}

		// Extract database name
		cfg.Database = strings.TrimPrefix(u.Path, "/")

		// Extract username and password
		if u.User != nil {
			cfg.Username = u.User.Username()
			if password, ok := u.User.Password(); ok {
				cfg.Password = password
			}
		}

		// Extract SSL mode from query parameters
		params := u.Query()
		if sslMode := params.Get("sslmode"); sslMode != "" {
			cfg.SSLMode = sslMode
		} else {
			cfg.SSLMode = "prefer" // Default
		}
	} else {
		// Parse key=value format
		parts := strings.Fields(connString)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}

			key, value := kv[0], kv[1]
			switch key {
			case "host":
				cfg.Host = value
			case "port":
				fmt.Sscanf(value, "%d", &cfg.Port)
			case "dbname", "database":
				cfg.Database = value
			case "user", "username":
				cfg.Username = value
			case "password":
				cfg.Password = value
			case "sslmode":
				cfg.SSLMode = value
			}
		}

		// Set defaults
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
		if cfg.SSLMode == "" {
			cfg.SSLMode = "prefer"
		}
	}

	return cfg, nil
}
