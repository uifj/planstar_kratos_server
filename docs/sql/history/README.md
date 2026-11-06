# history — 重构前的历史快照（请勿执行）

本目录保存 2026-11-06 全量规范重构**之前**的产物，仅用于对照与追溯，**不代表当前 schema**。

当前有效的建库脚本在上一级目录：`00_all.sql` / `01..13_*.sql` / `verify.sql`。

## 文件说明

| 文件 | 说明 | 为何不可用 |
| --- | --- | --- |
| `00_all_faithful.sql` | 未做 TINYINT 归一化的原样版 | 89 表 / 61 外键 / 旧表名，与当前 schema 不同 |
| `VALIDATION.md` | 旧实跑验证报告（16/16 PASS） | 结论针对旧版 1,444 列 / 61 外键 |
| `validation.json` | 旧验证机器可读结果 | 同上，数字已过期 |
| `preflight.json` | 旧静态预检（61 条外键） | 当前 schema 外键为 0 |
| `index.html` | 旧验证可视化报告 | 同上 |
| `manifest.json` | 旧生成清单与 sha256 | 哈希、语句数均已过期 |
| `run_log_clean.txt` / `run_log_faithful.txt` | 旧执行日志 | 对应旧版 SQL |
| `tools/build_sql_rebuildnew.py` | 旧生成器 | 依赖已删除的 `docs/sql_new/`，运行会覆盖当前结果 |
| `tools/verify_mysql.py` | 旧验证脚本 | 内置旧的 61 外键 / 89 表期望值 |

## 使用建议

- 需要重建当前库：使用上一级的 `00_all.sql`（或按 `01..13` 顺序执行）。
- 需要复核：使用上一级的 `verify.sql`。
- 需要对比历史差异：只读本目录文件，不要执行、不要修改。
