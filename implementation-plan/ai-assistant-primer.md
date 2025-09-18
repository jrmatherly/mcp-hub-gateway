# MCP Portal - AI Assistant Primer

**Audience**: AI coding assistants  
**Last Updated**: September 18, 2025  
**Prerequisites**: Basic knowledge of web development and Docker  
**Time to Read**: 10 minutes

---

[Home](../README.md) > [Implementation Plan](./README.md) > **AI Assistant Primer**

## Executive Summary (1-minute read)

**MCP Portal** is a new web-based management interface that wraps the existing, mature **MCP Gateway CLI** (Docker plugin). The portal provides Azure AD authentication and PostgreSQL persistence on top of the CLI, creating a secure multi-user web application for managing Model Context Protocol (MCP) servers.

### Key Context

- **NEW**: Portal web UI (Next.js frontend + Go backend)
- **EXISTING**: MCP Gateway CLI (mature, production-ready Docker plugin)
- **ARCHITECTURE**: Portal executes CLI commands and parses output for web interface
- **DEPLOYMENT**: Works WITHOUT Docker Desktop (standalone Docker Engine)
- **AUTHENTICATION**: Azure EntraID with JWT tokens
- **PERSISTENCE**: PostgreSQL with Row-Level Security (RLS)
- **REAL-TIME**: WebSocket/SSE for streaming CLI output to web clients

### Project Status

- **Phase**: Phase 4 DEPLOYMENT (95% complete - Containerization working)
- **Timeline**: On track - Phase 1, 2 & 3 COMPLETE, Phase 4 containerization working
- **Current Status**: Backend fully operational, Frontend 100% implemented, Docker deployment solution working
- **Codebase**: ~40,000+ lines total (25,000 Go backend + 15,000 TypeScript/React frontend)
- **Critical Gap**: Testing infrastructure must expand to 50%+ before production deployment
- **Recent Progress**: Completed Docker containerization with working deployment solution
- **Infrastructure**: Simplified deployment with working Docker files and consolidated configuration
- **Deployment**: Working Docker solution with Dockerfile.mcp-portal and docker-compose.mcp-portal.yml

### Critical Deployment Gaps - Catalog System (Discovered 2025-09-18)

**IMPORTANT**: Analysis revealed the Portal's catalog functionality will fail in containers without fixes:

1. **Missing Volume Mounts**: Catalog directory (`~/.docker/mcp/`) not persisted
2. **CLI Plugin Not Installed**: Binary exists but not in Docker plugin location
3. **HOME Directory Issue**: CLI expects `~/.docker/mcp` but container needs explicit HOME
4. **No Catalog Environment Variables**: Missing feature flags and configuration

**Required Fixes** (See reports/CATALOG_DEPLOYMENT_ANALYSIS_2025-09-18.md):
- Add volume mount: `mcp-catalog:/home/portal/.docker/mcp`
- Install CLI plugin to `~/.docker/cli-plugins/docker-mcp`
- Set environment: `HOME=/home/portal`
- Enable feature: `MCP_PORTAL_CATALOG_FEATURE_ENABLED=true`

---

## 1. Quick Context (1-minute read)

### What is MCP?

The Model Context Protocol (MCP) is an open protocol that standardizes how AI applications connect to external data sources and tools. MCP servers run in Docker containers and provide tools, resources, and prompts to AI clients.

### What does the MCP Gateway CLI do?

The **existing** CLI (`docker mcp`) manages MCP server lifecycle:

- Lists, enables, disables MCP servers
- Manages server configurations and secrets
- Handles OAuth flows for service connections
- Provides unified interface for AI models to access multiple MCP servers

### What is the Portal?

The **new** Portal is a web UI that:

- Wraps the existing CLI without reimplementing functionality
- Adds Azure AD authentication for multi-user access
- Provides PostgreSQL persistence for user-specific configurations
- Offers real-time updates via WebSocket streaming of CLI output
- Enables bulk operations and improved user experience

---

## 2. Project Architecture Overview

### High-Level Architecture

```
User → Web Browser → Portal Frontend (Next.js)
                         ↓
                    Portal Backend (Go)
                         ↓
              CLI Bridge Service (Security Layer)
                         ↓
                MCP Gateway CLI (Existing)
                         ↓
                Docker Engine + MCP Servers
```

### Core Components

#### Frontend (Next.js)

- **Authentication**: Azure AD integration with MSAL.js
- **UI Framework**: Tailwind CSS + Shadcn/ui components
- **State Management**: React Query + Zustand
- **Real-time**: WebSocket connections for CLI streaming

#### Backend (Go)

- **HTTP Server**: Native Go with Gorilla Mux
- **Authentication**: JWT (RS256) token validation
- **Database**: PostgreSQL 17 with Row-Level Security
- **Cache**: Redis for session and data caching

#### CLI Integration Layer (New - Critical Component)

- **CLI Bridge Service**: Secure command execution wrapper
- **Security Manager**: Command validation and parameter sanitization
- **Output Parser**: Structured parsing of CLI output (JSON, logs, tables)
- **Stream Manager**: Real-time WebSocket events from CLI operations

### Data Flow Example

```
1. User clicks "Enable Server" in web UI
2. Portal validates JWT token
3. CLI Bridge executes: docker mcp server enable serverX --user userY
4. Output Parser converts CLI output to JSON
5. Stream Manager sends real-time updates via WebSocket
6. Database stores operation result for audit
7. Frontend updates UI with server status
```

---

## 3. Key Technical Decisions

### CLI Wrapper Pattern (Not Reimplementation)

**Decision**: Portal wraps existing CLI rather than reimplementing functionality

**Rationale**:

- Leverages mature, tested CLI codebase
- Maintains feature parity automatically
- Reduces implementation risk and time
- Enables consistent behavior across CLI and web interfaces

### Non-Docker Desktop Architecture

**Decision**: Support standalone Docker Engine deployment

**Benefits**:

- Works in server environments without Docker Desktop
- Better for production deployments
- Supports containerized environments (Docker-in-Docker)

### Multi-User Catalog Architecture (New Design 2025-09-18)

**Decision**: File-based catalog management without Docker Desktop dependency

**Architecture Components**:

1. **FileCatalogManager**: YAML-based catalog storage at `/app/data/catalogs/`
2. **Admin Base Catalogs**: Controlled by administrators, inherited by all users
3. **User Augmentation**: Personal customizations layer on top of base catalogs
4. **Catalog Inheritance**: Admin Base → User Personal → Merged Result
5. **Per-User Isolation**: Separate containers with user-specific volumes

**Key Features** (See reports/DOCKER_DESKTOP_INDEPENDENT_DESIGN_2025-09-18.md):
- No Docker Desktop secrets API required (uses environment variables)
- File-based persistence with PostgreSQL metadata
- Dynamic port allocation (20000-29999 range) for user containers
- User-specific Docker networks (`mcp-net-{user-id}`)
- AES-256-GCM encryption for sensitive user configurations

### Authentication Strategy

- **Primary**: Azure EntraID for enterprise integration
- **Session Management**: JWT tokens with Redis storage
- **Authorization**: Row-Level Security in PostgreSQL

---

## 4. CLI Integration Strategy

### Command Mapping Pattern

Every Portal API endpoint maps to specific CLI commands:

| Portal Action | API Endpoint                                | CLI Command                                          | Timeout | Stream |
| ------------- | ------------------------------------------- | ---------------------------------------------------- | ------- | ------ |
| List Servers  | `GET /api/v1/servers`                       | `docker mcp server list --format json --user {user}` | 5s      | No     |
| Enable Server | `POST /api/v1/servers/{id}/enable`          | `docker mcp server enable {id} --user {user}`        | 30s     | Yes    |
| View Logs     | `GET /api/v1/servers/{id}/logs?follow=true` | `docker mcp server logs {id} --follow --user {user}` | ∞       | Yes    |

### Security Framework

```go
type SecurityManager struct {
    allowedCommands map[string]*CommandSpec // Whitelist validation
    validator       *InputValidator         // Parameter sanitization
    resourceLimits  *ResourceLimits         // Execution constraints
    auditLogger     *AuditLogger           // Security event logging
}
```

### Output Parsing

- **JSON Parser**: For structured CLI output (`--format json`)
- **Table Parser**: For tabular CLI output with regex patterns
- **Log Parser**: For streaming log output with timestamp extraction
- **Progress Parser**: For long-running operations with progress indicators

---

## 5. Development Workflow

### Project Structure (Phase 1: 100% Complete, Phase 2: 100% Complete, Phase 3: 100% Complete)

```
cmd/docker-mcp/portal/          # Portal CLI subcommand (integrated)
├── server/                     # HTTP server infrastructure ✅
│   ├── server.go              # Main HTTP server (842 lines)
│   ├── handlers/              # Response utilities ✅
│   │   └── response.go        # (306 lines)
│   └── middleware/            # Complete middleware stack ✅
│       └── middleware.go      # (369 lines)
├── auth/                      # Authentication system ✅
│   ├── azure.go              # Azure AD OAuth2 (424 lines)
│   ├── jwt.go                # JWT token processing
│   ├── jwks.go               # JWT key management
│   ├── session_impl.go       # Redis sessions (274 lines)
│   └── types.go              # Auth type definitions
├── executor/                  # CLI execution framework ✅
│   ├── executor.go           # Secure command execution
│   ├── mock.go               # Testing framework
│   ├── executor_test.go      # Comprehensive tests
│   └── types.go              # Type definitions (316 lines)
├── security/                  # Security components ✅
│   ├── audit/                # Audit logging service
│   │   ├── audit.go          # (233 lines)
│   │   └── mock.go
│   ├── ratelimit/            # Rate limiting service
│   │   └── ratelimit.go      # (437 lines)
│   └── crypto/               # Encryption service
│       └── encryption.go     # AES-256-GCM (523 lines)
├── config/                   # Configuration management ✅
├── cache/                    # Redis cache layer ✅
└── database/                 # Database layer ✅
    └── migrations/           # Database migrations
        └── 002_enable_rls_security.sql # (406 lines)

cmd/docker-mcp/portal/frontend/ # Next.js app (75% complete)
├── app/                       # App router pages
├── components/                # React components
│   ├── dashboard/            # ServerCard, ServerList, ServerGrid
│   └── ui/                   # Shadcn/ui components
├── lib/                      # Utilities and API client
└── hooks/                    # React Query hooks
```

### Key Implementation Patterns

**Constructor Naming Convention**: All services use `Create*` pattern:

- `CreateAzureADService()` NOT `NewAzureADService()`
- `CreateJWTService()` NOT `NewJWTService()`
- `CreateExecutor()` NOT `NewExecutor()`

**Interface Adapter Pattern**: Used for incompatible interfaces:

- `AuditLoggerAdapter` wraps `audit.Logger` for executor compatibility
- `RateLimiterAdapter` wraps `ratelimit.RateLimiter` for middleware

**Security-First Design**:

- Command whitelisting with `CommandType` enums
- Input sanitization with regex validation
- Rate limiting on all endpoints
- Comprehensive audit logging

### Local Development Setup

```bash
# 1. Setup unified environment configuration
cp .env.example .env
# Edit .env with your Azure AD and database configuration

# 2. Start development environment
make portal-dev-up

# Or using docker-compose directly
docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up

# 3. Access portal at http://localhost:3000
# Backend API available at http://localhost:8080
```

### Testing Strategy

- **Unit Tests**: Individual components (CLI bridge, parsers, handlers)
- **Integration Tests**: CLI command execution with real containers
- **Security Tests**: Command injection prevention
- **E2E Tests**: Complete user workflows

---

## 6. Common Tasks & Patterns

### Adding a New API Endpoint

1. **Define CLI Command**: Identify existing CLI command to wrap
2. **Create Handler**: Add endpoint in appropriate handler file
3. **Add Security**: Update command whitelist in security manager
4. **Add Parser**: Create parser if CLI output format is new
5. **Add Tests**: Unit + integration tests for the endpoint
6. **Update Frontend**: Add UI components and API calls

### CLI Integration Checklist

```go
// 1. Define command specification
cmdSpec := &CommandSpec{
    Name: "server_enable",
    Args: []ArgumentSpec{
        {Name: "server_id", Type: STRING, Required: true},
        {Name: "user_id", Type: UUID, Required: true},
    },
    MaxRuntime: 30 * time.Second,
    RequiredPermissions: []Permission{MANAGE_SERVERS},
}

// 2. Implement parser for output format
type ServerEnableParser struct{}
func (p *ServerEnableParser) Parse(output []byte) (*ParsedResult, error) {
    // Parse CLI JSON output
}

// 3. Add streaming if needed
if cmdSpec.SupportsStreaming() {
    streamID := mgr.StartStream(userID, cmd)
    // Send stream updates via WebSocket
}
```

### Error Handling Pattern

```go
// CLI errors map to HTTP status codes
func (h *ServerHandler) mapCLIError(err *CLIError) (int, error) {
    switch err.Type {
    case CLIErrorNotFound:
        return http.StatusNotFound, err
    case CLIErrorPermission:
        return http.StatusForbidden, err
    case CLIErrorTimeout:
        return http.StatusRequestTimeout, err
    default:
        return http.StatusInternalServerError, err
    }
}
```

---

## 7. Testing Approach

### Test Pyramid

```
E2E Tests (Playwright)
    ↑
Integration Tests (Real CLI + Containers)
    ↑
Unit Tests (Mocked CLI + Components)
```

### CLI Integration Testing

```go
func TestServerEnable(t *testing.T) {
    // Setup test environment with real CLI
    bridge := setupCLIBridge(t)

    // Execute command through bridge
    result, err := bridge.Execute(&Command{
        Name: "server_enable",
        Args: map[string]interface{}{
            "server_id": "test-server",
            "user_id": "test-user",
        },
    })

    // Verify parsing and output
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Contains(t, result.Data, "enabled")
}
```

### Security Testing

- **Command Injection**: Verify parameter sanitization prevents injection
- **Authorization**: Test user isolation and permission checking
- **Input Validation**: Verify all inputs are properly validated
- **Audit Trail**: Ensure all operations are logged

---

## 8. Deployment Strategy

### Container Architecture

**Recent Infrastructure Changes (2025-09-18):**

- **Docker Directory Cleanup**: Moved obsolete docker/ scripts to TEMP_DEL/
- **Working Containerization**: Dockerfile.mcp-portal and docker-compose.mcp-portal.yml solution complete
- **Simplified Configuration**: Single .env file with NEXT_PUBLIC_SITE_URL for sitemap generation
- **Build Fixes**: Resolved all hadolint errors, Tailwind CSS @apply issues, ESLint Next.js 15 config
- **Git Tracking**: Removed generated files (sitemap, robots.txt) from tracking

```dockerfile
# Portal includes both web service and CLI binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-cli ./cmd/docker-mcp
RUN go build -o portal ./cmd/docker-mcp/portal

FROM alpine:latest
RUN apk --no-cache add ca-certificates docker-cli
COPY --from=builder /app/mcp-cli ./bin/
COPY --from=builder /app/portal ./bin/
ENV PATH="/root/bin:${PATH}"
CMD ["./bin/portal"]
```

### Production Deployment (Without Docker Desktop)

```bash
# Simple production deployment with unified configuration
cp .env.example .env
# Edit .env with your configuration

# Start all services
docker-compose up -d

# Services available at:
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

### Security Considerations

- **Socket Access**: Portal needs Docker socket access for CLI operations
- **User Permissions**: Run with docker group membership
- **Network Isolation**: Separate network for MCP services
- **Secret Management**: Environment-based configuration

---

## 9. Links to Detailed Documentation

### Implementation Documents

- **[Technical Architecture](./03-architecture/technical-architecture.md)**: Complete system design
- **[CLI Integration Architecture](./03-architecture/cli-integration-architecture.md)**: CLI wrapper implementation details
- **[API Specification](./03-architecture/api-specification.md)**: REST endpoints with CLI command mapping
- **[CLI Command Mapping](./03-architecture/cli-command-mapping.md)**: Detailed command-to-endpoint mapping
- **[Database Schema](./03-architecture/database-schema.md)**: PostgreSQL tables and RLS policies

### Phase Documentation

- **[Phase 1: Foundation](./02-phases/phase-1-foundation.md)**: Infrastructure setup (Weeks 1-2)
- **[Phase 2: Core Features](./02-phases/phase-2-core-features.md)**: Backend implementation (Weeks 3-4)
- **[Phase 3: Frontend](./02-phases/phase-3-frontend.md)**: UI development (Weeks 5-6)
- **[Phase 4: Deployment](./02-phases/phase-4-deployment.md)**: Production preparation (Weeks 7-8)

### Operational Documentation

- **[Development Setup](./04-guides/development-setup.md)**: Local development environment
- **[Testing Plan](./04-guides/testing-plan.md)**: Test strategy and implementation
- **[Deployment Guide](./04-guides/deployment-guide.md)**: Production deployment
- **[Deployment Without Docker Desktop](./04-guides/deployment-without-docker-desktop.md)**: Standalone deployment

### Existing MCP Documentation

- **[MCP Gateway](../docs/mcp-gateway.md)**: CLI usage and capabilities
- **[Security](../docs/security.md)**: MCP security model
- **[Examples](../examples/README.md)**: Usage examples and patterns

---

## 10. Glossary of Terms

| Term                    | Definition                                                      |
| ----------------------- | --------------------------------------------------------------- |
| **MCP**                 | Model Context Protocol - open standard for AI tool connectivity |
| **MCP Server**          | Containerized service providing tools/resources to AI models    |
| **MCP Gateway CLI**     | Docker plugin (`docker mcp`) for managing MCP servers           |
| **Portal**              | New web interface wrapping the existing CLI                     |
| **CLI Bridge Service**  | Security layer that executes CLI commands for web interface     |
| **Output Parser**       | Component that converts CLI output to structured JSON           |
| **Stream Manager**      | Component handling real-time WebSocket updates                  |
| **RLS**                 | Row-Level Security in PostgreSQL for user data isolation        |
| **Azure EntraID**       | Microsoft's identity platform (formerly Azure AD)               |
| **Container Lifecycle** | Docker container start/stop/restart operations                  |

### Architecture Terms

- **Wrapper Pattern**: Portal wraps CLI without reimplementing functionality
- **Command Mapping**: Direct mapping of web actions to CLI commands
- **Security Manager**: Component validating commands before execution
- **Audit Trail**: Complete logging of all CLI operations for security

### Development Terms

- **Phase-based Implementation**: 8-week project split into 4 phases
- **CLI-First Approach**: Web interface functionality derives from CLI capabilities
- **Streaming Operations**: Long-running CLI commands with real-time progress updates
- **User Isolation**: Each user has separate CLI context and data access

---

## 11. Multi-User Catalog System Roadmap (New - 2025-09-18)

### Implementation Plan Overview

**Timeline**: 8-week phased implementation
**Priority**: Critical for production multi-tenant deployment
**Design Approach**: Docker Desktop-independent, file-based catalogs

### Phase Breakdown

#### Phase 1: Core Infrastructure (Weeks 1-2)
- Fix Docker deployment gaps (volume mounts, CLI plugin, environment)
- Implement FileCatalogManager for YAML-based storage
- Create database schema extensions for catalog metadata
- Build catalog inheritance engine

#### Phase 2: Catalog Management (Weeks 3-4)
- Admin interface for base catalog management
- User customization UI for personal catalogs
- Import/export functionality
- Catalog validation and testing framework

#### Phase 3: User Isolation (Weeks 5-6)
- UserOrchestrator for container lifecycle management
- PortManager for dynamic port allocation (20000-29999)
- Per-user Docker network creation
- Resource limits and quota enforcement

#### Phase 4: Production Hardening (Weeks 7-8)
- Performance optimization and caching
- Security audit and penetration testing
- Monitoring and observability integration
- Documentation and deployment guides

### Key Implementation Files

**Backend Components** (To be created):
```
cmd/docker-mcp/portal/catalog/
├── file_manager.go      # FileCatalogManager implementation
├── inheritance.go       # Catalog inheritance engine
├── user_orchestrator.go # User container management
├── port_manager.go      # Dynamic port allocation
└── merge_strategy.go    # Catalog merging logic
```

**Database Migrations** (To be created):
```sql
-- 003_catalog_multi_user.sql
CREATE TABLE catalog_configs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    catalog_type VARCHAR(50), -- 'admin_base', 'user_personal'
    is_enabled BOOLEAN DEFAULT true,
    precedence INTEGER DEFAULT 100,
    config_data JSONB
);
```

### Critical Success Factors

1. **No Docker Desktop Dependencies**: All features work with standard Docker Engine
2. **User Isolation**: Complete data and network separation between users
3. **Performance**: Catalog resolution <100ms, cache hit ratio >90%
4. **Security**: No cross-user data access, encrypted sensitive data
5. **Scalability**: Support 100+ concurrent users

### References

- **Analysis Reports**: `/reports/CATALOG_DEPLOYMENT_ANALYSIS_2025-09-18.md`
- **Implementation Plan**: `/reports/MULTI_USER_CATALOG_IMPLEMENTATION_PLAN_2025-09-18.md`
- **Technical Design**: `/reports/DOCKER_DESKTOP_INDEPENDENT_DESIGN_2025-09-18.md`
- **Feature Specs**: `/docs/feature-specs/` directory

## Getting Started Checklist

When beginning work on MCP Portal:

### Understanding Phase

- [ ] Read this primer document completely
- [ ] Review [Technical Architecture](./03-architecture/technical-architecture.md)
- [ ] Scan [CLI Integration Architecture](./03-architecture/cli-integration-architecture.md)
- [ ] Check current [Project Status](./01-planning/project-tracker.md)

### Setup Phase

- [ ] Follow [Development Setup Guide](./04-guides/development-setup.md)
- [ ] Verify CLI functionality: `docker mcp --help`
- [ ] Test existing CLI commands: `docker mcp server list`
- [ ] Set up Azure AD development environment

### Implementation Phase

- [ ] Review current phase tasks (likely [Phase 1](./02-phases/phase-1-foundation.md))
- [ ] Understand CLI command mapping for your area
- [ ] Set up test environment with real MCP servers
- [ ] Begin with CLI Bridge Service if working on integration layer

### Quality Phase

- [ ] Write tests that include CLI integration
- [ ] Verify security: no command injection possible
- [ ] Test with real Docker containers
- [ ] Ensure audit logging captures all operations

Remember: **The CLI is mature and works**. The Portal's job is to provide a secure, user-friendly web interface to existing CLI functionality, not to reimplement the core MCP server management logic.
