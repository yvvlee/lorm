# LORM - 轻量级 Go ORM

[English](README.md)

LORM 是一个为 Go 设计的轻量级 ORM。它尽量保持 API 简洁、SQL 显式，并
通过代码生成提供模型元数据和类型化字段访问器。

## 为什么选择 LORM

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

这会生成 `_lorm_gen.go` 文件，包含 `TableName()`、`Fields()`、`New()`、
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
	Where(builder.Eq{u.Fields().ID(): user.ID}).
	Get(ctx)

_, err = engine.Update[*User]().
	ID(user.ID).
	SetMap(map[string]any{
		u.Fields().Name(): "Jane Doe",
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
	Select(u.Fields().ID()).
	FindCols[int64](ctx)

count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[uint64](ctx)

ids, total, err := engine.Query[*User]().
	Select(u.Fields().ID()).
	OrderBy(u.Fields().ID()).
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
		Where(builder.Like(u.Fields().Email(), "%@gmail.com")).
		OrderBy(u.Fields().ID() + " DESC").
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

下面这组结果采集于 2026 年 8 月 20 日，使用 Go 1.27.0。
每个 ORM 都在独立的 `go test` 进程中执行一次。

SQLite 按 ORM 分别执行：

```bash
for orm in lorm gorm xorm ent; do
  CGO_ENABLED=1 go test -run '^$' -bench "/${orm}$" -benchmem -count=1
done
```

每次执行 MySQL ORM 前都会重启 MySQL 容器。等待 `mysqladmin ping`
成功后，再固定等待 10 秒：

```bash
docker restart mysql
# 等待 mysqladmin ping 成功。
sleep 10
ORMCRUD_DB=mysql go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

每次执行 PostgreSQL ORM 前都会重启对应容器，并等待 `pg_isready`
成功：

```bash
docker restart postgres
ORMCRUD_DB=postgres go test -run '^$' -bench '/<orm>$' -benchmem -count=1
```

将 `<orm>` 分别替换为 `lorm`、`gorm`、`xorm` 和 `ent`。
每个 ORM 使用独立的 benchmark 数据库。完整的有限次数就绪检查命令见
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
| Create | 320,185 | 279,823 | 291,430 | **270,549** | 4 | 18.35% |
| ReadByID | **21,189** | 24,540 | 30,467 | 26,395 | 1 | 0.00% |
| ReadByIDComplex | **22,487** | 26,886 | 32,928 | 29,012 | 1 | 0.00% |
| UpdateByID | 342,819 | 346,223 | 324,730 | **288,283** | 3 | 18.92% |
| DeleteByID | 344,690 | 310,801 | 336,944 | **273,470** | 4 | 26.04% |
| BatchCreate100 | **1,198,248** | 1,326,222 | 3,226,068 | 1,510,362 | 1 | 0.00% |
| BatchRead100 | **521,933** | 759,109 | 894,268 | 583,942 | 1 | 0.00% |
| BatchRead100Complex | **761,215** | 992,272 | 1,140,149 | 841,553 | 1 | 0.00% |
| BatchUpdate100 | 398,032 | 450,038 | **374,384** | 401,475 | 2 | 6.32% |
| BatchDelete100 | 690,346 | 765,703 | **591,269** | 595,752 | 3 | 16.76% |

MySQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent | lorm 名次 | 距第一名 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **906,321** | 1,361,265 | 973,385 | 928,810 | 1 | 0.00% |
| ReadByID | **240,317** | 302,950 | 284,257 | 249,565 | 1 | 0.00% |
| ReadByIDComplex | **245,672** | 298,108 | 305,488 | 248,823 | 1 | 0.00% |
| UpdateByID | **879,429** | 1,410,948 | 879,682 | 1,440,935 | 1 | 0.00% |
| DeleteByID | 822,350 | 1,031,374 | 871,111 | **751,133** | 2 | 9.48% |
| BatchCreate100 | 7,846,680 | 8,017,562 | **7,364,879** | 8,164,799 | 2 | 6.54% |
| BatchRead100 | 2,297,721 | 3,442,311 | 1,955,529 | **1,543,769** | 3 | 48.84% |
| BatchRead100Complex | 3,716,681 | 5,032,195 | **2,309,288** | 3,182,881 | 3 | 60.94% |
| BatchUpdate100 | 5,398,099 | 5,773,743 | **4,933,102** | 6,097,586 | 2 | 9.43% |
| BatchDelete100 | **4,047,424** | 4,165,851 | 4,114,345 | 4,097,822 | 1 | 0.00% |

PostgreSQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent | lorm 名次 | 距第一名 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Create | **882,606** | 1,245,377 | 902,820 | 916,939 | 1 | 0.00% |
| ReadByID | 135,480 | **134,977** | 143,964 | 141,048 | 2 | 0.37% |
| ReadByIDComplex | **136,348** | 138,018 | 141,529 | 140,041 | 1 | 0.00% |
| UpdateByID | **959,347** | 1,063,873 | 1,250,379 | 1,339,732 | 1 | 0.00% |
| DeleteByID | 945,561 | 1,167,307 | 859,569 | **856,562** | 3 | 10.39% |
| BatchCreate100 | **7,620,672** | 7,793,676 | 8,074,182 | 8,180,815 | 1 | 0.00% |
| BatchRead100 | **648,760** | 907,110 | 1,115,124 | 759,521 | 1 | 0.00% |
| BatchRead100Complex | **937,986** | 1,268,048 | 1,491,468 | 1,065,715 | 1 | 0.00% |
| BatchUpdate100 | 1,325,303 | 1,463,390 | 1,505,688 | **1,323,338** | 2 | 0.15% |
| BatchDelete100 | 1,288,657 | 1,467,285 | 1,238,411 | **1,216,635** | 3 | 5.92% |

这次结果可以概括为：

- 在 SQLite 上，`lorm` 在 10 个 `ns/op` 场景里有 5 个最快。
- 在 MySQL 上，`lorm` 在 10 个 `ns/op` 场景里有 5 个最快。
- 在 PostgreSQL 上，`lorm` 在 10 个 `ns/op` 场景里有 6 个最快。
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
