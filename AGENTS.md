# AGENTS.md

Instructions for AI coding assistants working on MCP Gateway & Portal.

## Project Context

**Two projects in one repository:**

- **MCP Gateway CLI**: Docker plugin for MCP server management (Go 1.24+)
- **MCP Portal**: Web UI that wraps the CLI - NOT a reimplementation (Go backend + Next.js)

**Current Status**: Portal Phase 4 (60% - BLOCKED), Phase 5 OAuth (80% implemented but unstable)

## Setup Commands

```bash
# Build CLI plugin
make docker-mcp

# Run tests
make test

# Start portal
docker mcp portal serve

# Deploy portal with Docker
./deploy-mcp-portal.sh
```

## Code Style

**Go:**

- Context as first parameter
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Test files alongside source (`*_test.go`)

**TypeScript:**

- Functional components with hooks
- TypeScript strict mode
- Tailwind CSS for styling

## Testing

```bash
# Run all tests
make test

# Integration tests
make integration

# Coverage (target: 50%+, current: 11%)
go test -cover ./...
```

**Critical**: Fix code, not tests. Never skip tests to pass builds.

## Key Guidelines

### Security

- **NEVER** pass user input directly to CLI commands
- Validate and sanitize all parameters
- Use parameter whitelists

### Architecture

- Portal wraps CLI via command execution
- Portal does NOT reimplement MCP functionality
- Use parallel operations for multiple files

### File Structure

```
/cmd/docker-mcp/portal/    # Portal code
/implementation-plan/      # Documentation
/reports/                  # Project reports
```

## Current Priorities

1. **🔴 CRITICAL - Build System Stability**: Fix Go module vendor dependencies and test compilation
2. **🔴 CRITICAL - Repository Cleanup**: Resolve 8 uncommitted files in portal/features/
3. **🟡 IMPORTANT - OAuth System Stabilization**: Test and validate 80% complete OAuth implementation

## Pull Request Guidelines

1. Feature branches only (`feature/amazing-feature`)
2. Run `make lint test` before committing
3. Fix compilation errors and test failures
4. Update documentation for API changes

## References

- Setup: [README.md](./README.md), [QUICKSTART.md](./QUICKSTART.md)
- Architecture: [/implementation-plan/03-architecture/](./implementation-plan/03-architecture/)
- Progress: [project-tracker.md](./implementation-plan/01-planning/project-tracker.md)
