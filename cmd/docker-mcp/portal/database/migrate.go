// Package database provides database connection and migration functionality for the MCP Portal
package database

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationConfig contains configuration for database migrations
type MigrationConfig struct {
	DatabaseURL string
	SchemaName  string // Default: "public"
	TableName   string // Default: "schema_migrations"
	Verbose     bool
}

// Migrator handles database migrations
type Migrator struct {
	config   *MigrationConfig
	db       *sql.DB
	migrate  *migrate.Migrate
	instance string
}

// NewMigrator creates a new migration handler
func NewMigrator(config *MigrationConfig) (*Migrator, error) {
	if config.SchemaName == "" {
		config.SchemaName = "public"
	}
	if config.TableName == "" {
		config.TableName = "schema_migrations"
	}

	// Parse and validate database URL
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Migrator{
		config:   config,
		db:       db,
		instance: extractDBName(config.DatabaseURL),
	}, nil
}

// Initialize sets up the migration system
func (m *Migrator) Initialize() error {
	// Create migrations source from embedded filesystem
	sourceInstance, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver instance
	driver, err := postgres.WithInstance(m.db, &postgres.Config{
		MigrationsTable: m.config.TableName,
		SchemaName:      m.config.SchemaName,
	})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m.migrate, err = migrate.NewWithInstance(
		"iofs", sourceInstance,
		m.instance, driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if m.config.Verbose {
		m.migrate.Log = &migrateLogger{verbose: true}
	}

	return nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if m.migrate == nil {
		if err := m.Initialize(); err != nil {
			return err
		}
	}

	err := m.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		if m.config.Verbose {
			fmt.Println("No new migrations to apply")
		}
		return nil
	}

	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	if m.migrate == nil {
		if err := m.Initialize(); err != nil {
			return err
		}
	}

	err := m.migrate.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		if m.config.Verbose {
			fmt.Println("No migrations to roll back")
		}
		return nil
	}

	return nil
}

// Steps runs a specific number of migrations
// Positive n: migrate up n migrations
// Negative n: migrate down n migrations
func (m *Migrator) Steps(n int) error {
	if m.migrate == nil {
		if err := m.Initialize(); err != nil {
			return err
		}
	}

	err := m.migrate.Steps(n)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration steps failed: %w", err)
	}

	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	if m.migrate == nil {
		if err := m.Initialize(); err != nil {
			return 0, false, err
		}
	}

	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}

	return version, dirty, nil
}

// Force sets the migration version without running migrations
// Use with caution - this is for fixing broken migration states
func (m *Migrator) Force(version int) error {
	if m.migrate == nil {
		if err := m.Initialize(); err != nil {
			return err
		}
	}

	return m.migrate.Force(version)
}

// Status returns the current migration status
func (m *Migrator) Status() (*MigrationStatus, error) {
	version, dirty, err := m.Version()
	if err != nil && err.Error() != "no migration" {
		return nil, err
	}

	// Count total available migrations
	var totalMigrations int
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			totalMigrations++
		}
	}

	return &MigrationStatus{
		CurrentVersion:    version,
		IsDirty:           dirty,
		TotalMigrations:   totalMigrations,
		DatabaseName:      m.instance,
		MigrationsTable:   m.config.TableName,
		SchemaName:        m.config.SchemaName,
		PendingMigrations: calculatePending(version, totalMigrations),
	}, nil
}

// Close releases database resources
func (m *Migrator) Close() error {
	var errs []error

	if m.migrate != nil {
		if sourceErr, dbErr := m.migrate.Close(); sourceErr != nil || dbErr != nil {
			if sourceErr != nil {
				errs = append(errs, fmt.Errorf("source close error: %w", sourceErr))
			}
			if dbErr != nil {
				errs = append(errs, fmt.Errorf("database close error: %w", dbErr))
			}
		}
	}

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// MigrationStatus contains information about migration state
type MigrationStatus struct {
	CurrentVersion    uint
	IsDirty           bool
	TotalMigrations   int
	PendingMigrations int
	DatabaseName      string
	MigrationsTable   string
	SchemaName        string
}

// String returns a formatted status message
func (s *MigrationStatus) String() string {
	status := "clean"
	if s.IsDirty {
		status = "dirty"
	}

	return fmt.Sprintf(
		"Database: %s\n"+
			"Schema: %s\n"+
			"Migration Table: %s\n"+
			"Current Version: %d (%s)\n"+
			"Total Migrations: %d\n"+
			"Pending Migrations: %d",
		s.DatabaseName,
		s.SchemaName,
		s.MigrationsTable,
		s.CurrentVersion,
		status,
		s.TotalMigrations,
		s.PendingMigrations,
	)
}

// migrateLogger implements migrate.Logger interface
type migrateLogger struct {
	verbose bool
}

func (l *migrateLogger) Printf(format string, v ...any) {
	if l.verbose {
		fmt.Printf("[MIGRATE] "+format+"\n", v...)
	}
}

func (l *migrateLogger) Verbose() bool {
	return l.verbose
}

// Helper functions

func extractDBName(databaseURL string) string {
	// Extract database name from connection string
	// Format: postgres://user:pass@host:port/dbname?params
	parts := strings.Split(databaseURL, "/")
	if len(parts) > 3 {
		dbPart := parts[len(parts)-1]
		if idx := strings.Index(dbPart, "?"); idx > 0 {
			return dbPart[:idx]
		}
		return dbPart
	}
	return "postgres"
}

func calculatePending(currentVersion uint, totalMigrations int) int {
	// Simple calculation - assumes migrations are numbered sequentially
	// In reality, you'd need to check actual migration files
	if currentVersion >= uint(totalMigrations) {
		return 0
	}
	return totalMigrations - int(currentVersion)
}

// ValidateConnection tests the database connection
func ValidateConnection(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Check PostgreSQL version
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	// Check if we can create tables (basic permission check)
	_, err = db.Exec(`
		CREATE TEMP TABLE test_permissions (
			id SERIAL PRIMARY KEY,
			test_column TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("insufficient database permissions: %w", err)
	}

	return nil
}
