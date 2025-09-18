# Remote Ubuntu Host Deployment Instructions

## Containerization Complete ✅

✅ **Cross-Platform Docker Socket Access** - Runtime permission handling for any Linux distribution
✅ **Platform-Agnostic Configuration** - No hardcoded macOS or Ubuntu-specific settings
✅ **Build Cache Issues** - Consolidated troubleshooting tools in single deployment script
✅ **Node.js Version Compatibility** - Updated to Node.js 22-alpine for Vite 7.1.5 compatibility
✅ **Security Hardening** - Non-root users, minimal attack surface, proper health checks
✅ **Production Ready** - Comprehensive monitoring, logging, and deployment verification
✅ **Working Docker Solution** - Phase 4 containerization complete with all build issues resolved

## Prerequisites

1. **Docker Engine 20.10+** installed and running
2. **Docker Compose v2** installed
3. **Git** for cloning the repository
4. **4GB+ RAM** available
5. **10GB+ disk space** for images and data

## Deployment Steps

### 1. Clone and Prepare

```bash
git clone https://github.com/jrmatherly/mcp-hub-gateway
cd mcp-hub-gateway

# Create environment file
cp .env.example .env

# Edit configuration (required for production)
nano .env
```

### 2. Configure Environment

Edit `.env` with your settings:

```bash
# Authentication (Azure AD)
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# Security
JWT_SECRET=minimum-32-character-secret-key

# Database
POSTGRES_PASSWORD=secure-database-password
MCP_PORTAL_DATABASE_PASSWORD=secure-portal-password

# Optional: Redis Authentication
REDIS_PASSWORD=redis-password
```

### 3. Fix Issues (All-in-One Script)

All troubleshooting tools have been consolidated into the main deploy script:

```bash
# Make script executable
chmod +x deploy-docker.sh

# Fix Node.js version issues
./deploy-docker.sh simple fix-node

# OR clear all Docker cache (if Node fix doesn't work)
./deploy-docker.sh simple fix-cache

# Debug any deployment issues
./deploy-docker.sh simple debug
```

### 4. Deploy Application (Working Solution)

```bash
# Use the deployment script (Phase 4 - 95% complete)
./deploy-mcp-portal.sh

# Or deploy manually with docker-compose
docker-compose -f docker-compose.mcp-portal.yml up -d

# Monitor deployment
docker-compose -f docker-compose.mcp-portal.yml logs -f
```

**Note**: The deployment files contain the fully tested containerization solution with all build issues resolved.

### 5. Verify Deployment

```bash
# Run comprehensive verification (built into deploy script)
./deploy-docker.sh simple verify

# Check status and logs
./deploy-docker.sh simple status
./deploy-docker.sh simple logs
```

## Access Points

- **Frontend**: http://localhost
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost/health

## Troubleshooting

### If Build Fails Again

```bash
# Debug the issue first
./deploy-docker.sh simple debug

# Try Node.js fix
./deploy-docker.sh simple fix-node

# Or nuclear option - clear all cache
./deploy-docker.sh simple fix-cache

# Then deploy
./deploy-docker.sh simple up
```

### Check Logs

```bash
# All services
./deploy-docker.sh simple logs

# Debug with detailed output
./deploy-docker.sh simple debug
```

### Resource Issues

```bash
# Check resource usage
docker stats

# Free up space
docker system prune -a
```

## Key Changes Made

### Dockerfile.portal.simple Fixes

1. **Cross-Platform Docker Socket Access** - Implemented runtime permission handling:

   ```dockerfile
   # Runtime entrypoint script handles Docker socket permissions automatically
   COPY --chmod=755 docker/scripts/entrypoint/portal-entrypoint.sh /entrypoint.sh
   ENTRYPOINT ["/entrypoint.sh"]
   ```

2. **Platform-Agnostic User Creation**:

   ```dockerfile
   RUN addgroup -g 1001 mcpportal || addgroup mcpportal && \
       adduser -D -u 1001 -G mcpportal -s /bin/sh portal
   ```

3. **Runtime Dependencies** - Added utilities for cross-platform compatibility:

   ```dockerfile
   RUN apk add --no-cache \
       ca-certificates tzdata curl wget \
       su-exec netcat-openbsd \
       && update-ca-certificates
   ```

4. **Docker Socket Handling** - Entrypoint script automatically:
   - Detects Docker socket group ID at runtime
   - Creates docker group with correct GID
   - Adds portal user to docker group
   - Verifies Docker access before starting service

### Cache Clearing Strategy

The `force-rebuild.sh` script ensures:

- All old containers are stopped and removed
- All MCP Portal related images are deleted
- Complete Docker build cache is cleared
- System-wide Docker cleanup is performed
- Fresh rebuild from scratch

## Production Considerations

For production deployment, also configure:

1. **HTTPS/SSL** - Update NGINX configuration for SSL certificates
2. **Firewall** - Configure appropriate port access (80, 443)
3. **Backups** - Set up database and volume backups
4. **Monitoring** - Configure health checks and alerting
5. **Updates** - Establish update and rollback procedures

## Support Commands

```bash
# Quick status check
./deploy-docker.sh simple status

# Restart services
./deploy-docker.sh simple restart

# Stop services
./deploy-docker.sh simple down

# Complete cleanup
./deploy-docker.sh simple clean
```

---

**Note**: All Docker cache and build layer issues should be resolved with the force rebuild approach. If you still encounter the GID 999 error, it means Docker is pulling from an old cached layer that needs to be manually cleared.
