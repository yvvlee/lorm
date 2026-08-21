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

	if _, err := engine.Insert[*Product]().AddModels(
		&Product{Name: "Go in Action", Category: "book", Price: 3200, Status: "active"},
		&Product{Name: "Go Sticker Pack", Category: "merch", Price: 500, Status: "active"},
		&Product{Name: "Mechanical Keyboard", Category: "tool", Price: 8900, Status: "active"},
		&Product{Name: "Archived Notebook", Category: "tool", Price: 1200, Status: "archived"},
		&Product{Name: "Go CLI Cheatsheet", Category: "book", Price: 800, Status: "active"},
	).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	var p Product
	filtered, err := engine.Query[*Product]().
		Where(builder.And{
			builder.In(p.LormCols().Category(), []string{"book", "tool"}),
			builder.Or{
				builder.Like(p.LormCols().Name(), "%Go%"),
				builder.Lt(p.LormCols().Price(), int64(3000)),
			},
			builder.Eq{p.LormCols().Status(): "active"},
		}).
		OrderBy(p.LormCols().Price() + " ASC").
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("filtered products:")
	for _, item := range filtered {
		fmt.Printf("- id=%d name=%q category=%s price=%d status=%s\n",
			item.ID, item.Name, item.Category, item.Price, item.Status)
	}

	count, ok, err := engine.Query[*Product]().
		Select("COUNT(1)").
		Where(builder.Eq{p.LormCols().Status(): "active"}).
		GetCol[int64](ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("active product count: ok=%v count=%d\n", ok, count)
}
