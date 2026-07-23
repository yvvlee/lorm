package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestQueryModelPageCoverage(t *testing.T) {
	t.Run("normalizesZeroPageAndReturnsRows", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(2)})
		recorder.QueueQueryRows([]string{"id", "group"}, []driver.Value{int64(1), "staff"})
		engine := newScriptedEngine(t, recorder)

		list, total, err := Query[*reservedWordModel](engine).
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

		list, total, err := Query[*reservedWordModel](engine).Page(context.Background(), 2, 1)
		require.NoError(t, err)
		assert.Nil(t, list)
		assert.EqualValues(t, 1, total)
	})

	t.Run("returnsNilWhenOffsetOverflows", func(t *testing.T) {
		recorder := newScriptedQueryRecorder()
		recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(5)})
		engine := newScriptedEngine(t, recorder)

		page := uint64(1)<<63 + 1
		list, total, err := Query[*reservedWordModel](engine).Page(context.Background(), page, 2)
		require.NoError(t, err)
		assert.Nil(t, list)
		assert.EqualValues(t, 5, total)
		assert.Len(t, recorder.queryCalls, 1)
	})
}
