-- Migration: Create user configurations tables
-- This migration creates tables for storing user configurations and server configurations
-- with RLS (Row Level Security) enabled for multi-tenant isolation

-- Create user_configurations table
CREATE TABLE IF NOT EXISTS user_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('default', 'personal', 'team', 'environment', 'project')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'inactive', 'draft', 'archived')),
    owner_id UUID NOT NULL,
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'default',
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    version VARCHAR(50) DEFAULT '1.0.0',
    settings_encrypted BYTEA, -- AES-256-GCM encrypted JSON data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,

-- Constraints
CONSTRAINT uq_user_config_name_owner UNIQUE(name, owner_id),
    CONSTRAINT chk_name_format CHECK (name ~ '^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$'),
    CONSTRAINT chk_version_format CHECK (version ~ '^\d+\.\d+\.\d+$')
);

-- Create server_configurations table
CREATE TABLE IF NOT EXISTS server_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    server_id VARCHAR(255) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'configured',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

-- Constraints
CONSTRAINT uq_server_config_owner UNIQUE(owner_id, server_id),
    CONSTRAINT chk_server_id_format CHECK (server_id ~ '^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$')
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_configurations_owner_id ON user_configurations (owner_id);

CREATE INDEX IF NOT EXISTS idx_user_configurations_type ON user_configurations(type);

CREATE INDEX IF NOT EXISTS idx_user_configurations_status ON user_configurations (status);

CREATE INDEX IF NOT EXISTS idx_user_configurations_tenant_id ON user_configurations (tenant_id);

CREATE INDEX IF NOT EXISTS idx_user_configurations_active ON user_configurations (is_active)
WHERE
    is_active = true;

CREATE INDEX IF NOT EXISTS idx_user_configurations_default ON user_configurations (is_default)
WHERE
    is_default = true;

CREATE INDEX IF NOT EXISTS idx_user_configurations_name_search ON user_configurations USING gin (
    to_tsvector (
        'english',
        name || ' ' || COALESCE(display_name, '') || ' ' || COALESCE(description, '')
    )
);

CREATE INDEX IF NOT EXISTS idx_user_configurations_updated_at ON user_configurations (updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_server_configurations_owner_id ON server_configurations (owner_id);

CREATE INDEX IF NOT EXISTS idx_server_configurations_server_id ON server_configurations (server_id);

CREATE INDEX IF NOT EXISTS idx_server_configurations_status ON server_configurations (status);

-- Enable Row Level Security (RLS)
ALTER TABLE user_configurations ENABLE ROW LEVEL SECURITY;

ALTER TABLE server_configurations ENABLE ROW LEVEL SECURITY;

-- RLS Policies for user_configurations
CREATE POLICY user_configurations_isolation ON user_configurations
    FOR ALL
    TO authenticated
    USING (
        owner_id::text = current_setting('app.current_user_id', true)
        OR current_setting('app.current_user_role', true) = 'admin'
    )
    WITH CHECK (
        owner_id::text = current_setting('app.current_user_id', true)
        OR current_setting('app.current_user_role', true) = 'admin'
    );

-- RLS Policies for server_configurations
CREATE POLICY server_configurations_isolation ON server_configurations
    FOR ALL
    TO authenticated
    USING (
        owner_id::text = current_setting('app.current_user_id', true)
        OR current_setting('app.current_user_role', true) = 'admin'
    )
    WITH CHECK (
        owner_id::text = current_setting('app.current_user_id', true)
        OR current_setting('app.current_user_role', true) = 'admin'
    );

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_user_configurations_updated_at
    BEFORE UPDATE ON user_configurations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_server_configurations_updated_at
    BEFORE UPDATE ON server_configurations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default configuration types (for reference)
INSERT INTO
    user_configurations (
        name,
        display_name,
        description,
        type,
        status,
        owner_id,
        tenant_id,
        is_default,
        settings_encrypted
    )
VALUES (
        'system-default',
        'System Default Configuration',
        'Default system configuration template',
        'default',
        'active',
        '00000000-0000-0000-0000-000000000000', -- System user
        'system',
        true,
        NULL -- Will be populated by application
    ) ON CONFLICT (name, owner_id) DO NOTHING;

-- Create view for configuration statistics
CREATE OR REPLACE VIEW user_configuration_stats AS
SELECT
    owner_id,
    tenant_id,
    COUNT(*) as total_configs,
    COUNT(*) FILTER (
        WHERE
            is_active = true
    ) as active_configs,
    COUNT(*) FILTER (
        WHERE
            is_default = true
    ) as default_configs,
    COUNT(*) FILTER (
        WHERE
            type = 'personal'
    ) as personal_configs,
    COUNT(*) FILTER (
        WHERE
            type = 'team'
    ) as team_configs,
    COUNT(*) FILTER (
        WHERE
            type = 'project'
    ) as project_configs,
    COUNT(*) FILTER (
        WHERE
            status = 'draft'
    ) as draft_configs,
    MAX(updated_at) as last_updated,
    MIN(created_at) as first_created
FROM user_configurations
GROUP BY
    owner_id,
    tenant_id;

-- Grant appropriate permissions
GRANT
SELECT,
INSERT
,
UPDATE,
DELETE ON user_configurations TO authenticated;

GRANT
SELECT,
INSERT
,
UPDATE,
DELETE ON server_configurations TO authenticated;

GRANT SELECT ON user_configuration_stats TO authenticated;

-- Add comments for documentation
COMMENT ON
TABLE user_configurations IS 'Stores user-specific MCP gateway configurations with encrypted settings';

COMMENT ON
TABLE server_configurations IS 'Stores server-specific configurations and metadata for users';

COMMENT ON COLUMN user_configurations.settings_encrypted IS 'AES-256-GCM encrypted JSON containing sensitive configuration data';

COMMENT ON COLUMN user_configurations.type IS 'Configuration type: default, personal, team, environment, or project';

COMMENT ON COLUMN user_configurations.status IS 'Configuration status: active, inactive, draft, or archived';

COMMENT ON COLUMN server_configurations.config IS 'Server-specific configuration in JSONB format';

COMMENT ON COLUMN server_configurations.metadata IS 'Additional metadata about the server configuration';

-- Create configuration audit log table for tracking changes
CREATE TABLE IF NOT EXISTS user_configuration_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    config_id UUID NOT NULL,
    owner_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- CREATE, UPDATE, DELETE, ACTIVATE, DEACTIVATE
    old_values JSONB,
    new_values JSONB,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT NOW(),
        client_ip INET,
        user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_configuration_audit_config_id ON user_configuration_audit (config_id);

CREATE INDEX IF NOT EXISTS idx_user_configuration_audit_owner_id ON user_configuration_audit (owner_id);

CREATE INDEX IF NOT EXISTS idx_user_configuration_audit_changed_at ON user_configuration_audit (changed_at DESC);

-- Enable RLS on audit table
ALTER TABLE user_configuration_audit ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_configuration_audit_isolation ON user_configuration_audit
    FOR SELECT
    TO authenticated
    USING (
        owner_id::text = current_setting('app.current_user_id', true)
        OR current_setting('app.current_user_role', true) = 'admin'
    );

GRANT SELECT ON user_configuration_audit TO authenticated;

GRANT INSERT ON user_configuration_audit TO authenticated;

COMMENT ON
TABLE user_configuration_audit IS 'Audit trail for user configuration changes';