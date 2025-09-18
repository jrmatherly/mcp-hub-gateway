package userconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

// userConfigRepository implements UserConfigRepository interface
type userConfigRepository struct {
	pool       *pgxpool.Pool
	encryption crypto.Encryption
	masterKey  []byte // Master key for encryption
}

// CreateUserConfigRepository creates a new UserConfigRepository instance
func CreateUserConfigRepository(encryption crypto.Encryption) (UserConfigRepository, error) {
	pool := database.GetPool()
	if pool == nil {
		return nil, fmt.Errorf("failed to get database pool: pool is nil")
	}

	// Generate a master key for this repository instance
	masterKey, err := encryption.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	return &userConfigRepository{
		pool:       pool,
		encryption: encryption,
		masterKey:  masterKey,
	}, nil
}

// CreateConfig creates a new user configuration in the database
func (r *userConfigRepository) CreateConfig(
	ctx context.Context,
	userID string,
	config *UserConfig,
) error {
	// Encrypt settings
	settingsJSON, err := json.Marshal(config.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	encryptedData, err := r.encryption.Encrypt(settingsJSON, r.masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt settings: %w", err)
	}

	// Convert EncryptedData to Base64 for database storage
	encryptedSettings, err := encryptedData.ToBase64()
	if err != nil {
		return fmt.Errorf("failed to encode encrypted data: %w", err)
	}

	query := `
		INSERT INTO user_configurations (
			id, name, display_name, description, type, status,
			owner_id, tenant_id, is_default, is_active, version,
			settings_encrypted, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err = r.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.DisplayName,
		config.Description,
		string(config.Type),
		string(config.Status),
		config.OwnerID,
		config.TenantID,
		config.IsDefault,
		config.IsActive,
		config.Version,
		encryptedSettings,
		config.CreatedAt,
		config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user configuration: %w", err)
	}

	return nil
}

// GetConfig retrieves a user configuration by ID
func (r *userConfigRepository) GetConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*UserConfig, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status,
			owner_id, tenant_id, is_default, is_active, version,
			settings_encrypted, created_at, updated_at, last_used_at
		FROM user_configurations
		WHERE id = $1 AND owner_id = $2`

	var config UserConfig
	var encryptedSettings []byte
	var lastUsedAt *time.Time

	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&config.ID,
		&config.Name,
		&config.DisplayName,
		&config.Description,
		&config.Type,
		&config.Status,
		&config.OwnerID,
		&config.TenantID,
		&config.IsDefault,
		&config.IsActive,
		&config.Version,
		&encryptedSettings,
		&config.CreatedAt,
		&config.UpdatedAt,
		&lastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("configuration not found")
		}
		return nil, fmt.Errorf("failed to query user configuration: %w", err)
	}

	config.LastUsedAt = lastUsedAt

	// Decrypt settings
	if len(encryptedSettings) > 0 {
		// Parse Base64 encoded EncryptedData
		encryptedData, err := crypto.FromBase64(string(encryptedSettings))
		if err != nil {
			return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
		}

		decryptedSettings, err := r.encryption.Decrypt(encryptedData, r.masterKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt settings: %w", err)
		}

		if err := json.Unmarshal(decryptedSettings, &config.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &config, nil
}

// UpdateConfig updates an existing user configuration
func (r *userConfigRepository) UpdateConfig(
	ctx context.Context,
	userID string,
	config *UserConfig,
) error {
	// Encrypt settings
	settingsJSON, err := json.Marshal(config.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	encryptedData, err := r.encryption.Encrypt(settingsJSON, r.masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt settings: %w", err)
	}

	// Convert EncryptedData to Base64 for database storage
	encryptedSettings, err := encryptedData.ToBase64()
	if err != nil {
		return fmt.Errorf("failed to encode encrypted data: %w", err)
	}

	query := `
		UPDATE user_configurations SET
			display_name = $3,
			description = $4,
			status = $5,
			is_default = $6,
			is_active = $7,
			version = $8,
			settings_encrypted = $9,
			updated_at = $10
		WHERE id = $1 AND owner_id = $2`

	result, err := r.pool.Exec(ctx, query,
		config.ID,
		config.OwnerID,
		config.DisplayName,
		config.Description,
		string(config.Status),
		config.IsDefault,
		config.IsActive,
		config.Version,
		encryptedSettings,
		config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user configuration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("configuration not found or no permission")
	}

	return nil
}

// DeleteConfig deletes a user configuration
func (r *userConfigRepository) DeleteConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) error {
	query := `DELETE FROM user_configurations WHERE id = $1 AND owner_id = $2`

	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user configuration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("configuration not found or no permission")
	}

	return nil
}

// ListConfigs lists user configurations with filtering and pagination
func (r *userConfigRepository) ListConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) ([]*UserConfig, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status,
			owner_id, tenant_id, is_default, is_active, version,
			settings_encrypted, created_at, updated_at, last_used_at
		FROM user_configurations
		WHERE owner_id = $1`

	args := []interface{}{userID}
	argIndex := 2

	// Apply filters
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(filter.Type))
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(filter.Status))
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsDefault != nil {
		query += fmt.Sprintf(" AND is_default = $%d", argIndex)
		args = append(args, *filter.IsDefault)
		argIndex++
	}

	if filter.NamePattern != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+filter.NamePattern+"%")
		argIndex++
	}

	// Add ordering
	query += " ORDER BY "
	switch filter.SortBy {
	case "name":
		query += "name"
	case "created_at":
		query += "created_at"
	case "updated_at":
		query += "updated_at"
	default:
		query += "updated_at"
	}

	if filter.SortOrder == "ASC" {
		query += " ASC"
	} else {
		query += " DESC"
	}

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
			argIndex++
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user configurations: %w", err)
	}
	defer rows.Close()

	var configs []*UserConfig
	for rows.Next() {
		var config UserConfig
		var encryptedSettings []byte
		var lastUsedAt *time.Time

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.DisplayName,
			&config.Description,
			&config.Type,
			&config.Status,
			&config.OwnerID,
			&config.TenantID,
			&config.IsDefault,
			&config.IsActive,
			&config.Version,
			&encryptedSettings,
			&config.CreatedAt,
			&config.UpdatedAt,
			&lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user configuration: %w", err)
		}

		config.LastUsedAt = lastUsedAt

		// Decrypt settings
		if len(encryptedSettings) > 0 {
			// Parse Base64 encoded EncryptedData
			encryptedData, err := crypto.FromBase64(string(encryptedSettings))
			if err != nil {
				return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
			}

			decryptedSettings, err := r.encryption.Decrypt(encryptedData, r.masterKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt settings: %w", err)
			}

			if err := json.Unmarshal(decryptedSettings, &config.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over configurations: %w", err)
	}

	return configs, nil
}

// CountConfigs counts user configurations with filtering
func (r *userConfigRepository) CountConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) (int64, error) {
	query := `SELECT COUNT(*) FROM user_configurations WHERE owner_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	// Apply the same filters as ListConfigs
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(filter.Type))
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(filter.Status))
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsDefault != nil {
		query += fmt.Sprintf(" AND is_default = $%d", argIndex)
		args = append(args, *filter.IsDefault)
		argIndex++
	}

	if filter.NamePattern != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+filter.NamePattern+"%")
		argIndex++
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user configurations: %w", err)
	}

	return count, nil
}

// GetServerConfig retrieves server configuration for a user
func (r *userConfigRepository) GetServerConfig(
	ctx context.Context,
	userID string,
	serverName string,
) (*ServerConfig, error) {
	query := `
		SELECT
			server_id, config, metadata, status, created_at, updated_at
		FROM server_configurations
		WHERE owner_id = $1 AND server_id = $2`

	var config ServerConfig
	var configJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, userID, serverName).Scan(
		&config.ServerID,
		&configJSON,
		&metadataJSON,
		&config.Status,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("server configuration not found")
		}
		return nil, fmt.Errorf("failed to query server configuration: %w", err)
	}

	// Unmarshal JSON fields
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &config, nil
}

// SaveServerConfig saves or updates server configuration
func (r *userConfigRepository) SaveServerConfig(
	ctx context.Context,
	userID string,
	config *ServerConfig,
) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO server_configurations (
			owner_id, server_id, config, metadata, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) ON CONFLICT (owner_id, server_id) DO UPDATE SET
			config = EXCLUDED.config,
			metadata = EXCLUDED.metadata,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	_, err = r.pool.Exec(ctx, query,
		userID,
		config.ServerID,
		configJSON,
		metadataJSON,
		config.Status,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to save server configuration: %w", err)
	}

	config.CreatedAt = now
	config.UpdatedAt = now

	return nil
}

// DeleteServerConfig deletes server configuration
func (r *userConfigRepository) DeleteServerConfig(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	query := `DELETE FROM server_configurations WHERE owner_id = $1 AND server_id = $2`

	result, err := r.pool.Exec(ctx, query, userID, serverName)
	if err != nil {
		return fmt.Errorf("failed to delete server configuration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server configuration not found")
	}

	return nil
}

// ListServerConfigs lists all server configurations for a user
func (r *userConfigRepository) ListServerConfigs(
	ctx context.Context,
	userID string,
) ([]*ServerConfig, error) {
	query := `
		SELECT
			server_id, config, metadata, status, created_at, updated_at
		FROM server_configurations
		WHERE owner_id = $1
		ORDER BY server_id`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query server configurations: %w", err)
	}
	defer rows.Close()

	var configs []*ServerConfig
	for rows.Next() {
		var config ServerConfig
		var configJSON, metadataJSON []byte

		err := rows.Scan(
			&config.ServerID,
			&configJSON,
			&metadataJSON,
			&config.Status,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server configuration: %w", err)
		}

		// Unmarshal JSON fields
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &config.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over server configurations: %w", err)
	}

	return configs, nil
}
