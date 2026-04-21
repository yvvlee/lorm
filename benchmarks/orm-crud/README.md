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

Recorded on April 21, 2026 with:

```bash
cd benchmarks/orm-crud
ORMCRUD_DB=mysql go test -run '^$' -bench . -benchmem -count=1
ORMCRUD_DB=postgres go test -run '^$' -bench . -benchmem -count=1
```

Environment:

- OS/arch: `darwin/arm64`
- CPU: `Apple M1 Pro`

Treat these numbers as a directional snapshot. Re-run the suite on your target
database, schema, driver, and hardware before using them for a final decision.

### MySQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 2,429,837 | 2,907,363 | 2,053,178 | 2,086,018 |
| ReadByID | 1,311,045 | 1,530,371 | 1,395,336 | 1,381,610 |
| ReadByIDComplex | 1,394,428 | 1,364,298 | 1,397,521 | 1,496,263 |
| UpdateByID | 1,974,881 | 2,755,859 | 2,095,659 | 3,580,909 |
| DeleteByID | 1,684,043 | 2,666,697 | 1,951,541 | 1,747,895 |
| BatchCreate100 | 10,298,795 | 11,817,061 | 13,512,985 | 13,390,454 |
| BatchRead100 | 3,122,435 | 3,210,818 | 3,701,418 | 3,355,445 |
| BatchRead100Complex | 3,407,432 | 4,217,056 | 4,312,690 | 4,341,693 |
| BatchUpdate100 | 8,791,228 | 10,066,574 | 8,563,490 | 9,281,696 |
| BatchDelete100 | 6,529,993 | 8,409,838 | 6,407,384 | 6,607,578 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 7,975 | 11,080 | 7,582 | 8,855 |
| ReadByID | 8,621 | 8,146 | 12,874 | 10,240 |
| ReadByIDComplex | 9,325 | 8,863 | 13,867 | 11,504 |
| UpdateByID | 10,284 | 15,396 | 9,805 | 16,885 |
| DeleteByID | 1,663 | 5,694 | 4,121 | 1,975 |
| BatchCreate100 | 561,553 | 527,243 | 1,555,228 | 862,694 |
| BatchRead100 | 250,246 | 341,552 | 513,583 | 359,094 |
| BatchRead100Complex | 320,932 | 412,785 | 611,718 | 484,598 |
| BatchUpdate100 | 27,334 | 34,019 | 33,769 | 34,958 |
| BatchDelete100 | 18,786 | 25,337 | 32,907 | 26,912 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 146 | 152 | 124 | 170 |
| ReadByID | 177 | 142 | 301 | 228 |
| ReadByIDComplex | 202 | 167 | 326 | 253 |
| UpdateByID | 185 | 170 | 194 | 332 |
| DeleteByID | 35 | 66 | 119 | 44 |
| BatchCreate100 | 8,561 | 8,861 | 48,862 | 12,650 |
| BatchRead100 | 5,726 | 7,184 | 12,858 | 8,276 |
| BatchRead100Complex | 8,227 | 9,686 | 15,358 | 10,776 |
| BatchUpdate100 | 272 | 374 | 398 | 299 |
| BatchDelete100 | 133 | 272 | 331 | 163 |

### PostgreSQL

`ns/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 507,712 | 1,065,266 | 554,114 | 587,933 |
| ReadByID | 345,650 | 335,957 | 322,547 | 343,195 |
| ReadByIDComplex | 328,225 | 320,144 | 352,709 | 347,308 |
| UpdateByID | 581,154 | 961,128 | 580,756 | 1,366,905 |
| DeleteByID | 449,485 | 868,816 | 522,107 | 505,384 |
| BatchCreate100 | 4,289,600 | 4,582,903 | 5,865,117 | 4,271,954 |
| BatchRead100 | 1,360,090 | 1,671,265 | 1,786,444 | 1,365,145 |
| BatchRead100Complex | 1,800,473 | 2,091,973 | 2,249,731 | 1,942,180 |
| BatchUpdate100 | 1,094,846 | 1,857,574 | 1,640,801 | 1,889,459 |
| BatchDelete100 | 962,203 | 1,266,631 | 879,589 | 972,662 |

`B/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 8,144 | 11,860 | 8,981 | 8,680 |
| ReadByID | 9,705 | 8,490 | 13,824 | 10,585 |
| ReadByIDComplex | 10,699 | 9,503 | 15,089 | 12,121 |
| UpdateByID | 10,145 | 14,576 | 10,846 | 16,621 |
| DeleteByID | 1,705 | 5,696 | 4,236 | 1,902 |
| BatchCreate100 | 823,179 | 790,351 | 1,997,349 | 1,129,284 |
| BatchRead100 | 291,180 | 380,805 | 545,372 | 398,888 |
| BatchRead100Complex | 389,321 | 479,749 | 670,912 | 551,909 |
| BatchUpdate100 | 43,927 | 48,771 | 53,522 | 50,621 |
| BatchDelete100 | 35,765 | 40,942 | 51,455 | 43,211 |

`allocs/op`:

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 132 | 149 | 168 | 176 |
| ReadByID | 194 | 156 | 333 | 243 |
| ReadByIDComplex | 219 | 181 | 358 | 268 |
| UpdateByID | 164 | 146 | 238 | 345 |
| DeleteByID | 32 | 60 | 122 | 40 |
| BatchCreate100 | 7,334 | 8,036 | 52,571 | 13,719 |
| BatchRead100 | 6,153 | 7,608 | 12,903 | 8,798 |
| BatchRead100Complex | 8,653 | 10,110 | 15,403 | 11,298 |
| BatchUpdate100 | 273 | 371 | 629 | 412 |
| BatchDelete100 | 146 | 279 | 547 | 270 |
