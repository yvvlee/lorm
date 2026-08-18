package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestOptimisticLockFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	repo := engine.Repository[*Document]()
	doc := &Document{Title: "Draft", Version: 1}
	_, err = repo.Insert(ctx, doc)
	assert.NoError(t, err)

	firstCopy, err := repo.Get(ctx, doc.ID)
	assert.NoError(t, err)
	staleCopy, err := repo.Get(ctx, doc.ID)
	assert.NoError(t, err)

	firstCopy.Title = "Published"
	rowsAffected, err := engine.Update[*Document]().SetModel(firstCopy).Exec(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, rowsAffected)
	assert.Equal(t, 2, firstCopy.Version)

	staleCopy.Title = "Stale Write"
	rowsAffected, err = engine.Update[*Document]().SetModel(staleCopy).Exec(ctx)
	assert.NoError(t, err)
	assert.Zero(t, rowsAffected)
	assert.Equal(t, 1, staleCopy.Version)

	current, err := repo.Get(ctx, doc.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Published", current.Title)
	assert.Equal(t, 2, current.Version)
}
