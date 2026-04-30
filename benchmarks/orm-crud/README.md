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

Recorded on April 30, 2026 with:

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
| Create | 1,913,144 | 2,613,786 | 1,788,054 | 1,817,208 |
| ReadByID | 1,262,759 | 1,258,343 | 1,321,836 | 1,199,787 |
| ReadByIDComplex | 1,132,911 | 1,386,360 | 1,353,234 | 1,167,020 |
| UpdateByID | 1,690,392 | 2,496,637 | 1,486,555 | 4,337,282 |
| DeleteByID | 1,791,673 | 2,577,342 | 1,078,675 | 1,469,701 |
| BatchCreate100 | 9,816,171 | 10,372,265 | 14,042,917 | 11,062,927 |
| BatchRead100 | 3,668,220 | 4,154,132 | 4,040,834 | 3,969,475 |
| BatchRead100Complex | 4,653,190 | 4,950,072 | 4,541,484 | 4,436,015 |
| BatchUpdate100 | 8,032,387 | 8,486,618 | 6,988,625 | 8,843,135 |
| BatchDelete100 | 5,277,316 | 6,698,076 | 2,085,592 | 5,679,043 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 7,881 | 10,992 | 7,757 | 8,864 |
| ReadByID | 10,815 | 8,155 | 11,673 | 10,240 |
| ReadByIDComplex | 11,519 | 8,866 | 12,649 | 11,504 |
| UpdateByID | 10,277 | 15,396 | 9,982 | 16,887 |
| DeleteByID | 1,791 | 5,708 | 4,972 | 1,975 |
| BatchCreate100 | 561,968 | 527,086 | 1,573,028 | 862,170 |
| BatchRead100 | 252,106 | 341,115 | 515,174 | 359,089 |
| BatchRead100Complex | 322,755 | 412,789 | 613,322 | 484,599 |
| BatchUpdate100 | 27,347 | 33,963 | 33,801 | 34,949 |
| BatchDelete100 | 18,929 | 25,336 | 34,171 | 26,912 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 132 | 153 | 129 | 171 |
| ReadByID | 211 | 142 | 237 | 228 |
| ReadByIDComplex | 236 | 167 | 262 | 253 |
| UpdateByID | 173 | 171 | 198 | 333 |
| DeleteByID | 34 | 66 | 135 | 44 |
| BatchCreate100 | 8,551 | 8,862 | 49,261 | 12,656 |
| BatchRead100 | 5,759 | 7,182 | 12,757 | 8,276 |
| BatchRead100Complex | 8,259 | 9,686 | 15,258 | 10,776 |
| BatchUpdate100 | 263 | 374 | 398 | 299 |
| BatchDelete100 | 133 | 272 | 347 | 163 |

### PostgreSQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 518,071 | 947,152 | 555,380 | 591,445 |
| ReadByID | 298,036 | 431,630 | 309,724 | 373,995 |
| ReadByIDComplex | 344,650 | 474,724 | 317,436 | 399,537 |
| UpdateByID | 713,365 | 989,863 | 679,395 | 1,097,098 |
| DeleteByID | 463,347 | 863,456 | 276,317 | 462,638 |
| BatchCreate100 | 4,676,179 | 5,024,113 | 5,872,695 | 4,653,561 |
| BatchRead100 | 1,471,637 | 1,770,276 | 1,787,619 | 1,459,470 |
| BatchRead100Complex | 1,876,657 | 2,126,904 | 2,403,127 | 1,882,766 |
| BatchUpdate100 | 902,407 | 1,370,367 | 1,499,549 | 1,753,552 |
| BatchDelete100 | 809,539 | 1,429,966 | 531,330 | 950,048 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 8,038 | 11,848 | 9,162 | 8,678 |
| ReadByID | 11,899 | 8,502 | 12,622 | 10,585 |
| ReadByIDComplex | 12,892 | 9,500 | 13,872 | 12,121 |
| UpdateByID | 10,126 | 14,575 | 11,013 | 16,636 |
| DeleteByID | 1,833 | 5,679 | 5,081 | 1,902 |
| BatchCreate100 | 821,140 | 789,922 | 2,015,561 | 1,129,008 |
| BatchRead100 | 293,057 | 380,924 | 546,988 | 398,888 |
| BatchRead100Complex | 391,187 | 479,878 | 672,518 | 551,911 |
| BatchUpdate100 | 43,949 | 48,732 | 53,480 | 50,624 |
| BatchDelete100 | 35,904 | 40,947 | 52,716 | 43,217 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 118 | 148 | 171 | 176 |
| ReadByID | 228 | 156 | 269 | 243 |
| ReadByIDComplex | 253 | 181 | 294 | 268 |
| UpdateByID | 152 | 146 | 241 | 346 |
| DeleteByID | 31 | 60 | 140 | 40 |
| BatchCreate100 | 7,317 | 8,033 | 52,971 | 13,716 |
| BatchRead100 | 6,186 | 7,608 | 12,803 | 8,798 |
| BatchRead100Complex | 8,686 | 10,111 | 15,303 | 11,298 |
| BatchUpdate100 | 264 | 371 | 629 | 412 |
| BatchDelete100 | 146 | 279 | 565 | 270 |
