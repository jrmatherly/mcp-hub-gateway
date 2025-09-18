# MCP Portal Docker Deployment Guide

Complete Docker deployment guide for the MCP Portal with frontend and backend services.

## 🎯 Overview

This guide covers deploying the MCP Portal using Docker containers with the following architecture:

**Recent Changes (2025-01-20):**

- **Simplified Structure**: Consolidated from 4 Docker Compose files to 2 files
- **Unified Configuration**: Single `.env` file for all services
- **Standardized Ports**: API on 8080, Frontend on 3000
- **Production Ready**: All Dockerfile warnings resolved

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │   Database      │
│   (Next.js)     │◄──►│   (Go API)      │◄──►│  (PostgreSQL)   │
│   Port 3000     │    │   Port 8080     │    │   Port 5432     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
                        ┌─────────────────┐
                        │     Redis       │
                        │   (Cache)       │
                        │   Port 6379     │
                        └─────────────────┘
```

## 📋 Prerequisites

### Required Software

- Docker Engine 20.10+ or Docker Desktop
- Docker Compose v2.0+
- Git (for cloning repository)

### Required Configuration

1. **Azure AD Application** (for authentication)

   - Tenant ID, Client ID, Client Secret
   - Redirect URI configured for your domain

2. **Environment Variables**
   - Copy `.env.example` to `.env`
   - Update all placeholder values with your configuration

### System Requirements

- **Minimum**: 2 CPU cores, 4GB RAM, 10GB disk space
- **Recommended**: 4 CPU cores, 8GB RAM, 20GB disk space
- **Production**: 8 CPU cores, 16GB RAM, 50GB disk space

## 🚀 Quick Start

### 1. Clone and Setup

```bash
# Clone repository
git clone https://github.com/jrmatherly/mcp-hub-gateway
cd mcp-hub-gateway

# Setup environment
cp .env.example .env
# Edit .env with your configuration
```

### 2. Production Deployment

```bash
# Build and start all services
make portal-up

# Or using docker-compose directly
docker-compose up -d

# Check status
docker-compose ps
make portal-logs
```

### 3. Development Environment

```bash
# Start development environment with hot reload
make portal-dev-up

# Or using docker-compose directly
docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up

# Access development tools
open http://localhost:3000  # Frontend
open http://localhost:8080  # Backend API
```

## 🏗️ Build Options

### Frontend Docker Builds

#### Production Build

```bash
# Build production-optimized frontend
make docker-frontend

# Build with specific Node.js version
docker build -f Dockerfile.frontend \
  --build-arg NODE_VERSION=20.18-alpine \
  --build-arg BUILD_MODE=production \
  -t mcp-portal-frontend:latest .
```

#### Development Build

```bash
# Build development frontend with debugging
make docker-frontend-dev

# Build with hot reload support
docker build -f Dockerfile.frontend \
  --build-arg BUILD_MODE=development \
  -t mcp-portal-frontend:dev .
```

#### Static Export Build

```bash
# Build for CDN deployment
make docker-frontend-static

# Build static export
docker build -f Dockerfile.frontend \
  --build-arg BUILD_MODE=static \
  --build-arg NEXT_OUTPUT_MODE=export \
  -t mcp-portal-frontend:static .
```

### Backend Docker Builds

#### Production Build

```bash
# Build production backend
make docker-portal

# Build with custom binary name
docker build -f Dockerfile.portal \
  --build-arg BUILD_MODE=production \
  --build-arg DOCKER_MCP_PLUGIN_BINARY=docker-mcp \
  -t mcp-portal-backend:latest .
```

#### Development Build

```bash
# Build development backend
make docker-portal-dev

# Build with debugging enabled
docker build -f Dockerfile.portal \
  --build-arg BUILD_MODE=development \
  -t mcp-portal-backend:dev .
```

### Multi-Platform Builds

```bash
# Build for multiple platforms
make docker-portal-cross docker-frontend-cross

# Build and push to registry
docker buildx build \
  --platform=linux/amd64,linux/arm64 \
  --push \
  -f Dockerfile.frontend \
  -t registry.example.com/mcp-portal-frontend:latest .
```

## 🔧 Configuration

### Environment Variables

#### Required Variables

```bash
# Azure AD Configuration
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-secret

# JWT Security
JWT_SECRET=your-64-character-secret

# Database
MCP_PORTAL_DATABASE_PASSWORD=secure-password
```

#### Optional Variables

```bash
# Port Configuration
MCP_PORTAL_SERVER_PORT=8080
FRONTEND_PORT=3000

# Feature Flags
NEXT_PUBLIC_ENABLE_WEBSOCKET=true
NEXT_PUBLIC_ENABLE_ADMIN=true

# Performance Tuning
MCP_PORTAL_DATABASE_MAX_CONNECTIONS=20
NEXT_PUBLIC_API_TIMEOUT=30000
```

### Docker Compose Profiles

#### Standard Deployment

```bash
# Basic services (frontend, backend, database, redis)
docker-compose up
```

#### With MCP Gateway

```bash
# Include MCP Gateway service
docker-compose --profile gateway up
```

#### Development with Debug Tools

```bash
# Include pgAdmin and Redis Commander
docker-compose -f docker-compose.override.yaml --profile debug up
```

## 🔍 Monitoring and Health Checks

### Health Check Endpoints

#### Frontend Health Checks

```bash
# Application health
curl http://localhost:3000/

# Next.js internal health
curl http://localhost:3000/_next/static/health

# API proxy health
curl http://localhost:3000/api/health
```

#### Backend Health Checks

```bash
# Application health
curl http://localhost:8080/api/health

# Readiness check
curl http://localhost:8080/api/ready

# Database connectivity
curl http://localhost:8080/api/health/db
```

#### Database Health Checks

```bash
# PostgreSQL connectivity
docker-compose exec postgres pg_isready

# Redis connectivity
docker-compose exec redis redis-cli ping
```

### Monitoring Commands

```bash
# View logs
make portal-logs
docker-compose logs -f [service]

# Check container status
docker-compose ps

# Monitor resource usage
docker stats $(docker-compose ps -q)

# Monitor health status
watch -n 5 'docker-compose ps --format table'
```

## 🔒 Security Configuration

### Production Security Checklist

#### Container Security

- ✅ Non-root users in all containers
- ✅ Read-only file systems where possible
- ✅ Security options: `no-new-privileges:true`
- ✅ Minimal attack surface (Alpine base images)

#### Network Security

```bash
# Verify network isolation
docker network ls | grep mcp-portal

# Check exposed ports
docker-compose port frontend 3000
docker-compose port backend 8080
```

#### Secret Management

```bash
# Store secrets securely
echo "your-jwt-secret" | docker secret create jwt_secret -
echo "your-db-password" | docker secret create db_password -

# Use secrets in compose (production)
# See docker-compose.production.yml for secret integration
```

### SSL/TLS Configuration

#### HTTPS Frontend (Production)

```nginx
# Add to nginx reverse proxy
server {
    listen 443 ssl http2;
    server_name portal.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

## 🚀 Deployment Strategies

### Development Deployment

```bash
# Start development environment
make portal-dev-up

# Features:
# - Hot reload for frontend and backend
# - Debug logging enabled
# - Source code volume mounts
# - Development database with logging
```

### Staging Deployment

```bash
# Build and deploy staging
docker-compose \
  -f docker-compose.staging.yml up -d

# Features:
# - Production images with staging config
# - Monitoring and logging enabled
# - SSL termination at load balancer
```

### Production Deployment

```bash
# Deploy production services
docker-compose \
  -f docker-compose.production.yml up -d

# Features:
# - Multi-platform images
# - Secret management
# - Health checks and monitoring
# - Backup and restore procedures
```

### High Availability Deployment

```bash
# Deploy with replication
docker-compose \
  -f docker-compose.ha.yml up -d

# Features:
# - Multiple frontend/backend replicas
# - Database clustering
# - Load balancer integration
# - Automatic failover
```

## 📊 Performance Optimization

### Resource Limits

```yaml
# Add to docker-compose.yml
services:
  frontend:
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
        reservations:
          cpus: "0.5"
          memory: 256M

  backend:
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 1G
        reservations:
          cpus: "1.0"
          memory: 512M
```

### Caching Strategy

```bash
# Optimize Docker build cache
docker buildx build \
  --cache-from=type=registry,ref=registry.example.com/mcp-portal-cache \
  --cache-to=type=registry,ref=registry.example.com/mcp-portal-cache,mode=max \
  -f Dockerfile.frontend .

# Use volume caching for development
docker run -v frontend_cache:/var/cache/nextjs \
  -v node_modules:/app/node_modules \
  mcp-portal-frontend:dev
```

### Database Optimization

```sql
-- PostgreSQL performance tuning
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

## 🛠️ Troubleshooting

### Common Issues

#### Frontend Build Failures

```bash
# Check Node.js version compatibility
docker run --rm node:20.18-alpine node --version

# Verify dependencies
docker run --rm -v $(pwd)/cmd/docker-mcp/portal/frontend:/app -w /app \
  node:20.18-alpine npm ci --dry-run

# Clear build cache
docker builder prune -f
```

#### Backend Connection Issues

```bash
# Verify Docker socket access
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  alpine/socat -t 5 /var/run/docker.sock

# Check backend logs
docker-compose logs backend

# Test API connectivity
curl -v http://localhost:8080/api/health
```

#### Database Connection Issues

```bash
# Check database logs
docker-compose logs postgres

# Test database connectivity
docker-compose exec postgres \
  psql -U portal -d mcp_portal -c "SELECT version();"

# Reset database
docker-compose down -v
docker volume rm mcp-portal-postgres
```

#### Authentication Issues

```bash
# Verify Azure AD configuration
curl -X POST "https://login.microsoftonline.com/${AZURE_TENANT_ID}/oauth2/v2.0/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=${AZURE_CLIENT_ID}&client_secret=${AZURE_CLIENT_SECRET}&scope=https://graph.microsoft.com/.default&grant_type=client_credentials"

# Check JWT secret consistency
echo $JWT_SECRET | wc -c  # Should be 64+ characters
```

### Debug Commands

```bash
# Enter container for debugging
docker-compose exec frontend sh
docker-compose exec backend sh

# Check container resource usage
docker stats $(docker-compose ps -q)

# Network debugging
docker network inspect mcp-portal-network

# Volume debugging
docker volume inspect mcp-portal-postgres
```

## 📚 Additional Resources

### Makefile Targets

```bash
make docker-frontend         # Build production frontend
make docker-frontend-dev     # Build development frontend
make docker-portal           # Build production backend
make docker-portal-all       # Build both services
make portal-up               # Start production stack
make portal-dev-up           # Start development stack
make portal-test             # Run integration tests
make portal-clean            # Clean all artifacts
```

### Development URLs

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080 (standardized port)
- **Database**: localhost:5432 (internal only)
- **Redis**: localhost:6379 (internal only)
- **pgAdmin**: http://localhost:8082 (debug profile only)
- **Redis Commander**: http://localhost:8081 (debug profile only)

### File Structure

```
├── Dockerfile.frontend           # Frontend container build
├── Dockerfile.portal            # Backend container build
├── docker-compose.yaml         # Main production configuration
├── docker-compose.override.yaml    # Development overrides (auto-loaded)
├── Makefile                     # Build automation
└── cmd/docker-mcp/portal/
    ├── frontend/                # Next.js application
    └── database/migrations/     # Database schema
```

### Best Practices

1. **Always use environment variables** for configuration
2. **Pin specific image versions** for production
3. **Use multi-stage builds** for smaller images
4. **Implement proper health checks** for all services
5. **Mount Docker socket read-only** for security
6. **Use non-root users** in all containers
7. **Implement graceful shutdown** handling
8. **Monitor resource usage** and set limits
9. **Backup persistent volumes** regularly
10. **Use secrets management** for sensitive data

For more information, see the [implementation plan](./implementation-plan/README.md) and [security documentation](./docs/security.md).
