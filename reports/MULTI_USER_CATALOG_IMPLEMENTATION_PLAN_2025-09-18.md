# Multi-User Catalog Management Implementation Plan

**Date**: September 18, 2025
**Project**: MCP Gateway & Portal
**Subject**: Implementation of Admin-Controlled Base Catalogs with User Augmentation

## Executive Summary

This comprehensive plan details the implementation of a sophisticated multi-user catalog management system that addresses your specific requirements:

- ✅ **Admin-controlled base catalogs** that all users inherit
- ✅ **User-specific augmentation** allowing personal catalog customization
- ✅ **Per-user volume isolation** for security and data separation
- ✅ **Catalog inheritance with precedence** for conflict resolution

The system leverages existing Portal infrastructure (~25,000 lines of enterprise Go code) and the mature CLI catalog management features documented in `docs/feature-specs/`.

## Architecture Overview

### Catalog Inheritance Model

```
┌─────────────────────────────────────────┐
│         Admin Base Catalogs             │
│     (Controlled by Administrators)      │
└────────────────┬────────────────────────┘
                 │ Inherited by all users
                 ▼
┌─────────────────────────────────────────┐
│         User Personal Catalogs          │
│    (User-specific customizations)       │
└────────────────┬────────────────────────┘
                 │ Merged with precedence
                 ▼
┌─────────────────────────────────────────┐
│      Final Resolved Catalog             │
│   (What the user actually sees/uses)    │
└─────────────────────────────────────────┘
```

### Key Technical Components

1. **PostgreSQL with RLS**: Multi-tenant data isolation
2. **Per-User Volumes**: `/app/user-catalogs/{user-id}/.docker/mcp/`
3. **Inheritance Engine**: Smart catalog merging with precedence
4. **Cache Layer**: Redis for performance optimization

## Implementation Phases

### Phase 1: Core Infrastructure (2-3 weeks)

#### 1.1 Fix Docker Deployment Gaps

**Immediate Actions Required** (from validated analysis):

```dockerfile
# Dockerfile.mcp-portal - Add CLI Plugin Installation
RUN mkdir -p /home/portal/.docker/cli-plugins && \
    cp /app/backend/docker-mcp /home/portal/.docker/cli-plugins/docker-mcp && \
    chmod +x /home/portal/.docker/cli-plugins/docker-mcp
```

```yaml
# docker-compose.mcp-portal.yml - Add Catalog Volumes
volumes:
  - mcp-catalog:/home/portal/.docker/mcp
  - user-catalogs:/app/user-catalogs # Multi-user catalog storage
environment:
  HOME: /home/portal
  MCP_PORTAL_CATALOG_FEATURE_ENABLED: true
```

#### 1.2 Database Schema Extensions

```sql
-- Multi-user catalog support
CREATE TABLE catalog_configs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    catalog_type VARCHAR(50), -- 'admin_base', 'user', 'team'
    catalog_name VARCHAR(255),
    is_enabled BOOLEAN DEFAULT true,
    precedence INTEGER DEFAULT 100,
    config_data JSONB,
    parent_catalog_id UUID REFERENCES catalog_configs(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE catalog_configs ENABLE ROW LEVEL SECURITY;

CREATE POLICY catalog_access ON catalog_configs
    USING (
        user_id = current_user_id() OR
        catalog_type = 'admin_base' OR
        EXISTS (
            SELECT 1 FROM catalog_permissions
            WHERE catalog_id = id AND user_id = current_user_id()
        )
    );
```

### Phase 2: Catalog Inheritance System (2 weeks)

#### 2.1 Inheritance Engine Implementation

```go
// cmd/docker-mcp/portal/catalog/inheritance.go
package catalog

type InheritanceEngine struct {
    repo   Repository
    cache  *CatalogCache
    logger *slog.Logger
}

func (e *InheritanceEngine) ResolveCatalogForUser(
    ctx context.Context,
    userID string,
) (*ResolvedCatalog, error) {
    // 1. Load admin base catalogs
    baseCatalogs, err := e.repo.GetBaseCatalogs(ctx)

    // 2. Load user-specific catalogs
    userCatalogs, err := e.repo.GetUserCatalogs(ctx, userID)

    // 3. Apply merge strategy with precedence
    merged := e.mergeCatalogs(baseCatalogs, userCatalogs)

    // 4. Create user-specific volume with merged config
    volume, err := e.createUserVolume(ctx, userID, merged)

    return &ResolvedCatalog{
        UserID:       userID,
        BaseCatalogs: baseCatalogs,
        UserCatalogs: userCatalogs,
        MergedConfig: merged,
        VolumePath:   volume.Path,
    }, nil
}
```

#### 2.2 Volume Isolation Strategy

```go
// Per-user volume management
func (vm *VolumeManager) SetupUserCatalogVolume(
    ctx context.Context,
    userID string,
    catalogData []byte,
) error {
    // Create isolated directory for user
    userPath := fmt.Sprintf("/app/user-catalogs/%s/.docker/mcp", userID)

    // Ensure directory exists with proper permissions
    if err := os.MkdirAll(userPath, 0700); err != nil {
        return err
    }

    // Write merged catalog configuration
    catalogFile := filepath.Join(userPath, "catalog.json")
    if err := os.WriteFile(catalogFile, catalogData, 0600); err != nil {
        return err
    }

    // Set ownership to portal user
    return os.Chown(userPath, portalUID, portalGID)
}
```

### Phase 3: User Interface Implementation (2 weeks)

#### 3.1 Admin Interface for Base Catalog Management

```typescript
// Admin catalog management component
export function AdminCatalogManager() {
  const { data: baseCatalogs } = useBaseCatalogs();
  const updateMutation = useUpdateBaseCatalog();

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-4">Base Catalog Management</h2>

      {baseCatalogs?.map((catalog) => (
        <CatalogCard
          key={catalog.id}
          catalog={catalog}
          onUpdate={(updates) =>
            updateMutation.mutate({
              catalogId: catalog.id,
              updates,
            })
          }
        />
      ))}

      <Button onClick={handleCreateBaseCatalog}>Create New Base Catalog</Button>
    </div>
  );
}
```

#### 3.2 User Catalog Customization Interface

```typescript
// User catalog customization
export function UserCatalogCustomizer({ userId }: Props) {
  const { data: inheritance } = useUserCatalogInheritance(userId);
  const overrideMutation = useOverrideServer();

  return (
    <div>
      <h3>Your Catalog Configuration</h3>

      {/* Show inherited base catalogs (read-only) */}
      <section>
        <h4>Organization Catalogs (Admin-Controlled)</h4>
        {inheritance?.baseCatalogs.map((catalog) => (
          <BaseCatalogView key={catalog.id} catalog={catalog} />
        ))}
      </section>

      {/* User customizations */}
      <section>
        <h4>Your Customizations</h4>
        <ServerOverrideList
          servers={inheritance?.servers}
          onToggle={(serverId, enabled) =>
            overrideMutation.mutate({ serverId, enabled })
          }
        />
      </section>
    </div>
  );
}
```

### Phase 4: Advanced Features (1-2 weeks)

#### 4.1 Caching Strategy

```go
// Multi-layer caching for performance
type CatalogCache struct {
    redis      *redis.Client
    localCache *cache.Cache
}

func (c *CatalogCache) GetUserCatalog(
    ctx context.Context,
    userID string,
) (*ResolvedCatalog, error) {
    // Check local cache (L1)
    if cached := c.localCache.Get(userID); cached != nil {
        return cached.(*ResolvedCatalog), nil
    }

    // Check Redis (L2)
    key := fmt.Sprintf("catalog:%s", userID)
    if data, err := c.redis.Get(ctx, key).Bytes(); err == nil {
        var catalog ResolvedCatalog
        json.Unmarshal(data, &catalog)
        c.localCache.Set(userID, &catalog, 5*time.Minute)
        return &catalog, nil
    }

    // Resolve from database (L3)
    catalog, err := c.resolveFromDB(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Cache for next time
    c.cacheUserCatalog(ctx, userID, catalog)
    return catalog, nil
}
```

#### 4.2 Real-time Updates

```go
// Notify users when base catalogs change
func (s *CatalogService) NotifyBaseCatalogUpdate(
    ctx context.Context,
    baseCatalogID string,
) error {
    // Find affected users
    users, err := s.repo.GetUsersWithBaseCatalog(ctx, baseCatalogID)

    // Invalidate caches
    for _, userID := range users {
        s.cache.InvalidateUser(ctx, userID)

        // Send WebSocket notification
        s.ws.SendToUser(userID, Event{
            Type: "catalog_updated",
            Data: map[string]interface{}{
                "message": "Your catalog has been updated by an administrator",
                "catalogId": baseCatalogID,
            },
        })
    }

    return nil
}
```

## Deployment Configuration

### Updated Docker Compose

```yaml
# docker-compose.mcp-portal.yml (enhanced)
services:
  portal:
    # ... existing configuration ...
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - portal-data:/app/data
      - portal-logs:/app/logs
      # NEW: Multi-user catalog volumes
      - user-catalogs:/app/user-catalogs
      - base-catalogs:/app/base-catalogs
    environment:
      # NEW: Multi-user configuration
      MULTI_USER_CATALOGS_ENABLED: true
      CATALOG_INHERITANCE_ENABLED: true
      USER_VOLUME_ISOLATION: true
      BASE_CATALOG_PATH: /app/base-catalogs
      USER_CATALOG_PATH: /app/user-catalogs

volumes:
  user-catalogs:
    driver: local
  base-catalogs:
    driver: local
```

## Security Considerations

### 1. Volume Isolation

- Each user gets isolated directory: `/app/user-catalogs/{user-id}/`
- Strict file permissions (0700 for directories, 0600 for files)
- No cross-user access possible

### 2. Command Injection Prevention

- All catalog operations go through validated executor
- Parameter whitelisting and sanitization
- Audit logging of all operations

### 3. Row Level Security

- PostgreSQL RLS ensures data isolation
- Users can only see their own data + public base catalogs
- Admin role required for base catalog modifications

## Testing Strategy

### Test Scenarios

1. **Admin creates base catalog → All users inherit**
2. **User disables specific server → Override persists**
3. **Admin updates base → Users receive notification**
4. **Multiple users → Verify volume isolation**
5. **Cache invalidation → Verify consistency**

### Performance Targets

- Catalog resolution: <100ms average
- Cache hit ratio: >90%
- Volume creation: <2 seconds
- Concurrent users: 100+

## Migration Plan

### Week 1-2: Infrastructure

- Deploy database schema changes
- Update Docker configuration
- Add volume mounts

### Week 3-4: Core Features

- Deploy inheritance engine
- Implement volume isolation
- Add caching layer

### Week 5-6: User Interface

- Deploy admin interface
- Add user customization UI
- Enable real-time updates

### Week 7-8: Production

- Performance optimization
- Security hardening
- User documentation

## Success Metrics

### Technical Metrics

- ✅ All catalog operations work in containers
- ✅ Multi-user isolation verified
- ✅ Performance targets met
- ✅ Security audit passed

### Business Metrics

- ✅ Admin overhead reduced by 50%
- ✅ User onboarding time <5 minutes
- ✅ Support tickets reduced by 30%
- ✅ User satisfaction >4.5/5

## Next Steps

1. **Immediate**: Fix Docker deployment gaps (Phase 1.1)
2. **This Week**: Deploy database schema changes
3. **Next Week**: Implement inheritance engine
4. **Following Week**: Deploy user interfaces

## Conclusion

This plan provides a clear path to implement the multi-user catalog system with admin-controlled base catalogs and user-specific augmentation. The architecture leverages existing Portal infrastructure while adding the sophisticated inheritance model you require.

The phased approach ensures we can deliver value incrementally while maintaining system stability. With the existing Portal foundation and clear specifications from `docs/feature-specs/`, implementation can proceed immediately.

---

**Prepared by**: Enhanced Research Analysis & Documentation Experts
**Status**: Ready for Implementation
