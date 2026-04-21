package lorm

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

// Engine wraps a sql.DB and the driver-specific behavior lorm needs.
type Engine struct {
	config *Config
	db     *sql.DB
	logger Logger
}

type dialectConfig struct {
	placeholderFormat    builder.PlaceholderFormat
	escaper              names.Escaper
	supportsReturning    bool
	supportsLastInsertID bool
	supportsForUpdate    bool
}

// NewEngine opens a database connection and applies the provided options.
func NewEngine(driverName, dsn string, option ...Option) (*Engine, error) {
	db, err := connect(driverName, dsn)
	if err != nil {
		return nil, err
	}
	dialect := defaultDialectConfig(driverName)
	config := &Config{
		driverName:           driverName,
		dsn:                  dsn,
		placeholderFormat:    dialect.placeholderFormat,
		escaper:              dialect.escaper,
		supportsReturning:    dialect.supportsReturning,
		supportsLastInsertID: dialect.supportsLastInsertID,
		supportsForUpdate:    dialect.supportsForUpdate,
		logger:               defaultLogger,
	}
	for _, o := range option {
		o(config)
	}
	engine := &Engine{
		config: config,
		db:     db,
		logger: config.logger,
	}
	engine.init()
	return engine, nil
}

// Close closes the underlying database pool.
func (e *Engine) Close() error {
	return e.db.Close()
}

func (e *Engine) init() {
	if e.config.maxIdleConns > 0 {
		e.db.SetMaxIdleConns(e.config.maxIdleConns)
	}
	if e.config.maxOpenConns > 0 {
		e.db.SetMaxOpenConns(e.config.maxOpenConns)
	}
	if e.config.connMaxLifetime > 0 {
		e.db.SetConnMaxLifetime(e.config.connMaxLifetime)
	}
	if e.config.connMaxIdleTime > 0 {
		e.db.SetConnMaxIdleTime(e.config.connMaxIdleTime)
	}
}

// Placeholder returns the placeholder format configured for the engine.
func (e *Engine) Placeholder() builder.PlaceholderFormat {
	if e.config.placeholderFormat == nil {
		return builder.Question
	}
	return e.config.placeholderFormat
}

// Escaper returns the identifier escaper configured for the engine.
func (e *Engine) Escaper() names.Escaper {
	if e.config.escaper == nil {
		return names.NoEscaper
	}
	return e.config.escaper
}

// DriverName returns the configured database driver name.
func (e *Engine) DriverName() string {
	return e.config.driverName
}

// SupportsReturning returns true if the database driver supports RETURNING clause
func (e *Engine) SupportsReturning() bool {
	return e.config.supportsReturning
}

// SupportsLastInsertId returns true if the database driver supports LastInsertId
func (e *Engine) SupportsLastInsertId() bool {
	return e.config.supportsLastInsertID
}

// SupportsForUpdate returns true if the database driver supports FOR UPDATE.
func (e *Engine) SupportsForUpdate() bool {
	return e.config.supportsForUpdate
}

func (e *Engine) session(ctx context.Context) *session {
	if s, ok := ctx.Value(e).(*session); ok {
		return s
	}
	return &session{engine: e}
}

type sessionIDKey struct{}

// TX runs fn in a transaction and reuses the current session for nested calls.
func (e *Engine) TX(ctx context.Context, fn func(context.Context) error) error {
	return e.tx(ctx, nil, fn)
}

// TXWithOptions runs fn in a transaction using the provided sql.TxOptions.
//
// Nested calls still reuse the existing transaction from the incoming context.
func (e *Engine) TXWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(context.Context) error) error {
	return e.tx(ctx, opts, fn)
}

func (e *Engine) tx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context) error) error {
	// If a transaction is currently open, reuse the existing session
	if _, ok := ctx.Value(e).(*session); ok {
		return fn(ctx)
	}
	s, err := e.beginTxSession(ctx, opts)
	if err != nil {
		e.logger.ErrorContext(ctx, "BEGIN TRANSACTION failed", "err", err)
		return err
	}
	sessionID := uuid.NewString()
	e.logger.InfoContext(ctx, "BEGIN TRANSACTION", "sessionID", sessionID)
	defer func() {
		if err := s.close(); err != nil {
			e.logger.ErrorContext(ctx, "lorm close transaction session error", "sessionID", sessionID, "err", err)
		}
	}()
	innerCtx := context.WithValue(ctx, e, s)
	innerCtx = context.WithValue(innerCtx, sessionIDKey{}, sessionID)
	if err = fn(innerCtx); err != nil {
		e.logger.InfoContext(ctx, "ROLLBACK", "sessionID", sessionID, "err", err)
		return err
	}
	err = s.commit()
	e.logger.InfoContext(ctx, "COMMIT", "sessionID", sessionID, "err", err)
	return err
}

func (e *Engine) beginTxSession(ctx context.Context, opts *sql.TxOptions) (*session, error) {
	tx, err := e.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &session{
		engine: e,
		tx:     tx,
	}, nil
}

// Exec executes a statement against the current session or transaction.
func (e *Engine) Exec(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error", e.sqlLogFields(ctx, query, args, err, time.Since(startTime))...)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success", e.sqlLogFields(ctx, query, args, nil, time.Since(startTime))...)
	}()
	result, err = e.session(ctx).Exec(ctx, query, args...)
	return
}

// Query executes a query against the current session or transaction.
func (e *Engine) Query(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error", e.sqlLogFields(ctx, query, args, err, time.Since(startTime))...)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success", e.sqlLogFields(ctx, query, args, nil, time.Since(startTime))...)
	}()
	rows, err = e.session(ctx).Query(ctx, query, args...)
	return
}

// Exist reports whether the query returns at least one row.
func (e *Engine) Exist(ctx context.Context, query string, args ...any) (exist bool, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error", e.sqlLogFields(ctx, query, args, err, time.Since(startTime))...)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success", e.sqlLogFields(ctx, query, args, nil, time.Since(startTime))...)
	}()
	exist, err = e.session(ctx).Exist(ctx, query, args...)
	return
}

func (e *Engine) sqlLogFields(ctx context.Context, query string, args []any, err error, elapsed time.Duration) []any {
	fields := []any{
		"sessionID", ctx.Value(sessionIDKey{}),
		"SQL", query,
	}
	if err != nil {
		fields = append(fields, "err", err)
	}
	if e.config != nil && e.config.logSQLArgs {
		fields = append(fields, "args", args)
	}
	fields = append(fields, "executeTime", elapsed.Seconds())
	return fields
}

// connect to a database and verify with a ping.
func connect(driverName, dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Placeholder returns the default placeholder format for a driver name.
func Placeholder(driverName string) builder.PlaceholderFormat {
	return defaultDialectConfig(driverName).placeholderFormat
}

// Escaper returns the default identifier escaper for a driver name.
func Escaper(driverName string) names.Escaper {
	return defaultDialectConfig(driverName).escaper
}

func defaultDialectConfig(driverName string) dialectConfig {
	switch driverName {
	case //PostgreSQL
		"postgres", "postgresql", "pgx", "pq", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach", "crdb-postgres":
		return dialectConfig{
			placeholderFormat:    builder.Dollar,
			escaper:              names.NewQuoter('"', '"'),
			supportsReturning:    true,
			supportsLastInsertID: false,
			supportsForUpdate:    true,
		}
	case //SQLite
		"sqlite", "sqlite3", "ql":
		return dialectConfig{
			placeholderFormat:    builder.Dollar,
			escaper:              names.NewQuoter('"', '"'),
			supportsReturning:    false,
			supportsLastInsertID: true,
			supportsForUpdate:    false,
		}
	case //MySQL, MariaDB
		"mysql", "mariadb":
			return dialectConfig{
			placeholderFormat:    builder.Question,
			escaper:              names.NewQuoter('`', '`'),
			supportsReturning:    false,
			supportsLastInsertID: true,
			supportsForUpdate:    true,
		}
	default:
		return dialectConfig{
			placeholderFormat:    builder.Question,
			escaper:              names.NoEscaper,
			supportsReturning:    false,
			supportsLastInsertID: true,
			supportsForUpdate:    false,
		}
	}
}

// Execer executes SQL statements with context.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Querier executes SQL queries with context.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// DBProxy is the common interface shared by *sql.DB and *sql.Tx.
type DBProxy interface {
	Execer
	Querier
}
