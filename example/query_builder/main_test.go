package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestQueryBuilderFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
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
	filtered, err := engine.Select[*Product]().
		Where(builder.And{
			builder.In(p.Fields().Category(), []string{"book", "tool"}),
			builder.Or{
				builder.Like(p.Fields().Name(), "%Go%"),
				builder.Lt(p.Fields().Price(), int64(3000)),
			},
			builder.Eq{p.Fields().Status(): "active"},
		}).
		OrderBy(p.Fields().Price() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "Go CLI Cheatsheet", filtered[0].Name)

	count, ok, err := engine.Select[int64]().
		Select("COUNT(1)").
		From(p.TableName()).
		Where(builder.Eq{p.Fields().Status(): "active"}).
		Get(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, 4, count)
}
