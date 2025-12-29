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

lorm.Repository implements common single-table CRUD operations. You can embed lorm.Repository[*User] in UserRepositoryImpl, and then expose these common methods as needed through the UserRepository interface:

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
		OrderBy(r.Fields().ID()+" desc").
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
    lorm.WithPlaceholderFormat(builder.Dollar), // Set SQL placeholder, default is "?"
    lorm.WithEscaper(names.NewQuoter('"', '"')), // Set SQL escaper for table and column names, default is `` (e.g., select `id`,`desc`,`name` from `table`)
    lorm.WithMaxIdleConns(10), // Set maximum number of idle connections
    lorm.WithMaxOpenConns(100), // Set maximum number of open connections
    lorm.WithConnMaxLifetime(time.Hour), // Set maximum connection lifetime
    lorm.WithLogger(customLogger), // Set custom logger
)
```

## lormgen Code Generator Usage

lormgen is Lorm's code generator for automatically generating database table structure related code.

### Usage

```bash
lormgen [flags] <directory|file>...
```
It scans structures that embed lorm.UnimplementedTable and lorm.UnimplementedModel, and generates the necessary methods for using lorm.

### Parameters

- `--field-mapper`: Field name mapper, options: `snake` (snake_case), `camel` (camelCase), `same` (keep unchanged), default: `snake`
- `--table-mapper`: Table name mapper, options: `snake` (snake_case), `camel` (camelCase), `same` (keep unchanged), default: `snake`
- `--table-prefix`: Database table name prefix, default is empty
- `--table-suffix`: Database table name suffix, default is empty
- `--tag-key`: Field tag key name, default: `lorm`
- `--file-suffix`: Generated file suffix, default: `_lorm_gen`
- `--ignore`: Glob pattern for files to ignore, can be specified multiple times

### Usage Examples

```bash
# Generate code for all Go files in current directory
lormgen .

# Recursively generate code for specified directory and subdirectories
lormgen ./models/...

# Generate code with custom parameters
lormgen --table-prefix=t_ --table-suffix=_tab --field-mapper=camel ./models

# Ignore specific files
lormgen --ignore="*_temp.go" --ignore="*_old.go" ./models
```

### Model Definition
Define a database table model:
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
After running lormgen, the following code will be generated:
```go
// TableName returns the table name
func (m *User) TableName() string {
    return "user"
}

// Fields returns User field accessor
func (m *User) Fields() *User_Fields {}

type User_Fields struct {
    alias string
}

func (f *User_Fields) WithAlias(alias string) *User_Fields {}
func (f *User_Fields) ID() string {}
// ...other field accessor methods

// All returns all field names
func (f *User_Fields) All() []string {}
```

By default, table names use snake_case naming of the model. You can modify the table name mapping rules by adding the --table-mapper parameter to lormgen, or explicitly specify it by adding a tag to the embedded lorm.UnimplementedTable.

For example:
```go
type User struct {
    lorm.UnimplementedTable `lorm:"users"`
}
// This will generate the following code:
func (m *User) TableName() string {
    return "users"
}
```

By default, database field names use snake_case naming of the model. You can modify the field name mapping rules by adding the --field-mapper parameter to lormgen, or explicitly specify it by adding a tag to the field.

For example:
```go
type User struct {
    lorm.UnimplementedTable
    Name string `lorm:"username"`
}
// This will generate the following code:
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

Supports struct embedding:
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

// This will generate the following code:
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
Add a tag to the embedded struct to use as a prefix for nested fields:
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

// This will generate the following code:
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
Built-in tags:

lorm supports the following built-in tags to mark special field attributes:

- `primary_key`: Mark field as primary key
- `auto_increment`: Mark field as auto-increment
- `json`: Mark field to be stored in JSON format
- `created`: Mark field as creation time, automatically set to current time on insert
- `updated`: Mark field as update time, automatically set to current time on insert and update
- `version`: Mark field as optimistic lock version number, automatically incremented on update

Usage example:
```go
type User struct {
    lorm.UnimplementedTable
    ID        int64     `lorm:"primary_key;auto_increment"`
    Name      string
    Profile   *Profile  `lorm:"json"`
    CreatedAt time.Time `lorm:"created"`
    UpdatedAt time.Time `lorm:"updated"`
    Version   int       `lorm:"version"`
}

type Profile struct {
    Avatar string
    Bio    string
}
```


When database query results need to be mapped to a custom model rather than a database table model, you just need to embed lorm.UnimplementedModel.
It behaves almost identically to lorm.UnimplementedTable, but it won't generate a TableName() method, so you need to manually specify the table name when performing database operations with custom models.
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
	// SQL: select u.id user_id,u.name user_name,r.id role_id,r.name role_name
	//      from user u
	//      inner join role as r on u.role_id=r.id
	
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.