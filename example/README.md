# Examples

This directory contains runnable, self-contained examples for the most common
LORM workflows.

All examples use SQLite with a temporary database file, so you can run them
without any external database service.

## Run

```bash
go run ./quickstart
go run ./repository
go run ./transaction
go run ./custom_model
go run ./custom_conversion
go run ./json_field
go run ./pagination
go run ./optimistic_lock
go run ./query_builder
```

To regenerate the checked-in `_lorm_gen.go` files for all examples:

```bash
go generate ./...
```

## Included Examples

- `quickstart`: basic engine setup plus insert/query/update/delete
- `repository`: how to wrap `lorm.Repository[T]` and add custom query methods
- `transaction`: how to use `Engine.TX` and ensure every statement uses the transactional context
- `custom_model`: how to write explicit joins and map results into a projection model
- `custom_conversion`: how to store a custom field type using `driver.Valuer` and `sql.Scanner`
- `json_field`: how to store and update JSON fields with regular Go structs and slices
- `pagination`: how to use `Page(ctx, page, size)` and work with total counts
- `optimistic_lock`: how `version` fields participate in safe concurrent updates
- `query_builder`: how to combine `builder.And`, `builder.Or`, `In`, `Like`, and scalar `Select` queries

## Notes

- The examples intentionally avoid automatic relation loading. LORM expects join
  queries to stay explicit so SQL shape and cost remain visible.
- Generated model helper files are checked in so the examples compile directly
  after clone.
