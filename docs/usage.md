# LORM Comprehensive Usage Guide

[中文版](usage_zh.md) | [README](../README.md) | [Examples](../example/README.md)

This guide provides a comprehensive overview of LORM's architecture, APIs, conventions, and best practices.

---

## Table of Contents

- [1. Design Philosophy](#1-design-philosophy)
- [2. Installation & Toolchain](#2-installation--toolchain)
- [3. Model Definition & Struct Tags](#3-model-definition--struct-tags)
- [4. Code Generation (`lormgen`)](#4-code-generation-lormgen)
- [5. Engine Initialization & Configuration](#5-engine-initialization--configuration)
- [6. Core CRUD Operations](#6-core-crud-operations)
  - [Insert](#insert)
  - [Query](#query)
  - [Update](#update)
  - [Delete](#delete)
  - [Write Safety Guard](#write-safety-guard)
- [7. Type-Safe SQL Query Builder](#7-type-safe-sql-query-builder)
- [8. Pagination & Single-Column Queries](#8-pagination--single-column-queries)
- [9. Transaction Management & Concurrency](#9-transaction-management--concurrency)
- [10. Clean Architecture & Repository Pattern](#10-clean-architecture--repository-pattern)
- [11. Custom Projection Models & Joins](#11-custom-projection-models--joins)
- [12. Custom Field Types & JSON Serialization](#12-custom-field-types--json-serialization)
- [13. Statement Lifecycle & Best Practices](#13-statement-lifecycle--best-practices)

---

## 1. Design Philosophy

LORM is engineered around five fundamental principles:

1. **Zero-Reflection Runtime**: Rather than relying on runtime reflection (`reflect.ValueOf`, `reflect.TypeOf`) to inspect struct fields during query execution and scanning, LORM generates direct pointer and value accessors at compile time via `lormgen`.
2. **Compile-Time Type Safety**: SQL column references are verified at compile time through generated accessor methods (`u.LormCols().Name()`), eliminating typo-prone magic strings.
3. **Explicit SQL (No Magic)**: LORM avoids implicit joins, hidden lazy loading, and query fan-out. What you construct in Go is what gets executed against the database.
4. **Context-Propagated Transactions**: Transactions attach transparently to `context.Context`, keeping repository and domain service signatures clean and free of transaction handles.
5. **Clean Architecture Ready**: First-class support for the Repository pattern (`lorm.Repository[T]`), cleanly decoupling business domain logic from storage implementation details.

---

## 2. Installation & Toolchain

### Requirements
- **Go 1.27** or higher.

### Installing Dependencies and CLI Tool

```bash
# Add LORM library to your project
go get github.com/yvvlee/lorm

# Install the lormgen code generator binary
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

### Automated Code Generation with `go generate`

Add a `//go:generate` directive inside your model package (e.g., `model/generate.go`):

```go
package model

//go:generate lormgen .
```

Now you can regenerate all model helpers across the project with:

```bash
go generate ./...
```

---

## 3. Model Definition & Struct Tags

LORM supports two model classifications:
- **Table Models**: Represent a physical database table. Must embed `lorm.UnimplementedTable`.
- **Projection Models**: Represent arbitrary query results, custom JOIN outputs, or DTOs. Must embed `lorm.UnimplementedModel`.

### Table Model Example

```go
package model

import (
	"time"
	"github.com/yvvlee/lorm"
)

type User struct {
	lorm.UnimplementedTable `lorm:"users"` // Custom table name override (optional)

	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Email     string    `lorm:"email"`
	Age       int       `lorm:"age"`
	Version   int64     `lorm:"version"`             // Optimistic lock version counter
	CreatedAt time.Time `lorm:"created_at,created"`  // Auto-filled on insert
	UpdatedAt time.Time `lorm:"updated_at,updated"`  // Auto-refreshed on insert/update
}
```

### Struct Tag Reference Table

| Tag Item | Description | Applicable Types |
| :--- | :--- | :--- |
| `column_name` | Explicitly maps the struct field to the database column name. Defaults to `snake_case` if omitted. | All fields |
| `primary_key` | Marks the field as a primary key. Supports composite primary keys. | Scalars / Integers / Strings |
| `auto_increment` | Marks the field as auto-incrementing. Must also be marked as `primary_key`. | Integers |
| `created` | Auto-populates creation timestamp upon insert when the field is zero-valued. | `time.Time`, `sql.NullTime`, `int64`, `uint64`, `uint32`, `uint`, `int` (64-bit), `string`, and single pointers |
| `updated` | Auto-populates timestamp on insert (when zero) and automatically refreshes to current time on updates. | Same as `created` |
| `version` | Enables optimistic locking. Automatically incremented on `UPDATE`. Max 1 per model. | Integer types |
| `json` | Serializes/deserializes the field as JSON (using fast JSON codecs). | Structs, slices, maps |

### Struct Embedding & Flattening Rules

- **Embedded Structs**: Embedded structs are automatically flattened into the parent model's field accessors.
- **Prefixing**: Placing a tag on the embedded struct field prepends that tag value as a prefix to all flattened child column names.
- **Dedicated Lines**: Each field with a `lorm` tag must be declared on its own line (avoid `FieldA, FieldB string \`lorm:"..."\``).
- **Managed Time Rules**: For `int`/`uint` timestamps, values are stored as Unix seconds. For `string`, values use `time.DateTime` (`2006-01-02 15:04:05`). `int8`, `uint8`, `int16`, `uint16`, and `int32` are rejected.

---

## 4. Code Generation (`lormgen`)

`lormgen` parses Go source files, extracts struct metadata, and generates `*_lorm_gen.go` files in the same directory.

### CLI Usage

```bash
lormgen [flags] <directory|file>...
```

### Command Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--field-mapper` | `snake` | Field name mapper: `snake`, `camel`, or `same` |
| `--table-mapper` | `snake` | Table name mapper: `snake`, `camel`, or `same` |
| `--table-prefix` | `""` | Global prefix for generated table names (e.g. `tbl_`) |
| `--table-suffix` | `""` | Global suffix for generated table names |
| `--tag-key` | `lorm` | Struct tag key |
| `--file-suffix` | `_lorm_gen` | Suffix for generated Go source files |
| `--ignore` | `""` | Glob pattern to exclude files (can be specified multiple times) |

### What `lormgen` Generates

For every model, `lormgen` generates:
- `TableName() string`: Resolves the table name.
- `LormCols()`: Returns typed column name accessors (with `.WithAlias()` support).
- `LormFieldPtr(name string) any`: Direct pointer getter for zero-reflection `sql.Rows` scanning.
- `LormFieldValue(name string) any`: Direct value getter for zero-reflection query parameter binding.
- `LormModelDescriptor()`: Cached pointer to primary key metadata and field descriptors.
- Write hooks (`BeforeInsertHook`, `BeforeUpdateHook`) for automatic timestamps and versioning.

---

## 5. Engine Initialization & Configuration

`lorm.Engine` manages the underlying `*sql.DB` connection pool, SQL dialect behavior, transaction sessions, and logging.

### Basic Initialization

```go
package main

import (
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yvvlee/lorm"
)

func main() {
	// MySQL example
	engine, err := lorm.NewEngine(
		"mysql",
		"user:password@tcp(127.0.0.1:3306)/mydb?parseTime=true&charset=utf8mb4",
		lorm.WithMaxOpenConns(100),
		lorm.WithMaxIdleConns(20),
		lorm.WithConnMaxLifetime(time.Hour),
		lorm.WithConnMaxIdleTime(30*time.Minute),
		lorm.WithLogger(lorm.NewDefaultLogger()),
	)
	if err != nil {
		log.Fatalf("failed to initialize lorm engine: %v", err)
	}
	defer engine.Close()
}
```

### Multi-Database Dialects

LORM detects default dialect rules (placeholder format, identifier quoting, `RETURNING`, `FOR UPDATE`) automatically based on the driver name:

| Driver Name | Placeholder | Quoting Style | Auto ID Retrieval | Lock Support |
| :--- | :--- | :--- | :--- | :--- |
| `mysql` | `?` | Backticks (`` `col` ``) | `LastInsertId` | `FOR UPDATE` |
| `pgx` / `postgres` | `$1, $2` | Double Quotes (`"col"`) | `RETURNING` | `FOR UPDATE` |
| `sqlite3` | `?` | Double Quotes (`"col"`) | `LastInsertId` | N/A |

### Customizing Dialect Behavior

You can override dialect options at engine initialization:

```go
dialect := lorm.DefaultDialectConfig("pgx")
dialect.SupportsForUpdate = true

engine, err := lorm.NewEngine(
	"pgx",
	"postgres://user:pass@localhost:5432/mydb?sslmode=disable",
	lorm.WithDialectConfig(dialect),
)
```

---

## 6. Core CRUD Operations

### Insert

#### Single Insert
```go
user := &User{Name: "Alice", Email: "alice@example.com"}
rowsAffected, err := engine.Insert[*User]().
	AddModel(user).
	Exec(ctx)

// If the driver supports RETURNING or LastInsertId, user.ID is backfilled automatically.
```

#### Batch Insert & ID Backfill Semantics
```go
users := []*User{
	{Name: "Bob", Email: "bob@example.com"},
	{Name: "Charlie", Email: "charlie@example.com"},
}

// Standard batch insert (single multi-row SQL statement, highest performance):
_, err := engine.Insert[*User]().
	AddModels(users...).
	Exec(ctx)

// If you need auto-generated IDs backfilled into each model in a batch:
_, err := engine.Insert[*User]().
	AddModels(users...).
	RequireIDBackfill(). // Executes row-by-row inside an automatic transaction
	Exec(ctx)
```

> **Note on Primary Keys**:
> - If an `auto_increment` primary key is zero (`0`), LORM omits it from the `INSERT` column list so the database generates it.
> - If it is non-zero, LORM writes the explicit value.
> - Mixed batches with both zero and non-zero IDs are automatically grouped into contiguous sub-batches inside a single transaction.

---

### Query

#### Querying a Single Row (`Get`)
`Get` returns `(model T, found bool, err error)`. The second boolean indicates whether a matching row was found:

```go
var u User
user, found, err := engine.Query[*User]().
	Where(builder.Eq{u.LormCols().ID(): 42}).
	Get(ctx)
if err != nil {
	return err
}
if !found {
	log.Println("User not found")
}
```

#### Querying Multiple Rows (`Find`)
`Find` returns `([]T, error)`:

```go
var u User
c := u.LormCols()

users, err := engine.Query[*User]().
	Where(builder.Gte(c.Age(), 18)).
	OrderBy(c.CreatedAt() + " DESC").
	Limit(20).
	Find(ctx)
```

---

### Update

#### Partial Updates (`Set` / `SetMap`)
Always prefer `SetMap` or `Set` for partial updates to avoid overwriting unassigned fields with zero-values:

```go
var u User
c := u.LormCols()

rowsAffected, err := engine.Update[*User]().
	ID(42).
	SetMap(map[string]any{
		c.Name(): "Alice Updated",
		c.Age():  30,
	}).
	Exec(ctx)
```

#### Atomic Increments & Custom Expressions
```go
_, err := engine.Update[*User]().
	ID(42).
	SetExpr(c.Age(), "age + 1").
	Exec(ctx)
```

#### Full-Model Updates (`SetModel`)
`SetModel` updates all columns mapped by the model descriptor:

```go
user.Name = "Alice Full"
_, err := engine.Update[*User]().
	ID(user.ID).
	SetModel(user).
	Exec(ctx)
```

> ⚠️ **Warning**: `SetModel` writes zero values (e.g. `""`, `0`, `false`) to the database. It cannot be combined with `Set` or `SetMap`.

---

### Delete

#### Delete by Primary Key or Condition
```go
// Delete by ID
rowsAffected, err := engine.Delete[*User]().
	ID(42).
	Exec(ctx)

// Delete by Condition
var u User
rowsAffected, err := engine.Delete[*User]().
	Where(builder.Eq{u.LormCols().Email(): "spam@example.com"}).
	Exec(ctx)
```

---

### Write Safety Guard

To protect against catastrophic accidental table wipes, `Update.Exec` and `Delete.Exec` inspect the query and reject executions that lack a restrictive `WHERE` clause:

```go
// ❌ Fails with error: "update statement missing WHERE condition"
_, err := engine.Update[*User]().Set("status", "inactive").Exec(ctx)

// ✅ Explicitly bypass the guard when a full-table operation is intentional:
_, err := engine.Update[*User]().
	Set("status", "inactive").
	AllowGlobalWrite().
	Exec(ctx)
```

> **Note**: Raw SQL passed to `Engine.Exec` is outside this safety check. Callers remain responsible for validating SQL inputs.

---

## 7. Type-Safe SQL Query Builder

LORM includes a robust SQL builder package (`github.com/yvvlee/lorm/builder`). Using generated `LormCols()` ensures refactoring safety:

```go
var u User
c := u.LormCols()
```

### Common Predicates

| Predicate | Go Expression | Rendered SQL Snippet |
| :--- | :--- | :--- |
| **Equality** | `builder.Eq{c.Email(): "a@b.com"}` | `` `email` = ? `` |
| **Not Equal** | `builder.Ne(c.Age(), 0)` | `` `age` <> ? `` |
| **Comparisons** | `builder.Gt(c.Age(), 18)`, `builder.Lte(c.Age(), 60)` | `` `age` > ? ``, `` `age` <= ? `` |
| **IN / NOT IN** | `builder.In(c.ID(), []int64{1, 2, 3})` | `` `id` IN (?, ?, ?) `` |
| **NULL Checks** | `builder.IsNull(c.DeletedAt())`, `builder.IsNotNull(c.Email())` | `` `deleted_at` IS NULL `` |
| **LIKE** | `builder.Like(c.Name(), "John%")` | `` `name` LIKE ? `` |
| **PostgreSQL Array** | `builder.Any("roles", []string{"admin", "editor"})` | `"roles" = ANY(?)` |

> ⚠️ **Important on `builder.Eq`**: `builder.Eq{field: value}` always renders `field = ?` with `value` as a single parameter. It does NOT convert `nil` to `IS NULL` or expand slices into `IN (...)`. Use `builder.IsNull()` and `builder.In()` explicitly.

### Logical Combinations (`And` / `Or`)

```go
whereClause := builder.And(
	builder.Eq{c.Status(): "active"},
	builder.Or(
		builder.Gt(c.Score(), 90),
		builder.Eq{c.Role(): "admin"},
	),
)

users, err := engine.Query[*User]().
	Where(whereClause).
	Find(ctx)
```

### Table Aliases with `WithAlias`

When writing complex queries involving joins, use `WithAlias()`:

```go
var u User
c := u.LormCols().WithAlias("u")

ids, err := engine.Query[*User]().
	Select(c.ID()).
	From("users AS u").
	Where(builder.Like(c.Email(), "%@example.com")).
	OrderBy(c.ID() + " DESC").
	FindCols[int64](ctx)
```

---

## 8. Pagination & Single-Column Queries

### Pagination with Model Results (`Page`)

`Page(ctx, page, size)` computes the total row count and retrieves the requested page (1-indexed) in one seamless call:

```go
var u User
users, total, err := engine.Query[*User]().
	Where(builder.Eq{u.LormCols().Status(): "active"}).
	OrderBy(u.LormCols().ID() + " DESC").
	Page(ctx, 1, 20) // Page 1, 20 items per page

if err != nil {
	return err
}
log.Printf("Found %d users (Total: %d)", len(users), total)
```

### Single-Column Queries (`GetCol`, `FindCols`, `PageCols`)

When selecting a single column or aggregate value, use generic column scanners:

```go
var u User
c := u.LormCols()

// 1. Fetch single scalar value
count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[int64](ctx)

// 2. Fetch list of single column values
ids, err := engine.Query[*User]().
	Select(c.ID()).
	Where(builder.Gt(c.Age(), 21)).
	FindCols[int64](ctx)

// 3. Paginate single column values
pageIDs, total, err := engine.Query[*User]().
	Select(c.ID()).
	OrderBy(c.ID() + " ASC").
	PageCols[int64](ctx, 1, 50)
```

> **Note**: Single-column terminal methods require the SQL result to contain exactly one column. If multiple columns are returned, an error is returned.

---

## 9. Transaction Management & Concurrency

### Context-Propagated Transactions (`Engine.TX`)

LORM provides declarative transaction management. All repository or engine operations that receive the transactional `ctx` automatically participate in the transaction:

```go
err := engine.TX(ctx, func(txCtx context.Context) error {
	_, err := engine.Insert[*User]().
		AddModel(&User{Name: "User 1", Email: "u1@example.com"}).
		Exec(txCtx)
	if err != nil {
		return err // Automatically rolls back
	}

	_, err = engine.Insert[*User]().
		AddModel(&User{Name: "User 2", Email: "u2@example.com"}).
		Exec(txCtx)
	return err // Returning nil commits the transaction
})
```

### Nested Transactions

Calling `Engine.TX` with a `context` that already carries an active transaction session seamlessly reuses the existing transaction rather than opening a redundant one.

### Custom Transaction Options (`TXWithOptions`)

```go
opts := &sql.TxOptions{
	Isolation: sql.LevelSerializable,
	ReadOnly:  false,
}

err := engine.TXWithOptions(ctx, opts, func(txCtx context.Context) error {
	// Transaction runs with SERIALIZABLE isolation
	return nil
})
```

### Pessimistic Row Locking (`FOR UPDATE`)

Call `Lock` or `LockByField` on repositories inside an active transaction:

```go
err := engine.TX(ctx, func(txCtx context.Context) error {
	user, err := userRepo.Lock(txCtx, userID)
	if err != nil {
		return err
	}

	user.Balance -= 100
	_, err = userRepo.Update(txCtx, user)
	return err
})
```

> **Note**: Row locking requires an active transaction (`Engine.TX`). Outside a transaction, the database immediately releases the row lock, so LORM returns an error.

### Optimistic Locking (`version`)

Add a `version` tag to an integer field in your model:

```go
type Product struct {
	lorm.UnimplementedTable
	ID      int64 `lorm:"id,primary_key,auto_increment"`
	Stock   int   `lorm:"stock"`
	Version int64 `lorm:"version"`
}
```

When updating via `Update.SetModel(product)`:
1. LORM renders: `UPDATE products SET stock = ?, version = version + 1 WHERE id = ? AND version = ?`.
2. If another concurrent process modified the record first, `rowsAffected` will be `0`, indicating a conflict.

---

## 10. Clean Architecture & Repository Pattern

LORM's `lorm.Repository[T]` encapsulates common single-table operations (`Get`, `GetByField`, `Lock`, `Exist`, `Insert`, `InsertAll`, `Update`, `UpdateMap`, `Delete`, `DeleteByField`).

### Recommended Repository Implementation

```go
package repository

import (
	"context"
	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
	"yourproject/model"
)

// 1. Define clean interface for the domain layer
type UserRepository interface {
	Get(ctx context.Context, id any) (*model.User, error)
	Insert(ctx context.Context, user *model.User) (int64, error)
	Update(ctx context.Context, user *model.User) (int64, error)
	FindAdults(ctx context.Context, page, size uint64) ([]*model.User, uint64, error)
}

// 2. Concrete implementation embedding generic repository
type UserRepositoryImpl struct {
	*lorm.Repository[*model.User]
}

var _ UserRepository = (*UserRepositoryImpl)(nil)

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: engine.Repository[*model.User](),
	}
}

// 3. Implement custom query methods
func (r *UserRepositoryImpl) FindAdults(ctx context.Context, page, size uint64) ([]*model.User, uint64, error) {
	var u model.User
	return r.Engine.Query[*model.User]().
		Where(builder.Gte(u.LormCols().Age(), 18)).
		OrderBy(u.LormCols().ID() + " DESC").
		Page(ctx, page, size)
}
```

---

## 11. Custom Projection Models & Joins

When query results aggregate columns across multiple tables or do not map directly to a single table entity, embed `lorm.UnimplementedModel`:

```go
package model

import "github.com/yvvlee/lorm"

type UserOrderSummary struct {
	lorm.UnimplementedModel

	UserID       int64  `lorm:"user_id"`
	UserName     string `lorm:"user_name"`
	TotalOrders  int64  `lorm:"total_orders"`
	TotalSpent   int64  `lorm:"total_spent"`
}
```

Run `lormgen` to generate field scanners, then query with explicit SQL joins:

```go
summaries, err := engine.Query[*model.UserOrderSummary]().
	Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"COUNT(o.id) AS total_orders",
		"COALESCE(SUM(o.amount), 0) AS total_spent",
	).
	From("users AS u").
	LeftJoin("orders AS o ON u.id = o.user_id").
	GroupBy("u.id", "u.name").
	Find(ctx)
```

---

## 12. Custom Field Types & JSON Serialization

### Built-in JSON Fields

Annotate struct or slice fields with `,json` in the struct tag:

```go
type Address struct {
	City    string `json:"city"`
	Street  string `json:"street"`
	ZipCode string `json:"zip_code"`
}

type Customer struct {
	lorm.UnimplementedTable
	ID      int64    `lorm:"id,primary_key,auto_increment"`
	Address Address  `lorm:"address,json"` // Stored as JSON string/binary in DB
	Tags    []string `lorm:"tags,json"`    // Stored as JSON array in DB
}
```

### Custom Type Converters via `ScannerValuer`

For custom data formats, implement standard Go `driver.Valuer` and `sql.Scanner` interfaces on your type:

```go
package customtype

import (
	"database/sql/driver"
	"strings"
	"github.com/yvvlee/lorm"
)

type StringSlice []string

// Compile-time assertion
var _ lorm.ScannerValuer = (*StringSlice)(nil)

func (s StringSlice) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s *StringSlice) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	switch v := src.(type) {
	case string:
		*s = strings.Split(v, ",")
	case []byte:
		*s = strings.Split(string(v), ",")
	}
	return nil
}
```

---

## 13. Statement Lifecycle & Best Practices

1. **Lightweight Statement Creation**: Statement builders (`Query`, `Insert`, `Update`, `Delete`) are lightweight structs. Always instantiate a fresh chain per database call.
2. **Goroutine Safety**: Statement builders are mutable and **NOT** thread-safe across concurrent goroutines. Do not share a statement instance across goroutines.
3. **Cloning Statements**: To branch queries from a shared base condition, call `.Clone()`:
   ```go
   baseQuery := engine.Query[*User]().Where(builder.Eq{u.LormCols().Status(): "active"})
   adminQuery := baseQuery.Clone().Where(builder.Eq{u.LormCols().Role(): "admin"})
   ```
4. **Use Typed Column Accessors**: Always prefer `u.LormCols().FieldName()` over hardcoded strings to ensure compile-time safety and painless refactoring.

