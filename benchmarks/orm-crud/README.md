# ORM CRUD Benchmarks

This benchmark suite compares basic CRUD paths across:

- `lorm`
- `gorm`
- `xorm`
- `ent`

The suite lives in its own Go module so it does not modify the root project's
`go.mod`.

## Covered Operations

- Create one row
- Read one row by primary key
- Update one row by primary key
- Delete one row by primary key
- Create 100 rows
- Read 100 rows
- Update 100 rows
- Delete 100 rows

## Database Backends

By default the suite uses SQLite with a fresh temp database file per benchmark
case.

Other supported backends:

- `ORMCRUD_DB=mysql`
- `ORMCRUD_DB=postgres`

All backends use the same benchmark schema:

- table: `bench_users`
- integer auto-increment primary key
- indexed `name`
- unique `email`

## Run

SQLite, the default backend:

```bash
cd benchmarks/orm-crud
go test -bench . -benchmem
```

MySQL:

```bash
cd benchmarks/orm-crud
ORMCRUD_DB=mysql go test -bench . -benchmem
```

PostgreSQL:

```bash
cd benchmarks/orm-crud
ORMCRUD_DB=postgres go test -bench . -benchmem
```

To regenerate ent code:

```bash
cd benchmarks/orm-crud
go generate ./ent
```

## Latest Recorded Result

Recorded on April 20, 2026 with:

```bash
cd benchmarks/orm-crud
go test -bench . -benchmem
```

Environment:

- Backend: SQLite
- OS/arch: `linux/amd64`
- CPU: `Intel(R) Core(TM) i5-9400F CPU @ 2.90GHz`

Treat these numbers as a directional snapshot. Re-run the suite on your target
database, schema, driver, and hardware before using them for a final decision.

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 97,471 | 127,067 | 99,391 | 107,457 |
| ReadByID | 31,231 | 36,679 | 45,174 | 35,275 |
| UpdateByID | 85,727 | 104,658 | 93,461 | 123,181 |
| DeleteByID | 79,837 | 96,848 | 84,681 | 78,846 |
| BatchCreate100 | 1,139,752 | 724,101 | 4,025,133 | 853,383 |
| BatchRead100 | 346,392 | 427,952 | 725,096 | 411,414 |
| BatchUpdate100 | 211,472 | 211,800 | 185,189 | 186,621 |
| BatchDelete100 | 332,062 | 326,834 | 313,968 | 313,609 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 3,008 | 6,586 | 3,369 | 3,335 |
| ReadByID | 3,978 | 4,508 | 6,579 | 4,816 |
| UpdateByID | 2,792 | 7,500 | 4,441 | 6,238 |
| DeleteByID | 1,624 | 5,547 | 3,329 | 1,863 |
| BatchCreate100 | 121,457 | 97,535 | 1,125,734 | 252,276 |
| BatchRead100 | 59,003 | 89,032 | 217,501 | 84,680 |
| BatchUpdate100 | 14,271 | 18,368 | 19,059 | 20,351 |
| BatchDelete100 | 14,112 | 17,566 | 24,483 | 19,168 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 64 | 95 | 68 | 82 |
| ReadByID | 95 | 83 | 190 | 121 |
| UpdateByID | 63 | 84 | 113 | 163 |
| DeleteByID | 36 | 65 | 88 | 41 |
| BatchCreate100 | 2,110 | 1,987 | 43,564 | 3,964 |
| BatchRead100 | 1,293 | 1,869 | 5,643 | 1,831 |
| BatchUpdate100 | 241 | 282 | 283 | 183 |
| BatchDelete100 | 234 | 269 | 298 | 158 |
