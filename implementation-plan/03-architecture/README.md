# Technical Architecture & Design

[← Back to Implementation Plan](../README.md)

This directory contains all technical architecture documents, specifications, and design details for the MCP Portal.

## 🏢 Architecture Overview

### Core Architecture

- **[Technical Architecture](./technical-architecture.md)** - Complete system design and component overview
- **[CLI Integration Architecture](./cli-integration-architecture.md)** - Complete CLI integration design with executive summary

### Data & API Design

- **[Database Schema](./database-schema.md)** - PostgreSQL schema with Row-Level Security
- **[API Specification](./api-specification.md)** - REST API endpoints and contracts
- **[CLI Command Mapping](./cli-command-mapping.md)** - Complete CLI command reference

## Key Architectural Principles

### 🔒 Security First

- **Command Injection Prevention**: All CLI parameters validated and sanitized
- **Row-Level Security**: PostgreSQL RLS for multi-tenant data isolation
- **Azure AD Integration**: Enterprise authentication with JWT tokens
- **Audit Logging**: Comprehensive audit trail for all operations

### 🚀 Performance & Reliability

- **CLI Wrapper Pattern**: Leverage existing, mature CLI functionality
- **Rate Limiting**: Multi-tier rate limiting for API protection
- **Real-time Updates**: WebSocket/SSE for streaming CLI output
- **Resource Management**: Container lifecycle and resource monitoring

### 🔧 Maintainability

- **Clear Separation**: Frontend, backend, and CLI remain independent
- **Type Safety**: Full TypeScript/Go type definitions
- **Testing Strategy**: Unit, integration, and E2E testing coverage
- **Documentation**: Comprehensive API and architecture docs

## Integration Patterns

### CLI Execution Flow

```
Web Request → API Handler → CLI Executor → Docker CLI → Response Parser → Web Response
```

### Security Layers

```
Azure AD → JWT Validation → Rate Limiting → Parameter Validation → CLI Execution
```

## Navigation

### Other Sections

- [Planning](../01-planning/) - Progress tracking and project management
- [Phases](../02-phases/) - Detailed implementation phase documents
- [Guides](../04-guides/) - Development and deployment guides

### Quick Links

- [Main README](../README.md) - Project overview and status
- [AI Assistant Primer](../ai-assistant-primer.md) - Complete context for AI assistants
