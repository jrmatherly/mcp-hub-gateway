# Contributing Guide

## Building from Source

```bash
# Build the binary
make docker-mcp

# Cross-compile for all platforms
make docker-mcp-cross

# Run tests
make test
```

## Code Quality

```bash
# Format code
make format

# Run linter
make lint

# Run both
make lint format
```

## Code Style Conventions

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
    "github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/config"
    "github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/internal/docker"
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

### Running Tests

```bash
# Run all unit tests
make test

# Run integration tests
make integration

# Run specific test
go test -run TestServerList ./cmd/docker-mcp/server

# Run with coverage
go test -cover ./...
```

### Unit Test Coverage

```bash
# Generate HTML coverage report for ALL packages in one view
go test -cover -coverprofile=coverage.out ./... -short
go tool cover -html=coverage.out -o coverage.html && open coverage.html
```

### Testing Philosophy

- **Fix the code, not the test**: When tests fail, fix the underlying issue
- **Meaningful tests**: Tests should verify actual functionality
- **Test edge cases**: Tests that reveal limitations help improve code
- **Document purpose**: Each test should explain what it validates

## Security Considerations

### Command Injection Prevention

- **NEVER** pass user input directly to CLI commands
- Always validate and sanitize inputs
- Use parameter whitelists for CLI arguments
- Escape special characters in parameters

### Secret Management

- Never commit secrets to git
- Use Docker Desktop secrets API when available
- Use `.env` files for local development (gitignored)
- Environment variables for production

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following code style guidelines
4. Add tests for new functionality
5. Run `make lint test` before committing
6. Commit with descriptive message
7. Push to your fork
8. Create Pull Request with detailed description

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

Types: feat, fix, docs, style, refactor, test, chore

Example:

```
feat: add OAuth interceptor middleware

Implements automatic 401 response handling for MCP servers
with token refresh and retry logic.

Closes #123
```

## Code Review Checklist

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation updated if needed
- [ ] No security vulnerabilities introduced
- [ ] Follows project code style
- [ ] Performance impact considered
- [ ] Breaking changes documented
