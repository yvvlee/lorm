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

	repo := NewUserRepository(engine)
	var u User

	alice := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	if _, err := repo.Insert(ctx, alice); err != nil {
		log.Fatal(err)
	}

	if _, err := repo.InsertAll(ctx, []*User{
		{Name: "Bob", Email: "bob@example.com", Age: 17},
		{Name: "Carol", Email: "carol@example.com", Age: 42},
	}); err != nil {
		log.Fatal(err)
	}

	loadedAlice, err := repo.Get(ctx, alice.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("repository get: id=%d name=%s age=%d\n", loadedAlice.ID, loadedAlice.Name, loadedAlice.Age)

	if _, err := repo.UpdateMap(ctx, alice.ID, map[string]any{
		u.LormCols().Age(): 31,
	}); err != nil {
		log.Fatal(err)
	}

	adults, err := repo.ListAdults(ctx, 18)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("adult users:")
	for _, item := range adults {
		fmt.Printf("- id=%d name=%s age=%d\n", item.ID, item.Name, item.Age)
	}

	exists, err := repo.Exist(ctx, alice.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("alice exists before delete: %v\n", exists)

	if _, err := repo.Delete(ctx, alice.ID); err != nil {
		log.Fatal(err)
	}

	exists, err = repo.Exist(ctx, alice.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("alice exists after delete: %v\n", exists)
}
