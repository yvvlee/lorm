# LORM - Lightweight ORM for Go

[中文](README_ZH.md)

LORM is a lightweight ORM (Object-Relational Mapping) library for Go. It provides a simple and efficient way to interact with databases while maintaining high performance.

## Features

- Simple and intuitive API design
- Support for transactions
- Code generation tools for automatic model creation
- Support for multiple database drivers (MySQL, PostgreSQL, SQLite, etc.)
- Query builder with type safety
- Connection pooling and management
- Structured logging

## Installation

```bash
go get github.com/yvvlee/lorm
```

## Quick Start

### 1. Initialize Engine

```go
engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname")
if err != nil {
    log.Fatal(err)
}
defer engine.Close()
```

### 2. Define Models

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

### 3. Generate Code with lormgen

Before performing any database operations, you need to generate code using the `lormgen` tool:

```bash
# Install lormgen
go install github.com/yvvlee/lorm/cmd/lormgen@latest

# Generate code for your models
lormgen ./...
```

This will generate files with the `_lorm_gen.go` suffix that contain the necessary methods for database operations.

### 4. CRUD Operations

#### Insert

```go
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}
rowsAffected, err := lorm.Insert(ctx, engine, user)
```

#### Query

```go
// Get by ID
user, err := lorm.Query[*User](engine).
    Where(builder.Eq{u.Fields().ID(): 1}).
    Get(ctx)

// Query with conditions
users, err := lorm.Query[*User](engine).
    Where(builder.Eq{u.Fields().Name(): "John"}).
    Find(ctx)
```

#### Update

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

#### Delete

```go
var u User
rowsAffected, err := lorm.Delete(engine).
    From(u.TableName()).
    Where(builder.Eq{u.Fields().ID(): 1}).
    Exec(ctx)
```

> **Note**: These operations require the code generation step to be completed first.

## Recommended: Using Repository

LORM.Repository implements common single-table CRUD operations. You can embed lorm.Repository[*User] in UserRepositoryImpl, and then expose these common methods as needed through the UserRepository interface:

```go
type UserRepository interface {
	// The following methods are common methods that lorm.Repository[*User] has implemented, expose as needed
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
    
	// You can also add custom methods that need to be implemented in UserRepositoryImpl
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
		OrderBy(r.Fields().ID()).
		Page(pageNum, pageSize)
}
```

## Transaction Support

Through the TX method to start a transaction, the incoming ctx parameter of the callback function will carry the transaction session. All database operations in the callback function use this ctx, and lorm will automatically use the transaction session carried by this ctx.
If the callback function returns an error, the transaction will be rolled back, otherwise the transaction will be automatically committed.

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

## Configuration Options

LORM supports various configuration options:

```go
engine, err := lorm.NewEngine("mysql", "user:password@tcp(localhost:3306)/dbname",
    lorm.WithMaxIdleConns(10),
    lorm.WithMaxOpenConns(100),
    lorm.WithConnMaxLifetime(time.Hour),
    lorm.WithLogger(customLogger),
)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.