package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type scriptedQueryResult struct {
	columns []string
	rows    [][]driver.Value
}

type scriptedQueryRecorder struct {
	mu         sync.Mutex
	queryCalls []conversionCall
	results    []scriptedQueryResult
}

func newScriptedQueryRecorder() *scriptedQueryRecorder {
	return &scriptedQueryRecorder{}
}

func (r *scriptedQueryRecorder) QueueQueryRows(columns []string, rows ...[]driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copiedRows := make([][]driver.Value, len(rows))
	for i, row := range rows {
		copiedRows[i] = append([]driver.Value(nil), row...)
	}
	r.results = append(r.results, scriptedQueryResult{
		columns: append([]string(nil), columns...),
		rows:    copiedRows,
	})
}

func (r *scriptedQueryRecorder) RecordQuery(query string, args []driver.NamedValue) scriptedQueryResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queryCalls = append(r.queryCalls, conversionCall{query: query, args: namedValuesToArgs(args)})
	if len(r.results) == 0 {
		return scriptedQueryResult{}
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result
}

func (r *scriptedQueryRecorder) LastQuery() *conversionCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queryCalls) == 0 {
		return nil
	}
	call := r.queryCalls[len(r.queryCalls)-1]
	return &call
}

var scriptedQueryDriverSeq atomic.Uint64

func registerScriptedQueryDriver(recorder *scriptedQueryRecorder) string {
	driverName := "lorm_scripted_query_" + strconv.FormatUint(scriptedQueryDriverSeq.Add(1), 10)
	sql.Register(driverName, &scriptedQueryDriver{recorder: recorder})
	return driverName
}

func openScriptedQueryDB(t *testing.T, recorder *scriptedQueryRecorder) (*sql.DB, error) {
	t.Helper()
	driverName := registerScriptedQueryDriver(recorder)
	db, err := sql.Open(driverName, "")
	if err == nil {
		t.Cleanup(func() { _ = db.Close() })
	}
	return db, err
}

func newScriptedEngine(t *testing.T, recorder *scriptedQueryRecorder) *Engine {
	t.Helper()
	db, err := openScriptedQueryDB(t, recorder)
	require.NoError(t, err)
	return &Engine{
		config: &Config{
			driverName:           "mysql",
			placeholderFormat:    builder.Question,
			escaper:              names.NoEscaper,
			supportsReturning:    false,
			supportsLastInsertID: true,
			supportsForUpdate:    true,
			logger:               testLogger{},
		},
		db:     db,
		logger: testLogger{},
	}
}

type scriptedQueryDriver struct {
	recorder *scriptedQueryRecorder
}

func (d *scriptedQueryDriver) Open(string) (driver.Conn, error) {
	return &scriptedQueryConn{recorder: d.recorder}, nil
}

type scriptedQueryConn struct {
	recorder *scriptedQueryRecorder
}

func (c *scriptedQueryConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *scriptedQueryConn) Close() error                        { return nil }
func (c *scriptedQueryConn) Begin() (driver.Tx, error)           { return scriptedQueryTx{}, nil }
func (c *scriptedQueryConn) Ping(ctx context.Context) error      { return ctx.Err() }

func (c *scriptedQueryConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return conversionResult(1), nil
}

func (c *scriptedQueryConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	result := c.recorder.RecordQuery(query, args)
	return &scriptedQueryRows{columns: result.columns, rows: result.rows}, nil
}

type scriptedQueryTx struct{}

func (scriptedQueryTx) Commit() error   { return nil }
func (scriptedQueryTx) Rollback() error { return nil }

type scriptedQueryRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *scriptedQueryRows) Columns() []string { return r.columns }
func (r *scriptedQueryRows) Close() error      { return nil }

func (r *scriptedQueryRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.index]
	for i := range dest {
		dest[i] = row[i]
	}
	r.index++
	return nil
}

type txBehavior struct {
	beginErr    error
	commitErr   error
	rollbackErr error
}

var txBehaviorDriverSeq atomic.Uint64

func newTxBehaviorEngine(t *testing.T, behavior txBehavior) *Engine {
	t.Helper()
	driverName := "lorm_tx_behavior_" + strconv.FormatUint(txBehaviorDriverSeq.Add(1), 10)
	sql.Register(driverName, &txBehaviorDriver{behavior: behavior})

	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return &Engine{
		config: &Config{
			driverName:        driverName,
			placeholderFormat: builder.Question,
			escaper:           names.NoEscaper,
			logger:            testLogger{},
		},
		db:     db,
		logger: testLogger{},
	}
}

type txBehaviorDriver struct {
	behavior txBehavior
}

func (d *txBehaviorDriver) Open(string) (driver.Conn, error) {
	return &txBehaviorConn{behavior: d.behavior}, nil
}

type txBehaviorConn struct {
	behavior txBehavior
}

func (c *txBehaviorConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *txBehaviorConn) Close() error                        { return nil }
func (c *txBehaviorConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *txBehaviorConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.behavior.beginErr != nil {
		return nil, c.behavior.beginErr
	}
	return txBehaviorTx{behavior: c.behavior}, nil
}
func (c *txBehaviorConn) Ping(context.Context) error { return nil }

type txBehaviorTx struct {
	behavior txBehavior
}

func (tx txBehaviorTx) Commit() error   { return tx.behavior.commitErr }
func (tx txBehaviorTx) Rollback() error { return tx.behavior.rollbackErr }

var pingErrorDriverSeq atomic.Uint64

func registerPingErrorDriver(err error) string {
	driverName := "lorm_ping_error_" + strconv.FormatUint(pingErrorDriverSeq.Add(1), 10)
	sql.Register(driverName, &pingErrorDriver{err: err})
	return driverName
}

type pingErrorDriver struct {
	err error
}

func (d *pingErrorDriver) Open(string) (driver.Conn, error) {
	return &pingErrorConn{err: d.err}, nil
}

type pingErrorConn struct {
	err error
}

func (c *pingErrorConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *pingErrorConn) Close() error                        { return nil }
func (c *pingErrorConn) Begin() (driver.Tx, error)           { return scriptedQueryTx{}, nil }
func (c *pingErrorConn) Ping(context.Context) error          { return c.err }

type errorPlaceholder struct{}

func (errorPlaceholder) ReplacePlaceholders(string) (string, error) {
	return "", errors.New("replace failed")
}

func (errorPlaceholder) PlaceholderString() string { return "?" }
