# AGENTS.md

This file provides guidance to AI coding assistants working in this repository.

**Note:** CLAUDE.md, .clinerules, .cursorrules, .windsurfrules, and other AI config files are symlinks to AGENTS.md in this project.

# MCP Gateway & Portal

## Project Overview

This repository contains **TWO distinct but related projects**:

1. **MCP Gateway CLI** (EXISTING) - A mature Docker CLI plugin for managing Model Context Protocol (MCP) servers

   - Location: Main repository code
   - Language: Go 1.24+
   - Status: Production-ready
   - Can run with or without Docker Desktop

2. **MCP Portal** (NEW) - A web UI that wraps the CLI for enhanced user experience
   - Location: `/cmd/docker-mcp/portal/` (integrated CLI subcommand)
   - Frontend: Next.js + TypeScript (Phase 3 - 100% complete)
   - Backend: Go service that executes CLI commands
   - Documentation/Planning: `/implementation-plan/`
   - Status: ~91% complete - Phase 1 Complete (100%), Phase 2 Complete (100%), Phase 3 Complete (100%), Phase 4 Production Readiness (91%)

**CRITICAL**: The Portal is NOT reimplementing the CLI - it wraps and executes CLI commands, parsing their output for the web interface.

## Build & Commands

### Core Development Commands

```bash
# Build the Docker MCP CLI plugin
make docker-mcp                    # Build and install to ~/.docker/cli-plugins/

# Testing
make test                           # Run all tests
make integration                    # Run integration tests only
go test -count=1 ./... -run TestIntegration  # Integration tests with no caching

# Linting and Formatting
make lint                           # Run linters for all platforms
make lint-linux                     # Platform-specific linting
make format                         # Format code using Docker buildx

# Documentation
make docs                           # Generate CLI reference documentation

# Cross-platform Building
make docker-mcp-cross              # Build for all platforms
make docker-mcp-linux              # Linux only (amd64 + arm64)
make docker-mcp-darwin             # macOS only (amd64 + arm64)
make docker-mcp-windows            # Windows only (amd64 + arm64)

# Package and Release
make mcp-package                   # Create release tarballs

# Docker Images
make push-mcp-gateway              # Push gateway images
make push-module-image TAG=v1.0.0  # Push module image with tag

# Clean
make clean                         # Remove build artifacts and CLI plugin
```

### Running the MCP Gateway

```bash
# Run gateway with stdio transport (default)
docker mcp gateway run

# Run with streaming transport on port 8080
docker mcp gateway run --port 8080 --transport streaming

# Run with specific servers
docker mcp gateway run --servers server1,server2

# Run with verbose logging
docker mcp gateway run --verbose --log-calls

# Run in watch mode (auto-reload on config changes)
docker mcp gateway run --watch
```

### Server Management Commands

```bash
# Server operations
docker mcp server list             # List enabled servers
docker mcp server enable <name>    # Enable a server
docker mcp server disable <name>   # Disable a server
docker mcp server inspect <name>   # Get server details

# Catalog operations
docker mcp catalog init            # Initialize default catalog
docker mcp catalog ls              # List available catalogs
docker mcp catalog show <name>     # Show catalog servers

# Configuration
docker mcp config read             # Read configuration
docker mcp config write            # Write configuration

# Secrets management
docker mcp secret set <key>        # Set a secret
docker mcp secret rm <key>         # Remove a secret
docker mcp secret ls               # List secrets
```

## Code Style

### Go Code Conventions

#### Import Organization

```go
import (
    // Standard library
    "context"
    "encoding/json"
    "fmt"

    // Third-party packages
    "github.com/spf13/cobra"

    // Internal packages
    "github.com/docker/mcp-gateway/cmd/docker-mcp/internal/config"
    "github.com/docker/mcp-gateway/cmd/docker-mcp/internal/docker"
)
```

#### Naming Conventions

- **Packages**: Lowercase, single word preferred (`server`, `config`, `docker`)
- **Interfaces**: Verb + "er" suffix (`Client`, `Parser`, `Executor`)
- **Exported Types**: CamelCase (`CommandExecutor`, `OutputParser`)
- **Functions**: CamelCase for exported, camelCase for unexported
- **Constants**: CamelCase or UPPER_SNAKE_CASE for environment variables

#### Error Handling

```go
// Always check and return errors
list, err := server.List(ctx, docker)
if err != nil {
    return fmt.Errorf("failed to list servers: %w", err)
}

// Use errors.Is/As for error checking
if errors.Is(err, context.Canceled) {
    return nil
}

// Wrap errors with context
return fmt.Errorf("parsing output: %w", err)
```

#### Context Usage

- Always accept `context.Context` as first parameter
- Pass context through the entire call chain
- Use context for cancellation and timeouts

#### Testing Conventions

- Test files end with `_test.go`
- Test functions start with `Test`
- Use table-driven tests where appropriate
- Integration tests use `TestIntegration` prefix

### TypeScript/JavaScript Conventions (Portal)

#### Import Organization

```typescript
// React and external libraries
import React from "react";
import { useQuery } from "@tanstack/react-query";

// Internal components
import { ServerCard } from "@/components/ServerCard";

// Types and interfaces
import type { Server, Config } from "@/types";

// Utilities
import { parseCliOutput } from "@/utils/cli-parser";
```

#### Component Structure

- Use functional components with hooks
- Prefer TypeScript for type safety
- Use Tailwind CSS for styling
- Keep components focused and single-purpose

## Testing

### Testing Philosophy

**When tests fail, fix the code, not the test.**

Key principles:

- **Tests should be meaningful** - Avoid tests that always pass regardless of behavior
- **Test actual functionality** - Call the functions being tested, don't just check side effects
- **Failing tests are valuable** - They reveal bugs or missing features
- **Fix the root cause** - When a test fails, fix the underlying issue, don't hide the test
- **Test edge cases** - Tests that reveal limitations help improve the code
- **Document test purpose** - Each test should explain what it validates

### Running Tests

```bash
# Run all tests
make test

# Run specific test suite
go test ./cmd/docker-mcp/...

# Run with coverage
go test -cover ./...

# Run integration tests
make integration

# Run a single test
go test -run TestServerList ./cmd/docker-mcp/server
```

### Test Organization

- Unit tests: Alongside source files (`server.go` → `server_test.go`)
- Integration tests: Use `TestIntegration` prefix
- Test data: Store in `testdata/` directories
- Mocks: Use interfaces for dependency injection

## Security

### Security Considerations

#### Command Injection Prevention

- **NEVER** pass user input directly to CLI commands
- Always validate and sanitize inputs
- Use parameter whitelists for CLI arguments
- Escape special characters in parameters

#### Authentication & Authorization

- Portal uses Azure EntraID for authentication
- JWT tokens (RS256) for API access
- Row-Level Security in PostgreSQL
- User isolation for multi-tenant operations

#### Secret Management

- Never commit secrets to git
- Use Docker Desktop secrets API when available
- Fallback to `.env` files for local development
- Environment variables for production deployment

#### Container Security

- MCP servers run in isolated containers
- Minimal privileges granted to containers
- Network isolation between servers
- Resource limits enforced

## Directory Structure & File Organization

### Project Structure

```
mcp-gateway/
├── cmd/
│   └── docker-mcp/         # CLI plugin main application
│       ├── commands/       # CLI command implementations
│       ├── internal/       # Internal packages
│       ├── server/         # Server management
│       └── portal/         # Portal CLI subcommand
│           ├── executor/   # CLI execution framework
│           ├── database/   # Database layer and migrations
│           └── frontend/   # Next.js frontend (complete)
├── pkg/                    # Shared packages
├── docs/                   # Documentation
│   ├── generator/         # Documentation generation
│   └── *.md              # User guides
├── examples/              # Example configurations
├── implementation-plan/   # Portal planning documents
│   ├── 01-planning/      # Progress tracking and project management
│   ├── 02-phases/        # Phase-by-phase implementation plans
│   ├── 03-architecture/  # Technical specifications and design
│   ├── 04-guides/        # Development and deployment guides
│   └── ai-assistant-primer.md  # AI context document
├── reports/              # ALL project reports
│   └── *.md             # Various report types
├── test/                # Test utilities and data
├── vendor/              # Vendored dependencies
└── temp/                # Temporary files (gitignored)
```

### Reports Directory

ALL project reports and documentation should be saved to the `reports/` directory:

**Implementation Reports:**

- Phase validation: `PHASE_X_VALIDATION_REPORT.md`
- Implementation summaries: `IMPLEMENTATION_SUMMARY_[FEATURE].md`
- Feature completion: `FEATURE_[NAME]_REPORT.md`

**Testing & Analysis Reports:**

- Test results: `TEST_RESULTS_[DATE].md`
- Coverage reports: `COVERAGE_REPORT_[DATE].md`
- Performance analysis: `PERFORMANCE_ANALYSIS_[SCENARIO].md`
- Security scans: `SECURITY_SCAN_[DATE].md`

**Report Naming Conventions:**

- Use descriptive names: `[TYPE]_[SCOPE]_[DATE].md`
- Include dates: `YYYY-MM-DD` format
- Group with prefixes: `TEST_`, `PERFORMANCE_`, `SECURITY_`
- Markdown format: All reports end in `.md`

### Temporary Files & Debugging

All temporary files should be organized in `/temp` folder:

**Guidelines:**

- Never commit files from `/temp` directory
- Use `/temp` for debugging and analysis scripts
- Clean up `/temp` directory regularly
- Include `/temp/` in `.gitignore`

### Ignore Files Configuration

**`.dockerignore` Best Practices:**

- Uses exclusion-first pattern (`*` then `!` exceptions)
- **CRITICAL**: Never include `.git` directory in Docker context
- **NOTE**: `/docker/scripts` moved to TEMP_DEL/ (obsolete infrastructure)
- Excludes test binaries, logs, and temporary files

**`.gitignore` Comprehensive Coverage:**

- Go artifacts: `*.test`, `*.coverprofile`, `go.work.sum`
- Portal/Next.js: `.next/`, `out/`, `*.local`
- Temporary files: `*.tmp`, `/temp/`, `/tmp/`
- Logs and debug: `*.log`, `debug.test`
- Environment files: `.env`, `*.local`

## Configuration

### Development Environment Setup

#### Prerequisites

- Docker Engine (Docker Desktop optional)
- Go 1.24+
- Make
- Git

#### Environment Variables

```bash
# Docker configuration
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_MCP_PLUGIN_BINARY=docker-mcp

# Development settings (optional)
MCP_LOG_LEVEL=debug
MCP_GATEWAY_PORT=8080
MCP_WATCH_MODE=true
```

#### Local Development

```bash
# Clone repository
git clone https://github.com/jrmatherly/mcp-hub-gateway
cd mcp-hub-gateway

# Build and install CLI plugin
make docker-mcp

# Verify installation
docker mcp version

# Run gateway locally
docker mcp gateway run --verbose
```

### Portal Development

For Portal development, use:

```bash
# Backend development (CLI subcommand structure)
cd cmd/docker-mcp/portal
go test ./...

# Run portal tests
go test ./cmd/docker-mcp/portal/... -v

# Start portal service
docker mcp portal serve

# Database migrations
cd cmd/docker-mcp/portal/database/migrations
# Apply migrations when needed

# Frontend development (complete - Next.js with TypeScript)
cd cmd/docker-mcp/portal/frontend
npm install
npm run dev
```

## CLI Integration for Portal

### Key Integration Points

The Portal wraps the CLI through command execution:

#### Command Mapping (Quick Reference)

| UI Action       | CLI Command                        |
| --------------- | ---------------------------------- |
| List Servers    | `docker mcp server list --json`    |
| Enable Server   | `docker mcp server enable {name}`  |
| Disable Server  | `docker mcp server disable {name}` |
| Get Server Info | `docker mcp server inspect {name}` |

**📋 Complete mapping available**: See `/implementation-plan/03-architecture/cli-command-mapping.md` for full command coverage

#### Output Parsing Strategy

- Always use `--json` flag when available
- Parse structured output for web display
- Stream logs using `--follow` flag
- Handle both success and error outputs

#### Security Considerations

**⚠️ CRITICAL: Command Injection Prevention**

- Validate all parameters before execution
- Use command whitelisting
- Never execute arbitrary user input
- Implement timeout for long-running commands

**🛡️ Security Framework**: See `/docs/security.md` for complete security implementation guide

## Docker Infrastructure Updates (2025-09-18)

### Portal Docker Containerization Solution

**Working Docker Files (Phase 4 - 91% Complete):**

- `Dockerfile.mcp-portal` - Multi-stage build for Go backend and Next.js frontend (hadolint clean)
- `docker-compose.mcp-portal.yml` - Service orchestration with all dependencies
- `deploy-mcp-portal.sh` - Automated deployment script
- `MCP_PORTAL_DEPLOYMENT.md` - Complete deployment documentation

**Build Issues Resolved:**

1. **Architecture Understanding**: Portal is CLI subcommand, not separate services
2. **Environment Variables**: Made Azure AD variables optional in Zod validation
3. **ESLint Configuration**: Fixed for Next.js 15 with flat config
4. **Tailwind CSS v4**: Replaced all @apply directives with direct CSS properties
5. **Next.js Prerendering**: Fixed Client/Server component boundaries
6. **Build Configuration**: Fixed outputFileTracingRoot and experimental configs
7. **Hadolint Compliance**: Fixed all DL3021 errors in Dockerfile
8. **Infrastructure Cleanup**: Moved obsolete docker/ directory to TEMP_DEL/
9. **Sitemap Simplification**: Updated to use NEXT_PUBLIC_SITE_URL environment variable
10. **Git Tracking**: Removed generated files from version control

### Recent Infrastructure Improvements

**Major Infrastructure Cleanup Completed (2025-09-18):**

- **Docker Solution Working**: Dockerfile.mcp-portal and docker-compose.mcp-portal.yml fully functional
- **Infrastructure Cleanup**: Moved obsolete docker/ directory to TEMP_DEL/
  - **Reason**: Scripts referenced non-existent Docker architecture files
  - **Impact**: Simplified project structure and eliminated confusion
- **Environment Configuration**: Single `.env` file with NEXT_PUBLIC_SITE_URL added
- **Build Quality Improvements**:
  - Fixed all hadolint DL3021 errors in Dockerfile.mcp-portal
  - Resolved ESLint flat config issues for Next.js 15
  - Replaced all Tailwind CSS v4 @apply directives with direct CSS
  - Simplified sitemap configuration using environment variable

### Docker Setup

```bash
# Quick Start
cp .env.example .env
# Edit .env with your Azure AD and database configuration
# Add NEXT_PUBLIC_SITE_URL=http://localhost:3000 (or your domain)

# Production (Working Solution)
./deploy-mcp-portal.sh      # Automated deployment
# OR manually:
docker-compose -f docker-compose.mcp-portal.yml up -d
docker-compose -f docker-compose.mcp-portal.yml ps

# View logs
docker-compose -f docker-compose.mcp-portal.yml logs -f

# Services available:
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

**Note**: The old docker/ directory scripts are obsolete and have been moved to TEMP_DEL/. Use the working Docker solution above.

### Environment Configuration

The `.env` file now provides unified configuration for all services:

```bash
# Critical Shared Variables
JWT_SECRET=your-jwt-secret-minimum-32-characters  # Must be identical for frontend & backend
API_PORT=8080                                    # Standardized API port

# Service-specific prefixes
MCP_PORTAL_*                                     # Backend variables
NEXT_PUBLIC_*                                    # Frontend variables
```

## Portal Implementation Status (2025-01-20)

### Current Progress: Phase 1 Complete (100%), Phase 2 Complete (100%), Phase 3 Complete (100%), Phase 4 Production Readiness (91%)

#### ✅ Completed Components (~25,000 lines of enterprise-grade Go code across 50+ files)

1. **Complete HTTP Server Infrastructure** (`/cmd/docker-mcp/portal/server/`)

   - `server.go` - Full HTTP server with Gin framework (842 lines)
   - `handlers/response.go` - Response utilities (306 lines)
   - `middleware/middleware.go` - Complete middleware stack (369 lines)
   - RESTful API endpoints for all CLI operations
   - Adapter patterns for interface compatibility

2. **Complete Authentication System** (`/cmd/docker-mcp/portal/auth/`)

   - `azure.go` - Azure AD OAuth2 integration (424 lines)
   - `jwt.go` - JWT token validation and processing
   - `jwks.go` - JWT key management
   - `session_impl.go` - Redis session management (274 lines)
   - Constructor pattern using `Create*` naming convention

3. **Complete Security Framework** (`/cmd/docker-mcp/portal/security/`)

   - `crypto/encryption.go` - AES-256-GCM encryption (523 lines)
   - `audit/audit.go` - Comprehensive audit logging (300 lines)
   - `ratelimit/ratelimit.go` - Rate limiting (437 lines)
   - Command injection prevention and validation

4. **CLI Executor Framework** (`/cmd/docker-mcp/portal/executor/`)

   - Complete secure command execution with testing framework
   - Command whitelisting, input sanitization, rate limiting
   - Test suite with comprehensive coverage

5. **Database & Configuration**

   - Complete database layer with connection pooling
   - Configuration management system
   - Redis cache implementation
   - RLS security migrations (406 lines)

6. **MCP Server Catalog** (`/cmd/docker-mcp/portal/catalog/`)

   - Full CRUD service with CLI wrapper (2,543 lines)
   - Repository pattern with versioning
   - Custom server support and validation

7. **User Configuration Management** (`/cmd/docker-mcp/portal/config/`)

   - Encrypted storage with AES-256-GCM (2,847 lines)
   - Import/export functionality
   - Bulk operations with merge strategies

8. **Docker Container Lifecycle** (`/cmd/docker-mcp/portal/docker/`)

   - Complete container management (2,180 lines)
   - Health monitoring and resource limits
   - Container cleanup on disable

9. **Server State Management** (`/cmd/docker-mcp/portal/state/`)

   - Redis-based state caching (980 lines)
   - Real-time health monitoring
   - State transition validation

10. **Bulk Operations** (`/cmd/docker-mcp/portal/bulk/`)

    - Batch command execution (1,000+ lines)
    - Progress tracking and rollback
    - Parallel/sequential execution modes

11. **WebSocket/SSE Real-time** (`/cmd/docker-mcp/portal/realtime/`)
    - Connection management (600+ lines)
    - WebSocket and SSE support
    - Channel-based pub/sub

#### ⚠️ Testing Status

**Testing Coverage**: 1,801 lines of test code (11% coverage) vs ~25,000 production lines - **PRIORITY**: Expand to 50%+ for production readiness

- Comprehensive testing required for catalog service (2,543 lines)
- Authentication system needs security validation (698 lines)
- Docker service needs integration testing (2,180 lines)
- User configuration needs coverage (2,847 lines)

#### 📁 Current Portal Structure

```
cmd/docker-mcp/portal/       # Portal CLI subcommand (integrated)
├── auth/                   # Authentication system ✅
│   ├── azure.go           # Azure AD OAuth2 (424 lines)
│   ├── jwt.go             # JWT validation
│   ├── jwks.go            # JWT key management
│   ├── session_impl.go    # Redis sessions (274 lines)
│   └── types.go           # Auth type definitions
├── catalog/                # MCP Server Catalog ✅
│   ├── service.go         # Catalog service (800+ lines)
│   ├── repository.go      # Database operations (1,081 lines)
│   └── types.go           # Catalog types (425 lines)
├── config/                 # User Configuration ✅
│   ├── service.go         # Config service (561 lines)
│   ├── repository.go      # Encrypted storage (514 lines)
│   └── types.go           # Config types
├── docker/                 # Docker Container Lifecycle ✅
│   ├── service.go         # Container management (2,180 lines)
│   └── types.go           # Docker types
├── executor/               # CLI execution framework ✅
│   ├── executor.go        # Secure command execution
│   ├── executor_test.go   # Test suite
│   ├── mock.go            # Mock implementations
│   └── types.go           # Type definitions (316 lines)
├── security/               # Security components ✅
│   ├── audit/             # Audit logging (300 lines)
│   ├── ratelimit/         # Rate limiting (437 lines)
│   └── crypto/            # AES-256-GCM encryption (523 lines)
├── server/                 # HTTP server infrastructure ✅
│   ├── server.go          # Gin HTTP server (842 lines)
│   ├── handlers/          # Response utilities (306 lines)
│   └── middleware/        # Middleware stack (369 lines)
├── database/               # Database layer ✅
│   └── migrations/        # RLS migrations (406 lines)
└── frontend/              # Next.js (Phase 3 - 100% complete)
    ├── app/                # App router pages
    │   ├── dashboard/     # Dashboard with config management
    │   └── admin/         # Admin panel with user management, system monitoring
    ├── components/         # React components
    │   ├── dashboard/     # ServerCard, ConfigEditor, etc.
    │   ├── admin/         # UserManagement, SystemMonitoring
    │   └── ui/            # Shadcn/ui components
    ├── lib/                # API client and utilities
    │   ├── api-client.ts  # Full API client implementation
    │   └── realtime-client.ts # WebSocket/SSE client
    └── hooks/              # React Query hooks
        ├── use-websocket.ts # Real-time WebSocket hooks
        └── api/           # API-specific hooks
```

### Next Implementation Steps (Phase 4 - Deployment & Polish)

**Phase 3 Completed:**

1. ✅ **Real-time Updates** - WebSocket/SSE frontend integration complete
2. ✅ **Configuration Management UI** - Import/export interface complete
3. ✅ **Admin Panel** - User management interface complete

**Phase 4 - Production Readiness (91% Complete):**

✅ **Completed:**

1. **Docker Containerization** - Multi-stage build with hadolint compliance
2. **Environment Configuration** - Unified .env with Azure AD integration
3. **Build Quality** - ESLint, Tailwind CSS v4, Next.js 15 compatibility
4. **Infrastructure Cleanup** - Obsolete docker/ directory moved to TEMP_DEL/
5. **Catalog Test Fixes** - Fixed 78 compilation errors, vendoring issues resolved (2025-01-20)

🔄 **In Progress:**

1. **Test Coverage Expansion** - Target 50%+ for production readiness (Priority #1) - Catalog tests fixed
2. **Performance Optimization** - Frontend code splitting, API caching
3. **Security Audit** - Penetration testing, vulnerability assessment
4. **Monitoring Integration** - Production observability and alerting
5. **Documentation Finalization** - User guides and operational runbooks

## Portal Development References

### Phase-Based Documentation

**🔴 Critical - Must read for Portal work:**

- `/implementation-plan/README.md` - Implementation roadmap and phase overview
- `/implementation-plan/ai-assistant-primer.md` - Complete AI assistant context
- `/docs/security.md` - Security framework and command injection prevention

**🟡 Important - Recommended for Portal work:**

- `/implementation-plan/03-architecture/cli-command-mapping.md` - Complete CLI command mapping
- `/implementation-plan/03-architecture/cli-integration-architecture.md` - CLI integration patterns
- `/implementation-plan/03-architecture/technical-architecture.md` - System design details

**🟢 Optional - Reference as needed:**

- `/implementation-plan/03-architecture/database-schema.md` - PostgreSQL schema and migrations
- `/implementation-plan/04-guides/deployment-guide.md` - Docker deployment configuration
- `/examples/compose-static/README.md` - Docker Compose examples

### When to Reference Documentation

**Phase 1 - Initial Setup:**

- 🔴 Implementation roadmap (`/implementation-plan/README.md`)
- 🔴 AI context primer (`/implementation-plan/ai-assistant-primer.md`)
- 🔴 Security framework (`/docs/security.md`)

**Phase 2 - Backend Development:**

- 🟡 CLI integration patterns (`/implementation-plan/03-architecture/cli-integration-architecture.md`)
- 🟡 Technical architecture (`/implementation-plan/03-architecture/technical-architecture.md`)
- 🟢 Database design (`/implementation-plan/03-architecture/database-schema.md`)

**Phase 3 - Frontend Development:**

- 🟡 API specification (`/implementation-plan/03-architecture/api-specification.md`)
- 🟡 CLI command mapping (`/implementation-plan/03-architecture/cli-command-mapping.md`)
- 🟢 Deployment guide (`/implementation-plan/04-guides/deployment-guide.md`)

**Phase 4 - Testing & Deployment:**

- 🔴 Security validation (`/docs/security.md`)
- 🟢 Deployment guide (`/implementation-plan/04-guides/deployment-guide.md`)
- 🟢 Example configurations (`/examples/`)

## Agent Delegation & Tool Execution

### ⚠️ MANDATORY: Always Delegate to Specialists & Execute in Parallel

**When performing multiple operations, send all tool calls in a single message to execute them concurrently for optimal performance.**

#### Critical: Always Use Parallel Tool Calls

**IMPORTANT: Send all tool calls in a single message to execute them in parallel.**

**These cases MUST use parallel tool calls:**

- Multiple file reads or searches
- Independent grep/glob operations
- Gathering information from different sources
- Running multiple tests or checks

**Performance Impact:** Parallel tool execution is 3-5x faster than sequential calls.

## Contributing

### Pull Request Process

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Make changes following code style guidelines
4. Add tests for new functionality
5. Run `make lint test` before committing
6. Commit with descriptive message
7. Push to branch and create PR

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

Types: feat, fix, docs, style, refactor, test, chore

### Code Review Checklist

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation updated if needed
- [ ] No security vulnerabilities introduced
- [ ] Follows project code style
- [ ] Performance impact considered

## Quick Reference

### Most Common Tasks

```bash
# Build and test
make docker-mcp && make test

# Run gateway
docker mcp gateway run

# Enable a server
docker mcp server enable <name>

# Check logs
docker logs <container-name>

# Update documentation
make docs
```

### Troubleshooting

Common issues:

1. **Permission denied**: Check Docker socket permissions
2. **Server not starting**: Verify Docker is running
3. **Build fails**: Ensure Go 1.24+ is installed
4. **Tests fail**: Run `make clean` and rebuild

For detailed troubleshooting, see `/docs/troubleshooting.md`

## Additional Resources

### Core Documentation

- 🔴 [Implementation Plan](./implementation-plan/README.md) - **START HERE for Portal work**
- 🔴 [AI Assistant Primer](./implementation-plan/ai-assistant-primer.md) - Complete project context
- 🔴 [Security Guide](./docs/security.md) - Security framework and command injection prevention

### Technical References

- 🟡 [MCP Gateway Documentation](./docs/mcp-gateway.md) - Gateway architecture and CLI usage
- 🟡 [CLI Integration Architecture](./implementation-plan/03-architecture/cli-integration-architecture.md) - How Portal wraps CLI
- 🟢 [Examples](./examples/README.md) - Docker Compose and configuration patterns

### Portal-Specific Documentation

- 🟡 [Technical Architecture](./implementation-plan/03-architecture/technical-architecture.md) - System design details
- 🟡 [CLI Command Mapping](./implementation-plan/03-architecture/cli-command-mapping.md) - Complete command mappings
- 🟢 [Database Schema](./implementation-plan/03-architecture/database-schema.md) - PostgreSQL with RLS
- 🟢 [API Specification](./implementation-plan/03-architecture/api-specification.md) - REST endpoints
- 🟢 [Deployment Guide](./implementation-plan/04-guides/deployment-without-docker-desktop.md) - Production setup

## Notes for AI Assistants

### Portal Development Guidance

**Before starting Portal work:**

1. 🔴 **MUST READ**: `/implementation-plan/ai-assistant-primer.md` for complete context
2. 🔴 **SECURITY**: Review `/docs/security.md` for command injection prevention
3. 🔴 **ROADMAP**: Check `/implementation-plan/README.md` for current phase

**Key Portal Development Principles:**

1. **Two Projects**: This repo contains the existing CLI (main code) and Portal plans (`/implementation-plan/`)
2. **Wrapper Pattern**: The Portal wraps the CLI via command execution - it doesn't reimplement functionality
3. **CLI First**: Always check what CLI commands already exist before designing Portal features
4. **Security Critical**: Command injection prevention is paramount - validate ALL parameters
5. **Phase-Based Development**: Follow the 4-phase implementation plan in order
6. **Test Everything**: Never skip or disable tests to make things pass

**When to Reference Documentation:**

- **Starting new feature?** → Check phase documentation first (`/implementation-plan/02-phases/`)
- **CLI integration?** → See `/implementation-plan/03-architecture/cli-command-mapping.md`
- **Security concern?** → Always refer to `/docs/security.md`
- **Database work?** → Review RLS patterns in `/implementation-plan/03-architecture/database-schema.md`
- **Deployment?** → Use `/implementation-plan/04-guides/deployment-without-docker-desktop.md`

**Current Development Priorities (Phase 4 - Production Readiness):**

1. **Test Coverage Expansion** - Top priority for production readiness

   - Target: 50%+ coverage across all portal components
   - Focus: Authentication, catalog, docker, config services
   - Integration tests for CLI command execution

2. **Performance Optimization** - Production performance requirements

   - Frontend: Code splitting, lazy loading, caching strategies
   - Backend: API response optimization, database query tuning
   - Real-time: WebSocket connection pooling, SSE optimization

3. **Security Hardening** - Production security validation

   - Penetration testing of authentication flows
   - Command injection prevention validation
   - Rate limiting and DDoS protection testing

4. **Production Monitoring** - Observability and operational readiness
   - Health check endpoints and metrics
   - Logging aggregation and alerting
   - Performance monitoring and profiling

**Performance Requirements:**

- **Parallel Execution**: Always use parallel tool calls when possible (3-5x faster)
- **CLI Response Time**: Target <200ms for simple commands, <5s for complex operations
- **WebSocket Latency**: Keep streaming latency under 50ms
- **Database Queries**: Optimize for RLS, target <100ms query time
