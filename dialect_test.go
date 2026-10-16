package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLiteRuntimeDialect checks version boundaries and explicit overrides.
// Parameters: t manages the test lifecycle.
// Returns: none.
// Throws: none.
func TestSQLiteRuntimeDialect(t *testing.T) {
	cases := []struct {
		name       string
		driverName string
		version    string
		options    []Option
		returning  bool
	}{
		{
			name:       "old release",
			driverName: "sqlite3",
			version:    "3.34.1",
		},
		{
			name:       "first supported release",
			driverName: "sqlite3",
			version:    "3.35.0",
			returning:  true,
		},
		{
			name:       "sqlite driver name",
			driverName: "sqlite",
			version:    "3.53.4",
			returning:  true,
		},
		{
			name:       "numeric comparison",
			driverName: "sqlite3",
			version:    "3.9.0",
		},
		{
			name:       "legacy two components",
			driverName: "sqlite3",
			version:    "3.8",
		},
		{
			name:       "legacy fourth component",
			driverName: "sqlite3",
			version:    "3.8.10.2",
		},
		{
			name:       "future minor release",
			driverName: "sqlite3",
			version:    "3.100.0",
			returning:  true,
		},
		{
			name:       "future major release",
			driverName: "sqlite3",
			version:    "4.0.0",
			returning:  true,
		},
		{
			name:       "explicitly disabled",
			driverName: "sqlite3",
			version:    "3.53.4",
			options:    []Option{WithSupportsReturning(false)},
		},
		{
			name:       "explicitly enabled",
			driverName: "sqlite3",
			version:    "3.34.1",
			options:    []Option{WithSupportsReturning(true)},
			returning:  true,
		},
		{
			name:       "complete dialect override",
			driverName: "sqlite3",
			version:    "3.53.4",
			options:    []Option{WithDialectConfig(DefaultDialectConfig("sqlite3"))},
		},
		{
			name:       "last option wins",
			driverName: "sqlite3",
			version:    "3.53.4",
			options:    []Option{WithSupportsReturning(true), WithSupportsReturning(false)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn, dsn := newSQLiteVersionConnection(t, tc.version)
			engine, err := NewEngine(tc.driverName, dsn, tc.options...)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, engine.Close()) })
			assert.Equal(t, tc.returning, engine.SupportsReturning())
			assert.True(t, engine.SupportsLastInsertId())
			assert.False(t, engine.SupportsForUpdate())
			require.Len(t, conn.recorder.queryCalls, 1)
			assert.Equal(t, "SELECT sqlite_version()", conn.recorder.LastQuery().query)
		})
	}
}

// TestSQLiteInitializationFailure checks that unknown capabilities fail closed.
// Parameters: t manages the test lifecycle.
// Returns: none.
// Throws: none.
func TestSQLiteInitializationFailure(t *testing.T) {
	for _, version := range []string{"", "3", "3.x.0", "3.35.invalid", "-3.35.0", "3.35.0.0.0", "18446744073709551616.0.0"} {
		t.Run(version, func(t *testing.T) {
			conn, dsn := newSQLiteVersionConnection(t, version)
			engine, err := NewEngine("sqlite3", dsn)
			require.ErrorContains(t, err, "invalid SQLite version")
			assert.Nil(t, engine)
			assert.EqualValues(t, 1, conn.closeCalls.Load())
		})
	}
	t.Run("query and cleanup errors are preserved", testSQLiteQueryFailure)
	t.Run("initialization context reaches version query", testSQLiteDetectionCancellation)
}

func testSQLiteQueryFailure(t *testing.T) {
	conn, dsn := newSQLiteVersionConnection(t, "3.53.4")
	queryErr := errors.New("version query failed")
	conn.queryHook = func(context.Context) error { return queryErr }
	conn.closeErr = errors.New("connection close failed")

	engine, err := NewEngine("sqlite3", dsn)
	require.ErrorIs(t, err, queryErr)
	assert.ErrorIs(t, err, conn.closeErr)
	assert.Nil(t, engine)
	assert.EqualValues(t, 1, conn.closeCalls.Load())
}

func testSQLiteDetectionCancellation(t *testing.T) {
	conn, dsn := newSQLiteVersionConnection(t, "3.53.4")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn.queryHook = func(queryCtx context.Context) error {
		cancel()
		return queryCtx.Err()
	}

	engine, err := NewEngineContext(ctx, "sqlite3", dsn)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, engine)
	assert.EqualValues(t, 1, conn.closeCalls.Load())
}

// TestOtherDialectsSkipRuntimeDetection protects non-SQLite initialization.
// Parameters: t manages the test lifecycle.
// Returns: none.
// Throws: none.
func TestOtherDialectsSkipRuntimeDetection(t *testing.T) {
	for _, name := range []string{"postgres", "mysql", "custom"} {
		t.Run(name, func(t *testing.T) {
			recorder := newScriptedQueryRecorder()
			db, err := openScriptedQueryDB(t, recorder)
			require.NoError(t, err)
			dialect, err := detectDialectConfig(context.Background(), db, name)
			require.NoError(t, err)
			assert.Equal(t, DefaultDialectConfig(name), dialect)
			assert.Empty(t, recorder.queryCalls)
		})
	}
}

var (
	sqliteVersionDriverOnce sync.Once
	sqliteVersionDriverSeq  atomic.Uint64
	sqliteVersionDriverDB   sqliteVersionDriver
)

type sqliteVersionDriver struct {
	connections sync.Map
}

// Open supplies the connection assigned to this test's data source.
// Parameters: dsn identifies a registered test connection.
// Returns: the test connection and nil.
// Throws: panics if the test omitted connection registration.
func (d *sqliteVersionDriver) Open(dsn string) (driver.Conn, error) {
	conn, _ := d.connections.Load(dsn)
	return conn.(*sqliteVersionConn), nil
}

type sqliteVersionConn struct {
	*scriptedQueryConn
	queryHook  func(context.Context) error
	closeErr   error
	closeCalls atomic.Int64
}

// QueryContext allows errors to occur after Ping has succeeded.
// Parameters: ctx controls the query; query and args are the SQL request.
// Returns: scripted rows or the configured error.
// Throws: none.
func (c *sqliteVersionConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := c.queryHook(ctx); err != nil {
		return nil, err
	}
	return c.scriptedQueryConn.QueryContext(ctx, query, args)
}

// Close records whether initialization releases its connection on failure.
// Parameters: none.
// Returns: the configured close error.
// Throws: none.
func (c *sqliteVersionConn) Close() error {
	c.closeCalls.Add(1)
	return c.closeErr
}

func newSQLiteVersionConnection(t *testing.T, version string) (*sqliteVersionConn, string) {
	t.Helper()
	sqliteVersionDriverOnce.Do(func() {
		sql.Register("sqlite", &sqliteVersionDriverDB)
		sql.Register("sqlite3", &sqliteVersionDriverDB)
	})
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"sqlite_version()"}, []driver.Value{version})
	conn := &sqliteVersionConn{
		scriptedQueryConn: &scriptedQueryConn{recorder: recorder},
		queryHook:         func(context.Context) error { return nil },
	}
	dsn := strconv.FormatUint(sqliteVersionDriverSeq.Add(1), 10)
	sqliteVersionDriverDB.connections.Store(dsn, conn)
	t.Cleanup(func() { sqliteVersionDriverDB.connections.Delete(dsn) })
	return conn, dsn
}
