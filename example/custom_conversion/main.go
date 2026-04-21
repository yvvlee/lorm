package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/yvvlee/lorm"
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

	report := &Report{
		Title:  "quarterly-report",
		Scores: CSVInts{90, 95, 88},
	}
	if _, err := lorm.Insert[*Report](engine).AddModel(report).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	loaded, err := lorm.NewRepository[*Report](engine).Get(ctx, report.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("loaded report: id=%d title=%s scores=%v\n", loaded.ID, loaded.Title, loaded.Scores)

	if _, err := lorm.Update[*Report](engine).
		ID(report.ID).
		SetMap(map[string]any{
			"scores": CSVInts{100, 99, 98},
		}).
		Exec(ctx); err != nil {
		log.Fatal(err)
	}

	reloaded, err := lorm.NewRepository[*Report](engine).Get(ctx, report.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("updated report: id=%d title=%s scores=%v\n", reloaded.ID, reloaded.Title, reloaded.Scores)
}
