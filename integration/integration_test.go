package integration

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

type Test struct {
	UnimplementedTable
	ID         uint64 `lorm:"primary_key,auto_increment"`
	Int        int    `lorm:"index"`
	IntP       *int
	Bool       bool
	BoolP      *bool
	Str        string
	StrP       *string
	Timestamp  time.Time
	TimestampP *time.Time
	Datetime   time.Time
	DatetimeP  *time.Time
	Decimal    decimal.Decimal
	DecimalP   *decimal.Decimal
	IntSlice   []int     `lorm:"json"`
	IntSliceP  *[]int    `lorm:"json"`
	Struct     Sub       `lorm:"json"`
	StructP    *Sub      `lorm:"json"`
	CreatedAt  time.Time `lorm:"created"`
	UpdatedAt  time.Time `lorm:"updated"`
}

type Sub struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (*Test) TableName() string { return "test" }
func (*Test) New() Model        { return new(Test) }

func (m *Test) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "index":
		return &m.Int
	case "int_p":
		return &m.IntP
	case "bool":
		return &m.Bool
	case "bool_p":
		return &m.BoolP
	case "str":
		return &m.Str
	case "str_p":
		return &m.StrP
	case "timestamp":
		return &m.Timestamp
	case "timestamp_p":
		return &m.TimestampP
	case "datetime":
		return &m.Datetime
	case "datetime_p":
		return &m.DatetimeP
	case "decimal":
		return &m.Decimal
	case "decimal_p":
		return &m.DecimalP
	case "int_slice":
		return NewJSONFieldWrapper(&m.IntSlice)
	case "int_slice_p":
		return NewJSONFieldWrapper(&m.IntSliceP)
	case "struct":
		return NewJSONFieldWrapper(&m.Struct)
	case "struct_p":
		return NewJSONFieldWrapper(&m.StructP)
	case "created_at":
		return &m.CreatedAt
	case "updated_at":
		return &m.UpdatedAt
	default:
		return nil
	}
}

func (*Test) LormModelDescriptor() *ModelDescriptor {
	return testModelDescriptor
}

var testModelDescriptor = &ModelDescriptor{
	Name:      "Test",
	TableName: "test",
	Fields: []*FieldDescriptor{
		{Name: "ID", FullName: "ID", DBField: "id", Type: "uint64", Flag: FlagPrimaryKey | FlagAutoIncrement},
		{Name: "Int", FullName: "Int", DBField: "index", Type: "int"},
		{Name: "IntP", FullName: "IntP", DBField: "int_p", Type: "*int"},
		{Name: "Bool", FullName: "Bool", DBField: "bool", Type: "bool"},
		{Name: "BoolP", FullName: "BoolP", DBField: "bool_p", Type: "*bool"},
		{Name: "Str", FullName: "Str", DBField: "str", Type: "string"},
		{Name: "StrP", FullName: "StrP", DBField: "str_p", Type: "*string"},
		{Name: "Timestamp", FullName: "Timestamp", DBField: "timestamp", Type: "time.Time"},
		{Name: "TimestampP", FullName: "TimestampP", DBField: "timestamp_p", Type: "*time.Time"},
		{Name: "Datetime", FullName: "Datetime", DBField: "datetime", Type: "time.Time"},
		{Name: "DatetimeP", FullName: "DatetimeP", DBField: "datetime_p", Type: "*time.Time"},
		{Name: "Decimal", FullName: "Decimal", DBField: "decimal", Type: "decimal.Decimal"},
		{Name: "DecimalP", FullName: "DecimalP", DBField: "decimal_p", Type: "*decimal.Decimal"},
		{Name: "IntSlice", FullName: "IntSlice", DBField: "int_slice", Type: "[]int", Flag: FlagJson},
		{Name: "IntSliceP", FullName: "IntSliceP", DBField: "int_slice_p", Type: "*[]int", Flag: FlagJson},
		{Name: "Struct", FullName: "Struct", DBField: "struct", Type: "Sub", Flag: FlagJson},
		{Name: "StructP", FullName: "StructP", DBField: "struct_p", Type: "*Sub", Flag: FlagJson},
		{Name: "CreatedAt", FullName: "CreatedAt", DBField: "created_at", Type: "time.Time", Flag: FlagCreated},
		{Name: "UpdatedAt", FullName: "UpdatedAt", DBField: "updated_at", Type: "time.Time", Flag: FlagUpdated},
	},
}

func newTestRepository(engine *Engine) *Repository[*Test] {
	return NewRepository[*Test](engine)
}

var (
	sqlite3TestCheckOnce sync.Once
	sqlite3TestCheckErr  error
)

func sqlite3TestAvailable() (bool, string) {
	sqlite3TestCheckOnce.Do(func() {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			sqlite3TestCheckErr = err
			return
		}
		defer db.Close()
		if err = db.Ping(); err != nil {
			sqlite3TestCheckErr = err
			return
		}
		if _, err = db.Exec("SELECT 1"); err != nil {
			sqlite3TestCheckErr = err
		}
	})
	if sqlite3TestCheckErr != nil {
		return false, sqlite3TestCheckErr.Error()
	}
	return true, ""
}

func integrationTestDriverAndDSN(t *testing.T) (string, string) {
	t.Helper()
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")
	if driver != "" && dsn != "" {
		return driver, dsn
	}
	if ok, _ := sqlite3TestAvailable(); ok {
		return "sqlite3", "file:lorm_test.db?cache=shared&mode=memory"
	}
	t.Skipf("integration tests require DB_DRIVER and DB_DSN, or a working sqlite3 test driver")
	return "", ""
}

func mustIntegrationDriverAndDSN(t *testing.T) (string, string) {
	t.Helper()
	driver, dsn := integrationTestDriverAndDSN(t)
	if driver == "" || dsn == "" {
		t.Fatal("missing integration test driver or DSN")
	}
	return driver, dsn
}

func initIntegrationSQL(t *testing.T, driver string) string {
	t.Helper()
	var name string
	switch driver {
	case "postgres", "pgx":
		name = "postgres.sql"
	case "sqlite3", "sqlite":
		name = "sqlite.sql"
	case "mysql", "mariadb":
		name = "mysql.sql"
	default:
		t.Fatalf("unsupported integration test driver %q", driver)
	}
	content, err := os.ReadFile(filepath.Join("..", "testdata", name))
	require.NoError(t, err)
	return string(content)
}

func initEngine(t *testing.T) *Engine {
	t.Helper()
	driver, dsn := mustIntegrationDriverAndDSN(t)

	engine, err := NewEngine(driver, dsn)
	if err != nil {
		t.Fatalf("open test engine (%s): %v", driver, err)
	}
	ctx := context.TODO()
	for _, sql := range strings.Split(initIntegrationSQL(t, engine.DriverName()), ";") {
		sql = strings.TrimSpace(sql)
		if sql == "" {
			continue
		}
		_, err := engine.Exec(ctx, sql)
		require.NoError(t, err)
	}
	return engine
}

func TestEngine(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()
	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	models := []*Test{
		{
			Int:        1,
			Bool:       true,
			Str:        "a",
			Timestamp:  testTime,
			Datetime:   testTime,
			Decimal:    decimal.NewFromFloat(1.10),
			IntSlice:   []int{1, 2, 3},
			Struct:     Sub{ID: 1, Name: "haha"},
			IntP:       nil,
			BoolP:      nil,
			StrP:       nil,
			TimestampP: nil,
			DatetimeP:  nil,
			DecimalP:   nil,
			IntSliceP:  nil,
			StructP:    nil,
		},
		{
			Int:        2,
			IntP:       lo.ToPtr(2),
			Bool:       false,
			BoolP:      lo.ToPtr(true),
			Str:        "b",
			StrP:       lo.ToPtr("bb"),
			Timestamp:  testTime,
			TimestampP: &testTime,
			Datetime:   testTime,
			DatetimeP:  &testTime,
			Decimal:    decimal.NewFromFloat(2.12),
			DecimalP:   lo.ToPtr(decimal.NewFromFloat(2.13)),
			IntSlice:   []int{1, 2, 3},
			IntSliceP:  &[]int{11, 2, 3},
			Struct:     Sub{ID: 1, Name: "haha"},
			StructP:    &Sub{ID: 1, Name: "haha"},
		},
	}
	repo := newTestRepository(engine)
	rowsAffected, err := repo.InsertAll(ctx, models)
	require.NoError(t, err)
	assert.Equal(t, int64(2), rowsAffected)

	err = engine.TX(ctx, func(ctx context.Context) error {
		for _, model := range models {
			rowsAffected, err = repo.Insert(ctx, model)
			require.NoError(t, err)
			assert.Equal(t, int64(1), rowsAffected)
			assert.True(t, model.ID > 0)
		}
		return nil
	})
	require.NoError(t, err)

	model, err := repo.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, uint64(1), model.ID)

	model, err = repo.GetByField(ctx, "str", "a")
	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "a", model.Str)

	exist, err := repo.Exist(ctx, 1)
	require.NoError(t, err)
	assert.True(t, exist)

	exist, err = repo.Exist(ctx, 999)
	require.NoError(t, err)
	assert.False(t, exist)

	exist, err = repo.ExistByField(ctx, "str", "a")
	require.NoError(t, err)
	assert.True(t, exist)

	exist, err = repo.ExistByField(ctx, "str", "nonexistent")
	require.NoError(t, err)
	assert.False(t, exist)

	model, err = repo.Get(ctx, 1)
	require.NoError(t, err)
	model.Str = "updated_a"
	rowsAffected, err = repo.Update(ctx, model)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	updatedModel, err := repo.Get(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "updated_a", updatedModel.Str)

	rowsAffected, err = repo.UpdateMap(ctx, 1, map[string]any{"str": "updated_by_map"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	updatedModel, err = repo.Get(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "updated_by_map", updatedModel.Str)

	rowsAffected, err = repo.Delete(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	exist, err = repo.Exist(ctx, 3)
	require.NoError(t, err)
	assert.False(t, exist)

	testModel := &Test{
		Int:       999,
		Str:       "to_be_deleted",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(9.99),
		IntSlice:  []int{9, 9, 9},
		Struct:    Sub{ID: 9, Name: "delete_test"},
	}
	_, err = repo.Insert(ctx, testModel)
	require.NoError(t, err)

	rowsAffected, err = repo.DeleteByField(ctx, "str", "to_be_deleted")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	exist, err = repo.ExistByField(ctx, "str", "to_be_deleted")
	require.NoError(t, err)
	assert.False(t, exist)

	if !engine.SupportsForUpdate() {
		_, err = repo.Lock(ctx, 1)
		assert.ErrorContains(t, err, "does not support FOR UPDATE")
	} else {
		model, err = repo.Lock(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.Equal(t, uint64(1), model.ID)
	}

	if !engine.SupportsForUpdate() {
		_, err = repo.LockByField(ctx, "str", "updated_by_map")
		assert.ErrorContains(t, err, "does not support FOR UPDATE")
	} else {
		model, err = repo.LockByField(ctx, "str", "updated_by_map")
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.Equal(t, "updated_by_map", model.Str)
	}

	list, err := Query[*Test](engine).
		Where("id < ?", 3).
		Find(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	single, err := Query[*Test](engine).
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, single)
	assert.Equal(t, uint64(1), single.ID)

	ids, err := QueryCol[uint64](engine).
		Select("id").
		From("test").
		Where("id < ?", 3).
		OrderBy("id").
		Find(ctx)
	require.NoError(t, err)
	assert.Equal(t, []uint64{1, 2}, ids)

	id, ok, err := QueryCol[uint64](engine).
		Select("id").
		From("test").
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(1), id)

	res, err := Query[*Test](engine).Where("id = ?", -1).Limit(1).Get(ctx)
	require.NoError(t, err)
	assert.Nil(t, res)

	list, err = Query[*Test](engine).Where("id = ?", -1).Find(ctx)
	require.NoError(t, err)
	assert.Nil(t, list)

	list, err = Query[*Test](engine).Where("id > ?", 0).Find(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, list)

	m := &Test{
		Int:       100,
		Str:       "single",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(1.23),
		IntSlice:  []int{1},
		Struct:    Sub{ID: 1, Name: "x"},
	}
	rowsAffected, err = Insert[*Test](engine).AddModel(m).Exec(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	original := &Test{
		Int:       701,
		Str:       "repo_insert_ignore_original",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(7.01),
		IntSlice:  []int{7, 0, 1},
		Struct:    Sub{ID: 701, Name: "original"},
	}
	rowsAffected, err = repo.Insert(ctx, original)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	ignored := &Test{
		ID:        original.ID,
		Int:       702,
		Str:       "repo_insert_ignore_single",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(7.02),
		IntSlice:  []int{7, 0, 2},
		Struct:    Sub{ID: 702, Name: "ignored"},
	}
	_, err = repo.InsertIgnore(ctx, ignored)
	require.NoError(t, err)

	current, err := repo.Get(ctx, original.ID)
	require.NoError(t, err)
	assert.Equal(t, "repo_insert_ignore_original", current.Str)

	batch := []*Test{
		{
			ID:        original.ID,
			Int:       703,
			Str:       "repo_insert_ignore_all_duplicate",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(7.03),
			IntSlice:  []int{7, 0, 3},
			Struct:    Sub{ID: 703, Name: "duplicate"},
		},
		{
			Int:       704,
			Str:       "repo_insert_ignore_all_inserted",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(7.04),
			IntSlice:  []int{7, 0, 4},
			Struct:    Sub{ID: 704, Name: "inserted"},
		},
	}
	_, err = repo.InsertIgnoreAll(ctx, batch)
	require.NoError(t, err)

	inserted, err := repo.GetByField(ctx, "str", "repo_insert_ignore_all_inserted")
	require.NoError(t, err)
	require.NotNil(t, inserted)
	assert.NotZero(t, inserted.ID)
}

func TestInsertSingle(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	m := &Test{
		Int:       777,
		Str:       "single_insert",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(3.21),
		IntSlice:  []int{7},
		Struct:    Sub{ID: 7, Name: "s"},
	}
	rows, err := Insert[*Test](engine).AddModel(m).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m.ID > 0)
}

func TestInsertIgnore(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	m1 := &Test{
		Int:       888,
		Str:       "insert_ignore_test",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(8.88),
		IntSlice:  []int{8},
		Struct:    Sub{ID: 8, Name: "s8"},
	}
	rows, err := Insert[*Test](engine).AddModel(m1).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m1.ID > 0)

	m2 := &Test{
		ID:        m1.ID,
		Int:       999,
		Str:       "should_be_ignored",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(9.99),
		IntSlice:  []int{9},
		Struct:    Sub{ID: 9, Name: "s9"},
	}
	_, err = Insert[*Test](engine).Ignore().AddModel(m2).Exec(ctx)
	require.NoError(t, err)

	result, err := Query[*Test](engine).Where("id = ?", m1.ID).Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 888, result.Int)
	assert.Equal(t, "insert_ignore_test", result.Str)
}

func TestInsertIgnoreAll(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	m1 := &Test{
		Int:       111,
		Str:       "batch_insert_1",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(1.11),
		IntSlice:  []int{1},
		Struct:    Sub{ID: 1, Name: "s1"},
	}
	_, err := Insert[*Test](engine).AddModel(m1).Exec(ctx)
	require.NoError(t, err)

	models := []*Test{
		{
			ID:        m1.ID,
			Int:       222,
			Str:       "should_be_ignored",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(2.22),
			IntSlice:  []int{2},
			Struct:    Sub{ID: 2, Name: "s2"},
		},
		{
			Int:       333,
			Str:       "batch_insert_2",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(3.33),
			IntSlice:  []int{3},
			Struct:    Sub{ID: 3, Name: "s3"},
		},
	}
	_, err = Insert[*Test](engine).Ignore().AddModels(models...).Exec(ctx)
	require.NoError(t, err)

	result, err := Query[*Test](engine).Where("id = ?", m1.ID).Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 111, result.Int)
	assert.Equal(t, "batch_insert_1", result.Str)
}

func TestInsertIgnoreReturningRowsAffected(t *testing.T) {
	engine := initReturningCompatibleEngine(t)
	defer engine.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	existing := &Test{
		Int:       444,
		Str:       "returning_duplicate_key",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(4.44),
		IntSlice:  []int{4},
		Struct:    Sub{ID: 4, Name: "existing"},
	}
	rows, err := Insert[*Test](engine).AddModel(existing).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, existing.ID > 0)

	models := []*Test{
		{
			Int:       555,
			Str:       existing.Str,
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(5.55),
			IntSlice:  []int{5},
			Struct:    Sub{ID: 5, Name: "conflict"},
		},
		{
			Int:       666,
			Str:       "returning_inserted",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(6.66),
			IntSlice:  []int{6},
			Struct:    Sub{ID: 6, Name: "inserted"},
		},
	}
	rows, err = Insert[*Test](engine).Ignore().AddModels(models...).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.Zero(t, models[0].ID)
	assert.Zero(t, models[1].ID)

	inserted, err := Query[*Test](engine).Where("str = ?", "returning_inserted").Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, inserted)
	assert.True(t, inserted.ID > 0)
}

func TestInsertAllWithoutReturningDoesNotBackfillGeneratedIDs(t *testing.T) {
	engine := initEngine(t)
	if engine.SupportsReturning() {
		t.Skip("non-returning behavior only applies to LastInsertId-style dialects")
	}
	defer engine.Close()

	ctx := context.TODO()
	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	models := []*Test{
		{
			Int:       501,
			Str:       "batch_no_returning_1",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(5.01),
			IntSlice:  []int{5, 1},
			Struct:    Sub{ID: 51, Name: "b1"},
		},
		{
			Int:       502,
			Str:       "batch_no_returning_2",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(5.02),
			IntSlice:  []int{5, 2},
			Struct:    Sub{ID: 52, Name: "b2"},
		},
	}
	rows, err := Insert[*Test](engine).AddModels(models...).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, rows)
	assert.Zero(t, models[0].ID)
	assert.Zero(t, models[1].ID)
}

func TestDeleteExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Delete[*Test](e).Prefix("INVALID").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestDeleteByID(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	_, err := Insert[*Test](e).AddModel(&Test{
		Int:       42,
		Bool:      true,
		Str:       "typed_delete_target",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(4.20),
		IntSlice:  []int{4, 2},
		Struct:    Sub{ID: 42, Name: "typed"},
	}).Exec(ctx)
	require.NoError(t, err)

	target, err := Query[*Test](e).Where("str = ?", "typed_delete_target").Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, target)

	rowsAffected, err := Delete[*Test](e).ID(target.ID).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)

	deleted, err := Query[*Test](e).Where("id = ?", target.ID).Get(ctx)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestUpdateExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Update[*Test](e).Prefix("INVALID").Set("str", "x").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestQueryModelPageBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()
	insertBasicRows(t, e, ctx)

	_, _, err := Query[*Test](e).Page(ctx, 1, 0)
	assert.Error(t, err)

	list, total, err := Query[*Test](e).Where("id < ?", 0).Page(ctx, 1, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Nil(t, list)

	_, total, err = Query[*Test](e).Where("id > ?", 0).Page(ctx, 100, 10)
	require.NoError(t, err)
	assert.True(t, total > 0)

	list, total, err = Query[*Test](e).Where("id > ?", 0).Page(ctx, 1, 1)
	require.NoError(t, err)
	assert.True(t, total > 0)
	assert.LessOrEqual(t, len(list), 1)
}

func TestQueryModelExistBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()
	insertBasicRows(t, e, ctx)

	ex, err := Query[*Test](e).Where("id > ?", 0).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, ex)

	ex, err = Query[*Test](e).Where("id < ?", 0).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, ex)
}

func TestQueryColGetFalseAndError(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, ok, err := QueryCol[uint64](e).
		Select("id").
		From("test").
		Where("id < ?", 0).
		Limit(1).
		Get(ctx)
	require.NoError(t, err)
	assert.False(t, ok)

	_, _, err = QueryCol[uint64](e).Prefix("INVALID").
		Select("id").
		From("test").
		Limit(1).
		Get(ctx)
	assert.Error(t, err)
}

func TestQueryModelGetIgnoresExtraColumns(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	model := &Test{
		Int:       7,
		Bool:      true,
		Str:       "with_extra_column",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(7.77),
		IntSlice:  []int{7, 7, 7},
		Struct:    Sub{ID: 7, Name: "extra"},
	}
	_, err := Insert[*Test](e).AddModel(model).Exec(ctx)
	require.NoError(t, err)

	got, err := Query[*Test](e).
		Where("id = ?", model.ID).
		AddColumn("1 AS extra_value").
		Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.ID, got.ID)
	assert.Equal(t, model.Str, got.Str)
}

func TestQueryModelFindErrorBranch(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	_, err := Query[*Test](engine).Prefix("INVALID").Find(context.TODO())
	assert.Error(t, err)
}

func TestQueryModelGetErrorBranch(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Query[*Test](e).Prefix("INVALID /*error*/").Limit(1).Get(context.TODO())
	assert.Error(t, err)
}

func TestExecQueryExistErrorLogging(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	_, err := e.Exec(ctx, "INVALID SQL")
	assert.Error(t, err)

	_, err = e.Query(ctx, "INVALID SQL")
	assert.Error(t, err)

	_, err = e.Exist(ctx, "INVALID SQL")
	assert.Error(t, err)
}

func TestTXErrorBranch(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	err := e.TX(context.TODO(), func(ctx context.Context) error { return assert.AnError })
	assert.Error(t, err)
}

func TestTXPanicRollsBack(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()

	assert.PanicsWithValue(t, "boom", func() {
		_ = engine.TX(context.Background(), func(context.Context) error {
			panic("boom")
		})
	})
}

func TestUpdateSetModelRefreshesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	engine := newSQLiteSemanticsEngine(t)
	defer engine.Close()

	model := &updateSemanticsModel{Name: "draft", Version: 1}
	_, err := Insert[*updateSemanticsModel](engine).AddModel(model).Exec(ctx)
	require.NoError(t, err)

	repo := NewRepository[*updateSemanticsModel](engine)
	loaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	before := loaded.UpdatedAt

	time.Sleep(20 * time.Millisecond)

	loaded.Name = "published"
	rowsAffected, err := Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.True(t, loaded.UpdatedAt.After(before), "expected in-memory updated_at to advance")

	reloaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.True(t, reloaded.UpdatedAt.After(before), "expected persisted updated_at to advance")
}

func TestUpdateSetModelSyncsVersionBackToModel(t *testing.T) {
	ctx := context.Background()
	engine := newSQLiteSemanticsEngine(t)
	defer engine.Close()

	model := &updateSemanticsModel{Name: "draft", Version: 1}
	_, err := Insert[*updateSemanticsModel](engine).AddModel(model).Exec(ctx)
	require.NoError(t, err)

	repo := NewRepository[*updateSemanticsModel](engine)
	loaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), loaded.Version)

	loaded.Name = "published"
	rowsAffected, err := Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.Equal(t, int64(2), loaded.Version, "expected in-memory version to increment after successful update")

	loaded.Name = "published-again"
	rowsAffected, err = Update[*updateSemanticsModel](engine).SetModel(loaded).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	require.Equal(t, int64(3), loaded.Version)

	reloaded, err := repo.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), reloaded.Version)
}

func insertBasicRows(t *testing.T, e *Engine, ctx context.Context) {
	t.Helper()
	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	models := []*Test{
		{
			Int:       1,
			Bool:      true,
			Str:       "a",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(1.10),
			IntSlice:  []int{1, 2, 3},
			Struct:    Sub{ID: 1, Name: "haha"},
		},
		{
			Int:       2,
			Bool:      false,
			Str:       "b",
			Timestamp: testTime,
			Datetime:  testTime,
			Decimal:   decimal.NewFromFloat(2.12),
			IntSlice:  []int{1, 2, 3},
			Struct:    Sub{ID: 1, Name: "haha"},
		},
	}
	_, err := newTestRepository(e).InsertAll(ctx, models)
	require.NoError(t, err)
}

func TestCountBuilderIncludesNullGroup(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.Background()
	insertBasicRows(t, engine, ctx)

	countBuilder := builder.Select("str_p").From("test").GroupBy("str_p").ToCountBuilder()
	query, args, err := countBuilder.ToSql()
	require.NoError(t, err)
	rows, err := engine.Query(ctx, query, args...)
	require.NoError(t, err)
	defer rows.Close()

	var count uint64
	require.NoError(t, ScanCol(rows, &count))
	assert.EqualValues(t, 1, count)
}

func initReturningCompatibleEngine(t *testing.T) *Engine {
	t.Helper()
	if ok, reason := sqlite3TestAvailable(); !ok {
		t.Skipf("sqlite3 integration dependency unavailable: %s", reason)
	}
	engine, err := NewEngine(
		"sqlite3",
		"file:lorm_returning_test.db?cache=shared&mode=memory",
		WithPlaceholderFormat(builder.Dollar),
		WithEscaper(names.NewQuoter('"', '"')),
		WithSupportsReturning(true),
		WithSupportsLastInsertID(false),
		WithSupportsForUpdate(true),
	)
	require.NoError(t, err)

	ctx := context.TODO()
	for _, sql := range strings.Split(initIntegrationSQL(t, "sqlite3"), ";") {
		sql = strings.TrimSpace(sql)
		if sql == "" {
			continue
		}
		_, err = engine.Exec(ctx, sql)
		require.NoError(t, err)
	}
	_, err = engine.Exec(ctx, "CREATE UNIQUE INDEX idx_test_str_unique ON test(str)")
	require.NoError(t, err)
	return engine
}

type updateSemanticsModel struct {
	UnimplementedTable
	ID        int64
	Name      string
	Version   int64
	UpdatedAt time.Time
}

func (*updateSemanticsModel) TableName() string { return "update_semantics_models" }
func (*updateSemanticsModel) New() Model        { return new(updateSemanticsModel) }

func (m *updateSemanticsModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "name":
		return &m.Name
	case "version":
		return &m.Version
	case "updated_at":
		return &m.UpdatedAt
	default:
		return nil
	}
}

func (m *updateSemanticsModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "updateSemanticsModel",
		TableName: m.TableName(),
		Fields: []*FieldDescriptor{
			{Name: "ID", FullName: "ID", DBField: "id", Flag: FlagPrimaryKey | FlagAutoIncrement},
			{Name: "Name", FullName: "Name", DBField: "name"},
			{Name: "Version", FullName: "Version", DBField: "version", Flag: FlagVersion},
			{Name: "UpdatedAt", FullName: "UpdatedAt", DBField: "updated_at", Flag: FlagUpdated},
		},
	}
}

func newSQLiteSemanticsEngine(t *testing.T) *Engine {
	t.Helper()
	if ok, reason := sqlite3TestAvailable(); !ok {
		t.Skipf("sqlite3 integration dependency unavailable: %s", reason)
	}
	engine, err := NewEngine(
		"sqlite3",
		filepath.Join(t.TempDir(), "semantics.sqlite"),
		WithMaxOpenConns(1),
	)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = engine.Exec(ctx, `CREATE TABLE update_semantics_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version INTEGER NOT NULL,
		updated_at DATETIME NOT NULL
	)`)
	require.NoError(t, err)
	return engine
}
