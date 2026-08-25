# LORM - 轻量级 Go ORM

[English](README.md)

LORM 是一个轻量级 Go ORM。它保持 SQL 显式，通过代码生成提供模型元数据和
类型化列访问器，并把数据访问收敛在 Repository 边界内。

## 特点

- 生成模型元数据，避免运行时大量反射映射。
- 类型化列访问器，减少查询中手写列名。
- Repository 辅助类型覆盖常见 CRUD，复杂 SQL 仍然显式。
- 事务通过 `context.Context` 传递。
- 不做隐式 join、关联加载或隐藏式查询扩散。

## 数据库支持

| 数据库 | 驱动包 | driver name |
| --- | --- | --- |
| MySQL/MariaDB | `github.com/go-sql-driver/mysql` | `mysql` |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | `pgx` |
| SQLite | `github.com/mattn/go-sqlite3` | `sqlite3` |

MySQL/MariaDB 和 PostgreSQL 是第一优先级。SQLite 适用于本地开发和示例。
LORM 不会引入数据库驱动。应用只需按需导入自己的驱动。

## 安装

LORM 需要 Go 1.27 或更高版本。

```bash
go get github.com/yvvlee/lorm
go install github.com/yvvlee/lorm/cmd/lormgen@latest
```

## 文档

- [使用说明](docs/usage_zh.md)：模型、代码生成、CRUD、事务、Repository、配置和 `lormgen`。
- [可运行示例](example/README.md)：使用 SQLite 的完整流程。
- [API 文档](https://pkg.go.dev/github.com/yvvlee/lorm)：导出的包 API。
- [基准测试](benchmarks/orm-crud/README.md)：测试方法和完整结果。

## 贡献

欢迎提 issue 或提交 pull request。

## 许可证

MIT。详见 [LICENSE](LICENSE)。
