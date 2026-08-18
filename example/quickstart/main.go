package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/yvvlee/lorm/builder"
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

	var u User

	alice := &User{Name: "Alice", Email: "alice@example.com"}
	if _, err := engine.Insert[*User]().AddModel(alice).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	bob := &User{Name: "Bob", Email: "bob@example.com"}
	if _, err := engine.Insert[*User]().AddModel(bob).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	loadedAlice, ok, err := engine.Select[*User]().
		Where(builder.Eq{u.Fields().Email(): "alice@example.com"}).
		Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Fatal("alice not found")
	}
	fmt.Printf("loaded user: id=%d name=%s email=%s\n", loadedAlice.ID, loadedAlice.Name, loadedAlice.Email)

	if _, err := engine.Update[*User]().
		ID(loadedAlice.ID).
		SetMap(map[string]any{
			u.Fields().Name(): "Alice Updated",
		}).
		Exec(ctx); err != nil {
		log.Fatal(err)
	}

	users, err := engine.Select[*User]().
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("users after update:")
	for _, item := range users {
		fmt.Printf("- id=%d name=%s email=%s\n", item.ID, item.Name, item.Email)
	}

	if _, err := engine.Delete[*User]().ID(bob.ID).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	remaining, err := engine.Select[*User]().
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("remaining rows: %d\n", len(remaining))
}
