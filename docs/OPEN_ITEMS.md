# 治理批次闭环状态（2026-11-19）

> **本台账 33 项已全部闭环**：其余章节保留为原始盘点记录与证据（不作删除），
> 各项的最终状态以本节为准。闭环 = 已实施并验证，或已形成 ADR 决策并写明理由。
> 基线提交 `e514842` / `b65b394`；本批次改动尚未提交；如需入库请按 Conventional Commits 单独提交。

| ID | 状态 | 处置与依据 |
|---|---|---|
| A1 | ✅ 已决策关闭 | `ADR-0006`：阶段性停止 7 域拆包，改为共享内核 + 命名分组 + 严格治理断言；列出重新拆包触发条件 |
| A2 | ✅ 已完成 | `internal/platform/` 归位（conf/observability/mq/ops/pkg），42 个 `.go` import 重写，`go build ./...` 通过 |
| A3 | ✅ 已决策关闭 | `ADR-0006`：`service/{api,admin}` 暂不拆（私有 handler 成员共享），`service/events` 已验证可行路径 |
| A4 | ✅ 已决策关闭 | `ADR-0006`：`biz` 根包兼容层是 P1 有意设计的稳定门面，保留 |
| A5 | ✅ 已决策关闭 | `ADR-0006`：按域 CI/指标改为文档化域清单（`ADR-0002` §3），单进程单镜像下按域拆 CI 无收益 |
| A6 | ✅ 已完成 | `planstar_backend/.golangci.yml`（v2.15.0，SA* 规则）、CI lint 去掉 `continue-on-error`、本地 `0 issues.` |
| B1 | ✅ 已完成 | `ADR-0003`：`auth.allow_insecure_cookie` 显式豁免、`config.lite.yaml` 改 `planstar_app`、`TestLiteConfigStartsWithProductionEnv` 通过；E5/E6 一并修复 |
| B2 | ✅ 已完成 | `ADR-0004`：`mq/dedup.go` TTL 事件 ID 幂等 + 5 个 MQ 指标 + 3 条告警（promtool 20 rules SUCCESS）+ `scripts/dlq-replay.sh`；`catalog.*` 明确为预留扩展点 |
| B3 | ✅ 已完成 | `ADR-0005`：`internal/data/shared/errors.go` 统一映射 + 表驱动测试；`order.go` 的 1062 判断委托 shared；`dict_enum` 决策为「数据基线，不运行时读取」 |
| C1 | ✅ 已完成 | `notify/smtp.go` 实现 SMTP（MIME/CRLF 注入过滤/可注入 send）+ `smtp_test.go`；生产启用返回真实发送器，本地仍走日志 |
| C2 | ✅ 已完成 | `wiring.go` 构造 `NewObjectStorage` 并注入 `service.Service`，`ObjectStorageEnabled()` 记录启用状态 |
| C3 | ✅ 已完成 | 删除 `uploadMaxBodyBytes` / `uploadPathPrefix` 与 `bodyLimit` 路径放宽分支 |
| C4 | ✅ 已完成 | 删除零调用的 `httpx.Plain` |
| C5 | ✅ 已完成 | `service/auth.go` 注释改为「任何环境只打印不可逆指纹」 |
| C6 | ✅ 已完成 | `API_CONTRACT.md` 标注 `payParams` 占位与 `support_team` 无端点 |
| C7 | ✅ 已完成 | `config.yaml` 头部加醒目测试口令警告与 env 覆盖清单 |
| C8 | ✅ 已完成 | `conf.go` 允许 `sample_ratio=0` 关闭采样 + `trace_test.go` |
| C9 | ✅ 已完成 | `captcha.randomID` 返回 error、`randN` 失败降级、`shared.NewEvent` 失败用纳秒+序号生成唯一降级 ID |
| C10 | ✅ 已决策完成 | 选择「清理引用」：`config.prod.yaml` endpoint 置空、Prometheus collector job 与 Grafana Loki 数据源移除、compose env 清理、`operations.md` 顶部能力边界、3 份 README 记录启用步骤 |
| C11 | ✅ 已完成 | `docs/legacy/` 空目录删除，`docs/README.md` 改为归档材料索引 |
| D1 | ✅ 已关闭（环境限制） | 本机无 gcc/CGO，`-race` 由 CI Linux runner 覆盖；非代码缺陷，已在部署验证记录中说明 |
| D2 | ✅ 已完成 | CI 新增 `quality` job（`make ops-check-go` + `make test-integration`）；Makefile 增加 `ops-check-go` / `check-all` |
| D3 | ✅ 已完成 | 新增 `schema_integration_test.go`（build tag integration，校验 9 基线库与 5 张关键表）；既有 `ops_integration_test.go` 覆盖跨库退款事务 |
| D4 | ✅ 已完成 | 新增 `usecase_smoke_test.go`，覆盖 Auth / Order.MakeOrder / Catalog / Feed / Message / User 六个 usecase 的边界路径 |
| D5 | ✅ 已完成 | `ops_load_test.go` 的 p99 硬阈值改为 `PLANSTAR_STRICT_LOAD=1` 时才断言，默认只记录 |
| D6 | ✅ 已完成 | `go.mod`、`Dockerfile`、CI 四处统一为 Go `1.26.5` |
| D7 | ✅ 已完成 | `TestGoToolchainPinned` 治理断言校验三处版本一致 + `.golangci.yml` 存在；未加 `toolchain` 指令以避免本机下载不兼容工具链 |
| E1 | ✅ 已完成 | 参考材料迁至 `docs/research/go-wind-cms-split-proposal.md`，7 个 md 文件的引用全局同步 |
| E2 | ✅ 已完成 | 5 份归档笔记迁至 `docs/notes/`，`docs/README.md` 建立归档索引 |
| E3 | ✅ 已完成 | `docs/sql/README.md` 不再写死表数，统一为「以 `verify.sql` 实跑为准」，历史数字标注为不同阶段快照 |
| E4 | ✅ 已完成 | `docs/sql/README.md` 新增「两套执行入口」章节，说明分文件 SOURCE 顺序与 compose 字典序挂载不冲突 |
| E5 | ✅ 已完成 | `release.yml` 注释改为现行三文件 compose，去掉不存在的 `docker-compose.lite.yml` / `PLANSTAR_LITE` |
| E6 | ✅ 已完成 | `deploy/planstar.service` 的 WorkingDirectory / ExecStart / ExecStop 改为实际路径与三文件组合 |

**验证结果**：`gofmt` 0 未格式化、`go build ./...` 0、`go vet ./...` 0、`go test ./... -count=1` 0 失败、
`go test -tags integration ./...` 0 失败（DSN 缺失自动 skip）、`golangci-lint run ./...` `0 issues.`、
`promtool check rules` 20 rules SUCCESS、四份 compose `config -q` 通过、CI YAML 8 个 job 解析正常。

---
# 未完成项台账（Open Items）

> 盘点日期：2026-11-19 · 基线提交：`b65b394`（工作区干净）
> 事实源：全仓代码扫描、`docs/ADR/ADR-0001` §9、`docs/ADR/ADR-0002` §6、`docs/local-deploy-verification.md`、`docs/notes/PLANSTAR_DEEP_STUDY.md`
> 本文只做**登记与排序**，不含实施动作。每项给出客观证据、影响、建议动作与验收方式；完成后请回写状态并同步 `docs/ADR/README.md` 进度表。

## 0. 结论速览

| 类别 | 未完成数 | 最高优先级 | 说明 |
|---|---|---|---|
| A 架构治理阶段（P2/P3 剩余） | 6 | P1 | P0/P1 与门禁已闭环，P2 只完成首批 |
| B 待立项的独立功能 ADR | 3 | **P0** | 其中生产配置可启动性是上线阻断项 |
| C 功能实现缺口 | 11 | P1 | 多为「配置已写、能力未接线」 |
| D 测试与工程 | 7 | P1 | 含本机 `-race` 环境限制与集成测试覆盖 |
| E 文档与资料归档 | 6 | P2 | 数字口径、材料归位、死配置注释 |

**已完成、不再列为待办**：P0 基线治理、P1 共享内核与装配拆分、P3 严格门禁、目录改名（`frontend/`→`planstar_frontend/`、`planstar/`→`planstar_backend/`）、CI/Release 路径同步、空目录与 11 库口径治理、legacy 死链清理、本地 Docker local deploy 验证（PASS）。两项提交：`e514842`、`b65b394`。

---

## A. 架构治理阶段未完成（ADR-0001 / ADR-0002）

| ID | 项 | 客观证据 | 优先级 | 建议动作 | 验收 |
|---|---|---|---|---|---|
| **A1** | **P2 剩余业务域物理分包** | `planstar_backend/internal/biz/{identity,trade,project,wallet,community,ops,catalog}/` 全部不存在；`internal/biz` 仍是单包（含 `auth.go`、`project.go`、`order.go` 等 19 个实现 + 11 个测试平铺）；`internal/data/{identity,...}`、`internal/service/api` 同样不存在 | P1 | 按 `ADR-0002` §3 映射表**逐域**迁移：先下沉 DO 到 `biz/shared/do.go`，再把每域 usecase + repo 端口移入子包，根包暂留类型别名兼容层；每域一次提交 | 7 个域包可独立 `go test ./internal/{biz,data}/<domain>/...`；`go build ./...` 与 57 条路由集合不变 |
| **A2** | `platform/` 归位 | `planstar_backend/internal/platform/` 不存在；`conf/`、`observability/`、`mq/`、`ops/`、`pkg/` 仍在 `internal/` 一级 | P1 | 迁到 `internal/platform/`，同步所有 import、CI 路径与部署脚本；保持 `platform` 不反向 import `biz` | `go list -deps ./internal/platform/...` 不含 `planstar/internal/biz`；治理断言仍全绿 |
| **A3** | `service/{api,admin}` 按端拆分 | `internal/service/api/` 不存在；`server/admin.go` 内的运营 handler 仍与业务 server 同包 | P1 | 把 9108 运营 handler / `OpsAuth` / console 迁到 `service/admin/`，8098 handler 迁 `service/api/`；`events/` 已完成可作样板 | 错误映射两套（`service.fail` 409 / admin 400）语义不变；`TestWriteAdminErrorMapping` 通过 |
| **A4** | 删除 `biz` 根包兼容层 | `internal/biz/{errors.go,event.go,models.go,moderation.go}` 现为别名/包装层（P1 有意保留） | P1 | A1 完成后，让 `service`/`data`/`cmd` 直接引用域包，删除兼容层 | 根包无残留文件；`go vet ./...` 通过 |
| **A5** | P3 拆分接缝固化 | CI 的 `paths` 仍是 `planstar_backend/**` 整体；指标 label 未按域；无「若需拆服务」候选清单 | P2 | 域间只经消费方端口协作；按域 CI 路径、指标 label、文档归属；产出候选服务清单 | 每个域可独立构建/测试；端口清单与现有 18 个 repo 端口对账完整 |
| **A6** | `.golangci.yml` 缺失，lint 为报告式 | 根与 `planstar_backend/` 均无 `.golangci.yml`；`.github/workflows/ci.yml` 的 `golangci-lint` step 仍带 `continue-on-error: true` | P1 | 补配置（排除 `api/gen`、`*.pb.go`、生成目录，对 `_test.go` 放宽 `errcheck`），然后去掉 `continue-on-error` | `golangci-lint run ./...` 通过且 CI 该 job 为必过 |

> **A1 为何不能一次做完**：`biz`/`data`/`service` 共 100+ 文件共享包内私有符号（`s.fail`、`s.requireUser`、`r.d`、`fen()`、各 DO）。一次性拆包会让跨包引用失效，必须逐域抽端口、下沉 DO、重写 import 并分批跑全量测试。已完成的 `service/events` 子包与 `data/project.go` 拆分验证了迁移路径可行。

---

## B. 待立项的独立功能 ADR（ADR-0003 / 0004 / 0005）

`docs/ADR/` 目前只有 0001、0002 与模板；以下三项在 `ADR-0001` §8 已列为「建议另行立项」，**尚未形成 ADR 文件**。

### B1 · ADR-0003 生产配置不变量与 `config.lite.yaml` 可启动性（**P0，上线阻断**）

| 证据 | 内容 |
|---|---|
| `planstar_backend/configs/config.lite.yaml:12` | `env: prod` |
| `:49` | `cookie_secure: false`（注释标「ICP 备案期间临时」），且 `conf.Load` 的 env 覆盖清单里**没有** `cookie_secure` |
| `:29` | `data.mysql.user: root` |
| `:84-85` | `ops.token: ""`、`allowed_cidrs: []` |
| `planstar_backend/internal/conf/conf.go:407-469` | `validateProductionInvariants` 在 `env:prod` 强制 `cookie_secure=true`、`ops.token`/`allowed_cidrs` 非空、DB user≠root、口令非空、短信启用 |
| `.github/workflows/release.yml:91,146` | deploy job **硬编码** `PLANSTAR_CONFIG_FILE=config.lite.yaml` |
| `.github/workflows/release.yml:9` | 注释称叠加 `docker-compose.lite.yml`（该文件不存在） |
| `planstar_backend/deploy/planstar.service` | `ExecStart` / `ExecStop` 引用不存在的 `docker-compose.lite.yml` |

**影响**：以 `env:prod` 启动 `config.lite.yaml` 会被 `conf.Validate()` 拒绝，而 Release 部署恰好使用该档位 → 生产启动失败。
**建议动作**：三选一并落 ADR：① 让 `cookie_secure` 可经 `PLANSTAR_*` 覆盖并在 prod overlay 注入合规值；② `config.lite.yaml` 改回 `cookie_secure: true` 并把 HTTP 直连需求放到 server overlay；③ 调整生产不变量使其允许显式豁免（不推荐）。同步修 `release.yml` 注释、`planstar.service` 与 `docker-compose.lite.yml` 死引用。
**验收**：`PLANSTAR_OPS_TOKEN=x ... go run ./cmd/planstar -conf configs/config.lite.yaml` 在 prod 档位能启动；`deploy/preflight.sh` 通过。

### B2 · ADR-0004 MQ 消费可靠性（P1）

| 证据 | 内容 |
|---|---|
| `planstar_backend/internal/mq/consumer.go` | 无 `ev.ID` 去重、无去重表；`retryOrDLQ` 重试/重连会重复投递 |
| `:90-94` | DLQ `<queue>.dlq` 只声明与绑定，全仓无消费者、无告警、无重放工具 |
| `internal/biz/shared/event.go:40` | `_, _ = rand.Read(b)` 忽略错误（事件 ID 可能全零） |
| 发布点搜索 | `catalog.activity.changed` / `deleted` 仅出现在 `biz/shared/event.go:10-11`（定义）与 `mq/server.go:17-18`（订阅），**无任何 `NewEvent` 发布点**；目录增量同步实际靠 300s 全量重建兜底 |
| `biz` 发布失败 | `AuthUsecase.Login` / `OrderUsecase.MakeOrder` 中 `Publish` 失败仅 `Warn`，无 outbox/事务消息 |
| `consumer.go` 重试 | 消费者内 `time.After` 指数退避（1/2/4/8/16s）阻塞串行 `Run`，单条退避会拖住整队列 |

**影响**：重复站内通知；DLQ 消息无人处理；发布与主流程不原子。
**建议动作**：基于 `ev.ID` 的幂等消费表、DLQ 告警与重放工具、事务性 Outbox、重试改 TTL 延迟队列；补齐 `catalog.*` 发布点或删掉空链路。
**验收**：新增消费幂等单测 + 集成测试；DLQ 有告警规则与重放脚本。

### B3 · ADR-0005 错误映射集中化与 `dict_enum` 处置（P2）

| 证据 | 内容 |
|---|---|
| 全仓 `.go` 搜 `dict_enum` | **0 命中**；枚举为 `biz/project.go`、`biz/payment.go`、`biz/project_idea.go` 中的硬编码白名单 |
| `planstar_meta.dict_enum` | 建表 + seed 存在，但不参与运行时查询 |
| `internal/data/` | 无统一 driver→biz 错误映射 adapter；`data/order.go:401 isDuplicateKey` 是唯一公共 helper，`sql.ErrNoRows` 处理散落在十余个 repo |
| `AGENTS.md` | 已修正口径为「枚举是硬编码白名单」，但「字典表保留还是废弃」未决策 |

**建议动作**：明确决策：字典表继续作为数据基线保留（推荐，运行时读白名单）或废弃；抽 `data/shared/mapDriverError` 统一映射。
**验收**：ADR 落档；映射收敛后所有 data repo 不再散落 `sql.ErrNoRows` 分支。

---

## C. 功能实现缺口（配置已写、能力未接线）

| ID | 项 | 证据 | 优先级 | 建议动作 |
|---|---|---|---|---|
| **C1** | 生产邮件全丢 | `internal/pkg/notify/notify.go:134-136`：`NewEmail` 生产直接返回 `NoopEmail{}` 并 Warn `email provider not implemented` | P1 | 接入 SMTP provider，或在契约/运维手册中明确「不提供邮件通知」并删除 `Notify.Email` 配置项 |
| **C2** | OSS 未接线 | 全仓 `NewObjectStorage` **0 调用**；`conf.Storage.OSS` 与 `PLANSTAR_STORAGE_OSS_*` 是未使用配置；`internal/pkg/storage` 仅有自身测试 | P1 | 在 `cmd/planstar/wiring.go` 装配并注入需要的用例，或删除该配置段落避免误判 |
| **C3** | upload 端点缺失但保留常量 | `internal/service/service.go` 搜 `/v1/upload` **0 命中**；`internal/server/http.go:23-26` 仍保留 `uploadMaxBodyBytes=20MiB` 与 `uploadPathPrefix`，注释写明「当前无 upload 端点，保留常量供后续启用」 | P2 | 实现端点或删除常量与注释 |
| **C4** | `httpx.Plain` 历史兼容壳 | `internal/pkg/httpx/httpx.go:44-49` 定义仍在，全仓调用数 **0**（唯一命中是 `internal/governance/governance_test.go:237` 的断言字符串） | P2 | 确认无外部调用后删除，并同步 `API_CONTRACT.md` 中「仅为历史兼容保留」的表述 |
| **C5** | 验证码日志注释漂移 | `internal/service/auth.go:92` 注释「开发环境的日志实现会打印明文」；实际 `logSMS.SendCode` 只输出 `codeFingerprint`（SHA-256 前 8 字节），不打明文 | P2 | 修注释；保留「绝不打明文」的安全约束 |
| **C6** | 契约中的占位与未接表 | `docs/api/API_CONTRACT.md` 中 `payParams` 为占位 `{}`；`support_team` / `support_team_member` 表存在但无对应端点 | P2 | 补实现或在契约中标注「未实现/预留」 |
| **C7** | dev 基线含明文弱口令 | `planstar_backend/configs/config.yaml:20-25` `user/password: root`；`:55` `addr: amqp://root:root@127.0.0.1:5672/` | P2 | 生产 overlay 已强制 `${VAR:?}`，但建议把 dev 默认值改为占位符并在 `.env.example` 说明，避免误用于非本地 |
| **C8** | 采样率配置意图与实现不一致 | `internal/conf/conf.go:550-551`：`SampleRatio <= 0 || > 1` 一律改为 `1`；而 `config.local.yaml` / `config.lite.yaml` 写 `sample_ratio: 0` 想关闭采样（实际靠 endpoint 为空关闭 tracing） | P2 | 让 0 表示「关闭」，或用显式 `enabled` 字段；同步注释与配置 |
| **C9** | 随机数失败处理不一致 | `internal/pkg/captcha/captcha.go:43` `rand.Read` 失败 `panic`；`:124` `randN` 忽略错误；`internal/biz/shared/event.go:40` 忽略 `rand.Read` 错误 | P1 | 统一为「失败关闭并返回 error」，禁止 panic 与可预测 ID |
| **C10** | 可观测栈容器未编排 | `planstar_backend/deploy/{loki,promtail,otel}/` 各只有一份 README；四份 compose 均未定义 `loki`/`promtail`/`otel-collector` 服务，但 `config.prod.yaml` 的 `trace.endpoint=otel-collector:4317`、Prometheus 的 otel-collector job、Grafana Loki 数据源都在引用 | P1 | 二选一：补齐服务定义与配置；或删除配置/文档中的引用并明确「当前无链路与日志聚合」 |
| **C11** | `docs/legacy/` 仅占位 | 目录只有 `README.md`（历史文档未随仓库检出）；死链已清理，但目录仍空 | P2 | 要么补入真实废弃文档，要么删除目录并同步 `docs/README.md` |

---

## D. 测试与工程

| ID | 项 | 证据 | 优先级 | 建议动作 / 验收 |
|---|---|---|---|---|
| **D1** | `go test -race` 本机无法执行 | `-race requires cgo` + `gcc not found in %PATH%`（本机 Windows）；CI ubuntu runner 会跑 `go test -race ./...` | P2（环境） | 本机装 mingw/LLVM 或接受 CI 覆盖；已在 `docs/local-deploy-verification.md` 记录，非代码缺陷 |
| **D2** | CI 不跑运维面与集成测试 | `.github/workflows/ci.yml` 搜 `ops-check` / `test-integration` **0 命中**；`planstar_backend/Makefile:55` `check: fmt vet governance test` 不含 `ops-check` | P1 | 加 `make ops-check`（探针/鉴权/压测/告警配置）与可选 `test-integration` job；验收：CI job 全绿且集成测试有 `PLANSTAR_TEST_MYSQL_DSN` 时执行、缺失时 skip |
| **D3** | 集成测试覆盖不足 | `planstar_backend/internal/data/` 只有 `data_test.go` 与 `ops_integration_test.go`；auth/order/payment/wallet/project 的 SQL 无集成测试 | P1 | 按 `testutil.MySQLDatabase` 模式补关键事务（下单幂等、跨库退款审核、签到并发） |
| **D4** | 6 个 usecase 无 biz 单测 | `internal/biz/` 的 11 个测试文件覆盖 `chat/models/ops/payment/project_idea/project/quote/quote_ttl/report/shop/wallet`，**缺** `AuthUsecase`（登录/验证码/限流/时序防护）、`OrderUsecase.MakeOrder`、`CatalogUsecase`、`FeedUsecase`、`MessageUsecase`、`UserUsecase` | P1 | 用内存 fake repo 补表驱动单测；验收：`go test ./internal/biz -cover` 提升且核心链路有断言 |
| **D5** | 压测型单测的抖动风险 | `internal/server/ops_load_test.go` 是 100 goroutine × 200 请求并发压测，带 p99 硬阈值（`:133,143`） | P2 | 改为 build tag 隔离的基准测试，或放宽阈值并只在专用 job 跑 |
| **D6** | Go 工具链版本漂移 | `planstar_backend/go.mod`: `go 1.26.0`；`Dockerfile`: `golang:1.26.5`；`ci.yml`: `go-version: "1.26"`；本机实测 `go1.27.3` | P1 | 统一到同一 patch 版本并写入 `docs/notes/CODEGEN_VERIFICATION.md` 建议的工具链锁定；验收：`go version` 与三处一致 |
| **D7** | 工具链锁定建议未落地 | 根 `docs/notes/CODEGEN_VERIFICATION.md` 记录了 Go 1.27.3 与 1.25.7 错配导致 `http2.TrailerPrefix` / Ent 生成器崩溃，建议锁工具链 | P2 | 加 `toolchain` 指令或 `.tool-versions`，并在 CI 校验 |

---

## E. 文档与资料归档

| ID | 项 | 证据 | 优先级 | 建议动作 |
|---|---|---|---|---|
| **E1** | 参考材料仍在 ADR 目录 | `docs/research/go-wind-cms-split-proposal.md` 已在文首加「设计推演、非实战复盘」标注，但仍位于 `docs/ADR/`；`docs/research/` 不存在 | P2 | 迁到 `docs/research/go-wind-cms-split-proposal.md`，同步 `ADR-0001` §1.1、`docs/ADR/README.md` 与 `docs/README.md` 的引用 |
| **E2** | 根目录归档笔记未归位 | `docs/notes/PLANSTAR_DEEP_STUDY.md`、`docs/notes/MIDDLEWARE.md`、`docs/notes/CODEGEN_VERIFICATION.md`、`docs/notes/KRATOS_STUDY_NOTES.md`、`docs/notes/KRATOS_SHOP_STUDY_NOTES.md` 仍在仓库根；`docs/notes/` 不存在 | P2 | 迁到 `docs/notes/`，在 `docs/README.md` 标注「归档研究，不得作为实现/排障依据」 |
| **E3** | SQL 文档数字自相矛盾 | `docs/sql/README.md:13` 写「97 张业务表 + 3 张证据表」、`:52` 写「100 表」、`:62` 写「86 张业务表」、`:129` 又提到旧的「89 表」期望值 | P2 | 以 `docs/sql/verify.sql` 的验收口径统一，并在 README 注明「历史快照数字见 `history/`」 |
| **E4** | SQL 执行顺序口径冲突 | `docs/sql/README.md` 要求 `15_project_domain.sql` 在 `14_mock_seed.sql` **之前**，但四份 compose 按文件名字典序挂载（14 在 15 前），生产又用 `14_noop.sql` 同名覆盖 | P1 | 统一为实际生效顺序并核对 `00_all.sql` 是否已内含企划域表；验收：全新 MySQL 初始化后 `verify.sql` 全 PASS |
| **E5** | Release 注释与实现不符 | `.github/workflows/release.yml:9` 写 `PLANSTAR_LITE=1` 与 `docker-compose.lite.yml`；实际 deploy job 硬编码 `config.lite.yaml`，且该文件不存在（同 B1） | P2 | 修注释或补文件；与 B1 一并处理 |
| **E6** | systemd 单元引用死文件 | `planstar_backend/deploy/planstar.service` 的 `ExecStart` / `ExecStop` 使用 `docker-compose.lite.yml`（不存在） | P2 | 改为现行三文件 compose 组合，或删除该单元并说明用 compose 管理 |

---

## F. 已明确排除 / 非待办（避免误判）

| 项 | 说明 |
|---|---|
| 物理拆服务 / 双 BFF / gRPC 微服务化 | `ADR-0001` 方案 A 已否决；当前决策是模块化单体 |
| 全量 proto/buf 生成链 | `ADR-0001` 方案 D 已否决；Flutter 裸 JSON + Cookie 契约为硬约束 |
| 按域做一级目录 `internal/<domain>/{biz,data,service}` | `ADR-0001` 方案 E 已否决，作为远期可选 |
| `.archive/go-wind/` 两个参考项目 | 已加入 `.gitignore`，只读参考快照，不参与构建/CI/部署 |
| `admin/` 旧目录重排 | 已随 `e514842` 提交完成（`planstar_frontend/admin/`），CI paths 已同步 |
| `go test -race` 本机限制 | 环境限制，非代码缺陷；CI Linux runner 覆盖 |
| `site/assets/`、`site/openapi/`、`backend/` 空目录 | 已在 `e514842` 删除 |
| PolyForm Noncommercial 1.0.0 | 许可证约束，不是待办 |

---

## 状态维护约定

1. 完成某项后，在本表把状态改为已完成并注明提交 hash，同时更新 `docs/ADR/README.md` 进度表。
2. 新增未完成项必须给出**可复核证据**（`文件:行号`、grep 结果、构建/测试输出），不写主观评价。
3. B 类三项立项后，按 `docs/ADR/ADR-NNNN.md` 模板生成 ADR-0003/0004/0005，并在本文把对应条目改为「已有 ADR」。
4. 涉及对外契约（路径、Cookie、错误码、金额单位、分页参数）的整改，须显式标注「是否需要 Flutter 客户端协同」。
