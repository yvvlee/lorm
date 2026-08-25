# LORM - Lightweight ORM for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/yvvlee/lorm)](https://goreportcard.com/report/github.com/yvvlee/lorm)
[![Go Reference](https://pkg.go.dev/badge/github.com/yvvlee/lorm.svg)](https://pkg.go.dev/github.com/yvvlee/lorm)
[![Build Status](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml/badge.svg)](https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[中文](README_ZH.md)

LORM is a lightweight ORM for Go. It keeps SQL explicit, uses code generation
for model metadata and typed column accessors, and keeps data access behind
repository interfaces.

## Highlights

- Generated model metadata avoids reflection-heavy mapping at runtime.
- Typed column accessors reduce hand-written column names in queries.
- Repository helpers cover common CRUD while keeping complex SQL explicit.
- Transactions are propagated through `context.Context`.
- No implicit joins, relation loading, or hidden query fan-out.

## Database Support

| Database | Driver package | Driver name |
| --- | --- | --- |
| MySQL/MariaDB | `github.com/go-sql-driver/mysql` | `mysql` |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | `pgx` |
| SQLite | `github.com/mattn/go-sqlite3` | `sqlite3` |

MySQL/MariaDB and PostgreSQL are first-class targets. SQLite is supported for
local development and examples. LORM does not import database drivers; import
only the driver used by your application.

## Install

LORM requires Go 1.27 or later.

```bash
go get github.com/yvvlee/lorm
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

## Documentation

- [Usage guide](docs/usage.md): models, code generation, CRUD, transactions, repositories, configuration, and `lormgen`.
- [Runnable examples](example/README.md): self-contained workflows using SQLite.
- [API reference](https://pkg.go.dev/github.com/yvvlee/lorm): exported package API.
- [Benchmarks](benchmarks/orm-crud/README.md): methodology and complete results.

## Contributing

Open an issue or submit a pull request.

## License

MIT. See [LICENSE](LICENSE).
