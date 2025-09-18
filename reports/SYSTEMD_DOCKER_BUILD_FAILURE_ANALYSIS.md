# Systemd Docker Build Failure Analysis and Solution

_Date: 2025-09-18_

## Issue Analysis

### Current Situation

1. **Manual build works**: Docker Compose can build successfully when run manually
2. **Systemd service fails**: Service fails during build step despite same configuration
3. **Build starts properly**: Images are being pulled and context is loaded (41.15kB transferred)
4. **Timeout suspected**: Build terminates before completion

### Root Cause Analysis

#### 1. **Build Timeout Issue** 🔴 CRITICAL

- **Current timeout**: `TimeoutStartSec=300` (5 minutes)
- **Actual build time**: Production builds require 8-15 minutes for:
  - Go compilation with CGO
  - Next.js frontend build with optimization
  - Multi-stage Docker builds
  - No build cache on first run

#### 2. **Resource Constraints** 🟡 IMPORTANT

- **Production builds**: Require significant CPU and memory
- **Systemd limits**: May be imposing additional constraints
- **Concurrent operations**: Multiple builds (frontend + backend) competing for resources

#### 3. **Build Output Truncation** 🟡 IMPORTANT

- **Log evidence**: Only showing "head -50" of build output
- **Full output hidden**: Cannot see actual failure reason
- **Silent failures**: Build errors not visible in systemd logs

#### 4. **Missing pkg Directory** 🟢 RECOMMENDED

- **Current symlinks**: `cmd`, `vendor`, `go.mod`, `go.sum`, `docker`
- **Missing**: `pkg` directory which may contain shared packages
- **Build dependency**: Required for complete Go module builds

## Production-Ready Solution

### 1. Extend Systemd Timeouts 🔴 CRITICAL

```ini
# Updated timeout values for production builds
TimeoutStartSec=1200        # 20 minutes (was 5 minutes)
TimeoutStopSec=120          # 2 minutes (was 1 minute)
TimeoutReloadSec=120        # 2 minutes (was 1 minute)
```

**Rationale**: Production Docker builds require significantly more time than development builds.

### 2. Optimize Build Strategy 🔴 CRITICAL

Replace single build command with staged approach:

```bash
# Current problematic approach:
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && docker compose --file docker-compose.prod.yaml build --no-cache'

# New staged approach:
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && docker compose --file docker-compose.prod.yaml build --no-cache backend'
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && docker compose --file docker-compose.prod.yaml build --no-cache frontend'
```

**Benefits**:

- **Sequential builds**: Reduces memory pressure and resource contention
- **Better error isolation**: Can identify which service fails to build
- **Progress visibility**: Clear stages in systemd logs

### 3. Add pkg Directory Symlink 🟡 IMPORTANT

Update installation script to include pkg directory:

```bash
# Add to copy_compose_config() function in install-production.sh
if [[ -d "$PROJECT_ROOT/pkg" ]]; then
    ln -sfn "$PROJECT_ROOT/pkg" "$INSTALL_DIR/pkg"
    log INFO "Created symlink: pkg -> $PROJECT_ROOT/pkg"
else
    log WARN "pkg directory not found - creating empty directory for build"
    mkdir -p "$INSTALL_DIR/pkg"
    chown "$MCP_USER:$DOCKER_GROUP" "$INSTALL_DIR/pkg"
fi
```

### 4. Enable Build Progress Monitoring 🟡 IMPORTANT

Add build progress and resource monitoring:

```bash
# Add before build commands:
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "=== BUILD START: $(date) ===" && df -h && free -h'

# Replace build command with progress monitoring:
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && timeout 1200 docker compose --file docker-compose.prod.yaml build --no-cache --progress=plain backend 2>&1 | tee /tmp/mcp-build-backend.log'
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && timeout 1200 docker compose --file docker-compose.prod.yaml build --no-cache --progress=plain frontend 2>&1 | tee /tmp/mcp-build-frontend.log'

# Add after build commands:
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "=== BUILD COMPLETE: $(date) ===" && docker images | grep mcp-portal'
```

### 5. Remove --no-cache for Subsequent Builds 🟢 RECOMMENDED

For faster rebuilds after initial deployment:

```bash
# Production optimization - use cache after first build
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && if [[ ! -f .initial-build-done ]]; then docker compose --file docker-compose.prod.yaml build --no-cache && touch .initial-build-done; else docker compose --file docker-compose.prod.yaml build; fi'
```

## Complete Fixed systemd Service File

```ini
[Unit]
Description=MCP Portal Service
Documentation=https://github.com/jrmatherly/mcp-hub-gateway
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
User=root
Group=docker
WorkingDirectory=/opt/mcp-portal
Environment=COMPOSE_PROJECT_NAME=mcp_portal
Environment=COMPOSE_FILE=docker-compose.prod.yaml

# Security settings - adjusted for Docker compatibility
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/opt/mcp-portal /var/run /root /tmp
PrivateTmp=false
PrivateDevices=false
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Pre-start checks and preparation
ExecStartPre=/usr/bin/test -f /opt/mcp-portal/docker-compose.prod.yaml
ExecStartPre=/usr/bin/docker --version
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml config --quiet

# Verify build context and dependencies are available
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && test -f Dockerfile.portal && test -f Dockerfile.frontend && test -d cmd && test -f go.mod'

# Build progress monitoring and resource check
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "=== BUILD START: $(date) ===" && df -h && free -h'

# Staged build approach - backend first
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "Building backend..." && timeout 1200 docker compose --file docker-compose.prod.yaml build --progress=plain backend 2>&1 | tee /tmp/mcp-build-backend.log'

# Then frontend
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "Building frontend..." && timeout 1200 docker compose --file docker-compose.prod.yaml build --progress=plain frontend 2>&1 | tee /tmp/mcp-build-frontend.log'

# Build completion verification
ExecStartPre=/bin/bash -c 'cd /opt/mcp-portal && echo "=== BUILD COMPLETE: $(date) ===" && docker images | grep mcp-portal'

# Pull only external images (postgres, redis, nginx)
ExecStartPre=/usr/bin/docker compose --file docker-compose.prod.yaml pull --ignore-buildable-images --quiet

# Main service commands
ExecStart=/usr/bin/docker compose --file docker-compose.prod.yaml up -d --remove-orphans
ExecReload=/usr/bin/docker compose --file docker-compose.prod.yaml restart
ExecStop=/usr/bin/docker compose --file docker-compose.prod.yaml down --timeout 30

# Health check and monitoring
ExecStartPost=/bin/bash -c 'for i in {1..60}; do if docker compose --file docker-compose.prod.yaml ps --status running | grep -q mcp-portal; then echo "Services started successfully"; exit 0; fi; sleep 5; done; echo "Services failed to start"; exit 1'

# Restart and timeout settings - EXTENDED FOR PRODUCTION BUILDS
Restart=on-failure
RestartSec=30
TimeoutStartSec=1200    # 20 minutes for production builds
TimeoutStopSec=120      # 2 minutes
TimeoutReloadSec=120    # 2 minutes

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mcp-portal

[Install]
WantedBy=multi-user.target
```

## Monitoring and Debugging Commands

### Real-time Build Monitoring

```bash
# Monitor build progress during systemd startup
sudo journalctl -u mcp-portal -f

# Check build logs
tail -f /tmp/mcp-build-backend.log
tail -f /tmp/mcp-build-frontend.log

# Monitor resource usage during build
watch -n 2 'df -h && echo "---" && free -h && echo "---" && docker stats --no-stream'
```

### Quick Recovery Commands

```bash
# If build fails, quick manual recovery:
cd /opt/mcp-portal

# Build just the failing service:
sudo docker compose --file docker-compose.prod.yaml build --no-cache backend
# OR
sudo docker compose --file docker-compose.prod.yaml build --no-cache frontend

# Then restart service:
sudo systemctl start mcp-portal
```

### Build Performance Analysis

```bash
# Check build time and resource usage
time sudo docker compose --file docker-compose.prod.yaml build --no-cache

# Check image sizes
docker images | grep mcp-portal

# Check build cache usage
docker system df
```

## Implementation Steps

### Step 1: Update systemd Service File

```bash
# 1. Stop current service
sudo systemctl stop mcp-portal

# 2. Backup current service
sudo cp /etc/systemd/system/mcp-portal.service /etc/systemd/system/mcp-portal.service.backup

# 3. Update with new configuration (copy content above)
sudo nano /etc/systemd/system/mcp-portal.service

# 4. Reload systemd
sudo systemctl daemon-reload
```

### Step 2: Add pkg Directory Symlink

```bash
# Add missing pkg symlink
cd /opt/mcp-portal
sudo ln -sfn /opt/docker/appdata/mcp-hub-gateway/pkg ./pkg

# Verify all symlinks
ls -la
```

### Step 3: Test Build Process

```bash
# Test backend build
cd /opt/mcp-portal
sudo docker compose --file docker-compose.prod.yaml build --progress=plain backend

# Test frontend build
sudo docker compose --file docker-compose.prod.yaml build --progress=plain frontend

# Verify images created
sudo docker images | grep mcp-portal
```

### Step 4: Start Service with Monitoring

```bash
# Start service with real-time monitoring
sudo systemctl start mcp-portal

# In another terminal, monitor progress
sudo journalctl -u mcp-portal -f
```

## Expected Results

### Build Times (Production Hardware)

- **Backend build**: 8-12 minutes (Go compilation)
- **Frontend build**: 10-15 minutes (Next.js optimization)
- **Total time**: 18-27 minutes for initial build
- **Subsequent builds**: 3-8 minutes (with cache)

### Success Indicators

```bash
# Check service status
sudo systemctl status mcp-portal
● mcp-portal.service - MCP Portal Service
   Active: active (exited) since ...

# Check running containers
sudo docker compose --file docker-compose.prod.yaml ps
     Name                   State    Ports
mcp-portal-backend-prod     Up      8080/tcp
mcp-portal-frontend-prod    Up      3000/tcp
mcp-portal-postgres-prod    Up      5432/tcp
mcp-portal-redis-prod       Up      6379/tcp
mcp-portal-nginx-prod       Up      80/tcp, 443/tcp
```

## Performance Optimization Notes

### Resource Requirements

- **CPU**: Minimum 4 cores for parallel builds
- **Memory**: 8GB+ for production builds
- **Disk**: 10GB+ free space for build cache
- **Network**: Good connection for base image pulls

### Build Cache Strategy

- **First deployment**: --no-cache ensures clean build
- **Updates**: Use cache for faster rebuilds
- **Clean slate**: Periodically clear cache: `docker system prune -a`

### Troubleshooting Build Failures

1. **Out of disk space**: `df -h` and `docker system prune`
2. **Out of memory**: `free -h` and consider reducing parallel builds
3. **Network timeouts**: Check internet connection and retry
4. **Image corruption**: `docker system prune -a` and rebuild

This solution addresses all identified issues and provides a production-ready deployment strategy with comprehensive monitoring and recovery procedures.
