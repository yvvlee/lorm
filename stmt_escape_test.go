package lorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/names"
)

func TestEscapePredicateNestedSqlizers(t *testing.T) {
	escaped := escapePredicate(names.NewQuoter('`', '`'), builder.And{
		builder.Eq{"group": "staff"},
		builder.Or{
			builder.NotEq{"id": 7},
			builder.IsNull("`name`"),
		},
	})

	sql, args, err := escaped.(builder.Sqlizer).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "(`group` = ? AND (`id` <> ? OR `name` IS NULL))", sql)
	assert.Equal(t, []any{"staff", 7}, args)
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
			escaped := escapePredicate(names.NewQuoter('`', '`'), tt.pred)
			sql, _, err := escaped.(builder.Sqlizer).ToSql()
			require.NoError(t, err)
			assert.Equal(t, tt.want, sql)
		})
	}
}
