# Planstar 仓库深度学习报告

> 学习日期：2026-11-19 · 范围：`d:/workspaces/flutter_projects/planstar_app/planstar_kratos_server` 全仓
> 方法：分 6 条线并行通读（文档 / 入口与 HTTP 层 / 领域层 / 数据层 / 基础设施 / 建模部署前端），共覆盖 `planstar_backend/` 全部 104 个 Go 源文件与 30 个测试文件、全部配置与部署文件、`docs/` 全部有效文档，并对关键结论做源码复核与本地实测。
> 实测：本机 `go1.27.3 windows/amd64`（`go.mod` 声明 `go 1.26.0`）；`go build ./...` 退出码 0；`go test ./...` 15 个包全部 `ok`，无失败、无跳过（`cmd/planstar`、`mq`、`observability`、`httpx`、`testutil` 无测试文件）。

---

## 0. 先给结论：这个仓库到底是什么

它不是"一个 Kratos 服务"，而是**一个产品 + 三个可运行子项目 + 一批归档资料**的复合仓库：

| 组成 | 路径 | 性质 |
|---|---|---|
| 业务后端（主体） | `planstar_backend/` | 独立 Go module `planstar`，Kratos v3 单体，手写 `net/http.ServeMux`，端口 8098 / 9108 |
| 运营后台 | `planstar_frontend/admin/` | **另一个独立 Go module**（`cmd/server` + Wire + Kratos gRPC/HTTP）+ Umi Max 控制台，端口 9100 / 9101 / 9110 / nginx 80 |
| 官网 Web | `planstar_frontend/planstar_web/` | pnpm + Turborepo + Next.js 15 App Router，只读展示企划域 |
| 静态落地页 | `site/` | 纯 HTML，GitHub Pages 发布 |
| 契约与知识库 | `docs/`、根目录 4 份 `*NOTES.md` | 契约/架构/运维/交付文档 + 对上游 kratos-layout、kratos-shop 的学习笔记 |
| 归档/空目录 | 仓库根 kratos-layout 残留、`backend/`、`docs/legacy/`、`docs/ADR/`、`site/assets/`、`site/openapi/`、`planstar_backend/deploy/{loki,promtail,otel}/` | 参考或占位，**不是有效实现** |

**最容易误判的三件事**（下文逐条给证据）：

1. 仓库根的 `README.md` / `agents.md` 描述的"9 个库、登录失败 5xx+text/plain、发码恒 200"是**旧口径**，与当前代码不符。
2. `docs/notes/MIDDLEWARE.md`、`docs/notes/CODEGEN_VERIFICATION.md`、`docs/notes/KRATOS_STUDY_NOTES.md`、`docs/notes/KRATOS_SHOP_STUDY_NOTES.md` 写的是**上游 kratos-layout 模板和 kratos-shop 微服务**，不是本服务；`CLAUDE.md` 只是内容为 `AGENTS.md` 的链接桩。
3. `AGENTS.md` 说 `cmd` 是唯一装配点、无 Wire —— 这对 `planstar_backend/` 成立，但 `planstar_frontend/admin/` 是**使用 Wire 的**另一个服务，不可混为一谈。

许可证是 **PolyForm Noncommercial 1.0.0**：允许一切非商业用途，禁止商业使用，分发须保留 `Required Notice: Copyright (C) 2026 uifj`，违规有 32 天补救期。这是理解整个项目性质的前提。

---

## 1. 产品域：企划星在解决什么问题

唯一事实源 `docs/product/POSITIONING.md`。产品是**企划支持平台**，不是电商、不是众筹、不是售票：最小单元是"企划"，动词是"支持"，闭环终点是"达成并交付"。

- **两段闭环**：① 立项前，网友作为"星推官"投票通过/驳回并给理由 —— **不扣星币、不产生订单/流水**，UI 仍写"支持/不支持"，详情页必须标"尚未立项"；② 立项后，用户出资支持并交付回报。二者是同一闭环的两段，不是两个产品。
- **品牌约束**：slogan「让喜欢的企划，真的发生」；禁用"保本/收益/分红"；品类白名单 10 个一级品类（含面经/经验类），8 类黑名单（金融集资、慈善募捐、黄牛票务、违法违规、特殊资质、预付卡等）。
- **演进策略**：票务表 → 企划域 6 条兼容映射，「只增不改、新端只写新域、支付链路复用、履约三态化」。所以代码里同时存在 legacy `activity/ticket_*` 与新的 `project/pledge` 两套模型，这是**有意的历史兼容**，不是冗余。
- 用户分层 L1~L5、7 环节闭环、北极星指标"企划达成率"、五必选详情区块。

这条产品线解释了代码里大量"别扭"设计：为什么既有 `/v1/activities` 又有 `/v1/projects`；为什么投票 `coinTier` 只记录意愿不扣费；为什么 `pledge_status` 有 pending→paid→confirmed→shipped→delivered 五态。

---

## 2. 架构：三层形状 + 依赖倒置

```
client → DTO → service → DO → biz → DO → data → PO → storage
                            ^              ^
                            | 声明 repo 端口 | 实现 repo 端口
                            +--- 依赖倒置 ---+
```

- `service`：把 HTTP 请求解析成 DO、在边界校验、把 biz 错误映射成统一 JSON。不 import `data`。
- `biz`：纯 Go 的用例、DO、18 个哨兵错误、18 个 repo 接口、领域事件。实测 import 中**没有** `net/http`、`database/sql`、redis、amqp、elastic、gorm。
- `data`：实现端口，持有 10 个 `*sql.DB` + 1 个 Redis + 1 个 ES 客户端，把驱动错误映射成 biz 错误。
- `cmd/planstar_backend/main.go`：**唯一手工装配点**（约 300 行 `NewXxx`），全仓无 `ProviderSet`、无 Wire 生成文件。

**对 Kratos v3 的 5 条「有意偏差」**（`docs/architecture.md` §10，等价于事实上的 ADR，`docs/ADR/` 里只有一个空模板）：不用 `khttp.Middleware`、不用 proto/OpenAPI binding、不用 Wire、不用 `kratos/errors`、对外固定 `{code,message,traceId}`。理由：55+ 端点与 Flutter 裸 JSON / Cookie 契约已稳定运行，改 protobuf 是破坏性变更。Kratos 实际只承担 **App 生命周期、transport.Server、日志、Consul 注册**。

**模块边界**：业务单体（8098）与运营后台（独立进程）之间，Cookie `psSession` 与 `psAdminToken` 完全隔离；后台对业务库只有 SELECT（+ `user_info` 受限 UPDATE），审核/举报统一转调 `9108` 的 ops API，不可用返回 503 而不静默放行。

---

## 3. 启动时序与一次请求的完整生命周期

### 3.1 `main.go` 装配顺序（`planstar_backend/cmd/planstar_backend/main.go`）

1. `flag.Parse()` → JSON 结构化日志（`klog.NewLogger(observability.NewTraceHandler(slog.NewJSONHandler(...)))`，注入 `service.id/name/version`）→ `klog.SetDefault`。
2. **最先注册** `defer os.Exit(exitCode)`，保证后续所有 cleanup 先执行（LIFO 顺序技巧）。
3. `conf.Load(-conf)`，默认 `configs/config.yaml`；失败 `exitCode=1`。
4. **可观测/运维闸门先于数据层**：`NewMetrics` + `SetScrapeLimits` → `NewOpsAuth` + `Validate(env)`（生产无令牌直接启动失败）→ `InitTracer`（OTLP/gRPC，失败只记日志不阻断）。
5. `data.NewData`（逐库 Ping，任一失败启动失败）→ MQ `Publisher`（`mq.addr` 为空则注入 `biz.NopEventPublisher`）→ Consul 客户端（失败仅 Warn）→ `notify.NewSMS`（生产 fail-closed）。
6. `NewAuthRepoForEnvironment`，生产环境额外跑 `AssertNoTestCredentials`，命中弱口令即退出。
7. **手工装配 12 个 Usecase**：Auth / Catalog / Feed / Message / Order / User / Project / ProjectIdea / Payment / Wallet / Shop / Chat。
8. 后台 `appCtx`：ES 存在时 `EnsureIndex` + `go catalogRepo.RunSync(appCtx, 300s, ...)` 全量重建。
9. `http.NewServeMux()` → `service.New(...).Register(mux)` → 注册 DB / Catalog / Biz 指标到独立 Registry。
10. `ops.NewReadiness(buildOpsChecks(...))`，5 项探针：mysql/redis 为 critical，elasticsearch/rabbitmq/consul 为 optional，未配置即 `skipped`。
11. 三个 `transport.Server`：`hs`（8098 业务）、`adminSrv`（9108 运维）、`mqSrv`（MQ 消费者）。
12. `kratos.New(...)`：`StopTimeout(15s)`、`RegistrarTimeout(5s)`、`BeforeStop(MarkStopping + cancelApp)`、`AfterStart(MarkStarted)`；Consul 只注册业务端口 → `app.Run()`。

**信号处理**：全文件没有 `signal.NotifyContext`，SIGINT/SIGTERM 完全交给 `kratos.App.Run()`。优雅停机 = 先关就绪闸门摘流量 → 取消后台 goroutine → 15s 给在途请求与消费者收尾 → 5s 注销注册中心 → defer 链清理 MQ / Data / Tracer。

### 3.2 请求生命周期（`internal/server/http.go`）

```
Kratos transport (读10s/写30s/空闲60s)
  → corsFilter (kratos Filter; OPTIONS 直接 204 不进 next)
  → metrics.Middleware (W3C Extract + otel span + RED 指标 + 访问日志 + X-Trace-Id)
  → traceHeader
  → bodyLimit (1MiB 默认；/v1/upload 前缀 20MiB；超限 413)
  → timeoutHandler (ctx 15s；超时 504 TIMEOUT，缓冲丢弃)
  → recoverer (panic → 500 INTERNAL，堆栈只进日志)
  → Go 1.22 ServeMux「METHOD /path」
  → service handler (psSession → Redis 优先、回源 login_session)
  → biz usecase → data repo → 裸 JSON
```

CORS 白名单精确匹配（不支持 `*`），为空时按同源判定，用 `r.TLS` 判 scheme 而**不信任 `X-Forwarded-Proto`**；`clientIP` / `remoteIP` 一律取 TCP 对端，不信任 `X-Forwarded-For` —— 这两处是刻意的防伪造设计。

业务 server 直接改内嵌 `srv.Server` 的 Read/Write/Idle/ReadHeader 超时并接 `ConnState` 指标（注释说明 kratos v3 未暴露这些选项）。admin server 的中间件链更短：只有 `metrics.Middleware(recoverer(mux))`，写 handler 各自用 `http.MaxBytesReader(..., 16<<10)`。

### 3.3 就绪引擎（`internal/ops/readiness.go`）

`/healthz` 恒 200、不查依赖、只报进程存活（uptime + `Cache-Control: no-store`）；`/readyz` 才是真闸门：

- `MarkStarted` / `MarkStopping` 闸门，未启动或停机中直接 `ready=false`。
- 5 个探针**并发**执行，`TotalTimeout=3s`、单项 2s，失败退避 2→4→8…30s。
- `singleflight` 去重 + 2s TTL 缓存（响应带 `Cached: true`）。
- `sanitizeError` 脱敏 DSN/URL 口令并截 200 字；未配置依赖返回 `skipped`。
- 聚合：任一 critical 失败 → `fail`（HTTP 503）；仅 optional 失败 → `degraded`（仍 200，`Ready = status != fail`）。

---

## 4. HTTP 层：路由、错误与必须保留的契约怪癖

### 4.1 路由全清单（源码 `internal/service/service.go:73-144`，Go 1.22 ServeMux 精确匹配）

**实测 57 条注册 / 56 个唯一端点**（`GET /v1/user/` 是 `GET /v1/user` 的尾斜杠别名）。`docs/api/API_CONTRACT.md:9` 写的是「56 条 / 55 唯一端点」，**比代码少一条**；`internal/server/routes_test.go` 的 `TestRoutesMatchContract` 只做归一化集合双向比对、不校验条数，所以测试仍然通过。

| 分组 | 端点 |
|---|---|
| 认证 | `GET /v1/user/captcha`、`POST /v3/user/sendLoginCode`、`POST /v1/user/login` |
| Legacy 目录 | `GET /v1/home/banners`、`/v1/activities/hot`、`/v1/activities`、`GET /v3/activities/{id}` |
| 动态 | `GET /v4/feed/hot`、`/v1/feed`、`/v1/tweets/{id}`、`/v1/userReply` |
| 消息 / 客服 | `GET /v1/messages`、`/v1/messages/unreadCount`、`POST /v1/messages/read`、`GET /v1/chat/threads`、`GET\|POST /v1/chat/threads/{id}/messages`、`POST /v1/chat/threads/{id}/read` |
| 订单支付 | `GET /v1/order/list`、`/v1/order/detail`、`POST /v1/unifiedOrder/confirm`、`POST /v3/unifiedOrder/makeOrder`、`POST /v1/unifiedPay/pay`、`GET /v1/unifiedPay/payDetail`、`POST\|GET /v1/order/{id}/refund` |
| 用户地址 | `GET /v1/user`（+尾斜杠别名）、`GET\|POST /v1/myaddress`、`DELETE /v1/myaddress/{id}`、`POST /v1/myaddress/default/{id}` |
| 钱包签到 | `GET\|POST /v1/user/signIn`、`GET /v1/moneyaccount`、`/v1/moneyaccount/sequences` |
| 星币商城 | `GET /v1/shop/goods`、`/v1/shop/goods/{id}`、`POST /v1/shop/goods/{id}/skus/{skuId}/redeem` |
| 企划域 | `GET /v1/categories`、`/v1/projects/hot`、`/v1/projects`、`/v1/projects/{id}`、`/{id}/reward-tiers`、`/{id}/updates`、`/{id}/faqs`、`/{id}/reviews`(GET/POST)、`/{id}/reports`(POST)、`/{id}/pledges`(POST)、`GET /v1/pledges`、`/v1/pledges/{id}`、`/v1/creators/{id}`、`/v1/rankings` |
| 企划创意 | `GET /v1/projectIdeas/feed`、`GET /v1/projectIdeas/{ideaId}`、`POST /v1/projectIdeas/{ideaId}/vote` |

**版本号错配是历史包袱，契约明确禁止「顺手统一」**：发码 `/v3/`、登录 `/v1/`、详情 `/v3/activities/{id}`、列表 `/v1/activities`、热动态 `/v4/feed/hot`、下单 `/v3/unifiedOrder/makeOrder`、确认 `/v1/unifiedOrder/confirm`。

### 4.2 运维 / admin 端口（9108）

`GET /healthz`、`GET /readyz` 匿名；`GET /metrics` 与全部 `/admin*` 必须 `OpsAuth`（Bearer + 可选 CIDR）：

- `GET /admin/console`（内嵌单页 HTML）、`GET /admin`、`GET /admin/`
- `GET /admin/reports`、`POST /admin/reports/{id}/handle`
- `GET /admin/projects/{id}`、`POST /admin/projects/{id}/review`
- `POST /admin/feeds/{id}/status`、`POST /admin/goods/{id}/status`、`POST /admin/goods-skus/{id}/status`
- `POST /admin/refunds/{id}/review`

`OpsAuth` 是 **fail-closed**：无令牌时返回 401 + `WWW-Authenticate: Bearer`，绝不"打告警后放行"；令牌比对用 `subtle.ConstantTimeCompare`；`actor = "token:"+sha256(token)[:8]`；每次调用输出 `ops audit` 日志（`audit=true/actor/action/target/result/remote_addr/status`）并计入指标，日志不含令牌明文。副作用：浏览器直接打开 `GET /admin/console` 会 401，需反代注入头或 curl。

`/metrics` 用独立 `prometheus.Registry` + `promhttp.HandlerFor(EnableOpenMetrics:true)`，并发上限 8 / 超时 5s，超限立即 503 并计数。

### 4.3 错误映射（`service.fail()`，18 分支）

统一 `{"code","message","traceId"}` + 响应头 `X-Trace-Id` 同值；成功响应是**裸 JSON**（无 `{code,message,data}` 信封）。

| HTTP | code | 触发 |
|---|---|---|
| 400 | `BAD_REQUEST` / `INSUFFICIENT_COINS` / `INVALID_COIN_TIER` / `REASON_REQUIRED` / `INVALID_REASON_CODE` | 参数、余额、投票档位/理由 |
| 401 | `UNAUTHORIZED` | 未登录、登录失败 |
| 403 | `FORBIDDEN` / `IDEA_SELF_VOTE_FORBIDDEN` | 越权、自投 |
| 404 | `NOT_FOUND` / `IDEA_NOT_FOUND` | 资源不存在 |
| 409 | `PRICE_CHANGED` / `SOLD_OUT` / `LIMIT_EXCEEDED` / `QUOTE_EXPIRED` / `REFUND_NOT_ALLOWED` / `CONFLICT` | 订单链路专用 |
| 413 | `PAYLOAD_TOO_LARGE` | body 超限 |
| 429 | `RATE_LIMITED` | 发码/投票限流 |
| 500 | `INTERNAL` | 兜底（细节只进日志） |
| 504 | `TIMEOUT` | ctx 超时（中间件） |

admin server 有**第二套映射**：`ErrRefundNotAllowed` 在 9108 是 **400**，主站是 409；`BlacklistBlockedError` 用 `errors.As` 输出 400 + `reasons`/`blocked.Hits`。

### 4.4 契约怪癖逐条核实（`agents.md` 的 5 条 vs 代码事实）

| `agents.md` 口径 | 代码事实 | 判定 |
|---|---|---|
| 认证用 `Cookie: psSession`，非 Bearer | ✅ `session.CookieKey="psSession"`，HttpOnly + SameSite=Lax + 30 天，prod 加 Secure；`FromRequest` 仍兼容同名请求头 | 有效 |
| **登录失败 `5xx` + `text/plain`** | ❌ `auth.go:132-137`：`ErrBadRequest`→400，其它→**401 `UNAUTHORIZED`** + 统一 JSON。`httpx.Plain` 全仓**零调用**，`API_CONTRACT.md:39` 明确写"业务错误不再使用 text/plain" | **已过时，不要按此改代码** |
| 发码总是 200 + `{success,needCaptcha,errorMsg}` | ⚠️ 参数错/手机号非法/图形码错/发送失败确实恒 200；但进程内限流与 Redis 分布式限流命中时返回 **429 `RATE_LIMITED`** | 部分过时 |
| 金额：DB 存元，API 返回分 | ✅ `biz.Fen()` = `math.Round(元×100)`；`DECIMAL(12,2)`；service 层零换算 | 有效 |
| 分页：`page` vs `pageIndex` | ✅ 目录/动态/企划用 `page`，消息/订单用 `pageIndex`；另有 `cursor/size`、`limit/before`、`nextPageKey` 三套 | 有效但不完整 |
| 枚举取自 `dict_enum` | ⚠️ `planstar_meta.dict_enum` 存在且 seed，但**代码里枚举是硬编码白名单**（`biz/project.go`、`biz/payment.go`），没有运行时读取字典表 | 名不副实 |

其它必须记住的隐式约定：

- `1 星币 = 100 分`，但 `priceCoins` / `todayReward` 的单位是**星币个数**；时间多为**毫秒戳**（`birthday=0` 未设、`paidAt=0` 未支付），而订单 `createTime` 是 `yyyy-MM-dd HH:mm:ss`、流水 `happenTime` 是 RFC3339。
- `pageSize<=0 或 >100` **回落到 20**（不是截断到 100），归一化在 biz 层；`nextPageKey` 返回的是**下一页页码**而非游标，只有 `/v1/projectIdeas/feed` 是真 keyset 游标 `hotScore_ideaId`。
- 字段别名泛滥：`activityId|activity_id|showId`、`count|buyCount|quantity`，由 `httpx.StrAny/Int64Any/BoolAny` 兼容；同义字段重复输出（`orderId|id`）。
- `/v1/myaddress/default/{id}` 对不存在/非本人地址**不报 404，仍返回 `{}`**；`GET /admin/reports` 的 `total` 是本次返回条数；`/v1/userReply` 无结果时把 `tweetId` 当 `activity_id` 回退。
- 成功响应里 `dvid` 只在新建时写响应头，`psSession` **只写 Cookie，禁止写 JS 可读响应头**（`session.go` 注释）；`API_CONTRACT.md:69` 说"响应头 psSession"是文档错误。

---

## 5. 领域层 `internal/biz`：19 个实现文件 + 10 个测试

### 5.1 文件地图

| 文件 | 职责 |
|---|---|
| `models.go` | 共享 DO + `Fen/FmtDT/FmtDTp/BirthTs` 转换助手（导出给 data 层用） |
| `auth.go` | `UserAccount`、**7 个通用错误**、`AuthRepo`/`AuthCache` 端口、`AuthUsecase` |
| `errors.go` | 订单链路 6 个 sentinel errors |
| `event.go` | 4 类领域事件、`DomainEvent`、`EventPublisher`、`NopEventPublisher` |
| `catalog.go` / `feed.go` / `message.go` / `chat.go` / `user.go` | 读面用例与端口 |
| `quote.go` / `order.go` | **权威定价** + 下单 |
| `payment.go` / `wallet.go` / `shop.go` | 支付、退款、钱包、星币商城 |
| `project.go` / `project_idea.go` / `moderation.go` | 企划域（DO 大集合、18 方法端口）、创意叠卡、黑名单审核 |
| `ops.go` / `report.go` | 运营写、举报单 |

### 5.2 14 个 Usecase 的核心逻辑

- **`AuthUsecase.Login`**：手机号 → 查用户 → 短信码分支（`VerifyLoginCode` 原子消费，不存在则 `CreateUser` 且 `newUser=true`）／密码分支（用 `dummyPasswordHash()` 抹平**手机号枚举时序侧信道** + PBKDF2 校验）→ `Status!=1` 拒绝 → `session.NewToken` → `CreateSession`（落 `login_session`）+ `cache.SaveSession`（Redis，失败只告警）→ 新用户发 `user.registered`。
- **`AuthUsecase.SendLoginCode`**：`crypto/rand` 逐位生成 6 位（避免取模偏置，随机源失败**失败关闭**）→ `SaveLoginCode(5min)` → `SMSSender.SendCode`。验证码在 DB 里存的是 **HMAC-SHA256 前 8 字节 hex 摘要**，消费用单条 `DELETE ... AND code=? AND expire_time>NOW()` + `RowsAffected()==1` 保证原子一次性。
- **`OrderUsecase.Quote` / `MakeOrder`** 是本仓最精密的一段：
  - `confirm` 生成 `quoteId/version/expiresAt`（默认 15min，`PLANSTAR_QUOTE_TTL_SECONDS` 可覆盖），快照存 Redis（无 Redis 降级进程内 TTL）。
  - `makeOrder` **忽略客户端金额的定价作用**：`resolveQuote` → 校验 quote 归属 → `VerifyQuote` 四重比对（过期 / 版本 / 服务端重算总额 / 客户端 `Amount`）→ 用服务端快照**覆盖** activityId/session/ticketCategoryId/quantity/payAmount/totalAmount → `CreateOrder` 单事务 → 发 `order.created`（失败只 Warn，不回滚）。
  - 幂等：`Idempotency-Key` 缺失时 `deriveIdempotencyKey = "auto:" + sha256(userID|quoteID|version|…|addressID)[:16]`（37 字符 < VARCHAR(128)）；data 层用 `order_idempotency` 唯一键，**把 quoteID 也写进同一张表**实现"一次报价只落一单"，1062 冲突回滚整单并回读首单。
- **`PaymentUsecase`**：渠道白名单 wechat/alipay/unionpay/balance；`Pay` 按 `(order_number, channel)` 幂等，过期单再调用会重置 pending 并续期 30 分钟；`PayDetail` 按 now 把过期 pending 置 `expired`（**只读归一化、不落库**）；退款按 `order_id` 幂等，仅订单状态 1~5 可退，原因码白名单 5 个。
- **`WalletUsecase.SignIn`**：`SignInRewardCoins=100` 星币，同日重复 `AlreadySigned=true` 不重复入账；data 层实际入账 `reward*100` 分。注意 `SignInStatus.TodayReward` 返回的是**星币个数**，与 `wallet.go` 注释"奖励单位为分"量纲语义不一致（`wallet_test.go` 断言 `Balance == 100*100`）。
- **`ProjectUsecase`**：排序/筹资状态枚举白名单（`hot|new|ending|achievement`、`preheating|ongoing|achieved|delivered|failed`）；`Updates` 支持者可见性三态（游客遇 supporterOnly→401、非支持者→403、支持者→含专享）；`CreatePledge` 数量上限 99、留言 255 rune；`IsReviewRisky` 10 个风控词命中置 `status=2` 审核态；`AchievementPermille` 达成率千分比；`Rankings` 默认 hot/limit 10/上限 50。
- **`ProjectIdeaUsecase.Vote`**：支持票必须档位 5/10/20（**仅意愿、不扣星币**）且强制清空理由码；反对票必须理由码白名单 + 文本 ≤100 rune，`other` 必填文本；作者自投 `ErrIdeaSelfVote`；重复提交返回 200 + `alreadyVoted=true` 不重复计数；`IdeaSupportRate` 三位小数。
- **`ProjectUsecase.ReviewProject`**（`moderation.go`）：`approve` 前若黑名单命中 → `BlacklistBlockedError`（`Unwrap→ErrBadRequest`），否则 `publish_status=1`；`reject` 恒置 3。黑名单 = 6 类关键词（小写 Contains）+ 3 项"空气企划"（`goalAmount<=0`、无交付计划、无风险披露）。
- **`OpsUsecase`**：三个 `Set*Status` 限 `{0,1}`；`ReviewRefund(approve)` 置退款 success + 订单 7 + 回写已退金额，`reject` 只置 rejected 且订单不变。

### 5.3 18 个 repo 端口与实现者

`AuthRepo`/`AuthCache`、`EventPublisher`、`CatalogRepo`/`CatalogReader`、`QuoteStore`、`OrderRepo`、`PaymentRepo`、`WalletRepo`、`ShopRepo`、`UserRepo`、`ChatRepo`、`FeedRepo`、`MessageRepo`、`ProjectRepo`(18 方法)、`ProjectIdeaRepo`、`ReportRepo`、`OpsRepo` —— 全部由 `internal/data/` 一一实现，唯一例外是 `EventPublisher` 由 `internal/mq/Publisher` 实现（未配置时 `NopEventPublisher`）。

### 5.4 领域事件

4 个 routing key：`user.registered`、`order.created`、`catalog.activity.changed`、`catalog.activity.deleted`。`DomainEvent{ID(16B随机hex), Type, UserID, OccurredAt, Data}`。**发布点只有 2 个**（Auth 新用户、Order 创建成功），`catalog.*` 两个事件 biz 内**没有发布者**，只在消费端和文档里存在 —— 目录增量同步实际靠 300s 全量重建兜底。发布失败只 Warn、无 outbox，消费端 `service/event.go` 对未知事件 Warn 后 `return nil`（会被 Ack）。

### 5.5 分层合规性的 4 处轻微越界

1. `quote.go:58 QuoteTTLFromEnv()` 在领域层直接 `os.Getenv`，配置读取侵入（更合理是 conf 注入）。
2. `auth.go` 直接调用 `captcha.New`、`secret.HashPassword`、`session.NewToken` 等具体技术工具，而非全部经端口。
3. `models.go` 导出 `Fen/FmtDT/...` 给 data 层用，方向相反（但无循环依赖）。
4. `order.go` / `auth.go` 用 `log/slog` 承担技术关注点。

**没有**优惠券的 DO/usecase/算法，只有 `UserProfile.CouponCount` 展示计数；`planstar_marketing.coupon` 等表存在但业务未接。SKU 级定价也尚未接入（`quote.go:148`）。

---

## 6. 数据层 `internal/data`：19 个文件

### 6.1 库的口径必须先说清

| 口径 | 数量 | 说明 |
|---|---|---|
| 基线建库 | **9** | `docs/sql/01_init_databases.sql` / `00_all.sql`：user / catalog / ticket / order / payment / marketing / community / content / meta |
| `data.go` 连接池 | **10** | `domains` 常量实际 10 个 key，多了 `shop`（`planstar_backend/sql/18_star_coin_shop.sql` 创建） |
| 运行时总库数 | **11** | 再加 `planstar_admin`（`19_admin_console.sql`），由运营后台独立进程使用，业务 data 层不连 |
| 实际被 repo 取用 | **8** | `marketing`、`meta` 池被打开但无 repo 使用；`planstar_marketing.daily_sign_in` 通过限定名在 payment 池上访问 |

`DB()` 用 `fmt.Sprintf("planstar_%s", domain)` 生成 DSN 默认库，因此**未限定表名走本库、限定名可跨库**——这是全部分库设计的物理前提。10 个池共享一个 `otelsql.Register("mysql")` 驱动（`sync.Once`，注册失败回退原生），每池 `MaxOpen=20 / MaxIdle=5 / ConnMaxLifetime=30m`，启动逐库 `Ping`，任一失败即启动失败。

### 6.2 一致性与事务

**核心假设：11 个逻辑库全在同一 MySQL 实例**。跨库 JOIN 直接用 `planstar_xxx.table` 限定名，跨库事务统一用**单个 `*sql.Tx` + 限定名 DML**：`CreatePledge`（catalog.reward_tier + order.pledge + catalog.project 聚合）、`CreateOrder`（order + ticket 库存）、`SignIn`（marketing + payment）、`Redeem`（shop + payment）、`CreatePayment/Refund`（payment + `planstar_order.order_info FOR UPDATE`）、`ReviewRefund`（payment + order）。`project.go:504` 注释明确废弃了"先提交扣库存、失败手工补偿"的旧写法。

**没有 Saga / TCC / outbox / 对账补偿**；一致性手段限于同实例事务、`SELECT ... FOR UPDATE` 悲观锁、条件更新（`WHERE stock>=?` / `remaining_num>=?`）、唯一键幂等。只读跨池聚合（`UserRepo.Profile`、`Balance`）不包事务，存在短暂不一致窗口。

SQL 层规约（`docs/sql/verify.sql` 验收口径）：86 业务表 + 3 张 `_rev_*` 证据表 = 89 表、**0 物理外键**（原 61 条被 `12_constraints.sql` 移除，改为普通索引 + 应用层校验）、全 InnoDB + utf8mb4_unicode_ci、单表二级索引 ≤5、无 FLOAT/DOUBLE、所有列有注释、统一 `gmt_create/gmt_modified`（**无 `created_at/updated_at`、无 `version` 乐观锁、无 `deleted_at`**），软删用 `status=0`。所有外部输入走 `?` 占位符，动态拼接仅限代码内常量分支。

N+1 是已知成本：`Shop.Goods` 逐商品查 SKU、`ProjectRepo.Detail` 十余次串行查询拼装。

### 6.3 Redis

key 全部 `planstar:` 前缀：`planstar:session:<token>`（30d）、`:captcha:<id>`（5min，GetDel 一次性）、`:send:<mobile>`（60s SetNX 冷却）、`:login:fail:<mobile>`（10min，TxPipelined Incr+Expire，≥3 触发图形码）、`:quote:<quoteID>`（15min）。

- **短信验证码不走 Redis**，落 `login_code` 表存 HMAC 摘要 + 原子 DELETE 消费。
- **幂等不在 Redis**，全靠 MySQL 唯一键。
- **没有 Redis 分布式锁**，并发控制靠 MySQL 行锁 + CAS 条件更新。
- `AuthCache` 与 `QuoteStore` 在 Redis 失败时回退进程内 `memEntry` + 惰性 `sweep()`；发码的 Redis `SetNX` 闸门失败按 **fail-closed 拒绝**。
- **无缓存击穿/穿透防护**（无 singleflight、空值缓存、布隆过滤）。`ideaVoteLimiter` 是纯进程内滑窗（用户 60/h、IP 300/h），多实例部署会把阈值放大 N 倍。

### 6.4 Elasticsearch

别名 `planstar-activities`，具体索引 `planstar-activities-YYYYMMDDHHMMSS`（1 shard / 0 replica）。同步三路：① `RunSync` 启动一次 + 每 300s 全量重建，Bulk 每 500 条，`Refresh` 后 `Alias().Add(new).Remove(old)` 原子切换，失败删新索引保留旧；② MQ 增量 upsert/delete（但无发布点）；③ **业务事务内无双写、无 outbox**。

查询侧只有"热门且无 category/city 过滤"走 ES：`TermQuery(status=1)` + `Sort(want_watch_num desc, rank_weight desc, activity_id desc)`，拿到 ID 后**回 MySQL 补全并按 ES 顺序重排**（防读模型陈旧）。ES 未配置返回 `errNoElastic` 静默降级；查询失败回退 MySQL 并累加 `planstar_catalog_es_fallback_total`。

### 6.5 错误映射

**没有统一的 driver→biz 映射 helper**，规则分散手写。唯一公共辅助是 `isDuplicateKey`（MySQL 1062）与 `rowQuerier`（适配 `*sql.DB`/`*sql.Tx`）。`sql.ErrNoRows` 按场景转 `nil` / `biz.ErrNotFound` / `ErrIdeaNotFound` / `ErrForbidden`；1062 通常不是错误，而是"重试生成新单号 / 回滚后回读首单 / 标记已投票 / 开户重锁"。`redis.Nil` 转 `(false,nil)` 或继续查内存；context 超时不在 data 层映射，由 HTTP 中间件统一转 504。

两处刻意的反直觉处理：`VerifyLoginCode` 吞掉 DB 错误返回 false；`setOpsStatus` / `report.Handle` 先 `SELECT 1` 或回查判存在，而不用 `RowsAffected`（避免"设成原值影响行数 0"被误判为 not found）。

---

## 7. 基础设施

### 7.1 RabbitMQ（`internal/mq/`，5 文件）

- 拓扑：topic exchange `planstar.exchange`（durable）、queue `planstar.api.events`（durable）、`Qos(prefetch=32)`；DLX `<exchange>.dlx`，DLQ `<queue>.dlq` 绑定 `failed.<queue>`；4 个 routing key。
- Publisher：懒连接、`Confirm(false)` 逐条等 broker ack（nack / channel 关闭 / 5s 超时均失败）、persistent、W3C `traceparent` 注入、`sync.Mutex` 串行化；`Health()` 用独立 3s 短连接探测。
- Consumer：`autoAck=false`；panic recover → 直接 Nack 进 DLQ；`MaxRetry=5`，`x-retry-count` 头 + 指数退避 1/2/4/8/16s，用默认交换机重投原队列，**只有重投成功才 Ack**；成功后 Ack。改用 broker 级 DLX 是为了修掉旧版"手工 Publish 后无条件 Ack"丢消息的做法。
- Server 实现 kratos `transport.Server`，`Addr==""` 空跑，断线每 5s 重连，`Stop` 关 ctx 等 `done`。

**风险清单**：无消费幂等（重试/重连会产生重复站内通知）；DLQ 无消费者、无告警、无重放工具；发布失败只 Warn；重试是消费者内 `time.After` 阻塞，串行 `Run` 下单条退避会阻塞整队列吞吐（最长 16s）；`QueueDeclare` 无参数漂移处理（已存在且无 DLX 参数的队列会声明失败进入重连循环）。

### 7.2 可观测性（单文件 498 行）

指标统一 `planstar` 前缀，注册在**独立 Registry**（含 process/go collector）：

- HTTP RED：`planstar_http_requests_total{method,route,status}`、`request_duration_seconds`（0.005~5s buckets）、`requests_in_flight{method}`、`request/response_size_bytes`（ExponentialBuckets 256,4,8）、`active_connections`、`build_info{version,go_version,env}`。
- 运维：`ops_probe_status` / `probe_duration_seconds{probe,check}`、`ops_probe_cache_hits_total`、`ops_auth_failed_total{endpoint}`、`ops_audit_total{action,result}`、`metrics_scrape_rejected_total`。
- DB：`planstar_db_connections{domain,state}`（自定义 Collector，抓取时读快照）、`wait_count_total` / `wait_duration_seconds_total{domain}`。
- 跨包注册：biz 的 `planstar_biz_{login,order_confirm,order_created}_total{result}`、data 的 `planstar_catalog_es_fallback_total`。

`routeLabel/isID` 把数字 ID 与 ≥16 位 hex 归一成 `:id` 防高基数。

Tracing 实际是 **OTLP/gRPC**（不是 Jaeger 直连）：`ParentBased(TraceIDRatioBased(ratio))`、`WithBatcher(3s)`、`TraceContext+Baggage` 传播；HTTP span 名 `METHOD <route>`，MQ span 名 `consume <routingKey>`；otelsql / redisotel 自动插桩。`NewTraceHandler` 给每条 slog 注入 `trace_id`/`span_id`，`httpx.Error` 把 `X-Trace-Id` 同时写进响应体与头。

**注意配置陷阱**：`conf.go` 把 `sample_ratio<=0` 强制改成 1，所以 `config.local/lite.yaml` 里的 `0` 不生效，实际靠 endpoint 为空关闭 tracing。

### 7.3 `internal/pkg/` 六个子包

| 包 | 关键实现 |
|---|---|
| `captcha` | 8 随机字节 → 16 hex id、4 位数字码（`crypto/rand.Int` 逐位）、120×44 PNG 自绘 5×7 点阵 + 干扰线 + 噪点 |
| `secret` | 手写 PBKDF2-HMAC-SHA256（10000 轮、16B salt、32B DK），格式 `$pbkdf2-sha256$iter$salt$hash`，`hmac.Equal` 常量时间比对；未用 bcrypt/argon2 |
| `session` | `psSession`/`dvid` Cookie，`SessionTTL=30d`、`DeviceTTL=7d`；`NewToken()` 32B→64hex；**本包无签名/过期校验**，有效性由 MySQL `login_session` + Redis + `SessionExpireDays` 决定 |
| `httpx` | 裸 `JSON`、`Plain`（已无调用方）、`Error`（统一错误体）、`ReadJSON/ReadMap`、多别名 `StrAny/Int64Any/BoolAny` |
| `notify` | `SMSSender`/`EmailSender` 端口 + `NoopSMS/logSMS/aliyunSMS`；生产 fail-closed，`codeFingerprint` 只记 SHA-256 前 8 字节 hex，**绝不明文打码**；阿里云走 Dypnsapi `SendSmsVerifyCodeWithOptions`（6 位/300s）且拒绝自由文本；**`NewEmail` 生产返回 `NoopEmail`，邮件全丢** |
| `storage` | `ObjectStorage` + `aliyunStorage`（CDN URL 或预签名 900s），但 **`main.go` 完全没有接线**，`conf.Storage.OSS` 与 `PLANSTAR_STORAGE_OSS_*` 目前是未使用的配置 |

---

## 8. 数据建模与迁移链

`docs/sql/` 21 个 SQL + `planstar_backend/sql/` 8 个 SQL + 1 个 sh。`00_all.sql`（299KB）是 01..13 与领域增量的合并一键脚本，compose **只挂载它**而非逐号文件。

- 设计基线：9 库 86 业务表 + 3 证据表；`12_constraints.sql` 去掉 61 条外键（其中 24 条跨库）；`13_query_indexes.sql` 是 no-op（8 个候选索引已内联进 DDL）；`90_teardown.sql` 逆序删 9 个基线库（破坏性）。
- `planstar_meta.dict_enum` + `_rev_*` 三张逆向证据表（217 类 / 1736 字段 / 2363 行 seed），不参与业务查询。
- 新域核心：`project`（`goal_amount/raised_amount` DECIMAL(12,2)、`funding_status`、`is_all_or_nothing`、tag_ids/gallery/faq JSON、4 复合索引）、`reward_tier`、`pledge`（五态 + `delivery_type` physical/digital/right/offline/none）、`project_idea`/`project_idea_vote`。
- 增量脚本链（compose 按**文件名字典序**挂载）：`00_all.sql` → `docs/sql/14_mock_seed.sql`（14 演示用户、四品类、企划 1001..5002、支持/支付/退款/优惠券种子；生产用 `planstar_backend/sql/14_noop.sql` 同名覆盖为零种子）→ `15_auth_support.sql`（PBKDF2 列、login_code、演示账号）→ `16_production_hardening.sql`（`order_idempotency`）→ `17_pay_refund_chat.sql` → `18_star_coin_shop.sql`（shop 库 + daily_sign_in）→ `19_admin_console.sql`（admin 库）→ `20_project_idea.sql`（60 条创意种子）→ 生产额外 `21_disable_test_accounts.sql`、`22_create_app_user.sh`（最小权限 DML 账号）。
- `docs/sql/history/` 与 `optional/PROPOSALS.sql` 是只读对照，**明确禁止执行** `history/tools/build_sql_rebuildnew.py`（依赖已删目录，会覆盖当前结果）。

`planstar_backend/deploy/backup.sh` 与 `22_create_app_user.sh` 按 **11 库**授权/备份，`preflight.sh` 会校验 backup 脚本库名与建库脚本一致。

---

## 9. 配置、环境与部署

### 9.1 四份配置（`planstar_backend/configs/`）

| 文件 | env | 要点 |
|---|---|---|
| `config.yaml` | dev | 全 127.0.0.1、root/root 明文、ES/MQ/Consul/tracing 全开、ratio 1.0 |
| `config.local.yaml` | local | 允许测试口令、地址改 compose 服务名、ES/MQ/Consul/tracing 关、短信 logSMS 只打指纹 |
| `config.prod.yaml` | prod | `cookie_secure:true`、otel-collector:4317、ratio 0.1、`service_host=planstar-api`、密钥全由 env 注入 |
| `config.lite.yaml` | prod | 4C8G 低配，关重组件；**`cookie_secure:false` 标注「ICP 备案期间临时」** |

> `agents.md` 提到的 `config.docker.yaml` 实际在 `planstar_frontend/admin/configs/`，不在 `planstar_backend/configs/`。

`conf.Load` 顺序刻意把 `PLANSTAR_ENV` 覆盖放在**最前**（保证 Cookie Secure 按真实环境推导），然后 `applyDefaults` → 其余 env 覆盖 → `Validate`。`IsProductionLikeEnv = !IsLocalEnv` 是**默认拒绝**设计：staging / docker / 空值 / 拼写错误全部按生产处理。

`Validate` 的生产不变量（fail-fast）非常严格：`allow_test_credentials=false`、`CookieSecure=true`、`ops.token` 与 `ops.allowed_cidrs` 非空、MySQL/Redis 口令非空且 user≠root、`notify.sms.enabled=true` 且 provider∈{aliyun,aliyun-dypns} 且 4 个密钥齐全；OSS 启用还要校验 endpoint/bucket/ak/sk。**副作用**：`config.lite.yaml` 里 `mysql.user:root`、`sms.enabled:false`、`cookie_secure:false`、空 ops 令牌，且 `cookie_secure` **没有环境变量可覆盖** —— 该档位以 `env:prod` 启动必然被拒，属于明确的未完成项。

完整 env 覆盖清单见 `conf.go:276-347`（`PLANSTAR_*` 与 `ADMIN_*` 两套），样例在 `planstar_backend/.env.example` / `.env.prod.example`。

### 9.2 四份 compose

| 文件 | 定位 |
|---|---|
| `docker-compose.yml` (base, 491行) | 单机全栈：mysql 8.4 / redis 7 / traefik v3.1 / planstar-api / admin / admin-web / prometheus / alertmanager / grafana / 两个 exporter；rabbitmq / ES / consul / jaeger 需 `COMPOSE_PROFILES=full`；9108、9110、dashboard 只绑 127.0.0.1；api 容器 `read_only:true` + tmpfs + no-new-privileges |
| `docker-compose.local.yml` | standalone 3 容器（mysql/redis/api），`name: planstar-local`，弱口令 `planstar_local_only`，注释明确严禁公网 |
| `docker-compose.prod.yml` | 4C8G/2M 档：所有口令 `${VAR:?}` 强制、`14_noop.sql` 覆盖种子、`config.lite.yaml`、禁用测试口令、Traefik 升 v3.6 + DNS Challenge（火山引擎 `VOLC_*`）、资源限额 |
| `docker-compose.server.yml` | 部署侧最后叠加：关闭不兼容的火山 TOS/OSS、备案期把 8098 从回环改公网 |

`Dockerfile` 两阶段：`golang:1.26.5` builder（`CGO_ENABLED=0 -trimpath -ldflags "-s -w -X main.Version"`）→ `debian:stable-slim`（非 root 用户 `planstar`、`TZ=Asia/Shanghai`、`HEALTHCHECK /healthz`）。控制台另有 `node:22-alpine → nginx:alpine` 的独立 Dockerfile。

脚本：`up-local.sh`（3 容器 + `/healthz` 30×2s 轮询）、`up-prod.sh`（preflight → 落 0600 token → server overlay → `/readyz` 45×2s 轮询）、`deploy.sh`（远程一键推送，令牌经 stdin 同步避免出现在进程列表）、`deploy/preflight.sh`（25 个必需键、`<REPLACE_ME>`、tag 正则、禁测试口令、11 库名一致性、`compose config -q`）、`deploy/backup.sh`（11 库 mysqldump + .env 快照，保留 7 天）。`deploy/planstar.service` 引用的 `docker-compose.lite.yml` **不存在**，是历史残留。

**可观测栈缺口**：`deploy/loki/`、`promtail/`、`otel/` 三个目录为空，4 份 compose 都**没有定义** `loki` / `promtail` / `otel-collector` 服务；但 Grafana 数据源、Prometheus 的 otel-collector job、告警描述、`config.prod.yaml` 的 `trace.endpoint=otel-collector:4317` 都在引用它们。链路与日志聚合目前是「配置已写、容器未编排」。

`prometheus.yml` 7 个 job、15s 抓取，`authorization.credentials_file=/etc/prometheus/ops_token`；`alerts.yml` 6 组约 16 条规则（up=0 2m、5xx >5%/10m、P95>2s、429>1/s、下单失败>20%、ES 回退>0.5/s、DB 池>80%、Ops 鉴权失败>1/s 等）；Alertmanager 因不支持 `--config.expand-env`，用 `entrypoint.sh` awk 渲染 `AM_*`（umask 077）；Grafana 2 个面板共 27 个 panel；Traefik `dynamic.yml` 明确**移除** grafana/prometheus/jaeger 的公网 :80 路由，改走 SSH 隧道。

### 9.3 CI/CD（`.github/` 仅 4 文件）

- `ci.yml`：push/PR main，paths 命中 `planstar_backend/** admin/** docs/** scripts/**`；6 job —— `go`（1.26 build/vet/gofmt/`go test -race`）、`admin-go`、`admin-console`（Node 22 npm ci/build）、`contract`（`TestContract|TestRoutesMatchContract`）、`compose`（两份 `config -q`）、`observability-config`（promtool / Grafana JSON / amtool）。CI **不跑** `make ops-check` / `test-integration`，集成测试不在门禁内。
- `release.yml`：tag `v*.*.*` → GHCR 四标签（semver / major.minor / 原 tag / latest）+ GitHub Release；`deploy` job 仅当 `vars.SERVER_HOST != ''`，先 runner 侧正则白名单校验 tag（防 RCE）、远端二次校验，再 `preflight.sh` → config -q → pull → up -d → `/readyz` 自检，写 GitHub Deployments `production`。secrets：`SERVER_HOST/USER/SSH_KEY/PORT/APP_DIR/GHCR_TOKEN`。
- `pages.yml`：`site/**` 变更 → 直接上传整个 `site/`（无构建步骤）→ `deploy-pages@v4`。
- `dependabot.yml`：每周 gomod(`/planstar`) / github-actions / docker 三类，PR 上限 3/2/2。

> CI paths 写的是 `admin/**`，但根目录**没有** `admin/`，实际 Go module 在 `planstar_frontend/admin/` —— 疑为目录重命名后 CI 未同步，`admin-go` / `admin-console` 两个 job 可能从未被路径触发。

### 9.4 smoke 脚本

`scripts/smoke-local.sh` / `.ps1` 6 项：`/healthz` 200、`/readyz` 200/503 且含 checks、`/metrics` 无令牌 401、`/metrics` 带 Bearer 200 且含 `planstar_build_info`、`/v1/activities` 200 可解析、发码非 5xx。可用 `PS_API_BASE` / `PS_ADMIN_BASE` 覆盖地址。

---

## 10. 前端与站点

### 10.1 `planstar_frontend/planstar_web/`（约 83 源码文件）

pnpm 9.15.9 + Turborepo 2 + TS 5 strict monorepo；`apps/web` 是 Next.js 15.5.4 App Router + React 19 + Tailwind v4；共享包 `@planstar/design-tokens`、`@planstar/ui`、`@planstar/api-client`、`@planstar/config`；测试 Vitest 3 + Testing Library + Playwright。

路由：`/`、`/projects`、`/projects/[id]`、`/rankings`、`/about`、`not-found`、`error`、`loading`。首页 `force-dynamic` 并行加载 5 个楼层，`load()` 把异常降级成 `{data,error,message}` 实现**楼层独立降级**。`next.config.ts` 把 `/api/backend/:path*` rewrite 到 `PLANSTAR_API_BASE`（默认 `http://127.0.0.1:8098/`）；`createServerApiClient()` 每请求新建，从 `next/headers` 取 `psSession` + `dvid` 注入以**避免 SSR 串号**。`api-client` 处理裸 JSON、15s AbortController、`credentials:'include'`、`x-trace-id` 透传、非 2xx 抛 `ApiError`；只接 10 个只读企划端点。无全局状态库，Server Components + props 下传，仅主题/菜单用 `useState`。类型定义与 `biz` 的 JSON tag 对齐，金额单位分、达成率千分比、时间毫秒。

### 10.2 `planstar_frontend/admin/`

**独立 Go 服务**（`cmd/server/main.go` + Wire + `api/kratos/admin`、`api/planstar/admin`、`migrations/0001_seed_admin.sql`）+ Umi Max 4 / Ant Design Pro 3 / antd 6 / React 19 控制台；Biome + Jest + husky。

控制台路由：`/user/login`、`/overview`、`/admins`、`/audit-logs`、`/planstar` 下 10 个子页（users / projects / ideas / goods / goods-orders / orders / pay-orders / refunds / reports / feeds）。`openapi.yaml`（protoc-gen-openapi，65KB）列 31 个 `/v1` 路径，`services/index.ts` 用 `protoc-gen-typescript-http` 生成的客户端，发 `application/protojson`。`shared.ts` 的 `useCursorPager` 专门适配后端 `nextPageToken` 与 ProTable 页码模型的差异。

鉴权：JWT + HttpOnly Cookie `psAdminToken`（`cookie_secure:true`、8h TTL、登录 5 次/600s 锁 900s），`ADMIN_AUTH_JWT_SECRET` 必须 ≥32 字节；前端 `getInitialState()` 调 `Current({})`，`access.ts` 派生 `canAdmin/canOperate/canView` 三级角色，`PasswordChangeGate` 强制首登改密。

### 10.3 `site/`

`index.html` 纯静态单页（内联 CSS、无外部依赖）；**`site/assets/` 与 `site/openapi/` 都是空目录**。仓库里真正的 OpenAPI 是 `planstar_frontend/admin/openapi.yaml`，业务契约是 `docs/api/API_CONTRACT.md`（43KB）与 `ADMIN_CONTRACT.md`（12KB）。

---

## 11. 测试体系与质量实测

**30 个 `_test.go`**，全部 colocated、表驱动、手写 fake/stub（`memXxxRepo`、`fakeXxxAPI`），**不用 mock 框架、不用 testify**：

| 层 | 数量 | 覆盖 |
|---|---|---|
| `internal/biz` | 10 | `Fen/FmtDT`、`CalcQuoteTotal`/`VerifyQuote`、支付退款幂等、签到 20 并发只入账一次、聊天游标、投票全分支、黑名单审核、ops 边界、report/shop |
| `internal/pkg` | 8 | captcha PNG、PBKDF2、Cookie HttpOnly、生产 fail-closed、日志不明文、阿里云 fake API、OSS CDN/presign |
| `internal/conf` | 2 | 10 项非法配置、env 覆盖、环境默认拒绝、11 项生产不安全配置、HMAC key 回退 |
| `internal/server` | 6 | 契约路由双向比对、fail-closed、CIDR、审计脱敏、探针脱敏、100 goroutine × 200 请求压测（p99 阈值） |
| `internal/ops` | 1 | 全 OK / 关键失败 / 降级 / skipped / 启动停机闸门 / 缓存 TTL / 失败退避 / singleflight 并发与 goroutine 泄漏 |
| `internal/service` | 2 | 创意投票限流、`deriveIdempotencyKey` 稳定性与跨用户隔离 |
| `internal/data` | 2 | 空 ES addr 不建客户端；`ops_integration_test.go` |

**集成测试**（唯一 `//go:build integration`）：`data/ops_integration_test.go`，`testutil.MySQLDatabase(t, "planstar_xxx")` 从 `PLANSTAR_TEST_MYSQL_DSN` opt-in，未设置/不可达即 `t.Skipf`，`t.Cleanup(Cleanup)` 删种子；用 4 个域连接池复现生产多池，验证上下架与跨库退款审核事务。

`Makefile`：`build`（`-X main.Version`）、`run`、`tidy`、`fmt`（仅 `gofmt -l` 检查）、`test`（`go test -race ./...`）、`vet`、`check: fmt vet test`、`ops-check`（探针单测 + 可选 promtool/amtool/compose）、`docker`、`sql-apply`、`test-integration`（`go test -tags integration ./...`，DSN 未设自动 skip）、`cover`。

**实测**：`go build ./...` 退出 0；`go test ./...` 15 个包全 `ok`。本机 Go 1.27.3 与 `go.mod` 的 1.26.0 / 镜像的 `golang:1.26.5` 存在版本漂移 —— `docs/notes/CODEGEN_VERIFICATION.md` 记录过 Go 1.27.3 与 1.25.7 错配导致 `http2.TrailerPrefix`、Ent 生成器崩溃等问题，锁工具链是待办。

**领域层测试缺口**：19 个实现文件只有 10 个有测试，`AuthUsecase`（登录/验证码/限流/时序防护）、`OrderUsecase.MakeOrder`、`CatalogUsecase`、`FeedUsecase`、`MessageUsecase`、`UserUsecase` 6 个用例完全无 biz 单测。data 层除 ops 外，auth/order/payment/wallet/project 的 SQL **都没有集成测试**。

**代码风险点**：
- 唯一 `panic` 在 `pkg/captcha/captcha.go:43`（`rand.Read` 失败），与 `session.NewToken` "不允许 panic" 的设计不一致；`biz/event.go:40` 与 `captcha.go:124` 忽略随机源错误（事件 ID 可能全零 / nil 解引用）。
- 生产邮件（`NoopEmail`）、OSS（`Noop`，未接线）、`NopEventPublisher` 都是静默丢弃。
- `config.yaml` 默认 `amqp://root:root@127.0.0.1:5672/` 明文弱口令（仅 dev，生产由 overlay 强制覆盖）。
- `ops_load_test.go` 是带 p99 硬阈值的并发压测型单测，在 CI `go test -race` 下有环境抖动风险。

---

## 12. 文档与代码的偏差清单（最有实践价值的一节）

按"会不会误导改代码"排序：

1. **`agents.md` / 根 `README.md` 的登录失败 5xx+text/plain 已失效** —— 代码是 401 JSON，`httpx.Plain` 零调用，`API_CONTRACT.md` 明确声明不再使用。若照 AGENTS 改代码会破坏 Flutter 客户端与 401 验收。
2. **发码"恒 200"不完整** —— 双重限流命中是 429，`docs/README.md` 的验收记录与 `PRODUCTION_DELIVERY.md` 都实测过 429/401/413。
3. **库数量三套口径**：文档 9 库 / 连接池 10 个 / 运行时 11 个（`shop` + `admin`）。`architecture.md` §4 的"固定 9 个逻辑库"过时。
4. **路由条数**：代码 57 条/56 唯一端点 vs 契约 56/55；`docs/README.md`（11-07）还停留在"23 端点、企划域未实现"，`architecture.md` 写 24 条 —— 这两份是旧基线，以 `API_CONTRACT.md`（11-11）与 `service.go` 为准。
5. **`ADMIN_CONTRACT.md` 缺 4 条运营写路由**（feeds/goods/goods-skus 上下架、退款审核），且称"本地令牌为空时鉴权关闭"与 `OpsAuth` 的 fail-closed 冲突；`GET /admin/console` 实际会 401。
6. **`API_CONTRACT.md` 说登录下发响应头 `psSession`**，代码只写 HttpOnly Cookie（明确禁止 JS 可读头）。
7. **`config.lite.yaml` 不可启动**（见 9.1）；`release.yml` 头部注释说 `PLANSTAR_LITE=1` 可选，但 deploy job 硬编码 `config.lite.yaml`。
8. **`docs/legacy/` 是空目录、ADR 只有模板、`site/assets`/`openapi` 空、`backend/` 空** —— 但 `docs/README.md` 与 `architecture.md` 里有多条指向 legacy 文件的死链；`docs/ADR/` 里**没有任何已记录的架构决策**。
9. **CI paths 指向不存在的 `admin/`**；`docs/notes/CODEGEN_VERIFICATION.md` / `docs/notes/KRATOS_STUDY_NOTES.md` 批评的 `.github/workflows/go.yml` 和 `go-version: ^1.20` 也不存在（实际只有 `ci.yml`，Go 1.26）。
10. **可观测栈容器未编排**（loki/promtail/otel-collector 目录空），但多处配置引用。
11. **短信/OSS 状态互相矛盾**：`aliyun-sms-oss.md` 说默认关闭 Noop；`PRODUCTION_DELIVERY.md` 说生产强制 Dypnsapi、对象存储实际是火山 TOS/S3 故 overlay 关闭且 OSS 变量仍需非空占位。`docs/sql/README.md` 要求 `15_project_domain.sql` 在 `14_mock_seed.sql` 之前，但 compose 按字典序执行（14 在 15 前）。
12. **`docs/sql/README.md` 自身数字打架**："重构后规模" 97 表/1760 列/271 索引 vs "实跑结果" 86 表/89 全表/1590 列 vs `history/VALIDATION.md` 89 表/1444 列/61 外键。

---

## 13. 如果要继续改这个仓库：优先级建议

**P0（会致线上事故或改错契约）**：同步 `agents.md` / 根 `README.md` 的 5 条契约怪癖与代码；修 `config.lite.yaml` 使其能通过 `Validate`（或让 prod overlay 显式覆盖 `cookie_secure`）；确认 `PLANSTAR_QUOTE_TTL_SECONDS` / `PLANSTAR_ES_SYNC_INTERVAL_SECONDS` 的读取实现与 `conf.go` 的覆盖清单是否一致；修 CI 的 `admin/**` 路径。

**P1（数据/消息可靠性）**：给 MQ 消费加基于 `ev.ID` 的去重表，给 DLQ 加告警与重放工具，发布改事务性 Outbox（这正是 `docs/notes/KRATOS_SHOP_STUDY_NOTES.md` §12 建议"搬回的三件事"之一）；补 `catalog.activity.changed/deleted` 发布点或删掉空链路；把重试从消费者内阻塞改成延迟队列/TTL 队列。

**P2（工程质量）**：补 `AuthUsecase` / `OrderUsecase.MakeOrder` 与主要 repo 的测试；把 `QuoteTTLFromEnv` 从 biz 移到 conf 注入；统一 `sql.ErrNoRows`/驱动错误映射 helper；消除 `Shop.Goods` / `Project.Detail` 的 N+1；统一分页阈值（现在通用 20/100、创意 10/20、聊天 20/100、举报 50/200）；锁 Go 工具链版本；补齐 loki/promtail/otel-collector 编排或删掉引用。

---

## 14. 速查表

| 我想找… | 去哪 |
|---|---|
| 路由事实源 | `planstar_backend/internal/service/service.go:71-146` |
| 错误码映射 | `planstar_backend/internal/service/service.go:181-224` / `internal/server/admin.go:225-234` |
| 依赖装配 | `planstar_backend/cmd/planstar_backend/main.go`（全 300 行） |
| 配置字段与 env 覆盖 | `planstar_backend/internal/conf/conf.go`（`Load` :267、`applyDefaults` :511、`Validate` :357） |
| 中间件链 | `planstar_backend/internal/server/http.go` / `internal/observability/observability.go:237-311` |
| 就绪引擎 | `planstar_backend/internal/ops/readiness.go` |
| 领域端口与用例 | `planstar_backend/internal/biz/*.go`（19 实现） |
| 分库与连接池 | `planstar_backend/internal/data/data.go:45-49` |
| 权威定价 | `planstar_backend/internal/biz/quote.go` + `order.go` |
| MQ 拓扑 | `planstar_backend/internal/mq/consumer.go` / `publisher.go` / `server.go` |
| 指标清单 | `planstar_backend/internal/observability/observability.go:56-157` |
| SQL 全量安装 | `docs/sql/00_all.sql`；验收 `docs/sql/verify.sql` |
| 客户端契约 | `docs/api/API_CONTRACT.md`；后台 `docs/api/ADMIN_CONTRACT.md` |
| 架构事实源 | `docs/architecture.md`；产品 `docs/product/POSITIONING.md` |
| 部署手册 | `docs/deployment.md` / `docs/operations.md` / `docs/PRODUCTION_DELIVERY.md` |

**常用命令**
```bash
cd planstar_backend
go build ./...                 # 实测通过
go vet ./...
go test ./...                  # 15 包全绿
make check                     # gofmt -l + vet + go test -race
make test-integration          # 需 PLANSTAR_TEST_MYSQL_DSN，未设自动 skip
make ops-check                 # 探针单测 + 可选 promtool/amtool/compose
./up-local.sh                  # 3 容器本地栈
./up-prod.sh                   # 生产三文件 compose
```
```bash
curl 127.0.0.1:9108/healthz
curl 127.0.0.1:9108/readyz
curl -H "Authorization: Bearer $PLANSTAR_OPS_TOKEN" 127.0.0.1:9108/metrics
scripts/smoke-local.sh
```

---

## 15. 本次学习明确排除的内容

- `planstar_frontend/admin/node_modules`、`planstar_frontend/*/node_modules`、`.next`、`dist`、`*.tsbuildinfo` 等约 15 万个第三方依赖文件（只分析源码与配置）。
- `docs/sql/00_all.sql`（299KB）等大 SQL 的逐行 DDL —— 已通过 `verify.sql`、`12_constraints.sql`、各分文件结构、建表清单与迁移链掌握其设计，未逐列复核。
- `planstar_frontend/admin` 的 Go 业务实现细节（第二个 Go module 的 `internal/` 逐文件）与控制台每个 TSX 组件 —— 已掌握其模块边界、31 个 API、鉴权与角色矩阵。
- 4 份根目录 `*NOTES.md` 是上游模板/微服务的学习笔记，仅提取了"与本项目的对照结论"，未按本项目代码逐条复现。

这些排除项不影响对当前可运行系统的理解；若要针对 `planstar_frontend/admin` 的 Go 服务或 SQL 逐表做同等深度的分析，可再单独展开。
