package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExistBuilderPreservesArgumentsAndOriginalQuery(t *testing.T) {
	b := Select("id").With("WITH wanted AS (SELECT ? AS id)", 3).
		From("wanted").Distinct().Where("id > ?", 1).Limit(2).Offset(1)
	before, beforeArgs, err := b.ToSql()
	require.NoError(t, err)
	query, args, err := b.ToExistBuilder(false).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "WITH wanted AS (SELECT ? AS id) SELECT 1 FROM (SELECT DISTINCT id FROM wanted WHERE id > ? LIMIT 2 OFFSET 1) AS lorm_exist LIMIT 1", query)
	assert.Equal(t, []any{3, 1}, args)
	after, afterArgs, err := b.ToSql()
	require.NoError(t, err)
	assert.Equal(t, before, after)
	assert.Equal(t, beforeArgs, afterArgs)
}
