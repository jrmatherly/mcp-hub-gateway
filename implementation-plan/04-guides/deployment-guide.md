# Deployment Guide

## Overview

Complete deployment guide for the MCP Portal in production environments using Docker and VMware infrastructure.

## Prerequisites

### Infrastructure Requirements

```yaml
minimum_requirements:
  cpu: 4 cores
  memory: 8GB RAM
  disk: 100GB SSD
  network: 1Gbps
  os: Ubuntu 22.04 LTS

recommended_requirements:
  cpu: 8 cores
  memory: 16GB RAM
  disk: 250GB SSD (NVMe preferred)
  network: 10Gbps
  os: Ubuntu 22.04 LTS
  high_availability: 3 nodes minimum
```

### Software Dependencies

```bash
# Required software
- Docker Engine 24.0+
- Docker Compose 2.20+
- PostgreSQL 17+
- Redis 8+
- Nginx 1.24+
- Certbot (for SSL)
```

### Network Requirements

```yaml
ports:
  - 80/tcp # HTTP (redirect to HTTPS)
  - 443/tcp # HTTPS
  - 3000/tcp # Portal UI (internal)
  - 8080/tcp # API (internal)
  - 5432/tcp # PostgreSQL (internal)
  - 6379/tcp # Redis (internal)

firewall_rules:
  inbound:
    - 443/tcp from 0.0.0.0/0
    - 22/tcp from management_subnet

  outbound:
    - 443/tcp to Azure AD endpoints
    - 443/tcp to Docker Registry
    - 443/tcp to monitoring endpoints
```

## Pre-Deployment Checklist

### Environment Configuration

```bash
# Create deployment directory
mkdir -p /opt/mcp-portal/{config,data,logs,backups}

# Set permissions
chown -R mcp-portal:mcp-portal /opt/mcp-portal
chmod 750 /opt/mcp-portal
```

### SSL Certificates

```bash
# Generate SSL certificates with Let's Encrypt
certbot certonly --standalone -d mcp-portal.company.com

# Verify certificates
ls -la /etc/letsencrypt/live/mcp-portal.company.com/
```

### Environment Variables

```bash
# /opt/mcp-portal/config/.env
# Database Configuration
DATABASE_URL=postgresql://mcp_user:${DB_PASSWORD}@postgres:5432/mcp_portal
DB_MAX_CONNECTIONS=25
DB_POOL_SIZE=10

# Redis Configuration
REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379/0
REDIS_MAX_CONNECTIONS=50

# Azure AD Configuration
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET}
AZURE_REDIRECT_URI=https://mcp-portal.company.com/auth/callback

# Portal Configuration
PORTAL_HOST=0.0.0.0
PORTAL_PORT=3000
API_PORT=8080
NODE_ENV=production

# Security
JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}
SESSION_SECRET=${SESSION_SECRET}

# Monitoring
SENTRY_DSN=https://xxx@sentry.io/xxx
PROMETHEUS_ENABLED=true
LANGFUSE_PUBLIC_KEY=${LANGFUSE_KEY}
```

## Deployment Steps

### Step 1: Database Setup

```bash
# Start PostgreSQL
docker run -d \
  --name mcp-postgres \
  --restart unless-stopped \
  -e POSTGRES_DB=mcp_portal \
  -e POSTGRES_USER=mcp_user \
  -e POSTGRES_PASSWORD_FILE=/run/secrets/db_password \
  -v /opt/mcp-portal/data/postgres:/var/lib/postgresql/data \
  -v /opt/mcp-portal/config/postgres-secrets:/run/secrets:ro \
  --network mcp-network \
  postgres:17-alpine

# Run migrations
docker exec mcp-postgres psql -U mcp_user -d mcp_portal < /migrations/001_initial_schema.sql
docker exec mcp-postgres psql -U mcp_user -d mcp_portal < /migrations/002_enable_rls.sql
docker exec mcp-postgres psql -U mcp_user -d mcp_portal < /migrations/003_seed_data.sql
```

### Step 2: Redis Setup

```bash
# Start Redis
docker run -d \
  --name mcp-redis \
  --restart unless-stopped \
  -v /opt/mcp-portal/data/redis:/data \
  --network mcp-network \
  redis:8-alpine \
  redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
```

### Step 3: Build and Deploy Application

```bash
# Build Portal image
cd /opt/mcp-portal
docker build -t mcp-portal:latest -f Dockerfile.portal .

# Run Portal
docker run -d \
  --name mcp-portal \
  --restart unless-stopped \
  --env-file /opt/mcp-portal/config/.env \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /opt/mcp-portal/logs:/app/logs \
  --network mcp-network \
  -p 3000:3000 \
  -p 8080:8080 \
  mcp-portal:latest
```

### Step 4: Nginx Configuration

```nginx
# /etc/nginx/sites-available/mcp-portal
upstream portal_backend {
    least_conn;
    server localhost:8080;
}

upstream portal_frontend {
    server localhost:3000;
}

server {
    listen 80;
    server_name mcp-portal.company.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name mcp-portal.company.com;

    ssl_certificate /etc/letsencrypt/live/mcp-portal.company.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mcp-portal.company.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Frontend
    location / {
        proxy_pass http://portal_frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API
    location /api {
        proxy_pass http://portal_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts for long-running operations
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket
    location /ws {
        proxy_pass http://portal_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Step 5: Docker Compose Deployment

```yaml
# docker-compose.production.yml
version: "3.8"

services:
  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: mcp_portal
      POSTGRES_USER: mcp_user
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/migrations:ro
    secrets:
      - db_password
    networks:
      - mcp-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mcp_user"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:8-alpine
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - redis_data:/data
    networks:
      - mcp-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  portal:
    image: mcp-portal:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      - NODE_ENV=production
      - DATABASE_URL=postgresql://mcp_user:${DB_PASSWORD}@postgres:5432/mcp_portal
      - REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./logs:/app/logs
    ports:
      - "3000:3000"
      - "8080:8080"
    networks:
      - mcp-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
  redis_data:

networks:
  mcp-network:
    driver: bridge

secrets:
  db_password:
    file: ./secrets/db_password.txt
```

## VMware Deployment

### vSphere Configuration

```yaml
# VM Template
vm_template:
  name: MCP-Portal-Ubuntu-22.04
  cpu: 8
  memory: 16384
  disk:
    - size: 100GB
      type: thin
  network:
    - name: VM Network
      type: E1000E

# Resource Pool
resource_pool:
  name: MCP-Portal-Pool
  cpu:
    reservation: 4000
    limit: 16000
  memory:
    reservation: 8192
    limit: 32768
```

### VM Provisioning Script

```bash
#!/bin/bash
# provision-vm.sh

# Update system
apt-get update && apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-linux-x86_64" \
  -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Configure firewall
ufw allow 22/tcp
ufw allow 443/tcp
ufw --force enable

# Create application user
useradd -m -s /bin/bash mcp-portal
usermod -aG docker mcp-portal

# Setup directories
mkdir -p /opt/mcp-portal/{config,data,logs,backups}
chown -R mcp-portal:mcp-portal /opt/mcp-portal
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "mcp-portal"
    static_configs:
      - targets: ["localhost:8080"]
    metrics_path: "/metrics"
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "MCP Portal Production",
    "panels": [
      {
        "title": "Request Rate",
        "query": "rate(http_requests_total[5m])"
      },
      {
        "title": "Response Time",
        "query": "histogram_quantile(0.95, http_request_duration_seconds)"
      },
      {
        "title": "Active Users",
        "query": "mcp_portal_active_sessions"
      },
      {
        "title": "Container Count",
        "query": "mcp_portal_running_containers"
      }
    ]
  }
}
```

## Backup and Recovery

### Backup Script

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/mcp-portal/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Database backup
docker exec mcp-postgres pg_dump -U mcp_user mcp_portal | \
  gzip > "$BACKUP_DIR/db_backup_$DATE.sql.gz"

# Configuration backup
tar -czf "$BACKUP_DIR/config_backup_$DATE.tar.gz" /opt/mcp-portal/config/

# Docker volumes backup
docker run --rm \
  -v postgres_data:/data \
  -v "$BACKUP_DIR:/backup" \
  alpine tar -czf "/backup/postgres_data_$DATE.tar.gz" /data

# Upload to remote storage (optional)
aws s3 cp "$BACKUP_DIR/db_backup_$DATE.sql.gz" s3://backup-bucket/mcp-portal/

# Cleanup old backups (>30 days)
find "$BACKUP_DIR" -type f -mtime +30 -delete
```

### Recovery Procedure

```bash
#!/bin/bash
# restore.sh

BACKUP_FILE=$1
RESTORE_DATE=$(date +%Y%m%d_%H%M%S)

# Stop services
docker-compose down

# Restore database
gunzip -c "$BACKUP_FILE" | docker exec -i mcp-postgres psql -U mcp_user mcp_portal

# Restore volumes
docker run --rm \
  -v postgres_data:/data \
  -v "$BACKUP_DIR:/backup" \
  alpine sh -c "cd /data && tar -xzf /backup/postgres_data_backup.tar.gz"

# Start services
docker-compose up -d
```

## Health Checks

### Application Health

```bash
# Health check endpoint
curl -f http://localhost:8080/api/health

# Expected response
{
  "status": "healthy",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "docker": "healthy"
  }
}
```

### System Health Script

```bash
#!/bin/bash
# health-check.sh

# Check services
for service in mcp-portal mcp-postgres mcp-redis; do
  if ! docker ps | grep -q $service; then
    echo "ERROR: $service is not running"
    exit 1
  fi
done

# Check disk space
DISK_USAGE=$(df -h /opt/mcp-portal | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 80 ]; then
  echo "WARNING: Disk usage is above 80%"
fi

# Check memory
MEMORY_USAGE=$(free | grep Mem | awk '{print ($3/$2) * 100}')
if (( $(echo "$MEMORY_USAGE > 80" | bc -l) )); then
  echo "WARNING: Memory usage is above 80%"
fi

echo "All systems operational"
```

## Troubleshooting

### Common Issues

#### Database Connection Failed

```bash
# Check PostgreSQL status
docker logs mcp-postgres

# Test connection
docker exec -it mcp-postgres psql -U mcp_user -d mcp_portal -c "SELECT 1"

# Check network
docker network inspect mcp-network
```

#### High Memory Usage

```bash
# Check container stats
docker stats --no-stream

# Analyze memory usage
docker exec mcp-portal cat /proc/meminfo

# Restart with memory limits
docker update --memory="4g" --memory-swap="4g" mcp-portal
```

#### SSL Certificate Issues

```bash
# Renew certificates
certbot renew

# Reload Nginx
nginx -s reload

# Verify certificates
openssl x509 -in /etc/letsencrypt/live/mcp-portal.company.com/cert.pem -text -noout
```

## Maintenance

### Regular Maintenance Tasks

```yaml
daily:
  - Check system health
  - Review error logs
  - Monitor disk space
  - Backup database

weekly:
  - Update Docker images
  - Review security logs
  - Clean old logs
  - Test backup restoration

monthly:
  - Security updates
  - SSL certificate renewal
  - Performance review
  - Capacity planning
```

### Update Procedure

```bash
#!/bin/bash
# update.sh

# Pull latest images
docker pull mcp-portal:latest

# Backup current state
./backup.sh

# Stop and update
docker-compose down
docker-compose pull
docker-compose up -d

# Verify health
sleep 30
curl -f http://localhost:8080/api/health || {
  echo "Health check failed, rolling back"
  docker-compose down
  docker-compose up -d --force-recreate
}
```

## Security Hardening

### System Hardening

```bash
# Disable root SSH
sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config

# Configure fail2ban
apt-get install fail2ban
systemctl enable fail2ban

# Setup UFW firewall
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 443/tcp
ufw enable
```

### Docker Security

```yaml
# Security options in docker-compose
security_opt:
  - no-new-privileges:true
  - seccomp:unconfined
cap_drop:
  - ALL
cap_add:
  - NET_BIND_SERVICE
read_only: true
```

## Performance Tuning

### Database Optimization

```sql
-- Analyze tables
ANALYZE;

-- Vacuum database
VACUUM ANALYZE;

-- Reindex
REINDEX DATABASE mcp_portal;
```

### Docker Optimization

```bash
# Prune unused resources
docker system prune -a -f

# Configure logging
docker run --log-driver=json-file --log-opt max-size=10m --log-opt max-file=3
```

## Disaster Recovery

### RTO/RPO Targets

- **RTO**: 4 hours
- **RPO**: 1 hour

### DR Procedure

1. Provision new infrastructure
2. Restore from latest backup
3. Update DNS records
4. Verify all services
5. Notify users

## Support Information

### Logs Location

- Application: `/opt/mcp-portal/logs/`
- Nginx: `/var/log/nginx/`
- Docker: `docker logs <container>`

### Support Contacts

- DevOps Team: devops@company.com
- On-call: +1-xxx-xxx-xxxx
- Escalation: management@company.com
