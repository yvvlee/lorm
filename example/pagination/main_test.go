package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestPaginationFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if exampleutil.IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	posts := []*Post{
		{Title: "Getting Started with LORM", Category: "guide"},
		{Title: "Repository Pattern with LORM", Category: "guide"},
		{Title: "Transactions Done Explicitly", Category: "guide"},
		{Title: "JSON Fields in Practice", Category: "advanced"},
		{Title: "Projection Models and Joins", Category: "advanced"},
	}
	_, err = engine.Insert[*Post]().AddModels(posts...).Exec(ctx)
	assert.NoError(t, err)

	var p Post
	pageRows, total, err := engine.Query[*Post]().
		OrderBy(p.Fields().ID()+" ASC").
		Page(ctx, 2, 2)
	assert.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, pageRows, 2)
	assert.Equal(t, "Transactions Done Explicitly", pageRows[0].Title)
}
