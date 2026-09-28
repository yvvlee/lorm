package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func SearchHandler(repo *ProductRepo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /products/search", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request SearchRequest
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid search parameters: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "request body must contain exactly one JSON object", http.StatusBadRequest)
			return
		}
		if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := repo.SearchPage(r.Context(), request.Query, request.Pagination)
		if err != nil {
			log.Printf("search products: %v", err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("encode search response: %v", err)
		}
	})
	return mux
}
