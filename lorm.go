package lorm

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type Engine struct {
	config *Config
	db     *sql.DB
	logger Logger
}

func NewEngine(driverName, dsn string, option ...Option) (*Engine, error) {
	db, err := connect(driverName, dsn)
	if err != nil {
		return nil, err
	}
	config := &Config{
		driverName:        driverName,
		dsn:               dsn,
		placeholderFormat: Placeholder(driverName),
		escaper:           Escaper(driverName),
		logger:            defaultLogger,
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

func (e *Engine) Placeholder() builder.PlaceholderFormat {
	if e.config.placeholderFormat == nil {
		return builder.Question
	}
	return e.config.placeholderFormat
}

func (e *Engine) Escaper() names.Escaper {
	if e.config.escaper == nil {
		return names.NoEscaper
	}
	return e.config.escaper
}

func (e *Engine) DriverName() string {
	return e.config.driverName
}

// SupportsReturning returns true if the database driver supports RETURNING clause
func (e *Engine) SupportsReturning() bool {
	switch e.config.driverName {
	case "postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach":
		return true
	default:
		return false
	}
}

// SupportsLastInsertId returns true if the database driver supports LastInsertId
func (e *Engine) SupportsLastInsertId() bool {
	switch e.config.driverName {
	case "postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach":
		return false
	case //Oracle
		"oci8", "ora", "oracle", "goracle", "godror":
		return false
	default:
		return true
	}
}

func (e *Engine) session(ctx context.Context) *session {
	if s, ok := ctx.Value(e).(*session); ok {
		return s
	}
	return &session{engine: e}
}

type sessionIDKey struct{}

func (e *Engine) TX(ctx context.Context, fn func(context.Context) error) error {
	// If a transaction is currently open, get the transaction session
	if _, ok := ctx.Value(e).(session); ok {
		return fn(ctx)
	}
	s, err := e.beginTxSession(ctx)
	sessionID := uuid.NewString()
	e.logger.InfoContext(ctx, "BEGIN TRANSACTION", "sessionID", sessionID, "err", err)
	if err != nil {
		return err
	}
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

func (e *Engine) beginTxSession(ctx context.Context) (*session, error) {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &session{
		engine: e,
		tx:     tx,
	}, nil
}

func (e *Engine) Exec(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error",
				"sessionID", ctx.Value(sessionIDKey{}),
				"err", err,
				"SQL", query,
				"args", args,
				"executeTime", time.Since(startTime).Seconds(),
			)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success",
			"sessionID", ctx.Value(sessionIDKey{}),
			"SQL", query,
			"args", args,
			"executeTime", time.Since(startTime).Seconds(),
		)
	}()
	result, err = e.session(ctx).Exec(ctx, query, args...)
	return
}

func (e *Engine) Query(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error",
				"sessionID", ctx.Value(sessionIDKey{}),
				"err", err,
				"SQL", query,
				"args", args,
				"executeTime", time.Since(startTime).Seconds(),
			)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success",
			"sessionID", ctx.Value(sessionIDKey{}),
			"SQL", query,
			"args", args,
			"executeTime", time.Since(startTime).Seconds(),
		)
	}()
	rows, err = e.session(ctx).Query(ctx, query, args...)
	return
}

func (e *Engine) Exist(ctx context.Context, query string, args ...any) (exist bool, err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			e.logger.ErrorContext(ctx, "SQL execute error",
				"err", err,
				"SQL", query,
				"args", args,
				"executeTime", time.Since(startTime).Seconds(),
			)
			return
		}
		e.logger.InfoContext(ctx, "SQL execute success",
			"SQL", query,
			"args", args,
			"executeTime", time.Since(startTime).Seconds(),
		)
	}()
	exist, err = e.session(ctx).Exist(ctx, query, args...)
	return
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

func Placeholder(driverName string) builder.PlaceholderFormat {
	switch driverName {
	case //PostgreSQL
		"postgres", "postgresql", "pgx", "pq", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach", "crdb-postgres",
		//SQLite
		"sqlite", "sqlite3", "ql":
		return builder.Dollar
	case //Oracle
		"oci8", "ora", "oracle", "goracle", "godror":
		return builder.Colon
	case //SQL Server
		"sqlserver", "mssql", "azuresql":
		return builder.AtP
	case //MySQL, MariaDB
		"mysql", "mariadb":
		return builder.Question
	default:
		return builder.Question
	}
}

func Escaper(driverName string) names.Escaper {
	switch driverName {
	case //PostgreSQL
		"postgres", "postgresql", "pgx", "pq", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach", "crdb-postgres":
		return names.NewQuoter('"', '"')
	case //Oracle
		"oci8", "ora", "oracle", "goracle", "godror":
		return names.NewQuoter('"', '"')
	case //SQLite
		"sqlite", "sqlite3", "ql":
		return names.NewQuoter('"', '"')
	case //SQL Server
		"sqlserver", "mssql", "azuresql":
		return names.NewQuoter('[', ']')
	case //MySQL, MariaDB
		"mysql", "mariadb":
		return names.NewQuoter('`', '`')
	default:
		return names.NoEscaper
	}
}

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type DBProxy interface {
	Execer
	Querier
}
