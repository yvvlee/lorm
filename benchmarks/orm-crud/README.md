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

Recorded on May 15, 2026 with:

```bash
cd benchmarks/orm-crud
ORMCRUD_DB=mysql go test -run '^$' -bench . -benchmem -count=1
ORMCRUD_DB=postgres go test -run '^$' -bench . -benchmem -count=1
```

Environment:

- OS/arch: `darwin/arm64`
- CPU: `Apple M1 Pro`
- MySQL: `8.4.6`
- PostgreSQL: `18.1`

Treat these numbers as a directional snapshot. Re-run the suite on your target
database, schema, driver, and hardware before using them for a final decision.

### MySQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 1,131,710 | 1,637,111 | 1,135,509 | 1,175,650 |
| ReadByID | 915,563 | 944,430 | 860,244 | 891,313 |
| ReadByIDComplex | 864,774 | 844,448 | 877,075 | 893,103 |
| UpdateByID | 1,129,918 | 1,550,989 | 1,129,751 | 2,246,057 |
| DeleteByID | 1,031,749 | 1,519,377 | 1,040,008 | 1,002,177 |
| BatchCreate100 | 8,568,069 | 9,221,738 | 9,517,663 | 9,264,969 |
| BatchRead100 | 2,214,064 | 2,357,154 | 2,673,822 | 2,458,871 |
| BatchRead100Complex | 2,828,336 | 3,037,882 | 3,343,130 | 2,979,974 |
| BatchUpdate100 | 6,963,767 | 7,179,490 | 6,770,990 | 6,630,287 |
| BatchDelete100 | 5,415,382 | 5,366,887 | 5,024,017 | 5,089,361 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 7,906 | 11,072 | 7,761 | 8,869 |
| ReadByID | 10,816 | 8,151 | 11,673 | 10,240 |
| ReadByIDComplex | 11,520 | 8,857 | 12,649 | 11,504 |
| UpdateByID | 10,716 | 13,613 | 10,934 | 16,944 |
| DeleteByID | 1,791 | 5,692 | 2,897 | 1,977 |
| BatchCreate100 | 562,427 | 528,155 | 1,572,787 | 864,439 |
| BatchRead100 | 252,142 | 341,609 | 515,275 | 359,096 |
| BatchRead100Complex | 322,829 | 413,005 | 613,305 | 484,603 |
| BatchUpdate100 | 27,510 | 32,616 | 34,765 | 34,872 |
| BatchDelete100 | 18,927 | 25,336 | 31,706 | 26,912 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 134 | 154 | 129 | 172 |
| ReadByID | 211 | 142 | 237 | 228 |
| ReadByIDComplex | 236 | 167 | 262 | 253 |
| UpdateByID | 179 | 171 | 210 | 335 |
| DeleteByID | 34 | 66 | 56 | 45 |
| BatchCreate100 | 8,554 | 8,866 | 49,267 | 12,662 |
| BatchRead100 | 5,759 | 7,185 | 12,758 | 8,276 |
| BatchRead100Complex | 8,260 | 9,687 | 15,258 | 10,776 |
| BatchUpdate100 | 268 | 372 | 406 | 295 |
| BatchDelete100 | 133 | 272 | 268 | 163 |

### PostgreSQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 336,537 | 656,602 | 348,557 | 324,479 |
| ReadByID | 219,798 | 213,939 | 225,971 | 219,507 |
| ReadByIDComplex | 218,168 | 220,517 | 227,949 | 219,516 |
| UpdateByID | 321,847 | 669,114 | 654,825 | 880,432 |
| DeleteByID | 303,406 | 627,643 | 297,395 | 296,478 |
| BatchCreate100 | 2,908,882 | 2,980,674 | 4,608,589 | 2,782,881 |
| BatchRead100 | 987,042 | 1,237,686 | 1,407,165 | 1,073,514 |
| BatchRead100Complex | 1,519,387 | 1,882,029 | 2,038,405 | 1,636,961 |
| BatchUpdate100 | 721,963 | 1,034,409 | 940,146 | 723,326 |
| BatchDelete100 | 562,371 | 833,474 | 540,432 | 521,771 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 8,041 | 11,868 | 9,150 | 8,673 |
| ReadByID | 11,900 | 8,502 | 12,621 | 10,584 |
| ReadByIDComplex | 12,893 | 9,505 | 13,872 | 12,120 |
| UpdateByID | 10,485 | 12,774 | 11,816 | 16,670 |
| DeleteByID | 1,833 | 5,680 | 3,010 | 1,903 |
| BatchCreate100 | 821,761 | 790,847 | 2,016,207 | 1,129,679 |
| BatchRead100 | 293,065 | 381,044 | 546,975 | 398,889 |
| BatchRead100Complex | 391,194 | 479,972 | 672,533 | 551,907 |
| BatchUpdate100 | 44,131 | 47,378 | 54,686 | 49,762 |
| BatchDelete100 | 35,910 | 40,955 | 50,219 | 43,219 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 118 | 149 | 172 | 177 |
| ReadByID | 228 | 156 | 269 | 243 |
| ReadByIDComplex | 253 | 181 | 294 | 268 |
| UpdateByID | 158 | 144 | 243 | 346 |
| DeleteByID | 31 | 60 | 59 | 40 |
| BatchCreate100 | 7,323 | 8,039 | 52,973 | 13,723 |
| BatchRead100 | 6,186 | 7,609 | 12,803 | 8,798 |
| BatchRead100Complex | 8,686 | 10,111 | 15,303 | 11,298 |
| BatchUpdate100 | 271 | 369 | 643 | 405 |
| BatchDelete100 | 146 | 279 | 484 | 270 |

## Summary

- MySQL `ns/op`: `lorm` is fastest in 4 of 10 cases, `xorm` in 3,
  `ent` in 2, and `gorm` in 1.
- PostgreSQL `ns/op`: `lorm` is fastest in 5 of 10 cases, `ent` in 4,
  and `gorm` in 1.
- `B/op`: `lorm` has the lowest allocation bytes in 6 of 10 MySQL cases and
  7 of 10 PostgreSQL cases.
