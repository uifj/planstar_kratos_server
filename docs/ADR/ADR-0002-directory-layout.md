# ADR-0002: 目录结构优化方案 —— 技术层为一级目录、层内按业务域分包

- **状态**: ACCEPTED（设计已选定；本阶段不移动任何文件）
- **分类**: ARC（关联 DEBT / TST）
- **优先级**: P1
- **影响维度**: 认知障碍 / 上下文膨胀 / 生产功能
- **日期**: 2026-11-19
- **来源**: `ADR-0001` 决策 D2/D3/D6 的落地细化；全仓审计 `PLANSTAR_DEEP_STUDY.md`；`.archive/go-wind/` 目录实践核对
- **修复 commit**: `e514842`（P2 首批：`service/events` 子包与 `data/project.go` 拆分，2026-11-19）

---

## 0. 本 ADR 回答什么

`ADR-0001` 决定「模块化单体，不拆进程」。本 ADR 把该决策翻译为**可执行的目录结构**：现状树 → 目标树 → 逐文件迁移映射 → 包边界与依赖矩阵 → 循环依赖的拆解办法 → 分阶段顺序与守卫。

**一句话原则**：**技术层保持一级目录，业务域作为二级包；横切能力收敛到 `platform/`；共享内核先行，域包后拆。**

> 这个原则直接来自 go-wind 的反面实证：`go-wind-cms/core/service/internal/` 的 `data` 114 个平铺 .go、`service` 104 个平铺 .go；`go-wind-admin` 的 `internal/data` ≈200、`internal/service` ≈179 全平铺，领域信息只编码在文件名前缀里。两者都在 `api/protos/<域>/` 做了契约按域分包，**唯独实现层没跟上**——即「契约按域、实现按层」的失衡。planstar 反过来做：不引入 proto，但让实现层也按域分包。

---

## 1. 现状目录树与量化基线

```
planstar_kratos_server/                          # 仓库根（非 Go module，仅容纳多子项目）
├── AGENTS.md / agents.md / CLAUDE.md / README.md
├── MIDDLEWARE.md / CODEGEN_VERIFICATION.md
├── KRATOS_STUDY_NOTES.md / KRATOS_SHOP_STUDY_NOTES.md   # 归档模板研究，非本服务
├── PLANSTAR_DEEP_STUDY.md                     # 全仓学习报告（本次新增，根目录）
├── LICENSE                                    # PolyForm Noncommercial 1.0.0
├── .archive/go-wind/{go-wind-admin,go-wind-cms}/        # 只读参考快照
├── .github/workflows/{ci,release,pages}.yml + dependabot.yml
├── backend/                                   # ← 0 文件
├── docs/
│   ├── ADR/                                   # ADR-NNNN.md + planstar_arch_optimize.md（非 ADR）
│   ├── api/{API_CONTRACT,ADMIN_CONTRACT}.md
│   ├── sql/  (21 .sql + history/ + optional/)
│   ├── product/POSITIONING.md
│   ├── legacy/                                # ← 0 文件，但多处文档指向它
│   └── architecture.md / deployment.md / operations.md / README.md / ...
├── planstar_frontend/
│   ├── admin/                                 # 独立 Go module + Umi 控制台
│   └── planstar_web/                          # pnpm + Next.js 15
├── planstar_backend/                                  # ★ 唯一业务后端 Go module（module planstar）
│   ├── cmd/planstar_backend/main.go                   # 327 行，全部装配
│   ├── internal/
│   │   ├── biz/        29 .go  (19 实现 + 10 测试)   ← 平铺，无子包
│   │   ├── data/       19 .go  (17 实现 + 2 测试)    ← 平铺，无子包
│   │   ├── service/    18 .go  (16 实现 + 2 测试)    ← 平铺，无子包
│   │   ├── server/     10 .go  (5 实现 + 5 测试)
│   │   ├── pkg/        15 .go  (captcha/secret/session/httpx/notify/storage)
│   │   ├── conf/        3 .go
│   │   ├── mq/          5 .go
│   │   ├── observability/ 1 .go (498 行 / 16 KB)
│   │   ├── ops/         2 .go
│   │   └── testutil/    1 .go
│   ├── configs/  deploy/  sql/  *.yml  Dockerfile  Makefile
├── scripts/{smoke-local.sh,smoke-local.ps1}
└── site/index.html  (+ assets/ 与 openapi/ 均 0 文件)
```

> **⚠ 上树是「意图布局」，不是当前 git 状态。** `git --no-pager status --short`（2026-11-19 实测）显示大量 `D admin/...`（含 `cmd/server/wire.go`、`wire_gen.go`、`internal/data/ent/...`、`console/`）与未跟踪的 `?? planstar_frontend/`，说明 `admin/` 正被迁往 `planstar_frontend/admin/` 但**尚未提交**；`.archive/`、`.agents/`、`PLANSTAR_DEEP_STUDY.md`、`docs/ADR/planstar_arch_optimize.md` 亦为未跟踪。
> 因此本 ADR 的目录方案**以重排完成后的布局为准**，实施前置条件是先提交该重排（见 `ADR-0001` Q11 / D7 / P0）。任何 `ci.yml` paths、部署脚本与文档链接的修正必须与重排同批完成，否则 `admin-go` / `admin-console` 两个 job 会失去触发源。
> 附带事实：迁出的 `planstar_frontend/admin/` 是**使用 Wire + Ent + proto 的另一个独立 Go module**（存在 `cmd/server/wire.go` 与 `wire_gen.go`），而 `planstar_backend/` 是手写装配、无 Wire/proto 的单体 —— 两个 module 的架构差异是硬边界，本 ADR 的目录方案**只作用于 `planstar_backend/`**。

**量化基线（实测）**：`internal/` 共 103 个 `.go`；最大文件 `data/project.go` 34,037 B、`conf/conf.go` 23,163 B、`biz/project.go` 20,239 B、`observability/observability.go` 16,076 B；`service/service.go:73-144` 单函数承载 57 条路由；`cmd/planstar_backend/main.go` 327 行单文件装配全部组件。

**已触发的拆分信号**（与 `ADR-0001` G1/G2 阈值对照）：单文件 > 30 KB（`data/project.go`）、装配文件 > 300 行（`main.go`）、单包承载 ≥ 7 个业务域（`biz`/`data`/`service` 均是）。

---

## 2. 目标目录树

### 2.1 仓库根（治理后）

```
planstar_kratos_server/
├── AGENTS.md                          # ★ 单一权威规范（修正 5 条契约怪癖 + 目录约定 + 依赖禁止项）
├── CLAUDE.md                          # 仅 `@AGENTS.md` 指针（对齐 go-wind 实践）
├── README.md                          # 仓库导航，指向 docs/README.md
├── LICENSE
├── .archive/                          # 只读参考快照；README 注明「不参与构建、不作为实现依据」
├── .github/workflows/                 # paths 修为 planstar_frontend/admin/**，补必过 Go 门禁
├── docs/
│   ├── README.md                      # 有效文档索引（含「失真→Deprecated→legacy」流程）
│   ├── ADR/
│   │   ├── README.md                  # ★ ADR 台账索引（编号/标题/状态/日期/分类）
│   │   ├── ADR-NNNN.md               # 模板
│   │   ├── ADR-0001-...md            # 架构治理总纲
│   │   └── ADR-0002-...md            # 本 ADR
│   ├── research/                      # ★ 研究/推演材料区（planstar_arch_optimize.md 建议迁入并标注）
│   ├── notes/                         # ★ 归档学习笔记（根目录 4 份 *NOTES.md + PLANSTAR_DEEP_STUDY.md）
│   ├── api/{API_CONTRACT.md, ADMIN_CONTRACT.md, openapi.yaml(可选只读)}
│   ├── sql/  product/  architecture.md  deployment.md  operations.md ...
│   └── legacy/                        # 要么填充真实废弃文档，要么删除并清理死链
├── planstar_frontend/{admin,planstar_web}/     # 保持，职责与 module 边界不变
├── planstar_backend/                          # ★ 唯一业务后端（详见 2.2）
├── scripts/                           # smoke + 新增 check-* 一致性脚本（P0 建议）
└── site/                              # 删除空的 assets/ 与 openapi/，或填充
```

**根层决策**：
- **根目录不再是「kratos-layout 模板残骸 + 多子项目 + 学习笔记」的混合物**：4 份上游 `*NOTES.md`、`CODEGEN_VERIFICATION.md`、`MIDDLEWARE.md` 与本次 `PLANSTAR_DEEP_STUDY.md` 统一归 `docs/notes/`（或 `.archive/`），并在 `docs/README.md` 显式标注「归档研究，不得作为实现/排障依据」。
- `backend/`（0 文件）、`docs/legacy/`（0 文件）、`site/assets/`、`site/openapi/`、`planstar_backend/deploy/{loki,promtail,otel}/`（0 文件）按 `ADR-0001` D7 处置。
- `.archive/go-wind/` 明确为**只读参考**：用于 ADR 引用证据，不参与任何构建、CI、部署与依赖解析。
- **先完成 `admin/` → `planstar_frontend/admin/` 重排提交**（见 §1 的 git 中间态告警），并把 CI paths、部署脚本、`docs/README.md` 索引中的旧 `admin/...` 引用同批修正。两个 Go module 的边界必须保持：`planstar_frontend/admin/` 用 Wire + Ent + proto，`planstar_backend/` 手写装配无生成链；本 ADR 的域分包方案**不作用于 `planstar_frontend/admin/`**，避免把两种装配范式混在同一套目录约定里。

### 2.2 `planstar_backend/`（核心改造）

```
planstar_backend/
├── cmd/planstar_backend/
│   ├── main.go                        # ★ ≤80 行：flag / 日志 / kratos.New / app.Run
│   └── wiring.go                      # ★ 分层小节 + 按域 RegisterXxx + cleanups LIFO rollback
├── internal/
│   ├── biz/                           # 纯 Go 领域层（一级目录保留）
│   │   ├── shared/                    # ★ 共享内核：DO 基础、错误、事件、单位、分页、端口公共类型
│   │   │   ├── money.go               #   Fen / yuan<->cent（原 biz/models.go 的导出助手）
│   │   │   ├── time.go                #   FmtDT / FmtDTp / BirthTs
│   │   │   ├── errors.go              #   18 个 sentinel errors（原 auth.go + errors.go + project_idea.go）
│   │   │   ├── event.go               #   DomainEvent / EventPublisher / NopEventPublisher / NewEvent
│   │   │   ├── paging.go              #   page/pageIndex 归一化、上限回落规则
│   │   │   └── do.go                  #   跨域共享 DO（AuthUser / LoginBean / TicketOrder 等）
│   │   ├── identity/                  # 认证、会话、用户资料、地址
│   │   ├── catalog/                   # Banner、活动目录、ES 读模型端口
│   │   ├── project/                   # 企划、创意、黑名单审核、用户侧举报
│   │   ├── trade/                     # 权威定价、订单、支付、退款
│   │   ├── wallet/                    # 钱包、签到、星币商城
│   │   ├── community/                 # 动态、消息、客服会话
│   │   └── ops/                       # 运营写、上下架、退款审核、举报处置
│   ├── data/                          # 端口实现（与 biz 域一一镜像）
│   │   ├── shared/                    # ★ *Data、10 连接池、Redis/ES 客户端、mapDriverError、rowQuerier、事务 helper
│   │   ├── identity/  catalog/  project/  trade/  wallet/  community/  ops/
│   │   └── es/                        # ES 索引/别名/同步（原 es_catalog.go，技术读模型独立成包）
│   ├── service/                       # HTTP 适配层（按「端」而非按域，见 §4.3 理由）
│   │   ├── api/                       # 8098 客户端：routes.go + 各域 handler + fail()
│   │   ├── admin/                     # 9108 运维：admin handler + OpsAuth + console（原 server/admin*.go 中的业务部分）
│   │   └── events/                    # MQ 事件分派（原 service/event.go）
│   ├── platform/                      # ★ 横切技术能力（原 conf/observability/mq/ops/pkg 归位）
│   │   ├── conf/                      #   配置结构、默认值、env 覆盖、生产不变量
│   │   ├── observability/             #   指标、OTLP tracing、trace 日志、中间件
│   │   ├── mq/                        #   Publisher / Consumer / DLX / trace 传播 / Health
│   │   ├── readiness/                 #   /healthz /readyz 引擎（原 internal/ops）
│   │   ├── httpx/                     #   JSON / Error / ReadJSON / 多别名解析（含 net/http）
│   │   ├── token/                     # ★ 纯令牌算法，无 net/http（从 pkg/session 拆出）
│   │   ├── cookie/                    # ★ Cookie/DeviceID 适配，含 net/http（从 pkg/session 拆出）
│   │   ├── captcha/  secret/          #   纯标准库实现（PNG / PBKDF2）
│   │   ├── notify/  storage/          #   SMS/Email/OSS 实现（接口定义在消费方 biz 域）
│   │   └── authz/                     #   OpsAuth（Bearer + CIDR + ops audit）
│   ├── server/                        # 仅 transport 组装：http.go / admin.go / middleware 链
│   └── testutil/                      # MySQL 集成测试 helper（保持）
├── configs/  deploy/  sql/  *.yml  Dockerfile  Makefile
└── go.mod                             # module planstar 不变
```

**保持不动**：`configs/` 四份配置、`deploy/` 告警与 Traefik 配置、`sql/` 增量脚本、四份 compose、`Dockerfile`、`go.mod` 的依赖清单、11 个逻辑库、55+ 端点的路径与响应。

---

## 3. 业务域划分与逐文件迁移映射

7 个域的划分依据：与产品两段闭环（立项前星推官投票 / 立项后出资交付）对齐，且能把 14 个 usecase、18 个 repo 端口、57 条路由、11 个库无歧义地归并。

| 域 | 主要端点 | 主要库 | `biz/` 原文件 → 目标 | `data/` 原文件 → 目标 | `service/` 原文件 → 目标 |
|---|---|---|---|---|---|
| **identity** | captcha / sendLoginCode / login / user / myaddress | `user`（+ `payment` 钱包读） | `auth.go`, `user.go` | `auth.go`, `auth_cache.go`, `user.go` | `auth.go`, `user.go` |
| **catalog** | banners / activities / activities/hot / activityDetail | `catalog`, `ticket`, `content` + ES | `catalog.go` | `catalog.go`（`es_catalog.go` → `data/es/`） | `catalog.go` |
| **project** | categories / projects / pledges / creators / rankings / projectIdeas / reports(POST) / reviews | `catalog`, `order`, `user` | `project.go`, `project_idea.go`, `moderation.go`, `report.go`(用户侧 `CreateReport`) | `project.go`, `project_idea.go`, `report.go`(用户侧) | `project.go`, `project_idea.go`, `idea_vote_limiter.go` |
| **trade** | unifiedOrder/confirm / makeOrder / unifiedPay / order/refund / order/list | `order`, `payment`, `ticket`, `catalog` | `quote.go`, `order.go`, `payment.go`, `errors.go`(订单错误→ `biz/shared/errors.go`) | `order.go`, `payment.go` | `order.go`, `payment.go`, `order_idempotency_test.go` |
| **wallet** | signIn / moneyaccount / shop/goods / redeem | `payment`, `marketing`, `shop` | `wallet.go`, `shop.go` | `wallet.go`, `shop.go` | `wallet.go`, `shop.go` |
| **community** | feed / tweets / userReply / messages / chat | `community`, `user` | `feed.go`, `message.go`, `chat.go` | `feed.go`, `message.go`, `chat.go` | `feed.go`, `message.go`, `chat.go` |
| **ops** | `/admin/*` 全部（9108） | `community`, `shop`, `payment`, `order`, `catalog` | `ops.go`, `report.go`(处置 `ReportUsecase`)，审核视图复用 `project.Moderation` 端口 | `ops.go`, `report.go`(处置) | `admin/`（原 `server/admin.go` 中的运营 handler 迁入 `service/admin/`） |

**跨域裁决（必须写清，否则 P2 会卡住）**：

1. `report.go` 一分为二：**用户提交举报**（`CreateReport`，双白名单 + 24h 去重）归 `biz/project/`；**运营处置举报单**（`ReportUsecase.List/Handle`）归 `biz/ops/`。共享 `Report` DO 放 `biz/shared/do.go`。
2. `moderation.go` 的**黑名单规则与 `CheckProjectBlacklist`** 归 `biz/project/`；`ReviewProject` 的 HTTP 入口在 `service/admin/`，通过 `project` 域暴露的 `Moderation` 端口调用，**`ops` 域不得直接 import `data/project`**。
3. `service/event.go` 的 `EventHandler` 归 `service/events/`，它依赖两个消费方端口：`community.Notify`（落站内通知）与 `catalog.CatalogIndexer`（ES 增量），由 `wiring.go` 注入；**不 import 任何 `data/*` 实现**（现状已成立）。
4. `CatalogReader` / `QuoteStore` 定义在 `biz/trade/`，但实现在 `data/trade/`（`OrderRepo` 同时实现三个端口）；`GetActivitySnapshot` 需要读 catalog/ticket 库，属于「**trade 域声明端口、data/trade 跨库实现**」，不因此把 `catalog` 域并进 `trade`。
5. `ProjectRepo` 的 18 个方法里有 `Moderation` / `SetPublishStatus`，属运营审核，仍归 `biz/project/`（聚合根在 project），由 `service/admin/` 通过 project usecase 调用。

**测试文件跟随实现同域迁移**（`biz` 10 个 `_test.go`、`service` 2 个、`data` 2 个），保持 colocated 约定不变；`internal/testutil/` 保持全局。

---

## 4. 包边界规则与依赖矩阵

### 4.1 允许的依赖方向

```
cmd/planstar  ──>  server / service / data / biz / platform   (唯一装配点，允许看到全部)
service/*     ──>  biz/* , biz/shared , platform/*            (禁止 data/*)
biz/<domain>  ──>  biz/shared , 标准库 , platform/{token,captcha,secret}(纯算法)  (禁止 data/*、service/*、net/http、database/sql)
biz/shared    ──>  仅标准库
data/<domain> ──>  biz/<domain>(实现其端口) , biz/shared , data/shared , platform/*(注入)
data/shared   ──>  conf , 驱动 SDK , biz/shared
platform/*    ──>  不得 import biz/*  （只被上层通过参数/接口使用）
server/*      ──>  service/* , platform/* , conf
```

### 4.2 循环依赖的关键拆解（P1 必须先完成）

现状 `biz` 是单 package，拆子包后有 4 处必然撞循环，必须先抽共享内核：

| 冲突点 | 现状 | 处置 |
|---|---|---|
| `data/*` 大量调用 `biz.Fen/FmtDT/BirthTs`（`biz/models.go` 明确注释「导出给 data 层使用」） | biz → 无、data → biz | 下沉到 `biz/shared/{money,time}.go`；`data/*` 只 import `biz/shared`，**不再 import 任何 `biz/<domain>` 的工具函数** |
| `biz/auth.go:21-27` 定义 7 个通用错误、`biz/errors.go` 6 个订单错误、`biz/project_idea.go` 5 个错误，`service.fail()` 与 `server/admin.go` 都要 `errors.Is` | 错误散在 3 个领域文件 | 统一收敛到 `biz/shared/errors.go`（18 sentinel + `BlacklistBlockedError`），域包只 import shared |
| `biz/event.go` 的 `DomainEvent`/`EventPublisher` 被 `biz/auth.go`、`biz/order.go`、`service/events/`、`platform/mq/` 共用 | 单包内无所谓，拆包后方向混乱 | 放 `biz/shared/event.go`；**`EventPublisher` 端口定义在 shared（消费方），`platform/mq/Publisher` 实现它**；`platform/mq` 不 import `biz/<domain>` |
| `notify.SMSSender` 接口现在定义在 `platform/notify`，却被 `biz.AuthUsecase` 依赖 | 领域依赖技术包 | 接口移到 `biz/identity/port.go`，`platform/notify` 只提供实现；由 `wiring.go` 注入。同理 `AuthCache` 端口保持 `biz/identity`，实现留 `data/identity` |

**额外净化（让「biz 纯 Go」断言真正成立）**：当前 `biz/auth.go` import `pkg/session`（该包含 `net/http` 的 `SetCookie/FromRequest`）与 `pkg/captcha`（`image/png`）。`go list -deps ./internal/biz/...` 会传递命中 `net/http`。P1 把 `pkg/session` 拆为：
- `platform/token/`：`NewToken()` 32B→64hex、`SessionTTL`、`SessionExpireDays` 常量，**零 `net/http`**，`biz/identity` 只依赖它；
- `platform/cookie/`：`SetCookie` / `FromRequest` / `DeviceIDWithOptions`，含 `net/http`，只被 `service/api` 与 `server` 使用。

`captcha`（纯 `image/png` + `crypto/rand`）与 `secret`（PBKDF2，纯标准库）可保留被 `biz` 直接引用，不破坏断言。

### 4.3 为什么 `service/` 按「端」拆而不是按域拆

- `service.go` 的 `fail()` 是**全局唯一错误映射**（18 分支），`Register(mux)` 是**全局唯一路由表**，二者按域切开会让「契约条数断言」和「错误码一致性」失去单一锚点。
- 三个端（8098 客户端 / 9108 运维 / MQ 消费者）的鉴权与中间件语义截然不同：`OpsAuth` fail-closed + CIDR + `ops audit`，客户端是 `psSession` Cookie，MQ 是 `amqp.Table` trace 传播。按端分包让安全边界在目录上可见。
- 域内聚由 `biz/<domain>` + `data/<domain>` 承担，`service/` 只做薄适配，符合 C5。

### 4.4 依赖规则的机器校验

```bash
go list -deps ./internal/biz/...     | grep -E 'planstar_backend/internal/(data|service|server)|net/http|database/sql|redis|amqp|elastic' && exit 1
go list -deps ./internal/service/... | grep 'planstar_backend/internal/data' && exit 1
go list -deps ./internal/platform/...| grep 'planstar_backend/internal/biz' && exit 1
```

`platform/mq` 实现 `biz/shared.EventPublisher` 属**实现方依赖接口定义方**，方向正确；若出现 `platform → biz/<domain>` 即为违规。

---

## 5. 装配点改造（`ADR-0001` D3 的细化）

现状 `main.go` 327 行一气呵成。目标：

```
main.go（≤80 行）
  flag.Parse → 日志 → conf.Load → kratos.New(opts...) → app.Run
  opts 中的 BeforeStart/AfterStart/BeforeStop/AfterStop 显式声明：
    BeforeStart:  readiness 闸门前的依赖自检
    AfterStart:   readiness.MarkStarted() + 启动 ES RunSync 后台 goroutine
    BeforeStop:   readiness.MarkStopping() + cancelApp()
    AfterStop:    （由 wiring 返回的 rollback 承担）

wiring.go
  // ── 一、平台能力 ──────────────────────  metrics / tracer / conf / mq / consul / sms / oss
  // ── 二、数据与共享内核 ────────────────  data/shared.NewData + 10 池 Ping
  // ── 三、按域注册 ──────────────────────
  //    RegisterIdentity(deps) / RegisterCatalog(deps) / RegisterProject(deps)
  //    RegisterTrade(deps) / RegisterWallet(deps) / RegisterCommunity(deps) / RegisterOps(deps)
  //    每个返回 (handlers, cleanup, error)，单个函数 ≤30 实参
  // ── 四、传输层 ────────────────────────  server.NewHTTPServer / NewAdminServer / mq.NewServer
  // ── 五、就绪与指标 ────────────────────  readiness + RegisterDBStats + biz metrics
  cleanups []func() + rollback()  // LIFO；启动失败与优雅停机共用
```

**借鉴与规避**：
- 借鉴 `go-wind-cms/app/core/service/cmd/server/wiring.go:41-49,438` 的分层小节 + `cleanups/rollback` LIFO，以及 `backend/tools/register/main.go` 的 `register:*` 锚点 + `skipIf` 幂等注入思想。
- **规避**其 439 行 / 80+ 实参的巨型 wiring：每个域一个 `RegisterXxx`，主干只做编排。
- **规避** `go-wind-admin/wiring_ent.go` 的构造期副作用（默认数据播种、`AttachEntityHooks`、`Resync` 靠构造顺序隐式生效）：ES `RunSync`、指标注册、`MarkStarted` 一律走显式生命周期钩子。
- 保持 `defer os.Exit(exitCode)` 最先注册的现有技巧，以及 `StopTimeout(15s)` / `RegistrarTimeout(5s)` 不变。

---

## 6. 分阶段迁移顺序与守卫

| 阶段 | 动作 | 守卫（不满足即停） |
|---|---|---|
| **P0 基线**（不改 Go 包路径） | **先收尾 `admin/` → `planstar_frontend/admin/` 重排提交**；修 `AGENTS.md` 5 条契约口径；ADR 台账与索引；`planstar_arch_optimize.md` 标注/迁 `docs/research/`；根 `docs/notes/` 归位；8 个空目录处置；11 库口径统一；`ci.yml` paths 修 `planstar_frontend/admin/**` + 必过 Go 门禁；`routes_test.go` 增条数断言；错误码表单源化；`Makefile` 补 `check`/`lint` 并修 `go vet ./...` | `go build/vet/test -race` 全绿；CI 全 job 通过；契约集合与 P0 前快照逐字一致；`git status` 无 `D admin/...` 与 `?? planstar_frontend/` 并存 |
| **P1 共享内核 + 装配** | 建 `biz/shared`（money/time/errors/event/paging/do）；`data/shared`（`*Data`/`mapDriverError`/`rowQuerier`/事务 helper）；`platform/` 归位；`pkg/session` 拆 `token`/`cookie`；`notify.SMSSender` 端口移到 `biz/identity`；`QuoteTTLFromEnv` 改为 conf 注入；`main.go` 瘦身 + `wiring.go` 分层小节 + 显式生命周期钩子 | `go list -deps` 三条断言全绿且 `biz` 不再传递命中 `net/http`；`/healthz`、`/readyz`、三 server 行为不变；`main.go ≤ 80 行` |
| **P2 层内按域分包** | 按 §3 映射表逐域迁移（建议顺序：`identity → catalog → community → wallet → trade → project → ops`，先简单后跨域）；每域一次提交；`service/{api,admin,events}` 拆分；`server/admin*.go` 的运营 handler 迁入 `service/admin/` | 每域提交后 `go build ./... && go test -race ./...`；无循环依赖；30 个单测 + `test-integration` 全绿；路由集合 == P0 快照；阈值断言全绿 |
| **P3 接缝固化** | 域间禁止 import 对方 `data`；跨域一律「消费方定义端口 + wiring 注入适配器」；按域 CI 路径 / 指标 label / 文档归属；产出「若需拆服务」的候选域清单 | 每域可独立 `go test ./internal/{biz,data}/<domain>/...`；端口清单与 18 个 repo 端口对账完整 |

**每阶段独立提交、独立回滚**；禁止跨阶段混合重构与功能修改（对齐 `CONTRIBUTING.md`「一个 PR 只解决一个问题」）。

### 阈值与守卫（写进 `AGENTS.md` 与 CI）

| 指标 | 阈值 | 现状触发项 |
|---|---|---|
| 单包 `.go` 文件数 | ≤ 40（建议 ≤ 20） | `biz` 29（含测试）暂未超，但承载 7 域 |
| 单包承载业务域数 | ≤ 1 | `biz`/`data`/`service` 均 7 域 → **触发** |
| 单文件 | ≤ 30 KB 或 ≤ 800 行 | `data/project.go` 34 KB → **触发** |
| `main.go` | ≤ 80 行 | 327 行 → **触发** |
| `wiring.go` 单注册函数实参 | ≤ 30 | 现状单文件全量装配 → **触发** |
| 装配主干行数 | ≤ 300 | 327 行 → **触发** |

阈值来自 go-wind 的臃肿实证（core `data`=114、`service`=104、`grpc_server.go` 90+ 形参、`wiring.go` 439 行），并留出 planstar 当前规模的合理余量。

---

## 7. 后果与影响（Consequences）

- **正面**：目录自解释；依赖方向与目录阈值成为 CI 断言；装配可扩展；`biz` 真正纯 Go（拆分 `session` 后传递依赖也干净）；跨域调用有明确规则；未来拆服务可按域整体迁出。
- **负面/风险**：103 个文件 import 路径重写、Git blame 断裂；P1 共享内核若未先完成，P2 必然循环依赖；`data/shared` 集中事务 helper 时须保留「跨库单 `*sql.Tx` + 限定名」的硬约束注释，否则分包后容易被误改成本地事务；`platform/` 归位可能让 `observability` 被 `data/es/` 反向引用，须通过参数注入而非全局单例；CI 新增门禁短期拦住既有 PR。
- **中性**：域划分固化后成为 code owner 与需求归属依据；`docs/ADR/` 台账与 `docs/notes/` 归档区建立；`.archive/go-wind/` 正式定位为只读证据源；本方案不改变任何运行时行为，**无需灰度、数据迁移或 Flutter 协同**。

---

## 8. 验证（Verification）

```bash
cd planstar_backend
go build ./... && go vet ./... && go test -race ./... && make test-integration

# 结构
test (Get-Content cmd/planstar_backend/main.go).Count -le 80
# 任一 internal/{biz,data,service}/<domain>/*.go 数 ≤ 40；单文件 ≤ 30KB / 800 行

# 依赖方向（三条断言，见 §4.4）
# 路由契约：57 条注册 / 56 唯一端点，尾斜杠别名恰好 1 条，集合 == P0 快照
go test ./internal/server -run 'TestContract|TestRoutesMatchContract' -count=1 -v

# 业务行为护栏
scripts/smoke-local.sh
# /readyz 五探针（mysql/redis critical，es/mq/consul optional）输出与治理前一致
# /metrics 21 项指标名称与 label 不变

# 文档一致性（P0）
# docs/ADR/README.md 索引条目数 == ADR 文件数
# docs/README.md 有效索引无死链；agents.md 5 条契约口径与代码一致
```

---

## 9. 关联（Links）

- 代码: `planstar_backend/internal/{biz,data,service}/`、`planstar_backend/cmd/planstar_backend/main.go#L45-L301`、`planstar_backend/internal/biz/models.go`、`planstar_backend/internal/pkg/session/session.go`、`planstar_backend/internal/data/data.go#L45-L49`
- 文档: `docs/ADR/ADR-0001-modular-monolith-governance.md`（上位决策）、`docs/ADR/ADR-NNNN.md`（模板）、`docs/ADR/planstar_arch_optimize.md`（参考材料，经 ADR-0001 §1.1 复核）、`docs/api/API_CONTRACT.md`、`docs/architecture.md#§10`、`agents.md`、`PLANSTAR_DEEP_STUDY.md`
- 参考快照: `.archive/go-wind/go-wind-cms/backend/{AGENTS.md,app/core/service/cmd/server/wiring.go,tools/register/main.go}`、`.archive/go-wind/go-wind-admin/{CONTRIBUTING.md,backend/{AGENTS.md,Makefile,app.mk,.golangci.yml,tools/register/main.go}}`
- 关联 ADR: `ADR-0001`（D2/D3/D6 的上位决策）；`ADR-0003` 起为独立功能性 ADR（见 ADR-0001 §8）
