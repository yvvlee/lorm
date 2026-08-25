package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

// Engine wraps a sql.DB and the driver-specific behavior lorm needs.
// An Engine should be created once per database and used as a global singleton.
// Do not copy an Engine value; transaction isolation relies on pointer identity.
type Engine struct {
	config                   *Config
	db                       *sql.DB
	logger                   Logger
	defaultSelectProjections sync.Map
}

// NewEngine opens a database connection and applies the provided options.
// It uses a background context for the initial connectivity check; use
// NewEngineContext to supply a context with timeout or cancellation.
func NewEngine(driverName, dsn string, option ...Option) (*Engine, error) {
	return NewEngineContext(context.Background(), driverName, dsn, option...)
}

// NewEngineContext is like NewEngine but accepts a context for the initial
// connectivity check so callers can enforce a timeout.
func NewEngineContext(ctx context.Context, driverName, dsn string, option ...Option) (*Engine, error) {
	db, err := connect(ctx, driverName, dsn)
	if err != nil {
		return nil, err
	}
	config := &Config{
		driverName: driverName,
		dsn:        dsn,
		Dialect:    DefaultDialectConfig(driverName),
		logger:     defaultLogger,
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
	if e.config.maxIdleConnsSet {
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
	if e.config.Dialect.PlaceholderFormat == nil {
		return builder.Question
	}
	return e.config.Dialect.PlaceholderFormat
}

// Escaper returns the identifier escaper configured for the engine.
func (e *Engine) Escaper() names.Escaper {
	if e.config.Dialect.Escaper == nil {
		return names.NoEscaper
	}
	return e.config.Dialect.Escaper
}

// DriverName returns the configured database driver name.
func (e *Engine) DriverName() string {
	return e.config.driverName
}

// SupportsReturning returns true if the database driver supports RETURNING clause
func (e *Engine) SupportsReturning() bool {
	return e.config.Dialect.SupportsReturning
}

// SupportsLastInsertId returns true if the database driver supports LastInsertId
func (e *Engine) SupportsLastInsertId() bool {
	return e.config.Dialect.SupportsLastInsertID
}

// SupportsForUpdate returns true if the database driver supports FOR UPDATE.
func (e *Engine) SupportsForUpdate() bool {
	return e.config.Dialect.SupportsForUpdate
}

// IgnoreStrategy returns the configured insert-ignore strategy for the dialect.
func (e *Engine) IgnoreStrategy() IgnoreStrategy {
	return e.config.Dialect.IgnoreStrategy
}

func (e *Engine) session(ctx context.Context) *session {
	if s, ok := ctx.Value(e).(*session); ok {
		return s
	}
	return &session{engine: e}
}

func (e *Engine) isTransactionSession(ctx context.Context) bool {
	s, ok := ctx.Value(e).(*session)
	return ok && s != nil && s.engine == e && s.tx != nil
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
	if e.logger == nil {
		return e.txWithoutLogging(ctx, opts, fn)
	}
	tx, err := e.db.BeginTx(ctx, opts)
	if err != nil {
		e.logger.ErrorContext(ctx, "BEGIN TRANSACTION failed", "err", err)
		return err
	}
	sessionID := uuid.NewString()
	e.logger.InfoContext(ctx, "BEGIN TRANSACTION", "sessionID", sessionID)

	s := &session{engine: e, tx: tx}
	defer func() {
		if p := recover(); p != nil {
			rbErr := tx.Rollback()
			e.logger.ErrorContext(ctx, "ROLLBACK (panic)", "sessionID", sessionID, "panic", p, "rbErr", rbErr)
			panic(p)
		}
	}()

	innerCtx := context.WithValue(ctx, e, s)
	innerCtx = context.WithValue(innerCtx, sessionIDKey{}, sessionID)
	if err = fn(innerCtx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			e.logger.ErrorContext(ctx, "ROLLBACK failed", "sessionID", sessionID, "err", err, "rbErr", rbErr)
			return errors.Join(err, rbErr)
		}
		e.logger.InfoContext(ctx, "ROLLBACK", "sessionID", sessionID, "err", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		e.logger.ErrorContext(ctx, "COMMIT failed", "sessionID", sessionID, "err", err)
	} else {
		e.logger.InfoContext(ctx, "COMMIT", "sessionID", sessionID)
	}
	return err
}

func (e *Engine) txWithoutLogging(ctx context.Context, opts *sql.TxOptions, fn func(context.Context) error) (err error) {
	tx, err := e.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	s := &session{engine: e, tx: tx}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	innerCtx := context.WithValue(ctx, e, s)
	if err = fn(innerCtx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// Exec executes a statement against the current session or transaction.
func (e *Engine) Exec(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	if e.logger == nil {
		return e.session(ctx).Exec(ctx, query, args...)
	}
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

// SQL executes a raw query against the current session or transaction.
func (e *Engine) SQL(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	if e.logger == nil {
		return e.session(ctx).Query(ctx, query, args...)
	}
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
	if e.logger == nil {
		return e.session(ctx).Exist(ctx, query, args...)
	}
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

func sessionIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey{}).(string)
	return id, ok && id != ""
}

func (e *Engine) sqlLogFields(ctx context.Context, query string, args []any, err error, elapsed time.Duration) []any {
	fields := []any{
		"SQL", query,
	}
	if sessionID, ok := sessionIDFromContext(ctx); ok {
		fields = append([]any{"sessionID", sessionID}, fields...)
	}
	if err != nil {
		fields = append(fields, "err", err)
	}
	if e.config != nil && e.config.logSQLArgs {
		fields = append(fields, "args", args)
	}
	fields = append(fields, "latency", elapsed.Seconds())
	return fields
}

// connect to a database and verify with a ping.
func connect(ctx context.Context, driverName, dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		err = errors.Join(err, db.Close())
		return nil, err
	}
	return db, nil
}

// Placeholder returns the default placeholder format for a driver name.
func Placeholder(driverName string) builder.PlaceholderFormat {
	return DefaultDialectConfig(driverName).PlaceholderFormat
}

// Escaper returns the default identifier escaper for a driver name.
func Escaper(driverName string) names.Escaper {
	return DefaultDialectConfig(driverName).Escaper
}

// DefaultDialectConfig returns the built-in dialect behavior for a driver name.
func DefaultDialectConfig(driverName string) DialectConfig {
	switch driverName {
	case //PostgreSQL
		"postgres", "postgresql", "pgx", "pq", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach", "crdb-postgres":
		return DialectConfig{
			PlaceholderFormat:    builder.Dollar,
			Escaper:              names.NewQuoter('"', '"'),
			SupportsReturning:    true,
			SupportsLastInsertID: false,
			SupportsForUpdate:    true,
			IgnoreStrategy:       IgnoreConflictSuffix,
		}
	case //SQLite
		"sqlite", "sqlite3", "ql":
		return DialectConfig{
			PlaceholderFormat:    builder.Question,
			Escaper:              names.NewQuoter('"', '"'),
			SupportsReturning:    false,
			SupportsLastInsertID: true,
			SupportsForUpdate:    false,
			IgnoreStrategy:       IgnoreOrKeyword,
		}
	case //MySQL, MariaDB
		"mysql", "mariadb":
		return DialectConfig{
			PlaceholderFormat:    builder.Question,
			Escaper:              names.NewQuoter('`', '`'),
			SupportsReturning:    false,
			SupportsLastInsertID: true,
			SupportsForUpdate:    true,
			IgnoreStrategy:       IgnoreKeyword,
		}
	default:
		return DialectConfig{
			PlaceholderFormat:    builder.Question,
			Escaper:              names.NoEscaper,
			SupportsReturning:    false,
			SupportsLastInsertID: true,
			SupportsForUpdate:    false,
			IgnoreStrategy:       IgnoreKeyword,
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

// ScannerValuer combines the standard database read and write conversion interfaces.
type ScannerValuer interface {
	sql.Scanner
	driver.Valuer
}
