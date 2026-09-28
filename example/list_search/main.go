package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

//go:embed schema.sql
var schemaSQL string

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		return err
	}
	defer cleanup()
	repo := NewProductRepo(engine)
	if _, err := repo.InsertAll(context.Background(), []*Product{
		{Name: "Getting Started with Go", Category: "book", Price: 3200, Active: true},
		{Name: "Go in Practice", Category: "book", Price: 4800, Active: true},
		{Name: "Advanced Go", Category: "book", Price: 5600, Active: true},
		{Name: "Free Handbook", Category: "book", Price: 0, Active: false},
		{Name: "Keyboard", Category: "tool", Price: 8900, Active: true},
	}); err != nil {
		return err
	}
	// Exercise request parsing, repository queries, and responses in process.
	handler := SearchHandler(repo)
	for _, body := range []string{
		`{"category":"book","min_price":3000,"max_price":6000,"active":true,"page":2,"size":2}`,
		`{"max_price":0,"active":false,"page":1,"size":10}`,
		`{"page":1,"size":2}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/products/search", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			return fmt.Errorf("search returned %d: %s", response.Code, response.Body.String())
		}
		fmt.Printf("POST /products/search %s\n%s\n", body, response.Body.String())
	}
	return nil
}
