package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/builder/try"
)

func TestEmptyOrAndConstantConditions(t *testing.T) {
	engine := initEngine(t)
	defer engine.Close()
	ctx := context.Background()
	insertBasicRows(t, engine, ctx)

	for _, tt := range []struct {
		name  string
		pred  builder.Sqlizer
		count uint64
	}{
		{"empty or", builder.Or{}, 2},
		{"absent optional filters", builder.Or{try.Equal[string]("str", nil), builder.Or{}}, 2},
		{"empty or beside filter", builder.And{builder.Or{}, builder.Eq{"str": "a"}}, 1},
		{"true or", builder.Or{builder.NotIn("id", []int{}), builder.Eq{"str": "a"}}, 2},
		{"false or", builder.Or{builder.Or{}, builder.In("id", []int{})}, 0},
		{"nested true beside filter", builder.And{builder.Or{builder.Eq{}, builder.Eq{"str": "missing"}}, builder.Eq{"str": "a"}}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			models, err := engine.Query[*Test]().Where(tt.pred).Find(ctx)
			require.NoError(t, err)
			assert.Len(t, models, int(tt.count))
			count, err := engine.Query[*Test]().Where(tt.pred).Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.count, count)
		})
	}

	rows, err := engine.Delete[*Test]().Where(builder.Or{builder.In("id", []int{})}).Exec(ctx)
	require.NoError(t, err)
	assert.Zero(t, rows)
	_, err = engine.Delete[*Test]().Where(builder.Or{builder.Eq{}, builder.Eq{"str": "a"}}).Exec(ctx)
	require.ErrorContains(t, err, "requires a WHERE clause")
	count, err := engine.Query[*Test]().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)
}
