package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestQuickstartFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	var u User
	alice := &User{Name: "Alice", Email: "alice@example.com"}
	_, err = engine.Insert[*User]().AddModel(alice).Exec(ctx)
	assert.NoError(t, err)

	bob := &User{Name: "Bob", Email: "bob@example.com"}
	_, err = engine.Insert[*User]().AddModel(bob).Exec(ctx)
	assert.NoError(t, err)

	loadedAlice, ok, err := engine.Select[*User]().
		Where(builder.Eq{u.Fields().Email(): "alice@example.com"}).
		Get(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Alice", loadedAlice.Name)

	_, err = engine.Update[*User]().
		ID(loadedAlice.ID).
		SetMap(map[string]any{u.Fields().Name(): "Alice Updated"}).
		Exec(ctx)
	assert.NoError(t, err)

	users, err := engine.Select[*User]().
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "Alice Updated", users[0].Name)

	_, err = engine.Delete[*User]().ID(bob.ID).Exec(ctx)
	assert.NoError(t, err)

	remaining, err := engine.Select[*User]().
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, remaining, 1)
}
