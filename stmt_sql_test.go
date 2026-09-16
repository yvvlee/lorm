package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
)

func TestToSqlPreviewsWithoutExecutingOrChangingStatements(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, true, testLogger{})
	engine.config.Dialect.PlaceholderFormat = builder.Dollar
	selectStmt := engine.Query[*reservedWordModel]().Where(builder.Eq{"id": 7}).Desc("group")
	updateStmt := engine.Update[*reservedWordModel]().Set("group", "staff").ID(7)
	deleteStmt := engine.Delete[*reservedWordModel]().ID(7)
	for _, tt := range []struct {
		name string
		stmt builder.Sqlizer
		want string
		args []any
	}{
		{"select", selectStmt, "SELECT `id`, `group` FROM `order` WHERE `id` = $1 ORDER BY `group` DESC", []any{7}},
		{"update", updateStmt, "UPDATE `order` SET `group` = $1 WHERE `id` = $2", []any{"staff", 7}},
		{"delete", deleteStmt, "DELETE FROM `order` WHERE `id` = $1", []any{7}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for range 2 {
				query, args, err := tt.stmt.ToSql()
				require.NoError(t, err)
				assert.Equal(t, tt.want, query)
				assert.Equal(t, tt.args, args)
			}
		})
	}
	assert.Empty(t, recorder.Calls())
	assert.Empty(t, selectStmt.builder.GetColumns())
	// Adding a column after preview must not append it to a default projection.
	query, args, err := selectStmt.AddColumn("COUNT(*)").ToSql()
	require.NoError(t, err)
	assert.Equal(t, "SELECT COUNT(*) FROM `order` WHERE `id` = $1 ORDER BY `group` DESC", query)
	assert.Equal(t, []any{7}, args)
	_, err = updateStmt.Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "UPDATE `order` SET `group` = $1 WHERE `id` = $2", recorder.Last().query)
}

func TestToSqlPreservesStatementErrorsAndWriteGuards(t *testing.T) {
	engine := &Engine{config: &Config{}}
	for _, stmt := range []builder.Sqlizer{
		engine.Query[*testNoPrimaryKeyModel]().ID(1),
		engine.Update[*testNoPrimaryKeyModel]().ID(1),
		engine.Delete[*testNoPrimaryKeyModel]().ID(1),
	} {
		for range 2 {
			_, _, err := stmt.ToSql()
			require.ErrorContains(t, err, "primary key")
		}
	}
	stmt := engine.Delete[*Test]()
	query, _, err := stmt.ToSql()
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM test", query)
	_, err = stmt.Exec(context.Background())
	require.ErrorContains(t, err, "requires a WHERE clause")
}

func TestToSqlReturnsBuildAndPlaceholderErrors(t *testing.T) {
	engine := &Engine{config: &Config{}}
	_, _, err := engine.Update[*Test]().ToSql()
	require.ErrorContains(t, err, "Set clause")
	engine.config.Dialect.PlaceholderFormat = errorPlaceholder{}
	stmt := engine.Query[*Test]().Where("id = ?", 7)
	_, _, err = stmt.ToSql()
	require.ErrorContains(t, err, "replace failed")
	// A failed preview must leave the query available for another attempt.
	engine.config.Dialect.PlaceholderFormat = builder.Question
	_, args, err := stmt.ToSql()
	require.NoError(t, err)
	assert.Equal(t, []any{7}, args)
}
