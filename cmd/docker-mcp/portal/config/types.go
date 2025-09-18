// Package config provides configuration management for the MCP Portal
package config

import (
	"time"
)

// Config represents the complete portal configuration
type Config struct {
	Environment string         `mapstructure:"environment" json:"environment"`
	Server      ServerConfig   `mapstructure:"server"      json:"server"`
	Database    DatabaseConfig `mapstructure:"database"    json:"database"`
	Redis       RedisConfig    `mapstructure:"redis"       json:"redis"`
	Azure       AzureConfig    `mapstructure:"azure"       json:"azure"`
	Security    SecurityConfig `mapstructure:"security"    json:"security"`
	CLI         CLIConfig      `mapstructure:"cli"         json:"cli"`
}

// ServerConfig defines the HTTP server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"             json:"host"`
	Port            int           `mapstructure:"port"             json:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"     json:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"    json:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" json:"shutdown_timeout"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes" json:"max_header_bytes"`
	TLSEnabled      bool          `mapstructure:"tls_enabled"      json:"tls_enabled"`
	TLSCertFile     string        `mapstructure:"tls_cert_file"    json:"tls_cert_file"`
	TLSKeyFile      string        `mapstructure:"tls_key_file"     json:"tls_key_file"`
}

// DatabaseConfig defines PostgreSQL connection settings
type DatabaseConfig struct {
	Host              string        `mapstructure:"host"                json:"host"`
	Port              int           `mapstructure:"port"                json:"port"`
	Database          string        `mapstructure:"database"            json:"database"`
	Username          string        `mapstructure:"username"            json:"username"`
	Password          string        `mapstructure:"password"            json:"-"` // Never log passwords
	SSLMode           string        `mapstructure:"ssl_mode"            json:"ssl_mode"`
	MaxConnections    int           `mapstructure:"max_connections"     json:"max_connections"`
	MinConnections    int           `mapstructure:"min_connections"     json:"min_connections"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"   json:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"  json:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period" json:"health_check_period"`
	StatementTimeout  time.Duration `mapstructure:"statement_timeout"   json:"statement_timeout"`
}

// RedisConfig defines Redis connection settings
type RedisConfig struct {
	Addrs           []string      `mapstructure:"addrs"             json:"addrs"` // Support cluster mode
	Password        string        `mapstructure:"password"          json:"-"`     // Never log passwords
	DB              int           `mapstructure:"db"                json:"db"`
	MaxRetries      int           `mapstructure:"max_retries"       json:"max_retries"`
	MinRetryBackoff time.Duration `mapstructure:"min_retry_backoff" json:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff" json:"max_retry_backoff"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"      json:"dial_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"      json:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"     json:"write_timeout"`
	PoolSize        int           `mapstructure:"pool_size"         json:"pool_size"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"    json:"min_idle_conns"`
	MaxIdleTime     time.Duration `mapstructure:"max_idle_time"     json:"max_idle_time"`
	PoolTimeout     time.Duration `mapstructure:"pool_timeout"      json:"pool_timeout"`
	SessionTTL      time.Duration `mapstructure:"session_ttl"       json:"session_ttl"`
}

// AzureConfig defines Azure AD authentication settings
type AzureConfig struct {
	TenantID     string   `mapstructure:"tenant_id"     json:"tenant_id"`
	ClientID     string   `mapstructure:"client_id"     json:"client_id"`
	ClientSecret string   `mapstructure:"client_secret" json:"-"` // Never log secrets
	RedirectURL  string   `mapstructure:"redirect_url"  json:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"        json:"scopes"`
	Authority    string   `mapstructure:"authority"     json:"authority"`
}

// SecurityConfig defines security-related settings
type SecurityConfig struct {
	JWTSigningKey     string        `mapstructure:"jwt_signing_key"     json:"-"` // Never log secrets
	JWTIssuer         string        `mapstructure:"jwt_issuer"          json:"jwt_issuer"`
	JWTAudience       []string      `mapstructure:"jwt_audience"        json:"jwt_audience"`
	AccessTokenTTL    time.Duration `mapstructure:"access_token_ttl"    json:"access_token_ttl"`
	RefreshTokenTTL   time.Duration `mapstructure:"refresh_token_ttl"   json:"refresh_token_ttl"`
	CSRFTokenTTL      time.Duration `mapstructure:"csrf_token_ttl"      json:"csrf_token_ttl"`
	RateLimitRequests int           `mapstructure:"rate_limit_requests" json:"rate_limit_requests"`
	RateLimitWindow   time.Duration `mapstructure:"rate_limit_window"   json:"rate_limit_window"`
	AllowedOrigins    []string      `mapstructure:"allowed_origins"     json:"allowed_origins"`
	AllowedMethods    []string      `mapstructure:"allowed_methods"     json:"allowed_methods"`
	AllowedHeaders    []string      `mapstructure:"allowed_headers"     json:"allowed_headers"`
	CORSMaxAge        int           `mapstructure:"cors_max_age"        json:"cors_max_age"`
}

// CLIConfig defines MCP CLI integration settings
type CLIConfig struct {
	BinaryPath       string        `mapstructure:"binary_path"        json:"binary_path"`
	WorkingDir       string        `mapstructure:"working_dir"        json:"working_dir"`
	DockerSocket     string        `mapstructure:"docker_socket"      json:"docker_socket"`
	CommandTimeout   time.Duration `mapstructure:"command_timeout"    json:"command_timeout"`
	MaxConcurrent    int           `mapstructure:"max_concurrent"     json:"max_concurrent"`
	OutputBufferSize int           `mapstructure:"output_buffer_size" json:"output_buffer_size"`
	EnableDebug      bool          `mapstructure:"enable_debug"       json:"enable_debug"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	msg := "validation errors:"
	for _, err := range e {
		msg += "\n  " + err.Error()
	}
	return msg
}
