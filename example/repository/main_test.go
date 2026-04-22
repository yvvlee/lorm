package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestRepositoryFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	repo := NewUserRepository(engine)
	var u User

	alice := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	_, err = repo.Insert(ctx, alice)
	assert.NoError(t, err)

	_, err = repo.InsertAll(ctx, []*User{
		{Name: "Bob", Email: "bob@example.com", Age: 17},
		{Name: "Carol", Email: "carol@example.com", Age: 42},
	})
	assert.NoError(t, err)

	loadedAlice, err := repo.Get(ctx, alice.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", loadedAlice.Name)

	_, err = repo.UpdateMap(ctx, alice.ID, map[string]any{u.Fields().Age(): 31})
	assert.NoError(t, err)

	adults, err := repo.ListAdults(ctx, 18)
	assert.NoError(t, err)
	assert.Len(t, adults, 2)

	exists, err := repo.Exist(ctx, alice.ID)
	assert.NoError(t, err)
	assert.True(t, exists)

	_, err = repo.Delete(ctx, alice.ID)
	assert.NoError(t, err)

	exists, err = repo.Exist(ctx, alice.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}
