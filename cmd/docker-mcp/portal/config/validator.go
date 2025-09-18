package config

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// Validate validates the configuration
func Validate(c *Config) error {
	var errs ValidationErrors

	// Validate server config
	if err := validateServerConfig(&c.Server); err != nil {
		errs = append(errs, err...)
	}

	// Validate database config
	if err := validateDatabaseConfig(&c.Database); err != nil {
		errs = append(errs, err...)
	}

	// Validate Redis config
	if err := validateRedisConfig(&c.Redis); err != nil {
		errs = append(errs, err...)
	}

	// Validate Azure config
	if err := validateAzureConfig(&c.Azure); err != nil {
		errs = append(errs, err...)
	}

	// Validate security config
	if err := validateSecurityConfig(&c.Security); err != nil {
		errs = append(errs, err...)
	}

	// Validate CLI config
	if err := validateCLIConfig(&c.CLI); err != nil {
		errs = append(errs, err...)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateServerConfig(c *ServerConfig) ValidationErrors {
	var errs ValidationErrors

	// Validate host
	if c.Host == "" {
		errs = append(errs, ValidationError{
			Field:   "server.host",
			Message: "host cannot be empty",
		})
	} else if net.ParseIP(c.Host) == nil && c.Host != "localhost" {
		// Try to validate as hostname
		if _, err := net.LookupHost(c.Host); err != nil && c.Host != "0.0.0.0" {
			errs = append(errs, ValidationError{
				Field:   "server.host",
				Message: fmt.Sprintf("invalid host: %s", c.Host),
			})
		}
	}

	// Validate port
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "server.port",
			Message: fmt.Sprintf("invalid port: %d (must be 1-65535)", c.Port),
		})
	}

	// Validate timeouts
	if c.ReadTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:   "server.read_timeout",
			Message: "read timeout must be positive",
		})
	}

	if c.WriteTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:   "server.write_timeout",
			Message: "write timeout must be positive",
		})
	}

	if c.ShutdownTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:   "server.shutdown_timeout",
			Message: "shutdown timeout must be positive",
		})
	}

	// Validate TLS config
	if c.TLSEnabled {
		if c.TLSCertFile == "" {
			errs = append(errs, ValidationError{
				Field:   "server.tls_cert_file",
				Message: "TLS cert file required when TLS is enabled",
			})
		}
		if c.TLSKeyFile == "" {
			errs = append(errs, ValidationError{
				Field:   "server.tls_key_file",
				Message: "TLS key file required when TLS is enabled",
			})
		}
	}

	return errs
}

func validateDatabaseConfig(c *DatabaseConfig) ValidationErrors {
	var errs ValidationErrors

	// Validate host
	if c.Host == "" {
		errs = append(errs, ValidationError{
			Field:   "database.host",
			Message: "database host cannot be empty",
		})
	}

	// Validate port
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "database.port",
			Message: fmt.Sprintf("invalid database port: %d", c.Port),
		})
	}

	// Validate database name
	if c.Database == "" {
		errs = append(errs, ValidationError{
			Field:   "database.database",
			Message: "database name cannot be empty",
		})
	}

	// Validate username
	if c.Username == "" {
		errs = append(errs, ValidationError{
			Field:   "database.username",
			Message: "database username cannot be empty",
		})
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full", "prefer", "allow"}
	if !contains(validSSLModes, c.SSLMode) {
		errs = append(errs, ValidationError{
			Field: "database.ssl_mode",
			Message: fmt.Sprintf(
				"invalid SSL mode: %s (must be one of: %s)",
				c.SSLMode,
				strings.Join(validSSLModes, ", "),
			),
		})
	}

	// Validate connection pool settings
	if c.MaxConnections <= 0 {
		errs = append(errs, ValidationError{
			Field:   "database.max_connections",
			Message: "max connections must be positive",
		})
	}

	if c.MinConnections < 0 {
		errs = append(errs, ValidationError{
			Field:   "database.min_connections",
			Message: "min connections cannot be negative",
		})
	}

	if c.MinConnections > c.MaxConnections {
		errs = append(errs, ValidationError{
			Field: "database.min_connections",
			Message: fmt.Sprintf(
				"min connections (%d) cannot exceed max connections (%d)",
				c.MinConnections,
				c.MaxConnections,
			),
		})
	}

	return errs
}

func validateRedisConfig(c *RedisConfig) ValidationErrors {
	var errs ValidationErrors

	// Validate addresses
	if len(c.Addrs) == 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.addrs",
			Message: "at least one Redis address required",
		})
	}

	for i, addr := range c.Addrs {
		// Validate address format (host:port)
		if _, _, err := net.SplitHostPort(addr); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("redis.addrs[%d]", i),
				Message: fmt.Sprintf("invalid address format: %s (expected host:port)", addr),
			})
		}
	}

	// Validate DB number
	if c.DB < 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.db",
			Message: "database number cannot be negative",
		})
	}

	// Validate pool settings
	if c.PoolSize <= 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.pool_size",
			Message: "pool size must be positive",
		})
	}

	if c.MinIdleConns < 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.min_idle_conns",
			Message: "min idle connections cannot be negative",
		})
	}

	if c.MinIdleConns > c.PoolSize {
		errs = append(errs, ValidationError{
			Field: "redis.min_idle_conns",
			Message: fmt.Sprintf(
				"min idle connections (%d) cannot exceed pool size (%d)",
				c.MinIdleConns,
				c.PoolSize,
			),
		})
	}

	// Validate timeouts
	if c.DialTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.dial_timeout",
			Message: "dial timeout must be positive",
		})
	}

	if c.SessionTTL <= 0 {
		errs = append(errs, ValidationError{
			Field:   "redis.session_ttl",
			Message: "session TTL must be positive",
		})
	}

	return errs
}

func validateAzureConfig(c *AzureConfig) ValidationErrors {
	var errs ValidationErrors

	// These fields are required only if Azure auth is being used
	// We check if any Azure config is provided
	if c.TenantID != "" || c.ClientID != "" || c.RedirectURL != "" {
		// If any Azure config is provided, validate all required fields
		if c.TenantID == "" {
			errs = append(errs, ValidationError{
				Field:   "azure.tenant_id",
				Message: "tenant ID required for Azure authentication",
			})
		}

		if c.ClientID == "" {
			errs = append(errs, ValidationError{
				Field:   "azure.client_id",
				Message: "client ID required for Azure authentication",
			})
		}

		if c.RedirectURL == "" {
			errs = append(errs, ValidationError{
				Field:   "azure.redirect_url",
				Message: "redirect URL required for Azure authentication",
			})
		} else {
			// Validate redirect URL format
			if _, err := url.Parse(c.RedirectURL); err != nil {
				errs = append(errs, ValidationError{
					Field:   "azure.redirect_url",
					Message: fmt.Sprintf("invalid redirect URL: %v", err),
				})
			}
		}

		// Validate authority URL if provided
		if c.Authority != "" {
			if _, err := url.Parse(c.Authority); err != nil {
				errs = append(errs, ValidationError{
					Field:   "azure.authority",
					Message: fmt.Sprintf("invalid authority URL: %v", err),
				})
			}
		}
	}

	return errs
}

func validateSecurityConfig(c *SecurityConfig) ValidationErrors {
	var errs ValidationErrors

	// Validate JWT config
	if c.JWTIssuer == "" {
		errs = append(errs, ValidationError{
			Field:   "security.jwt_issuer",
			Message: "JWT issuer cannot be empty",
		})
	}

	if len(c.JWTAudience) == 0 {
		errs = append(errs, ValidationError{
			Field:   "security.jwt_audience",
			Message: "at least one JWT audience required",
		})
	}

	// Validate token TTLs
	if c.AccessTokenTTL <= 0 {
		errs = append(errs, ValidationError{
			Field:   "security.access_token_ttl",
			Message: "access token TTL must be positive",
		})
	}

	if c.RefreshTokenTTL <= 0 {
		errs = append(errs, ValidationError{
			Field:   "security.refresh_token_ttl",
			Message: "refresh token TTL must be positive",
		})
	}

	if c.AccessTokenTTL >= c.RefreshTokenTTL {
		errs = append(errs, ValidationError{
			Field:   "security.access_token_ttl",
			Message: "access token TTL must be less than refresh token TTL",
		})
	}

	// Validate rate limiting
	if c.RateLimitRequests <= 0 {
		errs = append(errs, ValidationError{
			Field:   "security.rate_limit_requests",
			Message: "rate limit requests must be positive",
		})
	}

	if c.RateLimitWindow <= 0 {
		errs = append(errs, ValidationError{
			Field:   "security.rate_limit_window",
			Message: "rate limit window must be positive",
		})
	}

	// Validate CORS config
	if len(c.AllowedOrigins) == 0 {
		errs = append(errs, ValidationError{
			Field:   "security.allowed_origins",
			Message: "at least one allowed origin required",
		})
	}

	if len(c.AllowedMethods) == 0 {
		errs = append(errs, ValidationError{
			Field:   "security.allowed_methods",
			Message: "at least one allowed method required",
		})
	}

	return errs
}

func validateCLIConfig(c *CLIConfig) ValidationErrors {
	var errs ValidationErrors

	// Validate binary path
	if c.BinaryPath == "" {
		errs = append(errs, ValidationError{
			Field:   "cli.binary_path",
			Message: "CLI binary path cannot be empty",
		})
	}

	// Validate working directory
	if c.WorkingDir != "" && !filepath.IsAbs(c.WorkingDir) {
		errs = append(errs, ValidationError{
			Field:   "cli.working_dir",
			Message: "working directory must be an absolute path",
		})
	}

	// Validate Docker socket
	if c.DockerSocket == "" {
		errs = append(errs, ValidationError{
			Field:   "cli.docker_socket",
			Message: "Docker socket path cannot be empty",
		})
	}

	// Validate command timeout
	if c.CommandTimeout <= 0 || c.CommandTimeout > 30*time.Minute {
		errs = append(errs, ValidationError{
			Field: "cli.command_timeout",
			Message: fmt.Sprintf(
				"command timeout must be between 1 second and 30 minutes, got: %v",
				c.CommandTimeout,
			),
		})
	}

	// Validate max concurrent
	if c.MaxConcurrent <= 0 {
		errs = append(errs, ValidationError{
			Field:   "cli.max_concurrent",
			Message: "max concurrent must be positive",
		})
	}

	// Validate output buffer size
	if c.OutputBufferSize <= 0 {
		errs = append(errs, ValidationError{
			Field:   "cli.output_buffer_size",
			Message: "output buffer size must be positive",
		})
	}

	return errs
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
