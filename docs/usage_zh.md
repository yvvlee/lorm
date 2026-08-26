# LORM 完整使用指南

[English](usage.md) | [README](../README_ZH.md) | [可运行示例](../example/README.md)

本文档系统介绍 LORM 的架构设计、API 规范、用法约定与生产最佳实践。

---

## 目录

- [1. 设计哲学](#1-设计哲学)
- [2. 安装与工具链](#2-安装与工具链)
- [3. 模型定义与 Struct Tag 规范](#3-模型定义与-struct-tag-规范)
- [4. 代码生成工具 (`lormgen`)](#4-代码生成工具-lormgen)
- [5. Engine 初始化与配置](#5-engine-初始化与配置)
- [6. 核心 CRUD 操作](#6-核心-crud-操作)
  - [插入数据 (Insert)](#插入数据-insert)
  - [查询数据 (Query)](#查询数据-query)
  - [更新数据 (Update)](#更新数据-update)
  - [删除数据 (Delete)](#删除数据-delete)
  - [防全表写安全哨兵](#防全表写安全哨兵)
- [7. 强类型 SQL 构建器](#7-强类型-sql-构建器)
- [8. 分页与单列查询](#8-分页与单列查询)
- [9. 事务管理与并发控制](#9-事务管理与并发控制)
- [10. 整洁架构与 Repository 最佳实践](#10-整洁架构与-repository-最佳实践)
- [11. 自定义投影模型与复杂 JOIN](#11-自定义投影模型与复杂-join)
- [12. 自定义字段类型与 JSON 序列化](#12-自定义字段类型与-json-序列化)
- [13. Statement 生命周期与最佳实践](#13-statement-生命周期与最佳实践)

---

## 1. 设计哲学

LORM 围绕以下五大核心原则构建：

1. **零反射运行时**：抛弃运行时的反射解析（`reflect.ValueOf`、`reflect.TypeOf`），通过 `lormgen` 在编译期生成直接字段指针与值访问器，提供极致的执行速度与超低内存分配。
2. **编译期类型安全**：SQL 字段引用通过生成的类型化方法（`u.LormCols().Name()`）调用，杜绝字符串拼写错误与字段重命名时的隐蔽缺陷。
3. **显式 SQL（拒绝黑盒魔法）**：不引入隐式 Join、不进行静默懒加载、无隐藏的 N+1 查询扩散。所写即所行，SQL 执行计划透明可控。
4. **基于 Context 的无感事务传递**：事务状态透明依附于标准 `context.Context`，调用链方法签名无需侵入 `*sql.Tx` 事务参数，各层边界清晰解耦。
5. **整洁架构天然友好**：内置泛型 `lorm.Repository[T]` 辅助基类，无缝适配领域驱动设计（DDD）与整洁架构，使业务逻辑与底层数据库访问完全解耦，单元测试极其易于 Mock。

---

## 2. 安装与工具链

### 环境要求
- **Go 1.27** 或更高版本。

### 安装依赖库与代码生成器

```bash
# 安装 LORM 核心库
go get github.com/yvvlee/lorm

# 安装 lormgen 命令行工具
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

### 配合 `go generate` 自动化生成

在包含数据模型的包中（如 `model/generate.go`）添加指令：

```go
package model

//go:generate lormgen .
```

后续只需在项目根目录运行以下命令即可重新生成全工程的模型辅助代码：

```bash
go generate ./...
```

---

## 3. 模型定义与 Struct Tag 规范

LORM 将模型分为两类：
- **数据表模型 (Table Model)**：映射数据库物理表，必须嵌入 `lorm.UnimplementedTable`。
- **投影模型 (Projection Model)**：映射多表关联查询结果或自定义 DTO，必须嵌入 `lorm.UnimplementedModel`。

### 表模型示例

```go
package model

import (
	"time"
	"github.com/yvvlee/lorm"
)

type User struct {
	lorm.UnimplementedTable `lorm:"users"` // 自定义表名（可选）

	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Name      string    `lorm:"name"`
	Email     string    `lorm:"email"`
	Age       int       `lorm:"age"`
	Version   int64     `lorm:"version"`             // 乐观锁版本号
	CreatedAt time.Time `lorm:"created_at,created"`  // 插入时零值自动填充时间
	UpdatedAt time.Time `lorm:"updated_at,updated"`  // 插入与更新时自动刷新时间
}
```

### Struct Tag 规范说明

| Tag 项 | 功能说明 | 适用类型 |
| :--- | :--- | :--- |
| `列名 (column_name)` | 显式指定该字段对应的数据库列名。缺省时默认转为 `snake_case`。 | 所有字段 |
| `primary_key` | 声明为主键字段。支持复合主键。 | 标量类型 / 整数 / 字符串 |
| `auto_increment` | 声明为自增字段。必须同时具备 `primary_key`。 | 整数类型 |
| `created` | 插入数据且字段为零值时，自动填充当前时间。 | `time.Time`, `sql.NullTime`, `int64`, `uint64`, `uint32`, `uint`, 64位 `int`, `string` 及其一层指针 |
| `updated` | 插入时零值填充时间，并在更新模型时自动刷新为当前时间。 | 同 `created` |
| `version` | 声明为乐观锁版本字段，更新时自动作为条件比对并自增。每个模型最多 1 个。 | 整数类型 |
| `json` | 自动进行 JSON 序列化与反序列化。 | 结构体、切片、Map |

### 结构体嵌套与展开规则

- **内嵌结构体展开**：内嵌结构体会被自动展开平铺到父模型的列访问器中。
- **列名前缀**：在内嵌结构体字段上声明 tag 可以为其展开的所有子列名添加统一前缀。
- **独立行声明**：凡带有 `lorm` tag 的字段必须单独占用一行，避免 `FieldA, FieldB string \`lorm:"..."\`` 这种合并声明。
- **时间类型约束**：时间字段为整数时保存 Unix 时间戳（秒）；为字符串时使用 `time.DateTime` (`2006-01-02 15:04:05`)。`int8`、`uint8`、`int16`、`uint16`、`int32` 不支持作为时间托管字段。

---

## 4. 代码生成工具 (`lormgen`)

`lormgen` 扫描 Go 源码文件，解析结构体元数据，并在同一目录下生成 `*_lorm_gen.go` 源码。

### 命令行用法

```bash
lormgen [flags] <directory|file>...
```

### 常用参数说明

| 参数 | 默认值 | 作用说明 |
| :--- | :--- | :--- |
| `--field-mapper` | `snake` | 字段名映射规则：`snake`、`camel` 或 `same` |
| `--table-mapper` | `snake` | 表名映射规则：`snake`、`camel` 或 `same` |
| `--table-prefix` | `""` | 生成表名的统一定义前缀（如 `t_`） |
| `--table-suffix` | `""` | 生成表名的统一定义后缀 |
| `--tag-key` | `lorm` | 结构体 Tag Key |
| `--file-suffix` | `_lorm_gen` | 生成 Go 文件的命名后缀 |
| `--ignore` | `""` | 忽略文件的 Glob 模式（可多次指定） |

### 生成的代码包含什么

`lormgen` 为每个模型生成：
- `TableName() string`：解析模型对应的数据表名。
- `LormCols()`：强类型列名访问器（支持 `.WithAlias()` 别名）。
- `LormFieldPtr(name string) any`：直接指针访问器，实现零反射快速 `sql.Rows` 结果扫描。
- `LormFieldValue(name string) any`：直接值访问器，实现零反射 SQL 参数提取。
- `LormModelDescriptor()`：主键元数据与字段描述符缓存指针。
- 写入 Hook（`BeforeInsertHook`、`BeforeUpdateHook`），处理时间戳与乐观锁自增。

---

## 5. Engine 初始化与配置

`lorm.Engine` 管理底层 `*sql.DB` 连接池、数据库方言规则、事务会话和日志系统。

### 初始化示例

```go
package main

import (
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/yvvlee/lorm"
)

func main() {
	// 以 MySQL 为例
	engine, err := lorm.NewEngine(
		"mysql",
		"user:password@tcp(127.0.0.1:3306)/mydb?parseTime=true&charset=utf8mb4",
		lorm.WithMaxOpenConns(100),
		lorm.WithMaxIdleConns(20),
		lorm.WithConnMaxLifetime(time.Hour),
		lorm.WithConnMaxIdleTime(30*time.Minute),
		lorm.WithLogger(lorm.NewDefaultLogger()),
	)
	if err != nil {
		log.Fatalf("初始化 Engine 失败: %v", err)
	}
	defer engine.Close()
}
```

### 多数据库方言支持

LORM 根据传入的 driver name 自动匹配标准方言行为：

| Driver Name | 占位符格式 | 标识符转义 | 自增 ID 获取机制 | 行锁语法支持 |
| :--- | :--- | :--- | :--- | :--- |
| `mysql` | `?` | 反引号 (`` `col` ``) | `LastInsertId` | `FOR UPDATE` |
| `pgx` / `postgres` | `$1, $2` | 双引号 (`"col"`) | `RETURNING` | `FOR UPDATE` |
| `sqlite3` | `?` | 双引号 (`"col"`) | `LastInsertId` | 不支持 |

### 自定义方言配置

若使用特殊的数据库驱动或需精细微调，可通过配置项覆盖：

```go
dialect := lorm.DefaultDialectConfig("pgx")
dialect.SupportsForUpdate = true

engine, err := lorm.NewEngine(
	"pgx",
	"postgres://user:pass@localhost:5432/mydb?sslmode=disable",
	lorm.WithDialectConfig(dialect),
)
```

---

## 6. 核心 CRUD 操作

### 插入数据 (Insert)

#### 单条插入
```go
user := &User{Name: "Alice", Email: "alice@example.com"}
rowsAffected, err := engine.Insert[*User]().
	AddModel(user).
	Exec(ctx)

// 驱动支持 RETURNING 或 LastInsertId 时，自增 ID 会自动回填至 user.ID
```

#### 批量插入与 ID 回填语义
```go
users := []*User{
	{Name: "Bob", Email: "bob@example.com"},
	{Name: "Charlie", Email: "charlie@example.com"},
}

// 1. 标准批量插入（单条多行 SQL，最高吞吐性能）：
_, err := engine.Insert[*User]().
	AddModels(users...).
	Exec(ctx)

// 2. 需要对每个模型回填自增 ID 时：
_, err := engine.Insert[*User]().
	AddModels(users...).
	RequireIDBackfill(). // 在事务中退化为逐行执行并回填 ID
	Exec(ctx)
```

> **主键状态处理机制**：
> - 自增主键为零值（`0`）时，LORM 会在 `INSERT` 语句中省略该列，由数据库自动生成。
> - 自增主键为非零值时，LORM 会显式插入该指定值。
> - 同一批次中若混有零值与非零值主键，LORM 会自动在同一事务中将其拆分为连续的子批次分别执行。

---

### 查询数据 (Query)

#### 查询单条记录 (`Get`)
`Get` 返回 `(model T, found bool, err error)`。第二个 bool 明确指示是否命中记录：

```go
var u User
user, found, err := engine.Query[*User]().
	Where(builder.Eq{u.LormCols().ID(): 42}).
	Get(ctx)
if err != nil {
	return err
}
if !found {
	log.Println("未找到该用户")
}
```

#### 查询多条记录 (`Find`)
`Find` 返回 `([]T, error)`：

```go
var u User
c := u.LormCols()

users, err := engine.Query[*User]().
	Where(builder.Gte(c.Age(), 18)).
	OrderBy(c.CreatedAt() + " DESC").
	Limit(20).
	Find(ctx)
```

---

### 更新数据 (Update)

#### 部分更新 (`Set` / `SetMap`)
做局部字段修改时，推荐使用 `SetMap` 或 `Set`，避免未赋值字段以零值写回：

```go
var u User
c := u.LormCols()

rowsAffected, err := engine.Update[*User]().
	ID(42).
	SetMap(map[string]any{
		c.Name(): "Alice Updated",
		c.Age():  30,
	}).
	Exec(ctx)
```

#### 原子累加与表达式更新
```go
_, err := engine.Update[*User]().
	ID(42).
	SetExpr(c.Age(), "age + 1").
	Exec(ctx)
```

#### 全模型更新 (`SetModel`)
`SetModel` 会按模型描述符更新所有映射列：

```go
user.Name = "Alice Full"
_, err := engine.Update[*User]().
	ID(user.ID).
	SetModel(user).
	Exec(ctx)
```

> ⚠️ **警告**：`SetModel` 会将结构体中的零值（如 `""`、`0`、`false`）完整写回数据库。该方法不能与 `Set` 或 `SetMap` 混用。

---

### 删除数据 (Delete)

#### 按主键或条件删除
```go
// 按主键删除
rowsAffected, err := engine.Delete[*User]().
	ID(42).
	Exec(ctx)

// 按条件删除
var u User
rowsAffected, err := engine.Delete[*User]().
	Where(builder.Eq{u.LormCols().Email(): "spam@example.com"}).
	Exec(ctx)
```

---

### 防全表写安全哨兵

为防止因代码疏漏执行了无条件的全局更新或删除，`Update.Exec` 与 `Delete.Exec` 会进行静态审查，拦截缺少有效 `WHERE` 条件的调用：

```go
// ❌ 拦截报错: "update statement missing WHERE condition"
_, err := engine.Update[*User]().Set("status", "inactive").Exec(ctx)

// ✅ 确实需要执行全表操作时，必须显式调用 AllowGlobalWrite()
_, err := engine.Update[*User]().
	Set("status", "inactive").
	AllowGlobalWrite().
	Exec(ctx)
```

> **注意**：通过 `Engine.Exec` 执行的原始 SQL 字符串不受此检查约束。

---

## 7. 强类型 SQL 构建器

LORM 内置了功能完备的 SQL 构建器（`github.com/yvvlee/lorm/builder`）。配合生成的 `LormCols()` 使用可享受编译期重构保障：

```go
var u User
c := u.LormCols()
```

### 常用条件表达式

| 运算类型 | Go 代码表达式 | 生成的 SQL 片段 |
| :--- | :--- | :--- |
| **等于** | `builder.Eq{c.Email(): "a@b.com"}` | `` `email` = ? `` |
| **不等于** | `builder.Ne(c.Age(), 0)` | `` `age` <> ? `` |
| **数值比较** | `builder.Gt(c.Age(), 18)`, `builder.Lte(c.Age(), 60)` | `` `age` > ? ``, `` `age` <= ? `` |
| **IN / NOT IN** | `builder.In(c.ID(), []int64{1, 2, 3})` | `` `id` IN (?, ?, ?) `` |
| **空值判断** | `builder.IsNull(c.DeletedAt())`, `builder.IsNotNull(c.Email())` | `` `deleted_at` IS NULL `` |
| **模糊匹配** | `builder.Like(c.Name(), "John%")` | `` `name` LIKE ? `` |
| **PostgreSQL 数组** | `builder.Any("roles", []string{"admin", "editor"})` | `"roles" = ANY(?)` |

> ⚠️ **关于 `builder.Eq` 的重要说明**：`builder.Eq{field: value}` 始终生成 `field = ?` 并将 `value` 作为一个驱动参数。它**不会**把 `nil` 改写为 `IS NULL`，也不会把切片自动展开为 `IN (...)`。空值判断与集合判断请分别显式使用 `builder.IsNull()` 与 `builder.In()`。

### 复合逻辑组合 (`And` / `Or`)

```go
whereClause := builder.And(
	builder.Eq{c.Status(): "active"},
	builder.Or(
		builder.Gt(c.Score(), 90),
		builder.Eq{c.Role(): "admin"},
	),
)

users, err := engine.Query[*User]().
	Where(whereClause).
	Find(ctx)
```

### 表别名支持 (`WithAlias`)

涉及关联查询与表别名时，调用 `WithAlias()`：

```go
var u User
c := u.LormCols().WithAlias("u")

ids, err := engine.Query[*User]().
	Select(c.ID()).
	From("users AS u").
	Where(builder.Like(c.Email(), "%@example.com")).
	OrderBy(c.ID() + " DESC").
	FindCols[int64](ctx)
```

---

## 8. 分页与单列查询

### 模型结果分页 (`Page`)

`Page(ctx, page, size)` 在单次调用中自动计算总记录数并获取对应页（从 1 开始）：

```go
var u User
users, total, err := engine.Query[*User]().
	Where(builder.Eq{u.LormCols().Status(): "active"}).
	OrderBy(u.LormCols().ID() + " DESC").
	Page(ctx, 1, 20) // 第 1 页，每页 20 条

if err != nil {
	return err
}
log.Printf("当前页获取 %d 条 (总记录数: %d)", len(users), total)
```

### 单列值查询 (`GetCol`, `FindCols`, `PageCols`)

当只查询单列标量或聚合值时，无需声明结构体，直接使用泛型列提取方法：

```go
var u User
c := u.LormCols()

// 1. 获取单行单列值
count, ok, err := engine.Query[*User]().
	Select("COUNT(1)").
	GetCol[int64](ctx)

// 2. 获取多行单列切片
ids, err := engine.Query[*User]().
	Select(c.ID()).
	Where(builder.Gt(c.Age(), 21)).
	FindCols[int64](ctx)

// 3. 分页获取单列数据
pageIDs, total, err := engine.Query[*User]().
	Select(c.ID()).
	OrderBy(c.ID() + " ASC").
	PageCols[int64](ctx, 1, 50)
```

> **注意**：单列终结方法要求 SQL 结果严格只有一列，若查询结果包含多列将直接返回错误。

---

## 9. 事务管理与并发控制

### 上下文透明事务 (`Engine.TX`)

LORM 提供声明式事务支持。所有接收事务 `ctx` 的仓储或引擎操作，均会自动加入当前事务：

```go
err := engine.TX(ctx, func(txCtx context.Context) error {
	_, err := engine.Insert[*User]().
		AddModel(&User{Name: "User 1", Email: "u1@example.com"}).
		Exec(txCtx)
	if err != nil {
		return err // 自动触发 ROLLBACK
	}

	_, err = engine.Insert[*User]().
		AddModel(&User{Name: "User 2", Email: "u2@example.com"}).
		Exec(txCtx)
	return err // 返回 nil 自动触发 COMMIT
})
```

### 嵌套事务支持

若在已持有事务的 `context` 下再次调用 `Engine.TX`，LORM 会自动复用当前事务会话，而不会重复开启底层物理事务。

### 事务选项配置 (`TXWithOptions`)

```go
opts := &sql.TxOptions{
	Isolation: sql.LevelSerializable,
	ReadOnly:  false,
}

err := engine.TXWithOptions(ctx, opts, func(txCtx context.Context) error {
	// 以 SERIALIZABLE 隔离级别执行
	return nil
})
```

### 悲观行锁 (`FOR UPDATE`)

在事务中调用 Repository 的 `Lock` 或 `LockByField` 方法：

```go
err := engine.TX(ctx, func(txCtx context.Context) error {
	user, err := userRepo.Lock(txCtx, userID)
	if err != nil {
		return err
	}

	user.Balance -= 100
	_, err = userRepo.Update(txCtx, user)
	return err
})
```

> **说明**：行级锁必须在活跃事务中（`Engine.TX`）调用。非事务环境下数据库会在语句执行后立即释放锁，因此 LORM 会直接返回错误。

### 乐观锁版本控制 (`version`)

在模型整数类型字段上标注 `version` tag：

```go
type Product struct {
	lorm.UnimplementedTable
	ID      int64 `lorm:"id,primary_key,auto_increment"`
	Stock   int   `lorm:"stock"`
	Version int64 `lorm:"version"`
}
```

通过 `Update.SetModel(product)` 更新时：
1. LORM 会自动生成：`UPDATE products SET stock = ?, version = version + 1 WHERE id = ? AND version = ?`。
2. 若版本号已被其他并发操作修改，`rowsAffected` 为 `0`，可据此识别并发冲突。

---

## 10. 整洁架构与 Repository 最佳实践

`lorm.Repository[T]` 封装了常用的单表数据访问方法（`Get`、`GetByField`、`Lock`、`Exist`、`Insert`、`InsertAll`、`Update`、`UpdateMap`、`Delete`、`DeleteByField`）。

### 推荐的 Repository 落地结构

```go
package repository

import (
	"context"
	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
	"yourproject/model"
)

// 1. 面向领域层定义清晰的接口
type UserRepository interface {
	Get(ctx context.Context, id any) (*model.User, error)
	Insert(ctx context.Context, user *model.User) (int64, error)
	Update(ctx context.Context, user *model.User) (int64, error)
	FindAdults(ctx context.Context, page, size uint64) ([]*model.User, uint64, error)
}

// 2. 仓储实现结构体内嵌泛型基类
type UserRepositoryImpl struct {
	*lorm.Repository[*model.User]
}

var _ UserRepository = (*UserRepositoryImpl)(nil)

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: engine.Repository[*model.User](),
	}
}

// 3. 实现自定义业务查询方法
func (r *UserRepositoryImpl) FindAdults(ctx context.Context, page, size uint64) ([]*model.User, uint64, error) {
	var u model.User
	return r.Engine.Query[*model.User]().
		Where(builder.Gte(u.LormCols().Age(), 18)).
		OrderBy(u.LormCols().ID() + " DESC").
		Page(ctx, page, size)
}
```

---

## 11. 自定义投影模型与复杂 JOIN

当查询结果涉及多表聚合或不属于任何单一数据表时，嵌入 `lorm.UnimplementedModel`：

```go
package model

import "github.com/yvvlee/lorm"

type UserOrderSummary struct {
	lorm.UnimplementedModel

	UserID       int64  `lorm:"user_id"`
	UserName     string `lorm:"user_name"`
	TotalOrders  int64  `lorm:"total_orders"`
	TotalSpent   int64  `lorm:"total_spent"`
}
```

运行 `lormgen` 生成扫描器后，编写显式 Join 查询：

```go
summaries, err := engine.Query[*model.UserOrderSummary]().
	Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"COUNT(o.id) AS total_orders",
		"COALESCE(SUM(o.amount), 0) AS total_spent",
	).
	From("users AS u").
	LeftJoin("orders AS o ON u.id = o.user_id").
	GroupBy("u.id", "u.name").
	Find(ctx)
```

---

## 12. 自定义字段类型与 JSON 序列化

### 内置 JSON 字段支持

在结构体或切片字段的 tag 中加入 `,json`：

```go
type Address struct {
	City    string `json:"city"`
	Street  string `json:"street"`
	ZipCode string `json:"zip_code"`
}

type Customer struct {
	lorm.UnimplementedTable
	ID      int64    `lorm:"id,primary_key,auto_increment"`
	Address Address  `lorm:"address,json"` // 自动在 DB 中存为 JSON 字符串/二进制
	Tags    []string `lorm:"tags,json"`    // 自动存为 JSON 数组
}
```

### 自定义类型转换 (`ScannerValuer`)

若字段需特殊自定义编解码，只需让类型实现标准库 `driver.Valuer` 与 `sql.Scanner` 接口：

```go
package customtype

import (
	"database/sql/driver"
	"strings"
	"github.com/yvvlee/lorm"
)

type StringSlice []string

// 编译期接口约束断言
var _ lorm.ScannerValuer = (*StringSlice)(nil)

func (s StringSlice) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s *StringSlice) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	switch v := src.(type) {
	case string:
		*s = strings.Split(v, ",")
	case []byte:
		*s = strings.Split(string(v), ",")
	}
	return nil
}
```

---

## 13. Statement 生命周期与最佳实践

1. **轻量构建与单次使用**：语句构建器（`Query`、`Insert`、`Update`、`Delete`）为轻量结构体，每次执行数据库操作应创建全新链条。
2. **并发非线程安全**：语句构建器在构造过程中为可变对象，**禁止**跨 Goroutine 共享同一个 statement 实例。
3. **条件派生与克隆**：若需从基础查询条件派生不同的子查询，使用 `.Clone()`：
   ```go
   baseQuery := engine.Query[*User]().Where(builder.Eq{u.LormCols().Status(): "active"})
   adminQuery := baseQuery.Clone().Where(builder.Eq{u.LormCols().Role(): "admin"})
   ```
4. **统一使用强类型列名**：优先使用 `u.LormCols().FieldName()`，杜绝魔法字符串，保障重构安全性。

