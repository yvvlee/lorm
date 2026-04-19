# lorm - 轻量级 Golang ORM

lorm 是一个为 Go 语言设计的轻量级ORM库。它提供了一种简单高效的方式来与数据库交互，同时保持高性能。

## 特性

- 简单直观的 API 设计
- 支持事务处理
- 提供代码生成工具，可自动生成模型
- 支持多种数据库驱动（MySQL、PostgreSQL、SQLite 等）
- 类型安全的查询构建器
- 连接池和管理
- 结构化日志记录

## 数据库支持

- 第一优先级： MySQL/MariaDB、Postgres
- 第二优先级： SQLite
- 其他：SQLServer、Oracle （不保证）

## 安装

```bash
go get github.com/yvvlee/lorm
```

## 快速开始

### 1. 初始化引擎

```go
engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname")
if err != nil {
    log.Fatal(err)
}
defer engine.Close()
```

### 2. 定义模型

```go
type User struct {
    lorm.UnimplementedTable
    ID        int64  `lorm:"id,primary_key,auto_increment"`
    Name      string `lorm:"name"`
    Email     string `lorm:"email"`
    CreatedAt time.Time `lorm:"created_at,created"`
    UpdatedAt time.Time `lorm:"updated_at,updated"`
}
```

### 3. 使用 lormgen 生成代码

```bash
# 安装 lormgen
go install github.com/yvvlee/lorm/cmd/lormgen@latest

# 为您的模型生成代码
lormgen ./...
```

这将生成带有 `_lorm_gen.go` 后缀的文件，其中包含数据库操作所需的方法。

### 4. 增删改查操作

#### 插入

```go
// 插入单个模型
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}
rowsAffected, err := lorm.Insert[*User](engine).AddModel(user).Exec(ctx)

// 批量插入多个模型
users := []*User{
    {Name: "John Doe", Email: "john@example.com"},
    {Name: "Jane Doe", Email: "jane@example.com"},
}
rowsAffected, err := lorm.Insert[*User](engine).AddModels(users...).Exec(ctx)
```

#### 查询

```go
// 根据 ID 获取
user, err := lorm.Query[*User](engine).
    Where(builder.Eq{u.Fields().ID(): 1}).
    Get(ctx)

// 条件查询多条记录
users, err := lorm.Query[*User](engine).
    Where(builder.Eq(u.Fields().Name(): "John"}).
    Find(ctx)
```

#### 更新

```go
var u User
rowsAffected, err := lorm.Update(engine).
    Table(u.TableName()).
    ID(1).
    SetMap(map[string]any{
        u.Fields().Name(): "Jane Doe",
    }).
    Exec(ctx)
```

#### 删除

```go
// 使用 typed helper 按主键删除
rowsAffected, err := lorm.DeleteModel[*User](engine).
    ID(1).
    Exec(ctx)

// 自定义条件删除
var u User
rowsAffected, err := lorm.Delete(engine).
    From(u.TableName()).
    Where(builder.Eq{u.Fields().ID(): 1}).
    Exec(ctx)

```

> **说明**: Statement builder 是轻量级对象，每次调用 `Query` / `Insert` / `Update` / `Delete` 都会创建独立实例。每个数据库操作都应重新起一条链式调用，不要在多个 goroutine 间共享同一个 statement。

## 事务支持

通过TX方法开启事务，回调函数的入参ctx中会携带事务session，回调函数中的数据库操作都使用这个ctx，lorm就会自动使用这个ctx携带的事务session。
回调函数如果返回了error，则事务会被回滚，否则事务将自动提交

```go
err := engine.TX(context.Background(), func(ctx context.Context) error {
    user1 := &User{Name: "User 1"}
    _, err := lorm.Insert[*User](engine).AddModel(user1).Exec(ctx)
    if err != nil {
        return err
    }
    
    user2 := &User{Name: "User 2"}
    _, err = lorm.Insert[*User](engine).AddModel(user2).Exec(ctx)
    if err != nil {
        return err
    }
    
    return nil
})
```

## 强烈推荐使用Repository

lorm.Repository[T Table] 实现了常用的单表CRUD操作， 你可以在UserRepositoryImpl中内嵌lorm.Repository[*User]，
然后通过接口UserRepository按需暴露这些常用方法，


```go
type UserRepository interface {
	//以下方法为常用方法，lorm.Repository[*User]已实现，按需暴露
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
    
	//也可以添加自定义方法，需要自行在UserRepositoryImpl中实现
	PageGmailUsers(ctx context.Context, pageNum, pageSize uint64) ([]*User,uint64, error)
}

var _ UserRepository = (*UserRepositoryImpl).(nil)

type UserRepositoryImpl struct {
	lorm.Repository[*User]
}

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: lorm.NewRepository[*User](engine),
	}
}

func (r *UserRepositoryImpl) PageGmailUsers(ctx context.Context, pageNum, pageSize uint64) ([]*User,uint64, error)  {
	var u User
	return lorm.Query[*User](r.Engine).
		From(u.TableName()).
		Where(builder.Like(u.Fields().Email(), "%@gmail.com")).
		OrderBy(r.Fields().ID()+" desc").
		Page(pageNum, pageSize)
}


```

## 配置选项

lorm 支持多种配置选项：

```go
engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname",
    lorm.WithPlaceholderFormat(builder.Dollar),//设置SQL占位符，默认为"?"
    lorm.WithEscaper(names.NewQuoter('"', '"')), //设置SQL转义符，用于转义表名和列名中的特殊字符，默认为 ``(如 select `id`,`desc`,`name` from `table`)
    lorm.WithMaxIdleConns(10),// 设置最大空闲连接数
    lorm.WithMaxOpenConns(100),// 设置打开连接的最大数量
    lorm.WithConnMaxLifetime(time.Hour),// 设置连接最大存活时长
    lorm.WithLogger(customLogger), // 设置自定义logger
)
```

## lormgen 代码生成器使用

lormgen 是 Lorm 的代码生成器，用于自动生成数据库表结构相关的代码。

### 使用方法

```bash
lormgen [flags] <directory|file>...
```
它会扫描文件中嵌入了 lorm.UnimplementedTable 和 lorm.UnimplementedModel 的结构体，为其生成使用lorm必要的方法

### 参数说明

- `--field-mapper`: 表字段名称映射器，可选值: `snake`(蛇形命名), `camel`(驼峰命名), `same`(保持不变)，默认值: `snake`
- `--table-mapper`: 数据库表名映射器，可选值: `snake`(蛇形命名), `camel`(驼峰命名), `same`(保持不变)，默认值: `snake`
- `--table-prefix`: 数据库表名前缀，默认为空
- `--table-suffix`: 数据库表名后缀，默认为空
- `--tag-key`: 表字段标签键名，默认值: `lorm`
- `--file-suffix`: 生成文件的后缀名，默认值: `_lorm_gen`
- `--ignore`: 忽略文件的通配符模式，可多次指定

### 使用示例

```bash
# 生成当前目录下所有 Go 文件的代码
lormgen .

# 递归生成指定目录及子目录下的代码
lormgen ./models/...

# 使用自定义参数生成代码
lormgen --table-prefix=t_ --table-suffix=_tab --field-mapper=camel ./models

# 忽略特定文件
lormgen --ignore="*_temp.go" --ignore="*_old.go" ./models
```

### 模型定义
定义数据表模型：
```go
type User struct {
    lorm.UnimplementedTable 
    ID                      int `lorm:"primary_key,auto_increment"`
    Name                    string
    Age                     int
    CreatedAt               time.Time `lorm:"created"`
    UpdatedAt               time.Time `lorm:"updated"`
}
```
运行lorgen之后，将会生成以下代码
```go
// TableName 返回表名
func (m *User) TableName() string {
    return "user"
}

// Fields 返回User字段获取器
func (m *User) Fields() *User_Fields {}

type User_Fields struct {
    alias string
}

func (f *User_Fields) WithAlias(alias string) *User_Fields {}
func (f *User_Fields) ID() string {}
//..其他字段获取方法

// All 获取所有字段名
func (f *User_Fields) All() []string {}
```

表名默认使用模型的蛇形命名，你可以通过添加lormgen的--table-mapper参数修改表名映射规则， 也可以通过给嵌入的lorm.UnimplementedTable添加tag来显示指定

例如：
```go
type User struct {
    lorm.UnimplementedTable `lorm:"users"`
}
//这将会生成以下代码：
func (m *User) TableName() string {
    return "users"
}
```

数据库字段名默认使用模型的蛇形命名，你可以通过添加lormgen的--field-mapper参数修改字段名映射规则， 也可以通过给字段添加tag来显示指定

例如：
```go
type User struct {
    lorm.UnimplementedTable
    Name string `lorm:"username"`
}
//这将会生成以下代码：
type User_Fields struct {
    alias string
}

func (f *User_Fields) Name() string {
    if f.alias == "" {
        return "username"
    }
    return f.alias + ".username"
}
```

支持结构体嵌入：
```go
type Metadata struct {
	ID int64
	Name string
}

type User struct {
    lorm.UnimplementedTable
	
    Metadata
    Age int
}

//这将会生成以下代码：
type User_Fields struct {
    alias string
}

func (f *User_Fields) ID() string {
    if f.alias == "" {
        return "id"
    }
    return f.alias + ".id"
}
func (f *User_Fields) Name() string {
    if f.alias == "" {
        return "name"
    }
    return f.alias + ".name"
}
func (f *User_Fields) Age() string {
    if f.alias == "" {
        return "age"
    }
    return f.alias + ".age"
}

```
给嵌入的结构体添加tag，作为内嵌字段的前缀：
```go
type Metadata struct {
	ID int64
	Name string
}

type User struct {
    lorm.UnimplementedTable
	
    Metadata `lorm:"user_"`
    Age int
}

//这将会生成以下代码：
type User_Fields struct {
    alias string
}

func (f *User_Fields) ID() string {
    if f.alias == "" {
        return "user_id"
    }
    return f.alias + ".user_id"
}
func (f *User_Fields) Name() string {
    if f.alias == "" {
        return "user_name"
    }
    return f.alias + ".user_name"
}
func (f *User_Fields) Age() string {
    if f.alias == "" {
        return "age"
    }
    return f.alias + ".age"
}

```
lorm支持以下内置tag来标记字段的特殊属性：

- `primary_key`: 标记字段为主键
- `auto_increment`: 标记字段为自增字段
- `json`: 标记字段以JSON格式存储
- `created`: 标记字段为创建时间，插入时自动设置当前时间
- `updated`: 标记字段为更新时间，插入和更新时自动设置当前时间
- `version`: 标记字段为乐观锁版本号，更新时自动递增

使用示例：
```go
type User struct {
    lorm.UnimplementedTable
    ID        int64     `lorm:"primary_key,auto_increment"`
    Name      string
    Profile   *Profile  `lorm:"json"`
    CreatedAt time.Time `lorm:"created_time,created"`
    UpdatedAt time.Time `lorm:"updated_time,updated"`
    Version   int       `lorm:"version"`
}

type Profile struct {
    Avatar string
    Bio    string
}
```

当前数据库查询结果需要映射为自定义模型而非数据表模型时，你只需要嵌入lorm.UnimplementedModel即可。
它的行为和lorm.UnimplementedTable几乎一致， 但它不会生成TableName()方法，所以你在用自定义模型进行数据库操作时需要手动指定表名。
```go
type UserRole struct {
	lorm.UnimplementedModel
	UserID int64
	UserName string
	RoleID int64
	RoleName string
}

func main() {
    engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Close()
    ctx := context.Background()
    models,err := Query[*UserRole](engine).
        Select("u.id user_id","u.name user_name","r.id role_id","r.name role_name").
        From("user").
        Alias("u").
        InnerJoin("role as r on u.role_id=r.id").
        Find(ctx)
	//SQL: select u.id user_id,u.name user_name,r.id role_id,r.name role_name
	//     from user u
	//     inner join role as r on u.role_id=r.id
	
}
```


## 贡献

欢迎贡献代码！请随时提交 Pull Request。

## 许可证

该项目采用 MIT 许可证 - 详情请见 [LICENSE](LICENSE) 文件。
