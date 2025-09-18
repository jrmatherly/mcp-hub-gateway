# Multi-User Catalog System Documentation
**Date:** 2025-09-18
**Phase:** 4 - Production Readiness (90% Complete)
**Component:** MCP Portal - Multi-User Catalog Management

## Executive Summary

The Multi-User Catalog System extends the MCP Portal's catalog management capabilities to support enterprise-grade, multi-tenant environments. It implements a hierarchical inheritance model where administrators can define base catalogs that users can customize while maintaining centralized control.

## Architecture Overview

### Component Structure

```
catalog/
├── service_multiuser.go    # Multi-user catalog service with inheritance
├── inheritance.go           # Catalog inheritance and merging engine
├── file_manager.go          # File-based catalog persistence
├── encryption_adapter.go    # AES-256-GCM encryption for sensitive data
├── repository.go            # Database repository pattern
├── service.go               # Core catalog service
└── types.go                 # Type definitions and models
```

### Key Components

1. **MultiUserCatalogService** (`service_multiuser.go`)
   - Extends base catalog service with multi-user support
   - Manages user-specific catalog customizations
   - Implements secure catalog import/export
   - Handles server enable/disable operations

2. **InheritanceEngine** (`inheritance.go`)
   - Manages catalog inheritance hierarchy
   - Resolves conflicts between catalog layers
   - Caches resolved catalogs for performance
   - Tracks conflict resolution details

3. **FileCatalogManager** (`file_manager.go`)
   - Persists catalogs to file system
   - Implements atomic file operations
   - Manages catalog versioning
   - Handles backup and recovery

4. **EncryptionAdapter** (`encryption_adapter.go`)
   - AES-256-GCM encryption for sensitive data
   - Key derivation and rotation support
   - Secure storage of user customizations

## Catalog Inheritance Model

### Hierarchy Levels

```
┌─────────────────────────┐
│   System Default        │  Highest Priority
├─────────────────────────┤
│   Admin Base Catalog    │  Administrator-defined
├─────────────────────────┤
│   Team Catalog          │  Team/department level
├─────────────────────────┤
│   User Personal Catalog │  User customizations
└─────────────────────────┘  Lowest Priority
```

### Merge Strategy

1. **Override Strategy**: Higher priority catalogs override lower priority
2. **Append Strategy**: Servers are accumulated from all levels
3. **Selective Merge**: Fine-grained control per server configuration

### Conflict Resolution

```go
type ConflictDetail struct {
    ServerName       string  // Name of the conflicting server
    WinningSource    string  // Source that won the conflict
    OverriddenSource string  // Source that was overridden
    Reason           string  // Explanation of resolution
}
```

## API Endpoints

### Multi-User Catalog Management

```http
# Get merged catalog for user
GET /api/v1/catalogs/user/{userId}/merged
Authorization: Bearer {token}

# Import user catalog
POST /api/v1/catalogs/user/{userId}/import
Content-Type: application/json
{
    "format": "json|yaml",
    "data": "...",
    "override": boolean
}

# Export user catalog
GET /api/v1/catalogs/user/{userId}/export?format=json

# Update user customizations
PATCH /api/v1/catalogs/user/{userId}/customize
{
    "server_overrides": {},
    "disabled_servers": [],
    "custom_servers": []
}

# Enable/disable server for user
POST /api/v1/catalogs/user/{userId}/servers/{serverName}/enable
POST /api/v1/catalogs/user/{userId}/servers/{serverName}/disable
```

### Admin Catalog Management

```http
# Create admin base catalog
POST /api/v1/catalogs/admin/base
{
    "name": "enterprise-base",
    "servers": {},
    "metadata": {}
}

# Update admin base catalog
PUT /api/v1/catalogs/admin/base/{catalogId}

# List admin base catalogs
GET /api/v1/catalogs/admin/base

# Set inheritance configuration
POST /api/v1/catalogs/admin/inheritance
{
    "enabled": true,
    "priority": ["personal", "team", "admin_base"],
    "merge_rules": ["override", "append"]
}
```

## Security Features

### Encryption
- **Algorithm**: AES-256-GCM
- **Key Management**: Per-user key derivation
- **Data Protection**: All sensitive catalog data encrypted at rest
- **Key Rotation**: Support for periodic key rotation

### Access Control
- **Row-Level Security**: PostgreSQL RLS policies
- **JWT Authentication**: RS256 signed tokens
- **Azure AD Integration**: Enterprise SSO support
- **Audit Logging**: Comprehensive audit trail

### Command Injection Prevention
- **Input Validation**: Whitelist-based parameter validation
- **Command Sanitization**: Escape special characters
- **Rate Limiting**: Prevent abuse and DOS attacks
- **Timeout Control**: Maximum execution time limits

## Usage Examples

### User Workflow

```go
// Get merged catalog for user
catalog, err := service.GetUserCatalogMerged(ctx, userID)

// Import custom catalog
err := service.ImportUserCatalog(ctx, userID, catalogData, "json")

// Enable specific server
err := service.EnableUserServer(ctx, userID, "github-copilot")

// Export catalog configuration
data, err := service.ExportUserCatalog(ctx, userID, "yaml")
```

### Admin Workflow

```go
// Create admin base catalog
adminCatalog := &Catalog{
    Name: "enterprise-base",
    Type: CatalogTypeAdminBase,
    Servers: defaultServers,
}
err := service.CreateAdminBaseCatalog(ctx, adminCatalog)

// Configure inheritance
config := &InheritanceConfig{
    Enabled: true,
    Priority: []CatalogType{
        CatalogTypePersonal,
        CatalogTypeTeam,
        CatalogTypeAdminBase,
    },
}
err := service.SetInheritanceConfig(ctx, config)
```

## Performance Optimization

### Caching Strategy
- **Redis Cache**: 5-minute TTL for resolved catalogs
- **Memory Cache**: In-process cache for frequently accessed data
- **Cache Invalidation**: Automatic on catalog updates
- **Batch Operations**: Bulk updates for efficiency

### Database Optimization
- **Indexed Queries**: Optimized for common access patterns
- **Connection Pooling**: Max 25 connections
- **Query Optimization**: N+1 query prevention
- **Transaction Management**: ACID compliance

## Testing Coverage

### Test Suites Created
1. **service_multiuser_test.go** (773 lines)
   - Multi-user catalog operations
   - Inheritance and merging logic
   - Import/export functionality
   - Concurrent access scenarios

2. **repository_test.go** (580 lines)
   - Database CRUD operations
   - Transaction handling
   - Query optimization
   - Error handling

### Coverage Areas
- Unit tests for all public methods
- Integration tests for CLI execution
- Concurrent access testing
- Error condition coverage
- Mock implementations for dependencies

## Migration Guide

### From Single-User to Multi-User

1. **Enable Multi-User Mode**
   ```bash
   docker mcp catalog init --multi-user
   ```

2. **Create Admin Base Catalog**
   ```bash
   docker mcp catalog create --type admin_base --name enterprise-base
   ```

3. **Import Existing Catalogs**
   ```bash
   docker mcp catalog import --source legacy.yaml --type admin_base
   ```

4. **Configure Inheritance**
   ```bash
   docker mcp catalog inheritance --enable --priority personal,team,admin
   ```

## Troubleshooting

### Common Issues

1. **Catalog Merge Conflicts**
   - Check inheritance configuration
   - Review conflict resolution logs
   - Verify priority settings

2. **Performance Issues**
   - Check cache configuration
   - Monitor Redis memory usage
   - Review database query performance

3. **Permission Errors**
   - Verify RLS policies
   - Check JWT token claims
   - Review audit logs

### Debug Commands

```bash
# View catalog inheritance chain
docker mcp catalog debug inheritance --user {userId}

# Check cache status
docker mcp catalog debug cache

# Validate catalog structure
docker mcp catalog validate --file catalog.yaml

# Export debug information
docker mcp catalog debug export --output debug.json
```

## Future Enhancements

### Planned Features
1. **Catalog Marketplace**: Share catalogs across organizations
2. **Version Control**: Git-based catalog versioning
3. **Policy Engine**: Fine-grained access control policies
4. **Analytics Dashboard**: Usage metrics and insights
5. **Automated Testing**: Catalog validation pipelines

### Performance Improvements
1. **GraphQL API**: Efficient data fetching
2. **Event Sourcing**: Audit trail and time travel
3. **Distributed Cache**: Redis Cluster support
4. **Read Replicas**: Database scaling

## Compliance & Standards

### Security Compliance
- **OWASP Top 10**: Addressed all relevant vulnerabilities
- **CIS Benchmarks**: Docker and Kubernetes security
- **PCI DSS**: Data protection standards
- **SOC 2**: Security controls implementation

### Code Quality
- **Test Coverage**: Target 80% coverage
- **Code Review**: Mandatory PR reviews
- **Static Analysis**: golangci-lint integration
- **Documentation**: Comprehensive inline docs

## Support & Resources

### Documentation
- [API Reference](./api-specification.md)
- [CLI Command Reference](./cli-command-mapping.md)
- [Security Guide](./security.md)
- [Deployment Guide](./deployment-guide.md)

### Contact
- **GitHub Issues**: Report bugs and feature requests
- **Slack Channel**: #mcp-portal-support
- **Email**: mcp-support@docker.com

## Appendix

### Configuration Schema

```yaml
# catalog.yaml
version: "1.0"
type: "admin_base"
metadata:
  name: "Enterprise Base Catalog"
  description: "Default catalog for all users"
  maintainer: "platform-team@company.com"

inheritance:
  enabled: true
  priority:
    - personal
    - team
    - admin_base
  merge_rules:
    - override
    - append

servers:
  github-copilot:
    command: "npx"
    args: ["@github/copilot-mcp"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
    capabilities:
      - code_completion
      - chat

  postgresql:
    command: "docker"
    args: ["run", "mcp-postgres"]
    env:
      DATABASE_URL: "${DATABASE_URL}"
    capabilities:
      - database_queries
      - schema_management
```

### Database Schema

```sql
-- Admin catalogs table
CREATE TABLE admin_catalogs (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(50),
    servers JSONB,
    metadata JSONB,
    version INT DEFAULT 1,
    user_id VARCHAR(255),
    source_url VARCHAR(500),
    source_type VARCHAR(50),
    imported_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Inheritance configuration
CREATE TABLE inheritance_configs (
    catalog_id UUID PRIMARY KEY,
    enabled BOOLEAN DEFAULT true,
    priority JSONB,
    merge_rules JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_admin_catalogs_type ON admin_catalogs(type);
CREATE INDEX idx_admin_catalogs_user_id ON admin_catalogs(user_id);
CREATE INDEX idx_admin_catalogs_status ON admin_catalogs(status);
```

---

**Document Version:** 1.0
**Last Updated:** 2025-09-18
**Status:** Production Ready (90% Complete)