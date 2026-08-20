package exampleutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteEngine(t *testing.T) {
	engine, cleanup, err := NewSQLiteEngine("CREATE TABLE demo (id INTEGER PRIMARY KEY, name TEXT);")
	if IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example helper unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	_, err = engine.Exec(context.Background(), "INSERT INTO demo(name) VALUES (?)", "ok")
	assert.NoError(t, err)
}

func TestNewSQLiteEngineInitSchemaError(t *testing.T) {
	engine, cleanup, err := NewSQLiteEngine("CREATE TABLE demo (id INTEGER PRIMARY KEY); BROKEN SQL")
	if IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example helper unavailable: %v", err)
	}
	assert.Nil(t, engine)
	assert.Nil(t, cleanup)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "init schema:")
}
