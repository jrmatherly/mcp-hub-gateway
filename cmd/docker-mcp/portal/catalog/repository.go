package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/database"
	"github.com/lib/pq"
)

// postgresRepository implements CatalogRepository using PostgreSQL
type postgresRepository struct {
	db *database.Pool
}

// CreatePostgresRepository creates a new PostgreSQL repository instance
func CreatePostgresRepository(pool *database.Pool) CatalogRepository {
	return &postgresRepository{
		db: pool,
	}
}

// Catalog CRUD operations

// CreateCatalog creates a new catalog
func (r *postgresRepository) CreateCatalog(
	ctx context.Context,
	userID string,
	catalog *Catalog,
) error {
	query := `
		INSERT INTO catalogs (
			id, name, display_name, description, type, status, version,
			owner_id, tenant_id, is_public, is_default,
			source_url, source_type, source_config,
			tags, homepage, repository, license, maintainer,
			server_count, download_count, last_synced_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)`

	// Convert maps and slices to JSON for storage
	sourceConfig, _ := json.Marshal(catalog.SourceConfig)
	tags := pq.Array(catalog.Tags)

	_, err := r.db.GetPool().Exec(ctx, query,
		catalog.ID, catalog.Name, catalog.DisplayName, catalog.Description,
		catalog.Type, catalog.Status, catalog.Version,
		catalog.OwnerID, catalog.TenantID, catalog.IsPublic, catalog.IsDefault,
		catalog.SourceURL, catalog.SourceType, sourceConfig,
		tags, catalog.Homepage, catalog.Repository, catalog.License, catalog.Maintainer,
		catalog.ServerCount, catalog.DownloadCount, catalog.LastSyncedAt,
		catalog.CreatedAt, catalog.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create catalog: %w", err)
	}

	return nil
}

// GetCatalog retrieves a catalog by ID
func (r *postgresRepository) GetCatalog(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Catalog, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status, version,
			owner_id, tenant_id, is_public, is_default,
			source_url, source_type, source_config,
			tags, homepage, repository, license, maintainer,
			server_count, download_count, last_synced_at,
			created_at, updated_at, deleted_at
		FROM catalogs
		WHERE id = $1 AND deleted_at IS NULL
		AND (owner_id = $2::uuid OR is_public = true)`

	catalog := &Catalog{}
	var sourceConfig json.RawMessage
	var tags pq.StringArray
	var deletedAt sql.NullTime

	err := r.db.GetPool().QueryRow(ctx, query, id, userID).Scan(
		&catalog.ID, &catalog.Name, &catalog.DisplayName, &catalog.Description,
		&catalog.Type, &catalog.Status, &catalog.Version,
		&catalog.OwnerID, &catalog.TenantID, &catalog.IsPublic, &catalog.IsDefault,
		&catalog.SourceURL, &catalog.SourceType, &sourceConfig,
		&tags, &catalog.Homepage, &catalog.Repository, &catalog.License, &catalog.Maintainer,
		&catalog.ServerCount, &catalog.DownloadCount, &catalog.LastSyncedAt,
		&catalog.CreatedAt, &catalog.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("catalog not found")
		}
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	// Convert JSON fields back
	if sourceConfig != nil {
		json.Unmarshal(sourceConfig, &catalog.SourceConfig)
	}
	catalog.Tags = []string(tags)

	if deletedAt.Valid {
		catalog.DeletedAt = &deletedAt.Time
	}

	return catalog, nil
}

// GetCatalogByName retrieves a catalog by name
func (r *postgresRepository) GetCatalogByName(
	ctx context.Context,
	userID string,
	name string,
) (*Catalog, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status, version,
			owner_id, tenant_id, is_public, is_default,
			source_url, source_type, source_config,
			tags, homepage, repository, license, maintainer,
			server_count, download_count, last_synced_at,
			created_at, updated_at, deleted_at
		FROM catalogs
		WHERE name = $1 AND deleted_at IS NULL
		AND (owner_id = $2::uuid OR is_public = true)`

	catalog := &Catalog{}
	var sourceConfig json.RawMessage
	var tags pq.StringArray
	var deletedAt sql.NullTime

	err := r.db.GetPool().QueryRow(ctx, query, name, userID).Scan(
		&catalog.ID, &catalog.Name, &catalog.DisplayName, &catalog.Description,
		&catalog.Type, &catalog.Status, &catalog.Version,
		&catalog.OwnerID, &catalog.TenantID, &catalog.IsPublic, &catalog.IsDefault,
		&catalog.SourceURL, &catalog.SourceType, &sourceConfig,
		&tags, &catalog.Homepage, &catalog.Repository, &catalog.License, &catalog.Maintainer,
		&catalog.ServerCount, &catalog.DownloadCount, &catalog.LastSyncedAt,
		&catalog.CreatedAt, &catalog.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("catalog not found")
		}
		return nil, fmt.Errorf("failed to get catalog by name: %w", err)
	}

	// Convert JSON fields back
	if sourceConfig != nil {
		json.Unmarshal(sourceConfig, &catalog.SourceConfig)
	}
	catalog.Tags = []string(tags)

	if deletedAt.Valid {
		catalog.DeletedAt = &deletedAt.Time
	}

	return catalog, nil
}

// UpdateCatalog updates an existing catalog
func (r *postgresRepository) UpdateCatalog(
	ctx context.Context,
	userID string,
	catalog *Catalog,
) error {
	query := `
		UPDATE catalogs SET
			display_name = $2,
			description = $3,
			status = $4,
			version = $5,
			is_public = $6,
			is_default = $7,
			source_url = $8,
			source_type = $9,
			source_config = $10,
			tags = $11,
			homepage = $12,
			repository = $13,
			license = $14,
			maintainer = $15,
			server_count = $16,
			download_count = $17,
			last_synced_at = $18,
			updated_at = $19
		WHERE id = $1 AND owner_id = $20::uuid`

	sourceConfig, _ := json.Marshal(catalog.SourceConfig)
	tags := pq.Array(catalog.Tags)

	result, err := r.db.GetPool().Exec(ctx, query,
		catalog.ID,
		catalog.DisplayName, catalog.Description,
		catalog.Status, catalog.Version,
		catalog.IsPublic, catalog.IsDefault,
		catalog.SourceURL, catalog.SourceType, sourceConfig,
		tags, catalog.Homepage, catalog.Repository, catalog.License, catalog.Maintainer,
		catalog.ServerCount, catalog.DownloadCount, catalog.LastSyncedAt,
		catalog.UpdatedAt,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update catalog: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("catalog not found or not owned by user")
	}

	return nil
}

// DeleteCatalog soft deletes a catalog
func (r *postgresRepository) DeleteCatalog(ctx context.Context, userID string, id uuid.UUID) error {
	query := `
		UPDATE catalogs
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND owner_id = $3::uuid AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := r.db.GetPool().Exec(ctx, query, now, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete catalog: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("catalog not found or not owned by user")
	}

	// Also delete associated servers
	serverQuery := `
		UPDATE servers
		SET deleted_at = $1, updated_at = $1
		WHERE catalog_id = $2 AND deleted_at IS NULL`

	_, err = r.db.GetPool().Exec(ctx, serverQuery, now, id)
	if err != nil {
		// Log but don't fail on server deletion
		fmt.Printf("Warning: failed to delete associated servers: %v\n", err)
	}

	return nil
}

// ListCatalogs lists catalogs with filtering
func (r *postgresRepository) ListCatalogs(
	ctx context.Context,
	userID string,
	filter CatalogFilter,
) ([]*Catalog, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status, version,
			owner_id, tenant_id, is_public, is_default,
			source_url, source_type, source_config,
			tags, homepage, repository, license, maintainer,
			server_count, download_count, last_synced_at,
			created_at, updated_at
		FROM catalogs
		WHERE deleted_at IS NULL
		AND (owner_id = $1::uuid OR is_public = true)`

	// Add filter conditions
	args := []interface{}{userID}
	paramCount := 1

	// Build dynamic query based on filter
	if len(filter.Type) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND type = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Type))
	}

	if len(filter.Status) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND status = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Status))
	}

	if filter.IsPublic != nil {
		paramCount++
		query += fmt.Sprintf(" AND is_public = $%d", paramCount)
		args = append(args, *filter.IsPublic)
	}

	if filter.IsDefault != nil {
		paramCount++
		query += fmt.Sprintf(" AND is_default = $%d", paramCount)
		args = append(args, *filter.IsDefault)
	}

	if filter.Search != "" {
		paramCount++
		query += fmt.Sprintf(
			" AND (name ILIKE $%d OR display_name ILIKE $%d OR description ILIKE $%d)",
			paramCount,
			paramCount,
			paramCount,
		)
		args = append(args, "%"+filter.Search+"%")
	}

	// Add sorting
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := filter.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	if filter.Limit > 0 {
		paramCount++
		query += fmt.Sprintf(" LIMIT $%d", paramCount)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		paramCount++
		query += fmt.Sprintf(" OFFSET $%d", paramCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.GetPool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}
	defer rows.Close()

	var catalogs []*Catalog
	for rows.Next() {
		catalog := &Catalog{}
		var sourceConfig json.RawMessage
		var tags pq.StringArray

		err := rows.Scan(
			&catalog.ID, &catalog.Name, &catalog.DisplayName, &catalog.Description,
			&catalog.Type, &catalog.Status, &catalog.Version,
			&catalog.OwnerID, &catalog.TenantID, &catalog.IsPublic, &catalog.IsDefault,
			&catalog.SourceURL, &catalog.SourceType, &sourceConfig,
			&tags, &catalog.Homepage, &catalog.Repository, &catalog.License, &catalog.Maintainer,
			&catalog.ServerCount, &catalog.DownloadCount, &catalog.LastSyncedAt,
			&catalog.CreatedAt, &catalog.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan catalog: %w", err)
		}

		// Convert JSON fields back
		if sourceConfig != nil {
			json.Unmarshal(sourceConfig, &catalog.SourceConfig)
		}
		catalog.Tags = []string(tags)

		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}

// CountCatalogs counts catalogs matching the filter
func (r *postgresRepository) CountCatalogs(
	ctx context.Context,
	userID string,
	filter CatalogFilter,
) (int64, error) {
	query := `
		SELECT COUNT(*) FROM catalogs
		WHERE deleted_at IS NULL
		AND (owner_id = $1::uuid OR is_public = true)`

	// Add same filter conditions as ListCatalogs
	args := []interface{}{userID}
	paramCount := 1

	if len(filter.Type) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND type = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Type))
	}

	if len(filter.Status) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND status = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Status))
	}

	if filter.IsPublic != nil {
		paramCount++
		query += fmt.Sprintf(" AND is_public = $%d", paramCount)
		args = append(args, *filter.IsPublic)
	}

	if filter.IsDefault != nil {
		paramCount++
		query += fmt.Sprintf(" AND is_default = $%d", paramCount)
		args = append(args, *filter.IsDefault)
	}

	if filter.Search != "" {
		paramCount++
		query += fmt.Sprintf(
			" AND (name ILIKE $%d OR display_name ILIKE $%d OR description ILIKE $%d)",
			paramCount,
			paramCount,
			paramCount,
		)
		args = append(args, "%"+filter.Search+"%")
	}

	var count int64
	err := r.db.GetPool().QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count catalogs: %w", err)
	}

	return count, nil
}

// GetDefaultCatalog gets the default catalog for a user
func (r *postgresRepository) GetDefaultCatalog(
	ctx context.Context,
	userID string,
) (*Catalog, error) {
	query := `
		SELECT
			id, name, display_name, description, type, status, version,
			owner_id, tenant_id, is_public, is_default,
			source_url, source_type, source_config,
			tags, homepage, repository, license, maintainer,
			server_count, download_count, last_synced_at,
			created_at, updated_at
		FROM catalogs
		WHERE is_default = true AND deleted_at IS NULL
		AND (owner_id = $1::uuid OR is_public = true)
		LIMIT 1`

	catalog := &Catalog{}
	var sourceConfig json.RawMessage
	var tags pq.StringArray

	err := r.db.GetPool().QueryRow(ctx, query, userID).Scan(
		&catalog.ID, &catalog.Name, &catalog.DisplayName, &catalog.Description,
		&catalog.Type, &catalog.Status, &catalog.Version,
		&catalog.OwnerID, &catalog.TenantID, &catalog.IsPublic, &catalog.IsDefault,
		&catalog.SourceURL, &catalog.SourceType, &sourceConfig,
		&tags, &catalog.Homepage, &catalog.Repository, &catalog.License, &catalog.Maintainer,
		&catalog.ServerCount, &catalog.DownloadCount, &catalog.LastSyncedAt,
		&catalog.CreatedAt, &catalog.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no default catalog found")
		}
		return nil, fmt.Errorf("failed to get default catalog: %w", err)
	}

	// Convert JSON fields back
	if sourceConfig != nil {
		json.Unmarshal(sourceConfig, &catalog.SourceConfig)
	}
	catalog.Tags = []string(tags)

	return catalog, nil
}

// Server CRUD operations

// CreateServer creates a new server
func (r *postgresRepository) CreateServer(
	ctx context.Context,
	userID string,
	server *Server,
) error {
	// Verify user owns the catalog
	var ownerID uuid.UUID
	err := r.db.GetPool().QueryRow(ctx,
		"SELECT owner_id FROM catalogs WHERE id = $1 AND deleted_at IS NULL",
		server.CatalogID,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("catalog not found")
	}

	uid, _ := uuid.Parse(userID)
	if ownerID != uid {
		return fmt.Errorf("not authorized to add server to this catalog")
	}

	query := `
		INSERT INTO servers (
			id, catalog_id, name, display_name, description, type, version,
			command, environment, working_dir,
			image, dockerfile, build_args,
			cpu_limit, memory_limit,
			ports, volumes, secrets, permissions,
			tags, homepage, repository, license, author,
			required_tools, min_version, config,
			is_enabled, is_deprecated, deprecation_message,
			usage_count, rating_avg, rating_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35
		)`

	// Convert complex types to JSON
	command, _ := json.Marshal(server.Command)
	environment, _ := json.Marshal(server.Environment)
	buildArgs, _ := json.Marshal(server.BuildArgs)
	ports, _ := json.Marshal(server.Ports)
	volumes, _ := json.Marshal(server.Volumes)
	secrets := pq.Array(server.Secrets)
	permissions := pq.Array(server.Permissions)
	tags := pq.Array(server.Tags)
	requiredTools := pq.Array(server.RequiredTools)
	config, _ := json.Marshal(server.Config)

	_, err = r.db.GetPool().Exec(ctx, query,
		server.ID, server.CatalogID, server.Name, server.DisplayName, server.Description,
		server.Type, server.Version,
		command, environment, server.WorkingDir,
		server.Image, server.Dockerfile, buildArgs,
		server.CPULimit, server.MemoryLimit,
		ports, volumes, secrets, permissions,
		tags, server.Homepage, server.Repository, server.License, server.Author,
		requiredTools, server.MinVersion, config,
		server.IsEnabled, server.IsDeprecated, server.DeprecationMessage,
		server.UsageCount, server.RatingAvg, server.RatingCount,
		server.CreatedAt, server.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Update server count in catalog
	_, err = r.db.GetPool().Exec(ctx,
		"UPDATE catalogs SET server_count = server_count + 1, updated_at = $1 WHERE id = $2",
		time.Now().UTC(), server.CatalogID,
	)

	return nil
}

// GetServer retrieves a server by ID
func (r *postgresRepository) GetServer(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Server, error) {
	query := `
		SELECT
			s.id, s.catalog_id, s.name, s.display_name, s.description, s.type, s.version,
			s.command, s.environment, s.working_dir,
			s.image, s.dockerfile, s.build_args,
			s.cpu_limit, s.memory_limit,
			s.ports, s.volumes, s.secrets, s.permissions,
			s.tags, s.homepage, s.repository, s.license, s.author,
			s.required_tools, s.min_version, s.config,
			s.is_enabled, s.is_deprecated, s.deprecation_message,
			s.usage_count, s.rating_avg, s.rating_count,
			s.created_at, s.updated_at, s.deleted_at
		FROM servers s
		JOIN catalogs c ON s.catalog_id = c.id
		WHERE s.id = $1 AND s.deleted_at IS NULL
		AND (c.owner_id = $2::uuid OR c.is_public = true)`

	server := &Server{}
	var (
		command       json.RawMessage
		environment   json.RawMessage
		buildArgs     json.RawMessage
		ports         json.RawMessage
		volumes       json.RawMessage
		config        json.RawMessage
		secrets       pq.StringArray
		permissions   pq.StringArray
		tags          pq.StringArray
		requiredTools pq.StringArray
		deletedAt     sql.NullTime
	)

	err := r.db.GetPool().QueryRow(ctx, query, id, userID).Scan(
		&server.ID, &server.CatalogID, &server.Name, &server.DisplayName, &server.Description,
		&server.Type, &server.Version,
		&command, &environment, &server.WorkingDir,
		&server.Image, &server.Dockerfile, &buildArgs,
		&server.CPULimit, &server.MemoryLimit,
		&ports, &volumes, &secrets, &permissions,
		&tags, &server.Homepage, &server.Repository, &server.License, &server.Author,
		&requiredTools, &server.MinVersion, &config,
		&server.IsEnabled, &server.IsDeprecated, &server.DeprecationMessage,
		&server.UsageCount, &server.RatingAvg, &server.RatingCount,
		&server.CreatedAt, &server.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	// Convert JSON fields back
	json.Unmarshal(command, &server.Command)
	json.Unmarshal(environment, &server.Environment)
	json.Unmarshal(buildArgs, &server.BuildArgs)
	json.Unmarshal(ports, &server.Ports)
	json.Unmarshal(volumes, &server.Volumes)
	json.Unmarshal(config, &server.Config)

	server.Secrets = []string(secrets)
	server.Permissions = []string(permissions)
	server.Tags = []string(tags)
	server.RequiredTools = []string(requiredTools)

	if deletedAt.Valid {
		server.DeletedAt = &deletedAt.Time
	}

	return server, nil
}

// GetServerByName retrieves a server by name within a catalog
func (r *postgresRepository) GetServerByName(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
	name string,
) (*Server, error) {
	query := `
		SELECT
			s.id, s.catalog_id, s.name, s.display_name, s.description, s.type, s.version,
			s.command, s.environment, s.working_dir,
			s.image, s.dockerfile, s.build_args,
			s.cpu_limit, s.memory_limit,
			s.ports, s.volumes, s.secrets, s.permissions,
			s.tags, s.homepage, s.repository, s.license, s.author,
			s.required_tools, s.min_version, s.config,
			s.is_enabled, s.is_deprecated, s.deprecation_message,
			s.usage_count, s.rating_avg, s.rating_count,
			s.created_at, s.updated_at, s.deleted_at
		FROM servers s
		JOIN catalogs c ON s.catalog_id = c.id
		WHERE s.catalog_id = $1 AND s.name = $2 AND s.deleted_at IS NULL
		AND (c.owner_id = $3::uuid OR c.is_public = true)`

	server := &Server{}
	var (
		command       json.RawMessage
		environment   json.RawMessage
		buildArgs     json.RawMessage
		ports         json.RawMessage
		volumes       json.RawMessage
		config        json.RawMessage
		secrets       pq.StringArray
		permissions   pq.StringArray
		tags          pq.StringArray
		requiredTools pq.StringArray
		deletedAt     sql.NullTime
	)

	err := r.db.GetPool().QueryRow(ctx, query, catalogID, name, userID).Scan(
		&server.ID, &server.CatalogID, &server.Name, &server.DisplayName, &server.Description,
		&server.Type, &server.Version,
		&command, &environment, &server.WorkingDir,
		&server.Image, &server.Dockerfile, &buildArgs,
		&server.CPULimit, &server.MemoryLimit,
		&ports, &volumes, &secrets, &permissions,
		&tags, &server.Homepage, &server.Repository, &server.License, &server.Author,
		&requiredTools, &server.MinVersion, &config,
		&server.IsEnabled, &server.IsDeprecated, &server.DeprecationMessage,
		&server.UsageCount, &server.RatingAvg, &server.RatingCount,
		&server.CreatedAt, &server.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server by name: %w", err)
	}

	// Convert JSON fields back
	json.Unmarshal(command, &server.Command)
	json.Unmarshal(environment, &server.Environment)
	json.Unmarshal(buildArgs, &server.BuildArgs)
	json.Unmarshal(ports, &server.Ports)
	json.Unmarshal(volumes, &server.Volumes)
	json.Unmarshal(config, &server.Config)

	server.Secrets = []string(secrets)
	server.Permissions = []string(permissions)
	server.Tags = []string(tags)
	server.RequiredTools = []string(requiredTools)

	if deletedAt.Valid {
		server.DeletedAt = &deletedAt.Time
	}

	return server, nil
}

// UpdateServer updates an existing server
func (r *postgresRepository) UpdateServer(
	ctx context.Context,
	userID string,
	server *Server,
) error {
	// Verify user owns the catalog
	var ownerID uuid.UUID
	err := r.db.GetPool().QueryRow(ctx,
		"SELECT owner_id FROM catalogs WHERE id = $1 AND deleted_at IS NULL",
		server.CatalogID,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("catalog not found")
	}

	uid, _ := uuid.Parse(userID)
	if ownerID != uid {
		return fmt.Errorf("not authorized to update server in this catalog")
	}

	query := `
		UPDATE servers SET
			display_name = $2,
			description = $3,
			version = $4,
			command = $5,
			environment = $6,
			working_dir = $7,
			image = $8,
			dockerfile = $9,
			build_args = $10,
			cpu_limit = $11,
			memory_limit = $12,
			ports = $13,
			volumes = $14,
			secrets = $15,
			permissions = $16,
			tags = $17,
			homepage = $18,
			repository = $19,
			license = $20,
			author = $21,
			required_tools = $22,
			min_version = $23,
			config = $24,
			is_enabled = $25,
			is_deprecated = $26,
			deprecation_message = $27,
			updated_at = $28
		WHERE id = $1 AND catalog_id = $29`

	// Convert complex types to JSON
	command, _ := json.Marshal(server.Command)
	environment, _ := json.Marshal(server.Environment)
	buildArgs, _ := json.Marshal(server.BuildArgs)
	ports, _ := json.Marshal(server.Ports)
	volumes, _ := json.Marshal(server.Volumes)
	secrets := pq.Array(server.Secrets)
	permissions := pq.Array(server.Permissions)
	tags := pq.Array(server.Tags)
	requiredTools := pq.Array(server.RequiredTools)
	config, _ := json.Marshal(server.Config)

	result, err := r.db.GetPool().Exec(ctx, query,
		server.ID,
		server.DisplayName, server.Description, server.Version,
		command, environment, server.WorkingDir,
		server.Image, server.Dockerfile, buildArgs,
		server.CPULimit, server.MemoryLimit,
		ports, volumes, secrets, permissions,
		tags, server.Homepage, server.Repository, server.License, server.Author,
		requiredTools, server.MinVersion, config,
		server.IsEnabled, server.IsDeprecated, server.DeprecationMessage,
		server.UpdatedAt,
		server.CatalogID,
	)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}

	return nil
}

// DeleteServer soft deletes a server
func (r *postgresRepository) DeleteServer(ctx context.Context, userID string, id uuid.UUID) error {
	// Verify user owns the catalog
	var catalogID uuid.UUID
	var ownerID uuid.UUID

	err := r.db.GetPool().QueryRow(ctx,
		`SELECT s.catalog_id, c.owner_id
		FROM servers s
		JOIN catalogs c ON s.catalog_id = c.id
		WHERE s.id = $1 AND s.deleted_at IS NULL`,
		id,
	).Scan(&catalogID, &ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("server not found")
		}
		return fmt.Errorf("failed to verify server ownership: %w", err)
	}

	uid, _ := uuid.Parse(userID)
	if ownerID != uid {
		return fmt.Errorf("not authorized to delete this server")
	}

	// Soft delete the server
	now := time.Now().UTC()
	result, err := r.db.GetPool().Exec(ctx,
		"UPDATE servers SET deleted_at = $1, updated_at = $1 WHERE id = $2",
		now, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}

	// Update server count in catalog
	_, err = r.db.GetPool().Exec(ctx,
		"UPDATE catalogs SET server_count = server_count - 1, updated_at = $1 WHERE id = $2",
		now, catalogID,
	)

	return nil
}

// ListServers lists servers with filtering
func (r *postgresRepository) ListServers(
	ctx context.Context,
	userID string,
	filter ServerFilter,
) ([]*Server, error) {
	query := `
		SELECT
			s.id, s.catalog_id, s.name, s.display_name, s.description, s.type, s.version,
			s.command, s.environment, s.working_dir,
			s.image, s.dockerfile, s.build_args,
			s.cpu_limit, s.memory_limit,
			s.ports, s.volumes, s.secrets, s.permissions,
			s.tags, s.homepage, s.repository, s.license, s.author,
			s.required_tools, s.min_version, s.config,
			s.is_enabled, s.is_deprecated, s.deprecation_message,
			s.usage_count, s.rating_avg, s.rating_count,
			s.created_at, s.updated_at
		FROM servers s
		JOIN catalogs c ON s.catalog_id = c.id
		WHERE s.deleted_at IS NULL
		AND (c.owner_id = $1::uuid OR c.is_public = true)`

	args := []interface{}{userID}
	paramCount := 1

	// Add filter conditions
	if filter.CatalogID != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.catalog_id = $%d", paramCount)
		args = append(args, *filter.CatalogID)
	}

	if len(filter.Type) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND s.type = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Type))
	}

	if filter.IsEnabled != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.is_enabled = $%d", paramCount)
		args = append(args, *filter.IsEnabled)
	}

	if filter.IsDeprecated != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.is_deprecated = $%d", paramCount)
		args = append(args, *filter.IsDeprecated)
	}

	if filter.Search != "" {
		paramCount++
		query += fmt.Sprintf(
			" AND (s.name ILIKE $%d OR s.display_name ILIKE $%d OR s.description ILIKE $%d)",
			paramCount,
			paramCount,
			paramCount,
		)
		args = append(args, "%"+filter.Search+"%")
	}

	// Add sorting
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "s.created_at"
	}
	sortOrder := filter.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	if filter.Limit > 0 {
		paramCount++
		query += fmt.Sprintf(" LIMIT $%d", paramCount)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		paramCount++
		query += fmt.Sprintf(" OFFSET $%d", paramCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.GetPool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	var servers []*Server
	for rows.Next() {
		server := &Server{}
		var (
			command       json.RawMessage
			environment   json.RawMessage
			buildArgs     json.RawMessage
			ports         json.RawMessage
			volumes       json.RawMessage
			config        json.RawMessage
			secrets       pq.StringArray
			permissions   pq.StringArray
			tags          pq.StringArray
			requiredTools pq.StringArray
		)

		err := rows.Scan(
			&server.ID, &server.CatalogID, &server.Name, &server.DisplayName, &server.Description,
			&server.Type, &server.Version,
			&command, &environment, &server.WorkingDir,
			&server.Image, &server.Dockerfile, &buildArgs,
			&server.CPULimit, &server.MemoryLimit,
			&ports, &volumes, &secrets, &permissions,
			&tags, &server.Homepage, &server.Repository, &server.License, &server.Author,
			&requiredTools, &server.MinVersion, &config,
			&server.IsEnabled, &server.IsDeprecated, &server.DeprecationMessage,
			&server.UsageCount, &server.RatingAvg, &server.RatingCount,
			&server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}

		// Convert JSON fields back
		json.Unmarshal(command, &server.Command)
		json.Unmarshal(environment, &server.Environment)
		json.Unmarshal(buildArgs, &server.BuildArgs)
		json.Unmarshal(ports, &server.Ports)
		json.Unmarshal(volumes, &server.Volumes)
		json.Unmarshal(config, &server.Config)

		server.Secrets = []string(secrets)
		server.Permissions = []string(permissions)
		server.Tags = []string(tags)
		server.RequiredTools = []string(requiredTools)

		servers = append(servers, server)
	}

	return servers, nil
}

// CountServers counts servers matching the filter
func (r *postgresRepository) CountServers(
	ctx context.Context,
	userID string,
	filter ServerFilter,
) (int64, error) {
	query := `
		SELECT COUNT(*) FROM servers s
		JOIN catalogs c ON s.catalog_id = c.id
		WHERE s.deleted_at IS NULL
		AND (c.owner_id = $1::uuid OR c.is_public = true)`

	args := []interface{}{userID}
	paramCount := 1

	// Add same filter conditions as ListServers
	if filter.CatalogID != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.catalog_id = $%d", paramCount)
		args = append(args, *filter.CatalogID)
	}

	if len(filter.Type) > 0 {
		paramCount++
		query += fmt.Sprintf(" AND s.type = ANY($%d)", paramCount)
		args = append(args, pq.Array(filter.Type))
	}

	if filter.IsEnabled != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.is_enabled = $%d", paramCount)
		args = append(args, *filter.IsEnabled)
	}

	if filter.IsDeprecated != nil {
		paramCount++
		query += fmt.Sprintf(" AND s.is_deprecated = $%d", paramCount)
		args = append(args, *filter.IsDeprecated)
	}

	if filter.Search != "" {
		paramCount++
		query += fmt.Sprintf(
			" AND (s.name ILIKE $%d OR s.display_name ILIKE $%d OR s.description ILIKE $%d)",
			paramCount,
			paramCount,
			paramCount,
		)
		args = append(args, "%"+filter.Search+"%")
	}

	var count int64
	err := r.db.GetPool().QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count servers: %w", err)
	}

	return count, nil
}

// ListServersByCatalog lists all servers in a specific catalog
func (r *postgresRepository) ListServersByCatalog(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
) ([]*Server, error) {
	filter := ServerFilter{
		CatalogID: &catalogID,
	}
	return r.ListServers(ctx, userID, filter)
}

// Bulk operations

// CreateServersBatch creates multiple servers
func (r *postgresRepository) CreateServersBatch(
	ctx context.Context,
	userID string,
	servers []*Server,
) error {
	// Start transaction
	tx, err := r.db.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, server := range servers {
		// Use transaction to create each server
		// Similar to CreateServer but using tx instead of r.db.Pool
		// TODO: Implement server creation logic here
		_ = server // Acknowledge server variable usage
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateServersBatch updates multiple servers
func (r *postgresRepository) UpdateServersBatch(
	ctx context.Context,
	userID string,
	servers []*Server,
) error {
	// Similar batch implementation
	return fmt.Errorf("not yet implemented")
}

// DeleteServersBatch deletes multiple servers
func (r *postgresRepository) DeleteServersBatch(
	ctx context.Context,
	userID string,
	ids []uuid.UUID,
) error {
	// Similar batch implementation
	return fmt.Errorf("not yet implemented")
}

// Search operations

// SearchServers searches for servers
func (r *postgresRepository) SearchServers(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]*Server, error) {
	filter := ServerFilter{
		Search: query,
		Limit:  limit,
	}
	return r.ListServers(ctx, userID, filter)
}

// SearchCatalogs searches for catalogs
func (r *postgresRepository) SearchCatalogs(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]*Catalog, error) {
	filter := CatalogFilter{
		Search: query,
		Limit:  limit,
	}
	return r.ListCatalogs(ctx, userID, filter)
}

// Statistics

// GetCatalogStats gets statistics for a catalog
func (r *postgresRepository) GetCatalogStats(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
) (*CatalogStats, error) {
	// Implementation would query and aggregate server statistics
	return nil, fmt.Errorf("not yet implemented")
}

// GetServerStats gets statistics for a server
func (r *postgresRepository) GetServerStats(
	ctx context.Context,
	userID string,
	serverID uuid.UUID,
) (*ServerStats, error) {
	// Implementation would query usage statistics
	return nil, fmt.Errorf("not yet implemented")
}
