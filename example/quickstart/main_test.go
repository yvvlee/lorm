package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm"
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
	_, err = lorm.Insert[*User](engine).AddModel(alice).Exec(ctx)
	assert.NoError(t, err)

	bob := &User{Name: "Bob", Email: "bob@example.com"}
	_, err = lorm.Insert[*User](engine).AddModel(bob).Exec(ctx)
	assert.NoError(t, err)

	loadedAlice, err := lorm.Query[*User](engine).
		Where(builder.Eq{u.Fields().Email(): "alice@example.com"}).
		Get(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", loadedAlice.Name)

	_, err = lorm.Update[*User](engine).
		ID(loadedAlice.ID).
		SetMap(map[string]any{u.Fields().Name(): "Alice Updated"}).
		Exec(ctx)
	assert.NoError(t, err)

	users, err := lorm.Query[*User](engine).
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "Alice Updated", users[0].Name)

	_, err = lorm.Delete[*User](engine).ID(bob.ID).Exec(ctx)
	assert.NoError(t, err)

	remaining, err := lorm.Query[*User](engine).
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, remaining, 1)
}
