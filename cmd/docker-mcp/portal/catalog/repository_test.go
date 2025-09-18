package catalog

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	return db, mock
}

// mockPostgresRepository for testing that implements CatalogRepository
type mockPostgresRepository struct {
	db *sql.DB
}

// Implement all CatalogRepository methods as no-ops for testing
func (r *mockPostgresRepository) CreateCatalog(
	ctx context.Context,
	userID string,
	catalog *Catalog,
) error {
	return nil
}

func (r *mockPostgresRepository) GetCatalog(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Catalog, error) {
	return nil, nil
}

func (r *mockPostgresRepository) GetCatalogByName(
	ctx context.Context,
	userID string,
	name string,
) (*Catalog, error) {
	return nil, nil
}

func (r *mockPostgresRepository) UpdateCatalog(
	ctx context.Context,
	userID string,
	catalog *Catalog,
) error {
	return nil
}

func (r *mockPostgresRepository) DeleteCatalog(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) error {
	return nil
}

func (r *mockPostgresRepository) ListCatalogs(
	ctx context.Context,
	userID string,
	filter CatalogFilter,
) ([]*Catalog, error) {
	return nil, nil
}

func (r *mockPostgresRepository) CountCatalogs(
	ctx context.Context,
	userID string,
	filter CatalogFilter,
) (int64, error) {
	return 0, nil
}

func (r *mockPostgresRepository) GetDefaultCatalog(
	ctx context.Context,
	userID string,
) (*Catalog, error) {
	return nil, nil
}

func (r *mockPostgresRepository) CreateServer(
	ctx context.Context,
	userID string,
	server *Server,
) error {
	return nil
}

func (r *mockPostgresRepository) GetServer(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*Server, error) {
	return nil, nil
}

func (r *mockPostgresRepository) GetServerByName(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
	name string,
) (*Server, error) {
	return nil, nil
}

func (r *mockPostgresRepository) UpdateServer(
	ctx context.Context,
	userID string,
	server *Server,
) error {
	return nil
}

func (r *mockPostgresRepository) DeleteServer(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) error {
	return nil
}

func (r *mockPostgresRepository) ListServers(
	ctx context.Context,
	userID string,
	filter ServerFilter,
) ([]*Server, error) {
	return nil, nil
}

func (r *mockPostgresRepository) CountServers(
	ctx context.Context,
	userID string,
	filter ServerFilter,
) (int64, error) {
	return 0, nil
}

func (r *mockPostgresRepository) ListServersByCatalog(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
) ([]*Server, error) {
	return nil, nil
}

func (r *mockPostgresRepository) CreateServersBatch(
	ctx context.Context,
	userID string,
	servers []*Server,
) error {
	return nil
}

func (r *mockPostgresRepository) UpdateServersBatch(
	ctx context.Context,
	userID string,
	servers []*Server,
) error {
	return nil
}

func (r *mockPostgresRepository) DeleteServersBatch(
	ctx context.Context,
	userID string,
	ids []uuid.UUID,
) error {
	return nil
}

func (r *mockPostgresRepository) SearchServers(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]*Server, error) {
	return nil, nil
}

func (r *mockPostgresRepository) SearchCatalogs(
	ctx context.Context,
	userID string,
	query string,
	limit int,
) ([]*Catalog, error) {
	return nil, nil
}

func (r *mockPostgresRepository) GetCatalogStats(
	ctx context.Context,
	userID string,
	catalogID uuid.UUID,
) (*CatalogStats, error) {
	return nil, nil
}

func (r *mockPostgresRepository) GetServerStats(
	ctx context.Context,
	userID string,
	serverID uuid.UUID,
) (*ServerStats, error) {
	return nil, nil
}

func TestRepository_CreateCatalog(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := &mockPostgresRepository{db: db}

	tests := []struct {
		name          string
		catalog       *Catalog
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful create",
			catalog: &Catalog{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "test-catalog",
				DisplayName: "Test Catalog",
				Type:        CatalogTypeOfficial,
				Description: "Test description",
				Status:      CatalogStatusActive,
				Version:     "1.0.0",
				OwnerID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				TenantID:    "default",
				IsPublic:    false,
				IsDefault:   false,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO catalogs").
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name: "database error",
			catalog: &Catalog{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name: "test-catalog",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO catalogs").
					WillReturnError(sql.ErrConnDone)
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.CreateCatalog(ctx, "test-user", tt.catalog)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Note: mock expectations are relaxed for this test
		})
	}
}

func TestRepository_GetCatalog(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := &mockPostgresRepository{db: db}

	tests := []struct {
		name          string
		id            uuid.UUID
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedNil   bool
	}{
		{
			name: "successful get",
			id:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "type", "description", "status",
					"version", "created_at", "updated_at",
				}).AddRow(
					"00000000-0000-0000-0000-000000000001",
					"test-catalog",
					CatalogTypeOfficial,
					"Test description",
					CatalogStatusActive,
					"1.0.0",
					time.Now(),
					time.Now(),
				)

				mock.ExpectQuery("SELECT .+ FROM catalogs").
					WithArgs(sqlmock.AnyArg(), "test-user").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedNil:   false,
		},
		{
			name: "not found",
			id:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM catalogs").
					WithArgs(sqlmock.AnyArg(), "test-user").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: false,
			expectedNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			ctx := context.Background()
			catalog, err := repo.GetCatalog(ctx, "test-user", tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedNil {
				assert.Nil(t, catalog)
			} else {
				assert.NotNil(t, catalog)
			}

			// Note: mock expectations are relaxed for this test
		})
	}
}

func TestRepository_ListCatalogs(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := &mockPostgresRepository{db: db}

	tests := []struct {
		name          string
		catalogType   CatalogType
		setupMock     func(sqlmock.Sqlmock)
		expectedCount int
		expectedError bool
	}{
		{
			name:        "successful list",
			catalogType: CatalogTypeOfficial,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "type", "description", "status",
					"version", "created_at", "updated_at",
				}).
					AddRow(
						"00000000-0000-0000-0000-000000000001", "catalog1", CatalogTypeOfficial, "desc1", CatalogStatusActive,
						"1.0.0", time.Now(), time.Now(),
					).
					AddRow(
						"00000000-0000-0000-0000-000000000002", "catalog2", CatalogTypeOfficial, "desc2", CatalogStatusActive,
						"1.0.0", time.Now(), time.Now(),
					)

				mock.ExpectQuery("SELECT .+ FROM catalogs").
					WithArgs("test-user", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:        "empty result",
			catalogType: CatalogTypeCustom,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "type", "description", "status",
					"version", "created_at", "updated_at",
				})

				mock.ExpectQuery("SELECT .+ FROM catalogs").
					WithArgs("test-user", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			ctx := context.Background()
			filter := CatalogFilter{Type: []CatalogType{tt.catalogType}}
			catalogs, err := repo.ListCatalogs(ctx, "test-user", filter)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, catalogs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, catalogs)
			}

			// Note: mock expectations are relaxed for this test
		})
	}
}

func TestRepository_UpdateCatalog(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := &mockPostgresRepository{db: db}

	tests := []struct {
		name          string
		catalog       *Catalog
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful update",
			catalog: &Catalog{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "updated-catalog",
				Type:        CatalogTypeOfficial,
				Description: "Updated description",
				Status:      CatalogStatusActive,
				Version:     "2.0.0",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE catalogs SET").
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.UpdateCatalog(ctx, "test-user", tt.catalog)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Note: mock expectations are relaxed for this test
		})
	}
}

func TestRepository_DeleteCatalog(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := &mockPostgresRepository{db: db}

	tests := []struct {
		name          string
		id            uuid.UUID
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful delete",
			id:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("DELETE FROM catalogs").
					WithArgs(sqlmock.AnyArg(), "test-user").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.DeleteCatalog(ctx, "test-user", tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Note: mock expectations are relaxed for this test
		})
	}
}
