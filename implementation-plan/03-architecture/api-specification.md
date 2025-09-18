# API Specification

## Overview

RESTful API specification for the MCP Portal backend services. This API acts as a bridge between the web portal and the MCP Gateway CLI, providing secure access to CLI functionality through REST endpoints with real-time updates via WebSocket.

## Base Configuration

### Base URL

```
Production: https://mcp-portal.company.com/api/v1
Staging: https://staging-mcp-portal.company.com/api/v1
Development: http://localhost:8080/api/v1
```

### Headers

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <JWT_TOKEN>
X-Request-ID: <UUID>
X-Client-Version: 1.0.0
```

### Authentication

All endpoints except `/auth/login` and `/health` require JWT authentication via Bearer token.

### CLI Integration Pattern

Most endpoints in this API execute underlying CLI commands and parse their output. Each endpoint specifies:

- **CLI Command**: The exact command executed
- **Parser Type**: How output is processed (JSON, Table, Log, etc.)
- **Timeout**: Maximum execution time
- **Async**: Whether operation returns immediately with operation ID
- **Streaming**: Whether real-time updates are sent via WebSocket

## Authentication Endpoints

### POST /auth/login

Initiate Azure AD OAuth flow.

**Request:**

```json
{
  "redirect_uri": "https://portal.company.com/auth/callback"
}
```

**Response (302 Redirect):**

```
Location: https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize?...
```

### POST /auth/callback

Handle OAuth callback and exchange code for tokens.

**Request:**

```json
{
  "code": "AZURE_AUTH_CODE",
  "state": "STATE_TOKEN"
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
  "expires_in": 900,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@company.com",
    "name": "John Doe",
    "role": "standard_user"
  }
}
```

### POST /auth/refresh

Refresh access token using refresh token.

**Request:**

```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "expires_in": 900
}
```

### POST /auth/logout

Logout and invalidate session.

**Request:**

```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response (204 No Content)**

### GET /auth/me

Get current user information.

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@company.com",
  "name": "John Doe",
  "role": "standard_user",
  "created_at": "2024-01-01T00:00:00Z",
  "last_login": "2024-01-15T10:30:00Z"
}
```

## Server Management Endpoints

### GET /servers

List all available MCP servers from catalog.

**CLI Command**: `docker mcp server list --format json --user {user_id} [--catalog {type}] [--category {category}] [--search {query}]`
**Parser**: JSONParser
**Timeout**: 5s
**Async**: No
**Streaming**: No

**Query Parameters:**

- `catalog_type` (string): Filter by catalog type (predefined|custom|all)
- `category` (string): Filter by category
- `search` (string): Search by name or description
- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (default: 20)

**Response (200 OK):**

```json
{
  "servers": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "github",
      "display_name": "GitHub",
      "description": "GitHub API integration",
      "image": "docker/mcp-github:latest",
      "catalog_type": "predefined",
      "category": "development",
      "tags": ["vcs", "github", "development"],
      "user_config": {
        "enabled": true,
        "container_state": "running",
        "container_id": "abc123def456",
        "last_state_change": "2024-01-15T10:00:00Z"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

### GET /servers/{id}

Get detailed information about a specific server.

**CLI Command**: `docker mcp server inspect {server_id} --format json --user {user_id}`
**Parser**: JSONParser
**Timeout**: 3s
**Async**: No
**Streaming**: No

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "github",
  "display_name": "GitHub",
  "description": "GitHub API integration",
  "image": "docker/mcp-github:latest",
  "catalog_type": "predefined",
  "category": "development",
  "tags": ["vcs", "github", "development"],
  "metadata": {
    "version": "1.2.3",
    "author": "Docker Inc",
    "documentation": "https://docs.example.com"
  },
  "user_config": {
    "enabled": true,
    "container_state": "running",
    "container_id": "abc123def456",
    "last_state_change": "2024-01-15T10:00:00Z",
    "config_metadata": {
      "auto_start": true,
      "restart_policy": "always"
    }
  },
  "container_stats": {
    "cpu_usage": "0.5%",
    "memory_usage": "128MB",
    "uptime": "2h 30m"
  }
}
```

### POST /servers/{id}/enable

Enable a server for the current user.

**CLI Command**: `docker mcp server enable {server_id} --user {user_id} --config-file {temp_config_file} [--auto-start]`
**Parser**: JSONParser
**Timeout**: 30s
**Async**: Yes
**Streaming**: Yes

**Request:**

```json
{
  "config": {
    "auto_start": true,
    "restart_policy": "always"
  }
}
```

**Response (202 Accepted):**

```json
{
  "message": "Server enabling in progress",
  "operation_id": "op_550e8400",
  "estimated_time": 5
}
```

### POST /servers/{id}/disable

Disable a server for the current user.

**CLI Command**: `docker mcp server disable {server_id} --user {user_id} [--force]`
**Parser**: JSONParser
**Timeout**: 15s
**Async**: Yes
**Streaming**: Yes

**Response (202 Accepted):**

```json
{
  "message": "Server disabling in progress",
  "operation_id": "op_550e8401",
  "estimated_time": 3
}
```

### POST /servers/{id}/restart

Restart a server container.

**CLI Command**: `docker mcp server restart {server_id} --user {user_id}`
**Parser**: JSONParser
**Timeout**: 30s
**Async**: Yes
**Streaming**: Yes

**Response (202 Accepted):**

```json
{
  "message": "Server restarting",
  "operation_id": "op_550e8402",
  "estimated_time": 10
}
```

### GET /servers/{id}/logs

Get server container logs.

**CLI Command**: `docker mcp server logs {server_id} --user {user_id} [--lines {n}] [--since {timestamp}] [--follow]`
**Parser**: LogParser
**Timeout**: 60s (infinite for follow=true)
**Async**: No (Yes for follow=true)
**Streaming**: Yes (when follow=true)

**Query Parameters:**

- `lines` (integer): Number of lines to return (default: 100)
- `since` (string): RFC3339 timestamp to start from
- `follow` (boolean): Stream logs in real-time

**Response (200 OK):**

```json
{
  "logs": [
    {
      "timestamp": "2024-01-15T10:00:00Z",
      "level": "info",
      "message": "Server started successfully"
    }
  ]
}
```

## Bulk Operations

### POST /servers/bulk

Perform bulk operations on multiple servers.

**CLI Command**: `docker mcp server bulk {operation} --servers {server_ids} --user {user_id} [--config-file {temp_config}]`
**Parser**: JSONParser
**Timeout**: 300s
**Async**: Yes
**Streaming**: Yes

**Request:**

```json
{
  "operation": "enable",
  "server_ids": [
    "550e8400-e29b-41d4-a716-446655440001",
    "550e8400-e29b-41d4-a716-446655440002"
  ],
  "config": {
    "auto_start": true
  }
}
```

**Response (202 Accepted):**

```json
{
  "operation_id": "bulk_op_123456",
  "total_servers": 2,
  "status": "processing",
  "progress_url": "/api/v1/operations/bulk_op_123456"
}
```

### GET /operations/{id}

Get bulk operation status.

**Response (200 OK):**

```json
{
  "operation_id": "bulk_op_123456",
  "status": "in_progress",
  "total": 2,
  "completed": 1,
  "failed": 0,
  "details": [
    {
      "server_id": "550e8400-e29b-41d4-a716-446655440001",
      "status": "completed",
      "message": "Server enabled successfully"
    },
    {
      "server_id": "550e8400-e29b-41d4-a716-446655440002",
      "status": "processing",
      "message": "Starting container"
    }
  ]
}
```

## Configuration Management

### GET /config

Get user's configuration.

**CLI Command**: `docker mcp config get --user {user_id} --format json`
**Parser**: JSONParser
**Timeout**: 5s
**Async**: No
**Streaming**: No

**Response (200 OK):**

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "servers": [
    {
      "server_id": "550e8400-e29b-41d4-a716-446655440001",
      "enabled": true,
      "config": {
        "auto_start": true,
        "restart_policy": "always"
      }
    }
  ],
  "preferences": {
    "theme": "dark",
    "notifications": true
  }
}
```

### PUT /config

Update user's configuration.

**CLI Command**: `docker mcp config set --user {user_id} --config-file {temp_config_file}`
**Parser**: JSONParser
**Timeout**: 10s
**Async**: No
**Streaming**: No

**Request:**

```json
{
  "preferences": {
    "theme": "light",
    "notifications": false
  }
}
```

**Response (200 OK):**

```json
{
  "message": "Configuration updated successfully"
}
```

### GET /config/export

Export user configuration.

**CLI Command**: `docker mcp config export --user {user_id} --format {format}`
**Parser**: RawParser
**Timeout**: 10s
**Async**: No
**Streaming**: No

**Query Parameters:**

- `format` (string): Export format (json|yaml) (default: json)

**Response (200 OK):**

```json
{
  "version": "1.0",
  "exported_at": "2024-01-15T10:00:00Z",
  "user": "user@company.com",
  "servers": [
    {
      "name": "github",
      "enabled": true,
      "config": {}
    }
  ]
}
```

### POST /config/import

Import configuration from file.

**CLI Command**: `docker mcp config import --user {user_id} --config-file {temp_config_file}`
**Parser**: JSONParser
**Timeout**: 15s
**Async**: No
**Streaming**: No

**Request:**

```json
{
  "version": "1.0",
  "servers": [
    {
      "name": "github",
      "enabled": true,
      "config": {}
    }
  ]
}
```

**Response (200 OK):**

```json
{
  "message": "Configuration imported successfully",
  "imported": 1,
  "skipped": 0,
  "errors": []
}
```

## Custom Servers

### POST /catalog/custom

Add a custom server to catalog.

**CLI Command**: `docker mcp catalog add --user {user_id} --definition-file {temp_definition_file}`
**Parser**: JSONParser
**Timeout**: 15s
**Async**: No
**Streaming**: No

**Request:**

```json
{
  "name": "custom-tool",
  "display_name": "Custom Tool",
  "description": "My custom MCP server",
  "image": "myregistry/custom-tool:latest",
  "category": "utilities",
  "tags": ["custom", "tool"]
}
```

**Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440003",
  "name": "custom-tool",
  "catalog_type": "custom",
  "created_at": "2024-01-15T10:00:00Z"
}
```

### PUT /catalog/custom/{id}

Update custom server definition.

**CLI Command**: `docker mcp catalog update {server_id} --user {user_id} --definition-file {temp_definition_file}`
**Parser**: JSONParser
**Timeout**: 15s
**Async**: No
**Streaming**: No

**Request:**

```json
{
  "description": "Updated description",
  "image": "myregistry/custom-tool:v2"
}
```

**Response (200 OK):**

```json
{
  "message": "Custom server updated successfully"
}
```

### DELETE /catalog/custom/{id}

Remove custom server from catalog.

**CLI Command**: `docker mcp catalog remove {server_id} --user {user_id}`
**Parser**: JSONParser
**Timeout**: 10s
**Async**: No
**Streaming**: No

**Response (204 No Content)**

## Admin Endpoints

### GET /admin/users

List all users (admin only).

**CLI Command**: N/A (Portal database query)
**Parser**: N/A
**Timeout**: 5s
**Async**: No
**Streaming**: No

**Query Parameters:**

- `role` (string): Filter by role
- `active` (boolean): Filter by active status
- `search` (string): Search by email or name

**Response (200 OK):**

```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@company.com",
      "name": "John Doe",
      "role": "standard_user",
      "created_at": "2024-01-01T00:00:00Z",
      "last_login": "2024-01-15T10:00:00Z",
      "is_active": true,
      "server_count": 5,
      "active_containers": 3
    }
  ]
}
```

### PUT /admin/users/{id}/role

Update user role (admin only).

**CLI Command**: N/A (Portal database update)
**Parser**: N/A
**Timeout**: 5s
**Async**: No
**Streaming**: No

**Request:**

```json
{
  "role": "team_admin"
}
```

**Response (200 OK):**

```json
{
  "message": "User role updated successfully"
}
```

### GET /admin/audit

Get audit logs (admin only).

**CLI Command**: N/A (Portal database query)
**Parser**: N/A
**Timeout**: 10s
**Async**: No
**Streaming**: No

**Query Parameters:**

- `user_id` (string): Filter by user
- `action` (string): Filter by action type
- `from` (string): Start date (RFC3339)
- `to` (string): End date (RFC3339)
- `page` (integer): Page number
- `limit` (integer): Items per page

**Response (200 OK):**

```json
{
  "logs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440010",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_email": "user@company.com",
      "action": "server_enabled",
      "resource_type": "server",
      "resource_id": "550e8400-e29b-41d4-a716-446655440001",
      "details": {
        "server_name": "github"
      },
      "ip_address": "192.168.1.1",
      "timestamp": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### GET /admin/metrics

Get system metrics (admin only).

**CLI Command**: Mixed (Portal metrics + `docker mcp system info --format json`)
**Parser**: Mixed (JSONParser for CLI data)
**Timeout**: 15s
**Async**: No
**Streaming**: No

**Response (200 OK):**

```json
{
  "users": {
    "total": 150,
    "active": 120,
    "by_role": {
      "super_admin": 2,
      "team_admin": 10,
      "standard_user": 138
    }
  },
  "servers": {
    "total": 25,
    "predefined": 20,
    "custom": 5,
    "enabled": 180,
    "running_containers": 145
  },
  "system": {
    "uptime": "15d 6h 30m",
    "api_requests_24h": 45678,
    "avg_response_time_ms": 85,
    "error_rate": 0.02
  }
}
```

## Health & Status

### GET /health

Health check endpoint (no auth required).

**CLI Command**: `docker mcp version --format json` (to verify CLI availability)
**Parser**: JSONParser
**Timeout**: 5s
**Async**: No
**Streaming**: No

**Response (200 OK):**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:00:00Z",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "docker": "healthy"
  }
}
```

### GET /status

Detailed system status.

**CLI Command**: `docker mcp system info --format json`
**Parser**: JSONParser
**Timeout**: 10s
**Async**: No
**Streaming**: No

**Response (200 OK):**

```json
{
  "services": {
    "api": "operational",
    "database": "operational",
    "cache": "operational",
    "docker": "operational"
  },
  "metrics": {
    "response_time_ms": 45,
    "active_connections": 23,
    "queue_size": 0
  }
}
```

## WebSocket Events

### Connection

```
wss://mcp-portal.company.com/ws

Headers:
Authorization: Bearer <JWT_TOKEN>
```

### Event Types

#### Server State Changed

```json
{
  "type": "SERVER_STATE_CHANGED",
  "timestamp": "2024-01-15T10:00:00Z",
  "data": {
    "server_id": "550e8400-e29b-41d4-a716-446655440001",
    "old_state": "stopped",
    "new_state": "running",
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

#### Configuration Updated

```json
{
  "type": "CONFIG_UPDATED",
  "timestamp": "2024-01-15T10:00:00Z",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "changes": ["preferences", "servers"]
  }
}
```

#### Bulk Operation Progress

```json
{
  "type": "BULK_OPERATION_PROGRESS",
  "timestamp": "2024-01-15T10:00:00Z",
  "data": {
    "operation_id": "bulk_op_123456",
    "completed": 5,
    "total": 10,
    "status": "in_progress"
  }
}
```

## Error Responses

### 400 Bad Request

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request parameters",
    "details": {
      "field": "server_id",
      "error": "Must be a valid UUID"
    }
  }
}
```

### 401 Unauthorized

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token"
  }
}
```

### 403 Forbidden

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Insufficient permissions"
  }
}
```

### 404 Not Found

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found",
    "details": {
      "resource": "server",
      "id": "550e8400-e29b-41d4-a716-446655440001"
    }
  }
}
```

### 409 Conflict

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Resource already exists",
    "details": {
      "resource": "custom_server",
      "name": "custom-tool"
    }
  }
}
```

### 429 Too Many Requests

```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many requests",
    "retry_after": 60
  }
}
```

### 500 Internal Server Error

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "An internal error occurred",
    "trace_id": "abc-123-def"
  }
}
```

### 503 Service Unavailable

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Service temporarily unavailable",
    "retry_after": 30
  }
}
```

## Rate Limiting

All endpoints are rate limited:

- **Standard endpoints**: 60 requests/minute
- **Bulk operations**: 10 requests/minute
- **Admin endpoints**: 30 requests/minute

Rate limit headers:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1673784000
```

## Pagination

Standard pagination parameters:

- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)

Response includes pagination metadata:

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

## API Versioning

API version is included in the URL path:

- Current: `/api/v1/`
- Future: `/api/v2/`

Version deprecation notice via headers:

```http
Sunset: Sat, 31 Dec 2024 23:59:59 GMT
Link: <https://docs.api.com/migrations/v2>; rel="deprecation"
```
