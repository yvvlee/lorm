package lorm

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountPreservesResultSemanticsAndResets(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(*SelectStmt[*reservedWordModel]) *SelectStmt[*reservedWordModel]
		want  string
		args  []any
	}{
		{"filtered", func(s *SelectStmt[*reservedWordModel]) *SelectStmt[*reservedWordModel] {
			return s.Where("id > ?", 2).Desc("id").Limit(1).Offset(20)
		}, "SELECT COUNT(1) FROM order WHERE id > ?", []any{int64(2)}},
		{"distinct", func(s *SelectStmt[*reservedWordModel]) *SelectStmt[*reservedWordModel] {
			return s.Select("group").Distinct()
		}, "SELECT COUNT(1) FROM (SELECT DISTINCT group FROM order) AS sub", []any{}},
		{"grouped", func(s *SelectStmt[*reservedWordModel]) *SelectStmt[*reservedWordModel] {
			return s.Select("group").GroupBy("group").Having("COUNT(*) > ?", 1)
		}, "SELECT COUNT(1) FROM (SELECT group FROM order GROUP BY group HAVING COUNT(*) > ?) AS sub", []any{int64(1)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := newScriptedQueryRecorder()
			recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(3)})
			engine := newScriptedEngine(t, recorder)
			stmt := tt.build(engine.Query[*reservedWordModel]())
			count, err := stmt.Count(context.Background())
			require.NoError(t, err)
			assert.EqualValues(t, 3, count)
			require.Len(t, recorder.queryCalls, 1)
			assert.Equal(t, tt.want, recorder.LastQuery().query)
			assert.Equal(t, tt.args, recorder.LastQuery().args)
			query, args, err := stmt.ToSql()
			require.NoError(t, err)
			assert.Equal(t, "SELECT id, group FROM order", query)
			assert.Empty(t, args)
		})
	}
}

func TestCountResetsAfterErrorsAndReturnsZeroForEmptyResults(t *testing.T) {
	recorder := newScriptedQueryRecorder()
	engine := newScriptedEngine(t, recorder)
	stmt := engine.Query[*reservedWordModel]().Where("id = ?", 1)
	recorder.QueueQueryRows([]string{"count"}, []driver.Value{"invalid count"})
	_, err := stmt.Count(context.Background())
	require.Error(t, err)
	recorder.QueueQueryRows([]string{"count"}, []driver.Value{int64(0)})
	count, err := stmt.Count(context.Background())
	require.NoError(t, err)
	assert.Zero(t, count)
	assert.Equal(t, "SELECT COUNT(1) FROM order", recorder.LastQuery().query)
	assert.Empty(t, recorder.LastQuery().args)

	invalid := engine.Query[*testNoPrimaryKeyModel]().ID(1)
	_, err = invalid.Count(context.Background())
	require.ErrorContains(t, err, "primary key")
	assert.Nil(t, invalid.err)
	assert.Len(t, recorder.queryCalls, 2)
}
