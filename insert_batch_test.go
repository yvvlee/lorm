package lorm

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertBatchSize(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		size, count, calls, transactions int
	}{
		{"default", 0, DefaultInsertBatchSize + 1, 2, 1},
		{"explicit", 2, 5, 3, 1},
		{"one", 1, 3, 3, 1},
		{"exact", 3, 3, 1, 0},
		{"large", math.MaxInt, 3, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := newCaptureSQLRecorder()
			engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
			models := make([]*reservedWordModel, tc.count)
			for i := range models {
				models[i] = &reservedWordModel{Group: "value"}
			}
			stmt := engine.Insert[*reservedWordModel]().AddModels(models...).Prefix("/* batch */").Ignore()
			if tc.size > 0 {
				stmt.BatchSize(tc.size)
			}
			_, err := stmt.Exec(context.Background())
			require.NoError(t, err)
			calls := recorder.Calls()
			require.Len(t, calls, tc.calls)
			assert.Len(t, recorder.BeginTxCalls(), tc.transactions)
			var count int
			for _, call := range calls {
				assert.Contains(t, call.query, "/* batch */ INSERT IGNORE")
				count += len(call.args)
			}
			assert.Equal(t, tc.count, count)
			for _, model := range models {
				assert.Zero(t, model.ID)
			}
		})
	}
}

func TestInsertBatchSizeCloneAndReset(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	stmt := engine.Insert[*reservedWordModel]().BatchSize(1).AddModels(&reservedWordModel{}, &reservedWordModel{})
	clone := stmt.Clone()
	_, err := clone.Exec(context.Background())
	require.NoError(t, err)
	assert.Len(t, recorder.Calls(), 2)
	recorder.Reset()
	_, err = stmt.Exec(context.Background())
	require.NoError(t, err)
	assert.Len(t, recorder.Calls(), 2)
	recorder.Reset()
	_, err = stmt.AddModels(&reservedWordModel{}, &reservedWordModel{}).Exec(context.Background())
	require.NoError(t, err)
	assert.Len(t, recorder.Calls(), 1)
	assert.Empty(t, recorder.BeginTxCalls())
}

func TestInsertBatchSizeRejectsInvalidSizeAndDifferentShapesBeforeSQL(t *testing.T) {
	for _, size := range []int{0, -1} {
		recorder := newCaptureSQLRecorder()
		engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
		_, err := engine.Insert[*reservedWordModel]().BatchSize(size).Exec(context.Background())
		require.ErrorContains(t, err, "batch size must be positive")
		assert.Empty(t, recorder.Calls())
		assert.Empty(t, recorder.BeginTxCalls())
	}
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
	_, err := engine.Insert[*reservedWordModel]().BatchSize(1).AddModels(&reservedWordModel{}, &reservedWordModel{ID: 42}).Exec(context.Background())
	require.ErrorContains(t, err, "different column shape")
	assert.Empty(t, recorder.Calls())
	assert.Empty(t, recorder.BeginTxCalls())
}
