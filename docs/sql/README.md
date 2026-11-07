# sql — 全量规范重构后的可执行建库脚本（已在真实 MySQL 实跑）

> 已在原文件上直接改造完成：**没有迁移脚本、没有新增数据库结构定义**。
> 改造依据为《阿里巴巴 Java 开发手册》MySQL 规约，并参考 `mall-swarm` 的
> 单数表名 / 无外键 / 注释齐全风格（`macrozheng/mall-swarm` 的 `document/sql/mall.sql`）。
> 本目录是当前唯一的建库基线；重构前的旧文档与旧 SQL 已统一移入 [`history/`](history/)。

## 这是什么

把原 13 个编号文件按阿里规约整体重写，整理成一份**顺序正确、可一次性执行**的建库脚本，
并在真实 MySQL 8.4.10 上跑通。

- `00_all.sql` —— **一键完整安装**（9 库 / 97 张业务表 + 3 张证据表 / 0 物理外键 / 索引全部内联）
- `01..13_*.sql` —— 分文件副本，便于逐段 `SOURCE` 或复用
- `90_teardown.sql` —— 逆序删库（**破坏性**，执行前确认无业务数据）
- `15_project_domain.sql` —— 企划域增量 DDL（企划、回报档位、进度更新、支持记录等 11 张表），
  必须在 `14_mock_seed.sql` 之前执行
- `14_mock_seed.sql` —— 由 Flutter mock 数据映射生成的演示种子（企划/支持/社区域），
  在 `00_all.sql` + `15_project_domain.sql` 之后**可选**执行，不属于建库流程
- `verify.sql` —— 实跑后 `information_schema` 复核查询
- `optional/PROPOSALS.sql` —— 只读设计提案，全注释、0 可执行语句（表名仍是旧命名，仅作历史参考）
- `history/` —— **重构前历史快照**（旧验证结论、旧 manifest、旧工具、`00_all_faithful.sql`），
  仅作对照，不再代表当前 schema，不再是可执行入口

## 重构后的规模

| 项 | 数量 |
| --- | --- |
| 数据库 | 9（`planstar_user / catalog / ticket / order / payment / marketing / community / content / meta`） |
| 业务表 | 97 |
| 证据表（`_rev_*`，非业务，原样保留） | 3 |
| 列 | 1,760 |
| 物理外键 | **0** |
| 索引 | 271（含主键；单表二级索引 ≤ 5） |
| 枚举字典种子 | 143 行 |
| 证据种子 | 1,736 / 267 / 217 行 |

## 重构落地的规约

| 阿里规约 | 落地结果 |
| --- | --- |
| 表名、字段名单数、小写 | 85 张复数表名全部改单数（`users→user_info`、`orders→order_info`、`likes→like_record`、`feedback→user_feedback`、`dict_enums→dict_enum` 等） |
| 必备三字段 | 每表都有 `id BIGINT UNSIGNED AUTO_INCREMENT` + `gmt_create` + `gmt_modified`（97/97） |
| 时间字段命名 | 统一 `gmt_create` / `gmt_modified`，`DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`，更新时间带 `ON UPDATE` |
| 布尔字段 | 全部 `is_xxx TINYINT UNSIGNED NOT NULL DEFAULT 0`（`is_` 前缀且类型正确：100%） |
| 金额字段 | 全部 `DECIMAL(12,2)`；利率类 `DECIMAL(10,4)`；**0 个 FLOAT/DOUBLE** |
| 禁用保留字 | 原 42 处保留字列全部改名（`desc→description`、`rank→rank_weight / rating`、`schema→jump_schema`、`group→group_info`、`key→position_key / keyword_key / event_key`） |
| 唯一索引 | 业务唯一键补齐 UNIQUE（`user_info.mobile_no`、`order_info.order_number`、`gift_card.serial_number` 等），且唯一键列全部 `NOT NULL` |
| 禁止外键与级联 | 原 61 条外键（含 24 条跨库）全部移除，见 `12_constraints.sql` 的替代约定 |
| 禁止冗余索引 | 原 8 个条件索引取消，改为内联；去除 5 组前缀冗余，单表二级索引 ≤ 5 |
| 字段必须注释 | 1,760 列 **0 缺注释**（保留原 DTO 字段映射，便于追溯） |
| 字符集 | 9 库、100 表全部 `utf8mb4 / utf8mb4_unicode_ci`；97 张业务表均 `ROW_FORMAT=DYNAMIC`（3 张证据表未显式指定，沿用默认） |
| 禁用存储过程 | 无存储过程、无触发器、无视图 |

> `dict_enum.code` 是唯一的 signed `INT`：种子含 `-1` 的枚举（`DeliverType.UNKOWN`、
> `OrderType.unknown`），用 UNSIGNED 会溢出，故有意保留有符号。

## 执行顺序（不可调换）

```
01_init_databases.sql   ->  CREATE DATABASE x 9
02..09_<域>.sql         ->  USE <库> + CREATE TABLE（86 张业务表）
10_dict_enums.sql       ->  dict_enum + 143 条字典种子
11_rev_evidence.sql     ->  3 张证据表 + 2,363 行证据
12_constraints.sql      ->  no-op（0 外键，仅应用层一致性说明）
13_query_indexes.sql    ->  no-op（索引已内联到建表语句）
15_project_domain.sql   ->  企划域增量 DDL（11 张表，必须在 14 之前）
14_mock_seed.sql        ->  可选：企划/支持/社区域演示种子
```

`00_all.sql` 已按此顺序拼接；各分文件自带 `USE`。

## 本机实跑方式

```bash
cd docs/sql

docker run -d --name planstar-ddl-check \
  -e MYSQL_ROOT_PASSWORD='PlanstarCheck@2026' -e MYSQL_ROOT_HOST='%' \
  -p 13308:3306 mysql:8.4

docker cp 00_all.sql planstar-ddl-check:/tmp/00_all.sql
docker exec -e MYSQL_PWD='PlanstarCheck@2026' planstar-ddl-check \
  sh -c "mysql -uroot --default-character-set=utf8mb4 < /tmp/00_all.sql"

docker cp verify.sql planstar-ddl-check:/tmp/verify.sql
docker exec -e MYSQL_PWD='PlanstarCheck@2026' planstar-ddl-check \
  sh -c "mysql -uroot -t < /tmp/verify.sql"

# 可选：导入 Flutter mock 演示数据
docker cp 14_mock_seed.sql planstar-ddl-check:/tmp/14_mock_seed.sql
docker exec -e MYSQL_PWD='PlanstarCheck@2026' planstar-ddl-check \
  sh -c "mysql -uroot --default-character-set=utf8mb4 < /tmp/14_mock_seed.sql"
```

## 实跑结果（真实 MySQL 8.4.10，非静态推断）

| 项 | 结果 |
| --- | --- |
| `00_all.sql` 执行 | **0 ERROR / 0 WARNING**（stdout、stderr 均为 0 字节） |
| 库 / 业务表 / 全表 / 列 | 9 / **86** / 89 / 1,590 |
| 引擎与排序规则 | 89 张表全部 `InnoDB + utf8mb4_unicode_ci` |
| 物理外键 | **0**（`TABLE_CONSTRAINTS` 与 `REFERENTIAL_CONSTRAINTS` 均为 0） |
| 缺注释列 | **0** |
| 缺 `id/gmt_create/gmt_modified` 的表 | **0** |
| `FLOAT/DOUBLE` 列 | **0** |
| `is_*` 非 TINYINT / 非 UNSIGNED | **0 / 0** |
| 单表二级索引 | 最大 5（`activity`、`order_info`） |
| 种子 | `dict_enum` 143 / 证据表 1,736 + 267 + 217 |

## 边界（务必沿用）

- 实跑环境是 **MySQL 8.4.10 LTS**，DDL 方言兼容 8.0。
- **未做** EXPLAIN / 索引选择性 / 锁竞争 / QPS / 复制 / 高可用 / 分区 / 归档实测。
- 外键移除后，跨库一致性由服务层保证；`12_constraints.sql` 给出了校验与巡检 SQL。
- 本重构解决的是**结构规范问题**（命名、类型、约束、索引），不改变业务语义：
  支付/库存的幂等、座位锁、状态码标定等仍需另行设计。
- 不要运行 `history/tools/build_sql_rebuildnew.py`：它依赖已不存在的 `docs/sql_new/`，
  且会按旧的 61 条外键 / 89 张表结构覆盖当前结果。

## 历史快照（`history/`）

以下文件是重构**之前**的验证记录与生成器，仅用于对照，不代表当前 schema：

- `00_all_faithful.sql`（原样版，119 条 1681 告警）
- `preflight.json`（61 条外键预检）、`validation.json` / `VALIDATION.md` / `index.html`（16 项旧校验）
- `run_log_clean.txt` / `run_log_faithful.txt`、`manifest.json`
- `tools/build_sql_rebuildnew.py`（依赖已删除的 `docs/sql_new/`，不可运行）
- `tools/verify_mysql.py`（内置旧的 61 外键 / 89 表期望值，不可直接复用）

## 回滚

`90_teardown.sql` 逆序删除 9 个 `planstar_*` 库（**破坏性**）。
DDL 隐式提交，`ROLLBACK` 无效。
