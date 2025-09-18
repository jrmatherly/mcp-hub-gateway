# Go Backend Environment Variables Reference

## Quick Start

### Minimal Development Setup

```bash
# Core environment configuration
export MCP_PORTAL_ENV=development

# Database connection (required)
export MCP_PORTAL_DATABASE_PASSWORD=your_postgres_password

# Azure AD authentication (required for auth)
export MCP_PORTAL_AZURE_TENANT_ID=your_tenant_id
export MCP_PORTAL_AZURE_CLIENT_ID=your_client_id
export MCP_PORTAL_AZURE_CLIENT_SECRET=your_client_secret

# JWT signing key (required for tokens)
export MCP_PORTAL_JWT_SIGNING_KEY=your_random_jwt_key

# Optional: Redis password if Redis requires auth
export MCP_PORTAL_REDIS_PASSWORD=your_redis_password
```

### Configuration Priority

1. **Environment Variables** (highest priority)
2. **Config File** (`portal.yaml`, `portal.json`)
3. **Default Values** (lowest priority)

## Complete Variable Reference

### Core Configuration

| Variable         | Type     | Default       | Required | Description                                              |
| ---------------- | -------- | ------------- | -------- | -------------------------------------------------------- |
| `MCP_PORTAL_ENV` | `string` | `development` | No       | Environment name: `development`, `staging`, `production` |

### Database Configuration

| Variable                                  | Type       | Default      | Required | Description                                                                   |
| ----------------------------------------- | ---------- | ------------ | -------- | ----------------------------------------------------------------------------- |
| `MCP_PORTAL_DATABASE_HOST`                | `string`   | `localhost`  | No       | PostgreSQL host                                                               |
| `MCP_PORTAL_DATABASE_PORT`                | `int`      | `5432`       | No       | PostgreSQL port                                                               |
| `MCP_PORTAL_DATABASE_DATABASE`            | `string`   | `mcp_portal` | No       | Database name                                                                 |
| `MCP_PORTAL_DATABASE_USERNAME`            | `string`   | `portal`     | No       | Database username                                                             |
| `MCP_PORTAL_DATABASE_PASSWORD`            | `string`   | _none_       | **Yes**  | Database password                                                             |
| `MCP_PORTAL_DATABASE_SSL_MODE`            | `string`   | `prefer`     | No       | SSL mode: `disable`, `require`, `verify-ca`, `verify-full`, `prefer`, `allow` |
| `MCP_PORTAL_DATABASE_MAX_CONNECTIONS`     | `int`      | `20`         | No       | Maximum database connections                                                  |
| `MCP_PORTAL_DATABASE_MIN_CONNECTIONS`     | `int`      | `2`          | No       | Minimum database connections                                                  |
| `MCP_PORTAL_DATABASE_MAX_CONN_LIFETIME`   | `duration` | `1h`         | No       | Connection lifetime                                                           |
| `MCP_PORTAL_DATABASE_MAX_CONN_IDLE_TIME`  | `duration` | `10m`        | No       | Connection idle timeout                                                       |
| `MCP_PORTAL_DATABASE_HEALTH_CHECK_PERIOD` | `duration` | `30s`        | No       | Health check interval                                                         |
| `MCP_PORTAL_DATABASE_STATEMENT_TIMEOUT`   | `duration` | `30s`        | No       | SQL statement timeout                                                         |

### Redis Configuration

| Variable                             | Type       | Default              | Required | Description                       |
| ------------------------------------ | ---------- | -------------------- | -------- | --------------------------------- |
| `MCP_PORTAL_REDIS_ADDRS`             | `string[]` | `["localhost:6379"]` | No       | Redis addresses (comma-separated) |
| `MCP_PORTAL_REDIS_PASSWORD`          | `string`   | _none_               | No\*     | Redis password                    |
| `MCP_PORTAL_REDIS_DB`                | `int`      | `0`                  | No       | Redis database number             |
| `MCP_PORTAL_REDIS_MAX_RETRIES`       | `int`      | `3`                  | No       | Maximum retry attempts            |
| `MCP_PORTAL_REDIS_MIN_RETRY_BACKOFF` | `duration` | `8ms`                | No       | Minimum retry backoff             |
| `MCP_PORTAL_REDIS_MAX_RETRY_BACKOFF` | `duration` | `512ms`              | No       | Maximum retry backoff             |
| `MCP_PORTAL_REDIS_DIAL_TIMEOUT`      | `duration` | `5s`                 | No       | Connection timeout                |
| `MCP_PORTAL_REDIS_READ_TIMEOUT`      | `duration` | `3s`                 | No       | Read timeout                      |
| `MCP_PORTAL_REDIS_WRITE_TIMEOUT`     | `duration` | `3s`                 | No       | Write timeout                     |
| `MCP_PORTAL_REDIS_POOL_SIZE`         | `int`      | `10`                 | No       | Connection pool size              |
| `MCP_PORTAL_REDIS_MIN_IDLE_CONNS`    | `int`      | `2`                  | No       | Minimum idle connections          |
| `MCP_PORTAL_REDIS_MAX_IDLE_TIME`     | `duration` | `5m`                 | No       | Maximum idle time                 |
| `MCP_PORTAL_REDIS_POOL_TIMEOUT`      | `duration` | `4s`                 | No       | Pool timeout                      |
| `MCP_PORTAL_REDIS_SESSION_TTL`       | `duration` | `24h`                | No       | Session TTL                       |

\*Required if Redis instance requires authentication

### Server Configuration

| Variable                             | Type       | Default   | Required | Description               |
| ------------------------------------ | ---------- | --------- | -------- | ------------------------- |
| `MCP_PORTAL_SERVER_HOST`             | `string`   | `0.0.0.0` | No       | Server bind address       |
| `MCP_PORTAL_SERVER_PORT`             | `int`      | `3000`    | No       | Server port               |
| `MCP_PORTAL_SERVER_READ_TIMEOUT`     | `duration` | `30s`     | No       | HTTP read timeout         |
| `MCP_PORTAL_SERVER_WRITE_TIMEOUT`    | `duration` | `30s`     | No       | HTTP write timeout        |
| `MCP_PORTAL_SERVER_SHUTDOWN_TIMEOUT` | `duration` | `10s`     | No       | Graceful shutdown timeout |
| `MCP_PORTAL_SERVER_MAX_HEADER_BYTES` | `int`      | `1048576` | No       | Maximum header size (1MB) |
| `MCP_PORTAL_SERVER_TLS_ENABLED`      | `bool`     | `false`   | No       | Enable TLS/HTTPS          |
| `MCP_PORTAL_SERVER_TLS_CERT_FILE`    | `string`   | _none_    | No\*     | TLS certificate file path |
| `MCP_PORTAL_SERVER_TLS_KEY_FILE`     | `string`   | _none_    | No\*     | TLS private key file path |

\*Required when `MCP_PORTAL_SERVER_TLS_ENABLED=true`

### Azure AD Authentication

| Variable                         | Type       | Default                             | Required | Description                    |
| -------------------------------- | ---------- | ----------------------------------- | -------- | ------------------------------ |
| `MCP_PORTAL_AZURE_TENANT_ID`     | `string`   | _none_                              | **Yes**  | Azure AD tenant ID             |
| `MCP_PORTAL_AZURE_CLIENT_ID`     | `string`   | _none_                              | **Yes**  | Azure AD application ID        |
| `MCP_PORTAL_AZURE_CLIENT_SECRET` | `string`   | _none_                              | **Yes**  | Azure AD client secret         |
| `MCP_PORTAL_AZURE_REDIRECT_URL`  | `string`   | _none_                              | **Yes**  | OAuth redirect URL             |
| `MCP_PORTAL_AZURE_AUTHORITY`     | `string`   | `https://login.microsoftonline.com` | No       | Azure AD authority URL         |
| `MCP_PORTAL_AZURE_SCOPES`        | `string[]` | `["openid", "profile", "email"]`    | No       | OAuth scopes (comma-separated) |

### Security Configuration

| Variable                                  | Type       | Default                                             | Required | Description                             |
| ----------------------------------------- | ---------- | --------------------------------------------------- | -------- | --------------------------------------- |
| `MCP_PORTAL_SECURITY_JWT_SIGNING_KEY`     | `string`   | _none_                                              | **Yes**  | JWT signing key (32+ chars)             |
| `MCP_PORTAL_SECURITY_JWT_ISSUER`          | `string`   | `mcp-portal`                                        | No       | JWT issuer claim                        |
| `MCP_PORTAL_SECURITY_JWT_AUDIENCE`        | `string[]` | `["mcp-portal"]`                                    | No       | JWT audience claims (comma-separated)   |
| `MCP_PORTAL_SECURITY_ACCESS_TOKEN_TTL`    | `duration` | `15m`                                               | No       | Access token lifetime                   |
| `MCP_PORTAL_SECURITY_REFRESH_TOKEN_TTL`   | `duration` | `7d`                                                | No       | Refresh token lifetime                  |
| `MCP_PORTAL_SECURITY_CSRF_TOKEN_TTL`      | `duration` | `24h`                                               | No       | CSRF token lifetime                     |
| `MCP_PORTAL_SECURITY_RATE_LIMIT_REQUESTS` | `int`      | `100`                                               | No       | Rate limit requests per window          |
| `MCP_PORTAL_SECURITY_RATE_LIMIT_WINDOW`   | `duration` | `1m`                                                | No       | Rate limit time window                  |
| `MCP_PORTAL_SECURITY_ALLOWED_ORIGINS`     | `string[]` | `["http://localhost:3000"]`                         | No       | CORS allowed origins (comma-separated)  |
| `MCP_PORTAL_SECURITY_ALLOWED_METHODS`     | `string[]` | `["GET", "POST", "PUT", "DELETE", "OPTIONS"]`       | No       | CORS allowed methods (comma-separated)  |
| `MCP_PORTAL_SECURITY_ALLOWED_HEADERS`     | `string[]` | `["Content-Type", "Authorization", "X-CSRF-Token"]` | No       | CORS allowed headers (comma-separated)  |
| `MCP_PORTAL_SECURITY_CORS_MAX_AGE`        | `int`      | `86400`                                             | No       | CORS preflight cache duration (seconds) |

### CLI Integration

| Variable                            | Type       | Default                | Required | Description                       |
| ----------------------------------- | ---------- | ---------------------- | -------- | --------------------------------- |
| `MCP_PORTAL_CLI_BINARY_PATH`        | `string`   | `docker`               | No       | Docker CLI binary path            |
| `MCP_PORTAL_CLI_WORKING_DIR`        | `string`   | `/var/lib/mcp-portal`  | No       | CLI working directory             |
| `MCP_PORTAL_CLI_DOCKER_SOCKET`      | `string`   | `/var/run/docker.sock` | No       | Docker socket path                |
| `MCP_PORTAL_CLI_COMMAND_TIMEOUT`    | `duration` | `5m`                   | No       | CLI command timeout               |
| `MCP_PORTAL_CLI_MAX_CONCURRENT`     | `int`      | `10`                   | No       | Maximum concurrent CLI operations |
| `MCP_PORTAL_CLI_OUTPUT_BUFFER_SIZE` | `int`      | `1048576`              | No       | CLI output buffer size (1MB)      |
| `MCP_PORTAL_CLI_ENABLE_DEBUG`       | `bool`     | `false`                | No       | Enable CLI debug logging          |

## Environment-Specific Configuration

### Development Environment

```bash
# Basic development setup
export MCP_PORTAL_ENV=development
export MCP_PORTAL_DATABASE_PASSWORD=dev_password
export MCP_PORTAL_REDIS_PASSWORD=dev_redis_password
export MCP_PORTAL_JWT_SIGNING_KEY=dev_jwt_key_min_32_chars_long
export MCP_PORTAL_AZURE_TENANT_ID=your_dev_tenant
export MCP_PORTAL_AZURE_CLIENT_ID=your_dev_client_id
export MCP_PORTAL_AZURE_CLIENT_SECRET=your_dev_secret

# Development-friendly settings
export MCP_PORTAL_SECURITY_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
export MCP_PORTAL_CLI_ENABLE_DEBUG=true
export MCP_PORTAL_SERVER_TLS_ENABLED=false
```

### Staging Environment

```bash
# Staging configuration
export MCP_PORTAL_ENV=staging
export MCP_PORTAL_DATABASE_HOST=staging-db.example.com
export MCP_PORTAL_DATABASE_SSL_MODE=require
export MCP_PORTAL_REDIS_ADDRS=staging-redis.example.com:6379
export MCP_PORTAL_SERVER_HOST=0.0.0.0
export MCP_PORTAL_SERVER_PORT=8080

# Security settings
export MCP_PORTAL_SECURITY_RATE_LIMIT_REQUESTS=500
export MCP_PORTAL_SECURITY_ALLOWED_ORIGINS=https://staging.yourcompany.com
export MCP_PORTAL_SERVER_TLS_ENABLED=true
export MCP_PORTAL_SERVER_TLS_CERT_FILE=/etc/ssl/certs/portal.crt
export MCP_PORTAL_SERVER_TLS_KEY_FILE=/etc/ssl/private/portal.key
```

### Production Environment

```bash
# Production configuration
export MCP_PORTAL_ENV=production
export MCP_PORTAL_DATABASE_HOST=prod-db.example.com
export MCP_PORTAL_DATABASE_PORT=5432
export MCP_PORTAL_DATABASE_SSL_MODE=verify-full
export MCP_PORTAL_DATABASE_MAX_CONNECTIONS=50
export MCP_PORTAL_REDIS_ADDRS=redis-1.example.com:6379,redis-2.example.com:6379

# Production server settings
export MCP_PORTAL_SERVER_HOST=0.0.0.0
export MCP_PORTAL_SERVER_PORT=443
export MCP_PORTAL_SERVER_TLS_ENABLED=true
export MCP_PORTAL_SERVER_READ_TIMEOUT=60s
export MCP_PORTAL_SERVER_WRITE_TIMEOUT=60s

# Production security
export MCP_PORTAL_SECURITY_RATE_LIMIT_REQUESTS=1000
export MCP_PORTAL_SECURITY_RATE_LIMIT_WINDOW=5m
export MCP_PORTAL_SECURITY_ALLOWED_ORIGINS=https://portal.yourcompany.com
export MCP_PORTAL_SECURITY_ACCESS_TOKEN_TTL=10m
export MCP_PORTAL_SECURITY_REFRESH_TOKEN_TTL=24h
```

## Configuration File Support

### YAML Configuration Example

Create `portal.yaml` in one of these locations:

- Current directory
- `./config/`
- `/etc/mcp-portal/`
- `$HOME/.mcp-portal/`

```yaml
environment: production

server:
  host: 0.0.0.0
  port: 3000
  tls_enabled: true
  tls_cert_file: /etc/ssl/certs/portal.crt
  tls_key_file: /etc/ssl/private/portal.key

database:
  host: localhost
  port: 5432
  database: mcp_portal
  username: portal
  ssl_mode: require
  max_connections: 20

redis:
  addrs:
    - localhost:6379
  db: 0
  session_ttl: 24h

azure:
  tenant_id: ${AZURE_TENANT_ID}
  client_id: ${AZURE_CLIENT_ID}
  authority: https://login.microsoftonline.com

security:
  jwt_issuer: mcp-portal
  jwt_audience:
    - mcp-portal
  access_token_ttl: 15m
  refresh_token_ttl: 7d
  allowed_origins:
    - https://portal.yourcompany.com

cli:
  binary_path: docker
  working_dir: /var/lib/mcp-portal
  command_timeout: 5m
```

**Note**: Sensitive values (passwords, secrets) should use environment variables even in config files.

## Security Best Practices

### Required Secrets

These **MUST** be set via secure environment variables:

```bash
# Database credentials
MCP_PORTAL_DATABASE_PASSWORD

# Redis credentials (if auth enabled)
MCP_PORTAL_REDIS_PASSWORD

# Azure AD credentials
MCP_PORTAL_AZURE_CLIENT_SECRET

# JWT signing key (generate secure random key)
MCP_PORTAL_JWT_SIGNING_KEY
```

### Secret Generation

```bash
# Generate secure JWT signing key
openssl rand -base64 32

# Generate secure database password
openssl rand -base64 24
```

### Secret Storage

**Production environments should use:**

- Kubernetes Secrets
- Docker Secrets
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault

**Never store secrets in:**

- Configuration files
- Container images
- Version control
- Log files

## Data Types & Formats

### Duration Format

Use Go duration strings:

```bash
# Valid duration formats
export MCP_PORTAL_DATABASE_STATEMENT_TIMEOUT=30s
export MCP_PORTAL_SECURITY_ACCESS_TOKEN_TTL=15m
export MCP_PORTAL_SECURITY_REFRESH_TOKEN_TTL=7d
export MCP_PORTAL_DATABASE_MAX_CONN_LIFETIME=1h
```

Valid units: `ns`, `us`, `ms`, `s`, `m`, `h`

### Array Format

Use comma-separated values:

```bash
# String arrays
export MCP_PORTAL_REDIS_ADDRS=redis-1:6379,redis-2:6379
export MCP_PORTAL_SECURITY_ALLOWED_ORIGINS=https://app.com,https://admin.com
export MCP_PORTAL_AZURE_SCOPES=openid,profile,email
```

### Boolean Format

Use standard boolean values:

```bash
# Boolean values
export MCP_PORTAL_SERVER_TLS_ENABLED=true
export MCP_PORTAL_CLI_ENABLE_DEBUG=false
```

Valid values: `true`, `false`, `1`, `0`

## Validation Rules

### Required Field Validation

The application validates configuration on startup:

**Database Requirements:**

- Host cannot be empty
- Port must be 1-65535
- Database name cannot be empty
- Username cannot be empty

**Azure Requirements (if any Azure config provided):**

- Tenant ID required
- Client ID required
- Redirect URL required and valid

**Security Requirements:**

- JWT issuer cannot be empty
- At least one JWT audience required
- Access token TTL < Refresh token TTL
- At least one allowed origin required

### Common Validation Errors

```bash
# Invalid port
MCP_PORTAL_DATABASE_PORT=99999
# Error: invalid database port: 99999

# Invalid SSL mode
MCP_PORTAL_DATABASE_SSL_MODE=invalid
# Error: invalid SSL mode: invalid (must be one of: disable, require, verify-ca, verify-full, prefer, allow)

# Invalid duration
MCP_PORTAL_SECURITY_ACCESS_TOKEN_TTL=invalid
# Error: time: invalid duration "invalid"

# Invalid Redis address
MCP_PORTAL_REDIS_ADDRS=localhost
# Error: invalid address format: localhost (expected host:port)
```

## Troubleshooting

### Configuration Loading Issues

**Check configuration loading order:**

1. Verify environment variables are set: `env | grep MCP_PORTAL`
2. Check config file location and syntax
3. Review validation errors in application logs

**Common issues:**

```bash
# Case sensitivity - variable names must be uppercase
export mcp_portal_env=development  # ❌ Wrong
export MCP_PORTAL_ENV=development  # ✅ Correct

# Incorrect prefix
export PORTAL_ENV=development      # ❌ Wrong
export MCP_PORTAL_ENV=development  # ✅ Correct
```

### Database Connection Issues

```bash
# Test connection manually
psql -h $MCP_PORTAL_DATABASE_HOST \
     -p $MCP_PORTAL_DATABASE_PORT \
     -U $MCP_PORTAL_DATABASE_USERNAME \
     -d $MCP_PORTAL_DATABASE_DATABASE

# Check SSL mode compatibility
export MCP_PORTAL_DATABASE_SSL_MODE=disable  # For local development
export MCP_PORTAL_DATABASE_SSL_MODE=require  # For production
```

### Redis Connection Issues

```bash
# Test Redis connection
redis-cli -h $(echo $MCP_PORTAL_REDIS_ADDRS | cut -d',' -f1 | cut -d':' -f1) \
          -p $(echo $MCP_PORTAL_REDIS_ADDRS | cut -d',' -f1 | cut -d':' -f2) \
          ping

# Test with password
redis-cli -h localhost -p 6379 -a $MCP_PORTAL_REDIS_PASSWORD ping
```

### JWT/Authentication Issues

```bash
# Verify JWT signing key length (minimum 32 characters)
echo -n "$MCP_PORTAL_JWT_SIGNING_KEY" | wc -c

# Test Azure AD configuration
curl -X GET "https://login.microsoftonline.com/$MCP_PORTAL_AZURE_TENANT_ID/v2.0/.well-known/openid_configuration"
```

### CLI Integration Issues

```bash
# Verify Docker CLI access
export MCP_PORTAL_CLI_BINARY_PATH=/usr/local/bin/docker
$MCP_PORTAL_CLI_BINARY_PATH version

# Check Docker socket permissions
ls -la $MCP_PORTAL_CLI_DOCKER_SOCKET
```

## Docker Configuration

### Docker Compose Example

```yaml
version: "3.8"
services:
  mcp-portal:
    image: mcp-portal:latest
    environment:
      # Core configuration
      MCP_PORTAL_ENV: production

      # Database
      MCP_PORTAL_DATABASE_HOST: postgres
      MCP_PORTAL_DATABASE_PASSWORD_FILE: /run/secrets/db_password

      # Redis
      MCP_PORTAL_REDIS_ADDRS: redis:6379
      MCP_PORTAL_REDIS_PASSWORD_FILE: /run/secrets/redis_password

      # Azure AD
      MCP_PORTAL_AZURE_TENANT_ID: ${AZURE_TENANT_ID}
      MCP_PORTAL_AZURE_CLIENT_ID: ${AZURE_CLIENT_ID}
      MCP_PORTAL_AZURE_CLIENT_SECRET_FILE: /run/secrets/azure_secret

      # JWT
      MCP_PORTAL_JWT_SIGNING_KEY_FILE: /run/secrets/jwt_key

    secrets:
      - db_password
      - redis_password
      - azure_secret
      - jwt_key
    ports:
      - "3000:3000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro

secrets:
  db_password:
    external: true
  redis_password:
    external: true
  azure_secret:
    external: true
  jwt_key:
    external: true
```

### Kubernetes ConfigMap & Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-portal-config
data:
  MCP_PORTAL_ENV: "production"
  MCP_PORTAL_DATABASE_HOST: "postgres-service"
  MCP_PORTAL_REDIS_ADDRS: "redis-service:6379"
  MCP_PORTAL_SERVER_HOST: "0.0.0.0"
  MCP_PORTAL_SERVER_PORT: "3000"

---
apiVersion: v1
kind: Secret
metadata:
  name: mcp-portal-secrets
type: Opaque
stringData:
  MCP_PORTAL_DATABASE_PASSWORD: "secure_db_password"
  MCP_PORTAL_REDIS_PASSWORD: "secure_redis_password"
  MCP_PORTAL_AZURE_CLIENT_SECRET: "azure_client_secret"
  MCP_PORTAL_JWT_SIGNING_KEY: "secure_jwt_signing_key_32_chars"
```

## Monitoring & Observability

### Health Check Configuration

```bash
# Enable comprehensive health checks
export MCP_PORTAL_DATABASE_HEALTH_CHECK_PERIOD=30s
export MCP_PORTAL_REDIS_DIAL_TIMEOUT=5s
export MCP_PORTAL_CLI_COMMAND_TIMEOUT=30s
```

### Logging Configuration

```bash
# Enable debug logging for troubleshooting
export MCP_PORTAL_CLI_ENABLE_DEBUG=true
```

### Metrics Collection

Set appropriate timeouts for metrics collection:

```bash
export MCP_PORTAL_DATABASE_STATEMENT_TIMEOUT=10s
export MCP_PORTAL_REDIS_READ_TIMEOUT=3s
export MCP_PORTAL_REDIS_WRITE_TIMEOUT=3s
```

---

## Summary

This comprehensive guide covers all environment variables for the MCP Portal Go backend. Key points:

1. **Required Variables**: Database password, Azure AD credentials, JWT signing key
2. **Configuration Priority**: Environment variables override config files override defaults
3. **Security**: Store sensitive values in secure environment variables only
4. **Validation**: Application validates configuration on startup with detailed error messages
5. **Flexibility**: Support for both environment variables and YAML configuration files

For additional support, refer to the application logs during startup for specific configuration validation errors.
