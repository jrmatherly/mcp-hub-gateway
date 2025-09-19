-- Feature Flag System Database Migrations
-- Create tables for storing feature flags, experiments, and participant data

-- ========================================
-- Feature Flag Configuration Table
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_configuration (
    id SERIAL PRIMARY KEY,
    version INTEGER NOT NULL DEFAULT 1,
    configuration JSONB NOT NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'development',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        created_by UUID NOT NULL
);

-- Ensure only one active configuration per environment
CREATE UNIQUE INDEX idx_feature_flag_config_active_env ON feature_flag_configuration (environment, active)
WHERE
    active = true;

-- ========================================
-- Feature Flags Table
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flags (
    name VARCHAR(100) PRIMARY KEY,
    type VARCHAR(20) NOT NULL DEFAULT 'boolean',
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT false,
    default_value JSONB,
    rollout_percentage INTEGER NOT NULL DEFAULT 0 CHECK (
        rollout_percentage >= 0
        AND rollout_percentage <= 100
    ),
    user_overrides JSONB DEFAULT '{}',
    server_overrides JSONB DEFAULT '{}',
    variants JSONB DEFAULT '[]',
    rules JSONB DEFAULT '[]',
    rollout_config JSONB,
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        created_by UUID NOT NULL,
        version INTEGER NOT NULL DEFAULT 1,
        tags JSONB DEFAULT '[]',
        deprecated BOOLEAN NOT NULL DEFAULT false,
        deleted_at TIMESTAMP
    WITH
        TIME ZONE
);

-- Indexes for efficient querying
CREATE INDEX idx_feature_flags_enabled ON feature_flags (enabled)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_feature_flags_type ON feature_flags (type) WHERE deleted_at IS NULL;

CREATE INDEX idx_feature_flags_updated_at ON feature_flags (updated_at)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_feature_flags_tags ON feature_flags USING GIN (tags)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_feature_flags_rollout_percentage ON feature_flags (rollout_percentage)
WHERE
    deleted_at IS NULL;

-- ========================================
-- Feature Flag Groups Table
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_groups (
    name VARCHAR(100) PRIMARY KEY,
    description TEXT,
    flags JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        deleted_at TIMESTAMP
    WITH
        TIME ZONE
);

-- ========================================
-- Feature Flag Experiments Table
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_experiments (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (
        status IN (
            'draft',
            'running',
            'paused',
            'completed'
        )
    ),
    flag_name VARCHAR(100) NOT NULL REFERENCES feature_flags (name),
    variants JSONB NOT NULL DEFAULT '[]',
    audience_filter JSONB,
    traffic_allocation INTEGER NOT NULL DEFAULT 100 CHECK (
        traffic_allocation >= 0
        AND traffic_allocation <= 100
    ),
    start_time TIMESTAMP
    WITH
        TIME ZONE,
        end_time TIMESTAMP
    WITH
        TIME ZONE,
        duration_seconds BIGINT,
        primary_metric VARCHAR(100),
        secondary_metrics JSONB DEFAULT '[]',
        results JSONB,
        created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        created_by UUID NOT NULL,
        deleted_at TIMESTAMP
    WITH
        TIME ZONE
);

-- Indexes for experiments
CREATE INDEX idx_experiments_status ON feature_flag_experiments (status)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_experiments_flag_name ON feature_flag_experiments (flag_name)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_experiments_start_time ON feature_flag_experiments (start_time)
WHERE
    deleted_at IS NULL;

CREATE INDEX idx_experiments_created_by ON feature_flag_experiments (created_by)
WHERE
    deleted_at IS NULL;

-- ========================================
-- Experiment Participants Table
-- ========================================
CREATE TABLE IF NOT EXISTS experiment_participants (
    experiment_id VARCHAR(100) NOT NULL REFERENCES feature_flag_experiments (id),
    user_id UUID NOT NULL,
    variant VARCHAR(100) NOT NULL,
    assigned_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        PRIMARY KEY (experiment_id, user_id)
);

-- Index for participant queries
CREATE INDEX idx_experiment_participants_user_id ON experiment_participants (user_id);

CREATE INDEX idx_experiment_participants_variant ON experiment_participants (experiment_id, variant);

CREATE INDEX idx_experiment_participants_assigned_at ON experiment_participants (assigned_at);

-- ========================================
-- Feature Flag Evaluations Table (for detailed tracking)
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    flag_name VARCHAR(100) NOT NULL,
    user_id UUID,
    tenant_id VARCHAR(100),
    server_name VARCHAR(200),
    enabled BOOLEAN NOT NULL,
    value JSONB,
    variant VARCHAR(100),
    reason VARCHAR(100),
    rule_matched VARCHAR(100),
    evaluation_time_ms INTEGER,
    cache_hit BOOLEAN NOT NULL DEFAULT false,
    request_id VARCHAR(100),
    remote_addr INET,
    user_agent TEXT,
    environment VARCHAR(50),
    evaluated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Partitioning by date for performance (optional)
-- CREATE TABLE feature_flag_evaluations_y2024m01 PARTITION OF feature_flag_evaluations
-- FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Indexes for evaluation queries
CREATE INDEX idx_evaluations_flag_name ON feature_flag_evaluations (flag_name, evaluated_at);

CREATE INDEX idx_evaluations_user_id ON feature_flag_evaluations (user_id, evaluated_at);

CREATE INDEX idx_evaluations_server_name ON feature_flag_evaluations (server_name, evaluated_at);

CREATE INDEX idx_evaluations_tenant_id ON feature_flag_evaluations (tenant_id, evaluated_at);

CREATE INDEX idx_evaluations_evaluated_at ON feature_flag_evaluations (evaluated_at);

-- ========================================
-- Feature Flag Audit Log Table
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    action VARCHAR(20) NOT NULL CHECK (
        action IN (
            'create',
            'update',
            'delete',
            'evaluate'
        )
    ),
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(200) NOT NULL,
    user_id VARCHAR(200),
    changes JSONB,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for audit queries
CREATE INDEX idx_audit_log_entity ON feature_flag_audit_log (
    entity_type,
    entity_id,
    timestamp
);

CREATE INDEX idx_audit_log_user_id ON feature_flag_audit_log (user_id, timestamp);

CREATE INDEX idx_audit_log_action ON feature_flag_audit_log (action, timestamp);

CREATE INDEX idx_audit_log_timestamp ON feature_flag_audit_log (timestamp);

-- ========================================
-- Feature Flag Metrics Table (aggregated)
-- ========================================
CREATE TABLE IF NOT EXISTS feature_flag_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    flag_name VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    hour INTEGER CHECK (
        hour >= 0
        AND hour <= 23
    ),
    total_evaluations BIGINT NOT NULL DEFAULT 0,
    true_evaluations BIGINT NOT NULL DEFAULT 0,
    false_evaluations BIGINT NOT NULL DEFAULT 0,
    unique_users BIGINT NOT NULL DEFAULT 0,
    avg_evaluation_time_ms DECIMAL(10, 2),
    max_evaluation_time_ms INTEGER,
    cache_hit_rate DECIMAL(5, 2),
    error_count BIGINT NOT NULL DEFAULT 0,
    variant_counts JSONB DEFAULT '{}',
    rule_matches JSONB DEFAULT '{}',
    created_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP
    WITH
        TIME ZONE NOT NULL DEFAULT NOW(),
        UNIQUE (flag_name, date, hour)
);

-- Indexes for metrics queries
CREATE INDEX idx_metrics_flag_date ON feature_flag_metrics (flag_name, date);

CREATE INDEX idx_metrics_date ON feature_flag_metrics (date);

-- ========================================
-- OAuth Feature Flag Integration Views
-- ========================================

-- View for OAuth-specific flags
CREATE OR REPLACE VIEW oauth_feature_flags AS
SELECT
    name,
    enabled,
    rollout_percentage,
    user_overrides,
    server_overrides,
    updated_at
FROM feature_flags
WHERE
    name LIKE 'oauth_%'
    AND deleted_at IS NULL;

-- View for active experiments
CREATE OR REPLACE VIEW active_experiments AS
SELECT
    id,
    name,
    flag_name,
    status,
    traffic_allocation,
    start_time,
    end_time,
    (
        SELECT COUNT(*)
        FROM experiment_participants ep
        WHERE
            ep.experiment_id = e.id
    ) as participant_count
FROM feature_flag_experiments e
WHERE
    status = 'running'
    AND deleted_at IS NULL;

-- ========================================
-- Functions and Triggers
-- ========================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_flag_experiments_updated_at
    BEFORE UPDATE ON feature_flag_experiments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_flag_groups_updated_at
    BEFORE UPDATE ON feature_flag_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_flag_configuration_updated_at
    BEFORE UPDATE ON feature_flag_configuration
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_flag_metrics_updated_at
    BEFORE UPDATE ON feature_flag_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to aggregate evaluation metrics
CREATE OR REPLACE FUNCTION aggregate_evaluation_metrics()
RETURNS VOID AS $$
BEGIN
    -- Aggregate hourly metrics from the evaluations table
    INSERT INTO feature_flag_metrics (
        flag_name,
        date,
        hour,
        total_evaluations,
        true_evaluations,
        false_evaluations,
        unique_users,
        avg_evaluation_time_ms,
        max_evaluation_time_ms,
        cache_hit_rate,
        error_count
    )
    SELECT
        flag_name,
        DATE(evaluated_at) as date,
        EXTRACT(hour FROM evaluated_at)::INTEGER as hour,
        COUNT(*) as total_evaluations,
        COUNT(*) FILTER (WHERE enabled = true) as true_evaluations,
        COUNT(*) FILTER (WHERE enabled = false) as false_evaluations,
        COUNT(DISTINCT user_id) as unique_users,
        AVG(evaluation_time_ms) as avg_evaluation_time_ms,
        MAX(evaluation_time_ms) as max_evaluation_time_ms,
        (COUNT(*) FILTER (WHERE cache_hit = true)::DECIMAL / COUNT(*) * 100) as cache_hit_rate,
        0 as error_count -- Would be populated from error tracking
    FROM feature_flag_evaluations
    WHERE evaluated_at >= NOW() - INTERVAL '1 hour'
    AND evaluated_at < NOW()
    GROUP BY flag_name, DATE(evaluated_at), EXTRACT(hour FROM evaluated_at)
    ON CONFLICT (flag_name, date, hour) DO UPDATE SET
        total_evaluations = EXCLUDED.total_evaluations,
        true_evaluations = EXCLUDED.true_evaluations,
        false_evaluations = EXCLUDED.false_evaluations,
        unique_users = EXCLUDED.unique_users,
        avg_evaluation_time_ms = EXCLUDED.avg_evaluation_time_ms,
        max_evaluation_time_ms = EXCLUDED.max_evaluation_time_ms,
        cache_hit_rate = EXCLUDED.cache_hit_rate,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- Initial OAuth Feature Flags Data
-- ========================================

-- Insert default OAuth feature flags
INSERT INTO
    feature_flags (
        name,
        type,
        description,
        enabled,
        default_value,
        created_by,
        tags
    )
VALUES (
        'oauth_enabled',
        'boolean',
        'Master switch for OAuth functionality',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "security"]'
    ),
    (
        'oauth_auto_401',
        'boolean',
        'Automatic 401 detection and token refresh',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "automation"]'
    ),
    (
        'oauth_dcr',
        'boolean',
        'Dynamic Client Registration support',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "dcr"]'
    ),
    (
        'oauth_provider_github',
        'boolean',
        'GitHub OAuth provider support',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "provider", "github"]'
    ),
    (
        'oauth_provider_google',
        'boolean',
        'Google OAuth provider support',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "provider", "google"]'
    ),
    (
        'oauth_provider_microsoft',
        'boolean',
        'Microsoft OAuth provider support',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "provider", "microsoft"]'
    ),
    (
        'oauth_docker_secrets',
        'boolean',
        'Docker Desktop secrets integration',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "docker", "secrets"]'
    ),
    (
        'oauth_token_refresh',
        'boolean',
        'Automatic token refresh',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "tokens"]'
    ),
    (
        'oauth_token_storage',
        'boolean',
        'Token storage functionality',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "tokens", "storage"]'
    ),
    (
        'oauth_jwt_validation',
        'boolean',
        'JWT token validation',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "jwt", "security"]'
    ),
    (
        'oauth_https_required',
        'boolean',
        'Require HTTPS for OAuth',
        true,
        'true',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "security", "https"]'
    ),
    (
        'oauth_audit_logging',
        'boolean',
        'OAuth audit logging',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "audit", "logging"]'
    ),
    (
        'oauth_metrics',
        'boolean',
        'OAuth metrics collection',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "metrics"]'
    ),
    (
        'oauth_key_rotation',
        'boolean',
        'Automatic key rotation',
        false,
        'false',
        '00000000-0000-0000-0000-000000000000',
        '["oauth", "security", "rotation"]'
    ) ON CONFLICT (name) DO NOTHING;

-- Create OAuth feature flag group
INSERT INTO feature_flag_groups (name, description, flags) VALUES
('oauth_features', 'OAuth authentication and authorization features', '[
    "oauth_enabled",
    "oauth_auto_401",
    "oauth_dcr",
    "oauth_provider_github",
    "oauth_provider_google",
    "oauth_provider_microsoft",
    "oauth_docker_secrets",
    "oauth_token_refresh",
    "oauth_token_storage",
    "oauth_jwt_validation",
    "oauth_https_required",
    "oauth_audit_logging",
    "oauth_metrics",
    "oauth_key_rotation"
]'::jsonb)
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    flags = EXCLUDED.flags,
    updated_at = NOW();

-- Insert initial configuration
INSERT INTO feature_flag_configuration (version, configuration, environment, created_by) VALUES
(1, '{
    "version": 1,
    "environment": "development",
    "global_settings": {
        "default_enabled": false,
        "default_cache_ttl": "5m",
        "evaluation_timeout": "1s",
        "default_rollout_percentage": 0,
        "default_rollout_strategy": "percentage",
        "metrics_enabled": true,
        "metrics_interval": "1m",
        "tracking_enabled": true,
        "failure_mode": "fail_closed",
        "max_evaluation_time": "2s",
        "circuit_breaker_enabled": true
    }
}'::jsonb, 'development', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (environment, active) WHERE active = true DO NOTHING;

-- ========================================
-- Performance Optimization
-- ========================================

-- Create partial indexes for active flags only
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feature_flags_active_enabled ON feature_flags (name)
WHERE
    enabled = true
    AND deleted_at IS NULL;

-- Create composite index for evaluation queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_evaluations_composite ON feature_flag_evaluations (
    flag_name,
    user_id,
    evaluated_at DESC
);

-- ========================================
-- Data Retention Policy
-- ========================================

-- Function to clean up old evaluation data
CREATE OR REPLACE FUNCTION cleanup_old_evaluations(retention_days INTEGER DEFAULT 30)
RETURNS BIGINT AS $$
DECLARE
    deleted_count BIGINT;
BEGIN
    DELETE FROM feature_flag_evaluations
    WHERE evaluated_at < NOW() - INTERVAL '1 day' * retention_days;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    -- Log the cleanup
    INSERT INTO feature_flag_audit_log (action, entity_type, entity_id, metadata)
    VALUES ('delete', 'evaluation_cleanup', 'system', jsonb_build_object('deleted_count', deleted_count));

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- Security and Permissions
-- ========================================

-- Create role for feature flag management
-- Note: This would typically be done by a DBA
-- CREATE ROLE feature_flag_admin;
-- CREATE ROLE feature_flag_readonly;

-- Grant permissions (examples)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO feature_flag_admin;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO feature_flag_readonly;
-- GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO feature_flag_admin;