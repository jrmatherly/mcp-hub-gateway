package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

// Load loads the configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	var err error
	once.Do(func() {
		instance, err = loadConfig(configPath)
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// Get returns the current configuration instance
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return instance
}

// Reload reloads the configuration from file and environment
func Reload(configPath string) (*Config, error) {
	mu.Lock()
	defer mu.Unlock()

	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	instance = config
	return instance, nil
}

// loadConfig loads configuration from file and environment
func loadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set up environment variable binding
	v.SetEnvPrefix("MCP_PORTAL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Load config file if specified
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			// Config file is optional, only log if it exists but can't be read
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	} else {
		// Try to find config in standard locations
		v.SetConfigName("portal")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/mcp-portal")
		v.AddConfigPath("$HOME/.mcp-portal")

		// ReadInConfig will fail if no config file is found, which is ok
		_ = v.ReadInConfig()
	}

	// Unmarshal into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply environment overrides for sensitive fields
	applyEnvOverrides(&config)

	// Validate configuration
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Environment defaults
	v.SetDefault("environment", "development")

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 3000)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.max_header_bytes", 1048576) // 1MB
	v.SetDefault("server.tls_enabled", false)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.database", "mcp_portal")
	v.SetDefault("database.username", "portal")
	v.SetDefault("database.ssl_mode", "prefer")
	v.SetDefault("database.max_connections", 20)
	v.SetDefault("database.min_connections", 2)
	v.SetDefault("database.max_conn_lifetime", "1h")
	v.SetDefault("database.max_conn_idle_time", "10m")
	v.SetDefault("database.health_check_period", "30s")
	v.SetDefault("database.statement_timeout", "30s")

	// Redis defaults
	v.SetDefault("redis.addrs", []string{"localhost:6379"})
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.min_retry_backoff", "8ms")
	v.SetDefault("redis.max_retry_backoff", "512ms")
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 2)
	v.SetDefault("redis.max_idle_time", "5m")
	v.SetDefault("redis.pool_timeout", "4s")
	v.SetDefault("redis.session_ttl", "24h")

	// Azure defaults
	v.SetDefault("azure.authority", "https://login.microsoftonline.com")
	v.SetDefault("azure.scopes", []string{"openid", "profile", "email"})

	// Security defaults
	v.SetDefault("security.jwt_issuer", "mcp-portal")
	v.SetDefault("security.jwt_audience", []string{"mcp-portal"})
	v.SetDefault("security.access_token_ttl", "15m")
	v.SetDefault("security.refresh_token_ttl", "7d")
	v.SetDefault("security.csrf_token_ttl", "24h")
	v.SetDefault("security.rate_limit_requests", 100)
	v.SetDefault("security.rate_limit_window", "1m")
	v.SetDefault("security.allowed_origins", []string{"http://localhost:3000"})
	v.SetDefault("security.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault(
		"security.allowed_headers",
		[]string{"Content-Type", "Authorization", "X-CSRF-Token"},
	)
	v.SetDefault("security.cors_max_age", 86400)

	// CLI defaults
	v.SetDefault("cli.binary_path", "docker")
	v.SetDefault("cli.working_dir", "/var/lib/mcp-portal")
	v.SetDefault("cli.docker_socket", "/var/run/docker.sock")
	v.SetDefault("cli.command_timeout", "5m")
	v.SetDefault("cli.max_concurrent", 10)
	v.SetDefault("cli.output_buffer_size", 1048576) // 1MB
	v.SetDefault("cli.enable_debug", false)
}

// applyEnvOverrides applies environment variable overrides for sensitive fields
func applyEnvOverrides(config *Config) {
	// Database password
	if password := os.Getenv("MCP_PORTAL_DATABASE_PASSWORD"); password != "" {
		config.Database.Password = password
	}

	// Redis password
	if password := os.Getenv("MCP_PORTAL_REDIS_PASSWORD"); password != "" {
		config.Redis.Password = password
	}

	// Azure client secret
	if secret := os.Getenv("MCP_PORTAL_AZURE_CLIENT_SECRET"); secret != "" {
		config.Azure.ClientSecret = secret
	}

	// JWT signing key
	if key := os.Getenv("MCP_PORTAL_JWT_SIGNING_KEY"); key != "" {
		config.Security.JWTSigningKey = key
	}
}

// GetDatabaseURL builds a PostgreSQL connection URL from the config
func (c *Config) GetDatabaseURL() string {
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)

	if c.Database.StatementTimeout > 0 {
		url += fmt.Sprintf("&statement_timeout=%d", int(c.Database.StatementTimeout.Milliseconds()))
	}

	return url
}

// GetRedisAddr returns the primary Redis address
func (c *Config) GetRedisAddr() string {
	if len(c.Redis.Addrs) > 0 {
		return c.Redis.Addrs[0]
	}
	return "localhost:6379"
}

// GetServerAddr returns the server listen address
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsProduction checks if the configuration is for production
func (c *Config) IsProduction() bool {
	env := os.Getenv("MCP_PORTAL_ENV")
	return env == "production" || env == "prod"
}

// IsDevelopment checks if the configuration is for development
func (c *Config) IsDevelopment() bool {
	env := os.Getenv("MCP_PORTAL_ENV")
	return env == "development" || env == "dev" || env == ""
}

// GetCLIBinaryPath returns the full path to the CLI binary
func (c *Config) GetCLIBinaryPath() string {
	if filepath.IsAbs(c.CLI.BinaryPath) {
		return c.CLI.BinaryPath
	}

	// Try to find in PATH
	if path, err := exec.LookPath(c.CLI.BinaryPath); err == nil {
		return path
	}

	// Default to docker
	return "docker"
}
