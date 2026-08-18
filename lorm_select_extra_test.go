package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestSelectScalarGetSuccessCoverage(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	recorder.QueueQueryRows([]string{"id"}, []driver.Value{int64(9)})
	engine := newScriptedEngine(t, recorder)

	value, ok, err := engine.Select[int64]().
		Select("id").
		From("conversion_models").
		Where("name = ?", "alpha").
		Get(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, 9, value)

	call := recorder.LastQuery()
	require.NotNil(t, call)
	assert.Equal(t, "SELECT id FROM conversion_models WHERE name = ? LIMIT 1", call.query)
	assert.Equal(t, []any{"alpha"}, call.args)
}

func TestSelectModelGetAndFindCoverage(t *testing.T) {
	t.Run("getSuccess", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(3), "ops"})
		engine := newScriptedEngine(t, recorder)

		model, ok, err := engine.Select[*reservedWordModel]().
			Where(builder.Eq{"group": "ops"}).
			Get(context.Background())
		require.NoError(t, err)
		require.True(t, ok)
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

		list, err := engine.Select[*reservedWordModel]().
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

		list, err := engine.Select[int64]().
			Select("id").
			From("reserved_word_models").
			Find(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, list)
	})
}
