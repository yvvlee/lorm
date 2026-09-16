package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
)

func TestInsertValuesWithSubquery(t *testing.T) {
	e := initEngine(t)
	t.Cleanup(func() { _ = e.Close() })
	ctx := context.Background()
	_, err := e.Exec(ctx, "CREATE TABLE lorm_subquery_values (id BIGINT PRIMARY KEY, name VARCHAR(100))")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = e.Exec(ctx, "DROP TABLE lorm_subquery_values") })
	query, args, err := builder.Insert("lorm_subquery_values").Columns("id", "name").
		Values(1, builder.Select().AddColumn("?", "from subquery")).ToSql()
	require.NoError(t, err)
	result, err := e.Exec(ctx, query, args...)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, affected)
	rows, err := e.SQL(ctx, "SELECT name FROM lorm_subquery_values WHERE id = ?", 1)
	require.NoError(t, err)
	defer rows.Close()
	var name string
	require.NoError(t, lorm.ScanCol(rows, &name))
	require.Equal(t, "from subquery", name)
}
