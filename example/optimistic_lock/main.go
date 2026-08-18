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

	repo := engine.Repository[*Document]()
	doc := &Document{
		Title:   "Draft",
		Version: 1,
	}
	if _, err := repo.Insert(ctx, doc); err != nil {
		log.Fatal(err)
	}

	firstCopy, err := repo.Get(ctx, doc.ID)
	if err != nil {
		log.Fatal(err)
	}
	staleCopy, err := repo.Get(ctx, doc.ID)
	if err != nil {
		log.Fatal(err)
	}

	firstCopy.Title = "Published"
	rowsAffected, err := engine.Update[*Document]().SetModel(firstCopy).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("first update rows=%d version(in memory)=%d\n", rowsAffected, firstCopy.Version)

	staleCopy.Title = "Stale Write"
	rowsAffected, err = engine.Update[*Document]().SetModel(staleCopy).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("stale update rows=%d version(in memory)=%d\n", rowsAffected, staleCopy.Version)

	current, err := repo.Get(ctx, doc.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("current document: id=%d title=%q version=%d\n", current.ID, current.Title, current.Version)
}
