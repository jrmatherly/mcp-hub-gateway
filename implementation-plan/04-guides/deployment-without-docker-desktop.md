# Deployment Without Docker Desktop

## Overview

The MCP Portal is designed to work with the Docker Engine directly without requiring Docker Desktop, making it suitable for server deployments, cloud environments, and resource-constrained systems. This document outlines the deployment strategies and configuration requirements.

## Docker Engine Configuration

### Standalone Docker Engine Setup

#### Linux (Ubuntu/Debian)

```bash
# Install Docker Engine
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group
sudo usermod -aG docker $USER

# Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Verify installation
docker version
```

#### CentOS/RHEL/Fedora

```bash
# Install Docker Engine
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker

# Configure user permissions
sudo usermod -aG docker $USER
```

### Docker Socket Configuration

The MCP Portal requires access to the Docker socket for container management:

```yaml
# docker-compose.yml
services:
  mcp-portal:
    image: docker/mcp-portal:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - DOCKER_HOST=unix:///var/run/docker.sock
    user: "1000:999" # user:docker group
```

### Socket Security Considerations

```bash
# Create dedicated docker group for portal
sudo groupadd docker-portal

# Add portal user to group
sudo usermod -aG docker-portal mcp-portal

# Set appropriate socket permissions
sudo chown root:docker-portal /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock
```

## Container Runtime Configuration

### Docker Engine Daemon Configuration

Create `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "5"
  },
  "storage-driver": "overlay2",
  "storage-opts": ["overlay2.override_kernel_check=true"],
  "default-ulimits": {
    "nofile": {
      "Hard": 64000,
      "Name": "nofile",
      "Soft": 64000
    }
  },
  "default-runtime": "runc",
  "runtimes": {
    "runc": {
      "path": "runc"
    }
  },
  "exec-opts": ["native.cgroupdriver=systemd"],
  "registry-mirrors": [],
  "insecure-registries": [],
  "debug": false,
  "experimental": false,
  "features": {
    "buildkit": true
  },
  "builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "20GB"
    }
  }
}
```

### Resource Limits

```bash
# Configure systemd service limits
sudo mkdir -p /etc/systemd/system/docker.service.d

# Create override configuration
cat > /etc/systemd/system/docker.service.d/override.conf << EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
EOF

# Reload systemd and restart Docker
sudo systemctl daemon-reload
sudo systemctl restart docker
```

## Network Management

### Bridge Network Configuration

```bash
# Create dedicated bridge network for MCP containers
docker network create \
  --driver bridge \
  --subnet=10.20.0.0/16 \
  --ip-range=10.20.240.0/20 \
  --opt com.docker.network.bridge.name=mcp-bridge \
  mcp-network
```

### Network Security Rules

```bash
# Configure iptables for container isolation
iptables -I DOCKER-USER -i mcp-bridge -o mcp-bridge -j ACCEPT
iptables -I DOCKER-USER -i mcp-bridge ! -o mcp-bridge -j ACCEPT
iptables -I DOCKER-USER -o mcp-bridge -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -I DOCKER-USER -o mcp-bridge -j DROP
```

### DNS Configuration

```yaml
# Custom DNS configuration for containers
services:
  mcp-server-github:
    dns:
      - 1.1.1.1
      - 8.8.8.8
    dns_search:
      - internal.company.com
```

## Volume Management

### Persistent Storage Strategy

```bash
# Create dedicated volume directory
sudo mkdir -p /opt/mcp-portal/{data,config,logs}
sudo chown -R 1000:1000 /opt/mcp-portal

# Set SELinux context (if applicable)
sudo setsebool -P container_manage_cgroup 1
sudo semanage fcontext -a -t container_file_t "/opt/mcp-portal(/.*)?"
sudo restorecon -R /opt/mcp-portal
```

### Volume Configuration

```yaml
# docker-compose.yml volumes section
volumes:
  portal_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/mcp-portal/data

  portal_config:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/mcp-portal/config

  portal_logs:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/mcp-portal/logs
```

### Backup Strategy

```bash
#!/bin/bash
# backup-volumes.sh

BACKUP_DIR="/backup/mcp-portal/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Stop portal services
docker compose stop

# Create volume backups
docker run --rm \
  -v portal_data:/source:ro \
  -v "$BACKUP_DIR":/backup \
  alpine:latest \
  tar czf /backup/data.tar.gz -C /source .

docker run --rm \
  -v portal_config:/source:ro \
  -v "$BACKUP_DIR":/backup \
  alpine:latest \
  tar czf /backup/config.tar.gz -C /source .

# Restart portal services
docker compose start

echo "Backup completed: $BACKUP_DIR"
```

## Secret Management

### Docker Secrets (Swarm Mode)

```yaml
# docker-compose.yml for swarm deployment
services:
  mcp-portal:
    secrets:
      - jwt_private_key
      - azure_client_secret
      - postgres_password
    environment:
      - JWT_PRIVATE_KEY_FILE=/run/secrets/jwt_private_key
      - AZURE_CLIENT_SECRET_FILE=/run/secrets/azure_client_secret
      - POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password

secrets:
  jwt_private_key:
    external: true
  azure_client_secret:
    external: true
  postgres_password:
    external: true
```

### Environment File Management

```bash
# .env file with restricted permissions
chmod 600 .env
chown root:root .env

# .env content
DATABASE_URL=postgresql://user:password@postgres:5432/mcp_portal
REDIS_URL=redis://redis:6379/0
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
JWT_SECRET_KEY=your-jwt-secret
```

### External Secret Management

```bash
# HashiCorp Vault integration example
#!/bin/bash
# load-secrets.sh

export VAULT_ADDR="https://vault.company.com"
export VAULT_TOKEN="$(vault write -field=token auth/aws/login role=mcp-portal)"

# Fetch secrets from Vault
JWT_SECRET=$(vault kv get -field=jwt_secret secret/mcp-portal)
AZURE_SECRET=$(vault kv get -field=client_secret secret/azure-ad)
DB_PASSWORD=$(vault kv get -field=password secret/postgres)

# Export for Docker Compose
export JWT_SECRET_KEY="$JWT_SECRET"
export AZURE_CLIENT_SECRET="$AZURE_SECRET"
export POSTGRES_PASSWORD="$DB_PASSWORD"
```

## System Service Configuration

### Systemd Service for Docker Compose

Create `/etc/systemd/system/mcp-portal.service`:

```ini
[Unit]
Description=MCP Portal Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/mcp-portal
Environment=COMPOSE_PROJECT_NAME=mcp_portal
ExecStartPre=/usr/bin/docker compose pull --quiet
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
ExecReload=/usr/bin/docker compose restart
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
```

### Service Management Commands

```bash
# Enable and start service
sudo systemctl enable mcp-portal.service
sudo systemctl start mcp-portal.service

# Check service status
sudo systemctl status mcp-portal.service

# View service logs
sudo journalctl -u mcp-portal.service -f
```

## Production Deployment Architecture

### Multi-Host Deployment with Docker Swarm

```yaml
# docker-stack.yml
version: "3.8"

services:
  mcp-portal:
    image: docker/mcp-portal:latest
    deploy:
      replicas: 3
      placement:
        constraints:
          - node.role == worker
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    networks:
      - mcp-overlay
    secrets:
      - jwt_private_key
      - azure_client_secret

networks:
  mcp-overlay:
    driver: overlay
    attachable: true
    encrypted: true
```

### Load Balancer Configuration (Nginx)

```nginx
# /etc/nginx/sites-available/mcp-portal
upstream mcp_portal {
    least_conn;
    server portal1.internal:3000 weight=1 max_fails=3 fail_timeout=30s;
    server portal2.internal:3000 weight=1 max_fails=3 fail_timeout=30s;
    server portal3.internal:3000 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl http2;
    server_name mcp-portal.company.com;

    ssl_certificate /etc/ssl/certs/mcp-portal.crt;
    ssl_certificate_key /etc/ssl/private/mcp-portal.key;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";

    # WebSocket support
    location /ws {
        proxy_pass http://mcp_portal;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }

    # API endpoints
    location /api/ {
        proxy_pass http://mcp_portal;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Frontend assets
    location / {
        proxy_pass http://mcp_portal;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring and Health Checks

### Docker Health Checks

```dockerfile
# Dockerfile health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:3000/api/health || exit 1
```

### External Monitoring

```bash
# check-portal-health.sh
#!/bin/bash

PORTAL_URL="https://mcp-portal.company.com"
HEALTH_ENDPOINT="$PORTAL_URL/api/health"

# Check health endpoint
RESPONSE=$(curl -s -w "%{http_code}" "$HEALTH_ENDPOINT" -o /dev/null)

if [ "$RESPONSE" -eq 200 ]; then
    echo "Portal is healthy"
    exit 0
else
    echo "Portal health check failed: HTTP $RESPONSE"
    exit 1
fi
```

### Log Aggregation

```yaml
# Fluentd configuration for log collection
services:
  fluentd:
    image: fluent/fluentd:v1.16
    volumes:
      - ./fluentd/conf:/fluentd/etc
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
    ports:
      - "24224:24224"
      - "24224:24224/udp"

  mcp-portal:
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: mcp.portal
```

## Troubleshooting

### Common Issues and Solutions

#### Docker Socket Permission Issues

```bash
# Fix socket permissions
sudo chown root:docker /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock

# Check socket accessibility
docker info
```

#### Container Resource Limits

```bash
# Check system resources
docker system df
docker system prune -f

# Monitor container resources
docker stats --no-stream
```

#### Network Connectivity Issues

```bash
# Check network configuration
docker network ls
docker network inspect mcp-network

# Test container connectivity
docker run --rm --network mcp-network alpine:latest ping postgres
```

### Diagnostic Commands

```bash
# Collect diagnostic information
#!/bin/bash
# diagnose-portal.sh

echo "=== Docker Version ==="
docker version

echo "=== Docker System Info ==="
docker system info

echo "=== Container Status ==="
docker compose ps

echo "=== Portal Logs ==="
docker compose logs mcp-portal --tail=50

echo "=== Network Configuration ==="
docker network ls
docker network inspect mcp-network

echo "=== Volume Information ==="
docker volume ls
docker volume inspect portal_data

echo "=== Resource Usage ==="
docker stats --no-stream
```

## Performance Tuning

### Docker Engine Optimization

```bash
# Optimize Docker daemon for production
echo '{
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  },
  "default-ulimits": {
    "nofile": {
      "Hard": 64000,
      "Soft": 64000
    }
  },
  "max-concurrent-downloads": 10,
  "max-concurrent-uploads": 5
}' | sudo tee /etc/docker/daemon.json

sudo systemctl restart docker
```

### Container Resource Allocation

```yaml
# docker-compose.yml resource limits
services:
  mcp-portal:
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 2G
        reservations:
          cpus: "0.5"
          memory: 512M
```

## Advanced CLI Integration Deployment

### CLI Binary Management in Production

```dockerfile
# Multi-stage Dockerfile for portal with CLI integration
FROM golang:1.24-alpine AS cli-builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mcp-cli ./cmd/docker-mcp
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o portal ./cmd/docker-mcp/portal

FROM alpine:3.19
RUN apk --no-cache add ca-certificates docker-cli curl jq
WORKDIR /app

# Install CLI binary and portal
COPY --from=cli-builder /build/mcp-cli /usr/local/bin/
COPY --from=cli-builder /build/portal /app/
RUN chmod +x /usr/local/bin/mcp-cli /app/portal

# Create non-root user for security
RUN addgroup -g 1000 portal && \
    adduser -D -s /bin/sh -u 1000 -G portal portal && \
    mkdir -p /app/{config,logs,temp} && \
    chown -R portal:portal /app

# Health check with CLI validation
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:3000/api/health || exit 1

USER portal
EXPOSE 3000
CMD ["./portal", "serve"]
```

### Production Container Security

```yaml
# docker-compose.prod.yml with security hardening
version: "3.8"

services:
  mcp-portal:
    image: mcp-portal:${VERSION:-latest}
    user: "1000:999" # portal user:docker group
    read_only: true
    tmpfs:
      - /app/temp:noexec,nosuid,size=1G
      - /tmp:noexec,nosuid,size=500M
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - portal_config:/app/config:ro
      - portal_logs:/app/logs
    environment:
      - MCP_CLI_PATH=/usr/local/bin/mcp-cli
      - DOCKER_HOST=unix:///var/run/docker.sock
      - PORTAL_CONFIG_DIR=/app/config
      - PORTAL_TEMP_DIR=/app/temp
      - PORTAL_LOG_LEVEL=info
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    networks:
      - mcp-network
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

networks:
  mcp-network:
    driver: bridge
    ipam:
      config:
        - subnet: 10.20.0.0/16
    driver_opts:
      com.docker.network.bridge.name: mcp-bridge
      com.docker.network.bridge.enable_icc: "false"
      com.docker.network.bridge.enable_ip_masquerade: "true"
```

### CLI Command Execution Environment

```bash
#!/bin/bash
# setup-cli-environment.sh

set -euo pipefail

# Create dedicated CLI execution environment
create_cli_environment() {
    # Create isolated directory structure
    mkdir -p /opt/mcp-cli/{bin,config,temp,logs}

    # Set up proper permissions
    chown -R 1000:999 /opt/mcp-cli
    chmod 755 /opt/mcp-cli/{bin,config,logs}
    chmod 700 /opt/mcp-cli/temp  # Restrict temp directory

    # Create systemd tmpfiles.d entry for temp directory cleanup
    cat > /etc/tmpfiles.d/mcp-cli.conf << 'EOF'
# Clean up MCP CLI temporary files older than 1 hour
d /opt/mcp-cli/temp 0700 1000 999 - -
D /opt/mcp-cli/temp/* - - - 1h -
EOF
}

# Configure Docker socket access with security
setup_docker_access() {
    # Create dedicated Docker group for MCP
    groupadd -f docker-mcp

    # Add portal user to docker-mcp group
    usermod -aG docker-mcp portal || useradd -G docker-mcp portal

    # Set up socket permissions
    chown root:docker-mcp /var/run/docker.sock
    chmod 660 /var/run/docker.sock

    # Create systemd service for socket permission management
    cat > /etc/systemd/system/mcp-docker-socket.service << 'EOF'
[Unit]
Description=MCP Docker Socket Permission Manager
Wants=docker.socket
After=docker.socket

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/bash -c 'chown root:docker-mcp /var/run/docker.sock && chmod 660 /var/run/docker.sock'
ExecReload=/bin/bash -c 'chown root:docker-mcp /var/run/docker.sock && chmod 660 /var/run/docker.sock'

[Install]
WantedBy=multi-user.target
EOF

    systemctl enable mcp-docker-socket.service
    systemctl start mcp-docker-socket.service
}

# Set up CLI execution monitoring
setup_monitoring() {
    # Create log rotation for CLI command logs
    cat > /etc/logrotate.d/mcp-cli << 'EOF'
/opt/mcp-cli/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 1000 999
    postrotate
        systemctl reload mcp-portal || true
    endscript
}
EOF

    # Set up resource monitoring
    cat > /etc/systemd/system/mcp-cli-monitor.service << 'EOF'
[Unit]
Description=MCP CLI Resource Monitor
After=mcp-portal.service

[Service]
Type=simple
User=portal
Group=portal
ExecStart=/opt/mcp-cli/bin/monitor-resources.sh
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
}

# Create CLI resource monitoring script
create_resource_monitor() {
    cat > /opt/mcp-cli/bin/monitor-resources.sh << 'EOF'
#!/bin/bash
# CLI resource monitoring script

METRICS_FILE="/opt/mcp-cli/logs/metrics.log"
CHECK_INTERVAL=30

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Check CLI processes
    cli_processes=$(pgrep -f "mcp-cli" | wc -l)

    # Check temp directory usage
    temp_usage=$(du -sm /opt/mcp-cli/temp | cut -f1)

    # Check Docker socket access
    docker_access=0
    if docker version >/dev/null 2>&1; then
        docker_access=1
    fi

    # Log metrics in JSON format
    echo "{\"timestamp\":\"$timestamp\",\"cli_processes\":$cli_processes,\"temp_usage_mb\":$temp_usage,\"docker_access\":$docker_access}" >> "$METRICS_FILE"

    sleep $CHECK_INTERVAL
done
EOF

    chmod +x /opt/mcp-cli/bin/monitor-resources.sh
}

# Execute setup
main() {
    echo "Setting up MCP CLI execution environment..."
    create_cli_environment
    setup_docker_access
    setup_monitoring
    create_resource_monitor

    echo "CLI environment setup completed successfully!"
}

main "$@"
```

### Security Hardening Script

```bash
#!/bin/bash
# security-hardening.sh

set -euo pipefail

# Harden CLI execution security
harden_cli_security() {
    # Set up AppArmor profile for CLI execution
    cat > /etc/apparmor.d/mcp-cli << 'EOF'
#include <tunables/global>

/usr/local/bin/mcp-cli {
  #include <abstractions/base>
  #include <abstractions/nameservice>

  # Allow CLI binary execution
  /usr/local/bin/mcp-cli mr,

  # Allow access to Docker socket
  /var/run/docker.sock rw,

  # Allow access to CLI directories
  /opt/mcp-cli/** rw,
  /app/temp/** rw,

  # Deny dangerous capabilities
  deny capability sys_admin,
  deny capability sys_module,
  deny capability sys_chroot,

  # Allow necessary capabilities
  capability net_bind_service,
  capability setuid,
  capability setgid,

  # Limit file access
  deny /etc/shadow r,
  deny /etc/passwd w,
  deny /root/** rwx,
  deny /home/** rwx,

  # Allow temp file creation
  /tmp/** rw,
  /var/tmp/** rw,
}
EOF

    # Load AppArmor profile
    apparmor_parser -r /etc/apparmor.d/mcp-cli || echo "AppArmor not available, skipping"
}

# Set up audit logging for CLI commands
setup_audit_logging() {
    # Configure auditd rules for CLI monitoring
    cat >> /etc/audit/rules.d/mcp-cli.rules << 'EOF'
# Monitor CLI binary execution
-w /usr/local/bin/mcp-cli -p x -k mcp_cli_execution

# Monitor Docker socket access
-w /var/run/docker.sock -p rw -k docker_socket_access

# Monitor temp file creation
-w /opt/mcp-cli/temp -p wa -k mcp_temp_files

# Monitor configuration access
-w /opt/mcp-cli/config -p ra -k mcp_config_access
EOF

    # Restart auditd to load new rules
    systemctl restart auditd || echo "auditd not available"
}

# Configure fail2ban for portal protection
setup_fail2ban() {
    cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[mcp-portal-auth]
enabled = true
port = 443,80
protocol = tcp
filter = mcp-portal-auth
logpath = /opt/mcp-portal/logs/portal.log
maxretry = 3
bantime = 7200

[mcp-portal-api]
enabled = true
port = 443,80
protocol = tcp
filter = mcp-portal-api
logpath = /opt/mcp-portal/logs/portal.log
maxretry = 10
bantime = 1800
EOF

    # Create custom filters
    cat > /etc/fail2ban/filter.d/mcp-portal-auth.conf << 'EOF'
[Definition]
failregex = .*authentication failed.*client_ip="<HOST>".*
ignoreregex =
EOF

    cat > /etc/fail2ban/filter.d/mcp-portal-api.conf << 'EOF'
[Definition]
failregex = .*rate_limited.*client_ip="<HOST>".*
ignoreregex =
EOF

    systemctl enable fail2ban
    systemctl restart fail2ban
}

# Main execution
main() {
    echo "Applying security hardening..."
    harden_cli_security
    setup_audit_logging
    setup_fail2ban
    echo "Security hardening completed!"
}

main "$@"
```

### Performance Optimization Configuration

```yaml
# performance-tuning.yml
version: "3.8"

services:
  mcp-portal:
    image: mcp-portal:${VERSION}
    deploy:
      resources:
        limits:
          cpus: "4.0"
          memory: 4G
        reservations:
          cpus: "1.0"
          memory: 1G
      placement:
        constraints:
          - node.labels.performance == high
    environment:
      # Go runtime optimization
      - GOGC=100
      - GOMAXPROCS=4
      - GOMEMLIMIT=3GiB

      # CLI execution optimization
      - MCP_CLI_POOL_SIZE=20
      - MCP_CLI_TIMEOUT=30s
      - MCP_CLI_CACHE_TTL=300s

      # Database connection optimization
      - DB_MAX_CONNECTIONS=50
      - DB_MAX_IDLE_CONNECTIONS=10
      - DB_CONNECTION_MAX_LIFETIME=1h

      # Redis optimization
      - REDIS_POOL_SIZE=20
      - REDIS_POOL_TIMEOUT=5s
      - REDIS_IDLE_TIMEOUT=300s

  postgres:
    image: postgres:17-alpine
    environment:
      # PostgreSQL performance tuning
      - POSTGRES_INITDB_ARGS="--data-checksums"
    command: >
      postgres
      -c shared_preload_libraries=pg_stat_statements
      -c max_connections=100
      -c shared_buffers=256MB
      -c effective_cache_size=1GB
      -c maintenance_work_mem=64MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100
      -c random_page_cost=1.1
      -c effective_io_concurrency=200
      -c work_mem=4MB
      -c min_wal_size=1GB
      -c max_wal_size=4GB
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 512M

  redis:
    image: redis:8-alpine
    command: >
      redis-server
      --maxmemory 512mb
      --maxmemory-policy allkeys-lru
      --save 900 1
      --save 300 10
      --save 60 10000
      --tcp-keepalive 300
      --timeout 0
    deploy:
      resources:
        limits:
          memory: 768M
        reservations:
          memory: 256M
```

This comprehensive deployment guide provides enterprise-grade reliability, security, and performance optimization for the MCP Portal with full CLI integration capability, ensuring production-ready operation without Docker Desktop dependency.
