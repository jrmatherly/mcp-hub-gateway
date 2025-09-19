# AGENTS.md

AI assistant guidance for the MCP Gateway & Portal repository.

**Note:** CLAUDE.md, .clinerules, .cursorrules, .windsurfrules are symlinks to this file.

## Project Overview

**Two distinct projects in this repository:**

1. **MCP Gateway CLI** - Docker CLI plugin for MCP server management (Go 1.24+)
   - Location: Main repository code
   - Status: Production-ready

2. **MCP Portal** - Web UI wrapper for the CLI (Not a reimplementation)
   - Location: `/cmd/docker-mcp/portal/`
   - Frontend: Next.js + TypeScript
   - Backend: Go service executing CLI commands
   - Status: Phase 4 (91% complete), Phase 5 (OAuth) planned

**CRITICAL**: The Portal wraps the CLI via command execution - it does NOT reimplement MCP functionality.

## Essential Guidelines

### Security First
- **Command Injection Prevention**: NEVER pass user input directly to CLI
- **Parameter Validation**: Whitelist and sanitize all inputs
- **Authentication**: Azure AD with JWT tokens (RS256)
- **Secrets**: Never commit secrets, use Docker Desktop secrets or env vars

### Testing Requirements
- **Minimum Coverage**: 50% for production readiness
- **Test Location**: Alongside source files (`*_test.go`)
- **Philosophy**: Fix the code, not the test
- **Current Gap**: 11% coverage - PRIORITY for Phase 4

### Development Patterns
- **Parallel Execution**: Always use parallel tool calls when possible
- **Error Handling**: Wrap errors with context (`fmt.Errorf`)
- **Context Usage**: Pass `context.Context` as first parameter
- **CLI Integration**: Portal executes commands, parses output

## Key Locations

```
/cmd/docker-mcp/portal/     # Portal backend
  ├── server/               # HTTP server (Gin)
  ├── auth/                 # Azure AD authentication
  ├── executor/             # CLI command execution
  └── frontend/             # Next.js application

/implementation-plan/       # Project documentation
  ├── 01-planning/         # Progress tracking
  ├── 02-phases/           # Phase documentation
  └── 03-architecture/     # Technical specs
```

## Current Development Priorities

### Phase 4 - Production Readiness (91% Complete)
1. **Test Coverage Expansion** - Target 50%+ (currently 11%)
2. **Performance Optimization** - Caching, query tuning
3. **Security Hardening** - Final audit

### Phase 5 - OAuth Integration (Planning)
1. OAuth interceptor for automatic 401 handling
2. DCR bridge service for Azure AD
3. Docker Desktop secrets integration (optional)
4. Multi-provider support (GitHub, Google, Microsoft)

## Quick Commands

```bash
# Build
make docker-mcp              # Build CLI plugin
make test                    # Run tests

# Portal
docker mcp portal serve      # Start portal
./deploy-mcp-portal.sh      # Deploy with Docker

# Gateway
docker mcp gateway run       # Start gateway
docker mcp server list       # List servers
```

## Important Reminders

1. **Architecture**: Portal is a CLI wrapper, not a reimplementation
2. **Security**: Command injection prevention is critical
3. **Testing**: Never skip tests to make builds pass
4. **Documentation**: Update docs when changing functionality
5. **Performance**: Use parallel operations for multi-file tasks

## References

For detailed information, see:
- **Setup**: [README.md](./README.md), [QUICKSTART.md](./QUICKSTART.md)
- **Development**: [development-setup.md](./implementation-plan/04-guides/development-setup.md)
- **Architecture**: [technical-architecture.md](./implementation-plan/03-architecture/technical-architecture.md)
- **Progress**: [project-tracker.md](./implementation-plan/01-planning/project-tracker.md)
- **Contributing**: [CONTRIBUTING.md](./CONTRIBUTING.md)

---
*This file provides essential context only. Detailed documentation is in the referenced files.*