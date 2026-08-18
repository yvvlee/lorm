package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

//go:embed schema.sql
var schemaSQL string

func main() {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		log.Fatal(err)
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
	if _, err := engine.Insert[*UserProfile]().AddModel(profile).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	loaded, err := engine.Repository[*UserProfile]().Get(ctx, profile.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("loaded profile: id=%d name=%s tags=%v theme=%s emails_enabled=%v\n",
		loaded.ID, loaded.Name, loaded.Tags, loaded.Preferences.Theme, loaded.Preferences.EmailsEnabled)

	loaded.Tags = append(loaded.Tags, "lorm")
	loaded.Preferences.Theme = "dark"
	loaded.Preferences.EmailsEnabled = false
	if _, err := engine.Update[*UserProfile]().SetModel(loaded).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	reloaded, err := engine.Repository[*UserProfile]().Get(ctx, profile.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("updated profile: id=%d tags=%v theme=%s emails_enabled=%v\n",
		reloaded.ID, reloaded.Tags, reloaded.Preferences.Theme, reloaded.Preferences.EmailsEnabled)
}
