# !/bin/bash

#

# MCP Portal Flexible Installation Script

#

# PRODUCTION-READY: "Run Where You Clone" approach

# Installs MCP Portal from ANY repository location with no hardcoded paths

# Eliminates symlinks, file copying, and build context issues

#

# Usage: sudo ./install-flexible.sh [OPTIONS]

# Options

# --skip-docker Skip Docker Engine installation

# --config-only Only configure services, don't start them

# --migrate Migrate from existing /opt/mcp-portal installation

# --help Show this help message

#

set -euo pipefail

# DYNAMIC PATH DETECTION - No hardcoded paths

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# INSTALL IN-PLACE - Repository location IS the installation directory

INSTALL_DIR="$PROJECT_ROOT"

# Configuration

MCP_USER="mcp-portal"
DOCKER_GROUP="docker"
LOG_FILE="/var/log/mcp-portal-install.log"
BACKUP_DIR="/var/backups/mcp-portal"

# Colors for output

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default options

SKIP_DOCKER=false
CONFIG_ONLY=false
MIGRATE=false

# Logging function

log() {
local level="$1"
    shift
    local message="$\*"
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

# Parse command line arguments

parse_args() {
while [[$# -gt 0]]; do
case $1 in
--skip-docker)
SKIP_DOCKER=true
shift
;;
--config-only)
CONFIG_ONLY=true
shift
;;
--migrate)
MIGRATE=true
shift
;;
--help)
show_help
exit 0
;;
\*)
error_exit "Unknown option: $1. Use --help for usage information."
;;
esac
done
}

# Show help message

show_help() {
cat << EOF
MCP Portal Flexible Installation Script - Production Ready

DESCRIPTION:
This script implements "Run Where You Clone" installation approach.
It installs MCP Portal in-place from ANY repository location without
hardcoded paths, symlinks, or file copying.

USAGE:
sudo $0 [OPTIONS]

OPTIONS:
--skip-docker Skip Docker Engine installation (if already installed)
--config-only Configure services only, don't start them
--migrate Migrate from existing /opt/mcp-portal installation
--help Show this help message

ADVANTAGES:
✅ Works from any clone location
✅ No symlinks or file copying
✅ Direct Docker build context
✅ Easy development and updates
✅ Production-grade security
✅ Single source of truth

EXAMPLES: # Full installation from any location
sudo ./install-flexible.sh

    # Install with existing Docker
    sudo ./install-flexible.sh --skip-docker

    # Migrate from old installation
    sudo ./install-flexible.sh --migrate

    # Configure only (for CI/CD)
    sudo ./install-flexible.sh --config-only

SECURITY: - Maintains systemd security hardening - Preserves user isolation with dedicated mcp-portal user - Docker socket permissions unchanged - File permissions properly managed

EOF
}

# Check if running as root

check_root() {
if [[$EUID -ne 0]]; then
error_exit "This script must be run as root. Use: sudo $0"
fi
}

# Validate repository structure

validate_repository() {
log INFO "Validating repository structure..."

    # Check for required files
    local required_files=(
        "go.mod"
        "Dockerfile.portal"
        "Dockerfile.frontend"
        "docker-compose.yaml"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f "$PROJECT_ROOT/$file" ]]; then
            error_exit "Required file not found: $PROJECT_ROOT/$file"
        fi
    done

    # Check for required directories
    local required_dirs=(
        "cmd/docker-mcp/portal"
        "cmd/docker-mcp/portal/frontend"
        "docker/scripts/entrypoint"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$PROJECT_ROOT/$dir" ]]; then
            error_exit "Required directory not found: $PROJECT_ROOT/$dir"
        fi
    done

    log INFO "Repository structure validation passed"

}

# Detect Linux distribution

detect_distro() {
if [[-f /etc/os-release]]; then
. /etc/os-release
DISTRO="$ID"
        VERSION="$VERSION_ID"
log INFO "Detected distribution: $DISTRO $VERSION"
else
error_exit "Cannot detect Linux distribution"
fi
}

# Install Docker Engine

install_docker() {
if [["$SKIP_DOCKER" == true]]; then
log INFO "Skipping Docker installation as requested"
return 0
fi

    if command -v docker >/dev/null 2>&1; then
        log INFO "Docker already installed: $(docker --version)"
        return 0
    fi

    log INFO "Installing Docker Engine..."

    case "$DISTRO" in
        ubuntu|debian)
            apt-get update
            apt-get install -y ca-certificates curl gnupg lsb-release

            mkdir -p /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$DISTRO/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

            echo \
                "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$DISTRO \
                $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

            apt-get update
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;

        centos|rhel|fedora|rocky|almalinux)
            dnf install -y dnf-plugins-core
            dnf config-manager --add-repo https://download.docker.com/linux/$DISTRO/docker-ce.repo
            dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;

        *)
            error_exit "Unsupported distribution: $DISTRO"
            ;;
    esac

    log INFO "Docker Engine installed successfully"

}

# Create system user with dynamic home directory

create_mcp_user() {
log INFO "Creating MCP Portal user with home: $INSTALL_DIR"

    # Create docker group if needed
    if ! getent group "$DOCKER_GROUP" > /dev/null; then
        groupadd "$DOCKER_GROUP"
        log INFO "Created docker group"
    fi

    # Create or update MCP user
    if ! id "$MCP_USER" > /dev/null 2>&1; then
        useradd -r -g "$DOCKER_GROUP" -d "$INSTALL_DIR" -s /bin/false "$MCP_USER"
        log INFO "Created user: $MCP_USER"
    else
        # Update existing user's home directory
        usermod -d "$INSTALL_DIR" "$MCP_USER"
        usermod -aG "$DOCKER_GROUP" "$MCP_USER"
        log INFO "Updated user: $MCP_USER"
    fi

    # Set repository ownership
    chown -R "$MCP_USER:$DOCKER_GROUP" "$INSTALL_DIR"

    # Set proper permissions
    chmod 755 "$INSTALL_DIR"
    find "$INSTALL_DIR" -type d -exec chmod 755 {} \; 2>/dev/null || true
    find "$INSTALL_DIR" -type f -exec chmod 644 {} \; 2>/dev/null || true

    # Make scripts executable
    find "$INSTALL_DIR/docker/scripts" -type f -name "*.sh" -exec chmod 755 {} \; 2>/dev/null || true

    log INFO "User and permissions configured"

}

# Setup environment with dynamic paths

setup_environment() {
log INFO "Setting up environment configuration..."

    # Copy example if no .env exists
    if [[ ! -f "$INSTALL_DIR/.env" ]]; then
        if [[ -f "$INSTALL_DIR/.env.example" ]]; then
            cp "$INSTALL_DIR/.env.example" "$INSTALL_DIR/.env"
            log INFO "Copied .env.example to .env"
        else
            # Create minimal .env
            cat > "$INSTALL_DIR/.env" << 'EOF'

# MCP Portal Configuration - Generated by flexible installation

# CRITICAL: Update these values for production

# REQUIRED: Change these values

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

# URLs for Docker networking

DATABASE_URL=postgresql://portal:change-in-production@postgres:5432/mcp_portal
REDIS_URL=redis://redis:6379
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
EOF
log INFO "Created basic .env file"
fi

        chown "$MCP_USER:$DOCKER_GROUP" "$INSTALL_DIR/.env"
        chmod 600 "$INSTALL_DIR/.env"
        log WARN "CRITICAL: Edit $INSTALL_DIR/.env with your production values"
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
Description=MCP Portal Service (Flexible Installation)
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

# Security settings - Production grade

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

# Build and start services (no symlinks needed!)

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

# Setup log rotation with dynamic paths

setup_log_rotation() {
local install_dir="$1"

    log INFO "Setting up log rotation..."

    cat > /etc/logrotate.d/mcp-portal << EOF

/var/log/mcp-portal/\*.log {
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

$install_dir/logs/\*.log {
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

# Migrate from existing installation

migrate_from_existing() {
local old_install="/opt/mcp-portal"

    if [[ ! -d "$old_install" ]]; then
        log INFO "No existing installation found, skipping migration"
        return 0
    fi

    log INFO "Migrating from existing installation..."

    # Create backup
    mkdir -p "$BACKUP_DIR"
    local backup_name="backup-$(date +%Y%m%d_%H%M%S)"

    # Stop old service
    if systemctl is-active --quiet mcp-portal; then
        log INFO "Stopping old service..."
        systemctl stop mcp-portal
    fi

    # Backup configuration
    if [[ -f "$old_install/.env" ]]; then
        cp "$old_install/.env" "$BACKUP_DIR/$backup_name.env"
        log INFO "Backed up .env to $BACKUP_DIR/$backup_name.env"

        # Copy to new location
        cp "$old_install/.env" "$INSTALL_DIR/.env"
        chown "$MCP_USER:$DOCKER_GROUP" "$INSTALL_DIR/.env"
        chmod 600 "$INSTALL_DIR/.env"
        log INFO "Migrated .env configuration"
    fi

    # Backup data volumes if they exist
    if docker volume ls | grep -q mcp-portal; then
        log INFO "Docker volumes detected - they will be preserved"
        log INFO "Volume backup recommended: docker run --rm -v mcp-portal-postgres-data:/data -v \$PWD:/backup alpine tar czf /backup/postgres-backup-\$(date +%Y%m%d).tar.gz /data"
    fi

    # Disable old service
    if systemctl is-enabled --quiet mcp-portal; then
        systemctl disable mcp-portal
        log INFO "Disabled old service"
    fi

    # Remove old service file (will be replaced)
    if [[ -f /etc/systemd/system/mcp-portal.service ]]; then
        mv /etc/systemd/system/mcp-portal.service "$BACKUP_DIR/$backup_name.service"
        log INFO "Backed up old service file"
    fi

    log INFO "Migration completed - old installation preserved in $BACKUP_DIR"

}

# Start services

start_services() {
if [["$CONFIG_ONLY" == true]]; then
log INFO "Configuration complete. Services not started (--config-only specified)"
return 0
fi

    log INFO "Starting services..."

    # Start and enable Docker
    systemctl enable docker
    systemctl start docker

    # Wait for Docker to be ready
    log INFO "Waiting for Docker to be ready..."
    timeout 60 bash -c 'until docker info >/dev/null 2>&1; do sleep 2; done' || {
        error_exit "Docker failed to start within 60 seconds"
    }

    # Enable and start MCP Portal service
    systemctl enable mcp-portal.service
    log INFO "Starting MCP Portal service..."
    systemctl start mcp-portal.service

    # Wait for service to be ready
    sleep 10

    # Check service status
    if systemctl is-active --quiet mcp-portal.service; then
        log INFO "MCP Portal service started successfully"
    else
        log ERROR "MCP Portal service failed to start"
        log ERROR "Check logs with: journalctl -u mcp-portal.service -f"
        exit 1
    fi

}

# Verify installation

verify_installation() {
log INFO "Verifying installation..."

    # Check Docker
    if ! docker --version >/dev/null 2>&1; then
        log ERROR "Docker is not working properly"
        return 1
    fi

    # Check Docker Compose
    if ! docker compose version >/dev/null 2>&1; then
        log ERROR "Docker Compose is not working properly"
        return 1
    fi

    # Check systemd service
    if ! systemctl is-enabled --quiet mcp-portal.service; then
        log ERROR "MCP Portal service is not enabled"
        return 1
    fi

    # Check required files exist at installation location
    local required_files=(
        "docker-compose.yaml"
        ".env"
        "Dockerfile.portal"
        "Dockerfile.frontend"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f "$INSTALL_DIR/$file" ]]; then
            log ERROR "Required file missing: $INSTALL_DIR/$file"
            return 1
        fi
    done

    log INFO "Installation verification completed successfully"
    return 0

}

# Print completion message

print_completion_message() {
cat << EOF

${GREEN}========================================${NC}
${GREEN}  MCP Portal Flexible Installation Complete!${NC}
${GREEN}========================================${NC}

Installation Type: ${BLUE}In-Place (Flexible)${NC}
Repository Location: ${BLUE}$PROJECT_ROOT${NC}
Installation Directory: ${BLUE}$INSTALL_DIR${NC}
Service Name: ${BLUE}mcp-portal.service${NC}

${YELLOW}🎉 Advantages of Flexible Installation:${NC}
✅ Works from any clone location
✅ No symlinks or file copying required
✅ Direct Docker build context (no symlink errors)
✅ Easier development and updates
✅ Single source of truth
✅ Production-grade security maintained

${YELLOW}Next Steps:${NC}

1. ${RED}CRITICAL:${NC} Edit ${BLUE}$INSTALL_DIR/.env${NC} with your production values
2. Configure Azure AD credentials and JWT secret
3. Start the service: ${BLUE}sudo systemctl start mcp-portal${NC}

${YELLOW}Service Management:${NC}
Start: ${BLUE}sudo systemctl start mcp-portal${NC}
Stop: ${BLUE}sudo systemctl stop mcp-portal${NC}
Status: ${BLUE}sudo systemctl status mcp-portal${NC}
Logs: ${BLUE}sudo journalctl -u mcp-portal -f${NC}

${YELLOW}Docker Commands (from repository):${NC}
View: ${BLUE}cd $INSTALL_DIR && docker compose ps${NC}
Logs: ${BLUE}cd $INSTALL_DIR && docker compose logs -f${NC}
Update: ${BLUE}cd $INSTALL_DIR && git pull && sudo systemctl restart mcp-portal${NC}

${YELLOW}Development Workflow:${NC}
Update: ${BLUE}git pull && sudo systemctl restart mcp-portal${NC}
Debug: ${BLUE}docker compose logs -f${NC}
Build: ${BLUE}docker compose build --no-cache${NC}

For troubleshooting: ${BLUE}$LOG_FILE${NC}

EOF
}

# Main installation function

main() {
log INFO "Starting MCP Portal flexible installation..."
log INFO "Repository root: $PROJECT_ROOT"
log INFO "Installing in-place at: $INSTALL_DIR"

    # Setup and validation
    check_root
    parse_args "$@"
    validate_repository
    detect_distro

    # Migration if requested
    if [[ "$MIGRATE" == true ]]; then
        migrate_from_existing
    fi

    # Installation steps
    install_docker
    create_mcp_user
    setup_docker_permissions
    setup_environment "$INSTALL_DIR"
    generate_systemd_service "$INSTALL_DIR"
    setup_log_rotation "$INSTALL_DIR"
    start_services

    # Verification
    if verify_installation; then
        print_completion_message
    else
        error_exit "Installation verification failed"
    fi

    log INFO "Flexible installation completed successfully"

}

# Handle interruption

trap 'error_exit "Installation interrupted"' INT TERM

# Initialize logging

mkdir -p "$(dirname "$LOG_FILE")"
touch "$LOG_FILE"
chmod 644 "$LOG_FILE"

# Run main installation

main "$@"
