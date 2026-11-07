> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组

# 运维手册（planstar 单体）

本文以当前代码为准：探针实现在 `planstar/internal/ops/readiness.go` 与
`planstar/internal/server/admin.go`，鉴权/审计在 `planstar/internal/server/ops_auth.go`，
指标与追踪在 `planstar/internal/observability/observability.go`。

## 1. 探针契约

### 1.1 端点总表

| 端点 | 监听 | 语义 | 鉴权 | 成功 | 失败 |
| --- | --- | --- | --- | --- | --- |
| `GET /healthz` | `9108`（容器内 `0.0.0.0:9108`；宿主机仅发布 `127.0.0.1:9108`） | **存活**：只表示进程与 HTTP server 正常，不检查任何依赖 | 匿名 | `200`，JSON `status="ok"` | 进程未启动/端口不可达 |
| `GET /readyz` | 同上 | **就绪**：并发检查已配置依赖，带缓存与退避 | 匿名 | `200`，`ready=true`，`status=ok` 或 `degraded` | 关键依赖失败或进程启动中/停机中：`503`，`ready=false`，`status=fail` |
| `GET /metrics` | 同上 | Prometheus / OpenMetrics 指标 | `Authorization: Bearer <PLANSTAR_OPS_TOKEN>` + 可选 CIDR | `200` | 缺少/错误令牌 `401`；来源不在白名单 `403` |
| `GET/POST /admin*` | 同上 | 运营后台（举报处置、企划审核） | 同 `/metrics` | `200` | `401` / `403` |

两个探针均返回 `Content-Type: application/json` 与 `Cache-Control: no-store`。

### 1.2 `/healthz` 完整响应

```json
{
  "service": "planstar.api",
  "version": "planstar.api.v1",
  "status": "ok",
  "timestamp": "2026-11-07T10:00:00.123456+08:00",
  "uptime_seconds": 3600
}
```

`/healthz` 永远不触发依赖探测，适合作为容器 `healthcheck`；容器编排的
`livenessProbe` 也只应指向这里。

### 1.3 `/readyz` 完整响应

健康：

```json
{
  "service": "planstar.api",
  "version": "planstar.api.v1",
  "status": "ok",
  "ready": true,
  "timestamp": "2026-11-07T10:00:00.123456+08:00",
  "checked_at": "2026-11-07T10:00:00.130456+08:00",
  "cached": false,
  "uptime_seconds": 3600,
  "checks": [
    {"name": "mysql", "critical": true, "status": "ok", "latency_ms": 1.2},
    {"name": "redis", "critical": true, "status": "ok", "latency_ms": 0.4},
    {"name": "elasticsearch", "critical": false, "status": "ok", "latency_ms": 6.8},
    {"name": "rabbitmq", "critical": false, "status": "skipped", "latency_ms": 0},
    {"name": "consul", "critical": false, "status": "skipped", "latency_ms": 0}
  ]
}
```

关键依赖失败（HTTP 503）：

```json
{
  "service": "planstar.api",
  "version": "planstar.api.v1",
  "status": "fail",
  "ready": false,
  "timestamp": "2026-11-07T10:00:00.123456+08:00",
  "checked_at": "2026-11-07T10:00:02.130456+08:00",
  "cached": false,
  "uptime_seconds": 3600,
  "checks": [
    {"name": "mysql", "critical": true, "status": "fail", "latency_ms": 2000.3, "error": "dial tcp 10.0.0.12:3306: i/o timeout"},
    {"name": "redis", "critical": true, "status": "ok", "latency_ms": 0.5},
    {"name": "elasticsearch", "critical": false, "status": "ok", "latency_ms": 6.1},
    {"name": "rabbitmq", "critical": false, "status": "skipped", "latency_ms": 0},
    {"name": "consul", "critical": false, "status": "skipped", "latency_ms": 0}
  ]
}
```

### 1.4 `checks[]` 语义

| 字段 | 含义 |
| --- | --- |
| `name` | 固定顺序：`mysql`、`redis`、`elasticsearch`、`rabbitmq`、`consul` |
| `critical` | 是否关键依赖。默认 `mysql=true`、`redis=true`，`elasticsearch`/`rabbitmq`/`consul=false`，可由 `ops.readiness.<name>.critical` 覆盖 |
| `status` | `ok` / `fail` / `skipped`。`skipped` 表示依赖未配置（Probe 为 nil），不计入失败 |
| `latency_ms` | 本次探测耗时（毫秒，保留三位小数） |
| `error` | 失败原因；已脱敏（URL/DSN 中的用户名口令替换为 `***`），最长 200 字符 |

聚合规则：

- 任一 `critical=true` 的检查为 `fail` → `status=fail`、`ready=false`、HTTP `503`。
- 只有非关键检查为 `fail` → `status=degraded`、`ready=true`、HTTP `200`。
- 全部 `ok`/`skipped` → `status=ok`、`ready=true`、HTTP `200`。

`skipped` 的触发条件（部署侧未配置）：Redis 地址为空、Elasticsearch 地址为空、
MQ 地址为空、Consul 地址为空。默认生产 compose 全部配置，因此正常应看到
`mysql`/`redis`/`elasticsearch`/`rabbitmq`/`consul` 均为 `ok`。

超时与缓存：

- 单项超时默认 `2s`（`ops.readiness.<name>.timeout_seconds`）。
- 整轮总超时默认 `3s`（`ops.probe_total_timeout_seconds`）；超时后未完成的检查
  记为 `fail`，`error="probe timed out"`。
- 结果缓存默认 `2s`（`ops.probe_cache_ttl_seconds`）。`cached=true` 表示本次响应
  来自缓存；`checked_at` 是最近一次真实探测完成时间，`timestamp` 是该轮探测开始时间。
- 启动闸门：`AfterStart` 之前 `/readyz` 直接返回 `503` 且 `checks` 为空。
- 停机闸门：收到 `SIGTERM`/`SIGINT` 后 `BeforeStop` 先标记 stopping，`/readyz`
  立即变为 `503`，让负载均衡摘流，再执行优雅停机。

## 2. 鉴权与审计

### 2.1 Bearer 令牌

- 令牌来源：`PLANSTAR_OPS_TOKEN` 环境变量，覆盖 `ops.token`；空值表示**关闭鉴权**
  （仅限本地开发），未配置时每个端点只打印一次 `ops auth disabled` 警告。
- 生产 fail-fast：`server.env=prod` 且令牌为空时，进程在启动阶段直接报错退出
  （`生产环境必须配置 PLANSTAR_OPS_TOKEN`）。
- 生成方式：

  ```bash
  openssl rand -hex 32
  # 或
  head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
  ```

  写入 `planstar/.env` 的 `PLANSTAR_OPS_TOKEN=...`，并同步到
  `planstar/deploy/prometheus/ops_token`（见第 3.4 节）。
- 校验使用常量时间比较；审计中的 actor 为 `token:<sha256 前 8 位>`，不记录令牌明文。

### 2.2 CIDR 白名单

- `PLANSTAR_OPS_ALLOWED_CIDRS=10.0.0.0/8,192.168.1.0/24`，逗号分隔；非法 CIDR
  启动即失败。
- 白名单为空表示不限制来源。
- 取 TCP 对端地址（`RemoteAddr`），**不信任 `X-Forwarded-For`**，避免伪造头绕过。
- 校验顺序：先判来源（失败 `403 FORBIDDEN`），再判令牌（失败 `401 UNAUTHORIZED`，
  并返回 `WWW-Authenticate: Bearer`）。

### 2.3 审计日志

所有运维端点访问与运营写操作都会输出一条结构化日志：

```text
msg="ops audit" audit=true actor=token:1a2b3c4d action=ops.metrics.access target=metrics
result=ok remote_addr=127.0.0.1:52344 status=200
```

| 字段 | 说明 |
| --- | --- |
| `audit` | 固定 `true`，便于 LogQL 精确匹配 |
| `actor` | `token:<hash8>` / `anonymous` / `system` |
| `action` | `ops.<endpoint>.access`、`report.handle`、`project.review`、`ops.probe.state` |
| `target` | 端点名、举报 ID、企划 ID 或 `readyz` |
| `result` | `ok` / `client_error` / `error` / `unauthorized` / `forbidden` / 业务结果（`success`、`blocked`、`not_found` 等） |
| `remote_addr` | TCP 对端地址（仅端点访问） |
| `status` | HTTP 状态码（仅端点访问） |

探针聚合状态发生变化时，另有一条 `actor=system`、`action=ops.probe.state` 的审计日志，
可直接与告警时间线对齐。

Loki 查询：

```logql
{service="planstar-api"} |= "ops audit"
{service="planstar-api"} |= "ops audit" |= "unauthorized"
{service="planstar-api"} |= "ops audit" |= "action=report.handle"
{service="planstar-api"} |= "ops audit" |= "action=ops.probe.state"
```

指标侧对应 `planstar_ops_audit_total{action,result}` 与
`planstar_ops_auth_failed_total{endpoint}`。

## 3. 指标目录

### 3.1 指标清单

`/metrics` 每次抓取返回全部指标，编码为 Prometheus 文本格式，并启用 OpenMetrics
（响应头 `Content-Type: application/openmetrics-text`；客户端可用
`Accept: application/openmetrics-text`）。

| 指标 | 类型 | 标签 | 用途 |
| --- | --- | --- | --- |
| `planstar_build_info` | gauge | `version`、`go_version`、`env` | 构建信息，值恒为 1；用于确认线上版本与镜像 tag 一致 |
| `planstar_http_requests_total` | counter | `method`、`route`、`status` | 请求量、5xx 比例、429 限流 |
| `planstar_http_request_duration_seconds` | histogram | `method`、`route` | P50/P95/P99 延迟；bucket 到 5s |
| `planstar_http_requests_in_flight` | gauge | `method` | 并发在途请求，判断堆积 |
| `planstar_http_request_size_bytes` | histogram | `method`、`route` | 请求体大小分布 |
| `planstar_http_response_size_bytes` | histogram | `method`、`route` | 响应体大小分布 |
| `planstar_http_active_connections` | gauge | — | 当前打开的 HTTP 连接数（按 `ConnState` 增减） |
| `planstar_ops_probe_status` | gauge | `probe`、`check` | 探针状态：`1=ok/skipped`，`0=fail` |
| `planstar_ops_probe_duration_seconds` | gauge | `probe`、`check` | 最近一次单项探测耗时 |
| `planstar_ops_probe_cache_hits_total` | counter | — | `/readyz` 命中缓存次数 |
| `planstar_ops_metrics_scrape_rejected_total` | counter | — | `/metrics` 因超过并发上限被拒绝的抓取次数 |
| `planstar_ops_auth_failed_total` | counter | `endpoint` | 运维端点鉴权失败次数（401/403） |
| `planstar_ops_audit_total` | counter | `action`、`result` | 审计操作计数 |
| `planstar_db_connections` | gauge | `domain`、`state`（`open`/`idle`/`in_use`/`max`） | 9 个 `planstar_*` 库连接池快照 |
| `planstar_db_wait_count_total` | counter | `domain` | 连接池累计等待连接次数 |
| `planstar_db_wait_duration_seconds_total` | counter | `domain` | 连接池累计等待时长 |
| `planstar_catalog_es_fallback_total` | counter | — | 目录查询因 ES 失败回退 MySQL 的次数 |
| `planstar_biz_login_total` | counter | `result`（`success`/`failure`/`captcha`） | 登录结果分布 |
| `planstar_biz_order_confirm_total` | counter | `result`（`success`/`failure`） | 确认报价结果 |
| `planstar_biz_order_created_total` | counter | `result`（`success`/`price_changed`/`sold_out`/`limit_exceeded`/`quote_expired`/`error`） | 下单结果，定位业务失败原因 |
| `go_*` | 多种 | — | goroutine、堆内存、GC |
| `process_*` | 多种 | — | CPU、FD、常驻内存 |

探针指标的 `probe` 标签有两个来源：

- `probe="readyz"`：`/readyz` handler 每次真实评估后写入，供告警使用。
- `probe="readiness"`：后台 `OnResult` 在同一轮评估后写入，供面板对照。
- 命中缓存时只累加 `planstar_ops_probe_cache_hits_total`，不更新状态/耗时。

### 3.2 基数控制

- `route` 使用归一化模板：数字 ID 段替换为 `:id`，16 进制长 token 同样归一化，
  不按原始 URL 打点。
- `check` 固定为 5 个依赖名；`endpoint` 固定为 `metrics`/`admin`；`status` 为标准
  三位状态码；`domain` 固定为 9 个逻辑库。
- 新增标签前必须评估组合数，禁止把用户 ID、手机号、订单号写入标签。

### 3.3 OpenMetrics 与 exemplar

`promhttp.HandlerOpts{EnableOpenMetrics: true}` 允许在 histogram bucket 上携带
exemplar（trace_id）。Grafana/Prometheus 需以 OpenMetrics 协商抓取；当前
`prometheus.yml` 未显式设置 `scrape_protocols`，Prometheus 默认会协商 OpenMetrics
并保留 exemplar。

### 3.4 Prometheus 抓取与令牌文件

`planstar/deploy/prometheus/prometheus.yml`：

```yaml
- job_name: planstar-api
  metrics_path: /metrics
  authorization:
    type: Bearer
    credentials_file: /etc/prometheus/ops_token
  static_configs:
    - targets: ["planstar-api:9108"]
```

- `ops_token` 由 `deploy.sh` 从 `$PLANSTAR_OPS_TOKEN` 写入，权限 `0600`，已在
  `planstar/.gitignore` 中忽略；compose 以只读方式挂载到
  `/etc/prometheus/ops_token`。
- 手动部署时必须自行创建该文件：

  ```bash
  cd planstar
  mkdir -p deploy/prometheus
  (umask 077; printf '%s' "$PLANSTAR_OPS_TOKEN" > deploy/prometheus/ops_token)
  chmod 600 deploy/prometheus/ops_token
  ```

- 抓取间隔 `15s`，评估间隔 `15s`；另有 `mysql`（mysqld-exporter `9104`）、
  `redis`（redis-exporter `9121`）、`consul`（`/v1/agent/metrics`）、
  `prometheus`、`otel-collector`（`8889`）job。
- 当前没有 RabbitMQ / Elasticsearch exporter；ES 降级用
  `planstar_catalog_es_fallback_total` 观察，RabbitMQ 用管理台 `15672` 与 DLQ 观察。

## 4. 告警清单与处置

规则文件：`planstar/deploy/prometheus/rules/alerts.yml`（5 组，共 12 条）。

| 告警 | 关键表达式 | 阈值 / 持续 | 级别 | 处置 |
| --- | --- | --- | --- | --- |
| `PlanstarServiceDown` | `up{job="planstar-api"} == 0` | `2m` | critical | 先核对 `deploy/prometheus/ops_token` 是否与 `PLANSTAR_OPS_TOKEN` 一致（401 也会表现为抓取失败），再看 `docker compose ps planstar-api` 与日志；确认进程/端口/网络 |
| `PlanstarReadinessFailing` | `planstar_ops_probe_status{probe="readyz"} == 0` | `3m` | critical | 按 `checks[]` 中 `status=fail` 的检查项排障；优先 `mysql`/`redis`，见第 7.3 节决策树 |
| `PlanstarHighErrorRate5xx` | 5xx 请求速率 / 总请求速率 | `> 5%`，`10m` | warning | Grafana RED 行定位 route/status；查 Loki `request failed`、`panic recovered`；确认 `/readyz` 与最近发布 |
| `PlanstarCriticalErrorRate5xx` | 同上 | `> 10%`，`5m` | critical | 用户已明显受损；立即检查依赖与最近发布，必要时回滚（deployment.md 第 3 节） |
| `PlanstarHighLatencyP95` | `histogram_quantile(0.95, ...request_duration_seconds_bucket)` | `> 2s`，`10m` | warning | 按 route 定位；查慢 SQL、ES 查询、Redis 阻塞、下游超时（请求总超时 15s） |
| `PlanstarInflightSaturation` | `sum(planstar_http_requests_in_flight)` | `> 50`，`5m` | warning | 看 in-flight 曲线、P95、goroutine 数；排查慢查询/连接池耗尽/突发流量 |
| `PlanstarRateLimitSpike` | `sum(rate(...status="429"[5m]))` | `> 1/s`，`10m` | warning | 查 `send code rate limited` 日志中的 `mobile`/`ip`/`dimension`；必要时网关临时封禁来源 IP |
| `PlanstarOrderFailureRate` | 非 success 下单 / 全部下单 | `> 20%`，`10m` | warning | 按 `result` 区分 `price_changed`/`sold_out`/`limit_exceeded`/`quote_expired`/`error`；`error` 高优先查 MySQL 写入与 MQ 发布 |
| `PlanstarESFallbackSpike` | `sum(rate(planstar_catalog_es_fallback_total[5m]))` | `> 0.5/s`，`10m` | warning | 查 ES `_cluster/health`、磁盘水位、JVM 堆；检索 `fallback to mysql` 日志 |
| `PlanstarDBConnectionSaturation` | `in_use / max`（按 domain） | `> 0.8`，`10m` | warning | 查慢查询/长事务/连接泄漏（`planstar_db_wait_count_total` 增长）；必要时临时调大连接池或重启泄漏实例 |
| `PlanstarInfraDown` | `up{job=~"mysql|redis|consul"} == 0` | `2m` | critical | 查对应 exporter/实例容器、端口与网络；MySQL/Redis 故障会直接让 `/readyz` 503 |
| `PlanstarOpsAuthFailureSpike` | `sum(rate(planstar_ops_auth_failed_total[5m]))` | `> 1/s`，`10m` | warning | 若刚轮换令牌，确认 `ops_token` 已同步；否则排查来源 IP 暴力探测，结合 `planstar_ops_audit_total` 与审计日志 |

### 4.1 Alertmanager 邮件通知

`planstar/deploy/alertmanager/alertmanager.yml` 只保留 `${AM_*}` 占位，由
`entrypoint.sh` 在启动前渲染到 `/tmp`（`umask 077`）后启动，避免把 SMTP 凭据写进仓库。

| 环境变量 | 含义 |
| --- | --- |
| `AM_SMTP_SMARTHOST` | SMTP 地址，如 `smtp.example.com:587` |
| `AM_SMTP_FROM` | 发件人，如 `planstar-alert@example.com` |
| `AM_SMTP_AUTH_USERNAME` | SMTP 用户名 |
| `AM_SMTP_AUTH_PASSWORD` | SMTP 密码/授权码 |
| `AM_ALERT_EMAIL_TO` | 收件人地址 |

- `smtp_require_tls: true`，本地 MailHog/smtp4dev（明文 1025）调试时临时改为 `false`，不要提交。
- `critical` 每 `1h` 重复提醒；`warning` 每 `4h`；`group_wait=30s`、`group_interval=5m`；
  恢复通知 `send_resolved: true`。
- 配置校验（本地）：

  ```bash
  # 先渲染占位再校验，直接对模板运行 amtool 会因 ${VAR} 报错
  ALERTMANAGER_TEMPLATE=deploy/alertmanager/alertmanager.yml \
  ALERTMANAGER_RENDERED=/tmp/alertmanager.yml \
  ALERTMANAGER_RENDER_ONLY=1 \
  AM_SMTP_SMARTHOST=smtp.example.com:587 AM_SMTP_FROM=alert@example.com \
  AM_SMTP_AUTH_USERNAME=user AM_SMTP_AUTH_PASSWORD=pass \
  AM_ALERT_EMAIL_TO=ops@example.com \
  sh deploy/alertmanager/entrypoint.sh >/dev/null
  docker run --rm --entrypoint amtool \
    -v /tmp/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro \
    prom/alertmanager:v0.27.0 check-config /etc/alertmanager/alertmanager.yml
  ```

## 5. OpenTelemetry 链路追踪

### 5.1 拓扑

```text
planstar-api --OTLP/gRPC--> otel-collector:4317 --batch--> jaeger:4317
                                                        |
                                                        +--> Jaeger UI :16686
```

- 应用只面向 collector，不直连 Jaeger；collector 统一做 `memory_limiter`（512MiB）
  与 `batch`（5s）。
- collector 同时开放 OTLP/HTTP `4318`、健康检查 `13133`、自身指标 `8889`
  （Prometheus `job="otel-collector"`）；宿主机仅回环发布 `127.0.0.1:4317/4318`。
- Jaeger all-in-one 开启 `COLLECTOR_OTLP_ENABLED=true`，OTLP 接收端为容器内 `4317`。

### 5.2 配置键

| 配置键 | 生产值 | 说明 |
| --- | --- | --- |
| `trace.endpoint` | `otel-collector:4317` | 支持 `host:port` 或带 scheme 的 URL；留空则 tracing 降级为 no-op |
| `trace.insecure` | `true` | 明文 gRPC，适用内网 collector；未配置默认 `true` |
| `trace.env` | `prod` | 写入资源属性 `deployment.environment` |
| `trace.sample_ratio` | `0.1` | 父级采样 + TraceIDRatio，生产 10% |

环境变量覆盖：`PLANSTAR_TRACE_ENDPOINT`、`PLANSTAR_TRACE_INSECURE`（`true/false/1/0`）。

### 5.3 自动埋点范围

- HTTP：`planstar/internal/observability.Middleware` 为业务与 9108 的每个请求创建
  server span（`METHOD route`），记录 `http.method`/`http.target`/`http.route`/
  `http.status_code`。
- MySQL：`otelsql.Register("mysql", ...)`，9 个库共用一个带埋点的驱动，span 属性
  含 `db.system=mysql`；driver 注册失败时回退原生 `mysql`，不影响业务。
- Redis：`redisotel.InstrumentTracing(rdb)`，命令级 span 沿用全局 TracerProvider。
- 日志：`observability.NewTraceHandler` 给每条日志补 `trace_id`/`span_id`。

### 5.4 collector 故障行为

- `otlptracegrpc.New` 不建立长连接，collector 未启动不会阻塞应用启动。
- 运行期 collector 不可用时，SDK 批量导出失败并重试/丢弃，业务请求不受影响；
  表现为 Jaeger 查不到新 trace、collector 自身 `up` 下降或日志出现导出错误。
- 处置：`docker compose ps otel-collector jaeger`；`docker compose logs --tail=200 otel-collector`；
  恢复后无需重启应用（SDK 自动重连）。
- 采样率为 0.1 时，未采样请求仍会生成 `X-Trace-Id` 与日志 `trace_id`，但 Jaeger
  查不到 span。

## 6. 自保护机制

| 机制 | 行为 |
| --- | --- |
| 结果缓存 | `/readyz` 结果缓存 `2s`，缓存期内不重复探测依赖 |
| singleflight | 并发请求同时到期时只允许一次真实刷新，避免探针风暴 |
| 失败退避 | 单项连续失败按 `2s → 4s → 8s → … → 30s` 退避；退避期内直接返回上次失败结果 |
| 单项超时 | 默认 `2s`；超时记为 `fail`，错误信息为受控文案 |
| 总超时 | 默认 `3s`；超时后未完成的检查统一记为 `fail` |
| 错误脱敏 | DSN/URL 用户名口令替换为 `***`，错误截断到 200 字符 |
| 运维鉴权 | `/metrics` 与 `/admin*` 强制 Bearer；生产未配置令牌启动失败 |
| admin 超时 | 9108 读头 `5s`、读/写 `10s`、空闲 `60s`；运营写接口请求体上限 16 KiB |
| metrics 并发预算 | `ops.metrics_max_inflight`（默认 8）与 `ops.metrics_timeout_seconds`（默认 5）在进程内生效：信号量占满时直接返回 `503` 并累加 `planstar_ops_metrics_scrape_rejected_total`，不排队 |
| 优雅停机 | `BeforeStop` 先关闭就绪闸门（`/readyz` 立即 503），再停止业务 HTTP/admin/MQ；最长等待 15s |
| 请求保护 | 业务端口请求体默认 1 MiB（`/v1/upload` 放宽到 20 MiB），单请求超时默认 15s |

## 7. 压测与排障

### 7.1 端点压测

压测使用 `httptest` + 真实 `NewProbeHandlers`/`OpsAuth`，不依赖外部 MySQL/Redis：

```bash
cd planstar
go test ./internal/server -run TestOpsEndpointsLoad -count=1 -v
```

仓库根提供等价包装：

```bash
bash scripts/ops-loadtest.sh
powershell -ExecutionPolicy Bypass -File scripts\ops-loadtest.ps1
```

输出示例（数字随机器变化）：

```text
OPS LOAD TEST healthz qps=... p50=...us p95=...us p99=...us errors=0 error_rate=0.00%
OPS LOAD TEST readyz  qps=... p50=...us p95=...us p99=...us errors=0 error_rate=0.00%
OPS LOAD TEST qps=... p50=...us p95=...us p99=...us errors=0 error_rate=0.00%
```

断言：错误率为 0；`/healthz` P99 `< 50ms`；`/readyz` P99 `< 200ms`。

### 7.2 本地烟雾测试

```bash
# 仓库根
sh scripts/smoke-local.sh
PLANSTAR_OPS_TOKEN=<token> sh scripts/smoke-local.sh

# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1
$env:PLANSTAR_OPS_TOKEN = '<token>'
powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1
```

可选环境变量：`PS_API_BASE`（默认 `http://127.0.0.1:8098`）、
`PS_ADMIN_BASE`（默认 `http://127.0.0.1:9108`）、`PLANSTAR_OPS_TOKEN`。

脚本覆盖：`/healthz` 200 且 `status=ok`；`/readyz` 200/503 且包含
`checks`/`ready`，503 时打印失败检查名；`/metrics` 无令牌 401、带令牌 200 且包含
`planstar_build_info`；目录接口 200 JSON；发码接口非 5xx。令牌为空时跳过
`/metrics` 两项并给出提示。

### 7.3 排障决策树（以 `checks[]` 为准）

```text
/healthz 非 200 或连接失败
  → 进程未启动/端口未发布：docker compose ps planstar-api、logs、127.0.0.1:9108 映射

/readyz 503（ready=false）
  ├─ checks 为空 → 启动闸门未打开（等 application started）或已进入 stopping
  ├─ mysql=fail  → 第 8.1 节（磁盘/连接数/慢查询/锁）
  ├─ redis=fail  → 第 8.2 节（密码/内存/AOF）
  └─ 其他检查 fail 但 mysql/redis ok
       → 总超时导致未完成（看 error="probe timed out"），检查依赖延迟与 ops.probe_total_timeout_seconds

/readyz 200 但 status=degraded
  ├─ elasticsearch=fail → 第 8.4 节（可选依赖，业务回退 MySQL）
  ├─ rabbitmq=fail      → 第 8.3 节（可选依赖，通知可能延迟）
  └─ consul=fail        → 第 8.5 节（可选依赖，注册失败影响新实例上线）

/readyz 正常但 Prometheus up{job="planstar-api"}=0
  → 抓取鉴权/网络：核对 ops_token、401/403 指标、job target、容器 DNS

/metrics 401 → Bearer 缺失或令牌不一致（轮换后未同步 ops_token）
/metrics 403 → 来源不在 PLANSTAR_OPS_ALLOWED_CIDRS（注意取 TCP 对端 IP）

延迟/错误率高
  → Grafana RED 行：按 route 看 P95 与 5xx；再查慢 SQL、ES、Redis、连接池 wait 指标
```

### 7.4 常用即时命令

```bash
# 结构化探针
curl -fsS http://127.0.0.1:9108/healthz
curl -sS http://127.0.0.1:9108/readyz | python3 -m json.tool

# 带鉴权抓指标
curl -fsS -H "Authorization: Bearer $PLANSTAR_OPS_TOKEN" \
  http://127.0.0.1:9108/metrics | grep -E 'planstar_(build_info|ops_probe|db_connections)'

# 审计日志
docker compose logs --tail=200 planstar-api | grep 'ops audit'
```

## 8. 常见故障处置

> 告警规则注解中提到的「docs/operations.md 5.x」即本节各小节（历史编号）。

### 8.1 MySQL 不可用（critical）

- **表现**：启动阶段 `NewData` 失败、进程退出；运行中业务请求 `500 INTERNAL`，
  `/readyz` 503 且 `checks[]` 中 `mysql.status=fail`。
- **处置**：
  1. `docker compose ps mysql`、`docker compose logs --tail=200 mysql`。
  2. 检查磁盘、连接数、`max_connections`、慢查询与锁等待。
  3. 恢复后连接池自动重连；`/readyz` 转 200 即恢复。
  4. 若数据损坏，按 [deployment.md](deployment.md) 第 4 节恢复备份；恢复前停止
     `planstar-api` 写入。

### 8.2 Redis 不可用（critical）

- **表现**：启动 `Ping` 失败则进程退出；运行中 `/readyz` 503 且
  `redis.status=fail`。
- **影响**：
  - 会话读取回源 `login_session`，已登录用户通常不受影响；
  - 图形验证码无法写入/消费，`/v1/user/captcha` 与需要图形码的登录会失败（失败关闭）；
  - 发码冷却与登录失败计数在进程内，不受 Redis 影响。
- **处置**：`redis-cli -a <password> ping`、内存与 AOF 磁盘；恢复后无需重启应用。
  Redis 数据丢失只会让用户重新登录。

### 8.3 RabbitMQ 不可用（optional，不触发 503）

- **表现**：`/readyz` 返回 `200 + status=degraded`，`rabbitmq.status=fail`；未配置
  地址时为 `skipped`。应用启动时消费者每 5s 重试连接。
- **影响**：发布失败只记录 `publish ... failed`，下单/注册主流程仍成功，但站内通知
  可能缺失；超过 5 次重试的消息进入 `planstar.api.events.dlq`。
- **处置**：
  1. `docker compose logs rabbitmq`，检查磁盘、内存告警与连接数。
  2. 管理台 `http://<host>:15672` 查看 `planstar.api.events` 与 `.dlq` 深度。
  3. DLQ 有消息时先确认业务影响，再人工重放或归档；不要直接清空。

### 8.4 Elasticsearch 不可用（optional，不触发 503）

- **表现**：启动 `ClusterHealth` 失败会记录并继续；运行中 `/readyz` 为
  `200 + status=degraded`，`elasticsearch.status=fail`；热门榜回退 MySQL。
- **影响**：`GET /v1/activities/hot` 无过滤条件时 ES 查询失败会记录
  `elasticsearch hot query failed, fallback to mysql` 并走 MySQL 排序；
  `planstar_catalog_es_fallback_total` 增长。同步任务失败只记日志。
- **处置**：
  1. `curl -fsS http://<host>:9200/_cluster/health?pretty`，确认 status 非 red、磁盘水位、JVM 堆。
  2. Linux 宿主机确认 `vm.max_map_count >= 262144`。
  3. 恢复后无需手工重建索引：服务按 `sync_interval_seconds` 周期同步；如需立即刷新，
     重启 `planstar-api` 会先 `EnsureIndex` 再同步。

### 8.5 Consul 不可用（optional 探针，但注册失败会阻止启动）

- **表现**：`registry.consul.address` 已配置且启动时不可达，Kratos registrar 注册
  失败，`app.Run` 退出；运行中 `/readyz` 可返回 `degraded`（`consul.status=fail`）。
- **影响**：不影响已运行实例的业务请求，但新实例无法上线、服务发现失效。
- **处置**：恢复 Consul 后重启 `planstar-api`；临时可清空
  `registry.consul.address` 跳过注册（需同步调整调用方发现方式）。

### 8.6 503 / 504 / 429

| 状态 | 含义 | 首查 |
| --- | --- | --- |
| `503` | `/readyz` 关键依赖（MySQL/Redis）失败，或启动/停机闸门关闭 | `checks[]`、`application started/stopping` 日志 |
| `504 TIMEOUT` | 单请求超过 `server.http.request_timeout_seconds`（默认 15s） | 慢 SQL、ES 查询、下游锁等待 |
| `429 RATE_LIMITED` | 发码冷却/IP 限频 | `send code rate limited` 日志中的 `mobile`/`ip`/`dimension` |
| `413 PAYLOAD_TOO_LARGE` | 请求体超过 1 MiB | 客户端是否上传大 JSON；上传类路径需走 `/v1/upload` 前缀 |

## 9. 日志与 traceId 检索

### 9.1 日志

- 应用日志输出到容器 stdout（`slog` 文本），compose 限制 `max-size=20m`、`max-file=5`。
- 关键字段：`service.id`、`service.name`、`service.version`、`trace_id`、`span_id`、
  `mobile`（已脱敏）、`path`、`err`。
- 访问日志：每个请求结束输出 `http request`，包含 `method`、`route`（归一化模板，
  如 `/v1/activities/:id`）、`status`、`duration_ms`、`remote_addr`。`remote_addr` 取
  连接对端地址，不信任 `X-Forwarded-For`。
- 生命周期日志：`application starting` → `application started` → `application stopping`
  → `application stopped`。
- Promtail 采集 Docker 日志，Loki 标签包含 `container` 与 `service`
  （`deploy/promtail/promtail-config.yml`）。

Loki 查询：

```logql
{service="planstar-api"} |= "trace_id"
{service="planstar-api"} |= "http request"
{service="planstar-api"} |= "ops audit"
{service="planstar-api"} |= "ops auth disabled"
{service="planstar-api"} |= "panic recovered"
{service="planstar-api"} |= "send code rate limited"
{service="planstar-api"} |= "application stopping"
{service="planstar-api"} |= "tracing enabled"
```

### 9.2 traceId

1. 客户端响应头 `X-Trace-Id` 或错误体 `traceId` 拿到十六进制串。
2. Jaeger UI（`http://<host>:16686`）选择服务 `planstar.api`，按 Trace ID 搜索；
   环境属性为 `deployment.environment`（生产 `prod`）。
3. 用同一 ID 在 Loki 搜索 `{service="planstar-api"} |= "<traceId>"`。
4. 采样率为 `trace.sample_ratio`（生产 0.1）；未采样的请求仍有 `X-Trace-Id` 与日志
   `trace_id`，但 Jaeger 查不到 span。

### 9.3 优雅停机

- 收到 `SIGTERM`/`SIGINT` 后，`kratos.App` 依次执行 `BeforeStop`（关闭就绪闸门）、
  注销 Consul 实例、并发停止业务 HTTP / admin HTTP / MQ 消费者，最长等待 15s
  （`StopTimeout`），最后执行 `AfterStop`。
- 注册中心注销超时 5s（`RegistrarTimeout`），超时不会阻塞进程退出。
- 停机顺序在 `app.Run` 返回后继续：MQ publisher → MySQL/Redis/ES 客户端 → tracing。
- 排查「停机卡住」时先看是否停在 `application stopping` 之后；`/readyz` 在停机阶段
  应为 503，这是预期行为。

## 10. 日常巡检清单

- [ ] `docker compose ps`：所有服务 `healthy`/`running`，`planstar-api` 无反复重启。
- [ ] `curl -fsS 127.0.0.1:9108/healthz` 返回 `status=ok`；`/readyz` 返回 200 且
      `checks[]` 无 `fail`（或仅 optional 项 `degraded`）。
- [ ] `/metrics` 带 Bearer 可抓取，`planstar_build_info` 的 `version` 与发布记录一致。
- [ ] Prometheus Targets：`planstar-api`、`mysql`、`redis`、`consul`、`otel-collector` 全部 UP。
- [ ] `planstar_ops_auth_failed_total` 无持续增长；`deploy/prometheus/ops_token` 与
      `.env` 中 `PLANSTAR_OPS_TOKEN` 一致，权限 `0600`。
- [ ] 5xx 比例、P95 延迟、429 数量、在途请求、`planstar_db_wait_count_total` 在基线内。
- [ ] MySQL 磁盘、连接数、慢查询、备份状态；9 个 `planstar_*` 库存在。
- [ ] Redis 内存、evicted_keys、AOF 磁盘；无持续拒绝连接。
- [ ] RabbitMQ 队列深度、消费者数、DLQ 是否为空。
- [ ] Elasticsearch 集群 health、JVM 堆、磁盘水位；`planstar-activities` 文档数与上架活动数接近。
- [ ] Jaeger 能查到 `planstar.api` 新 trace；`otel-collector` 无持续导出错误。
- [ ] Alertmanager（`9093`）无静默过期；SMTP 配置变更后 `amtool check-config` 通过。
- [ ] 最近一次 MySQL 备份成功，抽查可恢复；备份文件保留周期符合要求。
- [ ] Traefik 证书有效期、80/443 转发、Grafana/Loki 面板可访问。
- [ ] 日志错误扫描：`panic recovered`、`request failed`、`ops audit` 中的异常结果、
      `publish order.created failed`。
- [ ] 确认线上镜像 tag 与发布记录一致，回滚 tag 可用。
