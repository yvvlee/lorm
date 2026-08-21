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
Its fluent API is inspired by
[Squirrel](https://github.com/Masterminds/squirrel), while LORM keeps the
builder scoped to repository implementations and its own generated model
metadata.

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

Recommended `database/sql` drivers:

| Database | Driver package | Driver name for `NewEngine` |
| --- | --- | --- |
| MySQL/MariaDB | `github.com/go-sql-driver/mysql` | `mysql` |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | `pgx` |
| SQLite | `github.com/mattn/go-sqlite3` | `sqlite3` |

LORM does not import database drivers from the root module. Import only the
drivers your application uses:

```go
import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)
```

## Installation

LORM requires Go 1.27 or later.

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
`Fields()`, `New()`, `LormFieldPtr()`, `LormFieldValue()`, and
`LormModelDescriptor()`, plus the write hooks for table models.

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
_, err = engine.Insert[*User]().
	AddModel(user).
	Exec(ctx)

// Query
savedUser, ok, err := engine.Query[*User]().
	Where(builder.Eq{u.Fields().ID(): user.ID}).
	Get(ctx)

// Update
_, err = engine.Update[*User]().
	ID(user.ID).
	SetMap(map[string]any{
		u.Fields().Name(): "Jane Doe",
	}).
	Exec(ctx)

// Delete
_, err = engine.Delete[*User]().
	ID(user.ID).
	Exec(ctx)
```

> **Note**: Code generation is required before using model-based APIs.

> **Update note**: `Update.SetModel(model)` performs a full-field update. Zero
> values are written as well, so prefer `SetMap` / `Set` for partial updates.
> `SetModel` cannot be mixed with `Set` or `SetMap`.

> **Write safety note**: `Update.Exec` and `Delete.Exec` make a best-effort
> attempt to reject statements without a restrictive `WHERE` clause. The check
> cannot identify every logically tautological condition and does not replace
> application-level validation. Do not treat it as an absolute guarantee
> against full-table writes. Raw SQL executed through `Engine.Exec` is outside
> this check. Call `AllowGlobalWrite()` explicitly for an intentional
> full-table operation; callers remain responsible for verifying the write
> scope and protecting their data.

> **Where note**: `builder.Eq{field: value}` always renders `field = ?` and
> binds `value` as a single argument. It does not special-case `nil` into
> `IS NULL`, dereference pointers, call `driver.Valuer`, or expand slices into
> `IN (...)`. Use `builder.IsNull(field)` / `builder.IsNotNull(field)` for null
> checks, and `builder.In` / `builder.NotIn` for membership predicates.

> **Insert note**: single-row inserts backfill generated IDs when the driver
> supports `RETURNING` or `LastInsertId`. Batch inserts do not backfill IDs by
> default. Call `RequireIDBackfill()` to execute the batch one row at a time in
> a transaction and backfill each inserted model. A zero auto-increment primary
> key is omitted so the database can generate it. A non-zero value is inserted
> explicitly. Mixed batches are split into consecutive groups in one transaction.

> **Get note**: `Query.Get` returns `(T, bool, error)`. The boolean reports
> whether a row was found. Repository `Get` helpers retain `(T, error)` and
> return the zero value of `T` when no row matches.

The type argument of `Query` must be a model pointer. Declare the result value
type on the terminal method when selecting one column:

```go
ids, err := engine.Query[*User]().
	Select(u.Fields().ID()).
	FindCols[int64](ctx)

count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[uint64](ctx)

ids, total, err := engine.Query[*User]().
	Select(u.Fields().ID()).
	OrderBy(u.Fields().ID()).
	PageCols[int64](ctx, page, size)
```

Single-column terminal methods require exactly one result column and return an
error for multi-column results.

Statement builders are cheap to create. Build a fresh `Query` / `Insert` /
`Update` / `Delete` chain for each operation, and do not share the same
statement across goroutines.
Use `Clone()` when one operation needs to branch from an existing statement.
Terminal methods still reset only the statement they are called on.

## Examples

See [example/README.md](example/README.md) for runnable, self-contained examples.

The examples live in their own module:

- `cd example && go run ./quickstart`
- `cd example && go run ./repository`
- `cd example && go run ./transaction`
- `cd example && go run ./custom_model`
- `cd example && go run ./custom_conversion`
- `cd example && go run ./json_field`
- `cd example && go run ./pagination`
- `cd example && go run ./optimistic_lock`
- `cd example && go run ./query_builder`

## Transactions

`Engine.TX` starts a transaction, passes a transactional `context.Context` into
the callback, and automatically commits or rolls back.

```go
err := engine.TX(context.Background(), func(ctx context.Context) error {
	_, err := engine.Insert[*User]().
		AddModel(&User{Name: "User 1", Email: "user1@example.com"}).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = engine.Insert[*User]().
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
		Repository: engine.Repository[*User](),
	}
}

func (r *UserRepositoryImpl) PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error) {
	var u User
	return r.Engine.Query[*User]().
		Where(builder.Like(u.Fields().Email(), "%@gmail.com")).
		OrderBy(u.Fields().ID() + " DESC").
		Page(ctx, page, size)
}
```

> **Note**: `Lock` and `LockByField` append `FOR UPDATE` and require the context
> passed to `Engine.TX(...)` or `Engine.TXWithOptions(...)`. Calls outside a
> transaction return an error.

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

roles, err := engine.Query[*UserRole]().
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

var _ lorm.ScannerValuer = (*CSVInts)(nil)

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

`ScannerValuer` combines these two standard interfaces. The compile-time
assertion above is optional, but it catches an incomplete implementation before
the program runs. LORM does not require a separate conversion protocol.

See [example/custom_conversion/main.go](example/custom_conversion/main.go) for a
runnable example.

## Configuration

```go
dialect := lorm.DefaultDialectConfig("pgx")
dialect.SupportsForUpdate = true

engine, err := lorm.NewEngine(
	"pgx",
	"postgres://user:password@localhost:5432/dbname?sslmode=disable",
	lorm.WithDialectConfig(dialect),
	lorm.WithMaxIdleConns(10),
	lorm.WithMaxOpenConns(100),
	lorm.WithConnMaxLifetime(time.Hour),
	lorm.WithConnMaxIdleTime(30*time.Minute),
	lorm.WithLogger(customLogger),
)
```

`DialectConfig` stores driver-specific SQL behavior in one place:
placeholder style, identifier quoting, `RETURNING`, `LastInsertId`,
`FOR UPDATE`, and `INSERT IGNORE` syntax. Built-in defaults are selected from
the driver name. Use `WithDialectConfig` to replace the whole dialect config,
or `WithPlaceholderFormat`, `WithEscaper`, and `WithSupports...` helpers to
override one field.

## Benchmarks

The benchmark suite lives in [benchmarks/orm-crud](benchmarks/orm-crud).

Results below were captured on August 20, 2026 with Go 1.27.0. Each ORM
was run once in a separate `go test` process.

SQLite was run once per ORM:

```bash
for orm in lorm gorm xorm ent; do
  CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -count=1
done
```

Before every MySQL run, the MySQL container was restarted. The benchmark
waited for `mysqladmin ping` to succeed, then waited another 10 seconds:

```bash
docker restart mysql
# Wait for mysqladmin ping to succeed.
sleep 10
ORMCRUD_DB=mysql go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

Before every PostgreSQL run, its container was restarted and the benchmark
waited for `pg_isready` to succeed:

```bash
docker restart postgres
ORMCRUD_DB=postgres go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

Replace `<orm>` with `lorm`, `gorm`, `xorm`, or `ent`. Each ORM uses
separate benchmark databases. The complete bounded readiness command is in the
benchmark suite README.

Environment:

- OS/arch: `darwin/arm64`
- CPU: `Apple M1 Pro`
- Go: `go1.27.0`
- MySQL: `9.7.0`
- PostgreSQL: `18.6`
- `sonic/ast` fell back to `encoding/json` under Go 1.27.

Ranks compare all four ORMs; lower is better. `Gap to best` is calculated as
`(lorm / best - 1) * 100%`.

SQLite, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 320,185 | 279,823 | 291,430 | **270,549** | 4 | 18.35% |
| ReadByID | **21,189** | 24,540 | 30,467 | 26,395 | 1 | 0.00% |
| ReadByIDComplex | **22,487** | 26,886 | 32,928 | 29,012 | 1 | 0.00% |
| UpdateByID | 342,819 | 346,223 | 324,730 | **288,283** | 3 | 18.92% |
| DeleteByID | 344,690 | 310,801 | 336,944 | **273,470** | 4 | 26.04% |
| BatchCreate100 | **1,198,248** | 1,326,222 | 3,226,068 | 1,510,362 | 1 | 0.00% |
| BatchRead100 | **521,933** | 759,109 | 894,268 | 583,942 | 1 | 0.00% |
| BatchRead100Complex | **761,215** | 992,272 | 1,140,149 | 841,553 | 1 | 0.00% |
| BatchUpdate100 | 398,032 | 450,038 | **374,384** | 401,475 | 2 | 6.32% |
| BatchDelete100 | 690,346 | 765,703 | **591,269** | 595,752 | 3 | 16.76% |

MySQL, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **906,321** | 1,361,265 | 973,385 | 928,810 | 1 | 0.00% |
| ReadByID | **240,317** | 302,950 | 284,257 | 249,565 | 1 | 0.00% |
| ReadByIDComplex | **245,672** | 298,108 | 305,488 | 248,823 | 1 | 0.00% |
| UpdateByID | **879,429** | 1,410,948 | 879,682 | 1,440,935 | 1 | 0.00% |
| DeleteByID | 822,350 | 1,031,374 | 871,111 | **751,133** | 2 | 9.48% |
| BatchCreate100 | 7,846,680 | 8,017,562 | **7,364,879** | 8,164,799 | 2 | 6.54% |
| BatchRead100 | 2,297,721 | 3,442,311 | 1,955,529 | **1,543,769** | 3 | 48.84% |
| BatchRead100Complex | 3,716,681 | 5,032,195 | **2,309,288** | 3,182,881 | 3 | 60.94% |
| BatchUpdate100 | 5,398,099 | 5,773,743 | **4,933,102** | 6,097,586 | 2 | 9.43% |
| BatchDelete100 | **4,047,424** | 4,165,851 | 4,114,345 | 4,097,822 | 1 | 0.00% |

PostgreSQL, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **882,606** | 1,245,377 | 902,820 | 916,939 | 1 | 0.00% |
| ReadByID | 135,480 | **134,977** | 143,964 | 141,048 | 2 | 0.37% |
| ReadByIDComplex | **136,348** | 138,018 | 141,529 | 140,041 | 1 | 0.00% |
| UpdateByID | **959,347** | 1,063,873 | 1,250,379 | 1,339,732 | 1 | 0.00% |
| DeleteByID | 945,561 | 1,167,307 | 859,569 | **856,562** | 3 | 10.39% |
| BatchCreate100 | **7,620,672** | 7,793,676 | 8,074,182 | 8,180,815 | 1 | 0.00% |
| BatchRead100 | **648,760** | 907,110 | 1,115,124 | 759,521 | 1 | 0.00% |
| BatchRead100Complex | **937,986** | 1,268,048 | 1,491,468 | 1,065,715 | 1 | 0.00% |
| BatchUpdate100 | 1,325,303 | 1,463,390 | 1,505,688 | **1,323,338** | 2 | 0.15% |
| BatchDelete100 | 1,288,657 | 1,467,285 | 1,238,411 | **1,216,635** | 3 | 5.92% |

Notes from this run:

- On SQLite, `lorm` is fastest in 5 of the 10 `ns/op` cases.
- On MySQL, `lorm` is fastest in 5 of the 10 `ns/op` cases.
- On PostgreSQL, `lorm` is fastest in 6 of the 10 `ns/op` cases.
- `lorm` has the lowest `B/op` in 9 of the 10 SQLite cases, 9 of the 10
  MySQL cases, and 9 of the 10 PostgreSQL cases. The `allocs/op`
  distribution is also 9, 9, and 9 out of 10, respectively.

Treat these numbers as directional rather than universal. Re-run the suite on
your target database, schema, driver, and hardware before making a decision.

See [benchmarks/orm-crud/README.md](benchmarks/orm-crud/README.md)
for the benchmark scope, setup, and full `ns/op`, `B/op`, and `allocs/op`
tables.

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
- `updated`: fills a zero value on insert and always refreshes on model update
- `version`: enables optimistic-lock style version increments on update

Generator behavior worth knowing:

- Table names default to snake_case and can be overridden with a tag on the
  embedded `lorm.UnimplementedTable`.
- Field names default to snake_case and can be overridden with field tags.
- If a field needs a `lorm` tag, declare it on its own line instead of grouped
  declarations such as `A, B int`.
- Embedded structs are flattened into the generated field accessors.
- Tags on embedded structs can prepend a prefix to the flattened field names.
- The generator rejects unknown tag items and duplicate database column names,
  including conflicts introduced by flattening embedded structs.
- An `auto_increment` field must also be marked as `primary_key`.
- Each model may have at most one `version` field.
- `created` and `updated` support only `time.Time`, `sql.NullTime`, `int64`,
  `uint64`, `uint32`, `uint`, `int` on 64-bit targets, `string`, and one pointer
  level to those types. Integers store Unix seconds and strings use
  `time.DateTime`.
- `int8`, `uint8`, `int16`, `uint16`, and `int32` are rejected for managed time
  fields. Generated files with an `int` managed time field reject 32-bit builds.

## Contributing

Contributions are welcome. Feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for
details.
