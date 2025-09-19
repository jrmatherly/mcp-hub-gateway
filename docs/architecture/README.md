# Architecture Documentation

**MCP Gateway & Portal Architecture Documentation Suite**

This directory contains comprehensive architecture documentation following modern practices with C4 models, Architecture Decision Records (ADRs), and detailed system designs.

## 📋 Documentation Index

### 🏗️ System Architecture

- [**C4-01-Context.md**](./C4-01-Context.md) - System context and external dependencies
- [**C4-02-Container.md**](./C4-02-Container.md) - High-level runtime containers and communication
- [**C4-03-Components.md**](./C4-03-Components.md) - Internal component structure and relationships
- [**data-flow-architecture.md**](./data-flow-architecture.md) - Data flows and processing pipelines
- [**security-architecture.md**](./security-architecture.md) - Security model and threat analysis

### 🎯 Design Decisions

- [**ADR-001-cli-wrapper-pattern.md**](./decisions/ADR-001-cli-wrapper-pattern.md) - CLI wrapper vs reimplementation
- [**ADR-002-azure-ad-oauth.md**](./decisions/ADR-002-azure-ad-oauth.md) - Azure AD integration strategy
- [**ADR-003-dcr-bridge-design.md**](./decisions/ADR-003-dcr-bridge-design.md) - Dynamic Client Registration approach
- [**ADR-004-database-rls-security.md**](./decisions/ADR-004-database-rls-security.md) - Row-Level Security implementation

### 🔗 Integration Architecture

- [**service-dependencies.md**](./service-dependencies.md) - Service dependency map and contracts
- [**api-contracts.md**](./api-contracts.md) - API specifications and interface definitions
- [**deployment-architecture.md**](./deployment-architecture.md) - Deployment patterns and infrastructure

### 📊 Technical Reference

- [**technology-stack.md**](./technology-stack.md) - Complete technology inventory and versions
- [**performance-architecture.md**](./performance-architecture.md) - Performance characteristics and bottlenecks
- [**monitoring-observability.md**](./monitoring-observability.md) - Monitoring, logging, and alerting design

## 🚀 Quick Navigation

### For New Developers

1. Start with [C4-01-Context.md](./C4-01-Context.md) for system overview
2. Read [CLI Wrapper ADR](./decisions/ADR-001-cli-wrapper-pattern.md) to understand core pattern
3. Review [C4-02-Container.md](./C4-02-Container.md) for runtime architecture

### For Security Review

1. [security-architecture.md](./security-architecture.md) - Comprehensive security model
2. [ADR-002-azure-ad-oauth.md](./decisions/ADR-002-azure-ad-oauth.md) - Authentication decisions
3. [data-flow-architecture.md](./data-flow-architecture.md) - Data handling patterns

### For Operations

1. [deployment-architecture.md](./deployment-architecture.md) - Infrastructure patterns
2. [monitoring-observability.md](./monitoring-observability.md) - Operational visibility
3. [service-dependencies.md](./service-dependencies.md) - Service relationships

## 📈 Current Architecture Status

**System Maturity**: Production-ready CLI, Portal at 80% completion
**Documentation Coverage**: Comprehensive with C4 models and ADRs
**Security Posture**: Defense-in-depth with Azure AD and RLS
**Performance Profile**: Optimized for developer workflows (<200ms operations)

## 🛠️ Architecture Tooling

**Diagramming**: Mermaid for all architecture diagrams
**Decision Tracking**: ADR format for architectural choices
**API Documentation**: OpenAPI 3.0 specifications
**Dependency Analysis**: Automated service mapping

---

**Last Updated**: September 19, 2025
**Architecture Version**: 2.0 (Phase 5 OAuth Integration)
**Review Schedule**: Monthly architecture reviews with stakeholders
