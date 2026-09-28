# List Search and Pagination

The handler parses search and pagination parameters into `Query` and `Pagination`. The repository returns matching rows and pagination metadata.

- `try` replaces per-filter `if` checks and composes with other builders.
- `Page` shares one WHERE definition between the row and count queries.

## Filters: if vs. try

All filter fields in `Query` are pointers. `nil` omits a filter; pointers to `0`, `false`, or an empty string still apply it. Consider category and active status:

With explicit checks, each filter needs an `if`:

```go
func applyFilters(stmt *lorm.SelectStmt[*Product], query Query) *lorm.SelectStmt[*Product] {
    var p Product
    cols := p.LormCols()
    if query.Category != nil {
        stmt = stmt.Where(builder.Eq{cols.Category(): *query.Category})
    }
    if query.Active != nil {
        stmt = stmt.Where(builder.Eq{cols.Active(): *query.Active})
    }
    return stmt
}
```

`try` converts pointers into conditions. Builders ignore empty conditions:

```go
stmt := r.Engine.Query[*Product]().Where(builder.And{
    try.Equal(cols.Category(), query.Category),
    try.Equal(cols.Active(), query.Active),
})
```

Conditions can also be nested. For example, match name or category within a price range:

```go
builder.And{
    builder.Or{
        try.Equal(cols.Name(), query.Name),
        try.Equal(cols.Category(), query.Category),
    },
    try.Range(cols.Price(), query.MinPrice, query.MaxPrice),
}
```

If both name and category are omitted, the entire `Or` is ignored. The runnable example uses `And` to match all supplied filters.

## Pagination: repeated filters vs. Page

Separate row and count queries each need the same filters, even with an `applyFilters` helper:

```go
items, err := applyFilters(r.Engine.Query[*Product](), query).
    Asc(cols.ID()).
    Limit(pagination.Size).
    Offset((pagination.Page - 1) * pagination.Size).
    Find(ctx)
if err != nil {
    return nil, err
}

total, err := applyFilters(r.Engine.Query[*Product](), query).Count(ctx)
if err != nil {
    return nil, err
}
```

`Page` returns both rows and the total from the `stmt` built above:

```go
items, total, err := stmt.
    Asc(cols.ID()).
    Page(ctx, pagination.Page, pagination.Size)
if err != nil {
    return nil, err
}
```

`Page` queries the count first, then the requested rows. Both SQL statements reuse the same filters. The row query is skipped when the count is zero or the page is out of range.

## Run

From the `example` directory:

```bash
CGO_ENABLED=1 go run ./list_search
CGO_ENABLED=1 go test ./list_search
```

The example uses a temporary SQLite database and executes HTTP requests in process with `httptest`. No frontend or external service is needed.

## Request and response

`POST /products/search` accepts JSON. Names use exact matching; prices are in cents.

```json
{
  "category": "book",
  "min_price": 3000,
  "max_price": 6000,
  "active": true,
  "page": 2,
  "size": 2
}
```

```json
{
  "items": [
    {"id": 3, "name": "Advanced Go", "category": "book", "price": 5600, "active": true}
  ],
  "page": 2,
  "size": 2,
  "total": 3,
  "total_pages": 2
}
```

Omitted or `null` filters are ignored. Both `page` (greater than 0) and `size` (1–100) are required. Invalid parameters return HTTP 400. Results are ordered by unique ID. Empty pages return `items: []`; `total` remains the count of all matching rows.

See [repo.go](repo.go) for the query and [handler.go](handler.go) for request handling.
