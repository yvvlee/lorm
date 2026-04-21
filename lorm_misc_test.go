package lorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm/builder"
)

func TestPlaceholderBranches(t *testing.T) {
	assert.Equal(t, builder.Dollar, Placeholder("postgres"))
	assert.Equal(t, builder.Dollar, Placeholder("postgresql"))
	assert.Equal(t, builder.Dollar, Placeholder("pgx"))
	assert.Equal(t, builder.Dollar, Placeholder("pq"))
	assert.Equal(t, builder.Dollar, Placeholder("ql"))
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
	_, err := connect("__invalid_driver__", "dsn")
	assert.Error(t, err)

	// Registered driver with bad DSN: use postgres
	// driver imported in lorm_test.go
	_, err = connect("postgres", "postgres://wrong:wrong@127.0.0.1:1/db")
	assert.Error(t, err)
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
