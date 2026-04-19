# ORM CRUD Benchmarks

This benchmark suite compares basic CRUD paths across:

- `lorm`
- `GORM`
- `XORM`
- `ent`

The suite lives in its own Go module so it does not modify the root project's
`go.mod`.

## Covered operations

- Create one row
- Read one row by primary key
- Update one row by primary key
- Delete one row by primary key

## Database

All benchmarks use SQLite with a fresh temp database file per benchmark case and
the same schema:

- `bench_users`
- integer auto-increment primary key
- indexed `name`
- unique `email`

## Run

```bash
cd benchmarks/orm-crud
go test -bench . -benchmem
```

To regenerate ent code:

```bash
cd benchmarks/orm-crud
go generate ./ent
```
