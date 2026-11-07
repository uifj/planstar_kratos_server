> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组

# Planstar 后端文档

本仓库**唯一实际运行的服务**是 [`planstar/`](../planstar) 下的 Go 单体（Kratos v3）。
业务 HTTP 监听 `8098`，运维端口监听 `9108`（`/metrics`、`/healthz`、`/readyz`）。

仓库根目录的 `admin/`、根 `openapi.yaml`、根 `go.mod`、根 `docker-compose.yml` 等属于历史 kratos-shop 模板遗留，**不参与构建、不参与部署、不是架构事实**。CI 只构建 `planstar/`。

## 四条阅读路径

| 路径 | 入口文档 | 适合场景 |
| --- | --- | --- |
| 架构 | [architecture.md](architecture.md) | 了解分层、依赖方向、请求生命周期、MySQL/Redis/ES/RabbitMQ 与降级策略 |
| 接口 | [api/API_CONTRACT.md](api/API_CONTRACT.md) | 对接 Flutter 客户端，确认方法、路径、鉴权、字段、错误码、金额与分页 |
| 部署 | [deployment.md](deployment.md) | 本地一键启动、生产部署、镜像发布、回滚、备份恢复、启动排障 |
| 运维 | [operations.md](operations.md) | 健康检查语义、指标、告警阈值、日志/traceId 检索、故障处置、巡检 |

## 有效文档索引

| 文档 | 内容 | 状态 |
| --- | --- | --- |
| [architecture.md](architecture.md) | 当前单体架构与技术事实 | 有效 |
| [api/API_CONTRACT.md](api/API_CONTRACT.md) | 客户端接口契约（23 个唯一端点 + `GET /v1/user/` 尾斜杠别名） | 有效 |
| [deployment.md](deployment.md) | 部署与回滚手册 | 有效 |
| [operations.md](operations.md) | 运维与故障处置手册 | 有效 |
| [sql/README.md](sql/README.md) | 9 个 `planstar_*` 库建库脚本与演示数据说明 | 有效（数据基线） |

## legacy 目录用途

[legacy/](legacy/README.md) 存放与当前代码不符的历史资料，仅作对照，**不得作为实现或排障依据**：

- [legacy/architecture-microservices.md](legacy/architecture-microservices.md)：已废弃的 kratos-shop 微服务拆分（gRPC、Consul、`shop_*`、PostgreSQL），顶部已标注 `Deprecated`。
- [legacy/configuration.md](legacy/configuration.md)、[legacy/database.md](legacy/database.md)、[legacy/development.md](legacy/development.md)、[legacy/evolution-plan.md](legacy/evolution-plan.md)、[legacy/feasibility-p1.md](legacy/feasibility-p1.md)：旧模板时期的配置、库表、开发与演进说明，已整体迁入本目录，**不得作为当前实现依据**。

## 文档维护约定

- 路由、请求字段、响应字段或错误码发生变化时，必须同步更新 `docs/api/API_CONTRACT.md`。
- 每份有效文档顶部保留「适用版本 / 最后核对 / 维护」元信息；核对日期随代码基线更新。
- 新文档只能补充到本索引；失真文档先标注 `Deprecated`，再迁入 `legacy/`。

## 本地验收记录（2026-11-07）

环境：Windows + Docker Desktop，`planstar/docker-compose.yml` 全栈（MySQL 8.4 / Redis 7 / RabbitMQ / Elasticsearch 7.17 / Consul / Jaeger / Prometheus / Grafana / Traefik）。

| 检查项 | 结果 |
|---|---|
| `docker compose up -d --build`（清空卷后从零启动） | 通过，15 个容器全部 Healthy/Started |
| `GET :9108/healthz`、`/readyz` | 200 / 200 |
| 契约测试 `go test ./internal/server -run TestRoutesMatchContract` | 通过（23 个唯一端点） |
| `gofmt -l .`、`go build ./...`、`go vet ./...`、`go test ./...` | 全部通过 |
| `flutter analyze lib test packages/infras/core/lib`、`flutter test` | 无问题 / 140 项通过 |
| 登录（发码 → 登录） | 200，`psSession` Cookie 含 `HttpOnly; Secure; SameSite=Lax; Max-Age=2592000` |
| 发码重复请求 | 第二次 429 `RATE_LIMITED` |
| 未登录访问订单 | 401 `UNAUTHORIZED` |
| 1.2 MB 请求体 | 413 `PAYLOAD_TOO_LARGE` |
| 服务端报价 | 200，返回 `quoteId/unitPrice/totalAmount/version/expiresAt`（单位：分） |
| 下单 + 同 `Idempotency-Key` 重放 | 同一订单号，无重复记录 |
| 篡改金额 | 409 `PRICE_CHANGED` |
| 数量 7 | 409 `LIMIT_EXCEEDED` |
| 库存置 1 后下单 2 张 | 409 `SOLD_OUT`，库存未变负 |
| `/metrics` 业务与降级指标 | `planstar_biz_order_confirm_total`、`planstar_catalog_es_fallback_total` 可采集 |
| ES 停机演练 | 停止 Elasticsearch 后 `/v1/activities`、`/v1/activities/hot` 均 200，`planstar_catalog_es_fallback_total` 由 0→1，MySQL 兜底生效 |
| Traefik / mysqld-exporter | 修复 :8080 entryPoint 冲突与 exporter v0.15 凭据方式后，15 个容器全部 Up，`8080/api/version`、`9104/metrics`（`mysql_up 1`）可用 |
| Kratos 生命周期对齐 | `StopTimeout=15s`、`RegistrarTimeout=5s`、四个启停钩子；`docker compose stop` 后日志依次出现 `application stopping` / `application stopped`，重启后 `/readyz` 200 |
| 配置启动校验 | `conf.Validate` 覆盖地址/端口/URL/超时顺序，`go test ./internal/conf` 通过；`PLANSTAR_*` 地址类环境变量可覆盖 |
| HTTP 访问日志 | 每个请求输出 `http request`，含 `method`/`route`/`status`/`duration_ms`/`remote_addr`/`trace_id` |
| MQ 事件订阅 | `rabbitmqctl list_bindings` 确认 `planstar.api.events` 绑定 4 个 routing key（含 `catalog.activity.*`） |
| 报价过期运行态演练（R1） | `confirm` 后对 `planstar:quote:<quoteId>` 执行 `PEXPIRE 1`，2s 后 `makeOrder` 返回 409 `QUOTE_EXPIRED`；Redis 键已删除，`planstar_biz_order_created_total{result="quote_expired"}` 0→1，`order_info` 无新增记录 |
| ES 全量重建压测（R2） | `PLANSTAR_ES_SYNC_INTERVAL_SECONDS=30` 下连续 3 分钟并发查询 `/v1/activities` 与 `/v1/activities/hot`：21,448 次请求 **0 次非 200**；期间 6 次全量重建、7 个时间戳索引依次原子切换别名并即时删除旧索引；0 次 `elasticsearch sync failed`、0 次 MySQL 兜底、0 个 5xx |

未覆盖/遗留：
1. 支付 SDK、真实锁座、IM 长连接不在本次范围。
2. `make` 未安装于当前 Windows 环境，`make check` 由分项命令逐一验证；Linux/macOS 与 CI 可直接执行。
3. 目录增量同步已接入 MQ 消费入口，但生产端（运营后台）不在本仓库，尚无事件生产者。
