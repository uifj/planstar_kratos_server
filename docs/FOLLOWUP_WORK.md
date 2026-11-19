# 后续工作与未完成项台账（FOLLOWUP WORK）

> 盘点日期：2026-11-19 · 基线提交：`9ad1e8f`（本批次 3 笔提交：`fc1463a` / `684810f` / `9ad1e8f`，工作区干净）
> 与 `docs/OPEN_ITEMS.md` 的关系：那份台账的 33 项已「闭环」，但**闭环不等于全部实现**——其中含
> ①已实施并验证、②经 ADR 决策主动不做、③环境限制由 CI 覆盖、④明确标注为预留扩展点四类。
> 本文只登记 **②/③/④ 中仍需要后续代码、配置或验证动作** 的部分，以及新发现的口径偏差。

## 0. 分类说明

| 标记 | 含义 |
|---|---|
| `TODO` | 明确还没做，需要后续批次实施 |
| `DEBT` | 已决策接受的有时间边界的技术债，到期必须处理 |
| `VERIFY` | 代码已改但只做过静态/本地验证，缺端到端或真实环境验证 |
| `ENV` | 本机环境限制，代码无缺陷，但本地门禁无法覆盖 |
| `TRIGGER` | 满足触发条件才启动，当前不做（见 `ADR-0006`） |

---

## 1. 仍待实施的代码工作

| ID | 状态 | 项 | 客观证据 | 影响 / 建议动作 | 验收 |
|---|---|---|---|---|---|
| **F1** | `TRIGGER` | **7 个业务域物理分包** | `internal/biz/` 顶层仍有 **31** 个 `.go`、子目录只有 `shared`；`internal/data/` 22 个 + `shared`；`internal/service/` 17 个 + `events` | `ADR-0006` 决策阶段性停止拆包。触发条件（任一）：单包 > 60 个 `.go`、两个以上团队并行高频改同一包、或真要拆多进程；届时按 `ADR-0002` §3 映射表**逐域一次提交** | 触发后按域可独立 `go test`，57 条路由集合不变 |
| **F2** | `TRIGGER` | **`service/{api,admin}` 按端拆分** | `internal/service/` 无 `api` / `admin` 子目录；`s.fail`、`s.requireUser`、`s.authCfg`、`s.limit` 被全部 handler 共享 | `ADR-0006` 决策暂不拆；先抽公共 handler 基类再按端拆。`service/events` 已验证可行路径 | 两套错误映射（`service.fail` 409 / admin 400）语义不变 |
| **F3** | `TODO` | **data 层错误映射未收敛** | `internal/data/*.go` 中 `sql.ErrNoRows` 仍有 **47 处**；`data/shared/errors.go` 已提供 `MapQueryError` / `IsDuplicateKey` / `MapRedisError` / `IsNoRows`，目前仅 `data/order.go` 的 `isDuplicateKey` 委托了 shared | `ADR-0005` 决策分批收敛；后续按 repo 逐个替换，保留 1062「幂等命中而非错误」语义 | `grep -c 'sql.ErrNoRows' internal/data/*.go` 单调下降；全量测试通过 |
| **F4** | `TODO` | **MQ 多实例去重** | `internal/platform/mq/dedup.go` 是**单实例内存** TTL 表（`sync.Mutex` + `map`）；`ADR-0004` 明确「多实例部署时同一事件可能在不同实例各处理一次」 | 若生产横向扩容 `planstar-api`，需改为 Redis `SETNX event:{id}` 或 MySQL 去重表 | 起两个消费者实例投递同一 `ev.ID`，站内通知只落一条 |
| **F5** | `TODO` | **MQ 重试队列化** | `internal/platform/mq/consumer.go` 的 `retryOrDLQ` 仍在消费者内 `time.After` 指数退避（1/2/4/8/16s），串行 `Run` 下单条退避会拖住整队列 | `ADR-0004` 明确列为后续批次：改 TTL 延迟队列 + 独立重试 exchange/queue 拓扑 | 单条消息退避期间，队列其它消息仍能正常消费 |
| **F6** | `TODO` | **`catalog.activity.*` 发布点** | 全仓 `NewEvent(...EventCatalog...)` / `Publish(...EventCatalog...)` 命中 **0**；只有 `biz/shared/event.go` 常量与 `mq/server.go` 订阅；目录读模型靠 `RunSync` 全量重建兜底 | `ADR-0004` 定位为预留扩展点。真正出现目录写入口（运营后台/数据同步）时补齐发布者，否则删除空链路 | 新增发布点后有消费端集成测试；或删掉 routingKey 与消费分支 |
| **F7** | `TODO` | **`dict_enum` 与硬编码白名单同步巡检** | 运行时读取 `dict_enum` **0 命中**（`ADR-0005` 决策：字典表只作数据基线）；`biz/project.go` / `payment.go` / `project_idea.go` 为硬编码白名单，两者可能漂移 | 补一个离线巡检脚本（`docs/sql/` 或 `planstar_backend/scripts/`），比对字典表种子与代码常量，纳入 CI 报告项 | 巡检脚本能报出不一致项；CI 有该 step |
| **F8** | `TODO` | **`docs/sql/verify.sql` 校验口径仍过时** | `verify.sql:4,13,18` 仍写「86 业务表 / 89 全库表期望」；`00_all.sql` 实含 **106** 个 `CREATE TABLE`（含企划域增量）；`docs/sql/README.md` 已改为「以 verify.sql 实跑为准」，但脚本本身没更新 | 把 `verify.sql` 改为不写死总数：分组计数 + 必存在表清单；或按 `00_all.sql` + `planstar_backend/sql/15..20_*.sql` 的实际产物重设期望 | 全新 MySQL 初始化后 `verify.sql` 全 PASS，且能发现缺表 |
| **F9** | `TODO` | **核心事务集成测试不足** | 新增 `data/schema_integration_test.go` 只校验 9 基线库与 5 张关键表**结构**；既有 `ops_integration_test.go` 只覆盖跨库退款审核；`go test -tags integration ./...` 在无 `PLANSTAR_TEST_MYSQL_DSN` 时自动 skip | 补下单幂等（重复单号/报价）、签到并发、钱包扣减、创意投票频控等真实事务用例 | 有 MySQL 的 CI job 中集成测试实际执行且通过（非 skip） |
| **F10** | `TODO` | **6 个 usecase 单测只覆盖边界** | 新增 `biz/usecase_smoke_test.go` 用 stub 覆盖空参数、分页归一、超限量等**校验前置路径**，未覆盖登录全链路、报价权威计算、下单幂等、消息通知等核心算法 | 按 `ADR-0002` 域分包时，每域补 2~3 个表驱动核心流程用例 | `go test ./internal/biz -cover` 明显提升且核心分支有断言 |
| **F11** | `TODO` | **`/v1/upload` 端点与 OSS 消费者** | `internal/service/*.go` 中 `/v1/upload` 命中 **0**；`cmd/planstar/wiring.go` 已构造 `NewObjectStorage` 并注入 `Service.objects`，但该字段当前无调用方；`server/http.go` 的 upload 常量与放宽分支已在治理批次删除 | OSS 目前是「已接线但未使用」。要么实现端点（需同步契约条数 57→58 与客户端），要么在契约里明确标注「对象存储为预留能力」 | 二选一并同步 `API_CONTRACT.md` §1 条数与路由清单断言 |
| **F12** | `TODO` | **`payParams` / 组队支持端点** | `docs/api/API_CONTRACT.md` 已标注 `payParams` 为占位 `{}`、`support_team` / `support_team_member` 表无对应端点（治理批次只做了标注） | 接支付 SDK / 组队业务时补实现；否则保留「预留、未实现」标注 | 契约标注与代码一致；新增端点时走路由清单断言 |

---

## 2. 决策性技术债（有时间边界）

| ID | 状态 | 项 | 客观证据 | 到期动作 |
|---|---|---|---|---|
| **F13** | `DEBT` | **ICP 备案期 Cookie 明文豁免** | `configs/config.lite.yaml:51` `cookie_secure: false`、`:56` `allow_insecure_cookie: true`；`ADR-0003` 明确「备案完成切回 HTTPS 后必须删除开关并改回 true」 | 备案完成 → 删除 `allow_insecure_cookie`（含 `conf` 兼容代码与 env 覆盖）、`cookie_secure: true`、更新 `ADR-0003` 状态；豁免期内必须配合 `PLANSTAR_OPS_ALLOWED_CIDRS` 与 Traefik 访问限制 |
| **F14** | `DEBT` | **链路追踪端到端未启用** | `configs/config.prod.yaml` 的 `trace.endpoint: ""`；`deploy/prometheus/prometheus.yml` 的 otel-collector job 与 Grafana Loki 数据源已移除；`docker-compose.yml:224` 虽定义 `jaeger`，但 `:359` 注明属于 `full` profile、默认不启动（`docker compose config --services` 无 jaeger）；`otel-collector` 未编排 | 按 `deploy/{loki,promtail,otel}/README.md` 的启用步骤独立变更，恢复 endpoint / job / 数据源并验证镜像可拉取；在此之前 `docs/operations.md` 的 Loki/OTel/Jaeger 排障与巡检条目属于**目标态**，应跳过 |
| **F15** | `DEBT` | **事务性 Outbox 明确不做** | `ADR-0004` 决策：MQ 只承载站内通知与 ES 增量，发布失败只 Warn + `planstar_mq_publish_failed_total`；订单一致性由 MySQL 单事务 + 唯一键保证 | 若未来 MQ 开始承载资金/订单一致性事件，**必须重新评审**并引入 Outbox；当前不要误认为「已具备 exactly-once」 |
| **F16** | `DEBT` | **`dict_enum` 双份定义** | 字典表有建表 + 143 条种子，但运行时不读；代码白名单是唯一生效源 | 见 F7 巡检脚本；长期若决定废弃字典表，需另立 ADR |

---

## 3. 仅静态/本地验证，缺真实环境验证

| ID | 状态 | 项 | 已做的验证 | 缺什么 |
|---|---|---|---|---|
| **F17** | `VERIFY` | **DLQ 重放工具** | `scripts/dlq-replay.sh` 已落盘；`promtool check rules` 20 rules SUCCESS；MQ 指标有单测 | 未在**真实 RabbitMQ**（含 management 插件）上跑过 `--dry-run` 与 `--execute`；启用 MQ 的生产档位需实测重放与幂等效果 |
| **F18** | `VERIFY` | **SMTP 邮件通道** | `notify/smtp.go` + `smtp_test.go` 用可注入的 fake `send` 覆盖 MIME/CRLF 注入/错误透传 | 未对真实 SMTP 服务器验证 `net/smtp.SendMail`、TLS/465、凭据注入与投递成功率 |
| **F19** | `VERIFY` | **生产部署档位** | 本地 `docker-compose.local.yml` 三容器（mysql/redis/api）`/healthz` 200、`/readyz` 200、业务接口 200、`/metrics` 401、smoke 退出 0（`docs/local-deploy-verification.md` §7）；四份 compose `config -q` 通过 | **未做** `prod + server` 三文件叠加的真实部署/发布演练；`release.yml` 的 deploy job、Traefik 证书、Alertmanager 通知、`deploy/preflight.sh` 均未在目标环境跑过 |
| **F20** | `VERIFY` | **前端两个 module 的 CI** | `planstar_frontend/admin` 本地 `go build ./...` 通过；CI YAML 解析出 8 个 job | `admin-go` / `admin-console` / `planstar_web` 的 CI job 未在 GitHub Actions 实际执行；`frontend/admin/console` 的 `npm test`/build 未本地验证 |
| **F21** | `VERIFY` | **集成测试实际执行** | `go test -tags integration ./...` 退出 0，但本机/无 DSN 环境下**自动 skip**；CI `quality` job 调用 `make test-integration` 同样无 DSN 时跳过 | 需要一个带 MySQL service 的 CI job 或本地 DSN，让 `schema_integration_test.go` 与 `ops_integration_test.go` 真正执行 |
| **F22** | `VERIFY` | **告警规则联动** | `promtool check rules` 语法通过；3 条 MQ 告警表达式已写 | 未用真实 Prometheus + 模拟指标验证触发/静默/通知链路（Alertmanager → 邮件/钉钉） |

---

## 4. 环境限制（非代码缺陷）

| ID | 状态 | 项 | 证据 | 处置 |
|---|---|---|---|---|
| **F23** | `ENV` | **`go test -race` 本机不可执行** | 实测 `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`，且本机 `gcc not found in %PATH%` | CI `go` job 的 `ci.yml:57` 在 ubuntu runner 上执行 `go test -race ./...`；本机如需覆盖需装 mingw/LLVM |
| **F24** | `ENV` | **Docker Hub 不可达** | `docker compose up -d --build` 拉 `moby/buildkit:buildx-stable-1` 超时；本地改用「内置 builder `docker build` + `up -d --no-build`」两段式 | CI/Linux 构建不受影响；本地复现请用 `docs/local-deploy-verification.md` §3 的命令，不要直接 `--build` |
| **F25** | `DEBT` | **工具链只做版本断言** | `go.mod` / `Dockerfile` / `ci.yml` 四处均为 Go `1.26.5`，`TestGoToolchainPinned` 校验一致；但 `go.mod` **无 `toolchain` 指令** | 若本机/构建机 Go 版本漂移导致生成器或 `http2.TrailerPrefix` 兼容问题（见 `docs/notes/CODEGEN_VERIFICATION.md`），再补 `toolchain` 或 `.tool-versions` |
| **F26** | `DEBT` | **golangci-lint 版本固定** | `.github/workflows/ci.yml:166-170` 固定 `v2.15.0`，`.golangci.yml` 为 v2 schema；本地 `0 issues.` | Go 版本升级时需同步评估 linter 版本与 SA* 规则，避免 CI 突然失败 |

---

## 5. 已完成、不再需要动作（避免重复登记）

- `frontend/`→`planstar_frontend/`、`planstar/`→`planstar_backend/` 改名与全量路径同步（`e514842`）。
- P0/P1 治理：错误码单源、路由清单断言、ADR 台账、契约口径修正、`main.go` 26 行 / `wiring.go` / `probes.go` 拆分、`biz/shared` 与 `data/shared` 共享内核。
- `internal/platform/` 归位（conf/observability/mq/ops/pkg），42 个 Go 文件 import 重写（`fc1463a`）。
- 严格治理门禁（阈值默认阻断、依赖方向、传递依赖、错误码、ADR 台账、契约怪癖、路由清单、工具链版本）、CI lint 必过、`quality` job。
- `ADR-0003` 生产配置可启动性、`ADR-0004` MQ 单实例幂等 + 指标 + 告警 + 重放工具、`ADR-0005` 错误映射内核 + dict_enum 决策、`ADR-0006` P2 收敛决策。
- 本地 Docker local deploy 两次 PASS（初始验证 + 治理批次复测）。
- 空目录清理、参考材料归档 `docs/research/`、研究笔记归档 `docs/notes/`。
- **明确否决、不再列入待办**：物理拆服务/双 BFF/gRPC 微服务化、全量 proto/buf 生成链、按域做一级目录（`ADR-0001`）、事务性 Outbox（F15 除外需重新评审的场景）。

---

## 6. 维护约定

1. 完成某项后把状态改为 `DONE` 并注明提交 hash；同步 `docs/OPEN_ITEMS.md` 与 `docs/ADR/README.md`。
2. `TRIGGER` 类项不得因为「看起来该做」而提前启动，必须记录触发条件是否被满足。
3. `VERIFY` 类项在真实环境验证通过前，**不得在文档中表述为「已上线可用」**。
4. 新增项必须给出可复核证据（路径:行号 / grep 计数 / 命令输出），不写主观评价。
5. 涉及对外契约（路由条数、Cookie、错误码、金额单位、分页参数）的整改，须标注「是否需要 Flutter 客户端协同」。
