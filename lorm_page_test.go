package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestSelectModelPageCoverage(t *testing.T) {
	t.Run("normalizesZeroPageAndReturnsRows", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(2)})
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(1), "staff"})
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Where(builder.Eq{"group": "staff"}).
			Page(context.Background(), 0, 1)
		require.NoError(t, err)
		assert.EqualValues(t, 2, total)
		require.Len(t, list, 1)
		assert.EqualValues(t, 1, list[0].ID)
		assert.Equal(t, "staff", list[0].Group)
	})

	t.Run("returnsNilWhenOffsetExceedsCount", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(1)})
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().Page(context.Background(), 2, 1)
		require.NoError(t, err)
		assert.Nil(t, list)
		assert.EqualValues(t, 1, total)
	})

	t.Run("returnsNilWhenOffsetOverflows", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(5)})
		engine := newScriptedEngine(t, recorder)

		page := uint64(1)<<63 + 1
		list, total, err := engine.Query[*reservedWordModel]().Page(context.Background(), page, 2)
		require.NoError(t, err)
		assert.Nil(t, list)
		assert.EqualValues(t, 5, total)
		assert.Len(t, recorder.queryCalls, 1)
	})
}

func TestSelectPageColsCoverage(t *testing.T) {
	t.Run("returnsColumnPage", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(3)})
		recorder.QueueQueryRows(
			[]string{"id"},
			[]driver.Value{int64(1)},
			[]driver.Value{int64(2)},
		)
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Select("id").
			Where("group = ?", "staff").
			OrderBy("id").
			PageCols[int64](context.Background(), 0, 2)
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, list)
		assert.EqualValues(t, 3, total)
		require.Len(t, recorder.queryCalls, 2)
		assert.Equal(t, "SELECT COUNT(1) FROM order WHERE group = ?", recorder.queryCalls[0].query)
		assert.Equal(t, "SELECT id FROM order WHERE group = ? ORDER BY id LIMIT 2 OFFSET 0", recorder.queryCalls[1].query)
	})

	t.Run("countsDistinctProjection", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(2)})
		recorder.QueueQueryRows([]string{"group"}, []driver.Value{"ops"})
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Select("group").
			Distinct().
			PageCols[string](context.Background(), 1, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"ops"}, list)
		assert.EqualValues(t, 2, total)
		require.Len(t, recorder.queryCalls, 2)
		assert.Equal(t, "SELECT COUNT(1) FROM (SELECT DISTINCT group FROM order) AS sub", recorder.queryCalls[0].query)
	})

	t.Run("countsGroupedProjection", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(1)})
		recorder.QueueQueryRows([]string{"group"}, []driver.Value{"staff"})
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Select("group").
			GroupBy("group").
			Having("COUNT(1) > ?", 1).
			PageCols[string](context.Background(), 1, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"staff"}, list)
		assert.EqualValues(t, 1, total)
		require.Len(t, recorder.queryCalls, 2)
		assert.Equal(t, "SELECT COUNT(1) FROM (SELECT group FROM order GROUP BY group HAVING COUNT(1) > ?) AS sub", recorder.queryCalls[0].query)
	})

	t.Run("returnsCountWithScanError", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(1)})
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(1), "staff"})
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Select("id", "group").
			PageCols[int64](context.Background(), 1, 10)
		assert.ErrorContains(t, err, "exactly one column")
		assert.Nil(t, list)
		assert.EqualValues(t, 1, total)
	})

	t.Run("countErrorSkipsDataQuery", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows(
			[]string{"left", "right"},
			[]driver.Value{int64(1), int64(2)},
		)
		engine := newScriptedEngine(t, recorder)

		list, total, err := engine.Query[*reservedWordModel]().
			Select("id").
			PageCols[int64](context.Background(), 1, 10)
		assert.ErrorContains(t, err, "exactly one column")
		assert.Nil(t, list)
		assert.Zero(t, total)
		assert.Len(t, recorder.queryCalls, 1)
	})

	t.Run("rejectsZeroSizeBeforeQuery", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		engine := newScriptedEngine(t, recorder)

		_, _, err := engine.Query[*reservedWordModel]().
			Select("id").
			PageCols[int64](context.Background(), 1, 0)
		assert.ErrorContains(t, err, "size can not be zero")
		assert.Empty(t, recorder.queryCalls)
	})
}
