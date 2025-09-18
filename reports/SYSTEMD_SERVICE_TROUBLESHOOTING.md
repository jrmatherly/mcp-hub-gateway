# MCP Portal Systemd Service Troubleshooting Guide

*Date: 2025-01-20*

## Issue: Service Fails to Start

Error message:
```
Job for mcp-portal.service failed because the control process exited with error code.
```

## Root Causes Identified & Fixed

### 1. Missing Environment File (.env)
**Problem**: `docker-compose.prod.yaml` requires `.env` file but installation script didn't copy it.

**Fix Applied**: Updated `install-production.sh` to:
- Copy `.env.example` to `/opt/mcp-portal/.env` as template
- Set proper permissions (600) for security
- Display warning to edit with production values

### 2. Overly Restrictive Systemd Security Settings
**Problem**: Security settings were too strict for Docker operations.

**Original Settings** (Too Restrictive):
```ini
NoNewPrivileges=true
ProtectSystem=strict
PrivateTmp=true
PrivateDevices=true
ReadWritePaths=/opt/mcp-portal
```

**Fixed Settings** (Docker Compatible):
```ini
NoNewPrivileges=false         # Docker needs to create privileged containers
ProtectSystem=full           # Less restrictive than 'strict'
PrivateTmp=false             # Docker needs access to /tmp
PrivateDevices=false         # Docker needs device access
ReadWritePaths=/opt/mcp-portal /var/run  # Added /var/run for docker.sock
```

### 3. Path Issues (Previously Fixed)
- Service file location: Fixed from `systemd/mcp-portal.service` to `mcp-portal.service`
- Project root calculation: Fixed to go up two directories instead of one

## Troubleshooting Steps

### 1. Check Service Status
```bash
# View service status
sudo systemctl status mcp-portal.service

# View detailed logs
sudo journalctl -xeu mcp-portal.service

# Follow logs in real-time
sudo journalctl -fu mcp-portal.service
```

### 2. Common Issues & Solutions

#### Issue: "docker-compose.prod.yaml: no such file"
**Solution**:
```bash
# Check if file was copied
ls -la /opt/mcp-portal/

# If missing, copy manually
sudo cp /path/to/mcp-gateway/docker-compose.prod.yaml /opt/mcp-portal/
sudo chown root:docker /opt/mcp-portal/docker-compose.prod.yaml
```

#### Issue: ".env: no such file"
**Solution**:
```bash
# Copy and configure
sudo cp /path/to/mcp-gateway/.env.example /opt/mcp-portal/.env
sudo chmod 600 /opt/mcp-portal/.env
sudo nano /opt/mcp-portal/.env  # Edit with production values
```

#### Issue: "permission denied" for docker.sock
**Solution**:
```bash
# Fix Docker socket permissions
sudo chmod 666 /var/run/docker.sock
# Or add user to docker group
sudo usermod -aG docker $USER
```

#### Issue: "docker compose: command not found"
**Solution**:
```bash
# Install Docker Compose plugin
sudo apt-get update
sudo apt-get install docker-compose-plugin
# Or for RHEL/CentOS
sudo dnf install docker-compose-plugin
```

### 3. Manual Service Testing

Before starting via systemd, test manually:

```bash
# Change to installation directory
cd /opt/mcp-portal

# Test docker-compose configuration
sudo docker compose --file docker-compose.prod.yaml config

# Try starting manually
sudo docker compose --file docker-compose.prod.yaml up

# If it works, stop with Ctrl+C and use systemd
sudo systemctl start mcp-portal
```

### 4. Reset and Reinstall

If issues persist, reset and reinstall:

```bash
# Stop and disable service
sudo systemctl stop mcp-portal
sudo systemctl disable mcp-portal

# Remove service files
sudo rm /etc/systemd/system/mcp-portal.service
sudo systemctl daemon-reload

# Clean installation directory
sudo rm -rf /opt/mcp-portal

# Re-run installation with fixes
cd /path/to/mcp-gateway/docker/production
sudo ./install-production.sh
```

## Configuration Checklist

Before starting the service, ensure:

- [ ] `.env` file exists at `/opt/mcp-portal/.env`
- [ ] `.env` contains all required variables (check `.env.example`)
- [ ] `docker-compose.prod.yaml` exists at `/opt/mcp-portal/`
- [ ] Docker daemon is running: `sudo systemctl status docker`
- [ ] User has docker group membership: `groups`
- [ ] Docker socket has correct permissions: `ls -la /var/run/docker.sock`

## Required Environment Variables

Critical variables that MUST be set in `.env`:

```bash
# Azure AD (REQUIRED)
AZURE_TENANT_ID=your-actual-tenant-id
AZURE_CLIENT_ID=your-actual-client-id
AZURE_CLIENT_SECRET=your-actual-secret

# JWT Secret (REQUIRED - generate with: openssl rand -base64 64)
JWT_SECRET=your-generated-jwt-secret

# Database Password (REQUIRED - change from default)
POSTGRES_PASSWORD=strong-password-here

# API Configuration
API_PORT=8080
NEXT_PUBLIC_API_URL=http://your-domain:8080
```

## Verification After Fixes

```bash
# Reload systemd after service file changes
sudo systemctl daemon-reload

# Start the service
sudo systemctl start mcp-portal

# Check if running
sudo systemctl status mcp-portal

# View running containers
cd /opt/mcp-portal && sudo docker compose ps

# Check application health
curl http://localhost:8080/api/health
```

## Summary of All Fixes

1. ✅ Fixed systemd service file path reference
2. ✅ Fixed PROJECT_ROOT calculation in installation script
3. ✅ Added .env file copying to installation script
4. ✅ Relaxed systemd security settings for Docker compatibility
5. ✅ Added /var/run to ReadWritePaths for docker.sock access

The service should now start successfully after these fixes are applied.