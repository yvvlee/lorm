package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/builder/try"
)

func TestWritesRejectEmptyOrAndNestedTrueConditions(t *testing.T) {
	for _, pred := range []builder.Sqlizer{
		builder.Or{},
		builder.Or{try.Equal[int]("id", nil), builder.Or{}},
		builder.And{builder.Eq{}, builder.Eq{}},
		builder.Or{builder.Eq{}, builder.Eq{"id": 7}},
		builder.Or{builder.NotIn("id", []int{}), builder.Eq{"id": 7}},
	} {
		recorder := newCaptureSQLRecorder()
		engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
		update := engine.Update[*reservedWordModel]().Set("group", "all").Where(pred)
		delete := engine.Delete[*reservedWordModel]().Where(pred)
		_, err := update.Exec(context.Background())
		require.ErrorContains(t, err, "requires a WHERE clause")
		_, err = delete.Exec(context.Background())
		require.ErrorContains(t, err, "requires a WHERE clause")
		assert.Empty(t, recorder.Calls())

		// Exec resets the rejected statements; explicit global writes still omit empty WHERE clauses.
		_, err = update.Set("group", "all").Where(pred).AllowGlobalWrite().Exec(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "UPDATE `order` SET `group` = ?", recorder.Last().query)
		_, err = delete.Where(pred).AllowGlobalWrite().Exec(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "DELETE FROM `order`", recorder.Last().query)
	}
}

func TestEmptyOrPreservesOtherFiltersAndBoundParameters(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, true, testLogger{})
	engine.config.Dialect.PlaceholderFormat = builder.Dollar
	filter := builder.And{
		builder.Or{try.Equal[int]("id", nil)},
		builder.Or{builder.In("id", []int{}), builder.Eq{"id": 7}},
	}
	query, args, err := engine.Query[*reservedWordModel]().Where(filter).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT `id`, `group` FROM `order` WHERE ((`id` = $1))", query)
	assert.Equal(t, []any{7}, args)
	_, err = engine.Update[*reservedWordModel]().Set("group", "staff").Where(filter).Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "UPDATE `order` SET `group` = $1 WHERE ((`id` = $2))", recorder.Last().query)
	assert.Equal(t, []any{"staff", int64(7)}, recorder.Last().args)
	_, err = engine.Delete[*reservedWordModel]().Where(filter).Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM `order` WHERE ((`id` = $1))", recorder.Last().query)
	assert.Equal(t, []any{int64(7)}, recorder.Last().args)

	// An explicit false predicate is restrictive and must not become an empty group.
	_, err = engine.Delete[*reservedWordModel]().Where(builder.Or{builder.In("id", []int{})}).Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM `order` WHERE (1=0)", recorder.Last().query)
	assert.Empty(t, recorder.Last().args)
}

type changingWriteCondition struct {
	calls int
	first string
}

func (p *changingWriteCondition) ToSql() (string, []any, error) {
	p.calls++
	if p.calls == 1 {
		return p.first, nil, nil
	}
	return "id = 7", nil, nil
}

func TestWriteGuardUsesTheExecutedCondition(t *testing.T) {
	for _, first := range []string{"1=1", "", "id = 3", "1=0"} {
		for _, operation := range []string{"update", "delete"} {
			t.Run(operation+"/"+first, func(t *testing.T) {
				recorder := newCaptureSQLRecorder()
				engine := newCaptureSQLEngine(t, recorder, false, testLogger{})
				pred := &changingWriteCondition{first: first}
				var err error
				if operation == "update" {
					_, err = engine.Update[*reservedWordModel]().Set("group", "all").Where(pred).Exec(context.Background())
				} else {
					_, err = engine.Delete[*reservedWordModel]().Where(pred).Exec(context.Background())
				}
				assert.Equal(t, 1, pred.calls)
				if first == "1=1" || first == "" {
					require.ErrorContains(t, err, "requires a WHERE clause")
					assert.Empty(t, recorder.Calls())
				} else {
					require.NoError(t, err)
					assert.Contains(t, recorder.Last().query, "WHERE ")
					assert.Contains(t, recorder.Last().query, first)
				}
			})
		}
	}
}
