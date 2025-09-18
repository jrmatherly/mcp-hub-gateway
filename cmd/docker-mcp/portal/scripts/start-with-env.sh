#!/bin/bash
# =================================================================
# MCP Portal Unified Environment Startup Script
# =================================================================
# This script sources the unified .env.local file and starts
# the backend service with proper environment variable mapping
# =================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default paths
ENV_FILE="${ENV_FILE:-./frontend/.env.local}"
PORTAL_CMD="${PORTAL_CMD:-docker mcp portal serve}"

# Check if env file exists
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${RED}Error: Environment file not found: $ENV_FILE${NC}"
    echo "Please copy .env.local.unified.example to .env.local and configure it"
    exit 1
fi

echo -e "${GREEN}Loading environment from: $ENV_FILE${NC}"

# Source the environment file
set -a  # Export all variables
source "$ENV_FILE"
set +a

# Map frontend variables to backend format if not already set
# This allows the unified file to work with the Go backend

# Azure AD Configuration
export MCP_PORTAL_AZURE_TENANT_ID="${MCP_PORTAL_AZURE_TENANT_ID:-$AZURE_TENANT_ID}"
export MCP_PORTAL_AZURE_CLIENT_ID="${MCP_PORTAL_AZURE_CLIENT_ID:-$AZURE_CLIENT_ID}"
export MCP_PORTAL_AZURE_CLIENT_SECRET="${MCP_PORTAL_AZURE_CLIENT_SECRET:-$AZURE_CLIENT_SECRET}"

# JWT Security
export MCP_PORTAL_SECURITY_JWT_SIGNING_KEY="${MCP_PORTAL_SECURITY_JWT_SIGNING_KEY:-$JWT_SECRET}"

# Server Configuration
export MCP_PORTAL_SERVER_PORT="${MCP_PORTAL_SERVER_PORT:-$API_PORT}"
export MCP_PORTAL_SERVER_HOST="${MCP_PORTAL_SERVER_HOST:-$MCP_PORTAL_SERVER_HOST}"

# Database Configuration (if DATABASE_URL is provided, parse it)
if [ -n "$DATABASE_URL" ]; then
    # Parse PostgreSQL URL: postgresql://user:pass@host:port/database
    if [[ "$DATABASE_URL" =~ postgresql://([^:]+):([^@]+)@([^:]+):([^/]+)/(.+) ]]; then
        export MCP_PORTAL_DATABASE_USERNAME="${MCP_PORTAL_DATABASE_USERNAME:-${BASH_REMATCH[1]}}"
        export MCP_PORTAL_DATABASE_PASSWORD="${MCP_PORTAL_DATABASE_PASSWORD:-${BASH_REMATCH[2]}}"
        export MCP_PORTAL_DATABASE_HOST="${MCP_PORTAL_DATABASE_HOST:-${BASH_REMATCH[3]}}"
        export MCP_PORTAL_DATABASE_PORT="${MCP_PORTAL_DATABASE_PORT:-${BASH_REMATCH[4]}}"
        export MCP_PORTAL_DATABASE_DATABASE="${MCP_PORTAL_DATABASE_DATABASE:-${BASH_REMATCH[5]}}"
        echo -e "${GREEN}Parsed DATABASE_URL successfully${NC}"
    fi
fi

# Redis Configuration (if REDIS_URL is provided, parse it)
if [ -n "$REDIS_URL" ]; then
    # Parse Redis URL: redis://[:password]@host:port[/db]
    if [[ "$REDIS_URL" =~ redis://([^@]+@)?([^:]+):([^/]+)(/([0-9]+))? ]]; then
        export MCP_PORTAL_REDIS_ADDRS="${MCP_PORTAL_REDIS_ADDRS:-${BASH_REMATCH[2]}:${BASH_REMATCH[3]}}"
        export MCP_PORTAL_REDIS_DB="${MCP_PORTAL_REDIS_DB:-${BASH_REMATCH[5]:-0}}"
        echo -e "${GREEN}Parsed REDIS_URL successfully${NC}"
    fi
fi

# Environment mode
export MCP_PORTAL_ENV="${MCP_PORTAL_ENV:-$NODE_ENV}"

# Validate required variables
MISSING_VARS=()

# Check required variables
[ -z "$MCP_PORTAL_AZURE_TENANT_ID" ] && MISSING_VARS+=("AZURE_TENANT_ID")
[ -z "$MCP_PORTAL_AZURE_CLIENT_ID" ] && MISSING_VARS+=("AZURE_CLIENT_ID")
[ -z "$MCP_PORTAL_AZURE_CLIENT_SECRET" ] && MISSING_VARS+=("AZURE_CLIENT_SECRET")
[ -z "$MCP_PORTAL_SECURITY_JWT_SIGNING_KEY" ] && MISSING_VARS+=("JWT_SECRET")

if [ ${#MISSING_VARS[@]} -gt 0 ]; then
    echo -e "${RED}Error: Missing required environment variables:${NC}"
    printf '%s\n' "${MISSING_VARS[@]}"
    echo -e "${YELLOW}Please configure these in your .env.local file${NC}"
    exit 1
fi

# Show configuration summary
echo -e "${GREEN}=== Configuration Summary ===${NC}"
echo "Environment: ${MCP_PORTAL_ENV:-development}"
echo "Server: ${MCP_PORTAL_SERVER_HOST:-0.0.0.0}:${MCP_PORTAL_SERVER_PORT:-8080}"
echo "Azure Tenant: ${MCP_PORTAL_AZURE_TENANT_ID}"
echo "Azure Client: ${MCP_PORTAL_AZURE_CLIENT_ID}"
echo "JWT Secret: [CONFIGURED]"

if [ -n "$MCP_PORTAL_DATABASE_HOST" ]; then
    echo "Database: ${MCP_PORTAL_DATABASE_HOST}:${MCP_PORTAL_DATABASE_PORT:-5432}/${MCP_PORTAL_DATABASE_DATABASE}"
fi

if [ -n "$MCP_PORTAL_REDIS_ADDRS" ]; then
    echo "Redis: ${MCP_PORTAL_REDIS_ADDRS}"
fi

echo -e "${GREEN}=============================${NC}"

# Start the portal backend
echo -e "${GREEN}Starting MCP Portal backend...${NC}"
exec $PORTAL_CMD "$@"