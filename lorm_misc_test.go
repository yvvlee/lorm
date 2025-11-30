package lorm

import (
	"github.com/stretchr/testify/assert"
	"github.com/yvvlee/lorm/builder"
	"testing"
)

func TestPlaceholderBranches(t *testing.T) {
	assert.Equal(t, builder.Dollar, Placeholder("postgres"))
	assert.Equal(t, builder.Dollar, Placeholder("pgx"))
	assert.Equal(t, builder.Dollar, Placeholder("ql"))
	assert.Equal(t, builder.Colon, Placeholder("oci8"))
	assert.Equal(t, builder.Colon, Placeholder("godror"))
	assert.Equal(t, builder.AtP, Placeholder("sqlserver"))
	assert.Equal(t, builder.Question, Placeholder("mysql"))
	assert.Equal(t, builder.Question, Placeholder("unknown"))
}

func TestEscaperBranches(t *testing.T) {
	// PostgreSQL / Oracle / SQLite
	q := Escaper("postgres")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	q = Escaper("oci8")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	q = Escaper("ql")
	assert.Equal(t, "\"x\"", q.Escape("x"))

	// SQL Server
	q = Escaper("sqlserver")
	assert.Equal(t, "[x]", q.Escape("x"))

	// MySQL
	q = Escaper("mysql")
	assert.Equal(t, "`x`", q.Escape("x"))

	// default no escaper
	q = Escaper("unknown")
	assert.Equal(t, "x", q.Escape("x"))
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
