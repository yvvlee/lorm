package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestQueryBuilderFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if exampleutil.IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	_, err = engine.Insert[*Product]().AddModels(
		&Product{Name: "Go in Action", Category: "book", Price: 3200, Status: "active"},
		&Product{Name: "Go Sticker Pack", Category: "merch", Price: 500, Status: "active"},
		&Product{Name: "Mechanical Keyboard", Category: "tool", Price: 8900, Status: "active"},
		&Product{Name: "Archived Notebook", Category: "tool", Price: 1200, Status: "archived"},
		&Product{Name: "Go CLI Cheatsheet", Category: "book", Price: 800, Status: "active"},
	).Exec(ctx)
	assert.NoError(t, err)

	var p Product
	filtered, err := engine.Query[*Product]().
		Where(builder.And{
			builder.In(p.LormCols().Category(), []string{"book", "tool"}),
			builder.Or{
				builder.Like(p.LormCols().Name(), "%Go%"),
				builder.Lt(p.LormCols().Price(), int64(3000)),
			},
			builder.Eq{p.LormCols().Status(): "active"},
		}).
		OrderBy(p.LormCols().Price() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "Go CLI Cheatsheet", filtered[0].Name)

	count, ok, err := engine.Query[*Product]().
		Select("COUNT(1)").
		Where(builder.Eq{p.LormCols().Status(): "active"}).
		GetCol[int64](ctx)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, 4, count)
}
