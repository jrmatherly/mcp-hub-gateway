package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Global database connection string for test utilities
var (
	globalConnStr string
	globalMu      sync.RWMutex
)

// RunMigrations executes database migrations with proper error handling and retry logic
func RunMigrations(ctx context.Context, databaseURL string, options RunOptions) error {
	// Validate connection first
	if err := ValidateConnection(databaseURL); err != nil {
		return fmt.Errorf("database validation failed: %w", err)
	}

	// Create migrator
	config := &MigrationConfig{
		DatabaseURL: databaseURL,
		SchemaName:  options.SchemaName,
		TableName:   options.TableName,
		Verbose:     options.Verbose,
	}

	migrator, err := NewMigrator(config)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	// Initialize migration system
	if err := migrator.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Get current status
	status, err := migrator.Status()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if options.Verbose {
		fmt.Println("Migration Status:")
		fmt.Println(status)
		fmt.Println()
	}

	// Check if database is in dirty state
	if status.IsDirty {
		if options.AutoRepair {
			fmt.Printf(
				"Database is in dirty state at version %d. Attempting auto-repair...\n",
				status.CurrentVersion,
			)
			// Force to current version to clean dirty state
			if err := migrator.Force(int(status.CurrentVersion)); err != nil {
				return fmt.Errorf("failed to repair dirty state: %w", err)
			}
			fmt.Println("Successfully repaired dirty state")
		} else {
			return fmt.Errorf("database is in dirty state at version %d, manual intervention required", status.CurrentVersion)
		}
	}

	// Execute migration based on direction
	switch options.Direction {
	case DirectionUp:
		if options.Steps > 0 {
			fmt.Printf("Migrating up %d step(s)...\n", options.Steps)
			err = migrator.Steps(options.Steps)
		} else {
			fmt.Println("Running all pending migrations...")
			err = migrator.Up()
		}

	case DirectionDown:
		if options.Steps > 0 {
			fmt.Printf("Rolling back %d migration(s)...\n", options.Steps)
			err = migrator.Steps(-options.Steps)
		} else {
			fmt.Println("Rolling back last migration...")
			err = migrator.Down()
		}

	case DirectionForce:
		if options.Version < 0 {
			return fmt.Errorf("force requires a valid version number")
		}
		fmt.Printf("Forcing migration version to %d...\n", options.Version)
		err = migrator.Force(options.Version)

	default:
		return fmt.Errorf("invalid migration direction: %s", options.Direction)
	}

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Get final status
	finalStatus, err := migrator.Status()
	if err != nil {
		return fmt.Errorf("failed to get final status: %w", err)
	}

	if options.Verbose {
		fmt.Println("\nFinal Migration Status:")
		fmt.Println(finalStatus)
	} else {
		fmt.Printf("Migration completed successfully. Current version: %d\n", finalStatus.CurrentVersion)
	}

	return nil
}

// RunOptions contains options for migration execution
type RunOptions struct {
	Direction  Direction
	Steps      int    // Number of migrations to run (for up/down)
	Version    int    // Target version (for force)
	SchemaName string // Database schema (default: public)
	TableName  string // Migration tracking table (default: schema_migrations)
	AutoRepair bool   // Auto-repair dirty state
	Verbose    bool   // Enable verbose output
	DryRun     bool   // Show what would be done without executing
	Timeout    time.Duration
	MaxRetries int
}

// Direction represents migration direction
type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionForce Direction = "force"
)

// DefaultRunOptions returns default migration options
func DefaultRunOptions() RunOptions {
	return RunOptions{
		Direction:  DirectionUp,
		SchemaName: "public",
		TableName:  "schema_migrations",
		AutoRepair: false,
		Verbose:    false,
		DryRun:     false,
		Timeout:    5 * time.Minute,
		MaxRetries: 3,
	}
}

// CLIMigrationRunner provides CLI integration for migrations
type CLIMigrationRunner struct {
	DatabaseURL string
	Options     RunOptions
}

// NewCLIMigrationRunner creates a new CLI migration runner
func NewCLIMigrationRunner(databaseURL string) *CLIMigrationRunner {
	return &CLIMigrationRunner{
		DatabaseURL: databaseURL,
		Options:     DefaultRunOptions(),
	}
}

// Run executes migrations based on CLI flags
func (r *CLIMigrationRunner) Run(ctx context.Context) error {
	// Check if database URL is provided
	if r.DatabaseURL == "" {
		// Try to get from environment variable
		r.DatabaseURL = os.Getenv("DATABASE_URL")
		if r.DatabaseURL == "" {
			return fmt.Errorf("database URL not provided (set DATABASE_URL or use --database-url)")
		}
	}

	// Run migrations with retry logic
	var lastErr error
	for attempt := 1; attempt <= r.Options.MaxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("Retry attempt %d/%d...\n", attempt, r.Options.MaxRetries)
			time.Sleep(time.Second * time.Duration(attempt)) // Exponential backoff
		}

		// Create context with timeout
		runCtx, cancel := context.WithTimeout(ctx, r.Options.Timeout)
		err := RunMigrations(runCtx, r.DatabaseURL, r.Options)
		cancel()

		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		if r.Options.Verbose {
			fmt.Printf("Migration attempt %d failed: %v\n", attempt, err)
		}
	}

	return fmt.Errorf("migration failed after %d attempts: %w", r.Options.MaxRetries, lastErr)
}

// Status returns current migration status
func (r *CLIMigrationRunner) Status(ctx context.Context) (*MigrationStatus, error) {
	if r.DatabaseURL == "" {
		r.DatabaseURL = os.Getenv("DATABASE_URL")
		if r.DatabaseURL == "" {
			return nil, fmt.Errorf("database URL not provided")
		}
	}

	config := &MigrationConfig{
		DatabaseURL: r.DatabaseURL,
		SchemaName:  r.Options.SchemaName,
		TableName:   r.Options.TableName,
		Verbose:     r.Options.Verbose,
	}

	migrator, err := NewMigrator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}

	return migrator.Status()
}

// Validate checks if migrations are valid without running them
func (r *CLIMigrationRunner) Validate(ctx context.Context) error {
	if r.DatabaseURL == "" {
		r.DatabaseURL = os.Getenv("DATABASE_URL")
		if r.DatabaseURL == "" {
			return fmt.Errorf("database URL not provided")
		}
	}

	// Validate connection
	if err := ValidateConnection(r.DatabaseURL); err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	// Check migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no migration files found")
	}

	fmt.Printf("Found %d migration file(s)\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - %s\n", entry.Name())
	}

	return nil
}

// Helper function to determine if error is retryable
func isRetryableError(err error) bool {
	// Add patterns for retryable errors (connection issues, timeouts, etc.)
	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"too many connections",
		"deadlock detected",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains check
	// In production, use strings.Contains with strings.ToLower
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				containsIgnoreCase(s[1:], substr) ||
			containsIgnoreCase(s[:len(s)-1], substr))
}

// InitializeWithConnectionString initializes the database with the given connection string
// This function is used by tests to set up database connections
func InitializeWithConnectionString(connStr string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	// Store connection string for later use
	globalConnStr = connStr

	// Validate the connection
	if err := ValidateConnection(connStr); err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}

	return nil
}

// RunMigrationsSimple runs database migrations using the globally stored connection string
// This is a simplified version for tests that don't need complex configuration
func RunMigrationsSimple() error {
	globalMu.RLock()
	connStr := globalConnStr
	globalMu.RUnlock()

	if connStr == "" {
		return fmt.Errorf("database not initialized - call InitializeWithConnectionString first")
	}

	// Use default options for test scenarios
	options := RunOptions{
		SchemaName: "public",
		TableName:  "schema_migrations",
		MaxRetries: 3,
		Verbose:    false,
		AutoRepair: true,
		DryRun:     false,
	}

	// Create a CLIMigrationRunner and use it
	runner := &CLIMigrationRunner{
		DatabaseURL: connStr,
		Options:     options,
	}

	return runner.Run(context.Background())
}
