# Docker Compose Volume Conflict Fix

*Date: 2025-09-18*

## Issue Identified

The systemd service was failing with error:
```
services.frontend.volumes[0]: target /var/cache/nextjs already mounted as services.frontend.tmpfs[2]
```

## Root Cause

The `docker-compose.prod.yaml` file had a **duplicate mount conflict** for `/var/cache/nextjs`:

1. **Line 112**: Mounted as tmpfs (in-memory filesystem)
   ```yaml
   tmpfs:
     - /var/cache/nextjs:noexec,nosuid,size=500M
   ```

2. **Line 134**: Mounted as a volume
   ```yaml
   volumes:
     - frontend-cache:/var/cache/nextjs
   ```

Docker Compose cannot mount the same path twice, causing the validation to fail.

## Fixes Applied

### 1. Removed Duplicate Volume Mount
**File**: `docker-compose.prod.yaml`
- Removed the `volumes` section from the frontend service (line 133-134)
- Added comment explaining that tmpfs mount is already in use

### 2. Removed Unused Volume Definition
**File**: `docker-compose.prod.yaml`
- Removed the `frontend-cache` volume definition (lines 465-471)
- This volume is no longer needed since we're using tmpfs

### 3. Removed Obsolete Version Attribute
**File**: `docker-compose.prod.yaml`
- Removed `version: "3.8"` (line 26)
- Modern Docker Compose doesn't require this and warns it's obsolete

## Benefits of the Fix

1. **Resolves Conflict**: Service can now start without mount conflicts
2. **Better Performance**: tmpfs (RAM-based) is faster than disk volumes for cache
3. **Cleaner Configuration**: Removed obsolete and duplicate configurations
4. **Security**: tmpfs with `noexec,nosuid` flags is more secure for cache

## How to Apply the Fix

For users who have already copied the file to `/opt/mcp-portal/`:

```bash
# Stop the service
sudo systemctl stop mcp-portal

# Copy the fixed file
sudo cp /path/to/mcp-gateway/docker-compose.prod.yaml /opt/mcp-portal/

# Test the configuration
cd /opt/mcp-portal
sudo docker compose --file docker-compose.prod.yaml config

# If no errors, start the service
sudo systemctl start mcp-portal
sudo systemctl status mcp-portal
```

## Verification

After applying the fix, you should see:
- No "already mounted" errors
- No "version is obsolete" warnings
- Service starts successfully

## Summary

The issue was caused by conflicting mount definitions for the frontend cache directory. By using only the tmpfs mount (which is better for performance anyway), we resolved the conflict and simplified the configuration.