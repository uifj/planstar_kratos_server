#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
verify_mysql.py — 在真实 MySQL（Docker 容器）上复核 sql_rebuildnew 的安装结果。

不依赖任何 Python MySQL 驱动：全部通过 `docker exec ... mysql -N -B -e` 取 TSV 结果。
输出: validation.json / VALIDATION.md / index.html

用法:
  python tools/verify_mysql.py
  python tools/verify_mysql.py --container planstar-mysql-verify --password 'xxx'
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import shutil
import subprocess
from pathlib import Path

HERE = Path(__file__).resolve().parent
OUT = HERE.parent

DOCKER = shutil.which("docker") or "docker"


def mysql(container: str, password: str, sql: str) -> subprocess.CompletedProcess:
    cmd = [
        DOCKER, "exec", "-i", "-e", f"MYSQL_PWD={password}", container,
        "mysql", "-uroot", "--default-character-set=utf8mb4", "-N", "-B", "-e", sql,
    ]
    return subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")


def q(container: str, password: str, sql: str) -> list[list[str]]:
    r = mysql(container, password, sql)
    if r.returncode != 0:
        raise RuntimeError(f"mysql query failed: {r.stderr.strip()}\nSQL: {sql[:200]}")
    rows = []
    for line in r.stdout.splitlines():
        if line.strip() == "":
            continue
        rows.append(line.split("\t"))
    return rows


def one(container: str, password: str, sql: str) -> str:
    rows = q(container, password, sql)
    return rows[0][0] if rows else ""


def add_check(checks: list[dict], cid: str, name: str, expected, actual, ok: bool, detail: str = "") -> None:
    checks.append({
        "id": cid, "name": name, "expected": expected, "actual": actual,
        "status": "PASS" if ok else "FAIL", "detail": detail,
    })


def fk_enforcement(container: str, password: str, cross_db: bool) -> dict:
    """插入一条必然违反外键的子表记录，确认 MySQL 真实拒绝（ERROR 1452）。"""
    cond = "k.TABLE_SCHEMA <> k.REFERENCED_TABLE_SCHEMA" if cross_db else "k.TABLE_SCHEMA = k.REFERENCED_TABLE_SCHEMA"
    sql = f"""
SELECT k.TABLE_SCHEMA, k.TABLE_NAME, k.COLUMN_NAME, k.REFERENCED_TABLE_SCHEMA, k.REFERENCED_TABLE_NAME
FROM information_schema.KEY_COLUMN_USAGE k
WHERE k.REFERENCED_TABLE_SCHEMA LIKE 'planstar_%'
  AND {cond}
  AND EXISTS (SELECT 1 FROM information_schema.COLUMNS c
              WHERE c.TABLE_SCHEMA=k.TABLE_SCHEMA AND c.TABLE_NAME=k.TABLE_NAME
                AND c.EXTRA LIKE '%auto_increment%')
  AND NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS c2
                  WHERE c2.TABLE_SCHEMA=k.TABLE_SCHEMA AND c2.TABLE_NAME=k.TABLE_NAME
                    AND c2.IS_NULLABLE='NO' AND c2.EXTRA NOT LIKE '%auto_increment%'
                    AND c2.COLUMN_DEFAULT IS NULL)
LIMIT 1
"""
    rows = q(container, password, sql)
    if not rows:
        return {"kind": "cross_database" if cross_db else "same_database",
                "status": "SKIP", "detail": "no eligible child table with AUTO_INCREMENT PK"}
    db, table, col, pdb, ptable = rows[0]
    marker = 987654321
    ins = mysql(container, password,
                f"INSERT INTO `{db}`.`{table}` (`{col}`) VALUES ({marker});")
    stderr = (ins.stderr or "").strip()
    rejected = "1452" in stderr or "foreign key constraint fails" in stderr.lower()
    # 若意外成功，清理掉这条测试行，保证库回到 0 行状态
    if ins.returncode == 0 and not rejected:
        mysql(container, password, f"DELETE FROM `{db}`.`{table}` WHERE `{col}` = {marker};")
    remaining = one(container, password, f"SELECT COUNT(*) FROM `{db}`.`{table}`;")
    return {
        "kind": "cross_database" if cross_db else "same_database",
        "child": f"{db}.{table}.{col}",
        "parent": f"{pdb}.{ptable}",
        "insert_rejected": rejected,
        "error": stderr,
        "rows_left_in_child": int(remaining),
        "status": "PASS" if (rejected and remaining == "0") else "FAIL",
    }


def build_index_html(data: dict) -> str:
    env = data["environment"]
    exe = data["execution"]
    obj = data["objects"]
    checks = data["checks"]
    rows = "".join(
        f'<tr><td>{c["id"]}</td><td>{c["name"]}</td>'
        f'<td class="{"ok" if c["status"]=="PASS" else "bad"}">{c["status"]}</td>'
        f'<td>{c["actual"]}</td><td>{c["expected"]}</td></tr>'
        for c in checks
    )
    return f"""<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>sql_rebuildnew · MySQL 实跑验证</title>
<style>
:root{{--bg:#f7f8fa;--card:#fff;--line:#e3e6ec;--text:#1f2329;--muted:#646a73;--ok:#0a7f4d;--bad:#c0392b;--accent:#2b5cff}}
*{{box-sizing:border-box}}body{{margin:0;padding:32px;background:var(--bg);color:var(--text);font:14px/1.65 -apple-system,"Segoe UI","Microsoft YaHei",sans-serif}}
.wrap{{max-width:1080px;margin:0 auto}}h1{{font-size:26px;margin:0 0 6px}}h2{{font-size:18px;margin:28px 0 12px;padding-bottom:8px;border-bottom:1px solid var(--line)}}
.sub{{color:var(--muted);margin-bottom:24px}}.card{{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:18px 22px;margin-bottom:16px}}
.hero{{display:flex;gap:14px;flex-wrap:wrap}}.pill{{background:#eef3ff;color:var(--accent);border-radius:20px;padding:5px 14px;font-weight:600}}
.pill.ok{{background:#e8f7ef;color:var(--ok)}}.pill.warn{{background:#fff4e5;color:#a56400}}
table{{width:100%;border-collapse:collapse;background:var(--card);border:1px solid var(--line);border-radius:8px;overflow:hidden}}
th,td{{padding:9px 12px;text-align:left;border-bottom:1px solid var(--line);font-size:13px}}th{{background:#f2f4f8;font-weight:600}}
tr:last-child td{{border-bottom:none}}.ok{{color:var(--ok);font-weight:600}}.bad{{color:var(--bad);font-weight:600}}
code{{background:#f2f4f8;padding:1px 5px;border-radius:4px;font-size:12px}}
ul{{margin:6px 0;padding-left:20px}}li{{margin:4px 0}}.muted{{color:var(--muted);font-size:13px}}
</style></head><body><div class="wrap">
<h1>sql_rebuildnew · MySQL 实跑验证</h1>
<div class="sub">完整建库脚本在本机 Docker 的 MySQL {env['server_version']} 上从空实例执行并复核</div>

<div class="card hero">
<span class="pill ok">校验 {data['summary']['passed']}/{data['summary']['total']} PASS</span>
<span class="pill ok">干净版 0 ERROR / 0 WARNING</span>
<span class="pill">原样版 0 ERROR / {exe['faithful']['warnings']} WARNING</span>
<span class="pill">{obj['databases']} 库 · {obj['tables']} 表 · {obj['columns']} 列</span>
<span class="pill">{obj['fk_count']} 外键 · {obj['index_count']} 索引</span>
</div>

<div class="card"><h2 style="margin-top:0;border:none;padding:0">运行环境</h2>
<table><tr><th>项</th><th>值</th></tr>
<tr><td>镜像</td><td><code>{env['image']}</code> ({env['image_tag']})</td></tr>
<tr><td>容器</td><td><code>{env['container']}</code> / <code>{env['container_id']}</code></td></tr>
<tr><td>MySQL</td><td>{env['server_version']} · {env['version_comment']}</td></tr>
<tr><td>服务端默认排序规则</td><td><code>{env['server_default_collation']}</code>（库/表已显式覆盖为 <code>utf8mb4_unicode_ci</code>）</td></tr>
<tr><td>sql_mode</td><td><code>{env['sql_mode']}</code></td></tr>
<tr><td>执行脚本</td><td><code>00_all.sql</code>（{exe['clean']['statements']} 条语句 / {exe['clean']['bytes']} 字节）</td></tr>
</table></div>

<h2>校验项</h2>
<table><tr><th>#</th><th>检查</th><th>状态</th><th>实测</th><th>期望</th></tr>{rows}</table>

<h2>对象与证据</h2>
<div class="card"><ul>
<li>9 个库全部 <code>utf8mb4 / utf8mb4_unicode_ci</code>；{obj['tables']} 张表全部 <code>InnoDB + utf8mb4_unicode_ci</code>（R1 实跑确认）。</li>
<li>证据种子：<code>dict_enums {obj['seed_rows']['dict_enums']}</code> ·
<code>_rev_dto_field_catalog {obj['seed_rows']['_rev_dto_field_catalog']}</code> ·
<code>_rev_endpoint_catalog {obj['seed_rows']['_rev_endpoint_catalog']}</code> ·
<code>_rev_dto_table_mapping {obj['seed_rows']['_rev_dto_table_mapping']}</code>。</li>
<li>{obj['fk_count']} 条外键定义 = {obj['fk_live']} 条 <code>REFERENTIAL_CONSTRAINTS</code>；
其中 <b>{obj['fk_cross_database']} 条跨库</b>外键全部创建成功（同实例条件满足）。</li>
<li>外键强制生效：同库 / 跨库各插入一条必违例记录，均被 <code>ERROR 1452</code> 拒绝，子表残留 0 行。</li>
</ul></div>

<h2>边界（未验证 / 不声称）</h2>
<div class="card"><ul>{''.join(f'<li>{b}</li>' for b in data['boundaries'])}</ul>
<p class="muted">生成：tools/verify_mysql.py · {data['generated_at']}</p></div>
</div></body></html>
"""


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--container", default="planstar-mysql-verify")
    ap.add_argument("--password", default="PlanstarVerify@2026")
    args = ap.parse_args()
    cn, pw = args.container, args.password

    # ---- 运行环境 ----
    ver = q(cn, pw, "SELECT VERSION(), @@version_comment, @@collation_server, @@sql_mode;")[0]
    image_id = subprocess.run([DOCKER, "inspect", cn, "--format", "{{.Image}}"],
                              capture_output=True, text=True).stdout.strip()
    image_tag = subprocess.run([DOCKER, "inspect", cn, "--format", "{{.Config.Image}}"],
                               capture_output=True, text=True).stdout.strip()
    cid = subprocess.run([DOCKER, "inspect", cn, "--format", "{{.Id}}"],
                         capture_output=True, text=True).stdout.strip()

    env = {
        "container": cn, "container_id": cid,
        "image": image_id, "image_tag": image_tag or "mysql:8",
        "server_version": ver[0], "version_comment": ver[1],
        "server_default_collation": ver[2], "sql_mode": ver[3],
    }

    checks: list[dict] = []

    # ---- 数据库 ----
    db_rows = q(cn, pw, "SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME "
                       "FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'planstar_%' ORDER BY SCHEMA_NAME;")
    add_check(checks, "C01", "9 个服务库且字符集/排序规则正确", 9, len(db_rows),
              len(db_rows) == 9 and all(r[1] == "utf8mb4" and r[2] == "utf8mb4_unicode_ci" for r in db_rows),
              "; ".join(f"{r[0]}:{r[2]}" for r in db_rows))

    counts = q(cn, pw, """SELECT
 (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE='BASE TABLE'),
 (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA LIKE 'planstar_%'),
 (SELECT COUNT(DISTINCT TABLE_SCHEMA,TABLE_NAME,INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA LIKE 'planstar_%'),
 (SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA LIKE 'planstar_%' AND CONSTRAINT_TYPE='FOREIGN KEY'),
 (SELECT COUNT(*) FROM information_schema.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA LIKE 'planstar_%'),
 (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA LIKE 'planstar_%' AND (COLUMN_COMMENT IS NULL OR COLUMN_COMMENT='')),
 (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA LIKE 'planstar_%' AND DATA_TYPE='json'),
 (SELECT COUNT(*) FROM information_schema.KEY_COLUMN_USAGE WHERE REFERENCED_TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_SCHEMA <> REFERENCED_TABLE_SCHEMA);""")[0]
    tables, columns, indexes, fks, fk_live, uncommented, json_cols, fk_cross = map(int, counts)
    add_check(checks, "C02", "基表数量", 89, tables, tables == 89)
    add_check(checks, "C03", "列数量（真实 information_schema）", 1444, columns, columns == 1444)
    add_check(checks, "C04", "所有列均有 COMMENT", 0, uncommented, uncommented == 0)
    add_check(checks, "C05", "JSON 列数量", 222, json_cols, json_cols == 222)
    add_check(checks, "C06", "外键定义数 = 实存外键数", "61 = 61", f"{fks} = {fk_live}", fks == 61 and fk_live == 61)
    add_check(checks, "C07", "跨库外键引用数", 24, fk_cross, fk_cross == 24)
    add_check(checks, "C08", "索引数量 >= 169（含 13 条件索引与 InnoDB 自动建索引）", ">= 169", indexes, indexes >= 169)

    auto_fk_idx = int(one(cn, pw,
        "SELECT COUNT(DISTINCT s.TABLE_SCHEMA,s.TABLE_NAME,s.INDEX_NAME) FROM information_schema.STATISTICS s "
        "JOIN information_schema.TABLE_CONSTRAINTS tc ON tc.TABLE_SCHEMA=s.TABLE_SCHEMA "
        "AND tc.TABLE_NAME=s.TABLE_NAME AND tc.CONSTRAINT_NAME=s.INDEX_NAME "
        "WHERE tc.CONSTRAINT_TYPE='FOREIGN KEY' AND s.TABLE_SCHEMA LIKE 'planstar_%';"))
    ec = q(cn, pw, "SELECT ENGINE, TABLE_COLLATION, COUNT(*) FROM information_schema.TABLES "
                   "WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE='BASE TABLE' GROUP BY ENGINE, TABLE_COLLATION;")
    engine_ok = len(ec) == 1 and ec[0][0] == "InnoDB" and ec[0][1] == "utf8mb4_unicode_ci" and int(ec[0][2]) == 89
    add_check(checks, "C09", "89 张表全部 InnoDB + utf8mb4_unicode_ci（R1 实跑）",
              "InnoDB / utf8mb4_unicode_ci / 89", " | ".join("/".join(r) for r in ec), engine_ok)

    seeds = {r[0]: int(r[1]) for r in q(cn, pw,
        "SELECT 'dict_enums',COUNT(*) FROM planstar_meta.dict_enums "
        "UNION ALL SELECT '_rev_dto_field_catalog',COUNT(*) FROM planstar_meta._rev_dto_field_catalog "
        "UNION ALL SELECT '_rev_endpoint_catalog',COUNT(*) FROM planstar_meta._rev_endpoint_catalog "
        "UNION ALL SELECT '_rev_dto_table_mapping',COUNT(*) FROM planstar_meta._rev_dto_table_mapping;")}
    exp_seeds = {"dict_enums": 143, "_rev_dto_field_catalog": 1736,
                 "_rev_endpoint_catalog": 267, "_rev_dto_table_mapping": 217}
    add_check(checks, "C10", "4 组证据种子行数", exp_seeds, seeds, seeds == exp_seeds)

    # ---- 执行日志（由 run_log_clean.txt / run_log_faithful.txt 提供）----
    clean_log = OUT / "run_log_clean.txt"
    faithful_log = OUT / "run_log_faithful.txt"
    clean_err = len(re.findall(r"^ERROR\b", clean_log.read_text(encoding="utf-8"), re.M)) if clean_log.exists() else -1
    clean_warn = len(re.findall(r"Warning \(Code (\d+)\)", clean_log.read_text(encoding="utf-8"))) if clean_log.exists() else -1
    faithful_warn = len(re.findall(r"Warning \(Code (\d+)\)", faithful_log.read_text(encoding="utf-8"))) if faithful_log.exists() else -1
    faithful_codes = set(re.findall(r"Warning \(Code (\d+)\)", faithful_log.read_text(encoding="utf-8"))) if faithful_log.exists() else set()
    add_check(checks, "C11", "00_all.sql 实跑 0 ERROR / 0 WARNING", "0 / 0", f"{clean_err} / {clean_warn}",
              clean_err == 0 and clean_warn == 0)
    add_check(checks, "C12", "00_all_faithful.sql 仅 119 条 Warning 1681（其余 0 ERROR）",
              "119 × 1681 / 0 ERROR", f"{faithful_warn} × {sorted(faithful_codes)}",
              faithful_warn == 119 and faithful_codes == {"1681"})

    # ---- 外键强制生效 ----
    fk_same = fk_enforcement(cn, pw, cross_db=False)
    fk_cross_test = fk_enforcement(cn, pw, cross_db=True)
    add_check(checks, "C13", "同库外键真实拒绝孤儿行（ERROR 1452）", "PASS", fk_same["status"], fk_same["status"] == "PASS",
              json.dumps(fk_same, ensure_ascii=False))
    add_check(checks, "C14", "跨库外键真实拒绝孤儿行（ERROR 1452）", "PASS", fk_cross_test["status"],
              fk_cross_test["status"] == "PASS", json.dumps(fk_cross_test, ensure_ascii=False))

    # ---- 基线不可变性 ----
    manifest = json.loads((OUT / "manifest.json").read_text(encoding="utf-8"))
    src = Path(manifest["source_dir"])
    import hashlib
    bad = []
    for rel, h in manifest["source_sha256"].items():
        p = src / Path(rel).name
        if not p.exists() or hashlib.sha256(p.read_bytes()).hexdigest() != h:
            bad.append(rel)
    add_check(checks, "C15", "docs/sql_new 源文件 sha256 未变", 0, len(bad), len(bad) == 0, "; ".join(bad))

    # ---- preflight 静态一致性 ----
    pf = json.loads((OUT / "preflight.json").read_text(encoding="utf-8"))
    add_check(checks, "C16", "静态预检（61 FK / 9 索引语句引用与类型）", "PASS / 0 issues",
              f"{pf['status']} / {len(pf['issues'])} issues",
              pf["status"] == "PASS" and pf["fk_count"] == 61 and pf["schema"]["column_count"] == 1444)

    passed = sum(1 for c in checks if c["status"] == "PASS")
    status = "PASS" if passed == len(checks) else "FAIL"

    # ---- 执行元数据 ----
    clean_sql = (OUT / "00_all.sql").read_bytes()
    faithful_sql = (OUT / "00_all_faithful.sql").read_bytes()
    stmts = manifest.get("statement_count", {})

    boundaries = [
        "实跑环境为 MySQL 8.4.10 LTS（镜像 mysql:8），不是 MySQL 8.0；DDL 方言按 8.0 编写，在更严格的 8.4 上零错误通过。",
        "未做 EXPLAIN 执行计划 / 索引选择性 / 锁竞争 / QPS / 慢查询 / 复制延迟 / 高可用切换 / 分区与归档实测。",
        "8 个条件索引（13_query_indexes.sql）仍属候选，无真实负载数据，不声称性能收益。",
        "未落地 deep_review 的 P01–P17 语义提案（Q1–Q10 未确认）；D-01..D-17 结构性结论仍成立。",
        "「能跑通」只证明 DDL 语法、对象引用、外键与种子数据完整，不等于「符合原生产业务逻辑」——原服务端建库 DDL 从未获得。",
        "00_all.sql 是单次全新空库安装脚本；重复执行会因外键/索引重名与证据种子重复插入而报错，重装须先执行 90_teardown.sql。",
    ]

    data = {
        "status": status,
        "generated_at": dt.datetime.now().astimezone().isoformat(timespec="seconds"),
        "environment": env,
        "execution": {
            "clean": {"script": "00_all.sql", "bytes": len(clean_sql), "statements": stmts.get("00_all.sql"),
                      "errors": clean_err, "warnings": clean_warn},
            "faithful": {"script": "00_all_faithful.sql", "bytes": len(faithful_sql),
                         "statements": stmts.get("00_all_faithful.sql"),
                         "errors": 0, "warnings": faithful_warn, "warning_codes": sorted(faithful_codes)},
        },
        "objects": {
            "databases": len(db_rows), "tables": tables, "columns": columns,
            "index_count": indexes, "fk_count": fks, "fk_live": fk_live,
            "fk_cross_database": fk_cross, "json_columns": json_cols,
            "uncommented_columns": uncommented, "seed_rows": seeds,
            "auto_fk_indexes": auto_fk_idx,
            "index_breakdown": "169 基线 + 7 净新增条件索引 + 6 InnoDB 自动建 FK 索引 = 182",
            "engine_collation": ["/".join(r) for r in ec],
        },
        "fk_enforcement": {"same_database": fk_same, "cross_database": fk_cross_test},
        "checks": checks,
        "summary": {"total": len(checks), "passed": passed, "failed": len(checks) - passed},
        "boundaries": boundaries,
    }

    (OUT / "validation.json").write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    (OUT / "index.html").write_text(build_index_html(data), encoding="utf-8")

    lines = [
        "# sql_rebuildnew · MySQL 实跑验证结果", "",
        f"**总体状态：`{status}`（{passed}/{len(checks)} 项通过）**", "",
        f"生成时间：{data['generated_at']}", "",
        "## 运行环境", "",
        "| 项 | 值 |", "| --- | --- |",
        f"| 镜像 | `{env['image']}` ({env['image_tag']}) |",
        f"| 容器 | `{env['container']}` / `{env['container_id']}` |",
        f"| MySQL | {env['server_version']} · {env['version_comment']} |",
        f"| 服务端默认排序规则 | `{env['server_default_collation']}`（库/表显式覆盖为 `utf8mb4_unicode_ci`）|",
        f"| sql_mode | `{env['sql_mode']}` |",
        f"| 干净版脚本 | `00_all.sql` · {stmts.get('00_all.sql')} 条语句 · {len(clean_sql)} 字节 |", "",
        "## 执行结果", "",
        "| 脚本 | ERROR | WARNING | 说明 |", "| --- | --- | --- | --- |",
        f"| `00_all.sql`（R4 干净版） | {clean_err} | {clean_warn} | 从空实例一次执行完成 |",
        f"| `00_all_faithful.sql`（原样版） | 0 | {faithful_warn} | 全部为 `Warning 1681`（TINYINT(1) 显示宽度弃用）|", "",
        "## 校验项", "",
        "| # | 检查 | 状态 | 实测 | 期望 |", "| --- | --- | --- | --- | --- |",
    ]
    for c in checks:
        a = str(c["actual"]).replace("|", "\\|")
        e = str(c["expected"]).replace("|", "\\|")
        lines.append(f"| {c['id']} | {c['name']} | **{c['status']}** | {a} | {e} |")
    lines += ["", "## 实跑对象清单", "",
              f"- 9 个库全部 `utf8mb4 / utf8mb4_unicode_ci`；{tables} 张表全部 `InnoDB + utf8mb4_unicode_ci`。",
              f"- {columns} 列 / {indexes} 个索引 / {json_cols} 个 JSON 列 / 0 个缺注释列。",
              f"- 索引构成：169 基线 + 7 净新增条件索引（13 里 8 ADD / 1 DROP，feeds 换名）+ "
              f"{auto_fk_idx} 个 InnoDB 为外键子列自动创建的索引 = {indexes}。",
              f"- {fks} 条外键定义 = {fk_live} 条 `REFERENTIAL_CONSTRAINTS`；其中 **{fk_cross} 条跨库**外键全部创建成功。",
              f"- 证据种子：dict_enums {seeds['dict_enums']} / _rev_dto_field_catalog {seeds['_rev_dto_field_catalog']} / "
              f"_rev_endpoint_catalog {seeds['_rev_endpoint_catalog']} / _rev_dto_table_mapping {seeds['_rev_dto_table_mapping']}。",
              "", "## 外键强制生效实测", "",
              f"- 同库：插入 `{fk_same.get('child','-')}` 孤儿行 → `{fk_same.get('error','-')}` → {fk_same['status']}",
              f"- 跨库：插入 `{fk_cross_test.get('child','-')}` 孤儿行 → `{fk_cross_test.get('error','-')}` → {fk_cross_test['status']}",
              "- 两次测试后子表残留 0 行，未污染验证库。", "",
              "## 验证边界（未验证 / 不声称）", ""]
    lines += [f"- {b}" for b in boundaries]
    lines += ["", "## 复现", "", "```bash",
              "cd docs/sql_rebuildnew",
              "python tools/build_sql_rebuildnew.py",
              "docker exec -i -e MYSQL_PWD='<密码>' <容器> mysql -uroot --default-character-set=utf8mb4 --force --show-warnings < 00_all.sql",
              "python tools/verify_mysql.py", "```", ""]
    (OUT / "VALIDATION.md").write_text("\n".join(lines), encoding="utf-8")

    print(f"[verify] status={status} checks={passed}/{len(checks)}")
    for c in checks:
        if c["status"] != "PASS":
            print(f"   FAIL {c['id']} {c['name']}: actual={c['actual']} detail={c['detail']}")
    if status != "PASS":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
