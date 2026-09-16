package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestEscapePredicateNestedSqlizers(t *testing.T) {
	escaped, err := escapePredicate(names.NewQuoter('`', '`'), builder.And{
		builder.Eq{"group": "staff"},
		builder.Or{
			builder.NotEq{"id": 7},
			builder.IsNull("`name`"),
		},
	})
	require.NoError(t, err)

	sql, args, err := escaped.(builder.Sqlizer).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "(`group` = ? AND (`id` <> ? OR `name` IS NULL))", sql)
	assert.Equal(t, []any{"staff", 7}, args)
}

func TestEscapedMapColumnCollisions(t *testing.T) {
	for _, quote := range []byte{'`', '"'} {
		escaper := names.NewQuoter(quote, quote)
		column := escaper.Escape("id")
		values := map[string]any{"id": nil, column: 2}
		for _, pred := range []any{
			values, builder.Eq(values), builder.NotEq(values),
			builder.And{builder.Eq{"name": "alice"}, builder.Or{builder.NotEq(values)}},
			builder.Or{builder.Eq{}, builder.Eq(values)},
		} {
			_, err := escapePredicate(escaper, pred)
			require.ErrorContains(t, err, "duplicate column")
		}
		_, err := escapeMap(escaper, map[string]any{"users.id": 1, escaper.Escape("users.id"): 2})
		require.ErrorContains(t, err, "duplicate column")

		// Quoted dots are literal, so these remain distinct identifiers.
		_, err = escapeMap(escaper, map[string]any{"users.id": 1, string(quote) + "users.id" + string(quote): 2})
		require.NoError(t, err)
	}
}

func TestStatementsRejectEscapedMapCollisionsBeforeSQL(t *testing.T) {
	recorder := newConversionRecorder()
	e := newConversionTestEngine(t, recorder)
	e.config.Dialect.Escaper = names.NewQuoter('`', '`')
	ctx := context.Background()
	values := map[string]any{"id": 1, "`id`": 2}

	query := e.Query[*conversionModel]().Where(values).Where("name = ?", "alice")
	_, _, err := query.Clone().ToSql()
	require.ErrorContains(t, err, "duplicate column")
	_, err = query.Find(ctx)
	require.ErrorContains(t, err, "duplicate column")
	_, _, err = query.ID(1).ToSql()
	require.NoError(t, err, "terminal methods must clear the previous error")

	_, err = e.Query[*conversionModel]().GroupBy("id").Having(builder.Or{builder.NotEq(values)}).Count(ctx)
	require.ErrorContains(t, err, "duplicate column")
	_, err = e.Update[*conversionModel]().Set("name", "alice").Where(values).Exec(ctx)
	require.ErrorContains(t, err, "duplicate column")
	_, err = e.Update[*conversionModel]().ID(1).SetMap(values).Exec(ctx)
	require.ErrorContains(t, err, "duplicate column")
	_, err = e.Delete[*conversionModel]().Where(values).AllowGlobalWrite().Exec(ctx)
	require.ErrorContains(t, err, "duplicate column")
	require.Empty(t, recorder.execCalls)
	require.Empty(t, recorder.queryCalls)
}

func TestEscapePredicateFieldHelpers(t *testing.T) {
	tests := []struct {
		name string
		pred builder.Sqlizer
		want string
	}{
		{name: "in", pred: builder.In("group", []int{1, 2}), want: "`group` IN (?,?)"},
		{name: "not in", pred: builder.NotIn("group", []int{1, 2}), want: "`group` NOT IN (?,?)"},
		{name: "like", pred: builder.Like("group", "staff%"), want: "`group` LIKE ?"},
		{name: "not like", pred: builder.NotLike("group", "staff%"), want: "`group` NOT LIKE ?"},
		{name: "ilike", pred: builder.ILike("group", "staff%"), want: "`group` ILIKE ?"},
		{name: "not ilike", pred: builder.NotILike("group", "staff%"), want: "`group` NOT ILIKE ?"},
		{name: "is null", pred: builder.IsNull("group"), want: "`group` IS NULL"},
		{name: "is not null", pred: builder.IsNotNull("group"), want: "`group` IS NOT NULL"},
		{name: "less than", pred: builder.Lt("group", 1), want: "`group` < ?"},
		{name: "less than or equal", pred: builder.Lte("group", 1), want: "`group` <= ?"},
		{name: "greater than", pred: builder.Gt("group", 1), want: "`group` > ?"},
		{name: "greater than or equal", pred: builder.Gte("group", 1), want: "`group` >= ?"},
		{name: "between", pred: builder.Between("group", 1, 2), want: "`group` BETWEEN ? AND ?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped, err := escapePredicate(names.NewQuoter('`', '`'), tt.pred)
			require.NoError(t, err)
			sql, _, err := escaped.(builder.Sqlizer).ToSql()
			require.NoError(t, err)
			assert.Equal(t, tt.want, sql)
		})
	}
}
