# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests (requires a running database)
go test -v ./...

# Run tests against specific databases
DB_DRIVER=mysql DB_DSN="root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&charset=utf8mb4" go test -v ./...
DB_DRIVER=pgx DB_DSN="host=127.0.0.1 user=postgres password=123456 dbname=test port=5432 sslmode=disable" go test -v ./...
DB_DRIVER=sqlite3 DB_DSN="file:test.db?cache=shared&mode=memory" CGO_ENABLED=1 go test -v ./...

# Run a single test
DB_DRIVER=sqlite3 DB_DSN="file:test.db?cache=shared&mode=memory" CGO_ENABLED=1 go test -v -run TestName ./...

# Build
go build ./...

# Install the code generator
go install github.com/yvvlee/lorm/cmd/lormgen@latest

# Run lormgen on model files
lormgen ./...

# Run isolated ORM benchmark suite
cd benchmarks/orm-crud
go test -run '^$' -bench . -benchmem

# Run benchmarks against a specific backend
ORMCRUD_DB=sqlite go test -run '^$' -bench . -benchmem
ORMCRUD_DB=mysql go test -run '^$' -bench . -benchmem
ORMCRUD_DB=postgres go test -run '^$' -bench . -benchmem
```

## Architecture

LORM is a lightweight, type-safe ORM for Go built on `database/sql`. It uses code generation to avoid runtime reflection for model field access.

### Core flow

```
User code
  → Query/Insert/Update/DeleteModel[T]() — typed entry points in select.go, insert.go, update.go, delete.go
  → Delete() — untyped delete builder for custom table/condition deletes
  → builder/* — SQL string construction (SelectBuilder, InsertBuilder, etc.)
  → Engine (lorm.go) — holds the *sql.DB, resolves transactions from context
  → database/sql
```

### Key components

- **`lorm.go`** — `Engine` struct: connection management, transaction dispatch via `TX()`, placeholder format, and SQL escaper configuration.
- **`descriptor.go`** — `ModelDescriptor`: field metadata parsed from `lorm` struct tags (`primary_key`, `auto_increment`, `created`, `updated`, `version`, `json`). Used at query execution time, not at builder construction time.
- **`repository.go`** — `Repository[T Table]`: pre-built CRUD wrapper. Intended to be embedded in user-defined repository structs.
- **`session.go`** — Transaction session stored in `context.Context`. `Engine.TX()` injects a session; all statement `Exec` calls check the context for an active session before using the plain `*sql.DB`.
- **`scanner.go`** — Maps `sql.Rows` back into model instances using `Model.LormFieldPtr(name)` for per-instance field pointers.
- **`builder/`** — Pure SQL construction, no engine dependency. Dialect differences (placeholder `?` vs `$N`, quoting style) are injected via `PlaceholderFormat` and `Escaper` interfaces.
- **`names/`** — SQL identifier quoters for different dialects (backtick, double-quote, brackets).
- **`cmd/lormgen/`** — Code generator. Scans structs embedding `lorm.UnimplementedTable` or `lorm.UnimplementedModel` and emits `*_lorm_gen.go` files with `TableName()`, `Fields()`, `LormModelDescriptor()`, `New()`, and `LormFieldPtr(name)` methods.
- **`benchmarks/orm-crud/`** — Separate benchmark submodule for comparing `lorm`, `GORM`, `XORM`, and `ent` on single-row and batch CRUD across SQLite/MySQL/PostgreSQL. Its dependencies are intentionally isolated from the root module.

### Code generation

Models must embed `lorm.UnimplementedTable` (for table models) or `lorm.UnimplementedModel` (for custom query result types). Running `lormgen` generates `*_lorm_gen.go` beside each source file. These generated files must exist before the package compiles — do not delete them without regenerating.

Generated models implement `LormFieldPtr(name string) any`. Runtime code uses that method for:

- scanning rows back into a specific model instance
- extracting insert/update values without building a per-call field map

For JSON-tagged fields, generated `LormFieldPtr` returns `lorm.NewJSONFieldWrapper(&field)`.

### Struct tags

The `lorm` tag supports two syntaxes:
- Column name only: `` `lorm:"column_name"` ``
- Flags only or mixed: `` `lorm:"primary_key,auto_increment"` `` or `` `lorm:"col_name,primary_key"` ``

Built-in flag values: `primary_key`, `auto_increment`, `json`, `created`, `updated`, `version`.

### Multi-database dialect handling

Database-specific behavior is configured on the `Engine` via options:
- `WithPlaceholderFormat` — driver-dependent placeholder format. Current defaults are:
  - MySQL/MariaDB: `builder.Question`
  - PostgreSQL and SQLite: `builder.Dollar`
  - Oracle: `builder.Colon`
  - SQL Server: `builder.AtP`
- `WithEscaper` — controls identifier quoting (backtick, double-quote, brackets)

The `INSERT IGNORE` syntax and `RETURNING` clause are handled per-driver in `insert.go`. Repository locking uses `FOR UPDATE` only on explicitly supported drivers and returns an error otherwise.

### Statement lifecycle

`Query`, `Insert`, `Update`, `DeleteModel`, and `Delete` create a fresh builder for each call. There is no `sync.Pool` or manual release path for statement builders.

Statement values are mutable and intended for a single logical operation. Create a new statement chain per query/update/insert/delete, and do not share the same statement across goroutines.
