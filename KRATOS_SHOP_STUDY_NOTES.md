# kratos-shop 深度学习笔记

> 研究对象：<https://github.com/arixbit/kratos-shop>
> 本地副本：`C:/Users/21263/WorkBuddy/kratos-shop`
> 分析基线：`main` @ `4f2b349`（2026-09-13）
> 撰写日期：2026-11-05
> 对照物：本工作区 `planstar_kratos_server`（Kratos **v3** + `AGENTS.md` 分层契约）

---

## 0. 一句话结论

这是一个 **Kratos v2.9.2 / Go 1.26.5** 的「微服务商城」教学型项目：8 个独立 Go module（6 个 gRPC 领域服务 + 2 个 HTTP BFF），以 Consul 做服务发现、RabbitMQ 做事件驱动，在订单链路上实现了**事务性 Outbox + 幂等消费 + 重试/DLQ + 超时取消**的完整 Saga 雏形。它的价值在于把「微服务之间怎么协作」讲得很具体；代价是分层不如 Kratos v3 layout 严谨（biz 层直接依赖 proto / gorm / JWT）、proto 契约靠复制分发且已产生漂移、以及一批明文凭据与单机状态的安全隐患。

**本次已实测：8 个 module 全部 `go build ./...` 通过（见第 13 节）。**

---

## 1. 拉取过程与仓库元数据

### 1.1 拉取

HTTPS `git clone` 在本环境被代理阻断（`Failed to connect to github.com:443 over proxy 127.0.0.1`），SSH 可用：

```bash
cd /c/Users/21263/WorkBuddy
git clone git@github.com:arixbit/kratos-shop.git kratos-shop
# Cloning into 'kratos-shop'... Updating files: 100% (776/776), done.
```

> 同会话早前 `git push` 走 SSH 也成功，说明本环境 GitHub SSH 通、HTTPS 不通；`api.github.com` / `raw.githubusercontent.com` / `codeload.github.com` 的 curl 正常。

### 1.2 GitHub 元数据（api.github.com，2026-11-05）

| 项 | 值 |
| --- | --- |
| 描述 | `kratos 框架微服务商城` |
| 主语言 | Go |
| 默认分支 | `main` |
| Stars / Forks / Issues | 233 / 38 / 27 |
| 创建 / 最后推送 | 2022-02-08 / 2026-09-13 |
| License | MIT |
| 主页 | `http://arixbit.github.io/kratos-shop/` |
| 唯一 tag | `v0.1.0` (`3d9a977`) |
| 总提交数 | 165 |
| HEAD | `4f2b349` Merge PR #30 (dependabot protobuf) |

分支列表里能看到大量 `dependabot/go_modules/...`、`dependabot/docker/...`，以及 `gh-pages`、`mocktest`、`rockytest` 三个功能/实验分支。最近的功能提交是 `4faf024 feat: complete order inventory flow`（2026-08-15）——即订单/库存事件链路是项目后期补上的核心特性。

### 1.3 文件规模

```
776 files (不含 .git)
  .go 290   .proto 142   .ts 85   .yaml 62   .md 54   .tsx 23
  .sql 20   .yml 12      .mod 8   .sum 8    .sh 7    .js 7 ...
```

按顶层目录：

| 目录 | 文件数 | 说明 |
| --- | --- | --- |
| `service/` | 381 | 6 个领域微服务，各自独立 `go.mod` |
| `web/` | 137 | Ant Design Pro v6 运营后台前端 |
| `shop/` | 87 | C 端商城 BFF（HTTP，8097） |
| `admin/` | 78 | 后台管理 BFF（HTTP，9099） |
| `deploy/` | 38 | docker-compose、K8s Kustomize、Prometheus/Grafana/Loki/Traefik |
| `sql/` | 20 | 6 个库的建表 + golang-migrate 迁移 |
| `scripts/` | 8 | 初始化、seed、迁移、备份恢复、e2e |
| `docs/` | 7 | 架构、配置、数据库、开发指南、演进方案、可行性报告 |

**8 个 Go module**（不是 go workspace，也没有根 `go.mod`）：

```
admin/  shop/  service/{user,goods,cart,order,inventory,payment}/go.mod
```

`module` 名都是单词（`module user`、`module shop`…），不是带域名的路径——这是老式 layout 的遗留，跨 module 无法直接 import，只能通过 gRPC 通信，反而强化了服务边界。

---

## 2. 技术栈全景

### 2.1 版本基线（与本地 v3 项目对比）

| 维度 | kratos-shop | 本工作区 planstar_kratos_server |
| --- | --- | --- |
| Kratos | **v2.9.2** | **v3** (`v3.0.0-...`) |
| Go | `go 1.26.5` | `go 1.25.7` |
| 模块组织 | 8 个独立 module | 单 module |
| 日志 | `kratos/v2/log`（`log.Helper`） | `log/slog` |
| ORM | **GORM v1.31.2** + `gorm.io/driver/postgres` | **Ent 0.14.6** |
| 数据库 | PostgreSQL 18（6 个独立库） | MySQL 运行时 / SQLite 测试 |
| 缓存 | `go-redis/redis/v8` + `redisotel` | 无（config 里有但不用） |
| 搜索 | `olivere/elastic/v7`（ES 7.17.22） | 无 |
| 服务发现 | `contrib/registry/consul/v2` + `hashicorp/consul/api` | 无 |
| MQ | `rabbitmq/amqp091-go` | 无 |
| 配置 | `config/file` + `conf.proto` | `config/file` + `env` + `conf.proto` |
| 校验 | **protoc-gen-validate (PGV)** `validate.rules` | **AIP** `filtering/ordering/pagination` + `field_behavior` |
| API 生成 | `kratos proto client` / `protoc-gen-go-http/v2` / `protoc-gen-go-errors/v2` | **Buf** + `protoc-gen-go-http/v3` + `protoc-gen-openapi` |
| 主键 | `int64` 自增 | **UUIDv7** |
| 删除 | GORM `DeletedAt` 软删 | `status` 翻转 |
| 分页 | 手写 `paginate()` / `Offset+Limit` | AIP `PageToken` + UUIDv7 tie-breaker |
| 认证 | `kratos/v2/middleware/auth/jwt` + `golang-jwt/jwt/v5` | 本次会话新接入的 contrib JWT |
| 追踪 | `kratos/v2/middleware/tracing` + **Jaeger exporter** | OTel contrib `tracing`/`metrics` |
| 指标 | `otel/exporters/prometheus` + `promhttp` 独立 `:910x` | OTel contrib `metrics` |
| 测试 | Ginkgo v1 + Gomega + gomock | 标准库 `testing` + fake repo |

### 2.2 关键直接依赖（以 `service/order/go.mod` 为例）

```
github.com/go-kratos/kratos/v2                     v2.9.2
github.com/go-kratos/kratos/contrib/registry/consul/v2  v2.0.0-20260404-f149714c
github.com/envoyproxy/protoc-gen-validate         v1.3.3
github.com/google/wire                            v0.7.0
github.com/hashicorp/consul/api                  v1.34.4
github.com/go-redis/redis/v8                     v8.11.5 (+ extra/redisotel v0.3.0)
github.com/rabbitmq/amqp091-go                   v1.13.0
github.com/onsi/ginkgo v1.16.5 / gomega v1.19.0
github.com/golang/mock                           v1.6.0
github.com/prometheus/client_golang              v1.24.1
go.opentelemetry.io/otel                         v1.45.0 (+ sdk, jaeger, prometheus exporters)
google.golang.org/grpc                           v1.83.0
google.golang.org/protobuf                      v1.36.11
gorm.io/driver/postgres v1.6.2 / gorm.io/gorm   v1.31.2
```

其他服务额外用到：`anaskhan96/go-password-encoder`（用户密码）、`mojocn/base64Captcha`（验证码）、`olivere/elastic/v7`（商品搜索）、`golang-jwt/jwt/v5`、`gorilla/handlers`（CORS）、`golang.org/x/time/rate`（限流）。

> 注意：`go.opentelemetry.io/otel/exporters/jaeger v1.17.0` 是 **已被上游废弃** 的 exporter（官方推荐 OTLP），属于技术债；`onsi/ginkgo v1` 也已停止维护。

---

## 3. 整体架构

### 3.1 8 个应用服务

| 服务 | 类型 | 端口 | 数据库 | 职责 |
| --- | --- | --- | --- | --- |
| `shop` | HTTP BFF | 8097（HTTP）/ 9001（gRPC，实际未启用） | 无 | C 端聚合、JWT 签发/刷新、验证码、限流 |
| `admin` | HTTP BFF | 9099 | 无（读 `shop_user`） | 运营后台聚合、管理员鉴权、看板、RBAC 权限点 |
| `service/user` | gRPC | 50051 | `shop_user` | 用户、密码校验、收货地址 CRUD |
| `service/goods` | gRPC | 50052 | `shop_goods` | 商品/SKU/分类/品牌/规格/属性、ES 搜索、库存查询、消费订单事件 |
| `service/cart` | gRPC | 50053 | `shop_cart` | 购物车增删改查 |
| `service/order` | gRPC | 50054 | `shop_order` | 创建订单、聚合 user/cart/goods、Outbox、看板统计 |
| `service/inventory` | gRPC | 50055 | `shop_inventory` | 库存查询/预占/确认扣减/释放，消费订单事件 |
| `service/payment` | gRPC | 50056 | `shop_payment` | 支付单创建与回调、发布支付结果事件 |

每个服务注册到 Consul，服务名为 `shop.user.service`、`shop.goods.service`、`shop.cart.service`、`shop.order.service`、`shop.inventory.service`、`shop.payment.service`、`shop.api`、`admin.api`。调用端统一用 `discovery:///shop.xxx.service` 作为 endpoint，配合 `grpc.WithDiscovery(rr)`。

### 3.2 三层网络

```
浏览器
  │
  ▼
Traefik :80  ── localhost → shop BFF :8097
             └─ admin.localhost → admin BFF :9099
                    │
                    │ gRPC (Consul discovery)
        ┌───────────┼───────────┬───────────┐
        ▼           ▼           ▼           ▼
     user:50051  goods:50052  cart:50053  order:50054 ──► payment:50056
                                            │                  │
                                            └── RabbitMQ ◄──────┘
                                                  │
                                          inventory:50055 / goods
```

`shop` 与 `admin` 不直接连业务库，只做协议转换（gRPC → HTTP/JSON）与按端裁剪。`order` 是唯一同时调用 3 个下游服务的领域服务。

### 3.3 基础设施（`deploy/docker-compose.yml`，22 个 service）

| 类别 | 组件与镜像 |
| --- | --- |
| 存储 | `postgres:18`、`redis:7`、`elasticsearch:7.17.22` |
| 消息 | `rabbitmq:3-management-alpine` |
| 注册发现 | `hashicorp/consul:1.19.1` |
| 追踪 | `jaegertracing/all-in-one:1.57`（Collector 14268 / UI 16686） |
| 指标 | `prom/prometheus:v2.53.0`、3 个 exporter（自有服务 + postgres + redis）、`grafana/grafana:11.1.0`、`prom/alertmanager:v0.27.0` |
| 日志 | `grafana/loki:3.1.0` + `grafana/promtail:3.1.0` |
| 网关 | `traefik:v3.1` |
| 应用 | 8 个自建服务（`build` 各自目录 Dockerfile） |

每个应用容器把 `deploy/configs/<svc>/` 挂到 `/data/conf`，启动命令 `./<svc> -conf /data/conf`。

---

## 4. 单服务内部结构（以 `service/user` 为范式）

```
service/user/
├── api/user/v1/            user.proto, error_reason.proto + 4 个生成文件
├── cmd/user/               main.go, wire.go, wire_gen.go
├── configs/                config.yaml, registry.yaml, config.local.example.yaml
├── internal/
│   ├── biz/                User/Address usecase + Repo 接口 + Transaction 接口
│   ├── data/               Data(*gorm.DB,*redis.Client) + PO + Repo 实现
│   ├── domain/             ★ 额外包：Address DO
│   ├── service/            DTO ↔ DO 转换、proto 响应组装
│   ├── server/             gRPC Server、Consul Registrar
│   ├── conf/               conf.proto / conf.pb.go
│   ├── pkg/metrics/        /metrics + /healthz 独立 HTTP
│   ├── mocks/              mockgen 产物
│   └── testdata/
├── third_party/            google/api、validate、errors、protobuf、openapi/v3
├── generate.go             //go:generate kratos proto client api
├── Makefile / Dockerfile / openapi.yaml / go.mod
```

### 4.1 三层 + 一个额外的 `domain` 包

标准的 Kratos layout 是 `service → biz → data`。这个项目额外引入 `internal/domain/`，且**用法不一致**：

- `service/user/internal/biz/user.go` 里 `type User struct` 定义在 **biz 包**（含 `gorm.DeletedAt` 字段！）。
- `service/user/internal/biz/address.go` 的 `AddressRepo` 却操作 `domain.Address`（定义在 `internal/domain/address.go`）。
- `service/cart`、`service/goods`、`service/order`、`service/inventory`、`service/payment` 则把**全部**领域模型放在 `internal/domain/`，biz 只放 usecase + repo 接口。

所以实际形态是三种 DO 归属混用：

| 服务 | DO 位置 | 说明 |
| --- | --- | --- |
| user | `biz.User` + `domain.Address` | 一半一半，且 `biz.User` 带 `gorm.DeletedAt` |
| goods / cart / order / inventory / payment | `internal/domain/` | 较统一，`domain` 就是 DO |
| shop / admin (BFF) | `biz.*`（且方法直接返回 `v1.*` proto） | biz 与 DTO 混合 |

PO（持久化对象）在 `data/` 里，带 `gorm` tag 与 `TableName()`；转换用自由函数 `modelToResponse` / `modelToBizResponse` 或 PO 方法 `ToDomain()`。这与 v3 layout 要求的「`new<Resource>(do)` / `toBiz(po)`」思路一致，只是命名不统一。

### 4.2 Wire 装配（`cmd/user/wire_gen.go`）

```go
func initApp(confServer *conf.Server, registry *conf.Registry, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
	db := data.NewDB(confData)
	client := data.NewRedis(confData)
	dataData, cleanup, err := data.NewData(confData, logger, db, client)
	userRepo := data.NewUserRepo(dataData, logger)
	userUsecase := biz.NewUserUsecase(userRepo, logger)
	addressRepo := data.NewAddressRepo(dataData, logger)
	addressUsecase := biz.NewAddressUsecase(addressRepo, logger)
	userService := service.NewUserService(userUsecase, addressUsecase, logger)
	grpcServer := server.NewGRPCServer(confServer, userService, logger)
	registrar := server.NewRegistrar(registry)
	app := newApp(logger, grpcServer, registrar)
	return app, func() { cleanup() }, nil
}
```

ProviderSet 分四处：`server.ProviderSet`、`data.ProviderSet`、`biz.ProviderSet`、`service.ProviderSet`，与 v3 相同。`data.ProviderSet` 里还有 `NewTransaction`（`*Data` 实现 `biz.Transaction`，用 `contextTxKey{}` 在 ctx 中传递 GORM tx）：

```go
func (d *Data) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}
```

### 4.3 启动生命周期（`cmd/user/main.go`）

```go
logger := log.With(log.NewStdLogger(os.Stdout), "ts", log.DefaultTimestamp, ...)
c := config.New(config.WithSource(file.NewSource(flagconf)))
c.Load()
var bc conf.Bootstrap; c.Scan(&bc)
setTracerProvider(bc.Trace.Endpoint)       // Jaeger + 100% 采样
metrics.Init("0.0.0.0:9101")                // 独立 /metrics + /healthz
var rc conf.Registry; c.Scan(&rc)           // ★ Registry 单独 Scan，不在 Bootstrap 里
app, cleanup, err := initApp(bc.Server, &rc, bc.Data, logger)
app.Run()
```

注意两点：
1. **`Registry` 不在 `Bootstrap` message 内**，`conf.proto` 里它是顶层 message，靠 Kratos 读取配置目录下所有 YAML 后「同一个 config 实例 Scan 两次」拿到。这是 v2 layout 的常见写法，v3 通常把 registry 合进 Bootstrap。
2. `conf.Server` 里声明了 `HTTP`（`0.0.0.0:8000`），但 `newApp` 只传 `gs`，**HTTP 段是死配置**（所有 6 个 gRPC 服务都如此）。

### 4.4 服务端中间件链

`service/user/internal/server/grpc.go`：

```go
grpc.Middleware(
	recovery.Recovery(),
	tracing.Server(),
	logging.Server(logger),
	metrics.Server(metrics.WithRequests(requests), metrics.WithSeconds(seconds)),
)
```

`shop/internal/server/http.go`（BFF，链更长）：

```go
http.Middleware(
	rateLimitMiddleware(),            // 自实现，按 IP 令牌桶
	recovery.Recovery(),
	validate.Validator(),             // PGV
	tracing.Server(),
	kmetrics.Server(...),
	selector.Server(                  // 白名单放行，其余校验 JWT
		jwt.Server(func(*jwt2.Token) (interface{}, error) { return []byte(ac.JwtKey), nil },
			jwt.WithSigningMethod(jwt2.SigningMethodHS256)),
	).Match(NewWhiteListMatcher()).Build(),
	logging.Server(logger),
),
http.Filter(handlers.CORS(...)),
```

白名单按 **canonical operation** 匹配（不是 HTTP path）：

```go
whiteList["/shop.shop.v1.Shop/Login"] = struct{}{}
whiteList["/shop.shop.v1.Shop/Register"] = struct{}{}
whiteList["/shop.shop.v1.Shop/Captcha"] = struct{}{}
whiteList["/shop.shop.v1.Shop/RefreshToken"] = struct{}{}
whiteList["/shop.shop.v1.Shop/GoodsList"] = struct{}{}
whiteList["/shop.shop.v1.Shop/GoodsDetail"] = struct{}{}
```

`admin` 在此基础上多一个 `adminOnlyMiddleware()`，从 JWT `MapClaims["AuthorityId"]` 取角色，`== 2` 才放行，否则 `403 ADMIN_ONLY`。

下游 gRPC 客户端链（`shop/internal/data/data.go`）：

```go
grpc.DialInsecure(ctx,
	grpc.WithEndpoint("discovery:///shop.user.service"),
	grpc.WithDiscovery(rr),
	grpc.WithMiddleware(tracing.Client(), recovery.Recovery()),
	grpc.WithTimeout(2*time.Second),
	grpc.WithOptions(grpcx.WithStatsHandler(&tracing.ClientHandler{})),
)
```

**这正好是本会话早前为 v3 项目接入 `selector`/`jwt`/`tracing`/`metrics` 等中间件时的 v2 对应写法**——可以对照 `internal/server/middleware.go` 看两代 API 的差异（v2 是 `middleware/auth/jwt` + `selector.Server(...).Match(...).Build()`，v3 是 contrib 包 + `middleware.Chain`）。

---

## 5. 与 `AGENTS.md` 分层契约的差异（重要）

`AGENTS.md` 规定：`service` 只说 DTO↔DO、`biz` 只说纯 DO（永不 import proto / PO / storage client）、`data` 只说 DO↔PO。kratos-shop 没有这份契约，实际代码有多处穿透：

| 位置 | 现象 | 违反点 |
| --- | --- | --- |
| `service/user/internal/biz/user.go` | `import "gorm.io/gorm"`，`User.DeletedAt gorm.DeletedAt` | biz 依赖存储驱动 |
| `shop/internal/biz/user.go` | `import v1 "shop/api/shop/v1"`，方法返回 `*v1.CaptchaReply` / `*v1.RegisterReply` | biz 直接说 DTO |
| `shop/internal/biz/*.go` | import `conf`、`jwt`、`pkg/captcha` | biz 依赖框架与配置 |
| `shop/internal/biz/user.go` | `jwt.FromContext(ctx)` 取 claims → `getUid(ctx)` | 认证上下文泄漏到领域层 |
| `shop/internal/data/user.go` | `userRepo` 实现 `biz.UserRepo`，但内部是 **gRPC client** 而非 DB | repo 语义被复用为远程调用适配器（其实是合理的 DIP，但命名与语义易误导） |
| `service/order/internal/biz/order.go` | import `cartV1` / `goodsV1` / `userV1` 三个下游 proto + `pkg/mq` | biz 依赖传输契约与 MQ |
| `service/goods/internal/biz/es_goods.go` | import `olivere/elastic/v7` 并直接构造 ES Query | biz 依赖存储查询语言 |
| `service/order/internal/service/order.go` | 启动 RabbitMQ consumer、`json.Unmarshal` 事件、调用 usecase 改状态 | service 层承担了消息适配层职责 |

**客观评价**：这些穿透对一个「学习 Kratos 各组件如何串起来」的示例项目是可接受的，甚至更直观；但如果要作为生产模板，应把 `domain` 提升为唯一 DO 归属、把 MQ consumer 移到独立的 `internal/mqadapter`（或 data 层）、把 JWT claims 解析收敛到 service 层再以纯 `userID int64` 传入 biz。

---

## 6. 核心业务链路深挖

### 6.1 登录 / 注册 / JWT（shop BFF）

1. `GET /api/users/captcha` → `base64Captcha.NewDriverDigit(80,250,5,0.7,80)`，验证码存 `base64Captcha.DefaultMemStore`（**进程内存**）。
2. `POST /api/users/login` → `UserUsecase.PassWordLogin`：
   - `captcha.Store.Verify(req.CaptchaId, req.Captcha, true)` 校验并清除；
   - `UserByMobile` 远程调 `user` 服务；`CheckPassword` 远程校验；
   - 签发 `auth.CustomClaims{ID, NickName, AuthorityId, RegisteredClaims{ExpiresAt: now+30d, Issuer:"Gyl"}}`，HS256。
3. `POST /api/users/refresh` → `auth.ParseToken(token, key)` 后**重新签一个 30 天**的 token。

`GetUid(ctx)` 从 JWT `MapClaims["ID"]`（float64）取用户 ID，被 cart / order / payment / address 各 usecase 复用。PGV 规则集中在 `shop.proto`（`string.len=11`、`min_len:8`、`int64.gt=0` 等），由 `validate.Validator()` 统一拦截。

密码哈希（`service/user/internal/data/user.go`）：

```go
// PBKDF2-SHA512, saltLen=16, iterations=10000, keyLen=32
// 格式：$pbkdf2-sha512$<salt>$<encodedPwd>
func EncryptPassword(psd string) string { ... }
func VerifyPassword(psd, encryptedPassword string) (bool, error) { ... }
```

### 6.2 创建订单：跨服务聚合 + 服务端价格重校验

`service/order/internal/biz/order.go::CreateOrder` 是最有教学价值的一段：

1. **校验购物车归属**：`cartRPC.ListCart(userId)` 建 `cartMap[cartId]`，逐条比对 `item.CartId` 存在且 `SkuId` 一致，防止前端伪造条目。
2. **服务端定价**：`goodsRPC.SkuList(skuIds)` 拿 SKU，`price = PromotionPrice > 0 ? PromotionPrice : Price`；与购物车 `GoodsPrice` 和请求 `SkuPrice` 双重比对，不一致返回 `400 PRICE_CHANGED`。**金额一律不信客户端**。
3. **查地址**：`userRPC.GetAddress(id, uid)`。
4. 生成 `orderSn = yyyyMMddHHmmss + 6 位随机数`，构造 `domain.Order`（初始状态 `OrderStatusInventoryPending`）、`OrderAddress`、`[]OrderGoods`。
5. **同一事务写订单 + Outbox**：`repo.Create(ctx, od, address, items, outbox)` 在 `ExecTx` 里一次提交（见 `data/order.go` L100-149）。
6. 下单成功后**尽力而为**删除购物车条目（失败只记日志，不阻塞订单）。

### 6.3 事务性 Outbox + 事件驱动 Saga（项目核心）

**状态机**（`service/order/internal/domain/order.go`）：

```
OrderStatusInventoryPending(0) ──inventory.locked──► PendingPayment(1) ──payment.success──► Paid(2)
        │                                                  │                                   │
        └──inventory.lock.failed / 超时──► Cancelled(5)     └──取消──► Cancelled(5)            ├──Ship──► Shipped(3)
                                                                                             └──Refund──► Refunded(7)
Paid ──► Signed(4) ──► Completed(6)
```

状态迁移全部走 `UpdateStatusIf(ctx, orderSn, from, to)` 做 **CAS**（`WHERE status = from`），因此 `MarkInventoryLocked` / `MarkPaid` / `CancelOrder` 天然幂等（返回 `changed=false` 时直接 `return nil`）。

**Outbox 中继**：`orderRepo.Create` 把 `order_event_outbox(status=0)` 与订单同事务写入；`NewOrderUsecase` 启动两个 goroutine：

```go
go uc.relayLoop()    // 每 1s: RelayOutbox → ListPendingOutbox(50) → Publish → MarkOutboxPublished
go uc.timeoutLoop()  // 每 30s: CancelExpiredOrders → ListPendingTimeout(15min) → CancelOrder
```

发布失败 `IncrementOutboxRetry`，`RetryCount+1 >= 5` 则 `MarkOutboxFailed`。

**Exchange / Routing key / 消费者全表**：

| Exchange | Routing key | 发布者 | 队列 | 消费者 / 动作 |
| --- | --- | --- | --- | --- |
| `order.exchange` (topic) | `order.created` | order (outbox relay) | `q.order.created` | inventory：`TryLock` |
| | `order.cancelled` | order | 同上 | inventory：`Release` |
| | `order.paid` | order | 同上 + `q.order.paid.goods` | inventory：`ConfirmDeduct`；goods：累加销量/更新 ES |
| `inventory.exchange` | `inventory.locked` | inventory | `q.inventory.result` | order：`MarkInventoryLocked` → PendingPayment |
| | `inventory.lock.failed` | inventory | 同上 | order：`MarkInventoryFailed` → Cancelled |
| | `inventory.deducted` / `inventory.release.failed` / `inventory.deduct.failed` / `inventory.released` | inventory | （无订阅者） | 预留 |
| `payment.exchange` | `payment.success` | payment | `q.payment.success` | order：`MarkPaid` → Paid，并再发 `order.paid` |
| `mall.dlx` | `failed.<queue>` | 所有 consumer | — | 超过 5 次重试后进死信 |

**幂等与可靠性**（`service/*/internal/pkg/mq/consumer.go`，三个服务各复制了一份）：

- 每条事件带 `event_id`；inventory / goods 用 `consumed_event` 表（`event_id` 主键）做 `IsConsumed` / `MarkConsumed`。
- `handleOrderCreated` 明确区分「业务失败」（如库存不足，属终态，仍记 consumed，发 `inventory.lock.failed`）与「DB/网络错误」（不记 consumed，交给 MQ 重试）——注释写得很清楚：

  ```go
  // 业务失败（库存不足等）也视为终态并幂等记录，避免无限重试；DB/网络错误不记录，交给 MQ 重试
  if opErr == nil || kerrors.FromError(opErr) != nil {
      _ = s.uc.MarkConsumed(ctx, evt.EventID, evt.OrderSn)
  }
  ```

- `Consumer.retryOrDLQ` 用 `x-retry-count` header 计数，`MaxRetry=5`，重试前 `time.Sleep((attempts+1)s)`，超限发 `mall.dlx` 后 Ack。
- 所有 consumer 都在 `for { NewConsumer → Run → Close → Sleep(5s) }` 里自愈重连；`Publisher.ensure()` 也做断线重连 + `ExchangeDeclare`。
- 消息 `DeliveryMode: amqp.Persistent`，队列 `durable=true`，手动 Ack。

### 6.4 库存：Redis 预扣 + DB 行锁 + 版本 + 流水

`service/inventory/internal/data/inventory.go::TryLock` 是双阶段：

1. **Redis 快速预扣**：`tryLockRedis`（key `stock:sku:<id>`），不足直接 `400 STOCK_NOT_ENOUGH`；DB 事务失败则 `rollbackRedisLock`。
2. **DB 事务**：
   ```go
   tx.Clauses(clause.Locking{Strength:"UPDATE"}).Where("sku_id = ?", ...).First(&inv)
   if inv.Inventory - inv.Locked < item.Num { return STOCK_NOT_ENOUGH }
   // 幂等：同 order_sn + sku_id 已存在 InventoryLock 则跳过
   tx.Model(&Inventory{}).Updates(map[string]interface{}{
       "locked": gorm.Expr("locked + ?", item.Num),
       "version": gorm.Expr("version + 1"),
   })
   tx.Create(&InventoryLock{...}); tx.Create(&InventoryFlow{Type:"lock"})
   ```
3. 成功后 `cacheInvalidate`。

`ConfirmDeduct` / `Release` 同理，写 `inventory_flows` 流水。查询走 `Query` 的 cache-aside（Redis → 回填 PG）。

### 6.5 支付（本地 mock）

`POST /api/payment/create` → `shop.PaymentUsecase.Create`（默认 `channel="mock"`）→ `payment.CreatePayment` 生成 `PAY-yyyyMMddHHmmssNNNNNN`。
`POST /api/payment/mock-pay` → `payment.PaymentCallback`：

- `MarkPaid` CAS 防重复回调；
- 通过 `orderRPC.GetOrder` 校验 `order.Status == "待支付"`；
- 发布 `payment.success`（含 `payment_no/order_sn/user_id/amount/trade_no/paid_at`）→ order 消费后 `MarkPaid` → 发 `order.paid` → inventory `ConfirmDeduct` + goods 更新销量。

> 注意：这里 `payment` 用**中文字符串** `"待支付"` 比对 `order.proto` 的 `status string`，而 `order` 的 `domain.OrderStatus` 是 int 常量。耦合在字符串字面量上，属于脆弱点。

### 6.6 商品与 ES 搜索

`service/goods/internal/data/es_goods.go` 定义 `goods` 索引 mapping（`name`/`goods_brief`/`sku.sku_name` 为 text+standard analyzer，`brands_id`/`category_id`/`is_hot` 等为精确字段，`sku` 为 nested）。`biz/EsGoodsUsecase.GoodsList` 组装 `elastic.BoolQuery`（multi_match、term、range、terms、field sort），`esRepo.GoodsList` 返回命中的 `goodsIds`，再 `repo.GoodsListByIDs` 回表查 PG 补齐详情——典型的 **ES 检索 ID + DB 回表**。

`goods/internal/pkg/mq/consumer.go` 让商品服务消费 `order.paid` 更新销量；`service/goods/api/service/inventory/v1/inventory.proto` 是商品服务查询库存用的客户端 stub 副本。

### 6.7 后台管理（admin BFF）

`admin/api/admin/v1/admin.proto` 有 **32 个 RPC**：登录/验证码/刷新、用户列表与地址、商品/分类/品牌的 CRUD 与上下架、订单列表/详情/发货/退款、`DashboardStats`、`PermissionList` / `RolePermissionList` / `RolePermissionUpdate`。

- 鉴权链多一层 `adminOnlyMiddleware()`（`AuthorityId == 2`）。
- 权限模型：`sql/09_permissions.sql` 建 `permissions` + `role_permissions` 两张表，`admin/internal/biz/permission.go` 提供 `ListCodesByRole` / `AssignRole`，前端按 code 做按钮级隐藏。
- `web/admin`（Ant Design Pro v6 / Umi Max 4 / React 19 / antd 6）页面目录：`dashboard`、`user(s)`、`orders`、`goods`、`system`、`exception`，`src/access.ts` 做前端权限。

---

## 7. 数据模型

6 个独立 PostgreSQL 库（物理隔离，符合微服务数据库边界）：

| 库 | 主要表 |
| --- | --- |
| `shop_user` | `users`、`user_addresses` |
| `shop_goods` | `brands`、`categories`、`goods_category_brands`、`goods_types`、`goods_type_brands`、`specifications_attrs`、`specifications_attr_values`、`goods_attr_groups`、`goods_attrs`、`goods_attr_values`、`goods`、`goods_skus`、`goods_specification_skus`、`goods_inventories`、`goods_images`、`consumed_event` |
| `shop_cart` | `shop_carts` |
| `shop_order` | `orders`、`order_address`、`order_goods`、`order_pays`、`order_event_outbox` |
| `shop_inventory` | `inventories`、`inventory_locks`、`inventory_flows`、`consumed_event` |
| `shop_payment` | `payments` |
| `shop_user`（admin 复用） | `permissions`、`role_permissions` |

`sql/migrations/<db>/000001_init.{up,down}.sql` 提供 golang-migrate 版本化迁移，`scripts/migrate.sh` / `init-db.sh` / `seed-demo.sh` / `backup.sh` / `restore.sh` / `e2e.sh` 覆盖初始化与运维。`scripts/generate_seed_demo.py` 生成 100 用户 / 50 商品 / 每人 3~5 笔模拟订单。

`consumed_event` 在 goods 与 inventory 两个库各建一份——因为幂等记录要跟着各自的业务事务走，不能跨库共享。

---

## 8. 可观测性与配置

| 能力 | 实现 |
| --- | --- |
| 链路追踪 | `setTracerProvider` → Jaeger exporter，`ParentBased(TraceIDRatioBased(1.0))` 全采样；`tracing.Server()` / `tracing.Client()` + `ClientHandler`；`redisotel.TracingHook` 覆盖 Redis |
| 指标 | `otel/exporters/prometheus` + `sdkmetric.NewMeterProvider`；`promhttp.Handler()` 挂独立 `:9101~9108/metrics`；`metrics.Server` 中间件记录 `server_requests_code_total` / `server_requests_seconds_bucket` |
| 健康 | 同端口 `/healthz` 返回 `ok`；Consul `WithHealthCheck(true)`；Compose healthcheck |
| 日志 | `log.NewStdLogger` + `tracing.TraceID()/SpanID()` 注入字段；Loki+Promtail 采集容器日志 |
| 告警 | `deploy/prometheus/rules/alerts.yml` + Alertmanager |
| 网关 | Traefik `localhost` → shop、`admin.localhost` → admin |

**配置体系**：Kratos `file.NewSource(目录)` 读取目录下所有非隐藏 YAML 并按文件名合并。仓库提交 `config.yaml` + `registry.yaml` 作为 Demo 默认；`config.local.yaml`（已 gitignore）排在后面，同名键覆盖——这就是文档里的「本地覆盖机制」，Compose 挂 `/data/conf`、K8s 用 ConfigMap + Secret 都复用这套机制。

`conf.proto` 的 `Bootstrap` 各服务略有差异：

| 服务 | Bootstrap 字段 |
| --- | --- |
| user / cart | `Server`, `Data`, `Trace` |
| goods / order / inventory / payment | + `Mq`, `Service`（下游 endpoint）, 部分 + `Auth` |
| shop | `Server`, `Data`, `Trace`, `Auth`, `Service` |
| admin | 同 shop（Service 只有 user/goods/order） |

`Registry` 始终是顶层 message，独立 `Scan`。

---

## 9. 代码生成与构建

每个 module 自带 `Makefile` + `generate.go`（`//go:generate kratos proto client api`）。生成链是 **Kratos v2 经典 protoc 插件**：

```make
init:  go install protoc-gen-go / protoc-gen-go-grpc / kratos/cmd/kratos/v2
             / protoc-gen-go-http/v2 / protoc-gen-go-errors/v2 / protoc-gen-openapi
api:   protoc --go_out --go-http_out --go-grpc_out --openapi_out --validate_out ...
errors:protoc --go-errors_out ...
config:protoc --go_out ... internal/**/*.proto
wire:  cd cmd/<svc> && wire
build: go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/<svc> ./cmd/<svc>
```

根 `Makefile` 只是遍历 8 个 module 执行 `build` / `test` / `vet` / `tidy` / `build-image`，另加 `up` / `down` / `infra-up` / `db-init` / `seed-demo` / `migrate-up` / `e2e` / `site-openapi`。

Dockerfile 是两阶段：`golang:1.26.5` 构建 → `debian:stable-slim` 运行，`VOLUME /data/conf`，每服务 `EXPOSE` 自己的端口。

> 与 v3 项目的差异：v3 用 Buf（`buf.gen.yaml` + `buf.gen.config.yaml`）统一生成，`protoc-gen-go-http/v3`、AIP 插件、`protoc-gen-openapi`，且明确「不要手改 `*.pb.go` / `wire_gen.go`」。kratos-shop 用 protoc + kratos CLI，把 `third_party/` 里一整套 google/protobuf、validate、openapi proto 复制到每个 module（142 个 proto 里绝大多数是这些副本）。

---

## 10. 测试策略

- 框架：**Ginkgo v1 + Gomega v1.19 + gomock v1.6**（`biz_suite_test.go` 里 `RunSpecs` + `BeforeEach` 建 controller）。
- 22 个 `*_test.go`，集中在 `user`（biz/data）、`order`（biz/domain/data/service/mq）、`inventory`（data）、`payment`（data/service）、`cart`（data）、`goods`（data）。
- `mockgen` 由 `//go:generate mockgen -destination=../mocks/... . <Interface>` 生成到 `internal/mocks/`。
- 覆盖点：密码哈希/校验、地址 CRUD、购物车合并、订单状态机与 outbox relay、MQ `deliveryRetryCount`、支付回调幂等、限流器（v3 项目里我们写过的同类测试）。
- 根 `make test` 遍历各 module `go test ./...`；`make e2e` 跑 `scripts/e2e.sh`（需服务已启动）。
- CI：`.github/workflows/ci.yml`（含 Redis service，从提交历史 `ci: add Redis service for tests` 可见）、`pages.yml`、`release.yml`。

---

## 11. 部署形态

| 形态 | 命令 | 说明 |
| --- | --- | --- |
| Docker 一键 | `make up` | 8 应用 + 全套基础设施 |
| 本地开发 | `make infra-up` → `scripts/init-db.sh` → `make build` → 各终端启动 | 便于断点 |
| 多机独立 | 各服务 `make build` / 交叉编译 → 拷贝 `bin/` + `configs/` → 改 `registry.yaml` 指向同一 Consul | 服务间靠 discovery 互找 |
| Kubernetes | `deploy/k8s/base/` Kustomize（namespace / apps / ingress / kustomization）+ `scripts/k8s-config.sh` | ConfigMap + Secret 挂同目录 |

`make up` 会占用 5432/6379/5672/8500/9200/80 等端口。

---

## 12. 与 v3 layout / `AGENTS.md` 的对照总结

| 主题 | kratos-shop (v2) | 本工作区 v3 + AGENTS.md |
| --- | --- | --- |
| 契约分发 | proto 文件复制到每个调用方 module（**已漂移**） | Buf 单一 `api/` 目录 + 生成 stub，调用方 import 同一包 |
| 分层 | `service/biz/data` + 混用 `domain`，多处穿透 | 严格 DTO→DO→PO，biz 纯 Go，方向不可反 |
| 依赖注入 | Wire（同） | Wire（同），`cmd` 唯一装配点 |
| 校验 | PGV `validate.rules` | AIP `field_behavior` / `filtering` / `ordering` |
| 列表查询 | 手写 `paginate` | AIP PageToken + `ListOption` helpers |
| 错误 | `errors.New(code, reason, msg)` 内联字符串 | `error_reason.proto` 枚举 + `Err<Resource><Cause>` |
| 中间件 | 各服务各自 `NewXXXServer` 里内联链 | `internal/server/middleware.go` 统一组合 + 配置开关 + biz 端口/data 适配器 |
| 传输 | 6 个 gRPC + 2 个 HTTP BFF | 单服务 HTTP + gRPC + 流式 SSE/WebSocket |
| 一致性 | Outbox + Saga + 幂等 + DLQ（**值得借鉴**） | 单库事务，无跨服务一致性问题 |

**最值得搬回 v3 项目的三件事**：
1. **事务性 Outbox**：业务写入与事件落库同事务，独立 relay 发布，避免「DB 成功但 MQ 失败」的双写不一致。
2. **CAS 状态迁移 + 幂等消费表**：`UpdateStatusIf(from,to)` 与 `consumed_event` 让所有事件处理天然可重放。
3. **服务端价格重校验**：订单金额绝不采信客户端传参。

**最不该搬的三件事**：proto 复制分发、biz 层 import proto/gorm/JWT、明文凭据入库。

---

## 13. 实际编译验证（本次执行）

环境：`/c/Program Files/Go/bin/go.exe`（go1.27.3，≥ go.mod 的 1.26.5，`GOTOOLCHAIN=auto` 不触发下载），`GOPROXY=https://goproxy.cn,direct`。

```bash
for d in admin service/user service/goods service/cart \
         service/order service/inventory service/payment shop; do
  (cd "$d" && go build ./...)
done
```

结果：**8/8 全部 `exit=0`**，无编译错误。首次运行下载了 consul/api、pgx、otel、elastic/v7、amqp091-go、base64Captcha 等依赖。

> 未执行 `go test ./...`（测试依赖 PostgreSQL/Redis/Consul/RabbitMQ 实例，需先 `make infra-up`），也未启动整套 Compose。

---

## 14. 问题 / 风险清单

### P0 — 安全

1. **明文凭据入库**：`configs/config.yaml` 与 `deploy/configs/*/config.yaml` 里硬编码 `postgres://postgres:root@...`、`redis.password: root`、`mq.addr: amqp://root:root@...`、`auth.jwt_key: hqFr%3ddt32DGlSTOI5cO6@TH#fFwYnP$S`。文档用「仅 Demo」+ `config.local.yaml` 覆盖机制缓解，但密钥已进 git 历史。**任何真实部署必须换钥并清理历史。**
2. **admin 与 shop 的 JWT key 不一致**：`admin` 用 `...@TH#Admin`，`shop`/`order` 用 `...@TH#fFwYnP$S`。若期望「一个 token 通两端」会失败；若是刻意隔离，`admin` 自己签发/校验，命名与文档都没说明，容易踩坑。
3. **`RefreshToken` 可无限续期过期 token**：`auth.ParseToken` 使用 `jwt.WithoutClaimsValidation()`，`ExpiresAt` 不被校验，任何历史 token（含已过期/已泄露）都能换一张新的 30 天 token，且无 refresh token 表/黑名单。
4. **JWT 有效期 30 天 + HS256 共享密钥**，无 `jti`/撤销机制。
5. **`X-Forwarded-For` 直接信任**：`clientIP()` 取 header 首值作为限流 key，客户端可随意伪造绕过限流。
6. **验证码存单机内存** `base64Captcha.DefaultMemStore`：多副本部署时验证码校验必然随机失败。

### P1 — 正确性与一致性

7. **限流器 `rateLimits map[string]*rate.Limiter` 永不淘汰**：每个新 IP 永久占用一条记录，长时间运行内存无限增长（shop 与 admin 各一份）。
8. **订单与购物车非原子**：`CreateOrder` 成功后循环 `DeleteCart`，失败仅记日志；若库存预占失败订单被取消，购物车条目已被删光，用户需重新加购。缺少补偿事务。
9. **Outbox relay 与 consumer 的自愈循环用 `context.Background()`**，不随 `kratos.App` 优雅退出；`relayLoop`/`timeoutLoop` goroutine 没有 stop channel。
10. **`payment.PaymentCallback` 用中文字面量 `order.Status != "待支付"`** 与 order 侧 int 状态机耦合，状态文案一改就静默失效。
11. **`shop/internal/biz/user.go::CreateUser`** 里 `createUser, err := uc.uRepo.CreateUser(...)` 未检查 `err` 就取 `createUser.ID`，下游报错会 nil 解引用。
12. **`service/user/internal/service/user.go::UpdateUser`** 在 `user == false` 时 `return nil, err`（此时 `err == nil`），返回 `(nil, nil)` 语义异常。
13. **`data/user.go::ListUser`** 先 `Find(&users)` 全表拿 `RowsAffected` 当 total，再 `Scopes(paginate)` 查一次；数据量大时全表扫描。
14. **`goods/internal/biz/es_goods.go`** 把 `elastic.NewFieldSort("click_num").Desc()` 追加到 `ShouldQuery`（查询子句而非排序子句），排序条件放错位置。
15. **`shop/cmd/shop/main.go` 的 `newApp` 只注册 `hs`，`gs` 被注释掉**：Wire 构建了 `grpcServer`（配置里 `0.0.0.0:9001`），但从不启动；README 却把 9001 列为默认端口。
16. 6 个 gRPC 服务的 `config.yaml` 都配了 `server.http.addr: 0.0.0.0:8000`，但 `newApp` 只传 gRPC Server，该段为死配置；多服务同时本地起还会端口冲突（若哪天启用了 HTTP）。

### P2 — 工程与一致性

17. **proto 契约复制且已漂移**（实测 md5）：

    | 文件 | 副本数 | 不同版本 |
    | --- | --- | --- |
    | `cart.proto` | 3 | 1 |
    | `goods.proto` | 4 | **2** |
    | `inventory.proto` | 2 | 1 |
    | `order.proto` | 4 | **3** |
    | `payment.proto` | 2 | 1 |
    | `user.proto` | 4 | **2**（`shop`/`admin` 少 `GetAddress`，`order`/`user` 有） |

    `shop/api/service/order/v1/order.proto` 只有 4 个 RPC，`admin` 与 `service/order` 是完整 9 个；`service/payment/api/service/order/v1/order.proto` 又是另一个裁剪版。任何字段变更都要手工同步 N 处。
18. **`error_reason.proto` 包名漂移**：`shop`/`admin` 是 `helloworld.v1`（模板残留），`service/order` 是 `Order.v1`（大写），其余是 `xxx.v1`；且枚举值基本都是 `USER_NOT_FOUND` / `CONTENT_MISSING` 两个占位。
19. **`MQ Consumer` 实现复制 3 份**（order / inventory / goods），`Publisher` 复制 2 份，任何修复要改多处。
20. **`docs/README.md` 说「v2」且「`web/` 不在本次维护范围内」**，而根 README 用大段篇幅介绍 `web/admin` 的页面与 `make seed-demo`——文档口径不一致。
21. **Ginkgo v1 / Jaeger exporter / `pkg/errors` 均为已停维护或已废弃依赖**；Dependabot 分支大量堆积未合并。
22. **测试覆盖不均**：`shop`/`admin`/`goods`(biz)/`user`(service) 基本没有测试；BFF 的 JWT、限流、selector 白名单无单测。
23. `third_party/` 整套 google/protobuf + openapi proto 在每个 module 重复存放，是 142 个 proto 的主要来源，仓库体积与维护成本偏高。
24. `Name = "shop.user.service"` 但 `kratos.ID(id+"user service")` 含空格；`shop` 的 `Version = "shop.api.v1"` 是硬编码字符串而非 ldflags 注入。

---

## 15. 命令速查

```bash
# 一键起全套（需 Docker）
make up / make down / make ps / make logs

# 本地开发
make infra-up
./scripts/init-db.sh           # 幂等初始化 6 库 + mock 数据
make build                     # 8 个 module
make seed-demo                 # 演示数据
cd service/user && ./bin/user -conf configs   # 其余 7 个同理

# 验证
curl -X POST http://127.0.0.1:8097/api/users/login \
  -H 'Content-Type: application/json' \
  -d '{"mobile":"13800138000","password":"12345678"}'
# 演示账号：普通 13800138000/12345678，管理员 13501167215/12345678

# 代码生成（单 module）
make init && make api && make errors && make config && make generate && make wire

# 测试
make test / make vet / make test-cover / make e2e

# 数据库迁移
make migrate-up / make migrate-down
```

代码生成与编译验证（本次实测）：

```bash
export PATH="/c/Program Files/Go/bin:$PATH"
export GOPROXY=https://goproxy.cn,direct
export GOTOOLCHAIN=auto
for d in admin service/user service/goods service/cart \
         service/order service/inventory service/payment shop; do
  (cd "$d" && go build ./...)
done   # 8/8 exit=0
```

---

## 16. 学习路线建议

按「从外到内、从静态到动态」读这个仓库效率最高：

1. `README.md` → `docs/architecture.md` → `docs/development.md`（先建立拓扑与端口心智模型）。
2. `service/user/` 通读一遍（最小的完整垂直切片：proto → service → biz → data → PO → Wire → main）。
3. `shop/` 读 `internal/server/http.go` + `internal/data/data.go`（BFF 如何把 gRPC 客户端伪装成 Repo）。
4. `service/order/internal/biz/order.go` + `data/order.go` + `pkg/mq/*`（最核心的 Outbox/Saga）。
5. `service/inventory/internal/service/inventory.go` + `data/inventory.go`（消费端幂等与库存锁）。
6. `service/payment/internal/service/payment.go` → 回到 `order` 的 `handlePaymentSuccess`，闭环。
7. `deploy/docker-compose.yml` + `sql/*.sql` + `scripts/init-db.sh`（把运行时与数据模型对上）。
8. 最后对照 `AGENTS.md` 逐条找分层穿透，思考「如果要把它重构到 v3 clean arch，哪些接口要搬、哪些要拆」。

---

*本笔记基于本地 `main@4f2b349` 全量源码阅读 + GitHub API 元数据 + 8 module 实际编译验证写成；所有代码片段均摘自对应文件，未做主观改写。*
