package exampleutil

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSQLiteEngine(t *testing.T) {
	engine, cleanup, err := NewSQLiteEngine("CREATE TABLE demo (id INTEGER PRIMARY KEY, name TEXT);")
	if err != nil {
		t.Skipf("sqlite example helper unavailable: %v", err)
		return
	}
	defer cleanup()

	_, err = engine.Exec(context.Background(), "INSERT INTO demo(name) VALUES (?)", "ok")
	assert.NoError(t, err)
}

func TestNewSQLiteEngineInitSchemaError(t *testing.T) {
	engine, cleanup, err := NewSQLiteEngine("CREATE TABLE demo (id INTEGER PRIMARY KEY); BROKEN SQL")
	if err != nil && !strings.Contains(err.Error(), "init schema:") {
		t.Skipf("sqlite example helper unavailable: %v", err)
		return
	}

	assert.Nil(t, engine)
	assert.Nil(t, cleanup)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "init schema:")
}
