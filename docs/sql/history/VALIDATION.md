# sql_rebuildnew · MySQL 实跑验证结果

**总体状态：`PASS`（16/16 项通过）**

生成时间：2026-11-05T23:56:39+08:00

## 运行环境

| 项 | 值 |
| --- | --- |
| 镜像 | `sha256:8dbcf531a03aade657e181b9cf2f1d1803ce621a1d55610cb44cb531ab7d7db6` (8dbcf531a03a) |
| 容器 | `planstar-mysql-verify` / `b37b28717154768480f630d903c3837fa1a0a5b16e77a3c36d110751e8b25caf` |
| MySQL | 8.4.10 · MySQL Community Server - GPL |
| 服务端默认排序规则 | `utf8mb4_0900_ai_ci`（库/表显式覆盖为 `utf8mb4_unicode_ci`）|
| sql_mode | `ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION` |
| 干净版脚本 | `00_all.sql` · 196 条语句 · 293175 字节 |

## 执行结果

| 脚本 | ERROR | WARNING | 说明 |
| --- | --- | --- | --- |
| `00_all.sql`（R4 干净版） | 0 | 0 | 从空实例一次执行完成 |
| `00_all_faithful.sql`（原样版） | 0 | 119 | 全部为 `Warning 1681`（TINYINT(1) 显示宽度弃用）|

## 校验项

| # | 检查 | 状态 | 实测 | 期望 |
| --- | --- | --- | --- | --- |
| C01 | 9 个服务库且字符集/排序规则正确 | **PASS** | 9 | 9 |
| C02 | 基表数量 | **PASS** | 89 | 89 |
| C03 | 列数量（真实 information_schema） | **PASS** | 1444 | 1444 |
| C04 | 所有列均有 COMMENT | **PASS** | 0 | 0 |
| C05 | JSON 列数量 | **PASS** | 222 | 222 |
| C06 | 外键定义数 = 实存外键数 | **PASS** | 61 = 61 | 61 = 61 |
| C07 | 跨库外键引用数 | **PASS** | 24 | 24 |
| C08 | 索引数量 >= 169（含 13 条件索引与 InnoDB 自动建索引） | **PASS** | 182 | >= 169 |
| C09 | 89 张表全部 InnoDB + utf8mb4_unicode_ci（R1 实跑） | **PASS** | InnoDB/utf8mb4_unicode_ci/89 | InnoDB / utf8mb4_unicode_ci / 89 |
| C10 | 4 组证据种子行数 | **PASS** | {'dict_enums': 143, '_rev_dto_field_catalog': 1736, '_rev_endpoint_catalog': 267, '_rev_dto_table_mapping': 217} | {'dict_enums': 143, '_rev_dto_field_catalog': 1736, '_rev_endpoint_catalog': 267, '_rev_dto_table_mapping': 217} |
| C11 | 00_all.sql 实跑 0 ERROR / 0 WARNING | **PASS** | 0 / 0 | 0 / 0 |
| C12 | 00_all_faithful.sql 仅 119 条 Warning 1681（其余 0 ERROR） | **PASS** | 119 × ['1681'] | 119 × 1681 / 0 ERROR |
| C13 | 同库外键真实拒绝孤儿行（ERROR 1452） | **PASS** | PASS | PASS |
| C14 | 跨库外键真实拒绝孤儿行（ERROR 1452） | **PASS** | PASS | PASS |
| C15 | docs/sql_new 源文件 sha256 未变 | **PASS** | 0 | 0 |
| C16 | 静态预检（61 FK / 9 索引语句引用与类型） | **PASS** | PASS / 0 issues | PASS / 0 issues |

## 实跑对象清单

- 9 个库全部 `utf8mb4 / utf8mb4_unicode_ci`；89 张表全部 `InnoDB + utf8mb4_unicode_ci`。
- 1444 列 / 182 个索引 / 222 个 JSON 列 / 0 个缺注释列。
- 索引构成：169 基线 + 7 净新增条件索引（13 里 8 ADD / 1 DROP，feeds 换名）+ 6 个 InnoDB 为外键子列自动创建的索引 = 182。
- 61 条外键定义 = 61 条 `REFERENTIAL_CONSTRAINTS`；其中 **24 条跨库**外键全部创建成功。
- 证据种子：dict_enums 143 / _rev_dto_field_catalog 1736 / _rev_endpoint_catalog 267 / _rev_dto_table_mapping 217。

## 外键强制生效实测

- 同库：插入 `planstar_user.login_sessions.user_id` 孤儿行 → `ERROR 1452 (23000) at line 1: Cannot add or update a child row: a foreign key constraint fails (`planstar_user`.`login_sessions`, CONSTRAINT `fk_login_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`))` → PASS
- 跨库：插入 `planstar_payment.gift_card_usages.order_id` 孤儿行 → `ERROR 1452 (23000) at line 1: Cannot add or update a child row: a foreign key constraint fails (`planstar_payment`.`gift_card_usages`, CONSTRAINT `fk_gift_card_usages_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`))` → PASS
- 两次测试后子表残留 0 行，未污染验证库。

## 验证边界（未验证 / 不声称）

- 实跑环境为 MySQL 8.4.10 LTS（镜像 mysql:8），不是 MySQL 8.0；DDL 方言按 8.0 编写，在更严格的 8.4 上零错误通过。
- 未做 EXPLAIN 执行计划 / 索引选择性 / 锁竞争 / QPS / 慢查询 / 复制延迟 / 高可用切换 / 分区与归档实测。
- 8 个条件索引（13_query_indexes.sql）仍属候选，无真实负载数据，不声称性能收益。
- 未落地 deep_review 的 P01–P17 语义提案（Q1–Q10 未确认）；D-01..D-17 结构性结论仍成立。
- 「能跑通」只证明 DDL 语法、对象引用、外键与种子数据完整，不等于「符合原生产业务逻辑」——原服务端建库 DDL 从未获得。
- 00_all.sql 是单次全新空库安装脚本；重复执行会因外键/索引重名与证据种子重复插入而报错，重装须先执行 90_teardown.sql。

## 复现

```bash
cd docs/sql_rebuildnew
python tools/build_sql_rebuildnew.py
docker exec -i -e MYSQL_PWD='<密码>' <容器> mysql -uroot --default-character-set=utf8mb4 --force --show-warnings < 00_all.sql
python tools/verify_mysql.py
```
