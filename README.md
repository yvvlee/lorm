# LORM - Lightweight ORM for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/yvvlee/lorm)](https://goreportcard.com/report/github.com/yvvlee/lorm)
[![Go Reference](https://pkg.go.dev/badge/github.com/yvvlee/lorm.svg)](https://pkg.go.dev/github.com/yvvlee/lorm)
[![Build Status](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml/badge.svg)](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[中文](README_ZH.md)

LORM is a high-performance, lightweight ORM for Go that keeps the API small,
favors explicit SQL, and uses code generation to provide model metadata and
typed field accessors.

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

- **High performance with low allocation overhead**: the current [benchmarks](#benchmarks) cover SQLite, MySQL, and PostgreSQL. LORM is fastest in 23 of 30 latency cases and remains close to the leader in the other cases. It also ranks first in 27 of 30 `B/op` cases and 27 of 30 `allocs/op` cases.
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
`LormCols()`, `New()`, `LormFieldPtr()`, `LormFieldValue()`, and
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
	Where(builder.Eq{u.LormCols().ID(): user.ID}).
	Get(ctx)

// Update
_, err = engine.Update[*User]().
	ID(user.ID).
	SetMap(map[string]any{
		u.LormCols().Name(): "Jane Doe",
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

> **Column names**: When filling SQL builder predicates or clauses, prefer the
> generated `LormCols()` accessors over hand-written database column strings.
> This reuses the configured field mapping and avoids spelling mistakes. Call
> `WithAlias()` when the table has an SQL alias.

```go
var u User
c := u.LormCols()

users, err := engine.Query[*User]().
	Where(builder.Eq{c.Email(): "alice@example.com"}).
	OrderBy(c.CreatedAt() + " DESC").
	Find(ctx)
// SQL (MySQL): SELECT `id`, `name`, `email`, `created_at`, `updated_at` FROM `users` WHERE `email` = ? ORDER BY created_at DESC

_, err = engine.Update[*User]().
	ID(1).
	SetMap(map[string]any{c.Name(): "Alice Updated"}).
	Exec(ctx)
// SQL (MySQL): UPDATE `users` SET `name` = ? WHERE `id` = ?
```

```go
var u User
c := u.LormCols().WithAlias("u")

ids, err := engine.Query[*User]().
	Select(c.ID()).
	From("users AS u").
	Where(builder.Like(c.Email(), "%@example.com")).
	OrderBy(c.ID() + " DESC").
	FindCols[int64](ctx)
// SQL (MySQL): SELECT u.id FROM users AS u WHERE u.email LIKE ? ORDER BY u.id DESC
```

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
	Select(u.LormCols().ID()).
	FindCols[int64](ctx)

count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[uint64](ctx)

ids, total, err := engine.Query[*User]().
	Select(u.LormCols().ID()).
	OrderBy(u.LormCols().ID()).
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
		Where(builder.Like(u.LormCols().Email(), "%@gmail.com")).
		OrderBy(u.LormCols().ID() + " DESC").
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

The tables below were recaptured on August 21, 2026 with Go 1.27.0. Each backend
uses three rounds and a separate `go test` process for each ORM. SQLite creates a
fresh temporary database per benchmark case. MySQL and PostgreSQL containers are
restarted before every ORM run, checked for readiness, and given a 10-second
warm-up. All runs use a 1-second benchmark window. See the benchmark suite README
for the complete methodology and detailed `B/op` and `allocs/op` tables.

SQLite:

```bash
for round in 1 2 3; do
  for orm in lorm gorm xorm ent; do
    CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -benchtime=1s -count=1
  done
done
```

Before every MySQL ORM run, the MySQL container was restarted. The benchmark
waited for `mysqladmin ping` to succeed, then waited another 10 seconds:

```bash
set -e

wait_for_container() {
  local container=$1
  shift
  for attempt in {1..60}; do
    if docker exec "$container" "$@" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

for round in 1 2 3; do
  for orm in lorm gorm xorm ent; do
    docker restart mysql
    wait_for_container mysql mysqladmin ping -h 127.0.0.1 -uroot -p123456
    sleep 10
    ORMCRUD_DB=mysql CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -benchtime=1s -count=1
  done
done
```

PostgreSQL used the same loop with `postgres` and `pg_isready`:

```bash
set -e

wait_for_container() {
  local container=$1
  shift
  for attempt in {1..60}; do
    if docker exec "$container" "$@" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

for round in 1 2 3; do
  for orm in lorm gorm xorm ent; do
    docker restart postgres
    wait_for_container postgres pg_isready -U postgres -d postgres
    sleep 10
    ORMCRUD_DB=postgres CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -benchtime=1s -count=1
  done
done
```

Each ORM uses a separate benchmark database. The bounded readiness commands
are also included in the benchmark suite README.

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
| Create | **252,015** | 298,490 | 338,170 | 260,103 | 1 | 0.00% |
| ReadByID | **20,366** | 24,706 | 30,231 | 25,980 | 1 | 0.00% |
| ReadByIDComplex | **22,883** | 26,901 | 32,652 | 29,272 | 1 | 0.00% |
| UpdateByID | 278,586 | 328,557 | **277,573** | 330,246 | 2 | 0.36% |
| DeleteByID | 330,649 | 313,611 | 307,297 | **302,728** | 4 | 9.22% |
| BatchCreate100 | **1,208,006** | 1,315,249 | 2,894,899 | 1,575,225 | 1 | 0.00% |
| BatchRead100 | **523,365** | 734,031 | 898,918 | 588,354 | 1 | 0.00% |
| BatchRead100Complex | **802,839** | 996,810 | 1,141,333 | 847,751 | 1 | 0.00% |
| BatchUpdate100 | 409,773 | 446,832 | **390,940** | 395,562 | 3 | 4.82% |
| BatchDelete100 | 603,935 | 673,145 | **579,423** | 627,659 | 2 | 4.23% |

MySQL, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **813,586** | 1,019,484 | 957,907 | 878,117 | 1 | 0.00% |
| ReadByID | **251,329** | 255,958 | 260,688 | 261,192 | 1 | 0.00% |
| ReadByIDComplex | **253,863** | 254,349 | 254,338 | 261,335 | 1 | 0.00% |
| UpdateByID | **848,445** | 1,093,546 | 883,750 | 1,272,905 | 1 | 0.00% |
| DeleteByID | 765,645 | 888,310 | **751,124** | 770,066 | 2 | 1.93% |
| BatchCreate100 | **6,929,813** | 7,318,879 | 7,710,617 | 7,705,074 | 1 | 0.00% |
| BatchRead100 | **1,221,528** | 2,025,584 | 1,839,542 | 1,439,332 | 1 | 0.00% |
| BatchRead100Complex | **2,104,338** | 2,596,185 | 2,637,094 | 2,296,573 | 1 | 0.00% |
| BatchUpdate100 | **5,055,485** | 5,752,827 | 5,545,599 | 5,460,566 | 1 | 0.00% |
| BatchDelete100 | **4,013,751** | 4,530,486 | 4,110,886 | 4,293,134 | 1 | 0.00% |

PostgreSQL, `ns/op` (lower is better):

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 844,910 | 1,183,252 | 932,022 | **821,597** | 2 | 2.84% |
| ReadByID | **135,021** | 137,157 | 145,584 | 141,350 | 1 | 0.00% |
| ReadByIDComplex | **134,513** | 137,986 | 144,760 | 139,264 | 1 | 0.00% |
| UpdateByID | **864,272** | 1,025,507 | 1,108,039 | 1,162,746 | 1 | 0.00% |
| DeleteByID | 821,153 | 996,480 | 839,258 | **774,647** | 2 | 6.00% |
| BatchCreate100 | 6,428,797 | **5,594,340** | 8,263,492 | 7,090,537 | 2 | 14.92% |
| BatchRead100 | **663,685** | 931,042 | 1,168,889 | 759,619 | 1 | 0.00% |
| BatchRead100Complex | **943,754** | 1,252,891 | 1,494,433 | 1,097,075 | 1 | 0.00% |
| BatchUpdate100 | **1,239,973** | 1,373,150 | 1,446,940 | 1,269,617 | 1 | 0.00% |
| BatchDelete100 | **1,109,514** | 1,381,718 | 1,139,948 | 1,252,851 | 1 | 0.00% |

Notes from this run:

- On SQLite, `lorm` is fastest in 6 of the 10 `ns/op` cases.
- On MySQL, `lorm` is fastest in 9 of the 10 `ns/op` cases.
- On PostgreSQL, `lorm` is fastest in 8 of the 10 `ns/op` cases.
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
