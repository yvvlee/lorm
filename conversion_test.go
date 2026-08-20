package lorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// csvInts implements driver.Valuer and sql.Scanner to demonstrate
// custom type conversion without lorm-specific interfaces.
type csvInts []int

var _ ScannerValuer = (*csvInts)(nil)

func (c csvInts) Value() (driver.Value, error) {
	if len(c) == 0 {
		return []byte{}, nil
	}
	parts := make([]string, len(c))
	for i, item := range c {
		parts[i] = strconv.Itoa(item)
	}
	return []byte(strings.Join(parts, ",")), nil
}

func (c *csvInts) Scan(src any) error {
	if c == nil {
		return fmt.Errorf("csvInts destination is nil")
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	case nil:
		*c = nil
		return nil
	default:
		return fmt.Errorf("cannot scan %T into csvInts", src)
	}
	if len(data) == 0 {
		*c = nil
		return nil
	}
	parts := strings.Split(string(data), ",")
	result := make([]int, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return err
		}
		result[i] = value
	}
	*c = result
	return nil
}

type conversionModel struct {
	UnimplementedTable
	ID    int64
	Name  string
	Codes csvInts
}

func (*conversionModel) TableName() string { return "conversion_models" }
func (*conversionModel) New() Model        { return new(conversionModel) }

func (m *conversionModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	case "codes":
		return &m.Codes
	default:
		return nil
	}
}

func (*conversionModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:        "conversionModel",
		TableName:   "conversion_models",
		PrimaryKeys: []string{"id"},
		Fields: []*FieldDescriptor{
			{Name: "ID", FullName: "ID", DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement},
			{Name: "Name", FullName: "Name", DBField: "name"},
			{Name: "Codes", FullName: "Codes", DBField: "codes"},
		},
	}
}

func TestCustomConversionIsUsedForInsertAndUpdateArgs(t *testing.T) {
	recorder := newConversionRecorder()
	engine := newConversionTestEngine(t, recorder)
	ctx := context.Background()

	model := &conversionModel{Name: "alpha", Codes: csvInts{1, 2, 3}}
	rowsAffected, err := engine.Insert[*conversionModel]().AddModel(model).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	call := recorder.LastExec()
	require.NotNil(t, call)
	assert.Contains(t, call.query, "INSERT INTO conversion_models")
	require.Len(t, call.args, 2)
	assert.Equal(t, "alpha", call.args[0])
	assert.Equal(t, []byte("1,2,3"), call.args[1])

	recorder.Reset()
	rowsAffected, err = engine.Update[*conversionModel]().
		ID(7).
		SetMap(map[string]any{"codes": csvInts{4, 5, 6}}).
		Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	call = recorder.LastExec()
	require.NotNil(t, call)
	assert.Contains(t, call.query, "UPDATE conversion_models")
	require.Len(t, call.args, 2)
	assert.Equal(t, []byte("4,5,6"), call.args[0])
	assert.Equal(t, int64(7), call.args[1])
}

func TestCustomConversionIsUsedForQueryArgsAndScan(t *testing.T) {
	recorder := newConversionRecorder()
	recorder.SetQueryRows([]string{"id", "name", "codes"}, []driver.Value{int64(9), "alpha", []byte("7,8,9")})
	engine := newConversionTestEngine(t, recorder)

	model, ok, err := engine.Query[*conversionModel]().
		Where("codes = ?", csvInts{1, 2, 3}).
		Get(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, model)
	assert.EqualValues(t, 9, model.ID)
	assert.Equal(t, "alpha", model.Name)
	assert.Equal(t, csvInts{7, 8, 9}, model.Codes)

	call := recorder.LastQuery()
	require.NotNil(t, call)
	assert.Contains(t, call.query, "SELECT id, name, codes FROM conversion_models")
	require.Len(t, call.args, 1)
	assert.Equal(t, []byte("1,2,3"), call.args[0])
}

type conversionRecorder struct {
	mu          sync.Mutex
	execCalls   []conversionCall
	queryCalls  []conversionCall
	queryCols   []string
	queryValues []driver.Value
}

type conversionCall struct {
	query string
	args  []any
}

func newConversionRecorder() *conversionRecorder {
	return &conversionRecorder{}
}

func (r *conversionRecorder) RecordExec(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execCalls = append(r.execCalls, conversionCall{query: query, args: namedValuesToArgs(args)})
}

func (r *conversionRecorder) RecordQuery(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queryCalls = append(r.queryCalls, conversionCall{query: query, args: namedValuesToArgs(args)})
}

func (r *conversionRecorder) SetQueryRows(columns []string, values []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queryCols = append([]string(nil), columns...)
	r.queryValues = append([]driver.Value(nil), values...)
}

func (r *conversionRecorder) QueryRows() ([]string, []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.queryCols...), append([]driver.Value(nil), r.queryValues...)
}

func (r *conversionRecorder) LastExec() *conversionCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.execCalls) == 0 {
		return nil
	}
	call := r.execCalls[len(r.execCalls)-1]
	return &call
}

func (r *conversionRecorder) LastQuery() *conversionCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queryCalls) == 0 {
		return nil
	}
	call := r.queryCalls[len(r.queryCalls)-1]
	return &call
}

func (r *conversionRecorder) QueryCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.queryCalls)
}

func (r *conversionRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execCalls = nil
	r.queryCalls = nil
}

func namedValuesToArgs(args []driver.NamedValue) []any {
	values := make([]any, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return values
}

var conversionDriverSeq atomic.Uint64

func newConversionTestEngine(t *testing.T, recorder *conversionRecorder) *Engine {
	t.Helper()
	driverName := "lorm_conversion_driver_" + strconv.FormatUint(conversionDriverSeq.Add(1), 10)
	sql.Register(driverName, &conversionDriver{recorder: recorder})

	db, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return &Engine{
		config: &Config{
			driverName: "mysql",
			Dialect: DialectConfig{
				PlaceholderFormat:    Placeholder("mysql"),
				Escaper:              Escaper("unknown"),
				SupportsReturning:    false,
				SupportsLastInsertID: true,
				SupportsForUpdate:    true,
			},
			logger: testLogger{},
		},
		db:     db,
		logger: testLogger{},
	}
}

type conversionDriver struct {
	recorder *conversionRecorder
}

func (d *conversionDriver) Open(string) (driver.Conn, error) {
	return &conversionConn{recorder: d.recorder}, nil
}

type conversionConn struct {
	recorder *conversionRecorder
}

func (c *conversionConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *conversionConn) Close() error                        { return nil }
func (c *conversionConn) Begin() (driver.Tx, error)           { return conversionTx{}, nil }

func (c *conversionConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.recorder.RecordExec(query, args)
	return conversionResult(1), nil
}

func (c *conversionConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.recorder.RecordQuery(query, args)
	columns, values := c.recorder.QueryRows()
	return &conversionRows{columns: columns, values: values}, nil
}

type conversionTx struct{}

func (conversionTx) Commit() error   { return nil }
func (conversionTx) Rollback() error { return nil }

type conversionResult int64

func (r conversionResult) LastInsertId() (int64, error) { return int64(r), nil }
func (r conversionResult) RowsAffected() (int64, error) { return int64(r), nil }

type conversionRows struct {
	columns []string
	values  []driver.Value
	read    bool
}

func (r *conversionRows) Columns() []string { return r.columns }
func (r *conversionRows) Close() error      { return nil }
func (r *conversionRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	for i := range dest {
		dest[i] = r.values[i]
	}
	r.read = true
	return nil
}
