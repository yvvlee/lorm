package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestJSONFieldFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	profile := &UserProfile{
		Name: "Alice",
		Tags: []string{"golang", "sqlite", "json"},
		Preferences: Preferences{
			Theme:         "light",
			EmailsEnabled: true,
		},
	}
	_, err = engine.Insert[*UserProfile]().AddModel(profile).Exec(ctx)
	assert.NoError(t, err)

	repo := engine.Repository[*UserProfile]()
	loaded, err := repo.Get(ctx, profile.ID)
	assert.NoError(t, err)
	assert.Equal(t, []string{"golang", "sqlite", "json"}, loaded.Tags)
	assert.Equal(t, "light", loaded.Preferences.Theme)

	loaded.Tags = append(loaded.Tags, "lorm")
	loaded.Preferences.Theme = "dark"
	loaded.Preferences.EmailsEnabled = false
	_, err = engine.Update[*UserProfile]().SetModel(loaded).Exec(ctx)
	assert.NoError(t, err)

	reloaded, err := repo.Get(ctx, profile.ID)
	assert.NoError(t, err)
	assert.Equal(t, []string{"golang", "sqlite", "json", "lorm"}, reloaded.Tags)
	assert.Equal(t, "dark", reloaded.Preferences.Theme)
	assert.False(t, reloaded.Preferences.EmailsEnabled)
}
