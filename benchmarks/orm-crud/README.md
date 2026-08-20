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
- Read one row by primary key with complex JSON field deserialization
- Update one row by primary key
- Delete one row by primary key
- Create 100 rows
- Read 100 rows
- Read 100 rows with complex JSON field deserialization
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
- nullable scalar fields such as `alias`, `age_p`, `active_p`
- non-null JSON-style fields such as `tags`, `meta`, `profile`, `contacts`

Update benchmarks explicitly write `updated_at` so ORM-specific auto timestamp
behavior does not change the measured field set. Update and delete benchmarks
also check the affected row count where the ORM exposes it.

## Run

SQLite, the default backend:

```bash
cd benchmarks/orm-crud
CGO_ENABLED=1 go test -bench . -benchmem
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

Recorded on August 20, 2026 with Go 1.27.0. Each ORM was run once in a
separate `go test` process.

SQLite was run once per ORM:

```bash
for orm in lorm gorm xorm ent; do
  CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -count=1
done
```

The database-backed runs used this bounded readiness check:

```bash
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
```

Before every MySQL run, the container was restarted. The benchmark waited for
`mysqladmin ping` to succeed, then waited another 10 seconds:

```bash
docker restart mysql
wait_for_container mysql mysqladmin ping -h 127.0.0.1 -uroot -p123456
sleep 10
ORMCRUD_DB=mysql go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

Before every PostgreSQL run, the container was restarted and the benchmark
waited for `pg_isready` to succeed:

```bash
docker restart postgres
wait_for_container postgres pg_isready -U postgres -d postgres
ORMCRUD_DB=postgres go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

Replace `<orm>` with `lorm`, `gorm`, `xorm`, or `ent`. Each ORM uses
separate benchmark databases.

Environment:

- OS/arch: `darwin/arm64`
- CPU: `Apple M1 Pro`
- Go: `go1.27.0`
- SQLite driver: `github.com/mattn/go-sqlite3 v1.14.50`
- MySQL: `9.7.0`
- PostgreSQL: `18.6`
- `sonic/ast` fell back to `encoding/json` under Go 1.27.

Treat these numbers as a directional snapshot. Re-run the suite on your target
database, schema, driver, and hardware before using them for a final decision.

Ranks compare all four ORMs for each metric; lower is better and ties share the
same rank. `Gap to best` is calculated as `(lorm / best - 1) * 100%`.

### SQLite

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 320,185 | 279,823 | 291,430 | 270,549 | 4 | 18.35% |
| ReadByID | 21,189 | 24,540 | 30,467 | 26,395 | 1 | 0.00% |
| ReadByIDComplex | 22,487 | 26,886 | 32,928 | 29,012 | 1 | 0.00% |
| UpdateByID | 342,819 | 346,223 | 324,730 | 288,283 | 3 | 18.92% |
| DeleteByID | 344,690 | 310,801 | 336,944 | 273,470 | 4 | 26.04% |
| BatchCreate100 | 1,198,248 | 1,326,222 | 3,226,068 | 1,510,362 | 1 | 0.00% |
| BatchRead100 | 521,933 | 759,109 | 894,268 | 583,942 | 1 | 0.00% |
| BatchRead100Complex | 761,215 | 992,272 | 1,140,149 | 841,553 | 1 | 0.00% |
| BatchUpdate100 | 398,032 | 450,038 | 374,384 | 401,475 | 2 | 6.32% |
| BatchDelete100 | 690,346 | 765,703 | 591,269 | 595,752 | 3 | 16.76% |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 5,404 | 10,863 | 6,795 | 7,570 | 1 | 0.00% |
| ReadByID | 6,102 | 7,410 | 11,510 | 9,028 | 1 | 0.00% |
| ReadByIDComplex | 7,132 | 8,455 | 12,666 | 10,054 | 1 | 0.00% |
| UpdateByID | 8,780 | 11,955 | 9,556 | 14,256 | 1 | 0.00% |
| DeleteByID | 1,472 | 5,715 | 2,817 | 1,927 | 1 | 0.00% |
| BatchCreate100 | 394,732 | 356,144 | 1,477,517 | 697,109 | 2 | 10.83% |
| BatchRead100 | 212,206 | 306,021 | 489,156 | 265,932 | 1 | 0.00% |
| BatchRead100Complex | 317,495 | 412,139 | 608,825 | 372,321 | 1 | 0.00% |
| BatchUpdate100 | 19,068 | 23,403 | 25,837 | 25,643 | 1 | 0.00% |
| BatchDelete100 | 11,977 | 17,723 | 23,968 | 19,232 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 100 | 155 | 127 | 166 | 1 | 0.00% |
| ReadByID | 122 | 147 | 287 | 226 | 1 | 0.00% |
| ReadByIDComplex | 129 | 154 | 294 | 233 | 1 | 0.00% |
| UpdateByID | 164 | 151 | 210 | 326 | 2 | 8.61% |
| DeleteByID | 36 | 66 | 51 | 41 | 1 | 0.00% |
| BatchCreate100 | 6,258 | 7,097 | 49,574 | 11,483 | 1 | 0.00% |
| BatchRead100 | 4,506 | 6,037 | 11,365 | 6,323 | 1 | 0.00% |
| BatchRead100Complex | 5,592 | 7,127 | 12,454 | 7,412 | 1 | 0.00% |
| BatchUpdate100 | 248 | 351 | 403 | 275 | 1 | 0.00% |
| BatchDelete100 | 129 | 269 | 261 | 158 | 1 | 0.00% |

### MySQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 906,321 | 1,361,265 | 973,385 | 928,810 | 1 | 0.00% |
| ReadByID | 240,317 | 302,950 | 284,257 | 249,565 | 1 | 0.00% |
| ReadByIDComplex | 245,672 | 298,108 | 305,488 | 248,823 | 1 | 0.00% |
| UpdateByID | 879,429 | 1,410,948 | 879,682 | 1,440,935 | 1 | 0.00% |
| DeleteByID | 822,350 | 1,031,374 | 871,111 | 751,133 | 2 | 9.48% |
| BatchCreate100 | 7,846,680 | 8,017,562 | 7,364,879 | 8,164,799 | 2 | 6.54% |
| BatchRead100 | 2,297,721 | 3,442,311 | 1,955,529 | 1,543,769 | 3 | 48.84% |
| BatchRead100Complex | 3,716,681 | 5,032,195 | 2,309,288 | 3,182,881 | 3 | 60.94% |
| BatchUpdate100 | 5,398,099 | 5,773,743 | 4,933,102 | 6,097,586 | 2 | 9.43% |
| BatchDelete100 | 4,047,424 | 4,165,851 | 4,114,345 | 4,097,822 | 1 | 0.00% |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 7,209 | 11,184 | 7,806 | 8,995 | 1 | 0.00% |
| ReadByID | 5,975 | 7,331 | 11,128 | 9,431 | 1 | 0.00% |
| ReadByIDComplex | 6,539 | 7,892 | 11,958 | 10,548 | 1 | 0.00% |
| UpdateByID | 10,661 | 13,756 | 11,158 | 16,545 | 1 | 0.00% |
| DeleteByID | 1,543 | 5,739 | 2,889 | 2,004 | 1 | 0.00% |
| BatchCreate100 | 577,748 | 537,926 | 1,578,788 | 873,494 | 2 | 7.40% |
| BatchRead100 | 165,029 | 257,612 | 433,660 | 276,313 | 1 | 0.00% |
| BatchRead100Complex | 224,663 | 318,033 | 520,483 | 391,094 | 1 | 0.00% |
| BatchUpdate100 | 28,458 | 32,786 | 34,967 | 35,012 | 1 | 0.00% |
| BatchDelete100 | 19,686 | 25,387 | 31,674 | 26,936 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 128 | 162 | 131 | 179 | 1 | 0.00% |
| ReadByID | 90 | 116 | 229 | 202 | 1 | 0.00% |
| ReadByIDComplex | 97 | 123 | 236 | 209 | 1 | 0.00% |
| UpdateByID | 193 | 177 | 227 | 325 | 2 | 9.04% |
| DeleteByID | 39 | 66 | 54 | 45 | 1 | 0.00% |
| BatchCreate100 | 8,651 | 9,476 | 49,575 | 13,267 | 1 | 0.00% |
| BatchRead100 | 3,483 | 5,007 | 10,513 | 6,103 | 1 | 0.00% |
| BatchRead100Complex | 4,576 | 6,103 | 11,608 | 7,198 | 1 | 0.00% |
| BatchUpdate100 | 277 | 378 | 420 | 301 | 1 | 0.00% |
| BatchDelete100 | 134 | 272 | 266 | 163 | 1 | 0.00% |

### PostgreSQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 882,606 | 1,245,377 | 902,820 | 916,939 | 1 | 0.00% |
| ReadByID | 135,480 | 134,977 | 143,964 | 141,048 | 2 | 0.37% |
| ReadByIDComplex | 136,348 | 138,018 | 141,529 | 140,041 | 1 | 0.00% |
| UpdateByID | 959,347 | 1,063,873 | 1,250,379 | 1,339,732 | 1 | 0.00% |
| DeleteByID | 945,561 | 1,167,307 | 859,569 | 856,562 | 3 | 10.39% |
| BatchCreate100 | 7,620,672 | 7,793,676 | 8,074,182 | 8,180,815 | 1 | 0.00% |
| BatchRead100 | 648,760 | 907,110 | 1,115,124 | 759,521 | 1 | 0.00% |
| BatchRead100Complex | 937,986 | 1,268,048 | 1,491,468 | 1,065,715 | 1 | 0.00% |
| BatchUpdate100 | 1,325,303 | 1,463,390 | 1,505,688 | 1,323,338 | 2 | 0.15% |
| BatchDelete100 | 1,288,657 | 1,467,285 | 1,238,411 | 1,216,635 | 3 | 5.92% |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 7,585 | 11,956 | 9,217 | 8,767 | 1 | 0.00% |
| ReadByID | 7,063 | 7,718 | 12,091 | 9,799 | 1 | 0.00% |
| ReadByIDComplex | 7,899 | 8,546 | 13,199 | 11,193 | 1 | 0.00% |
| UpdateByID | 10,343 | 12,864 | 11,973 | 16,077 | 1 | 0.00% |
| DeleteByID | 1,560 | 5,694 | 2,977 | 1,901 | 1 | 0.00% |
| BatchCreate100 | 837,087 | 795,665 | 2,021,520 | 1,134,823 | 2 | 5.21% |
| BatchRead100 | 206,459 | 298,144 | 465,774 | 316,636 | 1 | 0.00% |
| BatchRead100Complex | 293,805 | 386,358 | 580,389 | 459,449 | 1 | 0.00% |
| BatchUpdate100 | 45,075 | 47,542 | 54,921 | 49,862 | 1 | 0.00% |
| BatchDelete100 | 36,642 | 40,955 | 50,176 | 43,199 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | 114 | 152 | 172 | 179 | 1 | 0.00% |
| ReadByID | 107 | 130 | 261 | 217 | 1 | 0.00% |
| ReadByIDComplex | 114 | 137 | 268 | 224 | 1 | 0.00% |
| UpdateByID | 167 | 147 | 255 | 329 | 2 | 13.61% |
| DeleteByID | 36 | 60 | 57 | 40 | 1 | 0.00% |
| BatchCreate100 | 7,505 | 8,342 | 53,273 | 14,025 | 1 | 0.00% |
| BatchRead100 | 3,915 | 5,439 | 10,562 | 6,631 | 1 | 0.00% |
| BatchRead100Complex | 5,010 | 6,538 | 11,661 | 7,727 | 1 | 0.00% |
| BatchUpdate100 | 277 | 372 | 653 | 408 | 1 | 0.00% |
| BatchDelete100 | 147 | 278 | 482 | 270 | 1 | 0.00% |

## Summary

- SQLite `ns/op`: `lorm` ranks first in 5 of 10 cases.
- MySQL `ns/op`: `lorm` ranks first in 5 of 10 cases.
- PostgreSQL `ns/op`: `lorm` ranks first in 6 of 10 cases.
- `B/op`: `lorm` has the lowest allocation bytes in 9 of 10 SQLite cases,
  9 of 10 MySQL cases, and 9 of 10 PostgreSQL cases.
- `allocs/op`: `lorm` has the lowest allocation count in 9 of 10 SQLite
  cases, 9 of 10 MySQL cases, and 9 of 10 PostgreSQL cases.
