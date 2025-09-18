# Docker Image Build Fix - Service Startup Solution

_Date: 2025-09-17_

## Issue Identified

The service was failing with:

```
Error response from daemon: pull access denied for mcp-portal-frontend, repository does not exist
Error response from daemon: pull access denied for mcp-portal-backend, repository does not exist
```

## Root Cause

The systemd service was trying to **pull** images `mcp-portal-frontend` and `mcp-portal-backend` from Docker Hub, but these images:

1. Don't exist in any public registry
2. Need to be built locally from Dockerfiles
3. Were never built before trying to start the service

Additionally, the build context was misaligned:

- Repository cloned to: `/opt/docker/appdata/mcp-hub-gateway/`
- Docker Compose expects: `/opt/mcp-portal/` as build context
- Source code was not accessible during build

## Fixes Applied

### 1. Updated Systemd Service File

**File**: `docker/production/mcp-portal.service`

Added build step before pull:

```bash
# Build images if they don't exist (backend and frontend are local builds)
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml build --quiet
# Pull only external images (postgres, redis, nginx)
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml pull --ignore-buildable-images --quiet
```

### 2. Updated Installation Script

**File**: `docker/production/install-production.sh`

#### Path Fixes

- Fixed PROJECT_ROOT calculation (go up 2 directories)
- Removed incorrect `systemd/` subdirectory reference

#### Build Context Solution

Instead of copying the entire repository, the script now creates **symlinks** from `/opt/mcp-portal/` to the actual source in `/opt/docker/appdata/mcp-hub-gateway/`:

```bash
# Symlinks created for build context:
/opt/mcp-portal/cmd -> /opt/docker/appdata/mcp-hub-gateway/cmd
/opt/mcp-portal/pkg -> /opt/docker/appdata/mcp-hub-gateway/pkg
/opt/mcp-portal/go.mod -> /opt/docker/appdata/mcp-hub-gateway/go.mod
/opt/mcp-portal/go.sum -> /opt/docker/appdata/mcp-hub-gateway/go.sum
/opt/mcp-portal/docker -> /opt/docker/appdata/mcp-hub-gateway/docker
```

This approach:

- Avoids duplicating the entire repository
- Maintains single source of truth
- Allows Docker build to access source code
- Preserves disk space

### 3. Docker Compose Volume Conflict Fix

**File**: `docker-compose.prod.yaml`

Removed duplicate volume mount for `/var/cache/nextjs` that was causing:

```
services.frontend.volumes[0]: target /var/cache/nextjs already mounted as services.frontend.tmpfs[2]
```

## How the Solution Works

1. **Installation Script** (`install-production.sh`):

   - Copies only essential files (Dockerfiles, .env, docker-compose.prod.yaml)
   - Creates symlinks to source directories
   - Sets up proper build context in `/opt/mcp-portal/`

2. **Systemd Service** (`mcp-portal.service`):

   - First builds local images (backend, frontend)
   - Then pulls external images (postgres, redis, nginx)
   - Starts all services with docker compose

3. **Docker Compose** (`docker-compose.prod.yaml`):
   - Uses build context `.` which is now `/opt/mcp-portal/`
   - Accesses source through symlinks
   - Builds images with proper Dockerfiles

## Manual Fix Steps (For Current Installation)

### Option 1: Quick Manual Fix

```bash
# 1. Stop the failing service
sudo systemctl stop mcp-portal

# 2. Create symlinks for build context
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/cmd /opt/mcp-portal/cmd
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/pkg /opt/mcp-portal/pkg
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/go.mod /opt/mcp-portal/go.mod
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/go.sum /opt/mcp-portal/go.sum
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/docker /opt/mcp-portal/docker

# 3. Copy Dockerfiles if not already present
sudo cp /opt/docker/appdata/mcp-hub-gateway/Dockerfile.portal /opt/mcp-portal/
sudo cp /opt/docker/appdata/mcp-hub-gateway/Dockerfile.frontend /opt/mcp-portal/

# 4. Build the images manually
cd /opt/mcp-portal
sudo docker compose --file docker-compose.prod.yaml build

# 5. Copy updated service file
sudo cp /opt/docker/appdata/mcp-hub-gateway/docker/production/mcp-portal.service /etc/systemd/system/
sudo systemctl daemon-reload

# 6. Start the service
sudo systemctl start mcp-portal
sudo systemctl status mcp-portal
```

### Option 2: Full Reinstall with Fixes

```bash
# 1. Stop and remove old installation
sudo systemctl stop mcp-portal
sudo systemctl disable mcp-portal
sudo rm -rf /opt/mcp-portal

# 2. Re-run installation with updated script
cd /opt/docker/appdata/mcp-hub-gateway/docker/production
sudo ./install-production.sh

# 3. Configure environment
sudo nano /opt/mcp-portal/.env
# Update with real values for AZURE_TENANT_ID, JWT_SECRET, etc.

# 4. Start service
sudo systemctl start mcp-portal
```

## Build Process Explanation

The application consists of:

- **External Images** (pulled from Docker Hub):

  - `postgres:17-alpine`
  - `redis:8-alpine`
  - `nginx:1.27-alpine`

- **Local Images** (must be built):
  - `mcp-portal-backend` (from Dockerfile.portal)
  - `mcp-portal-frontend` (from Dockerfile.frontend)

## Verification

After applying the fix:

```bash
# Check if images exist
sudo docker images | grep mcp-portal

# Should show:
# mcp-portal-backend    latest    xxxxx
# mcp-portal-frontend   latest    xxxxx

# Check service status
sudo systemctl status mcp-portal

# Check running containers
cd /opt/mcp-portal
sudo docker compose ps
```

## Important Notes

1. **First-time build**: The initial build will take several minutes as it compiles Go code and builds the Next.js application
2. **Build cache**: Subsequent builds will be faster due to Docker layer caching
3. **Symlinks advantage**: Changes to source code are immediately available for rebuilds without copying
4. **Environment variables**: Still need to be configured properly in `.env` file

## Summary of All Issues Fixed

The service was failing due to multiple issues:

1. **Read-only filesystem error**: Docker Compose couldn't create `/root/.docker` directory due to systemd's `ProtectSystem=full` setting
2. **Missing Docker images**: Service tried to pull local images that need to be built
3. **Build context misalignment**: Source code wasn't accessible through symlinks

Solutions applied:

1. **Fixed systemd security settings**: Changed `ProtectSystem` from `full` to `strict` and added `/root` to `ReadWritePaths`
2. **Added build step**: Service now builds images before attempting to pull
3. **Created proper symlinks**: Installation script creates all necessary symlinks for build context
4. **Pre-created Docker directory**: Installation script ensures `/root/.docker` exists

The service can now properly build and run the application containers. The symlink approach is more efficient than copying the entire repository and maintains a single source of truth.
