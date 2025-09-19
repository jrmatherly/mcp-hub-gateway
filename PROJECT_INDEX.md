# MCP Gateway & Portal - Project Index

**Generated**: September 19, 2025
**Project Status**: ~80% Complete (Build System Blocked)
**Repository**: github.com/jrmatherly/mcp-hub-gateway

## 🚨 Critical Status Summary

### Current Blockers

1. **Go Module Dependencies**: Vendor directory inconsistencies preventing compilation
2. **Test Compilation**: 78+ test failures blocking validation
3. **Uncommitted Work**: 8 files in portal/features/ need review
4. **Frontend Build**: Next.js import violations

### Phase Status

| Phase                  | Progress | Status         | Key Issue                       |
| ---------------------- | -------- | -------------- | ------------------------------- |
| Phase 1: Foundation    | 100%     | ✅ Complete    | None                            |
| Phase 2: Core Features | 100%     | ✅ Complete    | None                            |
| Phase 3: Frontend      | 100%     | ✅ Complete    | None                            |
| Phase 4: Deployment    | 60%      | 🔴 BLOCKED     | Build system failures           |
| Phase 5: OAuth         | 80%      | 🟡 Implemented | Cannot test due to build issues |

## 📁 Project Structure

### Core Components

```
mcp-gateway/
├── cmd/docker-mcp/           # CLI Plugin & Gateway
│   ├── main.go               # Entry point for Docker CLI plugin
│   ├── commands/             # CLI command implementations
│   │   ├── catalog/          # Catalog management commands
│   │   ├── server/           # Server lifecycle commands
│   │   ├── gateway/          # Gateway runtime commands
│   │   └── tools/            # Tool discovery commands
│   │
│   └── portal/               # Portal Web Application (NEW)
│       ├── server/           # HTTP API server
│       ├── auth/             # Azure AD authentication
│       ├── catalog/          # MCP server catalog
│       ├── config/           # User configurations
│       ├── docker/           # Container lifecycle
│       ├── features/         # ⚠️ OAuth implementation (uncommitted)
│       ├── security/         # Encryption, audit, rate limiting
│       ├── database/         # PostgreSQL with RLS
│       └── frontend/         # Next.js application
```

### Documentation Structure

```
├── docs/                     # CLI documentation
│   ├── mcp-gateway.md        # Gateway guide
│   ├── catalog.md            # Catalog management
│   ├── message-flow.md       # Architecture diagrams
│   └── troubleshooting.md    # Common issues
│
├── implementation-plan/      # Portal planning & tracking
│   ├── 01-planning/          # Project tracking
│   │   └── project-tracker.md # Real-time progress
│   ├── 02-phases/            # Phase documentation
│   │   ├── phase-1-foundation.md
│   │   ├── phase-2-core-features.md
│   │   ├── phase-3-frontend.md
│   │   ├── phase-4-deployment.md
│   │   └── phase-5-oauth-authentication.md
│   ├── 03-architecture/      # Technical design
│   │   ├── technical-architecture.md
│   │   ├── api-specification.md
│   │   └── database-schema.md
│   ├── 04-guides/            # Development guides
│   └── ai-assistant-primer.md # AI context document
│
└── reports/                  # Analysis reports
    ├── PRODUCTION_DEPLOYMENT_COMPLETE.md
    └── DOCKER_DESKTOP_INDEPENDENT_DESIGN_2025-09-18.md
```

## 🏗️ Architecture Overview

### System Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Browser   │────▶│ Portal Front │────▶│Portal Backend│
└─────────────┘     │   (Next.js)  │     │   (Go API)   │
                    └──────────────┘     └──────┬───────┘
                                                 │
                                                 ▼
                    ┌──────────────────────────────────┐
                    │     CLI Executor Framework       │
                    │  (Secure Command Execution)      │
                    └────────────┬─────────────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────────────┐
                    │    Docker MCP CLI Plugin         │
                    │  (Existing, Mature Gateway)      │
                    └────────────┬─────────────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────────────┐
                    │    Docker Engine / Containers    │
                    │     (MCP Server Instances)       │
                    └──────────────────────────────────┘
```

### OAuth Architecture (Phase 5 - 80% Complete)

```
┌────────────────┐
│ Portal Users   │──Azure AD──▶ Portal Authentication
└────────────────┘
┌────────────────┐
│ MCP Servers    │──OAuth──▶ OAuth Interceptor ──▶ Token Store
└────────────────┘           ├── 401 Detection
                             ├── Token Refresh
                             └── Provider Registry

┌────────────────┐
│ DCR Bridge     │──Azure AD Graph API──▶ App Registration
└────────────────┘  ├── RFC 7591 Handler
                    └── Key Vault Storage
```

## 💻 Technology Stack

### Backend (Go)

- **Framework**: Native Go HTTP + Gin router
- **Authentication**: Azure AD OAuth2, JWT (RS256)
- **Database**: PostgreSQL 17 with Row-Level Security
- **Cache**: Redis 8 for sessions
- **Security**: AES-256-GCM encryption, audit logging

### Frontend (TypeScript/React)

- **Framework**: Next.js 15.5.3 with App Router
- **UI**: Tailwind CSS v4, Shadcn/ui components
- **State**: React Query v5, Zustand
- **Auth**: MSAL.js for Azure AD
- **Real-time**: WebSocket/SSE for live updates

### Infrastructure

- **Container**: Docker 20.10+
- **Orchestration**: docker-compose
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus metrics, Fluent Bit logging

## 📊 Codebase Statistics

### Lines of Code

- **Total**: ~50,000+ lines
- **Go Backend**: ~25,000 lines
- **TypeScript Frontend**: ~15,000 lines
- **OAuth Implementation**: ~9,000 lines (portal/features/)
- **Tests**: ~1,800 lines (11% coverage - needs 50%+)

### File Count

- **Go Files**: 80+ files
- **TypeScript/React**: 40+ files
- **SQL Migrations**: 4 files
- **Documentation**: 30+ markdown files

## 🔐 Security Features

### Implemented

- ✅ Azure AD authentication with JWT tokens
- ✅ PostgreSQL Row-Level Security (RLS)
- ✅ AES-256-GCM encryption for sensitive data
- ✅ Command injection prevention
- ✅ Rate limiting on all endpoints
- ✅ Comprehensive audit logging
- ✅ OAuth token management with refresh

### Pending Validation

- 🔴 Security testing blocked by build issues
- 🔴 Penetration testing cannot proceed
- 🔴 OAuth flow validation blocked

## 🚀 Quick Start Commands

### Fix Build Issues First

```bash
# 1. Fix Go dependencies
go mod tidy
go mod vendor

# 2. Check uncommitted work
git status
git diff cmd/docker-mcp/portal/features/

# 3. Attempt build
make docker-mcp

# 4. Try tests (will likely fail)
make test
```

### Development (After Fixes)

```bash
# CLI development
docker mcp --help
docker mcp gateway run

# Portal development
make portal-dev-up
cd cmd/docker-mcp/portal/frontend && npm run dev
```

## 📝 Key Documentation

### Must Read First

1. [AI Assistant Primer](implementation-plan/ai-assistant-primer.md) - Current context and status
2. [Project Tracker](implementation-plan/01-planning/project-tracker.md) - Real-time progress
3. [QUICKSTART.md](QUICKSTART.md) - Getting started guide (with warnings)

### Architecture & Design

1. [Technical Architecture](implementation-plan/03-architecture/technical-architecture.md)
2. [CLI Integration Architecture](implementation-plan/03-architecture/cli-integration-architecture.md)
3. [API Specification](implementation-plan/03-architecture/api-specification.md)

### Phase Documentation

1. [Phase 5 OAuth](implementation-plan/02-phases/phase-5-oauth-authentication.md) - 80% complete
2. [Phase 4 Deployment](implementation-plan/02-phases/phase-4-deployment.md) - 60% blocked

## 🎯 Immediate Priorities

### Critical (Must Fix First)

1. **Resolve Go module vendor dependencies**

   - Run `go mod tidy` and `go mod vendor`
   - Fix import resolution errors

2. **Fix test compilation failures**

   - Address 78+ compilation errors
   - Enable basic test execution

3. **Handle uncommitted OAuth work**
   - Review 8 files in portal/features/
   - Commit or revert as appropriate

### Important (After Critical)

1. **Validate OAuth implementation**

   - Test 401 interceptor functionality
   - Verify DCR bridge operations
   - Test token refresh mechanisms

2. **Expand test coverage**
   - Current: 11%
   - Target: 50%+
   - Focus on critical paths

## 🔗 Related Resources

### Internal

- [README.md](README.md) - Main project documentation
- [AGENTS.md](AGENTS.md) - AI assistant guidelines
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development guidelines

### External

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Docker Desktop Docs](https://docs.docker.com/desktop/)
- [GitHub Repository](https://github.com/jrmatherly/mcp-hub-gateway)

## 📅 Timeline

### Completed

- **September 16, 2025**: Phase 1-2 completed
- **January 20, 2025**: Phase 3 completed
- **September 17-18, 2025**: OAuth implementation (80%)

### Current

- **September 19, 2025**: Build system issues discovered
- **Status**: Project blocked pending dependency fixes

### Next Milestones

- Fix build system (immediate)
- Validate OAuth implementation
- Complete Phase 4 deployment
- Production readiness assessment

## 🤝 Contact & Support

- **Repository**: [github.com/jrmatherly/mcp-hub-gateway](https://github.com/jrmatherly/mcp-hub-gateway)
- **Issues**: [GitHub Issues](https://github.com/jrmatherly/mcp-hub-gateway/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jrmatherly/mcp-hub-gateway/discussions)

---

_This index provides a comprehensive overview of the MCP Gateway & Portal project as of September 19, 2025. The project has substantial OAuth implementation complete but is currently blocked by critical build system issues that must be resolved before further progress._
