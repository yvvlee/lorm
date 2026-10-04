# LORM - 高性能、零反射的 Go ORM

<p align="center">
  <a href="https://github.com/yvvlee/lorm"><img src="https://img.shields.io/badge/LORM-Go%20ORM-00ADD8?style=for-the-badge&logo=go" alt="LORM"></a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/yvvlee/lorm"><img src="https://pkg.go.dev/badge/github.com/yvvlee/lorm.svg" alt="Go Reference"></a>
  <a href="https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml"><img src="https://github.com/yvvlee/lorm/actions/workflows/unit_test.yml/badge.svg" alt="Build Status"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <img src="https://img.shields.io/badge/Go-%3E%3D%201.27.0-007D9C?logo=go" alt="Go Version">
</p>

<p align="center">
  <a href="README_ZH.md">简体中文</a> | <a href="README.md">English</a>
</p>

---

**LORM** 是一个专为 Go 开发者打造的高性能、零反射、强类型安全的轻量级 ORM 与 SQL 构建器。

在追求**极致性能**、**工程可维护性**与**确定性执行**的设计哲学下，LORM 舍弃了传统 ORM 沉重的运行时反射与黑盒魔法，通过编译期代码生成（`lormgen`）提供直接字段映射与强类型列名访问器，坚持显式 SQL 构造，并通过 Go 标准库 `context.Context` 提供完全透明的事务传递。

---

## 💡 为什么选择 LORM？

传统 Go ORM 往往在提供便利的同时牺牲了性能与可控性——带来沉重的反射开销、隐式关联加载、不可控的 N+1 查询扩散以及繁琐的事务传递。LORM 为解决这些痛点而生：

- ⚡ **零反射极致性能**：通过编译期代码生成（`lormgen`）生成直连字段指针与值访问器，彻底消除查询扫描（Scan）与模型映射时的反射损耗，达到媲美原生 `database/sql` 的吞吐与极低内存分配。
- 🛡️ **编译期类型安全**：自动生成强类型列名访问器（`u.LormCols().Name()`），彻底告别字符串硬编码列名，字段重命名与拼写错误在编译期立即可知。
- 🔍 **显式 SQL 哲学（拒绝黑盒魔法）**：所写即所行。无隐式 Join、无静默懒加载、无隐藏的查询扩散，SQL 执行计划完全透明受控。
- 🧩 **灵活直观的条件构造器**：支持条件组合与嵌套，可按参数动态添加筛选条件，减少分支判断，让复杂查询更易表达和维护。
- 🔄 **基于 Context 的无感事务传递**：事务状态绑定在标准 `context.Context` 中透传，业务层方法无需侵入显式传递事务对象（`*sql.Tx`），天然支持嵌套事务复用与自动回滚。
- 🏛️ **整洁架构与 Repository 友好**：内置泛型 `lorm.Repository[T]` 辅助基类，无缝契合领域驱动设计（DDD）与整洁架构，使业务逻辑与数据访问解耦，单元测试 Mock 极其简单。

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    业务逻辑 / 服务层                    │
└────────────────────────────┬────────────────────────────┘
                             │ 调用领域仓储接口
┌────────────────────────────▼────────────────────────────┐
│                    仓储层 (Repository)                  │
│        (内嵌 lorm.Repository[T] 或自定义复杂查询)       │
└────────────────────────────┬────────────────────────────┘
                             │
       ┌─────────────────────┴─────────────────────┐
       ▼                                           ▼
┌─────────────────────────────┐   ┌─────────────────────────────┐
│    Engine 引擎与事务会话     │   │      强类型 SQL 构建器      │
│  (通过 Context 透明透传事务) │   │   (代码生成的强类型列名访问) │
└──────────────┬──────────────┘   └──────────────┬──────────────┘
               │                                 │
               └─────────────────┬───────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────┐
│              模型元数据描述符与直连字段访问器            │
│            (`lormgen` 编译期代码生成 - 零反射)           │
└────────────────────────────────┬─────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────┐
│                   Go 标准库 database/sql                 │
└───────┬────────────────────────┼─────────────────────────┘
        │                        │                         │
┌───────▼───────┐        ┌───────▼───────┐         ┌───────▼───────┐
│     MySQL     │        │  PostgreSQL   │         │    SQLite     │
└───────────────┘        └───────────────┘         └───────────────┘
```

---

## 🗄️ 数据库支持

| 数据库 | 驱动包 | Driver Name | 支持状态 |
| :--- | :--- | :--- | :--- |
| **MySQL / MariaDB** | `github.com/go-sql-driver/mysql` | `mysql` | 支持 |
| **PostgreSQL** | `github.com/jackc/pgx/v5/stdlib` | `pgx` | 支持 |
| **SQLite** | `github.com/mattn/go-sqlite3` | `sqlite3` | 支持 |

> LORM 基于 `database/sql`，由应用导入所需的数据库驱动。

---

## 📦 安装

LORM 需要 **Go 1.27.0** 或更高版本。

```bash
# 安装 LORM 核心库
go get github.com/yvvlee/lorm

# 安装代码生成器 CLI 工具
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

---

## 🚀 快速上手

### 1. 定义数据模型

嵌入 `lorm.UnimplementedTable`，并通过 `lorm` struct tag 标注字段：

```go
package model

import (
	"time"
	"github.com/yvvlee/lorm"
)

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

运行 `lormgen` 生成零反射元数据与类型化列访问器：

```bash
lormgen ./...
```

这会自动生成 `*_lorm_gen.go`，包含 `LormCols()`、直连指针 Scanner 和写入 Hook。

### 3. 初始化 Engine 并执行 CRUD

```go
package main

import (
	"context"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
)

func main() {
	engine, err := lorm.NewEngine(
		"mysql",
		"user:password@tcp(127.0.0.1:3306)/mydb?parseTime=true&charset=utf8mb4",
		lorm.WithMaxOpenConns(50),
		lorm.WithMaxIdleConns(10),
	)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	var u User

	// --- 1. 插入数据 (Insert) ---
	newUser := &User{Name: "Alice", Email: "alice@example.com"}
	_, err = engine.Insert[*User]().AddModel(newUser).Exec(ctx)
	// 自增 ID 会自动回填到 newUser.ID

	// --- 2. 强类型列名查询 (Query) ---
	user, found, err := engine.Query[*User]().
		Where(builder.Eq{u.LormCols().Email(): "alice@example.com"}).
		Get(ctx)
	if err != nil || !found {
		log.Printf("未找到用户或发生错误: %v", err)
	}

	// --- 3. 部分更新 (Partial Update) ---
	_, err = engine.Update[*User]().
		ID(user.ID).
		SetMap(map[string]any{
			u.LormCols().Name(): "Alice Smith",
		}).
		Exec(ctx)

	// --- 4. 上下文驱动的事务 (Transaction) ---
	err = engine.TX(ctx, func(txCtx context.Context) error {
		_, err := engine.Update[*User]().
			ID(user.ID).
			Set(u.LormCols().Name(), "Alice Cooper").
			Exec(txCtx) // 自动参与当前事务
		return err
	})

	// --- 5. 删除数据 (Delete) ---
	_, err = engine.Delete[*User]().ID(user.ID).Exec(ctx)
}
```

---

## ✨ 核心特性一览

### 1. 强类型列名访问器与别名支持

通过生成的方法引用列名，减少手写字符串：

```go
var u User
cols := u.LormCols()

// 强类型 WHERE 与 ORDER BY
users, err := engine.Query[*User]().
	Where(builder.Eq{cols.Email(): "alice@example.com"}).
	Desc(cols.CreatedAt()).
	Find(ctx)

// 表别名支持（例如 users AS u）
uCols := u.LormCols().WithAlias("u")
ids, err := engine.Query[*User]().
	Select(uCols.ID()).
	From("users AS u").
	Where(builder.Like(uCols.Email(), "%@example.com")).
	FindCols[int64](ctx)
```

### 2. 基于 Context 的事务管理

事务保存在 `context.Context` 中，嵌套 `TX` 自动复用外层事务会话：

```go
err := engine.TX(ctx, func(txCtx context.Context) error {
	// 任何接收 txCtx 的 Repository 或 Engine 调用均自动加入同一事务
	if err := userRepo.UpdateBalance(txCtx, fromID, -100); err != nil {
		return err // 返回 error 自动触发 ROLLBACK
	}
	if err := userRepo.UpdateBalance(txCtx, toID, 100); err != nil {
		return err
	}
	return nil // 返回 nil 自动触发 COMMIT
})
```

### 3. Repository 模式

通过仓储接口封装数据库访问：

```go
type UserRepository interface {
	Get(ctx context.Context, id any) (*User, error)
	Insert(ctx context.Context, user *User) (int64, error)
	FindActiveUsers(ctx context.Context, minAge int) ([]*User, error)
}

type UserRepositoryImpl struct {
	*lorm.Repository[*User]
}

func NewUserRepository(engine *lorm.Engine) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		Repository: engine.Repository[*User](),
	}
}

func (r *UserRepositoryImpl) FindActiveUsers(ctx context.Context, minAge int) ([]*User, error) {
	var u User
	return r.Engine.Query[*User]().
		Where(builder.Gte(u.LormCols().Age(), minAge)).
		Find(ctx)
}
```

### 4. 内置乐观锁与时间戳自动维护

通过 tag 声明 `version`、`created` 或 `updated`：

```go
type Product struct {
	lorm.UnimplementedTable
	ID        int64     `lorm:"id,primary_key,auto_increment"`
	Stock     int       `lorm:"stock"`
	Version   int64     `lorm:"version"`            // 更新时自动校验并自增（CAS 乐观锁）
	CreatedAt time.Time `lorm:"created_at,created"` // 插入时零值自动填充当前时间
	UpdatedAt time.Time `lorm:"updated_at,updated"` // 插入/更新时自动刷新当前时间
}
```

### 5. 自定义类型与 JSON 字段

直接将复杂结构体或切片映射为 JSON：

```go
type Profile struct {
	Avatar string   `json:"avatar"`
	Tags   []string `json:"tags"`
}

type Account struct {
	lorm.UnimplementedTable
	ID      int64   `lorm:"id,primary_key,auto_increment"`
	Profile Profile `lorm:"profile,json"` // 自动进行 JSON 序列化与反序列化
}
```

若字段需要特殊的数据库转换逻辑，让该类型实现标准库 `sql.Scanner` 与 `driver.Valuer`（`lorm.ScannerValuer`）。

### 6. 全表写入限制

`Update.Exec` 与 `Delete.Exec` 默认拒绝无约束 WHERE 条件的语句。需要全表更新或删除时，显式调用 `AllowGlobalWrite()`：

```go
// 返回错误：缺少 WHERE 约束条件
_, err := engine.Update[*User]().Set("status", "inactive").Exec(ctx)

// 显式允许全表更新
_, err := engine.Update[*User]().
	Set("status", "inactive").
	AllowGlobalWrite().
	Exec(ctx)
```

---

## 📊 性能基准测试

CRUD 测试对比 LORM、GORM、XORM 和 Ent，覆盖 SQLite、MySQL 和 PostgreSQL。结果记录于 2026 年 8 月 21 日，环境为 Go 1.27.0、`darwin/arm64`（Apple M1 Pro），取三轮独立测试的中位数。

下表统计 LORM 在四个 ORM 中指标最低的用例数。每种数据库包含 10 个用例，覆盖单行、100 行批量操作及 JSON 读取。

| 指标 | SQLite | MySQL | PostgreSQL |
| :--- | ---: | ---: | ---: |
| 单次操作耗时（`ns/op`） | 6/10 | 9/10 | 7/10 |
| 单次操作分配字节数（`B/op`） | 9/10 | 9/10 | 9/10 |
| 单次操作分配次数（`allocs/op`） | 9/10 | 9/10 | 9/10 |

各用例数据和复跑命令见[完整报告](benchmarks/orm-crud/README.md)。[SQL 构建实验](benchmarks/sql-rendering/README.md)单独测量查询构建开销，与数据库执行耗时区分。

---

## 📚 详细文档

- 📖 **[完整使用指南](docs/usage_zh.md)**：模型定义规范、SQL 构建器用法、事务进阶、Repository 模式实践、`lormgen` 参数详解。
- 💡 **[可运行示例集](example/README.md)**：10 个可运行示例，涵盖增删改查、[列表搜索与分页](example/list_search/)、事务和自定义投影等场景。
- 📑 **[Go Reference API 文档](https://pkg.go.dev/github.com/yvvlee/lorm)**：查看导出的 Package API 与类型定义。
- 📊 **[基准测试报告](benchmarks/orm-crud/README.md)**：多数据库 Benchmark 方法论与详细数据对比。

---

## 🤝 参与贡献

非常欢迎提交 Issue、提出功能建议或发起 Pull Request！
可在 [Issues 页面](https://github.com/yvvlee/lorm/issues) 与我们交流。

---

## 📄 开源许可证

本项目采用 [MIT 许可证](LICENSE)。
