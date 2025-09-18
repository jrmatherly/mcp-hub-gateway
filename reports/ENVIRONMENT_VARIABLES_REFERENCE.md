# MCP Portal Environment Variables Reference

## 📚 Next.js Environment Variable Rules

**Important**: In Next.js, the `.env.local` file can contain BOTH server-side and client-side variables:

- **Variables WITHOUT `NEXT_PUBLIC_` prefix**: Server-side only, NOT exposed to browser
- **Variables WITH `NEXT_PUBLIC_` prefix**: Exposed to the browser bundle

## Quick Start

### Frontend (.env.local) - Complete Configuration

```bash
# Copy cmd/docker-mcp/portal/frontend/.env.local.example to .env.local
# Update with your specific values

# ========== SERVER-SIDE VARIABLES (Safe for secrets) ==========
# Azure AD (Required for authentication)
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret  # Keep this SECRET

# Security (Required)
JWT_SECRET=your-jwt-secret-minimum-32-characters

# Database (Optional)
DATABASE_URL=postgresql://user:pass@host:5432/db
REDIS_URL=redis://host:6379

# ========== CLIENT-SIDE VARIABLES (Exposed to browser) ==========
# Backend URLs
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080

# Azure AD Public Config
NEXT_PUBLIC_AZURE_REDIRECT_URI=http://localhost:3000/auth/callback
NEXT_PUBLIC_AZURE_POST_LOGOUT_URI=http://localhost:3000
NEXT_PUBLIC_AZURE_SCOPES=openid profile email User.Read

# Optional UI Settings
NEXT_PUBLIC_DEFAULT_THEME=system
NEXT_PUBLIC_DEBUG=false
```

### Backend Go Service Environment

````bash
# For the Go backend service (docker mcp portal serve)

## Complete Variable Reference

### Frontend Variables (NEXT_PUBLIC_)

#### **API Configuration**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | Yes | `http://localhost:8080` | Backend API URL |
| `NEXT_PUBLIC_WS_URL` | Yes | `ws://localhost:8080` | WebSocket URL for real-time updates |

#### **Azure AD OAuth (Public)**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_AZURE_REDIRECT_URI` | Yes | `http://localhost:3000/auth/callback` | OAuth callback URL |
| `NEXT_PUBLIC_AZURE_POST_LOGOUT_URI` | No | `http://localhost:3000` | Post-logout redirect |
| `NEXT_PUBLIC_AZURE_AUTHORITY` | No | Constructed from tenant | Azure AD authority URL |
| `NEXT_PUBLIC_AZURE_SCOPES` | No | `openid profile User.Read` | OAuth scopes |

#### **Development & Debugging**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_DEBUG` | No | `false` | Enable debug logging |
| `NEXT_PUBLIC_LOG_LEVEL` | No | `INFO` | Frontend log level (DEBUG, INFO, WARN, ERROR, NONE) |
| `NEXT_PUBLIC_API_TIMEOUT` | No | `30000` | API request timeout (ms) |
| `NEXT_PUBLIC_WS_RECONNECT_INTERVAL` | No | `5000` | WebSocket reconnect interval (ms) |

#### **Feature Flags**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_ENABLE_WEBSOCKET` | No | `true` | Enable WebSocket connections |
| `NEXT_PUBLIC_ENABLE_SSE` | No | `true` | Enable Server-Sent Events |
| `NEXT_PUBLIC_ENABLE_ADMIN` | No | `true` | Enable admin panel features |
| `NEXT_PUBLIC_ENABLE_BULK_OPS` | No | `true` | Enable bulk operations |

#### **Security Configuration**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_TOKEN_STORAGE` | No | `localStorage` | Token storage (localStorage, sessionStorage, memory) |
| `NEXT_PUBLIC_SESSION_TIMEOUT` | No | `60` | Session timeout (minutes) |
| `NEXT_PUBLIC_ENABLE_CSRF` | No | `true` | Enable CSRF protection |

#### **UI Configuration**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_DEFAULT_THEME` | No | `system` | Default theme (light, dark, system) |
| `NEXT_PUBLIC_DEFAULT_PAGE_SIZE` | No | `20` | Items per page in lists |
| `NEXT_PUBLIC_STATUS_REFRESH_INTERVAL` | No | `10` | Status refresh interval (seconds) |

#### **Error Reporting**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_SENTRY_DSN` | No | - | Sentry DSN for error tracking |
| `NEXT_PUBLIC_ENABLE_ERROR_REPORTING` | No | `false` | Enable error reporting |

#### **Build Configuration**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SKIP_ENV_VALIDATION` | No | `false` | Skip T3 env validation (use with caution) |
| `ANALYZE` | No | `false` | Enable bundle analyzer |

### Server-Side Variables

#### **Azure AD (Confidential)**
| Variable | Required | Description |
|----------|----------|-------------|
| `AZURE_TENANT_ID` | Yes | Azure AD tenant ID |
| `AZURE_CLIENT_ID` | Yes | Azure AD application ID |
| `AZURE_CLIENT_SECRET` | Yes | Azure AD client secret |

#### **Security & JWT**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | - | JWT signing key (min 32 chars) |
| `NODE_ENV` | No | `development` | Environment mode |

#### **Session Configuration**
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SESSION_COOKIE_NAME` | No | `mcp-portal-session` | Session cookie name |
| `SESSION_COOKIE_SECURE` | No | `false` | Cookie secure flag |
| `SESSION_COOKIE_HTTPONLY` | No | `true` | Cookie HTTP-only flag |
| `SESSION_COOKIE_SAMESITE` | No | `lax` | Cookie SameSite policy |

#### **Database & Cache**
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | No | PostgreSQL connection URL |
| `REDIS_URL` | No | Redis connection URL |

### Backend-Specific (MCP_PORTAL_ prefix)

Used by Go backend with Viper configuration system:

#### **Environment Detection**
| Variable | Description |
|----------|-------------|
| `MCP_PORTAL_ENV` | Environment name (development, staging, production) |

#### **Password Overrides**
| Variable | Description |
|----------|-------------|
| `MCP_PORTAL_DATABASE_PASSWORD` | Database password override |
| `MCP_PORTAL_REDIS_PASSWORD` | Redis password override |
| `MCP_PORTAL_AZURE_CLIENT_SECRET` | Azure client secret override |
| `MCP_PORTAL_JWT_SIGNING_KEY` | JWT signing key override |

### Development Only Variables

| Variable | Context | Description |
|----------|---------|-------------|
| `CUSTOM_KEY` | Next.js | Custom environment variable |
| `SITE_URL` | Sitemap | Site URL for sitemap generation |
| `SENTRY_DSN` | Server | Server-side Sentry configuration |
| `APP_ENV` | General | Application environment |
| `APP_VERSION` | General | Application version |
| `NEXT_PUBLIC_APP_VERSION` | Client | Client-side version display |
| `SERVER_NAME` | Sentry | Server name for Sentry |

## Security Guidelines

### ✅ Safe for Frontend (.env.local)
- All `NEXT_PUBLIC_*` variables
- Non-sensitive configuration (timeouts, themes, feature flags)
- Public Azure AD configuration (redirect URIs, scopes)

### ❌ Never in Frontend
- `AZURE_CLIENT_SECRET` - Confidential client credential
- `JWT_SECRET` - Server-side signing key
- `DATABASE_URL` - Database credentials
- `REDIS_URL` - Cache credentials
- Any password or API key

### 🔐 Server Environment Only
Set these at the container/server level:
- Docker environment variables
- Kubernetes secrets
- Cloud provider secret management
- Docker Compose environment files

## Environment-Specific Examples

### Development
```bash
# Frontend .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_DEBUG=true
NEXT_PUBLIC_LOG_LEVEL=DEBUG

# Server environment
AZURE_TENANT_ID=dev-tenant-id
AZURE_CLIENT_ID=dev-client-id
AZURE_CLIENT_SECRET=dev-secret
JWT_SECRET=dev-jwt-secret-at-least-32-chars
NODE_ENV=development
````

### Production

```bash
# Frontend .env.local
NEXT_PUBLIC_API_URL=https://api.company.com
NEXT_PUBLIC_WS_URL=wss://api.company.com
NEXT_PUBLIC_AZURE_REDIRECT_URI=https://portal.company.com/auth/callback

# Server environment (via secrets management)
AZURE_TENANT_ID=prod-tenant-id
AZURE_CLIENT_ID=prod-client-id
AZURE_CLIENT_SECRET=prod-secret-from-vault
JWT_SECRET=prod-jwt-secret-from-vault
DATABASE_URL=postgresql://user:pass@prod-db:5432/mcp_portal
REDIS_URL=redis://prod-redis:6379
NODE_ENV=production
SESSION_COOKIE_SECURE=true
```

## Validation

### Frontend Validation (T3 Env)

The frontend uses T3 Env with Zod for runtime validation:

- Build-time validation for required variables
- Type-safe access throughout the application
- Automatic environment variable discovery

### Backend Validation (Viper)

The backend uses Viper for configuration:

- Automatic environment variable mapping
- `MCP_PORTAL_` prefix support
- Default value management
- Configuration file merging

## Troubleshooting

### Common Issues

1. **"AZURE_CLIENT_ID is not defined"**

   - Check if server-side variables are set at container level
   - Don't put sensitive Azure variables in frontend .env.local

2. **API calls failing**

   - Verify `NEXT_PUBLIC_API_URL` matches backend service
   - Check if backend is running on specified port

3. **WebSocket connection issues**

   - Ensure `NEXT_PUBLIC_WS_URL` uses correct protocol (ws/wss)
   - Verify WebSocket endpoint is accessible

4. **Authentication not working**
   - Check Azure AD app registration matches redirect URIs
   - Verify tenant ID and client ID are correct
   - Ensure client secret is set on server-side

### Debug Steps

1. **Check T3 Env validation**: Look for build-time errors
2. **Verify variable access**: Use browser dev tools to check `process.env`
3. **Backend logs**: Check if server-side variables are loaded
4. **Network requests**: Verify API URLs are correct in browser network tab

## Migration from Legacy Configuration

### From QUICKSTART.md Variables

| Old Variable     | New Variable          | Notes                    |
| ---------------- | --------------------- | ------------------------ |
| `API_BASE_URL`   | `NEXT_PUBLIC_API_URL` | Use NEXT_PUBLIC variant  |
| `ENCRYPTION_KEY` | Remove                | Not used in current code |

### Security Fixes

- Remove `DATABASE_URL` from frontend
- Remove `REDIS_URL` from frontend
- Remove `AZURE_CLIENT_SECRET` from frontend
- Move all secrets to server environment

## Unified Configuration Solution

### Using the Unified .env.local File

The project now includes a unified configuration approach that allows both frontend and backend to use a single `.env.local` file:

1. **Unified Configuration File**: `/cmd/docker-mcp/portal/frontend/.env.local.unified.example`

   - Groups related variables together
   - Uses variable substitution to prevent duplication
   - Works for both Next.js frontend and Go backend

2. **Wrapper Scripts**: `/cmd/docker-mcp/portal/scripts/`
   - `start-with-env.sh` - Maps frontend variables to backend format
   - `start-dev.sh` - Starts both services with unified config
   - Automatically parses DATABASE_URL and REDIS_URL

### Variable Matching Requirements

**Critical Matches Required:**

1. **JWT Secret**: Must be IDENTICAL between services

   - Frontend: `JWT_SECRET`
   - Backend: `MCP_PORTAL_SECURITY_JWT_SIGNING_KEY`

2. **Azure AD Credentials**: Must match exactly

   - Frontend: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`
   - Backend: `MCP_PORTAL_AZURE_*` equivalents

3. **API Port**: Must coordinate
   - Frontend: `NEXT_PUBLIC_API_URL` must include correct port
   - Backend: `MCP_PORTAL_SERVER_PORT` must match

### Quick Start with Unified Configuration

```bash
# 1. Copy the unified example
cp cmd/docker-mcp/portal/frontend/.env.local.unified.example \
   cmd/docker-mcp/portal/frontend/.env.local

# 2. Edit with your values
vim cmd/docker-mcp/portal/frontend/.env.local

# 3. Start both services
./cmd/docker-mcp/portal/scripts/start-dev.sh
```

This reference provides the definitive guide for configuring environment variables based on actual code usage and security best practices.
