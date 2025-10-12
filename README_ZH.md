# LORM - 轻量级 Golang ORM

LORM 是一个为 Go 语言设计的轻量级ORM库。它提供了一种简单高效的方式来与数据库交互，同时保持高性能。

## 特性

- 简单直观的 API 设计
- 支持事务处理
- 提供代码生成工具，可自动生成模型
- 支持多种数据库驱动（MySQL、PostgreSQL、SQLite 等）
- 类型安全的查询构建器
- 连接池和管理
- 结构化日志记录

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
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}
rowsAffected, err := lorm.Insert(ctx, engine, user)

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
var u User
rowsAffected, err := lorm.Delete(engine).
    From(u.TableName()).
    Where(builder.Eq{u.Fields().ID(): 1}).
    Exec(ctx)

```

## 事务支持

通过TX方法开启事务，回调函数的入参ctx中会携带事务session，回调函数中的数据库操作都使用这个ctx，lorm就会自动使用这个ctx携带的事务session。
回调函数如果返回了error，则事务会被回滚，否则事务将自动提交

```go
err := engine.TX(context.Background(), func(ctx context.Context) error {
    user1 := &User{Name: "User 1"}
    _, err := lorm.Insert(ctx, engine, user1)
    if err != nil {
        return err
    }
    
    user2 := &User{Name: "User 2"}
    _, err := lorm.Insert(ctx, engine, user2)
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

LORM 支持多种配置选项：

```go
engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname",
    lorm.WithMaxIdleConns(10),
    lorm.WithMaxOpenConns(100),
    lorm.WithConnMaxLifetime(time.Hour),
    lorm.WithLogger(customLogger),
)
```

## 贡献

欢迎贡献代码！请随时提交 Pull Request。

## 许可证

该项目采用 MIT 许可证 - 详情请见 [LICENSE](LICENSE) 文件。