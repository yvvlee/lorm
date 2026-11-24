package lorm

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewEngineWithDBRejectsNil(t *testing.T) {
	for _, name := range []string{"sqlite3", "postgres"} {
		engine, err := NewEngineWithDB(context.Background(), nil, name)
		require.ErrorContains(t, err, "non-nil")
		require.Nil(t, engine)
	}
}

func TestNewEngineWithDBSharesPool(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)
	engine, err := NewEngineWithDB(context.Background(), db, "postgres", WithLogger(nil), WithMaxOpenConns(3))
	require.NoError(t, err)
	require.Same(t, db, engine.DB())
	require.Equal(t, 3, db.Stats().MaxOpenConnections)
	require.Nil(t, engine.logger)
	require.Nil(t, recorder.LastQuery())
	recorder.QueueQueryRows([]string{"value"}, []driver.Value{int64(42)})
	rows, err := engine.SQL(context.Background(), "SELECT 42")
	require.NoError(t, err)
	require.True(t, rows.Next())
	var value int64
	require.NoError(t, rows.Scan(&value))
	require.EqualValues(t, 42, value)
	require.NoError(t, rows.Close())
	require.NoError(t, engine.Close())
	require.Error(t, db.Ping())
}

func TestNewEngineWithDBDetectionFailureKeepsPool(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"version"}, []driver.Value{"invalid"})
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)
	engine, err := NewEngineWithDB(context.Background(), db, "sqlite3")
	require.Error(t, err)
	require.Nil(t, engine)
	require.NoError(t, db.Ping())
}

func TestSQLLogFieldsCapacityAndValues(t *testing.T) {
	engine := &Engine{
		config: &Config{
			logSQLArgs: true,
		},
	}
	ctx := context.WithValue(context.Background(), sessionIDKey{}, "session")
	failure := errors.New("query failed")
	for _, count := range []int{0, 1000} {
		args := make([]any, count)
		fields := engine.sqlLogFields(ctx, "SELECT ?", args, failure, time.Second)
		require.Equal(t, []any{"sessionID", "session", "SQL", "SELECT ?", "err", failure, "args", args, "latency", float64(1)}, fields)
		require.Equal(t, 10, cap(fields))
	}
}
