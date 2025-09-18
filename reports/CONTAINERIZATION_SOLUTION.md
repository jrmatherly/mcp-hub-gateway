# MCP Portal Containerization Solution

## Executive Summary

This document provides a comprehensive containerization solution that **eliminates the need for the problematic installation script** and provides reliable, production-ready deployment using pure Docker technology.

## Problem Analysis

### Root Causes of Installation Failures

1. **Complex System Dependencies**

   - Systemd service creation requiring host modifications
   - Terminal detection failures in non-interactive environments
   - Docker buildx configuration requiring specific permissions
   - Environment variable and path detection issues

2. **Installation Script Complexity**

   - Over 1600 lines of bash with intricate error handling
   - Host system modifications (users, groups, systemd services)
   - Terminal UI dependencies causing failures in automated environments
   - Silent failures due to background process execution

3. **Environment Fragility**
   - Requires specific Linux distributions
   - Root privileges for host modifications
   - Docker socket permissions and group management
   - Build context symlink issues

## Solution: Native Docker Containerization

### Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     NGINX       │    │   Next.js       │    │   Go Backend    │
│  Load Balancer  │────│   Frontend      │    │   (MCP Portal)  │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
            ┌─────────────────────┼─────────────────────┐
            │                    │                     │
  ┌─────────────────┐   ┌─────────────────┐  ┌─────────────────┐
  │   PostgreSQL    │   │      Redis      │  │   Docker Sock   │
  │   Database      │   │     Cache       │  │   (for CLI)     │
  └─────────────────┘   └─────────────────┘  └─────────────────┘
```

### Key Benefits

1. **No Host Installation Required**

   - Everything runs in containers
   - No systemd service creation
   - No user/group management
   - No host system modifications

2. **Platform Independent**

   - Works on any Docker-capable system
   - Windows, macOS, Linux support
   - Cloud or on-premise deployment
   - Development and production consistency

3. **Simplified Operations**
   - Single command deployment
   - Automated dependency management
   - Built-in health checks
   - Easy scaling and updates

## Deployment Options

### Option 1: Simple Deployment (Recommended)

**Use Case**: Quick setup, development, testing, small production

**Command**:

```bash
./deploy-docker.sh simple up
```

**Features**:

- Lightweight configuration
- Fast startup time
- Resource efficient
- Easy to understand

### Option 2: Production Deployment

**Use Case**: Full production with security hardening

**Command**:

```bash
./deploy-docker.sh production up
```

**Features**:

- Security hardening (non-root users, read-only filesystem)
- Resource limits and monitoring
- Comprehensive logging
- Performance optimization

### Option 3: Development Environment

**Use Case**: Active development with hot reload

**Command**:

```bash
./deploy-docker.sh development up
```

**Features**:

- Hot reload for frontend
- Development tools
- Debug configurations
- Volume mounts for code

## Step-by-Step Deployment Guide

### Prerequisites

1. **Docker Installation**

   ```bash
   # Install Docker Engine 20.10+
   curl -fsSL https://get.docker.com | sh

   # Verify installation
   docker --version
   docker compose version
   ```

2. **System Resources**
   - 4GB+ RAM available
   - 10GB+ disk space
   - Network connectivity for image pulls

### Deployment Steps

1. **Clone Repository**

   ```bash
   git clone https://github.com/jrmatherly/mcp-hub-gateway
   cd mcp-hub-gateway
   ```

2. **Configure Environment**

   ```bash
   # Copy environment template
   cp .env.example .env

   # Edit configuration (required for production)
   nano .env
   ```

3. **Deploy Application**

   ```bash
   # Simple deployment (recommended for first-time)
   ./deploy-docker.sh simple up

   # Or production deployment
   ./deploy-docker.sh production up
   ```

4. **Verify Deployment**

   ```bash
   # Check service status
   ./deploy-docker.sh simple status

   # View logs
   ./deploy-docker.sh simple logs
   ```

5. **Access Application**
   - Frontend: http://localhost
   - Backend API: http://localhost:8080
   - Health Check: http://localhost/health

## Configuration Management

### Environment Variables

#### Required for Production

```bash
# Authentication (Azure AD)
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# Security
JWT_SECRET=minimum-32-character-secret-key

# Database
POSTGRES_PASSWORD=secure-database-password
MCP_PORTAL_DATABASE_PASSWORD=secure-portal-password

# Optional: Redis Authentication
REDIS_PASSWORD=redis-password
```

#### Optional Configuration

```bash
# API Configuration
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_ENABLE_ADMIN=true

# Database Tuning
DB_MAX_CONNECTIONS=50
DB_MAX_IDLE_CONNECTIONS=10

# Performance
GOGC=100
GOMAXPROCS=4
```

### SSL/HTTPS Setup

For production HTTPS, update NGINX configuration:

1. **Obtain SSL Certificates**

   ```bash
   # Using Let's Encrypt
   certbot certonly --webroot -w /var/www/html -d your-domain.com
   ```

2. **Update Docker Compose**

   ```yaml
   nginx:
     volumes:
       - /etc/letsencrypt:/etc/ssl:ro
     ports:
       - "443:443"
   ```

3. **Enable HTTPS in NGINX config**
   - Uncomment SSL server block in `docker/nginx/simple.conf`
   - Update certificate paths

## Monitoring and Operations

### Health Checks

All services include built-in health checks:

```bash
# Check individual service health
docker compose -f docker-compose.simple.yaml ps

# View health check logs
docker compose -f docker-compose.simple.yaml logs nginx
```

### Logging

Centralized logging with rotation:

```bash
# View all logs
./deploy-docker.sh simple logs

# View specific service logs
docker compose -f docker-compose.simple.yaml logs backend
docker compose -f docker-compose.simple.yaml logs frontend
```

### Performance Monitoring

```bash
# Resource usage
docker stats

# Service metrics
docker compose -f docker-compose.simple.yaml top
```

### Backup Operations

```bash
# Database backup
docker compose -f docker-compose.simple.yaml exec postgres \
  pg_dump -U postgres mcp_portal > backup.sql

# Volume backup
docker run --rm -v mcp-portal-postgres-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup.tar.gz -C /data .
```

## Troubleshooting Guide

### Common Issues

#### 1. Port Conflicts

**Error**: `Port 80 is already in use`

**Solution**:

```bash
# Check what's using the port
sudo lsof -i :80

# Stop conflicting service
sudo systemctl stop apache2  # or nginx

# Or use different ports
export NGINX_HTTP_PORT=8080
```

#### 2. Docker Permission Issues

**Error**: `permission denied while trying to connect to Docker daemon`

**Solution**:

```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo
sudo ./deploy-docker.sh simple up
```

#### 3. Memory Issues

**Error**: Container exits with code 137

**Solution**:

```bash
# Increase Docker memory limit
# In Docker Desktop: Settings > Resources > Advanced > Memory

# Or reduce resource limits in compose file
deploy:
  resources:
    limits:
      memory: 1G  # Reduce from 2G
```

#### 4. Build Failures

**Error**: `failed to build frontend`

**Solution**:

```bash
# Clear build cache
docker system prune -a

# Build with no cache
docker compose -f docker-compose.simple.yaml build --no-cache

# Check build logs
docker compose -f docker-compose.simple.yaml build frontend
```

### Advanced Troubleshooting

#### Database Connection Issues

```bash
# Test database connectivity
docker compose -f docker-compose.simple.yaml exec backend \
  wget -q --spider postgres:5432

# Check database logs
docker compose -f docker-compose.simple.yaml logs postgres

# Connect to database directly
docker compose -f docker-compose.simple.yaml exec postgres \
  psql -U postgres -d mcp_portal
```

#### Network Connectivity

```bash
# Test network connectivity between services
docker compose -f docker-compose.simple.yaml exec frontend \
  wget -q --spider backend:8080

# Check network configuration
docker network ls
docker network inspect mcp-portal-network
```

## Security Considerations

### Container Security

1. **Non-root Users**: All services run as non-root users
2. **Read-only Filesystem**: Containers use read-only root filesystem where possible
3. **Capability Restrictions**: Minimal capabilities granted (drop ALL, add specific)
4. **Resource Limits**: CPU and memory limits prevent resource exhaustion
5. **Security Options**: no-new-privileges enabled

### Network Security

1. **Isolated Networks**: Services communicate on isolated Docker networks
2. **No External Database Ports**: Database only accessible within container network
3. **Rate Limiting**: NGINX rate limiting for API endpoints
4. **Security Headers**: Comprehensive security headers in NGINX

### Data Security

1. **Secrets Management**: Environment variables for secrets (upgrade to Docker Secrets for production)
2. **Database Encryption**: PostgreSQL with authentication
3. **SSL/TLS**: HTTPS configuration available
4. **Audit Logging**: Comprehensive audit trails

## Migration from Installation Script

### For Existing Installations

If you have an existing systemd installation:

1. **Stop Existing Services**

   ```bash
   sudo systemctl stop mcp-portal
   sudo systemctl disable mcp-portal
   ```

2. **Backup Data** (if needed)

   ```bash
   # Backup database
   sudo -u postgres pg_dump mcp_portal > backup.sql
   ```

3. **Clean Installation**

   ```bash
   # Remove systemd service
   sudo rm /etc/systemd/system/mcp-portal.service
   sudo systemctl daemon-reload
   ```

4. **Deploy with Docker**

   ```bash
   ./deploy-docker.sh simple up
   ```

5. **Restore Data** (if needed)
   ```bash
   # Restore database
   docker compose -f docker-compose.simple.yaml exec -T postgres \
     psql -U postgres mcp_portal < backup.sql
   ```

## Performance Optimization

### Production Tuning

1. **Resource Allocation**

   ```yaml
   # Adjust based on your system
   deploy:
     resources:
       limits:
         memory: 4G
         cpus: "2.0"
   ```

2. **Database Optimization**

   ```yaml
   postgres:
     command: >
       postgres
       -c shared_buffers=512MB
       -c effective_cache_size=2GB
       -c maintenance_work_mem=128MB
   ```

3. **Redis Tuning**
   ```yaml
   redis:
     command: >
       redis-server
       --maxmemory 1gb
       --maxmemory-policy allkeys-lru
   ```

### Scaling Considerations

1. **Horizontal Scaling**: Use Docker Swarm or Kubernetes for multi-node deployment
2. **Load Balancing**: Add additional NGINX instances behind a load balancer
3. **Database Scaling**: Consider PostgreSQL clustering for high availability
4. **Monitoring**: Implement Prometheus + Grafana for comprehensive monitoring

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy MCP Portal
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Deploy to Production
        run: |
          ./deploy-docker.sh production up
          ./deploy-docker.sh production status
```

### GitLab CI Example

```yaml
deploy:
  stage: deploy
  script:
    - ./deploy-docker.sh production up
    - ./deploy-docker.sh production status
  only:
    - main
```

## Conclusion

This containerization solution provides:

1. **Reliability**: Eliminates installation script complexity and failures
2. **Portability**: Works on any Docker-capable system
3. **Security**: Production-ready security hardening
4. **Scalability**: Easy to scale and monitor
5. **Maintainability**: Simple operations and troubleshooting

The solution completely bypasses the problematic installation script while providing superior deployment reliability and operational simplicity.

## Quick Start Commands

```bash
# 1. Deploy (one command!)
./deploy-docker.sh simple up

# 2. Check status
./deploy-docker.sh simple status

# 3. View logs
./deploy-docker.sh simple logs

# 4. Stop services
./deploy-docker.sh simple down

# 5. Clean up everything
./deploy-docker.sh simple clean
```

**Result**: Working MCP Portal accessible at http://localhost in under 5 minutes, with no host system modifications required.

---

_This solution eliminates all installation script dependencies and provides a robust, production-ready containerized deployment approach._
