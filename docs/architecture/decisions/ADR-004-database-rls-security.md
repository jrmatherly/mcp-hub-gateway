# ADR-004: PostgreSQL Row-Level Security Implementation

**Status**: Accepted
**Date**: 2025-09-17
**Deciders**: Security Team, Database Team, Architecture Team
**Technical Story**: Portal needs multi-tenant data isolation and fine-grained access control

## Context and Problem Statement

The MCP Portal serves multiple users and organizations, each with their own MCP server configurations, OAuth credentials, and usage data. We need to ensure that:

1. **Data Isolation**: Users can only access their own data and configurations
2. **Multi-Tenancy**: Support multiple organizations with complete data separation
3. **Audit Security**: Prevent tampering with audit logs and security events
4. **Performance**: Maintain query performance while enforcing security policies
5. **Compliance**: Meet enterprise security requirements for data access control

Traditional application-level authorization can be bypassed by SQL injection, direct database access, or application bugs. We need database-level security that cannot be circumvented.

## Decision Drivers

- **Defense in Depth**: Database-level security as additional layer beyond application logic
- **Zero Trust**: No implicit trust in application layer for sensitive data access
- **Multi-Tenancy**: Clean separation between different organizations and users
- **Audit Integrity**: Tamper-proof audit logs that cannot be modified by users
- **Performance**: Security policies must not significantly impact query performance
- **Compliance**: SOC 2, GDPR, and enterprise security requirement compliance
- **Developer Experience**: Security policies should be transparent to application developers

## Considered Options

### Option A: Application-Level Authorization Only

**Description**: Implement all access control in the Go application layer with careful query construction.

**Pros**:

- Simple database schema without security policies
- Full control over authorization logic in application code
- Easier to debug and modify authorization rules
- No database-specific features required

**Cons**:

- Vulnerable to SQL injection bypassing authorization
- Can be bypassed by direct database access
- Application bugs can lead to data leaks
- No protection against privileged user abuse
- Difficult to audit and verify security compliance

### Option B: Database Views with Security Checks

**Description**: Create database views that include authorization checks and restrict application access to views only.

**Pros**:

- Database-level enforcement of access controls
- Application can use simple queries against views
- Easier to implement than full RLS policies
- Compatible with most database systems

**Cons**:

- Views can become complex and hard to maintain
- Limited flexibility for dynamic authorization rules
- Performance issues with complex view queries
- Still vulnerable to direct table access if permissions misconfigured
- Difficult to handle complex multi-tenant scenarios

### Option C: PostgreSQL Row-Level Security (RLS)

**Description**: Use PostgreSQL's native Row-Level Security feature with policies based on user context.

**Pros**:

- Native database-level security enforcement
- Cannot be bypassed by application bugs or SQL injection
- Flexible policy language for complex authorization rules
- Transparent to application code - standard SQL queries work
- Excellent performance when properly indexed
- Built-in audit capabilities

**Cons**:

- PostgreSQL-specific feature - vendor lock-in
- Requires careful policy design and testing
- Complex debugging when policies are incorrect
- Need to manage database user context properly

## Decision Outcome

**Chosen Option**: Option C - PostgreSQL Row-Level Security (RLS)

**Rationale**:

- Provides strongest security guarantees that cannot be bypassed
- PostgreSQL is already chosen for other technical reasons
- Transparent to application developers - no query changes needed
- Excellent performance characteristics with proper indexing
- Native audit capabilities and tamper resistance
- Meets enterprise security and compliance requirements

## Implementation Design

### Database Schema with RLS Policies

```sql
-- Users table with organization separation
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    azure_ad_object_id UUID NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see users in their organization
CREATE POLICY users_organization_isolation ON users
    FOR ALL
    TO portal_app
    USING (organization_id = current_setting('app.current_organization_id')::UUID);

-- Policy: Users can only modify their own record
CREATE POLICY users_self_modification ON users
    FOR UPDATE
    TO portal_app
    USING (id = current_setting('app.current_user_id')::UUID);
```

### MCP Server Configurations with User Isolation

```sql
CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255) NOT NULL,
    configuration JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE mcp_servers ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their own servers
CREATE POLICY mcp_servers_user_isolation ON mcp_servers
    FOR ALL
    TO portal_app
    USING (user_id = current_setting('app.current_user_id')::UUID);

-- Policy: Organization admins can see all servers in their org
CREATE POLICY mcp_servers_org_admin_access ON mcp_servers
    FOR SELECT
    TO portal_app
    USING (
        organization_id = current_setting('app.current_organization_id')::UUID
        AND current_setting('app.current_user_role') = 'admin'
    );
```

### Audit Logs with Tamper Protection

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB NOT NULL DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Policy: Audit logs are read-only and only visible to organization admins
CREATE POLICY audit_logs_read_only ON audit_logs
    FOR SELECT
    TO portal_app
    USING (
        organization_id = current_setting('app.current_organization_id')::UUID
        AND current_setting('app.current_user_role') IN ('admin', 'auditor')
    );

-- Policy: Only system can insert audit logs
CREATE POLICY audit_logs_system_insert ON audit_logs
    FOR INSERT
    TO portal_app
    WITH CHECK (current_setting('app.system_operation') = 'true');

-- No update or delete policies - audit logs are immutable
```

### OAuth Credentials with Enhanced Security

```sql
CREATE TABLE oauth_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    server_id UUID NOT NULL REFERENCES mcp_servers(id),
    provider VARCHAR(100) NOT NULL,
    encrypted_credentials BYTEA NOT NULL, -- AES-256-GCM encrypted
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE oauth_credentials ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their own OAuth credentials
CREATE POLICY oauth_credentials_user_isolation ON oauth_credentials
    FOR ALL
    TO portal_app
    USING (user_id = current_setting('app.current_user_id')::UUID);

-- Policy: System can access credentials for OAuth operations
CREATE POLICY oauth_credentials_system_access ON oauth_credentials
    FOR SELECT
    TO portal_system
    USING (true); -- System role has full read access for OAuth operations
```

## Application Integration

### Setting RLS Context in Go Application

```go
type RLSContext struct {
    UserID         uuid.UUID
    OrganizationID uuid.UUID
    UserRole       string
    SystemOperation bool
}

func (db *Database) SetRLSContext(ctx context.Context, rlsCtx *RLSContext) error {
    tx, err := db.conn.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    // Set session variables for RLS policies
    queries := []string{
        fmt.Sprintf("SET LOCAL app.current_user_id = '%s'", rlsCtx.UserID),
        fmt.Sprintf("SET LOCAL app.current_organization_id = '%s'", rlsCtx.OrganizationID),
        fmt.Sprintf("SET LOCAL app.current_user_role = '%s'", rlsCtx.UserRole),
        fmt.Sprintf("SET LOCAL app.system_operation = '%t'", rlsCtx.SystemOperation),
    }

    for _, query := range queries {
        if _, err := tx.ExecContext(ctx, query); err != nil {
            tx.Rollback()
            return fmt.Errorf("failed to set RLS context: %w", err)
        }
    }

    return tx.Commit()
}
```

### Middleware for Automatic RLS Context

```go
func (s *Server) RLSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract user context from JWT token
        userClaims, exists := c.Get("user")
        if !exists {
            c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
            return
        }

        claims := userClaims.(*UserClaims)

        // Set RLS context for this request
        rlsCtx := &RLSContext{
            UserID:         claims.UserID,
            OrganizationID: claims.OrganizationID,
            UserRole:       claims.Role,
            SystemOperation: false,
        }

        if err := s.db.SetRLSContext(c.Request.Context(), rlsCtx); err != nil {
            c.AbortWithStatusJSON(500, gin.H{"error": "database context setup failed"})
            return
        }

        c.Next()
    }
}
```

### Query Examples with RLS

```go
// Standard query - RLS automatically filters results
func (s *ServerService) GetUserServers(ctx context.Context, userID uuid.UUID) ([]MCPServer, error) {
    // RLS context already set by middleware
    // This query automatically only returns servers owned by the current user
    query := `SELECT id, name, image, configuration, enabled FROM mcp_servers WHERE enabled = true`

    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    var servers []MCPServer
    for rows.Next() {
        var server MCPServer
        if err := rows.Scan(&server.ID, &server.Name, &server.Image,
                          &server.Configuration, &server.Enabled); err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        servers = append(servers, server)
    }

    return servers, nil
}

// System operation - bypass RLS for OAuth operations
func (s *OAuthService) GetCredentialsForServer(ctx context.Context, serverID uuid.UUID) (*OAuthCredentials, error) {
    // Set system operation context
    rlsCtx := &RLSContext{
        SystemOperation: true,
    }
    if err := s.db.SetRLSContext(ctx, rlsCtx); err != nil {
        return nil, fmt.Errorf("failed to set system context: %w", err)
    }

    query := `SELECT encrypted_credentials FROM oauth_credentials WHERE server_id = $1`
    var encryptedCreds []byte
    if err := s.db.QueryRowContext(ctx, query, serverID).Scan(&encryptedCreds); err != nil {
        return nil, fmt.Errorf("credential lookup failed: %w", err)
    }

    return s.decryptCredentials(encryptedCreds)
}
```

## Security Policies by Table

### Organizations Table

```sql
-- Policy: Users can only see their own organization
CREATE POLICY organizations_member_access ON organizations
    FOR SELECT
    TO portal_app
    USING (id = current_setting('app.current_organization_id')::UUID);

-- Policy: Only system admins can modify organizations
CREATE POLICY organizations_system_admin_only ON organizations
    FOR ALL
    TO portal_admin
    USING (true);
```

### User Sessions Table

```sql
-- Policy: Users can only access their own sessions
CREATE POLICY user_sessions_self_access ON user_sessions
    FOR ALL
    TO portal_app
    USING (user_id = current_setting('app.current_user_id')::UUID);

-- Policy: Sessions expire automatically
CREATE POLICY user_sessions_expiry_check ON user_sessions
    FOR SELECT
    TO portal_app
    USING (expires_at > NOW());
```

### Configuration Table

```sql
-- Policy: Organization-level configuration access
CREATE POLICY configurations_org_access ON configurations
    FOR ALL
    TO portal_app
    USING (organization_id = current_setting('app.current_organization_id')::UUID);

-- Policy: Only admins can modify system configurations
CREATE POLICY configurations_admin_modify ON configurations
    FOR UPDATE
    TO portal_app
    USING (
        organization_id = current_setting('app.current_organization_id')::UUID
        AND current_setting('app.current_user_role') = 'admin'
        AND config_type != 'system'
    );
```

## Performance Optimization

### Indexing Strategy for RLS

```sql
-- Indexes to support RLS policy performance
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_azure_ad_object_id ON users(azure_ad_object_id);

CREATE INDEX idx_mcp_servers_user_id ON mcp_servers(user_id);
CREATE INDEX idx_mcp_servers_organization_id ON mcp_servers(organization_id);
CREATE INDEX idx_mcp_servers_user_org ON mcp_servers(user_id, organization_id);

CREATE INDEX idx_oauth_credentials_user_id ON oauth_credentials(user_id);
CREATE INDEX idx_oauth_credentials_server_id ON oauth_credentials(server_id);

CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

### Query Performance Monitoring

```sql
-- View to monitor RLS policy performance
CREATE VIEW rls_performance_stats AS
SELECT
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_tup_ins,
    n_tup_upd,
    n_tup_del
FROM pg_stat_user_tables
WHERE schemaname = 'public';
```

## Operational Procedures

### RLS Policy Testing

```sql
-- Test framework for RLS policies
CREATE OR REPLACE FUNCTION test_rls_policy(
    test_user_id UUID,
    test_org_id UUID,
    test_role TEXT,
    expected_rows INTEGER,
    table_name TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    actual_rows INTEGER;
BEGIN
    -- Set test context
    PERFORM set_config('app.current_user_id', test_user_id::TEXT, true);
    PERFORM set_config('app.current_organization_id', test_org_id::TEXT, true);
    PERFORM set_config('app.current_user_role', test_role, true);

    -- Execute test query
    EXECUTE format('SELECT COUNT(*) FROM %I', table_name) INTO actual_rows;

    -- Check result
    RETURN actual_rows = expected_rows;
END;
$$ LANGUAGE plpgsql;
```

### Monitoring and Alerting

```yaml
rls_monitoring:
  policy_violations:
    query: "SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active' AND query LIKE '%SECURITY_VIOLATION%'"
    threshold: 0
    action: immediate_alert

  performance_degradation:
    query: "SELECT AVG(seq_scan / NULLIF(idx_scan, 0)) FROM pg_stat_user_tables WHERE schemaname = 'public'"
    threshold: 0.1 # More than 10% sequential scans
    action: performance_review

  context_setting_errors:
    log_pattern: "ERROR.*app\\.current_user_id"
    action: investigate_application_logic
```

## Security Benefits

### Defense in Depth

- **Application Layer**: Input validation, authorization checks, secure coding practices
- **Database Layer**: RLS policies prevent data access even if application is compromised
- **Network Layer**: TLS encryption, network segmentation, firewall rules
- **Infrastructure Layer**: Container isolation, secrets management, monitoring

### Threat Mitigation

- **SQL Injection**: RLS policies limit data access even with successful injection
- **Application Bugs**: Authorization bugs cannot bypass database-level security
- **Privileged User Abuse**: Database users still subject to RLS policies
- **Data Exfiltration**: Impossible to access data outside assigned organization/user
- **Audit Tampering**: Immutable audit logs with read-only policies

## Positive Consequences

- **Security Guarantee**: Database-level enforcement cannot be bypassed by application bugs
- **Multi-Tenancy**: Clean data separation between organizations and users
- **Audit Integrity**: Tamper-proof audit logs with database-level protection
- **Developer Productivity**: Transparent to application code - no query changes needed
- **Compliance**: Meets enterprise security requirements for data access control
- **Performance**: Excellent performance when properly indexed

## Negative Consequences

- **PostgreSQL Dependency**: Tied to PostgreSQL-specific features
- **Debugging Complexity**: RLS policy issues can be difficult to debug
- **Testing Overhead**: Need comprehensive testing of RLS policies
- **Operational Complexity**: Additional database administration procedures
- **Context Management**: Application must properly manage RLS context

## Mitigation Strategies

### Debugging and Troubleshooting

```sql
-- Debug view to understand RLS context
CREATE VIEW current_rls_context AS
SELECT
    current_setting('app.current_user_id', true) as user_id,
    current_setting('app.current_organization_id', true) as organization_id,
    current_setting('app.current_user_role', true) as user_role,
    current_setting('app.system_operation', true) as system_operation;

-- Function to explain RLS policy application
CREATE OR REPLACE FUNCTION explain_rls_query(query_text TEXT)
RETURNS TABLE(query_plan TEXT) AS $$
BEGIN
    RETURN QUERY EXECUTE 'EXPLAIN (ANALYZE, BUFFERS) ' || query_text;
END;
$$ LANGUAGE plpgsql;
```

### Testing Automation

```go
func TestRLSPolicies(t *testing.T) {
    testCases := []struct {
        name        string
        userID      uuid.UUID
        orgID       uuid.UUID
        role        string
        table       string
        expectedRows int
    }{
        {"User sees own servers", testUserID, testOrgID, "user", "mcp_servers", 3},
        {"Admin sees all org servers", adminUserID, testOrgID, "admin", "mcp_servers", 10},
        {"User from other org sees nothing", otherUserID, otherOrgID, "user", "mcp_servers", 0},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := context.Background()

            // Set RLS context
            rlsCtx := &RLSContext{
                UserID:         tc.userID,
                OrganizationID: tc.orgID,
                UserRole:       tc.role,
            }

            err := db.SetRLSContext(ctx, rlsCtx)
            require.NoError(t, err)

            // Test query
            var count int
            query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tc.table)
            err = db.QueryRowContext(ctx, query).Scan(&count)
            require.NoError(t, err)

            assert.Equal(t, tc.expectedRows, count)
        })
    }
}
```

## Future Enhancements

### Advanced RLS Features

- **Time-based Policies**: Policies that change based on time of day or date
- **Geographic Policies**: Location-based access restrictions
- **Dynamic Policies**: Policies that adapt based on user behavior or risk score
- **Policy Versioning**: Ability to version and rollback RLS policies

### Integration Enhancements

- **Policy Management UI**: Web interface for managing RLS policies
- **Policy Testing Framework**: Automated testing of policy changes
- **Performance Analytics**: Detailed performance analysis of RLS queries
- **Compliance Reporting**: Automated compliance reports based on RLS enforcement

## Validation Criteria

### Security Validation

- [ ] Users cannot access data outside their organization
- [ ] Users cannot access other users' MCP servers or credentials
- [ ] Audit logs cannot be modified or deleted by regular users
- [ ] SQL injection attempts are blocked by RLS policies
- [ ] Direct database access respects RLS policies

### Performance Validation

- [ ] Query performance degradation <10% with RLS enabled
- [ ] Proper index usage for all RLS policy queries
- [ ] Connection pooling works correctly with RLS context
- [ ] No significant memory overhead from RLS policies

### Operational Validation

- [ ] RLS context properly set for all application requests
- [ ] Error handling gracefully manages RLS context failures
- [ ] Monitoring alerts fire correctly for policy violations
- [ ] Backup and restore procedures preserve RLS policies

## Related Decisions

- **ADR-001**: CLI wrapper pattern affects how audit logs are generated for CLI operations
- **ADR-002**: Azure AD OAuth integration provides user context for RLS policies
- **ADR-003**: DCR bridge operations require system-level RLS context bypass

## References

- [PostgreSQL Row Level Security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [PostgreSQL Security Best Practices](https://www.postgresql.org/docs/current/sql-security-label.html)
- [Multi-Tenant Data Architecture](https://docs.microsoft.com/en-us/azure/sql-database/sql-database-design-patterns-multi-tenancy-saas-applications)
- [Database Security Patterns](https://owasp.org/www-project-cheat-sheets/cheatsheets/Database_Security_Cheat_Sheet.html)

---

**ADR**: Architecture Decision Record
**Last Updated**: September 19, 2025
**Next Review**: December 2025 (quarterly review)
**Status**: Implemented - RLS policies active in production with comprehensive testing
