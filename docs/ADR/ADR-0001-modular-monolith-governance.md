# ADR-0001: 采用模块化单体架构治理，拒绝物理拆服务与全量契约生成

- **状态**: ACCEPTED（设计决策已选定；实施按 P0–P3 分批落地，本阶段不改动任何代码）
- **分类**: ARC（关联 DEBT / TST / CFG）
- **优先级**: P1
- **影响维度**: 认知障碍 / 上下文膨胀 / 幻觉源 / 生产功能
- **日期**: 2026-11-19
- **来源**: 全仓代码审计（见根目录 `PLANSTAR_DEEP_STUDY.md`）+ `.archive/go-wind/{go-wind-cms,go-wind-admin}` 代码实践核对 + `docs/ADR/planstar_arch_optimize.md` 适用性复核
- **修复 commit**: —（本 ADR 仅为设计文档）

---

## 0. 决策摘要（TL;DR）

| 项 | 结论 |
|---|---|
| **最终选型** | **模块化单体（Modular Monolith）**：保留单进程、11 个逻辑库同实例与手写 `ServeMux` 契约；在既有 `biz/data/service` 三层内部按业务域分包，抽取共享内核，装配点按域注册，契约与文档改为机器可校验 |
| **明确否决** | ① 照复盘文章做「双 BFF + 多业务服务 + 库物理隔离」的微服务化；② 引入全量 proto/buf 生成链；③ 只补文档不动代码；④ 按域做一级目录（`internal/<domain>/{biz,data,service}`） |
| **关键理由** | 文章的"实战复盘"经核对为**设计推演**（仓库内无商城业务、五层架构未落地、Wire 已废弃、服务发现实为 etcd），不能作为事实依据；go-wind 真实代码最大的教训恰恰是**实现层巨型平包**（core 的 `data` 114 文件 / `service` 104 文件、`wiring.go` 439 行 80+ 实参），而 planstar 已具备其缺失的 `biz` 层与依赖倒置，应加固而非推翻 |
| **本阶段产物** | 仅 `docs/ADR/` 下 3 份文档（本 ADR、`ADR-0002`、索引 `README.md`），零代码改动 |

---

## 1. 现状问题（Context / Problems）

### 1.1 参考基线的适用性必须先纠偏

`docs/ADR/planstar_arch_optimize.md`（43.4 KB，日期 2026-06-18，带博客式 frontmatter）自称「基于 go-wind-cms 的官网+商城双业务渐进拆分**实战复盘**」。对 `.archive/go-wind/` 真实代码逐条核对后：

| 文章主张 | 仓库事实 | 判定 |
|---|---|---|
| admin/app/core 三层、BFF 不碰 DB | `.archive/go-wind/go-wind-cms/backend/app/{admin,app,core}/service/` 存在；`backend/AGENTS.md:42-46` 写死「BFF 是否访问 DB = ❌」；两个 BFF 的 `internal/` 搜 `entgo.io/ent \| database/sql \| gorm.io \| sql.DB` 均 0 命中 | **事实·吻合** |
| 后期新增商城业务、Core 变成双业务大包 | 全仓无 `*order*.go` / `*cart*.go` / `*inventory*.go` / `core.proto`；「订单/库存」仅出现在 `backend/sql/*-demo-data.sql` 的字典种子与通知模板字符串 | **推演，未发生** |
| 已有 `core.proto` 且越来越大 | 实际是 `api/protos/<域>/service/v1/` 下 **209 个按域拆分**的 proto | **与代码不符** |
| 目标五层 `system/cms/shop/service` 已落地 | `backend/app/{system,cms,shop}` 全部不存在；文章第三章、第四章本身是方案体（「将…重命名」） | **未落地** |
| 用 Google Wire | `go.mod` 无 `google/wire`；`CHANGELOG.md:263`「移除 google/wire，依赖装配改为手写 wiring.go」；三份 `wiring.go` 注释「替代原 google/wire 生成」 | **描述的是已废弃的历史状态** |
| 服务发现 Consul / 消息队列 Kafka | 注册中心实为 **etcd**（`configs/registry.yaml:2 type: "etcd"`，compose 中 consul 段被注释）；`go.mod` 搜 `kafka` 0 命中；异步为 Asynq/Redis | **与代码不符** |
| Kratos 版本 / 收益数字 | 实为 **v2.9.2 + Ent v0.14.6 + Go 1.26.4**；「发布风险降低 80%」「30s→5-10s」「成本降 40%」在全仓唯一出现于该文自身，无 benchmark / 日志 / 文档支撑 | **无证据的定性估计** |

> **治理结论**：该文可作为「架构演进思路的参考读物」，**不得**作为本 ADR 的事实依据或量化决策依据。ADR 内所有 go-wind 引用一律以 `.archive` 内一手代码与 `backend/AGENTS.md` 为准。
> 另：该文放在 `docs/ADR/` 目录内且不是 ADR 格式，本身就是「幻觉源」——见 §5 决策 D4。

### 1.2 go-wind 真实代码给出的正反面教材

**正面（可迁移）**

1. **手写 DI + 分层小节 + LIFO cleanup**：`go-wind-cms/backend/app/core/service/cmd/server/wiring.go` 用「一、基础设施 → 二、仓储层 → 三、服务层 → 四、传输层」分段，`cleanups []func()` + `rollback()` 让启动失败与优雅停机共用一条路径（`:41-49, :438`）。planstar 本就无 Wire，落地成本为零。
2. **`register:*` 锚点 + 幂等登记器**：`backend/tools/register/main.go` 一次注入仓储行、服务行、server 形参与路由注册 5 处，`skipIf` 保证幂等；`backend/AGENTS.md:389` 的关键论断是「漏登记 = 编译错误在准确行，而 Wire 时代忘跑 `make wire` 是静默漏装配」。
3. **契约按域分包**：`api/protos/{identity,content,permission,audit,...}/service/v1/`（cms 209 proto / 29 域；admin 149 proto / 21 域），BFF 域用 `i_` 前缀并 `import` 复用源领域 message，可裁剪对外 API 面积；`buf.yaml` 用 `breaking: FILE`，proto 里有 `reserved 10,11,12` + `reserved "visits",...` 的版本化实证。
4. **规范单一权威**：`AGENTS.md` 一份写死分层禁止项、`CLAUDE.md` 只做 `@AGENTS.md` 指针；`CONTRIBUTING.md` 固化 Conventional Commits 与 PR 自测清单；`.golangci.yml` 用 `exclusions.paths` 排除 `api/gen`、`*.pb.go`、ORM 生成目录，保证 lint 报的每条都能在源头修。
5. **Makefile 分层门禁**：根 `Makefile` 管工具链/代码生成/多服务自动发现（`SRCS_MK` 通配符），`app.mk` 管单服务 build/run，`gen` 与 `build` 分离（避免「改了契约没重生成就发版」）；`go-wind-admin` 有 269 个 `*_sqlite_test.go` 内存库白盒测试，可离线跑。

**反面（必须引以为戒）**

1. **实现层巨型平包**：`go-wind-cms/core/service/internal/` 只有 `data`(114 个平铺 .go)、`service`(104 个平铺 .go)、`server`(2)，**无 `biz`、无 `conf`**；`grpc_server.go:54-130` 形参 90+ 个 service，`wiring.go:387-430` 一次传 80+ 实参。`go-wind-admin` 同构：`internal/data` 约 200 文件、`internal/service` 约 179 文件全平铺，领域信息只编码在文件名前缀里。
2. **依赖倒置只做对一半**：`go-wind-cms` 的 `data/user_repo.go:35-69` 定义了 `UserRepo` 接口且构造函数返回接口，但 `service/user_service.go:48-55` 又持有多个具体 `*data.XxxRepo`，13 个 service 直接 `import internal/data/ent`；`AGENTS.md:121`「构造函数返回接口」未全面执行。planstar 现有 18 个 repo 端口 + 纯 Go `biz` 层**优于**这套实践，不能倒退。
3. **规范写全、机器不拦**：`go-wind-admin/.github/workflows/` 只有 3 条（README 三语一致性、更新掩码白名单一致性、SSO E2E），**没有 `go test` / `golangci-lint` / `go vet` 门禁**；`docs/backend_project_struct.md:32-51` 列出的 pkg 子包 `authorizer/crypto/eventbus/...` 实际都不存在，`backend/README.md:8` 仍写 Wire；`configs/server.yaml` 开 `enable_tracing: true` 但 tracer 插件被注释、无 OTel 打点。
4. **"BFF 零业务逻辑"是口号**：`admin/service/internal/collab/` 有 ygo CRDT 状态管理、`audit_export_service.go:29-58` 做 5 万行分页聚合导出，app 侧 `seo_service.go` 生成 sitemap、还有手写路由绕过 proto 契约。准确表述应是「零 DB / 零本地事务 / 薄持久化」。
5. **巨型 wiring 与构造期副作用**：`wiring_ent.go` 近 400 行、上百实参；默认数据播种、脚本 `Resync`、`AttachEntityHooks` 都靠构造顺序隐式生效，无显式生命周期钩子。

### 1.3 planstar 自身的问题清单（量化证据）

实测环境：本机 `go1.27.3`（`planstar_backend/go.mod` 声明 `go 1.26.0`，Dockerfile 用 `golang:1.26.5`）；`go build ./...` 退出 0；`go test ./...` 15 个包全绿。

| 编号 | 问题 | 证据 |
|---|---|---|
| **Q1 实现层平包** | `internal/` 各包单包文件数：`biz` 29、`data` 19、`service` 18、`pkg` 15、`server` 10；三层内部**零领域子包**，55+ 端点的 handler、14 个 usecase、18 个 repo 端口全部平铺。最大文件 `data/project.go` 34,037 B、`conf/conf.go` 23,163 B、`biz/project.go` 20,239 B | 目录统计命令输出；`planstar_backend/internal/service/service.go:73-144` 单文件承载 57 条路由 |
| **Q2 装配集中** | `cmd/planstar_backend/main.go` 327 行，单文件完成日志、配置、指标、鉴权、追踪、10 库连接、MQ、Consul、SMS、12 个 usecase、3 个 server、就绪探针、kratos 选项；无分层小节、无按域注册函数 | `main.go:45-301` |
| **Q3 共享内核缺位** | `biz/models.go` 导出 `Fen/FmtDT/FmtDTp/BirthTs` **给 data 层使用**，`biz/auth.go` 定义全局错误，`biz/event.go` 定义事件；DO/错误/事件/单位换算混在领域层根包。一旦按域拆包，这些会立刻产生循环依赖 | `biz/models.go`、`biz/auth.go:21-27`、`biz/event.go`；`biz/quote.go:58 QuoteTTLFromEnv` 还在领域层直接 `os.Getenv` |
| **Q4 契约与规范漂移** | `agents.md` 的 5 条「必须保留的契约怪癖」中，「登录失败 5xx+text/plain」已被代码否定（`service/auth.go:132-137` 是 400/401 JSON，`httpx.Plain` 全仓零调用）；「发码恒 200」漏了 429；「枚举取自 dict_enum」与硬编码白名单不符。路由代码 57 条/56 唯一端点，`API_CONTRACT.md:9` 写 56/55，而 `routes_test.go` 只比集合不比条数，测试照样通过 | 源码复核 + `go test ./...` 实测 |
| **Q5 契约无机器校验** | 事实源是 Markdown 文档 + `service.go` 路由表，二者靠人工同步；错误码是 `service.go:181-223` 的 18 个字符串字面量；`ADMIN_CONTRACT.md` 缺 4 条运营写路由，且称「本地令牌为空鉴权关闭」与 `OpsAuth` fail-closed 冲突 | `docs/api/{API_CONTRACT,ADMIN_CONTRACT}.md`、`server/ops_auth.go:80-91` |
| **Q6 ADR 台账失效** | 治理前 `docs/ADR/` 只有 `ADR-NNNN.md` 模板 + 43 KB 的非 ADR 复盘文章，**没有任何已记录的架构决策**；`git status --short` 显示 `planstar_arch_optimize.md` 为未跟踪（`??`）材料，即它并非仓库既有决策记录；`docs/legacy/` 0 文件却有多条文档链接指向它 | 目录列举 + `git status --short` + 前次审计 |
| **Q7 空目录与死链** | 实测 0 文件的目录：`backend/`、`docs/legacy/`、`site/assets/`、`site/openapi/`、`planstar_backend/deploy/{loki,promtail,otel}/`；但 Grafana 数据源、Prometheus 的 otel-collector job、`config.prod.yaml:61-65` 的 `trace.endpoint=otel-collector:4317` 都在引用未编排的容器 | 目录统计；`PLANSTAR_DEEP_STUDY.md` §9.2 |
| **Q8 CI 门禁缺口** | `.github/workflows/ci.yml` 的 paths 含 `admin/**`；CI 不跑 `make ops-check` / `test-integration`，无文档一致性校验。**关于 `admin/**` 的准确判定见 Q11**：它不是「从未存在的写错路径」，而是目录重排后未同步的残留 | `ci.yml` 路径输出实测 + `git status --short` |
| **Q9 多口径与配置矛盾** | 库数量三套口径（文档 9 / `data.go:46-49` 连接池 10 / 运行时 11）；`config.lite.yaml` 以 `env:prod` 启动必被 `conf.Validate()` 拒绝（`cookie_secure:false` 无 env 可覆盖），但 `release.yml` 的 deploy job 硬编码该档位 | `PLANSTAR_DEEP_STUDY.md` §12 |
| **Q10 装配与边界未沉淀为可执行规则** | `AGENTS.md` 写「service 不 import data、biz 纯 Go」，但没有 lint / CI 断言；`data` 层驱动错误映射散落手写，无统一 adapter（`data/order.go:401 isDuplicateKey` 是唯一公共 helper） | `data/` 全包扫描 |
| **Q11 仓库处于目录重排中间态** | `git status --short` 显示大量 `D admin/...`（含 `cmd/server/wire.go`、`wire_gen.go`、`internal/data/ent/...`、`console/`）与 `?? planstar_frontend/`（未跟踪），即 `admin/` 正被迁往 `planstar_frontend/admin/` 但尚未提交；同时 `?? .archive/`、`.agents/`、`PLANSTAR_DEEP_STUDY.md`、`docs/ADR/planstar_arch_optimize.md` 均为未跟踪。**CI paths、文档链接、部署脚本与 ADR 引用必须以「重排完成后」的布局为准，不能按当前工作区瞬时状态定结论** | `git --no-pager status --short`（2026-11-19 实测） |

**问题定性**：planstar 的**分层与依赖倒置是健康的**（`biz` 无 `net/http`/`database/sql`/redis/amqp/elastic 引用，18 个端口全部由 `data`/`mq` 实现），问题集中在「**单体内缺少领域边界的物理表达**」「**装配与契约没有工程化治理**」「**文档与代码没有机器一致性**」「**目录重排未完成导致的多套路径口径**」。这是**结构债与治理债**，不是架构选型错误。

> **对本 ADR 的约束**：Q11 意味着任何目录相关决策都必须在「`admin/` → `planstar_frontend/admin/` 重排提交完成」之后实施，且 P0 必须同时校验 CI paths、`.github/workflows/`、部署脚本、`docs/README.md` 索引四处路径一致。

---

## 2. 优化目标（Goals）

按可度量方式表述，全部可在实施后用命令验证：

| 目标 | 度量方式 | 目标值 |
|---|---|---|
| **G1 单包职责单一** | 任一 `internal/{biz,data,service}` 子包的 `.go` 文件数 / 承载业务域数 | ≤ 20 文件且 ≤ 1 个业务域；单文件 ≤ 30 KB 或 ≤ 800 行 |
| **G2 装配可扩展** | 主装配函数实参数、`main.go` 行数 | `main.go` ≤ 80 行（只留 flag/log/App）；`wiring.go` 单个 `RegisterXxx` 域注册函数 ≤ 30 实参；新增一个端点/用例不再要求改全局装配主干 |
| **G3 依赖方向可静态检查** | `go list -deps` 断言 | `biz/*` 不出现 `data/*`、`service/*`、`net/http`、`database/sql`；`service/*` 不出现 `data/*`；`platform/*` 不出现 `biz/*`；**纳入 CI 必过 job** |
| **G4 契约零漂移** | 路由/错误码/契约文档一致性 | 路由条数、`METHOD /path` 集合、错误码表三者在 CI 中双向比对，`API_CONTRACT.md`、`ADMIN_CONTRACT.md`、`agents.md` 中的过时口径全部清除 |
| **G5 决策可追溯** | ADR 台账 | `docs/ADR/` 内每个文件都是 ADR 格式；非 ADR 材料迁出并标注性质；有索引与状态字段 |
| **G6 幻觉源清零** | 空目录/死链/口径一致性 | 8 个 0 文件目录要么填充要么删除；11 库口径在代码、SQL、文档、backup 脚本中统一；CI 路径与真实目录一致 |
| **G7 零运行时回归** | 对外行为 | 55+ 端点的 method/path/响应结构/Cookie/错误码/金额单位/分页参数**逐字不变**；`go build ./...`、`go vet ./...`、`go test -race ./...` 全绿；`/healthz`、`/readyz`、`/metrics` 语义不变 |

**非目标（明确不做）**：不提升并发上限、不改数据库拓扑、不改进业务算法、不修 MQ 幂等/DLQ 缺口（这些属于独立的功能性 ADR，见 §7 关联）、不引入新中间件。

---

## 3. 治理范围与约束（Scope & Constraints）

### 3.1 范围内（In Scope）

- `planstar_backend/internal/` 全部 103 个 Go 文件的**包归属**（仅本阶段做设计，不移动文件）。
- `planstar_backend/cmd/planstar_backend/` 装配结构、`planstar_backend/Makefile` 与门禁目标。
- `docs/ADR/`、`docs/api/`、`docs/README.md`、根 `agents.md` / `README.md` / `CLAUDE.md` 的口径与台账。
- `.github/workflows/ci.yml` 的 paths 与 job 组合。
- 仓库根目录结构：`.archive/`、`planstar_frontend/`、`site/`、`scripts/`、`backend/`、根目录 4 份归档 `*NOTES.md` 与 `PLANSTAR_DEEP_STUDY.md` 的定位。

### 3.2 范围外（Out of Scope）

- `planstar_frontend/admin` 的 Go 服务内部实现、`planstar_frontend/planstar_web` 的组件细节、`docs/sql/` 的 DDL 逐列。
- `planstar_backend/deploy/` 下中间件配置的具体参数、告警阈值、Docker 镜像版本。
- 任何业务规则（定价、限流阈值、状态机、黑名单词表）。

### 3.3 硬约束（不得违反）

| 编号 | 约束 | 来源 |
|---|---|---|
| **C1 对外契约冻结** | 55+ 端点的 `METHOD /path`（含 `/v1` `/v3` `/v4` 版本错配与 `/v1/user/` 尾斜杠别名）、裸 JSON 无信封、`Cookie: psSession`、金额「DB 元 / API 分」、分页 `page` vs `pageIndex`、字段别名、`{code,message,traceId}` 错误体**逐字保持**。Flutter 客户端已按此上线 | `docs/api/API_CONTRACT.md:5-11`、`POSITIONING.md`、`PRODUCTION_DELIVERY.md` |
| **C2 单进程单实例** | 保持一个 Kratos App、三个 `transport.Server`（业务 8098 / 运维 9108 / MQ 消费者）、11 个逻辑库同 MySQL 实例；**不引入 gRPC 跨进程、服务发现调用、BFF 层** | `docs/architecture.md` §1、`main.go:250-294` |
| **C3 事务模型不变** | 跨库一致性继续由「同一 `*sql.Tx` + `planstar_xxx.table` 限定名 + `FOR UPDATE` + 条件更新 + 唯一键幂等」保证；**禁止把现有跨库单事务拆成跨服务调用**（会立刻引入分布式事务） | `data/project.go:504-505` 注释、`docs/sql/12_constraints.sql` |
| **C4 不引入 proto/Wire** | `docs/architecture.md` §10 的 5 条「有意偏差」继续有效：无 `khttp.Middleware`、无 proto/OpenAPI binding、无 Wire、无 `kratos/errors`、固定错误体。理由：契约稳定，改造是破坏性变更 | `docs/architecture.md` §10 |
| **C5 分层依赖倒置不倒退** | `service` 不 import `data`；`biz` 纯 Go（禁 `net/http`/`database/sql`/redis/amqp/elastic）；repo 端口定义在消费方 `biz`，实现在 `data`；`cmd` 唯一装配点 | `agents.md`、实测 import 扫描 |
| **C6 资源与部署档位** | 单机 4C8G / 2M 带宽、低配 `config.lite.yaml` 关闭 ES/MQ/Consul/tracing；治理不得增加常驻进程或内存预算（当前 ≈5.6 G） | `docs/deployment.md` §2.7、`PRODUCTION_DELIVERY.md` |
| **C7 降级语义保持** | MySQL/Redis critical，ES/MQ/Consul/Jaeger optional；未配置时 `NopEventPublisher` / 搜索回退 MySQL / `skipped` 探针行为不变 | `main.go:112-117,319-353`、`ops/readiness.go` |
| **C8 许可与提交规范** | PolyForm Noncommercial 1.0.0；Conventional Commits；不得提交真实凭据 | `LICENSE`、`agents.md` |
| **C9 本阶段只写文档** | 不生成、不移动、不删除任何代码与配置；实施需另开变更并逐阶段验证 | 本次任务要求 |

### 3.4 外部协作标注

- **后端组**：ADR 实施主体；负责 P1–P2 包迁移与装配改造。
- **Flutter 客户端**：C1 保证零改动；若未来任何契约变更需显式知会并另立 ADR。
- **运营/平台**：`docs/ADR/` 台账、CI 门禁、部署脚本路径变更需同步；`planstar_frontend/admin` 团队需确认 CI paths 修正。
- **文档维护方**：`docs/README.md` 的「有效文档索引」需同步，避免治理后产生新的死链。

---

## 4. 备选方案及取舍理由（Options & Trade-offs）

### 方案 A：照复盘文章做「五层轻中台」物理拆分

双 BFF（app/admin）+ `system`/`project`/`trade` 业务服务 + `system_db`/`project_db`/`trade_db` 三库物理隔离 + proto/buf + Consul 服务发现，四阶段灰度。

- **优点**：发布隔离、故障半径收敛、资源弹性、资金数据独立审计；文章第五章给出收益对照。
- **致命问题**：
  1. **依据不成立**：§1.1 已核对，文章的商城业务、五层落地、Wire/Consul/Kafka、收益数字均非事实；「Core 大包」在真实代码里是「无 biz 层的胖服务」，与 planstar 有 `biz` 层的状况不同类。
  2. **直接违反 C1/C2/C3**：跨库单事务（`CreatePledge`、`CreateOrder`、`Redeem`、`ReviewRefund`）拆进程后必须引入 Saga/TCC/Outbox；55+ 裸 JSON + Cookie 端点要全部重写为网关转发；单机 4C8G 无法承载 5+ 进程与 etcd/Consul。
  3. **历史已否决过**：`docs/legacy/` 的设计意图（已废弃 gRPC/Consul/`shop_*`/PostgreSQL 微服务拆分）与 `docs/architecture.md` §10「有意偏差」正是为了避免这条路径。
  4. go-wind-cms 的反面实证：三进程换来 209 proto → 690+ 生成文件、10 份 buf 配置、`wiring.go` 439 行、BFF 仍藏业务逻辑（collab/审计导出/SEO），且 CI 无后端质量门禁。
- **结论：否决。** 仅保留其「按业务域划分边界」「渐进式、存量兼容」两条**思想**，在进程内实现。

### 方案 B：只补文档与口径，不动目录结构

修正 `agents.md`、补齐 ADR、修 CI paths、统一 11 库口径、清理空目录。

- **优点**：成本最低、零回归风险；能解决 Q4/Q6/Q7/Q8/Q9 中的大部分。
- **问题**：不解决 Q1/Q2/Q3/Q5/Q10 —— 平包继续膨胀（`data/project.go` 已 34 KB）、`main.go` 327 行继续承担全部装配、共享内核缺位使未来任何分包都会撞循环依赖、契约仍靠人工同步。文档修正后仍会随下一次迭代再次漂移（go-wind-admin 的 `docs/backend_project_struct.md` 就是先例）。
- **结论：不作为独立方案，但其全部动作吸收为决策的 P0 阶段。**

### 方案 C：模块化单体 —— 层内按域分包 + 共享内核 + 装配分域注册 + 契约机器校验（**采纳**）

保持 `biz/data/service` 三层与技术层一级目录不变，在每层内部按业务域建二级子包；抽出 `biz/shared` 共享内核；`cmd/planstar` 拆出 `wiring.go` 并按域注册；契约、文档、依赖方向、目录阈值全部纳入 CI。

- **优点**：
  1. 完全满足 C1–C8，运行时行为零变化，改动是「包路径 + import 重写」，可用编译器兜底。
  2. 正面吸收 go-wind 的 5 条可迁移实践（手写 DI 分层小节 + LIFO cleanup、`register:*` 幂等登记、契约按域、规范单一权威、Makefile 分层门禁），同时避开其 5 条反面（平包、倒置半途而废、无 CI 门禁、口号式 BFF、构造期副作用）。
  3. 保留 planstar 相对 go-wind 的**优势**：`biz` 纯 Go、18 个 repo 端口、30 个表驱动单测、独立 Registry 的 21 项指标、`/readyz` 五探针 —— 这些不因分包受损。
  4. 域边界一旦物理化，未来若确需拆服务，可按域整体迁出并复用同一批端口定义（P3 接缝），而不必先做大规模重构。
- **代价/风险**：一次性 import 路径重写影响 103 个文件；`biz` 拆子包后 `Fen/FmtDT`、全局错误、`DomainEvent` 必须先下沉到 `biz/shared`，否则循环依赖；跨域调用需要明确的编排规则；Git 历史与 CI/部署脚本路径需同步。
- **结论：采纳。** 详细目录树、包边界规则、迁移顺序与阈值见 `ADR-0002`。

### 方案 D：全量 proto/buf 契约化（在 C 之前或同时）

为 55+ 端点补 proto、引入 `buf generate`、多端代码生成与 breaking 检查。

- **优点**：强类型契约、机器可校验、多端 SDK 自动生成；go-wind 的 `reserved` 与 `breaking: FILE` 是成熟实践。
- **致命问题**：违反 C4 与 C1；会改变裸 JSON/Cookie/字段别名与错误体；go-wind-cms 实证维护成本（209 proto、约 690 生成文件、6 插件、8 份多端模板、双版本 `buf.yaml` 并存、`clean:true` 清空 GraphQL 产物的坑）；planstar 只有一套 Go 后端 + 一套 Flutter 客户端，无多语言/跨团队契约需求。
- **结论：本阶段否决。** 吸收其「契约可机器校验」的**目标**，用轻量手段实现：保持 `service.go` 路由表为事实源，增加条数/集合双向断言与错误码表一致性检查，可选生成一个只读 `openapi.yaml` 供 `site/` 与前端消费（不改运行时）。

### 方案 E：按域做一级目录 `internal/<domain>/{biz,data,service}`

- **优点**：域内聚最强；未来拆服务时可整目录搬走；贴近 go-wind 的 `app/<domain>/service/` 形态。
- **问题**：
  1. 与 `agents.md` 固化的「DTO→service→DO→biz→DO→data→PO」三层依赖倒置表达冲突；`biz` 与 `data` 同处一个域包目录后，「biz 纯 Go」不再能用一个 `go list -deps` 断言覆盖（需逐域配置）。
  2. 跨域共享的 DO/错误/事件/单位换算仍需抽 common 包，实际复杂度和 C 相当，但多出 7×3=21 个一级包。
  3. `data` 层的 `*Data`（10 连接池、Redis、ES）与 `service` 层的统一 `fail()` 错误映射、`OpsAuth`、中间件链是全局横切，按域一级分包会迫使它们被复制或反向依赖。
  4. 违反「最小概念变更」：团队已习惯 `internal/biz|data|service`，改造认知成本高于收益。
- **结论：本阶段否决**，作为 P3 之后若真要拆进程时的可选演进方向记录在案。

### 取舍矩阵

| 维度（权重） | A 微服务化 | B 只补文档 | **C 模块化单体** | D 全量 proto | E 域一级目录 |
|---|---|---|---|---|---|
| 满足硬约束 C1–C8 | ✗ | ✓ | **✓** | ✗ | △ |
| 解决 Q1/Q2/Q3 结构债 | △ | ✗ | **✓** | ✗ | ✓ |
| 解决 Q4–Q10 治理债 | △ | ✓ | **✓** | ✓ | △ |
| 运行时回归风险 | 高 | 无 | **低（编译期兜底）** | 高 | 中 |
| 实施成本 | 极高 | 低 | **中** | 高 | 中高 |
| 依据可靠性 | ✗（推演） | ✓ | **✓（一手代码）** | ✓ | ✓ |
| 为未来拆分留接缝 | ✓ | ✗ | **✓** | ✓ | ✓ |

---

## 5. 最终决策（Decision）

### D1 架构形态：模块化单体，不拆进程

保留单 Kratos App、三个 `transport.Server`、11 个逻辑库同实例、手写 `ServeMux`、无 proto/Wire。**业务域边界在进程内用 Go 包表达，不用网络边界表达**。理由见方案 A/E 否决项与 C2/C3/C4。

### D2 目录结构：技术层为一级目录，层内按业务域分包

```
internal/{biz,data,service}/<domain>/     # 域子包
internal/biz/shared/                      # 共享内核（DO/错误/事件/单位/分页）
internal/data/shared/                     # 连接池与驱动错误映射 adapter
internal/platform/                        # 横切技术能力
internal/server/                          # 仅 transport 与中间件
cmd/planstar_backend/{main.go, wiring.go}          # 装配拆分
```

**域划分（7 个，与 11 个逻辑库的映射在 ADR-0002 给出）**：`identity` / `catalog` / `project` / `trade` / `wallet` / `community` / `ops`。
理由：这 7 个域与现有 14 个 usecase、18 个 repo 端口、55+ 端点能一一归并，且与产品两段闭环（企划立项前投票 / 立项后出资）吻合，不引入新的业务概念。

### D3 装配治理：`main.go` 瘦身为启动壳，`wiring.go` 按域注册

- `main.go` 只保留 `flag.Parse`、日志、`kratos.New`/`app.Run`，目标 ≤ 80 行。
- `wiring.go` 采用 go-wind 的**分层小节 + `cleanups []func()` LIFO rollback**，但把每个域收敛为独立 `Register<Domain>(deps) (handlers, cleanup, error)`，避免复刻 439 行 / 80+ 实参的巨型 wiring。
- 引入 `register:*` 锚点注释与幂等登记检查（可先做 CI grep 断言，脚本化放到 P2），并把「漏登记 = 编译错误」作为设计前提。
- **明确禁止**把启动期副作用塞进构造函数：ES `RunSync`、指标注册、`readiness.MarkStarted` 等一律走显式 `BeforeStart`/`AfterStart`/`BeforeStop` 钩子（针对 go-wind-admin 的反面教材）。

### D4 ADR 台账与参考材料治理

- `docs/ADR/` 从此**只放 ADR 格式文档**，新增 `README.md` 索引（编号 / 标题 / 状态 / 日期 / 分类 / 关联）。
- `docs/ADR/planstar_arch_optimize.md` **重定位为研究材料**（建议 `docs/research/go-wind-cms-split-proposal.md`），并在文首标注「**设计推演，非实战复盘；技术栈与 `.archive` 快照有出入；收益数字未经测量**」；在索引中列为「参考材料·不作为决策依据」。本阶段仅提出建议，不移动文件（C9）。
- 根目录 4 份 `*NOTES.md` 与 `PLANSTAR_DEEP_STUDY.md` 归入「归档/研究」类，在 `docs/README.md` 的有效文档索引中显式标注**不得作为实现或排障依据**（对齐 `docs/README.md` 已有约定）。
- 后续所有架构决策必须落 ADR 编号文件，`docs/README.md` 的「失真文档先 Deprecated 再迁 legacy」流程保持不变。

### D5 契约与规范治理：轻量、机器可校验，不引入生成链

- `service.go` 路由表仍是唯一事实源；`routes_test.go` 增加**条数断言**与「尾斜杠别名只允许 1 条」，修掉 57 vs 56/55 的漂移不报警问题。
- 建立单一错误码表（Go 常量 / 生成式表格），`service.fail()` 与 `server/admin.go` 第二套映射共同引用，CI 校验 `API_CONTRACT.md` 的码表与之逐字一致；补齐 `ADMIN_CONTRACT.md` 缺的 4 条运营写路由。
- 修正 `agents.md` 的 5 条契约怪癖（登录失败 401 JSON、发码 429 例外、枚举硬编码白名单等），并给每条加**可执行的 grep 断言**，防止再次漂移。
- 可选（P1）：生成只读 `docs/api/openapi.yaml` 供 `site/` 与前端消费，运行时不读取、不改变 handler。

### D6 依赖方向与目录阈值的 CI 门禁

- 新增必过 job：`go build ./... && go vet ./... && golangci-lint run && go test -race ./...`（**针对 go-wind-admin「规范写全、机器不拦」的缺陷**）。
- 依赖断言：`go list -deps` 检查 `biz/*` 不含 `data/*`/`net/http`/`database/sql`、`service/*` 不含 `data/*`、`platform/*` 不含 `biz/*`。
- 目录阈值断言：单包 > 40 个 `.go`、单文件 > 30 KB 或 > 800 行、装配函数 > 50 实参、wiring 文件 > 300 行 → CI 失败。
- 修 `ci.yml` 的 `admin/**` → `planstar_frontend/admin/**`，并把 `docs/ADR/**` 纳入 paths；补文档一致性检查（借鉴 go-wind-admin 的 `scripts/check-readme-parity.mjs` 思路，但必须与 Go 门禁同属必过 job）。**该修正依赖 Q11 的目录重排先落地**：若 `admin/` 仍是 git 索引中的已跟踪路径而 `planstar_frontend/` 仍是未跟踪，直接改 paths 会让 `admin-go` / `admin-console` 两个 job 彻底失去触发源；必须「重排提交 + CI 路径修正」同批完成。
- `Makefile` 固定 `gen` / `lint` / `vet` / `test` / `cover` / `check` 同名 target，并修正 `make vet` 类空跑陷阱（用 `go vet ./...`）。

### D7 空目录与多口径统一（P0 阶段完成）

8 个 0 文件目录：要么补齐内容（`deploy/{loki,promtail,otel}` 与 compose 服务定义），要么删除（`backend/`、`site/assets/`、`site/openapi/`、`docs/legacy/`），并清理指向它们的死链。11 库口径在 `data.go`、`docs/sql/`、`docs/architecture.md`、`deploy/backup.sh`、`22_create_app_user.sh`、`preflight.sh` 中统一为「9 基线 + `shop` + `admin` = 11 运行时库，`data` 层连 10 个」。

**目录重排收尾（Q11）**：先完成 `admin/` → `planstar_frontend/admin/` 的提交（消除 `D admin/...` 与 `?? planstar_frontend/` 并存状态），再一次性核对 `.github/workflows/*`、`planstar_backend/deploy.sh`、`release.yml` 的 `APP_DIR` 相关路径、`docs/README.md` 有效索引、`planstar_frontend/admin/` 自身文档中的 `admin/...` 引用；`.archive/`、`.agents/` 与新增研究材料同步纳入 `.gitignore` 或文档索引，避免持续以未跟踪状态存在。

### D8 分阶段落地顺序（本阶段不执行）

| 阶段 | 内容 | 前置条件 | 验收 |
|---|---|---|---|
| **P0 基线治理** | D4 + D5 + D6 + D7：文档口径、ADR 台账、CI 门禁、空目录、11 库口径、契约测试增强；**并收尾 Q11 的目录重排** | 先完成 `admin/` → `planstar_frontend/admin/` 重排提交（不改 Go 包路径） | `go build/vet/test -race` 全绿；CI 全 job 通过；契约条数断言通过；`git status` 不再出现 `D admin/...` 与 `?? planstar_frontend/` 并存 |
| **P1 抽共享内核 + 装配拆分** | `biz/shared`（DO/错误/事件/`Fen`/分页）、`data/shared`（`*Data`/`mapDriverError`/`rowQuerier`）、`platform/` 归位；`QuoteTTLFromEnv` 改 conf 注入；`main.go` 瘦身 + `wiring.go` 分层小节 + LIFO cleanup | P0 通过；先跑 `go list -deps` 基线快照 | 依赖断言全绿；`/healthz`/`/readyz`/三 server 行为不变；`main.go ≤ 80 行` |
| **P2 层内按域分包** | 7 个域在 `biz/data/service` 内落地；`service/{api,admin,events}` 拆分；跨域调用走消费方端口 + 装配层适配器 | P1 通过且无循环依赖 | 阈值断言全绿；30 个单测 + `test-integration` 全绿；路由集合与 P0 快照逐字一致 |
| **P3 拆分接缝固化** | 域间禁止直接 import 对方 `data`；按域 CI 路径、指标 label、文档归属；输出「如需拆服务」的候选清单 | P2 稳定一个迭代 | 每个域可独立 `go test`；端口清单完整 |

> 每阶段独立提交、独立回滚；任一阶段 `go build ./...` 不通过即停止，不进入下一阶段。

---

## 6. 影响（Consequences）

### 6.1 正面

- **结构可读性**：7 个业务域在目录树上自解释；新人定位「订单相关代码」从「在 19 个平铺文件里 grep」变成「进 `biz/trade` + `data/trade`」。
- **可检查性**：分层依赖与目录阈值从「写在 AGENTS.md 里靠自觉」变成 CI 必过断言，直接解决 go-wind-admin 式的治理幻觉。
- **契约安全**：路由条数、错误码、`agents.md` 口径三者机器校验，消除 57 vs 56/55、登录失败 5xx、枚举来源等已发现的漂移。
- **装配可扩展**：新增用例/端点不再修改 327 行主干；LIFO cleanup 让启动失败与优雅停机共用一条路径，降低资源泄漏风险。
- **未来可选性**：域边界 + 端口清单 = 低成本的未来拆分接缝，但**不现在付分布式代价**（对齐复盘文章 §7「拒绝两个极端」中唯一仍然成立的思想）。
- **决策可追溯**：`docs/ADR/` 变为真正台账；把 43 KB 推演文章标注为研究材料，消除最大的幻觉源。

### 6.2 负面 / 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| 103 个文件的 import 路径重写 | Git 历史/ blame 断裂、review 量大 | 按域分批提交；每批只动一个域并立即跑全量测试；`goimports` 统一处理；在 ADR 记录映射表 |
| `biz` 拆包触发循环依赖 | 编译失败，P2 停滞 | **P1 必须先完成共享内核**；`biz/shared` 单向被依赖；跨域协作只允许「消费方定义端口 + 装配层注入适配器」 |
| 分包后跨库事务被误用 | 一致性回退（C3） | 在 `data/shared` 集中提供事务 helper 与「跨库仍用单 `*sql.Tx` + 限定名」的注释与 lint 提示；`ops_integration_test.go` 扩展为回归护栏 |
| `platform/` 归位引入反向依赖 | 分层破坏 | `platform` 不 import `biz`；当前 `observability` 被 `data/es_catalog.go` 使用属正向，装配时通过参数注入而非全局单例 |
| CI 新增门禁导致既有 PR 被拦 | 短期合入摩擦 | P0 先以「只报告不阻断」运行一个迭代，再转必过 |
| 文档与目录同步遗漏 | 产生新死链 | `docs/README.md` 索引 + `scripts/check-*-parity` 纳入 CI；治理后复核所有指向 `docs/legacy/`、`admin/` 的链接 |
| `config.lite.yaml` 等既有缺陷被治理动作掩盖 | 生产启动失败仍在 | 明确列为**独立功能性 ADR**（见 §7），不在本 ADR 内顺手修，避免决策范围膨胀 |

### 6.3 中性

- `docs/ADR/` 从「模板 + 一篇博客」变为正式决策台账，后续任何架构改动都需新增 ADR，流程成本上升但可追溯。
- `.archive/go-wind/` 被正式定位为**只读参考快照**（不参与构建、不作为事实源之外的依据）；根目录归档笔记与研究材料需统一标注。
- 7 个域的划分本身是一次业务概念固化，会成为后续需求归属与 code owner 划分的依据。
- 本 ADR 不改变任何运行时行为，因此**不需要**灰度发布、数据迁移或客户端协同（C1）。

---

## 7. 验证（Verification）

以下命令在实施各阶段后须全部通过；本 ADR 阶段仅用于确认「设计可验证」，不执行代码改动。

```bash
# G7 基线（P0–P3 每阶段必跑）
cd planstar_backend
go build ./...
go vet ./...
go test -race ./...
make test-integration                 # 未设 PLANSTAR_TEST_MYSQL_DSN 时自动 skip

# G3 依赖方向断言（P1 起）
go list -deps ./internal/biz/...     | grep -E 'planstar_backend/internal/(data|service)|net/http|database/sql' && exit 1 || true
go list -deps ./internal/service/... | grep 'planstar_backend/internal/data' && exit 1 || true
go list -deps ./internal/platform/...| grep 'planstar_backend/internal/biz' && exit 1 || true

# G4 契约一致性（P0 起）
go test ./internal/server -run 'TestContract|TestRoutesMatchContract' -count=1 -v
# 断言：路由集合 == API_CONTRACT.md；条数 == 57；尾斜杠别名恰好 1 条
# 断言：错误码表 == service.fail() == admin.go 映射（含 REFUND_NOT_ALLOWED 409/400 双语义）
# 断言：httpx.Plain 零调用；session.SetCookie 只写 Set-Cookie，不写 psSession 响应头
grep -rn 'httpx.Plain' --include='*.go' . | wc -l        # 期望 1（仅定义处）

# G1/G2 目录阈值（P2 起，建议脚本化）
# 任一包 *.go > 40 / 单文件 > 30KB 或 > 800 行 / wiring 实参 > 50 / main.go > 80 行 => 失败

# G6 空目录与口径（P0）
# 8 个 0 文件目录已填充或删除；11 库口径在以下位置逐字一致：
grep -rn 'planstar_shop\|planstar_admin\|domains' \
  internal/data/data.go docs/sql/ docs/architecture.md deploy/backup.sh sql/22_create_app_user.sh
grep -n 'admin/\*\*' .github/workflows/ci.yml            # 期望为 planstar_frontend/admin/**

# G5 ADR 台账
# docs/ADR/ 内每个 .md（除模板与 README）均含「状态/分类/优先级/日期/背景/决策/后果/验证/关联」
# docs/ADR/README.md 索引条目数 == ADR 文件数；研究材料已标注「设计推演」
```

**回滚判据**：任一断言失败即回滚到该阶段起点（P0–P3 各为独立提交）。**业务行为回归护栏**：`scripts/smoke-local.sh` 6 项 + `/readyz` 五探针输出与治理前快照逐字比对。

---

## 8. 关联（Links）

- 代码: `planstar_backend/cmd/planstar_backend/main.go#L45-L301`、`planstar_backend/internal/service/service.go#L71-L224`、`planstar_backend/internal/data/data.go#L45-L49`、`planstar_backend/internal/biz/models.go`、`planstar_backend/internal/server/routes_test.go`
- 文档: `docs/ADR/ADR-NNNN.md`（模板）、`docs/ADR/planstar_arch_optimize.md`（参考材料，本 ADR §1.1 已复核）、`docs/api/API_CONTRACT.md#L5-L11`、`docs/api/ADMIN_CONTRACT.md`、`docs/architecture.md#§10`、`docs/README.md`、`agents.md`、`PLANSTAR_DEEP_STUDY.md`（现状审计基线）
- 参考快照: `.archive/go-wind/go-wind-cms/backend/{AGENTS.md,app/core/service/cmd/server/wiring.go,api/protos/}`、`.archive/go-wind/go-wind-admin/{CONTRIBUTING.md,backend/{AGENTS.md,Makefile,app.mk,.golangci.yml}}`（**只读参考，不参与构建**）
- 关联 ADR: `ADR-0002`（目录结构优化与分阶段迁移方案，本决策的落地细化）
- 建议另行立项的独立 ADR（不在本 ADR 范围）：
  - `ADR-0003` 生产配置不变量与 `config.lite.yaml` 可启动性修复（Q9 / C6）
  - `ADR-0004` MQ 消费幂等、DLQ 治理与事务性 Outbox（功能性缺口，见 `PLANSTAR_DEEP_STUDY.md` §13）
  - `ADR-0005` 错误映射集中化与 `dict_enum` 运行时化/废弃决策（Q10 / Q9）

---

## 9. 实施状态（2026-11-19 首次执行）

> 本 ADR 的决策内容不因执行而改变；本节只记录**已落地**与**待续**项，便于接续。

### 9.1 已落地：P0 基线治理（不改 Go 包路径）

| 决策项 | 落地内容 |
|---|---|
| D4 | 新增 `docs/ADR/README.md` 台账索引；`planstar_arch_optimize.md` 文首加「设计推演、非实战复盘」性质标注；归档笔记定位写入索引 |
| D5 | `agents.md` 的 5 条契约怪癖按代码修正（登录失败 401 JSON、发码 429 例外、枚举为硬编码白名单、只写 HttpOnly Cookie、版本号禁止统一）；`API_CONTRACT.md` 条数 57/56、附录 56、登录端点清单、`psSession` 响应头；`ADMIN_CONTRACT.md` fail-closed 修正 + 补齐 4 条运营写路由与审计 action |
| D5 | 新增 `internal/pkg/errcode` 单一错误码表（21 个码）；`service`/`server`/`observability`/`httpx` 全部改为引用常量；`API_CONTRACT.md` 补 `SERVICE_UNAVAILABLE` |
| D5 | `routes_test.go` 新增 `TestRouteInventory`（57 条 / 56 唯一端点 / 恰好 1 条尾斜杠别名）与 `TestContractDeclaresSameInventory` |
| D6 | 新增 `internal/governance` 治理断言包（依赖方向、传递依赖、错误码单源、ADR 台账、契约怪癖、目录阈值报告模式）；`Makefile` 新增 `governance` / `lint` target，`check` 纳入治理断言；`.github/workflows/ci.yml` 新增必过 `governance` job + 报告式 lint/阈值 step |
| D6 / Q11 | CI 全部 `admin/` 路径统一为 `planstar_frontend/admin/`（paths、working-directory、cache-dependency-path），文件头标注「需与目录重排提交同批落地」 |
| D7 | 删除 `backend/`、`site/assets/`、`site/openapi/` 三个无引用空目录；`docs/legacy/` 与 `deploy/{loki,promtail,otel}/` 补 README 说明；清除 `docs/README.md`、`docs/architecture.md` 的 legacy 死链 |
| D7 | 11 库口径统一到 `docs/architecture.md`、`operations.md`（3 处）、`deployment.md`（2 处）、`docs/README.md`、`agents.md` |

### 9.2 已落地：P1 安全子集（兼容层渐进，零运行时行为变化）

| 内容 | 说明 |
|---|---|
| `internal/biz/shared` 共享内核 | 18 个哨兵错误 + `BlacklistBlockedError`、领域事件与 `EventPublisher` 端口、`Fen` / `FmtDT` / `FmtDTp` / `BirthTs` / `MaskMobile`。**`biz` 根包保留别名与包装兼容层**，现有 100+ 文件零 import 改写，为 P2 分包消除循环依赖 |
| `internal/pkg/token` | 纯算法令牌包（无 `net/http`）；`session` 委托它并保留 `NewToken` / `SessionTTL` / `DeviceTTL` API；`biz/auth.go` 改用 `token.New()` |
| 消费方端口 | `biz/auth.go` 自行声明 `SMSSender` 端口与 noop 实现，不再 import `internal/pkg/notify`（该包传递依赖阿里云 SDK 与 `net/http`）；`main.go` 传入的 `notify.NewSMS` 结果结构化满足新接口，装配无需改动 |
| 实测 | `go list -deps ./internal/biz` 中 `net/http` / `database/sql` / redis / amqp / elastic **0 命中**；`TestBizHasNoTransitiveInfraDependency` 已固化为 CI 断言 |
| 报价 TTL 注入 | `conf.QuoteTTL()` 读 `PLANSTAR_QUOTE_TTL_SECONDS`；`biz` 删除 `QuoteTTLFromEnv` 与 `os`/`strconv` 依赖，改为 `DefaultQuoteTTL` 常量 + `NewOrderUsecase(..., ttl ...time.Duration)` 可选注入；`data/order.go` 回退 TTL 改用常量；新增 `conf/quote_test.go`、`biz/quote_ttl_test.go` |

### 9.3 待续（下一阶段，需另开变更）

1. **Q11 目录重排提交**：工作区仍有 `D admin/...` 与 `?? planstar_frontend/`，需用户执行 `git add -A admin frontend && git commit`（涉及 git 操作，本阶段未代为执行）。CI 路径已按目标布局修正，重排提交前 `admin-go` / `admin-console` 不会被触发。
2. **D3 装配拆分**：`main.go` 仍 354 行（阈值 80），需拆 `wiring.go` 分层小节 + 按域 `RegisterXxx` + LIFO cleanup + 显式生命周期钩子。
3. **P2 七域分包**：按 `ADR-0002` §3 映射表逐域迁移 `biz` / `data` / `service`，并删除 `biz` 根包兼容层；完成后把 `PLANSTAR_GOVERNANCE_STRICT` 置 1、去掉 CI 中 lint/阈值的 `continue-on-error`。
4. **`platform/` 归位**：`conf`/`observability`/`mq`/`ops`/`pkg` 迁到 `internal/platform/`，同步 import 与 CI/部署路径。
5. **`.golangci.yml`**：目前 CI lint 为报告式，需补配置后转必过。
6. 独立功能性 ADR-0003/0004/0005（见 §8）不在本轮范围。

### 9.4 验证结果（2026-11-19 实测）

| 命令 | 结果 |
|---|---|
| `gofmt -l .` | 空 |
| `go build ./...` | 退出 0 |
| `go vet ./...` | 退出 0 |
| `go test ./... -count=1` | 18 个包全绿（含新增 `internal/governance`、`internal/biz/shared`、`internal/pkg/token`、`internal/pkg/errcode`） |
| `go list -deps ./internal/biz \| grep -E '^(net/http\|database/sql\|...)'` | 0 命中 |
| `go test -race ./...` | **本机无法执行**：`-race requires cgo` 且 `gcc not found in %PATH%`（`PRODUCTION_DELIVERY.md` 已记录此环境限制）；CI 的 ubuntu runner 会执行 `go test -race ./...` |

**行为不变性**：57 条路由与 `METHOD /path` 集合、统一错误体与 21 个错误码、`Cookie: psSession`、金额「DB 元 / API 分」、分页参数、裸 JSON 成功响应、`/healthz` `/readyz` `/metrics` 语义，均未改变；`TestRouteInventory`、`TestRoutesMatchContract`、`TestWriteAdminErrorMapping`、`TestOpsAuth*` 等既有用例全部通过。

---

## 9.5 第二次执行：目录改名、装配拆分、门禁与 Docker 验证（2026-11-19）

### 目录改名（用户要求）

- `frontend/` → `planstar_frontend/`，`planstar/` → `planstar_backend/`。
- 两个 Go module 名 `planstar` 与 `planstar/admin` **保持不变**：module path 与物理目录解耦，因此 100+ Go 文件的 import 无需改写；实测两个 module 均 `go build ./...` 通过。
- CI / Release / `deploy.sh` / `preflight.sh` / `backup.sh` / `Makefile` / `docs/**` 中的路径全量同步；compose 内 `../docs/sql`、`./sql`、`./configs`、`build.context: .` 均为相对自身路径，无需改动。
- 刻意保护未替换：`@planstar/*`（pnpm 包名）、`/v1/planstar/*`（admin HTTP 路径）、`module planstar/admin`、`planstar_*` 库名、`planstar-api` 服务名、`PLANSTAR_*`、`planstar.exchange`、Prometheus namespace。批量替换曾误伤前两类，已还原并做残留扫描（0 命中）。

### P1 收尾

- `cmd/planstar/main.go` 由 354 行拆为 `main.go`(26) + `wiring.go`(251) + `probes.go`(63)；`run() error` 承载全部装配，资源清理仍按原 LIFO 顺序（后台任务 → MQ → Data → tracing），`main()` 只负责 `flag.Parse` 与退出码。
- 显式保留 `BeforeStart` / `AfterStart` / `BeforeStop` / `AfterStop` 钩子，未把启动副作用塞进构造函数（规避 go-wind-admin 的反面教材）。

### P2 首批落地

- `service/event.go` → `internal/service/events/handler.go`：`CatalogIndexer` 端口声明在消费方，`wiring.go` 直接注入 `data.CatalogRepo`，events 不 import `data` / `amqp`。
- `internal/data/project.go`（34 KB / 1121 行，唯一超阈值的文件）按读模型 / 写模型 / 辅助函数拆成 3 个同 package 文件，零语义变化，全部 < 15 KB。

### P3 门禁固化

- `TestLayoutThresholds` 由「报告式」改为**默认严格阻断**（`PLANSTAR_GOVERNANCE_STRICT=0` 才降级），CI 去掉 `continue-on-error`；当前基线全部达标。
- 修正 `scripts/smoke-local.{sh,ps1}` 中与 `OpsAuth` fail-closed 矛盾的提示文案。

### 本地 Docker 验证（PASS）

`docker-compose.local.yml` 三容器启动成功；`/healthz` 200、`/readyz` 200（mysql/redis ok，es/mq/consul skipped 符合低配档位）、`/v1/categories` 200、`/v1/activities` 200、发码 200、`/metrics` 无令牌 401、`scripts/smoke-local.ps1` 退出 0。过程中处置了 BuildKit 镜像拉取失败（改用内置 builder + `--no-build`）与旧数据卷口令不一致（`down -v` 重建）。完整记录见 `docs/local-deploy-verification.md`。

### 仍待续（下一批次）

- **P2 剩余 6 个业务域的物理分包**：`biz`/`data`/`service` 三单包共 100+ 文件共享包内私有符号（`s.fail`、`s.requireUser`、`r.d`、`fen()`、各 DO），一次性拆包会使跨包引用失效，必须逐域抽取端口、下沉 DO、重写 import 并分批跑全量测试。已完成的 `service/events` 与 `data/project.go` 拆分验证了迁移路径；按 `ADR-0002` §3 映射表逐批推进。
- `platform/` 归位、`.golangci.yml`（CI lint 目前报告式）、按域 CI 路径与指标 label。
- 独立功能性 ADR-0003/0004/0005（见 §8）。