# MCP Gateway & Portal - Quick Start Guide

Get up and running with both the MCP CLI Plugin and Portal quickly. This guide covers both development and usage scenarios.

## 🚀 Quick Overview

This repository contains **two projects**:

1. **🔧 MCP CLI Plugin & Gateway** - Docker CLI plugin for MCP server management
2. **🌐 MCP Portal** - Web interface with Azure AD auth and multi-user support

**Current Status**: Portal is ~91% complete (Phases 1-3 done, Phase 4 at 91% complete)

**Recent Updates (2025-01-20)**: Fixed catalog test compilation errors, working on test coverage expansion

## Prerequisites

### For CLI Development & Usage

- **Docker Engine** (Docker Desktop optional - see note below)
- **Go 1.24+** (for development)
- **Make** (for build commands)

**Docker Engine Options:**

- **Option 1: Docker Desktop** (Recommended for local development)
- **Option 2: Standalone Docker Engine** (For servers/production)
  - Use the deployment script: `./deploy-mcp-portal.sh`
  - Or install Docker manually: `curl -fsSL https://get.docker.com | sh`

### For Portal Development (Additional)

- **PostgreSQL 17+** (with Row-Level Security support)
- **Redis 8+** (for sessions and caching)
- **Node.js 22+** and **npm 10+** (for frontend)

## 🔧 CLI Plugin Quick Start

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/jrmatherly/mcp-hub-gateway.git
cd mcp-gateway

# Build and install the CLI plugin
make docker-mcp

# Verify installation
docker mcp --help
```

### 2. Basic Usage

```bash
# Initialize the MCP catalog
docker mcp catalog init

# List available servers in catalog
docker mcp catalog show docker-mcp

# Enable some servers
docker mcp server enable server1 server2

# List enabled servers
docker mcp server list

# Start the gateway
docker mcp gateway run

# In another terminal, check tools
docker mcp tools list
```

### 3. Common Development Tasks

```bash
# Run tests
make test

# Run integration tests
make integration

# Lint code
make lint

# Format code
make format

# Build for all platforms
make docker-mcp-cross

# Clean build artifacts
make clean
```

## 🌐 Portal Quick Start

### 1. Environment Setup

```bash
# Setup unified environment configuration
cp .env.example .env
# Edit .env with your Azure AD configuration and secrets

# Generate JWT secret (required)
JWT_SECRET=$(openssl rand -base64 64)
echo "Use this JWT secret: $JWT_SECRET"
# Copy this value to JWT_SECRET in your .env file
```

### 2. Production Deployment with Docker (Working Solution)

```bash
# Use the Docker solution (Phase 4 - 90% complete)
./deploy-mcp-portal.sh

# Or manually with docker-compose
docker-compose -f docker-compose.mcp-portal.yml up -d

# Check status
docker-compose -f docker-compose.mcp-portal.yml ps

# View logs
docker-compose -f docker-compose.mcp-portal.yml logs -f
```

**Note**: The deployment files contain the fully tested containerization solution with:

- Fixed hadolint errors in Dockerfile.mcp-portal
- Simplified sitemap configuration with NEXT_PUBLIC_SITE_URL
- Cleaned up obsolete docker/ directory scripts
- Working multi-stage build for Go backend and Next.js frontend

Services will be available at:

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/api/health

### 3. Development Environment (Optional)

```bash
# Start development environment with hot reload
make portal-dev-up

# Or using docker-compose directly for development
docker-compose -f docker-compose.mcp-portal.yml up

# For local development without containers
# Backend (Terminal 1)
docker mcp portal serve

# Frontend (Terminal 2)
cd cmd/docker-mcp/portal/frontend
npm install
npm run dev
```

Development features:

- Hot reload for frontend and backend
- Debug logging enabled
- Development database with verbose logging

### 4. Required Configuration

Before configuring the environment, generate the required cryptographic secrets:

```bash
# Generate JWT Secret (64 bytes, base64 encoded)
JWT_SECRET=$(openssl rand -base64 64)
echo "JWT_SECRET=$JWT_SECRET"

# Save to a temporary file for easy copying
echo "# Generated secret for Portal configuration" > temp_secrets.txt
echo "JWT_SECRET=$JWT_SECRET" >> temp_secrets.txt
echo "# IMPORTANT: Use this SAME value for both frontend and backend!" >> temp_secrets.txt
echo "Secret saved to temp_secrets.txt"
```

**⚠️ CRITICAL: JWT Secret Must Match!**

- **SAME SECRET**: The JWT secret MUST be identical for both:
  - Frontend: `JWT_SECRET` in `.env.local`
  - Backend: `MCP_PORTAL_SECURITY_JWT_SIGNING_KEY` environment variable
- **Why**: The backend signs tokens with this secret, and the frontend verifies them. If they don't match, authentication will fail.
- **Security**: Use a strong secret (minimum 32 characters, recommend 64+ characters)
- **Per Environment**: Use different secrets for dev/staging/prod
- **Never commit this secret to version control!**

**Alternative one-liner command:**

```bash
# For JWT Secret
openssl rand -base64 64
```

**Additional Environment Setup:**

```bash
# Add NEXT_PUBLIC_SITE_URL for sitemap generation
NEXT_PUBLIC_SITE_URL=http://localhost:3000
# Or your production domain
NEXT_PUBLIC_SITE_URL=https://your-domain.com
```

### 5. Configuration Details

The unified `.env` file configures all services:

```bash
# =================================================================
# Key Configuration Sections in .env
# =================================================================

# Azure AD Configuration (REQUIRED)
AZURE_TENANT_ID=your-tenant-id-here
AZURE_CLIENT_ID=your-client-id-here
AZURE_CLIENT_SECRET=your-client-secret-here

# JWT Secret (REQUIRED - Same for frontend & backend)
JWT_SECRET=your-jwt-secret-minimum-32-characters

# API Configuration
API_PORT=8080
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080

# Database
POSTGRES_DB=mcp_portal
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-in-production

# See .env.example for complete configuration options
```

**Key Changes in Docker Infrastructure (2025-09-18):**

- **Working Deployment**: Dockerfile.mcp-portal and docker-compose.mcp-portal.yml solution
- **Cleaned Infrastructure**: Moved obsolete docker/ scripts to TEMP_DEL/
- **Simplified Configuration**: Single .env file with NEXT_PUBLIC_SITE_URL
- **Build Fixes**: All hadolint errors resolved, Tailwind CSS @apply issues fixed
- **Git Cleanup**: Removed generated sitemap/robots.txt files from tracking

## 📁 Project Structure

```
mcp-gateway/
├── cmd/docker-mcp/            # CLI plugin main application
│   ├── commands/              # CLI command implementations
│   ├── internal/              # Internal packages
│   └── portal/                # Portal CLI subcommand & backend
│       ├── auth/              # Authentication system
│       ├── catalog/           # MCP server catalog service
│       ├── config/            # User configuration management
│       ├── docker/            # Docker container lifecycle
│       ├── executor/          # CLI execution framework
│       ├── security/          # Security components
│       ├── server/            # HTTP server & API
│       ├── database/          # Database layer & migrations
│       └── frontend/          # Next.js frontend application
│           ├── app/           # App router pages
│           ├── components/    # React components
│           ├── lib/           # API client & utilities
│           └── hooks/         # React Query hooks
├── docs/                      # CLI documentation
├── implementation-plan/       # Portal planning & architecture
├── examples/                  # Example configurations
├── pkg/                       # Shared packages
└── test/                      # Test utilities
```

## 🛠️ Development Workflows

### CLI Development

```bash
# Build and test cycle
make docker-mcp && make test

# Test specific component
go test ./cmd/docker-mcp/server/...

# Run with verbose logging
docker mcp gateway run --verbose

# Debug specific server
docker mcp server inspect server-name
```

### Portal Development

**Quick Development (Docker Compose):**

```bash
# Start development environment
make portal-dev-up

# View logs
make portal-logs

# Stop services
make portal-down
```

**Local Development (No Containers):**

```bash
# Backend tests
go test ./cmd/docker-mcp/portal/...

# Frontend development
cd cmd/docker-mcp/portal/frontend
npm run dev

# Check API endpoints
curl http://localhost:8080/api/health
```

**Docker Commands:**

```bash
# Production stack
make portal-up           # Start all services
make portal-down         # Stop all services
make portal-logs         # View logs

# Development stack (with debug tools)
make portal-dev-up       # Start with development overrides
make portal-debug        # Start with debug tools (pgAdmin, Redis Commander)

# Build and testing
make portal-build        # Build all images
make portal-test         # Run integration tests
make portal-clean        # Clean all artifacts
```

## 🔧 Common Tasks

### Adding a New API Endpoint

1. **Define the endpoint** in `cmd/docker-mcp/portal/server/server.go`
2. **Create handler** in `cmd/docker-mcp/portal/server/handlers/`
3. **Add service logic** in relevant service (e.g., `catalog/`, `config/`)
4. **Update frontend** API client in `cmd/docker-mcp/portal/frontend/lib/api.ts`
5. **Add React Query hook** in `cmd/docker-mcp/portal/frontend/hooks/`

### Creating a New Frontend Component

1. **Create component** in `cmd/docker-mcp/portal/frontend/components/`
2. **Add TypeScript types** if needed
3. **Integrate with API** using React Query hooks
4. **Add to page** in `cmd/docker-mcp/portal/frontend/app/`
5. **Test component** with `npm run test`

### Database Migrations

```bash
# Check current migrations
cd cmd/docker-mcp/portal/database/migrations
ls -la *.sql

# Apply manually if needed (be careful!)
psql -U postgres -h localhost -d mcp_portal -f 001_initial_schema.sql
```

### Debugging Tips

**CLI Issues:**

```bash
# Enable verbose logging
docker mcp gateway run --verbose --log-calls

# Check Docker containers
docker ps
docker logs <container-name>

# Verify CLI installation
docker mcp version
```

**Portal Issues (Docker Compose):**

```bash
# Check all service logs
make portal-logs

# Check specific service
docker-compose logs backend
docker-compose logs frontend
docker-compose logs postgres

# Check service health
docker-compose ps
curl http://localhost:8080/api/health
curl http://localhost:3000/

# Debug with development tools
make portal-debug  # Includes pgAdmin and Redis Commander
```

**Portal Configuration Issues:**

```bash
# Verify environment file
cat .env

# Check JWT secret consistency
grep JWT_SECRET .env

# Test database connectivity
docker-compose exec postgres psql -U postgres -d mcp_portal -c "SELECT NOW();"

# Test Redis connectivity
docker-compose exec redis redis-cli ping
```

## 🧪 Testing

### CLI Testing

```bash
# Run all tests
make test

# Run specific test suite
go test ./cmd/docker-mcp/server/

# Run integration tests
make integration

# Test with coverage
go test -cover ./...
```

### Portal Testing

```bash
# Run portal integration tests
make portal-test

# Backend unit tests
go test ./cmd/docker-mcp/portal/...

# Frontend tests (local development)
cd cmd/docker-mcp/portal/frontend
npm run test
npm run test:coverage
```

## 🚀 Building for Production

### CLI Build

```bash
# Build for current platform
make docker-mcp

# Build for all platforms
make docker-mcp-cross

# Create release packages
make mcp-package
```

### Portal Build

```bash
# Build all Portal services
make portal-build

# Or build individual components
make docker-portal        # Backend only
make docker-frontend      # Frontend only
make docker-portal-all    # Both services

# Production deployment
docker-compose up -d      # Start production stack

# Production deployment (using standalone Docker Engine)
# Note: Ensure your user has Docker socket access:
sudo usermod -aG docker $USER  # Then log out/in for group changes
docker-compose -f docker-compose.mcp-portal.yml up -d
```

## 📚 Next Steps

### CLI Development

- Read [MCP Gateway Documentation](./docs/mcp-gateway.md)
- Explore [Message Flow Diagrams](./docs/message-flow.md)
- Check [Catalog Guide](./docs/catalog.md)

### Portal Development

- Review [Implementation Plan](./implementation-plan/README.md)
- Study [Technical Architecture](./implementation-plan/03-architecture/technical-architecture.md)
- Read [Security Framework](./docs/security.md)

### Getting Help

- Check [Troubleshooting Guide](./docs/troubleshooting.md)
- Browse [Implementation Guides](./implementation-plan/04-guides/)
- Ask questions in [GitHub Discussions](https://github.com/jrmatherly/mcp-hub-gateway/discussions)

## 🎯 Quick Reference Commands

```bash
# Essential CLI commands
docker mcp --help                    # Show all commands
docker mcp server list               # List enabled servers
docker mcp gateway run               # Start gateway
docker mcp tools list                # List available tools

# Essential Portal commands
make portal-up                       # Start production stack
make portal-dev-up                   # Start development stack
make portal-logs                     # View all service logs

# Build commands
make docker-mcp                     # Build CLI
make test                           # Run tests
npm run build                       # Build frontend (from frontend dir)

# Development commands
make lint                           # Lint code
make portal-dev-up                  # Start development environment
make portal-debug                   # Start with debug tools
```

---

**Welcome to MCP Gateway & Portal development!** 🎉

For detailed information, check the main [README.md](./README.md) and [implementation plan](./implementation-plan/README.md).
