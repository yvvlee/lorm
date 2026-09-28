package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestSearchHTTP(t *testing.T) {
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	repo := NewProductRepo(engine)
	_, err = repo.InsertAll(context.Background(), []*Product{
		{Name: "A", Category: "book", Price: 100, Active: true},
		{Name: "B", Category: "book", Price: 200, Active: true},
		{Name: "C", Category: "book", Price: 300, Active: true},
		{Name: "D", Category: "book", Price: 0, Active: false},
		{Name: "", Category: "tool", Price: 400, Active: true},
	})
	require.NoError(t, err)
	handler := SearchHandler(repo)
	for _, tc := range []struct {
		name       string
		body       string
		names      []string
		total      uint64
		totalPages uint64
		page       uint64
		size       uint64
	}{
		{"combined second page", `{"category":"book","min_price":100,"max_price":300,"active":true,"page":2,"size":2}`, []string{"C"}, 3, 2, 2, 2},
		{"omitted conditions", `{"page":1,"size":2}`, []string{"A", "B"}, 5, 3, 1, 2},
		{"zero and false", `{"max_price":0,"active":false,"page":1,"size":10}`, []string{"D"}, 1, 1, 1, 10},
		{"empty string", `{"name":"","page":1,"size":10}`, []string{""}, 1, 1, 1, 10},
		{"name and lower bound", `{"name":"B","min_price":200,"page":1,"size":10}`, []string{"B"}, 1, 1, 1, 10},
		{"no matches", `{"category":"missing","page":1,"size":10}`, []string{}, 0, 0, 1, 10},
		{"past last page", `{"page":4,"size":2}`, []string{}, 5, 3, 4, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/products/search", strings.NewReader(tc.body)))
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var result PageResult
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			require.NotNil(t, result.Items)
			names := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				names = append(names, item.Name)
			}
			require.Equal(t, tc.names, names)
			require.Equal(t, tc.total, result.Total)
			require.Equal(t, tc.totalPages, result.TotalPages)
			require.Equal(t, Pagination{Page: tc.page, Size: tc.size}, result.Pagination)
		})
	}
	for _, body := range []string{
		`{`, `null`, `{}`, `{"page":0,"size":10}`, `{"page":1,"size":101}`,
		`{"page":18446744073709551615,"size":10}`,
		`{"page":1,"size":10,"active":"false"}`,
		`{"page":1,"size":10,"min_price":-1}`,
		`{"page":1,"size":10,"min_price":200,"max_price":100}`,
		`{"page":1,"size":10,"unknown":true}`,
		`{"page":1,"size":10} {}`,
	} {
		t.Run("invalid "+body, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/products/search", strings.NewReader(body)))
			require.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
}
