# Installation Script Path Fixes

*Date: 2025-01-20*

## Issue Identified

When running `/docker/production/install-production.sh`, the script failed with:
```
cp: cannot stat '/opt/docker/appdata/mcp-hub-gateway/docker/production/systemd/mcp-portal.service': No such file or directory
```

## Root Cause Analysis

The installation script had two path-related issues:

### Issue 1: Incorrect systemd service path
**Location**: Line 290
- **Expected**: `$SCRIPT_DIR/systemd/mcp-portal.service`
- **Actual**: `$SCRIPT_DIR/mcp-portal.service` (no `systemd/` subdirectory)

### Issue 2: Incorrect PROJECT_ROOT calculation
**Location**: Line 20
- **Expected**: Repository root (where `docker-compose.prod.yaml` exists)
- **Calculated**: One directory up from script (in `docker/` directory)
- **Required**: Two directories up from script (repository root)

## Path Structure

```
mcp-gateway/                    # <-- PROJECT_ROOT (needed)
├── docker-compose.prod.yaml    # <-- File to copy
├── docker/                     # <-- Previously calculated as PROJECT_ROOT (wrong)
│   └── production/
│       ├── install-production.sh    # <-- SCRIPT_DIR
│       ├── mcp-portal.service       # <-- Service file (no systemd/ subdir)
│       └── daemon.json
```

## Fixes Applied

### Fix 1: Remove incorrect systemd/ subdirectory
```bash
# Before (Line 290):
cp "$SCRIPT_DIR/systemd/mcp-portal.service" /etc/systemd/system/

# After:
cp "$SCRIPT_DIR/mcp-portal.service" /etc/systemd/system/
```

### Fix 2: Correct PROJECT_ROOT calculation
```bash
# Before (Line 20):
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# After:
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
```

## Verification

After these fixes, the script will correctly:
1. Find `mcp-portal.service` at `$SCRIPT_DIR/mcp-portal.service`
2. Find `docker-compose.prod.yaml` at `$PROJECT_ROOT/docker-compose.prod.yaml`
3. Find `daemon.json` at `$SCRIPT_DIR/daemon.json` (already correct)

## Testing Recommendations

To test the fixes:
```bash
# Run with dry-run first
sudo ./docker/production/install-production.sh --config-only

# Verify paths are logged correctly
# Look for:
# [INFO] Script directory: /path/to/docker/production
# [INFO] Project root: /path/to/mcp-gateway

# If paths look correct, run full installation
sudo ./docker/production/install-production.sh
```

## Impact

These fixes ensure the production installation script can:
- Successfully copy all required configuration files
- Install the systemd service properly
- Deploy the production Docker Compose configuration

The script is now ready for production deployment on systems without Docker Desktop.