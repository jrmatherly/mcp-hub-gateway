#!/bin/bash
# =============================================================================
# MCP Portal Working Deployment Script
# =============================================================================
# Improved deployment script with better error handling and Docker socket access
# Usage: ./deploy-mcp-portal.sh [build|start|stop|restart|logs|status|clean]
# =============================================================================

set -euo pipefail

# Configuration
COMPOSE_FILE="docker-compose.mcp-portal.yml"
DOCKERFILE="Dockerfile.mcp-portal"
ENV_FILE=".env"
ENV_TEMPLATE=".env.example"
PROJECT_NAME="mcp-portal"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

check_requirements() {
    log "Checking requirements..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        error "Docker Compose v2 is not installed. Please install Docker Compose."
        exit 1
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        error "Docker daemon is not running. Please start Docker."
        exit 1
    fi

    # Check Docker socket permissions
    check_docker_socket

    log "All requirements met ✓"
}

check_docker_socket() {
    log "Checking Docker socket access..."

    # Get Docker socket path
    DOCKER_SOCKET="${DOCKER_HOST:-/var/run/docker.sock}"

    if [[ "$DOCKER_SOCKET" == "unix://"* ]]; then
        DOCKER_SOCKET="${DOCKER_SOCKET#unix://}"
    fi

    if [ -S "$DOCKER_SOCKET" ]; then
        # Check if we can access the socket
        if [ -r "$DOCKER_SOCKET" ] && [ -w "$DOCKER_SOCKET" ]; then
            log "Docker socket is accessible ✓"

            # Get the Docker group ID for the compose file
            if [ -e "$DOCKER_SOCKET" ]; then
                DOCKER_GID=$(stat -c '%g' "$DOCKER_SOCKET" 2>/dev/null || stat -f '%g' "$DOCKER_SOCKET" 2>/dev/null || echo "999")
                export DOCKER_GID
                info "Docker group ID: $DOCKER_GID"
            fi
        else
            warning "Docker socket exists but may not be fully accessible"
            warning "Container management features may be limited"

            # Try to determine if we're in Docker Desktop
            if docker context show 2>/dev/null | grep -q "desktop"; then
                info "Docker Desktop detected - socket access will be handled automatically"
            else
                info "Consider adding your user to the docker group: sudo usermod -aG docker $USER"
            fi
        fi
    else
        warning "Docker socket not found at $DOCKER_SOCKET"
        warning "MCP Portal will not be able to manage containers"
    fi
}

setup_environment() {
    # Check if .env exists
    if [[ ! -f "$ENV_FILE" ]]; then
        if [[ -f "$ENV_TEMPLATE" ]]; then
            warning "No .env file found. Creating from template..."
            cp "$ENV_TEMPLATE" "$ENV_FILE"
            info "Please edit $ENV_FILE with your configuration values before starting."
            info "Especially set your Azure AD credentials and generate a JWT secret."
            echo ""
            echo "Generate a secure JWT secret with:"
            echo "  openssl rand -base64 64"
            echo ""
            return 1
        else
            error "No .env file or template found!"
            exit 1
        fi
    fi

    # Source the environment file
    set -a
    source "$ENV_FILE"
    set +a

    return 0
}

# =============================================================================
# Main Commands
# =============================================================================

cmd_build() {
    log "Building MCP Portal containers..."
    check_requirements

    if ! setup_environment; then
        warning "Environment not configured. Using default values for build."
    fi

    # Clean any previous failed builds
    docker builder prune -f 2>/dev/null || true

    log "Building with Dockerfile: $DOCKERFILE"
    docker compose -f "$COMPOSE_FILE" build --no-cache
    log "Build complete ✓"
}

cmd_start() {
    log "Starting MCP Portal..."
    check_requirements

    if ! setup_environment; then
        error "Environment not configured. Please setup .env first."
        exit 1
    fi

    # Start services
    docker compose -f "$COMPOSE_FILE" up -d

    log "Waiting for services to be healthy..."

    # Wait for PostgreSQL
    local postgres_ready=false
    for i in {1..30}; do
        if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U "${POSTGRES_USER:-mcp_user}" &> /dev/null; then
            postgres_ready=true
            break
        fi
        sleep 1
    done

    if [ "$postgres_ready" = true ]; then
        log "PostgreSQL is ready ✓"
    else
        warning "PostgreSQL may not be fully ready"
    fi

    # Wait for Redis
    local redis_ready=false
    for i in {1..30}; do
        if docker compose -f "$COMPOSE_FILE" exec -T redis redis-cli ping &> /dev/null; then
            redis_ready=true
            break
        fi
        sleep 1
    done

    if [ "$redis_ready" = true ]; then
        log "Redis is ready ✓"
    else
        warning "Redis may not be fully ready"
    fi

    # Give the portal time to start
    sleep 5

    # Check health
    cmd_status

    log "MCP Portal started successfully ✓"
    echo ""
    info "Access the portal at:"
    echo -e "  Frontend: ${GREEN}http://localhost:3000${NC}"
    echo -e "  Backend API: ${GREEN}http://localhost:8080${NC}"
    echo -e "  Health Check: ${GREEN}http://localhost:8080/api/health${NC}"
}

cmd_stop() {
    log "Stopping MCP Portal..."
    docker compose -f "$COMPOSE_FILE" down
    log "MCP Portal stopped ✓"
}

cmd_restart() {
    log "Restarting MCP Portal..."
    cmd_stop
    sleep 2
    cmd_start
}

cmd_logs() {
    log "Showing logs (Ctrl+C to exit)..."
    docker compose -f "$COMPOSE_FILE" logs -f
}

cmd_status() {
    log "Checking service status..."
    docker compose -f "$COMPOSE_FILE" ps

    echo ""
    log "Health checks:"

    # Check PostgreSQL
    if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready &> /dev/null; then
        echo -e "  PostgreSQL: ${GREEN}✓ Healthy${NC}"
    else
        echo -e "  PostgreSQL: ${RED}✗ Unhealthy${NC}"
    fi

    # Check Redis
    if docker compose -f "$COMPOSE_FILE" exec -T redis redis-cli ping &> /dev/null; then
        echo -e "  Redis: ${GREEN}✓ Healthy${NC}"
    else
        echo -e "  Redis: ${RED}✗ Unhealthy${NC}"
    fi

    # Check Portal API
    if curl -sf http://localhost:8080/api/health &> /dev/null; then
        echo -e "  Portal API: ${GREEN}✓ Healthy${NC}"
    else
        echo -e "  Portal API: ${RED}✗ Unhealthy${NC}"
    fi

    # Check Frontend
    if curl -sf http://localhost:3000 &> /dev/null; then
        echo -e "  Frontend: ${GREEN}✓ Healthy${NC}"
    else
        echo -e "  Frontend: ${RED}✗ Unhealthy${NC}"
    fi
}

cmd_clean() {
    warning "This will remove all containers, volumes, and images!"
    echo -n "Are you sure? (y/N): "
    read -r confirm

    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        log "Cleaning up..."
        docker compose -f "$COMPOSE_FILE" down -v
        docker rmi mcp-portal:working 2>/dev/null || true
        docker builder prune -f 2>/dev/null || true
        log "Cleanup complete ✓"
    else
        log "Cleanup cancelled"
    fi
}

cmd_shell() {
    log "Opening shell in portal container..."
    docker compose -f "$COMPOSE_FILE" exec portal /bin/sh
}

cmd_db() {
    log "Connecting to PostgreSQL..."
    docker compose -f "$COMPOSE_FILE" exec postgres psql -U "${POSTGRES_USER:-mcp_user}" -d "${POSTGRES_DB:-mcp_portal}"
}

cmd_debug() {
    log "Debug Information:"
    echo ""
    info "Docker Version:"
    docker --version
    echo ""
    info "Docker Compose Version:"
    docker compose version
    echo ""
    info "Docker Context:"
    docker context show
    echo ""
    info "Docker Info:"
    docker info 2>&1 | grep -E "Server Version|Storage Driver|Cgroup" || true
    echo ""
    info "Environment Variables:"
    env | grep -E "DOCKER_|COMPOSE_" | sort || true
    echo ""
    info "Docker Socket:"
    ls -la "${DOCKER_HOST:-/var/run/docker.sock}" 2>/dev/null || echo "Socket not accessible"
    echo ""
    info "Container Status:"
    docker compose -f "$COMPOSE_FILE" ps
}

# =============================================================================
# Usage Information
# =============================================================================

usage() {
    cat << EOF
MCP Portal Working Deployment Script

Usage: $0 [COMMAND]

Commands:
  build     Build the Docker containers
  start     Start all services
  stop      Stop all services
  restart   Restart all services
  logs      Show logs (follow mode)
  status    Check service health status
  clean     Remove containers, volumes, and images
  shell     Open shell in portal container
  db        Connect to PostgreSQL database
  debug     Show debug information
  help      Show this help message

Quick Start:
  1. Copy .env.example to .env and configure
  2. Run: $0 build
  3. Run: $0 start
  4. Access: http://localhost:3000

Requirements:
  - Docker Engine 20.10+
  - Docker Compose v2
  - 4GB+ RAM available
  - Ports 3000, 8080, 5432, 6379 available

EOF
}

# =============================================================================
# Main Entry Point
# =============================================================================

main() {
    case "${1:-help}" in
        build)
            cmd_build
            ;;
        start)
            cmd_start
            ;;
        stop)
            cmd_stop
            ;;
        restart)
            cmd_restart
            ;;
        logs)
            cmd_logs
            ;;
        status)
            cmd_status
            ;;
        clean)
            cmd_clean
            ;;
        shell)
            cmd_shell
            ;;
        db)
            cmd_db
            ;;
        debug)
            cmd_debug
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            error "Unknown command: $1"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"