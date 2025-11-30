package lorm

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	//nolint:revive
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed testdata/mysql.sql
var mysqlInitSQL string

//go:embed testdata/postgres.sql
var postgresInitSQL string

//go:embed testdata/sqlite.sql
var sqliteInitSQL string

func TestEngine(t *testing.T) {
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")

	// 如果没有设置环境变量，则默认使用MySQL进行测试
	if driver == "" || dsn == "" {
		driver = "mysql"
		dsn = "root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4&collation=utf8mb4_general_ci&loc=Local"
	}

	engine, err := NewEngine(driver, dsn)
	assert.Nil(t, err)
	defer engine.Close()
	testEngine(t, engine)
}

func testEngine(t *testing.T, engine *Engine) {
	ctx := context.TODO()

	var initSQL string
	switch engine.config.driverName {
	case "postgres":
		initSQL = postgresInitSQL
	case "sqlite3":
		initSQL = sqliteInitSQL
	default:
		initSQL = mysqlInitSQL
	}

	for _, sql := range strings.Split(initSQL, ";") {
		if sql == "" {
			continue
		}
		_, err := engine.Exec(ctx, sql)
		assert.Nil(t, err)
	}

	testTime, _ := time.ParseInLocation(time.DateTime, "2025-01-23 16:17:18", time.Local)
	models := []*Test{
		{
			Int:        1,
			IntP:       nil,
			Bool:       true,
			BoolP:      nil,
			Str:        "a",
			StrP:       nil,
			Timestamp:  testTime,
			TimestampP: nil,
			Datetime:   testTime,
			DatetimeP:  nil,
			Decimal:    decimal.NewFromFloat(1.10),
			DecimalP:   nil,
			IntSlice:   []int{1, 2, 3},
			IntSliceP:  nil,
			Struct:     Sub{ID: 1, Name: "haha"},
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
	repo := NewTestRepository(engine)
	rowsAffected, err := repo.InsertAll(ctx, models)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), rowsAffected)

	err = engine.TX(ctx, func(ctx context.Context) error {
		for _, model := range models {
			rowsAffected, err = repo.Insert(ctx, model)
			assert.Nil(t, err)
			assert.Equal(t, int64(1), rowsAffected)
			assert.True(t, model.ID > 0)
		}
		return nil
	})
	assert.Nil(t, err)

	// Test Get method
	t.Run("Get", func(t *testing.T) {
		model, err := repo.Get(ctx, 1)
		assert.Nil(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, uint64(1), model.ID)
	})

	// Test GetByField method
	t.Run("GetByField", func(t *testing.T) {
		model, err := repo.GetByField(ctx, "str", "a")
		assert.Nil(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "a", model.Str)
	})

	// Test Exist method
	t.Run("Exist", func(t *testing.T) {
		exist, err := repo.Exist(ctx, 1)
		assert.Nil(t, err)
		assert.True(t, exist)

		exist, err = repo.Exist(ctx, 999)
		assert.Nil(t, err)
		assert.False(t, exist)
	})

	// Test ExistByField method
	t.Run("ExistByField", func(t *testing.T) {
		exist, err := repo.ExistByField(ctx, "str", "a")
		assert.Nil(t, err)
		assert.True(t, exist)

		exist, err = repo.ExistByField(ctx, "str", "nonexistent")
		assert.Nil(t, err)
		assert.False(t, exist)
	})

	// Test Update method
	t.Run("Update", func(t *testing.T) {
		model, err := repo.Get(ctx, 1)
		assert.Nil(t, err)
		model.Str = "updated_a"
		rowsAffected, err := repo.Update(ctx, model)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		// Verify update success
		updatedModel, err := repo.Get(ctx, 1)
		assert.Nil(t, err)
		assert.Equal(t, "updated_a", updatedModel.Str)
	})

	// Test UpdateMap method
	t.Run("UpdateMap", func(t *testing.T) {
		data := map[string]any{
			"str": "updated_by_map",
		}
		rowsAffected, err := repo.UpdateMap(ctx, 1, data)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		// Verify update success
		updatedModel, err := repo.Get(ctx, 1)
		assert.Nil(t, err)
		assert.Equal(t, "updated_by_map", updatedModel.Str)
	})

	// Test Delete method
	t.Run("Delete", func(t *testing.T) {
		rowsAffected, err := repo.Delete(ctx, 3)
		assert.Nil(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		// Verify deletion success
		exist, err := repo.Exist(ctx, 3)
		assert.Nil(t, err)
		assert.False(t, exist)
	})

	// Test DeleteByField method
	t.Run("DeleteByField", func(t *testing.T) {
		// Insert a test record first
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
		assert.Nil(t, err)

		rowsAffected, err := repo.DeleteByField(ctx, "str", "to_be_deleted")
		assert.Nil(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		// Verify deletion success
		exist, err := repo.ExistByField(ctx, "str", "to_be_deleted")
		assert.Nil(t, err)
		assert.False(t, exist)
	})

	// Test Lock method
	t.Run("Lock", func(t *testing.T) {
		model, err := repo.Lock(ctx, 1)
		assert.Nil(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, uint64(1), model.ID)
	})

	// Test LockByField method
	t.Run("LockByField", func(t *testing.T) {
		model, err := repo.LockByField(ctx, "str", "updated_by_map")
		assert.Nil(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "updated_by_map", model.Str)
	})

	list, err := Query[*Test](engine).
		Where("id < ?", 3).
		Find(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(list))
	single, err := Query[*Test](engine).
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, single)
	assert.Equal(t, single.ID, uint64(1))
	ids, err := QueryCol[uint64](engine).
		From("test").
		Columns("id").
		Where("id < ?", 3).
		Find(ctx)
	assert.Nil(t, err)
	assert.Equal(t, ids, []uint64{1, 2})
	id, exist, err := QueryCol[uint64](engine).
		From("test").
		Columns("id").
		Where("id < ?", 2).
		Limit(1).
		Get(ctx)
	assert.Nil(t, err)
	assert.True(t, exist)
	assert.Equal(t, id, uint64(1))

	// QueryModelStmt.Get 空结果
	t.Run("QueryModel Get empty", func(t *testing.T) {
		res, err := Query[*Test](engine).Where("id = ?", -1).Limit(1).Get(ctx)
		assert.Nil(t, err)
		assert.Nil(t, res)
	})

	// QueryModelStmt.Find 空/非空
	t.Run("QueryModel Find variants", func(t *testing.T) {
		list, err := Query[*Test](engine).Where("id = ?", -1).Find(ctx)
		assert.Nil(t, err)
		assert.Nil(t, list)
		list, err = Query[*Test](engine).Where("id > ?", 0).Find(ctx)
		assert.Nil(t, err)
		assert.True(t, len(list) > 0)
	})

	// InsertAll 单元素分支
	t.Run("InsertAll single branch", func(t *testing.T) {
		m := &Test{Int: 100, Str: "single", Timestamp: testTime, Datetime: testTime, Decimal: decimal.NewFromFloat(1.23), IntSlice: []int{1}, Struct: Sub{ID: 1, Name: "x"}}
		rows, err := InsertAll(ctx, engine, []*Test{m})
		assert.Nil(t, err)
		assert.Equal(t, int64(1), rows)
	})
}
