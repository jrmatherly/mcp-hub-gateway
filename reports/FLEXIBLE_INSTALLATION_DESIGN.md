# Flexible Installation Path Design

_Date: 2025-09-18_

## Problem Analysis

The current installation system is rigid and fails when:

- Repository is cloned to any location other than expected paths
- Symlinks break Docker build context
- Hardcoded `/opt/mcp-portal` path assumptions throughout system
- Complex file copying and linking operations

## Recommended Solution: In-Place Installation

### Core Principle: "Run Where You Clone"

Eliminate separate installation directory. Configure all services to run from the actual repository location.

### Architecture Changes

#### Before (Rigid)

```
Clone Location: /opt/docker/appdata/mcp-hub-gateway/
      ↓ (copy + symlink)
Install Location: /opt/mcp-portal/
      ↓ (hardcoded paths)
Services: Expect /opt/mcp-portal/
```

#### After (Flexible)

```
Clone Location: ANY_PATH/mcp-hub-gateway/
      ↓ (auto-detect)
Install Location: SAME_AS_CLONE
      ↓ (dynamic paths)
Services: Use detected path
```

## Implementation Strategy

### 1. Dynamic Path Detection

**Installation Script Changes:**

```bash
#!/bin/bash
# Flexible installation script

# DYNAMIC PATH DETECTION
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"  # Navigate to repo root
INSTALL_DIR="$PROJECT_ROOT"  # Install in-place

# REMOVE HARDCODED PATH
# OLD: INSTALL_DIR="/opt/mcp-portal"
# NEW: Use actual repository location

log INFO "Repository location: $PROJECT_ROOT"
log INFO "Installing in-place at: $INSTALL_DIR"
```

### 2. Dynamic systemd Service

**Template-Based Service File:**

```bash
# Generate service file with actual paths
generate_systemd_service() {
    local working_dir="$1"

    cat > /etc/systemd/system/mcp-portal.service << EOF
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
WorkingDirectory=$working_dir
Environment=COMPOSE_PROJECT_NAME=mcp_portal
Environment=COMPOSE_FILE=docker-compose.yaml

# Security settings
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=$working_dir /var/run /root
PrivateTmp=false
PrivateDevices=false
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Pre-start checks
ExecStartPre=/usr/bin/test -f $working_dir/docker-compose.yaml
ExecStartPre=/usr/bin/docker --version
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml config --quiet

# Build and start
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml build
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml pull --ignore-buildable-images
ExecStart=/usr/bin/docker compose --file docker-compose.yaml up -d --remove-orphans

# Management commands
ExecReload=/usr/bin/docker compose --file docker-compose.yaml restart
ExecStop=/usr/bin/docker compose --file docker-compose.yaml down --timeout 30

# Health check
ExecStartPost=/bin/bash -c 'for i in {1..30}; do if docker compose ps --status running | grep -q mcp-portal; then exit 0; fi; sleep 2; done; exit 1'

# Restart settings
Restart=on-failure
RestartSec=10
TimeoutStartSec=300
TimeoutStopSec=60

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mcp-portal

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 /etc/systemd/system/mcp-portal.service
    systemctl daemon-reload
}
```

### 3. Docker Compose Path Flexibility

**Environment Variable Injection:**

```bash
# Create .env with dynamic paths
setup_environment() {
    local project_root="$1"

    cat > "$project_root/.env" << EOF
# Auto-generated environment configuration
# Installation Location: $project_root
# Generated: $(date)

# Project paths
PROJECT_ROOT=$project_root
INSTALL_DIR=$project_root

# Service configuration
MCP_PORTAL_SERVER_PORT=8080
COMPOSE_PROJECT_NAME=mcp_portal

# Docker configuration
DOCKER_SOCKET_PATH=/var/run/docker.sock

# Database configuration (defaults)
POSTGRES_DB=mcp_portal
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-in-production

# Portal database user
MCP_PORTAL_DATABASE_USERNAME=portal
MCP_PORTAL_DATABASE_PASSWORD=change-in-production

# Redis configuration
REDIS_PASSWORD=

# JWT configuration (MUST be changed)
JWT_SECRET=CHANGE-THIS-TO-A-SECURE-32-CHARACTER-SECRET

# Azure AD configuration (MUST be configured)
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# Session configuration
SESSION_COOKIE_NAME=mcp-portal-session
SESSION_COOKIE_SECURE=false
SESSION_COOKIE_HTTPONLY=true
SESSION_COOKIE_SAMESITE=lax

# URLs for Docker networking
DATABASE_URL=postgresql://portal:change-in-production@postgres:5432/mcp_portal
REDIS_URL=redis://redis:6379

# API URL for frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
EOF

    log INFO "Environment file created at $project_root/.env"
    log WARN "IMPORTANT: Edit $project_root/.env with your production values"
}
```

### 4. User and Permission Management

**Flexible User Setup:**

```bash
# Create user with dynamic home directory
create_mcp_user() {
    local install_dir="$1"
    local mcp_user="mcp-portal"

    # Create user with install directory as home
    if ! id "$mcp_user" > /dev/null 2>&1; then
        useradd -r -g docker -d "$install_dir" -s /bin/false "$mcp_user"
        log INFO "Created user: $mcp_user with home: $install_dir"
    else
        # Update existing user's home directory
        usermod -d "$install_dir" "$mcp_user"
        usermod -aG docker "$mcp_user"
        log INFO "Updated user: $mcp_user home to: $install_dir"
    fi

    # Set ownership of repository
    chown -R "$mcp_user:docker" "$install_dir"

    # Ensure proper permissions
    chmod 755 "$install_dir"
    find "$install_dir" -type d -exec chmod 755 {} \;
    find "$install_dir" -type f -exec chmod 644 {} \;

    # Make scripts executable
    find "$install_dir/docker/scripts" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null || true
}
```

## 5. Complete Flexible Installation Script

**New install-flexible.sh:**

```bash
#!/bin/bash
#
# MCP Portal Flexible Installation Script
#
# This script installs MCP Portal from ANY repository location
# No hardcoded paths, no symlinks, no file copying required
#
# Usage: sudo ./install-flexible.sh [OPTIONS]
#

set -euo pipefail

# DYNAMIC PATH DETECTION
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# INSTALL IN-PLACE - NO SEPARATE DIRECTORY
INSTALL_DIR="$PROJECT_ROOT"

# User configuration
MCP_USER="mcp-portal"
DOCKER_GROUP="docker"
LOG_FILE="/var/log/mcp-portal-install.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    case "$level" in
        INFO)  echo -e "${GREEN}[INFO]${NC} $message" ;;
        WARN)  echo -e "${YELLOW}[WARN]${NC} $message" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} $message" ;;
        DEBUG) echo -e "${BLUE}[DEBUG]${NC} $message" ;;
    esac

    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

error_exit() {
    log ERROR "$1"
    exit 1
}

# Main installation function
main() {
    log INFO "Starting MCP Portal flexible installation..."
    log INFO "Repository root: $PROJECT_ROOT"
    log INFO "Installing in-place at: $INSTALL_DIR"

    # Verify we're in a valid repository
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]] || [[ ! -f "$PROJECT_ROOT/Dockerfile.portal" ]]; then
        error_exit "Invalid repository: $PROJECT_ROOT does not contain required files"
    fi

    # Check root permissions
    if [[ $EUID -ne 0 ]]; then
        error_exit "This script must be run as root. Use: sudo $0"
    fi

    # Installation steps
    detect_distro
    install_docker_if_needed
    create_mcp_user "$INSTALL_DIR"
    setup_docker_permissions
    setup_environment "$INSTALL_DIR"
    generate_systemd_service "$INSTALL_DIR"
    setup_log_rotation "$INSTALL_DIR"

    # Start services
    systemctl enable mcp-portal.service
    log INFO "Service enabled. Start with: systemctl start mcp-portal"

    print_completion_message
}

# Detect Linux distribution
detect_distro() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        DISTRO="$ID"
        VERSION="$VERSION_ID"
        log INFO "Detected distribution: $DISTRO $VERSION"
    else
        error_exit "Cannot detect Linux distribution"
    fi
}

# Install Docker if not present
install_docker_if_needed() {
    if command -v docker >/dev/null 2>&1; then
        log INFO "Docker already installed: $(docker --version)"
        return 0
    fi

    log INFO "Installing Docker Engine..."
    # ... (existing Docker installation logic)
}

# Create MCP user with flexible home directory
create_mcp_user() {
    local install_dir="$1"

    log INFO "Creating MCP Portal user..."

    # Create docker group if needed
    if ! getent group "$DOCKER_GROUP" > /dev/null; then
        groupadd "$DOCKER_GROUP"
    fi

    # Create or update MCP user
    if ! id "$MCP_USER" > /dev/null 2>&1; then
        useradd -r -g "$DOCKER_GROUP" -d "$install_dir" -s /bin/false "$MCP_USER"
        log INFO "Created user: $MCP_USER"
    else
        usermod -d "$install_dir" "$MCP_USER"
        usermod -aG "$DOCKER_GROUP" "$MCP_USER"
        log INFO "Updated user: $MCP_USER"
    fi

    # Set repository ownership
    chown -R "$MCP_USER:$DOCKER_GROUP" "$install_dir"

    # Set proper permissions
    chmod 755 "$install_dir"
    find "$install_dir" -type d -exec chmod 755 {} \; 2>/dev/null || true
    find "$install_dir" -type f -exec chmod 644 {} \; 2>/dev/null || true

    # Make scripts executable
    find "$install_dir/docker/scripts" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null || true
}

# Setup Docker socket permissions
setup_docker_permissions() {
    log INFO "Setting up Docker permissions..."

    # Ensure docker group can access socket
    if [[ -S /var/run/docker.sock ]]; then
        chown root:docker /var/run/docker.sock
        chmod 660 /var/run/docker.sock
    fi

    # Create Docker config directory for root
    mkdir -p /root/.docker
    chown root:root /root/.docker
    chmod 700 /root/.docker
}

# Setup environment with dynamic paths
setup_environment() {
    local project_root="$1"

    log INFO "Setting up environment configuration..."

    # Copy example if no .env exists
    if [[ ! -f "$project_root/.env" ]]; then
        if [[ -f "$project_root/.env.example" ]]; then
            cp "$project_root/.env.example" "$project_root/.env"
            log INFO "Copied .env.example to .env"
        else
            # Create minimal .env
            cat > "$project_root/.env" << 'EOF'
# MCP Portal Configuration
# Generated by flexible installation

# REQUIRED: Change these values for production
JWT_SECRET=CHANGE-THIS-TO-A-SECURE-32-CHARACTER-SECRET
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# Database configuration
POSTGRES_DB=mcp_portal
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-in-production
MCP_PORTAL_DATABASE_USERNAME=portal
MCP_PORTAL_DATABASE_PASSWORD=change-in-production

# Service configuration
MCP_PORTAL_SERVER_PORT=8080
COMPOSE_PROJECT_NAME=mcp_portal

# URLs
DATABASE_URL=postgresql://portal:change-in-production@postgres:5432/mcp_portal
REDIS_URL=redis://redis:6379
NEXT_PUBLIC_API_URL=http://localhost:8080
EOF
            log INFO "Created basic .env file"
        fi

        chown "$MCP_USER:$DOCKER_GROUP" "$project_root/.env"
        chmod 600 "$project_root/.env"
        log WARN "CRITICAL: Edit $project_root/.env with your production values"
    else
        log INFO "Using existing .env file"
    fi
}

# Generate systemd service with dynamic working directory
generate_systemd_service() {
    local working_dir="$1"

    log INFO "Creating systemd service for working directory: $working_dir"

    cat > /etc/systemd/system/mcp-portal.service << EOF
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
WorkingDirectory=$working_dir
Environment=COMPOSE_PROJECT_NAME=mcp_portal
Environment=COMPOSE_FILE=docker-compose.yaml

# Security settings
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=$working_dir /var/run /root
PrivateTmp=false
PrivateDevices=false
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Pre-start checks
ExecStartPre=/usr/bin/test -f $working_dir/docker-compose.yaml
ExecStartPre=/usr/bin/docker --version
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml config --quiet

# Build and start services
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml build
ExecStartPre=/usr/bin/docker compose --file docker-compose.yaml pull --ignore-buildable-images
ExecStart=/usr/bin/docker compose --file docker-compose.yaml up -d --remove-orphans

# Management commands
ExecReload=/usr/bin/docker compose --file docker-compose.yaml restart
ExecStop=/usr/bin/docker compose --file docker-compose.yaml down --timeout 30

# Health check
ExecStartPost=/bin/bash -c 'for i in {1..30}; do if docker compose ps --status running | grep -q mcp-portal; then exit 0; fi; sleep 2; done; exit 1'

# Restart settings
Restart=on-failure
RestartSec=10
TimeoutStartSec=300
TimeoutStopSec=60

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mcp-portal

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 /etc/systemd/system/mcp-portal.service
    systemctl daemon-reload
    log INFO "Systemd service created and registered"
}

# Setup log rotation with dynamic paths
setup_log_rotation() {
    local install_dir="$1"

    log INFO "Setting up log rotation..."

    cat > /etc/logrotate.d/mcp-portal << EOF
/var/log/mcp-portal/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 mcp-portal docker
    postrotate
        systemctl reload mcp-portal || true
    endscript
}

$install_dir/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 mcp-portal docker
    postrotate
        if [ -f /var/run/mcp-portal.pid ]; then
            kill -USR1 \$(cat /var/run/mcp-portal.pid)
        fi
    endscript
}
EOF

    chmod 644 /etc/logrotate.d/mcp-portal

    # Create log directories
    mkdir -p /var/log/mcp-portal "$install_dir/logs"
    chown "$MCP_USER:$DOCKER_GROUP" /var/log/mcp-portal "$install_dir/logs"
    chmod 755 /var/log/mcp-portal "$install_dir/logs"
}

# Print completion message
print_completion_message() {
    cat << EOF

${GREEN}========================================${NC}
${GREEN}  MCP Portal Installation Complete!${NC}
${GREEN}========================================${NC}

Installation Type: ${BLUE}In-Place (Flexible)${NC}
Repository Location: ${BLUE}$PROJECT_ROOT${NC}
Service Name: ${BLUE}mcp-portal.service${NC}

${YELLOW}Next Steps:${NC}
1. ${RED}CRITICAL:${NC} Edit ${BLUE}$PROJECT_ROOT/.env${NC} with your production values
2. Configure Azure AD credentials and JWT secret
3. Start the service: ${BLUE}sudo systemctl start mcp-portal${NC}

${YELLOW}Service Management:${NC}
Start:    ${BLUE}sudo systemctl start mcp-portal${NC}
Stop:     ${BLUE}sudo systemctl stop mcp-portal${NC}
Status:   ${BLUE}sudo systemctl status mcp-portal${NC}
Logs:     ${BLUE}sudo journalctl -u mcp-portal -f${NC}

${YELLOW}Docker Commands (from repository):${NC}
View:     ${BLUE}cd $PROJECT_ROOT && docker compose ps${NC}
Logs:     ${BLUE}cd $PROJECT_ROOT && docker compose logs -f${NC}
Update:   ${BLUE}cd $PROJECT_ROOT && git pull && sudo systemctl restart mcp-portal${NC}

${YELLOW}Advantages of In-Place Installation:${NC}
✅ Works from any clone location
✅ No symlinks or file copying
✅ Easier development and updates
✅ Direct Docker build context
✅ Single source of truth

For troubleshooting: ${BLUE}$LOG_FILE${NC}

EOF
}

# Handle interruption
trap 'error_exit "Installation interrupted"' INT TERM

# Initialize logging
touch "$LOG_FILE"
chmod 644 "$LOG_FILE"

# Run main installation
main "$@"
```

## Benefits of In-Place Installation

### 1. **Path Flexibility**

- Works from ANY clone location
- No hardcoded paths
- Dynamic path detection

### 2. **Simplified Architecture**

- No separate installation directory
- No file copying or symlinks
- Direct Docker build context

### 3. **Development Friendly**

- `git pull && systemctl restart` for updates
- Changes immediately available
- No sync issues between locations

### 4. **Reduced Complexity**

- Fewer moving parts
- Easier troubleshooting
- Less disk space usage

### 5. **Production Ready**

- Same architecture for dev and prod
- Clear ownership and permissions
- Secure systemd configuration

## Migration Path for Existing Installations

### Option 1: Clean Migration

```bash
# 1. Stop current service
sudo systemctl stop mcp-portal
sudo systemctl disable mcp-portal

# 2. Backup current configuration
sudo cp /opt/mcp-portal/.env /tmp/mcp-portal-env-backup

# 3. Remove old installation
sudo rm -rf /opt/mcp-portal
sudo rm /etc/systemd/system/mcp-portal.service

# 4. Run flexible installation from repository
cd /opt/docker/appdata/mcp-hub-gateway
sudo ./docker/production/install-flexible.sh

# 5. Restore configuration
sudo cp /tmp/mcp-portal-env-backup /opt/docker/appdata/mcp-hub-gateway/.env

# 6. Start new service
sudo systemctl start mcp-portal
```

### Option 2: In-Place Conversion

```bash
# Convert existing installation to in-place
#!/bin/bash

SOURCE_DIR="/opt/docker/appdata/mcp-hub-gateway"
CURRENT_INSTALL="/opt/mcp-portal"

# Stop service
sudo systemctl stop mcp-portal

# Backup environment
sudo cp "$CURRENT_INSTALL/.env" "$SOURCE_DIR/.env"

# Remove symlinks and copied files
sudo rm -f "$CURRENT_INSTALL"/{cmd,pkg,go.mod,go.sum,docker}
sudo rm -f "$CURRENT_INSTALL"/{Dockerfile.*,.dockerignore,Makefile}

# Generate new service pointing to source
sudo "$SOURCE_DIR/docker/production/install-flexible.sh" --config-only

# Start service
sudo systemctl start mcp-portal
```

## Docker Compose Adjustments

**No changes needed** - Docker Compose already uses relative paths and `.` build context, which will work from any location.

## Conclusion

The in-place installation approach eliminates all path rigidity issues:

1. **Works from any clone location**
2. **No symlinks or file copying**
3. **Simpler Docker build context**
4. **Dynamic systemd service generation**
5. **Development-friendly workflow**
6. **Production-ready security**

This solution transforms the installation from rigid and fragile to flexible and robust.
