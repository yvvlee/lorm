package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestPaginationFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	posts := []*Post{
		{Title: "Getting Started with LORM", Category: "guide"},
		{Title: "Repository Pattern with LORM", Category: "guide"},
		{Title: "Transactions Done Explicitly", Category: "guide"},
		{Title: "JSON Fields in Practice", Category: "advanced"},
		{Title: "Projection Models and Joins", Category: "advanced"},
	}
	_, err = lorm.Insert[*Post](engine).AddModels(posts...).Exec(ctx)
	assert.NoError(t, err)

	var p Post
	pageRows, total, err := lorm.Query[*Post](engine).
		OrderBy(p.Fields().ID() + " ASC").
		Page(ctx, 2, 2)
	assert.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, pageRows, 2)
	assert.Equal(t, "Transactions Done Explicitly", pageRows[0].Title)
}
