package lorm

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
)

func TestPlaceholderBranches(t *testing.T) {
	assert.Equal(t, builder.Dollar, Placeholder("postgres"))
	assert.Equal(t, builder.Dollar, Placeholder("postgresql"))
	assert.Equal(t, builder.Dollar, Placeholder("pgx"))
	assert.Equal(t, builder.Dollar, Placeholder("pq"))
	assert.Equal(t, builder.Question, Placeholder("ql"))
	assert.Equal(t, builder.Question, Placeholder("mysql"))
	assert.Equal(t, builder.Question, Placeholder("unknown"))
}

func TestEscaperBranches(t *testing.T) {
	// PostgreSQL / SQLite
	q := Escaper("postgres")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	q = Escaper("postgresql")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	q = Escaper("ql")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	// MySQL
	q = Escaper("mysql")
	assert.Equal(t, "`x`", q.Escape("x"))

	// default no escaper
	q = Escaper("unknown")
	assert.Equal(t, "x", q.Escape("x"))
}

func TestSupportsForUpdateBranches(t *testing.T) {
	assert.True(t, newDialectTestEngine("mysql").SupportsForUpdate())
	assert.True(t, newDialectTestEngine("postgres").SupportsForUpdate())
	assert.True(t, newDialectTestEngine("postgresql").SupportsForUpdate())
	assert.True(t, newDialectTestEngine("pq").SupportsForUpdate())
	assert.False(t, newDialectTestEngine("sqlite3").SupportsForUpdate())
	assert.False(t, newDialectTestEngine("unknown").SupportsForUpdate())
}

func TestSupportsReturningBranches(t *testing.T) {
	assert.True(t, newDialectTestEngine("postgres").SupportsReturning())
	assert.True(t, newDialectTestEngine("postgresql").SupportsReturning())
	assert.True(t, newDialectTestEngine("pq").SupportsReturning())
	assert.True(t, newDialectTestEngine("pgx").SupportsReturning())
	assert.False(t, newDialectTestEngine("mysql").SupportsReturning())
}

func TestSupportsLastInsertIDBranches(t *testing.T) {
	assert.False(t, newDialectTestEngine("postgres").SupportsLastInsertId())
	assert.False(t, newDialectTestEngine("postgresql").SupportsLastInsertId())
	assert.False(t, newDialectTestEngine("pq").SupportsLastInsertId())
	assert.True(t, newDialectTestEngine("mysql").SupportsLastInsertId())
	assert.True(t, newDialectTestEngine("sqlite3").SupportsLastInsertId())
}

func TestConnectErrorBranches(t *testing.T) {
	// Unregistered driver should return error immediately
	_, err := connect(context.Background(), "__invalid_driver__", "dsn")
	assert.Error(t, err)

	// Alias without a matching registered driver should fail immediately.
	_, err = connect(context.Background(), "postgres", "postgres://wrong:wrong@127.0.0.1:1/db")
	assert.Error(t, err)
	assert.ErrorContains(t, err, `unknown driver "postgres"`)

	driverName := registerPingErrorDriver(errors.New("ping failed"))
	_, err = connect(context.Background(), driverName, "dsn")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "ping failed")
}

func TestEngineInitAppliesConfigAndFallbacks(t *testing.T) {
	db, err := openScriptedQueryDB(t, newScriptedQueryRecorder())
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer db.Close()

	engine := &Engine{
		config: &Config{
			maxIdleConns:    2,
			maxOpenConns:    3,
			connMaxLifetime: time.Second,
			connMaxIdleTime: time.Second,
		},
		db: db,
	}
	engine.init()

	assert.Equal(t, 3, db.Stats().MaxOpenConnections)
	assert.Equal(t, builder.Question, (&Engine{config: &Config{}}).Placeholder())
	assert.Equal(t, "field", (&Engine{config: &Config{}}).Escaper().Escape("field"))
}

func TestNewEngineAndContextCoverage(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	driverName := registerScriptedQueryDriver(recorder)

	engine, err := NewEngine(
		driverName,
		"",
		WithMaxIdleConns(2),
		WithMaxOpenConns(3),
		WithConnMaxLifetime(time.Second),
		WithConnMaxIdleTime(time.Second),
		WithLogger(nil),
	)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.IsType(t, noopLogger{}, engine.logger)
	assert.Equal(t, 3, engine.db.Stats().MaxOpenConnections)
	assert.NoError(t, engine.Close())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine, err = NewEngineContext(ctx, driverName, filepath.Join(t.TempDir(), "cancelled.db"))
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, engine)
}

func TestNoopLoggerMethods(t *testing.T) {
	logger := noopLogger{}
	logger.DebugContext(context.Background(), "debug")
	logger.InfoContext(context.Background(), "info")
	logger.WarnContext(context.Background(), "warn")
	logger.ErrorContext(context.Background(), "error")
}

func newDialectTestEngine(driverName string) *Engine {
	dialect := defaultDialectConfig(driverName)
	return &Engine{
		config: &Config{
			driverName:           driverName,
			placeholderFormat:    dialect.placeholderFormat,
			escaper:              dialect.escaper,
			supportsReturning:    dialect.supportsReturning,
			supportsLastInsertID: dialect.supportsLastInsertID,
			supportsForUpdate:    dialect.supportsForUpdate,
		},
	}
}
