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
