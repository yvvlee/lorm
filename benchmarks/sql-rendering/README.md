# SQL 构建缓冲区对比实验

2026-09-17：保留 `strings.Builder` 配合预留容量的实现。完整 SELECT 构建的耗时中位数下降约 12%～23%，分配量下降约 14%～33%。SQLite 完整读取的耗时变化不足 1%，本次样本无法确认它有稳定提升。

只直接替换类型并非全面更快。初筛中，短查询的分配次数从 8 次增加到 10 次。预留容量后降到 7 次。

## 实验方法

- 环境：Go 1.27.1、darwin/arm64、Apple M1 Pro，默认 GOMAXPROCS=10。
- 两版使用相同代码、输入和编译配置。仅 SELECT 渲染器与位置占位符转换器的缓冲区实现不同。
- 基线的两个渲染文件取自本次修改前的工作区，其内容与提交 `6ec8a9c` 一致。测试包含本次补充的渲染基准，以及前一轮加入的完整 SELECT 构建和 SQLite 读取基准。
- 先编译为独立测试程序，再串行运行，避免编译与基准竞争 CPU。每轮轮换版本顺序。
- 初筛四种实现，各运行 4 轮，每个基准 200ms。最终两版各运行 10 轮，每个基准 300ms。SQLite 也运行 10 轮，开启 CGO。
- 表中均为中位数。时间为 ns/op，内存为 B/op，次数为 allocs/op。
- 时间变化区间通过配对重采样估算：每个案例对同轮两版结果共同抽取 10 对样本，重复 10000 次，固定随机种子 20260917，取中位数比值变化的 2.5% 和 97.5% 分位数。它描述这次样本的波动，不代表其他硬件与数据库。

## 初筛：区分替换类型与预留容量的作用

| 实现 | SELECT 初始容量 | 占位符转换初始容量 |
| --- | --- | --- |
| baseline | bytes.Buffer 默认 | bytes.Buffer 默认 |
| strings | strings.Builder 默认 | strings.Builder 默认 |
| buffer_grow | bytes.Buffer，64 字节 | bytes.Buffer，输入 SQL 长度 |
| strings_grow（最终） | strings.Builder，64 字节 | strings.Builder，输入 SQL 长度 |

| 实现 | 短查询时间 | 短查询分配次数 | MySQL 配置完整构建 | PostgreSQL 配置完整构建 |
| --- | ---: | ---: | ---: | ---: |
| baseline | 259.00 | 8 | 794.95 | 1019.00 |
| strings | 278.55 | 10 | 707.35 | 863.80 |
| buffer_grow | 264.95 | 8 | 820.55 | 985.30 |
| strings_grow | 238.45 | 7 | 731.05 | 814.35 |

## 最终对比：完整 SELECT 构建

这一组包含语句创建、默认字段、标识符加引号、SQL 拼装和占位符转换，不访问数据库。

| 配置 | 基线时间 | 最终时间 | 时间变化 | 时间变化 95% 区间 | 字节：基线 → 最终 | 次数：基线 → 最终 |
| --- | ---: | ---: | ---: | --- | ---: | ---: |
| mysql | 823.30 | 726.55 | -11.75% | -14.21%～-9.18% | 2192 → 1888 | 28 → 27 |
| pgx | 1057.00 | 817.60 | -22.65% | -28.63%～-19.85% | 3248 → 2176 | 31 → 28 |

## 最终对比：渲染与占位符转换

语句与参数在计时前创建。短查询包含两列与主键条件；预备字段查询包含 20 列；复杂查询包含 JOIN、GROUP BY、HAVING、排序和分页；大 IN 查询包含 1000 个参数。

| 案例 | 基线时间 | 最终时间 | 时间变化 | 字节：基线 → 最终 | 次数：基线 → 最终 |
| --- | ---: | ---: | ---: | ---: | ---: |
| SelectRender/simple | 270.40 | 240.20 | -11.17% | 232 → 168 | 8 → 7 |
| SelectRender/prepared | 364.35 | 278.70 | -23.51% | 1128 → 824 | 9 → 8 |
| SelectRender/complex | 579.35 | 492.30 | -15.03% | 880 → 656 | 14 → 13 |
| SelectRender/large_in | 5479.00 | 4947.50 | -9.70% | 45424 → 42080 | 9 → 8 |
| ReplacePlaceholders/0 | 31.61 | 19.34 | -38.82% | 88 → 24 | 2 → 1 |
| ReplacePlaceholders/1 | 55.50 | 38.34 | -30.91% | 112 → 48 | 2 → 1 |
| ReplacePlaceholders/10 | 241.50 | 191.70 | -20.62% | 272 → 192 | 3 → 2 |
| ReplacePlaceholders/100 | 1910.50 | 1603.50 | -16.07% | 1411 → 723 | 6 → 3 |
| ReplacePlaceholders/1000 | 30230.50 | 26930.00 | -10.92% | 24581 → 17477 | 910 → 905 |

## 最终对比：SQLite 完整读取

沿用 ORM CRUD 基准中的 LORM 实现。单行与 100 行读取均包含数据库执行、模型分配及扫描。复杂字段场景包含更多 JSON 数据。

| 案例 | 基线时间 | 最终时间 | 时间变化 | 时间变化 95% 区间 | 字节：基线 → 最终 |
| --- | ---: | ---: | ---: | --- | ---: |
| ReadByID | 20676.5 | 20512.0 | -0.80% | -1.39%～+0.21% | 6066 → 5856 |
| ReadByIDComplex | 23086.0 | 22921.5 | -0.71% | -1.56%～+0.25% | 7102.5 → 6892.5 |
| BatchRead100 | 541544.5 | 540274.0 | -0.23% | -1.12%～+0.57% | 214316 → 213862 |
| BatchRead100Complex | 793709.5 | 786101.0 | -0.96% | -1.47%～+0.02% | 319582 → 319128 |

四个时间区间都跨过 0，因此本次不能确认 SQLite 读取耗时有稳定改善。分配量有减少。未运行 MySQL、PostgreSQL 实库对比；上面的 mysql、pgx 构建结果只表示方言配置。

## 复跑命令

在修改前后分别运行以下命令。严格比较时，先编译两版测试程序，再交替运行每一轮，避免版本与运行时段完全重合。

```bash
go test -run '^$' -bench '^BenchmarkSelectSQL$' -benchmem -benchtime=300ms -count=10 .
go test -run '^$' -bench '^Benchmark(SelectRender|ReplacePlaceholders)$' -benchmem -benchtime=300ms -count=10 ./builder
cd benchmarks/orm-crud
CGO_ENABLED=1 go test -run '^$' -bench '^Benchmark(ReadByID|ReadByIDComplex|BatchRead100|BatchRead100Complex)$/lorm$' -benchmem -benchtime=300ms -count=10 .
```

## 原始结果

- 初筛：[baseline](screen-baseline.txt)、[strings](screen-strings.txt)、[buffer_grow](screen-buffer_grow.txt)、[strings_grow](screen-strings_grow.txt)。
- 最终构建对比：[基线](confirm-baseline.txt)、[最终实现](confirm-strings_grow.txt)。
- SQLite 对比：[基线](sqlite-baseline.txt)、[最终实现](sqlite-strings_grow.txt)。
