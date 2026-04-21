package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/yvvlee/lorm"
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

	if _, err := lorm.Insert[*Product](engine).AddModels(
		&Product{Name: "Go in Action", Category: "book", Price: 3200, Status: "active"},
		&Product{Name: "Go Sticker Pack", Category: "merch", Price: 500, Status: "active"},
		&Product{Name: "Mechanical Keyboard", Category: "tool", Price: 8900, Status: "active"},
		&Product{Name: "Archived Notebook", Category: "tool", Price: 1200, Status: "archived"},
		&Product{Name: "Go CLI Cheatsheet", Category: "book", Price: 800, Status: "active"},
	).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	var p Product
	filtered, err := lorm.Query[*Product](engine).
		Where(builder.And{
			builder.In(p.Fields().Category(), []string{"book", "tool"}),
			builder.Or{
				builder.Like(p.Fields().Name(), "%Go%"),
				builder.Lt(p.Fields().Price(), int64(3000)),
			},
			builder.Eq{p.Fields().Status(): "active"},
		}).
		OrderBy(p.Fields().Price() + " ASC").
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("filtered products:")
	for _, item := range filtered {
		fmt.Printf("- id=%d name=%q category=%s price=%d status=%s\n",
			item.ID, item.Name, item.Category, item.Price, item.Status)
	}

	count, ok, err := lorm.QueryCol[int64](engine).
		Select("COUNT(1)").
		From(p.TableName()).
		Where(builder.Eq{p.Fields().Status(): "active"}).
		Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("active product count: ok=%v count=%d\n", ok, count)
}
