package lorm

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

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
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m.ID > 0)
}

func TestInsertAllEmpty(t *testing.T) {
	var models []*Test
	rows, err := Insert[*Test](&Engine{config: &Config{}}).AddModels(models...).Exec(context.TODO())
	assert.NoError(t, err)
	assert.EqualValues(t, 0, rows)
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
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)
	assert.True(t, m1.ID > 0)

	originalID := m1.ID

	m2 := &Test{
		ID:        originalID,
		Int:       999,
		Str:       "should_be_ignored",
		Timestamp: testTime,
		Datetime:  testTime,
		Decimal:   decimal.NewFromFloat(9.99),
		IntSlice:  []int{9},
		Struct:    Sub{ID: 9, Name: "s9"},
	}

	rows, err = Insert[*Test](engine).Ignore().AddModel(m2).Exec(ctx)
	assert.Nil(t, err)

	result, err := Query[*Test](engine).Where("id = ?", originalID).Get(ctx)
	assert.Nil(t, err)
	assert.Equal(t, int(888), result.Int)
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

	rows, err := Insert[*Test](engine).AddModel(m1).Exec(ctx)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, rows)

	originalID := m1.ID

	models := []*Test{
		{
			ID:        originalID,
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

	rows, err = Insert[*Test](engine).Ignore().AddModels(models...).Exec(ctx)
	assert.Nil(t, err)

	result, err := Query[*Test](engine).Where("id = ?", originalID).Get(ctx)
	assert.Nil(t, err)
	assert.Equal(t, int(111), result.Int)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.Zero(t, models[0].ID)
	assert.Zero(t, models[1].ID)

	inserted, err := Query[*Test](engine).Where("str = ?", "returning_inserted").Get(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, inserted)
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
	assert.NoError(t, err)
	assert.EqualValues(t, 2, rows)
	assert.Zero(t, models[0].ID)
	assert.Zero(t, models[1].ID)
}

func TestInsertStmtBuilderWrappers(t *testing.T) {
	stmt := Insert[*Test](&Engine{
		config: &Config{
			placeholderFormat: builder.Question,
			escaper:           names.NoEscaper,
		},
	}).
		Prefix("WITH audit AS ?", 0).
		PrefixExpr(builder.Expr("/* insert */")).
		Columns("id", "str").
		Values(1, "wrapped").
		Suffix("ON CONFLICT DO NOTHING").
		SuffixExpr(builder.Expr("RETURNING id"))

	sql, args, err := stmt.builder.ToSql()
	assert.NoError(t, err)
	assert.Contains(t, sql, "WITH audit AS ?")
	assert.Contains(t, sql, "INSERT INTO test (id,str) VALUES (?,?)")
	assert.Contains(t, sql, "ON CONFLICT DO NOTHING")
	assert.Contains(t, sql, "RETURNING id")
	assert.Equal(t, []any{0, 1, "wrapped"}, args)
}

func TestInsertExecWithReturningCoverage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(41)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		model := &conversionModel{Name: "alpha", Codes: csvInts{1, 2}}
		rowsAffected, err := Insert[*conversionModel](engine).AddModel(model).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
		assert.EqualValues(t, 41, model.ID)

		call := recorder.LastQuery()
		require.NotNil(t, call)
		assert.Contains(t, call.query, "RETURNING id")
	})

	t.Run("mismatch", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := Insert[*conversionModel](engine).AddModels(models...).Exec(context.Background())
		assert.ErrorContains(t, err, "expected 2 returned rows, got 1")
		assert.Zero(t, rowsAffected)
	})

	t.Run("ignoreAllowsPartialRows", func(t *testing.T) {
		recorder := newConversionRecorder()
		recorder.SetQueryRows([]string{"id"}, []driver.Value{int64(1)})
		engine := newConversionTestEngine(t, recorder)
		engine.config.driverName = "postgres"
		engine.config.supportsReturning = true
		engine.config.supportsLastInsertID = false

		models := []*conversionModel{
			{Name: "a", Codes: csvInts{1}},
			{Name: "b", Codes: csvInts{2}},
		}
		rowsAffected, err := Insert[*conversionModel](engine).Ignore().AddModels(models...).Exec(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, 1, rowsAffected)
	})
}

func initReturningCompatibleEngine(t *testing.T) *Engine {
	t.Helper()
	skipUnlessSQLite3Available(t)

	engine, err := NewEngine(
		"sqlite3",
		"file:lorm_returning_test.db?cache=shared&mode=memory",
		WithPlaceholderFormat(builder.Dollar),
		WithEscaper(names.NewQuoter('"', '"')),
	)
	assert.NoError(t, err)
	if err != nil {
		return nil
	}

	ctx := context.TODO()
	for _, sql := range strings.Split(sqliteInitSQL, ";") {
		sql = strings.TrimSpace(sql)
		if sql == "" {
			continue
		}
		_, err = engine.Exec(ctx, sql)
		assert.NoError(t, err)
	}
	_, err = engine.Exec(ctx, "CREATE UNIQUE INDEX idx_test_str_unique ON test(str)")
	assert.NoError(t, err)

	engine.config.driverName = "postgres"
	engine.config.supportsReturning = true
	engine.config.supportsLastInsertID = false
	engine.config.supportsForUpdate = true
	return engine
}
