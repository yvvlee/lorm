package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type reservedWordModel struct {
	UnimplementedTable
	ID    int64
	Group string
}

func (m *reservedWordModel) TableName() string { return "order" }
func (m *reservedWordModel) New() Model        { return new(reservedWordModel) }
func (m *reservedWordModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "group":
		return &m.Group
	default:
		return nil
	}
}
func (m *reservedWordModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		TableName:   m.TableName(),
		PrimaryKeys: []string{"id"},
		Fields: []*FieldDescriptor{
			{DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement},
			{DBField: "group"},
		},
	}
}

type manualPrimaryKeyModel struct {
	UnimplementedTable
	ID   string
	Name string
}

func (m *manualPrimaryKeyModel) TableName() string { return "manual_keys" }
func (m *manualPrimaryKeyModel) New() Model        { return new(manualPrimaryKeyModel) }
func (m *manualPrimaryKeyModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	default:
		return nil
	}
}
func (m *manualPrimaryKeyModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		TableName:   m.TableName(),
		PrimaryKeys: []string{"id"},
		Fields: []*FieldDescriptor{
			{DBField: "id", Flag: FlagPrimaryKey},
			{DBField: "name"},
		},
	}
}

func TestFlagFieldsKeepsNonAutoPrimaryKeysOutOfGeneratedKeyPath(t *testing.T) {
	model := &manualPrimaryKeyModel{ID: "manual-1", Name: "alice"}

	autoPrimaryKeys := model.LormModelDescriptor().FlagFields(FlagPrimaryKey | FlagAutoIncrement)
	require.Empty(t, autoPrimaryKeys)

	plan := model.LormBeforeInsert(HookTime{})
	assert.Equal(t, []string{"id", "name"}, plan.Columns)
	assert.False(t, plan.AutoIncrementZero)
}

func TestSelectEscapesModelTableAndPrimaryKeyPredicate(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	_, err := engine.Query[*reservedWordModel]().ID(7).Find(context.Background())
	require.NoError(t, err)
	call := recorder.Last()
	require.NotNil(t, call)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE `id` = ?", call.query)
	assert.Equal(t, []any{int64(7)}, call.args)
}

func TestRepositoryMethodsEscapeIdentifiers(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	repo := engine.Repository[*reservedWordModel]()
	ctx := context.Background()

	rowsAffected, err := repo.UpdateMap(ctx, 1, map[string]any{"group": "updated"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call := recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "UPDATE `order` SET `group` = ? WHERE `id` = ?", call.query)
	assert.Equal(t, []any{"updated", int64(1)}, call.args)

	recorder.Reset()
	rowsAffected, err = repo.DeleteByField(ctx, "group", "updated")
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call = recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "DELETE FROM `order` WHERE `group` = ?", call.query)
	assert.Equal(t, []any{"updated"}, call.args)

	recorder.Reset()
	exists, err := repo.ExistByField(ctx, "group", "updated")
	require.NoError(t, err)
	assert.False(t, exists)
	call = recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT 1 FROM `order` WHERE `group` = ? LIMIT 1", call.query)
	assert.Equal(t, []any{"updated"}, call.args)
}

func TestRepositoryWrapperMethodsWithCaptureEngine(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	repo := engine.Repository[*reservedWordModel]()
	ctx := context.Background()

	rowsAffected, err := repo.Insert(ctx, &reservedWordModel{Group: "first"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call := recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?)", call.query)
	assert.Equal(t, []any{"first"}, call.args)

	recorder.Reset()
	rowsAffected, err = repo.InsertAll(ctx, []*reservedWordModel{{Group: "a"}, {Group: "b"}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call = recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "INSERT INTO `order` (`group`) VALUES (?),(?)", call.query)
	assert.Equal(t, []any{"a", "b"}, call.args)

	recorder.Reset()
	rowsAffected, err = repo.InsertIgnore(ctx, &reservedWordModel{Group: "ignored"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call = recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "INSERT IGNORE INTO `order` (`group`) VALUES (?)", call.query)
	assert.Equal(t, []any{"ignored"}, call.args)

	recorder.Reset()
	rowsAffected, err = repo.InsertIgnoreAll(ctx, []*reservedWordModel{{Group: "x"}, {Group: "y"}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call = recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "INSERT IGNORE INTO `order` (`group`) VALUES (?),(?)", call.query)
	assert.Equal(t, []any{"x", "y"}, call.args)

	recorder.Reset()
	exists, err := repo.Exist(ctx, int64(7))
	require.NoError(t, err)
	assert.False(t, exists)
	call = recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT 1 FROM `order` WHERE `id` = ? LIMIT 1", call.query)
	assert.Equal(t, []any{int64(7)}, call.args)

	recorder.Reset()
	_, err = repo.Lock(ctx, int64(9))
	assert.ErrorContains(t, err, "requires a transaction session")
	assert.Empty(t, recorder.Calls())

	var model *reservedWordModel
	err = engine.TX(ctx, func(txCtx context.Context) error {
		model, err = repo.Lock(txCtx, int64(9))
		return err
	})
	require.NoError(t, err)
	assert.Nil(t, model)
	call = recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE `id` = ? LIMIT 1 FOR UPDATE", call.query)
	assert.Equal(t, []any{int64(9)}, call.args)
}

func TestRepositoryAcceptsNonIntegerPrimaryKeyArguments(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	engine.config.Dialect.Escaper = names.NoEscaper
	repo := engine.Repository[*manualPrimaryKeyModel]()
	ctx := context.Background()

	_, err := repo.Get(ctx, "manual-1")
	require.NoError(t, err)
	call := recorder.Last()
	assert.Equal(t, "query", call.kind)
	assert.Equal(t, "SELECT id, name FROM manual_keys WHERE id = ? LIMIT 1", call.query)
	assert.Equal(t, []any{"manual-1"}, call.args)

	recorder.Reset()
	rowsAffected, err := repo.Delete(ctx, "manual-1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	call = recorder.Last()
	assert.Equal(t, "exec", call.kind)
	assert.Equal(t, "DELETE FROM manual_keys WHERE id = ?", call.query)
	assert.Equal(t, []any{"manual-1"}, call.args)
}

func TestStmtMethodsEscapeIdentifiersWithoutRepositoryHelp(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	_, err := engine.Query[*reservedWordModel]().
		Where(builder.Eq{"group": "g1"}).
		Find(context.Background())
	require.NoError(t, err)
	call := recorder.Last()
	require.NotNil(t, call)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE `group` = ?", call.query)
	assert.Equal(t, []any{"g1"}, call.args)

	sqlStr, args, err := engine.Update[*reservedWordModel]().
		ID(1).
		SetMap(map[string]any{"group": "g2"}).
		builder.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "UPDATE `order` SET `group` = ? WHERE `id` = ?", sqlStr)
	assert.Equal(t, []any{"g2", 1}, args)

	sqlStr, args, err = engine.Delete[*reservedWordModel]().
		Where(builder.Eq{"group": "g3"}).
		builder.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM `order` WHERE `group` = ?", sqlStr)
	assert.Equal(t, []any{"g3"}, args)
}

func TestEngineDoesNotLogSQLArgsByDefault(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	logger := &recordingLogger{}
	engine := newCaptureSQLEngine(t, recorder, false, logger)

	_, err := engine.Exec(context.Background(), "UPDATE test SET secret = ?", "token")
	require.NoError(t, err)
	entry := logger.Last()
	require.NotNil(t, entry)
	assert.Equal(t, "info", entry.level)
	assert.False(t, hasLogKey(entry.args, "args"))
}

func TestEngineLogsSQLArgsWhenEnabled(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	logger := &recordingLogger{}
	engine := newCaptureSQLEngine(t, recorder, true, logger)

	_, err := engine.Exec(context.Background(), "UPDATE test SET secret = ?", "token")
	require.NoError(t, err)
	entry := logger.Last()
	require.NotNil(t, entry)
	assert.Equal(t, "info", entry.level)
	assert.True(t, hasLogKey(entry.args, "args"))
	assert.Equal(t, []any{"token"}, logValue(entry.args, "args"))
}

func TestEngineNilLoggerFastPathExecutesSQLAndTransaction(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, nil)
	ctx := context.Background()

	_, err := engine.Exec(ctx, "UPDATE test SET value = ?", "direct")
	require.NoError(t, err)

	err = engine.TX(ctx, func(txCtx context.Context) error {
		_, execErr := engine.Exec(txCtx, "UPDATE test SET value = ?", "transaction")
		return execErr
	})
	require.NoError(t, err)

	calls := recorder.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, []any{"direct"}, calls[0].args)
	assert.Equal(t, []any{"transaction"}, calls[1].args)
}

type captureSQLRecorder struct {
	mu           sync.Mutex
	calls        []captureSQLCall
	beginTxCalls []driver.TxOptions
}

type captureSQLCall struct {
	kind  string
	query string
	args  []any
}

type countingEscaper struct {
	calls atomic.Int64
}

func (e *countingEscaper) Escape(field string) string {
	e.calls.Add(1)
	return "[" + field + "]"
}

func TestEngineCachesDefaultProjectionByDescriptor(t *testing.T) {
	escaper := new(countingEscaper)
	engine := &Engine{config: &Config{Dialect: DialectConfig{Escaper: escaper}}}
	descriptor := &ModelDescriptor{
		Name: "cachedProjection",
		Fields: []*FieldDescriptor{
			{DBField: "id"},
			{DBField: "name"},
		},
	}

	type projectionResult struct {
		projection *builder.PreparedProjection
		err        error
	}
	const workers = 32
	results := make(chan projectionResult, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			projection, err := engine.defaultSelectProjection(descriptor)
			results <- projectionResult{projection: projection, err: err}
		}()
	}
	wait.Wait()
	close(results)

	var first *builder.PreparedProjection
	for result := range results {
		require.NoError(t, result.err)
		if first == nil {
			first = result.projection
			continue
		}
		assert.Same(t, first, result.projection)
	}
	assert.EqualValues(t, len(descriptor.Fields), escaper.calls.Load())

	sqlText, _, err := new(builder.SelectBuilder).
		SelectPrepared(first).
		From("users").
		ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT [id], [name] FROM users", sqlText)
}

func TestEngineRejectsInvalidDefaultProjectionDescriptors(t *testing.T) {
	engine := &Engine{config: &Config{}}

	_, err := engine.defaultSelectProjection(nil)
	assert.ErrorContains(t, err, "descriptor is nil")

	_, err = engine.defaultSelectProjection(&ModelDescriptor{Name: "empty"})
	assert.ErrorContains(t, err, "has no fields")

	_, err = engine.defaultSelectProjection(&ModelDescriptor{
		Name:   "nilField",
		Fields: []*FieldDescriptor{nil},
	})
	assert.ErrorContains(t, err, "nil field at index 0")

	_, err = engine.defaultSelectProjection(&ModelDescriptor{
		Name:   "emptyField",
		Fields: []*FieldDescriptor{{}},
	})
	assert.ErrorContains(t, err, "empty database field at index 0")
}

func newCaptureSQLRecorder() *captureSQLRecorder {
	return &captureSQLRecorder{}
}

func (r *captureSQLRecorder) Record(kind, query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]any, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	r.calls = append(r.calls, captureSQLCall{kind: kind, query: query, args: values})
}

func (r *captureSQLRecorder) Last() captureSQLCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return captureSQLCall{}
	}
	return r.calls[len(r.calls)-1]
}

func (r *captureSQLRecorder) Calls() []captureSQLCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]captureSQLCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

func (r *captureSQLRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
	r.beginTxCalls = nil
}

func (r *captureSQLRecorder) RecordBeginTx(opts driver.TxOptions) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginTxCalls = append(r.beginTxCalls, opts)
}

func (r *captureSQLRecorder) BeginTxCalls() []driver.TxOptions {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]driver.TxOptions, len(r.beginTxCalls))
	copy(calls, r.beginTxCalls)
	return calls
}

var captureSQLDriverSeq atomic.Uint64

func newCaptureSQLEngine(t *testing.T, recorder *captureSQLRecorder, logArgs bool, logger Logger) *Engine {
	t.Helper()
	driverName := registerCaptureSQLDriver(recorder)
	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	config := &Config{
		driverName: "mysql",
		Dialect: DialectConfig{
			PlaceholderFormat:    builder.Question,
			Escaper:              names.NewQuoter('`', '`'),
			SupportsReturning:    false,
			SupportsLastInsertID: true,
			SupportsForUpdate:    true,
		},
		logger:     logger,
		logSQLArgs: logArgs,
	}
	return &Engine{
		config: config,
		db:     db,
		logger: logger,
	}
}

func registerCaptureSQLDriver(recorder *captureSQLRecorder) string {
	driverName := "lorm_capture_sql_" + strconv.FormatUint(captureSQLDriverSeq.Add(1), 10)
	sql.Register(driverName, &captureSQLDriver{recorder: recorder})
	return driverName
}

type captureSQLDriver struct {
	recorder *captureSQLRecorder
}

func (d *captureSQLDriver) Open(string) (driver.Conn, error) {
	return &captureSQLConn{recorder: d.recorder}, nil
}

type captureSQLConn struct {
	recorder *captureSQLRecorder
}

func (c *captureSQLConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *captureSQLConn) Close() error                        { return nil }
func (c *captureSQLConn) Begin() (driver.Tx, error)           { return captureSQLTx{}, nil }
func (c *captureSQLConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.recorder.RecordBeginTx(opts)
	return captureSQLTx{}, nil
}
func (c *captureSQLConn) Ping(context.Context) error { return nil }

func (c *captureSQLConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.recorder.Record("exec", query, args)
	return captureSQLResult(1), nil
}

func (c *captureSQLConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.recorder.Record("query", query, args)
	return captureSQLRows{}, nil
}

type captureSQLTx struct{}

func (captureSQLTx) Commit() error   { return nil }
func (captureSQLTx) Rollback() error { return nil }

type captureSQLResult int64

func (r captureSQLResult) LastInsertId() (int64, error) { return int64(r), nil }
func (r captureSQLResult) RowsAffected() (int64, error) { return int64(r), nil }

type captureSQLRows struct{}

func (captureSQLRows) Columns() []string         { return []string{"dummy"} }
func (captureSQLRows) Close() error              { return nil }
func (captureSQLRows) Next([]driver.Value) error { return io.EOF }

type logEntry struct {
	level string
	msg   string
	args  []any
}

type recordingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func (l *recordingLogger) DebugContext(_ context.Context, msg string, args ...any) {
	l.record("debug", msg, args...)
}
func (l *recordingLogger) InfoContext(_ context.Context, msg string, args ...any) {
	l.record("info", msg, args...)
}
func (l *recordingLogger) WarnContext(_ context.Context, msg string, args ...any) {
	l.record("warn", msg, args...)
}
func (l *recordingLogger) ErrorContext(_ context.Context, msg string, args ...any) {
	l.record("error", msg, args...)
}

func (l *recordingLogger) record(level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	copied := append([]any(nil), args...)
	l.entries = append(l.entries, logEntry{level: level, msg: msg, args: copied})
}

func (l *recordingLogger) Last() *logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) == 0 {
		return nil
	}
	entry := l.entries[len(l.entries)-1]
	return &entry
}

func hasLogKey(args []any, key string) bool {
	for i := 0; i+1 < len(args); i += 2 {
		if k, ok := args[i].(string); ok && k == key {
			return true
		}
	}
	return false
}

func logValue(args []any, key string) any {
	for i := 0; i+1 < len(args); i += 2 {
		if k, ok := args[i].(string); ok && k == key {
			return args[i+1]
		}
	}
	return nil
}

func TestTXWithOptionsUsesProvidedTransactionOptions(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	err := engine.TXWithOptions(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  true,
	}, func(ctx context.Context) error {
		_, execErr := engine.Exec(ctx, "UPDATE test SET value = ?", "x")
		return execErr
	})
	require.NoError(t, err)

	beginCalls := recorder.BeginTxCalls()
	require.Len(t, beginCalls, 1)
	assert.Equal(t, driver.IsolationLevel(sql.LevelSerializable), beginCalls[0].Isolation)
	assert.True(t, beginCalls[0].ReadOnly)
}

func TestTXWithOptionsReusesExistingSession(t *testing.T) {
	engine := &Engine{config: &Config{}}
	existing := &session{engine: engine}
	ctx := context.WithValue(context.Background(), engine, existing)
	var called bool

	err := engine.TXWithOptions(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context) error {
		called = true
		assert.Same(t, existing, ctx.Value(engine))
		return nil
	})

	assert.True(t, called)
	assert.NoError(t, err)
}
