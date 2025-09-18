package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB represents the main database interface
type DB interface {
	GetPool() *pgxpool.Pool
	WithUser(ctx context.Context, userID string) (Conn, error)
	Transaction(ctx context.Context, userID string, fn TxFunc) error
	Health(ctx context.Context) error
	Close() error
	Stats() *pgxpool.Stat
}

// Conn represents a database connection with user context
type Conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
	CopyFrom(
		ctx context.Context,
		tableName pgx.Identifier,
		columnNames []string,
		rowSrc pgx.CopyFromSource,
	) (int64, error)
	Release()
}

// Tx represents a database transaction
type Tx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
	CopyFrom(
		ctx context.Context,
		tableName pgx.Identifier,
		columnNames []string,
		rowSrc pgx.CopyFromSource,
	) (int64, error)
	Prepare(ctx context.Context, name, sql string) (*pgxpool.Conn, error)
}

// TxFunc is a function that executes within a transaction
type TxFunc func(ctx context.Context, tx Tx) error

// PoolStats contains connection pool statistics
type PoolStats struct {
	AcquiredConns     int32
	ConstructingConns int32
	IdleConns         int32
	TotalConns        int32
	MaxConns          int32
}

// Repository represents a base repository interface
type Repository interface {
	// Query operations
	Get(ctx context.Context, userID string, id string, dest any) error
	List(ctx context.Context, userID string, filter Filter, dest any) error
	Count(ctx context.Context, userID string, filter Filter) (int64, error)

	// Mutation operations
	Create(ctx context.Context, userID string, entity any) error
	Update(ctx context.Context, userID string, id string, entity any) error
	Delete(ctx context.Context, userID string, id string) error

	// Batch operations
	CreateBatch(ctx context.Context, userID string, entities []any) error
	UpdateBatch(ctx context.Context, userID string, entities []any) error
	DeleteBatch(ctx context.Context, userID string, ids []string) error
}

// Filter represents query filters
type Filter struct {
	Where   map[string]any
	OrderBy string
	Limit   int
	Offset  int
	Joins   []Join
	Select  []string
	GroupBy []string
}

// Join represents a SQL join
type Join struct {
	Table     string
	Condition string
	Type      JoinType
}

// JoinType represents the type of SQL join
type JoinType string

const (
	InnerJoin JoinType = "INNER"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
	FullJoin  JoinType = "FULL"
)

// QueryBuilder helps construct SQL queries
type QueryBuilder struct {
	table   string
	selects []string
	wheres  []string
	args    []any
	orderBy string
	limit   int
	offset  int
	joins   []string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:   table,
		selects: []string{"*"},
		args:    make([]any, 0),
	}
}

// Select adds fields to select
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	qb.selects = fields
	return qb
}

// Where adds a where condition
func (qb *QueryBuilder) Where(condition string, args ...any) *QueryBuilder {
	qb.wheres = append(qb.wheres, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy sets the order by clause
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	qb.orderBy = orderBy
	return qb
}

// Limit sets the limit
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the offset
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Join adds a join clause
func (qb *QueryBuilder) Join(joinType JoinType, table string, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, string(joinType)+" JOIN "+table+" ON "+condition)
	return qb
}
