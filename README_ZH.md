# LORM - 轻量级 Go ORM

[English](README.md)

LORM 是一个为 Go 设计的高性能轻量级 ORM。它尽量保持 API 简洁、SQL 显式，并
通过代码生成提供模型元数据和类型化字段访问器。

## 为什么选择 LORM

- 高性能且分配开销低：当前[基准测试](#benchmark)覆盖 SQLite、MySQL 和 PostgreSQL。LORM 在 30 个耗时场景中有 23 个最快，其他场景与第一名接近。每次操作内存占用（`B/op`）和内存分配次数（`allocs/op`）都各有 27 个场景排名第一
- 所有数据访问优先通过 `Repository`，覆盖简单 CRUD 和复杂查询
- SQL builder 用于 Repository 的具体实现，复杂 SQL 仍然保持显式可控
- 通过代码生成提供模型元数据，减少运行时反射映射开销
- 提供类型化字段访问器，复用列名更安全
- 不做自动关联加载、隐式 join 或隐藏式查询扩散
- 事务辅助函数会自动复用 `context.Context` 中已有的事务 session
- 支持结构化日志、占位符格式和标识符转义配置

## 设计理念

LORM 希望把数据访问放在 Repository 边界内。

简单 CRUD 优先使用 `Repository[T]`。它足以覆盖大多数单表业务场景，也能
明显提升开发效率，让应用代码保持短小、直接、容易维护。

复杂查询、报表查询、搜索页和自定义 join 也放在 Repository 的具体实现中。
SQL builder 的职责是帮助 Repository 构建复杂 SQL，并让最终查询形状保持
显式、可审查。
它的链式 API 设计借鉴了
[Squirrel](https://github.com/Masterminds/squirrel)，但 LORM 会把 builder
限制在 Repository 实现和自身生成的模型元数据内使用。

LORM 有意不提供自动关联加载、隐式 eager loading、lazy loading，或者“魔法式”
的模型关联查询。这类能力在生产环境里非常容易失控：查询开销容易被隐藏，SQL
形状不稳定，代码稍有改动就可能引入意外的 N+1 查询或性能回退。

因此 LORM 更偏向显式 join、显式选择列、显式划分查询边界。业务代码关注
业务语义，Repository 实现负责数据库细节。

## 数据库支持

- 第一优先级：MySQL/MariaDB、PostgreSQL
- 第二优先级：SQLite

推荐的 `database/sql` 驱动：

| 数据库 | 驱动包 | 传给 `NewEngine` 的 driver name |
| --- | --- | --- |
| MySQL/MariaDB | `github.com/go-sql-driver/mysql` | `mysql` |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | `pgx` |
| SQLite | `github.com/mattn/go-sqlite3` | `sqlite3` |

LORM 主 module 不会引入数据库驱动。应用只需要按需空导入自己使用的驱动：

```go
import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)
```

## 安装

LORM 需要 Go 1.27 或更高版本。

```bash
go get github.com/yvvlee/lorm
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

## 快速开始

### 1. 定义模型

```go
type User struct {
	lorm.UnimplementedTable
	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Email     string    `lorm:"email"`
	CreatedAt time.Time `lorm:"created_at,created"`
	UpdatedAt time.Time `lorm:"updated_at,updated"`
}
```

### 2. 生成辅助代码

```bash
lormgen ./...
```

这会生成 `_lorm_gen.go` 文件，包含 `TableName()`、`LormCols()`、`New()`、
`LormFieldPtr()`、`LormFieldValue()`、`LormModelDescriptor()`，以及表模型的
写入 Hook。

### 3. 初始化引擎

```go
engine, err := lorm.NewEngine(
	"mysql",
	"user:password@tcp(localhost:3306)/dbname?parseTime=true",
)
if err != nil {
	log.Fatal(err)
}
defer engine.Close()
```

### 4. 执行增删改查

```go
ctx := context.Background()
var u User

user := &User{
	Name:  "John Doe",
	Email: "john@example.com",
}

_, err = engine.Insert[*User]().
	AddModel(user).
	Exec(ctx)

savedUser, ok, err := engine.Query[*User]().
	Where(builder.Eq{u.LormCols().ID(): user.ID}).
	Get(ctx)

_, err = engine.Update[*User]().
	ID(user.ID).
	SetMap(map[string]any{
		u.LormCols().Name(): "Jane Doe",
	}).
	Exec(ctx)

_, err = engine.Delete[*User]().
	ID(user.ID).
	Exec(ctx)
```

使用基于模型的 API 前，必须先完成代码生成。

> **更新说明**：`Update.SetModel(model)` 会执行整行更新。零值字段也会被写回，
> 所以做部分更新时应优先使用 `SetMap` 或 `Set`。`SetModel` 不能和 `Set`、
> `SetMap` 混用。

> **写入安全说明**：`Update.Exec` 和 `Delete.Exec` 会尽量拒绝没有有效 `WHERE`
> 条件的语句。这个检查不能识别所有逻辑上恒真的条件，也不能替代业务层校验，
> 不应把它当作防止全表写的绝对保证。直接使用 `Engine.Exec` 执行原始 SQL 时，
> 不受这项检查保护。确实需要操作整张表时，必须显式调用 `AllowGlobalWrite()`；
> 调用方必须自行确认写入范围，并对数据安全负责。

> **Where 说明**：`builder.Eq{field: value}` 始终生成 `field = ?`，
> 并把 `value` 作为一个参数传给驱动。它不会把 `nil` 改写成 `IS NULL`，
> 不会解指针，不会调用 `driver.Valuer`，也不会把切片展开成 `IN (...)`。
> 需要空值判断时，显式使用 `builder.IsNull(field)` 或
> `builder.IsNotNull(field)`。需要成员判断时，显式使用 `builder.In` 或
> `builder.NotIn`。

> **列名说明**：在 SQL builder 中填入字段名时，优先使用模型生成的
> `LormCols()`，不要手写数据库列名字符串。这样可以复用字段映射结果，减少列名
> 拼写错误。涉及表别名时，先调用 `WithAlias()`。

```go
var u User
c := u.LormCols()

users, err := engine.Query[*User]().
	Where(builder.Eq{c.Email(): "alice@example.com"}).
	OrderBy(c.CreatedAt() + " DESC").
	Find(ctx)
// SQL（MySQL）：SELECT `id`, `name`, `email`, `created_at`, `updated_at` FROM `users` WHERE `email` = ? ORDER BY created_at DESC

_, err = engine.Update[*User]().
	ID(1).
	SetMap(map[string]any{c.Name(): "Alice Updated"}).
	Exec(ctx)
// SQL（MySQL）：UPDATE `users` SET `name` = ? WHERE `id` = ?
```

```go
var u User
c := u.LormCols().WithAlias("u")

ids, err := engine.Query[*User]().
	Select(c.ID()).
	From("users AS u").
	Where(builder.Like(c.Email(), "%@example.com")).
	OrderBy(c.ID() + " DESC").
	FindCols[int64](ctx)
// SQL（MySQL）：SELECT u.id FROM users AS u WHERE u.email LIKE ? ORDER BY u.id DESC
```

> **Insert 说明**：单条插入会在驱动支持 `RETURNING` 或 `LastInsertId` 时回填
> 生成 ID。批量插入默认不回填 ID。调用 `RequireIDBackfill()` 后，批量插入会在
> 一个事务中退化为逐条执行，并回填每个实际插入模型的 ID。自增主键为零值时，
> 插入语句会省略主键列。自增主键为非零值时，插入语句会显式写入该值。混合批次
> 会按输入中连续的主键状态分组，并在一个事务中执行。

> **Get 说明**：`Query.Get` 返回 `(T, bool, error)`。第二个返回值表示是否找到
> 记录。Repository 的 `Get` 类方法保持 `(T, error)`，查不到时返回 `T` 的零值。

`Query` 的泛型参数必须是模型指针。查询单列时，在终结方法上声明列值类型：

```go
ids, err := engine.Query[*User]().
	Select(u.LormCols().ID()).
	FindCols[int64](ctx)

count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[uint64](ctx)

ids, total, err := engine.Query[*User]().
	Select(u.LormCols().ID()).
	OrderBy(u.LormCols().ID()).
	PageCols[int64](ctx, page, size)
```

单列终结方法要求 SQL 结果严格只有一列。多列结果会直接返回错误。

Statement builder 是轻量级对象。每次数据库操作都应重新创建一条新的
`Query` / `Insert` / `Update` / `Delete` 调用链，不要在多个 goroutine
之间共享同一个 statement。
需要基于已有条件派生查询时，使用 `Clone()`。
终态方法仍然只会重置被调用的那个 statement。

## 示例

可运行、可直接参考的完整示例见 [example/README.md](example/README.md)。

示例在独立 module 中：

- `cd example && go run ./quickstart`
- `cd example && go run ./repository`
- `cd example && go run ./transaction`
- `cd example && go run ./custom_model`
- `cd example && go run ./custom_conversion`
- `cd example && go run ./json_field`
- `cd example && go run ./pagination`
- `cd example && go run ./optimistic_lock`
- `cd example && go run ./query_builder`

## 事务

`Engine.TX` 会开启事务，把带事务信息的 `context.Context` 传入回调，并在
回调返回后自动提交或回滚。

```go
err := engine.TX(context.Background(), func(ctx context.Context) error {
	_, err := engine.Insert[*User]().
		AddModel(&User{Name: "User 1", Email: "user1@example.com"}).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = engine.Insert[*User]().
		AddModel(&User{Name: "User 2", Email: "user2@example.com"}).
		Exec(ctx)
	return err
})
```

如果在事务回调内部再次调用 `TX`，LORM 会复用当前 `context` 里的事务
session，而不是再开启一个新事务。

如果需要传入 `sql.TxOptions`，比如隔离级别或只读事务，请使用
`Engine.TXWithOptions`。嵌套调用时仍然会复用当前 `context` 中已有的事务。

## Repository 辅助类型

`lorm.Repository[T]` 封装了常见的单表 CRUD 路径。强烈推荐在实现结构体中
内嵌 `lorm.Repository[T]`，然后通过接口按需暴露方法。

Repository 是推荐的数据访问边界：

- 业务代码依赖稳定接口，不直接依赖 ORM。
- 方法签名不和事务绑定，业务层决定是否开启事务。
- SQL builder 留在 Repository 的具体实现中使用。
- 单元测试可以直接 mock Repository 接口。
- 表结构、join 和查询细节不会泄漏到业务代码。

简单 CRUD 可以直接复用内置方法；复杂查询也继续放在 Repository 的具体实现中，
需要时在实现内部使用 SQL builder：

```go
type UserRepository interface {
	// 以下方法为常用方法，lorm.Repository[*User] 已实现，按需暴露
	Get(ctx context.Context, id any) (*User, error)
	GetByField(ctx context.Context, field string, value any) (*User, error)
	Lock(ctx context.Context, id any) (*User, error)
	LockByField(ctx context.Context, field string, value any) (*User, error)
	Exist(ctx context.Context, id any) (bool, error)
	ExistByField(ctx context.Context, field string, value any) (bool, error)
	Update(ctx context.Context, user *User) (rowsAffected int64, err error)
	UpdateMap(ctx context.Context, id any, data map[string]any) (rowsAffected int64, err error)
	Insert(ctx context.Context, user *User) (rowsAffected int64, err error)
	InsertAll(ctx context.Context, users []*User) (rowsAffected int64, err error)
	Delete(ctx context.Context, id any) (rowsAffected int64, err error)
	DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error)

	// 自定义方法，需在 UserRepositoryImpl 中自行实现
	PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error)
}

var _ UserRepository = (*UserRepositoryImpl)(nil)

type UserRepositoryImpl struct {
	*lorm.Repository[*User]
}

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: engine.Repository[*User](),
	}
}

func (r *UserRepositoryImpl) PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error) {
	var u User
	return r.Engine.Query[*User]().
		Where(builder.Like(u.LormCols().Email(), "%@gmail.com")).
		OrderBy(u.LormCols().ID() + " DESC").
		Page(ctx, page, size)
}
```

> **说明**：`Lock` 和 `LockByField` 会在查询后面追加 `FOR UPDATE`。
> 调用时必须传入 `Engine.TX(...)` 或 `Engine.TXWithOptions(...)` 回调中的
> `context`。不在事务里调用会直接返回错误。

## 自定义投影模型

当查询结果并不直接对应某一张表模型时，可以嵌入
`lorm.UnimplementedModel`。

```go
type UserRole struct {
	lorm.UnimplementedModel
	UserID   int64
	UserName string
	RoleName string
}

roles, err := engine.Query[*UserRole]().
	Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"r.name AS role_name",
	).
	From("user AS u").
	InnerJoin("role AS r ON u.role_id = r.id").
	Find(ctx)
```

与 `UnimplementedTable` 不同，`UnimplementedModel` 不会生成 `TableName()`
方法，所以这类查询需要显式调用 `From(...)`。

## 自定义字段转换

如果某个字段不适合直接按普通值或 JSON 存储，就让字段类型实现标准库的
数据库接口：

- 写入时实现 `driver.Valuer`
- 读取时实现 `sql.Scanner`

```go
import "database/sql/driver"

type CSVInts []int

var _ lorm.ScannerValuer = (*CSVInts)(nil)

func (c CSVInts) Value() (driver.Value, error) {
	return []byte("1,2,3"), nil
}

func (c *CSVInts) Scan(src any) error {
	// 把 "1,2,3" 还原成切片
	return nil
}

type Report struct {
	lorm.UnimplementedTable
	ID     int64   `lorm:"id,primary_key,auto_increment"`
	Title  string  `lorm:"title"`
	Scores CSVInts `lorm:"scores"`
}
```

LORM 写参数时会走 `driver.Valuer`。
查结果时会走 `sql.Scanner`。

`ScannerValuer` 组合了这两个标准库接口。
上面的编译期断言不是必需的，但能在程序运行前发现接口实现不完整。
LORM 不要求额外实现一套转换协议。

可运行示例见
[example/custom_conversion/main.go](example/custom_conversion/main.go)。

## 配置

```go
dialect := lorm.DefaultDialectConfig("pgx")
dialect.SupportsForUpdate = true

engine, err := lorm.NewEngine(
	"pgx",
	"postgres://user:password@localhost:5432/dbname?sslmode=disable",
	lorm.WithDialectConfig(dialect),
	lorm.WithMaxIdleConns(10),
	lorm.WithMaxOpenConns(100),
	lorm.WithConnMaxLifetime(time.Hour),
	lorm.WithConnMaxIdleTime(30*time.Minute),
	lorm.WithLogger(customLogger),
)
```

`DialectConfig` 集中保存数据库方言行为：占位符格式、表名和字段名转义、
`RETURNING`、`LastInsertId`、`FOR UPDATE`、`INSERT IGNORE` 语法。
默认值会按 driver name 自动选择。
需要整体替换时用 `WithDialectConfig`。
只改一项时仍然可以用 `WithPlaceholderFormat`、`WithEscaper` 和 `WithSupports...`。

## Benchmark

benchmark 套件位于 [benchmarks/orm-crud](benchmarks/orm-crud)。

下面的表格重新采集于 2026 年 8 月 21 日，使用 Go 1.27.0。
三个数据库都执行三轮。每个 ORM 使用独立的 `go test` 进程。SQLite 每个
benchmark case 使用新的临时数据库。每次执行 MySQL 和 PostgreSQL ORM 前，
都会重启对应容器，检查就绪状态，再等待 10 秒。所有测试都使用 1 秒
benchmark 窗口。完整的方法和 `B/op`、`allocs/op` 结果见 benchmark 套件文档。

SQLite：

```bash
for round in 1 2 3; do
  for orm in lorm gorm xorm ent; do
    CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -benchtime=1s -count=1
  done
done
```

每次执行 MySQL ORM 前都会重启 MySQL 容器。等待 `mysqladmin ping`
成功后，再固定等待 10 秒：

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

PostgreSQL 使用相同的循环。每次执行前重启 `postgres`，并等待 `pg_isready`
成功：

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

每个 ORM 使用独立的 benchmark 数据库。有限次数的就绪检查命令也见
benchmark 套件文档。

测试环境：

- OS/架构：`darwin/arm64`
- CPU：`Apple M1 Pro`
- Go：`go1.27.0`
- MySQL：`9.7.0`
- PostgreSQL：`18.6`
- Go 1.27 下，`sonic/ast` 降级为 `encoding/json`。

名次在四个 ORM 之间按数值从小到大计算。差距按
`(lorm / 第一名 - 1) * 100%` 计算。

SQLite，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent | lorm 名次 | 距第一名 |
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

MySQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent | lorm 名次 | 距第一名 |
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

PostgreSQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent | lorm 名次 | 距第一名 |
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

这次结果可以概括为：

- 在 SQLite 上，`lorm` 在 10 个 `ns/op` 场景里有 6 个最快。
- 在 MySQL 上，`lorm` 在 10 个 `ns/op` 场景里有 9 个最快。
- 在 PostgreSQL 上，`lorm` 在 10 个 `ns/op` 场景里有 8 个最快。
- 这次测试里，`lorm` 在 SQLite、MySQL 和 PostgreSQL 的 10 个场景里
  都有 9 个拿到最低 `B/op`。`allocs/op` 也分别有
  9、9 和 9 个排名第一。

这些数字更适合作为趋势参考，而不是所有场景下的绝对结论。做技术选型前，
建议在你的目标数据库、schema、驱动和硬件环境上重新跑一遍。

更完整的 benchmark 范围、环境和结果表格见
[benchmarks/orm-crud/README.md](benchmarks/orm-crud/README.md)。

## lormgen

`lormgen` 会扫描嵌入了 `lorm.UnimplementedTable` 或
`lorm.UnimplementedModel` 的结构体，并生成 LORM 运行所需的方法。

用法：

```bash
lormgen [flags] <directory|file>...
```

常用参数：

- `--field-mapper`：`snake`、`camel` 或 `same`
- `--table-mapper`：`snake`、`camel` 或 `same`
- `--table-prefix`：生成表名前缀
- `--table-suffix`：生成表名后缀
- `--tag-key`：结构体 tag key，默认 `lorm`
- `--file-suffix`：生成文件后缀，默认 `_lorm_gen`
- `--ignore`：忽略文件的 glob 模式，可重复指定

示例：

```bash
lormgen .
lormgen ./models/...
lormgen --table-prefix=t_ --table-suffix=_tab --field-mapper=camel ./models
lormgen --ignore="*_temp.go" --ignore="*_old.go" ./models
```

内置 tag：

- `primary_key`：主键字段
- `auto_increment`：自增字段
- `json`：按 JSON 存储字段
- `created`：插入时自动填充零值字段
- `updated`：插入时填充零值字段，模型更新时总是刷新
- `version`：更新时自动递增的乐观锁版本字段

生成器的几个关键行为：

- 表名默认使用 snake_case，可以通过嵌入的 `lorm.UnimplementedTable`
  tag 覆盖。
- 字段名默认使用 snake_case，也可以通过字段 tag 覆盖。
- 如果字段需要 `lorm` tag，请单独写一行，不要和其他字段写成
  `A, B int` 这种合并声明。
- 内嵌结构体会被展开到生成的字段访问器中。
- 给内嵌结构体加 tag 可以为展开后的字段名加前缀。
- 生成器会拒绝未知的 tag 项和重复的数据库列名。内嵌结构体展开后产生的
  列名冲突也会报错。
- `auto_increment` 字段必须同时标记 `primary_key`。
- 每个模型最多只能有一个 `version` 字段。
- `created` 和 `updated` 只支持 `time.Time`、`sql.NullTime`、`int64`、
  `uint64`、`uint32`、`uint`、64 位目标上的 `int`、`string` 及其一层指针。
  整数保存 Unix 秒。字符串使用 `time.DateTime` 格式。
- `int8`、`uint8`、`int16`、`uint16` 和 `int32` 不能作为托管时间字段。
- 包含 `int` 托管时间字段的生成文件会拒绝在 32 位目标上编译。

## 贡献

欢迎提 issue 或提交 pull request。

## 许可证

本项目基于 MIT 许可证发布，详见 [LICENSE](LICENSE)。
