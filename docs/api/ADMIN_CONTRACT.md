# 企划星 · 运营后台契约（举报处理台 · 企划审核）

> 仅暴露在 `ADMIN_ADDR` 管理端口（容器内 `0.0.0.0:9108`，宿主机仅发布 `127.0.0.1:9108`），
> 与业务端口 `HTTP_ADDR`（`8098`）分离，不进入 Flutter 客户端契约。
> 数据源：`planstar_catalog.project_report`、`planstar_catalog.project`。

## 0. 访问方式与鉴权

`/admin*` 与 `/metrics` 统一由 `OpsAuth` 保护：

| 要求 | 说明 |
| --- | --- |
| Bearer 令牌 | `Authorization: Bearer <PLANSTAR_OPS_TOKEN>`；缺失或错误返回 `401 UNAUTHORIZED` 与 `WWW-Authenticate: Bearer` |
| 来源白名单 | 可选 `PLANSTAR_OPS_ALLOWED_CIDRS`（逗号分隔 CIDR）；不在白名单返回 `403 FORBIDDEN`。取 TCP 对端地址，不信任 `X-Forwarded-For` |
| 生产必填 | `server.env=prod` 且令牌为空时进程启动失败（`生产环境必须配置 PLANSTAR_OPS_TOKEN`） |
| 本地开发 | 令牌为空时鉴权关闭，仅适合本地；每个端点打印一次 `ops auth disabled` 警告 |

示例：

```bash
curl -fsS -H "Authorization: Bearer $PLANSTAR_OPS_TOKEN" \
  http://127.0.0.1:9108/admin/reports?status=pending

curl -fsS -H "Authorization: Bearer $PLANSTAR_OPS_TOKEN" \
  http://127.0.0.1:9108/metrics | head
```

`GET /admin/console` 同样受 Bearer 保护：浏览器地址栏无法附带 `Authorization` 头，
直接打开会收到 401。生产应通过内网反向代理注入凭据，或使用 `curl`/专用客户端访问。

### 0.1 探针端点摘要（匿名）

| 端点 | 成功 | 失败 | 语义 |
| --- | --- | --- | --- |
| `GET /healthz` | `200 {"status":"ok"}` | 连接失败 | 只表示进程存活，不检查依赖 |
| `GET /readyz` | `200 {"ready":true,"status":"ok\|degraded","checks":[...]}` | 关键依赖失败/启动或停机闸门关闭：`503 {"ready":false,"status":"fail"}` | 并发检查 MySQL/Redis（critical）与 ES/RabbitMQ/Consul（optional），带 2s 缓存与退避 |

`checks[]` 中 `status` 为 `ok`/`fail`/`skipped`，`skipped` 表示依赖未配置。
完整字段与缓存/超时语义见 [../operations.md](../operations.md) 第 1 节。

### 0.2 审计字段

每次访问与写操作输出 `msg="ops audit"`：

| 字段 | 取值 |
| --- | --- |
| `audit` | 固定 `true` |
| `actor` | `token:<sha256 前 8 位>` / `anonymous` / `system` |
| `action` | `ops.metrics.access`、`ops.admin.access`、`report.handle`、`project.review`、`ops.probe.state` |
| `target` | 端点名、举报 ID、企划 ID、`readyz` |
| `result` | `ok`/`client_error`/`error`/`unauthorized`/`forbidden` 或业务结果（`success`/`blocked`/`not_found` 等） |
| `remote_addr` | TCP 对端地址（仅端点访问） |
| `status` | HTTP 状态码（仅端点访问） |

审计不会记录令牌明文；Loki 查询：`{service="planstar-api"} |= "ops audit"`。

## 1. 举报列表

`GET /admin/reports`

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | string | 否 | `pending` / `accepted` / `rejected`；为空表示全部 |
| `limit` | int | 否 | 默认 50，上限 200 |

响应：

```json
{
  "items": [
    {
      "reportId": 9001,
      "targetType": "project",
      "targetId": 1004,
      "projectId": 1004,
      "userId": 6,
      "reasonCode": "fraud",
      "description": "怀疑众筹页面演示素材与他人作品高度相似，请核实原创性。",
      "evidence": [],
      "status": "accepted",
      "handleNote": "已要求创作者补充素材授权说明，页面暂标注「审核中」。",
      "handleTime": "2026-11-03T15:00:00+08:00",
      "gmtCreate": "2026-11-02T09:12:00+08:00"
    }
  ],
  "total": 1
}
```

`total` 为本次返回条数（受 `limit` 约束），不是全量计数。

错误：`400 BAD_REQUEST`（status 非法）、`500 INTERNAL`。

## 2. 举报处置

`POST /admin/reports/{id}/handle`

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | string | 是 | `accepted`（举报成立）/ `rejected`（举报不成立） |
| `note` | string | 否 | 处理结论；为空时按状态写入默认结论 |

响应：更新后的单条举报（结构同列表项，`evidence` 为字符串数组）。

错误：`400 BAD_REQUEST`、`404 NOT_FOUND`（举报单不存在）、`500 INTERNAL`。

## 3. 企划审核视图

`GET /admin/projects/{id}`

响应：

```json
{
  "projectId": 4006,
  "title": "某原创企划",
  "subtitle": "副标题",
  "categoryId": 4,
  "categoryName": "同人企划",
  "publishStatus": 2,
  "fundingStatus": "preheating",
  "hits": [],
  "blocked": false
}
```

`hits[]` 为黑名单命中项：`rule`（金融集资 / 慈善募捐 / 黄牛票务 / 违法违规 / 特殊资质 / 预付卡 / 空气企划）、
`keyword`（命中词或缺失项）。`blocked = hits.length > 0`。

说明：不过滤 `publish_status`，草稿（0）与驳回（3）企划同样可查，便于复审。

错误：`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`。

## 4. 企划审核（发布闸门）

`POST /admin/projects/{id}/review`

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `decision` | string | 是 | `approve`（通过，`publish_status=1`）/ `reject`（驳回，`publish_status=3`） |

响应：审核后的企划审核视图（结构同第 3 节）。

黑名单硬拦截规则（与 `POSITIONING.md` 9.3 一致，命中任意一条即禁止 `approve`）：

| 规则 | 判定方式 |
| --- | --- |
| 金融集资 | 标题/副标题/品类名/故事/资金用途/简介命中：股权、分红、利息、理财、虚拟货币、加密货币、比特币、usdt、集资、投资回报、保本、收益承诺 |
| 慈善募捐 | 命中：慈善、募捐、捐助、医疗救助、个人求助、筹款治病 |
| 黄牛票务 | 命中：代购、黄牛、转售门票、门票转售、抢票、代抢、现货 |
| 违法违规 | 命中：盗版、侵权、涉黄、赌博、传销、刷单、刷量、虚假宣传 |
| 特殊资质 | 命中：药品、医疗器械、保健品、活体动物、食品经营、处方药 |
| 预付卡 | 命中：预付卡、储值、预付式消费、资金池 |
| 空气企划 | `goal_amount <= 0`、交付计划为空、风险说明为空，任一成立即命中 |

`approve` 命中黑名单时返回 `400`，并附原因：

```json
{
  "code": "BAD_REQUEST",
  "message": "企划命中品类黑名单：金融集资(分红)",
  "reasons": [
    {"rule": "金融集资", "keyword": "分红"}
  ]
}
```

`reject` 不受黑名单限制，始终允许执行。

错误：`400 BAD_REQUEST`（decision 非法或命中黑名单）、`404 NOT_FOUND`、`500 INTERNAL`。

## 5. 操作台入口

`GET /admin/console`（别名 `GET /admin`、`GET /admin/`）

返回单页 HTML（`text/html; charset=utf-8`，无外部静态资源依赖），包含：

- **举报处理台**：按状态筛选、查看举报详情、填写处理结论并标记「成立 / 不成立」，调用第 1、2 节 JSON API。
- **企划审核（发布闸门）**：按企划 ID 查询黑名单判定结果，展示命中项；命中时禁用「通过并发布」，可直接驳回，调用第 3、4 节 JSON API。

页面仅能在管理端口打开；业务端口不注册该路径。

## 6. 安全约束

- 管理端口不参与业务网关路由，禁止公网直接暴露；compose 仅把 `9108` 发布到宿主机
  `127.0.0.1`，远程访问需经内网/跳板机或反向代理。
- 所有 `/admin*` 请求必须携带 `Authorization: Bearer <PLANSTAR_OPS_TOKEN>`；可叠加
  `PLANSTAR_OPS_ALLOWED_CIDRS` 限制来源。未授权返回 401，来源不允许返回 403。
- 处理动作写入 `status`、`handle_note`、`handle_time`，保留业务留痕；同时输出
  `ops audit` 结构化日志与 `planstar_ops_audit_total` 指标。
- 审计字段固定为 `audit/actor/action/target/result/remote_addr/status`，不记录令牌明文。
- 令牌轮换需同步更新 `PLANSTAR_OPS_TOKEN`、`planstar/.env` 与
  `planstar/deploy/prometheus/ops_token`；否则 Prometheus 抓取失败并触发
  `PlanstarOpsAuthFailureSpike`。
- 后续接入运营账号体系时，在 `OpsAuth` 之后增加 RBAC 中间件，不改动业务契约。
