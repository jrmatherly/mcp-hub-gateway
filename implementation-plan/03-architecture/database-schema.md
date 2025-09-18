# Database Schema

## Overview

PostgreSQL database with Row-Level Security (RLS) for multi-tenant isolation and application-level encryption for sensitive data.

## Database Configuration

```sql
-- Database creation
CREATE DATABASE mcp_portal
    WITH
    OWNER = mcp_admin
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = 100;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";           -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";            -- Encryption functions
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";  -- Query performance
```

## Schema Design

### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    azure_oid VARCHAR(255) UNIQUE NOT NULL,  -- Azure Object ID
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'standard_user',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,

    CONSTRAINT valid_role CHECK (role IN ('super_admin', 'team_admin', 'standard_user'))
);

CREATE INDEX idx_users_azure_oid ON users(azure_oid);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role) WHERE is_active = true;
```

### MCP Servers Catalog

```sql
CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    image VARCHAR(500) NOT NULL,  -- Docker image
    catalog_type VARCHAR(50) NOT NULL DEFAULT 'predefined',
    category VARCHAR(100),
    tags TEXT[],
    metadata JSONB DEFAULT '{}',  -- Additional server config
    version VARCHAR(50),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,

    CONSTRAINT valid_catalog_type CHECK (catalog_type IN ('predefined', 'custom')),
    CONSTRAINT unique_server_name UNIQUE(name, catalog_type)
);

CREATE INDEX idx_mcp_servers_name ON mcp_servers(name);
CREATE INDEX idx_mcp_servers_catalog_type ON mcp_servers(catalog_type);
CREATE INDEX idx_mcp_servers_tags ON mcp_servers USING GIN(tags);
```

### User Server Configurations

```sql
CREATE TABLE user_server_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT false,
    container_id VARCHAR(255),  -- Docker container ID when running
    container_state VARCHAR(50),
    encrypted_config TEXT,  -- AES-256 encrypted sensitive config
    config_metadata JSONB DEFAULT '{}',  -- Non-sensitive config
    last_state_change TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_user_server UNIQUE(user_id, server_id),
    CONSTRAINT valid_container_state CHECK (
        container_state IN ('pending', 'creating', 'running', 'stopping', 'stopped', 'error')
    )
);

CREATE INDEX idx_user_configs_user_id ON user_server_configs(user_id);
CREATE INDEX idx_user_configs_server_id ON user_server_configs(server_id);
CREATE INDEX idx_user_configs_enabled ON user_server_configs(enabled) WHERE enabled = true;
CREATE INDEX idx_user_configs_container_state ON user_server_configs(container_state);
```

### Audit Logs

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_action CHECK (
        action IN (
            'login', 'logout', 'server_enabled', 'server_disabled',
            'config_updated', 'config_exported', 'config_imported',
            'bulk_enable', 'bulk_disable', 'user_created', 'user_updated',
            'role_changed', 'custom_server_added', 'custom_server_removed'
        )
    )
) PARTITION BY RANGE (timestamp);

-- Create monthly partitions
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Add more partitions as needed

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
```

### Sessions

```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token VARCHAR(255) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_sessions_token ON user_sessions(session_token) WHERE is_active = true;
CREATE INDEX idx_sessions_expires ON user_sessions(expires_at) WHERE is_active = true;
```

### Teams (Future Enhancement)

```sql
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_team_member UNIQUE(team_id, user_id)
);
```

## Row-Level Security (RLS)

### Enable RLS

```sql
-- Enable RLS on sensitive tables
ALTER TABLE user_server_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;

-- Create application role
CREATE ROLE portal_app;
GRANT CONNECT ON DATABASE mcp_portal TO portal_app;
GRANT USAGE ON SCHEMA public TO portal_app;
GRANT ALL ON ALL TABLES IN SCHEMA public TO portal_app;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO portal_app;
```

### RLS Policies

```sql
-- User can only see their own configurations
CREATE POLICY user_configs_isolation ON user_server_configs
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::UUID);

-- User can only see their own sessions
CREATE POLICY user_sessions_isolation ON user_sessions
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::UUID);

-- Admins can see all configurations
CREATE POLICY admin_configs_access ON user_server_configs
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id')::UUID
            AND role IN ('super_admin', 'team_admin')
        )
    );
```

## Encryption Functions

### Encryption Setup

```sql
-- Create encryption functions using pgcrypto
CREATE OR REPLACE FUNCTION encrypt_sensitive_data(
    data TEXT,
    key TEXT
) RETURNS TEXT AS $$
BEGIN
    RETURN encode(
        encrypt(
            data::bytea,
            key::bytea,
            'aes'
        ),
        'base64'
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION decrypt_sensitive_data(
    encrypted_data TEXT,
    key TEXT
) RETURNS TEXT AS $$
BEGIN
    RETURN convert_from(
        decrypt(
            decode(encrypted_data, 'base64'),
            key::bytea,
            'aes'
        ),
        'UTF8'
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

## Triggers and Functions

### Update Timestamp Trigger

```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_mcp_servers_updated_at
    BEFORE UPDATE ON mcp_servers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_user_configs_updated_at
    BEFORE UPDATE ON user_server_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

### Audit Log Function

```sql
CREATE OR REPLACE FUNCTION create_audit_log(
    p_user_id UUID,
    p_action VARCHAR(100),
    p_resource_type VARCHAR(50),
    p_resource_id UUID,
    p_details JSONB,
    p_ip_address INET,
    p_user_agent TEXT
) RETURNS UUID AS $$
DECLARE
    v_audit_id UUID;
BEGIN
    INSERT INTO audit_logs (
        user_id, action, resource_type, resource_id,
        details, ip_address, user_agent
    ) VALUES (
        p_user_id, p_action, p_resource_type, p_resource_id,
        p_details, p_ip_address, p_user_agent
    ) RETURNING id INTO v_audit_id;

    RETURN v_audit_id;
END;
$$ LANGUAGE plpgsql;
```

### Session Cleanup Function

```sql
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions
    WHERE expires_at < CURRENT_TIMESTAMP
    OR (last_activity < CURRENT_TIMESTAMP - INTERVAL '30 minutes' AND is_active = true);

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Schedule cleanup job (using pg_cron or external scheduler)
-- SELECT cron.schedule('cleanup-sessions', '*/15 * * * *', 'SELECT cleanup_expired_sessions();');
```

### Audit Log Retention

```sql
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_logs
    WHERE timestamp < CURRENT_TIMESTAMP - INTERVAL '30 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Schedule daily cleanup
-- SELECT cron.schedule('cleanup-audit-logs', '0 2 * * *', 'SELECT cleanup_old_audit_logs();');
```

## Indexes for Performance

```sql
-- Composite indexes for common queries
CREATE INDEX idx_user_configs_lookup
    ON user_server_configs(user_id, enabled)
    WHERE enabled = true;

CREATE INDEX idx_audit_logs_user_action
    ON audit_logs(user_id, action, timestamp DESC);

CREATE INDEX idx_sessions_active
    ON user_sessions(user_id, is_active, expires_at)
    WHERE is_active = true;

-- Partial indexes for performance
CREATE INDEX idx_active_users
    ON users(role, is_active)
    WHERE is_active = true;

CREATE INDEX idx_running_containers
    ON user_server_configs(container_state)
    WHERE container_state = 'running';
```

## Views

### User Dashboard View

```sql
CREATE VIEW user_dashboard AS
SELECT
    u.id AS user_id,
    u.email,
    u.display_name,
    u.role,
    COUNT(DISTINCT usc.server_id) AS total_servers,
    COUNT(DISTINCT CASE WHEN usc.enabled THEN usc.server_id END) AS enabled_servers,
    COUNT(DISTINCT CASE WHEN usc.container_state = 'running' THEN usc.server_id END) AS running_servers
FROM users u
LEFT JOIN user_server_configs usc ON u.id = usc.user_id
GROUP BY u.id, u.email, u.display_name, u.role;
```

### Server Statistics View

```sql
CREATE VIEW server_statistics AS
SELECT
    s.id AS server_id,
    s.name,
    s.catalog_type,
    COUNT(DISTINCT usc.user_id) AS total_users,
    COUNT(DISTINCT CASE WHEN usc.enabled THEN usc.user_id END) AS active_users,
    COUNT(DISTINCT CASE WHEN usc.container_state = 'running' THEN usc.user_id END) AS running_instances
FROM mcp_servers s
LEFT JOIN user_server_configs usc ON s.id = usc.server_id
GROUP BY s.id, s.name, s.catalog_type;
```

## Migration Scripts

### Initial Schema Migration (001_initial_schema.sql)

```sql
-- Run all CREATE TABLE statements
-- Run all CREATE INDEX statements
-- Run all CREATE FUNCTION statements
-- Run all CREATE TRIGGER statements
```

### Enable RLS Migration (002_enable_rls.sql)

```sql
-- Enable RLS on tables
-- Create policies
-- Grant permissions
```

### Seed Data Migration (003_seed_data.sql)

```sql
-- Insert predefined MCP servers
INSERT INTO mcp_servers (name, display_name, description, image, catalog_type, category)
VALUES
    ('github', 'GitHub', 'GitHub API integration', 'docker/mcp-github:latest', 'predefined', 'development'),
    ('gitlab', 'GitLab', 'GitLab API integration', 'docker/mcp-gitlab:latest', 'predefined', 'development'),
    ('jira', 'Jira', 'Atlassian Jira integration', 'docker/mcp-jira:latest', 'predefined', 'project-management'),
    ('slack', 'Slack', 'Slack messaging integration', 'docker/mcp-slack:latest', 'predefined', 'communication');
```

## Backup and Restore

### Backup Script

```bash
#!/bin/bash
# backup_database.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/postgres"
DATABASE="mcp_portal"

# Full backup
pg_dump -h localhost -U mcp_admin -d $DATABASE -F c -f "$BACKUP_DIR/full_backup_$DATE.dump"

# Schema only
pg_dump -h localhost -U mcp_admin -d $DATABASE --schema-only -f "$BACKUP_DIR/schema_$DATE.sql"

# Data only
pg_dump -h localhost -U mcp_admin -d $DATABASE --data-only -f "$BACKUP_DIR/data_$DATE.sql"
```

### Restore Script

```bash
#!/bin/bash
# restore_database.sh

BACKUP_FILE=$1
DATABASE="mcp_portal"

# Restore from custom format dump
pg_restore -h localhost -U mcp_admin -d $DATABASE -c $BACKUP_FILE
```

## Performance Tuning

### PostgreSQL Configuration

```ini
# postgresql.conf optimizations
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 512MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 10MB
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 8
max_parallel_workers_per_gather = 4
max_parallel_workers = 8
```

## Monitoring Queries

### Active Sessions

```sql
SELECT
    user_id,
    COUNT(*) as session_count,
    MAX(last_activity) as last_seen
FROM user_sessions
WHERE is_active = true
AND expires_at > CURRENT_TIMESTAMP
GROUP BY user_id;
```

### Container Status

```sql
SELECT
    container_state,
    COUNT(*) as count
FROM user_server_configs
WHERE container_id IS NOT NULL
GROUP BY container_state;
```

### Audit Summary

```sql
SELECT
    action,
    COUNT(*) as count,
    DATE_TRUNC('hour', timestamp) as hour
FROM audit_logs
WHERE timestamp > CURRENT_TIMESTAMP - INTERVAL '24 hours'
GROUP BY action, hour
ORDER BY hour DESC, count DESC;
```
