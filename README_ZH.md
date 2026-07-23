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
`LormFieldPtr()` 和 `LormModelDescriptor()` 等方法。

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

_, err = lorm.Insert[*User](engine).
	AddModel(user).
	Exec(ctx)

savedUser, err := lorm.Query[*User](engine).
	Where(builder.Eq{u.Fields().ID(): user.ID}).
	Get(ctx)

_, err = lorm.Update[*User](engine).
	ID(user.ID).
	SetMap(map[string]any{
		u.Fields().Name(): "Jane Doe",
	}).
	Exec(ctx)

_, err = lorm.Delete[*User](engine).
	ID(user.ID).
	Exec(ctx)
```

使用基于模型的 API 前，必须先完成代码生成。

> **更新说明**：`Update.SetModel(model)` 会执行整行更新。零值字段也会被写回，
> 所以做部分更新时应优先使用 `SetMap` 或 `Set`。

> **Where 说明**：`builder.Eq{field: value}` 始终生成 `field = ?`，
> 并把 `value` 作为一个参数传给驱动。它不会把 `nil` 改写成 `IS NULL`，
> 不会解指针，不会调用 `driver.Valuer`，也不会把切片展开成 `IN (...)`。
> 需要空值判断时，显式使用 `builder.IsNull(field)` 或
> `builder.IsNotNull(field)`。需要成员判断时，显式使用 `builder.In` 或
> `builder.NotIn`。

> **Insert 说明**：批量插入只有在驱动能逐行返回生成主键时才会回填 ID。只支持
> `LastInsertId` 的方言不会为多行插入推算每条记录的主键。

> **Get 说明**：`Query.Get` 和 Repository 的 `Get` 类方法查不到行时，会返回
> `T` 的零值和 `nil` 错误。对 `*User` 这类指针模型来说，返回值就是
> `nil, nil`。

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
	_, err := lorm.Insert[*User](engine).
		AddModel(&User{Name: "User 1", Email: "user1@example.com"}).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = lorm.Insert[*User](engine).
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
		Repository: lorm.NewRepository[*User](engine),
	}
}

func (r *UserRepositoryImpl) PageGmailUsers(ctx context.Context, page, size uint64) ([]*User, uint64, error) {
	var u User
	return lorm.Query[*User](r.Engine).
		Where(builder.Like(u.Fields().Email(), "%@gmail.com")).
		OrderBy(u.Fields().ID() + " DESC").
		Page(ctx, page, size)
}
```

> **说明**：`Lock` 和 `LockByField` 会在查询后面追加 `FOR UPDATE`。
> 它们只有放在 `Engine.TX(...)` 或 `Engine.TXWithOptions(...)` 里才有实际的加锁意义。
> 不在事务里调用时，数据库不会在语句结束后继续保留这把行锁。

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

roles, err := lorm.Query[*UserRole](engine).
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

下面这组结果采集于 2026 年 5 月 15 日，执行命令为：

```bash
cd benchmarks/orm-crud
ORMCRUD_DB=mysql go test -run '^$' -bench . -benchmem -count=1
ORMCRUD_DB=postgres go test -run '^$' -bench . -benchmem -count=1
```

测试环境：

- OS/架构：`darwin/arm64`
- CPU：`Apple M1 Pro`
- MySQL：`8.4.6`
- PostgreSQL：`18.1`

MySQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | **1,131,710** | 1,637,111 | 1,135,509 | 1,175,650 |
| ReadByID | 915,563 | 944,430 | **860,244** | 891,313 |
| ReadByIDComplex | 864,774 | **844,448** | 877,075 | 893,103 |
| UpdateByID | 1,129,918 | 1,550,989 | **1,129,751** | 2,246,057 |
| DeleteByID | 1,031,749 | 1,519,377 | 1,040,008 | **1,002,177** |
| BatchCreate100 | **8,568,069** | 9,221,738 | 9,517,663 | 9,264,969 |
| BatchRead100 | **2,214,064** | 2,357,154 | 2,673,822 | 2,458,871 |
| BatchRead100Complex | **2,828,336** | 3,037,882 | 3,343,130 | 2,979,974 |
| BatchUpdate100 | 6,963,767 | 7,179,490 | 6,770,990 | **6,630,287** |
| BatchDelete100 | 5,415,382 | 5,366,887 | **5,024,017** | 5,089,361 |

PostgreSQL，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | 336,537 | 656,602 | 348,557 | **324,479** |
| ReadByID | 219,798 | **213,939** | 225,971 | 219,507 |
| ReadByIDComplex | **218,168** | 220,517 | 227,949 | 219,516 |
| UpdateByID | **321,847** | 669,114 | 654,825 | 880,432 |
| DeleteByID | 303,406 | 627,643 | 297,395 | **296,478** |
| BatchCreate100 | 2,908,882 | 2,980,674 | 4,608,589 | **2,782,881** |
| BatchRead100 | **987,042** | 1,237,686 | 1,407,165 | 1,073,514 |
| BatchRead100Complex | **1,519,387** | 1,882,029 | 2,038,405 | 1,636,961 |
| BatchUpdate100 | **721,963** | 1,034,409 | 940,146 | 723,326 |
| BatchDelete100 | 562,371 | 833,474 | 540,432 | **521,771** |

这次结果可以概括为：

- 在 MySQL 上，`lorm` 在 10 个 `ns/op` 场景里有 4 个最快：
  单行创建、批量创建、批量读取和复杂批量读取。
- 在 PostgreSQL 上，`lorm` 在 10 个 `ns/op` 场景里有 5 个最快，
  包括复杂单行读取、单行更新、批量读取、复杂批量读取和批量更新。
- 这次测试里，`lorm` 在 MySQL 的 10 个场景里有 6 个拿到最低 `B/op`，
  在 PostgreSQL 的 10 个场景里有 7 个拿到最低 `B/op`。

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
- `updated`：插入和更新时自动填充零值字段
- `version`：更新时自动递增的乐观锁版本字段

生成器的几个关键行为：

- 表名默认使用 snake_case，可以通过嵌入的 `lorm.UnimplementedTable`
  tag 覆盖。
- 字段名默认使用 snake_case，也可以通过字段 tag 覆盖。
- 如果字段需要 `lorm` tag，请单独写一行，不要和其他字段写成
  `A, B int` 这种合并声明。
- 内嵌结构体会被展开到生成的字段访问器中。
- 给内嵌结构体加 tag 可以为展开后的字段名加前缀。

## 贡献

欢迎提 issue 或提交 pull request。

## 许可证

本项目基于 MIT 许可证发布，详见 [LICENSE](LICENSE)。
