> 适用版本：planstar 单体 + admin 运营后台 · 最后核对：2026-11-07 · 维护：后端组

# Planstar 后端文档

本仓库实际运行两个服务：

- [`planstar_backend/`](../planstar) 业务 Go 单体（Kratos v3）：业务 HTTP `8098`，运维端口 `9108`（`/metrics`、`/healthz`、`/readyz`）。
- [`admin/`](../admin) 运营后台 Go 模块 + `admin/console` 控制台：HTTP `9100`、gRPC `9101`、运维 `9110`，通过 `https://admin.planstar.top` 暴露。

仓库根目录的 `openapi.yaml`、根 `go.mod`、根 `docker-compose.yml` 等仍属于历史 kratos-shop 模板遗留，**不参与构建、不参与部署、不是架构事实**。CI 构建 `planstar_backend/` 与 `admin/`。

## 阅读路径

| 路径 | 入口文档 | 适合场景 |
| --- | --- | --- |
| 架构 | [architecture.md](architecture.md) | 了解分层、依赖方向、请求生命周期、MySQL/Redis/ES/RabbitMQ 与降级策略 |
| 接口 | [api/API_CONTRACT.md](api/API_CONTRACT.md) | 对接 Flutter 客户端，确认方法、路径、鉴权、字段、错误码、金额与分页 |
| 部署 | [deployment.md](deployment.md) | 本地一键启动、生产部署、镜像发布、回滚、备份恢复、启动排障 |
| 运维 | [operations.md](operations.md) | 健康检查语义、指标、告警阈值、日志/traceId 检索、故障处置、巡检 |
| 运营后台 | [admin-integration.md](admin-integration.md) | admin 部署整合、17 项风险登记、上线检查单、回滚与外部服务启用 |

## 有效文档索引

| 文档 | 内容 | 状态 |
| --- | --- | --- |
| [product/POSITIONING.md](product/POSITIONING.md) | 产品定位、品牌表达、用户分层、业务闭环与品类边界的唯一事实源 | 有效 |
| [architecture.md](architecture.md) | 当前单体架构与技术事实 | 有效 |
| [api/API_CONTRACT.md](api/API_CONTRACT.md) | 客户端接口契约唯一事实源（57 条路由 / 56 个唯一端点，含企划域与创意投票；由 `planstar_backend/internal/server/routes_test.go` 机器校验） | 有效 |
| [api/ADMIN_CONTRACT.md](api/ADMIN_CONTRACT.md) | 运营后台契约：9108 管理端口举报处理与运营写、以及 `admin.planstar.top` 的 `/v1/admins/*`、`/v1/planstar/*` | 有效 |
| [admin-integration.md](admin-integration.md) | admin 与 planstar 的部署整合、风险、上线、回滚与外部服务 | 有效 |
| [aliyun-sms-oss.md](aliyun-sms-oss.md) | 阿里云号码认证短信（Dypnsapi）开通、RAM 最小权限、签名模板、错误码与 OSS/CDN 接入 | 有效 |
| [deployment.md](deployment.md) | 部署与回滚手册（含 4C8G/2M 低配档位） | 有效 |
| [operations.md](operations.md) | 运维与故障处置手册 | 有效 |
| [sql/README.md](sql/README.md) | 9 个基线 `planstar_*` 库建库脚本与演示数据说明（另有 `shop`/`admin` 增量脚本，运行时共 11 库） | 有效（数据基线） |
| [ADR/README.md](ADR/README.md) | 架构决策台账索引（ADR-0001 模块化单体 / ADR-0002 目录结构） | 有效（决策台账） |
| [local-deploy-verification.md](local-deploy-verification.md) | 目录改名后的 Docker 配置调整与本地 local deploy 验证记录（2026-11-19，PASS） | 有效 |
| [OPEN_ITEMS.md](OPEN_ITEMS.md) | **治理台账**：A 架构阶段 / B 功能 ADR / C 功能缺口 / D 测试工程 / E 文档归档共 33 项，逐项附证据与验收方式；**2026-11-19 已全部闭环**（顶部为闭环状态，后文保留原始盘点证据） | 有效（治理台账） |
| [PRODUCTION_AUDIT.md](PRODUCTION_AUDIT.md) | **商用生产级审计结论**（2026-11-19）：16 项阻断 / 17 项高风险中风险，含 3 项本轮引入的 RBAC 提权与 4 项目录改名遗留；**本轮未修复、未提交推送** | 有效（审计报告） |
| [FOLLOWUP_WORK.md](FOLLOWUP_WORK.md) | **后续工作台账**：从「闭环」中拆出的 26 项仍需代码/配置/验证动作的内容（TODO / DEBT / VERIFY / ENV / TRIGGER），含触发条件、证据与验收方式 | 有效（待办台账） |
| [E2E_VERIFICATION.md](E2E_VERIFICATION.md) | 本地 Docker 端到端验证：8098 全 57 条、9108 运维/RBAC、admin 前后端、admin-web 静态资源；0 个 5xx，web 端受环境限制未运行 | 有效（验证报告） |

## 归档与研究材料（**不得作为实现或排障依据**）

| 目录 | 内容 |
|---|---|
| [`ADR/`](ADR/README.md) | 正式架构决策台账 |
| [`research/`](research/go-wind-cms-split-proposal.md) | go-wind-cms 拆分设计推演（非实战复盘，收益数字未经测量；见 `ADR-0001` §1.1） |
| [`notes/`](notes/PLANSTAR_DEEP_STUDY.md) | 全仓学习报告与上游 kratos-layout / kratos-shop 模板研究笔记 |

> **2026-11-19 治理核查**：`docs/legacy/` 原为空目录，旧索引列出的
> `architecture-microservices.md` 等 6 个文件均未随本仓库检出；死链已清理，
> 空目录已删除。历史微服务方案（gRPC / Consul / `shop_*` / PostgreSQL）已被
> `architecture.md` 与 [ADR-0001](ADR/ADR-0001-modular-monolith-governance.md) 否决；
> 上游模板对照请查 `.archive/` 只读快照（已被 `.gitignore` 排除）。

## 文档维护约定

- 路由、请求字段、响应字段或错误码发生变化时，必须同步更新 `docs/api/API_CONTRACT.md`。
- 每份有效文档顶部保留「适用版本 / 最后核对 / 维护」元信息；核对日期随代码基线更新。
- 新文档只能补充到本索引；失真文档先标注 `Deprecated`，再迁入 `legacy/`。

## 本地验收记录（2026-11-07）

环境：Windows + Docker Desktop，`planstar_backend/docker-compose.yml` 全栈（MySQL 8.4 / Redis 7 / RabbitMQ / Elasticsearch 7.17 / Consul / Jaeger / Prometheus / Grafana / Traefik）。

| 检查项 | 结果 |
|---|---|
| `docker compose up -d --build`（清空卷后从零启动） | 通过，15 个容器全部 Healthy/Started |
| `GET :9108/healthz`、`/readyz` | 200 / 200 |
| 契约测试 `go test ./internal/server -run TestRoutesMatchContract` | 通过（23 个唯一端点） |
| `gofmt -l .`、`go build ./...`、`go vet ./...`、`go test ./...` | 全部通过 |
| `flutter analyze lib test packages/infras/core/lib`、`flutter test` | 无问题 / 153 项通过 |
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
