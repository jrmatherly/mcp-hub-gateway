# MCP Portal Startup Scripts

This directory contains scripts for starting the MCP Portal with unified environment configuration.

## Scripts

### start-with-env.sh

Wrapper script that sources the unified `.env.local` file and starts the backend service with proper environment variable mapping.

**Features:**

- Sources frontend's `.env.local` file
- Maps variables to backend's `MCP_PORTAL_` format
- Parses DATABASE_URL and REDIS_URL automatically
- Validates required configuration
- Shows configuration summary before starting

**Usage:**

```bash
# Basic usage (reads from ./frontend/.env.local)
./scripts/start-with-env.sh

# Custom env file location
ENV_FILE=/path/to/.env.local ./scripts/start-with-env.sh

# Pass additional arguments to portal serve command
./scripts/start-with-env.sh --verbose --debug
```

### start-dev.sh

Development startup script that runs both frontend and backend services using the unified configuration.

**Features:**

- Automatically uses tmux or screen if available
- Falls back to foreground execution if neither is installed
- Creates .env.local from example if missing
- Manages both services with single command
- Graceful shutdown on Ctrl-C

**Usage:**

```bash
# Start both services in development mode
./scripts/start-dev.sh
```

**With tmux:**

- Creates session named 'mcp-portal' with 3 windows:
  - Window 0: Backend service
  - Window 1: Frontend development server
  - Window 2: Log viewer
- Use Ctrl-B + window number to switch between services
- Use Ctrl-B + D to detach from session

**With screen:**

- Creates session named 'mcp-portal' with 2 screens:
  - Screen 0: Backend service
  - Screen 1: Frontend development server
- Use Ctrl-A + window number to switch between services
- Use Ctrl-A + D to detach from session

**Without tmux/screen:**

- Runs both services in the foreground
- Shows output from both services
- Use Ctrl-C to stop both services

## Environment Variable Mapping

The scripts automatically map between frontend and backend variable names:

| Frontend Variable     | Backend Variable                      |
| --------------------- | ------------------------------------- |
| `AZURE_TENANT_ID`     | `MCP_PORTAL_AZURE_TENANT_ID`          |
| `AZURE_CLIENT_ID`     | `MCP_PORTAL_AZURE_CLIENT_ID`          |
| `AZURE_CLIENT_SECRET` | `MCP_PORTAL_AZURE_CLIENT_SECRET`      |
| `JWT_SECRET`          | `MCP_PORTAL_SECURITY_JWT_SIGNING_KEY` |
| `API_PORT`            | `MCP_PORTAL_SERVER_PORT`              |
| `NODE_ENV`            | `MCP_PORTAL_ENV`                      |

## URL Parsing

The scripts can parse connection URLs into individual components:

**DATABASE_URL parsing:**

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/mydb
# Becomes:
MCP_PORTAL_DATABASE_USERNAME=user
MCP_PORTAL_DATABASE_PASSWORD=pass
MCP_PORTAL_DATABASE_HOST=localhost
MCP_PORTAL_DATABASE_PORT=5432
MCP_PORTAL_DATABASE_DATABASE=mydb
```

**REDIS_URL parsing:**

```bash
REDIS_URL=redis://localhost:6379/0
# Becomes:
MCP_PORTAL_REDIS_ADDRS=localhost:6379
MCP_PORTAL_REDIS_DB=0
```

## Installation

Make the scripts executable:

```bash
chmod +x scripts/*.sh
```

## Docker Integration

To use with Docker Compose, you can source the environment in your docker-compose.yml:

```yaml
services:
  backend:
    image: mcp-portal-backend
    env_file: ./frontend/.env.local
    environment:
      # Map variables using Docker Compose variable substitution
      MCP_PORTAL_AZURE_TENANT_ID: ${AZURE_TENANT_ID}
      MCP_PORTAL_AZURE_CLIENT_ID: ${AZURE_CLIENT_ID}
      MCP_PORTAL_AZURE_CLIENT_SECRET: ${AZURE_CLIENT_SECRET}
      MCP_PORTAL_SECURITY_JWT_SIGNING_KEY: ${JWT_SECRET}
```

## Troubleshooting

### Backend won't start

- Check that all required variables are set in `.env.local`
- Verify JWT_SECRET is at least 32 characters
- Ensure PostgreSQL and Redis are running if configured

### Frontend can't connect to backend

- Verify `NEXT_PUBLIC_API_URL` matches the backend port
- Check that backend is running on the expected port
- Ensure CORS is configured correctly

### Variables not being recognized

- Make sure variable names match exactly (case-sensitive)
- Check for typos in the `.env.local` file
- Verify the file is being sourced from the correct location

### Permission denied

- Make scripts executable: `chmod +x scripts/*.sh`
- Check file ownership and permissions
