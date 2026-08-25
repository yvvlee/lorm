# LORM Usage Guide

[中文](usage_zh.md) | [README](../README.md)

This guide covers the complete LORM workflow after installation. For
self-contained programs, see the [runnable examples](../example/README.md).

## Contents

- [Quick Start](#quick-start)
- [Examples](#examples)
- [Transactions](#transactions)
- [Repository Helper](#repository-helper)
- [Custom Projection Models](#custom-projection-models)
- [Custom Field Conversion](#custom-field-conversion)
- [Configuration](#configuration)
- [lormgen](#lormgen)

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

> **PostgreSQL arrays**: `builder.Any(field, values)` renders `field = ANY(?)`
> and passes `values` as one driver argument. `builder.NotAny(field, values)`
> renders `field <> ALL(?)`. These expressions require a PostgreSQL driver that
> encodes the slice as a PostgreSQL array and are not portable to other drivers.

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

See [example/README.md](../example/README.md) for runnable, self-contained examples.

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

See [example/custom_conversion/main.go](../example/custom_conversion/main.go) for a
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
