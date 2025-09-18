-- =============================================================================
-- Migration: 004_catalog_multi_user.sql
-- Purpose: Add multi-user catalog support with admin-controlled base catalogs
-- Date: 2025-09-18
-- =============================================================================

-- =============================================================================
-- Catalog Configuration Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS catalog_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    catalog_type VARCHAR(50) NOT NULL CHECK (catalog_type IN ('admin_base', 'user_personal', 'system_default')),
    catalog_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    is_enabled BOOLEAN DEFAULT true,
    is_mandatory BOOLEAN DEFAULT false, -- For admin catalogs that users cannot disable
    precedence INTEGER DEFAULT 100, -- Lower number = higher priority
    config_data JSONB NOT NULL DEFAULT '{}',
    parent_catalog_id UUID REFERENCES catalog_configs(id) ON DELETE SET NULL,
    source_url VARCHAR(500), -- Optional URL for remote catalog sources
    version VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),

-- Constraints
CONSTRAINT unique_user_catalog_name UNIQUE (user_id, catalog_name),
    CONSTRAINT check_admin_base_no_user CHECK (
        (catalog_type = 'admin_base' AND user_id IS NULL) OR
        (catalog_type != 'admin_base')
    ),
    CONSTRAINT check_user_personal_has_user CHECK (
        (catalog_type = 'user_personal' AND user_id IS NOT NULL) OR
        (catalog_type != 'user_personal')
    )
);

-- Indexes for performance
CREATE INDEX idx_catalog_user ON catalog_configs (user_id)
WHERE
    user_id IS NOT NULL;

CREATE INDEX idx_catalog_type ON catalog_configs (catalog_type);

CREATE INDEX idx_catalog_enabled ON catalog_configs (is_enabled)
WHERE
    is_enabled = true;

CREATE INDEX idx_catalog_precedence ON catalog_configs (precedence);

CREATE INDEX idx_catalog_parent ON catalog_configs (parent_catalog_id)
WHERE
    parent_catalog_id IS NOT NULL;

-- =============================================================================
-- Catalog Server Entries Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS catalog_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id UUID NOT NULL REFERENCES catalog_configs(id) ON DELETE CASCADE,
    server_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    image VARCHAR(500) NOT NULL,
    tag VARCHAR(100) DEFAULT 'latest',
    environment JSONB DEFAULT '{}',
    volumes JSONB DEFAULT '[]',
    ports JSONB DEFAULT '[]',
    command TEXT[],
    is_enabled BOOLEAN DEFAULT true,
    is_override BOOLEAN DEFAULT false, -- True if this overrides a base catalog entry
    overrides_server VARCHAR(255), -- Name of the base server being overridden
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

-- Constraints
CONSTRAINT unique_catalog_server UNIQUE (catalog_id, server_name) );

-- Indexes
CREATE INDEX idx_server_catalog ON catalog_servers (catalog_id);

CREATE INDEX idx_server_name ON catalog_servers (server_name);

CREATE INDEX idx_server_enabled ON catalog_servers (is_enabled)
WHERE
    is_enabled = true;

CREATE INDEX idx_server_override ON catalog_servers (overrides_server)
WHERE
    overrides_server IS NOT NULL;

-- =============================================================================
-- User Catalog Customizations Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS user_catalog_customizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_catalog_id UUID REFERENCES catalog_configs(id) ON DELETE CASCADE,
    base_server_name VARCHAR(255),
    action VARCHAR(50) NOT NULL CHECK (action IN ('disable', 'override', 'add')),
    custom_data JSONB DEFAULT '{}',
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

-- Constraints
CONSTRAINT unique_user_customization UNIQUE (user_id, base_catalog_id, base_server_name, action),
    CONSTRAINT check_action_requirements CHECK (
        (action = 'disable' AND base_server_name IS NOT NULL) OR
        (action = 'override' AND base_server_name IS NOT NULL AND custom_data IS NOT NULL) OR
        (action = 'add' AND custom_data IS NOT NULL)
    )
);

-- Indexes
CREATE INDEX idx_customization_user ON user_catalog_customizations (user_id);

CREATE INDEX idx_customization_catalog ON user_catalog_customizations (base_catalog_id);

CREATE INDEX idx_customization_server ON user_catalog_customizations (base_server_name);

CREATE INDEX idx_customization_action ON user_catalog_customizations (action);

-- =============================================================================
-- Catalog Audit Log Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS catalog_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID REFERENCES users (id),
    catalog_id UUID REFERENCES catalog_configs (id) ON DELETE SET NULL,
    server_id UUID REFERENCES catalog_servers (id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    created_at TIMESTAMP
    WITH
        TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_audit_user ON catalog_audit_log (user_id);

CREATE INDEX idx_audit_catalog ON catalog_audit_log (catalog_id);

CREATE INDEX idx_audit_action ON catalog_audit_log (action);

CREATE INDEX idx_audit_created ON catalog_audit_log (created_at DESC);

-- =============================================================================
-- Row Level Security (RLS)
-- =============================================================================

-- Enable RLS on all tables
ALTER TABLE catalog_configs ENABLE ROW LEVEL SECURITY;

ALTER TABLE catalog_servers ENABLE ROW LEVEL SECURITY;

ALTER TABLE user_catalog_customizations ENABLE ROW LEVEL SECURITY;

ALTER TABLE catalog_audit_log ENABLE ROW LEVEL SECURITY;

-- Catalog Configs Policies
CREATE POLICY catalog_configs_admin_all ON catalog_configs
    FOR ALL
    TO authenticated
    USING (
        -- Admins can see and manage all catalogs
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.user_id')::UUID
            AND role = 'admin'
        )
    );

CREATE POLICY catalog_configs_user_read ON catalog_configs
    FOR SELECT
    TO authenticated
    USING (
        -- Users can see admin base catalogs and their own personal catalogs
        catalog_type IN ('admin_base', 'system_default') OR
        user_id = current_setting('app.user_id')::UUID
    );

CREATE POLICY catalog_configs_user_write ON catalog_configs
    FOR INSERT
    TO authenticated
    WITH CHECK (
        -- Users can only create their own personal catalogs
        catalog_type = 'user_personal' AND
        user_id = current_setting('app.user_id')::UUID
    );

CREATE POLICY catalog_configs_user_update ON catalog_configs
    FOR UPDATE
    TO authenticated
    USING (
        -- Users can only update their own personal catalogs
        catalog_type = 'user_personal' AND
        user_id = current_setting('app.user_id')::UUID
    );

CREATE POLICY catalog_configs_user_delete ON catalog_configs
    FOR DELETE
    TO authenticated
    USING (
        -- Users can only delete their own personal catalogs
        catalog_type = 'user_personal' AND
        user_id = current_setting('app.user_id')::UUID
    );

-- Catalog Servers Policies
CREATE POLICY catalog_servers_admin_all ON catalog_servers
    FOR ALL
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.user_id')::UUID
            AND role = 'admin'
        )
    );

CREATE POLICY catalog_servers_user_read ON catalog_servers
    FOR SELECT
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM catalog_configs cc
            WHERE cc.id = catalog_id
            AND (
                cc.catalog_type IN ('admin_base', 'system_default') OR
                cc.user_id = current_setting('app.user_id')::UUID
            )
        )
    );

CREATE POLICY catalog_servers_user_write ON catalog_servers
    FOR INSERT
    TO authenticated
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM catalog_configs cc
            WHERE cc.id = catalog_id
            AND cc.catalog_type = 'user_personal'
            AND cc.user_id = current_setting('app.user_id')::UUID
        )
    );

CREATE POLICY catalog_servers_user_update ON catalog_servers
    FOR UPDATE
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM catalog_configs cc
            WHERE cc.id = catalog_id
            AND cc.catalog_type = 'user_personal'
            AND cc.user_id = current_setting('app.user_id')::UUID
        )
    );

CREATE POLICY catalog_servers_user_delete ON catalog_servers
    FOR DELETE
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM catalog_configs cc
            WHERE cc.id = catalog_id
            AND cc.catalog_type = 'user_personal'
            AND cc.user_id = current_setting('app.user_id')::UUID
        )
    );

-- User Customizations Policies
CREATE POLICY customizations_user_all ON user_catalog_customizations
    FOR ALL
    TO authenticated
    USING (user_id = current_setting('app.user_id')::UUID);

-- Audit Log Policies
CREATE POLICY audit_admin_read ON catalog_audit_log
    FOR SELECT
    TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.user_id')::UUID
            AND role = 'admin'
        )
    );

CREATE POLICY audit_user_read ON catalog_audit_log
    FOR SELECT
    TO authenticated
    USING (
        user_id = current_setting('app.user_id')::UUID
    );

-- =============================================================================
-- Helper Functions
-- =============================================================================

-- Function to merge catalogs for a user
CREATE OR REPLACE FUNCTION get_merged_catalog_for_user(p_user_id UUID)
RETURNS TABLE (
    server_name VARCHAR(255),
    display_name VARCHAR(255),
    description TEXT,
    image VARCHAR(500),
    tag VARCHAR(100),
    environment JSONB,
    volumes JSONB,
    ports JSONB,
    command TEXT[],
    is_enabled BOOLEAN,
    source_catalog VARCHAR(50),
    precedence INTEGER
) AS $$
BEGIN
    RETURN QUERY
    WITH base_servers AS (
        -- Get all admin base catalog servers
        SELECT
            cs.server_name,
            cs.display_name,
            cs.description,
            cs.image,
            cs.tag,
            cs.environment,
            cs.volumes,
            cs.ports,
            cs.command,
            cs.is_enabled AND NOT cc.is_mandatory AS is_enabled,
            'admin_base'::VARCHAR(50) as source_catalog,
            cc.precedence
        FROM catalog_servers cs
        JOIN catalog_configs cc ON cs.catalog_id = cc.id
        WHERE cc.catalog_type = 'admin_base'
        AND cc.is_enabled = true
        AND cs.is_enabled = true
    ),
    user_disabled AS (
        -- Get user disabled servers
        SELECT base_server_name
        FROM user_catalog_customizations
        WHERE user_id = p_user_id
        AND action = 'disable'
    ),
    user_overrides AS (
        -- Get user overridden servers
        SELECT
            ucc.base_server_name as server_name,
            (ucc.custom_data->>'display_name')::VARCHAR(255) as display_name,
            (ucc.custom_data->>'description')::TEXT as description,
            (ucc.custom_data->>'image')::VARCHAR(500) as image,
            COALESCE((ucc.custom_data->>'tag')::VARCHAR(100), 'latest') as tag,
            COALESCE(ucc.custom_data->'environment', '{}'::JSONB) as environment,
            COALESCE(ucc.custom_data->'volumes', '[]'::JSONB) as volumes,
            COALESCE(ucc.custom_data->'ports', '[]'::JSONB) as ports,
            NULL::TEXT[] as command,
            true as is_enabled,
            'user_override'::VARCHAR(50) as source_catalog,
            50 as precedence
        FROM user_catalog_customizations ucc
        WHERE ucc.user_id = p_user_id
        AND ucc.action = 'override'
    ),
    user_additions AS (
        -- Get user added servers
        SELECT
            cs.server_name,
            cs.display_name,
            cs.description,
            cs.image,
            cs.tag,
            cs.environment,
            cs.volumes,
            cs.ports,
            cs.command,
            cs.is_enabled,
            'user_personal'::VARCHAR(50) as source_catalog,
            cc.precedence
        FROM catalog_servers cs
        JOIN catalog_configs cc ON cs.catalog_id = cc.id
        WHERE cc.user_id = p_user_id
        AND cc.catalog_type = 'user_personal'
        AND cc.is_enabled = true
        AND cs.is_enabled = true
    )
    -- Combine all sources with precedence
    SELECT DISTINCT ON (combined.server_name)
        combined.server_name,
        combined.display_name,
        combined.description,
        combined.image,
        combined.tag,
        combined.environment,
        combined.volumes,
        combined.ports,
        combined.command,
        combined.is_enabled,
        combined.source_catalog,
        combined.precedence
    FROM (
        SELECT * FROM user_overrides
        UNION ALL
        SELECT * FROM user_additions
        UNION ALL
        SELECT * FROM base_servers bs
        WHERE bs.server_name NOT IN (SELECT base_server_name FROM user_disabled)
        AND bs.server_name NOT IN (SELECT server_name FROM user_overrides)
    ) combined
    ORDER BY combined.server_name, combined.precedence;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- =============================================================================
-- Triggers
-- =============================================================================

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_catalog_configs_updated_at BEFORE UPDATE ON catalog_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_catalog_servers_updated_at BEFORE UPDATE ON catalog_servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_customizations_updated_at BEFORE UPDATE ON user_catalog_customizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Initial Data (Optional)
-- =============================================================================

-- Insert system default catalog if it doesn't exist
INSERT INTO catalog_configs (
    catalog_type,
    catalog_name,
    display_name,
    description,
    is_enabled,
    is_mandatory,
    precedence,
    config_data
) VALUES (
    'system_default',
    'mcp-default',
    'MCP Default Catalog',
    'Default MCP server catalog provided by the system',
    true,
    true,
    1000,
    '{}'::JSONB
) ON CONFLICT (user_id, catalog_name) DO NOTHING;

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON
TABLE catalog_configs IS 'Stores catalog configurations for admin base and user personal catalogs';

COMMENT ON
TABLE catalog_servers IS 'Stores individual server entries within catalogs';

COMMENT ON
TABLE user_catalog_customizations IS 'Stores user-specific customizations to base catalogs';

COMMENT ON
TABLE catalog_audit_log IS 'Audit trail for catalog modifications';

COMMENT ON FUNCTION get_merged_catalog_for_user IS 'Returns merged catalog for a specific user applying inheritance rules';