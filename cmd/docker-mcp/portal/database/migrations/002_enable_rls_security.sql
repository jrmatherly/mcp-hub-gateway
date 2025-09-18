-- Migration: Enable Row-Level Security with Performance Optimizations
-- Version: 002
-- Description: Implements comprehensive RLS policies with security hardening

BEGIN;

-- ============================================================================
-- SECURITY FUNCTIONS
-- ============================================================================

-- Create secure user ID retrieval function with validation
CREATE OR REPLACE FUNCTION get_current_user_secure()
RETURNS UUID AS $$
DECLARE
    user_id_text TEXT;
    user_uuid UUID;
BEGIN
    -- Get user ID from session variable
    user_id_text := current_setting('app.current_user_id', true);

    -- Return NULL if not set (for public access scenarios)
    IF user_id_text IS NULL OR user_id_text = '' THEN
        RETURN NULL;
    END IF;

    -- Validate UUID format to prevent injection
    IF user_id_text !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN
        RAISE EXCEPTION 'Invalid user ID format: %', user_id_text
            USING ERRCODE = 'invalid_parameter_value';
    END IF;

    user_uuid := user_id_text::UUID;

    -- Verify user exists and is active
    IF NOT EXISTS (
        SELECT 1 FROM users
        WHERE id = user_uuid
        AND is_active = true
        AND deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION 'User not found or inactive: %', user_uuid
            USING ERRCODE = 'insufficient_privilege';
    END IF;

    RETURN user_uuid;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Create optimized admin check function
CREATE OR REPLACE FUNCTION is_admin(user_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT EXISTS (
        SELECT 1 FROM users
        WHERE id = user_uuid
        AND role IN ('super_admin', 'team_admin')
        AND is_active = true
        AND deleted_at IS NULL
    );
$$ LANGUAGE SQL STABLE SECURITY DEFINER;

-- Create team membership check function
CREATE OR REPLACE FUNCTION is_team_member(user_uuid UUID, team_uuid UUID)
RETURNS BOOLEAN AS $$
    SELECT EXISTS (
        SELECT 1 FROM team_members
        WHERE user_id = user_uuid
        AND team_id = team_uuid
        AND is_active = true
        AND deleted_at IS NULL
    );
$$ LANGUAGE SQL STABLE SECURITY DEFINER;

-- ============================================================================
-- ENABLE ROW LEVEL SECURITY ON ALL TABLES
-- ============================================================================

-- Enable RLS on all sensitive tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

ALTER TABLE teams ENABLE ROW LEVEL SECURITY;

ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;

ALTER TABLE mcp_servers ENABLE ROW LEVEL SECURITY;

ALTER TABLE user_server_configs ENABLE ROW LEVEL SECURITY;

ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

ALTER TABLE secrets ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- USERS TABLE POLICIES
-- ============================================================================

-- Users can only view their own profile (non-admins)
CREATE POLICY users_self_view ON users FOR
SELECT USING (
        id = get_current_user_secure ()
        OR is_admin (get_current_user_secure ())
    );

-- Users can update their own profile (non-sensitive fields)
CREATE POLICY users_self_update ON users FOR
UPDATE USING (
    id = get_current_user_secure ()
)
WITH
    CHECK (
        id = get_current_user_secure ()
        -- Prevent role escalation
        AND role = (
            SELECT role
            FROM users
            WHERE
                id = get_current_user_secure ()
        )
    );

-- Only admins can insert new users
CREATE POLICY users_admin_insert ON users FOR
INSERT
WITH
    CHECK (
        is_admin (get_current_user_secure ())
    );

-- Only admins can delete users (soft delete)
CREATE POLICY users_admin_delete ON users FOR DELETE USING (
    is_admin (get_current_user_secure ())
);

-- ============================================================================
-- TEAMS TABLE POLICIES
-- ============================================================================

-- Users can view teams they belong to
CREATE POLICY teams_member_view ON teams FOR
SELECT USING (
        is_team_member (
            get_current_user_secure (), id
        )
        OR is_admin (get_current_user_secure ())
    );

-- Team admins can update their teams
CREATE POLICY teams_admin_update ON teams FOR
UPDATE USING (
    EXISTS (
        SELECT 1
        FROM team_members
        WHERE
            team_id = teams.id
            AND user_id = get_current_user_secure ()
            AND role = 'admin'
    )
    OR is_admin (get_current_user_secure ())
);

-- ============================================================================
-- USER SERVER CONFIGS POLICIES
-- ============================================================================

-- Users can only access their own configurations
CREATE POLICY configs_user_access ON user_server_configs FOR ALL USING (
    user_id = get_current_user_secure ()
    OR is_admin (get_current_user_secure ())
)
WITH
    CHECK (
        user_id = get_current_user_secure ()
        OR is_admin (get_current_user_secure ())
    );

-- ============================================================================
-- AUDIT LOGS POLICIES
-- ============================================================================

-- Users can only view their own audit logs
CREATE POLICY audit_logs_self_view ON audit_logs FOR
SELECT USING (
        user_id = get_current_user_secure ()
        OR is_admin (get_current_user_secure ())
    );

-- Audit logs are insert-only (no update/delete)
CREATE POLICY audit_logs_insert_only ON audit_logs FOR
INSERT
WITH
    CHECK (true);

-- ============================================================================
-- SESSIONS POLICIES
-- ============================================================================

-- Users can only access their own sessions
CREATE POLICY sessions_user_access ON user_sessions FOR ALL USING (
    user_id = get_current_user_secure ()
)
WITH
    CHECK (
        user_id = get_current_user_secure ()
    );

-- ============================================================================
-- SECRETS POLICIES
-- ============================================================================

-- Secrets are only accessible by their owners
CREATE POLICY secrets_owner_access ON secrets FOR ALL USING (
    user_id = get_current_user_secure ()
)
WITH
    CHECK (
        user_id = get_current_user_secure ()
    );

-- ============================================================================
-- PERFORMANCE OPTIMIZATION INDEXES
-- ============================================================================

-- Create partial indexes for RLS predicates
CREATE INDEX IF NOT EXISTS idx_users_active ON users (id)
WHERE
    is_active = true
    AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_admin_role ON users (id, role)
WHERE
    role IN ('super_admin', 'team_admin')
    AND is_active = true;

CREATE INDEX IF NOT EXISTS idx_team_members_active ON team_members (user_id, team_id)
WHERE
    is_active = true
    AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_configs_by_user ON user_server_configs (user_id, server_id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_by_user_time ON audit_logs (user_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_sessions_active ON user_sessions (user_id, expires_at)
WHERE
    expires_at > CURRENT_TIMESTAMP;

-- ============================================================================
-- CREATE SECURITY BARRIER VIEWS FOR COMPLEX QUERIES
-- ============================================================================

-- Optimized view for user dashboard
CREATE OR REPLACE VIEW user_dashboard_view
WITH (security_barrier) AS
SELECT
    u.id,
    u.email,
    u.display_name,
    u.role,
    COUNT(DISTINCT usc.server_id) as server_count,
    COUNT(DISTINCT tm.team_id) as team_count,
    MAX(s.last_activity) as last_activity
FROM
    users u
    LEFT JOIN user_server_configs usc ON u.id = usc.user_id
    AND usc.deleted_at IS NULL
    LEFT JOIN team_members tm ON u.id = tm.user_id
    AND tm.is_active = true
    LEFT JOIN user_sessions s ON u.id = s.user_id
WHERE
    u.id = get_current_user_secure ()
GROUP BY
    u.id,
    u.email,
    u.display_name,
    u.role;

-- Grant appropriate permissions
GRANT SELECT ON user_dashboard_view TO portal_app;

-- ============================================================================
-- MONITORING AND PERFORMANCE TRACKING
-- ============================================================================

-- Create RLS performance monitoring function
CREATE OR REPLACE FUNCTION monitor_rls_performance()
RETURNS TABLE (
    table_name TEXT,
    policy_name TEXT,
    operation TEXT,
    estimated_cost FLOAT,
    index_used TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        schemaname || '.' || tablename as table_name,
        pol.polname::TEXT as policy_name,
        CASE pol.polcmd
            WHEN 'r' THEN 'SELECT'
            WHEN 'a' THEN 'INSERT'
            WHEN 'w' THEN 'UPDATE'
            WHEN 'd' THEN 'DELETE'
            ELSE 'ALL'
        END as operation,
        0.0::FLOAT as estimated_cost, -- Placeholder for actual cost
        ''::TEXT as index_used -- Placeholder for index usage
    FROM pg_policy pol
    JOIN pg_class c ON pol.polrelid = c.oid
    JOIN pg_tables t ON c.relname = t.tablename
    WHERE t.schemaname = 'public'
    ORDER BY table_name, policy_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- SECURITY AUDIT FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION audit_rls_security()
RETURNS TABLE (
    check_name TEXT,
    status TEXT,
    details TEXT
) AS $$
BEGIN
    -- Check if all sensitive tables have RLS enabled
    RETURN QUERY
    SELECT
        'RLS Enabled Check'::TEXT,
        CASE
            WHEN COUNT(*) = 0 THEN 'PASS'
            ELSE 'FAIL'
        END,
        CASE
            WHEN COUNT(*) = 0 THEN 'All sensitive tables have RLS enabled'
            ELSE 'Tables without RLS: ' || string_agg(tablename, ', ')
        END
    FROM pg_tables t
    LEFT JOIN pg_class c ON t.tablename = c.relname
    WHERE t.schemaname = 'public'
    AND t.tablename IN ('users', 'teams', 'user_server_configs', 'audit_logs', 'secrets')
    AND NOT c.relrowsecurity;

    -- Check for policies on RLS-enabled tables
    RETURN QUERY
    SELECT
        'RLS Policies Check'::TEXT,
        CASE
            WHEN COUNT(DISTINCT c.relname) = COUNT(DISTINCT p.polrelid) THEN 'PASS'
            ELSE 'WARNING'
        END,
        'Tables with policies: ' || COUNT(DISTINCT p.polrelid)::TEXT ||
        ' / Total RLS tables: ' || COUNT(DISTINCT c.oid)::TEXT
    FROM pg_class c
    LEFT JOIN pg_policy p ON c.oid = p.polrelid
    WHERE c.relrowsecurity;

    -- Check for bypass roles
    RETURN QUERY
    SELECT
        'BYPASSRLS Check'::TEXT,
        CASE
            WHEN COUNT(*) = 0 THEN 'PASS'
            ELSE 'WARNING'
        END,
        CASE
            WHEN COUNT(*) = 0 THEN 'No roles with BYPASSRLS'
            ELSE 'Roles with BYPASSRLS: ' || string_agg(rolname, ', ')
        END
    FROM pg_roles
    WHERE rolbypassrls = true
    AND rolname NOT IN ('postgres', 'rds_superuser'); -- Exclude system roles
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- GRANT PERMISSIONS
-- ============================================================================

-- Grant execute permissions on security functions
GRANT EXECUTE ON FUNCTION get_current_user_secure () TO portal_app;

GRANT EXECUTE ON FUNCTION is_admin (UUID) TO portal_app;

GRANT EXECUTE ON FUNCTION is_team_member (UUID, UUID) TO portal_app;

GRANT EXECUTE ON FUNCTION monitor_rls_performance () TO portal_app;

GRANT EXECUTE ON FUNCTION audit_rls_security () TO portal_admin;

-- ============================================================================
-- MIGRATION TRACKING
-- ============================================================================

INSERT INTO
    migration_history (
        version,
        description,
        applied_at
    )
VALUES (
        '002',
        'Enable RLS with performance optimizations',
        CURRENT_TIMESTAMP
    );

COMMIT;

-- ============================================================================
-- ROLLBACK SCRIPT (Save separately)
-- ============================================================================
/*
BEGIN;

-- Drop policies
DROP POLICY IF EXISTS users_self_view ON users;
DROP POLICY IF EXISTS users_self_update ON users;
DROP POLICY IF EXISTS users_admin_insert ON users;
DROP POLICY IF EXISTS users_admin_delete ON users;
DROP POLICY IF EXISTS teams_member_view ON teams;
DROP POLICY IF EXISTS teams_admin_update ON teams;
DROP POLICY IF EXISTS configs_user_access ON user_server_configs;
DROP POLICY IF EXISTS audit_logs_self_view ON audit_logs;
DROP POLICY IF EXISTS audit_logs_insert_only ON audit_logs;
DROP POLICY IF EXISTS sessions_user_access ON user_sessions;
DROP POLICY IF EXISTS secrets_owner_access ON secrets;

-- Disable RLS
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE teams DISABLE ROW LEVEL SECURITY;
ALTER TABLE team_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_servers DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_server_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE secrets DISABLE ROW LEVEL SECURITY;

-- Drop functions
DROP FUNCTION IF EXISTS get_current_user_secure();
DROP FUNCTION IF EXISTS is_admin(UUID);
DROP FUNCTION IF EXISTS is_team_member(UUID, UUID);
DROP FUNCTION IF EXISTS monitor_rls_performance();
DROP FUNCTION IF EXISTS audit_rls_security();

-- Drop views
DROP VIEW IF EXISTS user_dashboard_view;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_admin_role;
DROP INDEX IF EXISTS idx_team_members_active;
DROP INDEX IF EXISTS idx_configs_by_user;
DROP INDEX IF EXISTS idx_audit_logs_by_user_time;
DROP INDEX IF EXISTS idx_sessions_active;

DELETE FROM migration_history WHERE version = '002';

COMMIT;
*/