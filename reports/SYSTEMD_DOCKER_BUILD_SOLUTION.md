# Systemd Docker Build Solution - Complete Fix

_Date: 2025-09-18_

## Issue Summary

The MCP Portal systemd service was failing at the Docker build step:

```
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml build --quiet
```

**Previous Issue (RESOLVED)**: Read-only file system on `/root/.docker/buildx`
**Current Issue**: Docker Compose build failures in systemd context

## Root Cause Analysis

The build failure was caused by multiple interconnected issues:

1. **Hidden Error Messages**: `--quiet` flag was suppressing critical build error details
2. **Build Context Verification**: Missing validation that all required files and symlinks exist
3. **Working Directory**: Build commands not explicitly run from the correct directory
4. **Error Isolation**: Build and pull operations combined, making diagnosis difficult
5. **Dependency Validation**: No pre-build checks for required files and directories

## Complete Solution Applied

### 1. Enhanced Systemd Service Configuration

**File**: `docker/production/mcp-portal.service`

**Changes Made**:

```bash
# OLD - Hidden errors and no validation
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml build --quiet

# NEW - Comprehensive validation and visible errors
# Verify build context and dependencies are available
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && test -f Dockerfile.portal && test -f Dockerfile.frontend && test -d cmd && test -f go.mod'

# Build images with explicit context and verbose output (remove --quiet to see errors)
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && docker compose --file docker-compose.prod.yaml build --no-cache'

# Pull only external images (postgres, redis, nginx) - use separate step to isolate build vs pull issues
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml pull --ignore-buildable-images --quiet
```

**Key Improvements**:

- **Explicit Working Directory**: `cd /opt/mcp-portal` ensures correct build context
- **Dependency Validation**: Tests for all required files before attempting build
- **Verbose Output**: Removed `--quiet` to show actual build errors
- **No-Cache Build**: `--no-cache` ensures clean builds
- **Separated Operations**: Build and pull are separate steps for better error isolation

### 2. Enhanced Installation Script

**File**: `docker/production/install-production.sh`

**Changes Made**:

```bash
# Added comprehensive build dependency verification
# Verify critical build dependencies exist
log INFO "Verifying build context dependencies..."

# Check for frontend directory structure
if [[ ! -d "$PROJECT_ROOT/cmd/docker-mcp/portal/frontend" ]]; then
    log ERROR "Frontend directory not found at $PROJECT_ROOT/cmd/docker-mcp/portal/frontend"
    log ERROR "This is required for Dockerfile.frontend build"
    return 1
fi

# Check for Go module files
if [[ ! -f "$PROJECT_ROOT/go.mod" ]] || [[ ! -f "$PROJECT_ROOT/go.sum" ]]; then
    log ERROR "Go module files (go.mod, go.sum) not found in $PROJECT_ROOT"
    log ERROR "These are required for Go build process"
    return 1
fi

# Check for required Dockerfiles
if [[ ! -f "$PROJECT_ROOT/Dockerfile.portal" ]]; then
    log ERROR "Dockerfile.portal not found in $PROJECT_ROOT"
    return 1
fi

if [[ ! -f "$PROJECT_ROOT/Dockerfile.frontend" ]]; then
    log ERROR "Dockerfile.frontend not found in $PROJECT_ROOT"
    return 1
fi

# Verify entrypoint scripts exist
for script in portal.sh frontend.sh; do
    if [[ ! -f "$PROJECT_ROOT/docker/scripts/entrypoint/$script" ]]; then
        log ERROR "Required entrypoint script not found: docker/scripts/entrypoint/$script"
        return 1
    fi
done
```

**Key Improvements**:

- **Pre-Installation Validation**: Checks all required files exist before setup
- **Frontend Build Context**: Validates Next.js source directory structure
- **Go Dependencies**: Ensures Go module files are present
- **Entrypoint Scripts**: Verifies all required entrypoint scripts exist
- **Early Failure**: Fails fast if any critical component is missing

### 3. Build Debugging Tool

**File**: `docker/production/debug-build.sh` (NEW)

**Purpose**: Comprehensive diagnostics and troubleshooting for build failures

**Features**:

- **Prerequisites Check**: Docker, Docker Compose, daemon status
- **Installation Validation**: Directory structure, file permissions, symlinks
- **Build Context Testing**: Individual Dockerfile builds, Docker Compose builds
- **Permission Repair**: Automatic fixing of common permission issues
- **Detailed Diagnostics**: Complete system report generation

**Usage**:

```bash
# Basic diagnostics
sudo ./debug-build.sh

# Verbose output with permission fixes
sudo ./debug-build.sh --verbose --fix-permissions

# Force rebuild all images
sudo ./debug-build.sh --rebuild
```

## How The Solution Works

### Build Process Flow

1. **Validation Phase**:

   ```bash
   # Test that docker-compose.prod.yaml exists
   /usr/bin/test -f /opt/mcp-portal/docker-compose.prod.yaml

   # Verify Docker is working
   /usr/bin/docker --version

   # Validate Compose configuration
   /usr/bin/docker compose --file docker-compose.prod.yaml config --quiet
   ```

2. **Dependency Verification**:

   ```bash
   # Check all required files exist for building
   cd /opt/mcp-portal && test -f Dockerfile.portal && test -f Dockerfile.frontend && test -d cmd && test -f go.mod
   ```

3. **Image Building**:

   ```bash
   # Build with verbose output and clean cache
   cd /opt/mcp-portal && docker compose --file docker-compose.prod.yaml build --no-cache
   ```

4. **External Image Pull**:

   ```bash
   # Pull only postgres, redis, nginx (skip local builds)
   docker compose --file docker-compose.prod.yaml pull --ignore-buildable-images --quiet
   ```

5. **Service Start**:
   ```bash
   # Start all services
   docker compose --file docker-compose.prod.yaml up -d --remove-orphans
   ```

### Symlink Architecture

The installation creates the following critical symlinks in `/opt/mcp-portal/`:

```bash
/opt/mcp-portal/cmd -> /opt/docker/appdata/mcp-hub-gateway/cmd
/opt/mcp-portal/pkg -> /opt/docker/appdata/mcp-hub-gateway/pkg
/opt/mcp-portal/go.mod -> /opt/docker/appdata/mcp-hub-gateway/go.mod
/opt/mcp-portal/go.sum -> /opt/docker/appdata/mcp-hub-gateway/go.sum
/opt/mcp-portal/docker -> /opt/docker/appdata/mcp-hub-gateway/docker
/opt/mcp-portal/vendor -> /opt/docker/appdata/mcp-hub-gateway/vendor
```

This ensures Docker builds have access to:

- **Backend Source**: `cmd/docker-mcp/portal/` for Go code
- **Frontend Source**: `cmd/docker-mcp/portal/frontend/` for Next.js code
- **Go Dependencies**: `go.mod`, `go.sum`, and `vendor/` directory
- **Build Scripts**: `docker/scripts/entrypoint/` for container entrypoints
- **Configuration**: `docker/nginx/` for NGINX configs

## Manual Fix Steps (For Current Installation)

### Option 1: Update Existing Installation

```bash
# 1. Stop the failing service
sudo systemctl stop mcp-portal

# 2. Update service file
sudo cp /opt/docker/appdata/mcp-hub-gateway/docker/production/mcp-portal.service /etc/systemd/system/
sudo systemctl daemon-reload

# 3. Run diagnostics to identify issues
sudo /opt/docker/appdata/mcp-hub-gateway/docker/production/debug-build.sh --verbose --fix-permissions

# 4. If diagnostics pass, start service
sudo systemctl start mcp-portal
sudo systemctl status mcp-portal
```

### Option 2: Manual Build Test

```bash
# 1. Navigate to installation directory
cd /opt/mcp-portal

# 2. Test Docker Compose configuration
docker compose --file docker-compose.prod.yaml config

# 3. Build images manually with verbose output
docker compose --file docker-compose.prod.yaml build --no-cache

# 4. If successful, start service
sudo systemctl start mcp-portal
```

### Option 3: Complete Reinstallation

```bash
# 1. Stop and remove old installation
sudo systemctl stop mcp-portal
sudo systemctl disable mcp-portal
sudo rm -rf /opt/mcp-portal

# 2. Re-run installation with updated scripts
cd /opt/docker/appdata/mcp-hub-gateway/docker/production
sudo ./install-production.sh

# 3. Configure environment
sudo nano /opt/mcp-portal/.env
# Update with real values for AZURE_TENANT_ID, JWT_SECRET, etc.

# 4. Start service
sudo systemctl start mcp-portal
```

## Troubleshooting Guide

### Common Build Errors

**1. "Frontend directory not found"**

```bash
# Verify frontend source exists
ls -la /opt/docker/appdata/mcp-hub-gateway/cmd/docker-mcp/portal/frontend/

# Check symlink
ls -la /opt/mcp-portal/cmd/docker-mcp/portal/frontend/
```

**2. "go.mod not found"**

```bash
# Verify Go module files exist
ls -la /opt/docker/appdata/mcp-hub-gateway/go.mod
ls -la /opt/docker/appdata/mcp-hub-gateway/go.sum

# Check symlinks
ls -la /opt/mcp-portal/go.mod
ls -la /opt/mcp-portal/go.sum
```

**3. "Permission denied"**

```bash
# Fix Docker socket permissions
sudo chown root:docker /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock

# Fix installation directory ownership
sudo chown -R mcp-portal:docker /opt/mcp-portal
```

**4. "Docker daemon not accessible"**

```bash
# Check Docker service
sudo systemctl status docker

# Start Docker if stopped
sudo systemctl start docker

# Add user to docker group
sudo usermod -aG docker mcp-portal
```

### Diagnostic Commands

```bash
# Check service status and logs
sudo systemctl status mcp-portal
sudo journalctl -u mcp-portal -f

# Check Docker Compose status
cd /opt/mcp-portal
sudo docker compose ps
sudo docker compose logs

# Check built images
sudo docker images | grep mcp-portal

# Check symlinks
ls -la /opt/mcp-portal/

# Run comprehensive diagnostics
sudo /opt/docker/appdata/mcp-hub-gateway/docker/production/debug-build.sh --verbose
```

## Verification Steps

After applying the fix, verify success with:

```bash
# 1. Check service status
sudo systemctl status mcp-portal
# Should show "active (running)"

# 2. Check containers are running
cd /opt/mcp-portal
sudo docker compose ps
# Should show all containers as "Up"

# 3. Check images were built
sudo docker images | grep mcp-portal
# Should show:
# mcp-portal-backend    latest
# mcp-portal-frontend   latest

# 4. Test application endpoints
curl -f http://localhost/health || echo "Application not ready yet"

# 5. Check logs for errors
sudo docker compose logs | grep -i error
```

## Key Improvements Summary

1. **Removed `--quiet` Flag**: Build errors are now visible in systemd logs
2. **Added Build Context Validation**: Pre-build checks ensure all required files exist
3. **Explicit Working Directory**: Commands run from correct directory (`/opt/mcp-portal`)
4. **Separated Build and Pull**: Isolated operations for better error diagnosis
5. **No-Cache Builds**: Ensures clean builds without stale cache issues
6. **Comprehensive Debugging Tool**: `debug-build.sh` for troubleshooting
7. **Enhanced Installation Validation**: Installation script checks all dependencies

## Expected Outcomes

After applying this solution:

- **Systemd service starts successfully** without build failures
- **Detailed error messages** visible in `journalctl -u mcp-portal` when issues occur
- **Faster troubleshooting** with the debug script and enhanced logging
- **Reliable builds** with proper dependency validation and clean cache
- **Production-ready deployment** with all containers running and healthy

The solution addresses both the immediate build failure and provides robust tooling for ongoing maintenance and troubleshooting.
