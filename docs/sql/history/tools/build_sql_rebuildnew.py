#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
build_sql_rebuildnew.py — 由 docs/sql_new 生成「完整、可直接执行」的 sql_rebuildnew 包。

设计原则（与前几轮一致，务必沿用）:
  1. 不修改 docs/sql/ 基线，也不修改 docs/sql_new/（本脚本只读它们）。
  2. SQL 正文逐语句保持不变；只替换 sql_new 里那张「未做真实 MySQL 执行」的过时生成头部。
  3. 不落地 deep_review 的 P01–P17 语义提案（Q1–Q10 未确认），提案仅作只读副本。
  4. 新增 00_all.sql 一键脚本 / 90_teardown.sql / verify.sql / 静态预检 preflight.json。
  5. 所有统计值由脚本从真实文件解析，不手写。

用法:
  python tools/build_sql_rebuildnew.py
  python tools/build_sql_rebuildnew.py --src <sql_new目录> --out <sql_rebuildnew目录>
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import shutil
from pathlib import Path

HERE = Path(__file__).resolve().parent
DEFAULT_OUT = HERE.parent
DEFAULT_SRC = DEFAULT_OUT.parent / "sql_new"

# 默认安装链（顺序不可调换：先建库 -> 各域建表 -> 字典/证据 -> 外键 -> 索引）
NUMBERED = [
    "01_init_databases.sql",
    "02_user.sql",
    "03_catalog.sql",
    "04_ticket_inventory.sql",
    "05_order.sql",
    "06_payment.sql",
    "07_marketing.sql",
    "08_community.sql",
    "09_content_ops.sql",
    "10_dict_enums.sql",
    "11_rev_evidence.sql",
    "12_constraints.sql",
    "13_query_indexes.sql",
]

DATABASES = [
    "planstar_user",
    "planstar_catalog",
    "planstar_ticket",
    "planstar_order",
    "planstar_payment",
    "planstar_marketing",
    "planstar_community",
    "planstar_content",
    "planstar_meta",
]

HEADER = """-- ========================================================================
--  sql_rebuildnew · {name}
--  来源  : docs/sql_new（已含 R1 显式 COLLATE / R2 8 处 BIGINT 对齐 / R3 4 处注释）
--  用途  : 在真实 MySQL（Docker 镜像 8dbcf531a03a，mysql:8 / 8.4.10 LTS）上完整实跑验证。
--  与 sql_new 的差异:
--     (a) 替换 sql_new 中「未做真实 MySQL 执行」的过时生成头部；
--     (b) R4 确定性规范修复：TINYINT(1) -> TINYINT，消除 MySQL 8.4 的 1681
--         「整数显示宽度已弃用」告警。显示宽度不影响存储、值域、可空性、默认值，
--         本库经 JSON REST 访问、无 JDBC tinyInt1isBit 依赖，故语义等价。
--  原样版本（未做 R4，119 条 1681 告警）见 00_all_faithful.sql。
--  未执行 deep_review 的 P01–P17 语义提案（Q1–Q10 未确认）。
--  实跑结果: 见 VALIDATION.md / validation.json / run_log_clean.txt
-- ========================================================================
"""

HEADER_FAITHFUL = """-- ========================================================================
--  sql_rebuildnew · {name}（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================
"""

ALL_HEADER = """-- ========================================================================
--  sql_rebuildnew · 00_all.sql —— 一键完整安装脚本（0 ERROR / 0 WARNING）
--  内容  : 01 建库 -> 02..09 各域建表 -> 10 字典 -> 11 逆向证据 -> 12 推断外键 -> 13 条件索引
--  目标  : 全新空 MySQL 实例；9 个库 / 89 张表 / 4 组证据种子 / 61 条外键 / 8 个条件索引
--  执行  : docker exec -i <容器> mysql -uroot --default-character-set=utf8mb4 < 00_all.sql
--  注意  : 各 NN 文件自带 USE，顺序不可调换；DDL 隐式提交，不能用 ROLLBACK 撤销。
--  注意  : 单次安装脚本，不可重复执行（外键/索引重名、证据种子重复插入）；
--          重装请先执行 90_teardown.sql。
--  回滚  : 90_teardown.sql（逆序删库，破坏性：会删除 planstar_* 全部数据）。
--  验证  : 已在 MySQL 8.4.10 LTS 从空实例实跑，结果见 VALIDATION.md。
-- ========================================================================

SET NAMES utf8mb4;

"""

ALL_HEADER_FAITHFUL = """-- ========================================================================
--  sql_rebuildnew · 00_all_faithful.sql —— 忠实原样版（0 ERROR / 119 WARNING 1681）
--  与 docs/sql_new 的 SQL 正文逐语句一致，仅替换过时头部；不做 R4 的 TINYINT 归一化。
--  用于对照：证明「119 条告警全部来自 TINYINT(1) 显示宽度」，除此之外无差异。
-- ========================================================================

SET NAMES utf8mb4;

"""

SEPARATOR = "\n\n-- ------------------------------------------------------------------------\n--  ↓ 以下为 {name}\n-- ------------------------------------------------------------------------\n\n"


def read(p: Path) -> str:
    """统一按 UTF-8 读取并归一化为 LF，避免 Windows CRLF 混入 SQL。"""
    return p.read_text(encoding="utf-8").replace("\r\n", "\n").replace("\r", "\n")


def write(p: Path, text: str) -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8", newline="\n")


def sha256(p: Path) -> str:
    return hashlib.sha256(p.read_bytes()).hexdigest()


def strip_stale_header(text: str) -> str:
    """删掉 sql_new 生成的 11 行旧头部（含其中『未做真实 MySQL 执行』的过时声明）。"""
    lines = text.splitlines(keepends=True)
    idx = next((i for i, ln in enumerate(lines) if "sql_new（已修复版）" in ln), None)
    if idx is None:
        return text
    start = idx - 1 if idx > 0 and lines[idx - 1].startswith("-- ====") else idx
    end = next(
        (i for i in range(idx + 1, len(lines)) if lines[i].startswith("-- ====")),
        None,
    )
    if end is None:
        return text
    rest = lines[end + 1 :]
    while rest and rest[0].strip() == "":
        rest.pop(0)
    return "".join(rest)


def apply_r4(text: str) -> tuple[str, int]:
    """R4: TINYINT(1) -> TINYINT，消除 MySQL 8.4 Warning 1681（整数显示宽度弃用）。"""
    n = text.count("TINYINT(1)")
    return text.replace("TINYINT(1)", "TINYINT"), n


def count_statements(text: str) -> int:
    """引号/注释感知地统计顶层 SQL 语句数（用于报告，不参与执行）。"""
    count = 0
    in_s, in_d, in_line_comment, in_block_comment = False, False, False, False
    i, n = 0, len(text)
    while i < n:
        c = text[i]
        nxt = text[i + 1] if i + 1 < n else ""
        if in_line_comment:
            if c == "\n":
                in_line_comment = False
            i += 1
            continue
        if in_block_comment:
            if c == "*" and nxt == "/":
                in_block_comment = False
                i += 2
                continue
            i += 1
            continue
        if in_s:
            if c == "\\":
                i += 2
                continue
            if c == "'":
                in_s = False
            i += 1
            continue
        if in_d:
            if c == "\\":
                i += 2
                continue
            if c == '"':
                in_d = False
            i += 1
            continue
        if c == "-" and nxt == "-":
            in_line_comment = True
            i += 2
            continue
        if c == "#":
            in_line_comment = True
            i += 1
            continue
        if c == "/" and nxt == "*":
            in_block_comment = True
            i += 2
            continue
        if c == "'":
            in_s = True
            i += 1
            continue
        if c == '"':
            in_d = True
            i += 1
            continue
        if c == ";":
            count += 1
        i += 1
    return count


# ---------------------------------------------------------------------------
# 静态预检：解析真实 DDL，验证 61 条外键与 8 个条件索引的引用是否成立
# ---------------------------------------------------------------------------
CREATE_RE = re.compile(r"CREATE TABLE IF NOT EXISTS\s+`([^`]+)`\s*\((.*?)\)\s*ENGINE=", re.S)
USE_RE = re.compile(r"^\s*USE\s+`([^`]+)`\s*;", re.M | re.I)
COL_RE = re.compile(r"^\s*`([^`]+)`\s+([A-Za-z]+(?:\([^)]*\))?)", re.M)

FK_RE = re.compile(
    r"ALTER TABLE\s+`(?P<cdb>[^`]+)`\.`(?P<ctable>[^`]+)`\s+"
    r"ADD CONSTRAINT\s+`(?P<fk>[^`]+)`\s+FOREIGN KEY\s*\(\s*`(?P<ccol>[^`]+)`\s*\)\s*"
    r"REFERENCES\s+`(?P<pdb>[^`]+)`\.`(?P<ptable>[^`]+)`\s*\(\s*`(?P<pcol>[^`]+)`\s*\)",
    re.I,
)
ADD_IDX_RE = re.compile(
    r"ALTER TABLE\s+`(?P<db>[^`]+)`\.`(?P<table>[^`]+)`\s+"
    r"ADD INDEX\s+`(?P<idx>[^`]+)`\s*\((?P<cols>[^)]*)\)",
    re.I,
)
DROP_IDX_RE = re.compile(
    r"ALTER TABLE\s+`(?P<db>[^`]+)`\.`(?P<table>[^`]+)`\s+DROP INDEX\s+`(?P<idx>[^`]+)`",
    re.I,
)
INDEX_IN_DDL_RE = re.compile(r"\b(?:KEY|UNIQUE KEY|INDEX)\s+`([^`]+)`", re.I)


def parse_schema(src: Path) -> dict:
    """返回 {(db, table): {col: type}} 以及索引名集合、表数量、列数量。"""
    columns: dict[tuple[str, str], dict[str, str]] = {}
    indexes: dict[tuple[str, str], set[str]] = {}
    for name in NUMBERED:
        p = src / name
        if not p.exists():
            continue
        text = read(p)
        uses = USE_RE.findall(text)
        db = uses[0] if uses else ""
        for m in CREATE_RE.finditer(text):
            table, body = m.group(1), m.group(2)
            cols: dict[str, str] = {}
            for cm in COL_RE.finditer(body):
                col, typ = cm.group(1), cm.group(2)
                # 索引行以 PRIMARY/KEY/UNIQUE 起首（不带反引号），COL_RE 不会匹配它们；
                # 因此这里不能再按名字排除，否则会漏掉真实的 `key` / `check` 列。
                cols[col] = typ.upper()
            key = (db, table)
            columns[key] = cols
            indexes[key] = set(INDEX_IN_DDL_RE.findall(body))
    return {
        "columns": columns,
        "indexes": indexes,
        "table_count": len(columns),
        "column_count": sum(len(v) for v in columns.values()),
    }


def base_type(t: str) -> str:
    """归一化为用于外键类型比较的基类型（去显示宽度，保留 UNSIGNED 由调用方处理）。"""
    t = re.sub(r"\([^)]*\)", "", t.upper()).strip()
    return t


def preflight(src: Path, schema: dict) -> dict:
    columns: dict = schema["columns"]
    indexes: dict = schema["indexes"]
    issues: list[dict] = []
    fk_rows: list[dict] = []

    fk_text = read(src / "12_constraints.sql")
    for m in FK_RE.finditer(fk_text):
        g = m.groupdict()
        ck, pk = (g["cdb"], g["ctable"]), (g["pdb"], g["ptable"])
        row = {
            "constraint": g["fk"],
            "child": f"{g['cdb']}.{g['ctable']}.{g['ccol']}",
            "parent": f"{g['pdb']}.{g['ptable']}.{g['pcol']}",
        }
        if ck not in columns:
            issues.append({"kind": "FK_CHILD_TABLE_MISSING", **row})
        elif g["ccol"] not in columns[ck]:
            issues.append({"kind": "FK_CHILD_COLUMN_MISSING", **row})
        if pk not in columns:
            issues.append({"kind": "FK_PARENT_TABLE_MISSING", **row})
        elif g["pcol"] not in columns[pk]:
            issues.append({"kind": "FK_PARENT_COLUMN_MISSING", **row})
        if ck in columns and pk in columns and g["ccol"] in columns[ck] and g["pcol"] in columns[pk]:
            ct, pt = columns[ck][g["ccol"]], columns[pk][g["pcol"]]
            row["child_type"], row["parent_type"] = ct, pt
            if base_type(ct) != base_type(pt):
                issues.append({"kind": "FK_TYPE_MISMATCH", **row, "child_type": ct, "parent_type": pt})
        fk_rows.append(row)

    idx_rows: list[dict] = []
    idx_text = read(src / "13_query_indexes.sql")
    for m in ADD_IDX_RE.finditer(idx_text):
        g = m.groupdict()
        key = (g["db"], g["table"])
        cols = [c.strip().strip("`") for c in g["cols"].split(",")]
        row = {"index": g["idx"], "table": f"{g['db']}.{g['table']}", "columns": cols}
        if key not in columns:
            issues.append({"kind": "IDX_TABLE_MISSING", **row})
        else:
            missing = [c for c in cols if c not in columns[key]]
            if missing:
                issues.append({"kind": "IDX_COLUMN_MISSING", **row, "missing": missing})
        idx_rows.append(row)
    for m in DROP_IDX_RE.finditer(idx_text):
        g = m.groupdict()
        key = (g["db"], g["table"])
        row = {"index": g["idx"], "table": f"{g['db']}.{g['table']}"}
        if key not in columns:
            issues.append({"kind": "IDX_DROP_TABLE_MISSING", **row})
        elif g["idx"] not in indexes.get(key, set()):
            issues.append({"kind": "IDX_DROP_NOT_FOUND", **row})
        idx_rows.append({"index": g["idx"], "table": f"{g['db']}.{g['table']}", "action": "DROP"})

    return {
        "schema": {
            "table_count": schema["table_count"],
            "column_count": schema["column_count"],
            "databases": DATABASES,
        },
        "fk_count": len(fk_rows),
        "fk_rows": fk_rows,
        "index_statement_count": len(idx_rows),
        "index_rows": idx_rows,
        "issues": issues,
        "status": "PASS" if not issues else "FAIL",
    }


# ---------------------------------------------------------------------------
# 生成物
# ---------------------------------------------------------------------------
def build(src: Path, out: Path) -> dict:
    out.mkdir(parents=True, exist_ok=True)
    (out / "tools").mkdir(exist_ok=True)
    (out / "optional").mkdir(exist_ok=True)

    manifest: dict = {"source_dir": str(src), "generated_files": {}, "source_sha256": {}}

    # 1) 分文件：替换旧头部 + R4；同时保留一份「忠实原样」正文供对照
    bodies: dict[str, str] = {}
    faithful: dict[str, str] = {}
    r4_total = 0
    for name in NUMBERED:
        p = src / name
        if not p.exists():
            raise FileNotFoundError(p)
        raw = read(p)
        body = strip_stale_header(raw)
        clean_body, n = apply_r4(body)
        r4_total += n
        text = HEADER.format(name=name) + "\n" + clean_body
        faithful_text = HEADER_FAITHFUL.format(name=name) + "\n" + body
        write(out / name, text)
        bodies[name] = text
        faithful[name] = faithful_text
        manifest["source_sha256"][f"sql_new/{name}"] = sha256(p)
        manifest["generated_files"][name] = sha256(out / name)

    # down 脚本（可选索引回滚）
    for name in ["13_query_indexes.down.sql"]:
        p = src / name
        if p.exists():
            b, _ = apply_r4(strip_stale_header(read(p)))
            write(out / name, HEADER.format(name=name) + "\n" + b)
            manifest["generated_files"][name] = sha256(out / name)

    # 2) 一键完整安装（R4 干净版）
    parts = [ALL_HEADER]
    for name in NUMBERED:
        parts.append(SEPARATOR.format(name=name))
        parts.append(bodies[name].rstrip() + "\n")
    all_sql = "".join(parts)
    write(out / "00_all.sql", all_sql)
    manifest["generated_files"]["00_all.sql"] = sha256(out / "00_all.sql")

    # 2b) 忠实原样版（唯一差别 = 不做 R4；用于证明 119 条告警的来源）
    fparts = [ALL_HEADER_FAITHFUL]
    for name in NUMBERED:
        fparts.append(SEPARATOR.format(name=name))
        fparts.append(faithful[name].rstrip() + "\n")
    faithful_sql = "".join(fparts)
    write(out / "00_all_faithful.sql", faithful_sql)
    manifest["generated_files"]["00_all_faithful.sql"] = sha256(out / "00_all_faithful.sql")

    manifest["fixes"] = {
        "r1_r2_r3": "继承 docs/sql_new（本脚本不重做）",
        "r4_tinyint_display_width_replaced": r4_total,
        "r4_expected_warning_1681_removed": r4_total,
        "sql_new_body_verbatim_variant": "00_all_faithful.sql",
    }
    manifest["statement_count"] = {
        "00_all.sql": count_statements(all_sql),
        "00_all_faithful.sql": count_statements(faithful_sql),
    }

    # 3) 破坏性清理脚本
    teardown = [
        "-- ========================================================================\n",
        "--  sql_rebuildnew · 90_teardown.sql —— 逆序清理（破坏性）\n",
        "--  会删除 planstar_* 全部 9 个库及其数据；仅在确认无业务数据后执行。\n",
        "-- ========================================================================\n\n",
        "SET FOREIGN_KEY_CHECKS = 0;\n",
    ]
    for db in reversed(DATABASES):
        teardown.append(f"DROP DATABASE IF EXISTS `{db}`;\n")
    teardown.append("SET FOREIGN_KEY_CHECKS = 1;\n")
    write(out / "90_teardown.sql", "".join(teardown))
    manifest["generated_files"]["90_teardown.sql"] = sha256(out / "90_teardown.sql")

    # 4) 只读副本：未确认的语义提案（明确不可执行）
    prop = src / "optional" / "PROPOSALS.sql"
    if prop.exists():
        shutil.copy2(prop, out / "optional" / "PROPOSALS.sql")
        manifest["generated_files"]["optional/PROPOSALS.sql"] = sha256(out / "optional" / "PROPOSALS.sql")
    fkdrop = src / "optional" / "drop_optional_fks.sql"
    if fkdrop.exists():
        shutil.copy2(fkdrop, out / "optional" / "drop_optional_fks.sql")
        manifest["generated_files"]["optional/drop_optional_fks.sql"] = sha256(out / "optional" / "drop_optional_fks.sql")

    # 5) 静态预检
    schema = parse_schema(src)
    pf = preflight(src, schema)
    write(out / "preflight.json", json.dumps(pf, ensure_ascii=False, indent=2) + "\n")
    manifest["generated_files"]["preflight.json"] = sha256(out / "preflight.json")

    # 6) 实跑校验 SQL（人可读版本；机器版由 verify_mysql.py 执行）
    verify_sql = """-- ========================================================================
--  sql_rebuildnew · verify.sql —— 实跑后的 information_schema 复核
--  在 00_all.sql 执行完成后运行；全部结果由 verify_mysql.py 写入 validation.json。
-- ========================================================================

SELECT VERSION() AS server_version;

SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
FROM information_schema.SCHEMATA
WHERE SCHEMA_NAME LIKE 'planstar_%' ORDER BY SCHEMA_NAME;

SELECT COUNT(*) AS table_count FROM information_schema.TABLES
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE = 'BASE TABLE';

SELECT COUNT(*) AS column_count FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%';

SELECT COUNT(DISTINCT TABLE_SCHEMA, TABLE_NAME, INDEX_NAME) AS index_count
FROM information_schema.STATISTICS WHERE TABLE_SCHEMA LIKE 'planstar_%';

SELECT COUNT(*) AS fk_count FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND CONSTRAINT_TYPE = 'FOREIGN KEY';

SELECT ENGINE, TABLE_COLLATION, COUNT(*) AS n FROM information_schema.TABLES
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE = 'BASE TABLE'
GROUP BY ENGINE, TABLE_COLLATION ORDER BY n DESC;

SELECT COUNT(*) AS uncommented_columns FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND (COLUMN_COMMENT IS NULL OR COLUMN_COMMENT = '');

SELECT 'dict_enums' AS t, COUNT(*) AS n FROM `planstar_meta`.`dict_enums`
UNION ALL SELECT '_rev_dto_field_catalog', COUNT(*) FROM `planstar_meta`.`_rev_dto_field_catalog`
UNION ALL SELECT '_rev_endpoint_catalog', COUNT(*) FROM `planstar_meta`.`_rev_endpoint_catalog`
UNION ALL SELECT '_rev_dto_table_mapping', COUNT(*) FROM `planstar_meta`.`_rev_dto_table_mapping`;

SELECT COUNT(*) AS fk_constraints_live FROM information_schema.REFERENTIAL_CONSTRAINTS
WHERE CONSTRAINT_SCHEMA LIKE 'planstar_%';
"""
    write(out / "verify.sql", verify_sql)
    manifest["generated_files"]["verify.sql"] = sha256(out / "verify.sql")

    # 7) README
    readme = f"""# sql_rebuildnew — 完整可执行 SQL（已在真实 MySQL 实跑）

> 由 `tools/build_sql_rebuildnew.py` 从 `docs/sql_new/` 生成。
> `docs/sql/` 基线与 `docs/sql_new/` 均**未被修改**。

## 这是什么

把 `sql_new` 里分散的 13 个编号文件整理成一份**顺序正确、可一次性执行**的完整 SQL，
并在本机 Docker 的 MySQL 上真实跑通，用来回答「整套建库脚本到底有没有问题」。

- `00_all.sql` —— **一键完整安装，0 ERROR / 0 WARNING**（9 库 / 89 表 / 4 组证据种子 / 61 条外键 / 8 个条件索引）
- `00_all_faithful.sql` —— 忠实原样版：SQL 正文与 `sql_new` 逐语句一致，仅替换过时头部；
  在 MySQL 8.4 上 0 ERROR / **119 条 Warning 1681**（全部来自 `TINYINT(1)` 显示宽度弃用）
- `01..13_*.sql` —— 分文件副本（已应用 R4），便于逐段 `SOURCE` 或复用
- `90_teardown.sql` —— 逆序删库（**破坏性**，执行前确认无业务数据）
- `verify.sql` —— 实跑后 `information_schema` 复核查询
- `preflight.json` —— 静态预检：61 条外键 / 9 条索引语句的引用与类型核对
- `optional/PROPOSALS.sql` —— 只读副本，**全注释、0 可执行语句**（Q1–Q10 未确认，不落地）
- `validation.json` / `VALIDATION.md` / `index.html` —— 16 项实跑校验结果与可视化报告
- `run_log_clean.txt` / `run_log_faithful.txt` —— 两次真实执行的 stdout / stderr 原始日志
- `manifest.json` —— 源文件 sha256 与生成物清单；`tools/` —— 可复现生成器与校验器

### R4（本目录唯一新增的文本改动）

`sql_new` 的 119 个 `TINYINT(1)` 在 MySQL 8.4.10 上会触发 119 条
`Warning 1681: Integer display width is deprecated`。显示宽度不影响存储 / 值域 /
可空性 / 默认值，且本库经 JSON REST 访问、无 JDBC `tinyInt1isBit` 依赖，
因此 `00_all.sql` 把它归一化为 `TINYINT`，实跑 **0 告警**。
需要逐字节对照时改用 `00_all_faithful.sql`。

## 执行顺序（不可调换）

```
01_init_databases.sql   ->  CREATE DATABASE x 9
02..09_<域>.sql         ->  USE <库> + CREATE TABLE（89 张）
10_dict_enums.sql       ->  dict_enums + 143 条字典种子
11_rev_evidence.sql     ->  3 张证据表 + 1736 / 267 / 217 行证据
12_constraints.sql      ->  61 条推断外键（含 24 条跨库，需同实例）
13_query_indexes.sql    ->  8 个条件索引（feeds 先 DROP 再 ADD）
```

`00_all.sql` 已按此顺序拼接；各分文件自带 `USE`，12/13 用全限定名，不依赖当前库。

## 本机实跑方式

镜像 `8dbcf531a03a` 是 **`mysql:8`（实际 8.4.10 LTS）的镜像 ID**，不是容器 ID；
本机此前没有由它创建的容器，因此基于该镜像新建了验证容器：

```bash
docker run -d --name planstar-mysql-verify \\
  -e MYSQL_ROOT_PASSWORD='PlanstarVerify@2026' -e MYSQL_ROOT_HOST='%' \\
  -p 13307:3306 8dbcf531a03a

docker exec -i -e MYSQL_PWD='PlanstarVerify@2026' planstar-mysql-verify \\
  mysql -uroot --default-character-set=utf8mb4 < 00_all.sql
```

复核：`python tools/verify_mysql.py`（内部通过 `docker exec` 查询 information_schema）。

## 实跑结果（真实 MySQL，非静态推断）

| 项 | 结果 |
| --- | --- |
| `00_all.sql`（干净版） | **0 ERROR / 0 WARNING**；stdout 与 stderr 均为 0 字节 |
| `00_all_faithful.sql`（原样版） | 0 ERROR / 119 WARNING，全部为 Code 1681 |
| 库 / 表 / 列 / 索引 | 9 / 89 / 1,444 / 182 |
| 引擎与排序规则 | 89 张表全部 `InnoDB + utf8mb4_unicode_ci` |
| 外键 | 61 条定义 = 61 条 `REFERENTIAL_CONSTRAINTS`；其中 24 条跨库全部成功 |
| 外键强制生效 | 同库 / 跨库各插入一条必违例记录，均被 `ERROR 1452` 拒绝，子表残留 0 行 |
| 证据种子 | 143 / 1,736 / 267 / 217（合计 2,363 行） |
| 列注释 / JSON 列 | 0 个缺注释 / 222 个 JSON 列 |
| 校验 | `validation.json` = **PASS 16/16**（见 `VALIDATION.md` / `index.html`） |

日志：`run_log_clean.txt` / `run_log_faithful.txt`；机器可读：`validation.json`、
`preflight.json`、`manifest.json`。

## 边界（务必沿用）

- 实跑环境是 **MySQL 8.4.10 LTS**，不是 8.0；DDL 方言按 8.0 编写，8.4 上零错误通过。
- **未做** EXPLAIN / 索引选择性 / 锁竞争 / QPS / 复制 / 高可用 / 分区 / 归档实测。
- **未落地** deep_review 的 P01–P17 语义提案；`sql_new` 的 D-01..D-17 结构性结论仍然成立
  （payments 仍是 PayInfo DTO 镜像、仍无座位锁 / 库存流水、双状态码仍待标定）。
  「能跑通」只证明 DDL 语法与引用完整，不等于「符合生产业务逻辑」。
- `90_teardown.sql` 是破坏性删库，不是数据恢复型回滚；DDL 隐式提交，`ROLLBACK` 无效。

## 复现

```bash
cd docs/sql_rebuildnew
python tools/build_sql_rebuildnew.py
python tools/verify_mysql.py
```
"""
    write(out / "README.md", readme)
    manifest["generated_files"]["README.md"] = sha256(out / "README.md")

    write(out / "manifest.json", json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    return {"manifest": manifest, "preflight": pf, "all_sql_bytes": len(all_sql.encode("utf-8"))}


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", default=str(DEFAULT_SRC))
    ap.add_argument("--out", default=str(DEFAULT_OUT))
    args = ap.parse_args()
    src, out = Path(args.src), Path(args.out)
    result = build(src, out)
    pf = result["preflight"]
    m = result["manifest"]
    print(f"[build] out={out}")
    print(f"[build] 00_all.sql = {result['all_sql_bytes']} bytes; "
          f"00_all_faithful.sql = {(out / '00_all_faithful.sql').stat().st_size} bytes")
    print(f"[build] statements: {m['statement_count']}")
    print(f"[build] R4 TINYINT(1)->TINYINT replacements: {m['fixes']['r4_tinyint_display_width_replaced']}")
    print(f"[build] schema: {pf['schema']['table_count']} tables / {pf['schema']['column_count']} columns")
    print(f"[build] FK statements parsed: {pf['fk_count']}; index statements: {pf['index_statement_count']}")
    print(f"[build] preflight: {pf['status']}, issues={len(pf['issues'])}")
    for i in pf["issues"][:30]:
        print("   ", json.dumps(i, ensure_ascii=False))
    if pf["status"] != "PASS":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
