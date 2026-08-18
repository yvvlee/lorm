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

	posts := []*Post{
		{Title: "Getting Started with LORM", Category: "guide"},
		{Title: "Repository Pattern with LORM", Category: "guide"},
		{Title: "Transactions Done Explicitly", Category: "guide"},
		{Title: "JSON Fields in Practice", Category: "advanced"},
		{Title: "Projection Models and Joins", Category: "advanced"},
	}
	if _, err := engine.Insert[*Post]().AddModels(posts...).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	var p Post
	pageRows, total, err := engine.Select[*Post]().
		OrderBy(p.Fields().ID()+" ASC").
		Page(ctx, 2, 2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("page=2 size=2 total=%d returned=%d\n", total, len(pageRows))
	for _, item := range pageRows {
		fmt.Printf("- id=%d title=%q category=%s\n", item.ID, item.Title, item.Category)
	}
}
