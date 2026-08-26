# ORM CRUD Benchmarks

[English README](../../README.md) | [中文 README](../../README_ZH.md) | [Usage Guide](../../docs/usage.md)

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

Recorded on August 21, 2026 with Go 1.27.0 on `darwin/arm64` (Apple M1 Pro).
Each backend was measured in three independent rounds. Every ORM ran in a
separate `go test` process. The tables below show the median of those rounds.
This reduces transient database and host noise; it does not guarantee the same
result on every deployment.

SQLite creates a fresh temporary database for each benchmark case. Before every
MySQL and PostgreSQL ORM run, the corresponding container was restarted,
readiness was checked, and the server was given a 10-second warm-up. The
benchmark body is timed by Go; database setup and cleanup are outside the
measured loop. All three backends use a 1-second window to amortize connection
and commit jitter that made the previous short-window `UpdateByID` result
unstable.

SQLite command:

```bash
for round in 1 2 3; do
  for orm in lorm gorm xorm ent; do
    CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -benchtime=1s -count=1
  done
done
```

MySQL command used for the recorded result:

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

PostgreSQL used the same loop, with `postgres` restarted before every ORM run:

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

Each ORM uses a separate benchmark database.

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

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **5,405** | 10,864 | 6,795 | 7,570 | 1 | 0.00% |
| ReadByID | **6,103** | 7,405 | 11,512 | 9,025 | 1 | 0.00% |
| ReadByIDComplex | **7,133** | 8,458 | 12,670 | 10,055 | 1 | 0.00% |
| UpdateByID | **8,782** | 11,947 | 9,558 | 14,269 | 1 | 0.00% |
| DeleteByID | **1,472** | 5,718 | 2,817 | 1,927 | 1 | 0.00% |
| BatchCreate100 | 394,667 | **356,009** | 1,477,469 | 697,136 | 2 | 10.86% |
| BatchRead100 | **212,275** | 305,978 | 489,158 | 265,966 | 1 | 0.00% |
| BatchRead100Complex | **317,533** | 412,222 | 608,709 | 372,227 | 1 | 0.00% |
| BatchUpdate100 | **19,071** | 23,405 | 25,835 | 25,636 | 1 | 0.00% |
| BatchDelete100 | **11,977** | 17,726 | 23,968 | 19,232 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **100** | 155 | 127 | 166 | 1 | 0.00% |
| ReadByID | 122 | 147 | 287 | 226 | 1 | 0.00% |
| ReadByIDComplex | 129 | 154 | 294 | 233 | 1 | 0.00% |
| UpdateByID | 164 | **151** | 210 | 326 | 2 | 8.61% |
| DeleteByID | **36** | 66 | 51 | 41 | 1 | 0.00% |
| BatchCreate100 | **6,257** | 7,097 | 49,574 | 11,483 | 1 | 0.00% |
| BatchRead100 | **4,507** | 6,037 | 11,365 | 6,324 | 1 | 0.00% |
| BatchRead100Complex | **5,592** | 7,128 | 12,452 | 7,411 | 1 | 0.00% |
| BatchUpdate100 | **248** | 351 | 403 | 275 | 1 | 0.00% |
| BatchDelete100 | 129 | 269 | 261 | 158 | 1 | 0.00% |

### MySQL

`ns/op`:

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

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **7,211** | 11,192 | 7,806 | 8,990 | 1 | 0.00% |
| ReadByID | **5,976** | 7,337 | 11,123 | 9,427 | 1 | 0.00% |
| ReadByIDComplex | **6,531** | 7,902 | 11,957 | 10,558 | 1 | 0.00% |
| UpdateByID | **10,656** | 13,791 | 11,162 | 16,541 | 1 | 0.00% |
| DeleteByID | **1,543** | 5,743 | 2,888 | 2,004 | 1 | 0.00% |
| BatchCreate100 | 578,384 | **538,349** | 1,578,807 | 874,045 | 2 | 7.44% |
| BatchRead100 | **165,139** | 258,242 | 433,690 | 276,295 | 1 | 0.00% |
| BatchRead100Complex | **224,769** | 318,592 | 520,669 | 391,095 | 1 | 0.00% |
| BatchUpdate100 | **28,471** | 32,786 | 34,985 | 34,994 | 1 | 0.00% |
| BatchDelete100 | **19,686** | 25,379 | 31,674 | 26,936 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **128** | 162 | 131 | 179 | 1 | 0.00% |
| ReadByID | **90** | 116 | 229 | 202 | 1 | 0.00% |
| ReadByIDComplex | **97** | 123 | 236 | 209 | 1 | 0.00% |
| UpdateByID | 193 | **178** | 227 | 325 | 2 | 8.43% |
| DeleteByID | **39** | 66 | 54 | 45 | 1 | 0.00% |
| BatchCreate100 | **8,655** | 9,476 | 49,574 | 13,270 | 1 | 0.00% |
| BatchRead100 | **3,484** | 5,010 | 10,513 | 6,103 | 1 | 0.00% |
| BatchRead100Complex | **4,576** | 6,107 | 11,609 | 7,197 | 1 | 0.00% |
| BatchUpdate100 | **277** | 378 | 420 | 301 | 1 | 0.00% |
| BatchDelete100 | **134** | 272 | 266 | 163 | 1 | 0.00% |

### PostgreSQL

`ns/op`:

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

`B/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **7,594** | 11,949 | 9,225 | 8,764 | 1 | 0.00% |
| ReadByID | **7,062** | 7,705 | 12,096 | 9,801 | 1 | 0.00% |
| ReadByIDComplex | **7,911** | 8,562 | 13,207 | 11,194 | 1 | 0.00% |
| UpdateByID | **10,346** | 12,862 | 11,962 | 16,083 | 1 | 0.00% |
| DeleteByID | **1,560** | 5,697 | 2,977 | 1,901 | 1 | 0.00% |
| BatchCreate100 | 837,355 | **795,884** | 2,021,362 | 1,134,928 | 2 | 5.21% |
| BatchRead100 | **206,539** | 298,294 | 465,968 | 316,662 | 1 | 0.00% |
| BatchRead100Complex | **293,743** | 386,201 | 580,452 | 459,490 | 1 | 0.00% |
| BatchUpdate100 | **45,065** | 47,531 | 54,931 | 49,853 | 1 | 0.00% |
| BatchDelete100 | **36,632** | 40,954 | 50,176 | 43,199 | 1 | 0.00% |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent | lorm rank | Gap to best |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **115** | 152 | 172 | 179 | 1 | 0.00% |
| ReadByID | 107 | 130 | 261 | 217 | 1 | 0.00% |
| ReadByIDComplex | 114 | 137 | 268 | 224 | 1 | 0.00% |
| UpdateByID | 167 | **147** | 255 | 329 | 2 | 13.61% |
| DeleteByID | **36** | 60 | 57 | 40 | 1 | 0.00% |
| BatchCreate100 | **7,508** | 8,344 | 53,274 | 14,026 | 1 | 0.00% |
| BatchRead100 | **3,915** | 5,441 | 10,564 | 6,630 | 1 | 0.00% |
| BatchRead100Complex | **5,008** | 6,537 | 11,662 | 7,727 | 1 | 0.00% |
| BatchUpdate100 | **277** | 372 | 654 | 408 | 1 | 0.00% |
| BatchDelete100 | **147** | 279 | 482 | 270 | 1 | 0.00% |

## Summary

- SQLite `ns/op`: `lorm` ranks first in 6 of 10 cases.
- MySQL `ns/op`: `lorm` ranks first in 9 of 10 cases.
- PostgreSQL `ns/op`: `lorm` ranks first in 8 of 10 cases.
- `B/op`: `lorm` has the lowest allocation bytes in 9 of 10 cases on each backend.
- `allocs/op`: `lorm` has the lowest allocation count in 9 of 10 cases on each backend.
