package main

import (
	"context"
	"fmt"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
	"github.com/yvvlee/lorm/builder/try"
)

// Query defines supported filters. A nil pointer omits the condition.
// Pointers to empty strings, zero, and false still apply the condition.
type Query struct {
	Name     *string `json:"name"`
	Category *string `json:"category"`
	MinPrice *int64  `json:"min_price"`
	MaxPrice *int64  `json:"max_price"`
	Active   *bool   `json:"active"`
}

type Pagination struct {
	Page uint64 `json:"page"`
	Size uint64 `json:"size"`
}

type SearchRequest struct {
	Query
	Pagination
}

func (r SearchRequest) Validate() error {
	if r.Page == 0 || r.Size == 0 || r.Size > 100 {
		return fmt.Errorf("page must be greater than 0 and size must be between 1 and 100")
	}
	// Keep the offset within the database signed integer range.
	if r.Page-1 > ((1<<63)-1)/r.Size {
		return fmt.Errorf("pagination offset is too large")
	}
	if (r.MinPrice != nil && *r.MinPrice < 0) || (r.MaxPrice != nil && *r.MaxPrice < 0) {
		return fmt.Errorf("prices must not be negative")
	}
	if r.MinPrice != nil && r.MaxPrice != nil && *r.MinPrice > *r.MaxPrice {
		return fmt.Errorf("min_price must not exceed max_price")
	}
	return nil
}

type PageResult struct {
	Items []*Product `json:"items"`
	Pagination
	Total      uint64 `json:"total"`
	TotalPages uint64 `json:"total_pages"`
}

type ProductRepo struct {
	*lorm.Repository[*Product]
}

func NewProductRepo(engine *lorm.Engine) *ProductRepo {
	return &ProductRepo{Repository: engine.Repository[*Product]()}
}

// SearchPage returns matching rows and pagination metadata.
func (r *ProductRepo) SearchPage(ctx context.Context, query Query, pagination Pagination) (*PageResult, error) {
	if err := (SearchRequest{Query: query, Pagination: pagination}).Validate(); err != nil {
		return nil, err
	}
	var p Product
	cols := p.LormCols()
	// try skips nil pointers and composes with And.
	// Page shares these filters between the count and row queries.
	items, total, err := r.Engine.Query[*Product]().
		Where(builder.And{
			try.Equal(cols.Name(), query.Name),
			try.Equal(cols.Category(), query.Category),
			try.Range(cols.Price(), query.MinPrice, query.MaxPrice),
			try.Equal(cols.Active(), query.Active),
		}).
		Asc(cols.ID()).
		Page(ctx, pagination.Page, pagination.Size)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*Product{}
	}
	totalPages := total / pagination.Size
	if total%pagination.Size != 0 {
		totalPages++
	}
	return &PageResult{Items: items, Pagination: pagination, Total: total, TotalPages: totalPages}, nil
}
