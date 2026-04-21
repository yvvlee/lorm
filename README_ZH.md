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

LORM 有意不提供自动关联加载、隐式 eager loading、lazy loading，或者“魔法式”
的模型关联查询。这类能力在生产环境里非常容易失控：查询开销容易被隐藏，SQL
形状不稳定，代码稍有改动就可能引入意外的 N+1 查询或性能回退。

因此 LORM 更偏向显式 join、显式选择列、显式划分查询边界。业务代码关注
业务语义，Repository 实现负责数据库细节。

## 数据库支持

- 第一优先级：MySQL/MariaDB、PostgreSQL
- 第二优先级：SQLite
- 尽力支持：SQL Server、Oracle

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

_, err = lorm.DeleteModel[*User](engine).
	ID(user.ID).
	Exec(ctx)
```

使用基于模型的 API 前，必须先完成代码生成。

Statement builder 是轻量级对象。每次数据库操作都应重新创建一条新的
`Query` / `Insert` / `Update` / `Delete` 调用链，不要在多个 goroutine
之间共享同一个 statement。

## 示例

可运行、可直接参考的完整示例见 [example/README.md](example/README.md)。

- `go run ./example/quickstart`
- `go run ./example/repository`
- `go run ./example/transaction`
- `go run ./example/custom_model`
- `go run ./example/json_field`
- `go run ./example/pagination`
- `go run ./example/optimistic_lock`
- `go run ./example/query_builder`

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
	Get(ctx context.Context, id int64) (*User, error)
	GetByField(ctx context.Context, field string, value any) (*User, error)
	Lock(ctx context.Context, id int64) (*User, error)
	LockByField(ctx context.Context, field string, value any) (*User, error)
	Exist(ctx context.Context, id int64) (bool, error)
	ExistByField(ctx context.Context, field string, value any) (bool, error)
	Update(ctx context.Context, user *User) (rowsAffected int64, err error)
	UpdateMap(ctx context.Context, id int64, data map[string]any) (rowsAffected int64, err error)
	Insert(ctx context.Context, user *User) (rowsAffected int64, err error)
	InsertAll(ctx context.Context, users []*User) (rowsAffected int64, err error)
	Delete(ctx context.Context, id int64) (rowsAffected int64, err error)
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

## 配置

```go
engine, err := lorm.NewEngine(
	"postgres",
	"postgres://user:password@localhost:5432/dbname?sslmode=disable",
	lorm.WithPlaceholderFormat(builder.Dollar),
	lorm.WithEscaper(names.NewQuoter('"', '"')),
	lorm.WithMaxIdleConns(10),
	lorm.WithMaxOpenConns(100),
	lorm.WithConnMaxLifetime(time.Hour),
	lorm.WithConnMaxIdleTime(30*time.Minute),
	lorm.WithLogger(customLogger),
)
```

## Benchmark

benchmark 套件位于 [benchmarks/orm-crud](benchmarks/orm-crud)。

下面这组结果采集于 2026 年 4 月 20 日，执行命令为：

```bash
cd benchmarks/orm-crud
go test -bench . -benchmem
```

测试环境：

- 后端：SQLite（默认值，即 `ORMCRUD_DB=sqlite`）
- OS/架构：`linux/amd64`
- CPU：`Intel(R) Core(TM) i5-9400F CPU @ 2.90GHz`

单行 CRUD，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| Create | **97,471** | 127,067 | 99,391 | 107,457 |
| ReadByID | **31,231** | 36,679 | 45,174 | 35,275 |
| UpdateByID | **85,727** | 104,658 | 93,461 | 123,181 |
| DeleteByID | 79,837 | 96,848 | 84,681 | **78,846** |

100 行批量 CRUD，`ns/op`，数值越小越好：

| Benchmark | lorm | gorm | xorm | ent |
| --- | ---: | ---: | ---: | ---: |
| BatchCreate100 | 1,139,752 | **724,101** | 4,025,133 | 853,383 |
| BatchRead100 | **346,392** | 427,952 | 725,096 | 411,414 |
| BatchUpdate100 | 211,472 | 211,800 | **185,189** | 186,621 |
| BatchDelete100 | 332,062 | 326,834 | 313,968 | **313,609** |

这次结果可以概括为：

- `lorm` 在单行 create、read、update 上最快，单行 delete 与 `ent`
  基本处于同一量级。
- `lorm` 在批量 read 上最快，且批量 create 明显快于 `xorm`。
- 这台机器上的批量 create 最快的是 `gorm`。
- 这次测试里，`lorm` 在 8 个 benchmark 里的 7 个场景都拿到了最低的 `B/op`。

这些数字更适合作为趋势参考，而不是所有场景下的绝对结论。做技术选型前，
建议在你的目标数据库、schema、驱动和硬件环境上重新跑一遍。

也可以切换到 MySQL 或 PostgreSQL 运行同一套 benchmark：

```bash
ORMCRUD_DB=mysql go test -bench . -benchmem
ORMCRUD_DB=postgres go test -bench . -benchmem
```

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
- 内嵌结构体会被展开到生成的字段访问器中。
- 给内嵌结构体加 tag 可以为展开后的字段名加前缀。

## 贡献

欢迎提 issue 或提交 pull request。

## 许可证

本项目基于 MIT 许可证发布，详见 [LICENSE](LICENSE)。
