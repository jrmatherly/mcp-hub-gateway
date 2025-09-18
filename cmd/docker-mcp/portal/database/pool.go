// Package database provides database connection and migration functionality for the MCP Portal
package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
)

// Pool manages PostgreSQL connections with RLS support
type Pool struct {
	pool   *pgxpool.Pool
	config *config.DatabaseConfig
	mu     sync.RWMutex
	closed bool
}

// NewPool creates a new database connection pool
func NewPool(cfg *config.DatabaseConfig) (*Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration is required")
	}

	poolConfig, err := buildPoolConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build pool config: %w", err)
	}

	// Create the pool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Verify RLS is enabled
	if err := verifyRLS(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("RLS verification failed: %w", err)
	}

	return &Pool{
		pool:   pool,
		config: cfg,
	}, nil
}

// buildPoolConfig constructs the pgxpool configuration
func buildPoolConfig(cfg *config.DatabaseConfig) (*pgxpool.Config, error) {
	// Build connection string
	connString := buildConnectionString(cfg)

	// Parse the connection string
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Apply pool settings
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	// Configure connection initialization
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		// Verify connection is alive
		return conn.Ping(ctx) == nil
	}

	// Configure statement timeout on each connection
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Set default statement timeout
		_, err := conn.Exec(
			ctx,
			fmt.Sprintf("SET statement_timeout = '%dms'", cfg.StatementTimeout.Milliseconds()),
		)
		if err != nil {
			return fmt.Errorf("failed to set statement timeout: %w", err)
		}

		// Enable RLS for connection
		_, err = conn.Exec(ctx, "SET row_security = on")
		if err != nil {
			return fmt.Errorf("failed to enable RLS: %w", err)
		}

		return nil
	}

	return poolConfig, nil
}

// GetPool returns the underlying pgx pool
func (p *Pool) GetPool() *pgxpool.Pool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil
	}
	return p.pool
}

// WithUser returns a connection configured for a specific user
func (p *Pool) WithUser(ctx context.Context, userID string) (Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	// Acquire a connection from the pool
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	// Set the user context for RLS
	if userID != "" {
		_, err = conn.Exec(ctx, "SET LOCAL app.current_user = $1", userID)
		if err != nil {
			conn.Release()
			return nil, fmt.Errorf("failed to set user context: %w", err)
		}
	}

	return &UserConn{
		conn:   conn,
		userID: userID,
		pool:   p,
	}, nil
}

// Transaction executes a function within a database transaction with RLS context
func (p *Pool) Transaction(ctx context.Context, userID string, fn TxFunc) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	// Begin a transaction
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is handled properly
	defer func() {
		if err != nil {
			// Rollback on error
			_ = tx.Rollback(ctx)
		}
	}()

	// Set user context for the transaction
	if userID != "" {
		_, err = tx.Exec(ctx, "SET LOCAL app.current_user = $1", userID)
		if err != nil {
			return fmt.Errorf("failed to set user context in transaction: %w", err)
		}
	}

	// Execute the transaction function
	err = fn(ctx, &TxWrapper{tx: tx, userID: userID})
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit(ctx)
}

// Stats returns pool statistics
func (p *Pool) Stats() *pgxpool.Stat {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed || p.pool == nil {
		return nil
	}
	return p.pool.Stat()
}

// Health checks the health of the database connection pool
func (p *Pool) Health(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return fmt.Errorf("pool is closed")
	}

	// Ping the database
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check pool stats
	stats := p.pool.Stat()
	if stats.IdleConns() == 0 && stats.TotalConns() == stats.MaxConns() {
		return fmt.Errorf("connection pool exhausted: %d/%d connections in use",
			stats.TotalConns(), stats.MaxConns())
	}

	// Verify RLS is still enabled
	var rlsEnabled bool
	err := p.pool.QueryRow(ctx, "SHOW row_security").Scan(&rlsEnabled)
	if err != nil {
		return fmt.Errorf("failed to check RLS status: %w", err)
	}
	if !rlsEnabled {
		return fmt.Errorf("row-level security is disabled")
	}

	return nil
}

// Close closes the connection pool
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.pool.Close()
	return nil
}

// verifyRLS ensures RLS is properly configured
func verifyRLS(ctx context.Context, pool *pgxpool.Pool) error {
	// Check if RLS is enabled
	var rlsEnabled bool
	err := pool.QueryRow(ctx, "SHOW row_security").Scan(&rlsEnabled)
	if err != nil {
		return fmt.Errorf("failed to check RLS status: %w", err)
	}

	if !rlsEnabled {
		return fmt.Errorf("row-level security must be enabled")
	}

	// Verify critical tables have RLS enabled
	criticalTables := []string{"users", "servers", "configurations", "activity_logs"}
	for _, table := range criticalTables {
		var hasRLS bool
		err := pool.QueryRow(ctx, `
			SELECT relrowsecurity
			FROM pg_class
			WHERE relname = $1 AND relnamespace = 'public'::regnamespace
		`, table).Scan(&hasRLS)
		if err != nil {
			// Table might not exist yet (migrations not run)
			continue
		}

		if !hasRLS {
			return fmt.Errorf("table %s does not have RLS enabled", table)
		}
	}

	return nil
}

// UserConn represents a connection with user context
type UserConn struct {
	conn   *pgxpool.Conn
	userID string
	pool   *Pool
}

// Exec executes a query without returning any rows
func (c *UserConn) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return c.conn.Exec(ctx, sql, args...)
}

// Query executes a query that returns rows
func (c *UserConn) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.conn.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (c *UserConn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.conn.QueryRow(ctx, sql, args...)
}

// SendBatch sends a batch of queries
func (c *UserConn) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return c.conn.SendBatch(ctx, batch)
}

// CopyFrom performs a bulk insert
func (c *UserConn) CopyFrom(
	ctx context.Context,
	tableName pgx.Identifier,
	columnNames []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	return c.conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// Release returns the connection to the pool
func (c *UserConn) Release() {
	c.conn.Release()
}

// TxWrapper wraps a transaction with user context
type TxWrapper struct {
	tx     pgx.Tx
	userID string
}

// Exec executes a query without returning any rows
func (t *TxWrapper) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

// Query executes a query that returns rows
func (t *TxWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (t *TxWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// SendBatch sends a batch of queries
func (t *TxWrapper) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return t.tx.SendBatch(ctx, batch)
}

// CopyFrom performs a bulk insert
func (t *TxWrapper) CopyFrom(
	ctx context.Context,
	tableName pgx.Identifier,
	columnNames []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	return t.tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// Prepare creates a prepared statement
func (t *TxWrapper) Prepare(ctx context.Context, name, sql string) (*pgxpool.Conn, error) {
	// Note: pgx v5 doesn't have PreparedStatement anymore
	// This method should probably be removed or return the connection
	return nil, fmt.Errorf("prepared statements not supported in this interface")
}

// Package level pool instance for backward compatibility
var globalPool *Pool

// GetPool returns the global database pool instance
// This function provides backward compatibility for existing code
func GetPool() *pgxpool.Pool {
	if globalPool == nil {
		return nil
	}
	return globalPool.GetPool()
}

// SetGlobalPool sets the global pool instance
// This should be called during application initialization
func SetGlobalPool(pool *Pool) {
	globalPool = pool
}
