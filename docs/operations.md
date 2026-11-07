> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组

# 运维手册（planstar 单体）

## 1. 健康检查语义

| 端点 | 端口 | 语义 | 返回 |
| --- | --- | --- | --- |
| `GET /healthz` | 9108 | **存活**：只表示进程与 HTTP server 正常，不检查任何外部依赖 | `200 {"status":"ok"}` |
| `GET /readyz` | 9108 | **就绪**：3 秒内汇总 MySQL（9 库逐个 `Ping`）、Redis（已配置时）、Elasticsearch（已配置时） | 成功 `200 {"status":"ready"}`；失败 `503 {"status":"unavailable"}` |
| `GET /metrics` | 9108 | Prometheus 指标 | `text/plain` |

- `/readyz` **不检查** RabbitMQ、Consul、Jaeger；这些依赖不可用不会让就绪探针失败。
- `/readyz` 失败时日志打印 `readiness check failed` 与具体依赖错误，先看该日志再排障。
- 业务端口 8098 与运维端口 9108 分离；生产只对负载均衡/内网开放 9108。

## 2. 关键指标

| 指标 | 标签 | 用途 |
| --- | --- | --- |
| `planstar_http_requests_total` | `method`、`route`、`status` | 请求量、错误率、429 限流 |
| `planstar_http_request_duration_seconds` | `method`、`route` | P50/P95/P99 延迟（bucket 到 5s，总超时 15s） |
| `planstar_http_requests_in_flight` | `method` | 并发在途请求，判断堆积 |
| `planstar_catalog_es_fallback_total` | — | 目录查询因 ES 不可用/失败回退 MySQL 的累计次数 |
| `planstar_biz_login_total` | `result`（success/failure/captcha） | 登录成功、失败与触发图形码的次数 |
| `planstar_biz_order_confirm_total` | `result`（success/failure） | 确认报价成功/失败 |
| `planstar_biz_order_created_total` | `result`（success/price_changed/sold_out/limit_exceeded/quote_expired/error） | 下单结果分布，直接定位业务失败原因 |
| `go_*`、`process_*` | — | goroutine、堆内存、GC、CPU、FD |

Prometheus 抓取配置：`deploy/prometheus/prometheus.yml`，`job="planstar-api"`，目标 `planstar-api:9108`，间隔 15s。另有 `mysql`（mysqld-exporter）、`redis`（redis-exporter）、`consul` 三个基础设施 job。

当前**没有** RabbitMQ / Elasticsearch exporter；ES 侧故障除 `/readyz` 与组件自带接口外，可通过 `planstar_catalog_es_fallback_total` 的增长速率判断降级频率。

## 3. 告警阈值建议

现有规则（`deploy/prometheus/rules/alerts.yml`）：

| 告警 | 表达式 | 阈值 |
| --- | --- | --- |
| `PlanstarServiceDown` | `up{job="planstar-api"} == 0` | 持续 2 分钟，critical |
| `PlanstarInfraDown` | `up{job=~"mysql|redis|consul"} == 0` | 持续 2 分钟，critical |

建议补充（按业务量校准后再启用）：

| 场景 | 建议表达式 | 建议阈值 |
| --- | --- | --- |
| 5xx 比例 | `sum(rate(planstar_http_requests_total{status=~"5.."}[5m])) / sum(rate(planstar_http_requests_total[5m]))` | `> 5%` 持续 10m，warning；`> 10%` critical |
| P95 延迟 | `histogram_quantile(0.95, sum by (le) (rate(planstar_http_request_duration_seconds_bucket[5m])))` | `> 2s` 持续 10m |
| 在途堆积 | `sum(planstar_http_requests_in_flight)` | `> 50` 持续 5m（按实例容量调整） |
| 429 突增 | `sum(rate(planstar_http_requests_total{status="429"}[5m]))` | `> 1/s` 持续 10m |
| 就绪探针失败 | 外部 blackbox/HTTP 探测 `/readyz` | 连续 3 次 503 |
| MySQL 连接使用率 | `mysql_global_status_threads_connected / mysql_global_variables_max_connections` | `> 80%` 持续 10m |
| Redis 内存 | `redis_memory_used_bytes`（当前未设 maxmemory，需结合宿主机内存） | 超过实例内存 80% |
| 备份新鲜度 | 备份任务成功时间戳/文件年龄 | 超过 26h 未成功 |

> 若要告警 RabbitMQ 队列积压或 ES 健康，需额外部署 `rabbitmq_exporter` / `elasticsearch_exporter`，当前 compose 未包含。

## 4. 日志与 traceId 检索

### 4.1 日志

- 应用日志输出到容器 stdout（`slog` 文本），compose 限制 `max-size=20m`、`max-file=5`。
- 关键字段：`service.id`、`service.name`、`service.version`、`trace_id`、`span_id`、`mobile`（已脱敏）、`path`、`err`。
- 访问日志：每个请求结束输出 `http request`，包含 `method`、`route`（归一化模板，如 `/v1/activities/{id}`）、`status`、`duration_ms`、`remote_addr`。`remote_addr` 取连接对端地址，不信任 `X-Forwarded-For`。
- 生命周期日志：`application starting` → `application started` → `application stopping` → `application stopped`，用于确认发布时是否优雅退出。
- Promtail 采集 Docker 日志，Loki 标签包含 `container` 与 `service`（`deploy/promtail/promtail-config.yml`）。

Loki 查询示例：

```logql
{service="planstar-api"} |= "trace_id"
{service="planstar-api"} |= "http request"
{service="planstar-api"} |= "readiness check failed"
{service="planstar-api"} |= "panic recovered"
{service="planstar-api"} |= "send code rate limited"
{service="planstar-api"} |= "application stopping"
```

### 4.2 traceId

1. 客户端响应头 `X-Trace-Id` 或错误体 `traceId` 拿到 32 位/16 位十六进制串。
2. Jaeger UI（`http://<host>:16686`）选择服务 `planstar.api`，按 Trace ID 搜索；环境标签为 `trace.env`（生产 `prod`）。
3. 用同一 ID 在 Loki 搜索 `{service="planstar-api"} |= "<traceId>"`，日志字段名为 `trace_id`。
4. 采样率为 `trace.sample_ratio`（生产 0.1），未被采样的请求仍会生成 `X-Trace-Id` 与日志 `trace_id`，但 Jaeger 查不到 span。

### 4.3 优雅停机

- 收到 `SIGTERM`/`SIGINT` 后，`kratos.App` 依次执行 `BeforeStop`、注销 Consul 实例、并发停止业务 HTTP / admin HTTP / MQ 消费者，最长等待 15s（`StopTimeout`），最后执行 `AfterStop`。
- 注册中心注销超时 5s（`RegistrarTimeout`），超时不会阻塞进程退出。
- `docker compose stop` / `kubectl delete pod` 默认发送 `SIGTERM`；若 15s 后仍未退出，再发送 `SIGKILL`。
- 停机顺序在 `app.Run` 返回后继续：MQ publisher → MySQL/Redis/ES 客户端 → tracing。排查「停机卡住」时先看是否停在 `application stopping` 之后。

## 5. 常见故障处置

### 5.1 MySQL 不可用

- **表现**：启动阶段 `NewData` 失败、进程退出；运行中业务请求 `500 INTERNAL`，`/readyz` 503。
- **处置**：
  1. `docker compose ps mysql`、`docker compose logs --tail=200 mysql`。
  2. 检查磁盘、连接数、`max_connections`、慢查询与锁等待。
  3. 恢复 MySQL 后连接池自动重连；`/readyz` 转 200 即恢复。
  4. 若数据损坏，按 [deployment.md](deployment.md) 第 4 节恢复备份；恢复前停止 `planstar-api` 写入。

### 5.2 Redis 不可用

- **表现**：已配置 Redis 时启动 `Ping` 失败，进程启动失败；运行中会话缓存失效。
- **影响**：
  - 会话读取回源 `login_session`，已登录用户通常不受影响；
  - 图形验证码无法写入/消费，`/v1/user/captcha` 与需要图形码的登录会失败（失败关闭）；
  - 发码冷却与登录失败计数在进程内，不受 Redis 影响。
- **处置**：检查 `redis-cli -a <password> ping`、内存与 AOF 磁盘；恢复后无需重启应用（会话自动回源）。Redis 数据丢失只会让用户重新登录。

### 5.3 RabbitMQ 不可用

- **表现**：`/readyz` **不报警**；应用启动时消费者每 5s 重试连接；日志 `rabbitmq disabled`（未配置）或连接错误。
- **影响**：发布失败只记录 `publish ... failed`，下单/注册主流程仍成功，但站内通知可能缺失；消费者恢复后继续消费，超过 5 次重试的消息进入 `planstar.api.events.dlq`。
- **处置**：
  1. `docker compose logs rabbitmq`，检查磁盘、内存告警与连接数。
  2. 管理台 `http://<host>:15672` 查看 `planstar.api.events` 与 `.dlq` 深度。
  3. DLQ 有消息时先确认业务影响，再人工重放或归档；不要直接清空。

### 5.4 Elasticsearch 不可用

- **表现**：已配置 ES 时启动 `ClusterHealth` 失败，进程启动失败；运行中 `/readyz` 503，热门榜回退 MySQL。
- **影响**：`GET /v1/activities/hot` 无过滤条件时 ES 查询失败会记录 `elasticsearch hot query failed, fallback to mysql` 并走 MySQL 排序；同步任务失败只记日志。
- **处置**：
  1. `curl -fsS http://<host>:9200/_cluster/health?pretty`，确认 status 非 red、磁盘水位、JVM 堆。
  2. Linux 宿主机确认 `vm.max_map_count >= 262144`。
  3. 恢复后无需手工重建索引：服务按 `sync_interval_seconds` 周期同步；如需立即刷新，重启 `planstar-api` 会先 `EnsureIndex` 再同步。

### 5.5 Consul 不可用

- **表现**：`registry.consul.address` 已配置且启动时不可达，Kratos registrar 注册失败，`app.Run` 退出；`/readyz` 不检查 Consul。
- **影响**：不影响已运行实例的业务请求，但新实例无法上线、服务发现失效。
- **处置**：恢复 Consul 后重启 `planstar-api`；临时可清空 `registry.consul.address` 跳过注册（需同步调整调用方发现方式）。

### 5.6 503 / 504 / 429

| 状态 | 含义 | 首查 |
| --- | --- | --- |
| `503` | 就绪探针依赖失败（MySQL/Redis/ES） | `/readyz` 日志、对应组件 |
| `504 TIMEOUT` | 单请求超过 `server.http.request_timeout_seconds`（默认 15s） | 慢 SQL、ES 查询、下游锁等待 |
| `429 RATE_LIMITED` | 发码冷却/IP 限频 | `send code rate limited` 日志中的 `mobile`/`ip`/`dimension` |
| `413 PAYLOAD_TOO_LARGE` | 请求体超过 1 MiB | 客户端是否上传大 JSON；上传类路径需走 `/v1/upload` 前缀 |

## 6. 日常巡检清单

- [ ] `docker compose ps`：所有服务 `healthy`/`running`，`planstar-api` 无反复重启。
- [ ] `curl -fsS :9108/healthz` 与 `/readyz` 均成功。
- [ ] Prometheus Targets 页面 `planstar-api`、`mysql`、`redis`、`consul` 全部 UP。
- [ ] 5xx 比例、P95 延迟、429 数量在基线内；无持续在途堆积。
- [ ] MySQL 磁盘、连接数、慢查询、主从/备份状态；9 个 `planstar_*` 库存在。
- [ ] Redis 内存、evicted_keys、AOF 磁盘；无持续拒绝连接。
- [ ] RabbitMQ 队列深度、消费者数、DLQ 是否为空。
- [ ] Elasticsearch 集群 health、JVM 堆、磁盘水位；`planstar-activities` 文档数与上架活动数接近。
- [ ] 最近一次 MySQL 备份成功，抽查可恢复；备份文件保留周期符合要求。
- [ ] Traefik 证书有效期、80/443 转发、Grafana/Jaeger/Loki 面板可访问。
- [ ] 日志错误扫描：`panic recovered`、`request failed`、`readiness check failed`、`publish order.created failed`。
- [ ] 确认线上镜像 tag 与发布记录一致，回滚 tag 可用。
