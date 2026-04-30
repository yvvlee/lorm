package lorm

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestQueryModelExistBranches(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	ctx := context.TODO()

	// Prepare test data
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
	repo := NewTestRepository(e)
	_, err := repo.InsertAll(ctx, models)
	assert.Nil(t, err)

	ex, err := Query[*Test](e).Where("id > ?", 0).Exist(ctx)
	assert.Nil(t, err)
	assert.True(t, ex)

	ex, err = Query[*Test](e).Where("id < ?", 0).Exist(ctx)
	assert.Nil(t, err)
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
	assert.Nil(t, err)
	assert.False(t, ok)

	_, _, err = QueryCol[uint64](e).Prefix("INVALID").
		Select("id").
		From("test").
		Limit(1).
		Get(ctx)
	assert.Error(t, err)
}

func TestQueryColGetSuccessCoverage(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(9)})
	engine := newScriptedEngine(t, recorder)

	value, ok, err := QueryCol[int64](engine).
		Select("id").
		From("conversion_models").
		Where("name = ?", "alpha").
		Limit(1).
		Get(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, 9, value)

	call := recorder.LastQuery()
	require.NotNil(t, call)
	assert.Equal(t, "SELECT id FROM conversion_models WHERE name = ? LIMIT 1", call.query)
	assert.Equal(t, []any{"alpha"}, call.args)
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
	assert.NoError(t, err)

	got, err := Query[*Test](e).
		Where("id = ?", model.ID).
		AddColumn("1 AS extra_value").
		Get(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, model.ID, got.ID)
	assert.Equal(t, model.Str, got.Str)
}

func TestQueryModelGetAndFindCoverage(t *testing.T) {
	t.Run("getSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(3), "ops"})
		engine := newScriptedEngine(t, recorder)

		model, err := Query[*reservedWordModel](engine).
			Where(builder.Eq{"group": "ops"}).
			Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.EqualValues(t, 3, model.ID)
		assert.Equal(t, "ops", model.Group)
	})

	t.Run("findSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows(
			[]string{"id", "group"},
			[]driver.Value{int64(1), "a"},
			[]driver.Value{int64(2), "b"},
		)
		engine := newScriptedEngine(t, recorder)

		list, err := Query[*reservedWordModel](engine).
			Where("id > ?", 0).
			Find(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.EqualValues(t, 1, list[0].ID)
		assert.Equal(t, "a", list[0].Group)
		assert.EqualValues(t, 2, list[1].ID)
		assert.Equal(t, "b", list[1].Group)
	})

	t.Run("queryColFindSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(1)}, []driver.Value{int64(2)})
		engine := newScriptedEngine(t, recorder)

		list, err := QueryCol[int64](engine).
			Select("id").
			From("reserved_word_models").
			Find(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, list)
	})
}
