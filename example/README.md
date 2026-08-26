# LORM Runnable Examples

[English README](../README.md) | [中文 README](../README_ZH.md) | [Usage Guide](../docs/usage.md)

This directory contains self-contained, executable examples demonstrating real-world LORM usage patterns.

All examples use **SQLite** with temporary/in-memory databases, allowing you to run and experiment with them immediately without provisioning external database servers.

---

## 🚀 Quick Execution

Run any example directly from the `example` directory:

```bash
cd example

# Quickstart CRUD basics
go run ./quickstart

# Clean Architecture & Repository pattern
go run ./repository

# Context-propagated transactions
go run ./transaction

# Multi-table explicit JOINs & custom DTO models
go run ./custom_model

# Custom field conversion (ScannerValuer)
go run ./custom_conversion

# Transparent JSON fields & nested slices
go run ./json_field

# Pagination with total count & single column pagination
go run ./pagination

# Safe concurrent updates with optimistic locking (version)
go run ./optimistic_lock

# Complex SQL queries with builder expressions (And/Or/Like/In)
go run ./query_builder
```

To run the full test suite across all examples:

```bash
cd example
CGO_ENABLED=1 go test -v ./...
```

To regenerate the checked-in `_lorm_gen.go` helper files for all examples:

```bash
cd example
go generate ./...
```

---

## 📂 Example Catalog

| Directory | Core Scenario | Demonstrated Concepts & APIs |
| :--- | :--- | :--- |
| **[`quickstart`](quickstart/)** | Engine setup & basic CRUD | `NewEngine`, `Insert`, `Query.Get`, `Update.SetMap`, `Delete`, auto-increment ID backfill |
| **[`repository`](repository/)** | Clean Architecture / DDD | Embedding `*lorm.Repository[T]`, domain interface segregation, custom query encapsulation |
| **[`transaction`](transaction/)** | Declarative Transactions | `Engine.TX`, transparent transaction context propagation, automatic commit & rollback |
| **[`custom_model`](custom_model/)** | Multi-table JOINs & Projections | `UnimplementedModel`, `InnerJoin`, `LeftJoin`, `Select AS`, custom result mapping without reflection |
| **[`custom_conversion`](custom_conversion/)** | Custom DB Type Serialization | Implementing `lorm.ScannerValuer` (`driver.Valuer` + `sql.Scanner`) for custom domain types |
| **[`json_field`](json_field/)** | JSON Columns | `lorm:",json"` struct tag, storing nested structs and primitive slices seamlessly |
| **[`pagination`](pagination/)** | Result Pagination | `Page(ctx, page, size)`, `PageCols[T](ctx, page, size)`, total count computation |
| **[`optimistic_lock`](optimistic_lock/)** | Concurrency Conflict Detection | `lorm:"version"` tag, atomic version increment (`version = version + 1`), CAS conflict checks |
| **[`query_builder`](query_builder/)** | Complex SQL Predicates | `builder.And`, `builder.Or`, `builder.Like`, `builder.In`, `FindCols[T]`, `GetCol[T]` |

---

## 💡 Key Design Notes

1. **Zero-Setup Testing**: Every example creates and initializes its own SQLite database in a temporary file and cleans up automatically upon completion.
2. **Explicit SQL by Design**: The examples intentionally avoid implicit relationship loading or magic associations. All join operations and column selections are explicit, ensuring predictable performance and transparent execution plans.
3. **Pre-Generated Helpers Checked-In**: All `*_lorm_gen.go` files are pre-generated and checked into Git so the examples compile and run immediately after cloning the repository.

