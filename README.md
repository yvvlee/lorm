# LORM - Lightweight ORM for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/yvvlee/lorm)](https://goreportcard.com/report/github.com/yvvlee/lorm)
[![Go Reference](https://pkg.go.dev/badge/github.com/yvvlee/lorm.svg)](https://pkg.go.dev/github.com/yvvlee/lorm)
[![Build Status](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml/badge.svg)](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[中文](README_ZH.md)

LORM is a lightweight ORM for Go that keeps the API small, favors explicit SQL,
and uses code generation to provide model metadata and typed field accessors.

## Table of Contents

- [Why LORM](#why-lorm)
- [Design Philosophy](#design-philosophy)
- [Database Support](#database-support)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Examples](#examples)
- [Transactions](#transactions)
- [Repository Helper](#repository-helper)
- [Custom Projection Models](#custom-projection-models)
- [Custom Field Conversion](#custom-field-conversion)
- [Configuration](#configuration)
- [Benchmarks](#benchmarks)
- [lormgen](#lormgen)
- [Contributing](#contributing)
- [License](#license)

## Why LORM

- **Repository-first data access** for both simple CRUD and complex queries.
- **Flexible SQL builder inside repository implementations** for queries that still need explicit control.
- **Code-generated model metadata** instead of runtime reflection-heavy mapping.
- **Typed field accessors** that make column names easier to reuse safely.
- **No automatic relation loading**, implicit joins, or hidden query fan-out.
- **Transaction helper** that automatically reuses the transactional session from `context.Context`.
- **Structured logging** and configurable placeholder/identifier handling.

## Design Philosophy

LORM keeps data access behind repository interfaces.

Use `Repository[T]` for simple CRUD and for most single-table business flows.
It keeps application code short, stable, and easy to test.

Keep complex reads, reports, search pages, and custom joins inside repository
implementations as well. The SQL builder supports repository code and keeps the
final query shape explicit and reviewable.

LORM intentionally does not provide automatic relation loading, implicit eager
loading, lazy loading, or magic model association queries. In production
systems, those features are easy to lose control over: they can hide query
costs, make SQL shape unpredictable, introduce accidental N+1 patterns, and
turn simple code changes into performance regressions.

LORM prefers explicit joins, explicit selected columns, and explicit query
boundaries. Business code should stay focused on business intent, while
repository implementations own database details.

## Database Support

- **First-class**: MySQL/MariaDB, PostgreSQL
- **Secondary**: SQLite

## Installation

```bash
go get github.com/yvvlee/lorm
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

## Quick Start

### 1. Define a model

```go
type User struct {
	lorm.UnimplementedTable
	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Email     string    `lorm:"email"`
	CreatedAt time.Time `lorm:"created_at,created"`
	UpdatedAt time.Time `lorm:"updated_at,updated"`
}
```

### 2. Generate the helper code

```bash
lormgen ./...
```

This generates `_lorm_gen.go` files with methods such as `TableName()`,
`Fields()`, `New()`, `LormFieldPtr()`, and `LormModelDescriptor()`.

### 3. Open an engine

```go
engine, err := lorm.NewEngine(
	"mysql",
	"user:password@tcp(localhost:3306)/dbname?parseTime=true",
)
if err != nil {
	log.Fatal(err)
}
defer engine.Close()
```

### 4. Run CRUD operations

```go
ctx := context.Background()
var u User

user := &User{
	Name:  "John Doe",
	Email: "john@example.com",
}

// Insert
_, err = lorm.Insert[*User](engine).
	AddModel(user).
	Exec(ctx)

// Query
savedUser, err := lorm.Query[*User](engine).
	Where(builder.Eq{u.Fields().ID(): user.ID}).
	Get(ctx)

// Update
_, err = lorm.Update[*User](engine).
	ID(user.ID).
	SetMap(map[string]any{
		u.Fields().Name(): "Jane Doe",
	}).
	Exec(ctx)

// Delete
_, err = lorm.DeleteModel[*User](engine).
	ID(user.ID).
	Exec(ctx)
```

> **Note**: Code generation is required before using model-based APIs.

> **Update note**: `Update.SetModel(model)` performs a full-field update. Zero
> values are written as well, so prefer `SetMap` / `Set` for partial updates.

> **Where note**: `builder.Eq` does not expand slices into `IN (...)`. Use
> `builder.In` / `builder.NotIn` explicitly for membership predicates.

> **Insert note**: batch inserts only backfill generated IDs when the driver
> returns one generated value per inserted row. `LastInsertId`-only dialects do
> not infer per-row IDs for multi-row inserts.

Statement builders are cheap to create. Build a fresh `Query` / `Insert` /
`Update` / `Delete` chain for each operation, and do not share the same
statement across goroutines.
Use `Clone()` when one operation needs to branch from an existing statement.
Terminal methods still reset only the statement they are called on.

## Examples

See [example/README.md](example/README.md) for runnable, self-contained examples.

- `go run ./example/quickstart`
- `go run ./example/repository`
- `go run ./example/transaction`
- `go run ./example/custom_model`
- `go run ./example/custom_conversion`
- `go run ./example/json_field`
- `go run ./example/pagination`
- `go run ./example/optimistic_lock`
- `go run ./example/query_builder`

## Transactions

`Engine.TX` starts a transaction, passes a transactional `context.Context` into
the callback, and automatically commits or rolls back.

```go
err := engine.TX(context.Background(), func(ctx context.Context) error {
	_, err := lorm.Insert[*User](engine).
		AddModel(&User{Name: "User 1", Email: "user1@example.com"}).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = lorm.Insert[*User](engine).
		AddModel(&User{Name: "User 2", Email: "user2@example.com"}).
		Exec(ctx)
	return err
})
```

Nested `TX` calls reuse the existing transactional session carried by the
incoming context instead of opening a second transaction.

Use `Engine.TXWithOptions` when you need to pass `sql.TxOptions` such as
isolation level or read-only mode. Nested calls still reuse the current
transaction from context.

## Repository Helper

`lorm.Repository[T]` wraps the common single-table CRUD paths. It is highly
recommended to embed `lorm.Repository[T]` within an implementation struct and
selectively expose methods through an interface.

Repository is the recommended boundary for all database access:

- It gives business code a stable, ORM-agnostic interface.
- It keeps method signatures independent from transaction management.
- It keeps SQL builder usage inside repository implementations.
- It makes testing easier by mocking repository interfaces instead of database calls.
- It keeps table structure, joins, and query details out of business code.

Simple CRUD can reuse the built-in repository methods directly. Complex queries
should still live in repository implementations, using the SQL builder
internally when needed:

```go
type UserRepository interface {
	// Common methods implemented by lorm.Repository[*User], expose as needed
	Get(ctx context.Context, id any) (*User, error)
	GetByField(ctx context.Context, field string, value any) (*User, error)
	Lock(ctx context.Context, id any) (*User, error)
	LockByField(ctx context.Context, field string, value any) (*User, error)
	Exist(ctx context.Context, id any) (bool, error)
	ExistByField(ctx context.Context, field string, value any) (bool, error)
	Update(ctx context.Context, user *User) (rowsAffected int64, err error)
	UpdateMap(ctx context.Context, id any, data map[string]any) (rowsAffected int64, err error)
	Insert(ctx context.Context, user *User) (rowsAffected int64, err error)
	InsertAll(ctx context.Context, users []*User) (rowsAffected int64, err error)
	Delete(ctx context.Context, id any) (rowsAffected int64, err error)
	DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error)

	// Custom methods to be implemented in UserRepositoryImpl
	PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error)
}

var _ UserRepository = (*UserRepositoryImpl)(nil)

type UserRepositoryImpl struct {
	*lorm.Repository[*User]
}

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: lorm.NewRepository[*User](engine),
	}
}

func (r *UserRepositoryImpl) PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error) {
	var u User
	return lorm.Query[*User](r.Engine).
		Where(builder.Like(u.Fields().Email(), "%@gmail.com")).
		OrderBy(u.Fields().ID() + " DESC").
		Page(ctx, page, size)
}
```

> **Note**: `Lock` and `LockByField` append `FOR UPDATE`. They only have
> practical locking effect inside `Engine.TX(...)` or
> `Engine.TXWithOptions(...)`. Outside a transaction the database will not keep
> the row lock beyond the statement itself.

## Custom Projection Models

When query results do not map one-to-one to a table model, embed
`lorm.UnimplementedModel` instead of `lorm.UnimplementedTable`.

```go
type UserRole struct {
	lorm.UnimplementedModel
	UserID   int64
	UserName string
	RoleName string
}

roles, err := lorm.Query[*UserRole](engine).
	Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"r.name AS role_name",
	).
	From("user AS u").
	InnerJoin("role AS r ON u.role_id = r.id").
	Find(ctx)
```

Unlike `UnimplementedTable`, `UnimplementedModel` does not generate a
`TableName()` method, so you must specify `From(...)` yourself.

## Custom Field Conversion

For fields that should not be stored as plain values or JSON, implement the
standard database interfaces on the field type itself:

- `driver.Valuer` for writes
- `sql.Scanner` for reads

```go
import "database/sql/driver"

type CSVInts []int

func (c CSVInts) Value() (driver.Value, error) {
	return []byte("1,2,3"), nil
}

func (c *CSVInts) Scan(src any) error {
	// decode "1,2,3" back into the slice
	return nil
}

type Report struct {
	lorm.UnimplementedTable
	ID     int64   `lorm:"id,primary_key,auto_increment"`
	Title  string  `lorm:"title"`
	Scores CSVInts `lorm:"scores"`
}
```

LORM passes query arguments through `driver.Valuer`, and `database/sql` uses
`sql.Scanner` when scanning result columns back into the field.

See [example/custom_conversion/main.go](example/custom_conversion/main.go) for a
runnable example.

## Configuration

```go
engine, err := lorm.NewEngine(
	"pgx",
	"postgres://user:password@localhost:5432/dbname?sslmode=disable",
	lorm.WithPlaceholderFormat(builder.Dollar),
	lorm.WithEscaper(names.NewQuoter('"', '"')),
	lorm.WithMaxIdleConns(10),
	lorm.WithMaxOpenConns(100),
	lorm.WithConnMaxLifetime(time.Hour),
	lorm.WithConnMaxIdleTime(30*time.Minute),
	lorm.WithLogger(customLogger),
)
```

## Benchmarks

The benchmark suite lives in [benchmarks/orm-crud](benchmarks/orm-crud).

Results below were captured on April 20, 2026 with:

```bash
cd benchmarks/orm-crud
go test -bench . -benchmem
```

Environment:

- Backend: SQLite (default, `ORMCRUD_DB=sqlite`)
- OS/arch: `linux/amd64`
- CPU: `Intel(R) Core(TM) i5-9400F CPU @ 2.90GHz`

Single-row CRUD, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | **97,471** | 127,067 | 99,391 | 107,457 |
| ReadByID | **31,231** | 36,679 | 45,174 | 35,275 |
| UpdateByID | **85,727** | 104,658 | 93,461 | 123,181 |
| DeleteByID | 79,837 | 96,848 | 84,681 | **78,846** |

Batch CRUD with 100 rows, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| BatchCreate100 | 1,139,752 | **724,101** | 4,025,133 | 853,383 |
| BatchRead100 | **346,392** | 427,952 | 725,096 | 411,414 |
| BatchUpdate100 | 211,472 | 211,800 | **185,189** | 186,621 |
| BatchDelete100 | 332,062 | 326,834 | 313,968 | **313,609** |

Notes from this run:

- `lorm` leads single-row create, read, and update, and is effectively tied
  with `ent` on single-row delete.
- `lorm` is fastest for batch reads and dramatically faster than `xorm` on
  batch create.
- `gorm` is fastest in batch create on this machine.
- `lorm` has the lowest `B/op` in 7 of the 8 benchmark cases in this run.

Treat these numbers as directional rather than universal. Re-run the suite on
your target database, schema, driver, and hardware before making a decision.

The suite can also run against MySQL or PostgreSQL:

```bash
ORMCRUD_DB=mysql go test -bench . -benchmem
ORMCRUD_DB=postgres go test -bench . -benchmem
```

See [benchmarks/orm-crud/README.md](benchmarks/orm-crud/README.md)
for the benchmark scope, setup, and full result tables.

## lormgen

`lormgen` scans Go files that embed `lorm.UnimplementedTable` or
`lorm.UnimplementedModel` and generates the model helper methods that LORM
needs.

Usage:

```bash
lormgen [flags] <directory|file>...
```

Common flags:

- `--field-mapper`: `snake`, `camel`, or `same`
- `--table-mapper`: `snake`, `camel`, or `same`
- `--table-prefix`: prefix added to generated table names
- `--table-suffix`: suffix added to generated table names
- `--tag-key`: struct tag key, default `lorm`
- `--file-suffix`: generated file suffix, default `_lorm_gen`
- `--ignore`: glob pattern for ignored files, repeatable

Examples:

```bash
lormgen .
lormgen ./models/...
lormgen --table-prefix=t_ --table-suffix=_tab --field-mapper=camel ./models
lormgen --ignore="*_temp.go" --ignore="*_old.go" ./models
```

Built-in tags:

- `primary_key`: marks a primary key field
- `auto_increment`: marks an auto-increment field
- `json`: stores the field as JSON
- `created`: fills the field on insert when it is zero-valued
- `updated`: fills the field on insert/update when it is zero-valued
- `version`: enables optimistic-lock style version increments on update

Generator behavior worth knowing:

- Table names default to snake_case and can be overridden with a tag on the
  embedded `lorm.UnimplementedTable`.
- Field names default to snake_case and can be overridden with field tags.
- If a field needs a `lorm` tag, declare it on its own line instead of grouped
  declarations such as `A, B int`.
- Embedded structs are flattened into the generated field accessors.
- Tags on embedded structs can prepend a prefix to the flattened field names.

## Contributing

Contributions are welcome. Feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for
details.
