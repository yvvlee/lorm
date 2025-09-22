package lorm

import (
	"context"
	"database/sql"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type Engine struct {
	config *Config
	db     *sql.DB
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
	}
	for _, o := range option {
		o(config)
	}
	engine := &Engine{
		config: config,
		db:     db,
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
	return e.config.placeholderFormat
}
func (e *Engine) Escaper() names.Escaper {
	return e.config.escaper
}

func (e *Engine) DB(ctx context.Context) *Session {
	if s, ok := ctx.Value(e).(*Session); ok {
		return s
	}
	return &Session{engine: e}
}

func (e *Engine) TX(ctx context.Context, fn func(context.Context) error) error {
	//若当前开启了事务，则取事务session
	if _, ok := ctx.Value(e).(Session); ok {
		return fn(ctx)
	}
	session, err := e.beginTxSession(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := session.close(); err != nil {
			//todo log
			//s.logger.WithContext(ctx).Error("xorm close transaction session error: ", err)
		}
	}()
	if err = fn(context.WithValue(ctx, e, session)); err != nil {
		return err
	}
	return session.commit()
}

func (e *Engine) beginTxSession(ctx context.Context) (*Session, error) {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Session{
		engine: e,
		tx:     tx,
	}, nil
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
		"postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach",
		//SQLite
		"ql":
		return builder.Dollar
	case //Oracle
		"oci8", "ora", "goracle", "godror":
		return builder.Colon
	case //SQL Server
		"sqlserver", "mssql":
		return builder.AtP
	default:
		return builder.Question
	}
}

func Escaper(driverName string) names.Escaper {
	switch driverName {
	case //PostgreSQL
		"postgres", "pgx", "pq-timeouts", "cloudsqlpostgres", "nrpostgres", "cockroach",
		//Oracle
		"oci8", "ora", "goracle", "godror",
		//SQLite
		"ql":
		return names.NewQuoter('"', '"')
	case //SQL Server
		"sqlserver", "mssql":
		return names.NewQuoter('[', ']')
	case "mysql":
		return names.NewQuoter('`', '`')
	default:
		return names.NoEscaper
	}
}
