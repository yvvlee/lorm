package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestQuickstartFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if exampleutil.IsSQLiteDriverUnavailable(err) {
		t.Skipf("sqlite example unavailable: %v", err)
	}
	require.NoError(t, err)
	defer cleanup()

	var u User
	alice := &User{Name: "Alice", Email: "alice@example.com"}
	_, err = engine.Insert[*User]().AddModel(alice).Exec(ctx)
	assert.NoError(t, err)

	bob := &User{Name: "Bob", Email: "bob@example.com"}
	_, err = engine.Insert[*User]().AddModel(bob).Exec(ctx)
	assert.NoError(t, err)

	loadedAlice, ok, err := engine.Query[*User]().
		Where(builder.Eq{u.LormCols().Email(): "alice@example.com"}).
		Get(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "Alice", loadedAlice.Name)

	_, err = engine.Update[*User]().
		ID(loadedAlice.ID).
		SetMap(map[string]any{u.LormCols().Name(): "Alice Updated"}).
		Exec(ctx)
	assert.NoError(t, err)

	users, err := engine.Query[*User]().
		OrderBy(u.LormCols().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "Alice Updated", users[0].Name)

	_, err = engine.Delete[*User]().ID(bob.ID).Exec(ctx)
	assert.NoError(t, err)

	remaining, err := engine.Query[*User]().
		OrderBy(u.LormCols().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, remaining, 1)
}
