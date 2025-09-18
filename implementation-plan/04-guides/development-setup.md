# Development Setup Guide

## Overview

Complete guide for setting up the MCP Portal development environment on your local machine.

## System Requirements

### Hardware

- **CPU**: 4+ cores (8 recommended)
- **RAM**: 8GB minimum (16GB recommended)
- **Disk**: 20GB free space
- **OS**: macOS, Linux, or Windows with WSL2

### Software Prerequisites

```bash
# Required
- Go 1.24+
- Node.js 20+ and npm 10+
- Docker Desktop
- Git
- PostgreSQL 17+ (or Docker)
- Redis 8+ (or Docker)

# Recommended
- VS Code or GoLand IDE
- Postman or Insomnia
- TablePlus or DBeaver
- Redis Insight
```

## Initial Setup

### Step 1: Clone Repository

```bash
# Clone the repository
git clone https://github.com/docker/mcp-gateway.git
cd mcp-gateway

# Create feature branch
git checkout -b feature/portal-implementation
```

### Step 2: Install Go Dependencies

```bash
# Download Go modules
go mod download

# Verify installation
go mod verify

# Tidy dependencies
go mod tidy
```

### Step 3: Install Node Dependencies

```bash
# Navigate to portal directory
cd portal

# Install dependencies
npm install

# Verify installation
npm ls

# Return to root
cd ..
```

### Step 4: Environment Configuration

```bash
# Copy example environment file
cp .env.example .env.development

# Edit environment variables
vim .env.development
```

**.env.development**

```bash
# Database
DATABASE_URL=postgresql://postgres:password@localhost:5432/mcp_portal_dev
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=mcp_portal_dev

# Redis
REDIS_URL=redis://localhost:6379/0
REDIS_HOST=localhost
REDIS_PORT=6379

# Azure AD (Development)
AZURE_TENANT_ID=your-dev-tenant-id
AZURE_CLIENT_ID=your-dev-client-id
AZURE_CLIENT_SECRET=your-dev-secret
AZURE_REDIRECT_URI=http://localhost:3000/auth/callback

# Portal
PORTAL_HOST=localhost
PORTAL_PORT=3000
API_PORT=8080
NODE_ENV=development

# Security (Development Keys - DO NOT USE IN PRODUCTION)
JWT_SECRET=dev-jwt-secret-change-in-production
ENCRYPTION_KEY=dev-encryption-key-32-characters
SESSION_SECRET=dev-session-secret

# Docker
DOCKER_HOST=unix:///var/run/docker.sock

# Monitoring (Optional for dev)
SENTRY_DSN=
PROMETHEUS_ENABLED=false
LANGFUSE_PUBLIC_KEY=

# Development
DEBUG=true
LOG_LEVEL=debug
```

## Database Setup

### Option 1: Docker PostgreSQL

```bash
# Start PostgreSQL container
docker run -d \
  --name mcp-postgres-dev \
  -e POSTGRES_DB=mcp_portal_dev \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  -v mcp-postgres-data:/var/lib/postgresql/data \
  postgres:17-alpine

# Wait for database to be ready
docker exec mcp-postgres-dev pg_isready

# Create database
docker exec -it mcp-postgres-dev psql -U postgres -c "CREATE DATABASE mcp_portal_dev;"
```

### Option 2: Local PostgreSQL

```bash
# Create database
createdb mcp_portal_dev

# Verify connection
psql -d mcp_portal_dev -c "SELECT version();"
```

### Run Migrations

```bash
# Navigate to migrations directory
cd cmd/docker-mcp/portal/database/migrations

# Run migrations
migrate -path . -database "$DATABASE_URL" up

# Or using make
make migrate-up
```

## Redis Setup

### Option 1: Docker Redis

```bash
# Start Redis container
docker run -d \
  --name mcp-redis-dev \
  -p 6379:6379 \
  redis:8-alpine

# Verify connection
docker exec -it mcp-redis-dev redis-cli ping
```

### Option 2: Local Redis

```bash
# macOS
brew install redis
brew services start redis

# Linux
sudo apt-get install redis-server
sudo systemctl start redis

# Verify
redis-cli ping
```

## Azure AD Configuration

### Create Development App Registration

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to Azure Active Directory > App registrations
3. Click "New registration"
4. Configure:
   - Name: `MCP Portal Development`
   - Supported account types: Single tenant
   - Redirect URI: `http://localhost:3000/auth/callback`
5. Save Application (client) ID and Directory (tenant) ID
6. Create client secret under "Certificates & secrets"
7. Configure API permissions:
   - Microsoft Graph > User.Read
   - Microsoft Graph > email
   - Microsoft Graph > profile

### Update Local Configuration

```bash
# Update .env.development with Azure AD values
AZURE_TENANT_ID=your-tenant-id-from-azure
AZURE_CLIENT_ID=your-application-id-from-azure
AZURE_CLIENT_SECRET=your-client-secret-from-azure
```

## Building the Application

### Backend Build

```bash
# Build the portal service
go build -o bin/mcp-portal ./cmd/docker-mcp

# Run with specific command
./bin/mcp-portal portal serve

# Or use Make
make docker-mcp
```

### Frontend Build

```bash
# Navigate to portal directory
cd portal

# Development build with hot reload
npm run dev

# Production build
npm run build

# Run production build
npm run start
```

## Running the Application

### Using Docker Compose (Recommended)

```bash
# Start all services
docker-compose -f docker-compose.dev.yml up

# Start in background
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# Stop services
docker-compose -f docker-compose.dev.yml down
```

**docker-compose.dev.yml**

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_DB: mcp_portal_dev
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:8-alpine
    ports:
      - "6379:6379"

  portal-backend:
    build:
      context: .
      dockerfile: Dockerfile.dev
    command: go run ./cmd/docker-mcp portal serve
    volumes:
      - .:/app
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/mcp_portal_dev
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  portal-frontend:
    build:
      context: ./portal
      dockerfile: Dockerfile.dev
    command: npm run dev
    volumes:
      - ./portal:/app
      - /app/node_modules
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost:8080

volumes:
  postgres_data:
```

### Manual Start (Development Mode)

#### Terminal 1: Backend

```bash
# Set environment variables
source .env.development

# Run portal service
go run ./cmd/docker-mcp portal serve

# Or with hot reload (using air)
air -c .air.toml
```

#### Terminal 2: Frontend

```bash
cd portal
npm run dev
```

## Development Workflow

### Code Organization

```
cmd/docker-mcp/portal/
├── server.go           # Main server setup
├── auth/              # Authentication logic
├── handlers/          # HTTP handlers
├── database/          # Database layer
├── docker/            # Docker integration
└── websocket/         # Real-time updates

portal/
├── app/               # Next.js app directory
├── components/        # React components
├── lib/              # Utilities and helpers
├── hooks/            # Custom React hooks
└── types/            # TypeScript types
```

### Making Changes

#### Backend Development

```bash
# Make changes to Go files
vim cmd/docker-mcp/portal/handlers/servers.go

# Run tests
go test ./cmd/docker-mcp/portal/...

# Format code
go fmt ./...

# Lint code
golangci-lint run

# Build and test
go build -o bin/mcp-portal ./cmd/docker-mcp
./bin/mcp-portal portal serve
```

#### Frontend Development

```bash
# Make changes to React components
vim portal/components/ServerCard.tsx

# Run tests
npm test

# Type check
npm run type-check

# Lint code
npm run lint

# Format code
npm run format
```

### Testing

#### Unit Tests

```bash
# Backend tests
go test ./... -v

# Frontend tests
cd portal && npm test

# With coverage
go test ./... -cover
cd portal && npm run test:coverage
```

#### Integration Tests

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./... -tags=integration

# Cleanup
docker-compose -f docker-compose.test.yml down
```

#### E2E Tests

```bash
# Install Playwright
cd portal
npx playwright install

# Run E2E tests
npm run test:e2e

# Run with UI
npm run test:e2e:ui
```

## Development Tools

### VS Code Configuration

**.vscode/settings.json**

```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[typescriptreact]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

### Recommended Extensions

```json
{
  "recommendations": [
    "golang.go",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss",
    "prisma.prisma",
    "mtxr.sqltools",
    "mtxr.sqltools-driver-pg",
    "cweijan.vscode-redis-client"
  ]
}
```

### Git Hooks (using Husky)

```bash
# Install Husky
cd portal
npm install --save-dev husky
npx husky install

# Add pre-commit hook
npx husky add .husky/pre-commit "npm run lint && npm run type-check"
```

## Debugging

### Backend Debugging

#### VS Code Launch Configuration

**.vscode/launch.json**

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Portal",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/docker-mcp",
      "args": ["portal", "serve"],
      "env": {
        "DATABASE_URL": "postgresql://postgres:password@localhost:5432/mcp_portal_dev",
        "REDIS_URL": "redis://localhost:6379"
      }
    }
  ]
}
```

#### Delve Debugging

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Start debugging
dlv debug ./cmd/docker-mcp -- portal serve

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Frontend Debugging

#### Chrome DevTools

1. Start development server: `npm run dev`
2. Open Chrome DevTools (F12)
3. Go to Sources tab
4. Set breakpoints in TypeScript files

#### VS Code Debugging

**.vscode/launch.json**

```json
{
  "configurations": [
    {
      "name": "Debug Next.js",
      "type": "node",
      "request": "launch",
      "runtimeExecutable": "npm",
      "runtimeArgs": ["run", "dev"],
      "skipFiles": ["<node_internals>/**"],
      "cwd": "${workspaceFolder}/portal"
    }
  ]
}
```

## Common Issues

### Port Already in Use

```bash
# Find process using port
lsof -i :3000
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql -h localhost -U postgres -d mcp_portal_dev

# Check logs
docker logs mcp-postgres-dev
```

### Node Module Issues

```bash
# Clear cache and reinstall
cd portal
rm -rf node_modules package-lock.json
npm cache clean --force
npm install
```

### Docker Issues

```bash
# Reset Docker
docker system prune -a
docker volume prune

# Rebuild without cache
docker-compose build --no-cache
```

## Useful Commands

### Database Management

```bash
# Connect to database
psql -d mcp_portal_dev

# Run specific migration
migrate -path ./migrations -database "$DATABASE_URL" up 1

# Rollback migration
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Create new migration
migrate create -ext sql -dir ./migrations -seq create_users_table
```

### Docker Management

```bash
# View running containers
docker ps

# View logs
docker logs -f mcp-portal

# Execute command in container
docker exec -it mcp-portal bash

# Clean everything
docker-compose down -v --remove-orphans
```

### Development Scripts

```bash
# Create Makefile targets
make dev        # Start development environment
make test       # Run all tests
make lint       # Run linters
make build      # Build application
make clean      # Clean build artifacts
```

## Resources

### Documentation

- [Go Documentation](https://golang.org/doc/)
- [Next.js Documentation](https://nextjs.org/docs)
- [Docker Documentation](https://docs.docker.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Azure AD Documentation](https://docs.microsoft.com/en-us/azure/active-directory/)

### Tutorials

- [Building Go Web Applications](https://golang.org/doc/articles/wiki/)
- [Next.js Tutorial](https://nextjs.org/learn)
- [Docker for Developers](https://docs.docker.com/get-started/)

### Community

- Project Slack: #mcp-portal-dev
- GitHub Issues: https://github.com/docker/mcp-gateway/issues
- Stack Overflow: [mcp-portal] tag
